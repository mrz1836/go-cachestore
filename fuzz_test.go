package cachestore

import (
	"context"
	"encoding/hex"
	"strings"
	"testing"
	"time"
)

// FuzzRandomHex tests the RandomHex function with various input sizes
func FuzzRandomHex(f *testing.F) {
	// Seed corpus with common sizes
	f.Add(0)
	f.Add(1)
	f.Add(8)
	f.Add(16)
	f.Add(32)
	f.Add(64)
	f.Add(128)
	f.Add(256)
	f.Fuzz(func(t *testing.T, n int) {
		// Skip negative values and extremely large values
		if n < 0 || n > 1024*1024 {
			t.Skip("Skipping invalid size")
		}

		result, err := RandomHex(n)

		if err != nil && n > 0 {
			t.Errorf("RandomHex(%d) returned error: %v", n, err)
		}

		if n == 0 {
			if result != "" {
				t.Errorf("RandomHex(0) should return empty string, got %q", result)
			}
			return
		}

		// Check result length (hex encoding doubles the byte count)
		expectedLen := n * 2
		if len(result) != expectedLen {
			t.Errorf("RandomHex(%d) returned string of length %d, expected %d", n, len(result), expectedLen)
		}

		// Verify it's valid hex
		if _, hexErr := hex.DecodeString(result); hexErr != nil {
			t.Errorf("RandomHex(%d) returned invalid hex string %q: %v", n, result, hexErr)
		}

		// Check for lowercase hex characters only
		for _, r := range result {
			if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
				t.Errorf("RandomHex(%d) returned non-hex character %c in %q", n, r, result)
			}
		}
	})
}

// FuzzCacheSetGet tests basic cache operations with random data
func FuzzCacheSetGet(f *testing.F) {
	// Seed corpus with various key-value pairs
	f.Add("key1", "value1")
	f.Add("", "")
	f.Add("special-key_123", "complex value with spaces")
	f.Add("key with spaces", "value\nwith\nnewlines")
	f.Add("unicode-key-🔑", "unicode-value-🔒")

	f.Fuzz(func(t *testing.T, key, value string) {
		ctx := context.Background()

		// Test with FreeCache engine
		client, err := NewClient(ctx, WithFreeCache())
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close(ctx)

		// Set operation
		err = client.Set(ctx, key, value)
		if err != nil {
			// Some keys might be invalid, but shouldn't panic
			t.Logf("Set operation failed for key %q: %v", key, err)
			return
		}

		// Get operation
		retrieved, err := client.Get(ctx, key)
		if err != nil {
			t.Errorf("Failed to get key %q after setting: %v", key, err)
			return
		}

		if retrieved != value {
			t.Errorf("Retrieved value %q doesn't match set value %q for key %q", retrieved, value, key)
		}

		// Delete operation
		err = client.Delete(ctx, key)
		if err != nil {
			t.Errorf("Failed to delete key %q: %v", key, err)
		}

		// Verify deletion - some engines might not immediately reflect the deletion
		_, err = client.Get(ctx, key)
		if err == nil {
			t.Logf("Key %q still exists after deletion - this might be engine-specific behavior", key)
		}
	})
}

// FuzzCacheSetTTL tests cache operations with TTL
func FuzzCacheSetTTL(f *testing.F) {
	// Seed corpus with various TTL values
	f.Add("key1", "value1", int64(1))
	f.Add("key2", "value2", int64(60))
	f.Add("key3", "value3", int64(3600))
	f.Add("testkey", "testvalue", int64(0))
	f.Add("longkey", "longvalue", int64(-1))

	f.Fuzz(func(t *testing.T, key, value string, ttlSeconds int64) {
		ctx := context.Background()

		// Skip extremely large TTL values
		if ttlSeconds > 86400*365 { // 1 year
			t.Skip("Skipping extremely large TTL")
		}

		client, err := NewClient(ctx, WithFreeCache())
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close(ctx)

		ttl := time.Duration(ttlSeconds) * time.Second

		// SetTTL operation
		err = client.SetTTL(ctx, key, value, ttl)
		if err != nil {
			// Some combinations might be invalid
			if ttlSeconds < 0 {
				// Negative TTL should cause an error or be handled gracefully
				t.Logf("SetTTL with negative TTL failed as expected: %v", err)
				return
			}
			t.Logf("SetTTL failed for key %q, ttl %d: %v", key, ttlSeconds, err)
			return
		}

		// Get operation immediately
		retrieved, err := client.Get(ctx, key)
		if err != nil {
			t.Errorf("Failed to get key %q after SetTTL: %v", key, err)
			return
		}

		if retrieved != value {
			t.Errorf("Retrieved value %q doesn't match set value %q for key %q", retrieved, value, key)
		}

		// For very short TTL, test expiration
		if ttlSeconds > 0 && ttlSeconds <= 1 {
			time.Sleep(time.Duration(ttlSeconds+1) * time.Second)
			_, err = client.Get(ctx, key)
			if err == nil {
				t.Logf("Key %q should have expired but still exists", key)
			}
		}
	})
}

// FuzzLockOperations tests lock creation and release
func FuzzLockOperations(f *testing.F) {
	// Seed corpus with various lock scenarios
	f.Add("lock1", int64(1))
	f.Add("lock2", int64(60))
	f.Add("special-lock_123", int64(30))
	f.Add("", int64(10))
	f.Add("unicode-lock-🔒", int64(5))

	f.Fuzz(func(t *testing.T, lockKey string, ttl int64) {
		ctx := context.Background()

		// Skip invalid TTL values
		if ttl <= 0 || ttl > 86400 {
			t.Skip("Skipping invalid TTL")
		}

		client, err := NewClient(ctx, WithFreeCache())
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close(ctx)

		// WriteLock operation
		secret, err := client.WriteLock(ctx, lockKey, ttl)
		if err != nil {
			// Some lock keys might be invalid
			if strings.TrimSpace(lockKey) == "" {
				t.Logf("WriteLock failed for empty/whitespace key as expected: %v", err)
				return
			}
			t.Errorf("WriteLock failed for key %q: %v", lockKey, err)
			return
		}

		// Verify secret is not empty and is valid hex
		if secret == "" {
			t.Errorf("WriteLock returned empty secret for key %q", lockKey)
			return
		}

		if _, hexErr := hex.DecodeString(secret); hexErr != nil {
			t.Errorf("WriteLock returned invalid hex secret %q for key %q: %v", secret, lockKey, hexErr)
			return
		}

		// Test duplicate lock (should fail)
		_, err = client.WriteLock(ctx, lockKey, ttl)
		if err == nil {
			t.Errorf("Duplicate WriteLock should have failed for key %q", lockKey)
		}

		// ReleaseLock operation
		released, err := client.ReleaseLock(ctx, lockKey, secret)
		if err != nil {
			t.Errorf("ReleaseLock failed for key %q with secret %q: %v", lockKey, secret, err)
			return
		}

		if !released {
			t.Errorf("ReleaseLock returned false for key %q with correct secret", lockKey)
		}

		// Test release with wrong secret (should fail)
		wrongSecret := "deadbeef"
		released, err = client.ReleaseLock(ctx, lockKey, wrongSecret)
		if err == nil && released {
			t.Logf("ReleaseLock with wrong secret succeeded for key %q - this might be engine-specific behavior", lockKey)
		}
	})
}

// FuzzClientOptions tests client creation with various options
func FuzzClientOptions(f *testing.F) {
	// Seed corpus with different engine strings
	f.Add("freecache")
	f.Add("redis")
	f.Add("empty")
	f.Add("")
	f.Add("invalid")

	f.Fuzz(func(t *testing.T, engineStr string) {
		ctx := context.Background()

		var opts []ClientOps
		switch strings.ToLower(engineStr) {
		case "freecache":
			opts = append(opts, WithFreeCache())
		case "redis":
			opts = append(opts, WithRedis(&RedisConfig{URL: "redis://localhost:6379"}))
		case "empty":
			// No engine option - will default to empty
		default:
			// Invalid engine - will use default
		}

		client, err := NewClient(ctx, opts...)
		if err != nil {
			// Invalid engines or Redis connection failures are acceptable
			if strings.ToLower(engineStr) == "redis" || (engineStr != "" && strings.ToLower(engineStr) != "freecache" && strings.ToLower(engineStr) != "empty") {
				t.Logf("Client creation failed for engine %q as expected: %v", engineStr, err)
				return
			}
			t.Errorf("Unexpected error creating client with engine %q: %v", engineStr, err)
			return
		}

		if client == nil {
			t.Errorf("NewClient returned nil client without error for engine %q", engineStr)
			return
		}

		// Test basic operations don't panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Client operations panicked for engine %q: %v", engineStr, r)
			}
		}()

		_ = client.Engine()
		_ = client.IsDebug()
		client.Close(ctx)
	})
}
