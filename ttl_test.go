package cachestore

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/coocood/freecache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// steppedClock is a FreeCache clock that moves only when a test advances it
type steppedClock struct {
	now atomic.Uint32
}

// Now returns the clock's current second
func (c *steppedClock) Now() uint32 {
	return c.now.Load()
}

// advance moves the clock forward
func (c *steppedClock) advance(seconds uint32) {
	c.now.Add(seconds)
}

// newSteppedFreeCacheClient returns a client on its own FreeCache, whose clock the test moves
func newSteppedFreeCacheClient(ctx context.Context, t *testing.T) (ClientInterface, *steppedClock) {
	t.Helper()
	clock := &steppedClock{}
	clock.now.Store(1_000_000)
	c, err := NewClient(ctx, WithFreeCacheConnection(freecache.NewCacheCustomTimer(4*1024*1024, clock)))
	require.NoError(t, err)
	require.NotNil(t, c)
	return c, clock
}

// newMiniredisClient returns a client on its own miniredis server, and the server
func newMiniredisClient(ctx context.Context, t *testing.T) (ClientInterface, *miniredis.Miniredis) {
	t.Helper()
	s := miniredis.RunT(t)
	c, err := NewClient(ctx, WithRedis(&RedisConfig{
		MaxConnectionLifetime: DefaultRedisMaxIdleTimeout,
		MaxIdleTimeout:        DefaultRedisMaxIdleTimeout,
		URL:                   s.Addr(),
	}))
	require.NoError(t, err)
	require.NotNil(t, c)
	return c, s
}

// ttlWriter stores a value at key with a TTL, through one of the TTL methods
type ttlWriter struct {
	name  string
	write func(ctx context.Context, c ClientInterface, key string, ttl time.Duration) error
}

// modelWriters are the methods that store a model with a TTL
func modelWriters() []ttlWriter {
	return []ttlWriter{
		{name: "SetModel", write: func(ctx context.Context, c ClientInterface, key string, ttl time.Duration) error {
			return c.SetModel(ctx, key, &genericStruct{StringField: testValue}, ttl)
		}},
		{name: "SetModelValue", write: func(ctx context.Context, c ClientInterface, key string, ttl time.Duration) error {
			return SetModelValue(ctx, c, key, genericStruct{StringField: testValue}, ttl)
		}},
	}
}

// modelStored reports whether a model is stored at testKey
func modelStored(ctx context.Context, t *testing.T, c ClientInterface) bool {
	t.Helper()
	_, found, err := GetModelValue[genericStruct](ctx, c, testKey)
	require.NoError(t, err)
	return found
}

// TestSetTTLRoundsUpToWholeSeconds proves a fractional TTL rounds up to whole seconds on both engines
func TestSetTTLRoundsUpToWholeSeconds(t *testing.T) {
	ctx := context.Background()

	t.Run("freecache: 500ms lives until the next second", func(t *testing.T) {
		c, clock := newSteppedFreeCacheClient(ctx, t)
		require.NoError(t, c.SetTTL(ctx, testKey, testValue, 500*time.Millisecond))

		got, err := c.Get(ctx, testKey)
		require.NoError(t, err)
		assert.Equal(t, testValue, got, "present before the clock moves")

		clock.advance(1)
		got, err = c.Get(ctx, testKey)
		require.NoError(t, err)
		assert.Empty(t, got, "gone one second later")
	})

	t.Run("freecache: 1.5s lives two seconds", func(t *testing.T) {
		c, clock := newSteppedFreeCacheClient(ctx, t)
		require.NoError(t, c.SetTTL(ctx, testKey, testValue, 1500*time.Millisecond))

		clock.advance(1)
		got, err := c.Get(ctx, testKey)
		require.NoError(t, err)
		assert.Equal(t, testValue, got, "present one second later")

		clock.advance(1)
		got, err = c.Get(ctx, testKey)
		require.NoError(t, err)
		assert.Empty(t, got, "gone two seconds later")
	})

	t.Run("redis: 500ms is one second", func(t *testing.T) {
		c, s := newMiniredisClient(ctx, t)
		require.NoError(t, c.SetTTL(ctx, testKey, testValue, 500*time.Millisecond))
		assert.Equal(t, time.Second, s.TTL(testKey))
	})

	t.Run("redis: 1.5s is two seconds", func(t *testing.T) {
		c, s := newMiniredisClient(ctx, t)
		require.NoError(t, c.SetTTL(ctx, testKey, testValue, 1500*time.Millisecond))
		assert.Equal(t, 2*time.Second, s.TTL(testKey))
	})
}

// TestSetModelRoundsUpToWholeSeconds proves SetModel and SetModelValue round a fractional TTL up on both engines
func TestSetModelRoundsUpToWholeSeconds(t *testing.T) {
	ctx := context.Background()

	for _, w := range modelWriters() {
		t.Run(w.name+" freecache: 500ms lives until the next second", func(t *testing.T) {
			c, clock := newSteppedFreeCacheClient(ctx, t)
			require.NoError(t, w.write(ctx, c, testKey, 500*time.Millisecond))
			assert.True(t, modelStored(ctx, t, c), "present before the clock moves")

			clock.advance(1)
			assert.False(t, modelStored(ctx, t, c), "gone one second later")
		})

		t.Run(w.name+" freecache: 1.5s lives two seconds", func(t *testing.T) {
			c, clock := newSteppedFreeCacheClient(ctx, t)
			require.NoError(t, w.write(ctx, c, testKey, 1500*time.Millisecond))

			clock.advance(1)
			assert.True(t, modelStored(ctx, t, c), "present one second later")

			clock.advance(1)
			assert.False(t, modelStored(ctx, t, c), "gone two seconds later")
		})

		t.Run(w.name+" redis: 500ms is one second", func(t *testing.T) {
			c, s := newMiniredisClient(ctx, t)
			require.NoError(t, w.write(ctx, c, testKey, 500*time.Millisecond))
			assert.Equal(t, time.Second, s.TTL(testKey))
		})

		t.Run(w.name+" redis: 1.5s is two seconds", func(t *testing.T) {
			c, s := newMiniredisClient(ctx, t)
			require.NoError(t, w.write(ctx, c, testKey, 1500*time.Millisecond))
			assert.Equal(t, 2*time.Second, s.TTL(testKey))
		})
	}
}

// TestZeroTTLStoresWithNoExpiry proves a TTL of 0 means no expiry on both engines
func TestZeroTTLStoresWithNoExpiry(t *testing.T) {
	ctx := context.Background()
	const day = 24 * time.Hour

	t.Run("SetTTL freecache", func(t *testing.T) {
		c, clock := newSteppedFreeCacheClient(ctx, t)
		require.NoError(t, c.SetTTL(ctx, testKey, testValue, 0))

		clock.advance(uint32(day / time.Second))
		got, err := c.Get(ctx, testKey)
		require.NoError(t, err)
		assert.Equal(t, testValue, got, "present a day later")
	})

	t.Run("SetTTL redis", func(t *testing.T) {
		c, s := newMiniredisClient(ctx, t)
		require.NoError(t, c.SetTTL(ctx, testKey, testValue, 0))
		assert.Equal(t, time.Duration(0), s.TTL(testKey), "no TTL on the key")

		s.FastForward(day)
		got, err := c.Get(ctx, testKey)
		require.NoError(t, err)
		assert.Equal(t, testValue, got, "present a day later")
	})

	t.Run("SetModel freecache", func(t *testing.T) {
		c, clock := newSteppedFreeCacheClient(ctx, t)
		require.NoError(t, c.SetModel(ctx, testKey, &genericStruct{StringField: testValue}, 0))

		clock.advance(uint32(day / time.Second))
		assert.True(t, modelStored(ctx, t, c), "present a day later")
	})

	t.Run("SetModel redis", func(t *testing.T) {
		c, s := newMiniredisClient(ctx, t)
		require.NoError(t, c.SetModel(ctx, testKey, &genericStruct{StringField: testValue}, 0))
		assert.Equal(t, time.Duration(0), s.TTL(testKey), "no TTL on the key")

		s.FastForward(day)
		assert.True(t, modelStored(ctx, t, c), "present a day later")
	})
}
