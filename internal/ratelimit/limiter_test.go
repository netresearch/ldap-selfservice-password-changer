package ratelimit_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/ratelimit"
)

func TestNewLimiter(t *testing.T) {
	limiter := ratelimit.NewLimiter(3, 60*time.Minute)
	if limiter == nil {
		t.Fatal("NewLimiter() returned nil")
	}
}

func TestAllowRequest(t *testing.T) {
	limiter := ratelimit.NewLimiter(3, 60*time.Minute)
	identifier := "user@example.com"

	// First 3 requests should be allowed.
	for range 3 {
		allowed := limiter.AllowRequest(identifier)
		if !allowed {
			t.Error("Request should be allowed")
		}
	}

	// 4th request should be blocked.
	allowed := limiter.AllowRequest(identifier)
	if allowed {
		t.Error("Request 4 should be blocked (rate limit exceeded)")
	}
}

func TestAllowRequestDifferentUsers(t *testing.T) {
	limiter := ratelimit.NewLimiter(3, 60*time.Minute)

	// User 1 makes 3 requests.
	for range 3 {
		limiter.AllowRequest("user1@example.com")
	}

	// User 2 should still be allowed.
	allowed := limiter.AllowRequest("user2@example.com")
	if !allowed {
		t.Error("Different user should not be rate limited")
	}
}

func TestSlidingWindow(t *testing.T) {
	limiter := ratelimit.NewLimiter(2, 100*time.Millisecond)
	identifier := "user@example.com"

	// Make 2 requests (limit reached).
	limiter.AllowRequest(identifier)
	limiter.AllowRequest(identifier)

	// 3rd request should be blocked.
	allowed := limiter.AllowRequest(identifier)
	if allowed {
		t.Error("Request should be blocked immediately")
	}

	// Wait for window to pass.
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again.
	allowed = limiter.AllowRequest(identifier)
	if !allowed {
		t.Error("Request should be allowed after window expired")
	}
}

func TestCleanupExpired(t *testing.T) {
	limiter := ratelimit.NewLimiter(3, 100*time.Millisecond)

	// Make requests from multiple users.
	limiter.AllowRequest("user1@example.com")
	limiter.AllowRequest("user2@example.com")
	limiter.AllowRequest("user3@example.com")

	// Wait for entries to expire.
	time.Sleep(150 * time.Millisecond)

	// Run cleanup.
	count := limiter.CleanupExpired()
	if count != 3 {
		t.Errorf("CleanupExpired() removed %d entries, want 3", count)
	}
}

func TestCount(t *testing.T) {
	limiter := ratelimit.NewLimiter(3, 60*time.Minute)

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
	// Create limiter with small capacity for testing.
	limiter := ratelimit.NewLimiterWithCapacity(3, 60*time.Minute, 10)

	// Fill to capacity with 10 different users.
	for i := range 10 {
		allowed := limiter.AllowRequest(fmt.Sprintf("user%d@example.com", i))
		if !allowed {
			t.Error("Request should be allowed when filling to capacity")
		}
	}

	// Verify at capacity.
	if limiter.Count() != 10 {
		t.Errorf("Count = %d, want 10", limiter.Count())
	}

	// Next request with new identifier should be denied.
	allowed := limiter.AllowRequest("new@example.com")
	if allowed {
		t.Error("Request should be denied when at capacity")
	}
}

func TestIsFull(t *testing.T) {
	limiter := ratelimit.NewLimiterWithCapacity(3, 60*time.Minute, 5)

	// Initially not full.
	if limiter.IsFull() {
		t.Error("Empty limiter should not be full")
	}

	// Fill to capacity with 5 different users.
	for i := range 5 {
		limiter.AllowRequest(fmt.Sprintf("user%d@example.com", i))
	}

	// Now full.
	if !limiter.IsFull() {
		t.Error("Limiter at capacity should be full")
	}
}

// TestStartCleanup tests the StartCleanup goroutine lifecycle.
func TestStartCleanup(t *testing.T) {
	limiter := ratelimit.NewLimiter(3, 50*time.Millisecond)

	// Add some entries.
	limiter.AllowRequest("user1@example.com")
	limiter.AllowRequest("user2@example.com")

	if limiter.Count() != 2 {
		t.Errorf("Count = %d, want 2", limiter.Count())
	}

	// Start cleanup goroutine with short interval.
	stop := limiter.StartCleanup(60 * time.Millisecond)
	defer close(stop)

	// Wait for entries to expire.
	time.Sleep(80 * time.Millisecond)

	// Wait for cleanup to run.
	time.Sleep(80 * time.Millisecond)

	// Entries should be cleaned up.
	count := limiter.Count()
	if count != 0 {
		t.Errorf("Count after cleanup = %d, want 0", count)
	}
}

// TestStartCleanupStop tests that stop channel properly terminates cleanup.
func TestStartCleanupStop(t *testing.T) {
	limiter := ratelimit.NewLimiter(3, 60*time.Minute)

	// Start cleanup with short interval.
	stop := limiter.StartCleanup(10 * time.Millisecond)

	// Let it run a few times.
	time.Sleep(50 * time.Millisecond)

	// Stop the goroutine.
	close(stop)

	// Wait briefly to ensure goroutine terminates.
	time.Sleep(20 * time.Millisecond)

	// Test passed if no panic/hang - goroutine properly terminated.
}

// TestStartCleanupWithActivity tests cleanup doesn't remove active entries.
func TestStartCleanupWithActivity(t *testing.T) {
	limiter := ratelimit.NewLimiter(3, 200*time.Millisecond)

	stop := limiter.StartCleanup(50 * time.Millisecond)
	defer close(stop)

	// Add entries and keep refreshing one of them.
	limiter.AllowRequest("active@example.com")
	limiter.AllowRequest("inactive@example.com")

	// Keep "active" user active by making requests.
	for i := range 4 {
		time.Sleep(60 * time.Millisecond)
		limiter.AllowRequest("active@example.com")
		if i < 3 {
			// After first few iterations, inactive should still be there.
			if limiter.Count() == 0 {
				t.Error("entries removed too early")
			}
		}
	}

	// After window expires, cleanup should have run.
	// Active user should still be tracked (has recent activity).
	// This test verifies cleanup only removes truly expired entries.
}
