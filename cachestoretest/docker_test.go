package cachestoretest_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cachestore "github.com/mrz1836/go-cachestore"
	"github.com/mrz1836/go-cachestore/cachestoretest"
)

// TestEngineParity verifies cachestore behaves identically against real Redis
// and Valkey servers. It is skipped automatically when Docker is unavailable.
func TestEngineParity(t *testing.T) {
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
			client, url := engine.start(t)

			require.Equal(t, cachestore.Redis, client.Engine())
			assert.NotEmpty(t, url)

			// Set + Get round-trip
			const key, value = "parity:key", "parity-value"
			require.NoError(t, client.Set(ctx, key, value))

			got, err := client.Get(ctx, key)
			require.NoError(t, err)
			assert.Equal(t, value, got)

			// Delete (a missing key reads back as an empty string, no error)
			require.NoError(t, client.Delete(ctx, key))
			got, err = client.Get(ctx, key)
			require.NoError(t, err)
			assert.Empty(t, got)

			// Lock + release round-trip (exercises real server-side Lua)
			const lockKey = "parity:lock"
			secret, err := client.WriteLock(ctx, lockKey, 5)
			require.NoError(t, err)
			require.NotEmpty(t, secret)

			released, err := client.ReleaseLock(ctx, lockKey, secret)
			require.NoError(t, err)
			assert.True(t, released)
		})
	}
}
