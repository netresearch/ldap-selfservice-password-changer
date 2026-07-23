# ADR 0003: Configurable Reset Email Templates

**Status**: Accepted
**Date**: 2026-07-22
**Authors**: Sebastian Mendel

## Context

ADR 0002 introduced the self-service password reset flow with a hardcoded email: a `text/plain`-only body built by `buildResetEmailBody`, the literal subject `Password Reset Request`, and a bare `From:` header carrying only `SMTP_FROM_ADDRESS`.

Deployments that place this application in front of their own users need the reset email to match their corporate standards — branded HTML, a custom subject, a sender name recipients recognize, and routing headers that let helpdesk or list infrastructure classify the message. None of that was reachable without recompiling.

A second, smaller problem sat in the same code: the body text stated "This link will expire in 15 minutes" while the actual lifetime came from `RESET_TOKEN_EXPIRY_MINUTES`. Any deployment that changed the expiry sent users a false statement.

### Requirements

- Operator-supplied HTML body, plain-text body, and subject
- Sender identity beyond a bare address: `From:` display name and `Reply-To:`
- Arbitrary routing headers the application cannot anticipate
- The plain-text body and its accessibility path preserved, not replaced by HTML
- Header injection impossible through any operator-controlled value
- Misconfiguration detected at startup, not on the first send attempt
- Existing deployments continue to boot unchanged when no new variable is set
- No new packages and no new runtime dependencies

## Decision

Make the email content and headers operator-configurable through the Go template engine, with validation at both the configuration boundary and the message boundary.

### 1. Message Format: multipart/alternative

The message is `multipart/alternative` with the **plain-text part first and the HTML part second**, both encoded quoted-printable with `charset=UTF-8`.

- Part order follows the MIME convention that clients prefer the last alternative they can render: HTML-capable clients show the HTML, text clients and screen-reader workflows keep a real text body rather than a stripped tag soup.
- Quoted-printable makes UTF-8 content survive 7-bit-only relays, which the previous raw 8-bit send did not guarantee.
- Rejected: plain-text-only (does not meet the branding requirement) and a single content-type toggle (an operator choosing HTML would silently drop the accessibility path).

### 2. Override Mechanism: File Paths

Body templates are supplied as **file paths** via env var or flag (`EMAIL_TEMPLATE_HTML`, `EMAIL_TEMPLATE_TEXT`), intended to be bind-mounted in Docker. An unset path falls back to the template embedded in the binary via `//go:embed`.

- File paths keep multi-line HTML editable with normal tooling and reviewable in version control.
- Rejected: inline env strings (multi-line HTML in an env var is unreadable and shell-quoting-hostile) and a magic directory scanned for known filenames (implicit, and it makes "which file is actually in use" unanswerable from the configuration alone).
- The subject stays an inline string template (`EMAIL_TEMPLATE_SUBJECT`) because subjects are one-liners.

### 3. Configuration Naming

Content templates get a new `EMAIL_TEMPLATE_*` group. Sender identity stays in the existing `SMTP_*` group, so `SMTP_FROM_NAME` sits beside its sibling `SMTP_FROM_ADDRESS`. `Reply-To` is message-content routing rather than transport identity, so it is `EMAIL_REPLY_TO`.

A single global `GOPHERPASS_*` prefix for every variable in the application was considered and **deliberately deferred**. It is the better long-term convention, but adopting it here would mean renaming every pre-existing variable in one feature branch. Recorded as a separate cross-cutting change.

### 4. Two Deliberately Separate Header Layers

Header configuration is split into two layers with different ownership of correctness:

**Structured and validated** — `SMTP_FROM_NAME`, `EMAIL_REPLY_TO`. The application owns correctness: the display name is quoted or RFC 2047 encoded-word encoded as needed via `net/mail`, and the reply address is checked with `ValidateConfiguredAddress`.

`ValidateConfiguredAddress` is `mail.ParseAddress`, deliberately more permissive than the recipient-side `ValidateEmailAddress` regex. That regex demands a dotted TLD, which rejects `noreply@localhost`, `gopherpass@intranet` and IP-literal domains — senders a containerised app relaying through a local MTA uses routinely, and which booted on every previous release. Recipient addresses keep the strict regex: they derive from directory data rather than operator config, so narrowing what is accepted there limits attack surface instead of breaking a working deployment.

**Raw verbatim escape hatch** — `SMTP_HEADER_OVERRIDE_<NAME>=value`, env-only, built by scanning `os.Environ()` during `ParseArgs`, with `_` in the suffix mapped to `-` in the field name. The value is used exactly as given, with no encoding or quoting. The operator owns correctness; the application enforces only structural integrity.

The escape hatch exists because routing headers are open-ended. A structured API cannot anticipate every `X-*` header a helpdesk, ticketing system, or mailing list needs, and inventing a typed knob per header would never converge. Overrides are applied last and replace any field they name, including `From:` and `Reply-To:`.

### 5. Structural MIME Headers Are Refused

`MIME-Version`, `Content-Type` and `Content-Transfer-Encoding` cannot be set through `SMTP_HEADER_OVERRIDE_*`.

The message builder owns these: `MIME-Version` and `Content-Type` are appended **after** overrides are applied, and `Content-Transfer-Encoding` is written per MIME part. Permitting an override would emit a second, conflicting copy and corrupt the multipart structure — a message whose declared boundary no longer matches the body it wraps.

This is enforced in **both** layers, by design:

- `internal/options` treats a reserved name as a `ConfigError` and refuses to boot, so the operator gets a named error instead of silently broken mail.
- `internal/email` drops any reserved name it is handed anyway, so the email package does not depend on the options layer for its own structural correctness.

### 6. Header Values Reject Control Characters

`ValidateHeaderValue` rejects CR, LF, NUL, every other C0 control, and DEL. HTAB is allowed, being legal folding whitespace. The rendered subject is additionally forced to a single line by stripping CR and LF.

Without this, a header value is a header injection vector: a raw CR or LF lets an operator-supplied value terminate the field and inject further headers, or end the header block and forge a body. Other C0 controls and DEL are rejected because MTA handling of them is undefined — truncation at NUL or a 5xx for the whole message.

This validation lives in the `internal/email` package and runs over the **override map** in the message builder, not only at config-parse time. Overrides reach the builder from more than one direction, so the package that writes the wire format re-checks the values it was handed rather than trusting the options layer.

The builder's own `From:`, `To:` and `Reply-To:` fields do not pass through `ValidateHeaderValue`. They are constrained where they are produced instead: the recipient address is matched against the anchored `emailRegex` in `SendResetEmail`, `SMTP_FROM_ADDRESS` and `EMAIL_REPLY_TO` are parsed with `ValidateConfiguredAddress` in `ParseArgs`, and `SMTP_FROM_NAME` is both checked with `ValidateHeaderValue` in `ParseArgs` and emitted through `mail.Address.String()`, which quotes or RFC 2047-encodes what it is given.

### 7. Envelope Sender and Delivery Semantics

The SMTP envelope sender passed to `smtp.SendMail` remains `SMTP_FROM_ADDRESS` regardless of any `From:` header override. The envelope and the header are distinct, and only the envelope governs bounce routing.

`To:`, `Cc:` and `Bcc:` overrides are display-only: the envelope recipient is always the reset requester, so an override cannot add a delivery target. It is nevertheless visible to that recipient, which operators should account for before putting an internal address in one.

A cross-domain `From:` header override can break SPF, DKIM and DMARC alignment. The application does not attempt to detect this.

### 8. Fail-Fast Validation at Startup

`NewService` resolves each template (configured path or embedded default), parses it with the correct engine, and **dry-runs it against a sample context**. Parsed templates are cached on the `Service`; a send only executes them.

The two validation layers have different reach, which is worth stating exactly:

- **Unconditional**, in `ParseArgs`: an invalid `SMTP_FROM_NAME`, an invalid override field name, a control character in an override value, or an override naming a reserved structural header aborts boot with a non-zero exit, whatever `PASSWORD_RESET_ENABLED` is set to.
- **Only when password reset is enabled**: the two address checks — a malformed non-empty `SMTP_FROM_ADDRESS` and an invalid `EMAIL_REPLY_TO` — sit inside `if *fPasswordResetEnabled` in `ParseArgs`. Template resolution, parsing and the dry run happen inside `NewService`, which `buildRPCHandler` constructs only for `PASSWORD_RESET_ENABLED=true`. So a malformed sender or Reply-To, a missing template file, a parse error or an undefined field aborts boot only in that case; with the feature off, neither address is used and a broken `EMAIL_TEMPLATE_*` path is never read, so the process starts normally.

Fallback to embedded defaults applies only to an **unset** template, never to a **broken** one. Silently serving the default in place of a template the operator explicitly configured would hide the misconfiguration behind mail that still appears to work.

One exception is recorded honestly: an **empty** `SMTP_FROM_ADDRESS` logs a warning instead of aborting. An empty sender is broken in practice — it means `MAIL FROM:<>`, the null sender reserved for bounces, and a `From:` header with no address — but it was accepted before this feature, and refusing to boot on it would break existing deployments on upgrade. A non-empty but malformed value is still a hard error.

### 9. Template Data Contract

Both body templates and the subject template render against:

```go
type resetEmailData struct {
    ResetLink     string // {BaseURL}/reset-password?token={token}
    Token         string // raw token
    BaseURL       string // trimmed APP_BASE_URL
    Recipient     string // the "to" address
    ExpiryMinutes uint   // from RESET_TOKEN_EXPIRY_MINUTES
}
```

- The HTML body uses `html/template` for contextual auto-escaping. The text body and the subject use `text/template`.
- `ExpiryMinutes` fixes the pre-existing bug described in the Context: the embedded default now renders `{{.ExpiryMinutes}}` where the old body hardcoded "15 minutes" against a configurable expiry.
- The contract is intentionally minimal. It carries no username or LDAP display name: the email service receives only `(to, token)` today, and exposing more requires a `SendResetEmail` signature change and caller rework.

## Consequences

### Positive

1. **Branding without a fork**: subject, both bodies, sender name and routing headers are deployment configuration rather than source changes.

2. **Accessibility path preserved**: a real plain-text part is always present, and no configuration removes it.

3. **Correct expiry statement**: the email states the configured lifetime instead of a fixed 15 minutes.

4. **Misconfiguration surfaces at boot**: a broken header fails the process start unconditionally; a broken address or template does so when password reset is enabled, rather than surfacing on the first user's reset attempt as a silent non-delivery. Address and template checks are deliberately scoped to that path — neither value is used with the feature off, and refusing to boot over an unused setting would regress deployments upgrading with a stale placeholder.

5. **Header injection closed at the message boundary**: validation in `internal/email` holds regardless of how the configuration was assembled.

6. **Transport correctness**: quoted-printable with an explicit charset replaces raw 8-bit body bytes.

### Neutral

1. **`email.NewService` became `(*Service, error)`**: template resolution, parsing and the dry run all happen at construction, so construction can fail. This is an internal API, and all callers and tests were updated in the same change.

2. **Templates are read once at startup**: editing a bind-mounted template file requires a restart. Consistent with the rest of the configuration, which is also read once.

3. **Raw overrides shift responsibility to the operator**: values are emitted verbatim, so encoding and quoting of a non-ASCII override value are the operator's to get right. The structured knobs exist for the cases where that is not wanted.

### Negative

1. **An operator can send a broken-but-valid message**: a syntactically clean override that misstates `From:` will pass every check and still fail SPF, DKIM or DMARC at the receiver. The application cannot validate alignment it has no view of.

2. **`Bcc:` is not private**: an override sets a header on the message the requester receives. The name suggests otherwise, and this is documented rather than blocked.

3. **Larger configuration surface**: five new variables plus an open-ended prefix scan, each a place a deployment can get it wrong.

4. **Empty `SMTP_FROM_ADDRESS` still boots**: the warning-not-error exception above preserves upgrade compatibility at the cost of one misconfiguration that reaches runtime.

## Alternatives Considered

### Alternative 1: Plain-Text Only, Templated

**Rejected**:

- Does not meet the branding requirement that motivated the work
- Corporate reset mail is expected to carry a logo and house styling

### Alternative 2: Single Content-Type Toggle

**Rejected**:

- An operator selecting HTML would drop the plain-text part entirely
- Silently removes the accessibility path in exchange for branding
- `multipart/alternative` gives both without a choice

### Alternative 3: Inline Template Strings in Env Vars

**Rejected**:

- Multi-line HTML in an environment variable is unreadable and shell-quoting-hostile
- Not reviewable in version control as the markup it is
- Retained only for the subject, which is genuinely a one-liner

### Alternative 4: Magic Template Directory

**Rejected**:

- Implicit: a file appearing in a scanned directory changes behavior with no configuration change
- The active template cannot be determined from the configuration alone
- Explicit paths make the binding auditable

### Alternative 5: Global `GOPHERPASS_*` Prefix

**Deferred, not rejected**:

- The correct long-term convention for the whole configuration surface
- Requires renaming every pre-existing variable, which is a breaking change for all deployments
- Does not belong in a feature branch; recorded as separate work

### Alternative 6: Raw Header Map Only, No Structured Knobs

**Rejected**:

- Display-name quoting and RFC 2047 encoding are error-prone to hand-write
- `Reply-To` benefits from address validation the application already has
- The two-layer split puts each header where its correctness is cheapest to guarantee

### Alternative 7: Structured Knobs Only, No Escape Hatch

**Rejected**:

- Routing headers are open-ended; a typed knob per header would never converge
- Forces a source change for every new `X-*` header a deployment needs
- The integrity guard makes a verbatim map safe enough to expose

## Configuration Reference

### Content Templates

- `EMAIL_TEMPLATE_HTML` / `--email-template-html`: path to the HTML body template (default: embedded)
- `EMAIL_TEMPLATE_TEXT` / `--email-template-text`: path to the plain-text body template (default: embedded)
- `EMAIL_TEMPLATE_SUBJECT` / `--email-template-subject`: inline subject template (default: `Password Reset Request`)

### Structured Sender Identity

- `SMTP_FROM_NAME` / `--smtp-from-name`: `From:` display name, quoted or RFC 2047 encoded as needed (default: empty)
- `EMAIL_REPLY_TO` / `--email-reply-to`: `Reply-To:` address, validated (default: empty)

### Raw Header Overrides (env-only)

- `SMTP_HEADER_OVERRIDE_<NAME>=value`: sets header `<NAME>` with `_` mapped to `-`, value verbatim
- Example: `SMTP_HEADER_OVERRIDE_X_HELPDESK_TOPIC=password-reset` emits `X-Helpdesk-Topic: password-reset` — names go on the wire in canonical (Go `net/textproto`) casing regardless of how the variable was written
- Applied last; wins for any non-structural header it names
- Refused: `MIME-Version`, `Content-Type`, `Content-Transfer-Encoding`
- Refused: values containing CR, LF, NUL, other C0 controls, or DEL

### Inherited from ADR 0002

- `SMTP_FROM_ADDRESS`: sender address and SMTP envelope sender
- `RESET_TOKEN_EXPIRY_MINUTES`: supplies `ExpiryMinutes` to the templates
- `APP_BASE_URL`: supplies `BaseURL` and the reset link

## References

- [ADR 0002: Self-Service Password Reset Functionality](0002-password-reset-functionality.md)
- [Issue #627: Custom email templates](https://github.com/netresearch/ldap-selfservice-password-changer/issues/627)
- [RFC 5322 - Internet Message Format](https://tools.ietf.org/html/rfc5322)
- [RFC 2045 - MIME Part One: Format of Internet Message Bodies](https://tools.ietf.org/html/rfc2045)
- [RFC 2046 - MIME Part Two: Media Types](https://tools.ietf.org/html/rfc2046)
- [RFC 2047 - MIME Part Three: Message Header Extensions for Non-ASCII Text](https://tools.ietf.org/html/rfc2047)
- [Go html/template Package Documentation](https://pkg.go.dev/html/template)
- [OWASP Forgot Password Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Forgot_Password_Cheat_Sheet.html)
