package rpchandler

import (
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v3"
	ldap "github.com/netresearch/simple-ldap-go"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/options"
)

// Func is a type alias for RPC handler functions that process string parameters and return results or errors.
type Func = func(params []string) ([]string, error)

// LDAPClient interface for LDAP operations (enables testing).
type LDAPClient interface {
	FindUserByMail(mail string) (*ldap.User, error)
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

	// Extract client IP for rate limiting
	clientIP := extractClientIP(c)

	switch body.Method {
	case "change-password":
		return h.handleChangePassword(c, body.Params, clientIP)
	case "request-password-reset":
		return h.handleRequestPasswordReset(c, body.Params, clientIP)
	case "reset-password":
		return h.handleResetPassword(c, body.Params)
	default:
		return sendErrorResponse(c, http.StatusBadRequest, "method not found")
	}
}

// handleChangePassword processes change-password requests with IP-based rate limiting.
func (h *Handler) handleChangePassword(c fiber.Ctx, params []string, clientIP string) error {
	data, err := h.changePasswordWithIP(params, clientIP)
	if err != nil {
		return sendErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return sendSuccessResponse(c, data)
}

// handleRequestPasswordReset processes request-password-reset requests with IP-based rate limiting.
func (h *Handler) handleRequestPasswordReset(c fiber.Ctx, params []string, clientIP string) error {
	if h.tokenStore == nil {
		return sendErrorResponse(c, http.StatusBadRequest, "password reset feature not enabled")
	}
	data, err := h.requestPasswordResetWithIP(params, clientIP)
	if err != nil {
		return sendErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return sendSuccessResponse(c, data)
}

// handleResetPassword processes reset-password requests.
func (h *Handler) handleResetPassword(c fiber.Ctx, params []string) error {
	if h.tokenStore == nil {
		return sendErrorResponse(c, http.StatusBadRequest, "password reset feature not enabled")
	}
	data, err := h.resetPassword(params)
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
