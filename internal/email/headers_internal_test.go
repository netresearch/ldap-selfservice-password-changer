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
	if got := encodeSubject("Zurücksetzen"); !strings.HasPrefix(got, "=?utf-8?q?") &&
		!strings.HasPrefix(got, "=?UTF-8?q?") {
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
	// Plain ASCII display name with no specials: mail.Address.String() ALWAYS
	// quotes an all-printable display name, so the quoted form is correct.
	// Do NOT expect `ACME <noreply@acme.com>` — that assertion would fail.
	if got := formatFrom("ACME", "noreply@acme.com"); got != `"ACME" <noreply@acme.com>` {
		t.Errorf("ascii name = %q, want quoted form", got)
	}
	// Non-ASCII display name must be RFC 2047 encoded.
	if got := formatFrom("ACME Straße", "noreply@acme.com"); !strings.Contains(got, "=?utf-8?") &&
		!strings.Contains(got, "=?UTF-8?") {
		t.Errorf("non-ASCII name not encoded: %q", got)
	}
}

func TestApplyHeaderOverrides(t *testing.T) {
	base := []headerField{
		{key: "From", value: "noreply@acme.com"},
		{key: "To", value: "u@x.com"},
	}
	out := applyHeaderOverrides(base, map[string]string{
		"from":             "ACME <help@acme.com>", // canonical-key match, replaces
		"X-HelpDesk-Topic": "reset",                // new, appended
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
