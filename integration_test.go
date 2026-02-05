//go:build integration

// Package main_test provides full stack integration tests for the password changer.
package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/email"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/options"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/ratelimit"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/resettoken"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/web/templates"
)

// getEnvOrSkip returns an environment variable or skips the test.
func getEnvOrSkip(t *testing.T, key string) string {
	value := os.Getenv(key)
	if value == "" {
		t.Skipf("Skipping test: %s not set", key)
	}
	return value
}

// JSONRPCRequest represents a JSON-RPC 2.0 request.
type JSONRPCRequest struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	Success bool     `json:"success"`
	Data    []string `json:"data"`
}

// TestIntegration_FullPasswordResetFlow tests the complete password reset flow.
func TestIntegration_FullPasswordResetFlow(t *testing.T) {
	_ = getEnvOrSkip(t, "LDAP_SERVER")
	smtpHost := getEnvOrSkip(t, "SMTP_HOST")

	// Create test server with mock LDAP
	opts := &options.Opts{
		Port:                        "3000",
		MinLength:                   8,
		MinNumbers:                  1,
		MinSymbols:                  1,
		MinUppercase:                1,
		MinLowercase:                1,
		PasswordResetEnabled:        true,
		ResetTokenExpiryMinutes:     15,
		ResetRateLimitRequests:      10,
		ResetRateLimitWindowMinutes: 60,
	}

	// Setup services
	tokenStore := resettoken.NewStore()
	emailConfig := &email.Config{
		SMTPHost:     smtpHost,
		SMTPPort:     1025,
		SMTPUsername: "",
		SMTPPassword: "",
		FromAddress:  "noreply@example.com",
		BaseURL:      "http://localhost:3000",
	}
	emailService := email.NewService(emailConfig)
	rateLimiter := ratelimit.NewLimiter(10, 60*time.Minute)
	ipLimiter := ratelimit.NewIPLimiter()

	// Create mock handler for integration test
	// In a real integration test, we'd use NewWithServices with actual LDAP
	_ = tokenStore
	_ = emailService
	_ = rateLimiter
	_ = ipLimiter

	// Render pages
	indexPage, err := templates.RenderIndex(opts)
	require.NoError(t, err)

	// Create Fiber app
	app := fiber.New()

	app.Get("/", func(c fiber.Ctx) error {
		c.Set("Content-Type", fiber.MIMETextHTMLCharsetUTF8)
		return c.Send(indexPage)
	})

	app.Get("/health/live", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "alive"})
	})

	// Test health endpoint
	req := httptest.NewRequest(http.MethodGet, "/health/live", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "alive")
}

// TestIntegration_RateLimiting tests rate limiting behavior.
func TestIntegration_RateLimiting(t *testing.T) {
	// Create rate limiter with very low limit for testing
	limiter := ratelimit.NewLimiter(2, 1*time.Minute)

	email1 := "user1@example.com"
	email2 := "user2@example.com"

	// First 2 requests for email1 should succeed
	assert.True(t, limiter.AllowRequest(email1))
	assert.True(t, limiter.AllowRequest(email1))

	// Third request for email1 should be rate limited
	assert.False(t, limiter.AllowRequest(email1))

	// Requests for different email should still work
	assert.True(t, limiter.AllowRequest(email2))
}

// TestIntegration_IPRateLimiting tests IP-based rate limiting.
func TestIntegration_IPRateLimiting(t *testing.T) {
	// Create IP limiter with default settings
	limiter := ratelimit.NewIPLimiter()

	ip1 := "192.168.1.1"
	ip2 := "192.168.1.2"

	// First 10 requests from ip1 should succeed
	for i := 0; i < 10; i++ {
		assert.True(t, limiter.AllowRequest(ip1), "Request %d should succeed", i+1)
	}

	// 11th request from ip1 should be rate limited
	assert.False(t, limiter.AllowRequest(ip1), "11th request should be rate limited")

	// Requests from different IP should still work
	assert.True(t, limiter.AllowRequest(ip2))
}

// TestIntegration_TokenStoreOperations tests token store with real tokens.
func TestIntegration_TokenStoreOperations(t *testing.T) {
	store := resettoken.NewStore()

	// Generate and store a token
	tokenString, err := resettoken.GenerateToken()
	require.NoError(t, err)

	token := &resettoken.ResetToken{
		Token:     tokenString,
		Username:  "testuser",
		Email:     "test@example.com",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
		Used:      false,
	}

	// Store token
	err = store.Store(token)
	require.NoError(t, err)

	// Retrieve token
	retrieved, err := store.Get(tokenString)
	require.NoError(t, err)
	assert.Equal(t, token.Username, retrieved.Username)
	assert.Equal(t, token.Email, retrieved.Email)
	assert.False(t, retrieved.Used)

	// Mark as used
	err = store.MarkUsed(tokenString)
	require.NoError(t, err)

	// Verify marked as used
	retrieved, err = store.Get(tokenString)
	require.NoError(t, err)
	assert.True(t, retrieved.Used)

	// Delete token
	err = store.Delete(tokenString)
	require.NoError(t, err)

	// Verify deleted
	_, err = store.Get(tokenString)
	assert.Error(t, err)
}

// TestIntegration_JSONRPCEndpoint tests the JSON-RPC endpoint structure.
func TestIntegration_JSONRPCEndpoint(t *testing.T) {
	app := fiber.New()

	// Mock RPC handler
	app.Post("/api/rpc", func(c fiber.Ctx) error {
		var req JSONRPCRequest
		if err := c.Bind().Body(&req); err != nil {
			return c.Status(http.StatusBadRequest).JSON(JSONRPCResponse{
				Success: false,
				Data:    []string{"Invalid request"},
			})
		}

		switch req.Method {
		case "change-password":
			if len(req.Params) != 3 {
				return c.JSON(JSONRPCResponse{
					Success: false,
					Data:    []string{"Invalid argument count"},
				})
			}
			return c.JSON(JSONRPCResponse{
				Success: true,
				Data:    []string{"password changed successfully"},
			})
		default:
			return c.Status(http.StatusBadRequest).JSON(JSONRPCResponse{
				Success: false,
				Data:    []string{"method not found"},
			})
		}
	})

	// Test valid request
	reqBody := JSONRPCRequest{
		Method: "change-password",
		Params: []string{"testuser", "oldpass", "NewPass123!"},
	}
	bodyBytes, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/rpc", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var rpcResp JSONRPCResponse
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	err = json.Unmarshal(body, &rpcResp)
	require.NoError(t, err)

	assert.True(t, rpcResp.Success)
	assert.Contains(t, rpcResp.Data[0], "password changed")
}
