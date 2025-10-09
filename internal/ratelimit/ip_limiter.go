package ratelimit

import "time"

// IPLimiter tracks requests per IP address with stricter limits than email limiter.
// This helps catch flooding attacks before they exhaust the email limiter capacity.
type IPLimiter struct {
	limiter *Limiter
}

// NewIPLimiter creates a new IP-based rate limiter.
// Default configuration: 10 requests per IP per 60 minutes, max 1000 tracked IPs.
func NewIPLimiter() *IPLimiter {
	return &IPLimiter{
		limiter: NewLimiterWithCapacity(10, 60*time.Minute, 1000),
	}
}

// AllowRequest checks if a request from the given IP address is allowed.
// Returns true if allowed, false if rate limit exceeded or capacity reached.
func (ipl *IPLimiter) AllowRequest(ipAddress string) bool {
	return ipl.limiter.AllowRequest(ipAddress)
}

// CleanupExpired removes expired IP entries from the limiter.
// Returns the number of entries removed.
func (ipl *IPLimiter) CleanupExpired() int {
	return ipl.limiter.CleanupExpired()
}

// StartCleanup starts a background goroutine that periodically cleans up expired entries.
// Returns a channel that can be used to stop the cleanup goroutine.
func (ipl *IPLimiter) StartCleanup(interval time.Duration) chan struct{} {
	return ipl.limiter.StartCleanup(interval)
}

// Count returns the current number of tracked IP addresses.
func (ipl *IPLimiter) Count() int {
	return ipl.limiter.Count()
}

// IsFull returns true if the IP limiter is at maximum capacity.
func (ipl *IPLimiter) IsFull() bool {
	return ipl.limiter.IsFull()
}
