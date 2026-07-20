package rpchandler

import (
	"errors"
	"log/slog"
	"strings"
	"time"

	ldap "github.com/netresearch/simple-ldap-go"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/options"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/resettoken"
)

// errIdentifierRejected is returned by resolveResetUser when the supplied
// identifier does not match the configured mode (e.g. a username entered while
// the form only accepts email addresses). It is treated as "no user" by the
// caller to preserve the enumeration-safe generic response.
var errIdentifierRejected = errors.New("identifier rejected for configured reset mode")

// msgResetEmailSent is the enumeration-safe success message always returned
// for password reset requests, regardless of whether the account exists.
const msgResetEmailSent = "If an account exists, a reset email has been sent"

// EmailService interface for sending password reset emails.
type EmailService interface {
	SendResetEmail(to, token string) error
}

// RateLimiter interface for rate limiting password reset requests.
type RateLimiter interface {
	AllowRequest(identifier string) bool
}

// TokenStore interface for managing reset tokens.
type TokenStore interface {
	Store(token *resettoken.ResetToken) error
	Get(tokenString string) (*resettoken.ResetToken, error)
	MarkUsed(tokenString string) error
	Delete(tokenString string) error
	CleanupExpired() int
	Count() int
}

// requestPasswordReset handles password reset requests without IP context.
// This is maintained for backward compatibility with existing tests.
// New code should use requestPasswordResetWithIP for IP-based rate limiting.
func (h *Handler) requestPasswordReset(params []string) ([]string, error) {
	// For backward compatibility, call the IP-aware version with a placeholder IP
	// In production, this should not be called - Handle() uses requestPasswordResetWithIP
	return h.requestPasswordResetWithIP(params, "0.0.0.0")
}

// requestPasswordResetWithIP handles password reset requests with IP-based rate limiting.
// Always returns a generic success message to prevent user enumeration.
func (h *Handler) requestPasswordResetWithIP(params []string, clientIP string) ([]string, error) {
	// Validate parameter count
	if len(params) != 1 {
		return nil, ErrInvalidArgumentCount
	}

	emailOrUsername := params[0]

	// Generic success message (always returned to prevent enumeration)
	genericSuccess := []string{msgResetEmailSent}

	// Validate email length (RFC 5321 maximum)
	const MaxEmailLength = 254
	if len(emailOrUsername) > MaxEmailLength {
		// Return generic success to prevent enumeration
		slog.Warn("password_reset_email_too_long", "length", len(emailOrUsername))
		return genericSuccess, nil
	}

	// FIRST: Check IP-based rate limit (stricter, catches flooding)
	// This prevents attackers from using different emails to bypass rate limiting
	if h.ipLimiter != nil && !h.ipLimiter.AllowRequest(clientIP) {
		// IP is rate limited - return success but don't proceed
		slog.Warn("password_reset_ip_rate_limited", "ip", clientIP)
		return genericSuccess, nil
	}

	// SECOND: Check email-based rate limit (per-user protection)
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

	// Resolve the target account according to the configured identifier mode.
	// The lookup is authoritative server-side; a mismatch, an ambiguous email,
	// or any LDAP error all collapse to the generic success below.
	user, viaEmail, err := h.resolveResetUser(emailOrUsername)
	if err != nil {
		slog.Info("password_reset_user_not_resolved", "identifier", emailOrUsername, "error", err)
		return genericSuccess, nil
	}

	username := user.SAMAccountName

	// THIRD: Check the rate limit for the RESOLVED account. The pre-lookup
	// check above is keyed on the typed string, so one mailbox reachable via
	// several identifiers (email and username in "both" mode, or an account
	// with multiple mail values) would otherwise get one bucket per spelling,
	// multiplying the reset emails deliverable to it. The "account:" prefix
	// keeps this bucket disjoint from typed-identifier buckets (":" cannot
	// occur in a sAMAccountName).
	if !h.rateLimiter.AllowRequest("account:" + username) {
		slog.Warn("password_reset_account_rate_limited", "username", username)
		return genericSuccess, nil
	}

	// The reset link is always sent to the account's LDAP-registered address,
	// never to an arbitrary string typed by the requester. When the user was
	// found by email, the typed value IS a registered address (LDAP matched a
	// single account on it); otherwise use the address stored in the directory.
	recipient := emailOrUsername
	if !viaEmail {
		if user.Mail == nil || *user.Mail == "" {
			slog.Warn("password_reset_no_registered_mail", "username", username)
			return genericSuccess, nil
		}
		recipient = *user.Mail
	}

	// Create token metadata
	now := time.Now()
	// Safe conversion: ResetTokenExpiryMinutes is uint, typically small value (15-60)
	// Convert to time.Duration for expiration calculation
	//nolint:gosec // G115: small config value, safe for int64
	expiryDuration := time.Duration(h.opts.ResetTokenExpiryMinutes) * time.Minute
	token := &resettoken.ResetToken{
		Token:            tokenString,
		Username:         username,
		Email:            recipient,
		CreatedAt:        now,
		ExpiresAt:        now.Add(expiryDuration),
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

	// Send reset email to the LDAP-registered address
	err = h.emailService.SendResetEmail(recipient, tokenString)
	if err != nil {
		// Log error internally but return success to user
		// Token remains in store for potential retry
		slog.Error("password_reset_email_failed", "username", username, "email", recipient, "error", err)
		return genericSuccess, nil
	}

	slog.Info("password_reset_requested", "username", username, "email", recipient)

	// Return generic success message
	return genericSuccess, nil
}

// resolveResetUser looks up the account for a password reset request according
// to the configured identifier mode. It returns the resolved user, whether the
// lookup was performed by email (true) or by username (false), and any lookup
// error. Callers treat every error as "no user" to keep the response
// enumeration-safe.
//
// An empty or unrecognized mode defaults to email-only, matching the behavior
// before this option existed.
func (h *Handler) resolveResetUser(identifier string) (user *ldap.User, viaEmail bool, err error) {
	looksLikeEmail := strings.Contains(identifier, "@")

	mode := h.opts.ResetIdentifierMode
	if !mode.Valid() {
		mode = options.ResetIdentifierEmail
	}

	switch mode {
	case options.ResetIdentifierUsername:
		user, err = h.ldap.FindUserBySAMAccountName(identifier)
		return user, false, err
	case options.ResetIdentifierBoth:
		if looksLikeEmail {
			user, err = h.ldap.FindUserByMail(identifier)
			return user, true, err
		}
		user, err = h.ldap.FindUserBySAMAccountName(identifier)
		return user, false, err
	case options.ResetIdentifierEmail:
		fallthrough
	default:
		if !looksLikeEmail {
			return nil, false, errIdentifierRejected
		}
		user, err = h.ldap.FindUserByMail(identifier)
		return user, true, err
	}
}

// ErrSMTPFailure is a placeholder error for SMTP failures in testing scenarios.
var ErrSMTPFailure = errors.New("SMTP failure")
