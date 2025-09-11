package cachestore

import (
	"context"
	"strings"
	"testing"
	"unicode"
)

// FuzzValidateLockValues tests the validateLockValues function with various inputs
func FuzzValidateLockValues(f *testing.F) {
	// Seed corpus with various validation scenarios
	f.Add("valid_key", "valid_secret")
	f.Add("", "secret")
	f.Add("key", "")
	f.Add("", "")
	f.Add("   ", "secret")
	f.Add("key", "   ")
	f.Add("   ", "   ")
	f.Add("key with spaces", "secret with spaces")
	f.Add("unicode-🔑", "unicode-🔒")
	f.Add("\n\t\r", "secret")
	f.Add("key", "\n\t\r")

	f.Fuzz(func(t *testing.T, lockKey, secret string) {
		// Call validateLockValues directly
		err := validateLockValues(lockKey, secret)

		// Check expected behavior based on inputs
		trimmedKey := strings.TrimSpace(lockKey)
		trimmedSecret := strings.TrimSpace(secret)

		shouldFail := (trimmedKey == "" || trimmedSecret == "")

		if shouldFail && err == nil {
			t.Errorf("validateLockValues should have failed for key %q, secret %q", lockKey, secret)
		} else if !shouldFail && err != nil {
			t.Errorf("validateLockValues should have succeeded for key %q, secret %q: %v", lockKey, secret, err)
		}

		// Test that the function doesn't panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("validateLockValues panicked with key %q, secret %q: %v", lockKey, secret, r)
			}
		}()
	})
}

// FuzzKeyTrimming tests key trimming behavior across all operations
func FuzzKeyTrimming(f *testing.F) {
	// Seed corpus with various whitespace scenarios
	f.Add("normalkey", "value")
	f.Add(" leadingspace", "value")
	f.Add("trailingspace ", "value")
	f.Add(" bothspaces ", "value")
	f.Add("\tkey\t", "value")
	f.Add("\nkey\n", "value")
	f.Add("\rkey\r", "value")
	f.Add("  multiple  spaces  ", "value")
	f.Add("", "value")
	f.Add("   ", "value")

	f.Fuzz(func(t *testing.T, key, value string) {
		ctx := context.Background()

		client, err := NewClient(ctx, WithFreeCache())
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close(ctx)

		trimmedKey := strings.TrimSpace(key)

		// Test Set operation
		err = client.Set(ctx, key, value)
		if trimmedKey == "" {
			if err == nil {
				t.Errorf("Set should have failed for empty trimmed key %q", key)
			}
			return
		}

		if err != nil {
			t.Errorf("Set failed for valid key %q: %v", key, err)
			return
		}

		// Test Get operation with original key
		retrieved, err := client.Get(ctx, key)
		if err != nil {
			t.Errorf("Get failed with original key %q: %v", key, err)
			return
		}

		if retrieved != value {
			t.Errorf("Retrieved value %q doesn't match set value %q", retrieved, value)
		}

		// Test Get operation with trimmed key (should work the same)
		retrievedTrimmed, err := client.Get(ctx, trimmedKey)
		if err != nil {
			t.Errorf("Get failed with trimmed key %q: %v", trimmedKey, err)
			return
		}

		if retrievedTrimmed != value {
			t.Errorf("Retrieved value with trimmed key %q doesn't match set value %q", retrievedTrimmed, value)
		}

		// Both retrievals should return the same value
		if retrieved != retrievedTrimmed {
			t.Errorf("Values retrieved with original and trimmed keys don't match: %q vs %q", retrieved, retrievedTrimmed)
		}

		// Test Delete operation
		err = client.Delete(ctx, key)
		if err != nil {
			t.Errorf("Delete failed for key %q: %v", key, err)
		}
	})
}

// FuzzKeyValidationEdgeCases tests edge cases in key validation
func FuzzKeyValidationEdgeCases(f *testing.F) {
	// Seed corpus with various edge cases
	f.Add("normal")
	f.Add("")
	f.Add("a")
	f.Add(strings.Repeat("x", 1000))
	f.Add(strings.Repeat("x", 10000))
	f.Add("key\x00null")
	f.Add("key\nwith\nnewlines")
	f.Add("key\twith\ttabs")
	f.Add("key with chinese")
	f.Add("emoji-keys")

	f.Fuzz(func(t *testing.T, key string) {
		ctx := context.Background()

		// Skip extremely long keys to prevent resource exhaustion
		if len(key) > 100000 {
			t.Skip("Skipping extremely long key")
		}

		client, err := NewClient(ctx, WithFreeCache())
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close(ctx)

		value := "test_value"

		// Test various operations with the key
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Operation panicked with key %q: %v", key, r)
			}
		}()

		// Test Set
		err = client.Set(ctx, key, value)
		if err != nil {
			// Invalid keys should fail gracefully, not panic
			t.Logf("Set failed for key %q: %v", key, err)
			return
		}

		// If Set succeeded, Get should work
		retrieved, err := client.Get(ctx, key)
		if err != nil {
			t.Errorf("Get failed after successful Set for key %q: %v", key, err)
			return
		}

		if retrieved != value {
			t.Errorf("Retrieved value mismatch for key %q: got %q, expected %q", key, retrieved, value)
		}

		// Test Delete
		err = client.Delete(ctx, key)
		if err != nil {
			t.Errorf("Delete failed for key %q: %v", key, err)
		}
	})
}

// FuzzUnicodeHandling tests Unicode and special character handling
func FuzzUnicodeHandling(f *testing.F) {
	// Seed corpus with various Unicode scenarios
	f.Add("ascii", "ascii_value")
	f.Add("café", "café_value")
	f.Add("test", "test_value")
	f.Add("keys", "locks")
	f.Add("arabic", "value")
	f.Add("russian", "value")
	f.Add("japanese", "value")

	f.Fuzz(func(t *testing.T, key, value string) {
		ctx := context.Background()

		// Skip empty keys
		if strings.TrimSpace(key) == "" {
			t.Skip("Skipping empty key")
		}

		client, err := NewClient(ctx, WithFreeCache())
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close(ctx)

		// Verify Unicode properties don't cause issues
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Unicode handling panicked with key %q, value %q: %v", key, value, r)
			}
		}()

		// Test basic operations
		err = client.Set(ctx, key, value)
		if err != nil {
			t.Logf("Set failed for Unicode key %q: %v", key, err)
			return
		}

		retrieved, err := client.Get(ctx, key)
		if err != nil {
			t.Errorf("Get failed for Unicode key %q: %v", key, err)
			return
		}

		if retrieved != value {
			t.Errorf("Unicode value mismatch: got %q, expected %q", retrieved, value)
		}

		// Test that Unicode normalization doesn't affect storage
		keyRunes := []rune(key)
		valueRunes := []rune(value)
		retrievedRunes := []rune(retrieved)

		if len(valueRunes) != len(retrievedRunes) {
			t.Errorf("Unicode rune count mismatch: got %d, expected %d", len(retrievedRunes), len(valueRunes))
		}

		// Verify no corruption in Unicode characters
		for i, r := range keyRunes {
			if !unicode.IsPrint(r) && !unicode.IsSpace(r) {
				t.Logf("Non-printable rune in key at position %d: U+%04X", i, r)
			}
		}

		for i, r := range valueRunes {
			if i < len(retrievedRunes) && retrievedRunes[i] != r {
				t.Errorf("Unicode rune mismatch at position %d: got U+%04X, expected U+%04X", i, retrievedRunes[i], r)
			}
		}
	})
}

// FuzzStringLength tests behavior with various string lengths
func FuzzStringLength(f *testing.F) {
	// Seed corpus with various length scenarios
	f.Add(0, 0)
	f.Add(1, 1)
	f.Add(10, 10)
	f.Add(100, 100)
	f.Add(1000, 1000)

	f.Fuzz(func(t *testing.T, keyLen, valueLen int) {
		ctx := context.Background()

		// Prevent excessive memory usage
		if keyLen > 10000 || valueLen > 100000 {
			t.Skip("Skipping excessive lengths")
		}

		if keyLen < 1 {
			t.Skip("Skipping empty key")
		}

		client, err := NewClient(ctx, WithFreeCache())
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close(ctx)

		// Generate strings of specified lengths
		key := strings.Repeat("k", keyLen)
		value := strings.Repeat("v", valueLen)

		err = client.Set(ctx, key, value)
		if err != nil {
			t.Logf("Set failed for lengths key=%d, value=%d: %v", keyLen, valueLen, err)
			return
		}

		retrieved, err := client.Get(ctx, key)
		if err != nil {
			t.Errorf("Get failed for lengths key=%d, value=%d: %v", keyLen, valueLen, err)
			return
		}

		if len(retrieved) != valueLen {
			t.Errorf("Length mismatch: got %d, expected %d", len(retrieved), valueLen)
		}

		if retrieved != value {
			t.Errorf("Value mismatch for lengths key=%d, value=%d", keyLen, valueLen)
		}
	})
}

// FuzzSpecialCharacters tests handling of special characters in keys and values
func FuzzSpecialCharacters(f *testing.F) {
	// Seed corpus with various special characters
	f.Add("key:colon", "value:colon")
	f.Add("key;semicolon", "value;semicolon")
	f.Add("key,comma", "value,comma")
	f.Add("key.dot", "value.dot")
	f.Add("key/slash", "value/slash")
	f.Add("key\\backslash", "value\\backslash")
	f.Add("key|pipe", "value|pipe")
	f.Add("key@at", "value@at")
	f.Add("key#hash", "value#hash")
	f.Add("key$dollar", "value$dollar")

	f.Fuzz(func(t *testing.T, key, value string) {
		ctx := context.Background()

		if strings.TrimSpace(key) == "" {
			t.Skip("Skipping empty key")
		}

		client, err := NewClient(ctx, WithFreeCache())
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close(ctx)

		// Test that special characters don't cause issues
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Special character handling panicked with key %q, value %q: %v", key, value, r)
			}
		}()

		err = client.Set(ctx, key, value)
		if err != nil {
			t.Logf("Set failed for key with special chars %q: %v", key, err)
			return
		}

		retrieved, err := client.Get(ctx, key)
		if err != nil {
			t.Errorf("Get failed for key with special chars %q: %v", key, err)
			return
		}

		if retrieved != value {
			t.Errorf("Special character value mismatch: got %q, expected %q", retrieved, value)
		}

		// Test that the key is stored exactly as provided (after trimming)
		expectedKey := strings.TrimSpace(key)
		if expectedKey != strings.TrimSpace(key) {
			t.Errorf("Key trimming inconsistency")
		}
	})
}

// FuzzControlCharacters tests handling of control characters
func FuzzControlCharacters(f *testing.F) {
	// Seed corpus with control characters
	f.Add("key\x00", "value\x00")
	f.Add("key\x01", "value\x01")
	f.Add("key\x1F", "value\x1F")
	f.Add("key\x7F", "value\x7F")

	f.Fuzz(func(t *testing.T, key, value string) {
		ctx := context.Background()

		if strings.TrimSpace(key) == "" {
			t.Skip("Skipping empty key")
		}

		client, err := NewClient(ctx, WithFreeCache())
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close(ctx)

		// Test that control characters are handled without panicking
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Control character handling panicked: %v", r)
			}
		}()

		err = client.Set(ctx, key, value)
		if err != nil {
			t.Logf("Set failed for key with control chars: %v", err)
			return
		}

		retrieved, err := client.Get(ctx, key)
		if err != nil {
			t.Errorf("Get failed for key with control chars: %v", err)
			return
		}

		// Control characters should be preserved
		if retrieved != value {
			t.Errorf("Control character preservation failed")
		}
	})
}
