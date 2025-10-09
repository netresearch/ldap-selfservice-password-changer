package rpc

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/resettoken"
)

// EmailService interface for sending password reset emails
type EmailService interface {
	SendResetEmail(to, token string) error
}

// RateLimiter interface for rate limiting password reset requests
type RateLimiter interface {
	AllowRequest(identifier string) bool
}

// TokenStore interface for managing reset tokens
type TokenStore interface {
	Store(token *resettoken.ResetToken) error
	Get(tokenString string) (*resettoken.ResetToken, error)
	MarkUsed(tokenString string) error
	Delete(tokenString string) error
	CleanupExpired() int
	Count() int
}

// requestPasswordReset handles password reset requests.
// Always returns a generic success message to prevent user enumeration.
func (h *Handler) requestPasswordReset(params []string) ([]string, error) {
	// Validate parameter count
	if len(params) != 1 {
		return nil, ErrInvalidArgumentCount
	}

	emailOrUsername := params[0]

	// Generic success message (always returned to prevent enumeration)
	genericSuccess := []string{"If an account exists, a reset email has been sent"}

	// Check rate limit
	if !h.rateLimiter.AllowRequest(emailOrUsername) {
		// User is rate limited - return success but don't proceed
		slog.Warn("password_reset_rate_limited", "email", emailOrUsername)
		return genericSuccess, nil
	}

	// Generate token
	tokenString, err := resettoken.GenerateToken()
	if err != nil {
		// Log error internally but return success to user
		slog.Error("password_reset_token_generation_failed", "error", err)
		return genericSuccess, nil
	}

	// Look up user in LDAP by email to get SAMAccountName/username
	// This validates the user exists and retrieves their username for the token
	user, err := h.ldap.FindUserByMail(emailOrUsername)
	if err != nil {
		// User not found or LDAP error - return generic success (don't reveal)
		slog.Info("password_reset_user_not_found", "email", emailOrUsername)
		return genericSuccess, nil
	}

	username := user.SAMAccountName

	// Create token metadata
	now := time.Now()
	token := &resettoken.ResetToken{
		Token:            tokenString,
		Username:         username,
		Email:            emailOrUsername,
		CreatedAt:        now,
		ExpiresAt:        now.Add(time.Duration(h.opts.ResetTokenExpiryMinutes) * time.Minute),
		Used:             false,
		RequiresApproval: false, // Phase 1: no admin approval
	}

	// Store token
	err = h.tokenStore.Store(token)
	if err != nil {
		// Log error internally but return success to user
		slog.Error("password_reset_token_storage_failed", "username", username, "error", err)
		return genericSuccess, nil
	}

	// Send reset email
	err = h.emailService.SendResetEmail(emailOrUsername, tokenString)
	if err != nil {
		// Log error internally but return success to user
		// Token remains in store for potential retry
		slog.Error("password_reset_email_failed", "username", username, "email", emailOrUsername, "error", err)
		return genericSuccess, nil
	}

	slog.Info("password_reset_requested", "username", username, "email", emailOrUsername)

	// Return generic success message
	return genericSuccess, nil
}

// Placeholder error for testing
var ErrSMTPFailure = fmt.Errorf("SMTP failure")
