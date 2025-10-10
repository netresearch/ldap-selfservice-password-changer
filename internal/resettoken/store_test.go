package resettoken_test

import (
	"testing"
	"time"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/resettoken"
)

func TestNewStore(t *testing.T) {
	store := resettoken.NewStore()
	if store == nil {
		t.Fatal("NewStore() returned nil")
	}
}

func TestStoreAndGet(t *testing.T) {
	store := resettoken.NewStore()
	token := &resettoken.ResetToken{
		Token:     "test-token-123",
		Email:     "test@example.com",
		Username:  "testuser",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}

	err := store.Store(token)
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	retrieved, err := store.Get("test-token-123")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrieved.Email != token.Email {
		t.Errorf("Email = %q, want %q", retrieved.Email, token.Email)
	}
}

func TestGetNonExistentToken(t *testing.T) {
	store := resettoken.NewStore()

	_, err := store.Get("nonexistent-token")
	if err == nil {
		t.Error("Get() should return error for nonexistent token")
	}
}

func TestDeleteToken(t *testing.T) {
	store := resettoken.NewStore()
	token := &resettoken.ResetToken{
		Token:     "test-token-456",
		Email:     "test@example.com",
		Username:  "testuser",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}

	if err := store.Store(token); err != nil {
		t.Fatalf("Failed to store token: %v", err)
	}
	if err := store.Delete("test-token-456"); err != nil {
		t.Fatalf("Failed to delete token: %v", err)
	}

	_, err := store.Get("test-token-456")
	if err == nil {
		t.Error("Get() should return error after Delete()")
	}
}

func TestCount(t *testing.T) {
	store := resettoken.NewStore()

	if store.Count() != 0 {
		t.Errorf("Initial Count() = %d, want 0", store.Count())
	}

	token1 := &resettoken.ResetToken{
		Token:     "token1",
		Email:     "test1@example.com",
		Username:  "user1",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}
	token2 := &resettoken.ResetToken{
		Token:     "token2",
		Email:     "test2@example.com",
		Username:  "user2",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}

	if err := store.Store(token1); err != nil {
		t.Fatalf("Failed to store token1: %v", err)
	}
	if err := store.Store(token2); err != nil {
		t.Fatalf("Failed to store token2: %v", err)
	}

	if store.Count() != 2 {
		t.Errorf("Count() = %d, want 2", store.Count())
	}
}

func TestCleanupExpired(t *testing.T) {
	store := resettoken.NewStore()

	// Add expired token.
	expiredToken := &resettoken.ResetToken{
		Token:     "expired-token",
		Email:     "test@example.com",
		Username:  "testuser",
		CreatedAt: time.Now().Add(-20 * time.Minute),
		ExpiresAt: time.Now().Add(-5 * time.Minute),
	}
	if err := store.Store(expiredToken); err != nil {
		t.Fatalf("Failed to store expired token: %v", err)
	}

	// Add valid token.
	validToken := &resettoken.ResetToken{
		Token:     "valid-token",
		Email:     "test2@example.com",
		Username:  "testuser2",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}
	if err := store.Store(validToken); err != nil {
		t.Fatalf("Failed to store valid token: %v", err)
	}

	// Cleanup should remove expired token.
	count := store.CleanupExpired()
	if count != 1 {
		t.Errorf("CleanupExpired() = %d, want 1", count)
	}

	// Valid token should still exist.
	_, err := store.Get("valid-token")
	if err != nil {
		t.Error("Valid token should still exist after cleanup")
	}

	// Expired token should be gone.
	_, err = store.Get("expired-token")
	if err == nil {
		t.Error("Expired token should be removed after cleanup")
	}
}
