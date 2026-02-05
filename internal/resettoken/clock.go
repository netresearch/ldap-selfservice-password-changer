// Package resettoken manages password reset tokens with expiration and validation.
package resettoken

import "time"

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

// defaultClock is the clock used by default.
var defaultClock Clock = RealClock{}
