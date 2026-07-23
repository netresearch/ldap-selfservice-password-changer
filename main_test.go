package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	ldap "github.com/netresearch/simple-ldap-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/options"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/rpchandler"
)

// TestRunHealthCheckSuccess tests runHealthCheck with a successful health endpoint.
func TestRunHealthCheckSuccess(t *testing.T) {
	// Create a test server that returns HTTP 200
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/health/live", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"alive"}`)) //nolint:errcheck // test handler
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
		_, _ = w.Write([]byte(`{"status":"alive"}`)) //nolint:errcheck // test handler
	}))
	defer server.Close()

	exitCode := testableRunHealthCheck(server.URL + "/health/live")
	assert.Equal(t, 0, exitCode)
}

// testableRunHealthCheck exercises the real runHealthCheckAt at the default
// 3s timeout. Kept as a thin wrapper so existing tests read naturally.
func testableRunHealthCheck(endpoint string) int {
	return runHealthCheckAt(endpoint, 3*time.Second)
}

// testableRunHealthCheckWithTimeout is a thin wrapper that forwards to the
// real runHealthCheckAt. Previously a private duplicate — now it just aliases
// the production helper so tests cover the real code path.
func testableRunHealthCheckWithTimeout(endpoint string, timeout time.Duration) int {
	return runHealthCheckAt(endpoint, timeout)
}

// TestHealthCheckConstants tests the health check related constants.
func TestHealthCheckConstants(t *testing.T) {
	// Verify constants are reasonable
	assert.Equal(t, 3*time.Second, healthCheckTimeout, "health check timeout should be 3 seconds")
	assert.Equal(t, "http://localhost:3000/health/live", healthCheckEndpoint,
		"health check endpoint should be localhost:3000")
}

// TestRunHealthCheckDelegates verifies that runHealthCheck is a trivial
// wrapper around runHealthCheckAt using the production constants. We cannot
// easily assert a specific exit code because the hardcoded endpoint
// http://localhost:3000/health/live may or may not be reachable in the test
// environment. Instead, we stub the endpoint temporarily at that address
// by launching an httptest server, relying on httptest.NewServer's
// auto-assigned port.
//
// The core contract is: runHealthCheck must hit the configured
// healthCheckEndpoint with the healthCheckTimeout and return 0/1 accordingly.
// This is covered indirectly by runHealthCheckAt tests (which test the
// real code path with full coverage). Here we simply execute the
// zero-argument wrapper once to prove the delegation compiles and runs.
func TestRunHealthCheckDelegates(t *testing.T) {
	// Just invoking the function covers the wrapper; the behavior is
	// already fully tested via runHealthCheckAt. We cannot make a specific
	// assertion about the exit code because we don't control localhost:3000.
	_ = runHealthCheck()
	// Sanity: the constants used by runHealthCheck match documented values.
	assert.Equal(t, "http://localhost:3000/health/live", healthCheckEndpoint)
	assert.Equal(t, 3*time.Second, healthCheckTimeout)
}

// TestRunHealthCheckAtSuccess verifies the real runHealthCheckAt returns 0
// against a server answering with HTTP 200.
func TestRunHealthCheckAtSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"alive"}`)) //nolint:errcheck // test handler
	}))
	defer server.Close()

	exitCode := runHealthCheckAt(server.URL+"/health/live", 3*time.Second)
	assert.Equal(t, 0, exitCode, "runHealthCheckAt should return 0 when the server responds with HTTP 200")
}

// TestRunHealthCheckAtNon200 verifies the real runHealthCheckAt returns 1
// against a server answering with a non-OK status.
func TestRunHealthCheckAtNon200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	exitCode := runHealthCheckAt(server.URL+"/health/live", 3*time.Second)
	assert.Equal(t, 1, exitCode, "runHealthCheckAt should return 1 on non-200 responses")
}

// TestRunHealthCheckAtConnectionError verifies the real runHealthCheckAt
// returns 1 when the server is unreachable.
func TestRunHealthCheckAtConnectionError(t *testing.T) {
	// 127.0.0.1:1 is reserved and should reliably refuse connections.
	exitCode := runHealthCheckAt("http://127.0.0.1:1/health/live", 500*time.Millisecond)
	assert.Equal(t, 1, exitCode, "runHealthCheckAt should return 1 when the target refuses the connection")
}

// TestRunHealthCheckAtInvalidURL verifies the real runHealthCheckAt returns 1
// for a malformed URL (exercises the request construction error branch).
func TestRunHealthCheckAtInvalidURL(t *testing.T) {
	// A URL with a control character fails http.NewRequestWithContext.
	exitCode := runHealthCheckAt("http://127.0.0.1\x7f/", time.Second)
	assert.Equal(t, 1, exitCode, "runHealthCheckAt should return 1 for an invalid URL")
}

// TestIsHealthCheckInvocation tests detection of the --health-check flag.
func TestIsHealthCheckInvocation(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "nil", args: nil, want: false},
		{name: "only program name", args: []string{"app"}, want: false},
		{name: "health check", args: []string{"app", "--health-check"}, want: true},
		{name: "other flag", args: []string{"app", "--version"}, want: false},
		{name: "health check plus extra arg", args: []string{"app", "--health-check", "--verbose"}, want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isHealthCheckInvocation(tc.args))
		})
	}
}

// TestIsLDAPEncrypted verifies ldaps:// detection.
func TestIsLDAPEncrypted(t *testing.T) {
	tests := []struct {
		server string
		want   bool
	}{
		{"ldaps://dc.example.com:636", true},
		{"ldap://dc.example.com:389", false},
		{"", false},
		{"LDAPS://dc.example.com:636", false}, // case sensitive per implementation
		{"https://dc.example.com", false},
	}
	for _, tc := range tests {
		t.Run(tc.server, func(t *testing.T) {
			assert.Equal(t, tc.want, isLDAPEncrypted(tc.server))
		})
	}
}

// TestBuildEmailConfig verifies that options are mapped to the email config.
// Every field of email.Config gets a distinct sentinel value so a crossed
// mapping (e.g. TemplateHTMLPath fed from EmailTemplateText) fails here.
func TestBuildEmailConfig(t *testing.T) {
	opts := &options.Opts{
		SMTPHost:                "smtp.example.com",
		SMTPPort:                587,
		SMTPUsername:            "user",
		SMTPPassword:            "pass",
		SMTPFromAddress:         "noreply@example.com",
		SMTPFromName:            "Sentinel From Name",
		EmailReplyTo:            "sentinel-replyto@example.com",
		AppBaseURL:              "https://example.com",
		ResetTokenExpiryMinutes: 42,
		EmailTemplateSubject:    "Sentinel subject {{.Recipient}}",
		EmailTemplateHTML:       "/sentinel-html.html",
		EmailTemplateText:       "/sentinel-text.txt",
		SMTPHeaderOverrides: map[string]string{
			"X-Helpdesk-Topic":  "sentinel-topic",
			"X-Sentinel-Tenant": "sentinel-tenant",
		},
	}
	got := buildEmailConfig(opts)
	assert.Equal(t, "smtp.example.com", got.SMTPHost)
	assert.Equal(t, 587, got.SMTPPort)
	assert.Equal(t, "user", got.SMTPUsername)
	assert.Equal(t, "pass", got.SMTPPassword)
	assert.Equal(t, "noreply@example.com", got.FromAddress)
	assert.Equal(t, "Sentinel From Name", got.FromName)
	assert.Equal(t, "sentinel-replyto@example.com", got.ReplyTo)
	assert.Equal(t, "https://example.com", got.BaseURL)
	assert.Equal(t, uint(42), got.ExpiryMinutes)
	assert.Equal(t, "Sentinel subject {{.Recipient}}", got.SubjectTemplate)
	assert.Equal(t, "/sentinel-html.html", got.TemplateHTMLPath)
	assert.Equal(t, "/sentinel-text.txt", got.TemplateTextPath)
	assert.Equal(t, map[string]string{
		"X-Helpdesk-Topic":  "sentinel-topic",
		"X-Sentinel-Tenant": "sentinel-tenant",
	}, got.HeaderOverrides)
}

// TestResetRateLimitSettings verifies the rate limit setting extraction.
func TestResetRateLimitSettings(t *testing.T) {
	opts := &options.Opts{
		ResetRateLimitRequests:      5,
		ResetRateLimitWindowMinutes: 60,
	}
	req, window := resetRateLimitSettings(opts)
	assert.Equal(t, 5, req)
	assert.Equal(t, 60*time.Minute, window)
}

// TestResetRateLimitSettingsZero verifies zero values pass through.
func TestResetRateLimitSettingsZero(t *testing.T) {
	opts := &options.Opts{}
	req, window := resetRateLimitSettings(opts)
	assert.Equal(t, 0, req)
	assert.Equal(t, time.Duration(0), window)
}

// captureWarn runs fn with the default slog logger swapped for a JSON handler
// writing to a buffer, and returns the captured records.
func captureWarn(t *testing.T, fn func()) []map[string]any {
	t.Helper()

	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})))
	defer slog.SetDefault(previous)

	fn()

	var records []map[string]any
	for line := range strings.SplitSeq(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var rec map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &rec))
		records = append(records, rec)
	}
	return records
}

// TestWarnEmptySenderWithoutFromName verifies the plain empty-sender warning
// does not claim a display name is being dropped.
func TestWarnEmptySenderWithoutFromName(t *testing.T) {
	records := captureWarn(t, func() { warnEmptySender("") })

	require.Len(t, records, 1)
	assert.Equal(t, "smtp_from_address_empty", records[0]["msg"])
	assert.Equal(t, false, records[0]["from_name_dropped"])
	detail, ok := records[0]["detail"].(string)
	require.True(t, ok, "detail attribute missing or not a string")
	assert.Contains(t, detail, "SMTP_FROM_ADDRESS is not set")
	assert.NotContains(t, detail, "SMTP_FROM_NAME")
}

// TestWarnEmptySenderWithFromName covers SMTP_FROM_NAME set with an empty
// SMTP_FROM_ADDRESS: both pass their own startup checks, so the operator has to
// be told the display name is dropped rather than used.
func TestWarnEmptySenderWithFromName(t *testing.T) {
	records := captureWarn(t, func() { warnEmptySender("ACME IT") })

	require.Len(t, records, 1)
	assert.Equal(t, "smtp_from_address_empty", records[0]["msg"])
	assert.Equal(t, true, records[0]["from_name_dropped"])
	detail, ok := records[0]["detail"].(string)
	require.True(t, ok, "detail attribute missing or not a string")
	assert.Contains(t, detail, "SMTP_FROM_ADDRESS is not set")
	assert.Contains(t, detail, "SMTP_FROM_NAME is set but will be dropped")
}

// TestBuildRPCHandlerWarnsAboutDroppedFromName verifies the warning path is
// wired to the configured display name, not to a constant. The handler build
// is expected to fail: an unreadable template path makes email.NewService
// return before any LDAP dial, which keeps this test hermetic. The warning is
// emitted before that, so it is still captured.
func TestBuildRPCHandlerWarnsAboutDroppedFromName(t *testing.T) {
	opts := &options.Opts{
		PasswordResetEnabled: true,
		SMTPFromAddress:      "",
		SMTPFromName:         "ACME IT",
		EmailTemplateHTML:    filepath.Join(t.TempDir(), "does-not-exist.html"),
	}

	var handler *rpchandler.Handler
	var err error
	records := captureWarn(t, func() { handler, err = buildRPCHandler(opts) })

	require.Error(t, err, "handler build must fail on the unreadable template")
	assert.Nil(t, handler)

	require.Len(t, records, 1)
	assert.Equal(t, "smtp_from_address_empty", records[0]["msg"])
	assert.Equal(t, true, records[0]["from_name_dropped"])
}

// TestLogLDAPSecurityStatusDoesNotPanic verifies both ldap/ldaps cases run cleanly.
func TestLogLDAPSecurityStatusDoesNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		logLDAPSecurityStatus(&options.Opts{LDAP: ldap.Config{Server: "ldaps://host:636"}})
	})
	assert.NotPanics(t, func() {
		logLDAPSecurityStatus(&options.Opts{LDAP: ldap.Config{Server: "ldap://host:389"}})
	})
}

// TestBuildApp verifies that buildApp returns a non-nil Fiber app with
// security middleware hooked up — a GET / on /static path returns a 404
// because no routes are registered yet.
func TestBuildApp(t *testing.T) {
	app := buildApp("")
	require.NotNil(t, app)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/static/missing.txt", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// /static returns 404 for missing files, but security headers should be set.
	assert.NotEmpty(t, resp.Header.Get("X-Frame-Options"))
	assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	assert.NotEmpty(t, resp.Header.Get("Content-Security-Policy"))
}

// TestBuildApp_BrandingOverlayIsServed exercises the overlay through the real
// Fiber static middleware rather than through fs.FS directly: the middleware
// decides what path it hands to Open, so serving an overridden asset is the
// only proof the layering actually takes effect over HTTP.
func TestBuildApp_BrandingOverlayIsServed(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "logo.webp"), []byte("custom-logo-bytes"), 0o600))

	app := buildApp(dir)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/static/logo.webp", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "custom-logo-bytes", string(body), "the overlay file should win over the embedded logo")

	// An asset the overlay does not provide must still come from the embedded FS.
	req = httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/static/favicon.ico", http.NoBody)
	fallback, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = fallback.Body.Close() }()

	assert.Equal(t, http.StatusOK, fallback.StatusCode, "unoverridden assets must fall back to the embedded FS")
}

// TestRegisterCorePages verifies the real registerCorePages function wires
// up the / , /api/rpc and /health/live routes and serves them correctly.
func TestRegisterCorePages(t *testing.T) {
	app := buildApp("")

	indexBytes := []byte("<html>hi</html>")
	// Provide a tiny RPC handler stand-in to exercise the POST /api/rpc route.
	stubHandle := func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"method": "stub"})
	}
	registerCorePages(app, indexBytes, stubHandle)

	// GET / serves the supplied HTML bytes with HTML content-type.
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/html")
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "<html>hi</html>")

	// POST /api/rpc routes to the stub handler.
	req2 := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/rpc", http.NoBody)
	resp2, err := app.Test(req2)
	require.NoError(t, err)
	defer func() { _ = resp2.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp2.StatusCode)
	body2, err := io.ReadAll(resp2.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body2), "stub")

	// GET /health/live returns alive.
	req3 := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health/live", http.NoBody)
	resp3, err := app.Test(req3)
	require.NoError(t, err)
	defer func() { _ = resp3.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp3.StatusCode)
	body3, err := io.ReadAll(resp3.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body3), "alive")
}

// TestNewHandlerWithoutResetServicesLDAPError verifies that when the LDAP
// connection fails, newHandlerWithoutResetServices returns a non-nil error
// and a nil handler. It also exercises the goroutine-ordering fix: a failed
// handler init must NOT leak the IP limiter cleanup goroutine (the IP limiter
// should not be created until rpchandler.New succeeds).
//
// Previous name (TestRegisterCorePagesDoesNotPanic) was misleading — it never
// called registerCorePages.
func TestNewHandlerWithoutResetServicesLDAPError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping LDAP-dependent coverage in short mode")
	}
	opts := &options.Opts{
		Port: "3000",
		LDAP: ldap.Config{Server: "ldap://127.0.0.1:1", BaseDN: "dc=example,dc=com"},
	}

	// Record baseline goroutine count; after the failed init we expect no
	// long-lived goroutines spawned by the factory to remain.
	before := runtime.NumGoroutine()

	h, err := newHandlerWithoutResetServices(opts)
	require.Error(t, err, "LDAP dial to 127.0.0.1:1 must fail")
	assert.Nil(t, h, "no handler should be returned on error")

	// Give any stray cleanup goroutine a moment to start (if the ordering
	// regressed, a ticker-driven goroutine would still be alive here).
	time.Sleep(50 * time.Millisecond)
	after := runtime.NumGoroutine()
	assert.LessOrEqual(t, after, before+1,
		"failed handler init must not leak background cleanup goroutines (before=%d after=%d)", before, after)
}

// TestNewHandlerWithResetServicesLDAPError covers the mirror path for the
// reset-services factory. Same contract: error on LDAP failure, no leaked
// background goroutines.
func TestNewHandlerWithResetServicesLDAPError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping LDAP-dependent coverage in short mode")
	}
	opts := &options.Opts{
		Port: "3000",
		LDAP: ldap.Config{Server: "ldap://127.0.0.1:1", BaseDN: "dc=example,dc=com"},
		// Populate reset-related fields so we exercise the reset path.
		PasswordResetEnabled:        true,
		ResetTokenExpiryMinutes:     15,
		ResetRateLimitRequests:      3,
		ResetRateLimitWindowMinutes: 60,
		SMTPHost:                    "smtp.example.com",
		SMTPPort:                    587,
		SMTPFromAddress:             "noreply@example.com",
		AppBaseURL:                  "https://example.com",
	}

	before := runtime.NumGoroutine()

	h, err := newHandlerWithResetServices(opts)
	require.Error(t, err, "LDAP dial to 127.0.0.1:1 must fail")
	assert.Nil(t, h, "no handler should be returned on error")

	time.Sleep(50 * time.Millisecond)
	after := runtime.NumGoroutine()
	assert.LessOrEqual(t, after, before+1,
		"failed reset-handler init must not leak token-store or IP-limiter cleanup goroutines (before=%d after=%d)",
		before, after)
}

// TestNewHandlerWithResetServicesEmailInitError pins the fail-fast contract of
// ADR 0003: a configured-but-unreadable email template must abort handler
// construction. Falling back to the built-in templates would boot a server that
// silently ignores the operator's branding, which the ADR rules out. The error
// message is asserted because a fallback would still surface the later LDAP
// error and otherwise satisfy the error/nil-handler checks.
func TestNewHandlerWithResetServicesEmailInitError(t *testing.T) {
	opts := &options.Opts{
		Port:                        "3000",
		LDAP:                        ldap.Config{Server: "ldap://127.0.0.1:1", BaseDN: "dc=example,dc=com"},
		PasswordResetEnabled:        true,
		ResetTokenExpiryMinutes:     15,
		ResetRateLimitRequests:      3,
		ResetRateLimitWindowMinutes: 60,
		SMTPHost:                    "smtp.example.com",
		SMTPPort:                    587,
		SMTPFromAddress:             "noreply@example.com",
		AppBaseURL:                  "https://example.com",
		EmailTemplateText:           filepath.Join(t.TempDir(), "does-not-exist.txt"),
	}

	before := runtime.NumGoroutine()

	h, err := newHandlerWithResetServices(opts)
	require.Error(t, err, "an unreadable email template must abort handler construction")
	assert.Nil(t, h, "no handler should be returned on error")
	assert.Contains(t, err.Error(), "initialize email service",
		"the email service error must be propagated, not swallowed by a template fallback")

	time.Sleep(50 * time.Millisecond)
	after := runtime.NumGoroutine()
	assert.LessOrEqual(t, after, before+1,
		"a failed email-service init must not leak token-store or IP-limiter cleanup goroutines (before=%d after=%d)",
		before, after)
}

// TestRegisterResetPages verifies the reset pages render and respond correctly.
func TestRegisterResetPages(t *testing.T) {
	app := buildApp("")
	opts := validPasswordResetOpts(t)

	err := registerResetPages(app, opts)
	require.NoError(t, err)

	// /forgot-password
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/forgot-password", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/html")

	// /reset-password
	req2 := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/reset-password", http.NoBody)
	resp2, err := app.Test(req2)
	require.NoError(t, err)
	defer func() { _ = resp2.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp2.StatusCode)
	assert.Contains(t, resp2.Header.Get("Content-Type"), "text/html")
}

// validPasswordResetOpts returns Opts sufficient to render reset-related templates.
func validPasswordResetOpts(_ *testing.T) *options.Opts {
	return &options.Opts{
		Port:                        "3000",
		MinLength:                   8,
		MinNumbers:                  1,
		MinSymbols:                  1,
		MinUppercase:                1,
		MinLowercase:                1,
		PasswordResetEnabled:        true,
		ResetTokenExpiryMinutes:     15,
		ResetRateLimitRequests:      3,
		ResetRateLimitWindowMinutes: 60,
	}
}

// TestBuildServerConnectionFailure verifies the build flow returns an error
// when LDAP is unreachable (no reset services path).
func TestBuildServerConnectionFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow LDAP timeout test in short mode")
	}
	opts := &options.Opts{
		Port: "3000",
		LDAP: ldap.Config{
			Server: "ldap://127.0.0.1:1", // unreachable; fast fail
			BaseDN: "dc=example,dc=com",
		},
		ReadonlyUser:     "cn=readonly,dc=example,dc=com",
		ReadonlyPassword: "password",
	}
	app, err := buildServer(opts)
	assert.Error(t, err)
	assert.Nil(t, app)
}

// TestBuildServerConnectionFailureWithReset verifies the reset services path
// returns an error when LDAP is unreachable.
func TestBuildServerConnectionFailureWithReset(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow LDAP timeout test in short mode")
	}
	opts := &options.Opts{
		Port: "3000",
		LDAP: ldap.Config{
			Server: "ldap://127.0.0.1:1", // unreachable; fast fail
			BaseDN: "dc=example,dc=com",
		},
		ReadonlyUser:                "cn=readonly,dc=example,dc=com",
		ReadonlyPassword:            "password",
		PasswordResetEnabled:        true,
		ResetTokenExpiryMinutes:     15,
		ResetRateLimitRequests:      3,
		ResetRateLimitWindowMinutes: 60,
		SMTPHost:                    "smtp.example.com",
		SMTPPort:                    587,
		SMTPFromAddress:             "noreply@example.com",
		AppBaseURL:                  "https://example.com",
	}
	app, err := buildServer(opts)
	assert.Error(t, err)
	assert.Nil(t, app)
}

// TestRunInvokesHealthCheckPath verifies that run() takes the --health-check
// short-circuit and calls the injected health-check function. By stubbing
// healthCheckFunc we assert both (a) that the branch was taken and (b) that
// the specific exit code from the health check is propagated.
func TestRunInvokesHealthCheckPath(t *testing.T) {
	const sentinel = 0

	called := false
	t.Cleanup(restoreHealthCheckFunc(healthCheckFunc))
	healthCheckFunc = func() int {
		called = true
		return sentinel
	}

	code := run([]string{"app", "--health-check"})
	assert.True(t, called, "run() must invoke healthCheckFunc when --health-check is supplied")
	assert.Equal(t, sentinel, code, "run() must propagate the health-check exit code verbatim")
}

// TestRunInvokesHealthCheckPathFailure is the mirror of the success case and
// asserts that a failing health check yields exit code 1.
func TestRunInvokesHealthCheckPathFailure(t *testing.T) {
	const sentinel = 1

	t.Cleanup(restoreHealthCheckFunc(healthCheckFunc))
	healthCheckFunc = func() int { return sentinel }

	code := run([]string{"app", "--health-check"})
	assert.Equal(t, sentinel, code)
}

// restoreHealthCheckFunc returns a cleanup closure that restores the original
// healthCheckFunc — used with t.Cleanup to isolate tests that stub it.
func restoreHealthCheckFunc(original func() int) func() {
	return func() { healthCheckFunc = original }
}

// TestRunParseError verifies that run() returns 1 when options.ParseArgs
// rejects the supplied args. The previous version of this test was unreliable
// because it depended on options.Parse() reading os.Args (which under `go
// test` includes test-binary flags). Now that run() forwards args to
// options.ParseArgs directly, we can drive this deterministically.
//
// We force a parse error by clearing all required env vars and passing an
// arg slice that contains nothing but the program name, so every required
// field (ldap-server, base-dn, readonly-user, readonly-password) is missing.
func TestRunParseError(t *testing.T) {
	// Clear all required env vars so ParseArgs reports them as missing.
	t.Setenv("LDAP_SERVER", "")
	t.Setenv("LDAP_BASE_DN", "")
	t.Setenv("LDAP_READONLY_USER", "")
	t.Setenv("LDAP_READONLY_PASSWORD", "")

	// Run in a temp dir to guarantee no .env / .env.local is picked up.
	t.Chdir(t.TempDir())

	code := run([]string{"app"})
	assert.Equal(t, 1, code, "run() must return 1 when required options are missing")
}

// TestRunBuildServerError exercises the run() path where ParseArgs succeeds
// but buildServer fails (LDAP unreachable). This covers the initialization
// error branch and confirms run() does not call the health-check path when
// --health-check is absent.
func TestRunBuildServerError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow LDAP timeout test in short mode")
	}

	// Prevent environment from leaking in.
	t.Chdir(t.TempDir())

	// If the health-check branch is accidentally taken, this would panic.
	t.Cleanup(restoreHealthCheckFunc(healthCheckFunc))
	healthCheckFunc = func() int {
		t.Fatal("healthCheckFunc must NOT be invoked when --health-check is absent")
		return 0
	}

	code := run([]string{
		"app",
		"-ldap-server", "ldap://127.0.0.1:1",
		"-base-dn", "dc=example,dc=com",
		"-readonly-user", "cn=readonly,dc=example,dc=com",
		"-readonly-password", "secret",
	})
	assert.Equal(t, 1, code, "run() must return 1 when buildServer fails")
}

// BenchmarkRunHealthCheck benchmarks the health check operation.
func BenchmarkRunHealthCheck(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"alive"}`)) //nolint:errcheck // benchmark handler
	}))
	defer server.Close()

	b.ResetTimer()
	for b.Loop() {
		_ = testableRunHealthCheck(server.URL + "/health/live")
	}
}

// restoreRunSeams returns a cleanup closure restoring the buildServerFunc and
// shutdownContextFunc indirections after a test overrides them.
func restoreRunSeams(
	bs func(*options.Opts) (*fiber.App, error),
	sc func() (context.Context, context.CancelFunc),
) func() {
	return func() {
		buildServerFunc = bs
		shutdownContextFunc = sc
	}
}

// TestRunGracefulShutdown drives run() through the successful listen path with
// an already-canceled shutdown context, so fiber.Listen shuts down gracefully
// and run() returns 0. buildServerFunc is stubbed to avoid a real LDAP backend
// and shutdownContextFunc to avoid process signals.
func TestRunGracefulShutdown(t *testing.T) {
	t.Cleanup(restoreRunSeams(buildServerFunc, shutdownContextFunc))

	// Minimal app with no LDAP dependency.
	buildServerFunc = func(_ *options.Opts) (*fiber.App, error) {
		return fiber.New(), nil
	}
	// Cancel shortly after listen starts so the graceful-shutdown watcher,
	// which arms once the listener is bound, observes it and returns.
	shutdownContextFunc = func() (context.Context, context.CancelFunc) {
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		return ctx, cancel
	}

	done := make(chan int, 1)
	go func() {
		// Port 0 => an ephemeral free port, so the test never collides.
		done <- run([]string{
			"app",
			"-ldap-server", "ldap://127.0.0.1:389",
			"-base-dn", "dc=example,dc=com",
			"-readonly-user", "cn=readonly,dc=example,dc=com",
			"-readonly-password", "secret",
			"-port", "0",
		})
	}()

	select {
	case code := <-done:
		assert.Equal(t, 0, code, "run() must return 0 after a graceful shutdown")
	case <-time.After(10 * time.Second):
		t.Fatal("run() did not return after the shutdown context was canceled")
	}
}
