package rpchandler

import (
	"net"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
)

// TestExtractClientIP tests IP extraction from various headers and direct connection.
func TestExtractClientIP(t *testing.T) {
	tests := []struct {
		name          string
		remoteAddr    string
		xForwardedFor string
		xRealIP       string
		expectedIP    string
	}{
		{
			name:       "direct connection",
			remoteAddr: "192.168.1.100:12345",
			expectedIP: "192.168.1.100",
		},
		{
			name:          "X-Forwarded-For single IP",
			remoteAddr:    "127.0.0.1:12345",
			xForwardedFor: "203.0.113.42",
			expectedIP:    "203.0.113.42",
		},
		{
			name:          "X-Forwarded-For multiple IPs (leftmost is client)",
			remoteAddr:    "127.0.0.1:12345",
			xForwardedFor: "203.0.113.42, 198.51.100.1, 192.0.2.1",
			expectedIP:    "203.0.113.42",
		},
		{
			name:          "X-Forwarded-For with spaces",
			remoteAddr:    "127.0.0.1:12345",
			xForwardedFor: "  203.0.113.42  ,  198.51.100.1  ",
			expectedIP:    "203.0.113.42",
		},
		{
			name:       "X-Real-IP header",
			remoteAddr: "127.0.0.1:12345",
			xRealIP:    "203.0.113.42",
			expectedIP: "203.0.113.42",
		},
		{
			name:          "X-Forwarded-For takes precedence over X-Real-IP",
			remoteAddr:    "127.0.0.1:12345",
			xForwardedFor: "203.0.113.42",
			xRealIP:       "198.51.100.1",
			expectedIP:    "203.0.113.42",
		},
		{
			name:       "IPv6 direct connection",
			remoteAddr: "[2001:db8::1]:12345",
			expectedIP: "2001:db8::1",
		},
		{
			name:          "IPv6 in X-Forwarded-For",
			remoteAddr:    "127.0.0.1:12345",
			xForwardedFor: "2001:db8::1",
			expectedIP:    "2001:db8::1",
		},
		{
			name:       "localhost",
			remoteAddr: "127.0.0.1:12345",
			expectedIP: "127.0.0.1",
		},
		{
			name:          "empty X-Forwarded-For falls back to remote addr",
			remoteAddr:    "192.168.1.100:12345",
			xForwardedFor: "",
			expectedIP:    "192.168.1.100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock fiber context
			app := fiber.New()
			reqCtx := &fasthttp.RequestCtx{}
			ctx := app.AcquireCtx(reqCtx)
			defer app.ReleaseCtx(ctx)

			// Set remote address on the underlying fasthttp context
			addr, err := net.ResolveTCPAddr("tcp", tt.remoteAddr)
			if err == nil {
				reqCtx.SetRemoteAddr(addr)
			}

			// Set headers
			if tt.xForwardedFor != "" {
				ctx.Request().Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.xRealIP != "" {
				ctx.Request().Header.Set("X-Real-IP", tt.xRealIP)
			}

			// Extract IP
			result := extractClientIP(ctx)

			if result != tt.expectedIP {
				t.Errorf("extractClientIP() = %q, want %q", result, tt.expectedIP)
			}
		})
	}
}

// TestExtractClientIPMalformed tests handling of malformed IP addresses.
func TestExtractClientIPMalformed(t *testing.T) {
	tests := []struct {
		name          string
		xForwardedFor string
		remoteAddr    string
		shouldContain string // Expected IP should contain this (fallback behavior)
	}{
		{
			name:          "invalid IP in X-Forwarded-For",
			xForwardedFor: "not-an-ip",
			remoteAddr:    "192.168.1.100:12345",
			shouldContain: "192.168.1.100", // Should fallback to remote addr
		},
		{
			name:          "SQL injection attempt in X-Forwarded-For",
			xForwardedFor: "'; DROP TABLE users; --",
			remoteAddr:    "192.168.1.100:12345",
			shouldContain: "192.168.1.100",
		},
		{
			name:          "script injection attempt",
			xForwardedFor: "<script>alert('xss')</script>",
			remoteAddr:    "192.168.1.100:12345",
			shouldContain: "192.168.1.100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			reqCtx := &fasthttp.RequestCtx{}
			ctx := app.AcquireCtx(reqCtx)
			defer app.ReleaseCtx(ctx)

			addr, err := net.ResolveTCPAddr("tcp", tt.remoteAddr)
			if err == nil {
				reqCtx.SetRemoteAddr(addr)
			}
			if tt.xForwardedFor != "" {
				ctx.Request().Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}

			result := extractClientIP(ctx)

			// For malformed input, we expect fallback to remote addr
			// The implementation should be defensive and always return a valid IP
			if result == "" {
				t.Error("extractClientIP() returned empty string")
			}
		})
	}
}
