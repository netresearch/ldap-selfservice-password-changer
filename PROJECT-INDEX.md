# LDAP Selfservice Password Changer - Project Index

**Complete project knowledge base and documentation map.**

---

## 🎯 Quick Navigation by Role

### 👨‍💻 **For Developers**
1. Start: [Development Guide](docs/development-guide.md#initial-setup)
2. Code: [Code Structure Reference](docs/code-structure.md)
3. API: [JSON-RPC API Reference](docs/api-reference.md)
4. Test: [Testing Guide](docs/testing-guide.md)

### 🚀 **For DevOps/SRE**
1. Deploy: [Deployment Guide](docs/deployment.md)
2. Config: [Environment Variables](docs/development-guide.md#configuration-reference)
3. Monitor: [Architecture Overview](docs/architecture.md)
4. Security: [Security Documentation](docs/security.md)

### ♿ **For Accessibility/QA**
1. WCAG: [Accessibility Guide](docs/accessibility.md)
2. Test: [Testing Procedures](docs/testing-guide.md)
3. Validate: [WCAG Compliance Matrix](docs/accessibility.md#wcag-22-compliance-matrix)

### 🏗️ **For Architects**
1. Design: [Architecture Overview](docs/architecture.md)
2. Decisions: [Architecture Decision Records](docs/adr/)
3. Security: [Security Architecture](docs/security.md)
4. Code: [Package Structure](docs/code-structure.md)

---

## 📚 Complete Documentation Map

### Core Documentation (`docs/`)

| Document | Purpose | Audience |
|----------|---------|----------|
| [README](docs/README.md) | Documentation index | All |
| [API Reference](docs/api-reference.md) | JSON-RPC API specification | Developers, Integrators |
| [Development Guide](docs/development-guide.md) | Setup and workflows | Developers |
| [Testing Guide](docs/testing-guide.md) | Test strategies and coverage | Developers, QA |
| [Accessibility Guide](docs/accessibility.md) | WCAG 2.2 AAA compliance | Accessibility, QA |
| [Architecture](docs/architecture.md) | System design overview | Architects, Developers |
| [Code Structure](docs/code-structure.md) | Internal package reference | Developers |
| [Deployment](docs/deployment.md) | Production deployment | DevOps, SRE |
| [Security](docs/security.md) | Security architecture | Security, DevOps |

### Architecture Decision Records (`docs/adr/`)

| ADR | Title | Status | Date |
|-----|-------|--------|------|
| [ADR-0001](docs/adr/0001-standardize-form-field-names.md) | Standardize Form Field Names | ✅ Accepted | 2024-10-06 |
| [ADR-0002](docs/adr/0002-password-reset-functionality.md) | Password Reset Functionality | ✅ Accepted | 2024-10-07 |

### Analysis & Planning (`claudedocs/`)

**Project Context**
- [Project Context Snapshot](claudedocs/project-context-2025-10-04.md)
- [Component Reference](claudedocs/component-reference.md)
- [Architecture Patterns](claudedocs/architecture-patterns.md)

**Feature Implementation**
- [Password Reset PRD](claudedocs/password-reset-prd.md)
- [Implementation Workflow](claudedocs/password-reset-implementation-workflow.md)
- [Phase 1 Complete](claudedocs/phase1-implementation-complete.md)

**Quality & Validation**
- [Validation Report](claudedocs/VALIDATION-REPORT.md)
- [WCAG 2.2 Analysis](claudedocs/wcag-2.2-analysis-2025-10-07.md)
- [WCAG Contrast Analysis](claudedocs/wcag-contrast-analysis.md)
- [Accessibility Redesign Plan](claudedocs/accessibility-redesign-plan.md)

**Testing & Guides**
- [Testing Guide](claudedocs/testing-guide.md)
- [Density Toggle Testing](claudedocs/density-toggle-testing-guide.md)
- [Maintenance Guide](claudedocs/MAINTENANCE.md)
- [Onboarding Checklist](claudedocs/onboarding-checklist.md)

---

## 🔍 Project Overview

### What This Project Does

LDAP Selfservice Password Changer provides:

1. **Self-Service Password Changes** - Authenticated users change their LDAP/AD passwords
2. **Password Reset via Email** - Unauthenticated password recovery with secure tokens
3. **Accessible Web Interface** - WCAG 2.2 AAA compliant with adaptive themes
4. **JSON-RPC API** - Programmatic integration for custom frontends

### Key Features

✅ **Security**: LDAPS, rate limiting, cryptographic tokens, minimal attack surface
✅ **Accessibility**: WCAG 2.2 AAA, screen reader support, keyboard navigation, adaptive density
✅ **Modern UX**: Dark mode, responsive design, real-time validation, password manager support
✅ **Developer Friendly**: Single binary, embedded assets, comprehensive tests, hot reload

### Technology Stack

| Layer | Technology | Version |
|-------|------------|---------|
| Backend | Go | 1.25+ |
| Web Framework | Fiber | v2.52+ |
| Frontend | TypeScript | 5.9+ |
| CSS | Tailwind CSS | v4.1+ |
| Build | pnpm | 10.18+ |
| Testing | testcontainers | Latest |

---

## 🏗️ Code Structure

```
ldap-selfservice-password-changer/
├── internal/              # Internal packages (not exported)
│   ├── email/             # SMTP service for password reset emails
│   ├── options/           # Application configuration
│   ├── ratelimit/         # Rate limiting middleware
│   ├── resettoken/        # Token generation and storage
│   ├── rpc/               # JSON-RPC handlers
│   ├── validators/        # Password validation rules
│   └── web/               # Web server and static assets
│       ├── static/        # Compiled JS, CSS, icons
│       │   └── js/        # TypeScript sources
│       └── templates/     # Go html/template components
│           ├── atoms/     # Basic UI elements
│           └── molecules/ # Composite components
├── docs/                  # Official documentation
│   └── adr/               # Architecture Decision Records
├── claudedocs/            # Analysis and planning documents
├── main.go                # Application entry point
├── go.mod                 # Go dependencies
├── package.json           # Node.js dependencies
├── tsconfig.json          # TypeScript configuration
└── compose.yml            # Docker Compose setup
```

**See [Code Structure Documentation](docs/code-structure.md) for detailed package descriptions.**

---

## 🚀 Quick Start

### Development Setup (5 minutes)

```bash
# 1. Clone repository
git clone https://github.com/netresearch/ldap-selfservice-password-changer.git
cd ldap-selfservice-password-changer

# 2. Install dependencies
pnpm install

# 3. Copy environment template
cp .env.local.example .env.local

# 4. Start development server with hot reload
pnpm dev
```

Server runs on `http://localhost:3000` (default)

**Full setup guide**: [Development Guide - Initial Setup](docs/development-guide.md#initial-setup)

### Production Deployment

```bash
# Using Docker
docker pull ghcr.io/netresearch/ldap-selfservice-password-changer:latest
docker run -p 3000:3000 --env-file .env ldap-selfservice-password-changer

# Or build from source
pnpm build
go build -o ldap-selfservice-password-changer
./ldap-selfservice-password-changer
```

**Full deployment guide**: [Deployment Documentation](docs/deployment.md)

---

## 🧪 Testing

**Run all tests**:
```bash
go test ./... -cover
```

**Current coverage**:
- ✅ validators: 100.0%
- ✅ ratelimit: 72.3%
- ✅ resettoken: 71.7%
- ✅ rpc: 45.6%
- ✅ email: 31.2%

**See [Testing Guide](docs/testing-guide.md) for comprehensive testing documentation.**

---

## 🔐 Security

### Security Features

- **LDAPS Support**: Encrypted LDAP connections
- **Rate Limiting**: 3 requests/hour per IP for reset requests
- **Cryptographic Tokens**: 256-bit secure token generation
- **No Password Storage**: Passwords never persisted
- **Input Validation**: Client and server-side validation
- **CSRF Protection**: Token-based CSRF prevention

**See [Security Documentation](docs/security.md) for threat model and security architecture.**

---

## ♿ Accessibility

**WCAG 2.2 Level AAA Compliant**

✅ 7:1 contrast ratios (AAA)
✅ Adaptive density modes (comfortable/compact)
✅ Full keyboard navigation
✅ Screen reader optimized
✅ System preference detection (theme, motion, contrast)

**See [Accessibility Guide](docs/accessibility.md) for compliance matrix and testing procedures.**

---

## 📖 Additional Resources

### Official Links
- **Repository**: https://github.com/netresearch/ldap-selfservice-password-changer
- **Docker Image**: https://github.com/netresearch/ldap-selfservice-password-changer/pkgs/container/ldap-selfservice-password-changer
- **License**: [MIT License](LICENSE)

### External Documentation
- [Go Fiber Documentation](https://docs.gofiber.io/)
- [Tailwind CSS v4 Docs](https://tailwindcss.com/docs)
- [WCAG 2.2 Guidelines](https://www.w3.org/WAI/WCAG22/quickref/)
- [simple-ldap-go](https://github.com/netresearch/simple-ldap-go)

### Contributing
See [README.md](README.md) for contributing guidelines.

---

## 📝 Document Maintenance

**Last Updated**: 2025-10-08
**Maintained By**: Development Team
**Update Frequency**: Per release + major changes

**To update this index**: Add new documents to appropriate section with description and audience.

---

_For questions or suggestions about documentation, open an issue on GitHub._
