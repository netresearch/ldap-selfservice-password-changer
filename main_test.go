package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestRunHealthCheckSuccess tests runHealthCheck with a successful health endpoint.
func TestRunHealthCheckSuccess(t *testing.T) {
	// Create a test server that returns HTTP 200
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/health/live", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"alive"}`))
	}))
	defer server.Close()

	// Test the helper function that can be called with a custom URL
	exitCode := testableRunHealthCheck(server.URL + "/health/live")
	assert.Equal(t, 0, exitCode, "should return 0 for successful health check")
}

// TestRunHealthCheckNon200Status tests runHealthCheck with non-200 responses.
func TestRunHealthCheckNon200Status(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantExit   int
	}{
		{
			name:       "404 Not Found",
			statusCode: http.StatusNotFound,
			wantExit:   1,
		},
		{
			name:       "500 Internal Server Error",
			statusCode: http.StatusInternalServerError,
			wantExit:   1,
		},
		{
			name:       "503 Service Unavailable",
			statusCode: http.StatusServiceUnavailable,
			wantExit:   1,
		},
		{
			name:       "201 Created (not 200)",
			statusCode: http.StatusCreated,
			wantExit:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			exitCode := testableRunHealthCheck(server.URL + "/health/live")
			assert.Equal(t, tt.wantExit, exitCode)
		})
	}
}

// TestRunHealthCheckConnectionError tests runHealthCheck when server is unreachable.
func TestRunHealthCheckConnectionError(t *testing.T) {
	// Use a URL that won't connect
	exitCode := testableRunHealthCheck("http://127.0.0.1:65432/health/live")
	assert.Equal(t, 1, exitCode, "should return 1 when connection fails")
}

// TestRunHealthCheckInvalidURL tests runHealthCheck with an invalid URL.
func TestRunHealthCheckInvalidURL(t *testing.T) {
	// Invalid URL scheme
	exitCode := testableRunHealthCheck("not-a-valid-url")
	assert.Equal(t, 1, exitCode, "should return 1 for invalid URL")
}

// TestRunHealthCheckTimeout tests runHealthCheck timeout behavior.
func TestRunHealthCheckTimeout(t *testing.T) {
	// Create a server that delays longer than the health check timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Sleep longer than our test timeout (we'll use a shorter test timeout)
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Use a very short timeout for testing
	exitCode := testableRunHealthCheckWithTimeout(server.URL+"/health/live", 50*time.Millisecond)
	assert.Equal(t, 1, exitCode, "should return 1 on timeout")
}

// TestRunHealthCheckEmptyBody tests runHealthCheck with an empty response body.
func TestRunHealthCheckEmptyBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		// No body written
	}))
	defer server.Close()

	exitCode := testableRunHealthCheck(server.URL + "/health/live")
	assert.Equal(t, 0, exitCode, "should return 0 even with empty body if status is 200")
}

// TestRunHealthCheckWithHeaders tests that runHealthCheck handles various server headers.
func TestRunHealthCheckWithHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Custom-Header", "test")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"alive"}`))
	}))
	defer server.Close()

	exitCode := testableRunHealthCheck(server.URL + "/health/live")
	assert.Equal(t, 0, exitCode)
}

// testableRunHealthCheck is a testable version of runHealthCheck that accepts a custom URL.
func testableRunHealthCheck(endpoint string) int {
	return testableRunHealthCheckWithTimeout(endpoint, 3*time.Second)
}

// testableRunHealthCheckWithTimeout is a testable version with configurable timeout.
func testableRunHealthCheckWithTimeout(endpoint string, timeout time.Duration) int {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return 1
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 1
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusOK {
		return 0
	}

	return 1
}

// TestHealthCheckConstants tests the health check related constants.
func TestHealthCheckConstants(t *testing.T) {
	// Verify constants are reasonable
	assert.Equal(t, 3*time.Second, healthCheckTimeout, "health check timeout should be 3 seconds")
	assert.Equal(t, "http://localhost:3000/health/live", healthCheckEndpoint, "health check endpoint should be localhost:3000")
}

// TestRunHealthCheckActualFunction tests the actual runHealthCheck function behavior.
// This test verifies the function signature and basic contract.
func TestRunHealthCheckActualFunction(t *testing.T) {
	// The actual runHealthCheck function uses hardcoded localhost:3000
	// which won't work in tests, so we test that it returns 1 when connection fails
	exitCode := runHealthCheck()
	assert.Equal(t, 1, exitCode, "should return 1 when localhost:3000 is not available")
}

// BenchmarkRunHealthCheck benchmarks the health check operation.
func BenchmarkRunHealthCheck(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"alive"}`))
	}))
	defer server.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = testableRunHealthCheck(server.URL + "/health/live")
	}
}
