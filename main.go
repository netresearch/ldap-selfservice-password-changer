// GopherPass LDAP Self-Service Password Changer provides a web interface
// for users to change and reset their LDAP passwords.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/static"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/email"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/options"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/ratelimit"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/resettoken"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/rpc"
	webstatic "github.com/netresearch/ldap-selfservice-password-changer/internal/web/static"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/web/templates"
)

const (
	healthCheckTimeout  = 3 * time.Second
	healthCheckEndpoint = "http://localhost:3000/health/live"
)

func main() {
	// Handle --health-check flag early, before any other initialization
	if len(os.Args) == 2 && os.Args[1] == "--health-check" {
		os.Exit(runHealthCheck())
	}

	opts := options.Parse()

	// Log LDAP connection security status
	isEncrypted := strings.HasPrefix(opts.LDAP.Server, "ldaps://")
	slog.Info("ldap_connection_configuration",
		"server", opts.LDAP.Server,
		"encrypted", isEncrypted,
	)

	// Warn if using unencrypted LDAP connection
	if !isEncrypted {
		slog.Warn("ldap_connection_not_encrypted",
			"server", opts.LDAP.Server,
			"risk", "passwords transmitted in cleartext over network",
			"recommendation", "use ldaps:// for production deployments to encrypt credentials in transit",
		)
	}

	var rpcHandler *rpc.Handler
	var err error

	// Initialize password reset services if enabled
	if opts.PasswordResetEnabled {
		slog.Info("password reset feature enabled")

		// Initialize token store
		tokenStore := resettoken.NewStore()
		tokenStore.StartCleanup(5 * time.Minute)

		// Initialize email service
		// Safe conversion: SMTPPort is uint, typically 25/587/465 (well within int range)
		smtpPort := int(opts.SMTPPort) //nolint:gosec // G115: SMTPPort is 0-65535, safe for int
		emailConfig := email.Config{
			SMTPHost:     opts.SMTPHost,
			SMTPPort:     smtpPort,
			SMTPUsername: opts.SMTPUsername,
			SMTPPassword: opts.SMTPPassword,
			FromAddress:  opts.SMTPFromAddress,
			BaseURL:      opts.AppBaseURL,
		}
		emailService := email.NewService(&emailConfig)

		// Initialize email-based rate limiter (per-user protection)
		// Safe conversion: ResetRateLimitRequests is uint, typically small value (3-10)
		resetRequests := int(opts.ResetRateLimitRequests) //nolint:gosec // G115: small config value, safe for int
		// Safe conversion: ResetRateLimitWindowMinutes is uint, typically 60-120
		//nolint:gosec // G115: small config value, safe for int64
		resetWindowDuration := time.Duration(opts.ResetRateLimitWindowMinutes) * time.Minute
		rateLimiter := ratelimit.NewLimiter(resetRequests, resetWindowDuration)

		// Initialize IP-based rate limiter (DoS protection)
		// Default: 10 requests per IP per 60 minutes, max 1000 IPs tracked
		ipLimiter := ratelimit.NewIPLimiter()
		ipLimiter.StartCleanup(5 * time.Minute)

		// Create handler with password reset services
		rpcHandler, err = rpc.NewWithServices(opts, tokenStore, emailService, rateLimiter, ipLimiter)
		if err != nil {
			slog.Error("initialization failed", "error", err)
			os.Exit(1)
		}
	} else {
		// Create handler without password reset services
		// Still initialize IP limiter for change-password rate limiting
		ipLimiter := ratelimit.NewIPLimiter()
		ipLimiter.StartCleanup(5 * time.Minute)

		baseHandler, err := rpc.New(opts)
		if err != nil {
			slog.Error("initialization failed", "error", err)
			os.Exit(1)
		}
		// Add IP limiter to base handler
		baseHandler.SetIPLimiter(ipLimiter)
		rpcHandler = baseHandler
	}

	index, err := templates.RenderIndex(opts)
	if err != nil {
		slog.Error("failed to render page", "error", err)
		os.Exit(1)
	}

	app := fiber.New(fiber.Config{
		AppName:      "netresearch/ldap-selfservice-password-changer",
		BodyLimit:    4 * 1024,
		ReadTimeout:  10 * time.Second,  // Maximum time to read request (prevents slowloris)
		WriteTimeout: 10 * time.Second,  // Maximum time to write response
		IdleTimeout:  120 * time.Second, // Maximum time to keep idle connections alive
	})

	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))

	// Security headers middleware
	app.Use(helmet.New(helmet.Config{
		ContentSecurityPolicy: "default-src 'self'; " +
			"script-src 'self'; " +
			"style-src 'self'; " + // Tailwind styles served as external CSS
			"img-src 'self' data:; " +
			"font-src 'self'; " +
			"connect-src 'self'; " +
			"frame-ancestors 'none'; " +
			"base-uri 'self'; " +
			"form-action 'self'",
		XFrameOptions:      "DENY",
		ContentTypeNosniff: "nosniff",
		ReferrerPolicy:     "strict-origin-when-cross-origin",
		PermissionPolicy:   "geolocation=(), microphone=(), camera=()",
	}))

	app.Use("/static", static.New("", static.Config{
		FS:     webstatic.Static,
		MaxAge: 24 * 60 * 60,
	}))

	app.Get("/", func(c fiber.Ctx) error {
		c.Set("Content-Type", fiber.MIMETextHTMLCharsetUTF8)
		return c.Send(index)
	})

	// Password reset pages (only if feature is enabled)
	if opts.PasswordResetEnabled {
		forgotPasswordPage, err := templates.RenderForgotPassword()
		if err != nil {
			slog.Error("failed to render forgot password page", "error", err)
			os.Exit(1)
		}

		resetPasswordPage, err := templates.RenderResetPassword(opts)
		if err != nil {
			slog.Error("failed to render reset password page", "error", err)
			os.Exit(1)
		}

		app.Get("/forgot-password", func(c fiber.Ctx) error {
			c.Set("Content-Type", fiber.MIMETextHTMLCharsetUTF8)
			return c.Send(forgotPasswordPage)
		})

		app.Get("/reset-password", func(c fiber.Ctx) error {
			c.Set("Content-Type", fiber.MIMETextHTMLCharsetUTF8)
			return c.Send(resetPasswordPage)
		})
	}

	app.Post("/api/rpc", rpcHandler.Handle)

	// Health check endpoint for Docker HEALTHCHECK
	app.Get("/health/live", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "alive"})
	})

	slog.Info("starting server", "port", opts.Port)
	if err := app.Listen(":" + opts.Port); err != nil {
		slog.Error("failed to start web server", "error", err)
	}
}

// runHealthCheck performs an HTTP health check against the running application.
// Returns 0 if healthy (HTTP 200), 1 otherwise.
// Used by Docker HEALTHCHECK to verify the application is running correctly.
func runHealthCheck() int {
	ctx, cancel := context.WithTimeout(context.Background(), healthCheckTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthCheckEndpoint, http.NoBody)
	if err != nil {
		return 1
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 1
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusOK {
		return 0
	}

	return 1
}
