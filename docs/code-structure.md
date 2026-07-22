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
├── rpc/            # JSON-RPC handlers for all API methods
├── validators/     # Password validation rules and enforcement
└── web/            # Web server, static assets, and templates
```

---

## Package Details

### `internal/email`

**Purpose**: SMTP email service for sending password reset links.

**Files**:

- `service.go` - Email service implementation with SMTP configuration
- `service_test.go` - Unit and integration tests (31.2% coverage)

**Key Types**:

```go
type Service struct {
    smtpHost     string
    smtpPort     int
    smtpUsername string
    smtpPassword string
    fromAddress  string
    fromName     string
}
```

**Public API**:

- `NewService(...)` - Create new email service with SMTP config
- `SendPasswordResetEmail(to, token, resetURL string)` - Send reset email

**Dependencies**:

- `net/smtp` - SMTP client
- Environment variables for configuration

**Test Coverage**: 31.2%

- ✅ Service creation and configuration
- ✅ Email template rendering
- ⚠️ Limited SMTP connection testing (requires test server)

---

### `internal/options`

**Purpose**: Application configuration and environment variable management.

**Files**:

- `app.go` - Configuration struct and environment variable loading

**Key Types**:

```go
type Opts struct {
    // LDAP Configuration
    LDAPHost                 string
    LDAPPort                 int
    LDAPUseTLS               bool
    LDAPBaseDN               string
    LDAPUserAttribute        string

    // Password Policy
    MinLength                int
    MinNumbers               int
    MinSymbols               int
    MinUppercase             int
    MinLowercase             int
    PasswordCanIncludeUsername bool

    // Password Reset Feature
    PasswordResetEnabled     bool
    SMTPHost                 string
    SMTPPort                 int
    ResetTokenValidityHours  int

    // Server Configuration
    Port                     int
    TrustedProxies           []string
}
```

**Public API**:

- `LoadFromEnv()` - Load configuration from environment variables
- `Validate()` - Validate configuration completeness

**Configuration Sources**:

1. Environment variables (`.env` file)
2. Default values for optional settings
3. Validation on startup

---

### `internal/ratelimit`

**Purpose**: Rate limiting middleware to prevent abuse of password reset requests.

**Files**:

- `limiter.go` - Rate limiter implementation with IP-based tracking
- `limiter_test.go` - Comprehensive unit tests (72.3% coverage)

**Key Types**:

```go
type Limiter struct {
    maxRequests int
    window      time.Duration
    requests    map[string][]time.Time
    mu          sync.RWMutex
}
```

**Public API**:

- `NewLimiter(maxRequests int, window time.Duration)` - Create rate limiter
- `Allow(ip string) bool` - Check if IP is allowed to make request
- `Reset(ip string)` - Clear rate limit for IP (for testing)

**Implementation Details**:

- **Sliding window algorithm**: Tracks requests in time window
- **IP-based**: Uses client IP address as key
- **Thread-safe**: Uses RWMutex for concurrent access
- **Memory bounded**: Automatic cleanup of expired entries

**Default Configuration**:

- Max requests: 3
- Time window: 1 hour
- Per IP address

**Test Coverage**: 72.3%

- ✅ Basic allow/deny logic
- ✅ Sliding window behavior
- ✅ Concurrent access
- ✅ Reset functionality
- ⚠️ Memory cleanup not fully tested

---

### `internal/resettoken`

**Purpose**: Secure token generation and storage for password reset flow.

**Files**:

- `token.go` - Cryptographic token generation
- `token_test.go` - Token generation tests
- `store.go` - In-memory token storage with expiration
- `store_test.go` - Comprehensive store tests (71.7% coverage)

**Key Types**:

```go
// Token generation
func GenerateToken() (string, error)

// Token storage
type Store struct {
    tokens map[string]TokenData
    mu     sync.RWMutex
}

type TokenData struct {
    Email     string
    ExpiresAt time.Time
}
```

**Public API**:

```go
// Token operations
GenerateToken() (string, error)           // Generate 256-bit secure token
NewStore() *Store                         // Create token store
Store(token, email string, ttl time.Duration) // Store token with expiration
Validate(token string) (email string, error)  // Validate and consume token
Delete(token string)                      // Explicitly delete token
```

**Security Features**:

- **Cryptographically secure**: Uses `crypto/rand` for token generation
- **256-bit tokens**: Encoded as URL-safe base64 (43 characters)
- **Time-limited**: Configurable TTL (default 24 hours)
- **Single-use**: Tokens deleted after successful validation
- **Automatic expiration**: Background cleanup of expired tokens

**Test Coverage**: 71.7%

- ✅ Token generation and uniqueness
- ✅ Store/validate/delete operations
- ✅ Expiration handling
- ✅ Concurrent access
- ⚠️ Edge cases for cleanup timing

---

### `internal/rpchandler`

**Purpose**: JSON-RPC handlers for all API methods.

**Files**:

- `handler.go` - Main RPC router and middleware
- `dto.go` - Data transfer objects for RPC methods
- `change_password.go` - Password change RPC handler
- `request_password_reset.go` - Request reset token handler
- `request_password_reset_test.go` - Reset request tests
- `reset_password.go` - Complete password reset handler
- `reset_password_test.go` - Password reset tests

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

- Rate limiting check (3 requests/hour per IP)
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
- Delete token after successful reset

**Test Coverage**: 45.6%

- ✅ Happy path for all methods
- ✅ Error handling for invalid inputs
- ✅ LDAP integration with testcontainers
- ⚠️ Edge cases and error conditions partially covered

---

### `internal/validators`

**Purpose**: Password validation rules matching server-side policy.

**Files**:

- `validate.go` - Validation rule implementations
- `validate_test.go` - Comprehensive validation tests (100% coverage)

**Public API**:

```go
// Validation functions
ValidateMinLength(password string, minLength int) error
ValidateMinNumbers(password string, minNumbers int) error
ValidateMinSymbols(password string, minSymbols int) error
ValidateMinUppercase(password string, minUppercase int) error
ValidateMinLowercase(password string, minLowercase int) error
ValidateNoUsername(password, username string) error

// Combined validation
ValidatePassword(password, username string, opts *options.Opts) error
```

**Validation Rules**:

- ✅ Minimum length (configurable, default 8)
- ✅ Minimum numbers (configurable, default 1)
- ✅ Minimum symbols (configurable, default 1)
- ✅ Minimum uppercase (configurable, default 1)
- ✅ Minimum lowercase (configurable, default 1)
- ✅ Username exclusion (optional, default enabled)

**Test Coverage**: 100% ✅

- All validation rules tested
- Edge cases covered
- Combined validation tested
- Configuration variations tested

---

### `internal/web`

**Purpose**: Web server, static asset serving, and HTML template rendering.

**Structure**:

```
web/
├── static/
│   ├── js/
│   │   ├── app.ts              # Main page (password change)
│   │   ├── forgot-password.ts  # Password reset request
│   │   ├── reset-password.ts   # Password reset completion
│   │   └── validators.ts       # Client-side validation
│   ├── styles.css              # Compiled Tailwind CSS
│   ├── favicon.ico             # Browser favicon
│   ├── *.png                   # PWA icons
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
RenderForgotPassword() ([]byte, error)
RenderResetPassword(opts *options.Opts) ([]byte, error)
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
tsc                    # Compile TypeScript
uglify-js             # Minify for production
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

**Direct**:

- `github.com/gofiber/fiber/v2` - Web framework
- `github.com/joho/godotenv` - Environment variable loading
- `github.com/netresearch/simple-ldap-go` - LDAP client

**Testing**:

- `github.com/testcontainers/testcontainers-go` - Integration testing
- `github.com/testcontainers/testcontainers-go/modules/openldap` - LDAP test server
- `github.com/stretchr/testify` - Test assertions

### Node Dependencies (package.json)

**Build Tools**:

- `typescript` - Type-safe JavaScript
- `@tailwindcss/postcss` - CSS framework
- `uglify-js` - JavaScript minification
- `postcss` - CSS processing (minification via @tailwindcss/postcss / Lightning CSS)

**Development**:

- `concurrently` - Parallel script execution
- `nodemon` - File watching and hot reload
- `prettier` - Code formatting

---

## Testing Strategy

### Unit Tests

- **Package**: `internal/validators` - 100% coverage ✅
- **Package**: `internal/ratelimit` - 72.3% coverage
- **Package**: `internal/resettoken` - 71.7% coverage

### Integration Tests

- **Package**: `internal/rpchandler` - 45.6% coverage
- **Package**: `internal/email` - 31.2% coverage
- Uses testcontainers for real LDAP server
- Tests complete RPC workflows

### E2E Tests

- Recommended: Playwright for browser automation
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
- CSRF protection in Fiber middleware

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
