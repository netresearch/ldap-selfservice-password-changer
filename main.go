package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/helmet/v2"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/email"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/options"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/ratelimit"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/resettoken"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/rpc"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/web/static"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/web/templates"
)

func main() {
	opts := options.Parse()

	var rpcHandler *rpc.Handler
	var err error

	// Initialize password reset services if enabled
	if opts.PasswordResetEnabled {
		slog.Info("password reset feature enabled")

		// Initialize token store
		tokenStore := resettoken.NewStore()
		tokenStore.StartCleanup(5 * time.Minute)

		// Initialize email service
		emailConfig := email.Config{
			SMTPHost:     opts.SMTPHost,
			SMTPPort:     int(opts.SMTPPort),
			SMTPUsername: opts.SMTPUsername,
			SMTPPassword: opts.SMTPPassword,
			FromAddress:  opts.SMTPFromAddress,
			BaseURL:      opts.AppBaseURL,
		}
		emailService := email.NewService(emailConfig)

		// Initialize rate limiter
		rateLimiter := ratelimit.NewLimiter(
			int(opts.ResetRateLimitRequests),
			time.Duration(opts.ResetRateLimitWindowMinutes)*time.Minute,
		)

		// Create handler with password reset services
		rpcHandler, err = rpc.NewWithServices(opts, tokenStore, emailService, rateLimiter)
		if err != nil {
			slog.Error("initialization failed", "error", err)
			os.Exit(1)
		}
	} else {
		// Create handler without password reset services
		rpcHandler, err = rpc.New(opts)
		if err != nil {
			slog.Error("initialization failed", "error", err)
			os.Exit(1)
		}
	}

	index, err := templates.RenderIndex(opts)
	if err != nil {
		slog.Error("failed to render page", "error", err)
		os.Exit(1)
	}

	app := fiber.New(fiber.Config{
		AppName:   "netresearch/ldap-selfservice-password-changer",
		BodyLimit: 4 * 1024,
	})

	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))

	// Security headers middleware
	app.Use(helmet.New(helmet.Config{
		ContentSecurityPolicy: "default-src 'self'; " +
			"script-src 'self'; " +
			"style-src 'self' 'unsafe-inline'; " + // Tailwind requires inline styles
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

	app.Use("/static", filesystem.New(filesystem.Config{
		Root:   http.FS(static.Static),
		MaxAge: 24 * 60 * 60,
	}))

	app.Get("/", func(c *fiber.Ctx) error {
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

		app.Get("/forgot-password", func(c *fiber.Ctx) error {
			c.Set("Content-Type", fiber.MIMETextHTMLCharsetUTF8)
			return c.Send(forgotPasswordPage)
		})

		app.Get("/reset-password", func(c *fiber.Ctx) error {
			c.Set("Content-Type", fiber.MIMETextHTMLCharsetUTF8)
			return c.Send(resetPasswordPage)
		})
	}

	app.Post("/api/rpc", rpcHandler.Handle)

	slog.Info("starting server", "port", opts.Port)
	if err := app.Listen(":" + opts.Port); err != nil {
		slog.Error("failed to start web server", "error", err)
	}
}
