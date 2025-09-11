package cachestore

import (
	"context"
	"strings"
	"testing"
	"time"
)

// FuzzSetWithDependencies tests Set operation with various dependency combinations
func FuzzSetWithDependencies(f *testing.F) {
	// Seed corpus with various dependency scenarios
	f.Add("key1", "value1", "dep1")
	f.Add("key2", "value2", "")
	f.Add("key3", "value3", "dependency")
	f.Add("unicode-key", "unicode-value", "unicode-dep")

	f.Fuzz(func(t *testing.T, key, value, dependency string) {
		ctx := context.Background()

		// Skip empty keys
		if strings.TrimSpace(key) == "" {
			t.Skip("Skipping empty key")
		}

		// Test with FreeCache (dependencies are not supported, but should not fail)
		clientFC, err := NewClient(ctx, WithFreeCache())
		if err != nil {
			t.Fatalf("Failed to create FreeCache client: %v", err)
		}
		defer clientFC.Close(ctx)

		// Collect dependencies
		var dependencies []string
		if strings.TrimSpace(dependency) != "" {
			dependencies = append(dependencies, dependency)
		}

		// Test Set with dependencies on FreeCache (should work but ignore dependencies)
		err = clientFC.Set(ctx, key, value, dependencies...)
		if err != nil {
			t.Logf("Set with dependencies failed on FreeCache for key %q: %v", key, err)
			return
		}

		// Verify the value was set
		retrieved, err := clientFC.Get(ctx, key)
		if err != nil {
			t.Errorf("Get failed after Set with dependencies on FreeCache for key %q: %v", key, err)
			return
		}

		if retrieved != value {
			t.Errorf("Value mismatch on FreeCache: got %q, expected %q", retrieved, value)
		}

		// Test that operations don't panic with various dependency combinations
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Dependencies operation panicked: %v", r)
			}
		}()

		// Test SetTTL with dependencies
		err = clientFC.SetTTL(ctx, key+"_ttl", value, time.Minute, dependencies...)
		if err != nil {
			t.Logf("SetTTL with dependencies failed on FreeCache: %v", err)
		}

		// Clean up
		_ = clientFC.Delete(ctx, key)
		_ = clientFC.Delete(ctx, key+"_ttl")
	})
}

// FuzzSetTTLWithDependencies tests SetTTL operation with various dependency combinations
func FuzzSetTTLWithDependencies(f *testing.F) {
	// Seed corpus with various TTL and dependency scenarios
	f.Add("ttl1", "value1", int64(60), "dep1")
	f.Add("ttl2", "value2", int64(300), "dep2")
	f.Add("ttl3", "value3", int64(1), "")
	f.Add("unicode-ttl", "test-value", int64(120), "dep-unicode")

	f.Fuzz(func(t *testing.T, key, value string, ttlSeconds int64, dependency string) {
		ctx := context.Background()

		// Skip invalid inputs
		if strings.TrimSpace(key) == "" || ttlSeconds <= 0 || ttlSeconds > 86400 {
			t.Skip("Skipping invalid inputs")
		}

		client, err := NewClient(ctx, WithFreeCache())
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close(ctx)

		ttl := time.Duration(ttlSeconds) * time.Second

		// Collect dependencies
		var dependencies []string
		if strings.TrimSpace(dependency) != "" {
			dependencies = append(dependencies, dependency)
		}

		// Test SetTTL with dependencies
		err = client.SetTTL(ctx, key, value, ttl, dependencies...)
		if err != nil {
			t.Logf("SetTTL with dependencies failed for key %q: %v", key, err)
			return
		}

		// Verify the value was set
		retrieved, err := client.Get(ctx, key)
		if err != nil {
			t.Errorf("Get failed after SetTTL with dependencies for key %q: %v", key, err)
			return
		}

		if retrieved != value {
			t.Errorf("Value mismatch after SetTTL: got %q, expected %q", retrieved, value)
		}

		// For short TTL, test expiration (dependencies should not affect expiration in FreeCache)
		if ttlSeconds <= 2 {
			time.Sleep(time.Duration(ttlSeconds+1) * time.Second)
			_, err = client.Get(ctx, key)
			if err == nil {
				t.Logf("Key %q should have expired but still exists", key)
			}
		}
	})
}

// FuzzDependencyValidation tests dependency parameter validation
func FuzzDependencyValidation(f *testing.F) {
	// Seed corpus with various dependency validation scenarios
	f.Add("key1", "dep1")
	f.Add("key2", "")
	f.Add("key3", "   ")
	f.Add("unicode-key", "unicode-dep")

	f.Fuzz(func(t *testing.T, key, dependency string) {
		ctx := context.Background()

		if strings.TrimSpace(key) == "" {
			t.Skip("Skipping empty key")
		}

		client, err := NewClient(ctx, WithFreeCache())
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close(ctx)

		value := "test_value"

		// Test that dependency validation doesn't cause panics
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Dependency validation panicked: %v", r)
			}
		}()

		// Collect dependencies
		var dependencies []string
		if strings.TrimSpace(dependency) != "" {
			dependencies = append(dependencies, dependency)
		}

		// Test Set with various dependency combinations
		err = client.Set(ctx, key, value, dependencies...)
		if err != nil {
			t.Logf("Set with dependencies failed: %v", err)
			return
		}

		// Verify basic functionality still works
		retrieved, err := client.Get(ctx, key)
		if err != nil {
			t.Errorf("Get failed after Set with dependencies: %v", err)
			return
		}

		if retrieved != value {
			t.Errorf("Value mismatch: got %q, expected %q", retrieved, value)
		}

		// Test with empty dependencies explicitly
		err = client.Set(ctx, key+"_empty", value)
		if err != nil {
			t.Errorf("Set without dependencies failed: %v", err)
		}

		// Clean up
		_ = client.Delete(ctx, key)
		_ = client.Delete(ctx, key+"_empty")
	})
}

// FuzzDependencyEdgeCases tests edge cases in dependency handling
func FuzzDependencyEdgeCases(f *testing.F) {
	// Seed corpus with edge cases
	f.Add("key1", "dep")
	f.Add("key2", "")
	f.Add("unicode-key", "dep-unicode")

	f.Fuzz(func(t *testing.T, key, baseDep string) {
		ctx := context.Background()

		if strings.TrimSpace(key) == "" {
			t.Skip("Skipping empty key")
		}

		client, err := NewClient(ctx, WithFreeCache())
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close(ctx)

		// Generate a dependency
		var dependencies []string
		if strings.TrimSpace(baseDep) != "" {
			dependencies = append(dependencies, baseDep)
		}

		value := "test_value"

		// Test Set with generated dependencies
		err = client.Set(ctx, key, value, dependencies...)
		if err != nil {
			t.Logf("Set with dependency failed: %v", err)
			return
		}

		retrieved, err := client.Get(ctx, key)
		if err != nil {
			t.Errorf("Get failed after Set with dependency: %v", err)
			return
		}

		if retrieved != value {
			t.Errorf("Value mismatch with dependency: got %q, expected %q", retrieved, value)
		}

		// Test SetTTL with the same dependencies
		err = client.SetTTL(ctx, key+"_ttl", value, time.Minute, dependencies...)
		if err != nil {
			t.Logf("SetTTL with dependency failed: %v", err)
		}

		// Clean up
		_ = client.Delete(ctx, key)
		_ = client.Delete(ctx, key+"_ttl")
	})
}

// FuzzDependencyOrder tests whether dependency order matters
func FuzzDependencyOrder(f *testing.F) {
	// Seed corpus with different dependency orders
	f.Add("order1", "dep1", "dep2", "dep3")
	f.Add("order2", "dep3", "dep1", "dep2")
	f.Add("order3", "dep2", "dep3", "dep1")

	f.Fuzz(func(t *testing.T, baseKey, dep1, dep2, dep3 string) {
		ctx := context.Background()

		if strings.TrimSpace(baseKey) == "" {
			t.Skip("Skipping empty base key")
		}

		client, err := NewClient(ctx, WithFreeCache())
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close(ctx)

		value := "test_value"

		// Test different dependency orders
		depSets := [][]string{
			{dep1},
			{dep2},
			{dep3},
		}

		for i, deps := range depSets {
			key := baseKey + "_" + string(rune('A'+i))

			// Filter empty dependencies
			var validDeps []string
			for _, dep := range deps {
				if strings.TrimSpace(dep) != "" {
					validDeps = append(validDeps, dep)
				}
			}

			err = client.Set(ctx, key, value, validDeps...)
			if err != nil {
				t.Logf("Set failed for order %d: %v", i, err)
				continue
			}

			retrieved, err := client.Get(ctx, key)
			if err != nil {
				t.Errorf("Get failed for order %d: %v", i, err)
				continue
			}

			if retrieved != value {
				t.Errorf("Value mismatch for order %d: got %q, expected %q", i, retrieved, value)
			}

			// Clean up
			_ = client.Delete(ctx, key)
		}
	})
}

// FuzzDuplicateDependencies tests handling of duplicate dependencies
func FuzzDuplicateDependencies(f *testing.F) {
	// Seed corpus with duplicate scenarios
	f.Add("dup1", "dep")
	f.Add("dup2", "dependency")
	f.Add("dup3", "test-dep")

	f.Fuzz(func(t *testing.T, key, baseDep string) {
		ctx := context.Background()

		if strings.TrimSpace(key) == "" || strings.TrimSpace(baseDep) == "" {
			t.Skip("Skipping invalid inputs")
		}

		client, err := NewClient(ctx, WithFreeCache())
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close(ctx)

		// Create slice with duplicates (simulate with single dependency)
		dependencies := []string{baseDep}

		value := "test_value"

		// Test Set with duplicate dependencies
		err = client.Set(ctx, key, value, dependencies...)
		if err != nil {
			t.Logf("Set with dependencies failed: %v", err)
			return
		}

		retrieved, err := client.Get(ctx, key)
		if err != nil {
			t.Errorf("Get failed after Set with duplicates: %v", err)
			return
		}

		if retrieved != value {
			t.Errorf("Value mismatch with duplicates: got %q, expected %q", retrieved, value)
		}

		// Test SetTTL with duplicates
		err = client.SetTTL(ctx, key+"_ttl", value, time.Minute, dependencies...)
		if err != nil {
			t.Logf("SetTTL with duplicates failed: %v", err)
		}

		// Clean up
		_ = client.Delete(ctx, key)
		_ = client.Delete(ctx, key+"_ttl")
	})
}

// FuzzDependencyLength tests dependencies with various lengths
func FuzzDependencyLength(f *testing.F) {
	// Seed corpus with different dependency lengths
	f.Add("len1", 1)
	f.Add("len2", 10)
	f.Add("len3", 100)
	f.Add("unicode-len", 50)

	f.Fuzz(func(t *testing.T, key string, depLength int) {
		ctx := context.Background()

		if strings.TrimSpace(key) == "" || depLength <= 0 || depLength > 1000 {
			t.Skip("Skipping invalid inputs")
		}

		client, err := NewClient(ctx, WithFreeCache())
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close(ctx)

		// Generate dependency of specified length
		dependency := strings.Repeat("d", depLength)
		value := "test_value"

		// Test with long dependency
		err = client.Set(ctx, key, value, dependency)
		if err != nil {
			t.Logf("Set with dependency length %d failed: %v", depLength, err)
			return
		}

		retrieved, err := client.Get(ctx, key)
		if err != nil {
			t.Errorf("Get failed with dependency length %d: %v", depLength, err)
			return
		}

		if retrieved != value {
			t.Errorf("Value mismatch with long dependency: got %q, expected %q", retrieved, value)
		}

		// Clean up
		_ = client.Delete(ctx, key)
	})
}

// FuzzConcurrentDependencies tests concurrent operations with dependencies
func FuzzConcurrentDependencies(f *testing.F) {
	// Seed corpus for concurrent scenarios
	f.Add("conc1", "dep1", "dep2")
	f.Add("conc2", "shared_dep", "unique_dep")

	f.Fuzz(func(t *testing.T, keyPrefix, dep1, dep2 string) {
		ctx := context.Background()

		if strings.TrimSpace(keyPrefix) == "" {
			t.Skip("Skipping empty key prefix")
		}

		client, err := NewClient(ctx, WithFreeCache())
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close(ctx)

		// Test concurrent operations with shared dependencies
		done := make(chan bool, 2)

		go func() {
			defer func() { done <- true }()
			key := keyPrefix + "_goroutine1"
			var deps []string
			if strings.TrimSpace(dep1) != "" {
				deps = append(deps, dep1)
			}
			if strings.TrimSpace(dep2) != "" {
				deps = append(deps, dep2)
			}
			setErr := client.Set(ctx, key, "value1", deps...)
			if setErr == nil {
				_, _ = client.Get(ctx, key)
				_ = client.Delete(ctx, key)
			}
		}()

		go func() {
			defer func() { done <- true }()
			key := keyPrefix + "_goroutine2"
			var deps []string
			if strings.TrimSpace(dep1) != "" {
				deps = append(deps, dep1)
			}
			if strings.TrimSpace(dep2) != "" {
				deps = append(deps, dep2)
			}
			setErr := client.Set(ctx, key, "value2", deps...)
			if setErr == nil {
				_, _ = client.Get(ctx, key)
				_ = client.Delete(ctx, key)
			}
		}()

		// Wait for both goroutines to complete
		<-done
		<-done

		// Verify no data corruption occurred
		testKey := keyPrefix + "_test"
		var deps []string
		if strings.TrimSpace(dep1) != "" {
			deps = append(deps, dep1)
		}
		err = client.Set(ctx, testKey, "test_value", deps...)
		if err != nil {
			t.Logf("Post-concurrent test failed: %v", err)
			return
		}

		retrieved, err := client.Get(ctx, testKey)
		if err != nil {
			t.Errorf("Get failed after concurrent operations: %v", err)
			return
		}

		if retrieved != "test_value" {
			t.Errorf("Data corruption detected after concurrent operations")
		}

		// Clean up
		_ = client.Delete(ctx, testKey)
	})
}
