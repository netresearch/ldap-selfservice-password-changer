//nolint:testpackage // tests internal functions
package options

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnvStringOrDefault tests the envStringOrDefault function.
func TestEnvStringOrDefault(t *testing.T) {
	tests := []struct {
		name       string
		envName    string
		envValue   string
		setEnv     bool
		defaultVal string
		want       string
	}{
		{
			name:       "returns default when env not set",
			envName:    "TEST_ENV_NOT_SET_12345",
			setEnv:     false,
			defaultVal: "default_value",
			want:       "default_value",
		},
		{
			name:       "returns env value when set",
			envName:    "TEST_ENV_SET_STRING",
			envValue:   "env_value",
			setEnv:     true,
			defaultVal: "default_value",
			want:       "env_value",
		},
		{
			name:       "returns default when env is empty string",
			envName:    "TEST_ENV_EMPTY_STRING",
			envValue:   "",
			setEnv:     true,
			defaultVal: "default_value",
			want:       "default_value",
		},
		{
			name:       "returns env value with special characters",
			envName:    "TEST_ENV_SPECIAL_CHARS",
			envValue:   "ldaps://server.example.com:636",
			setEnv:     true,
			defaultVal: "ldap://localhost",
			want:       "ldaps://server.example.com:636",
		},
		{
			name:       "handles whitespace in env value",
			envName:    "TEST_ENV_WHITESPACE",
			envValue:   "  value with spaces  ",
			setEnv:     true,
			defaultVal: "default",
			want:       "  value with spaces  ", // Should preserve whitespace
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				t.Setenv(tt.envName, tt.envValue)
			}

			got := envStringOrDefault(tt.envName, tt.defaultVal)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestEnvIntOrDefault tests the envIntOrDefault function.
func TestEnvIntOrDefault(t *testing.T) {
	tests := []struct {
		name       string
		envName    string
		envValue   string
		setEnv     bool
		defaultVal uint64
		want       uint
		wantErr    bool
	}{
		{
			name:       "returns default when env not set",
			envName:    "TEST_INT_NOT_SET_12345",
			setEnv:     false,
			defaultVal: 42,
			want:       42,
		},
		{
			name:       "returns parsed env value when valid",
			envName:    "TEST_INT_VALID",
			envValue:   "100",
			setEnv:     true,
			defaultVal: 42,
			want:       100,
		},
		{
			name:       "returns zero when env is zero",
			envName:    "TEST_INT_ZERO",
			envValue:   "0",
			setEnv:     true,
			defaultVal: 42,
			want:       0,
		},
		{
			name:       "handles large valid uint16 value",
			envName:    "TEST_INT_LARGE",
			envValue:   "65535",
			setEnv:     true,
			defaultVal: 0,
			want:       65535,
		},
		{
			name:       "returns default and adds error for invalid value",
			envName:    "TEST_INT_INVALID",
			envValue:   "not_a_number",
			setEnv:     true,
			defaultVal: 42,
			want:       42,
			wantErr:    true,
		},
		{
			name:       "returns default and adds error for negative value",
			envName:    "TEST_INT_NEGATIVE",
			envValue:   "-5",
			setEnv:     true,
			defaultVal: 10,
			want:       10,
			wantErr:    true,
		},
		{
			name:       "returns default and adds error for overflow",
			envName:    "TEST_INT_OVERFLOW",
			envValue:   "999999",
			setEnv:     true,
			defaultVal: 100,
			want:       100,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				t.Setenv(tt.envName, tt.envValue)
			}

			errs := &ConfigError{}
			got := envIntOrDefault(tt.envName, tt.defaultVal, errs)
			assert.Equal(t, tt.want, got)
			if tt.wantErr {
				assert.True(t, errs.HasErrors(), "expected error to be recorded")
			} else {
				assert.False(t, errs.HasErrors(), "expected no errors")
			}
		})
	}
}

// TestEnvBoolOrDefault tests the envBoolOrDefault function.
func TestEnvBoolOrDefault(t *testing.T) {
	tests := []struct {
		name       string
		envName    string
		envValue   string
		setEnv     bool
		defaultVal bool
		want       bool
		wantErr    bool
	}{
		{
			name:       "returns default true when env not set",
			envName:    "TEST_BOOL_NOT_SET_TRUE",
			setEnv:     false,
			defaultVal: true,
			want:       true,
		},
		{
			name:       "returns default false when env not set",
			envName:    "TEST_BOOL_NOT_SET_FALSE",
			setEnv:     false,
			defaultVal: false,
			want:       false,
		},
		{
			name:       "returns true when env is true",
			envName:    "TEST_BOOL_TRUE",
			envValue:   "true",
			setEnv:     true,
			defaultVal: false,
			want:       true,
		},
		{
			name:       "returns false when env is false",
			envName:    "TEST_BOOL_FALSE",
			envValue:   "false",
			setEnv:     true,
			defaultVal: true,
			want:       false,
		},
		{
			name:       "handles 1 as true",
			envName:    "TEST_BOOL_ONE",
			envValue:   "1",
			setEnv:     true,
			defaultVal: false,
			want:       true,
		},
		{
			name:       "handles 0 as false",
			envName:    "TEST_BOOL_ZERO",
			envValue:   "0",
			setEnv:     true,
			defaultVal: true,
			want:       false,
		},
		{
			name:       "handles TRUE (uppercase)",
			envName:    "TEST_BOOL_UPPER_TRUE",
			envValue:   "TRUE",
			setEnv:     true,
			defaultVal: false,
			want:       true,
		},
		{
			name:       "handles FALSE (uppercase)",
			envName:    "TEST_BOOL_UPPER_FALSE",
			envValue:   "FALSE",
			setEnv:     true,
			defaultVal: true,
			want:       false,
		},
		{
			name:       "returns default and adds error for invalid value",
			envName:    "TEST_BOOL_INVALID",
			envValue:   "not_a_bool",
			setEnv:     true,
			defaultVal: true,
			want:       true,
			wantErr:    true,
		},
		{
			name:       "returns default and adds error for empty-ish value",
			envName:    "TEST_BOOL_MAYBE",
			envValue:   "maybe",
			setEnv:     true,
			defaultVal: false,
			want:       false,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				t.Setenv(tt.envName, tt.envValue)
			}

			errs := &ConfigError{}
			got := envBoolOrDefault(tt.envName, tt.defaultVal, errs)
			assert.Equal(t, tt.want, got)
			if tt.wantErr {
				assert.True(t, errs.HasErrors(), "expected error to be recorded")
			} else {
				assert.False(t, errs.HasErrors(), "expected no errors")
			}
		})
	}
}

// TestCheckRequired tests the checkRequired function.
func TestCheckRequired(t *testing.T) {
	tests := []struct {
		name           string
		fieldName      string
		value          string
		wantMissing    []string
		initialMissing []string
	}{
		{
			name:           "adds to missing when value is empty",
			fieldName:      "ldap-server",
			value:          "",
			initialMissing: []string{},
			wantMissing:    []string{"ldap-server"},
		},
		{
			name:           "does not add when value is non-empty",
			fieldName:      "ldap-server",
			value:          "ldaps://example.com",
			initialMissing: []string{},
			wantMissing:    []string{},
		},
		{
			name:           "appends to existing missing list",
			fieldName:      "base-dn",
			value:          "",
			initialMissing: []string{"ldap-server"},
			wantMissing:    []string{"ldap-server", "base-dn"},
		},
		{
			name:           "handles whitespace-only as non-empty",
			fieldName:      "readonly-user",
			value:          "   ",
			initialMissing: []string{},
			wantMissing:    []string{}, // Whitespace-only is NOT empty
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			missing := make([]string, len(tt.initialMissing))
			copy(missing, tt.initialMissing)
			value := tt.value

			checkRequired(tt.fieldName, &value, &missing)

			assert.Equal(t, tt.wantMissing, missing)
		})
	}
}

// TestOptsStruct tests the Opts struct fields and defaults.
func TestOptsStruct(t *testing.T) {
	opts := &Opts{
		Port:                        "3000",
		ReadonlyUser:                "cn=readonly,dc=example,dc=com",
		ReadonlyPassword:            "secret",
		MinLength:                   8,
		MinNumbers:                  1,
		MinSymbols:                  1,
		MinUppercase:                1,
		MinLowercase:                1,
		PasswordCanIncludeUsername:  false,
		PasswordResetEnabled:        true,
		ResetTokenExpiryMinutes:     15,
		ResetRateLimitRequests:      3,
		ResetRateLimitWindowMinutes: 60,
		SMTPHost:                    "smtp.example.com",
		SMTPPort:                    587,
		SMTPUsername:                "smtpuser",
		SMTPPassword:                "smtppass",
		SMTPFromAddress:             "noreply@example.com",
		AppBaseURL:                  "https://pwd.example.com",
		ResetUser:                   "cn=reset,dc=example,dc=com",
		ResetPassword:               "resetpass",
	}

	// Verify struct fields are accessible and have expected values
	assert.Equal(t, "3000", opts.Port)
	assert.Equal(t, "cn=readonly,dc=example,dc=com", opts.ReadonlyUser)
	assert.Equal(t, "secret", opts.ReadonlyPassword)
	assert.Equal(t, uint(8), opts.MinLength)
	assert.Equal(t, uint(1), opts.MinNumbers)
	assert.Equal(t, uint(1), opts.MinSymbols)
	assert.Equal(t, uint(1), opts.MinUppercase)
	assert.Equal(t, uint(1), opts.MinLowercase)
	assert.False(t, opts.PasswordCanIncludeUsername)
	assert.True(t, opts.PasswordResetEnabled)
	assert.Equal(t, uint(15), opts.ResetTokenExpiryMinutes)
	assert.Equal(t, uint(3), opts.ResetRateLimitRequests)
	assert.Equal(t, uint(60), opts.ResetRateLimitWindowMinutes)
	assert.Equal(t, "smtp.example.com", opts.SMTPHost)
	assert.Equal(t, uint(587), opts.SMTPPort)
	assert.Equal(t, "smtpuser", opts.SMTPUsername)
	assert.Equal(t, "smtppass", opts.SMTPPassword)
	assert.Equal(t, "noreply@example.com", opts.SMTPFromAddress)
	assert.Equal(t, "https://pwd.example.com", opts.AppBaseURL)
	assert.Equal(t, "cn=reset,dc=example,dc=com", opts.ResetUser)
	assert.Equal(t, "resetpass", opts.ResetPassword)
}

// TestEnvIntOrDefaultEdgeCases tests edge cases for default values.
func TestEnvIntOrDefaultEdgeCases(t *testing.T) {
	// Test default value boundaries
	tests := []struct {
		name       string
		defaultVal uint64
		want       uint
	}{
		{
			name:       "default 0",
			defaultVal: 0,
			want:       0,
		},
		{
			name:       "default max uint16",
			defaultVal: 65535,
			want:       65535,
		},
		{
			name:       "default typical port",
			defaultVal: 3000,
			want:       3000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use a unique env name that won't be set
			envName := "TEST_INT_EDGE_" + tt.name
			errs := &ConfigError{}
			got := envIntOrDefault(envName, tt.defaultVal, errs)
			assert.Equal(t, tt.want, got)
			assert.False(t, errs.HasErrors())
		})
	}
}

// TestEnvStringOrDefaultConcurrent tests concurrent access to env functions.
func TestEnvStringOrDefaultConcurrent(t *testing.T) {
	const envName = "TEST_CONCURRENT_ENV"
	const envValue = "concurrent_value"

	t.Setenv(envName, envValue)

	// Run concurrent reads
	done := make(chan bool)
	for range 100 {
		go func() {
			result := envStringOrDefault(envName, "default")
			require.Equal(t, envValue, result)
			done <- true
		}()
	}

	// Wait for all goroutines
	for range 100 {
		<-done
	}
}

// TestEnvBoolOrDefaultVariations tests various boolean string representations.
func TestEnvBoolOrDefaultVariations(t *testing.T) {
	tests := []struct {
		envValue string
		want     bool
	}{
		{"true", true},
		{"True", true},
		{"TRUE", true},
		{"1", true},
		{"t", true},
		{"T", true},
		{"false", false},
		{"False", false},
		{"FALSE", false},
		{"0", false},
		{"f", false},
		{"F", false},
	}

	for _, tt := range tests {
		t.Run(tt.envValue, func(t *testing.T) {
			envName := "TEST_BOOL_VAR_" + tt.envValue
			t.Setenv(envName, tt.envValue)

			errs := &ConfigError{}
			got := envBoolOrDefault(envName, !tt.want, errs)
			assert.Equal(t, tt.want, got)
			assert.False(t, errs.HasErrors())
		})
	}
}

// TestConfigError tests the ConfigError type.
func TestConfigError(t *testing.T) {
	t.Run("empty error has no errors", func(t *testing.T) {
		errs := &ConfigError{}
		assert.False(t, errs.HasErrors())
		assert.Empty(t, errs.Errors)
	})

	t.Run("Add appends errors", func(t *testing.T) {
		errs := &ConfigError{}
		errs.Add("first error")
		errs.Add("second error")
		assert.True(t, errs.HasErrors())
		assert.Len(t, errs.Errors, 2)
		assert.Contains(t, errs.Errors, "first error")
		assert.Contains(t, errs.Errors, "second error")
	})

	t.Run("Error returns joined message", func(t *testing.T) {
		errs := &ConfigError{}
		errs.Add("invalid port")
		errs.Add("missing server")
		msg := errs.Error()
		assert.Contains(t, msg, "configuration errors:")
		assert.Contains(t, msg, "invalid port")
		assert.Contains(t, msg, "missing server")
	})
}

// TestParseWithMissingRequired tests Parse with missing required options.
func TestParseWithMissingRequired(t *testing.T) {
	// Clear all required env vars to test missing required options
	t.Setenv("LDAP_SERVER", "")
	t.Setenv("LDAP_BASE_DN", "")
	t.Setenv("LDAP_READONLY_USER", "")
	t.Setenv("LDAP_READONLY_PASSWORD", "")

	// Parse should fail due to missing required options
	_, err := Parse()
	require.Error(t, err)

	var configErr *ConfigError
	require.True(t, errors.As(err, &configErr), "error should be *ConfigError")
	assert.True(t, configErr.HasErrors())
	assert.Contains(t, configErr.Error(), "required options missing")
}

// TestParseWithInvalidIntValue tests Parse with invalid integer values.
func TestParseWithInvalidIntValue(t *testing.T) {
	// Set required options
	t.Setenv("LDAP_SERVER", "ldap://localhost")
	t.Setenv("LDAP_BASE_DN", "dc=example,dc=com")
	t.Setenv("LDAP_READONLY_USER", "cn=admin")
	t.Setenv("LDAP_READONLY_PASSWORD", "secret")

	// Set invalid integer value
	t.Setenv("MIN_LENGTH", "not_a_number")

	_, err := Parse()
	require.Error(t, err)

	var configErr *ConfigError
	require.True(t, errors.As(err, &configErr), "error should be *ConfigError")
	assert.Contains(t, configErr.Error(), "invalid value for MIN_LENGTH")
}

// TestParseWithInvalidBoolValue tests Parse with invalid boolean values.
func TestParseWithInvalidBoolValue(t *testing.T) {
	// Set required options
	t.Setenv("LDAP_SERVER", "ldap://localhost")
	t.Setenv("LDAP_BASE_DN", "dc=example,dc=com")
	t.Setenv("LDAP_READONLY_USER", "cn=admin")
	t.Setenv("LDAP_READONLY_PASSWORD", "secret")

	// Set invalid boolean value
	t.Setenv("LDAP_IS_AD", "maybe")

	_, err := Parse()
	require.Error(t, err)

	var configErr *ConfigError
	require.True(t, errors.As(err, &configErr), "error should be *ConfigError")
	assert.Contains(t, configErr.Error(), "invalid value for LDAP_IS_AD")
}
