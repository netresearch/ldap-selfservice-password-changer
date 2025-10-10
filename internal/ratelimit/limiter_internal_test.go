package ratelimit

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestNewLimiter(t *testing.T) {
	limiter := NewLimiter(3, 60*time.Minute)
	if limiter == nil {
		t.Fatal("NewLimiter() returned nil")
		return
	}
	if limiter.maxRequests != 3 {
		t.Errorf("maxRequests = %d, want 3", limiter.maxRequests)
	}
	if limiter.window != 60*time.Minute {
		t.Errorf("window = %v, want 60m", limiter.window)
	}
}

func TestAllowRequest(t *testing.T) {
	limiter := NewLimiter(3, 60*time.Minute)
	identifier := "user@example.com"

	// First 3 requests should be allowed
	for i := 1; i <= 3; i++ {
		allowed := limiter.AllowRequest(identifier)
		if !allowed {
			t.Errorf("Request %d should be allowed", i)
		}
	}

	// 4th request should be blocked
	allowed := limiter.AllowRequest(identifier)
	if allowed {
		t.Error("Request 4 should be blocked (rate limit exceeded)")
	}
}

func TestAllowRequestDifferentUsers(t *testing.T) {
	limiter := NewLimiter(3, 60*time.Minute)

	// User 1 makes 3 requests
	for range 3 {
		limiter.AllowRequest("user1@example.com")
	}

	// User 2 should still be allowed
	allowed := limiter.AllowRequest("user2@example.com")
	if !allowed {
		t.Error("Different user should not be rate limited")
	}
}

func TestSlidingWindow(t *testing.T) {
	limiter := NewLimiter(2, 100*time.Millisecond)
	identifier := "user@example.com"

	// Make 2 requests (limit reached)
	limiter.AllowRequest(identifier)
	limiter.AllowRequest(identifier)

	// 3rd request should be blocked
	allowed := limiter.AllowRequest(identifier)
	if allowed {
		t.Error("Request should be blocked immediately")
	}

	// Wait for window to pass
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	allowed = limiter.AllowRequest(identifier)
	if !allowed {
		t.Error("Request should be allowed after window expired")
	}
}

func TestCleanupExpiredEntries(t *testing.T) {
	limiter := NewLimiter(3, 100*time.Millisecond)

	// Make requests from multiple users
	limiter.AllowRequest("user1@example.com")
	limiter.AllowRequest("user2@example.com")
	limiter.AllowRequest("user3@example.com")

	// Verify entries exist
	limiter.mu.RLock()
	initialCount := len(limiter.entries)
	limiter.mu.RUnlock()
	if initialCount != 3 {
		t.Errorf("Expected 3 entries, got %d", initialCount)
	}

	// Wait for entries to expire
	time.Sleep(150 * time.Millisecond)

	// Run cleanup
	count := limiter.CleanupExpired()
	if count != 3 {
		t.Errorf("CleanupExpired() removed %d entries, want 3", count)
	}

	// Verify entries are gone
	limiter.mu.RLock()
	finalCount := len(limiter.entries)
	limiter.mu.RUnlock()
	if finalCount != 0 {
		t.Errorf("Expected 0 entries after cleanup, got %d", finalCount)
	}
}

func TestConcurrentAccess(t *testing.T) {
	limiter := NewLimiter(100, 60*time.Minute)
	var wg sync.WaitGroup
	const goroutines = 50
	const requestsPerGoroutine = 2

	// Concurrent requests from same user
	identifier := "concurrent@example.com"
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range requestsPerGoroutine {
				limiter.AllowRequest(identifier)
			}
		}()
	}

	wg.Wait()

	// Verify request count is accurate (thread-safe)
	limiter.mu.RLock()
	entry, exists := limiter.entries[identifier]
	limiter.mu.RUnlock()

	if !exists {
		t.Fatal("Entry should exist for concurrent user")
	}

	expected := goroutines * requestsPerGoroutine
	if len(entry.timestamps) != expected {
		t.Errorf("Request count = %d, want %d", len(entry.timestamps), expected)
	}
}

func TestRateLimitReset(t *testing.T) {
	limiter := NewLimiter(1, 50*time.Millisecond)
	identifier := "reset@example.com"

	// Make first request (allowed)
	if !limiter.AllowRequest(identifier) {
		t.Error("First request should be allowed")
	}

	// Second request blocked
	if limiter.AllowRequest(identifier) {
		t.Error("Second request should be blocked")
	}

	// Wait for window
	time.Sleep(60 * time.Millisecond)

	// Should be allowed again
	if !limiter.AllowRequest(identifier) {
		t.Error("Request should be allowed after window reset")
	}
}

func TestCount(t *testing.T) {
	limiter := NewLimiter(3, 60*time.Minute)

	if limiter.Count() != 0 {
		t.Errorf("Initial count = %d, want 0", limiter.Count())
	}

	limiter.AllowRequest("user1@example.com")
	limiter.AllowRequest("user2@example.com")

	if limiter.Count() != 2 {
		t.Errorf("Count after 2 users = %d, want 2", limiter.Count())
	}
}

func TestCapacityEnforcement(t *testing.T) {
	// Create limiter with small capacity for testing
	limiter := NewLimiterWithCapacity(3, 60*time.Minute, 100)

	// Fill to capacity
	for i := range 100 {
		identifier := fmt.Sprintf("user%d@example.com", i)
		allowed := limiter.AllowRequest(identifier)
		if !allowed {
			t.Errorf("Request %d should be allowed when filling to capacity", i)
		}
	}

	// Verify at capacity
	if limiter.Count() != 100 {
		t.Errorf("Count = %d, want 100", limiter.Count())
	}

	// Next request with new identifier should be denied
	allowed := limiter.AllowRequest("new@example.com")
	if allowed {
		t.Error("Request should be denied when at capacity")
	}

	// Verify capacity not exceeded
	if limiter.Count() > 100 {
		t.Errorf("Count exceeded capacity: %d > 100", limiter.Count())
	}
}

func TestCapacityCleanupMakesRoom(t *testing.T) {
	// Create limiter with short window
	limiter := NewLimiterWithCapacity(1, 50*time.Millisecond, 50)

	// Fill to capacity with requests that will expire soon
	for i := range 50 {
		identifier := fmt.Sprintf("user%d@example.com", i)
		limiter.AllowRequest(identifier)
	}

	// Wait for entries to expire
	time.Sleep(60 * time.Millisecond)

	// New request should trigger cleanup and succeed
	allowed := limiter.AllowRequest("new@example.com")
	if !allowed {
		t.Error("Request should be allowed after expired entries are cleaned up")
	}

	// Verify cleanup occurred
	count := limiter.Count()
	if count > 10 {
		t.Errorf("Count after cleanup = %d, should be much smaller", count)
	}
}

func TestCapacityFailClosedBehavior(t *testing.T) {
	// Create limiter with small capacity
	limiter := NewLimiterWithCapacity(3, 60*time.Minute, 10)

	// Fill to capacity with valid rate limits
	for i := range 10 {
		identifier := fmt.Sprintf("user%d@example.com", i)
		limiter.AllowRequest(identifier)
	}

	// Try new identifier - should fail (no expired entries to clean)
	allowed := limiter.AllowRequest("new@example.com")
	if allowed {
		t.Error("Request should be denied when at capacity with active limits")
	}

	// Verify no active limits were evicted
	if limiter.Count() != 10 {
		t.Errorf("Count = %d, want 10 (no eviction should occur)", limiter.Count())
	}
}

func TestCapacityConcurrentAccess(t *testing.T) {
	limiter := NewLimiterWithCapacity(3, 60*time.Minute, 100)

	// Fill near capacity
	for i := range 90 {
		identifier := fmt.Sprintf("user%d@example.com", i)
		limiter.AllowRequest(identifier)
	}

	// Concurrent attempts to fill remaining capacity
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex
	const goroutines = 50

	for i := range goroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			identifier := fmt.Sprintf("concurrent%d@example.com", id)
			allowed := limiter.AllowRequest(identifier)
			if allowed {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// Verify capacity not exceeded
	finalCount := limiter.Count()
	if finalCount > 100 {
		t.Errorf("Count exceeded capacity: %d > 100", finalCount)
	}

	// Verify some succeeded and some failed
	if successCount == 0 {
		t.Error("No concurrent requests succeeded")
	}
	if successCount == goroutines {
		t.Error("All concurrent requests succeeded (capacity not enforced)")
	}
}

func TestIsFull(t *testing.T) {
	limiter := NewLimiterWithCapacity(3, 60*time.Minute, 10)

	// Initially not full
	if limiter.IsFull() {
		t.Error("Empty limiter should not be full")
	}

	// Add entries
	for i := range 5 {
		identifier := fmt.Sprintf("user%d@example.com", i)
		limiter.AllowRequest(identifier)
	}

	// Still not full
	if limiter.IsFull() {
		t.Error("Half-full limiter should not be full")
	}

	// Fill to capacity
	for i := 5; i < 10; i++ {
		identifier := fmt.Sprintf("user%d@example.com", i)
		limiter.AllowRequest(identifier)
	}

	// Now full
	if !limiter.IsFull() {
		t.Error("Limiter at capacity should be full")
	}
}

func BenchmarkAllowRequest(b *testing.B) {
	limiter := NewLimiter(1000, 60*time.Minute)
	identifier := "bench@example.com"

	b.ResetTimer()
	for range b.N {
		limiter.AllowRequest(identifier)
	}
}

func BenchmarkAllowRequestConcurrent(b *testing.B) {
	limiter := NewLimiter(100000, 60*time.Minute)

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			identifier := "bench" + string(rune(i%100)) + "@example.com"
			limiter.AllowRequest(identifier)
			i++
		}
	})
}
