package cachestore

import (
	"context"
	"testing"
	"time"

	"github.com/mrz1836/go-cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClient_WriteLock will test the method WriteLock()
func TestClient_WriteLock(t *testing.T) {

	testCases := getInMemoryTestCases(t)
	for _, testCase := range testCases {
		t.Run(testCase.name+" - missing lock key", func(t *testing.T) {
			var secret string
			c, err := NewClient(context.Background(), testCase.opts)
			require.NotNil(t, c)
			require.NoError(t, err)

			secret, err = c.WriteLock(context.Background(), "", 30)
			assert.Equal(t, "", secret)
			require.Error(t, err)
			require.EqualError(t, err, ErrKeyRequired.Error())
		})

		t.Run(testCase.name+" - valid lock", func(t *testing.T) {
			var secret string
			c, err := NewClient(context.Background(), testCase.opts)
			require.NotNil(t, c)
			require.NoError(t, err)

			secret, err = c.WriteLock(context.Background(), testKey, 30)
			assert.Len(t, secret, 64)
			require.NoError(t, err)

			defer func() {
				_, _ = c.ReleaseLock(context.Background(), testKey, secret)
			}()
		})

		t.Run(testCase.name+" - lock conflict", func(t *testing.T) {
			var secret string
			c, err := NewClient(context.Background(), testCase.opts)
			require.NotNil(t, c)
			require.NoError(t, err)

			secret, err = c.WriteLock(context.Background(), testKey, 30)
			assert.Len(t, secret, 64)
			require.NoError(t, err)

			defer func() {
				_, _ = c.ReleaseLock(context.Background(), testKey, secret)
			}()

			// Lock exists with different secret
			secret, err = c.WriteLock(context.Background(), testKey, 30)
			assert.Equal(t, "", secret)
			require.EqualError(t, err, "key is locked with a different secret: failed creating cache lock")
		})
	}

	// todo: add redis lock tests
}

// TestClient_WriteLockWithSecret will test the method WriteLockWithSecret()
func TestClient_WriteLockWithSecret(t *testing.T) {

	testCases := getInMemoryTestCases(t)
	for _, testCase := range testCases {
		t.Run(testCase.name+" - missing lock key", func(t *testing.T) {
			var secret string
			c, err := NewClient(context.Background(), testCase.opts)
			require.NotNil(t, c)
			require.NoError(t, err)

			secret, err = c.WriteLockWithSecret(context.Background(), "", "", 30)
			assert.Equal(t, "", secret)
			require.Error(t, err)
			require.EqualError(t, err, ErrKeyRequired.Error())
		})

		t.Run(testCase.name+" - valid lock", func(t *testing.T) {
			var secret string
			c, err := NewClient(context.Background(), testCase.opts)
			require.NotNil(t, c)
			require.NoError(t, err)

			secret, err = c.WriteLockWithSecret(context.Background(), testKey, "secret", 30)
			assert.Len(t, secret, 6)
			require.NoError(t, err)

			defer func() {
				_, _ = c.ReleaseLock(context.Background(), testKey, secret)
			}()
		})

		t.Run(testCase.name+" - lock conflict", func(t *testing.T) {
			var secret string
			c, err := NewClient(context.Background(), testCase.opts)
			require.NotNil(t, c)
			require.NoError(t, err)

			secret, err = c.WriteLockWithSecret(context.Background(), testKey, "secret", 30)
			assert.Len(t, secret, 6)
			require.NoError(t, err)

			defer func() {
				_, _ = c.ReleaseLock(context.Background(), testKey, secret)
			}()

			// Lock exists with different secret
			secret, err = c.WriteLockWithSecret(context.Background(), testKey, "secret2", 30)
			assert.Equal(t, "", secret)
			require.EqualError(t, err, "key is locked with a different secret: failed creating cache lock")
		})

		t.Run(testCase.name+" - update lock ttl", func(t *testing.T) {
			var secret string
			c, err := NewClient(context.Background(), testCase.opts)
			require.NotNil(t, c)
			require.NoError(t, err)

			secret, err = c.WriteLockWithSecret(context.Background(), testKey, "secret", 30)
			assert.Len(t, secret, 6)
			require.NoError(t, err)

			defer func() {
				_, _ = c.ReleaseLock(context.Background(), testKey, secret)
			}()

			secret, err = c.WriteLockWithSecret(context.Background(), testKey, "secret", 30)
			assert.Len(t, secret, 6)
			require.NoError(t, err)
		})
	}
}

// TestClient_ReleaseLock will test the method ReleaseLock()
func TestClient_ReleaseLock(t *testing.T) {

	testCases := getInMemoryTestCases(t)
	for _, testCase := range testCases {
		t.Run(testCase.name+" - missing lock key", func(t *testing.T) {
			var success bool
			c, err := NewClient(context.Background(), testCase.opts)
			require.NotNil(t, c)
			require.NoError(t, err)

			success, err = c.ReleaseLock(context.Background(), "", "some-value")
			assert.False(t, success)
			require.Error(t, err)
			require.EqualError(t, err, ErrKeyRequired.Error())
		})

		t.Run(testCase.name+" - missing secret", func(t *testing.T) {
			var success bool
			c, err := NewClient(context.Background(), testCase.opts)
			require.NotNil(t, c)
			require.NoError(t, err)

			success, err = c.ReleaseLock(context.Background(), testKey, "")
			assert.False(t, success)
			require.Error(t, err)
			require.EqualError(t, err, ErrSecretRequired.Error())
		})

		t.Run(testCase.name+" - valid release", func(t *testing.T) {
			var secret string
			var success bool
			c, err := NewClient(context.Background(), testCase.opts)
			require.NotNil(t, c)
			require.NoError(t, err)

			secret, err = c.WriteLock(context.Background(), testKey, 30)
			assert.Len(t, secret, 64)
			require.NoError(t, err)

			success, err = c.ReleaseLock(context.Background(), testKey, secret)
			assert.True(t, success)
			require.NoError(t, err)
		})

		t.Run(testCase.name+" - invalid secret", func(t *testing.T) {
			var secret string
			var success bool
			c, err := NewClient(context.Background(), testCase.opts)
			require.NotNil(t, c)
			require.NoError(t, err)

			secret, err = c.WriteLock(context.Background(), testKey, 30)
			assert.Len(t, secret, 64)
			require.NoError(t, err)

			success, err = c.ReleaseLock(context.Background(), testKey, secret+"-bad-key")
			require.Error(t, err)
			require.ErrorIs(t, err, cache.ErrLockMismatch)
			assert.False(t, success)
		})
	}

	// todo: add redis lock tests
}

// TestClient_WaitWriteLock will test the method WaitWriteLock()
func TestClient_WaitWriteLock(t *testing.T) {

	testCases := getInMemoryTestCases(t)
	for _, testCase := range testCases {

		t.Run(testCase.name+" - missing lock key", func(t *testing.T) {
			var ctx context.Context

			var secret string
			c, err := NewClient(ctx, testCase.opts)
			require.NotNil(t, c)
			require.NoError(t, err)

			secret, err = c.WaitWriteLock(ctx, "", 30, 10)
			assert.Equal(t, "", secret)
			require.Error(t, err)
			require.EqualError(t, err, ErrKeyRequired.Error())
		})

		t.Run(testCase.name+" - missing ttw", func(t *testing.T) {
			var ctx context.Context
			var secret string
			c, err := NewClient(ctx, testCase.opts)
			require.NotNil(t, c)
			require.NoError(t, err)

			secret, err = c.WaitWriteLock(ctx, testKey, 30, 0)
			assert.Equal(t, "", secret)
			require.Error(t, err)
			require.EqualError(t, err, ErrTTWCannotBeEmpty.Error())
		})

		t.Run(testCase.name+" - valid lock", func(t *testing.T) {
			var ctx context.Context
			var secret string
			c, err := NewClient(ctx, testCase.opts)
			require.NotNil(t, c)
			require.NoError(t, err)

			secret, err = c.WaitWriteLock(ctx, testKey, 30, 5)
			assert.Len(t, secret, 64)
			require.NoError(t, err)

			defer func() {
				_, _ = c.ReleaseLock(ctx, testKey, secret)
			}()
		})

		t.Run(testCase.name+" - lock jammed for a few seconds", func(t *testing.T) {
			var ctx context.Context
			var secret string
			c, err := NewClient(ctx, testCase.opts)
			require.NotNil(t, c)
			require.NoError(t, err)

			secret, err = c.WaitWriteLock(ctx, testKey, 2, 5)
			assert.Len(t, secret, 64)
			require.NoError(t, err)

			defer func() {
				_, _ = c.ReleaseLock(ctx, testKey, secret)
			}()

			testCase.FastForward(6 * time.Second)

			secret, err = c.WaitWriteLock(ctx, testKey, 10, 5)
			assert.Len(t, secret, 64)
			require.NoError(t, err)

			defer func() {
				_, _ = c.ReleaseLock(ctx, testKey, secret)
			}()
		})

		t.Run(testCase.name+" - lock jammed, never completes", func(t *testing.T) {
			var ctx context.Context
			var secret string
			c, err := NewClient(ctx, testCase.opts)
			require.NotNil(t, c)
			require.NoError(t, err)

			secret, err = c.WaitWriteLock(ctx, testKey, 30, 5)
			assert.Len(t, secret, 64)
			require.NoError(t, err)

			defer func() {
				_, _ = c.ReleaseLock(ctx, testKey, secret)
			}()

			secret, err = c.WaitWriteLock(ctx, testKey, 10, 2)
			assert.Equal(t, "", secret)
			require.Error(t, err)
		})
	}
}
