package resettoken_test

import (
	"encoding/base64"
	"testing"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/resettoken"
)

func TestGenerateToken(t *testing.T) {
	token, err := resettoken.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	// Token should be non-empty.
	if token == "" {
		t.Error("GenerateToken() returned empty string")
	}

	// Token should be base64 URL-safe encoded (no padding).
	decoded, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(token)
	if err != nil {
		t.Errorf("GenerateToken() returned invalid base64: %v", err)
	}

	// Decoded token should be 32 bytes (256 bits).
	if len(decoded) != 32 {
		t.Errorf("GenerateToken() length = %d, want 32 bytes", len(decoded))
	}
}

func TestTokenUniqueness(t *testing.T) {
	const iterations = 1000
	tokens := make(map[string]bool)

	for i := range iterations {
		token, err := resettoken.GenerateToken()
		if err != nil {
			t.Fatalf("GenerateToken() error = %v", err)
		}

		if tokens[token] {
			t.Errorf("GenerateToken() collision detected at iteration %d", i)
		}
		tokens[token] = true
	}

	if len(tokens) != iterations {
		t.Errorf("GenerateToken() uniqueness check failed: got %d unique tokens, want %d", len(tokens), iterations)
	}
}

func BenchmarkGenerateToken(b *testing.B) {
	for range b.N {
		_, err := resettoken.GenerateToken()
		if err != nil {
			b.Fatalf("GenerateToken() error = %v", err)
		}
	}
}
