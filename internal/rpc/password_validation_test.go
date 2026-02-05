package rpc

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/options"
)

// TestPluralize tests the pluralize function for singular/plural forms.
func TestPluralize(t *testing.T) {
	tests := []struct {
		name   string
		word   string
		amount uint
		want   string
	}{
		{
			name:   "singular (1)",
			word:   "number",
			amount: 1,
			want:   "number",
		},
		{
			name:   "plural (0)",
			word:   "number",
			amount: 0,
			want:   "numbers",
		},
		{
			name:   "plural (2)",
			word:   "number",
			amount: 2,
			want:   "numbers",
		},
		{
			name:   "plural (10)",
			word:   "symbol",
			amount: 10,
			want:   "symbols",
		},
		{
			name:   "plural (100)",
			word:   "letter",
			amount: 100,
			want:   "letters",
		},
		{
			name:   "singular with uppercase",
			word:   "Letter",
			amount: 1,
			want:   "Letter",
		},
		{
			name:   "plural with uppercase",
			word:   "Letter",
			amount: 2,
			want:   "Letters",
		},
		{
			name:   "empty word singular",
			word:   "",
			amount: 1,
			want:   "",
		},
		{
			name:   "empty word plural",
			word:   "",
			amount: 0,
			want:   "s",
		},
		{
			name:   "max uint",
			word:   "test",
			amount: ^uint(0),
			want:   "tests",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pluralize(tt.word, tt.amount)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestValidateNewPasswordMinLength tests minimum length validation.
func TestValidateNewPasswordMinLength(t *testing.T) {
	tests := []struct {
		name      string
		password  string
		minLength uint
		wantError bool
		errMsg    string
	}{
		{
			name:      "exact minimum length",
			password:  "Aa1!Aa1!",
			minLength: 8,
			wantError: false,
		},
		{
			name:      "below minimum length",
			password:  "Aa1!",
			minLength: 8,
			wantError: true,
			errMsg:    "at least 8 characters",
		},
		{
			name:      "zero minimum length",
			password:  "",
			minLength: 0,
			wantError: false,
		},
		{
			name:      "very long password",
			password:  "Aa1!" + string(make([]byte, 200)),
			minLength: 200,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &options.Opts{
				MinLength:    tt.minLength,
				MinNumbers:   0,
				MinSymbols:   0,
				MinUppercase: 0,
				MinLowercase: 0,
			}
			err := ValidateNewPassword(tt.password, "", opts)
			if tt.wantError {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateNewPasswordMaxLength tests maximum length validation.
func TestValidateNewPasswordMaxLength(t *testing.T) {
	longPassword := string(make([]byte, MaxPasswordLength+1))
	opts := &options.Opts{
		MinLength:    0,
		MinNumbers:   0,
		MinSymbols:   0,
		MinUppercase: 0,
		MinLowercase: 0,
	}

	err := ValidateNewPassword(longPassword, "", opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must not exceed 128 characters")

	// Exactly at max length should succeed
	exactMax := string(make([]byte, MaxPasswordLength))
	err = ValidateNewPassword(exactMax, "", opts)
	assert.NoError(t, err)
}

// TestValidateNewPasswordNumberRequirements tests minimum numbers validation.
func TestValidateNewPasswordNumberRequirements(t *testing.T) {
	tests := []struct {
		name       string
		password   string
		minNumbers uint
		wantError  bool
	}{
		{
			name:       "no numbers required, no numbers present",
			password:   "Password!",
			minNumbers: 0,
			wantError:  false,
		},
		{
			name:       "one number required, one present",
			password:   "Password1!",
			minNumbers: 1,
			wantError:  false,
		},
		{
			name:       "two numbers required, one present",
			password:   "Password1!",
			minNumbers: 2,
			wantError:  true,
		},
		{
			name:       "three numbers required, three present",
			password:   "Pass123!",
			minNumbers: 3,
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &options.Opts{
				MinLength:    0,
				MinNumbers:   tt.minNumbers,
				MinSymbols:   0,
				MinUppercase: 0,
				MinLowercase: 0,
			}
			err := ValidateNewPassword(tt.password, "", opts)
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "number")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateNewPasswordSymbolRequirements tests minimum symbols validation.
func TestValidateNewPasswordSymbolRequirements(t *testing.T) {
	tests := []struct {
		name       string
		password   string
		minSymbols uint
		wantError  bool
	}{
		{
			name:       "no symbols required, no symbols present",
			password:   "Password1",
			minSymbols: 0,
			wantError:  false,
		},
		{
			name:       "one symbol required, one present",
			password:   "Password1!",
			minSymbols: 1,
			wantError:  false,
		},
		{
			name:       "two symbols required, one present",
			password:   "Password1!",
			minSymbols: 2,
			wantError:  true,
		},
		{
			name:       "three symbols required, three present",
			password:   "Pass!@#1",
			minSymbols: 3,
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &options.Opts{
				MinLength:    0,
				MinNumbers:   0,
				MinSymbols:   tt.minSymbols,
				MinUppercase: 0,
				MinLowercase: 0,
			}
			err := ValidateNewPassword(tt.password, "", opts)
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "symbol")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateNewPasswordAllRequirementsCombined tests all requirements together.
func TestValidateNewPasswordAllRequirementsCombined(t *testing.T) {
	opts := &options.Opts{
		MinLength:                  8,
		MinNumbers:                 2,
		MinSymbols:                 2,
		MinUppercase:               2,
		MinLowercase:               2,
		PasswordCanIncludeUsername: false,
	}

	tests := []struct {
		name      string
		password  string
		username  string
		wantError bool
	}{
		{
			name:      "all requirements met",
			password:  "AAaa12!!",
			username:  "user",
			wantError: false,
		},
		{
			name:      "missing one uppercase",
			password:  "Aaaa12!!",
			username:  "user",
			wantError: true,
		},
		{
			name:      "missing one lowercase",
			password:  "AAaa12!!",
			username:  "user",
			wantError: false,
		},
		{
			name:      "contains username",
			password:  "AAuser12!!",
			username:  "user",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNewPassword(tt.password, tt.username, opts)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
