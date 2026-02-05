package rpc

import (
	"errors"
	"strings"
	"testing"
	"time"

	ldap "github.com/netresearch/simple-ldap-go"
	"github.com/stretchr/testify/require"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/options"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/resettoken"
)

const errInvalidOrExpiredToken = "invalid or expired token"

// Mock LDAP client for testing.
type mockResetLDAPClient struct {
	changePasswordError error
	resetPasswordError  error
}

func (m *mockResetLDAPClient) FindUserByMail(_ string) (*ldap.User, error) {
	return &ldap.User{SAMAccountName: "testuser"}, nil
}

func (m *mockResetLDAPClient) ChangePasswordForSAMAccountName(_, _, _ string) error {
	return m.changePasswordError
}

func (m *mockResetLDAPClient) ResetPasswordForSAMAccountName(_, _ string) error {
	return m.resetPasswordError
}

func TestResetPasswordValidToken(t *testing.T) {
	// Setup
	tokenStore := resettoken.NewStore()
	opts := &options.Opts{
		MinLength:                  8,
		MinNumbers:                 1,
		MinSymbols:                 1,
		MinUppercase:               1,
		MinLowercase:               1,
		PasswordCanIncludeUsername: false,
	}
	mockLDAP := &mockResetLDAPClient{}

	handler := &Handler{
		ldap:       mockLDAP,
		resetLDAP:  mockLDAP, // Use same mock for reset operations
		tokenStore: tokenStore,
		opts:       opts,
	}

	// Create a valid token
	token := &resettoken.ResetToken{
		Token:     "valid-token-123",
		Username:  "testuser",
		Email:     "test@example.com",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
		Used:      false,
	}
	err := tokenStore.Store(token)
	require.NoError(t, err)

	// Test password reset with valid token and password
	params := []string{"valid-token-123", "NewPass123!"}
	result, err := handler.resetPassword(params)
	require.NoError(t, err)

	if len(result) != 1 {
		t.Errorf("Expected 1 result, got %d", len(result))
	}

	expectedMsg := "Password reset successfully. You can now login."
	if result[0] != expectedMsg {
		t.Errorf("Result = %q, want %q", result[0], expectedMsg)
	}

	// Verify token is marked as used
	updatedToken, err := tokenStore.Get("valid-token-123")
	require.NoError(t, err)
	if !updatedToken.Used {
		t.Error("Token should be marked as used")
	}
}

func TestResetPasswordInvalidToken(t *testing.T) {
	tokenStore := resettoken.NewStore()
	opts := &options.Opts{MinLength: 8}
	mockLDAP := &mockResetLDAPClient{}

	handler := &Handler{
		ldap:       mockLDAP,
		resetLDAP:  mockLDAP, // Use same mock for reset operations
		tokenStore: tokenStore,
		opts:       opts,
	}

	params := []string{"nonexistent-token", "NewPass123!"}
	_, err := handler.resetPassword(params)

	if err == nil {
		t.Error("Expected error for invalid token")
	}

	expectedErr := errInvalidOrExpiredToken
	if err.Error() != expectedErr {
		t.Errorf("Error = %q, want %q", err.Error(), expectedErr)
	}
}

func TestResetPasswordExpiredToken(t *testing.T) {
	tokenStore := resettoken.NewStore()
	opts := &options.Opts{MinLength: 8}
	mockLDAP := &mockResetLDAPClient{}

	handler := &Handler{
		ldap:       mockLDAP,
		resetLDAP:  mockLDAP, // Use same mock for reset operations
		tokenStore: tokenStore,
		opts:       opts,
	}

	// Create an expired token
	token := &resettoken.ResetToken{
		Token:     "expired-token",
		Username:  "testuser",
		Email:     "test@example.com",
		CreatedAt: time.Now().Add(-20 * time.Minute),
		ExpiresAt: time.Now().Add(-5 * time.Minute), // Expired 5 minutes ago
		Used:      false,
	}
	err := tokenStore.Store(token)
	require.NoError(t, err)

	params := []string{"expired-token", "NewPass123!"}
	_, err = handler.resetPassword(params)

	if err == nil {
		t.Error("Expected error for expired token")
	}

	if err.Error() != errInvalidOrExpiredToken {
		t.Errorf("Error = %q, want %q", err.Error(), errInvalidOrExpiredToken)
	}
}

func TestResetPasswordUsedToken(t *testing.T) {
	tokenStore := resettoken.NewStore()
	opts := &options.Opts{MinLength: 8}
	mockLDAP := &mockResetLDAPClient{}

	handler := &Handler{
		ldap:       mockLDAP,
		resetLDAP:  mockLDAP, // Use same mock for reset operations
		tokenStore: tokenStore,
		opts:       opts,
	}

	// Create a used token
	token := &resettoken.ResetToken{
		Token:     "used-token",
		Username:  "testuser",
		Email:     "test@example.com",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
		Used:      true, // Already used
	}
	err := tokenStore.Store(token)
	require.NoError(t, err)

	params := []string{"used-token", "NewPass123!"}
	_, err = handler.resetPassword(params)

	if err == nil {
		t.Error("Expected error for used token")
	}

	if err.Error() != errInvalidOrExpiredToken {
		t.Errorf("Error = %q, want %q", err.Error(), errInvalidOrExpiredToken)
	}
}

func TestResetPasswordPolicyViolation(t *testing.T) {
	tokenStore := resettoken.NewStore()
	opts := &options.Opts{
		MinLength:    8,
		MinNumbers:   1,
		MinSymbols:   1,
		MinUppercase: 1,
		MinLowercase: 1,
	}
	mockLDAP := &mockResetLDAPClient{}

	handler := &Handler{
		ldap:       mockLDAP,
		resetLDAP:  mockLDAP, // Use same mock for reset operations
		tokenStore: tokenStore,
		opts:       opts,
	}

	// Create a valid token
	token := &resettoken.ResetToken{
		Token:     "test-token",
		Username:  "testuser",
		Email:     "test@example.com",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
		Used:      false,
	}
	err := tokenStore.Store(token)
	require.NoError(t, err)

	tests := []struct {
		name     string
		password string
		wantErr  string
	}{
		{"too short", "Short1!", "the new password must be at least 8 characters long"},
		{"no numbers", "Password!", "the new password must contain at least 1 number"},
		{"no symbols", "Password1", "the new password must contain at least 1 symbol"},
		{"no uppercase", "password1!", "the new password must contain at least 1 uppercase letter"},
		{"no lowercase", "PASSWORD1!", "the new password must contain at least 1 lowercase letter"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := []string{"test-token", tt.password}
			_, err := handler.resetPassword(params)

			if err == nil {
				t.Error("Expected password policy error")
			}

			if err.Error() != tt.wantErr {
				t.Errorf("Error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestResetPasswordInvalidArgumentCount(t *testing.T) {
	tokenStore := resettoken.NewStore()
	opts := &options.Opts{}
	mockLDAP := &mockResetLDAPClient{}

	handler := &Handler{
		ldap:       mockLDAP,
		resetLDAP:  mockLDAP, // Use same mock for reset operations
		tokenStore: tokenStore,
		opts:       opts,
	}

	tests := []struct {
		name   string
		params []string
	}{
		{"no params", []string{}},
		{"one param", []string{"token"}},
		{"too many params", []string{"token", "password", "extra"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := handler.resetPassword(tt.params)
			if !errors.Is(err, ErrInvalidArgumentCount) {
				t.Errorf("Expected ErrInvalidArgumentCount, got: %v", err)
			}
		})
	}
}

func TestResetPasswordEmptyPassword(t *testing.T) {
	tokenStore := resettoken.NewStore()
	opts := &options.Opts{}
	mockLDAP := &mockResetLDAPClient{}

	handler := &Handler{
		ldap:       mockLDAP,
		resetLDAP:  mockLDAP, // Use same mock for reset operations
		tokenStore: tokenStore,
		opts:       opts,
	}

	// Create a valid token
	token := &resettoken.ResetToken{
		Token:     "test-token",
		Username:  "testuser",
		Email:     "test@example.com",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
		Used:      false,
	}
	err := tokenStore.Store(token)
	require.NoError(t, err)

	params := []string{"test-token", ""}
	_, err = handler.resetPassword(params)

	if err == nil {
		t.Error("Expected error for empty password")
	}

	expectedErr := "the new password can't be empty"
	if err.Error() != expectedErr {
		t.Errorf("Error = %q, want %q", err.Error(), expectedErr)
	}
}

func TestResetPasswordUsernameInPassword(t *testing.T) {
	tokenStore := resettoken.NewStore()
	opts := &options.Opts{
		MinLength:                  8,
		MinNumbers:                 1,
		MinSymbols:                 1,
		MinUppercase:               1,
		MinLowercase:               1,
		PasswordCanIncludeUsername: false, // Username not allowed
	}
	mockLDAP := &mockResetLDAPClient{}

	handler := &Handler{
		ldap:       mockLDAP,
		resetLDAP:  mockLDAP, // Use same mock for reset operations
		tokenStore: tokenStore,
		opts:       opts,
	}

	// Create a valid token
	token := &resettoken.ResetToken{
		Token:     "test-token",
		Username:  "john",
		Email:     "john@example.com",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
		Used:      false,
	}
	err := tokenStore.Store(token)
	require.NoError(t, err)

	// Password contains username
	params := []string{"test-token", "Johnjohn123!"}
	_, err = handler.resetPassword(params)

	if err == nil {
		t.Error("Expected error for password containing username")
	}

	expectedErr := "the new password must not include the username"
	if err.Error() != expectedErr {
		t.Errorf("Error = %q, want %q", err.Error(), expectedErr)
	}
}

func TestResetPasswordLDAPFailure(t *testing.T) {
	tokenStore := resettoken.NewStore()
	opts := &options.Opts{
		MinLength:    8,
		MinNumbers:   1,
		MinSymbols:   1,
		MinUppercase: 1,
		MinLowercase: 1,
	}
	mockLDAP := &mockResetLDAPClient{
		resetPasswordError: errors.New("LDAP reset failed"),
	}

	handler := &Handler{
		ldap:       mockLDAP,
		resetLDAP:  mockLDAP,
		tokenStore: tokenStore,
		opts:       opts,
	}

	// Create a valid token
	token := &resettoken.ResetToken{
		Token:     "ldap-fail-token",
		Username:  "testuser",
		Email:     "test@example.com",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
		Used:      false,
	}
	err := tokenStore.Store(token)
	require.NoError(t, err)

	params := []string{"ldap-fail-token", "NewPass123!"}
	_, err = handler.resetPassword(params)

	if err == nil {
		t.Error("Expected error for LDAP failure")
	}

	// Error message should be generic (not exposing LDAP details)
	expectedErr := "failed to reset password; please contact your administrator if this problem persists"
	if err.Error() != expectedErr {
		t.Errorf("Error = %q, want %q", err.Error(), expectedErr)
	}
}

func TestResetPasswordLazyInitSuccess(t *testing.T) {
	tokenStore := resettoken.NewStore()
	opts := &options.Opts{
		MinLength:     8,
		MinNumbers:    1,
		MinSymbols:    1,
		MinUppercase:  1,
		MinLowercase:  1,
		ResetUser:     "cn=reset,dc=example,dc=com",
		ResetPassword: "resetpass",
		LDAP: ldap.Config{
			Server: "ldap://localhost:389",
			BaseDN: "dc=example,dc=com",
		},
	}
	mockLDAP := &mockResetLDAPClient{}

	handler := &Handler{
		ldap:       mockLDAP,
		resetLDAP:  nil, // Not initialized - should trigger lazy init
		tokenStore: tokenStore,
		opts:       opts,
	}

	// Create a valid token
	token := &resettoken.ResetToken{
		Token:     "lazy-init-token",
		Username:  "testuser",
		Email:     "test@example.com",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
		Used:      false,
	}
	err := tokenStore.Store(token)
	require.NoError(t, err)

	// This will fail at LDAP init because we can't connect to actual LDAP
	// but it tests the lazy init path
	params := []string{"lazy-init-token", "NewPass123!"}
	_, err = handler.resetPassword(params)

	// Should fail with LDAP init error
	if err == nil {
		t.Error("Expected error for LDAP init failure")
	}
	if !strings.Contains(err.Error(), "initialize password reset connection") &&
		!strings.Contains(err.Error(), "failed to reset password") {
		t.Errorf("Unexpected error: %q", err.Error())
	}
}

func TestResetPasswordLazyInitNotConfigured(t *testing.T) {
	tokenStore := resettoken.NewStore()
	opts := &options.Opts{
		MinLength:     8,
		MinNumbers:    1,
		MinSymbols:    1,
		MinUppercase:  1,
		MinLowercase:  1,
		ResetUser:     "", // Not configured
		ResetPassword: "", // Not configured
	}
	mockLDAP := &mockResetLDAPClient{}

	handler := &Handler{
		ldap:       mockLDAP,
		resetLDAP:  nil, // Not initialized and not configured
		tokenStore: tokenStore,
		opts:       opts,
	}

	// Create a valid token
	token := &resettoken.ResetToken{
		Token:     "not-configured-token",
		Username:  "testuser",
		Email:     "test@example.com",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
		Used:      false,
	}
	err := tokenStore.Store(token)
	require.NoError(t, err)

	params := []string{"not-configured-token", "NewPass123!"}
	_, err = handler.resetPassword(params)

	if err == nil {
		t.Error("Expected error when reset not configured")
	}
	expectedErr := "password reset not properly configured; please contact your administrator"
	if err.Error() != expectedErr {
		t.Errorf("Error = %q, want %q", err.Error(), expectedErr)
	}
}

func TestResetPasswordShortToken(t *testing.T) {
	tokenStore := resettoken.NewStore()
	opts := &options.Opts{MinLength: 8}
	mockLDAP := &mockResetLDAPClient{}

	handler := &Handler{
		ldap:       mockLDAP,
		resetLDAP:  mockLDAP,
		tokenStore: tokenStore,
		opts:       opts,
	}

	// Try with very short token that doesn't exist
	params := []string{"abc", "NewPass123!"}
	_, err := handler.resetPassword(params)

	if err == nil {
		t.Error("Expected error for invalid token")
	}
	if err.Error() != errInvalidOrExpiredToken {
		t.Errorf("Error = %q, want %q", err.Error(), errInvalidOrExpiredToken)
	}
}

func TestResetPasswordEmptyUsername(t *testing.T) {
	tokenStore := resettoken.NewStore()
	opts := &options.Opts{
		MinLength:    8,
		MinNumbers:   1,
		MinSymbols:   1,
		MinUppercase: 1,
		MinLowercase: 1,
	}
	mockLDAP := &mockResetLDAPClient{}

	handler := &Handler{
		ldap:       mockLDAP,
		resetLDAP:  mockLDAP,
		tokenStore: tokenStore,
		opts:       opts,
	}

	// Create a token with empty username
	token := &resettoken.ResetToken{
		Token:     "empty-username-token",
		Username:  "", // Empty username
		Email:     "test@example.com",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
		Used:      false,
	}
	err := tokenStore.Store(token)
	require.NoError(t, err)

	params := []string{"empty-username-token", "NewPass123!"}
	_, err = handler.resetPassword(params)

	if err == nil {
		t.Error("Expected error for empty username in token")
	}
	if err.Error() != errInvalidOrExpiredToken {
		t.Errorf("Error = %q, want %q", err.Error(), errInvalidOrExpiredToken)
	}
}

func TestResetPasswordMarkUsedError(t *testing.T) {
	// Use a mock token store that fails on MarkUsed
	mockTokenStore := &mockTokenStoreMarkUsedFails{
		tokens: make(map[string]*resettoken.ResetToken),
	}
	opts := &options.Opts{
		MinLength:    8,
		MinNumbers:   1,
		MinSymbols:   1,
		MinUppercase: 1,
		MinLowercase: 1,
	}
	mockLDAP := &mockResetLDAPClient{}

	handler := &Handler{
		ldap:       mockLDAP,
		resetLDAP:  mockLDAP,
		tokenStore: mockTokenStore,
		opts:       opts,
	}

	// Create a valid token
	token := &resettoken.ResetToken{
		Token:     "mark-used-fail-token",
		Username:  "testuser",
		Email:     "test@example.com",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
		Used:      false,
	}
	mockTokenStore.tokens[token.Token] = token

	params := []string{"mark-used-fail-token", "NewPass123!"}
	result, err := handler.resetPassword(params)
	// Should still succeed (password was changed, just logging the mark-used failure)
	if err != nil {
		t.Errorf("Expected success despite mark-used failure, got: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("Expected 1 result, got %d", len(result))
	}
}

// mockTokenStoreMarkUsedFails is a mock token store that fails on MarkUsed.
type mockTokenStoreMarkUsedFails struct {
	tokens map[string]*resettoken.ResetToken
}

func (m *mockTokenStoreMarkUsedFails) Store(token *resettoken.ResetToken) error {
	m.tokens[token.Token] = token
	return nil
}

func (m *mockTokenStoreMarkUsedFails) Get(tokenString string) (*resettoken.ResetToken, error) {
	if token, ok := m.tokens[tokenString]; ok {
		return token, nil
	}
	return nil, errors.New("token not found")
}

func (m *mockTokenStoreMarkUsedFails) MarkUsed(_ string) error {
	return errors.New("mark used failed")
}

func (m *mockTokenStoreMarkUsedFails) Delete(_ string) error {
	return nil
}

func (m *mockTokenStoreMarkUsedFails) CleanupExpired() int {
	return 0
}

func (m *mockTokenStoreMarkUsedFails) Count() int {
	return len(m.tokens)
}
