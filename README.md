<div align="center">
  <h1>GopherPass 🐹</h1>

Self-service password **change & reset** for **Active Directory and LDAP** — written in Go.

> Give users a simple, secure web interface to change or reset their directory passwords — no tickets, no scripts.

  <img src="./internal/web/static/logo.webp" height="256" alt="GopherPass Logo">

[![Check](https://github.com/netresearch/ldap-selfservice-password-changer/actions/workflows/check.yml/badge.svg)](https://github.com/netresearch/ldap-selfservice-password-changer/actions/workflows/check.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/netresearch/ldap-selfservice-password-changer)](https://goreportcard.com/report/github.com/netresearch/ldap-selfservice-password-changer)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![WCAG 2.2 AAA](https://img.shields.io/badge/WCAG%202.2-AAA%20Compliant-brightgreen?style=flat-square)](https://www.w3.org/WAI/WCAG22/quickref/?currentsidebar=%23col_customize&levels=aaa)

</div>

## Features

- **Active Directory & LDAP Support**: Works with both AD and OpenLDAP directory services
- **Dual Mode Operation**: Self-service password change (authenticated) and email-based password reset
- **Configurable Password Policies**: Enforce minimum length, numbers, symbols, uppercase, lowercase requirements
- **Security First**: LDAPS support, cryptographic token generation, rate limiting, no password storage
- **Real-Time Validation**: Client-side validation with immediate feedback on password requirements
- **Accessibility Excellence**: WCAG 2.2 Level AAA compliant with full keyboard navigation and screen reader support
- **Modern UX**: Dark mode support, adaptive density, responsive design, optimized for password managers
- **Production Ready**: Single binary deployment, Docker support, comprehensive logging
- **Developer Friendly**: Go backend, TypeScript frontend, Tailwind CSS, embedded assets

## Quick Start

### Docker (Recommended)

```bash
docker run -d -p 3000:3000 \
  -e LDAP_SERVER=ldaps://dc1.example.com:636 \
  -e LDAP_IS_AD=true \
  -e LDAP_BASE_DN=DC=example,DC=com \
  -e LDAP_READONLY_USER=readonly \
  -e LDAP_READONLY_PASSWORD=readonly \
  -e PASSWORD_RESET_ENABLED=true \
  -e SMTP_HOST=smtp.gmail.com \
  -e SMTP_PORT=587 \
  -e SMTP_USERNAME=notifications@example.com \
  -e SMTP_PASSWORD=your_app_password \
  -e SMTP_FROM_ADDRESS=noreply@example.com \
  -e APP_BASE_URL=https://password.example.com \
  ghcr.io/netresearch/ldap-selfservice-password-changer
```

Access at `http://localhost:3000`

### Native Installation

**Prerequisites:**

- Go 1.25+
- Node.js 24+
- Corepack (`npm i -g corepack`)

```bash
# Clone and build
git clone https://github.com/netresearch/ldap-selfservice-password-changer
cd ldap-selfservice-password-changer
corepack enable
pnpm install
pnpm build

# Configure (create .env.local or use flags)
cp .env.local.example .env.local
# Edit .env.local with your directory server details

# Run
./ldap-selfservice-password-changer
```

## Configuration

GopherPass is configured via environment variables or command-line flags. Key settings:

### Directory Connection

- `LDAP_SERVER` - Directory server URI (ldaps://server:636)
- `LDAP_IS_AD` - Set to `true` for Active Directory
- `LDAP_BASE_DN` - Base DN for user searches
- `LDAP_READONLY_USER` - Service account with read access
- `LDAP_READONLY_PASSWORD` - Service account password

### Password Policy

- `MIN_LENGTH` - Minimum password length (default: 8)
- `MIN_NUMBERS` - Required numeric characters (default: 1)
- `MIN_SYMBOLS` - Required special characters (default: 1)
- `MIN_UPPERCASE` - Required uppercase letters (default: 1)
- `MIN_LOWERCASE` - Required lowercase letters (default: 1)

### Password Reset Feature

- `PASSWORD_RESET_ENABLED` - Enable email-based password reset
- `SMTP_HOST` / `SMTP_PORT` - Mail server configuration
- `SMTP_USERNAME` / `SMTP_PASSWORD` - SMTP authentication
- `SMTP_FROM_ADDRESS` - Sender email address
- `APP_BASE_URL` - Base URL for reset links
- `RESET_TOKEN_EXPIRY_MINUTES` - Token validity (default: 15)
- `RESET_RATE_LIMIT_REQUESTS` - Max requests per window (default: 3)

For complete configuration options, run `./ldap-selfservice-password-changer --help`

## Password Reset Feature

The password reset feature allows users to reset forgotten passwords via secure email-based token verification.

### Key Features

- **Email-Based Verification**: Secure tokens sent via SMTP (Google Workspace supported)
- **Cryptographic Security**: 32-byte tokens generated with `crypto/rand`
- **Rate Limiting**: Configurable limits prevent abuse (default: 3 requests/hour per user)
- **Token Expiration**: Tokens expire after 15 minutes (configurable)
- **Single-Use Tokens**: Tokens cannot be reused after password reset
- **No User Enumeration**: Generic responses prevent account discovery
- **Directory Integration**: Automatic email-to-username lookup

### How It Works

1. User navigates to `/forgot-password` and enters their email
2. System looks up user in directory and generates secure token
3. Reset email sent with link: `https://your-domain.com/reset-password?token=XXX`
4. User clicks link, enters new password with real-time validation
5. Password updated in directory, token marked as used

### LDAP/AD Permissions

**Security Best Practice:** Use dedicated service accounts with minimal permissions.

**For Active Directory:**

- **Read-only account** (`LDAP_READONLY_USER`): Default Users group permissions
- **Reset account** (`LDAP_RESET_USER`, optional): Grant "Reset password" permission on user OU

**For OpenLDAP:**

```ldif
# Read-only access
access to dn.subtree="ou=users,dc=example,dc=com"
    by dn="cn=readonly,dc=example,dc=com" read

# Password reset access (optional dedicated account)
access to attrs=userPassword
    by dn="cn=password-reset,dc=example,dc=com" write
    by self write
    by * auth
```

**Note:** Connection must use LDAPS (`ldaps://`) for all password operations.

## Documentation

Comprehensive documentation is available in the [`docs/`](docs/) directory:

- **[API Reference](docs/api-reference.md)** - JSON-RPC API specification and validation rules
- **[Development Guide](docs/development-guide.md)** - Setup, workflows, and troubleshooting
- **[Testing Guide](docs/testing-guide.md)** - Testing strategies and recommendations
- **[Accessibility Guide](docs/accessibility.md)** - WCAG 2.2 AAA compliance and testing procedures
- **[Architecture](docs/architecture.md)** - System architecture overview

For a complete overview, see the [Documentation Index](docs/README.md).

## Development

### Docker Compose (Recommended)

```bash
# Copy example environment
cp .env.local.example .env.local
# Edit .env.local with your directory server details

# Start development environment with Mailhog
docker compose --profile dev up

# Application: http://localhost:3000
# Mailhog UI: http://localhost:8025 (view password reset emails)
```

### Native Development

```bash
corepack enable
pnpm install
cp .env.local.example .env.local

# Edit .env.local with your settings

# Run with hot-reload
pnpm dev
```

## Project Background

GopherPass was originally developed by [Netresearch DTT GmbH](https://www.netresearch.de/) for Active Directory environments and later expanded to support OpenLDAP. The project emphasizes security, accessibility, and user experience while maintaining simplicity and ease of deployment.

## Contributing

Contributions are welcome! Please:

- Follow [Conventional Commits](https://www.conventionalcommits.org/) for commit messages
- Use `gofmt` and `prettier` formatting standards
- Ensure tests pass and add new tests for features
- Update documentation for significant changes

## License

GopherPass is licensed under the [MIT License](LICENSE).

---

<div align="center">

**Built with ❤️ by [Netresearch DTT GmbH](https://www.netresearch.de/)**

[Documentation](docs/) • [Issues](https://github.com/netresearch/ldap-selfservice-password-changer/issues) • [Contributing](#contributing)

</div>
