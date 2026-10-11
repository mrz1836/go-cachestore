package cachestore

import (
	"runtime/debug"

	"github.com/coocood/freecache"
	"github.com/mrz1836/go-cache"
)

const (

	// DefaultCacheSize in bytes, where 1024 * 1024 represents a single Megabyte, and 100 * 1024*1024 represents 100 Megabytes.
	DefaultCacheSize = 100 * 1024 * 1024

	// DefaultGCPercent is the percentage when full it will run GC
	DefaultGCPercent = 20
)

// loadFreeCache will load the FreeCache client
//
// This is a default cache solution for running a local single server.
func loadFreeCache(cacheSize, percent int) (c *freecache.Cache) {
	// Set the defaults for cache size
	if cacheSize <= 0 {
		cacheSize = DefaultCacheSize
	}
	c = freecache.NewCache(cacheSize)

	// Set the default GC percent
	if percent <= 0 {
		percent = DefaultGCPercent
	}
	debug.SetGCPercent(percent)
	return c
}

// releasedLockSeconds is how long a released FreeCache lock's empty value lives
const releasedLockSeconds = 1

// writeLockFreeCache takes lockKey for secret in one atomic step
//
// One Update, under FreeCache's lock for the key, stores the secret when the key
// is missing, released (empty), or already holds the same secret, and leaves it
// alone otherwise. ttl is in seconds
func writeLockFreeCache(freeCacheClient *freecache.Cache, lockKey, secret string, ttl int64) (bool, error) { //nolint:unparam // bool is not being used yet
	secretBytes := []byte(secret)
	_, replaced, err := freeCacheClient.Update([]byte(lockKey), func(value []byte, found bool) ([]byte, bool, int) {
		if !found || len(value) == 0 || string(value) == secret {
			return secretBytes, true, int(ttl)
		}
		return nil, false, 0
	})
	if err != nil {
		return false, err
	}

	// Another secret holds the lock
	if !replaced {
		return false, cache.ErrLockMismatch
	}
	return true, nil
}

// releaseLockFreeCache releases lockKey in one atomic step when it holds secret
//
// FreeCache has no compare-and-delete, so one Update replaces the matching secret
// with an empty value that expires within releasedLockSeconds; Get reads it as
// missing, and writeLockFreeCache treats it as free. A missing or released key
// is already free
func releaseLockFreeCache(freeCacheClient *freecache.Cache, lockKey, secret string) (bool, error) {
	var heldByAnother bool
	_, _, err := freeCacheClient.Update([]byte(lockKey), func(value []byte, found bool) ([]byte, bool, int) {
		if found && string(value) == secret {
			return []byte{}, true, releasedLockSeconds
		}
		heldByAnother = found && len(value) > 0
		return nil, false, 0
	})
	if err != nil {
		return false, err
	}

	// Key found does not match the secret, do not remove
	if heldByAnother {
		return false, cache.ErrLockMismatch
	}
	return true, nil
}
