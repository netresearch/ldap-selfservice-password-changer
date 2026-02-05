// Package options handles application configuration and command-line flag parsing.
package options

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	ldap "github.com/netresearch/simple-ldap-go"
)

// Opts holds application configuration from environment variables and command-line flags.
type Opts struct {
	Port             string
	LDAP             ldap.Config
	ReadonlyUser     string
	ReadonlyPassword string

	MinLength                  uint
	MinNumbers                 uint
	MinSymbols                 uint
	MinUppercase               uint
	MinLowercase               uint
	PasswordCanIncludeUsername bool

	// Password Reset Configuration
	PasswordResetEnabled        bool
	ResetTokenExpiryMinutes     uint
	ResetRateLimitRequests      uint
	ResetRateLimitWindowMinutes uint
	SMTPHost                    string
	SMTPPort                    uint
	SMTPUsername                string
	SMTPPassword                string
	SMTPFromAddress             string
	AppBaseURL                  string

	// Optional dedicated service account for password reset operations
	// If not set, falls back to ReadonlyUser for backward compatibility
	ResetUser     string
	ResetPassword string
}

// ConfigError represents a configuration validation error.
type ConfigError struct {
	Errors []string
}

func (e *ConfigError) Error() string {
	return "configuration errors: " + strings.Join(e.Errors, "; ")
}

// Add adds an error message to the ConfigError.
func (e *ConfigError) Add(msg string) {
	e.Errors = append(e.Errors, msg)
}

// HasErrors returns true if there are any errors.
func (e *ConfigError) HasErrors() bool {
	return len(e.Errors) > 0
}

func checkRequired(name string, value *string, missing *[]string) {
	if *value == "" {
		*missing = append(*missing, name)
	}
}

func envStringOrDefault(name, d string) string {
	if v, exists := os.LookupEnv(name); exists && v != "" {
		return v
	}

	return d
}

func envIntOrDefault(name string, d uint64, errs *ConfigError) uint {
	raw := envStringOrDefault(name, strconv.FormatUint(d, 10))

	v, err := strconv.ParseUint(raw, 10, strconv.IntSize)
	if err != nil {
		errs.Add(fmt.Sprintf("invalid value for %s: %q is not a valid unsigned integer", name, raw))
		return uint(d) // Return default on error
	}

	return uint(v)
}

func envBoolOrDefault(name string, d bool, errs *ConfigError) bool {
	raw := envStringOrDefault(name, strconv.FormatBool(d))

	v, err := strconv.ParseBool(raw)
	if err != nil {
		errs.Add(fmt.Sprintf("invalid value for %s: %q is not a valid boolean", name, raw))
		return d // Return default on error
	}

	return v
}

// Parse parses command-line flags and environment variables into configuration options.
// Returns an error if required options are missing or values are invalid.
func Parse() (*Opts, error) {
	// Load .env files if they exist (native runs)
	// When running in Docker Compose, env vars are already injected via env_file
	//nolint:errcheck // .env files are optional in containerized environments
	_ = godotenv.Load(".env.local", ".env")

	// Collect all errors during parsing
	errs := &ConfigError{}

	// Use a custom FlagSet to allow multiple Parse() calls in tests
	fs := flag.NewFlagSet("gopherpass", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // Suppress flag parse error output; errors are collected in ConfigError

	var (
		fPort       = fs.String("port", envStringOrDefault("PORT", "3000"), "Port to listen on.")
		fLdapServer = fs.String(
			"ldap-server",
			envStringOrDefault("LDAP_SERVER", ""),
			"LDAP server URI, has to begin with `ldap://` or `ldaps://`. "+
				"If this is an ActiveDirectory server, this *has* to be `ldaps://`.",
		)
		fIsActiveDirectory = fs.Bool(
			"active-directory",
			envBoolOrDefault("LDAP_IS_AD", false, errs),
			"Mark the LDAP server as ActiveDirectory.",
		)
		fBaseDN       = fs.String("base-dn", envStringOrDefault("LDAP_BASE_DN", ""), "Base DN of your LDAP directory.")
		fReadonlyUser = fs.String(
			"readonly-user",
			envStringOrDefault("LDAP_READONLY_USER", ""),
			"User that can read all users in your LDAP directory.",
		)
		fReadonlyPassword = fs.String(
			"readonly-password",
			envStringOrDefault("LDAP_READONLY_PASSWORD", ""),
			"Password for the readonly user.",
		)

		fMinLength = fs.Uint(
			"min-length",
			envIntOrDefault("MIN_LENGTH", 8, errs),
			"Minimum length of the password.",
		)
		fMinNumbers = fs.Uint(
			"min-numbers",
			envIntOrDefault("MIN_NUMBERS", 1, errs),
			"Minimum amount of numbers in the password.",
		)
		fMinSymbols = fs.Uint(
			"min-symbols",
			envIntOrDefault("MIN_SYMBOLS", 1, errs),
			"Minimum amount of symbols in the password.",
		)
		fMinUppercase = fs.Uint(
			"min-uppercase",
			envIntOrDefault("MIN_UPPERCASE", 1, errs),
			"Minimum amount of uppercase letters in the password.",
		)
		fMinLowercase = fs.Uint(
			"min-lowercase",
			envIntOrDefault("MIN_LOWERCASE", 1, errs),
			"Minimum amount of lowercase letters in the password.",
		)
		fPasswordCanIncludeUsername = fs.Bool(
			"password-can-include-username",
			envBoolOrDefault("PASSWORD_CAN_INCLUDE_USERNAME", false, errs),
			"Enables that the password can include the username.",
		)

		// Password Reset flags
		fPasswordResetEnabled = fs.Bool(
			"password-reset-enabled",
			envBoolOrDefault("PASSWORD_RESET_ENABLED", false, errs),
			"Enable password reset feature.",
		)
		fResetTokenExpiryMinutes = fs.Uint(
			"reset-token-expiry-minutes",
			envIntOrDefault("RESET_TOKEN_EXPIRY_MINUTES", 15, errs),
			"Token validity duration in minutes.",
		)
		fResetRateLimitRequests = fs.Uint(
			"reset-rate-limit-requests",
			envIntOrDefault("RESET_RATE_LIMIT_REQUESTS", 3, errs),
			"Max password reset requests per time window.",
		)
		fResetRateLimitWindowMinutes = fs.Uint(
			"reset-rate-limit-window-minutes",
			envIntOrDefault("RESET_RATE_LIMIT_WINDOW_MINUTES", 60, errs),
			"Rate limit time window in minutes.",
		)
		fSMTPHost = fs.String(
			"smtp-host",
			envStringOrDefault("SMTP_HOST", "smtp.gmail.com"),
			"SMTP server hostname.",
		)
		fSMTPPort     = fs.Uint("smtp-port", envIntOrDefault("SMTP_PORT", 587, errs), "SMTP server port.")
		fSMTPUsername = fs.String(
			"smtp-username",
			envStringOrDefault("SMTP_USERNAME", ""),
			"SMTP authentication username.",
		)
		fSMTPPassword = fs.String(
			"smtp-password",
			envStringOrDefault("SMTP_PASSWORD", ""),
			"SMTP authentication password.",
		)
		fSMTPFromAddress = fs.String(
			"smtp-from-address",
			envStringOrDefault("SMTP_FROM_ADDRESS", ""),
			"Email sender address.",
		)
		fAppBaseURL = fs.String(
			"app-base-url",
			envStringOrDefault("APP_BASE_URL", ""),
			"Base URL for password reset links (e.g., https://pwd.example.com).",
		)

		// Optional dedicated service account for password reset (recommended for security)
		fResetUser = fs.String(
			"reset-user",
			envStringOrDefault("LDAP_RESET_USER", ""),
			"Optional dedicated user for password reset operations. "+
				"Falls back to readonly-user if not set.",
		)
		fResetPassword = fs.String(
			"reset-password",
			envStringOrDefault("LDAP_RESET_PASSWORD", ""),
			"Password for the dedicated reset user.",
		)
	)

	// Parse command-line arguments (skip program name)
	if err := fs.Parse(os.Args[1:]); err != nil {
		errs.Add(fmt.Sprintf("flag parsing error: %v", err))
	}

	// Collect all missing required options
	var missing []string
	checkRequired("ldap-server", fLdapServer, &missing)
	checkRequired("base-dn", fBaseDN, &missing)
	checkRequired("readonly-user", fReadonlyUser, &missing)
	checkRequired("readonly-password", fReadonlyPassword, &missing)

	// Add missing options to errors
	if len(missing) > 0 {
		errs.Add("required options missing: " + strings.Join(missing, ", "))
	}

	// Return error if any validation failed
	if errs.HasErrors() {
		return nil, errs
	}

	return &Opts{
		Port: *fPort,
		LDAP: ldap.Config{
			Server:            *fLdapServer,
			BaseDN:            *fBaseDN,
			IsActiveDirectory: *fIsActiveDirectory,
		},
		ReadonlyUser:     *fReadonlyUser,
		ReadonlyPassword: *fReadonlyPassword,

		MinLength:                  *fMinLength,
		MinNumbers:                 *fMinNumbers,
		MinSymbols:                 *fMinSymbols,
		MinUppercase:               *fMinUppercase,
		MinLowercase:               *fMinLowercase,
		PasswordCanIncludeUsername: *fPasswordCanIncludeUsername,

		PasswordResetEnabled:        *fPasswordResetEnabled,
		ResetTokenExpiryMinutes:     *fResetTokenExpiryMinutes,
		ResetRateLimitRequests:      *fResetRateLimitRequests,
		ResetRateLimitWindowMinutes: *fResetRateLimitWindowMinutes,
		SMTPHost:                    *fSMTPHost,
		SMTPPort:                    *fSMTPPort,
		SMTPUsername:                *fSMTPUsername,
		SMTPPassword:                *fSMTPPassword,
		SMTPFromAddress:             *fSMTPFromAddress,
		AppBaseURL:                  *fAppBaseURL,

		ResetUser:     *fResetUser,
		ResetPassword: *fResetPassword,
	}, nil
}

// MustParse is like Parse but prints an error to stderr and exits with status 1 on failure.
// Use this in main() when you want the old behavior.
//
// Deprecated: Prefer Parse() and handle errors explicitly.
func MustParse() *Opts {
	opts, err := Parse()
	if err != nil {
		// Use standard library to avoid import cycle
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	return opts
}
