package rpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	ldap "github.com/netresearch/simple-ldap-go"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/options"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/resettoken"
)

// TestHandle tests the main Handle function routing and body parsing.
func TestHandle(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           string
		wantStatus     int
		wantSuccess    bool
		wantErrContain string
	}{
		{
			name:        "change-password routes correctly",
			method:      "change-password",
			body:        `{"method":"change-password","params":["testuser","OldPass123!","NewPass456!"]}`,
			wantStatus:  http.StatusOK,
			wantSuccess: true,
		},
		{
			name:           "unknown method returns error",
			method:         "invalid-method",
			body:           `{"method":"invalid-method","params":[]}`,
			wantStatus:     http.StatusBadRequest,
			wantSuccess:    false,
			wantErrContain: "method not found",
		},
		{
			name:           "request-password-reset without token store returns error",
			method:         "request-password-reset",
			body:           `{"method":"request-password-reset","params":["user@example.com"]}`,
			wantStatus:     http.StatusBadRequest,
			wantSuccess:    false,
			wantErrContain: "password reset feature not enabled",
		},
		{
			name:           "reset-password without token store returns error",
			method:         "reset-password",
			body:           `{"method":"reset-password","params":["token123","newpass"]}`,
			wantStatus:     http.StatusBadRequest,
			wantSuccess:    false,
			wantErrContain: "password reset feature not enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create handler with mock LDAP
			handler := createTestHandler()

			// Create Fiber app for testing
			app := fiber.New()
			app.Post("/api/rpc", handler.Handle)

			// Create test request
			req := httptest.NewRequest(http.MethodPost, "/api/rpc", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			// Execute request
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test failed: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			// Read response body
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read response body: %v", err)
			}

			// Parse response
			var response JSONRPCResponse
			if err := json.Unmarshal(bodyBytes, &response); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			// Check status code
			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}

			// Check success field
			if response.Success != tt.wantSuccess {
				t.Errorf("success = %v, want %v", response.Success, tt.wantSuccess)
			}

			// Check error message if expected
			if tt.wantErrContain != "" {
				if len(response.Data) == 0 || !strings.Contains(response.Data[0], tt.wantErrContain) {
					t.Errorf("expected error containing %q, got %v", tt.wantErrContain, response.Data)
				}
			}
		})
	}
}

// TestHandleInvalidJSON tests Handle with malformed JSON body.
func TestHandleInvalidJSON(t *testing.T) {
	handler := createTestHandler()

	app := fiber.New()
	app.Post("/api/rpc", handler.Handle)

	req := httptest.NewRequest(http.MethodPost, "/api/rpc", strings.NewReader(`{invalid json`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Invalid JSON should result in an error (fiber returns 422 for binding errors by default)
	if resp.StatusCode == http.StatusOK {
		t.Error("expected non-OK status for invalid JSON")
	}
}

// TestHandleChangePasswordError tests change-password with LDAP error.
func TestHandleChangePasswordError(t *testing.T) {
	handler := createTestHandlerWithLDAPError("LDAP connection failed")

	app := fiber.New()
	app.Post("/api/rpc", handler.Handle)

	body := `{"method":"change-password","params":["testuser","OldPass123!","NewPass456!"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/rpc", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	var response JSONRPCResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Success {
		t.Error("expected success = false for LDAP error")
	}
}

// TestSendSuccessResponse tests the sendSuccessResponse helper.
func TestSendSuccessResponse(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c fiber.Ctx) error {
		return sendSuccessResponse(c, []string{"result1", "result2"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var response JSONRPCResponse
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Error("expected success = true")
	}
	if len(response.Data) != 2 || response.Data[0] != "result1" {
		t.Errorf("unexpected data: %v", response.Data)
	}
}

// TestSendErrorResponse tests the sendErrorResponse helper.
func TestSendErrorResponse(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c fiber.Ctx) error {
		return sendErrorResponse(c, http.StatusBadRequest, "test error message")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	var response JSONRPCResponse
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Success {
		t.Error("expected success = false")
	}
	if len(response.Data) != 1 || response.Data[0] != "test error message" {
		t.Errorf("unexpected data: %v", response.Data)
	}
}

// TestHandleWithPasswordResetEnabled tests Handle when password reset is enabled.
func TestHandleWithPasswordResetEnabled(t *testing.T) {
	handler := createTestHandlerWithResetEnabled()

	app := fiber.New()
	app.Post("/api/rpc", handler.Handle)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "request-password-reset with services enabled",
			body:       `{"method":"request-password-reset","params":["user@example.com"]}`,
			wantStatus: http.StatusOK, // Will succeed (rate limiter allows, email service mocked)
		},
		{
			name:       "reset-password with services enabled",
			body:       `{"method":"reset-password","params":["validtoken","NewPass123!"]}`,
			wantStatus: http.StatusInternalServerError, // Token not found (expected)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/rpc", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test failed: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != tt.wantStatus {
				bodyBytes, readErr := io.ReadAll(resp.Body)
				if readErr != nil {
					t.Fatalf("failed to read response body: %v", readErr)
				}
				t.Errorf("status = %d, want %d (body: %s)", resp.StatusCode, tt.wantStatus, string(bodyBytes))
			}
		})
	}
}

// TestHandleIPRateLimited tests that Handle respects IP rate limiting.
func TestHandleIPRateLimited(t *testing.T) {
	handler := createTestHandlerWithIPLimiterBlocked()

	app := fiber.New()
	app.Post("/api/rpc", handler.Handle)

	body := `{"method":"change-password","params":["testuser","OldPass123!","NewPass456!"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/rpc", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d (rate limited)", resp.StatusCode, http.StatusInternalServerError)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	if !bytes.Contains(bodyBytes, []byte("too many")) {
		t.Errorf("expected 'too many' in response, got: %s", string(bodyBytes))
	}
}

// Helper functions to create test handlers

func createTestHandler() *Handler {
	mockLDAP := &mockHandlerLDAP{}
	opts := &options.Opts{
		MinLength:                  8,
		MinNumbers:                 1,
		MinSymbols:                 1,
		MinUppercase:               1,
		MinLowercase:               1,
		PasswordCanIncludeUsername: false,
	}
	return &Handler{
		ldap:      mockLDAP,
		opts:      opts,
		ipLimiter: &mockHandlerIPLimiter{allowed: true},
	}
}

func createTestHandlerWithLDAPError(errMsg string) *Handler {
	h := createTestHandler()
	h.ldap = &mockHandlerLDAP{changePasswordError: errMsg}
	return h
}

func createTestHandlerWithIPLimiterBlocked() *Handler {
	h := createTestHandler()
	h.ipLimiter = &mockHandlerIPLimiter{allowed: false}
	return h
}

func createTestHandlerWithResetEnabled() *Handler {
	h := createTestHandler()
	h.tokenStore = &mockHandlerTokenStore{}
	h.emailService = &mockHandlerEmailService{}
	h.rateLimiter = &mockHandlerRateLimiter{allowed: true}
	return h
}

// Mock implementations

type mockHandlerLDAP struct {
	changePasswordError string
}

func (m *mockHandlerLDAP) FindUserByMail(_ string) (*ldap.User, error) {
	email := "user@example.com"
	return &ldap.User{SAMAccountName: "testuser", Mail: &email}, nil
}

func (m *mockHandlerLDAP) ChangePasswordForSAMAccountName(_, _, _ string) error {
	if m.changePasswordError != "" {
		return ldap.NewLDAPError("ChangePassword", "ldap://localhost", errors.New(m.changePasswordError))
	}
	return nil
}

func (m *mockHandlerLDAP) ResetPasswordForSAMAccountName(_, _ string) error {
	return nil
}

type mockHandlerIPLimiter struct {
	allowed bool
}

func (m *mockHandlerIPLimiter) AllowRequest(_ string) bool {
	return m.allowed
}

type mockHandlerTokenStore struct{}

func (m *mockHandlerTokenStore) Store(_ *resettoken.ResetToken) error {
	return nil
}

func (m *mockHandlerTokenStore) Get(_ string) (*resettoken.ResetToken, error) {
	return nil, errors.New("token not found")
}

func (m *mockHandlerTokenStore) MarkUsed(_ string) error {
	return nil
}

func (m *mockHandlerTokenStore) Delete(_ string) error {
	return nil
}

func (m *mockHandlerTokenStore) CleanupExpired() int {
	return 0
}

func (m *mockHandlerTokenStore) Count() int {
	return 0
}

type mockHandlerEmailService struct{}

func (m *mockHandlerEmailService) SendResetEmail(_, _ string) error {
	return nil // Success
}

type mockHandlerRateLimiter struct {
	allowed bool
}

func (m *mockHandlerRateLimiter) AllowRequest(_ string) bool {
	return m.allowed
}

// TestSetIPLimiter tests the SetIPLimiter method.
func TestSetIPLimiter(t *testing.T) {
	handler := createTestHandler()
	// Initially has an IP limiter set
	if handler.ipLimiter == nil {
		t.Error("expected initial ipLimiter to be set")
	}

	// Set a new IP limiter
	newLimiter := &mockHandlerIPLimiter{allowed: false}
	handler.SetIPLimiter(newLimiter)

	if handler.ipLimiter != newLimiter {
		t.Error("SetIPLimiter did not update the ipLimiter")
	}
}

// TestSetIPLimiterNil tests setting nil IP limiter.
func TestSetIPLimiterNil(t *testing.T) {
	handler := createTestHandler()
	handler.SetIPLimiter(nil)

	if handler.ipLimiter != nil {
		t.Error("expected ipLimiter to be nil after setting nil")
	}
}

// TestHandleRequestPasswordResetRateLimited tests that rate limiting returns generic success
// (to prevent user enumeration - we don't reveal that the request was rate limited).
func TestHandleRequestPasswordResetRateLimited(t *testing.T) {
	handler := createTestHandlerWithResetEnabled()
	// Make rate limiter deny requests
	handler.rateLimiter = &mockHandlerRateLimiter{allowed: false}

	app := fiber.New()
	app.Post("/api/rpc", handler.Handle)

	body := `{"method":"request-password-reset","params":["user@example.com"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/rpc", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Rate limited requests still return 200 OK to prevent enumeration
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d (rate limited should return OK)", resp.StatusCode, http.StatusOK)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	var response JSONRPCResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Success is true even when rate limited (to prevent enumeration)
	if !response.Success {
		t.Error("expected success = true (rate limiting should not reveal itself)")
	}
}

// TestHandleResetPasswordSuccess tests successful password reset.
func TestHandleResetPasswordSuccess(t *testing.T) {
	handler := createTestHandlerWithResetEnabled()
	// Setup token store to return a valid token
	handler.tokenStore = &mockHandlerTokenStoreWithToken{}
	// Set up the resetLDAP client (required for password reset)
	handler.resetLDAP = &mockHandlerLDAP{}

	app := fiber.New()
	app.Post("/api/rpc", handler.Handle)

	body := `{"method":"reset-password","params":["validtoken","NewPass123!"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/rpc", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			t.Fatalf("failed to read response body: %v", readErr)
		}
		t.Errorf("status = %d, want %d (body: %s)", resp.StatusCode, http.StatusOK, string(bodyBytes))
	}
}

// TestHandleChangePasswordSuccess tests successful password change with full response validation.
func TestHandleChangePasswordSuccess(t *testing.T) {
	handler := createTestHandler()

	app := fiber.New()
	app.Post("/api/rpc", handler.Handle)

	body := `{"method":"change-password","params":["testuser","OldPass123!","NewPass456!"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/rpc", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	var response JSONRPCResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Error("expected success = true")
	}
	if len(response.Data) == 0 {
		t.Error("expected non-empty data")
	}
}

// TestSendSuccessResponseWithEmptyData tests sendSuccessResponse with empty data.
func TestSendSuccessResponseWithEmptyData(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c fiber.Ctx) error {
		return sendSuccessResponse(c, []string{})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var response JSONRPCResponse
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Error("expected success = true")
	}
	if len(response.Data) != 0 {
		t.Errorf("expected empty data, got: %v", response.Data)
	}
}

// TestSendErrorResponseWithDifferentStatusCodes tests sendErrorResponse with various status codes.
func TestSendErrorResponseWithDifferentStatusCodes(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
		message    string
	}{
		{"internal server error", http.StatusInternalServerError, "internal error"},
		{"not found", http.StatusNotFound, "not found"},
		{"unauthorized", http.StatusUnauthorized, "unauthorized"},
		{"forbidden", http.StatusForbidden, "forbidden"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/test", func(c fiber.Ctx) error {
				return sendErrorResponse(c, tc.statusCode, tc.message)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test failed: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != tc.statusCode {
				t.Errorf("status = %d, want %d", resp.StatusCode, tc.statusCode)
			}

			var response JSONRPCResponse
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read response body: %v", err)
			}
			if err := json.Unmarshal(bodyBytes, &response); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if response.Success {
				t.Error("expected success = false")
			}
			if len(response.Data) != 1 || response.Data[0] != tc.message {
				t.Errorf("unexpected data: %v", response.Data)
			}
		})
	}
}

// mockHandlerTokenStoreWithToken is a token store that returns a valid token.
type mockHandlerTokenStoreWithToken struct{}

func (m *mockHandlerTokenStoreWithToken) Store(_ *resettoken.ResetToken) error {
	return nil
}

func (m *mockHandlerTokenStoreWithToken) Get(_ string) (*resettoken.ResetToken, error) {
	return &resettoken.ResetToken{
		Token:     "validtoken",
		Username:  "testuser",
		Email:     "test@example.com",
		Used:      false,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}, nil
}

func (m *mockHandlerTokenStoreWithToken) MarkUsed(_ string) error {
	return nil
}

func (m *mockHandlerTokenStoreWithToken) Delete(_ string) error {
	return nil
}

func (m *mockHandlerTokenStoreWithToken) CleanupExpired() int {
	return 0
}

func (m *mockHandlerTokenStoreWithToken) Count() int {
	return 1
}
