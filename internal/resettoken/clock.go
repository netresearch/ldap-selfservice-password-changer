// Package resettoken manages password reset tokens with expiration and validation.
package resettoken

import (
	"sync"
	"time"
)

// Clock is an interface for time operations, allowing for testing.
type Clock interface {
	Now() time.Time
}

// RealClock implements Clock using the actual system time.
type RealClock struct{}

// Now returns the current time.
func (RealClock) Now() time.Time {
	return time.Now()
}

// clockMu protects access to defaultClock for thread-safe testing.
var clockMu sync.RWMutex

// defaultClock is the clock used by default.
var defaultClock Clock = RealClock{}

// getClock returns the current clock in a thread-safe manner.
func getClock() Clock {
	clockMu.RLock()
	defer clockMu.RUnlock()
	return defaultClock
}
