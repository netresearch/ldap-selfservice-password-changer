# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

---

## [v1.5.0] - 2026-07-23

### Added

- **Configurable password-reset emails** ([#629](https://github.com/netresearch/ldap-selfservice-password-changer/pull/629), closes [#627](https://github.com/netresearch/ldap-selfservice-password-changer/issues/627)). The message was previously hardcoded: a `text/plain`-only body, the literal subject `Password Reset Request`, and a bare `From:` header. Operators can now supply Go templates for the HTML body, the plain-text body and the subject, set a sender display name and a `Reply-To`, and inject arbitrary routing headers. Everything is optional — unset values fall back to templates embedded in the binary, so an existing deployment behaves as before.
  - New settings: `EMAIL_TEMPLATE_HTML`, `EMAIL_TEMPLATE_TEXT`, `EMAIL_TEMPLATE_SUBJECT`, `SMTP_FROM_NAME`, `EMAIL_REPLY_TO`, and `SMTP_HEADER_OVERRIDE_*` (one variable per header, suffix `_` maps to `-`).
  - Template fields: `{{.ResetLink}}`, `{{.Token}}`, `{{.BaseURL}}`, `{{.Recipient}}`, `{{.ExpiryMinutes}}`.
  - Messages are now `multipart/alternative` (plain-text part first, HTML second), both quoted-printable with `charset=UTF-8`. The plain-text path is retained rather than replaced.
  - Templates are parsed and dry-run at startup, so a broken one aborts boot instead of surfacing on the first reset request.
- **RFC 5322 `Date:` header** on reset emails. Its absence is scored by spam filters, and not every MTA backfills it.

### Fixed

- **Password changes and resets now work against OpenLDAP** ([#636](https://github.com/netresearch/ldap-selfservice-password-changer/pull/636), closes [#633](https://github.com/netresearch/ldap-selfservice-password-changer/issues/633)). `simple-ldap-go` wrote the Active-Directory-only `unicodePwd` attribute for every password write, so both flows failed against any non-AD directory with `LDAP Result Code 17 "Undefined Attribute Type"` — total failure on the first attempt, with no configuration that avoided it, while the README advertised OpenLDAP support. Fixed upstream in `v1.12.1` (RFC 3062 Password Modify on non-AD directories) and `v1.12.2` (the self-service change binds as the user rather than issuing the operation on the pooled service-account connection). Verified end to end against a real OpenLDAP stack, each write confirmed by binding to the directory directly.
- **The email stated the wrong expiry.** The body was hardcoded to "15 minutes" while the real lifetime came from `RESET_TOKEN_EXPIRY_MINUTES`, so any deployment that changed the expiry told users something false. It now renders the configured value.
- **Header injection in the message builder.** Override values were written verbatim into the header block, so a value containing CR/LF could smuggle arbitrary headers into the message. Names and values are now validated in the email package itself, not only at config-parse time; CR, LF, NUL, other C0 controls and DEL are rejected.
- **A mistyped template path could hang startup.** `EMAIL_TEMPLATE_HTML=/dev/zero` blocked before the listener bound and grew until the OOM killer fired. Template reads now require a regular file and are capped at 1 MiB.
- `SMTP_FROM_NAME` set without `SMTP_FROM_ADDRESS` produced the malformed header `From: "ACME IT" <@>`; the display name is now dropped and startup warns.

### Changed

- **`SMTP_FROM_ADDRESS` is validated when password reset is enabled.** A malformed value now aborts startup rather than failing on the first reset request. An _empty_ value still boots — that was accepted before this release and refusing it would stop existing deployments from starting — but logs a warning. Addresses are checked with Go's RFC 5322 parser rather than a stricter pattern, so internal senders such as `noreply@localhost` are accepted.
- Documentation was consolidated and corrected across the repo: the local-run runbook moved out of a tool-specific directory into `docs/development-guide.md`, stale pnpm/Corepack instructions were replaced with the Bun reality, and several documented capabilities that the code never implemented (`RATE_LIMIT_*`, `TRUSTED_PROXIES`, a Docker-secrets `*_FILE` convention) were removed.
- Design rationale for the email work is recorded in [`docs/adr/0003-configurable-reset-email-templates.md`](docs/adr/0003-configurable-reset-email-templates.md).

### Internal

- `email.NewService` now returns `(*Service, error)`. This is a breaking change to an `internal/` package, which Go does not permit importing from outside the module, so it affects no consumer and does not warrant a major version.

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
