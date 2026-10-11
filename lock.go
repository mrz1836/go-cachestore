package cachestore

import (
	"context"
	"strings"
	"time"

	"github.com/mrz1836/go-cache"
	"github.com/pkg/errors"
)

// lockError is a failed WriteLock or WriteLockWithSecret
//
// Its text is the cause's followed by ErrLockCreateFailed's, as it has always
// been. It matches ErrLockCreateFailed and the cause, and ErrLockExists when
// another secret holds the lock
type lockError struct {
	cause error
	held  bool
}

// newLockError wraps the cause of a failed lock
func newLockError(cause error) error {
	return &lockError{cause: cause, held: errors.Is(cause, cache.ErrLockMismatch)}
}

// Error returns the cause's text followed by ErrLockCreateFailed's
func (e *lockError) Error() string {
	return e.cause.Error() + ": " + ErrLockCreateFailed.Error()
}

// Unwrap returns ErrLockCreateFailed, ErrLockExists when the lock is held, and the cause
func (e *lockError) Unwrap() []error {
	if e.held {
		return []error{ErrLockCreateFailed, ErrLockExists, e.cause}
	}
	return []error{ErrLockCreateFailed, e.cause}
}

// WriteLock will create a unique lock/secret with a TTL (seconds) to expire
// The lockKey is unique and should be deterministic
// The secret will be automatically generated and stored in the locked key (returned)
//
// The TTL must be at least one second: zero or less returns ErrInvalidTTL. Every
// failure to take the lock wraps ErrLockCreateFailed and its cause, and, when
// another secret holds the lock, ErrLockExists; compare with errors.Is
func (c *Client) WriteLock(ctx context.Context, lockKey string, ttl int64) (string, error) {
	var secret string
	var err error

	// Create a secret
	if secret, err = RandomHex(32); err != nil {
		// This will "ALMOST NEVER" error out
		return "", errors.Wrap(ErrSecretGenerationFailed, err.Error())
	}

	// Test the key and secret
	if err = validateLockValues(lockKey, secret); err != nil {
		return "", err
	}

	// A lock needs a TTL of at least one second
	if ttl <= 0 {
		return "", ErrInvalidTTL
	}

	// Lock using Redis
	if c.Engine() == Redis {
		if _, err = cache.WriteLock(
			ctx, c.options.redis, lockKey, secret, ttl,
		); err != nil {
			return "", newLockError(err)
		}
	} else if c.Engine() == FreeCache { // Lock using FreeCache
		if _, err = writeLockFreeCache(
			c.options.freeCache, lockKey, secret, ttl,
		); err != nil {
			return "", newLockError(err)
		}
	}

	return secret, nil
}

// WriteLockWithSecret will create a lock with the given secret with a TTL (seconds) to expire
// The lockKey is unique and should be deterministic
// The secret should be unique per instance/process that wants to acquire the lock
//
// Taking the lock again with the same secret succeeds and resets its TTL. The TTL
// must be at least one second: zero or less returns ErrInvalidTTL. Every failure
// to take the lock wraps ErrLockCreateFailed and its cause, and, when another
// secret holds the lock, ErrLockExists; compare with errors.Is
func (c *Client) WriteLockWithSecret(ctx context.Context, lockKey, secret string, ttl int64) (string, error) {
	var err error

	// Test the key and secret
	if err = validateLockValues(lockKey, secret); err != nil {
		return "", err
	}

	// A lock needs a TTL of at least one second
	if ttl <= 0 {
		return "", ErrInvalidTTL
	}

	// Lock using Redis
	if c.Engine() == Redis {
		if _, err = cache.WriteLock(
			ctx, c.options.redis, lockKey, secret, ttl,
		); err != nil {
			return "", newLockError(err)
		}
	} else if c.Engine() == FreeCache { // Lock using FreeCache
		if _, err = writeLockFreeCache(
			c.options.freeCache, lockKey, secret, ttl,
		); err != nil {
			return "", newLockError(err)
		}
	}

	return secret, nil
}

// WaitWriteLock will try to take a lock until it gets one or the TTW (in seconds) runs out
//
// While another secret holds the lock, it tries again every 10 milliseconds. It
// returns ctx's error as soon as ctx is done, any other failure at once, and,
// when the TTW runs out, the last attempt's error, which wraps ErrLockExists and
// ErrLockCreateFailed. A nil ctx waits with no cancellation. The lock TTL (in
// seconds) must be at least one: zero or less returns ErrInvalidTTL
func (c *Client) WaitWriteLock(ctx context.Context, lockKey string, ttl, ttw int64) (string, error) { //nolint:contextcheck // a nil ctx becomes context.Background(), so it waits without cancellation as it always has
	// Test the values
	if len(lockKey) == 0 {
		return "", ErrKeyRequired
	} else if ttw <= 0 {
		return "", ErrTTWCannotBeEmpty
	} else if ttl <= 0 {
		return "", ErrInvalidTTL
	}

	// A nil context waits with no cancellation, as it always has
	if ctx == nil {
		ctx = context.Background()
	}

	// Create the end time for the loop
	end := time.Now().Add(time.Duration(ttw) * time.Second)

	// Loop until we have a secret, a failure other than a held lock, or we are past the end time
	for {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", ctxErr
		}

		secret, err := c.WriteLock(ctx, lockKey, ttl)
		if err == nil {
			return secret, nil
		}
		if !errors.Is(err, ErrLockExists) || !time.Now().Before(end) {
			return "", err
		}

		if waitErr := waitForRetry(ctx, lockRetrySleepTime); waitErr != nil {
			return "", waitErr
		}
	}
}

// waitForRetry waits for d, or returns ctx's error if ctx is done first
func waitForRetry(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// ReleaseLock will release a given lock key only if the secret matches
//
// Taking a lock and releasing it are each one atomic step on both engines. On
// FreeCache, which has no compare-and-delete, a release leaves an empty value
// that expires within a second; Get reads it as missing, and the next lock
// takes it
func (c *Client) ReleaseLock(ctx context.Context, lockKey, secret string) (bool, error) {
	// Test the key and secret
	if err := validateLockValues(lockKey, secret); err != nil {
		return false, err
	}

	// Release the lock
	if c.Engine() == Redis {
		return cache.ReleaseLock(ctx, c.options.redis, lockKey, secret)
	}

	// Default is FreeCache
	return releaseLockFreeCache(c.options.freeCache, lockKey, secret)
}

// validateLockValues will validate and test the lock/secret values
func validateLockValues(lockKey, secret string) error {
	// Require a key to be present
	if len(strings.TrimSpace(lockKey)) == 0 {
		return ErrKeyRequired
	}

	// Require a secret to be present
	if len(strings.TrimSpace(secret)) == 0 {
		return ErrSecretRequired
	}
	return nil
}
