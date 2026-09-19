package rpchandler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gofiber/fiber/v3"
	ldap "github.com/netresearch/simple-ldap-go"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/options"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/turnstile"
)

// Func is a type alias for RPC handler functions that process string parameters and return results or errors.
type Func = func(params []string) ([]string, error)

// LDAPClient interface for LDAP operations (enables testing).
type LDAPClient interface {
	FindUserByMail(mail string) (*ldap.User, error)
	FindUserBySAMAccountName(sAMAccountName string) (*ldap.User, error)
	ChangePasswordForSAMAccountName(sAMAccountName, oldPassword, newPassword string) error
	ResetPasswordForSAMAccountName(sAMAccountName, newPassword string) error
}

// Handler processes JSON-RPC 2.0 requests for password management operations.
type Handler struct {
	ldap         LDAPClient
	resetLDAP    LDAPClient // Optional dedicated client for password reset operations (lazy-initialized)
	opts         *options.Opts
	tokenStore   TokenStore
	emailService EmailService
	rateLimiter  RateLimiter
	ipLimiter    IPLimiter // IP-based rate limiter for DoS protection
}

// IPLimiter interface for IP-based rate limiting.
type IPLimiter interface {
	AllowRequest(ipAddress string) bool
}

const msgTurnstileVerificationFailed = "Turnstile verification failed"

var errTurnstileVerification = errors.New("turnstile verification failed")

// New creates a basic Handler for password change operations without password reset services.
func New(opts *options.Opts) (*Handler, error) {
	ldapClient, err := ldap.New(opts.LDAP, opts.ReadonlyUser, opts.ReadonlyPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize LDAP connection: %w", err)
	}

	return &Handler{
		ldap: ldapClient,
		opts: opts,
	}, nil
}

// SetIPLimiter sets the IP limiter for the handler (used for change-password rate limiting).
func (h *Handler) SetIPLimiter(ipLimiter IPLimiter) {
	h.ipLimiter = ipLimiter
}

// NewWithServices creates a handler with password reset services.
func NewWithServices(
	opts *options.Opts,
	tokenStore TokenStore,
	emailService EmailService,
	rateLimiter RateLimiter,
	ipLimiter IPLimiter,
) (*Handler, error) {
	ldapClient, err := ldap.New(opts.LDAP, opts.ReadonlyUser, opts.ReadonlyPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize LDAP connection: %w", err)
	}

	// Reset LDAP client will be lazy-initialized on first password reset request
	// This prevents startup failures if reset account credentials are invalid
	// Falls back to readonly user if not set (backward compatible)
	var resetLDAP LDAPClient
	if opts.ResetUser == "" || opts.ResetPassword == "" {
		// Use readonly client immediately if no dedicated reset account configured
		resetLDAP = ldapClient
	}
	// If reset credentials are configured, resetLDAP will be nil and initialized on first use

	return &Handler{
		ldap:         ldapClient,
		resetLDAP:    resetLDAP,
		opts:         opts,
		tokenStore:   tokenStore,
		emailService: emailService,
		rateLimiter:  rateLimiter,
		ipLimiter:    ipLimiter,
	}, nil
}

// Handle processes incoming JSON-RPC 2.0 requests and routes them to appropriate handlers.
//
//nolint:stylecheck // ST1016: c matches fiber conventions, other methods use h
func (h *Handler) Handle(c fiber.Ctx) error {
	var body Request
	if err := c.Bind().Body(&body); err != nil {
		return fmt.Errorf("failed to parse request body: %w", err)
	}

	dispatch, known := methodHandlers[body.Method]
	if !known {
		return sendErrorResponse(c, http.StatusBadRequest, "method not found")
	}

	// The cross-cutting checks run here, in the order methodPolicies gives for
	// this method, so that every method is guarded by a list rather than by
	// whatever its own body remembers to do.
	stop, err := h.runGuards(c, body.Method, guardInput{
		params:         body.Params,
		clientIP:       extractClientIP(c),
		turnstileToken: body.TurnstileToken,
	})
	if stop {
		return err
	}

	return dispatch(h, c, body.Params)
}

// methodHandlers is the dispatch table. It is a map rather than a switch so
// that a test can compare its keys with methodPolicies: a method reachable
// here without a policy there would run with no guards at all.
var methodHandlers = map[string]func(h *Handler, c fiber.Ctx, params []string) error{
	"change-password":        (*Handler).handleChangePassword,
	"request-password-reset": (*Handler).handleRequestPasswordReset,
	"reset-password":         (*Handler).handleResetPassword,
}

// handleChangePassword processes change-password requests. The cross-cutting
// checks already ran; see methodPolicies.
func (h *Handler) handleChangePassword(c fiber.Ctx, params []string) error {
	data, err := h.changePassword(params)

	return respond(c, data, err)
}

// handleRequestPasswordReset processes request-password-reset requests.
func (h *Handler) handleRequestPasswordReset(c fiber.Ctx, params []string) error {
	data, err := h.requestPasswordReset(params)

	return respond(c, data, err)
}

// handleResetPassword processes reset-password requests.
func (h *Handler) handleResetPassword(c fiber.Ctx, params []string) error {
	data, err := h.resetPassword(params)

	return respond(c, data, err)
}

// respond turns a method result into the JSON-RPC response.
func respond(c fiber.Ctx, data []string, err error) error {
	if err != nil {
		return sendErrorResponse(c, http.StatusInternalServerError, err.Error())
	}

	return sendSuccessResponse(c, data)
}

// sendSuccessResponse sends a successful JSON-RPC response.
func sendSuccessResponse(c fiber.Ctx, data []string) error {
	if jsonErr := c.JSON(Response{
		Success: true,
		Data:    data,
	}); jsonErr != nil {
		return fmt.Errorf("failed to send success response: %w", jsonErr)
	}
	return nil
}

// sendErrorResponse sends an error JSON-RPC response.
func sendErrorResponse(c fiber.Ctx, statusCode int, message string) error {
	if jsonErr := c.Status(statusCode).JSON(Response{
		Success: false,
		Data:    []string{message},
	}); jsonErr != nil {
		return fmt.Errorf("failed to send error response: %w", jsonErr)
	}
	return nil
}

// verifyTurnstile verifies the provided Turnstile token when Cloudflare
// Turnstile protection is enabled.
func (h *Handler) verifyTurnstile(token, clientIP string) error {
	if !h.opts.CfTurnstileEnabled {
		return nil
	}

	if err := turnstile.Verify(
		h.opts.CfTurnstileSecret,
		token,
		clientIP,
		h.opts.CfTurnstileTimeout,
	); err != nil {
		slog.Warn("turnstile_verification_failed", "ip", clientIP, "error", err)
		return errTurnstileVerification
	}

	return nil
}
