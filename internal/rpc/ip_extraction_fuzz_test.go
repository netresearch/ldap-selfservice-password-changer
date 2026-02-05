//nolint:testpackage // tests internal functions
package rpc

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

// FuzzExtractClientIP fuzzes the extractClientIP function with malformed headers.
func FuzzExtractClientIP(f *testing.F) {
	// Seed corpus with various header values
	seeds := []struct {
		xForwardedFor string
		xRealIP       string
	}{
		{"", ""},
		{"192.168.1.1", ""},
		{"", "10.0.0.1"},
		{"192.168.1.1, 10.0.0.1, 172.16.0.1", ""},
		{"::1", ""},
		{"2001:db8::1", ""},
		{"invalid-ip", ""},
		{"192.168.1.1,invalid", ""},
		{" 192.168.1.1 ", ""},
		{"<script>alert('xss')</script>", ""},
		{"' OR '1'='1", ""},
		{"\x00\xff", ""},
		{"192.168.1.1\n10.0.0.1", ""},
		{"192.168.1.1\t10.0.0.1", ""},
		{"256.256.256.256", ""},
		{"-1.-1.-1.-1", ""},
		{"192.168.1", ""},
		{"192.168.1.1.1.1", ""},
		{"localhost", ""},
		{"example.com", ""},
		{"http://192.168.1.1", ""},
	}

	for _, s := range seeds {
		f.Add(s.xForwardedFor, s.xRealIP)
	}

	f.Fuzz(func(t *testing.T, xForwardedFor, xRealIP string) {
		app := fiber.New()

		var extractedIP string
		app.Get("/test", func(c fiber.Ctx) error {
			extractedIP = extractClientIP(c)
			return c.SendStatus(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
		if xForwardedFor != "" {
			req.Header.Set("X-Forwarded-For", xForwardedFor)
		}
		if xRealIP != "" {
			req.Header.Set("X-Real-IP", xRealIP)
		}

		resp, err := app.Test(req)
		if err != nil {
			// Fiber might reject malformed requests, which is fine
			return
		}
		defer func() { _ = resp.Body.Close() }()

		// The function should always return a valid IP address or fallback
		if extractedIP != "" {
			// If we got a result, it should be either a valid IP or the fallback
			if extractedIP != "0.0.0.0" {
				ip := net.ParseIP(extractedIP)
				if ip == nil {
					t.Errorf("extractClientIP returned invalid IP: %q", extractedIP)
				}
			}
		}
	})
}
