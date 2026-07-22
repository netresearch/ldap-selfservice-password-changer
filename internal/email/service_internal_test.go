package email

import (
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newTestService builds a Service from cfg, failing the test if NewService
// rejects it. cfg is taken by pointer because Config is large enough that
// copying it per call is wasteful, and NewService takes a pointer anyway.
func newTestService(t *testing.T, cfg *Config) *Service {
	t.Helper()
	s, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return s
}

func TestNewService_ConfigStored(t *testing.T) {
	s := newTestService(t, &Config{SMTPHost: "smtp.example.com", FromAddress: "noreply@example.com"})
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
	s := newTestService(t, &Config{BaseURL: "https://example.com"})
	if got := s.buildResetLink("test-token-123"); got != "https://example.com/reset-password?token=test-token-123" {
		t.Errorf("buildResetLink = %q", got)
	}
}

func TestBuildResetLinkWithTrailingSlash(t *testing.T) {
	s := newTestService(t, &Config{BaseURL: "https://example.com/"})
	if got := s.buildResetLink("test-token-123"); got != "https://example.com/reset-password?token=test-token-123" {
		t.Errorf("buildResetLink = %q", got)
	}
}

func TestSendResetEmail_RejectsInvalidAddress(t *testing.T) {
	s := newTestService(t, &Config{
		SMTPHost:      "localhost",
		SMTPPort:      1025,
		FromAddress:   "noreply@example.com",
		BaseURL:       "https://example.com",
		ExpiryMinutes: 15,
	})
	err := s.SendResetEmail("not-an-email", "token123")
	if err == nil || !strings.Contains(err.Error(), "invalid email") {
		t.Errorf("expected invalid-email error, got %v", err)
	}
}

// firstPartBody returns the decoded body of the first MIME part. NextPart (not
// NextRawPart) is deliberate here: it undoes the quoted-printable encoding, so
// the assertions can be written against the template output rather than its
// wire form.
func firstPartBody(t *testing.T, mr *multipart.Reader) string {
	t.Helper()
	p, err := mr.NextPart()
	if err != nil {
		t.Fatalf("read first part: %v", err)
	}
	b, err := io.ReadAll(p)
	if err != nil {
		t.Fatalf("read first part body: %v", err)
	}
	return string(b)
}

// TestBuildResetMessage_WiresConfigIntoTemplateData drives the whole
// SendResetEmail path up to the SMTP handoff and asserts every field of
// resetEmailData arrives in the rendered message carrying its configured
// value. The template emits one field per line so a hardcoded or crossed wire
// fails on the exact field. ExpiryMinutes is deliberately not 15: the bug this
// feature fixed was a body that said "15 minutes" regardless of configuration.
func TestBuildResetMessage_WiresConfigIntoTemplateData(t *testing.T) {
	textPath := filepath.Join(t.TempDir(), "body.txt")
	body := "link={{.ResetLink}}\ntoken={{.Token}}\nbase={{.BaseURL}}\n" +
		"recipient={{.Recipient}}\nexpiry={{.ExpiryMinutes}} minutes\n"
	if err := os.WriteFile(textPath, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	s := newTestService(t, &Config{
		FromAddress:      "noreply@example.com",
		BaseURL:          "https://reset.example.test/",
		ExpiryMinutes:    73,
		TemplateTextPath: textPath,
	})

	raw, err := s.buildResetMessage("user@example.com", "tok-9f3")
	if err != nil {
		t.Fatalf("buildResetMessage: %v", err)
	}

	hdr, mr := parseMessage(t, raw)
	if got := hdr.Get("To"); got != "user@example.com" {
		t.Errorf("To = %q, want user@example.com", got)
	}

	text := firstPartBody(t, mr)
	for _, want := range []string{
		"link=https://reset.example.test/reset-password?token=tok-9f3",
		"token=tok-9f3",
		"base=https://reset.example.test",
		"recipient=user@example.com",
		"expiry=73 minutes",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("rendered text body missing %q; got:\n%s", want, text)
		}
	}
}

func TestValidateEmailAddress(t *testing.T) {
	tests := []struct {
		name  string
		email string
		valid bool
	}{
		{"valid email", "user@example.com", true},
		{"valid with subdomain", "user@mail.example.com", true},
		{"valid with plus", "user+tag@example.com", true},
		{"invalid no @", "userexample.com", false},
		{"invalid no domain", "user@", false},
		{"invalid no user", "@example.com", false},
		{"invalid empty", "", false},
		{"invalid spaces", "user @example.com", false},
		{"invalid multiple @", "user@@example.com", false},
		{"invalid multiple @ separated", "user@domain@example.com", false},
		{"invalid no TLD", "user@localhost", false},
		{"invalid single letter TLD", "user@example.c", false},
		{"invalid just @", "@", false},
		{"invalid only domain", "example.com", false},
		{"leading dot (permissive)", ".user@example.com", true},               // Regex allows this
		{"trailing dot in local (permissive)", "user.@example.com", true},     // Regex allows this
		{"leading hyphen in domain (permissive)", "user@-example.com", true},  // Regex allows this
		{"trailing hyphen in domain (permissive)", "user@example-.com", true}, // Regex allows this
		{"invalid special chars", "user!#$%@example.com", false},
		{"invalid unicode", "üser@example.com", false},
		{"valid with hyphen", "user@my-domain.com", true},
		{"valid with numbers", "user123@example456.com", true},
		{"valid with dots", "first.last@example.com", true},
		{"valid with underscore", "user_name@example.com", true},
		{"valid with multiple subdomains", "user@mail.corp.example.com", true},
		{"very long local part (63 chars)", "a" + strings.Repeat("x", 62) + "@example.com", true},
		{"very long local part (64 chars)", strings.Repeat("x", 64) + "@example.com", true},
		{"very long local part (65+ chars)", strings.Repeat("x", 65) + "@example.com", true},
		{"very long domain", "user@" + strings.Repeat("a", 250) + ".com", true},
		{"maximum valid TLD length", "user@example." + strings.Repeat("a", 10), true},
		{"empty local part", "@example.com", false},
		{"empty domain part", "user@", false},
		{"whitespace in email", "user name@example.com", false},
		{"tab in email", "user\t@example.com", false},
		{"newline in email", "user\n@example.com", false},
		{"null byte", "user\x00@example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := ValidateEmailAddress(tt.email)
			if valid != tt.valid {
				t.Errorf("ValidateEmailAddress(%q) = %v, want %v", tt.email, valid, tt.valid)
			}
		})
	}
}

func TestCaseSensitivityHandling(t *testing.T) {
	// Email addresses should be treated case-insensitively for validation
	tests := []struct {
		email string
		valid bool
	}{
		{"User@Example.COM", true},
		{"USER@EXAMPLE.COM", true},
		{"user@example.com", true},
		{"User@example.COM", true},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			valid := ValidateEmailAddress(tt.email)
			if valid != tt.valid {
				t.Errorf("ValidateEmailAddress(%q) = %v, want %v", tt.email, valid, tt.valid)
			}
		})
	}
}

func TestDomainValidationEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		email string
		valid bool
	}{
		{"domain with numbers", "user@123.com", true},
		{"domain starts with number", "user@1example.com", true},
		{"all numeric domain", "user@123.456", false},                            // No TLD
		{"domain with consecutive dots (permissive)", "user@example..com", true}, // Regex allows
		{"domain ends with dot", "user@example.com.", false},
		{"domain starts with dot (permissive)", "user@.example.com", true}, // Regex allows
		{"IP address as domain", "user@192.168.1.1", false},                // Not in our regex
		{"domain too short (single char TLD)", "user@a.b", false},          // Requires 2+ char TLD
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := ValidateEmailAddress(tt.email)
			if valid != tt.valid {
				t.Errorf("ValidateEmailAddress(%q) = %v, want %v", tt.email, valid, tt.valid)
			}
		})
	}
}
