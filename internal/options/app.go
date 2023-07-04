package options

import "flag"

type Opts struct {
	LdapServer        string
	IsActiveDirectory bool
	BaseDN            string
	ReadonlyUser      string
	ReadonlyPassword  string

	MinLength                  int
	MinNumbers                 int
	MinSymbols                 int
	MinUppercase               int
	MinLowercase               int
	PasswordCanIncludeUsername bool
}

var (
	fLdapServer        = flag.String("ldap-server", "ldaps://localhost:636", "LDAP server URI, has to begin with `ldap://` or `ldaps://`. If this is an ActiveDirectory server, this *has* to be `ldaps://`.")
	fIsActiveDirectory = flag.Bool("active-directory", false, "Mark the LDAP server as ActiveDirectory.")
	fBaseDN            = flag.String("base-dn", "", "Base DN of your LDAP directory.")
	fReadonlyUser      = flag.String("readonly-user", "", "User that can read all users in your LDAP directory.")
	fReadonlyPassword  = flag.String("readonly-password", "", "Password for the readonly user.")

	fMinLength                  = flag.Int("min-length", 8, "Minimum length of the password.")
	fMinNumbers                 = flag.Int("min-numbers", 1, "Minimum amount of numbers in the password.")
	fMinSymbols                 = flag.Int("min-symbols", 1, "Minimum amount of symbols in the password.")
	fMinUppercase               = flag.Int("min-uppercase", 1, "Minimum amount of uppercase letters in the password.")
	fMinLowercase               = flag.Int("min-lowercase", 1, "Minimum amount of lowercase letters in the password.")
	fPasswordCanIncludeUsername = flag.Bool("password-can-match-username", false, "Enables that the password can match the password")
)

func panicWhenLtZero(name string, value *int) {
	if *value < 0 {
		panic("The option --" + name + " has to be greater than zero")
	}
}

func panicWhenLteZero(name string, value *int) {
	if *value <= 0 {
		panic("The option --" + name + " has to be greater than zero")
	}
}

func panicWhenEmpty(name string, value *string) {
	if *value == "" {
		panic("The option --" + name + " is required")
	}
}

func Parse() *Opts {
	if !flag.Parsed() {
		flag.Parse()
	}

	panicWhenEmpty("ldap-server", fLdapServer)
	panicWhenEmpty("base-dn", fBaseDN)
	panicWhenEmpty("readonly-user", fReadonlyUser)
	panicWhenEmpty("readonly-password", fReadonlyPassword)

	panicWhenLteZero("min-length", fMinLength)
	panicWhenLtZero("min-numbers", fMinNumbers)
	panicWhenLtZero("min-symbols", fMinSymbols)
	panicWhenLtZero("min-uppercase", fMinUppercase)

	return &Opts{
		LdapServer:        *fLdapServer,
		IsActiveDirectory: *fIsActiveDirectory,
		BaseDN:            *fBaseDN,
		ReadonlyUser:      *fReadonlyUser,
		ReadonlyPassword:  *fReadonlyPassword,

		MinLength:                  *fMinLength,
		MinNumbers:                 *fMinNumbers,
		MinSymbols:                 *fMinSymbols,
		MinUppercase:               *fMinUppercase,
		MinLowercase:               *fMinLowercase,
		PasswordCanIncludeUsername: *fPasswordCanIncludeUsername,
	}
}
