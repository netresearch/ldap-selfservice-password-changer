// Package options handles application configuration and command-line flag parsing.
package options

import (
	"flag"
	"log/slog"
	"os"
	"strconv"

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

func envIntOrDefault(name string, d uint64) uint {
	raw := envStringOrDefault(name, strconv.FormatUint(d, 10))

	v, err := strconv.ParseUint(raw, 10, 16)
	if err != nil {
		slog.Error("failed to parse environment variable as uint", "variable", name, "value", raw, "error", err)
		os.Exit(1)
	}

	return uint(v)
}

func envBoolOrDefault(name string, d bool) bool {
	raw := envStringOrDefault(name, strconv.FormatBool(d))

	v2, err := strconv.ParseBool(raw)
	if err != nil {
		slog.Error("failed to parse environment variable as bool", "variable", name, "value", raw, "error", err)
		os.Exit(1)
	}

	return v2
}

// Parse parses command-line flags and environment variables into configuration options.
func Parse() *Opts {
	// Load .env files if they exist (native runs)
	// When running in Docker Compose, env vars are already injected via env_file
	// Error is intentionally ignored - .env files are optional in containerized environments
	_ = godotenv.Load(".env.local", ".env")

	var (
		fPort       = flag.String("port", envStringOrDefault("PORT", "3000"), "Port to listen on.")
		fLdapServer = flag.String(
			"ldap-server",
			envStringOrDefault("LDAP_SERVER", ""),
			"LDAP server URI, has to begin with `ldap://` or `ldaps://`. "+
				"If this is an ActiveDirectory server, this *has* to be `ldaps://`.",
		)
		fIsActiveDirectory = flag.Bool(
			"active-directory",
			envBoolOrDefault("LDAP_IS_AD", false),
			"Mark the LDAP server as ActiveDirectory.",
		)
		fBaseDN       = flag.String("base-dn", envStringOrDefault("LDAP_BASE_DN", ""), "Base DN of your LDAP directory.")
		fReadonlyUser = flag.String(
			"readonly-user",
			envStringOrDefault("LDAP_READONLY_USER", ""),
			"User that can read all users in your LDAP directory.",
		)
		fReadonlyPassword = flag.String(
			"readonly-password",
			envStringOrDefault("LDAP_READONLY_PASSWORD", ""),
			"Password for the readonly user.",
		)

		fMinLength = flag.Uint(
			"min-length",
			envIntOrDefault("MIN_LENGTH", 8),
			"Minimum length of the password.",
		)
		fMinNumbers = flag.Uint(
			"min-numbers",
			envIntOrDefault("MIN_NUMBERS", 1),
			"Minimum amount of numbers in the password.",
		)
		fMinSymbols = flag.Uint(
			"min-symbols",
			envIntOrDefault("MIN_SYMBOLS", 1),
			"Minimum amount of symbols in the password.",
		)
		fMinUppercase = flag.Uint(
			"min-uppercase",
			envIntOrDefault("MIN_UPPERCASE", 1),
			"Minimum amount of uppercase letters in the password.",
		)
		fMinLowercase = flag.Uint(
			"min-lowercase",
			envIntOrDefault("MIN_LOWERCASE", 1),
			"Minimum amount of lowercase letters in the password.",
		)
		fPasswordCanIncludeUsername = flag.Bool(
			"password-can-include-username",
			envBoolOrDefault("PASSWORD_CAN_INCLUDE_USERNAME", false),
			"Enables that the password can include the password",
		)

		// Password Reset flags
		fPasswordResetEnabled = flag.Bool(
			"password-reset-enabled",
			envBoolOrDefault("PASSWORD_RESET_ENABLED", false),
			"Enable password reset feature.",
		)
		fResetTokenExpiryMinutes = flag.Uint(
			"reset-token-expiry-minutes",
			envIntOrDefault("RESET_TOKEN_EXPIRY_MINUTES", 15),
			"Token validity duration in minutes.",
		)
		fResetRateLimitRequests = flag.Uint(
			"reset-rate-limit-requests",
			envIntOrDefault("RESET_RATE_LIMIT_REQUESTS", 3),
			"Max password reset requests per time window.",
		)
		fResetRateLimitWindowMinutes = flag.Uint(
			"reset-rate-limit-window-minutes",
			envIntOrDefault("RESET_RATE_LIMIT_WINDOW_MINUTES", 60),
			"Rate limit time window in minutes.",
		)
		fSMTPHost = flag.String(
			"smtp-host",
			envStringOrDefault("SMTP_HOST", "smtp.gmail.com"),
			"SMTP server hostname.",
		)
		fSMTPPort     = flag.Uint("smtp-port", envIntOrDefault("SMTP_PORT", 587), "SMTP server port.")
		fSMTPUsername = flag.String(
			"smtp-username",
			envStringOrDefault("SMTP_USERNAME", ""),
			"SMTP authentication username.",
		)
		fSMTPPassword = flag.String(
			"smtp-password",
			envStringOrDefault("SMTP_PASSWORD", ""),
			"SMTP authentication password.",
		)
		fSMTPFromAddress = flag.String(
			"smtp-from-address",
			envStringOrDefault("SMTP_FROM_ADDRESS", ""),
			"Email sender address.",
		)
		fAppBaseURL = flag.String(
			"app-base-url",
			envStringOrDefault("APP_BASE_URL", ""),
			"Base URL for password reset links (e.g., https://pwd.example.com).",
		)

		// Optional dedicated service account for password reset (recommended for security)
		fResetUser = flag.String(
			"reset-user",
			envStringOrDefault("LDAP_RESET_USER", ""),
			"Optional dedicated user for password reset operations. "+
				"Falls back to readonly-user if not set.",
		)
		fResetPassword = flag.String(
			"reset-password",
			envStringOrDefault("LDAP_RESET_PASSWORD", ""),
			"Password for the dedicated reset user.",
		)
	)

	if !flag.Parsed() {
		flag.Parse()
	}

	// Collect all missing required options
	var missing []string
	checkRequired("ldap-server", fLdapServer, &missing)
	checkRequired("base-dn", fBaseDN, &missing)
	checkRequired("readonly-user", fReadonlyUser, &missing)
	checkRequired("readonly-password", fReadonlyPassword, &missing)

	// Report all missing options at once
	if len(missing) > 0 {
		slog.Error("required options missing", "options", missing)
		os.Exit(1)
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
	}
}
