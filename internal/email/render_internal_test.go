package email

import (
	"bytes"
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

	// Asserted against the literal text, not defaultSubjectTemplate: comparing
	// the output to the very constant the implementation substitutes cannot fail.
	if subject != "Password Reset Request" {
		t.Errorf("subject = %q, want %q", subject, "Password Reset Request")
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

// TestDefaultTemplatesContainSecurityWarning guards the security wording in the
// built-in templates. It replaces the pre-refactor TestEmailBodyContainsSecurityWarning
// and TestEmailBodyContainsExpirationInfo, which asserted the same phrases against
// the old hardcoded body. Without it, editing a default template could silently drop
// the "you didn't request this" reassurance from every reset email.
func TestDefaultTemplatesContainSecurityWarning(t *testing.T) {
	r, err := newRenderer(&Config{})
	if err != nil {
		t.Fatalf("newRenderer with defaults: %v", err)
	}

	_, text, html, err := r.render(resetEmailData{
		ResetLink:     "https://example.com/reset-password?token=abc",
		ExpiryMinutes: 15,
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	securityPhrases := []string{
		"If you didn't request",
		"safely ignore",
		"will not be changed",
	}
	for _, phrase := range securityPhrases {
		if !strings.Contains(text, phrase) {
			t.Errorf("default text body missing security phrase %q", phrase)
		}
		if !strings.Contains(html, phrase) {
			t.Errorf("default html body missing security phrase %q", phrase)
		}
	}

	// Expiry must be rendered from config, not hardcoded (the bug this feature fixed).
	if !strings.Contains(text, "15 minutes") {
		t.Errorf("default text body missing rendered expiry; got %q", text)
	}
	if !strings.Contains(html, "15 minutes") {
		t.Errorf("default html body missing rendered expiry")
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
	// A non-regular path must be rejected by the os.Stat mode check rather than
	// opened. EMAIL_TEMPLATE_HTML=/dev/zero previously stalled startup before
	// the listener bound and grew until the OOM killer fired; a directory is the
	// portable stand-in, and reaching os.Open would yield a read error whose
	// message does not name the real cause.
	t.Run("non-regular file rejected", func(t *testing.T) {
		_, err := newRenderer(&Config{TemplateTextPath: t.TempDir()})
		if err == nil {
			t.Fatal("expected error for a directory template path")
		}
		if !strings.Contains(err.Error(), "is not a regular file") {
			t.Errorf("error = %v, want it to report a non-regular file", err)
		}
	})
	t.Run("oversized file rejected", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "big.txt")
		// One byte over the cap: enough to trip the check, cheap to write.
		if err := os.WriteFile(p, bytes.Repeat([]byte("x"), maxTemplateBytes+1), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := newRenderer(&Config{TemplateTextPath: p})
		if err == nil {
			t.Fatal("expected error for a template file over the size cap")
		}
		if !strings.Contains(err.Error(), "exceeding the") {
			t.Errorf("error = %v, want it to report the size limit", err)
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
