package ratelimit

import (
	"sync"
	"time"
)

// maxIdentifiers is the default maximum number of identifiers that can be tracked simultaneously.
// This prevents unbounded memory growth from DoS attacks.
const maxIdentifiers = 10000

// Entry represents rate limit tracking for a single identifier.
type Entry struct {
	timestamps []time.Time // Timestamps of recent requests
}

// Limiter implements a sliding window rate limiter with capacity limits.
type Limiter struct {
	mu            sync.RWMutex
	entries       map[string]*Entry
	maxRequests   int           // Maximum requests allowed in window
	window        time.Duration // Time window for rate limiting
	maxIdentifiers int           // Maximum identifiers to track (capacity limit)
}

// NewLimiter creates a new rate limiter with the specified limits and default capacity.
func NewLimiter(maxRequests int, window time.Duration) *Limiter {
	return NewLimiterWithCapacity(maxRequests, window, maxIdentifiers)
}

// NewLimiterWithCapacity creates a new rate limiter with specified limits and capacity.
func NewLimiterWithCapacity(maxRequests int, window time.Duration, capacity int) *Limiter {
	return &Limiter{
		entries:        make(map[string]*Entry),
		maxRequests:    maxRequests,
		window:         window,
		maxIdentifiers: capacity,
	}
}

// AllowRequest checks if a request is allowed for the given identifier.
// Returns true if allowed, false if rate limit exceeded or capacity reached.
// If at capacity, attempts to cleanup expired entries before failing (fail-closed).
func (l *Limiter) AllowRequest(identifier string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	// Get or create entry for this identifier
	entry, exists := l.entries[identifier]
	if !exists {
		// Check capacity BEFORE creating new entry
		if len(l.entries) >= l.maxIdentifiers {
			// Try to cleanup expired entries to make room
			l.cleanupExpiredLocked(now, cutoff)

			// If still at capacity, fail closed
			if len(l.entries) >= l.maxIdentifiers {
				return false // System under heavy load
			}
		}

		entry = &Entry{
			timestamps: []time.Time{},
		}
		l.entries[identifier] = entry
	}

	// Remove timestamps outside the window (sliding window)
	validTimestamps := []time.Time{}
	for _, ts := range entry.timestamps {
		if ts.After(cutoff) {
			validTimestamps = append(validTimestamps, ts)
		}
	}
	entry.timestamps = validTimestamps

	// Check if rate limit exceeded
	if len(entry.timestamps) >= l.maxRequests {
		return false
	}

	// Add current request timestamp
	entry.timestamps = append(entry.timestamps, now)
	return true
}

// CleanupExpired removes entries with no recent requests.
// Returns the number of entries removed.
func (l *Limiter) CleanupExpired() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)
	return l.cleanupExpiredLocked(now, cutoff)
}

// cleanupExpiredLocked is the internal cleanup implementation.
// MUST be called with l.mu held (Lock, not RLock).
func (l *Limiter) cleanupExpiredLocked(now, cutoff time.Time) int {
	count := 0

	for identifier, entry := range l.entries {
		// Check if all timestamps are expired
		allExpired := true
		for _, ts := range entry.timestamps {
			if ts.After(cutoff) {
				allExpired = false
				break
			}
		}

		if allExpired {
			delete(l.entries, identifier)
			count++
		}
	}

	return count
}

// StartCleanup starts a background goroutine that periodically cleans up expired entries.
// Returns a channel that can be used to stop the cleanup goroutine.
func (l *Limiter) StartCleanup(interval time.Duration) chan struct{} {
	stop := make(chan struct{})

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				count := l.CleanupExpired()
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

// Count returns the current number of tracked identifiers.
// Useful for monitoring and debugging.
func (l *Limiter) Count() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.entries)
}

// IsFull returns true if the limiter is at maximum capacity.
// Useful for monitoring and capacity alerts.
func (l *Limiter) IsFull() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.entries) >= l.maxIdentifiers
}
