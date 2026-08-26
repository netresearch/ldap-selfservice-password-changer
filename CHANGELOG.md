# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

---

## [v1.6.2] - 2026-08-26

### Changed

- `simple-ldap-go` updated to [v1.16.0](https://github.com/netresearch/simple-ldap-go/releases/tag/v1.16.0), which closes the `CheckPasswordForDN` user-enumeration timing oracle upstream ([simple-ldap-go#219](https://github.com/netresearch/simple-ldap-go/issues/219)); this app authenticates via the sAMAccountName path, which already had the constant-time treatment, so this is hardening of the bundled library rather than a fix for an exposed path ([#671](https://github.com/netresearch/ldap-selfservice-password-changer/pull/671)).
- All Go dependencies updated across the module graph (indirect-only version moves) ([#669](https://github.com/netresearch/ldap-selfservice-password-changer/pull/669)); bun dev-dependency group updated ([#660](https://github.com/netresearch/ldap-selfservice-password-changer/pull/660)).

---

## [v1.6.1] - 2026-08-26

### Fixed

- OpenLDAP users whose uid is longer than 20 characters can now reset and change their password. The bundled `simple-ldap-go` applied Active Directory's sAMAccountName rules to every server, rejecting such users before any directory query with `sAMAccountName too long`; the dependency is updated to [v1.15.0](https://github.com/netresearch/simple-ldap-go/releases/tag/v1.15.0), which validates identifiers per server type ([#666](https://github.com/netresearch/ldap-selfservice-password-changer/issues/666)).

### Changed

- `go.mod` pins `toolchain go1.27.0` (was go1.26.6); the 1.26.x advisory rationale no longer applies. `go mod tidy` under the new toolchain pruned stale go.sum entries left over from the fiber 3.4→3.5 bump — dropping the `go-spew`/`go-difflib` indirects testify ≥ 1.11 no longer needs and moving `shamaton/msgpack/v3` to the v3.2.0 that fiber v3.5.0 already required. Test files were reformatted per Go 1.27 gofumpt and the `strptr` test helper was inlined to Go 1.26's `new(expr)`.

---

## [v1.6.0] - 2026-07-28

### Added

- **Custom branding** ([#639](https://github.com/netresearch/ldap-selfservice-password-changer/pull/639)). The product name, logo, page title and the Netresearch footer line were hardcoded. Five optional settings now cover them; a deployment that sets none of them looks exactly as before.
  - `BRANDING_PRODUCT_NAME` — the wordmark rendered next to the logo (default `GopherPass`).
  - `BRANDING_PAGE_TITLE` — the browser tab title of the start page only; the reset pages keep their own prefix and append the product name.
  - `BRANDING_LOGO_ALT` — alternative text for the logo. It takes effect only while no wordmark is shown, because a screen reader would otherwise announce the brand twice. Clearing `BRANDING_PRODUCT_NAME` for a logo-only header therefore _requires_ it; the combination of neither aborts startup.
  - `BRANDING_SHOW_ATTRIBUTION` — the "Built by Netresearch" footer line (default `true`). The footer link keeps pointing at the upstream project under any name.
  - `BRANDING_DIR` — a directory layered over the embedded static assets, replacing them file by file: `logo.webp`, `logo-dark.webp`, the eight favicon and home-screen icons, `site.webmanifest`, `browserconfig.xml`. Assets not supplied keep their built-in version; `styles.css` and `js/` are deliberately not replaceable. Any other filename aborts startup rather than being silently ignored, as does a file over 2 MiB or one that is not a regular file. The runtime image is `FROM scratch` and declares no volumes, so the directory has to be bind-mounted explicitly. Kubernetes ConfigMap and Secret volumes work — the dot-prefixed entries kubelet materializes are skipped.
  - `logo-dark.webp` is served in dark mode. The switch is class-based, so the variant follows the in-page toggle rather than only the OS setting, and dark mode needs JavaScript; without it the light logo is always shown. Deleting the dark variant from a running deployment falls back to the light logo instead of a broken image, but adding one needs a restart.
  - Files under `BRANDING_DIR` are served publicly and unauthenticated under `/static/`, so the directory must be operator-owned and mounted read-only. Lookups go through `os.Root` with `O_NONBLOCK` and a stat after opening, so a symlink out of the directory, an oversized file or a FIFO swapped in after startup is rejected at serve time and not only at validation time.

### Fixed

- **Out-of-range numeric settings are rejected at startup instead of wrapping** ([#644](https://github.com/netresearch/ldap-selfservice-password-changer/pull/644)). `SMTP_PORT`, `RESET_TOKEN_EXPIRY_MINUTES`, `RESET_RATE_LIMIT_WINDOW_MINUTES` and `RESET_RATE_LIMIT_REQUESTS` are parsed as `uint` over the full 64-bit range but converted to `int` / `time.Duration` afterwards. A value above that target range wrapped: a window of `18446744073709551615` minutes became `-1m`, which makes the sliding-window limiter drop every timestamp and let every request through, and a request count that large became `-1`, which blocks all of them. Both are now config errors, as is an `SMTP_PORT` above 65535. The values come from the operator's own configuration, not from a request, so this is a footgun rather than a remotely exploitable flaw — but the failing-open case sits on the password-reset endpoint and gives no sign that limiting has stopped.
- `browserconfig.xml` referenced the tile logo as `/mstile-150x150.png`, but static assets are served only under `/static`, and its tile colour disagreed with the `msapplication-TileColor` meta tag ([#640](https://github.com/netresearch/ldap-selfservice-password-changer/pull/640)). No template references the file and Windows probes it only at the site root, so it remains unreachable in practice; this makes its contents correct for a deployment that wires it up via `BRANDING_DIR`.

### Changed

- The per-IP rate limiter is documented ([#638](https://github.com/netresearch/ldap-selfservice-password-changer/pull/638)). Two independent in-memory limiters apply and only one of them is configurable: 10 requests per 60 minutes per IP across both endpoints, hardcoded, alongside the configurable per-identifier limit on reset requests. The README claim of a single configurable rate limit was corrected, and the completed `docs/security-quick-fix-guide.md` was removed. A stale root `.codecov.yml` targeting 70% was deleted: Codecov searches the root before `.github/`, so it shadowed the template-managed 80% config that the CI gate and `internal/AGENTS.md` both assume.
- `go.mod` pins `toolchain go1.26.5`: govulncheck reported 15 standard-library advisories against 1.26.1, all fixed in 1.26.2 or later.
- Dropped the sunset Go Report Card badge from the README ([#643](https://github.com/netresearch/ldap-selfservice-password-changer/pull/643)).

### Dependencies

- `netresearch/simple-ldap-go` → v1.13.0 ([#641](https://github.com/netresearch/ldap-selfservice-password-changer/pull/641))
- Bun group: `tailwindcss` and `@tailwindcss/postcss` → 4.3.3, `postcss` → 8.5.20, `prettier-plugin-tailwindcss` → 0.8.1, `typescript-eslint` → 8.64.0, and `typescript` → 7.0.2, which lifts the 6.0.x pin v1.4.0 introduced ([#642](https://github.com/netresearch/ldap-selfservice-password-changer/pull/642))

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
