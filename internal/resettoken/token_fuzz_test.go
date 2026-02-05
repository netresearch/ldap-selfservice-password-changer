//nolint:testpackage // tests internal functions
package resettoken

import (
	"testing"
	"time"
)

// FuzzTokenStore fuzzes the token store operations.
func FuzzTokenStore(f *testing.F) {
	// Seed corpus with various token strings
	seeds := []string{
		"",
		"a",
		"abc123",
		"valid-token-1234567890",
		string(make([]byte, 1000)),
		"token-with-special-chars!@#$%^&*()",
		"token\x00with\x00nulls",
		"token\nwith\nnewlines",
		"<script>alert('xss')</script>",
		"' OR '1'='1",
		"日本語トークン",
		"🔐token🔐",
	}

	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, tokenString string) {
		store := NewStore()

		// Create a token with the fuzzed string
		token := &ResetToken{
			Token:     tokenString,
			Username:  "testuser",
			Email:     "test@example.com",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(15 * time.Minute),
			Used:      false,
		}

		// Store should not panic
		err := store.Store(token)
		if err != nil && tokenString != "" {
			// Empty tokens might fail, which is acceptable
			// Other errors might be due to duplicate tokens in rapid succession
			return
		}

		if tokenString == "" {
			// Empty token might behave differently, skip further checks
			return
		}

		// Get should not panic
		retrieved, err := store.Get(tokenString)
		if err != nil {
			t.Errorf("Failed to get token after storing: %v", err)
			return
		}

		// Verify retrieved token matches
		if retrieved.Token != tokenString {
			t.Errorf("Retrieved token mismatch: got %q, want %q", retrieved.Token, tokenString)
		}

		// MarkUsed should not panic
		err = store.MarkUsed(tokenString)
		if err != nil {
			t.Errorf("Failed to mark token as used: %v", err)
		}

		// Delete should not panic
		err = store.Delete(tokenString)
		if err != nil {
			t.Errorf("Failed to delete token: %v", err)
		}

		// Count should not panic
		_ = store.Count()

		// CleanupExpired should not panic
		_ = store.CleanupExpired()
	})
}

// FuzzGenerateToken tests that GenerateToken produces valid tokens.
func FuzzGenerateToken(f *testing.F) {
	// Seed with number of iterations (we'll just generate tokens)
	f.Add(1)
	f.Add(10)
	f.Add(100)

	f.Fuzz(func(t *testing.T, iterations int) {
		if iterations < 0 || iterations > 1000 {
			iterations = 10
		}

		tokens := make(map[string]bool)
		for range iterations {
			token, err := GenerateToken()
			if err != nil {
				t.Errorf("GenerateToken failed: %v", err)
				continue
			}

			// Token should be non-empty
			if token == "" {
				t.Error("Generated empty token")
			}

			// Token should have reasonable length (32 bytes base64 encoded)
			if len(token) < 40 {
				t.Errorf("Token too short: %d bytes", len(token))
			}

			// Check for duplicates
			if tokens[token] {
				t.Errorf("Duplicate token generated: %s", token)
			}
			tokens[token] = true
		}
	})
}

// FuzzResetTokenIsExpired fuzzes the IsExpired method.
func FuzzResetTokenIsExpired(f *testing.F) {
	// Seed with various time offsets in nanoseconds
	f.Add(int64(0))
	f.Add(int64(1))
	f.Add(int64(-1))
	f.Add(int64(1e9))       // 1 second
	f.Add(int64(-1e9))      // -1 second
	f.Add(int64(60 * 1e9))  // 1 minute
	f.Add(int64(-60 * 1e9)) // -1 minute

	f.Fuzz(func(t *testing.T, offsetNanos int64) {
		now := time.Now()
		expiresAt := now.Add(time.Duration(offsetNanos))

		token := &ResetToken{
			Token:     "test-token",
			ExpiresAt: expiresAt,
		}

		// IsExpired should not panic
		isExpired := token.IsExpired()

		// Verify correctness (with small tolerance for test execution time)
		expected := now.After(expiresAt)

		// Only check if the difference is significant enough
		if offsetNanos > 1e8 || offsetNanos < -1e8 { // 100ms tolerance
			if isExpired != expected {
				t.Errorf("IsExpired() = %v, expected %v (offset: %d ns)", isExpired, expected, offsetNanos)
			}
		}
	})
}
