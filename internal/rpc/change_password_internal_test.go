package rpc

import (
	"errors"
	"strings"
	"testing"

	ldap "github.com/netresearch/simple-ldap-go"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/options"
)

const testClientIP = "203.0.113.42"

// TestPasswordCanIncludeUsername tests the username inclusion validation logic.
// with various case combinations to ensure case-insensitive checking works correctly.
func TestPasswordCanIncludeUsername(t *testing.T) {
	tests := []struct {
		name                       string
		username                   string
		password                   string
		passwordCanIncludeUsername bool
		shouldFail                 bool
		expectedError              string
	}{
		// When PasswordCanIncludeUsername is FALSE (username not allowed in password)
		{
			name:                       "reject exact match same case (disallowed)",
			username:                   "admin",
			password:                   "Admin123!",
			passwordCanIncludeUsername: false,
			shouldFail:                 true,
			expectedError:              "the new password must not include the username",
		},
		{
			name:                       "reject exact match different case (disallowed)",
			username:                   "admin",
			password:                   "Admin123!",
			passwordCanIncludeUsername: false,
			shouldFail:                 true,
			expectedError:              "the new password must not include the username",
		},
		{
			name:                       "reject uppercase username in password (disallowed)",
			username:                   "admin",
			password:                   "ADMIN123pass!",
			passwordCanIncludeUsername: false,
			shouldFail:                 true,
			expectedError:              "the new password must not include the username",
		},
		{
			name:                       "reject mixed case username in password (disallowed)",
			username:                   "johnsmith",
			password:                   "JohnSmith123!",
			passwordCanIncludeUsername: false,
			shouldFail:                 true,
			expectedError:              "the new password must not include the username",
		},
		{
			name:                       "reject username in middle of password (disallowed)",
			username:                   "admin",
			password:                   "Super_Admin_123!",
			passwordCanIncludeUsername: false,
			shouldFail:                 true,
			expectedError:              "the new password must not include the username",
		},
		{
			name:                       "accept password without username (disallowed)",
			username:                   "admin",
			password:                   "SecurePass123!",
			passwordCanIncludeUsername: false,
			shouldFail:                 false,
		},
		{
			name:                       "accept partial match not containing full username (disallowed)",
			username:                   "administrator",
			password:                   "Admin123!",
			passwordCanIncludeUsername: false,
			shouldFail:                 false,
		},

		// When PasswordCanIncludeUsername is TRUE (username allowed in password)
		{
			name:                       "allow username in password same case (allowed)",
			username:                   "admin",
			password:                   "Admin123!",
			passwordCanIncludeUsername: true,
			shouldFail:                 false,
		},
		{
			name:                       "allow username in password different case (allowed)",
			username:                   "admin",
			password:                   "Admin123!",
			passwordCanIncludeUsername: true,
			shouldFail:                 false,
		},
		{
			name:                       "allow password without username (allowed)",
			username:                   "admin",
			password:                   "SecurePass123!",
			passwordCanIncludeUsername: true,
			shouldFail:                 false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create minimal options for testing
			opts := &options.Opts{
				MinLength:                  8,
				MinNumbers:                 1,
				MinSymbols:                 1,
				MinUppercase:               1,
				MinLowercase:               1,
				PasswordCanIncludeUsername: tt.passwordCanIncludeUsername,
			}

			// Test the validation logic using ValidateNewPassword
			err := ValidateNewPassword(tt.password, tt.username, opts)

			if tt.shouldFail {
				if err == nil {
					t.Errorf("Expected password %q to be rejected when username is %q and PasswordCanIncludeUsername=%v",
						tt.password, tt.username, tt.passwordCanIncludeUsername)
				}
				if err != nil && !strings.Contains(err.Error(), "username") {
					t.Errorf("Expected error to mention username, got: %v", err)
				}
			} else if err != nil {
				t.Errorf(
					"Expected password %q to be accepted when username is %q and PasswordCanIncludeUsername=%v, got error: %v",
					tt.password, tt.username, tt.passwordCanIncludeUsername, err)
			}
		})
	}
}

// TestPasswordValidationEdgeCases tests edge cases for password validation.
func TestPasswordValidationEdgeCases(t *testing.T) {
	tests := []struct {
		name                       string
		username                   string
		password                   string
		passwordCanIncludeUsername bool
		shouldReject               bool
		description                string
	}{
		{
			name:                       "empty username",
			username:                   "",
			password:                   "SecurePass123!",
			passwordCanIncludeUsername: false,
			shouldReject:               false,
			description:                "Empty username should not cause rejection",
		},
		{
			name:                       "single character username",
			username:                   "a",
			password:                   "SecurePass123!",
			passwordCanIncludeUsername: false,
			shouldReject:               true,
			description:                "Password contains single character username",
		},
		{
			name:                       "unicode characters in username",
			username:                   "müller",
			password:                   "Müller123!",
			passwordCanIncludeUsername: false,
			shouldReject:               true,
			description:                "Unicode characters should be handled case-insensitively",
		},
		{
			name:                       "username with numbers",
			username:                   "user123",
			password:                   "User123Pass!",
			passwordCanIncludeUsername: false,
			shouldReject:               true,
			description:                "Username with numbers should be detected",
		},
		{
			name:                       "very long username",
			username:                   "verylongusername123456789",
			password:                   "VeryLongUsername123456789!",
			passwordCanIncludeUsername: false,
			shouldReject:               true,
			description:                "Long usernames should be handled correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &options.Opts{
				MinLength:                  8,
				MinNumbers:                 1,
				MinSymbols:                 1,
				MinUppercase:               1,
				MinLowercase:               1,
				PasswordCanIncludeUsername: tt.passwordCanIncludeUsername,
			}

			err := ValidateNewPassword(tt.password, tt.username, opts)
			shouldReject := err != nil && strings.Contains(err.Error(), "username")

			if shouldReject != tt.shouldReject {
				t.Errorf("%s: Expected shouldReject=%v, got shouldReject=%v (error: %v) for password=%q username=%q",
					tt.description, tt.shouldReject, shouldReject, err, tt.password, tt.username)
			}
		})
	}
}

// TestChangePasswordIPRateLimiting tests IP-based rate limiting on change-password endpoint.
func TestChangePasswordIPRateLimiting(t *testing.T) {
	mockLDAP := &mockChangePasswordLDAP{
		changePasswordError: nil,
	}

	opts := &options.Opts{
		MinLength:                  8,
		MinNumbers:                 1,
		MinSymbols:                 1,
		MinUppercase:               1,
		MinLowercase:               1,
		PasswordCanIncludeUsername: false,
	}

	// Create IP limiter with very low limit for testing
	ipLimiter := &mockIPLimiter{
		allowed: true,
		count:   0,
	}

	handler := &Handler{
		ldap:      mockLDAP,
		opts:      opts,
		ipLimiter: ipLimiter,
	}

	clientIP := testClientIP

	// First 5 requests should succeed
	for i := 1; i <= 5; i++ {
		result, err := handler.changePasswordWithIP(
			[]string{"testuser", "OldPass123!", "NewPass456!"},
			clientIP,
		)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}
		if len(result) != 1 || result[0] != "password changed successfully" {
			t.Errorf("Request %d: unexpected result: %v", i, result)
		}
		ipLimiter.count++
	}

	// 6th request should be rate limited
	ipLimiter.allowed = false
	result, err := handler.changePasswordWithIP(
		[]string{"testuser", "OldPass123!", "NewPass456!"},
		clientIP,
	)

	if err == nil {
		t.Error("Expected rate limit error, got nil")
	} else if !strings.Contains(err.Error(), "too many") {
		t.Errorf("Expected rate limit error with 'too many', got: %v", err)
	}
	if result != nil {
		t.Errorf("Expected nil result when rate limited, got: %v", result)
	}
}

// mockChangePasswordLDAP mocks LDAP client for change password tests.
type mockChangePasswordLDAP struct {
	changePasswordError error
}

func (m *mockChangePasswordLDAP) FindUserByMail(_ string) (*ldap.User, error) {
	return &ldap.User{SAMAccountName: "testuser"}, nil
}

func (m *mockChangePasswordLDAP) ChangePasswordForSAMAccountName(_, _, _ string) error {
	return m.changePasswordError
}

func (m *mockChangePasswordLDAP) ResetPasswordForSAMAccountName(_, _ string) error {
	return nil
}

// mockIPLimiter mocks IP rate limiter for testing.
type mockIPLimiter struct {
	allowed bool
	count   int
}

func (m *mockIPLimiter) AllowRequest(_ string) bool {
	return m.allowed
}

// TestChangePasswordEmptyInputs tests change-password with empty inputs.
func TestChangePasswordEmptyInputs(t *testing.T) {
	mockLDAP := &mockChangePasswordLDAP{}
	opts := &options.Opts{
		MinLength:    8,
		MinNumbers:   1,
		MinSymbols:   1,
		MinUppercase: 1,
		MinLowercase: 1,
	}
	ipLimiter := &mockIPLimiter{allowed: true}

	handler := &Handler{
		ldap:      mockLDAP,
		opts:      opts,
		ipLimiter: ipLimiter,
	}

	tests := []struct {
		name      string
		params    []string
		wantError string
	}{
		{
			name:      "empty username",
			params:    []string{"", "OldPass123!", "NewPass456!"},
			wantError: "username can't be empty",
		},
		{
			name:      "empty current password",
			params:    []string{"testuser", "", "NewPass456!"},
			wantError: "old password can't be empty",
		},
		{
			name:      "empty new password",
			params:    []string{"testuser", "OldPass123!", ""},
			wantError: "new password can't be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handler.changePasswordWithIP(tt.params, testClientIP)
			if err == nil {
				t.Errorf("Expected error containing %q, got nil", tt.wantError)
				return
			}
			if result != nil {
				t.Errorf("Expected nil result, got %v", result)
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("Expected error containing %q, got %q", tt.wantError, err.Error())
			}
		})
	}
}

// TestChangePasswordSamePassword tests when old and new passwords are the same.
func TestChangePasswordSamePassword(t *testing.T) {
	mockLDAP := &mockChangePasswordLDAP{}
	opts := &options.Opts{
		MinLength:    8,
		MinNumbers:   1,
		MinSymbols:   1,
		MinUppercase: 1,
		MinLowercase: 1,
	}
	ipLimiter := &mockIPLimiter{allowed: true}

	handler := &Handler{
		ldap:      mockLDAP,
		opts:      opts,
		ipLimiter: ipLimiter,
	}

	result, err := handler.changePasswordWithIP(
		[]string{"testuser", "SamePass123!", "SamePass123!"},
		testClientIP,
	)

	if err == nil {
		t.Error("Expected error when old and new passwords are the same")
		return
	}
	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
	if !strings.Contains(err.Error(), "can't be same") {
		t.Errorf("Expected error about same password, got %q", err.Error())
	}
}

// TestChangePasswordLDAPError tests when LDAP returns an error.
func TestChangePasswordLDAPError(t *testing.T) {
	mockLDAP := &mockChangePasswordLDAP{
		changePasswordError: errors.New("LDAP connection failed"),
	}
	opts := &options.Opts{
		MinLength:    8,
		MinNumbers:   1,
		MinSymbols:   1,
		MinUppercase: 1,
		MinLowercase: 1,
	}
	ipLimiter := &mockIPLimiter{allowed: true}

	handler := &Handler{
		ldap:      mockLDAP,
		opts:      opts,
		ipLimiter: ipLimiter,
	}

	result, err := handler.changePasswordWithIP(
		[]string{"testuser", "OldPass123!", "NewPass456!"},
		testClientIP,
	)

	if err == nil {
		t.Error("Expected error on LDAP failure")
		return
	}
	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
	if !strings.Contains(err.Error(), "failed to change password") {
		t.Errorf("Expected error about failed change, got %q", err.Error())
	}
}

// TestChangePasswordInvalidArgumentCount tests invalid parameter counts.
func TestChangePasswordInvalidArgumentCount(t *testing.T) {
	mockLDAP := &mockChangePasswordLDAP{}
	opts := &options.Opts{}
	ipLimiter := &mockIPLimiter{allowed: true}

	handler := &Handler{
		ldap:      mockLDAP,
		opts:      opts,
		ipLimiter: ipLimiter,
	}

	tests := []struct {
		name   string
		params []string
	}{
		{"no params", []string{}},
		{"one param", []string{"user"}},
		{"two params", []string{"user", "pass"}},
		{"four params", []string{"user", "old", "new", "extra"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := handler.changePasswordWithIP(tt.params, testClientIP)
			if !errors.Is(err, ErrInvalidArgumentCount) {
				t.Errorf("Expected ErrInvalidArgumentCount, got: %v", err)
			}
		})
	}
}

// TestChangePasswordNoIPLimiter tests when IP limiter is nil.
func TestChangePasswordNoIPLimiter(t *testing.T) {
	mockLDAP := &mockChangePasswordLDAP{}
	opts := &options.Opts{
		MinLength:    8,
		MinNumbers:   1,
		MinSymbols:   1,
		MinUppercase: 1,
		MinLowercase: 1,
	}

	handler := &Handler{
		ldap:      mockLDAP,
		opts:      opts,
		ipLimiter: nil, // No IP limiter
	}

	result, err := handler.changePasswordWithIP(
		[]string{"testuser", "OldPass123!", "NewPass456!"},
		testClientIP,
	)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if len(result) != 1 || result[0] != "password changed successfully" {
		t.Errorf("Expected success message, got %v", result)
	}
}

// TestChangePasswordValidationFailure tests password policy violations.
func TestChangePasswordValidationFailure(t *testing.T) {
	mockLDAP := &mockChangePasswordLDAP{}
	opts := &options.Opts{
		MinLength:                  12,
		MinNumbers:                 2,
		MinSymbols:                 2,
		MinUppercase:               2,
		MinLowercase:               2,
		PasswordCanIncludeUsername: false,
	}
	ipLimiter := &mockIPLimiter{allowed: true}

	handler := &Handler{
		ldap:      mockLDAP,
		opts:      opts,
		ipLimiter: ipLimiter,
	}

	tests := []struct {
		name        string
		newPassword string
		username    string
		wantError   string
	}{
		{
			name:        "too short",
			newPassword: "Short1!",
			username:    "user",
			wantError:   "at least 12 characters",
		},
		{
			name:        "not enough numbers",
			newPassword: "Password1!!!AA",
			username:    "user",
			wantError:   "2 numbers",
		},
		{
			name:        "contains username",
			newPassword: "Useruser12!!AA",
			username:    "user",
			wantError:   "username",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handler.changePasswordWithIP(
				[]string{tt.username, "OldPass123!", tt.newPassword},
				testClientIP,
			)
			if err == nil {
				t.Errorf("Expected error containing %q, got nil", tt.wantError)
				return
			}
			if result != nil {
				t.Errorf("Expected nil result, got %v", result)
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("Expected error containing %q, got %q", tt.wantError, err.Error())
			}
		})
	}
}
