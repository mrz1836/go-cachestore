package cachestore

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNegativeTTLIsRefused proves a negative TTL returns ErrInvalidTTL and stores nothing
func TestNegativeTTLIsRefused(t *testing.T) {
	ctx := context.Background()
	writers := append([]ttlWriter{
		{name: "SetTTL", write: func(ctx context.Context, c ClientInterface, key string, ttl time.Duration) error {
			return c.SetTTL(ctx, key, testValue, ttl)
		}},
	}, modelWriters()...)

	for _, e := range lockEngines() {
		for _, w := range writers {
			for _, ttl := range []time.Duration{-time.Nanosecond, -time.Second} {
				t.Run(e.name+" "+w.name+" "+ttl.String(), func(t *testing.T) {
					c := e.client(t)
					require.ErrorIs(t, w.write(ctx, c, testKey, ttl), ErrInvalidTTL)

					got, err := c.Get(ctx, testKey)
					require.NoError(t, err)
					assert.Empty(t, got, "nothing is stored")
				})
			}
		}
	}
}

// TestLockTTLMustBePositive proves a lock TTL of zero or less returns ErrInvalidTTL and takes no lock
func TestLockTTLMustBePositive(t *testing.T) {
	ctx := context.Background()
	takers := []struct {
		name string
		take func(c ClientInterface, ttl int64) (string, error)
	}{
		{name: "WriteLock", take: func(c ClientInterface, ttl int64) (string, error) {
			return c.WriteLock(ctx, testKey, ttl)
		}},
		{name: "WriteLockWithSecret", take: func(c ClientInterface, ttl int64) (string, error) {
			return c.WriteLockWithSecret(ctx, testKey, "secret", ttl)
		}},
		{name: "WaitWriteLock", take: func(c ClientInterface, ttl int64) (string, error) {
			return c.WaitWriteLock(ctx, testKey, ttl, 1)
		}},
	}

	for _, e := range lockEngines() {
		for _, tk := range takers {
			for _, ttl := range []int64{0, -1} {
				t.Run(e.name+" "+tk.name+" "+time.Duration(ttl*int64(time.Second)).String(), func(t *testing.T) {
					c := e.client(t)
					secret, err := tk.take(c, ttl)
					require.ErrorIs(t, err, ErrInvalidTTL)
					assert.Empty(t, secret)

					secret, err = c.WriteLock(ctx, testKey, 30)
					require.NoError(t, err, "the lock was not taken")
					assert.NotEmpty(t, secret)
				})
			}
		}

		t.Run(e.name+" an empty key answers first", func(t *testing.T) {
			c := e.client(t)
			_, err := c.WriteLock(ctx, "", 0)
			require.ErrorIs(t, err, ErrKeyRequired)

			_, err = c.WaitWriteLock(ctx, "", 0, 1)
			require.ErrorIs(t, err, ErrKeyRequired)
		})
	}
}
