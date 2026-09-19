package rpchandler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
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
	want := reflect.ValueOf(guardTurnstile).Pointer()

	for method, guards := range methodPolicies {
		if len(guards) == 0 {
			t.Errorf("%s: policy is empty", method)

			continue
		}

		last := reflect.ValueOf(guards[len(guards)-1]).Pointer()
		if last != want {
			t.Errorf("%s: last guard is not guardTurnstile", method)
		}

		for i, g := range guards[:len(guards)-1] {
			if reflect.ValueOf(g).Pointer() == want {
				t.Errorf("%s: guardTurnstile also runs at position %d, ahead of a local check", method, i)
			}
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

			app := fiber.New()
			app.Post("/api/rpc", handler.Handle)

			got := postRPC(t, app, tt.body)
			if got.status != tt.wantStatus {
				t.Errorf("status = %d, want %d (body: %s)", got.status, tt.wantStatus, got.body)
			}
			if !strings.Contains(got.body, tt.wantBody) {
				t.Errorf("body = %s, want %q", got.body, tt.wantBody)
			}
			if strings.Contains(got.body, msgTurnstileVerificationFailed) {
				t.Errorf("Turnstile answered the request although a local guard should have stopped it: %s", got.body)
			}
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
