package cachestore

import (
	"context"
	"sync"

	"github.com/coocood/freecache"
)

// fuzzCacheSize is the size of the shared in-memory cache used by every fuzz
// target. The production default FreeCache pre-allocates 100MB per instance
// (see DefaultCacheSize), and fuzz targets create a client on every execution
// (thousands of executions per run, across parallel workers). Allocating the
// default cache per iteration produced tens of GB/s of allocation churn that
// exhausted memory and caused CI fuzz workers to be terminated. A single, much
// smaller cache reused across executions keeps fuzzing hermetic and cheap while
// leaving the max entry size (a fraction of the cache) far above any realistic
// fuzz input.
const fuzzCacheSize = 32 * 1024 * 1024

var (
	fuzzCacheOnce sync.Once
	fuzzCache     *freecache.Cache
)

// sharedFuzzCache lazily allocates a single FreeCache shared by all fuzz
// executions within a test process. Fuzz targets and their seed corpora run
// sequentially within a process (fuzzing parallelism uses separate worker
// processes, each with its own cache), so a single instance is safe to reuse.
func sharedFuzzCache() *freecache.Cache {
	fuzzCacheOnce.Do(func() {
		fuzzCache = freecache.NewCache(fuzzCacheSize)
	})
	return fuzzCache
}

// newFuzzClient builds a cachestore client backed by the shared FreeCache,
// avoiding a fresh 100MB allocation on every fuzz execution. The cache is
// cleared first so each execution starts from clean, isolated state. Extra
// options (e.g. WithDebugging) are applied after the FreeCache connection.
func newFuzzClient(ctx context.Context, opts ...ClientOps) (ClientInterface, error) {
	cache := sharedFuzzCache()
	cache.Clear()
	return NewClient(ctx, append([]ClientOps{WithFreeCacheConnection(cache)}, opts...)...)
}
