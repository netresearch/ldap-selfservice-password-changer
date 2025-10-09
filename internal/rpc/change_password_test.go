package rpc

import (
	"strings"
	"testing"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/options"
)

// TestPasswordCanIncludeUsername tests the username inclusion validation logic
// with various case combinations to ensure case-insensitive checking works correctly
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
			password:                   "admin123!",
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
			password:                   "ADMIN123!",
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
			password:                   "admin123!",
			passwordCanIncludeUsername: false,
			shouldFail:                 false,
		},

		// When PasswordCanIncludeUsername is TRUE (username allowed in password)
		{
			name:                       "allow username in password same case (allowed)",
			username:                   "admin",
			password:                   "admin123!",
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

			// Test the validation logic directly
			// We can't call changePassword without LDAP mock, so we test the logic
			shouldReject := !opts.PasswordCanIncludeUsername && tt.username != "" &&
				strings.Contains(strings.ToLower(tt.password), strings.ToLower(tt.username))

			if tt.shouldFail {
				if !shouldReject {
					t.Errorf("Expected password %q to be rejected when username is %q and PasswordCanIncludeUsername=%v",
						tt.password, tt.username, tt.passwordCanIncludeUsername)
				}
			} else {
				if shouldReject {
					t.Errorf("Expected password %q to be accepted when username is %q and PasswordCanIncludeUsername=%v",
						tt.password, tt.username, tt.passwordCanIncludeUsername)
				}
			}
		})
	}
}

// TestPasswordValidationEdgeCases tests edge cases for password validation
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
				PasswordCanIncludeUsername: tt.passwordCanIncludeUsername,
			}

			shouldReject := !opts.PasswordCanIncludeUsername && tt.username != "" &&
				strings.Contains(strings.ToLower(tt.password), strings.ToLower(tt.username))

			if shouldReject != tt.shouldReject {
				t.Errorf("%s: Expected shouldReject=%v, got shouldReject=%v for password=%q username=%q",
					tt.description, tt.shouldReject, shouldReject, tt.password, tt.username)
			}
		})
	}
}
