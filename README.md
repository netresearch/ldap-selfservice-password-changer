<div align=center>
  <h1>LDAP Selfservice Password Changer</h1>

<span>LDAP Selfservice Password Changer is a web frontend and JSON RPC API for allowing your users to change their own passwords in your LDAP or ActiveDirectory server.</span>

  <img src="./internal/web/static/logo.webp" height="256">

[![WCAG 2.2 AAA](https://img.shields.io/badge/WCAG%202.2-AAA%20Compliant-brightgreen?style=flat-square)](https://www.w3.org/WAI/WCAG22/quickref/?currentsidebar=%23col_customize&levels=aaa)

</div>

## Features

- **Self-Service Password Changes**: Users can change their own LDAP/AD passwords without admin intervention
- **Password Reset via Email**: Users can reset forgotten passwords through secure email-based token verification
- **Configurable Password Policies**: Minimum length, numbers, symbols, uppercase, lowercase requirements
- **Rate Limiting**: Protection against abuse with configurable request limits (3 requests/hour)
- **Real-Time Validation**: Client-side validation with immediate feedback
- **Accessible**: WCAG 2.2 Level AAA compliant, full keyboard navigation, screen reader support, adaptive density
- **Dark Mode**: Three-state theme toggle (light/dark/auto) with 7:1 contrast ratios (AAA)
- **Password Manager Support**: Optimized for autofill with proper autocomplete attributes
- **Secure**: LDAPS support, cryptographic token generation, no password storage, minimal attack surface
- **Single Binary**: All assets embedded, easy deployment
- **Modern Stack**: Go backend, TypeScript frontend, Tailwind CSS

## Documentation

Comprehensive documentation is available in the [`docs/`](docs/) directory:

- **[API Reference](docs/api-reference.md)** - JSON-RPC API specification and validation rules
- **[Development Guide](docs/development-guide.md)** - Setup, workflows, and troubleshooting
- **[Testing Guide](docs/testing-guide.md)** - Testing strategies and recommendations
- **[Accessibility Guide](docs/accessibility.md)** - WCAG 2.2 AAA compliance and testing procedures
- **[Architecture](docs/architecture.md)** - System architecture overview

For a complete overview, see the [Documentation Index](docs/README.md).

## Quick Start

### For Developers

1. Clone the repository
2. Follow the [Development Guide](docs/development-guide.md) for detailed setup
3. Run `pnpm dev` for hot-reload development mode

### For Production

Use our [Docker image](https://github.com/netresearch/ldap-selfservice-password-changer/pkgs/container/ldap-selfservice-password-changer) or build from source.

## Password Reset Feature

The password reset feature allows users to reset forgotten passwords via email-based token verification.

### Key Features

- **Email-Based Verification**: Secure tokens sent via SMTP (Google Workspace supported)
- **Cryptographic Security**: 32-byte tokens generated with `crypto/rand`
- **Rate Limiting**: Configurable limits prevent abuse (default: 3 requests/hour per user)
- **Token Expiration**: Tokens expire after 15 minutes (configurable)
- **Single-Use Tokens**: Tokens cannot be reused after password reset
- **No User Enumeration**: Generic responses prevent account discovery
- **LDAP Integration**: Automatic email-to-username lookup

### How It Works

1. User navigates to `/forgot-password` and enters their email
2. System looks up user in LDAP and generates secure token
3. Reset email sent with link: `https://your-domain.com/reset-password?token=XXX`
4. User clicks link, enters new password with real-time validation
5. Password updated in LDAP, token marked as used

### Configuration

Enable password reset by setting these environment variables:

```bash
# Feature flag
PASSWORD_RESET_ENABLED=true

# Token settings
RESET_TOKEN_EXPIRY_MINUTES=15
RESET_RATE_LIMIT_REQUESTS=3
RESET_RATE_LIMIT_WINDOW_MINUTES=60

# SMTP configuration (Google Workspace example)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=notifications@example.com
SMTP_PASSWORD=your_app_password_here
SMTP_FROM_ADDRESS=noreply@example.com

# Application URL for reset links
APP_BASE_URL=https://password.example.com

# Optional: Dedicated service account for password reset (recommended for security)
# If not set, LDAP_READONLY_USER will be used (backward compatible)
LDAP_RESET_USER=cn=password-reset,dc=example,dc=com
LDAP_RESET_PASSWORD=reset_account_password
```

### LDAP Permissions

**Security Best Practice:** Use a dedicated service account for password reset operations with minimal permissions.

Configure **two separate service accounts** for better security isolation:

1. **LDAP_READONLY_USER** - Read-only access for authentication (change-password operations)
2. **LDAP_RESET_USER** - Write access ONLY for password reset operations (optional, falls back to LDAP_READONLY_USER if not set)

**Active Directory Permissions:**

For `LDAP_READONLY_USER` (required):

- Read access to user objects (default Users group permission)

For `LDAP_RESET_USER` (optional, recommended):

1. Open Active Directory Users and Computers
2. Right-click on the OU containing users → Properties → Security → Advanced
3. Add dedicated reset account with **"Reset password" permission** ONLY

**OpenLDAP Permissions:**

For `LDAP_READONLY_USER` (required):

```ldif
access to dn.subtree="ou=users,dc=example,dc=com"
    by dn="cn=readonly,dc=example,dc=com" read
```

For `LDAP_RESET_USER` (optional, recommended):

```ldif
access to attrs=userPassword
    by dn="cn=password-reset,dc=example,dc=com" write
    by self write
    by * auth
```

**Security Notes:**

- Connection must use LDAPS (`ldaps://`) for all password operations
- Dedicated reset account provides better security isolation and audit trails
- If `LDAP_RESET_USER` is not configured, `LDAP_READONLY_USER` needs both read and password reset permissions (less secure)

### API Methods

The password reset feature adds two new JSON-RPC methods:

- `request-password-reset` - Initiates password reset, sends email with token
- `reset-password` - Completes password reset with valid token

See [API Reference](docs/api-reference.md) for detailed specifications.

## Running

### Natively

If you want to run this service without a Docker container, you have to build it yourself.

Prerequisites:

- Go 1.25+
- Node.js 24+
- Corepack (`npm i -g corepack`)

You can configure this via a `.env.local` file or via command options (for more information you can run `./ldap-selfservice-password-changer --help`).

<!-- Multiline comment idea taken from https://stackoverflow.com/a/12797512 -->

```bash
corepack enable
pnpm i
pnpm build

./ldap-selfservice-password-changer \
  `# You can also configure these via environment variables,` \
  `# please see the .env file for available options.` \
  -ldap-server ldaps://dc1.example.com:636 -active-directory \
  -readonly-password readonly -readonly-user readonly \
  -base-dn DC=example,DC=com
```

### Docker Compose (Recommended for Development)

The easiest way to run the application for development and testing with email support:

```bash
# Copy example environment file
cp .env.local.example .env.local
# Edit .env.local with your LDAP server details

# Start with dev profile (includes Mailhog for email testing)
docker compose --profile dev up

# Application: http://localhost:3000
# Mailhog Web UI: http://localhost:8025 (view sent emails)
```

**What's included:**

- Application with hot-reload support
- Mailhog SMTP server for email testing (no real emails sent)
- Mailhog Web UI to view password reset emails
- Automatic service networking

**Available profiles:**

- `dev` - Development mode with Mailhog
- `test` - Testing mode with Mailhog

### Docker (Production)

We have a Docker image available [here](https://github.com/netresearch/ldap-selfservice-password-changer/pkgs/container/ldap-selfservice-password-changer).

You can ignore the warning that the service could not load a `.env` file.

<!-- Multiline comment idea taken from https://stackoverflow.com/a/12797512 -->

```bash
docker run \
  `# Run the password-changer container detached from the current terminal` \
  -d --name ldap-password-changer \
  `# You might want to mount your host SSL certificate directory,` \
  `# if you have a self-signed certificate for your LDAPS connection` \
  -v /etc/ssl/certs:/etc/ssl/certs:ro \
  -p 3000:3000 \
  `# LDAP Configuration` \
  -e LDAP_SERVER=ldaps://dc1.example.com:636 \
  -e LDAP_IS_AD=true \
  -e LDAP_BASE_DN=DC=example,DC=com \
  -e LDAP_READONLY_USER=readonly \
  -e LDAP_READONLY_PASSWORD=readonly \
  `# Password Reset Configuration (optional)` \
  -e PASSWORD_RESET_ENABLED=true \
  -e SMTP_HOST=smtp.gmail.com \
  -e SMTP_PORT=587 \
  -e SMTP_USERNAME=notifications@example.com \
  -e SMTP_PASSWORD=your_app_password \
  -e SMTP_FROM_ADDRESS=noreply@example.com \
  -e APP_BASE_URL=https://password.example.com \
  `# Optional: Dedicated reset account for better security isolation` \
  -e LDAP_RESET_USER=cn=password-reset,dc=example,dc=com \
  -e LDAP_RESET_PASSWORD=reset_password \
  ghcr.io/netresearch/ldap-selfservice-password-changer
```

**Alternative with command-line flags:**

```bash
docker run -d --name ldap-password-changer \
  -v /etc/ssl/certs:/etc/ssl/certs:ro \
  -p 3000:3000 \
  ghcr.io/netresearch/ldap-selfservice-password-changer \
  -ldap-server ldaps://dc1.example.com:636 -active-directory \
  -readonly-password readonly -readonly-user readonly \
  -base-dn DC=example,DC=com
```

### Health Checks

The Docker image provides HTTP health checking on port 3000:

**Docker:**

```bash
docker run \
  --health-cmd "exit 0" \
  --health-interval=30s \
  --health-timeout=3s \
  --health-retries=3 \
  # ... other flags
```

**Docker Compose:**

```yaml
services:
  ldap-password-changer:
    image: ghcr.io/netresearch/ldap-selfservice-password-changer
    healthcheck:
      test: ["CMD-SHELL", "exit 0"]
      interval: 30s
      timeout: 3s
      retries: 3
```

**Kubernetes:**

```yaml
livenessProbe:
  httpGet:
    path: /
    port: 3000
  initialDelaySeconds: 10
  periodSeconds: 30
```

### Debugging

The Docker image uses a minimal `scratch` base for security and size optimization:

**Characteristics:**

- ✅ Smaller image size (~12MB vs ~30MB)
- ✅ Reduced attack surface (no OS, shell, or utilities)
- ✅ Runs as non-root user (UID 65534)
- ❌ No shell available (cannot `docker exec -it container /bin/sh`)

**Debugging methods:**

- View logs: `docker logs <container-name>`
- Check application: `curl http://localhost:3000/`
- Application shows detailed errors via JSON-RPC responses

## Developing

### Option 1: Docker Compose (Recommended)

**Easiest setup with Mailhog for email testing:**

```bash
# Copy example environment file
cp .env.local.example .env.local
# Edit .env.local with your LDAP server details

# Start development environment
docker compose --profile dev up

# Application runs on http://localhost:3000
# Mailhog Web UI on http://localhost:8025 (view password reset emails)
```

**Benefits:**

- No local Go/Node.js installation required
- Mailhog included for email testing
- Consistent development environment
- Easy to test password reset emails

### Option 2: Native Development

Prerequisites:

- Go 1.25+
- Node.js 24+
- Corepack (`npm i -g corepack`)

```bash
corepack enable

# Install dependencies
pnpm i

# Copy and configure environment
cp .env.local.example .env.local
# Edit .env.local with your settings:
# - LDAP_SERVER, LDAP_BASE_DN, LDAP_READONLY_USER, LDAP_READONLY_PASSWORD
# - For password reset: PASSWORD_RESET_ENABLED=true, SMTP_*, APP_BASE_URL
# - For email testing: Use Mailhog (SMTP_HOST=localhost, SMTP_PORT=1025)

# Option A: Run Mailhog locally for email testing
docker run -d -p 1025:1025 -p 8025:8025 mailhog/mailhog:v1.0.1
# Mailhog Web UI: http://localhost:8025

# Running normally
pnpm start

# Running in dev mode (auto-restart on changes)
pnpm dev
```

## License

LDAP Selfservice Password Changer is licensed under the MIT license, for more information please refer to the [included LICENSE file](LICENSE).

## Contributing

Feel free to contribute by creating a Pull Request!

This project uses [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) for commit messages and the default `gofmt` and `prettier` formatting rules.
