package cachestoretest_test

import (
	"context"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cachestore "github.com/mrz1836/go-cachestore"
	"github.com/mrz1836/go-cachestore/cachestoretest"
)

// TestEngineParityTTLs verifies the TTL rule and the lock errors against real
// Redis and Valkey servers. It is skipped automatically when Docker is unavailable.
func TestEngineParityTTLs(t *testing.T) {
	engines := []struct {
		name  string
		start func(testing.TB) (cachestore.ClientInterface, string)
	}{
		{name: "redis", start: cachestoretest.StartRedis},
		{name: "valkey", start: cachestoretest.StartValkey},
	}

	for _, engine := range engines {
		t.Run(engine.name, func(t *testing.T) {
			ctx := context.Background()
			client, _ := engine.start(t)

			// pttl reads a key's remaining TTL in milliseconds (-1: no expiry, -2: missing)
			pttl := func(key string) int64 {
				t.Helper()
				conn, err := client.Redis().GetConnectionWithContext(ctx)
				require.NoError(t, err)
				defer client.Redis().CloseConnection(conn)
				ms, err := redis.Int64(conn.Do("PTTL", key))
				require.NoError(t, err)
				return ms
			}

			// A fraction rounds up to whole seconds
			require.NoError(t, client.SetTTL(ctx, "parity:half", "v", 500*time.Millisecond))
			ms := pttl("parity:half")
			assert.GreaterOrEqual(t, ms, int64(1))
			assert.LessOrEqual(t, ms, int64(1000))

			require.NoError(t, client.SetTTL(ctx, "parity:one-and-a-half", "v", 1500*time.Millisecond))
			ms = pttl("parity:one-and-a-half")
			assert.GreaterOrEqual(t, ms, int64(1001))
			assert.LessOrEqual(t, ms, int64(2000))

			// Zero is no expiry, for values and models alike
			require.NoError(t, client.SetTTL(ctx, "parity:forever", "v", 0))
			assert.Equal(t, int64(-1), pttl("parity:forever"))

			require.NoError(t, client.SetModel(ctx, "parity:model-forever", &struct{ V string }{V: "v"}, 0))
			assert.Equal(t, int64(-1), pttl("parity:model-forever"))

			// A negative TTL is refused, and nothing is stored
			require.ErrorIs(t, client.SetTTL(ctx, "parity:negative", "v", -time.Second), cachestore.ErrInvalidTTL)
			assert.Equal(t, int64(-2), pttl("parity:negative"))

			// A lock needs a TTL of at least one second
			_, err := client.WriteLock(ctx, "parity:lock", 0)
			require.ErrorIs(t, err, cachestore.ErrInvalidTTL)

			// A held lock's error says so
			secret, err := client.WriteLock(ctx, "parity:lock", 5)
			require.NoError(t, err)

			_, err = client.WriteLock(ctx, "parity:lock", 5)
			require.ErrorIs(t, err, cachestore.ErrLockExists)
			require.ErrorIs(t, err, cachestore.ErrLockCreateFailed)

			released, err := client.ReleaseLock(ctx, "parity:lock", secret)
			require.NoError(t, err)
			assert.True(t, released)
		})
	}
}
