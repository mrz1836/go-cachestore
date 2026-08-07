package cachestore

import (
	"context"

	"github.com/coocood/freecache"
)

// fuzzCacheSize is the size of the in-memory cache reused across the executions
// of a single fuzz target. The production default FreeCache pre-allocates 100MB
// per instance (see DefaultCacheSize), and a fuzz target creates a client on
// every execution (thousands of executions per run). Allocating the default
// cache per execution produced tens of GB/s of allocation churn that exhausted
// memory and caused CI fuzz workers to be terminated. A single, much smaller
// cache created once per target and reused keeps fuzzing hermetic and cheap
// while leaving the max entry size (a fraction of the cache) far above any
// realistic fuzz input.
const fuzzCacheSize = 32 * 1024 * 1024

// newFuzzCache allocates a FreeCache for reuse across the executions of a single
// fuzz target. Call it once in the target's outer function (before f.Fuzz) and
// pass the result to newFuzzClient. The fuzz callback runs sequentially within a
// worker process, so one instance is safe to reuse across executions.
func newFuzzCache() *freecache.Cache {
	return freecache.NewCache(fuzzCacheSize)
}

// newFuzzClient builds a cachestore client backed by cache, avoiding a fresh
// 100MB allocation on every fuzz execution. The cache is cleared first so each
// execution starts from clean, isolated state. Extra options (e.g.
// WithDebugging) are applied after the FreeCache connection.
func newFuzzClient(ctx context.Context, cache *freecache.Cache, opts ...ClientOps) (ClientInterface, error) {
	cache.Clear()
	return NewClient(ctx, append([]ClientOps{WithFreeCacheConnection(cache)}, opts...)...)
}
