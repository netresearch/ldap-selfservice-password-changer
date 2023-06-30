package rpc

import (
	"fmt"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/validators"
)

func pluralize(word string, amount int) string {
	if amount == 1 {
		return word
	}

	return word + "s"
}

func (c *Handler) changePassword(params []string) ([]string, error) {
	if len(params) != 3 {
		return nil, ErrInvalidArgumentCount
	}

	sAMAccountName := params[0]
	currentPassword := params[1]
	newPassword := params[2]

	if sAMAccountName == "" {
		return nil, fmt.Errorf("the username can't be empty")
	}

	if currentPassword == "" {
		return nil, fmt.Errorf("the old password can't be empty")
	}

	if newPassword == "" {
		return nil, fmt.Errorf("the new password can't be empty")
	}

	if currentPassword == newPassword {
		return nil, fmt.Errorf("the old password can't be same as the new one")
	}

	if len(newPassword) < c.opts.MinLength {
		return nil, fmt.Errorf("the new password must be at least %d characters long", c.opts.MinLength)
	}

	if !validators.MinNumbersInString(newPassword, c.opts.MinNumbers) {
		return nil, fmt.Errorf("the new password must contain at least %d %s", c.opts.MinNumbers, pluralize("number", c.opts.MinNumbers))
	}

	if !validators.MinSymbolsInString(newPassword, c.opts.MinSymbols) {
		return nil, fmt.Errorf("the new password must contain at least %d %s", c.opts.MinSymbols, pluralize("symbol", c.opts.MinSymbols))
	}

	if !validators.MinUppercaseLettersInString(newPassword, c.opts.MinUppercase) {
		return nil, fmt.Errorf("the new password must contain at least %d uppercase %s", c.opts.MinUppercase, pluralize("letter", c.opts.MinUppercase))
	}

	if !validators.MinLowercaseLettersInString(newPassword, c.opts.MinLowercase) {
		return nil, fmt.Errorf("the new password must contain at least %d lowercase %s", c.opts.MinLowercase, pluralize("letter", c.opts.MinLowercase))
	}

	if err := c.ldap.ChangePasswordForSAMAccountName(sAMAccountName, currentPassword, newPassword); err != nil {
		return nil, err
	}

	return []string{"password changed successfully"}, nil
}
