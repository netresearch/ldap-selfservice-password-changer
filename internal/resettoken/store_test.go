package resettoken

import (
	"sync"
	"testing"
	"time"
)

func TestNewStore(t *testing.T) {
	store := NewStore()
	if store == nil {
		t.Fatal("NewStore() returned nil")
	}
	if store.tokens == nil {
		t.Error("NewStore() tokens map not initialized")
	}
}

func TestStoreToken(t *testing.T) {
	store := NewStore()
	token := &ResetToken{
		Token:            "test-token-123",
		Username:         "testuser",
		Email:            "test@example.com",
		CreatedAt:        time.Now(),
		ExpiresAt:        time.Now().Add(15 * time.Minute),
		Used:             false,
		RequiresApproval: false,
	}

	err := store.Store(token)
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	// Verify token was stored
	retrieved, err := store.Get(token.Token)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if retrieved.Token != token.Token {
		t.Errorf("Get() token = %s, want %s", retrieved.Token, token.Token)
	}
	if retrieved.Username != token.Username {
		t.Errorf("Get() username = %s, want %s", retrieved.Username, token.Username)
	}
}

func TestStoreTokenDuplicate(t *testing.T) {
	store := NewStore()
	token := &ResetToken{
		Token:     "duplicate-token",
		Username:  "user1",
		Email:     "user1@example.com",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}

	// Store first time - should succeed
	err := store.Store(token)
	if err != nil {
		t.Fatalf("Store() first call error = %v", err)
	}

	// Store second time with same token - should fail
	token2 := &ResetToken{
		Token:     "duplicate-token",
		Username:  "user2",
		Email:     "user2@example.com",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}
	err = store.Store(token2)
	if err == nil {
		t.Error("Store() duplicate token should return error")
	}
}

func TestGetToken(t *testing.T) {
	store := NewStore()
	token := &ResetToken{
		Token:     "get-token-test",
		Username:  "getuser",
		Email:     "get@example.com",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}

	store.Store(token)

	// Get existing token
	retrieved, err := store.Get("get-token-test")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if retrieved.Username != "getuser" {
		t.Errorf("Get() username = %s, want getuser", retrieved.Username)
	}

	// Get non-existent token
	_, err = store.Get("nonexistent-token")
	if err == nil {
		t.Error("Get() non-existent token should return error")
	}
}

func TestMarkTokenUsed(t *testing.T) {
	store := NewStore()
	token := &ResetToken{
		Token:     "mark-used-token",
		Username:  "markuser",
		Email:     "mark@example.com",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
		Used:      false,
	}

	store.Store(token)

	// Mark as used
	err := store.MarkUsed("mark-used-token")
	if err != nil {
		t.Fatalf("MarkUsed() error = %v", err)
	}

	// Verify it's marked as used
	retrieved, _ := store.Get("mark-used-token")
	if !retrieved.Used {
		t.Error("MarkUsed() token not marked as used")
	}

	// Try to mark non-existent token
	err = store.MarkUsed("nonexistent-token")
	if err == nil {
		t.Error("MarkUsed() non-existent token should return error")
	}
}

func TestDeleteToken(t *testing.T) {
	store := NewStore()
	token := &ResetToken{
		Token:     "delete-token",
		Username:  "deleteuser",
		Email:     "delete@example.com",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}

	store.Store(token)

	// Delete token
	err := store.Delete("delete-token")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify it's deleted
	_, err = store.Get("delete-token")
	if err == nil {
		t.Error("Delete() token still exists after deletion")
	}

	// Try to delete non-existent token (should not error)
	err = store.Delete("nonexistent-token")
	if err != nil {
		t.Errorf("Delete() non-existent token should not error, got %v", err)
	}
}

func TestIsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "not expired",
			expiresAt: time.Now().Add(10 * time.Minute),
			want:      false,
		},
		{
			name:      "expired",
			expiresAt: time.Now().Add(-10 * time.Minute),
			want:      true,
		},
		{
			name:      "just expired",
			expiresAt: time.Now().Add(-1 * time.Second),
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &ResetToken{
				ExpiresAt: tt.expiresAt,
			}
			if got := token.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCleanupExpiredTokens(t *testing.T) {
	store := NewStore()

	// Add expired token
	expiredToken := &ResetToken{
		Token:     "expired-token",
		Username:  "expired",
		Email:     "expired@example.com",
		CreatedAt: time.Now().Add(-20 * time.Minute),
		ExpiresAt: time.Now().Add(-5 * time.Minute),
	}
	store.Store(expiredToken)

	// Add valid token
	validToken := &ResetToken{
		Token:     "valid-token",
		Username:  "valid",
		Email:     "valid@example.com",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}
	store.Store(validToken)

	// Run cleanup
	count := store.CleanupExpired()
	if count != 1 {
		t.Errorf("CleanupExpired() removed %d tokens, want 1", count)
	}

	// Verify expired token is gone
	_, err := store.Get("expired-token")
	if err == nil {
		t.Error("CleanupExpired() expired token still exists")
	}

	// Verify valid token still exists
	_, err = store.Get("valid-token")
	if err != nil {
		t.Error("CleanupExpired() removed valid token")
	}
}

func TestConcurrentAccess(t *testing.T) {
	store := NewStore()
	var wg sync.WaitGroup
	const goroutines = 100
	const operationsPerGoroutine = 10

	// Concurrent writes
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				token := &ResetToken{
					Token:     generateUniqueToken(id, j),
					Username:  "user",
					Email:     "user@example.com",
					CreatedAt: time.Now(),
					ExpiresAt: time.Now().Add(15 * time.Minute),
				}
				store.Store(token)
			}
		}(i)
	}

	wg.Wait()

	// Verify count
	store.mu.RLock()
	count := len(store.tokens)
	store.mu.RUnlock()

	expected := goroutines * operationsPerGoroutine
	if count != expected {
		t.Errorf("Concurrent access resulted in %d tokens, want %d", count, expected)
	}
}

func TestConcurrentReadWrite(t *testing.T) {
	store := NewStore()
	var wg sync.WaitGroup

	// Pre-populate with tokens
	for i := 0; i < 100; i++ {
		token := &ResetToken{
			Token:     generateUniqueToken(0, i),
			Username:  "user",
			Email:     "user@example.com",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(15 * time.Minute),
		}
		store.Store(token)
	}

	// Concurrent readers and writers
	for i := 0; i < 50; i++ {
		// Readers
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				store.Get(generateUniqueToken(0, j))
			}
		}(i)

		// Writers
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				token := &ResetToken{
					Token:     generateUniqueToken(id+1, j),
					Username:  "user",
					Email:     "user@example.com",
					CreatedAt: time.Now(),
					ExpiresAt: time.Now().Add(15 * time.Minute),
				}
				store.Store(token)
			}
		}(i)
	}

	wg.Wait()
}

// Helper function for generating unique tokens in tests
func generateUniqueToken(id, seq int) string {
	return string(rune(id)) + "-" + string(rune(seq)) + "-test-token"
}

func BenchmarkStoreToken(b *testing.B) {
	store := NewStore()
	tokens := make([]*ResetToken, b.N)
	for i := 0; i < b.N; i++ {
		tokens[i] = &ResetToken{
			Token:     generateUniqueToken(0, i),
			Username:  "bench",
			Email:     "bench@example.com",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(15 * time.Minute),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Store(tokens[i])
	}
}

func BenchmarkGetToken(b *testing.B) {
	store := NewStore()
	token := &ResetToken{
		Token:     "bench-token",
		Username:  "bench",
		Email:     "bench@example.com",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}
	store.Store(token)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Get("bench-token")
	}
}
