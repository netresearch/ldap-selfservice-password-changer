package ratelimit

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestNewIPLimiter(t *testing.T) {
	limiter := NewIPLimiter()
	if limiter == nil {
		t.Fatal("NewIPLimiter() returned nil")
	}
	if limiter.limiter == nil {
		t.Error("IPLimiter internal limiter not initialized")
	}
}

func TestIPLimiterAllowRequest(t *testing.T) {
	limiter := NewIPLimiter()
	ip := "192.168.1.1"

	// First 10 requests should be allowed (default limit)
	for i := 1; i <= 10; i++ {
		allowed := limiter.AllowRequest(ip)
		if !allowed {
			t.Errorf("Request %d should be allowed", i)
		}
	}

	// 11th request should be blocked
	allowed := limiter.AllowRequest(ip)
	if allowed {
		t.Error("Request 11 should be blocked (rate limit exceeded)")
	}
}

func TestIPLimiterDifferentIPs(t *testing.T) {
	limiter := NewIPLimiter()

	// IP1 makes 10 requests (reaches limit)
	for i := 0; i < 10; i++ {
		limiter.AllowRequest("192.168.1.1")
	}

	// IP2 should still be allowed
	allowed := limiter.AllowRequest("192.168.1.2")
	if !allowed {
		t.Error("Different IP should not be rate limited")
	}
}

func TestIPLimiterStricterThanEmail(t *testing.T) {
	ipLimiter := NewIPLimiter()
	emailLimiter := NewLimiter(3, 60*time.Minute)

	ip := "10.0.0.1"
	email := "user@example.com"

	// IP limiter allows 10 requests
	ipCount := 0
	for i := 0; i < 20; i++ {
		if ipLimiter.AllowRequest(ip) {
			ipCount++
		}
	}

	// Email limiter allows 3 requests
	emailCount := 0
	for i := 0; i < 20; i++ {
		if emailLimiter.AllowRequest(email) {
			emailCount++
		}
	}

	if ipCount != 10 {
		t.Errorf("IP limiter allowed %d requests, want 10", ipCount)
	}
	if emailCount != 3 {
		t.Errorf("Email limiter allowed %d requests, want 3", emailCount)
	}

	// Verify IP limit is stricter (more requests) than email limit
	// This catches flooding from single IP before hitting email limits
	if ipCount <= emailCount {
		t.Error("IP limiter should allow more requests than email limiter to catch floods first")
	}
}

func TestIPLimiterCapacity(t *testing.T) {
	limiter := NewIPLimiter()

	// Fill to capacity (1000 IPs)
	for i := 0; i < 1000; i++ {
		ip := fmt.Sprintf("192.168.%d.%d", i/256, i%256)
		allowed := limiter.AllowRequest(ip)
		if !allowed {
			t.Errorf("Request %d should be allowed when filling to capacity", i)
		}
	}

	// Verify at capacity
	if limiter.Count() != 1000 {
		t.Errorf("Count = %d, want 1000", limiter.Count())
	}

	// Next IP should be denied
	allowed := limiter.AllowRequest("10.0.0.1")
	if allowed {
		t.Error("Request should be denied when at capacity")
	}
}

func TestIPLimiterHybridWithEmail(t *testing.T) {
	ipLimiter := NewIPLimiter()
	emailLimiter := NewLimiter(3, 60*time.Minute)

	ip := "203.0.113.1"
	email1 := "user1@example.com"
	email2 := "user2@example.com"

	// Scenario: Single IP trying multiple emails
	// IP check should catch this pattern

	// First email - 3 requests allowed by email limiter
	for i := 0; i < 3; i++ {
		// Check IP first (as would happen in handler)
		ipAllowed := ipLimiter.AllowRequest(ip)
		emailAllowed := emailLimiter.AllowRequest(email1)

		if !ipAllowed || !emailAllowed {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// Second email - email limiter would allow, but IP limiter should still track
	for i := 0; i < 3; i++ {
		ipAllowed := ipLimiter.AllowRequest(ip)
		emailAllowed := emailLimiter.AllowRequest(email2)

		if !ipAllowed || !emailAllowed {
			t.Errorf("Request %d with second email should be allowed", i+4)
		}
	}

	// Continue with different emails - IP limiter should eventually block
	email3 := "user3@example.com"
	for i := 0; i < 4; i++ {
		ipLimiter.AllowRequest(ip)
		emailLimiter.AllowRequest(email3)
	}

	// By now IP has made 10 requests - next should be blocked
	ipAllowed := ipLimiter.AllowRequest(ip)
	if ipAllowed {
		t.Error("IP should be blocked after exceeding limit across multiple emails")
	}

	// Email limiter would NOT allow for email3 (made 4 requests, limit is 3)
	// But a NEW email should still be allowed
	email4 := "user4@example.com"
	emailAllowed := emailLimiter.AllowRequest(email4)
	if !emailAllowed {
		t.Error("Email limiter should still allow email4 (new email)")
	}

	// But IP limiter should block it
	ipAllowed = ipLimiter.AllowRequest(ip)
	if ipAllowed {
		t.Error("IP should remain blocked even for new email addresses")
	}
}

func TestIPLimiterConcurrentAccess(t *testing.T) {
	limiter := NewIPLimiter()
	var wg sync.WaitGroup
	const goroutines = 50

	// Concurrent requests from same IP
	ip := "198.51.100.1"
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allowed := limiter.AllowRequest(ip)
			if allowed {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Should allow exactly 10 requests (limit)
	if successCount != 10 {
		t.Errorf("Allowed %d requests, want 10", successCount)
	}
}

func TestIPLimiterCleanup(t *testing.T) {
	limiter := NewIPLimiter()

	// Add some IPs (limiter has 60-minute window, so won't expire quickly)
	// We'll just verify cleanup works
	limiter.AllowRequest("192.168.1.1")
	limiter.AllowRequest("192.168.1.2")

	if limiter.Count() != 2 {
		t.Errorf("Count = %d, want 2", limiter.Count())
	}

	// Cleanup won't remove these (not expired), but should run without error
	count := limiter.CleanupExpired()
	if count != 0 {
		t.Errorf("Cleanup removed %d entries, want 0 (not expired)", count)
	}

	// Verify count unchanged
	if limiter.Count() != 2 {
		t.Errorf("Count after cleanup = %d, want 2", limiter.Count())
	}
}

func TestIPLimiterIsFull(t *testing.T) {
	limiter := NewIPLimiter()

	// Initially not full
	if limiter.IsFull() {
		t.Error("Empty limiter should not be full")
	}

	// Fill to capacity
	for i := 0; i < 1000; i++ {
		ip := fmt.Sprintf("10.%d.%d.1", i/256, i%256)
		limiter.AllowRequest(ip)
	}

	// Now full
	if !limiter.IsFull() {
		t.Error("Limiter at capacity should be full")
	}
}

func TestIPv6Addresses(t *testing.T) {
	limiter := NewIPLimiter()

	ipv6 := "2001:db8::1"

	// Should work with IPv6 addresses
	for i := 0; i < 10; i++ {
		allowed := limiter.AllowRequest(ipv6)
		if !allowed {
			t.Errorf("IPv6 request %d should be allowed", i+1)
		}
	}

	// 11th should be blocked
	allowed := limiter.AllowRequest(ipv6)
	if allowed {
		t.Error("IPv6 request 11 should be blocked")
	}
}

func TestMixedIPv4IPv6(t *testing.T) {
	limiter := NewIPLimiter()

	ipv4 := "192.0.2.1"
	ipv6 := "2001:db8::1"

	// IPv4 requests
	for i := 0; i < 10; i++ {
		limiter.AllowRequest(ipv4)
	}

	// IPv6 should still be allowed (different identifier)
	allowed := limiter.AllowRequest(ipv6)
	if !allowed {
		t.Error("IPv6 should not be affected by IPv4 rate limit")
	}
}
