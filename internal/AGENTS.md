# Go Backend Services

<!-- Managed by agent: keep sections & order; edit content, not structure. Last updated: 2026-07-22 -->

**Scope**: Go backend packages in `internal/` directory

**See also**: [../AGENTS.md](../AGENTS.md) for global standards, [web/AGENTS.md](web/AGENTS.md) for frontend

## Overview

Backend services for LDAP selfservice password change/reset functionality. Organized as internal Go packages:

- **email/**: SMTP email service for password reset tokens
- **options/**: Configuration management from environment variables
- **ratelimit/**: sliding-window limiting. `Limiter` is generic over a string key; `IPLimiter` wraps it with a hardcoded 10 req/60 min/1000-IP configuration
- **resettoken/**: Cryptographic token generation and validation
- **rpchandler/**: JSON-RPC 2.0 API handlers (password change/reset) — renamed from `rpc/` to avoid Go stdlib `net/rpc` conflict
- **validators/**: Password policy validation logic
- **web/**: HTTP server setup, static assets, routing (see [web/AGENTS.md](web/AGENTS.md))

## Setup/Environment

**Environment variables** (configure in `.env.local`; names must match `options/app.go`):

```bash
# LDAP connection (the four below are the only *required* variables)
LDAP_SERVER=ldaps://ldap.example.com:636
LDAP_BASE_DN=dc=example,dc=com
LDAP_READONLY_USER=cn=readonly,dc=example,dc=com
LDAP_READONLY_PASSWORD=secret
LDAP_IS_AD=false   # optional; true for ActiveDirectory (then LDAP_SERVER must be ldaps://)

# Optional dedicated reset account (falls back to the readonly user)
LDAP_RESET_USER=cn=password-reset,dc=example,dc=com
LDAP_RESET_PASSWORD=secret

# Email for password reset
PASSWORD_RESET_ENABLED=true
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USERNAME=noreply@example.com
SMTP_PASSWORD=secret
SMTP_FROM_ADDRESS=noreply@example.com
APP_BASE_URL=https://passwd.example.com
RESET_IDENTIFIER_MODE=email   # optional; email (default), username, or both

# Email templating (optional; unset => built-in defaults)
SMTP_FROM_NAME=ACME IT
EMAIL_REPLY_TO=helpdesk@example.com
EMAIL_TEMPLATE_SUBJECT=[ACME] Reset your password
EMAIL_TEMPLATE_HTML=/config/email/reset.html
EMAIL_TEMPLATE_TEXT=/config/email/reset.txt
# Raw header escape hatch (suffix _ -> -): SMTP_HEADER_OVERRIDE_X_HELPDESK_TOPIC=...

# Rate limiting for the reset flow, per identifier (optional; window is in
# minutes, not a duration string). The separate per-IP limiter is hardcoded —
# no environment variable configures it, and there is no RATE_LIMIT_* prefix.
RESET_RATE_LIMIT_REQUESTS=3
RESET_RATE_LIMIT_WINDOW_MINUTES=60

# Token expiry (optional; minutes, not a duration string)
RESET_TOKEN_EXPIRY_MINUTES=15

# Password policy (optional; defaults shown)
MIN_LENGTH=8
MIN_NUMBERS=1
MIN_SYMBOLS=1
MIN_UPPERCASE=1
MIN_LOWERCASE=1
PASSWORD_CAN_INCLUDE_USERNAME=false

# Branding (optional; unset => stock appearance)
# BRANDING_DIR replaces individual static assets; only the allowlist in
# web/static/overlay.go is accepted and anything else aborts startup.
# Clearing BRANDING_PRODUCT_NAME requires BRANDING_LOGO_ALT, otherwise the
# header is a decorative image alone and the brand reaches sighted users only.
# Setting both is accepted, but the alt text yields to the wordmark.
BRANDING_PRODUCT_NAME=GopherPass
BRANDING_PAGE_TITLE=
BRANDING_LOGO_ALT=
BRANDING_SHOW_ATTRIBUTION=true
BRANDING_DIR=

# HTTP (optional)
PORT=3000
```

**Go toolchain**: Requires Go 1.26+ (specified in `go.mod`)

**Key dependencies**:

- `github.com/gofiber/fiber/v3` - HTTP server
- `github.com/netresearch/simple-ldap-go` - LDAP client
- `github.com/joho/godotenv` - Environment loading

## Build & Tests

```bash
# Development
go run .                      # Start server once (no hot-reload). For hot-reload, run `bun run dev` which invokes `air`.
go build -v ./...             # Compile all packages
go test -v ./...              # Run all tests with verbose output

# Specific package testing
go test ./internal/validators/...    # Test password validators
go test ./internal/ratelimit/...     # Test rate limiter
go test ./internal/resettoken/...    # Test token generation
go test -run TestSpecificFunction    # Run specific test

# Integration tests (build-tagged; skipped unless the services are reachable)
SMTP_HOST=localhost go test -tags=integration -v ./internal/email/...   # needs Mailpit

# Coverage
go test -cover ./...                 # Coverage summary
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out

# Build optimized binary
CGO_ENABLED=0 go build -ldflags="-w -s" -o ldap-passwd
```

**CI validation** (from `.github/workflows/check.yml`):

```bash
go mod download
go build -v ./...
go test -v ./...
```

## Code Style

**Go Standards**:

- Use `go fmt` (automatic via Prettier with go-template plugin)
- Follow [Effective Go](https://go.dev/doc/effective_go)
- Package-level documentation comments required
- Exported functions must have doc comments

**Project Conventions**:

- Internal packages only: No public API outside this project
- Error wrapping with context: `fmt.Errorf("context: %w", err)`
- Use structured logging (consider adding in future)
- Prefer explicit over implicit
- Use interfaces for testability (see `email/service.go`)

**Go 1.26 Idioms** (prefer these over older patterns):

- `wg.Go(func() { ... })` instead of `wg.Add(1); go func() { defer wg.Done(); ... }()`
- `errors.AsType[*T](err)` instead of `var x *T; errors.As(err, &x)`
- `for b.Loop() { ... }` instead of `for range b.N { ... }` in benchmarks
- `any` instead of `interface{}`
- Run `go fix ./...` after Go upgrades to auto-apply modernizations

**Naming**:

- `internal/package/file.go` - implementation
- `internal/package/file_test.go` - tests
- Descriptive variable names (not `x`, `y`, `tmp`)
- No stuttering: `email.Service`, not `email.EmailService`

**Error Handling**:

```go
// ✅ Good: wrap with context
if err != nil {
    return fmt.Errorf("failed to connect LDAP at %s: %w", config.URL, err)
}

// ❌ Bad: lose context
if err != nil {
    return err
}

// ❌ Worse: ignore
conn, _ := ldap.Dial(url)
```

**Testing**:

- Table-driven tests preferred
- Integration tests are build-tagged `integration` and gated on env vars (e.g. `SMTP_HOST`); there is no testcontainers dependency, so they skip silently when the service is absent
- Test files colocated with code: `validators/validate_test.go`
- Descriptive test names: `TestPasswordValidation_RequiresMinimumLength`

## Security

**LDAP Security**:

- Always use LDAPS in production (`ldaps://` URLs)
- Bind credentials in environment, never hardcoded
- Validate user input before LDAP queries (prevent injection)
- Use `simple-ldap-go` helpers to avoid raw LDAP filter construction

**Password Security**:

- Never log passwords (plain or hashed)
- No password storage - passwords go directly to LDAP
- Passwords only in memory during request lifetime
- HTTPS required for transport security

**Token Security**:

- Cryptographic random tokens (see `resettoken/token.go`)
- Configurable expiry via `RESET_TOKEN_EXPIRY_MINUTES` (default 15 minutes)
- Single-use tokens (invalidated after use)
- No token storage in logs or metrics

**Rate Limiting** — two distinct limiters, keep them apart:

- **Per-IP** (`ratelimit.NewIPLimiter`): 10 requests per 60-minute window, at most 1000 tracked IPs. Hardcoded in `ratelimit/ip_limiter.go`; **not** configurable by any environment variable. Applied to both the change and the reset endpoint.
- **Per-identifier, reset only** (`ratelimit.NewLimiter`): `RESET_RATE_LIMIT_REQUESTS` (default 3) per `RESET_RATE_LIMIT_WINDOW_MINUTES` (default 60). Keyed by the typed identifier (`typed:` prefix) and again by the resolved account (`account:` prefix) — never by IP.
- Both are in-memory (consider Redis for multi-instance)
- The client IP fed to the per-IP limiter comes from `rpchandler.extractClientIP`, which trusts `X-Forwarded-For`/`X-Real-IP` from any source — see the note in `rpchandler/` below.

**Input Validation**:

- Strict validation on all user inputs (see `validators/`)
- Reject malformed requests early
- Validate email format, username format, password policies
- No HTML/script injection vectors

## PR/Commit Checklist

**Before committing Go code**:

- [ ] Run `go fmt ./...` (or `bunx prettier --write .`)
- [ ] Run `go vet ./...` (static analysis)
- [ ] Run `go test ./...` (all tests pass)
- [ ] Run `go build` (compilation check)
- [ ] Update package doc comments if API changed
- [ ] Add/update tests for new functionality
- [ ] Check for sensitive data in logs
- [ ] Verify error messages provide useful context

**Testing requirements**:

- New features must have tests
- Bug fixes must have regression tests
- Aim for ≥80% coverage on changed packages
- Integration tests for external dependencies

**Documentation**:

- Update package doc comments (godoc)
- Update [docs/api-reference.md](../docs/api-reference.md) for RPC changes
- Update [docs/development-guide.md](../docs/development-guide.md) for new setup steps
- Update environment variable examples in `.env` and docs

## Good vs Bad Examples

**✅ Good: Type-safe configuration**

Pattern only — this project parses config with `flag` + `godotenv` in `options/app.go`, not with struct tags. The variable names below are the real ones.

```go
type Config struct {
    LDAPServer       string `env:"LDAP_SERVER" validate:"required,url"`
    ReadonlyUser     string `env:"LDAP_READONLY_USER" validate:"required"`
    ReadonlyPassword string `env:"LDAP_READONLY_PASSWORD" validate:"required"`
}

func LoadConfig() (*Config, error) {
    var cfg Config
    if err := env.Parse(&cfg); err != nil {
        return nil, fmt.Errorf("parse config: %w", err)
    }
    return &cfg, nil
}
```

**❌ Bad: Unsafe configuration**

```go
func LoadConfig() *Config {
    return &Config{
        LDAPServer: os.Getenv("LDAP_SERVER"),  // ❌ no validation, may be empty
    }
}
```

**✅ Good: Table-driven tests**

```go
func TestPasswordValidation(t *testing.T) {
    tests := []struct {
        name     string
        password string
        policy   PasswordPolicy
        wantErr  bool
    }{
        {"valid password", "Test123!", PasswordPolicy{MinLength: 8}, false},
        {"too short", "Ab1!", PasswordPolicy{MinLength: 8}, true},
        {"no numbers", "TestTest", PasswordPolicy{RequireNumbers: true}, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidatePassword(tt.password, tt.policy)
            if (err != nil) != tt.wantErr {
                t.Errorf("got error %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

**❌ Bad: Non-descriptive tests**

```go
func TestPassword(t *testing.T) {
    err := ValidatePassword("test")  // ❌ what policy? what's expected?
    if err == nil {
        t.Fail()
    }
}
```

**✅ Good: Interface for testability**

```go
type EmailService interface {
    SendResetToken(ctx context.Context, to, token string) error
}

type SMTPService struct {
    host string
    port int
}

func (s *SMTPService) SendResetToken(ctx context.Context, to, token string) error {
    // real implementation
}

// In tests, use mock implementation
type MockEmailService struct {
    SendFunc func(ctx context.Context, to, token string) error
}
```

**❌ Bad: Hard-to-test concrete dependency**

```go
func ResetPassword(username string) error {
    service := NewSMTPService()  // ❌ hardcoded, can't mock
    return service.SendEmail(...)
}
```

## When Stuck

**Go-specific issues**:

1. **Module issues**: `go mod tidy` to clean dependencies
2. **Import errors**: Check `go.mod` requires correct versions
3. **Test failures**: `go test -v ./... -run FailingTest` for verbose output
4. **LDAP connection**: Verify `LDAP_SERVER` format and network access
5. **Email testing**: integration tests need a reachable Mailpit and `SMTP_HOST` set; without them they skip rather than fail
6. **Rate limit testing**: Tests may fail if system time incorrect

**Debugging**:

```bash
# Verbose test output
go test -v ./internal/package/...

# Run specific test
go test -run TestName ./internal/package/

# Race detector (for concurrency issues)
go test -race ./...

# Build with debug info
go build -gcflags="all=-N -l"
```

**Common pitfalls**:

- **Nil pointer dereference**: Check error returns before using values
- **Context cancellation**: Always respect `context.Context` in long operations
- **Resource leaks**: Defer `Close()` calls immediately after acquiring resources
- **Goroutine leaks**: Ensure all goroutines can exit
- **Time zones**: Use `time.UTC` for consistency

## Package-Specific Notes

### email/

- Integration tests are behind the `integration` build tag and talk to Mailpit over SMTP + its HTTP API; nothing spins a container up for you (see `compose.yml`, `dev` profile)
- Mock `EmailService` interface for unit tests in other packages
- `NewService` returns `(*Service, error)` and validates templates at startup (parse + dry-run); template fields are `ResetLink`, `Token`, `BaseURL`, `Recipient`, `ExpiryMinutes`
- **To/Cc/Bcc overrides are display-only.** `sendEmail` passes a fixed `[]string{to}` to `smtp.SendMail`, so the SMTP envelope recipient is always the reset requester — `SMTP_HEADER_OVERRIDE_BCC` does **not** add a delivery target. `buildMIMEMessage` still writes it as a real, visible `Bcc:` header line in the mail the requester receives, so an address put there gains no delivery _and_ is disclosed to the user — never configure one that must stay hidden.
- **Header override names are canonicalized.** `applyHeaderOverrides` keys fields by `textproto.CanonicalMIMEHeaderKey`, so `SMTP_HEADER_OVERRIDE_X_HELPDESK_TOPIC` goes on the wire as `X-Helpdesk-Topic`, not `X-HelpDesk-Topic`. Values are rejected (fail-fast in `options`, hard error in `buildMIMEMessage`) if they contain CR, LF, NUL, any other C0 control or DEL; HTAB is allowed.
- **A cross-domain `From`-header override** creates a `From` vs envelope `MAIL FROM` mismatch that can break SPF/DKIM/DMARC alignment and hurt deliverability.

### options/

- Configuration loaded from environment via `godotenv`
- Validation happens at startup (fail-fast)
- See `.env.local.example` for required variables
- `branding.go` derives the UI branding through `NewBranding`, the single
  constructor. `buildBranding` only adapts it to the collecting-error style;
  `DefaultBranding()` is what an unconfigured deployment gets and what tests
  should build from. `templates.brandingFor` falls back to it when an `Opts`
  was assembled by hand, since a zero `Branding` would render an empty title.
- **`LogoAltText()`, not `LogoAlt`, belongs in templates.** The alt text is
  suppressed while a wordmark is shown so screen readers do not announce the
  brand twice; reading the raw field re-introduces the duplicate.

### web/static/

- `overlay.go` layers `BRANDING_DIR` over the embedded assets. Only the
  allowlist in that file is looked up, and the lookup goes through `os.Root`,
  so neither a traversal attempt nor a symlink planted in the directory can
  reach the rest of the filesystem. `styles.css` / `js/` are excluded on
  purpose so a deployment cannot silently break the accessibility guarantees
  the templates depend on.
- **`ValidateOverlay` is operator feedback, not the security boundary.** The
  directory can change after startup, so `Open` re-checks confinement, regular
  ness and the size cap on every request. `O_NONBLOCK` is required there: a
  FIFO would otherwise park a request inside `open(2)` forever.
- Unknown names are a **startup error**, not a silent no-op — but entries
  beginning with a dot are skipped, because Kubernetes materialises ConfigMap
  and Secret volumes as `..data` plus a timestamped directory.
- `logo-dark.webp` is the only allowlisted asset with no embedded counterpart.
  When it is missing, `Open` serves the light logo: the pages are rendered once
  at startup, so removing it later would otherwise leave dead markup behind.
- Adding a Tailwind class to a template needs `bun run css:build` **locally**
  before you can see it work: `styles.css` is gitignored and generated by
  `build:assets` (CI does this via `pre-build-cmd`), so a stale local build
  makes the class silently missing while CI is fine — and a class that Tailwind
  cannot scan statically is missing in both.

### ratelimit/

- In-memory store (map with mutex)
- Consider Redis for multi-instance deployments
- Tests use fixed time.Now for deterministic results
- `Limiter` is key-agnostic (`AllowRequest(identifier string)`); the caller chooses the key. `NewLimiter` caps tracked identifiers at `maxIdentifiers` (10000); `NewLimiterWithCapacity` takes the cap explicitly.
- **`IPLimiter` is not configurable.** `NewIPLimiter()` takes no arguments and hardcodes `NewLimiterWithCapacity(10, 60*time.Minute, 1000)`. Changing the per-IP limit requires a code change, not an env var.

### resettoken/

- Crypto/rand for token generation (never math/rand)
- Base64 URL encoding (safe for URLs)
- Store tokens server-side with expiry

### rpchandler/

- JSON-RPC 2.0 specification compliance
- Renamed from `rpc/` to avoid Go stdlib `net/rpc` conflict (revive var-naming)
- Types: `Request` (was `JSONRPC`), `Response` (was `JSONRPCResponse`) — avoid stuttering (`rpchandler.JSONRPCResponse`)
- Error codes defined in [docs/api-reference.md](../docs/api-reference.md)
- Request validation before processing
- Endpoint: `POST /api/rpc`
- **`extractClientIP` trusts proxy headers unconditionally.** It returns the first valid IP among the leftmost `X-Forwarded-For` value, `X-Real-IP`, and `fiber.Ctx.IP()`, falling back to `0.0.0.0`. There is no trusted-proxy allow-list — `fiber.New` in `main.go` sets no `TrustProxy`, and no option reads one. Any client that can reach the app directly picks its own per-IP rate-limit bucket by setting the header, so the app must not be exposed without a proxy that **overwrites** `X-Forwarded-For`.

### Health Check

- Endpoint: `GET /health/live` (returns `{"status": "alive"}`)
- Used by Docker HEALTHCHECK via `--health-check` flag
- Implemented in `main.go`

### validators/

- Pure functions (no side effects)
- Configurable policies from environment
- Clear error messages for user feedback
