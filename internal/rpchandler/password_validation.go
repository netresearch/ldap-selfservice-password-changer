package rpchandler

import (
	"errors"
	"fmt"
	"strings"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/options"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/validators"
)

// MaxPasswordLength defines the maximum allowed password length for LDAP systems.
const MaxPasswordLength = 128 // LDAP typical maximum.

func pluralize(word string, amount uint) string {
	if amount == 1 {
		return word
	}

	return word + "s"
}

// ValidateNewPassword validates a new password against configured requirements.
func ValidateNewPassword(password, username string, opts *options.Opts) error {
	// Check minimum length.
	if uint(len(password)) < opts.MinLength {
		return fmt.Errorf(
			"the new password must be at least %d characters long",
			opts.MinLength,
		)
	}

	// Check maximum length.
	if len(password) > MaxPasswordLength {
		return fmt.Errorf(
			"the new password must not exceed %d characters",
			MaxPasswordLength,
		)
	}

	// Check minimum numbers.
	if !validators.MinNumbersInString(password, opts.MinNumbers) {
		return fmt.Errorf(
			"the new password must contain at least %d %s",
			opts.MinNumbers,
			pluralize("number", opts.MinNumbers),
		)
	}

	// Check minimum symbols.
	if !validators.MinSymbolsInString(password, opts.MinSymbols) {
		return fmt.Errorf(
			"the new password must contain at least %d %s",
			opts.MinSymbols,
			pluralize("symbol", opts.MinSymbols),
		)
	}

	// Check minimum uppercase letters.
	if !validators.MinUppercaseLettersInString(password, opts.MinUppercase) {
		return fmt.Errorf(
			"the new password must contain at least %d uppercase %s",
			opts.MinUppercase,
			pluralize("letter", opts.MinUppercase),
		)
	}

	// Check minimum lowercase letters.
	if !validators.MinLowercaseLettersInString(password, opts.MinLowercase) {
		return fmt.Errorf(
			"the new password must contain at least %d lowercase %s",
			opts.MinLowercase,
			pluralize("letter", opts.MinLowercase),
		)
	}

	// Check username inclusion if configured.
	if !opts.PasswordCanIncludeUsername &&
		username != "" &&
		strings.Contains(strings.ToLower(password), strings.ToLower(username)) {
		return errors.New("the new password must not include the username")
	}

	return nil
}
