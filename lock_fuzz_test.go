package cachestore

import (
	"context"
	"encoding/hex"
	"strings"
	"testing"
	"time"
)

// FuzzWriteLockWithSecret tests lock creation with custom secrets
func FuzzWriteLockWithSecret(f *testing.F) {
	// Seed corpus with various secret combinations
	f.Add("lock1", "secret1", int64(10))
	f.Add("lock2", "deadbeef", int64(60))
	f.Add("testlock", "0123456789abcdef", int64(30))
	f.Add("", "secret", int64(5))
	f.Add("lock", "", int64(5))
	f.Add("unicode-🔒", "café", int64(15))
	f.Fuzz(func(t *testing.T, lockKey, secret string, ttl int64) {
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

		// WriteLockWithSecret operation
		returnedSecret, err := client.WriteLockWithSecret(ctx, lockKey, secret, ttl)

		// Validate inputs first
		if strings.TrimSpace(lockKey) == "" || strings.TrimSpace(secret) == "" {
			if err == nil {
				t.Errorf("WriteLockWithSecret should fail with empty lockKey or secret")
			}
			return
		}

		if err != nil {
			t.Logf("WriteLockWithSecret failed for key %q, secret %q: %v", lockKey, secret, err)
			return
		}

		// The returned secret should match the input secret
		if returnedSecret != secret {
			t.Errorf("WriteLockWithSecret returned different secret: got %q, expected %q", returnedSecret, secret)
		}

		// Test duplicate lock with same secret (should succeed or fail consistently)
		_, err2 := client.WriteLockWithSecret(ctx, lockKey, secret, ttl)
		if err2 == nil {
			t.Logf("Duplicate WriteLockWithSecret succeeded - this might be engine-specific behavior")
		}

		// Test duplicate lock with different secret (should fail)
		differentSecret := secret + "different"
		_, err3 := client.WriteLockWithSecret(ctx, lockKey, differentSecret, ttl)
		if err3 == nil {
			t.Errorf("WriteLockWithSecret with different secret should have failed for existing lock")
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
	})
}

// FuzzWaitWriteLock tests the wait-and-acquire lock functionality
func FuzzWaitWriteLock(f *testing.F) {
	// Seed corpus with various wait scenarios
	f.Add("waitlock1", int64(5), int64(1))
	f.Add("waitlock2", int64(10), int64(2))
	f.Add("testlock", int64(30), int64(5))
	f.Add("", int64(10), int64(1))
	f.Add("shortlock", int64(1), int64(1))

	f.Fuzz(func(t *testing.T, lockKey string, ttl, ttw int64) {
		ctx := context.Background()

		// Skip invalid values
		if ttl <= 0 || ttl > 300 || ttw <= 0 || ttw > 10 {
			t.Skip("Skipping invalid TTL or TTW values")
		}

		// Use a timeout context to prevent hanging
		ctx, cancel := context.WithTimeout(ctx, time.Duration(ttw+2)*time.Second)
		defer cancel()

		client, err := NewClient(ctx, WithFreeCache())
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close(ctx)

		// WaitWriteLock operation
		secret, err := client.WaitWriteLock(ctx, lockKey, ttl, ttw)

		if strings.TrimSpace(lockKey) == "" {
			if err == nil {
				t.Errorf("WaitWriteLock should fail with empty lockKey")
			}
			return
		}

		if err != nil {
			// Timeout or other errors are acceptable
			t.Logf("WaitWriteLock failed or timed out for key %q: %v", lockKey, err)
			return
		}

		// Verify secret is valid hex
		if secret == "" {
			t.Errorf("WaitWriteLock returned empty secret for key %q", lockKey)
			return
		}

		if _, hexErr := hex.DecodeString(secret); hexErr != nil {
			t.Errorf("WaitWriteLock returned invalid hex secret %q: %v", secret, hexErr)
			return
		}

		// Try to release the lock
		released, err := client.ReleaseLock(ctx, lockKey, secret)
		if err != nil {
			t.Errorf("ReleaseLock failed after WaitWriteLock: %v", err)
			return
		}

		if !released {
			t.Errorf("ReleaseLock returned false after successful WaitWriteLock")
		}
	})
}

// FuzzLockExpiration tests lock behavior with various expiration times
func FuzzLockExpiration(f *testing.F) {
	// Seed corpus with short expiration times for testing
	f.Add("explock1", int64(1))
	f.Add("explock2", int64(2))
	f.Add("explock3", int64(3))

	f.Fuzz(func(t *testing.T, lockKey string, ttl int64) {
		ctx := context.Background()

		// Only test with short TTLs for expiration testing
		if ttl <= 0 || ttl > 5 {
			t.Skip("Skipping invalid or long TTL for expiration test")
		}

		if strings.TrimSpace(lockKey) == "" {
			t.Skip("Skipping empty lockKey")
		}

		client, err := NewClient(ctx, WithFreeCache())
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close(ctx)

		// Create lock
		secret, err := client.WriteLock(ctx, lockKey, ttl)
		if err != nil {
			t.Errorf("WriteLock failed: %v", err)
			return
		}

		// Verify lock exists immediately
		_, err = client.WriteLock(ctx, lockKey, ttl)
		if err == nil {
			t.Errorf("Second WriteLock should have failed for existing lock")
		}

		// Wait for expiration
		time.Sleep(time.Duration(ttl+1) * time.Second)

		// Try to create lock again (should succeed if expired)
		newSecret, err := client.WriteLock(ctx, lockKey, ttl)
		if err != nil {
			t.Logf("Lock may not have expired yet or other issue: %v", err)
			// Try to release with original secret
			_, _ = client.ReleaseLock(ctx, lockKey, secret)
			return
		}

		// Clean up
		if newSecret != "" {
			_, _ = client.ReleaseLock(ctx, lockKey, newSecret)
		}
	})
}

// FuzzReleaseLockEdgeCases tests edge cases for lock release
func FuzzReleaseLockEdgeCases(f *testing.F) {
	// Seed corpus with various edge cases
	f.Add("lock1", "validsecret")
	f.Add("lock2", "")
	f.Add("", "secret")
	f.Add("lock3", "invalidsecret")
	f.Add("nonexistentlock", "anysecret")
	f.Add("lock4", "unicode🔑")

	f.Fuzz(func(t *testing.T, lockKey, secret string) {
		ctx := context.Background()

		client, err := NewClient(ctx, WithFreeCache())
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close(ctx)

		// Test ReleaseLock with various inputs
		released, err := client.ReleaseLock(ctx, lockKey, secret)

		// Empty key or secret should fail
		if strings.TrimSpace(lockKey) == "" || strings.TrimSpace(secret) == "" {
			if err == nil && released {
				t.Errorf("ReleaseLock should not succeed with empty key or secret")
			}
			return
		}

		// For non-existent locks or wrong secrets, should return false without error
		// or return an error - both are acceptable behaviors
		if err != nil {
			t.Logf("ReleaseLock failed for key %q, secret %q: %v", lockKey, secret, err)
		} else if released {
			t.Logf("ReleaseLock succeeded for key %q - lock may have existed", lockKey)
		} else {
			t.Logf("ReleaseLock returned false for key %q - lock likely didn't exist", lockKey)
		}

		// Test multiple releases (should be idempotent or fail gracefully)
		released2, err2 := client.ReleaseLock(ctx, lockKey, secret)
		if err2 != nil {
			t.Logf("Second ReleaseLock failed: %v", err2)
		} else if released2 {
			t.Logf("Second ReleaseLock succeeded - this might be unexpected")
		}
	})
}

// FuzzLockValidation tests the lock validation logic
func FuzzLockValidation(f *testing.F) {
	// Seed corpus with various validation scenarios
	f.Add("validkey", "validsecret")
	f.Add("", "secret")
	f.Add("key", "")
	f.Add("   ", "secret")
	f.Add("key", "   ")
	f.Add("very-long-key-that-might-exceed-limits-if-any-exist-in-the-system", "secret")
	f.Add("key", "very-long-secret-that-might-exceed-limits-if-any-exist-in-the-system")

	f.Fuzz(func(t *testing.T, lockKey, secret string) {
		ctx := context.Background()

		client, err := NewClient(ctx, WithFreeCache())
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close(ctx)

		// Test WriteLockWithSecret which will validate inputs
		returnedSecret, err := client.WriteLockWithSecret(ctx, lockKey, secret, 10)

		// Check if validation works as expected
		trimmedKey := strings.TrimSpace(lockKey)
		trimmedSecret := strings.TrimSpace(secret)

		if trimmedKey == "" || trimmedSecret == "" {
			if err == nil {
				t.Logf("WriteLockWithSecret with empty key/secret succeeded - this might be engine-specific behavior")
			}
			return
		}

		if err != nil {
			t.Logf("WriteLockWithSecret failed validation for key %q, secret %q: %v", lockKey, secret, err)
			return
		}

		if returnedSecret != secret {
			t.Errorf("Returned secret %q doesn't match input secret %q", returnedSecret, secret)
		}

		// Clean up
		_, _ = client.ReleaseLock(ctx, lockKey, secret)
	})
}
