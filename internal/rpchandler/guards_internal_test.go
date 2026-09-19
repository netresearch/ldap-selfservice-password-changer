package rpchandler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

// TestTurnstileIsTheLastGuardOfEveryPolicy pins the ordering property the
// policies exist for: Turnstile is the only guard that leaves the process, so
// every local check has to come before it. A policy that puts a local check
// after it would let an unauthenticated flood turn into outbound requests to
// Cloudflare.
func TestTurnstileIsTheLastGuardOfEveryPolicy(t *testing.T) {
	for method, steps := range methodPolicies {
		if len(steps) == 0 {
			t.Errorf("%s: policy is empty", method)

			continue
		}

		if last := steps[len(steps)-1].name; last != turnstileGuard {
			t.Errorf("%s: last guard is %q, want %q", method, last, turnstileGuard)
		}

		for i, step := range steps[:len(steps)-1] {
			if step.name == turnstileGuard {
				t.Errorf("%s: %q also runs at position %d, ahead of a local check", method, turnstileGuard, i)
			}
		}
	}
}

// TestEveryDispatchableMethodHasAPolicy is the property the guard chain rests
// on: a method that can be dispatched but has no policy runs with no
// cross-cutting checks at all. Comparing the two key sets fails the day
// somebody adds a method to one map and not the other.
func TestEveryDispatchableMethodHasAPolicy(t *testing.T) {
	for method := range methodHandlers {
		if _, ok := methodPolicies[method]; !ok {
			t.Errorf("%s is dispatchable but has no guard policy", method)
		}
	}

	for method := range methodPolicies {
		if _, ok := methodHandlers[method]; !ok {
			t.Errorf("%s has a guard policy but cannot be dispatched", method)
		}
	}
}

// TestUnknownMethodIsRejectedBeforeGuards is the other half of that property:
// a method with no policy runs no guards, so it must not reach dispatch
// either.
func TestUnknownMethodIsRejectedBeforeGuards(t *testing.T) {
	handler := createTestHandlerWithResetEnabled()

	app := fiber.New()
	app.Post("/api/rpc", handler.Handle)

	got := postRPC(t, app, `{"method":"not-a-method","params":[]}`)
	if got.status != http.StatusBadRequest {
		t.Errorf("status = %d, want %d (body: %s)", got.status, http.StatusBadRequest, got.body)
	}
	if !strings.Contains(got.body, "method not found") {
		t.Errorf("body = %s, want the method-not-found message", got.body)
	}
}

// TestLocalGuardsRunBeforeTurnstile drives each method with the per-IP limiter
// denying and Turnstile enabled without a token. Both would reject the
// request, so the answer says which ran first: the limiter's answer means the
// local check won, a 403 would mean Turnstile did.
func TestLocalGuardsRunBeforeTurnstile(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "change-password",
			body:       `{"method":"change-password","params":["testuser","OldPass123!","NewPass456!"]}`,
			wantStatus: http.StatusTooManyRequests,
			wantBody:   "too many password change attempts",
		},
		{
			name:       "request-password-reset",
			body:       `{"method":"request-password-reset","params":["user@example.com"]}`,
			wantStatus: http.StatusOK,
			wantBody:   msgResetEmailSent,
		},
		{
			name:       "reset-password",
			body:       `{"method":"reset-password","params":["validtoken","NewPass123!"]}`,
			wantStatus: http.StatusTooManyRequests,
			wantBody:   "too many password reset attempts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := createTestHandlerWithResetEnabled()
			handler.opts.CfTurnstileEnabled = true
			handler.ipLimiter = &mockHandlerIPLimiter{allowed: false}

			assertAnsweredByALocalGuard(t, newGuardApp(handler), tt.body, tt.wantStatus, tt.wantBody)
		})
	}
}

// TestResetRequestIdentifierGuardRunsBeforeTurnstile covers the one local
// guard that is not the per-IP limiter: the per-identifier bucket has to be
// consulted before verification too, or a junk token would cost an outbound
// call per attempt.
func TestResetRequestIdentifierGuardRunsBeforeTurnstile(t *testing.T) {
	handler := createTestHandlerWithResetEnabled()
	handler.opts.CfTurnstileEnabled = true
	handler.rateLimiter = &mockHandlerRateLimiter{allowed: false}

	app := fiber.New()
	app.Post("/api/rpc", handler.Handle)

	got := postRPC(t, app, `{"method":"request-password-reset","params":["user@example.com"]}`)
	if got.status != http.StatusOK {
		t.Errorf("status = %d, want %d (body: %s)", got.status, http.StatusOK, got.body)
	}
	if !strings.Contains(got.body, msgResetEmailSent) {
		t.Errorf("body = %s, want the generic success message", got.body)
	}
	if strings.Contains(got.body, msgTurnstileVerificationFailed) {
		t.Errorf("Turnstile answered the request although the identifier bucket should have stopped it: %s", got.body)
	}
}

// TestMalformedRequestIsRefusedBeforeTheLimiters keeps the parameter checks of
// change-password and request-password-reset ahead of the limiters, where they
// were before the guard chain: a malformed request must not spend a rate-limit
// slot or produce an outbound verification, and it must answer with the
// argument error rather than with whatever the next guard would have said.
//
// reset-password is deliberately absent: its parameter check sits in the
// method and ran after the limiter before this chain existed too, so a
// wrong-count request there does spend a slot. Adding a guard would change a
// 429 into a 500.
func TestMalformedRequestIsRefusedBeforeTheLimiters(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "change-password with two parameters",
			body:       `{"method":"change-password","params":["testuser","OldPass123!"]}`,
			wantStatus: http.StatusInternalServerError,
			wantBody:   ErrInvalidArgumentCount.Error(),
		},
		{
			name:       "request-password-reset with no parameters",
			body:       `{"method":"request-password-reset","params":[]}`,
			wantStatus: http.StatusInternalServerError,
			wantBody:   ErrInvalidArgumentCount.Error(),
		},
		{
			name:       "request-password-reset with an over-long identifier",
			body:       `{"method":"request-password-reset","params":["` + strings.Repeat("a", 255) + `@example.com"]}`,
			wantStatus: http.StatusOK,
			wantBody:   msgResetEmailSent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := createTestHandlerWithResetEnabled()
			handler.opts.CfTurnstileEnabled = true

			limiter := &countingIPLimiter{allowed: true}
			handler.ipLimiter = limiter
			identifiers := &countingRateLimiter{allowed: true}
			handler.rateLimiter = identifiers

			assertAnsweredByALocalGuard(t, newGuardApp(handler), tt.body, tt.wantStatus, tt.wantBody)

			if limiter.calls != 0 {
				t.Errorf("the per-IP limiter was consulted %d times for a malformed request", limiter.calls)
			}
			if identifiers.calls != 0 {
				t.Errorf("the per-identifier limiter was consulted %d times for a malformed request", identifiers.calls)
			}
		})
	}
}

// TestChangePasswordWithoutIPLimiterIsServed covers the nil limiter, which is
// the state New() leaves the handler in until SetIPLimiter is called. It
// replaces TestChangePasswordNoIPLimiter, which drove changePassword directly
// and stopped covering anything once the limiter moved into a guard.
func TestChangePasswordWithoutIPLimiterIsServed(t *testing.T) {
	handler := createTestHandler()
	handler.ipLimiter = nil

	app := fiber.New()
	app.Post("/api/rpc", handler.Handle)

	got := postRPC(t, app, `{"method":"change-password","params":["testuser","OldPass123!","NewPass456!"]}`)
	if got.status != http.StatusOK {
		t.Errorf("status = %d, want %d (body: %s)", got.status, http.StatusOK, got.body)
	}
}

// countingIPLimiter answers a fixed verdict and counts how often it was asked.
type countingIPLimiter struct {
	allowed bool
	calls   int
}

func (m *countingIPLimiter) AllowRequest(_ string) bool {
	m.calls++

	return m.allowed
}

// countingRateLimiter does the same for the per-identifier limiter.
type countingRateLimiter struct {
	allowed bool
	calls   int
}

func (m *countingRateLimiter) AllowRequest(_ string) bool {
	m.calls++

	return m.allowed
}

// TestGuardsAreSkippedWhenTurnstileIsDisabled keeps the other direction
// honest: with the feature off, a request with no token is served.
func TestGuardsAreSkippedWhenTurnstileIsDisabled(t *testing.T) {
	handler := createTestHandler()
	handler.opts.CfTurnstileEnabled = false

	app := fiber.New()
	app.Post("/api/rpc", handler.Handle)

	got := postRPC(
		t,
		app,
		`{"method":"change-password","params":["testuser","OldPass123!","NewPass456!"]}`,
	)
	if got.status != http.StatusOK {
		t.Errorf("status = %d, want %d (body: %s)", got.status, http.StatusOK, got.body)
	}

	// The literal, not msgPasswordChanged: this is the text a user reads, and
	// an assertion through the constant would follow it wherever it went.
	if !strings.Contains(got.body, "password changed successfully") {
		t.Errorf("body = %s, want the success message", got.body)
	}
}

// newGuardApp mounts the handler on the one route the guards run for.
func newGuardApp(h *Handler) *fiber.App {
	app := fiber.New()
	app.Post("/api/rpc", h.Handle)

	return app
}

// assertAnsweredByALocalGuard posts the body and checks which guard answered:
// the expected status and message, and — the point of these tests — that the
// answer did not come from Turnstile, which would mean a local check ran too
// late or not at all.
func assertAnsweredByALocalGuard(t *testing.T, app *fiber.App, body string, wantStatus int, wantBody string) {
	t.Helper()

	got := postRPC(t, app, body)
	if got.status != wantStatus {
		t.Errorf("status = %d, want %d (body: %s)", got.status, wantStatus, got.body)
	}
	if !strings.Contains(got.body, wantBody) {
		t.Errorf("body = %s, want %q", got.body, wantBody)
	}
	if strings.Contains(got.body, msgTurnstileVerificationFailed) {
		t.Errorf("Turnstile answered the request although a local guard should have stopped it: %s", got.body)
	}
}

// rpcResponse is what an RPC call answered: a status and a body, named so that
// a caller cannot mix them up.
type rpcResponse struct {
	status int
	body   string
}

// postRPC posts one JSON-RPC body through the app and returns the response.
func postRPC(t *testing.T, app *fiber.App, body string) rpcResponse {
	t.Helper()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/rpc", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading response body failed: %v", err)
	}

	return rpcResponse{status: resp.StatusCode, body: string(raw)}
}

// postResetRequest posts one request-password-reset call for an identifier.
func postResetRequest(t *testing.T, app *fiber.App, identifier string) rpcResponse {
	t.Helper()

	return postRPC(t, app, fmt.Sprintf(`{"method":"request-password-reset","params":[%q]}`, identifier))
}
