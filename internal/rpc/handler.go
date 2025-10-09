package rpc

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/options"
	ldap "github.com/netresearch/simple-ldap-go"
)

type Func = func(params []string) ([]string, error)

// LDAPClient interface for LDAP operations (enables testing)
type LDAPClient interface {
	FindUserByMail(mail string) (*ldap.User, error)
	ChangePasswordForSAMAccountName(sAMAccountName, oldPassword, newPassword string) error
	ResetPasswordForSAMAccountName(sAMAccountName, newPassword string) error
}

type Handler struct {
	ldap         LDAPClient
	resetLDAP    LDAPClient // Optional dedicated client for password reset operations (lazy-initialized)
	opts         *options.Opts
	tokenStore   TokenStore
	emailService EmailService
	rateLimiter  RateLimiter
	ipLimiter    IPLimiter // IP-based rate limiter for DoS protection
}

// IPLimiter interface for IP-based rate limiting
type IPLimiter interface {
	AllowRequest(ipAddress string) bool
}

func New(opts *options.Opts) (*Handler, error) {
	ldap, err := ldap.New(opts.LDAP, opts.ReadonlyUser, opts.ReadonlyPassword)
	if err != nil {
		return nil, err
	}

	return &Handler{
		ldap: ldap,
		opts: opts,
	}, nil
}

// SetIPLimiter sets the IP limiter for the handler (used for change-password rate limiting)
func (h *Handler) SetIPLimiter(ipLimiter IPLimiter) {
	h.ipLimiter = ipLimiter
}

// NewWithServices creates a handler with password reset services.
func NewWithServices(opts *options.Opts, tokenStore TokenStore, emailService EmailService, rateLimiter RateLimiter, ipLimiter IPLimiter) (*Handler, error) {
	ldapClient, err := ldap.New(opts.LDAP, opts.ReadonlyUser, opts.ReadonlyPassword)
	if err != nil {
		return nil, err
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

func (h *Handler) Handle(c *fiber.Ctx) error {
	var body JSONRPC
	if err := c.BodyParser(&body); err != nil {
		return err
	}

	wrapRPC := func(fn Func) error {
		data, err := fn(body.Params)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(JSONRPCResponse{
				Success: false,
				Data:    []string{err.Error()},
			})
		}

		return c.JSON(JSONRPCResponse{
			Success: true,
			Data:    data,
		})
	}

	// Extract client IP for rate limiting
	clientIP := extractClientIP(c)

	switch body.Method {
	case "change-password":
		// Use dedicated method that includes IP-based rate limiting
		data, err := h.changePasswordWithIP(body.Params, clientIP)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(JSONRPCResponse{
				Success: false,
				Data:    []string{err.Error()},
			})
		}
		return c.JSON(JSONRPCResponse{
			Success: true,
			Data:    data,
		})

	case "request-password-reset":
		// Check if password reset is enabled
		if h.tokenStore == nil {
			return c.Status(http.StatusBadRequest).JSON(JSONRPCResponse{
				Success: false,
				Data:    []string{"password reset feature not enabled"},
			})
		}
		// Use dedicated method that includes IP-based rate limiting
		data, err := h.requestPasswordResetWithIP(body.Params, clientIP)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(JSONRPCResponse{
				Success: false,
				Data:    []string{err.Error()},
			})
		}
		return c.JSON(JSONRPCResponse{
			Success: true,
			Data:    data,
		})

	case "reset-password":
		// Check if password reset is enabled
		if h.tokenStore == nil {
			return c.Status(http.StatusBadRequest).JSON(JSONRPCResponse{
				Success: false,
				Data:    []string{"password reset feature not enabled"},
			})
		}
		return wrapRPC(h.resetPassword)

	default:
		return c.Status(http.StatusBadRequest).JSON(JSONRPCResponse{
			Success: false,
			Data:    []string{"method not found"},
		})
	}
}
