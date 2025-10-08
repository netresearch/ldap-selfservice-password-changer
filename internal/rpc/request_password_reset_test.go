package rpc

import (
	"fmt"
	"testing"
	"time"

	ldap "github.com/netresearch/simple-ldap-go"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/ratelimit"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/resettoken"
)

// Mock email service for testing
type mockEmailService struct {
	lastTo    string
	lastToken string
	sendError error
}

func (m *mockEmailService) SendResetEmail(to, token string) error {
	m.lastTo = to
	m.lastToken = token
	return m.sendError
}

// Mock LDAP client for testing
type mockLDAPClient struct {
	findUserByMailError error
	users               map[string]*ldap.User
}

func (m *mockLDAPClient) FindUserByMail(mail string) (*ldap.User, error) {
	if m.findUserByMailError != nil {
		return nil, m.findUserByMailError
	}
	if user, ok := m.users[mail]; ok {
		return user, nil
	}
	return nil, fmt.Errorf("user not found")
}

func (m *mockLDAPClient) ChangePasswordForSAMAccountName(sAMAccountName, oldPassword, newPassword string) error {
	return nil
}

func (m *mockLDAPClient) ResetPasswordForSAMAccountName(sAMAccountName, newPassword string) error {
	return nil
}

func TestRequestPasswordResetValidEmail(t *testing.T) {
	// Setup
	tokenStore := resettoken.NewStore()
	limiter := ratelimit.NewLimiter(3, 60*time.Minute)
	mockEmail := &mockEmailService{}
	mockLDAP := &mockLDAPClient{
		users: map[string]*ldap.User{
			"test@example.com": {
				SAMAccountName: "testuser",
			},
		},
	}

	handler := &Handler{
		ldap:         mockLDAP,
		tokenStore:   tokenStore,
		emailService: mockEmail,
		rateLimiter:  limiter,
	}

	// Test valid email request
	params := []string{"test@example.com"}
	result, err := handler.requestPasswordReset(params)

	if err != nil {
		t.Fatalf("requestPasswordReset() unexpected error: %v", err)
	}

	// Should return generic success message
	if len(result) != 1 {
		t.Errorf("Expected 1 result, got %d", len(result))
	}

	expectedMsg := "If an account exists, a reset email has been sent"
	if result[0] != expectedMsg {
		t.Errorf("Result = %q, want %q", result[0], expectedMsg)
	}

	// Verify email was sent
	if mockEmail.lastTo != "test@example.com" {
		t.Errorf("Email sent to %q, want test@example.com", mockEmail.lastTo)
	}

	// Verify token was generated and stored
	if mockEmail.lastToken == "" {
		t.Error("No token generated")
	}

	// Verify token exists in store
	token, err := tokenStore.Get(mockEmail.lastToken)
	if err != nil {
		t.Errorf("Token not found in store: %v", err)
	}

	if token.Email != "test@example.com" {
		t.Errorf("Token email = %q, want test@example.com", token.Email)
	}
}

func TestRequestPasswordResetInvalidEmail(t *testing.T) {
	tokenStore := resettoken.NewStore()
	limiter := ratelimit.NewLimiter(3, 60*time.Minute)
	mockEmail := &mockEmailService{}
	mockLDAP := &mockLDAPClient{
		users: map[string]*ldap.User{},
	}

	handler := &Handler{
		ldap:         mockLDAP,
		tokenStore:   tokenStore,
		emailService: mockEmail,
		rateLimiter:  limiter,
	}

	tests := []struct {
		name  string
		email string
	}{
		{"empty", ""},
		{"no @", "notanemail"},
		{"no domain", "user@"},
		{"no user", "@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := []string{tt.email}
			result, err := handler.requestPasswordReset(params)

			// Should still return generic success (don't reveal invalid email)
			if err != nil {
				t.Errorf("Should not error on invalid email, got: %v", err)
			}

			if len(result) != 1 {
				t.Errorf("Expected 1 result, got %d", len(result))
			}
		})
	}
}

func TestRequestPasswordResetRateLimit(t *testing.T) {
	tokenStore := resettoken.NewStore()
	limiter := ratelimit.NewLimiter(2, 60*time.Minute) // Only 2 requests allowed
	mockEmail := &mockEmailService{}
	mockLDAP := &mockLDAPClient{
		users: map[string]*ldap.User{
			"ratelimit@example.com": {
				SAMAccountName: "ratelimituser",
			},
		},
	}

	handler := &Handler{
		ldap:         mockLDAP,
		tokenStore:   tokenStore,
		emailService: mockEmail,
		rateLimiter:  limiter,
	}

	email := "ratelimit@example.com"

	// First 2 requests should succeed
	for i := 0; i < 2; i++ {
		params := []string{email}
		_, err := handler.requestPasswordReset(params)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i+1, err)
		}
	}

	// 3rd request should be rate limited but still return success
	params := []string{email}
	result, err := handler.requestPasswordReset(params)

	if err != nil {
		t.Errorf("Rate limited request should not error, got: %v", err)
	}

	// Should still return generic success message
	if len(result) != 1 {
		t.Errorf("Expected 1 result, got %d", len(result))
	}

	// But email should NOT have been sent
	if mockEmail.lastTo == email {
		// This is tricky - we need to track how many emails were sent
		// For now, we'll check that token count is only 2
		if tokenStore.Count() > 2 {
			t.Errorf("Too many tokens generated: %d, want 2", tokenStore.Count())
		}
	}
}

func TestRequestPasswordResetInvalidArgumentCount(t *testing.T) {
	tokenStore := resettoken.NewStore()
	limiter := ratelimit.NewLimiter(3, 60*time.Minute)
	mockEmail := &mockEmailService{}
	mockLDAP := &mockLDAPClient{
		users: map[string]*ldap.User{},
	}

	handler := &Handler{
		ldap:         mockLDAP,
		tokenStore:   tokenStore,
		emailService: mockEmail,
		rateLimiter:  limiter,
	}

	tests := []struct {
		name   string
		params []string
	}{
		{"no params", []string{}},
		{"too many params", []string{"email@example.com", "extra"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := handler.requestPasswordReset(tt.params)
			if err != ErrInvalidArgumentCount {
				t.Errorf("Expected ErrInvalidArgumentCount, got: %v", err)
			}
		})
	}
}

func TestRequestPasswordResetTokenExpiration(t *testing.T) {
	tokenStore := resettoken.NewStore()
	limiter := ratelimit.NewLimiter(3, 60*time.Minute)
	mockEmail := &mockEmailService{}
	mockLDAP := &mockLDAPClient{
		users: map[string]*ldap.User{
			"test@example.com": {
				SAMAccountName: "testuser",
			},
		},
	}

	handler := &Handler{
		ldap:         mockLDAP,
		tokenStore:   tokenStore,
		emailService: mockEmail,
		rateLimiter:  limiter,
		// Would need to inject tokenExpiryMinutes config
	}

	params := []string{"test@example.com"}
	_, err := handler.requestPasswordReset(params)
	if err != nil {
		t.Fatalf("requestPasswordReset() error: %v", err)
	}

	// Verify token has expiration set (15 minutes from now)
	token, _ := tokenStore.Get(mockEmail.lastToken)

	expiryDuration := token.ExpiresAt.Sub(token.CreatedAt)
	expectedDuration := 15 * time.Minute

	// Allow 1 second tolerance for test execution time
	if expiryDuration < expectedDuration-time.Second || expiryDuration > expectedDuration+time.Second {
		t.Errorf("Token expiry duration = %v, want ~%v", expiryDuration, expectedDuration)
	}
}

func TestRequestPasswordResetEmailFailure(t *testing.T) {
	tokenStore := resettoken.NewStore()
	limiter := ratelimit.NewLimiter(3, 60*time.Minute)
	mockEmail := &mockEmailService{
		sendError: fmt.Errorf("SMTP connection failed"),
	}
	mockLDAP := &mockLDAPClient{
		users: map[string]*ldap.User{
			"test@example.com": {
				SAMAccountName: "testuser",
			},
		},
	}

	handler := &Handler{
		ldap:         mockLDAP,
		tokenStore:   tokenStore,
		emailService: mockEmail,
		rateLimiter:  limiter,
	}

	params := []string{"test@example.com"}
	result, err := handler.requestPasswordReset(params)

	// Should still return success (don't reveal SMTP failure to user)
	if err != nil {
		t.Errorf("Should not error on email failure, got: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("Expected 1 result, got %d", len(result))
	}

	// Token should still be created (for retry purposes)
	if tokenStore.Count() != 1 {
		t.Errorf("Token count = %d, want 1", tokenStore.Count())
	}
}
