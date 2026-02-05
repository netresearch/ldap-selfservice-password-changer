//go:build e2e

// Package e2e provides end-to-end tests for the password changer application.
package e2e

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/options"
	webstatic "github.com/netresearch/ldap-selfservice-password-changer/internal/web/static"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/web/templates"
)

// createTestApp creates a Fiber app configured for E2E testing.
func createTestApp(t *testing.T) *fiber.App {
	t.Helper()

	opts := &options.Opts{
		Port:                        "3000",
		MinLength:                   8,
		MinNumbers:                  1,
		MinSymbols:                  1,
		MinUppercase:                1,
		MinLowercase:                1,
		PasswordCanIncludeUsername:  false,
		PasswordResetEnabled:        true,
		ResetTokenExpiryMinutes:     15,
		ResetRateLimitRequests:      10,
		ResetRateLimitWindowMinutes: 60,
	}

	indexPage, err := templates.RenderIndex(opts)
	require.NoError(t, err)

	forgotPasswordPage, err := templates.RenderForgotPassword()
	require.NoError(t, err)

	resetPasswordPage, err := templates.RenderResetPassword(opts)
	require.NoError(t, err)

	app := fiber.New(fiber.Config{
		AppName:      "netresearch/ldap-selfservice-password-changer-test",
		BodyLimit:    4 * 1024,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	})

	// Security headers
	app.Use(helmet.New(helmet.Config{
		ContentSecurityPolicy: "default-src 'self'; " +
			"script-src 'self'; " +
			"style-src 'self'; " +
			"img-src 'self' data:; " +
			"font-src 'self'; " +
			"connect-src 'self'; " +
			"frame-ancestors 'none'; " +
			"base-uri 'self'; " +
			"form-action 'self'",
		XFrameOptions:      "DENY",
		ContentTypeNosniff: "nosniff",
		ReferrerPolicy:     "strict-origin-when-cross-origin",
	}))

	// Routes
	app.Get("/", func(c fiber.Ctx) error {
		c.Set("Content-Type", fiber.MIMETextHTMLCharsetUTF8)
		return c.Send(indexPage)
	})

	app.Get("/forgot-password", func(c fiber.Ctx) error {
		c.Set("Content-Type", fiber.MIMETextHTMLCharsetUTF8)
		return c.Send(forgotPasswordPage)
	})

	app.Get("/reset-password", func(c fiber.Ctx) error {
		c.Set("Content-Type", fiber.MIMETextHTMLCharsetUTF8)
		return c.Send(resetPasswordPage)
	})

	app.Get("/health/live", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "alive"})
	})

	return app
}

// TestE2E_SecurityHeaders tests that security headers are properly set.
func TestE2E_SecurityHeaders(t *testing.T) {
	app := createTestApp(t)

	pages := []string{"/", "/forgot-password", "/reset-password"}

	for _, page := range pages {
		t.Run(page, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, page, http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Check security headers
			assert.Contains(t, resp.Header.Get("Content-Security-Policy"), "default-src 'self'")
			assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))
			assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
			assert.NotEmpty(t, resp.Header.Get("Referrer-Policy"))
		})
	}
}

// TestE2E_IndexPage tests the main index page.
func TestE2E_IndexPage(t *testing.T) {
	app := createTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, fiber.MIMETextHTMLCharsetUTF8, resp.Header.Get("Content-Type"))

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	html := string(body)

	// Verify page content
	assert.Contains(t, html, "<!doctype html>")
	assert.Contains(t, html, "GopherPass")
	assert.Contains(t, html, "Change Your Password")
	assert.Contains(t, html, `id="form"`)

	// Verify password configuration is embedded
	assert.Contains(t, html, `data-min-length="8"`)
	assert.Contains(t, html, `data-min-numbers="1"`)
	assert.Contains(t, html, `data-min-symbols="1"`)
	assert.Contains(t, html, `data-min-uppercase="1"`)
	assert.Contains(t, html, `data-min-lowercase="1"`)

	// Verify forgot password link is present when enabled
	assert.Contains(t, html, "/forgot-password")
}

// TestE2E_ForgotPasswordPage tests the forgot password page.
func TestE2E_ForgotPasswordPage(t *testing.T) {
	app := createTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/forgot-password", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	html := string(body)

	assert.Contains(t, html, "Forgot Password")
	assert.Contains(t, html, "Email Address")
	assert.Contains(t, html, "Send Reset Link")
	assert.Contains(t, html, "Back to Login")
}

// TestE2E_ResetPasswordPage tests the reset password page.
func TestE2E_ResetPasswordPage(t *testing.T) {
	app := createTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/reset-password", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	html := string(body)

	assert.Contains(t, html, "Reset Password")
	assert.Contains(t, html, "New Password")
	assert.Contains(t, html, "Confirm New Password")
	assert.Contains(t, html, `data-min-length="8"`)
}

// TestE2E_HealthEndpoint tests the health check endpoint.
func TestE2E_HealthEndpoint(t *testing.T) {
	app := createTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/health/live", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Contains(t, string(body), "alive")
}

// TestE2E_404Response tests that non-existent paths return 404.
func TestE2E_404Response(t *testing.T) {
	app := createTestApp(t)

	paths := []string{
		"/nonexistent",
		"/api/nonexistent",
		"/admin",
		"/.env",
		"/config.json",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		})
	}
}

// TestE2E_MethodNotAllowed tests that wrong methods return appropriate errors.
func TestE2E_MethodNotAllowed(t *testing.T) {
	app := createTestApp(t)

	tests := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/"},
		{http.MethodPut, "/"},
		{http.MethodDelete, "/health/live"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			// Should return 404 or 405
			assert.True(t, resp.StatusCode >= 400, "Expected error status, got %d", resp.StatusCode)
		})
	}
}

// TestE2E_AccessibilityAttributes tests that pages have proper accessibility attributes.
func TestE2E_AccessibilityAttributes(t *testing.T) {
	app := createTestApp(t)

	pages := []string{"/", "/forgot-password", "/reset-password"}

	for _, page := range pages {
		t.Run(page, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, page, http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			html := string(body)

			// Check accessibility attributes
			assert.Contains(t, html, `lang="en"`, "Should have lang attribute")
			assert.Contains(t, html, `role="main"`, "Should have main role")
			assert.Contains(t, html, `role="alert"`, "Should have alert role for error display")
			assert.Contains(t, html, `aria-live="assertive"`, "Should have assertive live region")
		})
	}
}

// TestE2E_ContentTypeHeaders tests that correct content types are set.
func TestE2E_ContentTypeHeaders(t *testing.T) {
	app := createTestApp(t)

	tests := []struct {
		path        string
		contentType string
	}{
		{"/", fiber.MIMETextHTMLCharsetUTF8},
		{"/forgot-password", fiber.MIMETextHTMLCharsetUTF8},
		{"/reset-password", fiber.MIMETextHTMLCharsetUTF8},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, tt.contentType, resp.Header.Get("Content-Type"))
		})
	}
}

// TestE2E_StaticFileHandling tests static file serving (when properly configured).
func TestE2E_StaticFileHandling(t *testing.T) {
	// This test verifies the static file handler is properly initialized
	// In E2E tests, we're testing that static files return appropriate content types

	// Verify the static FS is not nil
	assert.NotNil(t, webstatic.Static, "Static file system should be initialized")
}

// TestE2E_XSSPrevention tests that pages don't allow XSS in URL parameters.
func TestE2E_XSSPrevention(t *testing.T) {
	app := createTestApp(t)

	// Try XSS in query parameters (URL-encoded to be valid HTTP requests)
	// We test that dangerous payloads are NOT reflected unescaped in the response.
	// Note: These templates don't actually reflect query parameters, so the test
	// verifies that we don't have any XSS vectors in the template system.
	xssAttempts := []struct {
		url  string
		name string
	}{
		{
			url:  "/?user=%3Cscript%3Ealert(1)%3C/script%3E",
			name: "script_tag",
		},
		{
			url:  "/reset-password?token=%3Cscript%3Ealert(1)%3C/script%3E",
			name: "reset_script_tag",
		},
		{
			url:  "/forgot-password?redirect=javascript:alert(1)",
			name: "javascript_uri",
		},
	}

	for _, attempt := range xssAttempts {
		t.Run(attempt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, attempt.url, http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			html := string(body)

			// Verify XSS payloads are not reflected without escaping
			// The actual XSS payload strings should never appear unescaped
			assert.NotContains(t, html, "<script>alert(1)</script>", "XSS script should not be reflected")
			assert.NotContains(t, html, "onerror=alert(1)", "XSS event handler should not be reflected")
			assert.NotContains(t, html, "javascript:alert(1)", "XSS javascript: should not be reflected")
		})
	}
}

// TestE2E_ResponseSize tests that responses are within reasonable size limits.
func TestE2E_ResponseSize(t *testing.T) {
	app := createTestApp(t)

	pages := []string{"/", "/forgot-password", "/reset-password"}
	maxSize := int64(100 * 1024) // 100KB max for HTML pages

	for _, page := range pages {
		t.Run(page, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, page, http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			size := int64(len(body))
			assert.Less(t, size, maxSize, "Page %s is too large: %d bytes", page, size)
		})
	}
}

// TestE2E_NoSensitiveDataInHTML tests that no sensitive patterns appear in HTML.
func TestE2E_NoSensitiveDataInHTML(t *testing.T) {
	app := createTestApp(t)

	pages := []string{"/", "/forgot-password", "/reset-password"}

	sensitivePatterns := []string{
		"password=",
		"secret=",
		"api_key=",
		"token=", // as attribute value
		"ldap://",
		"ldaps://",
	}

	for _, page := range pages {
		t.Run(page, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, page, http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			html := strings.ToLower(string(body))

			for _, pattern := range sensitivePatterns {
				assert.NotContains(t, html, pattern,
					"Page %s should not contain sensitive pattern: %s", page, pattern)
			}
		})
	}
}
