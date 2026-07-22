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
