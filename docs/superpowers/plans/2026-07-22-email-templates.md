# Custom Email Templates Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let operators customize the password-reset email (subject, HTML + text body, sender name, Reply-To, and arbitrary routing headers) via the Go template engine, with fail-fast validation and embedded defaults.

**Architecture:** All rendering and MIME assembly stays inside `internal/email`; the `SendResetEmail(to, token string) error` interface consumed by `rpchandler` is unchanged. `NewService` becomes fail-fast (`(*Service, error)`): it loads/parses/dry-runs templates at startup. Emails become `multipart/alternative` (text part first, HTML second) with quoted-printable bodies. Config plumbing follows the existing env+flag pattern in `internal/options`, plus an `os.Environ()` prefix scan for the raw `SMTP_HEADER_OVERRIDE_*` map.

**Tech Stack:** Go 1.26, stdlib `text/template` + `html/template`, `mime`, `net/mail`, `mime/multipart`, `mime/quotedprintable`, `net/textproto`, `net/smtp`.

**Spec:** `docs/superpowers/specs/2026-07-22-email-template-design.md`

## Global Constraints

Every task's requirements implicitly include these:

- Go 1.26. Use `any` (not `interface{}`); wrap every returned error with context via `fmt.Errorf("...: %w", err)`.
- **Fail-fast:** any misconfiguration (missing/unparseable/undefined-field template, invalid Reply-To, invalid override name/value, reserved override) aborts startup with a clear message. Embedded defaults are used only when a template is **unset**, never when it is **misconfigured** — no silent fallback.
- Email is `multipart/alternative`: **text part first, HTML part second**; both bodies quoted-printable-encoded with `charset=UTF-8`.
- The SMTP **envelope** sender passed to `smtp.SendMail` stays `SMTP_FROM_ADDRESS`, regardless of any `From:`-header override.
- Structural MIME headers (`MIME-Version`, `Content-Type`, `Content-Transfer-Encoding`) can **not** be set via `SMTP_HEADER_OVERRIDE_*`.
- HTML default template is accessibility-minded: semantic markup, high-contrast dark text on light background (≥7:1 body text), link shown as text as well as a button.
- Commits: [Conventional Commits](https://www.conventionalcommits.org/), `git commit -s` (DCO), **no AI attribution**, and end the message body with the `Claude-Session:` trailer already used on this branch.
- Before marking the feature done: `bunx prettier --write .`, `go build ./...`, `go test ./...`, `go vet ./...` all green (evidence required).

**Branch:** `feat/627-email-templates` (already created).

---

## File Structure

**Create:**
- `internal/email/templates/reset.txt.tmpl` — embedded default plain-text body.
- `internal/email/templates/reset.html.tmpl` — embedded default HTML body.
- `internal/email/render.go` — `resetEmailData`, embedded defaults, `renderer`, `newRenderer`, `render`, `loadTemplateSource`.
- `internal/email/headers.go` — header validators + encoders: `ValidateHeaderName`, `ValidateHeaderValue`, `formatFrom`, `encodeSubject`, `applyHeaderOverrides`, `headerField`.
- `internal/email/message.go` — `buildMIMEMessage`, `writeQPPart`.

**Modify:**
- `internal/email/service.go` — expand `Config`; `Service` holds `*renderer`; `NewService` → `(*Service, error)`; rewrite `SendResetEmail`/`sendEmail`; delete `buildResetEmailBody` and old `buildEmailMessage`.
- `internal/options/app.go` — new flags/env + `Opts` fields + `parseHeaderOverrides`.
- `main.go` — `buildEmailConfig` maps new opts (incl. `ExpiryMinutes`); handle `NewService` error.
- Tests: `internal/email/service_internal_test.go`, `internal/email/service_test.go`, `internal/email/email_fuzz_test.go`, `internal/rpchandler/handler_test.go`, `internal/options/app_test.go`; tag-gated: `internal/email/service_integration_test.go`, `internal/rpchandler/handler_integration_test.go`, `integration_test.go`.
- Docs: `.env.local.example`, `internal/CLAUDE.md`, and any `docs/*` config reference found by grep.

---

## Task 1: Template renderer + embedded defaults

**Files:**
- Create: `internal/email/templates/reset.txt.tmpl`
- Create: `internal/email/templates/reset.html.tmpl`
- Create: `internal/email/render.go`
- Test: `internal/email/render_internal_test.go`

**Interfaces:**
- Produces:
  - `type resetEmailData struct { ResetLink, Token, BaseURL, Recipient string; ExpiryMinutes uint }`
  - `func newRenderer(cfg *Config) (*renderer, error)`
  - `func (r *renderer) render(data resetEmailData) (subject, text, html string, err error)`
  - `const defaultSubjectTemplate = "Password Reset Request"`
- Consumes: `Config` fields `SubjectTemplate`, `TemplateTextPath`, `TemplateHTMLPath` (added in Task 4; this task references them, so add the three string fields to `Config` here if Task 4 not yet done — see note).

> **Note:** This task reads `cfg.SubjectTemplate`, `cfg.TemplateTextPath`, `cfg.TemplateHTMLPath`. If implementing strictly in order, add just those three `string` fields to the existing `Config` struct in `service.go` now; Task 4 adds the rest. The build stays green because the current `Config` is a plain struct with no constructor coupling.

- [ ] **Step 1: Create the embedded default text template**

Create `internal/email/templates/reset.txt.tmpl`:

```
Hi,

We received a request to reset the password for your account.
If you made this request, click the link below to continue:

{{.ResetLink}}

This link will expire in {{.ExpiryMinutes}} minutes.

If you didn't request a password reset, you can safely ignore this email.
Your password will not be changed.

--
LDAP Selfservice Password Changer
```

- [ ] **Step 2: Create the embedded default HTML template**

Create `internal/email/templates/reset.html.tmpl`:

```html
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Password reset request</title>
  </head>
  <body style="margin:0;padding:0;background:#ffffff;color:#1a1a1a;font-family:Arial,Helvetica,sans-serif;">
    <div style="max-width:600px;margin:0 auto;padding:24px;">
      <h1 style="font-size:20px;margin:0 0 16px;color:#1a1a1a;">Password reset request</h1>
      <p style="font-size:16px;line-height:1.5;margin:0 0 16px;">Hi,</p>
      <p style="font-size:16px;line-height:1.5;margin:0 0 16px;">
        We received a request to reset the password for your account. If you made this request, use the button below to
        continue:
      </p>
      <p style="margin:0 0 16px;">
        <a
          href="{{.ResetLink}}"
          style="display:inline-block;padding:12px 20px;background:#073763;color:#ffffff;text-decoration:none;border-radius:4px;font-size:16px;"
          >Reset your password</a
        >
      </p>
      <p style="font-size:16px;line-height:1.5;margin:0 0 16px;">
        Or paste this link into your browser:<br />
        <a href="{{.ResetLink}}" style="color:#073763;">{{.ResetLink}}</a>
      </p>
      <p style="font-size:16px;line-height:1.5;margin:0 0 16px;">This link will expire in {{.ExpiryMinutes}} minutes.</p>
      <p style="font-size:14px;line-height:1.5;color:#555555;margin:24px 0 0;">
        If you didn't request a password reset, you can safely ignore this email. Your password will not be changed.
      </p>
    </div>
  </body>
</html>
```

- [ ] **Step 3: Write the failing test**

Create `internal/email/render_internal_test.go`:

```go
package email

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewRenderer_Defaults(t *testing.T) {
	r, err := newRenderer(&Config{})
	if err != nil {
		t.Fatalf("newRenderer with defaults: %v", err)
	}

	subject, text, html, err := r.render(resetEmailData{
		ResetLink:     "https://example.com/reset-password?token=abc",
		Token:         "abc",
		BaseURL:       "https://example.com",
		Recipient:     "user@example.com",
		ExpiryMinutes: 20,
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if subject != defaultSubjectTemplate {
		t.Errorf("subject = %q, want %q", subject, defaultSubjectTemplate)
	}
	for _, want := range []string{"https://example.com/reset-password?token=abc", "20 minutes", "safely ignore"} {
		if !strings.Contains(text, want) {
			t.Errorf("text body missing %q", want)
		}
	}
	for _, want := range []string{"https://example.com/reset-password?token=abc", "20 minutes", "Reset your password"} {
		if !strings.Contains(html, want) {
			t.Errorf("html body missing %q", want)
		}
	}
}

func TestNewRenderer_CustomSubjectAndFiles(t *testing.T) {
	dir := t.TempDir()
	textPath := filepath.Join(dir, "body.txt")
	htmlPath := filepath.Join(dir, "body.html")
	if err := os.WriteFile(textPath, []byte("Reset for {{.Recipient}}: {{.ResetLink}}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(htmlPath, []byte("<p>{{.Recipient}} {{.ResetLink}}</p>"), 0o600); err != nil {
		t.Fatal(err)
	}

	r, err := newRenderer(&Config{
		SubjectTemplate:  "[ACME] Reset your password",
		TemplateTextPath: textPath,
		TemplateHTMLPath: htmlPath,
	})
	if err != nil {
		t.Fatalf("newRenderer: %v", err)
	}

	subject, text, _, err := r.render(resetEmailData{Recipient: "u@x.com", ResetLink: "https://x/y"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if subject != "[ACME] Reset your password" {
		t.Errorf("subject = %q", subject)
	}
	if !strings.Contains(text, "Reset for u@x.com: https://x/y") {
		t.Errorf("text = %q", text)
	}
}

func TestNewRenderer_Errors(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		if _, err := newRenderer(&Config{TemplateTextPath: "/no/such/file.txt"}); err == nil {
			t.Fatal("expected error for missing template file")
		}
	})
	t.Run("parse error", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "bad.txt")
		if err := os.WriteFile(p, []byte("{{ .ResetLink "), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := newRenderer(&Config{TemplateTextPath: p}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("undefined field caught by dry-run", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "bad.txt")
		if err := os.WriteFile(p, []byte("{{ .DoesNotExist }}"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := newRenderer(&Config{TemplateTextPath: p}); err == nil {
			t.Fatal("expected dry-run error for undefined field")
		}
	})
}
```

- [ ] **Step 4: Run test to verify it fails**

Run: `go test ./internal/email/ -run TestNewRenderer -v`
Expected: FAIL — `undefined: newRenderer` / `resetEmailData` (and `Config` has no `SubjectTemplate` field yet).

- [ ] **Step 5: Add the three template fields to Config (if not already present)**

In `internal/email/service.go`, add to the `Config` struct (full struct is finalized in Task 4):

```go
	SubjectTemplate  string // Inline subject template; empty => default
	TemplateHTMLPath string // Path to custom HTML body template; empty => embedded default
	TemplateTextPath string // Path to custom text body template; empty => embedded default
```

- [ ] **Step 6: Implement the renderer**

Create `internal/email/render.go`:

```go
package email

import (
	"bytes"
	_ "embed"
	"fmt"
	htmltemplate "html/template"
	"os"
	texttemplate "text/template"
)

//go:embed templates/reset.txt.tmpl
var defaultTextTemplate string

//go:embed templates/reset.html.tmpl
var defaultHTMLTemplate string

// defaultSubjectTemplate is used when Config.SubjectTemplate is empty.
const defaultSubjectTemplate = "Password Reset Request"

// resetEmailData is the context passed to every reset-email template.
type resetEmailData struct {
	ResetLink     string
	Token         string
	BaseURL       string
	Recipient     string
	ExpiryMinutes uint
}

// sampleResetData dry-runs templates at construction so a broken template
// fails at startup, not when the first reset email is sent.
var sampleResetData = resetEmailData{
	ResetLink:     "https://example.com/reset-password?token=sample",
	Token:         "sample",
	BaseURL:       "https://example.com",
	Recipient:     "user@example.com",
	ExpiryMinutes: 15,
}

// renderer holds the parsed subject/text/html templates.
type renderer struct {
	subject *texttemplate.Template
	text    *texttemplate.Template
	html    *htmltemplate.Template
}

// newRenderer loads, parses and dry-runs the templates. A configured path
// overrides the embedded default; an unset path uses it. Any failure is
// returned (fail-fast). Templates use missingkey=error so an undefined field
// surfaces during the dry-run rather than silently rendering "<no value>".
func newRenderer(cfg *Config) (*renderer, error) {
	subjectSrc := cfg.SubjectTemplate
	if subjectSrc == "" {
		subjectSrc = defaultSubjectTemplate
	}
	subjectTmpl, err := texttemplate.New("subject").Option("missingkey=error").Parse(subjectSrc)
	if err != nil {
		return nil, fmt.Errorf("parse subject template: %w", err)
	}

	textSrc, err := loadTemplateSource(cfg.TemplateTextPath, defaultTextTemplate)
	if err != nil {
		return nil, fmt.Errorf("text template: %w", err)
	}
	textTmpl, err := texttemplate.New("text").Option("missingkey=error").Parse(textSrc)
	if err != nil {
		return nil, fmt.Errorf("parse text template: %w", err)
	}

	htmlSrc, err := loadTemplateSource(cfg.TemplateHTMLPath, defaultHTMLTemplate)
	if err != nil {
		return nil, fmt.Errorf("html template: %w", err)
	}
	htmlTmpl, err := htmltemplate.New("html").Option("missingkey=error").Parse(htmlSrc)
	if err != nil {
		return nil, fmt.Errorf("parse html template: %w", err)
	}

	r := &renderer{subject: subjectTmpl, text: textTmpl, html: htmlTmpl}

	if _, _, _, err := r.render(sampleResetData); err != nil {
		return nil, fmt.Errorf("template dry-run: %w", err)
	}

	return r, nil
}

// loadTemplateSource returns the file content at path, or fallback when path
// is empty. A configured-but-unreadable path is an error.
func loadTemplateSource(path, fallback string) (string, error) {
	if path == "" {
		return fallback, nil
	}
	b, err := os.ReadFile(path) //#nosec G304 -- operator-controlled config path, intentional
	if err != nil {
		return "", fmt.Errorf("read %q: %w", path, err)
	}
	return string(b), nil
}

// render executes all three templates against data.
func (r *renderer) render(data resetEmailData) (subject, text, html string, err error) {
	var sb bytes.Buffer
	if err = r.subject.Execute(&sb, data); err != nil {
		return "", "", "", fmt.Errorf("render subject: %w", err)
	}
	var tb bytes.Buffer
	if err = r.text.Execute(&tb, data); err != nil {
		return "", "", "", fmt.Errorf("render text body: %w", err)
	}
	var hb bytes.Buffer
	if err = r.html.Execute(&hb, data); err != nil {
		return "", "", "", fmt.Errorf("render html body: %w", err)
	}
	return sb.String(), tb.String(), hb.String(), nil
}
```

> **Why `missingkey=error`:** without it, a typo like `{{.Reciptient}}` renders `<no value>` silently. With a struct data type, an unknown *field* is already a hard error at execute time — `missingkey` only affects maps — but setting it is defensive and documents intent. Keep it.

- [ ] **Step 7: Run test to verify it passes**

Run: `go test ./internal/email/ -run TestNewRenderer -v`
Expected: PASS (all three subtests + defaults + custom).

- [ ] **Step 8: Commit**

```bash
git add internal/email/render.go internal/email/render_internal_test.go internal/email/templates/ internal/email/service.go
git commit -s -m "$(cat <<'EOF'
feat(email): add reset-email template renderer with embedded defaults

Add resetEmailData contract, embedded text/HTML defaults, and a fail-fast
renderer that parses and dry-runs subject/text/html templates. Uses
ExpiryMinutes instead of the previously hardcoded "15 minutes".

Refs #627

Claude-Session: https://claude.ai/code/session_01GYyjR27xvXkN8aefvbWo8E
EOF
)"
```

---

## Task 2: Header validators + fuzz

**Files:**
- Create: `internal/email/headers.go` (validators + encoders portion)
- Test: `internal/email/headers_internal_test.go`
- Modify: `internal/email/email_fuzz_test.go`

**Interfaces:**
- Produces:
  - `func ValidateHeaderName(name string) error`
  - `func ValidateHeaderValue(value string) error`
  - `func encodeSubject(subject string) string`
  - `func formatFrom(name, address string) string`
  - `type headerField struct { key, value string }`
  - `func applyHeaderOverrides(fields []headerField, overrides map[string]string) []headerField`

- [ ] **Step 1: Write the failing test**

Create `internal/email/headers_internal_test.go`:

```go
package email

import (
	"strings"
	"testing"
)

func TestValidateHeaderName(t *testing.T) {
	valid := []string{"X-HelpDesk-Topic", "Reply-To", "X-Customer-ID", "List-Unsubscribe"}
	for _, n := range valid {
		if err := ValidateHeaderName(n); err != nil {
			t.Errorf("ValidateHeaderName(%q) unexpected error: %v", n, err)
		}
	}
	invalid := []string{"", "X HelpDesk", "X:Bad", "X-Bad\r", "Naïve"}
	for _, n := range invalid {
		if err := ValidateHeaderName(n); err == nil {
			t.Errorf("ValidateHeaderName(%q) expected error", n)
		}
	}
}

func TestValidateHeaderValue(t *testing.T) {
	if err := ValidateHeaderValue("normal value 123 @!#"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	for _, v := range []string{"line1\r\nInjected: yes", "with\rCR", "with\nLF"} {
		if err := ValidateHeaderValue(v); err == nil {
			t.Errorf("ValidateHeaderValue(%q) expected error", v)
		}
	}
}

func TestEncodeSubject(t *testing.T) {
	if got := encodeSubject("Password Reset Request"); got != "Password Reset Request" {
		t.Errorf("ASCII subject changed: %q", got)
	}
	if got := encodeSubject("Zurücksetzen"); !strings.HasPrefix(got, "=?utf-8?q?") && !strings.HasPrefix(got, "=?UTF-8?q?") {
		t.Errorf("non-ASCII subject not RFC 2047 encoded: %q", got)
	}
	if got := encodeSubject("line1\r\nline2"); strings.ContainsAny(got, "\r\n") {
		t.Errorf("subject still contains CR/LF: %q", got)
	}
}

func TestFormatFrom(t *testing.T) {
	if got := formatFrom("", "noreply@acme.com"); got != "noreply@acme.com" {
		t.Errorf("bare from = %q, want noreply@acme.com", got)
	}
	if got := formatFrom("ACME IT", "noreply@acme.com"); got != `"ACME IT" <noreply@acme.com>` {
		t.Errorf("named from = %q", got)
	}
	// Non-ASCII display name must be RFC 2047 encoded.
	if got := formatFrom("ACME Straße", "noreply@acme.com"); !strings.Contains(got, "=?utf-8?") && !strings.Contains(got, "=?UTF-8?") {
		t.Errorf("non-ASCII name not encoded: %q", got)
	}
}

func TestApplyHeaderOverrides(t *testing.T) {
	base := []headerField{
		{key: "From", value: "noreply@acme.com"},
		{key: "To", value: "u@x.com"},
	}
	out := applyHeaderOverrides(base, map[string]string{
		"from":            "ACME <help@acme.com>", // canonical-key match, replaces
		"X-HelpDesk-Topic": "reset",               // new, appended
	})

	var from, topic string
	var fromCount int
	for _, f := range out {
		switch f.key {
		case "From":
			from = f.value
			fromCount++
		case "X-Helpdesk-Topic":
			topic = f.value
		}
	}
	if fromCount != 1 {
		t.Errorf("From appears %d times, want 1", fromCount)
	}
	if from != "ACME <help@acme.com>" {
		t.Errorf("From = %q, want override value", from)
	}
	if topic != "reset" {
		t.Errorf("X-Helpdesk-Topic = %q, want reset", topic)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/email/ -run 'TestValidateHeader|TestEncodeSubject|TestFormatFrom|TestApplyHeaderOverrides' -v`
Expected: FAIL — undefined `ValidateHeaderName`, etc.

- [ ] **Step 3: Implement validators + encoders**

Create `internal/email/headers.go`:

```go
package email

import (
	"fmt"
	"mime"
	"net/mail"
	"net/textproto"
	"regexp"
	"sort"
	"strings"
)

// headerNameRegex matches an RFC 5322 field name: 1*ftext, where ftext is any
// printable US-ASCII char (33-126) except ':' (58). No spaces, no controls.
// Simple character class + '+', so no catastrophic backtracking (Sonar S5852).
var headerNameRegex = regexp.MustCompile(`^[!-9;-~]+$`)

// ValidateHeaderName reports whether name is a syntactically valid RFC 5322
// header field name.
func ValidateHeaderName(name string) error {
	if name == "" {
		return fmt.Errorf("empty header name")
	}
	if !headerNameRegex.MatchString(name) {
		return fmt.Errorf("invalid header name %q: must be printable ASCII without spaces or ':'", name)
	}
	return nil
}

// ValidateHeaderValue rejects values that would break message structure.
// A raw CR or LF enables header/body injection, so it is never permitted.
func ValidateHeaderValue(value string) error {
	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("header value must not contain CR or LF")
	}
	return nil
}

// encodeSubject forces a single line (CR/LF stripped) and RFC 2047-encodes the
// subject when it contains non-ASCII. Pure-ASCII subjects are unchanged.
func encodeSubject(subject string) string {
	subject = strings.ReplaceAll(subject, "\r", "")
	subject = strings.ReplaceAll(subject, "\n", "")
	return mime.QEncoding.Encode("utf-8", subject)
}

// formatFrom builds the From header value. With a display name it uses
// net/mail so the name is quoted/RFC 2047-encoded correctly; without one it
// emits the bare address (unchanged from prior behaviour).
func formatFrom(name, address string) string {
	if name == "" {
		return address
	}
	addr := mail.Address{Name: name, Address: address}
	return addr.String()
}

// headerField is one ordered header line.
type headerField struct {
	key   string
	value string
}

// applyHeaderOverrides applies overrides last: an override replaces any
// existing field with the same canonical key, otherwise appends. Keys are
// sorted for deterministic output. Values are used verbatim.
func applyHeaderOverrides(fields []headerField, overrides map[string]string) []headerField {
	names := make([]string, 0, len(overrides))
	for n := range overrides {
		names = append(names, n)
	}
	sort.Strings(names)

	for _, rawName := range names {
		key := textproto.CanonicalMIMEHeaderKey(rawName)
		value := overrides[rawName]
		replaced := false
		for i := range fields {
			if textproto.CanonicalMIMEHeaderKey(fields[i].key) == key {
				fields[i].value = value
				replaced = true
				break
			}
		}
		if !replaced {
			fields = append(fields, headerField{key: key, value: value})
		}
	}
	return fields
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/email/ -run 'TestValidateHeader|TestEncodeSubject|TestFormatFrom|TestApplyHeaderOverrides' -v`
Expected: PASS.

- [ ] **Step 5: Add the fuzz target**

Append to `internal/email/email_fuzz_test.go` (note: this file is `package email`, internal — it can call the unexported... no: it is `//nolint:testpackage // tests internal functions` with `package email`? It declares `package email`. The exported `ValidateHeaderName/Value` are reachable regardless). Add:

```go
// FuzzHeaderOverrideValidation fuzzes the header-override validators. They must
// never panic and must reject any value containing CR or LF.
func FuzzHeaderOverrideValidation(f *testing.F) {
	seeds := []struct{ name, value string }{
		{"X-HelpDesk-Topic", "reset"},
		{"", ""},
		{"X Bad", "value"},
		{"X-Inject", "a\r\nEvil: yes"},
		{"X-CR", "a\rb"},
		{"X-LF", "a\nb"},
		{"Naïve", "value"},
	}
	for _, s := range seeds {
		f.Add(s.name, s.value)
	}

	f.Fuzz(func(t *testing.T, name, value string) {
		_ = ValidateHeaderName(name) // must not panic

		err := ValidateHeaderValue(value)
		if strings.ContainsAny(value, "\r\n") && err == nil {
			t.Errorf("ValidateHeaderValue(%q) accepted CR/LF", value)
		}
	})
}
```

> Verify `email_fuzz_test.go` already imports `strings`; it does not. Add `"strings"` to its import block.

- [ ] **Step 6: Run fuzz briefly + full package test**

Run: `go test ./internal/email/ -run 'Fuzz' -v && go test ./internal/email/ -fuzz FuzzHeaderOverrideValidation -fuzztime 10s`
Expected: PASS; no new corpus crash.

- [ ] **Step 7: Commit**

```bash
git add internal/email/headers.go internal/email/headers_internal_test.go internal/email/email_fuzz_test.go
git commit -s -m "$(cat <<'EOF'
feat(email): add header validators, subject/From encoders, override merge

Add ValidateHeaderName/Value (reject CR/LF and bad field-names), RFC 2047
subject and From-name encoding via net/mail, and deterministic header-override
application. Fuzz the override validators.

Refs #627

Claude-Session: https://claude.ai/code/session_01GYyjR27xvXkN8aefvbWo8E
EOF
)"
```

---

## Task 3: Multipart MIME message builder

**Files:**
- Create: `internal/email/message.go`
- Test: `internal/email/message_internal_test.go`

**Interfaces:**
- Consumes: `Service` + `Config` (`FromName`, `FromAddress`, `ReplyTo`, `HeaderOverrides`); `headerField`, `formatFrom`, `encodeSubject`, `applyHeaderOverrides` (Task 2).
- Produces: `func (s *Service) buildMIMEMessage(to, subject, textBody, htmlBody string) ([]byte, error)`

> **Note:** this task references `Config.FromName`, `Config.ReplyTo`, `Config.HeaderOverrides`. If Task 4 hasn't run, add those fields to `Config` now (the full struct is finalized in Task 4).

- [ ] **Step 1: Write the failing test**

Create `internal/email/message_internal_test.go`. Imports: `bufio`, `bytes`, `io`, `mime`, `mime/multipart`, `mime/quotedprintable`, `net/textproto`, `strings`, `testing`.

```go
package email

import (
	"bufio"
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/textproto"
	"strings"
	"testing"
)

// parseMessage splits raw message bytes into headers + a multipart reader.
func parseMessage(t *testing.T, raw []byte) (textproto.MIMEHeader, *multipart.Reader) {
	t.Helper()
	r := bufio.NewReader(bytes.NewReader(raw))
	tp := textproto.NewReader(r)
	hdr, err := tp.ReadMIMEHeader()
	if err != nil && err != io.EOF {
		t.Fatalf("read headers: %v", err)
	}
	mediaType, params, err := mime.ParseMediaType(hdr.Get("Content-Type"))
	if err != nil {
		t.Fatalf("parse content-type: %v", err)
	}
	if mediaType != "multipart/alternative" {
		t.Fatalf("media type = %q, want multipart/alternative", mediaType)
	}
	return hdr, multipart.NewReader(r, params["boundary"])
}

func TestBuildMIMEMessage_Structure(t *testing.T) {
	s := &Service{config: Config{FromAddress: "noreply@acme.com"}}
	raw, err := s.buildMIMEMessage("user@x.com", "Password Reset Request", "TEXT BODY link=x", "<p>HTML BODY link=x</p>")
	if err != nil {
		t.Fatalf("buildMIMEMessage: %v", err)
	}

	hdr, mr := parseMessage(t, raw)
	if hdr.Get("From") != "noreply@acme.com" {
		t.Errorf("From = %q", hdr.Get("From"))
	}
	if hdr.Get("To") != "user@x.com" {
		t.Errorf("To = %q", hdr.Get("To"))
	}
	if hdr.Get("Subject") != "Password Reset Request" {
		t.Errorf("Subject = %q", hdr.Get("Subject"))
	}

	// Part 1 must be text/plain, part 2 text/html.
	wantTypes := []string{"text/plain", "text/html"}
	wantBodies := []string{"TEXT BODY", "HTML BODY"}
	for i := 0; ; i++ {
		p, err := mr.NextPart()
		if err == io.EOF {
			if i != 2 {
				t.Fatalf("got %d parts, want 2", i)
			}
			break
		}
		if err != nil {
			t.Fatalf("next part: %v", err)
		}
		mt, _, _ := mime.ParseMediaType(p.Header.Get("Content-Type"))
		if mt != wantTypes[i] {
			t.Errorf("part %d type = %q, want %q", i, mt, wantTypes[i])
		}
		if enc := p.Header.Get("Content-Transfer-Encoding"); enc != "quoted-printable" {
			t.Errorf("part %d CTE = %q, want quoted-printable", i, enc)
		}
		decoded, _ := io.ReadAll(quotedprintable.NewReader(p))
		if !strings.Contains(string(decoded), wantBodies[i]) {
			t.Errorf("part %d body missing %q; got %q", i, wantBodies[i], decoded)
		}
	}
}

func TestBuildMIMEMessage_FromNameAndReplyTo(t *testing.T) {
	s := &Service{config: Config{FromAddress: "noreply@acme.com", FromName: "ACME IT", ReplyTo: "help@acme.com"}}
	raw, err := s.buildMIMEMessage("user@x.com", "Sub", "t", "<p>h</p>")
	if err != nil {
		t.Fatalf("buildMIMEMessage: %v", err)
	}
	hdr, _ := parseMessage(t, raw)
	if hdr.Get("From") != `"ACME IT" <noreply@acme.com>` {
		t.Errorf("From = %q", hdr.Get("From"))
	}
	if hdr.Get("Reply-To") != "help@acme.com" {
		t.Errorf("Reply-To = %q", hdr.Get("Reply-To"))
	}
}

func TestBuildMIMEMessage_OverridePrecedence(t *testing.T) {
	s := &Service{config: Config{
		FromAddress:     "noreply@acme.com",
		FromName:        "ACME IT",
		HeaderOverrides: map[string]string{"From": "Custom <c@acme.com>", "X-HelpDesk-Topic": "reset"},
	}}
	raw, err := s.buildMIMEMessage("user@x.com", "Sub", "t", "<p>h</p>")
	if err != nil {
		t.Fatalf("buildMIMEMessage: %v", err)
	}
	hdr, _ := parseMessage(t, raw)
	if hdr.Get("From") != "Custom <c@acme.com>" {
		t.Errorf("From override not applied: %q", hdr.Get("From"))
	}
	if hdr.Get("X-Helpdesk-Topic") != "reset" {
		t.Errorf("routing header missing: %q", hdr.Get("X-Helpdesk-Topic"))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/email/ -run TestBuildMIMEMessage -v`
Expected: FAIL — `s.buildMIMEMessage` undefined.

- [ ] **Step 3: Implement the message builder**

Create `internal/email/message.go`:

```go
package email

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"mime/quotedprintable"
	"net/textproto"
)

// buildMIMEMessage assembles a multipart/alternative message (plain-text part
// first, HTML part second) with quoted-printable bodies, and returns the raw
// RFC 5322 message bytes. Header order: From, To, Subject, Reply-To, then
// operator overrides (applied last), then the structural MIME headers.
func (s *Service) buildMIMEMessage(to, subject, textBody, htmlBody string) ([]byte, error) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)

	if err := writeQPPart(mw, "text/plain; charset=UTF-8", textBody); err != nil {
		return nil, fmt.Errorf("write text part: %w", err)
	}
	if err := writeQPPart(mw, "text/html; charset=UTF-8", htmlBody); err != nil {
		return nil, fmt.Errorf("write html part: %w", err)
	}
	if err := mw.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	fields := []headerField{
		{key: "From", value: formatFrom(s.config.FromName, s.config.FromAddress)},
		{key: "To", value: to},
		{key: "Subject", value: encodeSubject(subject)},
	}
	if s.config.ReplyTo != "" {
		fields = append(fields, headerField{key: "Reply-To", value: s.config.ReplyTo})
	}
	fields = applyHeaderOverrides(fields, s.config.HeaderOverrides)
	fields = append(fields,
		headerField{key: "MIME-Version", value: "1.0"},
		headerField{key: "Content-Type", value: `multipart/alternative; boundary="` + mw.Boundary() + `"`},
	)

	var msg bytes.Buffer
	for _, f := range fields {
		fmt.Fprintf(&msg, "%s: %s\r\n", f.key, f.value)
	}
	msg.WriteString("\r\n")
	msg.Write(body.Bytes())

	return msg.Bytes(), nil
}

// writeQPPart writes one quoted-printable-encoded MIME part.
func writeQPPart(mw *multipart.Writer, contentType, content string) error {
	h := textproto.MIMEHeader{}
	h.Set("Content-Type", contentType)
	h.Set("Content-Transfer-Encoding", "quoted-printable")
	pw, err := mw.CreatePart(h)
	if err != nil {
		return err
	}
	qw := quotedprintable.NewWriter(pw)
	if _, err := qw.Write([]byte(content)); err != nil {
		return err
	}
	return qw.Close()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/email/ -run TestBuildMIMEMessage -v`
Expected: PASS (structure, From-name/Reply-To, override precedence).

- [ ] **Step 5: Commit**

```bash
git add internal/email/message.go internal/email/message_internal_test.go
git commit -s -m "$(cat <<'EOF'
feat(email): assemble multipart/alternative reset emails

Build text-first + HTML multipart messages with quoted-printable bodies,
ordered headers, and operator overrides applied last. The SMTP envelope
sender is unaffected by a From-header override.

Refs #627

Claude-Session: https://claude.ai/code/session_01GYyjR27xvXkN8aefvbWo8E
EOF
)"
```

---

## Task 4: Finalize Config + fail-fast NewService + SendResetEmail rewrite

**Files:**
- Modify: `internal/email/service.go`
- Modify: `internal/email/service_internal_test.go` (rewrite for removed methods + new signature)
- Modify: `internal/email/service_test.go` (external `email_test` package — `NewService` two-return)

**Interfaces:**
- Consumes: `newRenderer` (Task 1), `buildMIMEMessage` (Task 3).
- Produces:
  - Final `Config` struct (all fields).
  - `func NewService(config *Config) (*Service, error)`
  - `Service{ config Config; renderer *renderer }`
  - unchanged: `func (s *Service) SendResetEmail(to, token string) error`, `func (s *Service) buildResetLink(token string) string`, `func ValidateEmailAddress(email string) bool`

- [ ] **Step 1: Rewrite `service.go`**

Replace the whole file body (keep the package doc comment) with:

```go
// Package email provides SMTP email functionality for sending password reset tokens.
package email

import (
	"fmt"
	"net/smtp"
	"regexp"
	"strings"
)

// emailRegex is compiled once for performance.
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// Config holds the configuration for the email service.
type Config struct {
	SMTPHost     string // SMTP server hostname (e.g., smtp.gmail.com)
	SMTPPort     int    // SMTP server port (e.g., 587 for STARTTLS)
	SMTPUsername string // SMTP authentication username
	SMTPPassword string // SMTP authentication password
	FromAddress  string // Email sender address (also the SMTP envelope sender)
	FromName     string // Optional From display name
	ReplyTo      string // Optional Reply-To address
	BaseURL      string // Base URL for reset links (e.g., https://password.example.com)

	ExpiryMinutes uint // Token validity in minutes, surfaced to templates

	SubjectTemplate  string            // Inline subject template; empty => default
	TemplateHTMLPath string            // Path to custom HTML body template; empty => embedded default
	TemplateTextPath string            // Path to custom text body template; empty => embedded default
	HeaderOverrides  map[string]string // Raw header overrides (name => verbatim value)
}

// Service handles sending password reset emails.
type Service struct {
	config   Config
	renderer *renderer
}

// NewService creates an email service, loading and validating templates.
// It fails fast: a missing, unparseable, or field-invalid template returns an
// error rather than deferring the failure to the first send.
func NewService(config *Config) (*Service, error) {
	r, err := newRenderer(config)
	if err != nil {
		return nil, fmt.Errorf("initialize email templates: %w", err)
	}
	return &Service{config: *config, renderer: r}, nil
}

// SendResetEmail renders and sends a password reset email with a token link.
func (s *Service) SendResetEmail(to, token string) error {
	if !ValidateEmailAddress(to) {
		return fmt.Errorf("invalid email address: %s", to)
	}

	data := resetEmailData{
		ResetLink:     s.buildResetLink(token),
		Token:         token,
		BaseURL:       strings.TrimSuffix(s.config.BaseURL, "/"),
		Recipient:     to,
		ExpiryMinutes: s.config.ExpiryMinutes,
	}

	subject, textBody, htmlBody, err := s.renderer.render(data)
	if err != nil {
		return fmt.Errorf("render reset email: %w", err)
	}

	msg, err := s.buildMIMEMessage(to, subject, textBody, htmlBody)
	if err != nil {
		return fmt.Errorf("build reset email: %w", err)
	}

	return s.sendEmail(to, msg)
}

// sendEmail sends a pre-built message via SMTP.
func (s *Service) sendEmail(to string, msg []byte) error {
	addr := fmt.Sprintf("%s:%d", s.config.SMTPHost, s.config.SMTPPort)

	var auth smtp.Auth
	if s.config.SMTPUsername != "" {
		auth = smtp.PlainAuth("", s.config.SMTPUsername, s.config.SMTPPassword, s.config.SMTPHost)
	}

	if err := smtp.SendMail(addr, auth, s.config.FromAddress, []string{to}, msg); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// buildResetLink constructs the password reset URL with token.
func (s *Service) buildResetLink(token string) string {
	baseURL := strings.TrimSuffix(s.config.BaseURL, "/")
	return fmt.Sprintf("%s/reset-password?token=%s", baseURL, token)
}

// ValidateEmailAddress performs basic email validation.
func ValidateEmailAddress(email string) bool {
	if email == "" {
		return false
	}
	return emailRegex.MatchString(email)
}
```

> This deletes `buildResetEmailBody` and the old `buildEmailMessage`. The three `Config` template fields possibly added in Task 1/3 are now part of this canonical struct — remove any duplicate you added earlier.

- [ ] **Step 2: Rewrite `service_internal_test.go`**

The old tests target deleted methods (`buildResetEmailBody`, old `buildEmailMessage`) and the old `NewService` signature. Replace the file with tests for the surviving/behavioural surface:

```go
//nolint:testpackage // tests internal functions
package email

import (
	"strings"
	"testing"
)

func newTestService(t *testing.T, cfg Config) *Service {
	t.Helper()
	s, err := NewService(&cfg)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return s
}

func TestNewService_ConfigStored(t *testing.T) {
	s := newTestService(t, Config{SMTPHost: "smtp.example.com", FromAddress: "noreply@example.com"})
	if s.config.SMTPHost != "smtp.example.com" {
		t.Errorf("SMTPHost = %q", s.config.SMTPHost)
	}
	if s.renderer == nil {
		t.Error("renderer not initialized")
	}
}

func TestNewService_BrokenTemplateFailsFast(t *testing.T) {
	if _, err := NewService(&Config{SubjectTemplate: "{{ .Nope "}); err == nil {
		t.Fatal("expected error for unparseable subject template")
	}
}

func TestBuildResetLink(t *testing.T) {
	s := newTestService(t, Config{BaseURL: "https://example.com"})
	if got := s.buildResetLink("test-token-123"); got != "https://example.com/reset-password?token=test-token-123" {
		t.Errorf("buildResetLink = %q", got)
	}
}

func TestBuildResetLinkWithTrailingSlash(t *testing.T) {
	s := newTestService(t, Config{BaseURL: "https://example.com/"})
	if got := s.buildResetLink("test-token-123"); got != "https://example.com/reset-password?token=test-token-123" {
		t.Errorf("buildResetLink = %q", got)
	}
}

func TestSendResetEmail_RejectsInvalidAddress(t *testing.T) {
	s := newTestService(t, Config{SMTPHost: "localhost", SMTPPort: 1025, FromAddress: "noreply@example.com", BaseURL: "https://example.com", ExpiryMinutes: 15})
	err := s.SendResetEmail("not-an-email", "token123")
	if err == nil || !strings.Contains(err.Error(), "invalid email") {
		t.Errorf("expected invalid-email error, got %v", err)
	}
}

func TestSendResetEmail_RendersExpiryFromConfig(t *testing.T) {
	// Render path is exercised via buildMIMEMessage in message tests; here confirm
	// the data wiring by rendering directly through the renderer.
	s := newTestService(t, Config{BaseURL: "https://example.com", ExpiryMinutes: 42})
	_, text, _, err := s.renderer.render(resetEmailData{
		ResetLink:     s.buildResetLink("tok"),
		ExpiryMinutes: s.config.ExpiryMinutes,
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(text, "42 minutes") {
		t.Errorf("text body missing configured expiry; got %q", text)
	}
}
```

> The comprehensive `TestValidateEmailAddress`, `TestCaseSensitivityHandling`, and `TestDomainValidationEdgeCases` tables currently live in `service_internal_test.go`. **Preserve them** — copy those three functions verbatim into the rewritten file (they test `ValidateEmailAddress`, which is unchanged). Do not drop coverage.

- [ ] **Step 3: Fix `service_test.go` (external package) call sites**

`internal/email/service_test.go` is `package email_test` and calls `email.NewService(&config)` at 3 sites. Update each to the two-return form. Example for `TestNewService`:

```go
	service, err := email.NewService(&config)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	if service == nil {
		t.Fatal("NewService() returned nil")
	}
```

For the other two call sites (the `buildResetEmailBody`-based assertions), those methods no longer exist and are not accessible from an external package anyway — **delete any test in `service_test.go` that calls `service.buildResetEmailBody` or `service.buildEmailMessage`** (they were duplicated internal-only tests). Keep only what compiles against the exported surface (`SendResetEmail`, `ValidateEmailAddress`). Verify by reading the file first.

- [ ] **Step 4: Run email package tests**

Run: `go test ./internal/email/ -v`
Expected: PASS (all rewritten + preserved tests). No references to deleted methods remain.

- [ ] **Step 5: Commit**

```bash
git add internal/email/service.go internal/email/service_internal_test.go internal/email/service_test.go
git commit -s -m "$(cat <<'EOF'
feat(email)!: fail-fast NewService and template-rendered reset emails

Expand Config with sender name, Reply-To, expiry, template paths and header
overrides. NewService now returns an error and validates templates at
startup; SendResetEmail renders via the template engine and sends multipart.
Removes the hardcoded body/subject builders.

BREAKING CHANGE: email.NewService now returns (*Service, error).

Refs #627

Claude-Session: https://claude.ai/code/session_01GYyjR27xvXkN8aefvbWo8E
EOF
)"
```

---

## Task 5: Options — flags, env, and header-override prefix scan

**Files:**
- Modify: `internal/options/app.go`
- Test: `internal/options/app_test.go`

**Interfaces:**
- Consumes: `email.ValidateHeaderName`, `email.ValidateHeaderValue`, `email.ValidateEmailAddress` (Task 2 / existing).
- Produces: new `Opts` fields — `SMTPFromName, EmailReplyTo, EmailTemplateHTML, EmailTemplateText, EmailTemplateSubject string`; `SMTPHeaderOverrides map[string]string`; helper `func parseHeaderOverrides(environ []string, errs *ConfigError) map[string]string`.

- [ ] **Step 1: Write the failing test**

Add to `internal/options/app_test.go` (match existing test style; these use `t.Setenv` + `ParseArgs` with the required LDAP args). Include the minimal required flags so parsing succeeds:

```go
func requiredArgs() []string {
	return []string{
		"--ldap-server", "ldaps://ldap.example.com",
		"--base-dn", "dc=example,dc=com",
		"--readonly-user", "cn=ro,dc=example,dc=com",
		"--readonly-password", "secret",
	}
}

func TestParseArgs_EmailTemplateOptions(t *testing.T) {
	t.Setenv("SMTP_FROM_NAME", "ACME IT")
	t.Setenv("EMAIL_REPLY_TO", "help@acme.com")
	t.Setenv("EMAIL_TEMPLATE_SUBJECT", "[ACME] Reset")
	t.Setenv("EMAIL_TEMPLATE_HTML", "/config/reset.html")
	t.Setenv("EMAIL_TEMPLATE_TEXT", "/config/reset.txt")

	opts, err := ParseArgs(requiredArgs())
	if err != nil {
		t.Fatalf("ParseArgs: %v", err)
	}
	if opts.SMTPFromName != "ACME IT" {
		t.Errorf("SMTPFromName = %q", opts.SMTPFromName)
	}
	if opts.EmailReplyTo != "help@acme.com" {
		t.Errorf("EmailReplyTo = %q", opts.EmailReplyTo)
	}
	if opts.EmailTemplateSubject != "[ACME] Reset" {
		t.Errorf("EmailTemplateSubject = %q", opts.EmailTemplateSubject)
	}
	if opts.EmailTemplateHTML != "/config/reset.html" || opts.EmailTemplateText != "/config/reset.txt" {
		t.Errorf("template paths = %q / %q", opts.EmailTemplateHTML, opts.EmailTemplateText)
	}
}

func TestParseArgs_InvalidReplyTo(t *testing.T) {
	t.Setenv("EMAIL_REPLY_TO", "not-an-email")
	_, err := ParseArgs(requiredArgs())
	if err == nil || !strings.Contains(err.Error(), "EMAIL_REPLY_TO") {
		t.Fatalf("expected EMAIL_REPLY_TO validation error, got %v", err)
	}
}

func TestParseArgs_HeaderOverrides(t *testing.T) {
	t.Setenv("SMTP_HEADER_OVERRIDE_X_HELPDESK_TOPIC", "password-reset")
	t.Setenv("SMTP_HEADER_OVERRIDE_LIST_UNSUBSCRIBE", "<mailto:unsub@acme.com>")

	opts, err := ParseArgs(requiredArgs())
	if err != nil {
		t.Fatalf("ParseArgs: %v", err)
	}
	if opts.SMTPHeaderOverrides["X-HELPDESK-TOPIC"] != "password-reset" {
		t.Errorf("override map = %#v", opts.SMTPHeaderOverrides)
	}
	if opts.SMTPHeaderOverrides["LIST-UNSUBSCRIBE"] != "<mailto:unsub@acme.com>" {
		t.Errorf("override map = %#v", opts.SMTPHeaderOverrides)
	}
}

func TestParseArgs_HeaderOverrideRejectsCRLFAndReserved(t *testing.T) {
	t.Run("crlf", func(t *testing.T) {
		t.Setenv("SMTP_HEADER_OVERRIDE_X_EVIL", "a\r\nInjected: yes")
		if _, err := ParseArgs(requiredArgs()); err == nil {
			t.Fatal("expected CRLF rejection")
		}
	})
	t.Run("reserved", func(t *testing.T) {
		t.Setenv("SMTP_HEADER_OVERRIDE_CONTENT_TYPE", "text/plain")
		if _, err := ParseArgs(requiredArgs()); err == nil {
			t.Fatal("expected reserved-header rejection")
		}
	})
}
```

> Confirm `app_test.go` imports `strings`. If not, add it. If a `requiredArgs()` helper already exists in the test file, reuse it instead of redefining.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/options/ -run 'TestParseArgs_Email|TestParseArgs_Invalid|TestParseArgs_Header' -v`
Expected: FAIL — undefined `opts.SMTPFromName`, etc.

- [ ] **Step 3: Add imports + Opts fields**

In `internal/options/app.go` imports, add:

```go
	"net/textproto"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/email"
```

Add to the `Opts` struct (after `AppBaseURL`):

```go
	SMTPFromName         string
	EmailReplyTo         string
	EmailTemplateHTML    string
	EmailTemplateText    string
	EmailTemplateSubject string
	SMTPHeaderOverrides  map[string]string
```

- [ ] **Step 4: Add the flags**

In `ParseArgs`, alongside the other SMTP flags, add:

```go
		fSMTPFromName = fs.String(
			"smtp-from-name",
			envStringOrDefault("SMTP_FROM_NAME", ""),
			"Optional display name for the From header (e.g. \"ACME IT\").",
		)
		fEmailReplyTo = fs.String(
			"email-reply-to",
			envStringOrDefault("EMAIL_REPLY_TO", ""),
			"Optional Reply-To address for reset emails.",
		)
		fEmailTemplateHTML = fs.String(
			"email-template-html",
			envStringOrDefault("EMAIL_TEMPLATE_HTML", ""),
			"Path to a custom HTML email body template (Go template). Empty uses the built-in default.",
		)
		fEmailTemplateText = fs.String(
			"email-template-text",
			envStringOrDefault("EMAIL_TEMPLATE_TEXT", ""),
			"Path to a custom plain-text email body template (Go template). Empty uses the built-in default.",
		)
		fEmailTemplateSubject = fs.String(
			"email-template-subject",
			envStringOrDefault("EMAIL_TEMPLATE_SUBJECT", ""),
			"Subject line template (Go template). Empty uses \"Password Reset Request\".",
		)
```

- [ ] **Step 5: Add the prefix-scan helper + reserved set**

Add near the other helpers in `app.go`:

```go
const headerOverridePrefix = "SMTP_HEADER_OVERRIDE_"

// reservedHeaderOverride lists structural MIME headers that must not be
// overridden, keyed by canonical form.
var reservedHeaderOverride = map[string]bool{
	"Mime-Version":              true,
	"Content-Type":              true,
	"Content-Transfer-Encoding": true,
}

// parseHeaderOverrides scans environ for SMTP_HEADER_OVERRIDE_* entries and
// builds a header map (suffix "_" => "-"). Invalid names/values and reserved
// structural headers are reported into errs (fail-fast). The map key is the
// hyphenated header name; the email layer canonicalizes on use.
func parseHeaderOverrides(environ []string, errs *ConfigError) map[string]string {
	overrides := map[string]string{}
	for _, kv := range environ {
		eq := strings.IndexByte(kv, '=')
		if eq < 0 {
			continue
		}
		name := kv[:eq]
		if !strings.HasPrefix(name, headerOverridePrefix) {
			continue
		}
		suffix := name[len(headerOverridePrefix):]
		if suffix == "" {
			continue
		}
		value := kv[eq+1:]
		headerName := strings.ReplaceAll(suffix, "_", "-")

		if err := email.ValidateHeaderName(headerName); err != nil {
			errs.Add(fmt.Sprintf("invalid %s: %v", name, err))
			continue
		}
		if err := email.ValidateHeaderValue(value); err != nil {
			errs.Add(fmt.Sprintf("invalid value for %s: %v", name, err))
			continue
		}
		if reservedHeaderOverride[textproto.CanonicalMIMEHeaderKey(headerName)] {
			errs.Add(fmt.Sprintf("%s: cannot override structural MIME header", name))
			continue
		}
		overrides[headerName] = value
	}
	return overrides
}
```

- [ ] **Step 6: Wire validation + map into the returned Opts**

After the reset-identifier-mode validation block, add Reply-To validation and the scan:

```go
	if *fEmailReplyTo != "" && !email.ValidateEmailAddress(*fEmailReplyTo) {
		errs.Add(fmt.Sprintf("invalid value for EMAIL_REPLY_TO: %q is not a valid email address", *fEmailReplyTo))
	}

	headerOverrides := parseHeaderOverrides(os.Environ(), errs)
```

Add the new fields to the returned `&Opts{...}` literal (after `AppBaseURL: *fAppBaseURL,`):

```go
		SMTPFromName:         *fSMTPFromName,
		EmailReplyTo:         *fEmailReplyTo,
		EmailTemplateHTML:    *fEmailTemplateHTML,
		EmailTemplateText:    *fEmailTemplateText,
		EmailTemplateSubject: *fEmailTemplateSubject,
		SMTPHeaderOverrides:  headerOverrides,
```

> **Import-cycle check:** `email` must not import `options`. It does not (verified). This dependency direction (options → email) is one-way and safe.

- [ ] **Step 7: Run test to verify it passes**

Run: `go test ./internal/options/ -v`
Expected: PASS (new tests + existing).

- [ ] **Step 8: Commit**

```bash
git add internal/options/app.go internal/options/app_test.go
git commit -s -m "$(cat <<'EOF'
feat(options): add email template + header-override configuration

Add EMAIL_TEMPLATE_{HTML,TEXT,SUBJECT}, SMTP_FROM_NAME, EMAIL_REPLY_TO flags
and an SMTP_HEADER_OVERRIDE_* prefix scan. Reply-To and override names/values
are validated at startup; structural MIME headers cannot be overridden.

Refs #627

Claude-Session: https://claude.ai/code/session_01GYyjR27xvXkN8aefvbWo8E
EOF
)"
```

---

## Task 6: Wire into main.go + fix all NewService call sites

**Files:**
- Modify: `main.go`
- Modify: `internal/rpchandler/handler_test.go` (default build)
- Modify (tag-gated `integration`): `integration_test.go`, `internal/email/service_integration_test.go`, `internal/rpchandler/handler_integration_test.go`

**Interfaces:**
- Consumes: final `NewService(*Config) (*Service, error)` (Task 4); new `Opts` fields (Task 5).

- [ ] **Step 1: Update `buildEmailConfig` in `main.go`**

Replace `buildEmailConfig` (`main.go:67-78`) with:

```go
func buildEmailConfig(opts *options.Opts) email.Config {
	// Safe conversion: SMTPPort is uint, typically 25/587/465 (well within int range)
	smtpPort := int(opts.SMTPPort) //#nosec G115 -- SMTPPort is 0-65535, safe for int
	return email.Config{
		SMTPHost:         opts.SMTPHost,
		SMTPPort:         smtpPort,
		SMTPUsername:     opts.SMTPUsername,
		SMTPPassword:     opts.SMTPPassword,
		FromAddress:      opts.SMTPFromAddress,
		FromName:         opts.SMTPFromName,
		ReplyTo:          opts.EmailReplyTo,
		BaseURL:          opts.AppBaseURL,
		ExpiryMinutes:    opts.ResetTokenExpiryMinutes,
		SubjectTemplate:  opts.EmailTemplateSubject,
		TemplateHTMLPath: opts.EmailTemplateHTML,
		TemplateTextPath: opts.EmailTemplateText,
		HeaderOverrides:  opts.SMTPHeaderOverrides,
	}
}
```

- [ ] **Step 2: Handle the `NewService` error in `newHandlerWithResetServices`**

Replace `main.go:102-103`:

```go
	// Initialize email service
	emailConfig := buildEmailConfig(opts)
	emailService := email.NewService(&emailConfig)
```

with:

```go
	// Initialize email service (fails fast on bad templates/config).
	emailConfig := buildEmailConfig(opts)
	emailService, err := email.NewService(&emailConfig)
	if err != nil {
		return nil, fmt.Errorf("initialize email service: %w", err)
	}
```

> The function already declares `err` later via `h, err := ...`. Because this new `err` is introduced with `:=` before that line, change the later `h, err := rpchandler.NewWithServices(...)` to `h, err = ...` (no colon) to avoid a "no new variables on left side of :=" — OR keep `:=` there if `h` is new (it is). Go allows `:=` as long as at least one var on the left is new (`h`). So leaving the later line as `h, err :=` is fine. Verify compile.

- [ ] **Step 3: Fix `handler_test.go:65` (default build)**

Change:

```go
	emailService := email.NewService(emailConfig)
```

to:

```go
	emailService, err := email.NewService(emailConfig)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
```

> Read the surrounding lines first: if `err` is already in scope there, use `=`; if `emailConfig` is a value (not pointer), keep as-is (signature takes `*Config`). Confirm the existing call already passes a pointer.

- [ ] **Step 4: Fix tag-gated integration call sites**

In each of `integration_test.go:77`, `internal/email/service_integration_test.go:66,130,149`, and `internal/rpchandler/handler_integration_test.go:81`, change the `service := email.NewService(config)` form to:

```go
	service, err := email.NewService(config)
	require.NoError(t, err)
```

(Use the error-handling idiom already present in each file — `require.NoError` where testify is imported, else `if err != nil { t.Fatalf(...) }`.)

Additionally, in `service_integration_test.go` `TestIntegration_SendResetEmail`, add a multipart assertion after fetching the message:

```go
	contentTypes := foundMessage.Content.Headers["Content-Type"]
	require.NotEmpty(t, contentTypes)
	assert.Contains(t, contentTypes[0], "multipart/alternative", "reset email should be multipart")
```

> The existing `assert.Contains(t, foundMessage.Content.Body, testToken, ...)` still holds: the token contains no `=`, so quoted-printable does not alter that substring.

- [ ] **Step 5: Build the whole repo (both tag sets)**

Run:
```bash
go build ./...
go vet ./...
go test ./...
go test -tags integration ./... 2>&1 | head -40   # compiles integration files; tests skip without SMTP_HOST
```
Expected: `go build`, `go vet`, `go test ./...` all PASS. The `-tags integration` run compiles cleanly (individual tests may `Skip` without `SMTP_HOST`/Docker — skipping is acceptable, a compile error is not).

- [ ] **Step 6: Commit**

```bash
git add main.go internal/rpchandler/handler_test.go internal/rpchandler/handler_integration_test.go internal/email/service_integration_test.go integration_test.go
git commit -s -m "$(cat <<'EOF'
feat(main): wire email template config and fail-fast service init

Map the new email template/header options into email.Config and treat a
NewService error as fatal at startup. Update all NewService call sites for
the new (*Service, error) signature; assert multipart in the integration test.

Refs #627

Claude-Session: https://claude.ai/code/session_01GYyjR27xvXkN8aefvbWo8E
EOF
)"
```

---

## Task 7: Documentation

**Files:**
- Modify: `.env.local.example`
- Modify: `internal/CLAUDE.md`
- Modify: any `docs/*` config reference (grep first)

- [ ] **Step 1: Extend `.env.local.example`**

Append after the SMTP block (`APP_BASE_URL=` line):

```bash
# --- Email templates & headers (optional) ------------------------------
# Customize the reset email. Unset => built-in defaults.
# SMTP_FROM_NAME=ACME IT
# EMAIL_REPLY_TO=helpdesk@example.com
# EMAIL_TEMPLATE_SUBJECT=[ACME] Reset your password
# EMAIL_TEMPLATE_HTML=/config/email/reset.html   # Go template, {{.ResetLink}} etc.
# EMAIL_TEMPLATE_TEXT=/config/email/reset.txt
# Raw header injection for routing/helpdesk integrations. One var per header;
# the suffix maps _ -> - (e.g. below sets "X-HelpDesk-Topic"). CR/LF is rejected;
# MIME-Version / Content-Type / Content-Transfer-Encoding cannot be overridden.
# SMTP_HEADER_OVERRIDE_X_HELPDESK_TOPIC=password-reset
# Template fields: {{.ResetLink}} {{.Token}} {{.BaseURL}} {{.Recipient}} {{.ExpiryMinutes}}
```

- [ ] **Step 2: Update `internal/CLAUDE.md`**

In the "Required environment variables" block, add under the email section:

```bash
# Email templating (optional; unset => built-in defaults)
SMTP_FROM_NAME=ACME IT
EMAIL_REPLY_TO=helpdesk@example.com
EMAIL_TEMPLATE_SUBJECT=[ACME] Reset your password
EMAIL_TEMPLATE_HTML=/config/email/reset.html
EMAIL_TEMPLATE_TEXT=/config/email/reset.txt
# Raw header escape hatch (suffix _ -> -): SMTP_HEADER_OVERRIDE_X_HELPDESK_TOPIC=...
```

Add a one-line note under `email/` in "Package-Specific Notes": that `NewService` now returns `(*Service, error)` and validates templates at startup; template fields are `ResetLink, Token, BaseURL, Recipient, ExpiryMinutes`.

- [ ] **Step 3: Sweep docs/ for config references**

Run: `grep -rln "SMTP_FROM_ADDRESS\|APP_BASE_URL\|Password Reset Request" docs/ README.md 2>/dev/null`
For each hit that lists configuration or describes the email, add the new options consistently with the surrounding style. If none list per-var config, skip (do not invent a new doc).

- [ ] **Step 4: Format + final verification**

Run:
```bash
bunx prettier --write .
go build ./...
go test ./...
go vet ./...
```
Expected: all green; prettier leaves only intended changes.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -s -m "$(cat <<'EOF'
docs(email): document custom email template and header configuration

Add the new EMAIL_TEMPLATE_*, SMTP_FROM_NAME, EMAIL_REPLY_TO and
SMTP_HEADER_OVERRIDE_* options to .env.local.example and internal docs.

Refs #627

Claude-Session: https://claude.ai/code/session_01GYyjR27xvXkN8aefvbWo8E
EOF
)"
```

---

## Self-Review

**Spec coverage:**
- Multipart HTML+text → Task 1 (templates/renderer) + Task 3 (assembly). ✓
- File-path override + embedded defaults → Task 1 (`loadTemplateSource`). ✓
- `EMAIL_TEMPLATE_*` naming + subject inline template → Task 5 + Task 1. ✓
- `SMTP_FROM_NAME` (encoded via net/mail) + `EMAIL_REPLY_TO` (validated) → Task 2 + Task 5 + Task 3. ✓
- `SMTP_HEADER_OVERRIDE_*` prefix scan, `_`→`-`, verbatim value, CRLF+field-name guard, reserved-header block, precedence last, envelope unchanged → Task 2 + Task 3 + Task 5. ✓
- Fail-fast startup validation (parse + dry-run + config errors) → Task 1 + Task 4 + Task 5. ✓
- Data contract (`ResetLink, Token, BaseURL, Recipient, ExpiryMinutes`) + fixes hardcoded expiry → Task 1 + Task 4. ✓
- Testing (unit/options/integration/fuzz) → Tasks 1–6. ✓
- Docs → Task 7. ✓
- Out-of-scope (`GOPHERPASS_*`, username field) → not built. ✓

**Placeholder scan:** No "TBD/TODO/handle edge cases"; every code step carries complete code.

**Type consistency:** `NewService(*Config) (*Service, error)` used identically in Tasks 4, 6, and all call-site fixes. `resetEmailData` fields identical in Tasks 1, 3, 4. `headerField{key,value}` identical in Tasks 2, 3. `Config` template fields introduced provisionally in Task 1/3 and finalized (deduplicated) in Task 4 — implementer must remove the provisional duplicates, called out explicitly.
