// Package rpchandler provides JSON-RPC 2.0 handlers for password management operations.
package rpchandler

import (
	"errors"
	"log/slog"
)

// msgPasswordChanged is the success message returned after a password change.
const msgPasswordChanged = "password changed successfully"

// changePassword handles password change requests. The per-IP limiter and the
// Turnstile verification run ahead of it as guards; see methodPolicies.
func (c *Handler) changePassword(params []string) ([]string, error) {
	if len(params) != 3 {
		return nil, ErrInvalidArgumentCount
	}

	sAMAccountName := params[0]
	currentPassword := params[1]
	newPassword := params[2]

	if sAMAccountName == "" {
		return nil, errors.New("the username can't be empty")
	}

	if currentPassword == "" {
		return nil, errors.New("the old password can't be empty")
	}

	if newPassword == "" {
		return nil, errors.New("the new password can't be empty")
	}

	if currentPassword == newPassword {
		return nil, errors.New("the old password can't be same as the new one")
	}

	// Validate new password requirements.
	if err := ValidateNewPassword(newPassword, sAMAccountName, c.opts); err != nil {
		return nil, err
	}

	if err := c.ldap.ChangePasswordForSAMAccountName(sAMAccountName, currentPassword, newPassword); err != nil {
		slog.Error("password_change_failed", "username", sAMAccountName, "error", err)
		return nil, errors.New("failed to change password, please verify your current password is correct and try again")
	}

	slog.Info("password_changed", "username", sAMAccountName)
	return []string{msgPasswordChanged}, nil
}
