# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

---

## [v1.4.0] - 2026-07-21

### Added

- **Password reset by username or email** — new `RESET_IDENTIFIER_MODE` (`email`, the default; `username`; or `both`) lets users request a reset with a username when a shared email address is ambiguous (Active Directory permits non-unique `mail`). Reset links are always sent to the account's registered email address; per-IP, per-typed-identifier, and per-resolved-account rate limits apply ([#620](https://github.com/netresearch/ldap-selfservice-password-changer/pull/620))
- **Graceful shutdown** on SIGINT/SIGTERM so in-flight requests complete before the process exits ([#622](https://github.com/netresearch/ldap-selfservice-password-changer/pull/622))
- **Live password-policy checklist** and a linked, screen-reader-friendly error summary on the password-change form
- **Self-contained Docker dev stack** (OpenLDAP + Mailpit) via Compose profiles

### Changed

- CSS build reduced to `@tailwindcss/postcss` (Lightning CSS handles nesting, prefixing, and minification); removed `autoprefixer`, `postcss-nested`, and `cssnano` ([#622](https://github.com/netresearch/ldap-selfservice-password-changer/pull/622))
- Frontend modernized: `Element.replaceChildren()`, ESLint flat-config `defineConfig`, stricter `tsconfig` (`verbatimModuleSyntax`, `erasableSyntaxOnly`)
- Go: adopted 1.26 idioms (`slices` helpers, `min`) and `testing/synctest` for deterministic time-based tests
- Release pipeline: per-archive SBOMs, cosign signing, and SLSA build provenance/attestation

### Fixed

- Password reset guards against an LDAP client returning a nil user (previously would panic) ([#620](https://github.com/netresearch/ldap-selfservice-password-changer/pull/620))
- `bun run lint` no longer breaks under TypeScript 7 — TypeScript pinned to 6.0.x until `typescript-eslint` supports TS 7

### Security

- Resolved transitive advisory [GHSA-jxxr-4gwj-5jf2](https://github.com/advisories/GHSA-jxxr-4gwj-5jf2) (`brace-expansion`) via a package override

### Dependencies

- `gofiber/fiber/v3` → v3.4.0
- `netresearch/simple-ldap-go` → v1.12.0
- `valyala/fasthttp` → v1.72.0
- `alpine` → 3.24.1
- Routine Renovate/Dependabot updates across Go, Bun, and Docker

---

## [v1.3.0] - 2026-04-16

### Fixed

- **Password change bug**: Upgrade `simple-ldap-go` v1.9.0 → v1.10.0 — fixes `ChangePasswordForSAMAccountNameContext` passing empty username to credential creation, causing all password changes to fail (NRS-4340)
- Consistent `ValidateSAMAccountName` input validation across all sAMAccountName entrypoints (prevents LDAP injection)
- Exclude `node_modules` from golangci-lint (`linters.exclusions.paths`)

### Changed

- **Migrate from pnpm to Bun** — faster installs, resolves broken `pnpm audit` (npm retired audit API endpoint)
- All CI workflows, Dockerfile, Makefile, githooks updated for Bun

### Dependencies

- `simple-ldap-go` v1.9.0 → v1.10.0
- `golang.org/x/crypto` v0.49.0 → v0.50.0
- `golang.org/x/text` v0.35.0 → v0.36.0
- `golang.org/x/net` v0.52.0 → v0.53.0
- `go.opentelemetry.io/otel` v1.42.0 → v1.43.0
- All Node dependencies upgraded to latest

---

### Added

- **GopherPass Branding**: Introduced GopherPass as the public-facing name across all user touchpoints
- Comprehensive README overhaul highlighting both Active Directory and LDAP support equally
- CI/CD badges in README (Build status, Go Report Card, License, WCAG compliance)
- Improved attribution to Netresearch DTT GmbH in footer
- "GopherPass" branding text in page header for better visual identity

### Changed

- **UI Terminology Updates**: Updated all page titles and UI copy to use "GopherPass" branding
  - Main page title: "Password Changer" → "GopherPass — Self-service password change & reset"
  - Password change button: "Change Password" → "Update Password"
  - Success messages updated for clarity and consistency
  - Page titles standardized across all flows (index, forgot-password, reset-password)
- **README Transformation**: Complete rewrite emphasizing:
  - Equal prominence for Active Directory and LDAP support
  - "Password change & reset" dual functionality
  - Neutral "directory account" terminology instead of protocol-specific language
  - Improved quick start examples and configuration documentation
  - Enhanced feature descriptions and project background
- Footer attribution updated to "Built by Netresearch DTT GmbH — open source, written in Go"

### Technical Notes

- **No Breaking Changes**: All environment variables, CLI flags, module paths, and API endpoints remain unchanged
- **No Functional Changes**: This release contains presentation and documentation updates only
- **Backward Compatibility**: Existing deployments will continue to work without any configuration changes

---

## Project History

This changelog was introduced with the GopherPass branding initiative. For earlier project history, see the git commit log.
