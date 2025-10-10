package rpc

import (
	"errors"
	"log/slog"

	ldap "github.com/netresearch/simple-ldap-go"
)

// resetPassword handles completing a password reset with a valid token.
func (h *Handler) resetPassword(params []string) ([]string, error) {
	// Validate parameter count
	if len(params) != 2 {
		return nil, ErrInvalidArgumentCount
	}

	tokenString := params[0]
	newPassword := params[1]

	// Validate new password is not empty
	if newPassword == "" {
		return nil, errors.New("the new password can't be empty")
	}

	// Get token from store
	token, err := h.tokenStore.Get(tokenString)
	if err != nil {
		// Safely log token prefix (handle tokens shorter than 8 chars)
		prefix := tokenString
		if len(tokenString) > 8 {
			prefix = tokenString[:8]
		}
		slog.Warn("password_reset_invalid_token", "token_prefix", prefix)
		return nil, errors.New("invalid or expired token")
	}

	// Check if token is expired
	if token.IsExpired() {
		slog.Warn("password_reset_expired_token", "username", token.Username)
		return nil, errors.New("invalid or expired token")
	}

	// Check if token is already used
	if token.Used {
		slog.Warn("password_reset_reused_token", "username", token.Username)
		return nil, errors.New("invalid or expired token")
	}

	// Validate password against policy requirements.
	if err := ValidateNewPassword(newPassword, token.Username, h.opts); err != nil {
		return nil, err
	}

	// Validate username is present in token
	if token.Username == "" {
		return nil, errors.New("invalid or expired token")
	}

	// Lazy-initialize reset LDAP client if needed
	// This prevents startup failures if reset account credentials are invalid
	if h.resetLDAP == nil {
		if h.opts.ResetUser != "" && h.opts.ResetPassword != "" {
			h.resetLDAP, err = ldap.New(h.opts.LDAP, h.opts.ResetUser, h.opts.ResetPassword)
			if err != nil {
				slog.Error("password_reset_ldap_init_failed", "username", token.Username, "error", err)
				return nil, errors.New("failed to initialize password reset connection; please contact your administrator")
			}
		} else {
			// Should never happen due to handler initialization logic
			slog.Error("password_reset_not_configured", "username", token.Username)
			return nil, errors.New("password reset not properly configured; please contact your administrator")
		}
	}

	// Reset password in LDAP using dedicated reset client
	// IMPORTANT: This uses ResetPasswordForSAMAccountName (administrative reset, no old password)
	// This operation requires the LDAP service account to have:
	//   - Active Directory: "Reset password" permission on user objects
	//   - OpenLDAP: Write access to userPassword attribute
	//
	// Configuration notes:
	//   - For AD: Grant "Reset password" permission to the service account
	//   - For OpenLDAP: Ensure service account has appropriate ACL permissions
	//   - Connection must use LDAPS (ldaps://) for security
	//   - Best practice: Use dedicated LDAP_RESET_USER with minimal permissions
	//   - Fallback: Uses LDAP_READONLY_USER if LDAP_RESET_USER not configured
	err = h.resetLDAP.ResetPasswordForSAMAccountName(token.Username, newPassword)
	if err != nil {
		// Generic error to user, detailed error in logs
		slog.Error("password_reset_failed", "username", token.Username, "error", err)
		return nil, errors.New("failed to reset password; please contact your administrator if this problem persists")
	}

	// Mark token as used
	err = h.tokenStore.MarkUsed(tokenString)
	if err != nil {
		// Log error but proceed (password was already changed)
		slog.Warn("password_reset_token_mark_used_failed", "username", token.Username, "error", err)
	}

	slog.Info("password_reset_completed", "username", token.Username, "email", token.Email)

	// Return success message
	return []string{"Password reset successfully. You can now login."}, nil
}
