# Development Guide

## Prerequisites

### Required Tools

- **Go**: 1.26+ (tested with 1.26.0)
- **Bun**: no version pinned; CI installs the latest — runs the TypeScript and Tailwind toolchain
- **Git**: For version control

Bun and Go are installed **separately**, each with its own installer or package
manager. `package.json` carries no `packageManager` or `engines` pin, so there is
no shim or bootstrapper to enable first — install Bun, then run `bun install`.

### Optional Tools

- **Docker** + Compose: for the local OpenLDAP + Mailpit stack (see
  [Running the Full Stack Locally](#running-the-full-stack-locally))
- **air**: Go hot-reload used by `bun run dev`. Not a bun dependency — install it
  yourself: `go install github.com/air-verse/air@latest`
- **golangci-lint**: Go linting (CI runs it regardless)

### System Requirements

- Linux, macOS, or Windows with WSL2
- 2GB RAM minimum (4GB recommended)
- LDAP/ActiveDirectory server for testing (or the Compose dev stack)

## Initial Setup

### 1. Clone Repository

```bash
git clone https://github.com/netresearch/ldap-selfservice-password-changer.git
cd ldap-selfservice-password-changer
```

### 2. Install Dependencies

```bash
bun install
```

**What this does**:

- Installs the TypeScript compiler
- Installs Tailwind CSS and the PostCSS CLI
- Installs the linting and formatting tools (ESLint, Prettier + plugins)

**Go dependencies** are handled automatically by Go modules.

For a one-shot setup that also installs the git hooks:

```bash
make setup   # bun install + go mod download + git hooks
```

### 3. Configure LDAP Connection

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

### 4. Verify Setup

```bash
bun run build
./ldap-selfservice-password-changer --help
```

**Expected output**: Help text showing all available flags

## Development Workflows

### Development Mode (Hot Reload)

**Command**:

```bash
bun run dev
```

**What happens**:

1. Builds initial assets (`bun run build:assets` — TypeScript + CSS)
2. Starts three concurrent watchers:
   - **TypeScript watcher**: `tsc -w` (rebuilds on .ts changes)
   - **CSS watcher**: `postcss -w` (rebuilds on .css changes)
   - **Go watcher**: `air` (rebuilds and restarts the server on any file change)

**Output Example**:

```
15:32:41 - Starting compilation in watch mode...
Rebuilding...
building...
running...
Server listening on :3000
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
bun run build
```

**Steps**:

1. Compile TypeScript to JavaScript (`tsc`)
2. Build CSS with PostCSS + Tailwind
3. Compile the Go binary with all assets embedded

There is no separate minification step — no minifier is configured.

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
# Watch mode
bun run js:dev

# One-shot compile (doubles as the type check)
bun run js:build
```

**Output**: `internal/web/static/js/app.js`, `internal/web/static/js/validators.js`

#### CSS Only

```bash
# Watch mode
bun run css:dev

# One-shot build
bun run css:build
```

**Output**: `internal/web/static/styles.css`

#### Both at Once

```bash
bun run build:assets
```

#### Go Only

```bash
# Run from source
go run .

# Build binary
go build
```

The `bun run start` and `bun run build` scripts wrap these and rebuild the
embedded assets first — prefer them unless the assets are already current.

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
bunx prettier --write .

# Check formatting without changes
bunx prettier --check .
```

`make format` and `make format-check` wrap these.

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

### Code Quality & Linting

The project uses comprehensive linting and code quality tools for both Go and TypeScript/JavaScript.

#### Go Linting (golangci-lint)

**Install golangci-lint** (optional for local development):

```bash
# macOS
brew install golangci-lint

# Linux
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# Or use CI only (golangci-lint runs automatically in GitHub Actions)
```

**Run linting**:

```bash
# Full lint with all checks
golangci-lint run

# Fast lint (skips some slower analyzers)
golangci-lint run --fast

# Fix auto-fixable issues
golangci-lint run --fix
```

**Configuration**: `.golangci.yml` - 60+ enabled linters including:

- **Security**: gosec (G\* checks)
- **Bugs**: staticcheck (SA\* checks), ineffassign, unused
- **Style**: revive, stylecheck
- **Performance**: perfsprint, prealloc
- **Complexity**: gocyclo (max 15), dupl

**Pre-commit**: Runs automatically via the repo's git hook (`githooks/pre-commit`)

#### TypeScript/JavaScript Linting (ESLint)

**Run linting**:

```bash
# Check for issues
bun run lint

# Auto-fix issues
bun run lint:fix
```

**Configuration**: `eslint.config.js` - Modern flat config with:

- **TypeScript**: Strict type-checked rules
- **Style**: Consistent code patterns
- **Security**: No unsafe operations
- **Best practices**: Nullish coalescing, optional chaining

**Known Issues**: ~60 existing linting issues identified for gradual cleanup

**Pre-commit**: Runs automatically via the repo's git hook (warnings only for now)

#### Code Coverage

**Run tests with coverage**:

```bash
# Generate coverage report
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

# View coverage in browser
go tool cover -html=coverage.out
```

**CI Integration**: Coverage automatically uploaded to Codecov on PRs

**Target**: 80% minimum coverage threshold, enforced twice — in CI by
`go-check.yml` (which fails the build outright) and by Codecov's project and
patch statuses, configured in `.github/codecov.yml`. Total coverage currently
sits above 90%.

#### Pre-commit Hooks

The hooks live in `githooks/` and are copied into `.git/hooks/` — there is no
hook manager and no `postinstall` step, so a fresh clone has no hooks until you
install them:

```bash
make hooks   # cp githooks/* .git/hooks/ && chmod +x …
```

**What `githooks/pre-commit` runs**:

1. ✅ `bunx prettier --check .` (blocking)
2. ✅ `bun run js:build` — TypeScript type check (blocking)
3. ⚠️ `bun run lint` — ESLint (warning only)
4. ⚠️ `golangci-lint run` if installed, else `go vet ./...` (vet is blocking)
5. ✅ `go test -short ./...` (blocking)

**Bypassing checks** (emergency only):

```bash
git commit --no-verify -m "emergency fix"
```

**Updating hooks**: re-run `make hooks` after `githooks/` changes — the copies in
`.git/hooks/` do not update themselves.

#### Dependency Auditing

```bash
bun audit                        # all advisories
bun audit --audit-level=high     # high and critical only
bun audit --json                 # machine-readable
```

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
├── dev/                             # Local Compose dev stack fixtures
│   ├── seed.ldif                    # Seeded OpenLDAP users
│   └── setup-acl.sh                 # Self-service + reset ACLs
│
├── githooks/                        # Git hooks (installed via `make hooks`)
│   ├── pre-commit
│   └── commit-msg
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
├── package.json                     # Frontend toolchain dependencies
├── bun.lock                         # Bun lock file
├── tsconfig.json                    # TypeScript configuration
├── postcss.config.js                # PostCSS configuration
├── .air.toml                        # air (Go hot-reload) configuration
├── .prettierrc.mjs                  # Prettier configuration
├── .prettierignore                  # Prettier ignore patterns
├── Makefile                         # Task shortcuts
├── compose.yml                      # Local dev/test stack (OpenLDAP, Mailpit, app)
├── Dockerfile                       # Binary-selector image build
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

**Backend** - `internal/rpchandler/change_password.go`:

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

## Running the Full Stack Locally

For review and manual testing you want the whole thing: the app plus a seeded
OpenLDAP directory plus an SMTP sink that catches reset mail. The `dev` Compose
profile provides that.

### The Dockerfile Is a Binary-Selector

Read this before running `docker build` or `docker compose --build`.

The production `Dockerfile` does **no** `go build` and **no** `bun install`. It
`COPY`s pre-built binaries out of `bin/` — produced by the release CI's
cross-compile matrix — and picks the right one per `TARGETARCH`. From a clean
checkout `bin/` is empty, so the build fails with:

```
COPY bin/ldap-selfservice-password-changer-linux-* /tmp/: lstat /bin: no such file or directory
```

That is not a broken Dockerfile. It means you have to build the binary yourself
first — which is exactly what Route A does.

Two ways to get a running instance. Prefer **Route A**: everything sits on the
Compose network, so the app can reach Mailpit's internal SMTP.

### Route A: Build the Binary, Then Compose (recommended)

The frontend assets are embedded via `go:embed`, so build the assets _before_ the
Go binary. Cross-compile into the exact filename the Dockerfile expects
(`bin/<repo>-linux-<arch>`):

```bash
ARCH=$(go env GOARCH)                       # amd64 on WSL2/x86_64
bun install --frozen-lockfile
bun run build:assets                        # writes internal/web/static/{styles.css,js/*.js}
CGO_ENABLED=0 GOOS=linux GOARCH=$ARCH go build -trimpath \
  -ldflags="-w -s -X main.version=vX.Y.Z-rc -X main.build=$(git rev-parse --short HEAD)" \
  -o bin/ldap-selfservice-password-changer-linux-$ARCH .

# Pick non-default host ports to avoid collisions; the app container listens on 3000.
APP_PORT=3140 MAILPIT_WEB_PORT=8125 docker compose --profile dev up --build -d
```

The `dev` profile brings up `openldap` → `openldap-init` (seeds users, exits 0) →
`mailpit` → `app`.

- App: `http://localhost:3140` (change password) and `/forgot-password`
- Mailpit (catches reset emails): `http://localhost:8125`
- Health: `curl -s -o /dev/null -w '%{http_code}' http://localhost:3140/health/live` → `200`

**Container names are fixed** (`gopherpass-openldap`, `gopherpass-mailpit`,
`gopherpass-app`) and therefore _not_ project-scoped, so a stale container from an
earlier run collides. Clear them and retry:

```bash
docker rm -f gopherpass-openldap gopherpass-mailpit gopherpass-app
```

**`APP_BASE_URL` must move with `APP_PORT`.** `compose.yml` pins
`APP_BASE_URL: "http://localhost:3000"` in the `app` service's inline
`environment:` block, and inline values win over `.env.local`. Change only
`APP_PORT` and the app serves on `:3140` while every reset email it sends links
to `:3000`, where nothing is listening — the mail arrives and the link is dead.
Edit the `APP_BASE_URL` line in `compose.yml` to match the port you chose.

### Exercising Custom Email Templates

`EMAIL_TEMPLATE_HTML` and `EMAIL_TEMPLATE_TEXT` take paths that must resolve
**inside the app container**. The runtime image is `FROM scratch` and the `app`
service declares no `volumes:`, so a path that exists only on the host resolves
to nothing and startup fails with `text template: stat …: no such file or
directory`. Bind-mount the file and point the variable at the container path:

```yaml
# compose.yml, under the app service
volumes:
  - ./dev/reset.txt.tmpl:/config/email/reset.txt.tmpl:ro
environment:
  EMAIL_TEMPLATE_TEXT: /config/email/reset.txt.tmpl
```

Template fields are `{{.ResetLink}}`, `{{.Token}}`, `{{.BaseURL}}`,
`{{.Recipient}}` and `{{.ExpiryMinutes}}`. Leaving one of the two body templates
unset is fine — the unset side falls back to the template embedded in the binary,
which is a useful way to compare a custom body against the default in one message.

### Route B: Native Binary Against Compose Infra

Only if you don't want to rebuild the image. Note that Mailpit's SMTP port (1025)
is **internal-only** — it is deliberately not host-mapped — so a natively running
app **cannot send reset mail**. Use Route A if you need to exercise the reset
email. LDAP (389) and the Mailpit web UI _are_ host-mapped. The command below sets
no `MAILPIT_WEB_PORT`, so compose falls back to the default
(`"${MAILPIT_WEB_PORT:-8025}:8025"`) and the UI is on `http://localhost:8025`.
Prefix the command with `MAILPIT_WEB_PORT=<port>` to move it (Route A uses 8125).

```bash
docker compose up -d openldap openldap-init mailpit
# Assets are go:embed'd, so build them before `go run .` (a clean checkout
# otherwise fails with "pattern *.css: no matching files found").
bun install --frozen-lockfile
bun run build:assets
go run . -ldap-server ldap://127.0.0.1:389 -base-dn dc=netresearch,dc=local \
  -readonly-user cn=admin,dc=netresearch,dc=local -readonly-password admin -port 39443
```

### Seeded Users (`dev/seed.ldif`)

The stack's password policy is ≥10 chars with 1 number, 1 symbol, 1 uppercase and
1 lowercase.

| uid                             | mail                         | password         |
| ------------------------------- | ---------------------------- | ---------------- |
| `jdoe`                          | john.doe@netresearch.local   | `password`       |
| `jsmith`                        | jane.smith@netresearch.local | `password`       |
| `password-reset` (service acct) | —                            | `reset-password` |

### Driving a Reset by Username

This exercises `RESET_IDENTIFIER_MODE`. Set it to `both` via `.env.local` in the
worktree and recreate `app`. This works because `RESET_IDENTIFIER_MODE` is _not_
in the `app` service's inline `environment:` block — Compose's `env_file` only
supplies variables that block does not already define. A variable listed inline
always wins over `.env.local`, so appending e.g. `SMTP_HOST` there has no effect;
to change one of those, edit `compose.yml` instead. Then submit a **username** and
confirm Mailpit received mail addressed to the account's **registered** address,
not the typed identifier:

```bash
printf 'RESET_IDENTIFIER_MODE=both\n' >> .env.local  # append — never clobber an existing .env.local
APP_PORT=3140 MAILPIT_WEB_PORT=8125 docker compose --profile dev up -d --force-recreate app
curl -s -X DELETE http://localhost:8125/api/v1/messages
curl -s -X POST http://localhost:3140/api/rpc -H 'Content-Type: application/json' \
  -d '{"method":"request-password-reset","params":["jdoe"]}'
curl -s http://localhost:8125/api/v1/messages | \
  python3 -c "import json,sys;[print(m['To'][0]['Address'],'|',m['Subject']) for m in json.load(sys.stdin)['messages']]"
# → john.doe@netresearch.local | Password Reset Request   (NOT "jdoe")
```

### Teardown

```bash
docker compose --profile dev down -v      # or: docker rm -f gopherpass-{app,mailpit,openldap}
```

`make docker-up` / `make docker-down` wrap the plain up/down for the `dev`
profile; `make docker-logs` tails them.

## Docker Development

### Build the Image

`docker build` consumes pre-built binaries from `bin/` — see
[The Dockerfile Is a Binary-Selector](#the-dockerfile-is-a-binary-selector) above
for why, and build them first:

```bash
docker build -t ldap-password-changer .
```

**Build process**:

1. `binary-selector` stage (Alpine): picks `bin/…-linux-<arch>` for the target arch
2. Runtime stage: `scratch` — the binary plus the CA bundle, running as UID 65534

`make build-docker` wraps this.

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

**Error**: `bun: command not found`

**Solution**: Bun is installed separately — there is no bootstrapper in this repo.
Install it from [bun.com/docs/installation](https://bun.com/docs/installation) and
make sure `~/.bun/bin` is on your `PATH`.

**Error**: `go: module not found`

**Solution**: Download Go dependencies

```bash
go mod download
```

**Error**: `pattern *.css: no matching files found`

**Solution**: The frontend assets are embedded via `go:embed`, so they must exist
before the Go build. Run `bun run build:assets` first (or use `bun run build`,
which does it for you).

**Error**: `COPY bin/… lstat /bin: no such file or directory` during
`docker build` / `docker compose --build`

**Solution**: Expected on a clean checkout — the Dockerfile only selects
pre-built binaries. See
[The Dockerfile Is a Binary-Selector](#the-dockerfile-is-a-binary-selector).

### Hot Reload Not Working

**Issue**: Changes not reflected after file modification

**Solutions**:

1. Verify `air` is installed and on `PATH` (`go install github.com/air-verse/air@latest`)
2. Check `air` is watching the right paths (`.air.toml`)
3. Verify file permissions (ensure files are writable)
4. Restart development server: Ctrl+C, then `bun run dev`

### TypeScript Errors

**Error**: `TS2322: Type 'X' is not assignable to type 'Y'`

**Solution**: TypeScript strict mode enabled (tsconfig.json)

- Fix type errors (no `any` types allowed)
- Add proper type annotations
- Use type guards for nullable values

## Performance Tips

### Development Mode

- **Use source maps**: Automatically generated (tsconfig.json)
- **Skip tests**: Not run in dev mode (the pre-commit hook and CI run them)
- **Rebuild only what changed**: `bun run js:build` or `bun run css:build` instead
  of the full `bun run build:assets`

### Production Build

- **Static binary**: `CGO_ENABLED=0` for a portable, scratch-image-compatible
  binary — see the Route A build command above
- **Strip debug info**: `-trimpath -ldflags="-w -s"`, as the release build does

## Related Documentation

- [API Reference](api-reference.md) - JSON-RPC API contracts
- [Architecture Patterns](architecture-patterns.md) - Design decisions
- [Testing Guide](testing-guide.md) - Testing strategies
- [Component Reference](component-reference.md) - Package documentation

---

_Generated by /sc:index on 2025-10-04_
