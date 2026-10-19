package cachestoretest_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cachestore "github.com/mrz1836/go-cachestore"
	"github.com/mrz1836/go-cachestore/cachestoretest"
)

// TestEngineParityGetOrLoad verifies that two clients on one real Redis or
// Valkey server share one load through the fill lock, and that a waiter gives up
// when the lock stays held. It is skipped automatically when Docker is unavailable.
func TestEngineParityGetOrLoad(t *testing.T) {
	engines := []struct {
		name  string
		start func(testing.TB) (cachestore.ClientInterface, string)
	}{
		{name: "redis", start: cachestoretest.StartRedis},
		{name: "valkey", start: cachestoretest.StartValkey},
	}

	type result struct {
		value int
		err   error
	}

	for _, engine := range engines {
		t.Run(engine.name, func(t *testing.T) {
			ctx := t.Context()
			first, url := engine.start(t)
			second, err := cachestore.NewClient(ctx, cachestore.WithRedis(&cachestore.RedisConfig{URL: url}))
			require.NoError(t, err)
			t.Cleanup(func() { second.Close(ctx) })

			// The first client's load blocks until released, or until its context ends
			const key = "parity:get-or-load"
			release := make(chan struct{})
			var loads atomic.Int32
			load := func(loadCtx context.Context) (int, error) {
				loads.Add(1)
				select {
				case <-release:
					return 42, nil
				case <-loadCtx.Done():
					return 0, loadCtx.Err()
				}
			}

			firstResult := make(chan result, 1)
			go func() {
				value, loadErr := cachestore.GetOrLoad(ctx, first, key, time.Minute, load)
				firstResult <- result{value: value, err: loadErr}
			}()
			require.Eventually(t, func() bool { return loads.Load() == 1 }, 10*time.Second, time.Millisecond)

			// The second client finds the fill lock held, and polls for the value
			secondResult := make(chan result, 1)
			go func() {
				value, loadErr := cachestore.GetOrLoadWithOptions(ctx, second, key, time.Minute, load,
					cachestore.LoadOptions{PollInterval: 10 * time.Millisecond})
				secondResult <- result{value: value, err: loadErr}
			}()
			time.Sleep(50 * time.Millisecond)
			close(release)

			a, b := <-firstResult, <-secondResult
			require.NoError(t, a.err)
			require.NoError(t, b.err)
			assert.Equal(t, 42, a.value)
			assert.Equal(t, 42, b.value)
			assert.Equal(t, int32(1), loads.Load())

			// A fill lock that stays held makes a waiter give up after LockWait
			const heldKey = "parity:get-or-load-held"
			_, err = first.WriteLock(ctx, heldKey+":fill-lock", 30)
			require.NoError(t, err)

			_, err = cachestore.GetOrLoadWithOptions(ctx, second, heldKey, time.Minute, load,
				cachestore.LoadOptions{LockWait: 200 * time.Millisecond, PollInterval: 20 * time.Millisecond})
			require.ErrorIs(t, err, cachestore.ErrLoadInProgress)
			assert.Equal(t, int32(1), loads.Load())
		})
	}
}
