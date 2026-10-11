package cachestore

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// heldLockText is the error text of a lock another secret holds
const heldLockText = "key is locked with a different secret: failed creating cache lock"

// lockEngine builds a fresh client on one engine
type lockEngine struct {
	name   string
	client func(t *testing.T) ClientInterface
}

// lockEngines are FreeCache and Redis (on miniredis)
func lockEngines() []lockEngine {
	return []lockEngine{
		{name: "freecache", client: newFreeCacheClient},
		{name: "redis", client: func(t *testing.T) ClientInterface {
			c, _ := newMiniredisClient(t.Context(), t)
			return c
		}},
	}
}

// lockMethod takes a lock through WriteLock or WriteLockWithSecret
type lockMethod struct {
	name string
	lock func(ctx context.Context, c ClientInterface, key, secret string) (string, error)
}

// lockMethods are the two ways to take a lock
func lockMethods() []lockMethod {
	return []lockMethod{
		{name: "WriteLock", lock: func(ctx context.Context, c ClientInterface, key, _ string) (string, error) {
			return c.WriteLock(ctx, key, 30)
		}},
		{name: "WriteLockWithSecret", lock: func(ctx context.Context, c ClientInterface, key, secret string) (string, error) {
			return c.WriteLockWithSecret(ctx, key, secret, 30)
		}},
	}
}

// TestWaitWriteLockStopsWhenItsContextIsDone proves a wait ends soon after its context is canceled
func TestWaitWriteLockStopsWhenItsContextIsDone(t *testing.T) {
	for _, e := range lockEngines() {
		t.Run(e.name, func(t *testing.T) {
			c := e.client(t)
			_, err := c.WriteLock(context.Background(), testKey, 30)
			require.NoError(t, err)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			canceledAt := make(chan time.Time, 1)
			time.AfterFunc(50*time.Millisecond, func() {
				canceledAt <- time.Now()
				cancel()
			})

			secret, err := c.WaitWriteLock(ctx, testKey, 30, 10)
			returned := time.Now()

			require.ErrorIs(t, err, context.Canceled)
			assert.Empty(t, secret)
			assert.Less(t, returned.Sub(<-canceledAt), 500*time.Millisecond, "returns soon after the cancel")
		})
	}
}

// TestWaitWriteLockReturnsAStoreFailureAtOnce proves a failure other than a held lock ends the wait
func TestWaitWriteLockReturnsAStoreFailureAtOnce(t *testing.T) {
	ctx := context.Background()
	c, s := newMiniredisClient(ctx, t)
	s.Close()

	start := time.Now()
	secret, err := c.WaitWriteLock(ctx, testKey, 30, 5)
	elapsed := time.Since(start)

	require.ErrorIs(t, err, ErrLockCreateFailed)
	require.NotErrorIs(t, err, ErrLockExists)
	assert.Empty(t, secret)
	assert.Less(t, elapsed, time.Second, "returns at once")
}

// TestWaitWriteLockReportsTheHeldLockWhenTheWaitRunsOut proves the last attempt's error is returned
func TestWaitWriteLockReportsTheHeldLockWhenTheWaitRunsOut(t *testing.T) {
	for _, e := range lockEngines() {
		t.Run(e.name, func(t *testing.T) {
			ctx := context.Background()
			c := e.client(t)
			_, err := c.WriteLock(ctx, testKey, 30)
			require.NoError(t, err)

			secret, err := c.WaitWriteLock(ctx, testKey, 30, 1)
			assert.Empty(t, secret)
			require.ErrorIs(t, err, ErrLockExists)
			require.ErrorIs(t, err, ErrLockCreateFailed)
			require.EqualError(t, err, heldLockText)
		})
	}
}

// TestWaitWriteLockWithANilContextWaitsAsBefore proves a nil context still waits out its TTW
func TestWaitWriteLockWithANilContextWaitsAsBefore(t *testing.T) {
	for _, e := range lockEngines() {
		t.Run(e.name, func(t *testing.T) {
			var ctx context.Context // nil, as callers may pass it
			c := e.client(t)

			secret, err := c.WaitWriteLock(ctx, testKey, 30, 1)
			require.NoError(t, err)
			assert.NotEmpty(t, secret, "a free key is taken")

			start := time.Now()
			other, err := c.WaitWriteLock(ctx, testKey, 30, 1)
			elapsed := time.Since(start)

			assert.Empty(t, other)
			require.ErrorIs(t, err, ErrLockExists)
			assert.GreaterOrEqual(t, elapsed, 900*time.Millisecond, "a held key waits out the TTW")
			assert.Less(t, elapsed, 3*time.Second)
		})
	}
}

// TestWriteLockTellsAHeldLockFromAFailure proves a held lock's error wraps ErrLockExists and a failure's doesn't
func TestWriteLockTellsAHeldLockFromAFailure(t *testing.T) {
	ctx := context.Background()

	for _, e := range lockEngines() {
		for _, m := range lockMethods() {
			t.Run(e.name+" "+m.name+" held", func(t *testing.T) {
				c := e.client(t)
				_, err := c.WriteLockWithSecret(ctx, testKey, "holder", 30)
				require.NoError(t, err)

				secret, err := m.lock(ctx, c, testKey, "another")
				assert.Empty(t, secret)
				require.ErrorIs(t, err, ErrLockExists)
				require.ErrorIs(t, err, ErrLockCreateFailed)
				require.EqualError(t, err, heldLockText)
			})
		}
	}

	for _, m := range lockMethods() {
		t.Run("redis "+m.name+" store down", func(t *testing.T) {
			c, s := newMiniredisClient(ctx, t)
			s.Close()

			secret, err := m.lock(ctx, c, testKey, "another")
			assert.Empty(t, secret)
			require.ErrorIs(t, err, ErrLockCreateFailed)
			require.NotErrorIs(t, err, ErrLockExists)
			assert.True(t, strings.HasSuffix(err.Error(), ": "+ErrLockCreateFailed.Error()),
				"keeps the cause's text before ErrLockCreateFailed's: %q", err.Error())
		})
	}
}
