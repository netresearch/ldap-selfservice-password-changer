package resettoken

import (
	"fmt"
	"sync"
	"time"
)

// maxCapacity is the maximum number of tokens that can be stored simultaneously.
// This prevents unbounded memory growth from DoS attacks.
const maxCapacity = 10000

// ResetToken represents a password reset token with associated metadata.
type ResetToken struct {
	Token            string    // The unique token string
	Username         string    // LDAP username
	Email            string    // User's email address
	CreatedAt        time.Time // When the token was created
	ExpiresAt        time.Time // When the token expires
	Used             bool      // Whether the token has been used
	RequiresApproval bool      // Whether admin approval is required (Phase 2)
}

// Store is a thread-safe in-memory storage for reset tokens.
type Store struct {
	mu     sync.RWMutex
	tokens map[string]*ResetToken
}

// NewStore creates and initializes a new token store.
func NewStore() *Store {
	return &Store{
		tokens: make(map[string]*ResetToken),
	}
}

// Store adds a new token to the store.
// Returns an error if a token with the same token string already exists or if capacity is reached.
// If at capacity, attempts to cleanup expired tokens before failing.
// NEVER evicts non-expired tokens (fail-closed behavior).
func (s *Store) Store(token *ResetToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicate token
	if _, exists := s.tokens[token.Token]; exists {
		return fmt.Errorf("token already exists: %s", token.Token)
	}

	// Check capacity BEFORE storing
	if len(s.tokens) >= maxCapacity {
		// Try to cleanup expired tokens to make room
		count := s.cleanupExpiredLocked()

		// If still at capacity after cleanup, fail closed
		if len(s.tokens) >= maxCapacity {
			return fmt.Errorf("token store at capacity (%d tokens), please try again later", maxCapacity)
		}

		// Cleanup freed space, log for monitoring
		_ = count
	}

	s.tokens[token.Token] = token
	return nil
}

// Get retrieves a token by its token string.
// Returns an error if the token is not found.
func (s *Store) Get(tokenString string) (*ResetToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	token, exists := s.tokens[tokenString]
	if !exists {
		return nil, fmt.Errorf("token not found: %s", tokenString)
	}

	return token, nil
}

// MarkUsed marks a token as used.
// Returns an error if the token is not found.
func (s *Store) MarkUsed(tokenString string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	token, exists := s.tokens[tokenString]
	if !exists {
		return fmt.Errorf("token not found: %s", tokenString)
	}

	token.Used = true
	return nil
}

// Delete removes a token from the store.
// Does not return an error if the token doesn't exist.
func (s *Store) Delete(tokenString string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.tokens, tokenString)
	return nil
}

// CleanupExpired removes all expired tokens from the store.
// Returns the number of tokens removed.
func (s *Store) CleanupExpired() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cleanupExpiredLocked()
}

// cleanupExpiredLocked is the internal cleanup implementation.
// MUST be called with s.mu held (Lock, not RLock).
func (s *Store) cleanupExpiredLocked() int {
	count := 0
	for tokenString, token := range s.tokens {
		if token.IsExpired() {
			delete(s.tokens, tokenString)
			count++
		}
	}

	return count
}

// IsExpired checks if the token has expired.
func (t *ResetToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// StartCleanup starts a background goroutine that periodically cleans up expired tokens.
// Returns a channel that can be used to stop the cleanup goroutine.
func (s *Store) StartCleanup(interval time.Duration) chan struct{} {
	stop := make(chan struct{})

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				count := s.CleanupExpired()
				if count > 0 {
					// Log cleanup (would integrate with logging system)
					_ = count
				}
			case <-stop:
				return
			}
		}
	}()

	return stop
}

// Count returns the current number of tokens in the store.
// Useful for monitoring and debugging.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.tokens)
}

// IsFull returns true if the store is at maximum capacity.
// Useful for monitoring and capacity alerts.
func (s *Store) IsFull() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.tokens) >= maxCapacity
}
