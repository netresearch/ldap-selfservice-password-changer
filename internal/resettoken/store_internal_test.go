package resettoken

import (
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewStore(t *testing.T) {
	store := NewStore()
	if store == nil {
		t.Fatal("NewStore() returned nil")
		return
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
	require.NoError(t, err)

	// Verify token was stored
	retrieved, err := store.Get(token.Token)
	require.NoError(t, err)
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
	require.NoError(t, err)

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

	err := store.Store(token)
	require.NoError(t, err)

	// Get existing token
	retrieved, err := store.Get("get-token-test")
	require.NoError(t, err)
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

	err := store.Store(token)
	require.NoError(t, err)

	// Mark as used
	err = store.MarkUsed("mark-used-token")
	require.NoError(t, err)

	// Verify it's marked as used
	retrieved, err := store.Get("mark-used-token")
	require.NoError(t, err)
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

	err := store.Store(token)
	require.NoError(t, err)

	// Delete token
	err = store.Delete("delete-token")
	require.NoError(t, err)

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
	err := store.Store(expiredToken)
	require.NoError(t, err)

	// Add valid token
	validToken := &ResetToken{
		Token:     "valid-token",
		Username:  "valid",
		Email:     "valid@example.com",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}
	err = store.Store(validToken)
	require.NoError(t, err)

	// Run cleanup
	count := store.CleanupExpired()
	if count != 1 {
		t.Errorf("CleanupExpired() removed %d tokens, want 1", count)
	}

	// Verify expired token is gone
	_, err = store.Get("expired-token")
	if err == nil {
		t.Error("CleanupExpired() expired token still exists")
	}

	// Verify valid token still exists
	_, err = store.Get("valid-token")
	require.NoError(t, err)
}

func TestConcurrentAccess(t *testing.T) {
	store := NewStore()
	var wg sync.WaitGroup
	const goroutines = 100
	const operationsPerGoroutine = 10

	// Concurrent writes
	for i := range goroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := range operationsPerGoroutine {
				token := &ResetToken{
					Token:     generateUniqueToken(id, j),
					Username:  "user",
					Email:     "user@example.com",
					CreatedAt: time.Now(),
					ExpiresAt: time.Now().Add(15 * time.Minute),
				}
				if err := store.Store(token); err != nil {
					t.Errorf("Failed to store token: %v", err)
				}
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
	for i := range 100 {
		token := &ResetToken{
			Token:     generateUniqueToken(0, i),
			Username:  "user",
			Email:     "user@example.com",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(15 * time.Minute),
		}
		if err := store.Store(token); err != nil {
			t.Fatalf("Failed to pre-populate store: %v", err)
		}
	}

	// Concurrent readers and writers
	for i := range 50 {
		// Readers
		wg.Add(1)
		go func(_ int) {
			defer wg.Done()
			for j := range 100 {
				if _, err := store.Get(generateUniqueToken(0, j)); err != nil {
					t.Errorf("Failed to get token: %v", err)
				}
			}
		}(i)

		// Writers
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := range 10 {
				token := &ResetToken{
					Token:     generateUniqueToken(id+1, j),
					Username:  "user",
					Email:     "user@example.com",
					CreatedAt: time.Now(),
					ExpiresAt: time.Now().Add(15 * time.Minute),
				}
				if err := store.Store(token); err != nil {
					t.Errorf("Failed to store token: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()
}

// Helper function for generating unique tokens in tests.
func generateUniqueToken(id, seq int) string {
	return strconv.Itoa(id) + "-" + strconv.Itoa(seq) + "-test-token"
}

func TestCapacityEnforcement(t *testing.T) {
	store := NewStore()

	// Fill store to capacity (10,000 tokens)
	for i := range maxCapacity {
		token := &ResetToken{
			Token:     generateUniqueToken(0, i),
			Username:  "user",
			Email:     "user@example.com",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(15 * time.Minute),
		}
		err := store.Store(token)
		require.NoError(t, err, "Store() failed at token %d", i)
	}

	// Verify store is at capacity
	if store.Count() != maxCapacity {
		t.Errorf("Store count = %d, want %d", store.Count(), maxCapacity)
	}

	// Next store should fail with capacity error
	token := &ResetToken{
		Token:     "over-capacity-token",
		Username:  "user",
		Email:     "user@example.com",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}
	err := store.Store(token)
	if err == nil {
		t.Error("Store() should fail when at capacity")
	}
	if err != nil && !contains(err.Error(), "at capacity") {
		t.Errorf("Error should mention capacity, got: %v", err)
	}
}

func TestCapacityCleanupMakesRoom(t *testing.T) {
	store := NewStore()

	// Fill store with mix of expired and valid tokens
	expiredCount := 100
	validCount := maxCapacity - expiredCount

	// Add expired tokens
	for i := range expiredCount {
		token := &ResetToken{
			Token:     generateUniqueToken(1, i),
			Username:  "expired",
			Email:     "expired@example.com",
			CreatedAt: time.Now().Add(-20 * time.Minute),
			ExpiresAt: time.Now().Add(-5 * time.Minute), // Expired
		}
		err := store.Store(token)
		require.NoError(t, err)
	}

	// Add valid tokens to reach capacity
	for i := range validCount {
		token := &ResetToken{
			Token:     generateUniqueToken(2, i),
			Username:  "valid",
			Email:     "valid@example.com",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(15 * time.Minute),
		}
		err := store.Store(token)
		require.NoError(t, err)
	}

	// Verify at capacity
	if store.Count() != maxCapacity {
		t.Errorf("Store count = %d, want %d", store.Count(), maxCapacity)
	}

	// Try to store new token - should trigger cleanup and succeed
	newToken := &ResetToken{
		Token:     "new-after-cleanup",
		Username:  "newuser",
		Email:     "new@example.com",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}
	err := store.Store(newToken)
	require.NoError(t, err, "Store() should succeed after cleanup")

	// Verify expired tokens were removed
	count := store.Count()
	if count > maxCapacity {
		t.Errorf("Store count after cleanup = %d, should be <= %d", count, maxCapacity)
	}

	// Verify new token was stored
	retrieved, err := store.Get("new-after-cleanup")
	require.NoError(t, err, "New token should be stored after cleanup")
	if retrieved == nil || retrieved.Username != "newuser" {
		t.Error("Retrieved token doesn't match stored token")
	}
}

func TestCapacityFailClosedBehavior(t *testing.T) {
	store := NewStore()

	// Fill store with all valid tokens (no expired tokens to cleanup)
	for i := range maxCapacity {
		token := &ResetToken{
			Token:     generateUniqueToken(0, i),
			Username:  "user",
			Email:     "user@example.com",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(15 * time.Minute),
		}
		err := store.Store(token)
		require.NoError(t, err)
	}

	// Try to store new token - should fail because cleanup won't free space
	newToken := &ResetToken{
		Token:     "fail-closed-token",
		Username:  "newuser",
		Email:     "new@example.com",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}
	err := store.Store(newToken)
	if err == nil {
		t.Error("Store() should fail when at capacity with no expired tokens")
	}

	// Verify no valid tokens were evicted
	if store.Count() != maxCapacity {
		t.Errorf("Store count = %d, want %d (no eviction should occur)", store.Count(), maxCapacity)
	}

	// Verify new token was NOT stored
	_, err = store.Get("fail-closed-token")
	if err == nil {
		t.Error("Token should not be stored when capacity is full")
	}
}

func TestCapacityConcurrentAccessAtCapacity(t *testing.T) {
	store := NewStore()

	// Fill store to near capacity
	for i := range maxCapacity - 100 {
		token := &ResetToken{
			Token:     generateUniqueToken(0, i),
			Username:  "user",
			Email:     "user@example.com",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(15 * time.Minute),
		}
		err := store.Store(token)
		require.NoError(t, err)
	}

	// Concurrent attempts to fill remaining capacity
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex
	const goroutines = 200

	for i := range goroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			token := &ResetToken{
				Token:     generateUniqueToken(id+1, 0),
				Username:  "concurrent",
				Email:     "concurrent@example.com",
				CreatedAt: time.Now(),
				ExpiresAt: time.Now().Add(15 * time.Minute),
			}
			err := store.Store(token)
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// Verify store didn't exceed capacity
	finalCount := store.Count()
	if finalCount > maxCapacity {
		t.Errorf("Store exceeded capacity: %d > %d", finalCount, maxCapacity)
	}

	// Verify some requests succeeded and some failed (capacity enforced)
	if successCount == 0 {
		t.Error("No concurrent requests succeeded")
	}
	if successCount == goroutines {
		t.Error("All concurrent requests succeeded (capacity not enforced)")
	}
}

func TestIsFull(t *testing.T) {
	store := NewStore()

	// Initially not full
	if store.IsFull() {
		t.Error("Empty store should not be full")
	}

	// Add tokens
	for i := range maxCapacity / 2 {
		token := &ResetToken{
			Token:     generateUniqueToken(0, i),
			Username:  "user",
			Email:     "user@example.com",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(15 * time.Minute),
		}
		err := store.Store(token)
		require.NoError(t, err)
	}

	// Still not full
	if store.IsFull() {
		t.Error("Half-full store should not be full")
	}

	// Fill to capacity
	for i := maxCapacity / 2; i < maxCapacity; i++ {
		token := &ResetToken{
			Token:     generateUniqueToken(0, i),
			Username:  "user",
			Email:     "user@example.com",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(15 * time.Minute),
		}
		err := store.Store(token)
		require.NoError(t, err)
	}

	// Now full
	if !store.IsFull() {
		t.Error("Store at capacity should be full")
	}
}

// Helper function for string contains check.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func BenchmarkStoreToken(b *testing.B) {
	store := NewStore()
	tokens := make([]*ResetToken, b.N)
	for i := range b.N {
		tokens[i] = &ResetToken{
			Token:     generateUniqueToken(0, i),
			Username:  "bench",
			Email:     "bench@example.com",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(15 * time.Minute),
		}
	}

	b.ResetTimer()
	for i := range b.N {
		if err := store.Store(tokens[i]); err != nil {
			b.Fatalf("Failed to store token: %v", err)
		}
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
	if err := store.Store(token); err != nil {
		b.Fatalf("Failed to store benchmark token: %v", err)
	}

	b.ResetTimer()
	for range b.N {
		if _, err := store.Get("bench-token"); err != nil {
			b.Fatalf("Failed to get token: %v", err)
		}
	}
}

// TestStartCleanupGoroutine tests the StartCleanup function lifecycle.
func TestStartCleanupGoroutine(t *testing.T) {
	store := NewStore()

	// Add an expired token
	expiredToken := &ResetToken{
		Token:     "cleanup-test-expired",
		Username:  "expired",
		Email:     "expired@example.com",
		CreatedAt: time.Now().Add(-20 * time.Minute),
		ExpiresAt: time.Now().Add(-5 * time.Minute),
	}
	err := store.Store(expiredToken)
	require.NoError(t, err)

	// Add a valid token
	validToken := &ResetToken{
		Token:     "cleanup-test-valid",
		Username:  "valid",
		Email:     "valid@example.com",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}
	err = store.Store(validToken)
	require.NoError(t, err)

	// Verify both tokens exist
	if store.Count() != 2 {
		t.Errorf("Expected 2 tokens, got %d", store.Count())
	}

	// Start cleanup with very short interval for testing
	stop := store.StartCleanup(50 * time.Millisecond)

	// Wait for at least one cleanup cycle
	time.Sleep(150 * time.Millisecond)

	// Verify expired token was cleaned up
	if store.Count() != 1 {
		t.Errorf("Expected 1 token after cleanup, got %d", store.Count())
	}

	// Verify valid token still exists
	_, err = store.Get("cleanup-test-valid")
	require.NoError(t, err, "Valid token should still exist")

	// Stop the cleanup goroutine
	close(stop)

	// Give goroutine time to exit
	time.Sleep(100 * time.Millisecond)
}

// TestStartCleanupStop tests that the cleanup goroutine stops correctly.
func TestStartCleanupStop(t *testing.T) {
	store := NewStore()

	// Start cleanup
	stop := store.StartCleanup(10 * time.Millisecond)

	// Stop immediately
	close(stop)

	// Add an expired token after stopping
	expiredToken := &ResetToken{
		Token:     "post-stop-expired",
		Username:  "expired",
		Email:     "expired@example.com",
		CreatedAt: time.Now().Add(-20 * time.Minute),
		ExpiresAt: time.Now().Add(-5 * time.Minute),
	}
	err := store.Store(expiredToken)
	require.NoError(t, err)

	// Wait longer than cleanup interval
	time.Sleep(50 * time.Millisecond)

	// Expired token should still exist because cleanup was stopped
	if store.Count() != 1 {
		t.Errorf("Expected 1 token (cleanup should be stopped), got %d", store.Count())
	}
}

// TestStartCleanupMultiple tests starting multiple cleanup goroutines.
func TestStartCleanupMultiple(t *testing.T) {
	store := NewStore()

	// Start multiple cleanup goroutines (not recommended but should be safe)
	stop1 := store.StartCleanup(50 * time.Millisecond)
	stop2 := store.StartCleanup(50 * time.Millisecond)

	// Add an expired token
	expiredToken := &ResetToken{
		Token:     "multi-cleanup-expired",
		Username:  "expired",
		Email:     "expired@example.com",
		CreatedAt: time.Now().Add(-20 * time.Minute),
		ExpiresAt: time.Now().Add(-5 * time.Minute),
	}
	err := store.Store(expiredToken)
	require.NoError(t, err)

	// Wait for cleanup
	time.Sleep(150 * time.Millisecond)

	// Token should be cleaned up
	if store.Count() != 0 {
		t.Errorf("Expected 0 tokens, got %d", store.Count())
	}

	// Stop both
	close(stop1)
	close(stop2)

	// Give goroutines time to exit
	time.Sleep(100 * time.Millisecond)
}

// TestStartCleanupNoExpiredTokens tests cleanup when no tokens are expired.
func TestStartCleanupNoExpiredTokens(t *testing.T) {
	store := NewStore()

	// Add only valid tokens
	for i := range 5 {
		token := &ResetToken{
			Token:     generateUniqueToken(0, i),
			Username:  "valid",
			Email:     "valid@example.com",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(15 * time.Minute),
		}
		err := store.Store(token)
		require.NoError(t, err)
	}

	// Start cleanup
	stop := store.StartCleanup(50 * time.Millisecond)

	// Wait for at least one cleanup cycle
	time.Sleep(150 * time.Millisecond)

	// All tokens should still exist
	if store.Count() != 5 {
		t.Errorf("Expected 5 tokens, got %d", store.Count())
	}

	close(stop)
}

// =============================================================================
// Mock Clock Tests - Instant time-based testing without real waits
// =============================================================================

// TestIsExpiredWithMockClock tests token expiration using a mock clock.
// This allows instant testing without waiting for real time to pass.
func TestIsExpiredWithMockClock(t *testing.T) {
	baseTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	clock := NewMockClock(baseTime)
	cleanup := setTestClock(clock)
	defer cleanup()

	token := &ResetToken{
		Token:     "test-token",
		Username:  "testuser",
		Email:     "test@example.com",
		CreatedAt: baseTime,
		ExpiresAt: baseTime.Add(15 * time.Minute),
	}

	// Token should not be expired initially
	if token.IsExpired() {
		t.Error("token should not be expired at creation time")
	}

	// Advance clock by 14 minutes - still valid
	clock.Advance(14 * time.Minute)
	if token.IsExpired() {
		t.Error("token should not be expired before expiry time")
	}

	// Advance clock by 2 more minutes (total 16 minutes) - now expired
	clock.Advance(2 * time.Minute)
	if !token.IsExpired() {
		t.Error("token should be expired after expiry time")
	}
}

// TestIsExpiredExactBoundary tests expiration at exact boundary.
func TestIsExpiredExactBoundary(t *testing.T) {
	baseTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	clock := NewMockClock(baseTime)
	cleanup := setTestClock(clock)
	defer cleanup()

	token := &ResetToken{
		Token:     "boundary-token",
		Username:  "testuser",
		Email:     "test@example.com",
		CreatedAt: baseTime,
		ExpiresAt: baseTime.Add(15 * time.Minute),
	}

	// At exact expiry time - not expired (After, not Equal)
	clock.Set(baseTime.Add(15 * time.Minute))
	if token.IsExpired() {
		t.Error("token should not be expired at exact expiry time")
	}

	// 1 nanosecond after - expired
	clock.Advance(1 * time.Nanosecond)
	if !token.IsExpired() {
		t.Error("token should be expired 1ns after expiry time")
	}
}

// TestCleanupExpiredWithMockClock tests cleanup using mock clock for instant expiration.
func TestCleanupExpiredWithMockClock(t *testing.T) {
	baseTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	clock := NewMockClock(baseTime)
	cleanup := setTestClock(clock)
	defer cleanup()

	store := NewStore()

	// Create tokens with different expiry times
	token1 := &ResetToken{
		Token:     "expires-5min",
		Username:  "user1",
		Email:     "user1@example.com",
		CreatedAt: baseTime,
		ExpiresAt: baseTime.Add(5 * time.Minute),
	}
	token2 := &ResetToken{
		Token:     "expires-10min",
		Username:  "user2",
		Email:     "user2@example.com",
		CreatedAt: baseTime,
		ExpiresAt: baseTime.Add(10 * time.Minute),
	}
	token3 := &ResetToken{
		Token:     "expires-20min",
		Username:  "user3",
		Email:     "user3@example.com",
		CreatedAt: baseTime,
		ExpiresAt: baseTime.Add(20 * time.Minute),
	}

	require.NoError(t, store.Store(token1))
	require.NoError(t, store.Store(token2))
	require.NoError(t, store.Store(token3))
	if store.Count() != 3 {
		t.Errorf("Expected 3 tokens, got %d", store.Count())
	}

	// Advance 6 minutes - token1 should expire
	clock.Advance(6 * time.Minute)
	count := store.CleanupExpired()
	if count != 1 {
		t.Errorf("Should cleanup 1 token, got %d", count)
	}
	if store.Count() != 2 {
		t.Errorf("Expected 2 tokens, got %d", store.Count())
	}

	// Advance 5 more minutes (total 11) - token2 should expire
	clock.Advance(5 * time.Minute)
	count = store.CleanupExpired()
	if count != 1 {
		t.Errorf("Should cleanup 1 token, got %d", count)
	}
	if store.Count() != 1 {
		t.Errorf("Expected 1 token, got %d", store.Count())
	}

	// Verify token3 still exists
	_, err := store.Get("expires-20min")
	require.NoError(t, err, "token3 should still exist")
}

// TestRealClockNow verifies RealClock returns current time.
func TestRealClockNow(t *testing.T) {
	clock := RealClock{}
	before := time.Now()
	now := clock.Now()
	after := time.Now()

	if now.Before(before) {
		t.Error("clock.Now() should not be before time.Now()")
	}
	if now.After(after) {
		t.Error("clock.Now() should not be after time.Now()")
	}
}

// TestMockClockAdvance tests the Advance method of MockClock.
func TestMockClockAdvance(t *testing.T) {
	baseTime := time.Date(2026, 6, 15, 10, 30, 0, 0, time.UTC)
	clock := NewMockClock(baseTime)

	if !clock.Now().Equal(baseTime) {
		t.Errorf("Initial time should be %v, got %v", baseTime, clock.Now())
	}

	clock.Advance(1 * time.Hour)
	expected := baseTime.Add(1 * time.Hour)
	if !clock.Now().Equal(expected) {
		t.Errorf("After advancing 1h, time should be %v, got %v", expected, clock.Now())
	}

	clock.Advance(-30 * time.Minute)
	expected = expected.Add(-30 * time.Minute)
	if !clock.Now().Equal(expected) {
		t.Errorf("After going back 30min, time should be %v, got %v", expected, clock.Now())
	}
}

// TestMockClockSet tests the Set method of MockClock.
func TestMockClockSet(t *testing.T) {
	clock := NewMockClock(time.Now())

	newTime := time.Date(2030, 12, 31, 23, 59, 59, 0, time.UTC)
	clock.Set(newTime)

	if !clock.Now().Equal(newTime) {
		t.Errorf("After Set, time should be %v, got %v", newTime, clock.Now())
	}
}
