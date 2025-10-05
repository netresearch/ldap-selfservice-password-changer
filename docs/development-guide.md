# Development Guide

## Prerequisites

### Required Tools

- **Go**: 1.24.0+ (tested with 1.25.1)
- **Node.js**: v16+ (tested with v22)
- **pnpm**: 10.17.1+ (installed via Corepack)
- **Git**: For version control

### System Requirements

- Linux, macOS, or Windows with WSL2
- 2GB RAM minimum (4GB recommended)
- LDAP/ActiveDirectory server for testing

## Initial Setup

### 1. Clone Repository

```bash
git clone https://github.com/netresearch/ldap-selfservice-password-changer.git
cd ldap-selfservice-password-changer
```

### 2. Enable Corepack

```bash
corepack enable
```

This ensures pnpm version matches package.json specification (10.17.1).

### 3. Install Dependencies

```bash
pnpm install
```

**What this does**:

- Installs TypeScript compiler
- Installs Tailwind CSS and PostCSS tools
- Installs development tools (nodemon, concurrently, prettier)
- Downloads frontend dependencies (listed in package.json)

**Go dependencies** are handled automatically by Go modules.

### 4. Configure LDAP Connection

Create `.env.local` file:

```bash
touch .env.local
```

Add required configuration:

```bash
# LDAP Server Configuration
LDAP_SERVER="ldaps://dc1.example.com:636"
LDAP_IS_AD="true"
LDAP_BASE_DN="DC=example,DC=com"
LDAP_READONLY_USER="readonly"
LDAP_READONLY_PASSWORD="readonly-password"

# Optional: Password Validation Rules (defaults shown)
MIN_LENGTH="8"
MIN_NUMBERS="1"
MIN_SYMBOLS="1"
MIN_UPPERCASE="1"
MIN_LOWERCASE="1"
PASSWORD_CAN_INCLUDE_USERNAME="false"
```

**Configuration Notes**:

- `.env.local` is gitignored (never commit credentials)
- `.env` contains defaults (committed, no sensitive data)
- Command-line flags override environment variables

### 5. Verify Setup

```bash
pnpm build
./ldap-selfservice-password-changer --help
```

**Expected output**: Help text showing all available flags

## Development Workflows

### Development Mode (Hot Reload)

**Command**:

```bash
pnpm dev
```

**What happens**:

1. Builds initial assets (TypeScript + CSS)
2. Starts three concurrent watchers:
   - **TypeScript watcher**: `tsc -w` (rebuilds on .ts changes)
   - **CSS watcher**: `postcss -w` (rebuilds on .css changes)
   - **Go watcher**: `nodemon` (restarts server on any file change)

**Output Example**:

```
[js]  15:32:41 - Starting compilation in watch mode...
[css] Rebuilding...
[go]  [nodemon] starting `pnpm go:start`
[go]  Server listening on :3000
```

**File Change Behavior**:

| Change Type       | Action                            | Time to Reflect |
| ----------------- | --------------------------------- | --------------- |
| `*.ts` file       | TypeScript recompile → Go restart | ~2-3 seconds    |
| `tailwind.css`    | PostCSS rebuild → Go restart      | ~1-2 seconds    |
| `*.go` file       | Go restart only                   | ~1 second       |
| `*.html` template | Go restart only                   | ~1 second       |

**Access**: http://localhost:3000

**Stopping**: Ctrl+C (terminates all watchers)

### Production Build

**Command**:

```bash
pnpm build
```

**Steps**:

1. Compile TypeScript to JavaScript
2. Minify JavaScript with UglifyJS
3. Build CSS with PostCSS + Tailwind + CSSnano
4. Compile Go binary with all assets embedded

**Output**: `./ldap-selfservice-password-changer` executable

**Running**:

```bash
./ldap-selfservice-password-changer \
  -ldap-server ldaps://dc1.example.com:636 \
  -base-dn DC=example,DC=com \
  -readonly-user readonly \
  -readonly-password readonly-password \
  -active-directory
```

### Component Builds

#### TypeScript Only

```bash
# Development (with source maps)
pnpm js:dev

# Production (minified)
pnpm js:build
```

**Output**: `internal/web/static/js/app.js`, `internal/web/static/js/validators.js`

#### CSS Only

```bash
# Development (with watch)
pnpm css:dev

# Production (minified)
pnpm css:build
```

**Output**: `internal/web/static/styles.css`

#### Go Only

```bash
# Run without rebuild
pnpm go:start

# Build binary
pnpm go:build
```

**Output**: `./ldap-selfservice-password-changer`

### Testing Workflow

#### Unit Tests

```bash
go test ./...
```

**Current Coverage**: `internal/validators` package only

**Example Output**:

```
ok      github.com/netresearch/ldap-selfservice-password-changer/internal/validators    0.002s
```

#### Integration Testing

**Not implemented yet**. See [Testing Guide](testing-guide.md) for recommendations.

### Code Formatting

#### Automatic Formatting

```bash
# Format all files
pnpm prettier --write .

# Check formatting without changes
pnpm prettier --check .
```

**Files Formatted**:

- TypeScript (`.ts`)
- JavaScript (`.js`)
- CSS (`.css`)
- HTML (`.html`)
- Go templates (with prettier-plugin-go-template)
- Markdown (`.md`)

**Configuration**: `.prettierrc.mjs`

#### Manual Go Formatting

```bash
gofmt -w .
```

**Note**: Go formatting is standard `gofmt`, Prettier handles templates only.

## Project Structure

```
ldap-selfservice-password-changer/
├── main.go                           # Application entry point
│
├── internal/                         # Private application code
│   ├── options/
│   │   └── app.go                   # Configuration parsing
│   ├── rpc/
│   │   ├── handler.go               # RPC dispatcher
│   │   ├── dto.go                   # Request/response types
│   │   └── change_password.go       # Password change logic
│   ├── validators/
│   │   ├── validate.go              # Password validators
│   │   └── validate_test.go         # Unit tests
│   └── web/
│       ├── templates/
│       │   ├── templates.go         # Template rendering
│       │   └── index.html           # HTML template
│       ├── static/
│       │   ├── static.go            # embed.FS declaration
│       │   ├── *.css                # Compiled CSS (generated)
│       │   └── js/
│       │       ├── app.ts           # Main application
│       │       ├── validators.ts    # Frontend validators
│       │       ├── *.js             # Compiled JS (generated)
│       │       └── *.js.map         # Source maps (dev only)
│       └── tailwind.css             # CSS entry point
│
├── scripts/
│   └── minify.js                    # JavaScript minification
│
├── docs/                            # Documentation
│   ├── architecture.md
│   └── architecture.png
│
├── claudedocs/                      # Generated documentation
│   ├── project-context-2025-10-04.md
│   ├── api-reference.md
│   ├── architecture-patterns.md
│   ├── development-guide.md
│   └── ...
│
├── .env                             # Default configuration (committed)
├── .env.local                       # Local overrides (gitignored)
├── go.mod                           # Go dependencies
├── go.sum                           # Go dependency checksums
├── package.json                     # Node.js dependencies
├── pnpm-lock.yaml                   # pnpm lock file
├── tsconfig.json                    # TypeScript configuration
├── tailwind.config.js               # Tailwind CSS configuration
├── postcss.config.js                # PostCSS configuration
├── .prettierrc.mjs                  # Prettier configuration
├── .prettierignore                  # Prettier ignore patterns
├── Dockerfile                       # Multi-stage Docker build
└── README.md                        # Project README
```

## Configuration Reference

### Environment Variables

| Variable                        | Required | Default | Description                               |
| ------------------------------- | -------- | ------- | ----------------------------------------- |
| `LDAP_SERVER`                   | ✅ Yes   | -       | LDAP server URI (`ldap://` or `ldaps://`) |
| `LDAP_IS_AD`                    | No       | `false` | ActiveDirectory mode flag                 |
| `LDAP_BASE_DN`                  | ✅ Yes   | -       | Base DN for LDAP searches                 |
| `LDAP_READONLY_USER`            | ✅ Yes   | -       | Readonly LDAP user                        |
| `LDAP_READONLY_PASSWORD`        | ✅ Yes   | -       | Readonly user password                    |
| `MIN_LENGTH`                    | No       | `8`     | Minimum password length                   |
| `MIN_NUMBERS`                   | No       | `1`     | Minimum numeric characters                |
| `MIN_SYMBOLS`                   | No       | `1`     | Minimum symbol characters                 |
| `MIN_UPPERCASE`                 | No       | `1`     | Minimum uppercase letters                 |
| `MIN_LOWERCASE`                 | No       | `1`     | Minimum lowercase letters                 |
| `PASSWORD_CAN_INCLUDE_USERNAME` | No       | `false` | Allow username in password                |

### Command-Line Flags

All environment variables have corresponding flags:

```bash
--ldap-server string
    LDAP server URI (ldap:// or ldaps://)

--active-directory
    Mark LDAP server as ActiveDirectory

--base-dn string
    Base DN of LDAP directory

--readonly-user string
    User for LDAP read operations

--readonly-password string
    Password for readonly user

--min-length uint
    Minimum password length (default: 8)

--min-numbers uint
    Minimum numbers in password (default: 1)

--min-symbols uint
    Minimum symbols in password (default: 1)

--min-uppercase uint
    Minimum uppercase letters (default: 1)

--min-lowercase uint
    Minimum lowercase letters (default: 1)

--password-can-include-username
    Allow password to include username
```

**Example**:

```bash
./ldap-selfservice-password-changer \
  --ldap-server ldaps://dc1.example.com:636 \
  --active-directory \
  --base-dn DC=example,DC=com \
  --readonly-user readonly \
  --readonly-password secret \
  --min-length 12 \
  --min-numbers 2
```

## Common Development Tasks

### Adding a New Validator

#### 1. Add Go Validator

**File**: `internal/validators/validate.go`

```go
func MinDigitsInString(value string, amount uint) bool {
    var counter uint = 0
    for _, c := range value {
        if c >= '0' && c <= '9' {
            counter++
        }
    }
    return counter >= amount
}
```

#### 2. Add Test

**File**: `internal/validators/validate_test.go`

```go
func TestMinDigitsInString(t *testing.T) {
    if !MinDigitsInString("abc123", 3) {
        t.Error("Expected true for 3 digits")
    }
    if MinDigitsInString("abc1", 3) {
        t.Error("Expected false for 1 digit")
    }
}
```

#### 3. Add Frontend Validator

**File**: `internal/web/static/js/validators.ts`

```typescript
export const mustIncludeDigits = (amount: number) => (v: string) =>
  v.split("").filter((c) => !isNaN(+c)).length < amount
    ? `The input must include at least ${amount} ${pluralize("digit", amount)}`
    : "";
```

#### 4. Apply Validator

**Backend** - `internal/rpc/change_password.go`:

```go
if !validators.MinDigitsInString(newPassword, c.opts.MinDigits) {
    return nil, fmt.Errorf("the new password must contain at least %d %s",
        c.opts.MinDigits, pluralize("digit", c.opts.MinDigits))
}
```

**Frontend** - `internal/web/static/js/app.ts`:

```typescript
[
  "new",
  [
    mustNotBeEmpty,
    mustIncludeDigits(opts.minDigits)
    // ... other validators
  ]
];
```

### Adding Configuration Option

#### 1. Add to Options Struct

**File**: `internal/options/app.go`

```go
type Opts struct {
    // ... existing fields
    MinDigits uint
}
```

#### 2. Add Flag Definition

**File**: `internal/options/app.go`

```go
fMinDigits = flag.Uint("min-digits", envIntOrDefault("MIN_DIGITS", 2),
    "Minimum amount of digits in the password.")
```

#### 3. Add to Struct Construction

**File**: `internal/options/app.go`

```go
return &Opts{
    // ... existing fields
    MinDigits: *fMinDigits,
}
```

#### 4. Add to Template Data

**File**: `internal/web/templates/index.html`

```html
<script type="module" defer>
  import { init } from "/static/js/app.js";

  init({
    // ... existing options
    minDigits: +"{{ .opts.MinDigits }}"
  });
</script>
```

#### 5. Add TypeScript Type

**File**: `internal/web/static/js/app.ts`

```typescript
type Opts = {
  // ... existing fields
  minDigits: number;
};
```

## Docker Development

### Build Docker Image

```bash
docker build -t ldap-password-changer .
```

**Build Process**:

1. Frontend build (Node.js 22)
2. Backend build (Go 1.25-alpine)
3. Tests run (build fails if tests fail)
4. Runtime image (Alpine 3)

### Run Docker Container

```bash
docker run \
  -d --name ldap-password-changer \
  -p 3000:3000 \
  ghcr.io/netresearch/ldap-selfservice-password-changer \
  -ldap-server ldaps://dc1.example.com:636 \
  -base-dn DC=example,DC=com \
  -readonly-user readonly \
  -readonly-password readonly-password \
  -active-directory
```

**Environment Variable Alternative**:

```bash
docker run \
  -d --name ldap-password-changer \
  -p 3000:3000 \
  -e LDAP_SERVER=ldaps://dc1.example.com:636 \
  -e LDAP_BASE_DN=DC=example,DC=com \
  -e LDAP_READONLY_USER=readonly \
  -e LDAP_READONLY_PASSWORD=readonly-password \
  -e LDAP_IS_AD=true \
  ghcr.io/netresearch/ldap-selfservice-password-changer
```

### Self-Signed Certificates

If using self-signed LDAPS certificates:

```bash
docker run \
  -v /etc/ssl/certs:/etc/ssl/certs:ro \
  # ... other arguments
```

## Troubleshooting

### LDAP Connection Issues

**Error**: `could not connect to LDAP server`

**Solutions**:

1. Verify LDAP server URL (ldaps:// for SSL)
2. Check network connectivity: `telnet dc1.example.com 636`
3. Verify certificates if using LDAPS
4. Check firewall rules

### Build Failures

**Error**: `pnpm: command not found`

**Solution**: Enable Corepack

```bash
corepack enable
```

**Error**: `go: module not found`

**Solution**: Download Go dependencies

```bash
go mod download
```

### Hot Reload Not Working

**Issue**: Changes not reflected after file modification

**Solutions**:

1. Check nodemon is watching correct files (package.json:19)
2. Verify file permissions (ensure files are writable)
3. Restart development server: Ctrl+C, then `pnpm dev`

### TypeScript Errors

**Error**: `TS2322: Type 'X' is not assignable to type 'Y'`

**Solution**: TypeScript strict mode enabled (tsconfig.json)

- Fix type errors (no `any` types allowed)
- Add proper type annotations
- Use type guards for nullable values

## Performance Tips

### Development Mode

- **Disable minification**: Already disabled in dev mode
- **Use source maps**: Automatically generated (tsconfig.json:12)
- **Skip tests**: Not run in dev mode (only during build)

### Production Build

- **Enable minification**: `pnpm js:minify` (package.json:14)
- **Enable CSS optimization**: CSSnano active (postcss.config.js)
- **Static build**: CGO_ENABLED=0 for portable binary (Dockerfile:24)

## Related Documentation

- [API Reference](api-reference.md) - JSON-RPC API contracts
- [Architecture Patterns](architecture-patterns.md) - Design decisions
- [Testing Guide](testing-guide.md) - Testing strategies
- [Component Reference](component-reference.md) - Package documentation

---

_Generated by /sc:index on 2025-10-04_
