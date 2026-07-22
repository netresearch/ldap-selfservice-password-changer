# Code Structure Reference

Detailed documentation of internal packages and code organization.

---

## Package Overview

```
internal/
├── email/          # Email service for password reset notifications
├── options/        # Application configuration and environment variables
├── ratelimit/      # Rate limiting middleware for API protection
├── resettoken/     # Token generation, storage, and validation
├── rpchandler/     # JSON-RPC handlers for all API methods
├── validators/     # Password validation rules and enforcement
└── web/            # Web server, static assets, and templates
```

---

## Package Details

### `internal/email`

**Purpose**: Renders and sends the password reset email over SMTP.

**Files**:

- `service.go` - `Config`, `Service`, `NewService`, `SendResetEmail`, reset-link construction, recipient address validation
- `render.go` - Template loading, parsing and execution for subject, text body and HTML body
- `message.go` - `multipart/alternative` RFC 5322 message assembly with quoted-printable bodies
- `headers.go` - Address/header-name/header-value validation, RFC 2047 subject encoding, operator header overrides
- `templates/` - Embedded defaults: `reset.txt.tmpl` and `reset.html.tmpl`

**Key Types**:

```go
type Config struct {
    SMTPHost, SMTPUsername, SMTPPassword string
    SMTPPort                             int
    FromAddress, FromName, ReplyTo       string
    BaseURL                              string
    ExpiryMinutes                        uint

    SubjectTemplate  string            // inline template; empty => built-in default
    TemplateHTMLPath string            // file path; empty => embedded default
    TemplateTextPath string            // file path; empty => embedded default
    HeaderOverrides  map[string]string // raw header name => verbatim value
}

type Service struct {
    config   Config
    renderer *renderer
    now      func() time.Time // pinned in tests to assert the Date header
}
```

**Public API**:

- `NewService(config *Config) (*Service, error)` - Build the service, loading, parsing and dry-running all three templates. Fails fast: a missing, unparseable or field-invalid template is an error here, not at first send.
- `SendResetEmail(to, token string) error` - Render and send the reset email
- `ValidateEmailAddress(email string) bool` - Strict regex check for recipient addresses (derived from directory data)
- `ValidateConfiguredAddress(addr string) error` - Permissive RFC 5322 check for operator-supplied addresses, so senders such as `noreply@localhost` are accepted
- `ValidateHeaderName(name string) error` / `ValidateHeaderValue(value string) error` - Reject malformed names and control characters in override values

**Template data** (`resetEmailData`): `ResetLink`, `Token`, `BaseURL`, `Recipient`, `ExpiryMinutes`. Templates are parsed with `missingkey=error`, so an undefined field surfaces during the startup dry-run instead of rendering `<no value>`.

**Delivery semantics**: `sendEmail` passes a fixed `[]string{to}` to `smtp.SendMail`, so the SMTP envelope recipient is always the reset requester. `To`/`Cc`/`Bcc` overrides are display-only — they add no delivery target, and a `Bcc` override is written as a visible header line in the message the requester receives. `MIME-Version`, `Content-Type` and `Content-Transfer-Encoding` cannot be overridden.

**Dependencies**: `net/smtp`, `net/mail`, `net/textproto`, `mime`, `mime/multipart`, `mime/quotedprintable`, `text/template`, `html/template`, `embed` — all standard library.

---

### `internal/options`

**Purpose**: Application configuration and environment variable management.

**Files**:

- `app.go` - `Opts`, `ConfigError`, flag/environment parsing and validation

**Key Types**:

```go
type Opts struct {
    Port             string
    LDAP             ldap.Config // from github.com/netresearch/simple-ldap-go
    ReadonlyUser     string
    ReadonlyPassword string

    MinLength                  uint
    MinNumbers                 uint
    MinSymbols                 uint
    MinUppercase               uint
    MinLowercase               uint
    PasswordCanIncludeUsername bool

    // Password Reset Configuration
    PasswordResetEnabled        bool
    ResetIdentifierMode         ResetIdentifierMode
    ResetTokenExpiryMinutes     uint
    ResetRateLimitRequests      uint
    ResetRateLimitWindowMinutes uint
    SMTPHost                    string
    SMTPPort                    uint
    SMTPUsername                string
    SMTPPassword                string
    SMTPFromAddress             string
    AppBaseURL                  string

    SMTPFromName         string
    EmailReplyTo         string
    EmailTemplateHTML    string
    EmailTemplateText    string
    EmailTemplateSubject string
    SMTPHeaderOverrides  map[string]string

    // Optional dedicated reset account; falls back to ReadonlyUser
    ResetUser     string
    ResetPassword string
}

// ResetIdentifierMode is "email" (default), "username" or "both".
type ResetIdentifierMode string
```

**Public API**:

- `Parse() (*Opts, error)` - Parse `os.Args` after loading `.env` / `.env.local`
- `ParseArgs(args []string) (*Opts, error)` - Same, with an explicit argument slice
- `MustParse() *Opts` - `Parse`, exiting on error
- `ConfigError` - Accumulates validation failures; `Add`, `HasErrors`, `Error`
- `ResetIdentifierMode.Valid() bool` - Reports whether the mode is recognized

**Configuration Sources**:

Environment variables supply the _defaults_ of the `flag.FlagSet`, so an
explicitly passed flag wins over the environment, which wins over the built-in
default. `godotenv.Load(".env.local", ".env")` runs first and does not overwrite
variables already present in the process environment. Every validation failure
is collected into one `ConfigError` rather than aborting at the first.

There is no `TrustedProxies` option; see `internal/rpchandler/ip_extraction.go`
for the actual, allow-list-free client-IP handling.

---

### `internal/ratelimit`

**Purpose**: Sliding-window rate limiting to prevent abuse of the password change and reset endpoints.

**Files**:

- `limiter.go` - Generic, key-agnostic sliding-window limiter
- `ip_limiter.go` - Thin wrapper preconfigured for per-IP limiting
- `limiter_test.go`, `limiter_internal_test.go`, `ip_limiter_test.go`, `ip_limiter_internal_test.go` - Unit tests

**Key Types**:

```go
type Limiter struct {
    mu             sync.RWMutex
    entries        map[string]*Entry
    maxRequests    int
    window         time.Duration
    maxIdentifiers int
}
```

**Public API**:

- `NewLimiter(maxRequests int, window time.Duration) *Limiter` - Create limiter with the default capacity (10000 identifiers)
- `NewLimiterWithCapacity(maxRequests int, window time.Duration, capacity int) *Limiter` - Create limiter with an explicit capacity
- `AllowRequest(identifier string) bool` - Check and record a request for an arbitrary key
- `CleanupExpired() int` / `StartCleanup(interval time.Duration) chan struct{}` - Evict expired entries
- `Count() int` / `IsFull() bool` - Capacity introspection
- `NewIPLimiter() *IPLimiter` - Per-IP limiter with a **hardcoded** `NewLimiterWithCapacity(10, 60*time.Minute, 1000)`; it takes no arguments and no environment variable configures it

**Implementation Details**:

- **Sliding window algorithm**: Tracks requests in time window
- **Key-agnostic**: `AllowRequest` takes any string; the caller chooses whether it is an IP, a typed identifier, or a resolved account
- **Thread-safe**: Uses RWMutex for concurrent access
- **Memory bounded**: Capacity limit plus automatic cleanup of expired entries; fails closed when at capacity

**Default Configuration**:

- Per-IP limiter: 10 requests / 60 minutes, max 1000 IPs — hardcoded
- Reset limiter: `RESET_RATE_LIMIT_REQUESTS` (3) / `RESET_RATE_LIMIT_WINDOW_MINUTES` (60), keyed per identifier, not per IP

**Tested behaviour** (run `go test -cover ./internal/ratelimit/` for the number):

- ✅ Basic allow/deny logic
- ✅ Sliding window behavior
- ✅ Concurrent access
- ✅ Capacity limits and expiry cleanup

---

### `internal/resettoken`

**Purpose**: Secure token generation and storage for password reset flow.

**Files**:

- `token.go` - Cryptographic token generation
- `store.go` - In-memory token storage with expiration and capacity limit
- `clock.go` - `Clock` indirection so tests can pin the current time
- `token_test.go`, `token_fuzz_test.go`, `store_test.go`, `store_internal_test.go`, `clock_test.go` - Tests

**Key Types**:

```go
type ResetToken struct {
    Token            string
    Username         string
    Email            string
    CreatedAt        time.Time
    ExpiresAt        time.Time
    Used             bool
    RequiresApproval bool
}

type Store struct {
    mu     sync.RWMutex
    tokens map[string]*ResetToken
}
```

**Public API**:

```go
GenerateToken() (string, error)                          // 32 random bytes, URL-safe base64
NewStore() *Store                                        // Create token store
(*Store) Store(token *ResetToken) error                  // Insert; rejects duplicates and over-capacity
(*Store) Get(tokenString string) (*ResetToken, error)    // Look up; does not consume
(*Store) MarkUsed(tokenString string) error              // Flag a token as spent
(*Store) Delete(tokenString string) error                // Remove a token
(*Store) CleanupExpired() int                            // Evict expired tokens
(*Store) StartCleanup(interval time.Duration) chan struct{}
(*Store) Count() int
(*Store) IsFull() bool
(*ResetToken) IsExpired() bool
```

**Security Features**:

- **Cryptographically secure**: `crypto/rand` for token generation
- **256-bit tokens**: 32 random bytes, URL-safe base64 without padding (43 characters)
- **Time-limited**: `ExpiresAt` set by the caller from `RESET_TOKEN_EXPIRY_MINUTES` (default 15 minutes)
- **Single-use**: `reset_password` calls `MarkUsed` after a successful reset; the entry stays in the store until it expires and cleanup removes it
- **Capacity bounded**: `maxCapacity` is 10000 entries. At capacity the store first evicts expired tokens and, failing that, rejects the new token — it never evicts a live one
- **Automatic expiration**: `StartCleanup` runs background eviction

---

### `internal/rpchandler`

**Purpose**: JSON-RPC handlers for all API methods.

**Files**:

- `handler.go` - Main RPC router and middleware
- `dto.go` - Data transfer objects for RPC methods
- `change_password.go` - Password change RPC handler
- `request_password_reset.go` - Request reset token handler
- `reset_password.go` - Complete password reset handler
- `ip_extraction.go` - Client-IP resolution for the per-IP rate limiter
- `password_validation.go` - Server-side password policy enforcement

**RPC Methods**:

#### `change-password`

```typescript
Request: {
  method: "change-password",
  params: [username, currentPassword, newPassword]
}
Response: {
  success: true
}
```

**Handler**: `internal/rpchandler/change_password.go`

- Validates input parameters
- Authenticates with LDAP using current password
- Changes password via LDAP modify operation
- Returns success/error

#### `request-password-reset`

```typescript
Request: {
  method: "request-password-reset",
  params: [emailOrUsername]
}
Response: {
  success: true
}
```

**Handler**: `internal/rpchandler/request_password_reset.go`

- Rate limiting check: first per IP (10 requests/hour, hardcoded), then per typed identifier and per resolved account (3 requests/hour by default)
- Resolve the account per `RESET_IDENTIFIER_MODE` (`email` default, `username`, or `both`).
  In `both`, an input containing `@` is looked up by email, otherwise by username.
- Generate secure reset token
- Send reset email with token link — always to the account's LDAP-registered
  address, never to the typed identifier
- Always returns success (prevents account enumeration)

**`RESET_IDENTIFIER_MODE`**: `username` / `both` exist because Active Directory does
not enforce a unique `mail` attribute; an email shared by multiple accounts is
ambiguous and yields the generic success without sending mail (those users reset via
their unique username).

#### `reset-password`

```typescript
Request: {
  method: "reset-password",
  params: [token, newPassword]
}
Response: {
  success: true
}
```

**Handler**: `internal/rpchandler/reset_password.go`

- Validate and consume reset token
- Retrieve user email from token store
- Lookup user DN in LDAP
- Reset password via LDAP admin bind
- Mark the token used (`Store.MarkUsed`) after a successful reset; the entry is
  removed later by expiry cleanup, not here

**Tested behaviour** (run `go test -cover ./internal/rpchandler/` for the number):

- ✅ Happy path for all methods
- ✅ Error handling for invalid inputs
- ✅ Fuzz tests for client-IP extraction and password validation
- ✅ LDAP integration behind the `integration` build tag, against a real server

---

### `internal/validators`

**Purpose**: Character-class counting predicates used by the server-side password policy.

**Files**:

- `validate.go` - Validation rule implementations
- `validate_test.go` - Validation tests

**Public API**:

```go
MinNumbersInString(value string, amount uint) bool
MinSymbolsInString(value string, amount uint) bool
MinUppercaseLettersInString(value string, amount uint) bool
MinLowercaseLettersInString(value string, amount uint) bool
```

Each returns a bool, not an error, and counts ASCII runes only. The package holds
no minimum-length or username check: those, and the human-readable error
messages, live in `rpchandler.ValidateNewPassword`, which composes these four
predicates with `opts`.

**Validation Rules** (defaults from `internal/options`):

- ✅ Minimum length (`MIN_LENGTH`, default 8) — enforced in `rpchandler`
- ✅ Maximum length — `rpchandler.MaxPasswordLength`, a hardcoded 128, not configurable
- ✅ Minimum numbers (`MIN_NUMBERS`, default 1)
- ✅ Minimum symbols (`MIN_SYMBOLS`, default 1)
- ✅ Minimum uppercase (`MIN_UPPERCASE`, default 1)
- ✅ Minimum lowercase (`MIN_LOWERCASE`, default 1)
- ✅ Username exclusion (`PASSWORD_CAN_INCLUDE_USERNAME`, default false, i.e. excluded) — enforced in `rpchandler`

---

### `internal/web`

**Purpose**: Web server, static asset serving, and HTML template rendering.

**Structure**:

```
web/
├── static/
│   ├── js/                     # .ts sources plus the .js tsc emits beside them
│   │   ├── app.ts              # Main page (password change)
│   │   ├── forgot-password.ts  # Password reset request
│   │   ├── reset-password.ts   # Password reset completion
│   │   ├── *-init.ts           # Per-page bootstrap entry points
│   │   ├── theme-init.ts       # Theme applied before first paint
│   │   ├── density-init.ts     # Density applied before first paint
│   │   ├── toggles.ts          # Theme/density toggle wiring
│   │   ├── policy-ui.ts        # Password policy checklist rendering
│   │   ├── error-utils.ts      # Shared error formatting
│   │   └── validators.ts       # Client-side validation
│   ├── static.go               # embed.FS for this directory
│   ├── styles.css              # Compiled Tailwind CSS
│   ├── favicon.ico             # Browser favicon
│   ├── *.png, logo.webp, safari-pinned-tab.svg
│   ├── browserconfig.xml
│   └── site.webmanifest        # PWA manifest
├── templates/
│   ├── atoms/                  # Basic UI components
│   │   ├── button-primary.html
│   │   ├── button-secondary.html
│   │   ├── button-toggle.html
│   │   ├── icons.html
│   │   └── link.html
│   ├── molecules/              # Composite components
│   │   ├── density-init-script.html
│   │   ├── form-submit.html
│   │   ├── html-head.html
│   │   ├── input-field.html
│   │   ├── page-footer.html
│   │   ├── page-header.html
│   │   ├── page-title.html
│   │   ├── success-message.html
│   │   ├── theme-init-script.html
│   │   └── toggle-buttons.html
│   ├── index.html              # Password change page
│   ├── forgot-password.html    # Reset request page
│   └── reset-password.html     # Reset completion page
├── tailwind.css                # Tailwind source
└── templates.go                # Template rendering functions
```

#### TypeScript Modules

**`app.ts`** (Main Password Change Page)

- Theme toggle (light/dark/auto)
- Density toggle (comfortable/compact/auto)
- Password reveal buttons
- Real-time validation
- Form submission with RPC call
- Password strength indicators

**`forgot-password.ts`** (Reset Request)

- Email input with validation
- Theme and density toggles
- RPC call to request reset
- Success message display

**`reset-password.ts`** (Reset Completion)

- Token-based authentication
- New password input with validation
- Password strength indicators
- Theme and density toggles
- RPC call to reset password

**`validators.ts`** (Shared Validation)

- Client-side validation matching server rules
- Real-time feedback on input
- Error message generation
- Validator composition

#### Template System

**Atomic Design Pattern**:

- **Atoms**: Basic building blocks (buttons, icons, links)
- **Molecules**: Composite components (forms, headers, footers)
- **Pages**: Full page templates (index, forgot-password, reset-password)

**Template Rendering** (`templates.go`):

```go
RenderIndex(opts *options.Opts) ([]byte, error)
RenderForgotPassword(opts *options.Opts) ([]byte, error)
RenderResetPassword(opts *options.Opts) ([]byte, error)
MakeInputOpts(name, placeholder, inputType, autocomplete, help string) InputOpts
```

**Features**:

- Go `html/template` for server-side rendering
- Embedded templates (no external files)
- Configuration-driven (password policy displayed)
- Reusable components via template composition

---

## Frontend Build Pipeline

### Asset Compilation

**TypeScript → JavaScript**:

```bash
tsc                    # Compile TypeScript; no minifier is configured
```

**Tailwind CSS → CSS**:

```bash
postcss               # Process Tailwind directives
@tailwindcss/postcss  # Prefixing, nesting, minification via Lightning CSS
```

**Build Scripts** (`package.json`):

- `bun run build:assets` - Build both JS and CSS
- `bun run js:build` - TypeScript compilation (`tsc`)
- `bun run css:build` - Tailwind CSS compilation
- `bun run dev` - Watch mode with hot reload

### Go Embed

Static assets embedded in binary via `//go:embed`:

```go
//go:embed static
var staticFS embed.FS
```

**Benefits**:

- Single binary deployment
- No external file dependencies
- Simplified distribution

---

## Dependencies

### Go Dependencies (go.mod)

**Direct** (the full `require` block of `go.mod`):

- `github.com/gofiber/fiber/v3` - Web framework
- `github.com/joho/godotenv` - Environment variable loading
- `github.com/netresearch/simple-ldap-go` - LDAP client
- `github.com/valyala/fasthttp` - HTTP engine under Fiber
- `github.com/stretchr/testify` - Test assertions

There is no testcontainers dependency. Integration tests are gated by the
`integration` build tag and talk to services the developer or CI already
started; see the Testing Strategy section below.

### Node Dependencies (package.json)

All are `devDependencies`; the package has no runtime `dependencies`.

**Build Tools**:

- `typescript` - Type-safe JavaScript; `tsc` is the only JS build step, and no minifier is configured
- `tailwindcss` / `@tailwindcss/postcss` - CSS framework (minification via Lightning CSS)
- `postcss` / `postcss-cli` - CSS processing

**Development**:

- `eslint`, `typescript-eslint`, `@eslint/js`, `eslint-config-prettier` - Linting
- `prettier`, `prettier-plugin-go-template`, `prettier-plugin-tailwindcss` - Code formatting

`bun run dev` also invokes `air` for Go hot-reload; `air` is not declared in
`package.json` and must be installed separately.

---

## Testing Strategy

Per-package coverage percentages are not listed here — they go stale faster than
anyone updates them. Run `go test -cover ./...`, or read the
[Codecov dashboard](https://codecov.io/gh/netresearch/ldap-selfservice-password-changer).

### Unit Tests

- Default build, no tags: `go test ./...`
- `*_internal_test.go` files test unexported behaviour from inside the package
- `*_fuzz_test.go` files cover client-IP extraction, password validation and email input

### Integration Tests

- Build tag `integration`; `make test-integration` runs `go test -v -race -tags=integration ./...`
- Backing services come from `docker compose --profile test up`
- Configured through the same environment variables as the app; a test whose
  variables are unset skips instead of failing

### E2E Tests

- Build tag `e2e`, in `e2e/e2e_test.go`; `make test-e2e` runs
  `go test -v -race -tags=e2e ./e2e/...`
- Go and `httptest` against the assembled Fiber app — no browser automation
- See [Testing Guide](testing-guide.md) for setup

---

## Code Style and Conventions

### Go Code

- **Formatting**: `gofmt` standard
- **Linting**: `golint` compliance
- **Naming**: Exported functions capitalized, private lowercase
- **Error handling**: Explicit error returns, no panics in production code

### TypeScript Code

- **Strict mode**: Enabled in `tsconfig.json`
- **No `any` types**: Type safety enforced
- **Naming**: camelCase for variables, PascalCase for types
- **Module system**: ES modules with `.js` extension

### HTML Templates

- **Atomic design**: atoms < molecules < pages
- **Accessibility**: ARIA labels, semantic HTML
- **Formatting**: Prettier with go-template plugin

---

## Performance Considerations

### Backend

- **Connection pooling**: LDAP connections reused
- **Concurrent requests**: Fiber handles async I/O
- **Memory management**: Token store with automatic cleanup
- **Rate limiting**: Protects against abuse

### Frontend

- **Asset minification**: Lightning CSS via `@tailwindcss/postcss` (no JS minifier configured)
- **HTTP/2**: Parallel asset loading
- **Lazy loading**: Module imports for page-specific code
- **PWA**: Offline capability with service worker

---

## Security Architecture

See [Security Documentation](security.md) for comprehensive security architecture.

**Key security components in code**:

- `internal/ratelimit` - Abuse prevention
- `internal/resettoken` - Cryptographic token generation
- `internal/validators` - Input validation
- LDAPS support in LDAP client

There is no CSRF middleware: nothing in the tree references `csrf`, and the
security assessments record it as an accepted, unimplemented finding
(`docs/security-assessment-revised-2025-10-09.md`, WAF-02).

---

## Extending the Application

### Adding New RPC Methods

1. Define method in `internal/rpchandler/handler.go`
2. Create handler file `internal/rpchandler/method_name.go`
3. Write tests in `internal/rpchandler/method_name_test.go`
4. Update API documentation in `docs/api-reference.md`

### Adding New UI Pages

1. Create template in `internal/web/templates/page-name.html`
2. Create TypeScript in `internal/web/static/js/page-name.ts`
3. Add render function in `internal/web/templates/templates.go`
4. Add route in `main.go`
5. Update build scripts if needed

### Adding Configuration Options

1. Add field to `internal/options/app.go`
2. Add environment variable loading
3. Add validation if required
4. Update `.env.local.example`
5. Document in `docs/development-guide.md`

---

## Further Reading

- [API Reference](api-reference.md) - RPC method specifications
- [Development Guide](development-guide.md) - Setup and workflows
- [Testing Guide](testing-guide.md) - Testing strategies
- [Architecture](architecture.md) - System design overview

---

**Last Updated**: 2025-10-08
**Maintained By**: Development Team
