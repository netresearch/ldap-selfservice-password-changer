package ratelimit_test

import (
	"testing"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/ratelimit"
)

func TestNewIPLimiter(t *testing.T) {
	limiter := ratelimit.NewIPLimiter()
	if limiter == nil {
		t.Fatal("NewIPLimiter() returned nil")
	}
}

func TestIPLimiterAllowRequest(t *testing.T) {
	limiter := ratelimit.NewIPLimiter()
	ip := "192.168.1.1"

	// First 10 requests should be allowed.
	for range 10 {
		allowed := limiter.AllowRequest(ip)
		if !allowed {
			t.Error("Request should be allowed")
		}
	}

	// 11th request should be blocked.
	allowed := limiter.AllowRequest(ip)
	if allowed {
		t.Error("Request 11 should be blocked (rate limit exceeded)")
	}
}

func TestIPLimiterDifferentIPs(t *testing.T) {
	limiter := ratelimit.NewIPLimiter()

	// IP 1 makes 10 requests.
	for range 10 {
		limiter.AllowRequest("192.168.1.1")
	}

	// IP 2 should still be allowed.
	allowed := limiter.AllowRequest("192.168.1.2")
	if !allowed {
		t.Error("Different IP should not be rate limited")
	}
}

func TestIPLimiterCount(t *testing.T) {
	limiter := ratelimit.NewIPLimiter()

	limiter.AllowRequest("192.168.1.1")
	limiter.AllowRequest("192.168.1.2")

	if limiter.Count() != 2 {
		t.Errorf("Count = %d, want 2", limiter.Count())
	}
}

func TestIPLimiterCleanupExpired(t *testing.T) {
	limiter := ratelimit.NewIPLimiter()

	limiter.AllowRequest("192.168.1.1")
	limiter.AllowRequest("192.168.1.2")

	if limiter.Count() != 2 {
		t.Errorf("Count = %d, want 2", limiter.Count())
	}

	// Cleanup should not remove active entries.
	count := limiter.CleanupExpired()
	if count != 0 {
		t.Errorf("CleanupExpired() removed %d entries, want 0", count)
	}

	if limiter.Count() != 2 {
		t.Errorf("Count after cleanup = %d, want 2", limiter.Count())
	}
}

func TestIPLimiterIPv6(t *testing.T) {
	limiter := ratelimit.NewIPLimiter()
	ipv6 := "2001:0db8:85a3:0000:0000:8a2e:0370:7334"

	// IPv6 addresses should work.
	for range 10 {
		allowed := limiter.AllowRequest(ipv6)
		if !allowed {
			t.Error("IPv6 request should be allowed")
		}
	}

	// 11th request should be blocked.
	allowed := limiter.AllowRequest(ipv6)
	if allowed {
		t.Error("IPv6 request 11 should be blocked")
	}
}

func TestIPLimiterIPv4AndIPv6Separate(t *testing.T) {
	limiter := ratelimit.NewIPLimiter()
	ipv4 := "192.168.1.1"
	ipv6 := "2001:0db8:85a3:0000:0000:8a2e:0370:7334"

	// Exhaust IPv4 limit.
	for range 10 {
		limiter.AllowRequest(ipv4)
	}

	// IPv6 should still be allowed (different identifier).
	allowed := limiter.AllowRequest(ipv6)
	if !allowed {
		t.Error("IPv6 should not be rate limited by IPv4")
	}
}
