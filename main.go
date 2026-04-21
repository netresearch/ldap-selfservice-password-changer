// GopherPass LDAP Self-Service Password Changer provides a web interface
// for users to change and reset their LDAP passwords.
package main

import (
	"context"
	"fmt"
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
	"github.com/netresearch/ldap-selfservice-password-changer/internal/rpchandler"
	webstatic "github.com/netresearch/ldap-selfservice-password-changer/internal/web/static"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/web/templates"
)

const (
	healthCheckTimeout  = 3 * time.Second
	healthCheckEndpoint = "http://localhost:3000/health/live"

	defaultBodyLimit            = 4 * 1024
	defaultReadTimeout          = 10 * time.Second
	defaultWriteTimeout         = 10 * time.Second
	defaultIdleTimeout          = 120 * time.Second
	cleanupIntervalMinutes      = 5 * time.Minute
	staticCacheMaxAgeSeconds    = 24 * 60 * 60
	contentSecurityPolicyHeader = "default-src 'self'; " +
		"script-src 'self'; " +
		"style-src 'self' 'unsafe-inline'; " + // unsafe-inline needed for browser password managers (Bitwarden etc.)
		"img-src 'self' data:; " +
		"font-src 'self'; " +
		"connect-src 'self'; " +
		"frame-ancestors 'none'; " +
		"base-uri 'self'; " +
		"form-action 'self'"
)

// isHealthCheckInvocation returns true when the given args list asks for a
// standalone health-check run (used by the Docker HEALTHCHECK).
func isHealthCheckInvocation(args []string) bool {
	return len(args) == 2 && args[1] == "--health-check"
}

// isLDAPEncrypted reports whether the configured LDAP server URL uses ldaps://.
func isLDAPEncrypted(server string) bool {
	return strings.HasPrefix(server, "ldaps://")
}

// buildEmailConfig constructs an email.Config from application options.
// Extracted for testability.
func buildEmailConfig(opts *options.Opts) email.Config {
	// Safe conversion: SMTPPort is uint, typically 25/587/465 (well within int range)
	smtpPort := int(opts.SMTPPort) //#nosec G115 -- SMTPPort is 0-65535, safe for int
	return email.Config{
		SMTPHost:     opts.SMTPHost,
		SMTPPort:     smtpPort,
		SMTPUsername: opts.SMTPUsername,
		SMTPPassword: opts.SMTPPassword,
		FromAddress:  opts.SMTPFromAddress,
		BaseURL:      opts.AppBaseURL,
	}
}

// resetRateLimitSettings extracts rate limiting settings safely from options.
// Returns request count and the window duration.
func resetRateLimitSettings(opts *options.Opts) (int, time.Duration) {
	// Safe conversion: ResetRateLimitRequests is uint, typically small value (3-10)
	resetRequests := int(opts.ResetRateLimitRequests) //#nosec G115 -- small config value, safe for int
	// Safe conversion: ResetRateLimitWindowMinutes is uint, typically 60-120
	//#nosec G115 -- small config value, safe for int64
	resetWindowDuration := time.Duration(opts.ResetRateLimitWindowMinutes) * time.Minute
	return resetRequests, resetWindowDuration
}

// newHandlerWithResetServices wires up all reset-related services and returns
// a fully configured Handler. Any LDAP connection error is propagated.
//
// Background cleanup goroutines are started only AFTER handler initialisation
// succeeds. This guarantees that a failed handler init leaks nothing — neither
// the token store's cleanup goroutine nor the IP limiter's.
func newHandlerWithResetServices(opts *options.Opts) (*rpchandler.Handler, error) {
	// Initialize token store (cleanup started below, only on success).
	tokenStore := resettoken.NewStore()

	// Initialize email service
	emailConfig := buildEmailConfig(opts)
	emailService := email.NewService(&emailConfig)

	// Initialize email-based rate limiter (per-user protection)
	resetRequests, resetWindowDuration := resetRateLimitSettings(opts)
	rateLimiter := ratelimit.NewLimiter(resetRequests, resetWindowDuration)

	// Initialize IP-based rate limiter (cleanup started below, only on success).
	ipLimiter := ratelimit.NewIPLimiter()

	h, err := rpchandler.NewWithServices(opts, tokenStore, emailService, rateLimiter, ipLimiter)
	if err != nil {
		// Handler init failed — do NOT start background goroutines.
		return nil, fmt.Errorf("build handler with reset services: %w", err)
	}

	// Success: start background cleanup goroutines now that ownership of the
	// handler (and its dependencies) is being transferred to the caller.
	tokenStore.StartCleanup(cleanupIntervalMinutes)
	ipLimiter.StartCleanup(cleanupIntervalMinutes)

	return h, nil
}

// newHandlerWithoutResetServices creates a Handler without password reset
// services but still attaches an IP rate limiter for change-password.
//
// The IP limiter's cleanup goroutine is started only AFTER rpchandler.New
// succeeds so that a failed handler init does not leak a goroutine.
func newHandlerWithoutResetServices(opts *options.Opts) (*rpchandler.Handler, error) {
	baseHandler, err := rpchandler.New(opts)
	if err != nil {
		return nil, fmt.Errorf("build base handler: %w", err)
	}

	// Handler init succeeded — now safe to create the IP limiter and start
	// its background cleanup goroutine.
	ipLimiter := ratelimit.NewIPLimiter()
	ipLimiter.StartCleanup(cleanupIntervalMinutes)
	baseHandler.SetIPLimiter(ipLimiter)
	return baseHandler, nil
}

// buildRPCHandler selects the appropriate handler factory based on whether
// the password reset feature is enabled.
func buildRPCHandler(opts *options.Opts) (*rpchandler.Handler, error) {
	if opts.PasswordResetEnabled {
		slog.Info("password reset feature enabled")
		return newHandlerWithResetServices(opts)
	}
	return newHandlerWithoutResetServices(opts)
}

// logLDAPSecurityStatus logs information about the LDAP connection
// encryption status and emits a warning for cleartext configurations.
func logLDAPSecurityStatus(opts *options.Opts) {
	encrypted := isLDAPEncrypted(opts.LDAP.Server)
	slog.Info("ldap_connection_configuration",
		"server", opts.LDAP.Server,
		"encrypted", encrypted,
	)
	if !encrypted {
		slog.Warn("ldap_connection_not_encrypted",
			"server", opts.LDAP.Server,
			"risk", "passwords transmitted in cleartext over network",
			"recommendation", "use ldaps:// for production deployments to encrypt credentials in transit",
		)
	}
}

// buildApp builds a Fiber app with middleware preconfigured for this service.
// Routes are not registered here; use registerCorePages (and optionally
// registerResetPages when the reset feature is enabled) for that.
func buildApp() *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      "netresearch/ldap-selfservice-password-changer",
		BodyLimit:    defaultBodyLimit,
		ReadTimeout:  defaultReadTimeout,  // Maximum time to read request (prevents slowloris)
		WriteTimeout: defaultWriteTimeout, // Maximum time to write response
		IdleTimeout:  defaultIdleTimeout,  // Maximum time to keep idle connections alive
	})

	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))

	// Security headers middleware
	app.Use(helmet.New(helmet.Config{
		ContentSecurityPolicy: contentSecurityPolicyHeader,
		XFrameOptions:         "DENY",
		ContentTypeNosniff:    "nosniff",
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		PermissionPolicy:      "geolocation=(), microphone=(), camera=()",
	}))

	app.Use("/static", static.New("", static.Config{
		FS:     webstatic.Static,
		MaxAge: staticCacheMaxAgeSeconds,
	}))

	return app
}

// rpcHandleFunc is the minimal surface of *rpchandler.Handler used by
// registerCorePages; accepting a function makes the function testable without
// a fully wired Handler (which requires an LDAP connection).
type rpcHandleFunc = fiber.Handler

// registerCorePages registers the main routes that are always available: the
// index page, the RPC endpoint and the health check.
func registerCorePages(app *fiber.App, index []byte, rpcHandle rpcHandleFunc) {
	app.Get("/", func(c fiber.Ctx) error {
		c.Set("Content-Type", fiber.MIMETextHTMLCharsetUTF8)
		return c.Send(index)
	})

	app.Post("/api/rpc", rpcHandle)

	app.Get("/health/live", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "alive"})
	})
}

// registerResetPages registers the password reset pages when the feature is
// enabled. Returns any template rendering errors.
func registerResetPages(app *fiber.App, opts *options.Opts) error {
	forgotPasswordPage, err := templates.RenderForgotPassword()
	if err != nil {
		return fmt.Errorf("render forgot-password page: %w", err)
	}
	resetPasswordPage, err := templates.RenderResetPassword(opts)
	if err != nil {
		return fmt.Errorf("render reset-password page: %w", err)
	}

	app.Get("/forgot-password", func(c fiber.Ctx) error {
		c.Set("Content-Type", fiber.MIMETextHTMLCharsetUTF8)
		return c.Send(forgotPasswordPage)
	})

	app.Get("/reset-password", func(c fiber.Ctx) error {
		c.Set("Content-Type", fiber.MIMETextHTMLCharsetUTF8)
		return c.Send(resetPasswordPage)
	})

	return nil
}

// buildServer orchestrates all the pieces needed to produce a ready-to-listen
// Fiber app, returning an error instead of exiting the process.
func buildServer(opts *options.Opts) (*fiber.App, error) {
	logLDAPSecurityStatus(opts)

	rpcHandler, err := buildRPCHandler(opts)
	if err != nil {
		return nil, err
	}

	index, err := templates.RenderIndex(opts)
	if err != nil {
		return nil, fmt.Errorf("render index page: %w", err)
	}

	app := buildApp()
	registerCorePages(app, index, rpcHandler.Handle)

	if opts.PasswordResetEnabled {
		if err := registerResetPages(app, opts); err != nil {
			return nil, fmt.Errorf("register reset pages: %w", err)
		}
	}

	return app, nil
}

// Build-injected version metadata. Populated by release.yml's ldflags:
//
//	-X main.version=<tag>
//	-X main.build=<commit-sha>
//	-X main.buildTime=<commit-timestamp>
//
// `version` defaults to "dev" so local `go run .` / `go build` logs
// something meaningful; `build` and `buildTime` default to empty and
// are kept as declared receivers so the fleet-uniform ldflag string
// always has a target (no risk of unknown-symbol linker strictness).
// Logged on startup so operators can confirm which artifact is live.
var (
	version   = "dev"
	build     = ""
	buildTime = ""
)

// healthCheckFunc is the indirection used by run() to invoke the health check.
// Tests override this to assert that the --health-check branch is actually
// taken and to control the outcome deterministically. Production code uses
// the default value, which calls runHealthCheck.
var healthCheckFunc = runHealthCheck

// run is the testable entry point. It returns an exit code so main() only
// needs to call os.Exit. run never calls os.Exit itself.
//
// args is the full argv slice, including the program name at index 0, so that
// isHealthCheckInvocation can inspect args[1]. The remainder (args[1:]) is
// threaded into options.ParseArgs so that run() fully honors the args
// parameter and never falls back to os.Args.
func run(args []string) int {
	if isHealthCheckInvocation(args) {
		return healthCheckFunc()
	}

	// Forward args[1:] so tests can drive run() without os.Args interference.
	var flagArgs []string
	if len(args) > 1 {
		flagArgs = args[1:]
	}
	opts, err := options.ParseArgs(flagArgs)
	if err != nil {
		slog.Error("configuration error", "error", err)
		return 1
	}

	app, err := buildServer(opts)
	if err != nil {
		slog.Error("initialization failed", "error", err)
		return 1
	}

	slog.Info("starting server", "port", opts.Port, "version", version, "build", build, "buildTime", buildTime)
	if err := app.Listen(":" + opts.Port); err != nil {
		slog.Error("failed to start web server", "error", err)
		return 1
	}
	return 0
}

func main() {
	os.Exit(run(os.Args))
}

// runHealthCheck performs an HTTP health check against the running application
// using the default Docker HEALTHCHECK endpoint and timeout.
// Returns 0 if healthy (HTTP 200), 1 otherwise.
// Used by Docker HEALTHCHECK to verify the application is running correctly.
func runHealthCheck() int {
	return runHealthCheckAt(healthCheckEndpoint, healthCheckTimeout)
}

// runHealthCheckAt performs an HTTP health check against an arbitrary endpoint
// with a caller-supplied timeout. Returns 0 on HTTP 200, 1 otherwise.
// Extracted so tests can exercise the real code path against an httptest server.
func runHealthCheckAt(endpoint string, timeout time.Duration) int {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return 1
	}

	client := &http.Client{}
	// G704: endpoint is either the compile-time constant healthCheckEndpoint
	// (via runHealthCheck) or test-controlled (via runHealthCheckAt), never
	// user-supplied.
	resp, err := client.Do(req) //nolint:gosec,nolintlint
	if err != nil {
		return 1
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusOK {
		return 0
	}

	return 1
}
