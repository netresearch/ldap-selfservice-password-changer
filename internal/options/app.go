package options

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	ldap "github.com/netresearch/simple-ldap-go"
)

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
}

func panicWhenEmpty(name string, value *string) {
	if *value == "" {
		slog.Error("required option missing", "option", name)
		os.Exit(1)
	}
}

func envStringOrDefault(name, d string) string {
	if v, exists := os.LookupEnv(name); exists && v != "" {
		return v
	}

	return d
}

func envIntOrDefault(name string, d uint64) uint {
	raw := envStringOrDefault(name, fmt.Sprintf("%v", d))

	v, err := strconv.ParseUint(raw, 10, 8)
	if err != nil {
		slog.Error("failed to parse environment variable as uint", "variable", name, "value", raw, "error", err)
		os.Exit(1)
	}

	return uint(v)
}

func envBoolOrDefault(name string, d bool) bool {
	raw := envStringOrDefault(name, fmt.Sprintf("%v", d))

	v2, err := strconv.ParseBool(raw)
	if err != nil {
		slog.Error("failed to parse environment variable as bool", "variable", name, "value", raw, "error", err)
		os.Exit(1)
	}

	return v2
}

func Parse() *Opts {
	if err := godotenv.Load(".env.local", ".env"); err != nil {
		slog.Warn("could not load .env file", "error", err)
	}

	var (
		fPort              = flag.String("port", envStringOrDefault("PORT", "3000"), "Port to listen on.")
		fLdapServer        = flag.String("ldap-server", envStringOrDefault("LDAP_SERVER", ""), "LDAP server URI, has to begin with `ldap://` or `ldaps://`. If this is an ActiveDirectory server, this *has* to be `ldaps://`.")
		fIsActiveDirectory = flag.Bool("active-directory", envBoolOrDefault("LDAP_IS_AD", false), "Mark the LDAP server as ActiveDirectory.")
		fBaseDN            = flag.String("base-dn", envStringOrDefault("LDAP_BASE_DN", ""), "Base DN of your LDAP directory.")
		fReadonlyUser      = flag.String("readonly-user", envStringOrDefault("LDAP_READONLY_USER", ""), "User that can read all users in your LDAP directory.")
		fReadonlyPassword  = flag.String("readonly-password", envStringOrDefault("LDAP_READONLY_PASSWORD", ""), "Password for the readonly user.")

		fMinLength                  = flag.Uint("min-length", envIntOrDefault("MIN_LENGTH", 8), "Minimum length of the password.")
		fMinNumbers                 = flag.Uint("min-numbers", envIntOrDefault("MIN_NUMBERS", 1), "Minimum amount of numbers in the password.")
		fMinSymbols                 = flag.Uint("min-symbols", envIntOrDefault("MIN_SYMBOLS", 1), "Minimum amount of symbols in the password.")
		fMinUppercase               = flag.Uint("min-uppercase", envIntOrDefault("MIN_UPPERCASE", 1), "Minimum amount of uppercase letters in the password.")
		fMinLowercase               = flag.Uint("min-lowercase", envIntOrDefault("MIN_LOWERCASE", 1), "Minimum amount of lowercase letters in the password.")
		fPasswordCanIncludeUsername = flag.Bool("password-can-include-username", envBoolOrDefault("PASSWORD_CAN_INCLUDE_USERNAME", false), "Enables that the password can include the password")
	)

	if !flag.Parsed() {
		flag.Parse()
	}

	panicWhenEmpty("ldap-server", fLdapServer)
	panicWhenEmpty("base-dn", fBaseDN)
	panicWhenEmpty("readonly-user", fReadonlyUser)
	panicWhenEmpty("readonly-password", fReadonlyPassword)

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
	}
}
