package cachestore

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/coocood/freecache"
	"github.com/mrz1836/go-cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newFreeCacheClient returns a client on its own 4 MiB FreeCache
func newFreeCacheClient(t *testing.T) ClientInterface {
	t.Helper()
	c, err := NewClient(context.Background(), WithFreeCacheConnection(freecache.NewCache(4*1024*1024)))
	require.NoError(t, err)
	require.NotNil(t, c)
	return c
}

// TestFreeCacheLockHasOneHolderAtATime proves a FreeCache lock is taken and released atomically
func TestFreeCacheLockHasOneHolderAtATime(t *testing.T) {
	t.Run("acquire", func(t *testing.T) {
		const rounds = 1000
		const callers = 32

		ctx := context.Background()
		c := newFreeCacheClient(t)

		for round := range rounds {
			key := fmt.Sprintf("race-acquire-%d", round)
			start := make(chan struct{})
			var winners atomic.Int32
			var wg sync.WaitGroup
			for range callers {
				wg.Go(func() {
					<-start
					if secret, err := c.WriteLock(ctx, key, 30); err == nil && len(secret) > 0 {
						winners.Add(1)
					}
				})
			}
			close(start)
			wg.Wait()

			if n := winners.Load(); n != 1 {
				t.Fatalf("round %d: %d callers got the lock, want exactly 1", round, n)
			}
		}
	})

	t.Run("release", func(t *testing.T) {
		const workers = 8
		const turns = 200
		const key = "race-release"

		ctx := context.Background()
		c := newFreeCacheClient(t)

		var holders atomic.Int32
		var wg sync.WaitGroup
		for range workers {
			wg.Go(func() {
				for turn := range turns {
					// The TTL outlasts the test: every holder releases its lock itself
					var secret string
					for {
						var lockErr error
						if secret, lockErr = c.WriteLock(ctx, key, 30); lockErr == nil {
							break
						}
						runtime.Gosched()
					}

					if n := holders.Add(1); n > 1 {
						t.Errorf("turn %d: %d callers hold the lock at once, want 1", turn, n)
					}
					holders.Add(-1)

					released, err := c.ReleaseLock(ctx, key, secret)
					if err != nil || !released {
						t.Errorf("turn %d: releasing the held lock returned %v, %v; want true, nil", turn, released, err)
					}
				}
			})
		}
		wg.Wait()
	})

	t.Run("a release keeps another holder's lock", func(t *testing.T) {
		const key = "release-guard"

		ctx := context.Background()
		c := newFreeCacheClient(t)

		secretA, err := c.WriteLock(ctx, key, 30)
		require.NoError(t, err)
		require.NotEmpty(t, secretA)

		again, err := c.WriteLockWithSecret(ctx, key, secretA, 30)
		require.NoError(t, err)
		assert.Equal(t, secretA, again)

		released, err := c.ReleaseLock(ctx, key, "another-secret")
		require.ErrorIs(t, err, cache.ErrLockMismatch)
		assert.False(t, released)

		third, err := c.WriteLock(ctx, key, 30)
		require.Error(t, err)
		assert.Empty(t, third)

		released, err = c.ReleaseLock(ctx, key, secretA)
		require.NoError(t, err)
		assert.True(t, released)

		released, err = c.ReleaseLock(ctx, key, secretA)
		require.NoError(t, err)
		assert.True(t, released, "releasing a released lock succeeds, as before")

		value, err := c.Get(ctx, key)
		require.NoError(t, err)
		assert.Empty(t, value)

		next, err := c.WriteLock(ctx, key, 30)
		require.NoError(t, err)
		assert.NotEmpty(t, next)
	})
}
