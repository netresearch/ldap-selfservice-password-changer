package resettoken

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// GenerateToken generates a cryptographically secure random token.
// Returns a base64 URL-safe encoded string of 32 random bytes (256 bits).
func GenerateToken() (string, error) {
	// Generate 32 bytes of cryptographically secure random data
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}

	// Encode as base64 URL-safe string (no padding)
	token := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes)
	return token, nil
}
