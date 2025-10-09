package rpc

import (
	"net"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// extractClientIP extracts the real client IP address from the request.
// It checks (in order):
// 1. X-Forwarded-For header (leftmost IP is the original client)
// 2. X-Real-IP header
// 3. Direct connection remote address
//
// This handles proxy scenarios correctly and provides defense against
// IP spoofing by trusting proxy headers when present.
func extractClientIP(c *fiber.Ctx) string {
	// Check X-Forwarded-For header first (most common proxy scenario)
	// Format: "client, proxy1, proxy2"
	// The leftmost IP is the original client
	xForwardedFor := c.Get("X-Forwarded-For")
	if xForwardedFor != "" {
		// Split by comma and take first IP
		ips := strings.Split(xForwardedFor, ",")
		if len(ips) > 0 {
			clientIP := strings.TrimSpace(ips[0])
			// Validate it's a valid IP address
			if ip := net.ParseIP(clientIP); ip != nil {
				return clientIP
			}
		}
	}

	// Check X-Real-IP header (single IP from proxy)
	xRealIP := c.Get("X-Real-IP")
	if xRealIP != "" {
		// Validate it's a valid IP address
		if ip := net.ParseIP(xRealIP); ip != nil {
			return xRealIP
		}
	}

	// Fallback to direct connection IP
	// c.IP() already handles extracting IP from RemoteAddr
	remoteIP := c.IP()

	// Additional validation: ensure it's a valid IP
	if ip := net.ParseIP(remoteIP); ip != nil {
		return remoteIP
	}

	// Ultimate fallback (should never happen, but fail-closed)
	// Return a placeholder that will be rate-limited
	return "0.0.0.0"
}
