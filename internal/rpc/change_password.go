package rpc

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/validators"
)

func pluralize(word string, amount uint) string {
	if amount == 1 {
		return word
	}

	return word + "s"
}

// changePassword handles password change requests without IP context.
// This is maintained for backward compatibility with existing tests.
// New code should use changePasswordWithIP for IP-based rate limiting.
func (c *Handler) changePassword(params []string) ([]string, error) {
	// For backward compatibility, call the IP-aware version with a placeholder IP
	// In production, this should not be called - Handle() uses changePasswordWithIP
	return c.changePasswordWithIP(params, "0.0.0.0")
}

// changePasswordWithIP handles password change requests with IP-based rate limiting.
func (c *Handler) changePasswordWithIP(params []string, clientIP string) ([]string, error) {
	if len(params) != 3 {
		return nil, ErrInvalidArgumentCount
	}

	sAMAccountName := params[0]
	currentPassword := params[1]
	newPassword := params[2]

	// Check IP-based rate limit to prevent brute force attacks
	if c.ipLimiter != nil && !c.ipLimiter.AllowRequest(clientIP) {
		slog.Warn("password_change_ip_rate_limited", "ip", clientIP, "username", sAMAccountName)
		return nil, fmt.Errorf("too many password change attempts from your IP address, please try again later")
	}

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

	if len(newPassword) < int(c.opts.MinLength) {
		return nil, fmt.Errorf("the new password must be at least %d characters long", c.opts.MinLength)
	}

	const MaxPasswordLength = 128 // LDAP typical maximum
	if len(newPassword) > MaxPasswordLength {
		return nil, fmt.Errorf("the new password must not exceed %d characters", MaxPasswordLength)
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

	if !c.opts.PasswordCanIncludeUsername && sAMAccountName != "" && strings.Contains(strings.ToLower(newPassword), strings.ToLower(sAMAccountName)) {
		return nil, fmt.Errorf("the new password must not include the username")
	}

	if err := c.ldap.ChangePasswordForSAMAccountName(sAMAccountName, currentPassword, newPassword); err != nil {
		slog.Error("password_change_failed", "username", sAMAccountName, "error", err)
		return nil, fmt.Errorf("failed to change password, please verify your current password is correct and try again")
	}

	slog.Info("password_changed", "username", sAMAccountName)
	return []string{"password changed successfully"}, nil
}
