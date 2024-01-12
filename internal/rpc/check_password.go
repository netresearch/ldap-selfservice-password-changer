package rpc

import (
	"fmt"
	"strings"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/validators"
)

func (c *Handler) check(params []string) (any, error) {
	if len(params) != 4 {
		return nil, ErrInvalidArgumentCount
	}

	sAMAccountName := params[0]
	currentPassword := params[1]
	newPassword := params[2]
	newPasswordConfirm := params[3]

	errors := make([][]string, 4)
	errors[0] = make([]string, 0)
	errors[1] = make([]string, 0)
	errors[2] = make([]string, 0)
	errors[3] = make([]string, 0)

	if sAMAccountName == "" {
		errors[0] = append(errors[0], "The input must not be empty")
	}

	if currentPassword == "" {
		errors[1] = append(errors[1], "The input must not be empty")
	}

	if newPassword == "" {
		errors[2] = append(errors[2], "The input must not be empty")
	}
	if currentPassword == newPassword {
		errors[2] = append(errors[2], "The new password must be different from the current password")
	}
	if newPasswordConfirm == "" {
		errors[2] = append(errors[2], "The input must not be empty")
	}
	if newPassword != newPasswordConfirm {
		errors[2] = append(errors[2], "The passwords do not match")
	}
	if len(newPassword) < int(c.opts.MinLength) {
		errors[2] = append(errors[2], fmt.Sprintf("The new password must be at least %d characters long", c.opts.MinLength))
	}
	if !validators.MinNumbersInString(newPassword, c.opts.MinNumbers) {
		errors[2] = append(errors[2], fmt.Sprintf("The new password must contain at least %d %s", c.opts.MinNumbers, pluralize("number", c.opts.MinNumbers)))
	}
	if !validators.MinSymbolsInString(newPassword, c.opts.MinSymbols) {
		errors[2] = append(errors[2], fmt.Sprintf("The new password must contain at least %d %s", c.opts.MinSymbols, pluralize("symbol", c.opts.MinSymbols)))
	}
	if !validators.MinUppercaseLettersInString(newPassword, c.opts.MinUppercase) {
		errors[2] = append(errors[2], fmt.Sprintf("The new password must contain at least %d uppercase %s", c.opts.MinUppercase, pluralize("letter", c.opts.MinUppercase)))
	}
	if !validators.MinLowercaseLettersInString(newPassword, c.opts.MinLowercase) {
		errors[2] = append(errors[2], fmt.Sprintf("The new password must contain at least %d lowercase %s", c.opts.MinLowercase, pluralize("letter", c.opts.MinLowercase)))
	}
	if !c.opts.PasswordCanIncludeUsername && strings.Contains(sAMAccountName, newPassword) {
		errors[2] = append(errors[2], "The new password must not include the username")
	}

	if newPasswordConfirm == "" {
		errors[3] = append(errors[3], "The input must not be empty")
	}
	if newPassword != newPasswordConfirm {
		errors[3] = append(errors[3], "The passwords do not match")
	}

	return errors, nil
}
