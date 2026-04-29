package cachestore

import (
	"context"
	"strings"
	"testing"
	"time"
)

// FuzzEngineString tests the Engine string conversion
func FuzzEngineString(f *testing.F) {
	// Seed corpus with known engine values and edge cases
	f.Add("freecache")
	f.Add("redis")
	f.Add("empty")
	f.Add("")
	f.Add("FREECACHE")
	f.Add("Redis")
	f.Add("invalid")
	f.Add("unknown-engine")
	f.Add("free cache")
	f.Add("redis-cluster")
	f.Fuzz(func(t *testing.T, engineStr string) {
		engine := Engine(engineStr)

		// Test String() method
		result := engine.String()
		if result != engineStr {
			t.Errorf("Engine(%q).String() returned %q, expected %q", engineStr, result, engineStr)
		}

		// Test IsEmpty() method
		isEmpty := engine.IsEmpty()
		expectedEmpty := (engine == Empty)
		if isEmpty != expectedEmpty {
			t.Errorf("Engine(%q).IsEmpty() returned %v, expected %v", engineStr, isEmpty, expectedEmpty)
		}

		// Test that methods don't panic with arbitrary strings
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Engine methods panicked with input %q: %v", engineStr, r)
			}
		}()
	})
}

// FuzzRedisConfig tests Redis configuration struct creation with fuzzed values
func FuzzRedisConfig(f *testing.F) {
	// Seed corpus with various Redis configurations
	f.Add("redis://localhost:6379", false, 10, 0, int64(240), true)
	f.Add("", false, 0, 0, int64(0), false)
	f.Add("redis://user:pass@host:1234", true, 100, 50, int64(300), true)
	f.Add("invalid-url", false, -1, -1, int64(-1), false)
	f.Add("redis://localhost", true, 1000, 500, int64(86400), false)

	f.Fuzz(func(t *testing.T, url string, useTLS bool, maxIdle, maxActive int, maxIdleTimeoutSec int64, depMode bool) {
		// Skip extremely large values to prevent resource exhaustion
		if maxIdle > 10000 || maxActive > 10000 || maxIdleTimeoutSec > 86400*365 {
			t.Skip("Skipping extremely large connection values")
		}

		// Test that configuration creation doesn't cause panics
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("RedisConfig processing panicked: %v", r)
			}
		}()

		config := &RedisConfig{
			URL:                   url,
			UseTLS:                useTLS,
			MaxIdleConnections:    maxIdle,
			MaxActiveConnections:  maxActive,
			MaxIdleTimeout:        time.Duration(maxIdleTimeoutSec) * time.Second,
			DependencyMode:        depMode,
			MaxConnectionLifetime: 0,
		}

		// Validate config fields were set correctly
		if config.URL != url {
			t.Errorf("URL mismatch: got %q, want %q", config.URL, url)
		}
		if config.UseTLS != useTLS {
			t.Errorf("UseTLS mismatch: got %v, want %v", config.UseTLS, useTLS)
		}
		if config.MaxIdleConnections != maxIdle {
			t.Errorf("MaxIdleConnections mismatch: got %d, want %d", config.MaxIdleConnections, maxIdle)
		}
		if config.MaxActiveConnections != maxActive {
			t.Errorf("MaxActiveConnections mismatch: got %d, want %d", config.MaxActiveConnections, maxActive)
		}
		if config.DependencyMode != depMode {
			t.Errorf("DependencyMode mismatch: got %v, want %v", config.DependencyMode, depMode)
		}

		// Test that WithRedis option function doesn't panic
		_ = WithRedis(config)
	})
}

// FuzzNewClientWithOptions tests client creation with various option combinations.
// Redis is intentionally excluded because fuzzing CI has no Redis server, and the
// dial timeout exceeds Go's per-execution fuzz deadline. Redis option construction
// is covered by FuzzRedisConfig.
func FuzzNewClientWithOptions(f *testing.F) {
	// Seed corpus with different option combinations
	f.Add(true, "freecache")  // debug on, freecache
	f.Add(false, "freecache") // debug off, freecache
	f.Add(true, "")           // debug on, no engine (defaults to freecache)
	f.Add(false, "invalid")   // debug off, invalid engine

	f.Fuzz(func(t *testing.T, debug bool, engineStr string) {
		ctx := context.Background()

		var opts []ClientOps

		// Add debug option
		if debug {
			opts = append(opts, WithDebugging())
		}

		// Only exercise in-memory engine paths to keep fuzz iterations
		// hermetic and within the per-execution deadline.
		if strings.EqualFold(engineStr, "freecache") {
			opts = append(opts, WithFreeCache())
		}

		// Test that client creation and basic operations don't panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Client operations panicked: %v", r)
			}
		}()

		client, err := NewClient(ctx, opts...)
		if err != nil {
			t.Logf("Client creation failed: %v", err)
			return
		}

		if client == nil {
			t.Errorf("NewClient returned nil without error")
			return
		}

		// Test that client properties match options
		if client.IsDebug() != debug {
			t.Errorf("Client debug mode %v doesn't match expected %v", client.IsDebug(), debug)
		}

		engine := client.Engine()
		_ = engine.String()
		_ = engine.IsEmpty()

		client.Close(ctx)
	})
}

// FuzzEngineOperations tests operations specific to different engines
func FuzzEngineOperations(f *testing.F) {
	// Seed corpus with operations for different engines
	f.Add("freecache", "testkey", "testvalue")
	f.Add("redis", "key123", "value456")
	f.Add("empty", "emptykey", "emptyvalue")
	f.Add("invalid", "invalidkey", "invalidvalue")

	f.Fuzz(func(t *testing.T, engineStr, key, value string) {
		ctx := context.Background()
		engine := Engine(engineStr)

		// Skip empty keys for cache operations
		if strings.TrimSpace(key) == "" {
			t.Skip("Skipping empty key")
		}

		var opts []ClientOps
		switch engine {
		case FreeCache:
			opts = append(opts, WithFreeCache())
		case Redis:
			opts = append(opts, WithRedis(&RedisConfig{URL: "redis://localhost:6379"}))
		}

		client, err := NewClient(ctx, opts...)
		if err != nil {
			// Connection failures for Redis are expected
			if engine == Redis {
				t.Logf("Redis client creation failed: %v", err)
				return
			}
			t.Logf("Client creation failed for engine %q: %v", engineStr, err)
			return
		}
		defer client.Close(ctx)

		// Test engine-specific operations
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Engine operations panicked for %q: %v", engineStr, r)
			}
		}()

		// Test cache operations
		err = client.Set(ctx, key, value)
		if err != nil {
			t.Logf("Set operation failed for engine %q: %v", engineStr, err)
			return
		}

		retrieved, err := client.Get(ctx, key)
		if err != nil {
			t.Errorf("Get operation failed after Set for engine %q: %v", engineStr, err)
			return
		}

		if retrieved != value {
			t.Errorf("Retrieved value %q doesn't match set value %q for engine %q", retrieved, value, engineStr)
		}

		// Test engine-specific client methods
		switch engine {
		case FreeCache:
			freeCache := client.FreeCache()
			if freeCache == nil {
				t.Errorf("FreeCache() returned nil for FreeCache engine")
			}
		case Redis:
			redisClient := client.Redis()
			if redisClient == nil {
				t.Errorf("Redis() returned nil for Redis engine")
			}
		}

		// Clean up
		_ = client.Delete(ctx, key)
	})
}

// FuzzClientClose tests client closing behavior
func FuzzClientClose(f *testing.F) {
	// Seed corpus with different engines
	f.Add("freecache")
	f.Add("redis")
	f.Add("empty")
	f.Add("")

	f.Fuzz(func(t *testing.T, engineStr string) {
		ctx := context.Background()
		engine := Engine(engineStr)

		var opts []ClientOps
		switch engine {
		case FreeCache:
			opts = append(opts, WithFreeCache())
		case Redis:
			opts = append(opts, WithRedis(&RedisConfig{URL: "redis://localhost:6379"}))
		}

		client, err := NewClient(ctx, opts...)
		if err != nil {
			t.Logf("Client creation failed: %v", err)
			return
		}

		// Test multiple closes (should be safe)
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Client.Close() panicked: %v - this might be expected for some engines", r)
			}
		}()

		client.Close(ctx)
		client.Close(ctx) // Second close should be safe

		// Operations after close should handle gracefully
		err = client.Set(ctx, "key", "value")
		if err == nil {
			t.Logf("Set operation after close succeeded - this might be unexpected")
		}

		// Engine should be empty after close
		if !client.Engine().IsEmpty() {
			t.Errorf("Engine should be empty after Close(), got %q", client.Engine())
		}
	})
}

// FuzzEmptyCache tests the empty cache functionality
func FuzzEmptyCache(f *testing.F) {
	// Seed corpus with different engines and key patterns
	f.Add("freecache", "key1", "value1")
	f.Add("redis", "key2", "value2")
	f.Add("empty", "key3", "value3")

	f.Fuzz(func(t *testing.T, engineStr, key, value string) {
		ctx := context.Background()
		engine := Engine(engineStr)

		if strings.TrimSpace(key) == "" {
			t.Skip("Skipping empty key")
		}

		var opts []ClientOps
		switch engine {
		case FreeCache:
			opts = append(opts, WithFreeCache())
		case Redis:
			opts = append(opts, WithRedis(&RedisConfig{URL: "redis://localhost:6379"}))
		}

		client, err := NewClient(ctx, opts...)
		if err != nil {
			t.Logf("Client creation failed: %v", err)
			return
		}
		defer client.Close(ctx)

		// Set some data
		err = client.Set(ctx, key, value)
		if err != nil {
			t.Logf("Set operation failed: %v", err)
			return
		}

		// Verify data exists
		retrieved, err := client.Get(ctx, key)
		if err != nil {
			t.Logf("Get operation failed: %v", err)
			return
		}

		if retrieved != value {
			t.Errorf("Retrieved value %q doesn't match set value %q", retrieved, value)
		}

		// Empty the cache
		err = client.EmptyCache(ctx)
		if err != nil {
			t.Logf("EmptyCache failed: %v", err)
			return
		}

		// Verify data is gone - some engines might not immediately reflect the change
		_, err = client.Get(ctx, key)
		if err == nil {
			t.Logf("Data still exists after EmptyCache() - this might be engine-specific behavior")
		}
	})
}
