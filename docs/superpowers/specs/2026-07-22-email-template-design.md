# Custom email templates — design

**Issue:** [#627](https://github.com/netresearch/ldap-selfservice-password-changer/issues/627)
**Date:** 2026-07-22
**Status:** Approved (design phase)
**Related:** [#626](https://github.com/netresearch/ldap-selfservice-password-changer/issues/626) (UI branding — separate work)

## Problem

The password-reset email is fully hardcoded in `internal/email/service.go`: a
`text/plain`-only body (`buildResetEmailBody`), a literal subject
(`"Password Reset Request"`), and a bare `From:` header. Enterprises need the
reset email to match corporate standards — branded HTML, a custom subject, a
recognizable sender name, and helpdesk/list routing headers. This design makes
the email content and headers operator-configurable via the Go template engine,
while keeping the plain-text/accessibility path and failing fast on
misconfiguration.

## Scope

In scope:

- Operator-supplied HTML **and** plain-text body templates (multipart email).
- Operator-supplied subject template.
- Structured, validated sender identity: `From` display name and `Reply-To`.
- A raw header escape hatch for arbitrary routing headers.
- Fail-fast validation of all of the above at startup.

Out of scope (recorded, not built here):

- A global `GOPHERPASS_*` env-var prefix. The correct long-term convention, but
  a cross-cutting rename of every existing var — separate change.
- Templating the user's LDAP display name/username. The email service receives
  only `(to, token)` today; exposing more requires a `SendResetEmail` signature
  change and caller rework.
- #626 UI branding (logo, favicon, page title).

## Decisions (from brainstorming)

1. **Format:** HTML + plain-text `multipart/alternative`. Preserves rendering in
   all clients and the plain-text/accessibility path. (Not plain-only; not a
   single content-type toggle.)
2. **Override mechanism:** file paths via env/flag, bind-mounted in Docker.
   Unset ⇒ embedded defaults. (Not inline env strings; not a magic directory.)
3. **Naming group:** new `EMAIL_TEMPLATE_*` group for content; sender identity
   stays in the existing `SMTP_*` group. No global app prefix (out of scope).
4. **Templatable surface:** subject + HTML body + text body, plus structured
   `From` name / `Reply-To`, plus a raw header map.
5. **Two header layers:**
   - *Structured, safe:* `SMTP_FROM_NAME`, `EMAIL_REPLY_TO` — app validates and
     encodes; app owns correctness.
   - *Raw escape hatch:* `SMTP_HEADER_OVERRIDE_*` — verbatim value; operator owns
     correctness; app enforces only structural integrity.
6. **Override map expression:** prefix scan of `os.Environ()` (env-only, no flag
   mirror), suffix `_`→`-` for the header name.
7. **Failure mode:** fail fast at startup. Misconfiguration exits non-zero;
   only *unset* (not *broken*) falls back to embedded defaults.

## Configuration surface

### Content templates — `EMAIL_TEMPLATE_*`

| Env var | Flag | Default | Notes |
|---|---|---|---|
| `EMAIL_TEMPLATE_HTML` | `--email-template-html` | embedded default | Path to HTML body template file |
| `EMAIL_TEMPLATE_TEXT` | `--email-template-text` | embedded default | Path to plain-text body template file |
| `EMAIL_TEMPLATE_SUBJECT` | `--email-template-subject` | `Password Reset Request` | Inline string template (subjects are one-liners) |

### Structured header knobs — `SMTP_*` / `EMAIL_*`

| Env var | Flag | Default | Notes |
|---|---|---|---|
| `SMTP_FROM_NAME` | `--smtp-from-name` | *(empty)* | `From:` display name; RFC 2047 encoded-word if non-ASCII, quoted if it contains specials |
| `EMAIL_REPLY_TO` | `--email-reply-to` | *(empty)* | `Reply-To:` address; validated with `ValidateEmailAddress` |

Prefix rationale: `SMTP_FROM_NAME` sits with its sibling `SMTP_FROM_ADDRESS`
(sender identity, transport-adjacent); `Reply-To` is message-content routing and
groups with `EMAIL_TEMPLATE_*`, hence `EMAIL_REPLY_TO`. Both would ultimately
live under a `GOPHERPASS_*` prefix (out of scope).

### Raw header escape hatch — `SMTP_HEADER_OVERRIDE_*` (env-only)

- `SMTP_HEADER_OVERRIDE_<NAME>=value` → header `<NAME>` with `_`→`-`, value used
  verbatim (no encoding, no quoting — operator's responsibility).
  Example: `SMTP_HEADER_OVERRIDE_X_HELPDESK_TOPIC=password-reset` →
  `X-HelpDesk-Topic: password-reset`.
- Built by scanning `os.Environ()` for the prefix during `ParseArgs`.
- Header-name casing is normalized (field names are case-insensitive per RFC 5322).
- **Integrity guard (non-negotiable):** reject any value containing CR or LF, and
  require the resulting field-name to be a valid RFC 5322 token; violations feed
  `ConfigError` and fail startup. Operators needing N headers set N vars.
- **Precedence:** applied **last**; wins for any header it names, including
  overriding the `From:`/`Reply-To:` *headers*. The SMTP **envelope** sender
  passed to `smtp.SendMail` remains `SMTP_FROM_ADDRESS`, unaffected by an
  override of the `From:` header.

## Template data contract

Both body templates and the subject template render against:

```go
type resetEmailData struct {
    ResetLink     string // {BaseURL}/reset-password?token={token}
    Token         string // raw token
    BaseURL       string // trimmed APP_BASE_URL
    Recipient     string // the "to" address
    ExpiryMinutes uint    // from RESET_TOKEN_EXPIRY_MINUTES
}
```

- HTML body uses `html/template` (contextual auto-escaping). Text body and
  subject use `text/template`.
- `ExpiryMinutes` fixes an existing latent bug: today's default body hardcodes
  "15 minutes" while the real expiry is configurable. Embedded defaults will use
  `{{.ExpiryMinutes}}`.
- Contract is intentionally minimal and extensible; no username/display-name
  (see out of scope).

## Architecture & components

All changes live in `internal/email` plus config plumbing in `internal/options`
and `main.go`. No new packages.

### Embedded defaults

- Add `internal/email/templates/reset.html.tmpl` and `reset.txt.tmpl`, embedded
  via `//go:embed` (mirrors the `internal/web/templates` pattern).
- `reset.txt.tmpl` reproduces today's wording, with `{{.ExpiryMinutes}}` instead
  of the hardcoded "15 minutes".
- `reset.html.tmpl` is a simple, accessible HTML equivalent (semantic markup,
  high-contrast, a plainly labeled link — WCAG-minded).

### `email.Config` additions

```go
TemplateHTMLPath string
TemplateTextPath string
SubjectTemplate  string
FromName         string
ReplyTo          string
HeaderOverrides  map[string]string
ExpiryMinutes    uint
```

### Fail-fast constructor

- `NewService(*Config) (*Service, error)` (signature change; internal API only).
- On construction: resolve each template (configured file path or embedded
  default), parse with the correct engine, and **dry-run against a sample
  context**. Any error is returned and treated as fatal in `main.go`.
- Parsed `*template.Template` values (html body, text body, subject) are cached
  on `Service`; per-send only executes them.

### Message assembly

- `buildResetEmailBody` is replaced by a template renderer producing text + HTML.
- `buildEmailMessage` is rewritten to emit `multipart/alternative` via
  `mime/multipart`: **text part first, HTML part second** (clients prefer the
  last alternative).
- Body parts encoded **quoted-printable** (`mime/quotedprintable`) with
  `charset=UTF-8` — correct for UTF-8 over 7-bit-only relays, which today's raw
  8-bit send does not guarantee.
- Header order: `From` (optional display name), `To`, `Subject`, `Reply-To`,
  then the override map applied last, then MIME headers (`MIME-Version`,
  `Content-Type: multipart/alternative; boundary=…`).
- Rendered subject is forced single-line (CR/LF rejected/stripped) to prevent
  header injection through the subject.

### Options wiring

- `EMAIL_TEMPLATE_*` and the two structured header vars follow the existing
  `envStringOrDefault` + `fs.String` pattern.
- `SMTP_HEADER_OVERRIDE_*` map built by a small `os.Environ()` prefix scan in
  `ParseArgs`, with field-name/CRLF validation feeding `ConfigError`.
- `main.go`'s `buildEmailConfig` maps the new opts (including
  `ExpiryMinutes` from `ResetTokenExpiryMinutes`) into `email.Config`, and
  handles the new `NewService` error.

## Error handling

- Missing file, parse error, undefined-field on dry-run, invalid `EMAIL_REPLY_TO`,
  invalid override field-name, or CR/LF in an override value ⇒ `ConfigError` /
  `NewService` error ⇒ process exits non-zero at boot with a clear message.
- Never a silent fallback to defaults for a *misconfigured* template — fallback
  only applies to an *unset* one.
- Subject render forced single-line.

## Testing

- **Unit (`service_internal_test.go`):** default and custom rendering;
  `ExpiryMinutes` substitution; `From` display-name encoding (ASCII / non-ASCII
  RFC 2047 / specials quoted); override-map application and precedence over
  `From`/`Reply-To`; CR/LF and bad-field-name rejection; subject CR/LF stripping;
  multipart structure and quoted-printable correctness.
- **Options (`app_test.go`):** parsing of the new vars, prefix-scan map assembly,
  and each validation-failure path.
- **Integration (`service_integration_test.go`):** existing MailHog/testcontainers
  send still passes; assert the received message is multipart with both parts and
  that the custom subject/headers are present.
- **Fuzz (`email_fuzz_test.go`):** extend to fuzz override field-name/value
  validation.
- Update existing callers/tests for the `NewService` error return.

## Documentation

- Update `internal/CLAUDE.md` env-var list and `.env.local.example`.
- Update `docs/` email/config references and any api/development guide that lists
  configuration.
- Note the `SMTP_HEADER_OVERRIDE_*` integrity guard and precedence behavior.
