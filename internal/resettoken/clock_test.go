//nolint:testpackage // tests internal clock mechanism
package resettoken

import (
	"sync"
	"time"
)

// MockClock is a mock implementation of Clock for testing.
type MockClock struct {
	mu          sync.Mutex
	currentTime time.Time
}

// NewMockClock creates a new MockClock set to the given time.
func NewMockClock(t time.Time) *MockClock {
	return &MockClock{currentTime: t}
}

// Now returns the mock current time.
func (m *MockClock) Now() time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.currentTime
}

// Advance moves the clock forward by the given duration.
func (m *MockClock) Advance(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.currentTime = m.currentTime.Add(d)
}

// Set sets the clock to a specific time.
func (m *MockClock) Set(t time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.currentTime = t
}

// setTestClock sets the package-level clock for testing and returns a cleanup function.
func setTestClock(c Clock) func() {
	old := defaultClock
	defaultClock = c
	return func() { defaultClock = old }
}
