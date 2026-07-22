//nolint:testpackage // tests internal functions
package email

import (
	"bufio"
	"net/textproto"
	"strings"
	"testing"
)

// FuzzValidateEmailAddress fuzzes the ValidateEmailAddress function.
func FuzzValidateEmailAddress(f *testing.F) {
	// Seed corpus with various email patterns
	seeds := []string{
		"",
		"user@example.com",
		"user",
		"@example.com",
		"user@",
		"user@domain",
		"user@domain.com",
		"user+tag@example.com",
		"user.name@sub.example.com",
		"very-long-email-address-that-exceeds-reasonable-length@very-long-domain-name-that-exceeds-reasonable-length.com",
		"user@localhost",
		"user@127.0.0.1",
		"user@[127.0.0.1]",
		"<script>@example.com",
		"' OR '1'='1@example.com",
		"user\x00@example.com",
		"user\n@example.com",
		"user\t@example.com",
		"用户@例子.公司",
		"🔐@emoji.com",
		"user@例子.com",
		"a@b.co",
		"a@b.c",
	}

	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, email string) {
		// The function should not panic for any input
		result := ValidateEmailAddress(email)

		// Verify basic contract: empty email should always be invalid
		if email == "" && result {
			t.Error("Empty email should be invalid")
		}

		// If valid, it should contain @ and .
		if result {
			hasAt := false
			hasDot := false
			for _, c := range email {
				if c == '@' {
					hasAt = true
				}
				if c == '.' {
					hasDot = true
				}
			}
			if !hasAt {
				t.Errorf("Valid email %q should contain @", email)
			}
			if !hasDot {
				t.Errorf("Valid email %q should contain .", email)
			}
		}
	})
}

// baseHeaderFields are the fields buildMIMEMessage always emits for a Service
// configured with only a FromAddress (no FromName, no Reply-To). Canonical
// casing, because header keys are compared after canonicalisation.
var baseHeaderFields = []string{"From", "To", "Subject", "Mime-Version", "Content-Type"}

// FuzzHeaderOverrideValidation fuzzes the operator header-override path.
//
// The oracle is deliberately *not* a re-implementation of the validators —
// re-deriving "does this contain CR/LF?" on both sides can never fail. It is a
// structural property of the message the builder produces:
//
//   - Input the validators reject must make buildMIMEMessage fail; it must
//     never hand back a message.
//   - Input the validators accept must produce a header block holding exactly
//     the fields the builder was asked for and nothing more. An override that
//     smuggles in an extra header line fails this property no matter which
//     byte the validators let through, so weakening ValidateHeaderValue is
//     detected here rather than silently mirrored.
func FuzzHeaderOverrideValidation(f *testing.F) {
	seeds := []struct{ name, value string }{
		{"X-HelpDesk-Topic", "reset"},
		{"", ""},
		{"X Bad", "value"},
		{"X-Inject", "a\r\nEvil: yes"},
		{"X-Inject", "a\nBcc: attacker@evil.example"},
		{"X-Inject", "a\rBcc: attacker@evil.example"},
		{"X-Inject", "a\r\n\r\nsmuggled body"},
		{"X-Inject", "a\n\nsmuggled body"},
		{"X-Inject", "a\r\n\tfolded"},
		{"X-NUL", "a\x00b"},
		{"X-DEL", "a\x7fb"},
		{"X-Tab", "a\tb"},
		{"X-Empty", ""},
		{"X-Colon", "a: b"},
		{"X-Unicode", "Grüße"},
		{"Naïve", "value"},
		{"Subject", "operator subject"},
		{"From", "attacker@evil.example"},
		{"Reply-To", "helpdesk@acme.com"},
		{"mime-version", "9.9"},
		{"Content-Type", "text/plain"},
		{"X-Long", strings.Repeat("a", 2000)},
	}
	for _, s := range seeds {
		f.Add(s.name, s.value)
	}

	f.Fuzz(func(t *testing.T, name, value string) {
		nameErr := ValidateHeaderName(name) // must not panic
		valueErr := ValidateHeaderValue(value)

		// The builder owns the structural MIME headers and drops an override of
		// them before validating, so they carry no message-level property here.
		canonical := textproto.CanonicalMIMEHeaderKey(name)
		if reservedMIMEHeader[canonical] {
			return
		}

		svc := &Service{config: Config{
			FromAddress:     "noreply@acme.com",
			HeaderOverrides: map[string]string{name: value},
		}}
		msg, err := svc.buildMIMEMessage(
			"user@example.com", "Reset your password", "text body", "<p>html body</p>")

		if nameErr != nil || valueErr != nil {
			if err == nil {
				t.Fatalf("buildMIMEMessage built a message from a rejected override "+
					"name=%q value=%q (nameErr=%v valueErr=%v)", name, value, nameErr, valueErr)
			}
			return
		}
		if err != nil {
			t.Fatalf("buildMIMEMessage(name=%q, value=%q) = %v, want a message", name, value, err)
		}

		want := make(map[string]bool, len(baseHeaderFields)+1)
		for _, k := range baseHeaderFields {
			want[k] = true
		}
		want[canonical] = true

		headerBlock, _, found := strings.Cut(string(msg), "\r\n\r\n")
		if !found {
			t.Fatalf("message has no header/body separator (name=%q value=%q):\n%q", name, value, msg)
		}

		// Line-level check: catches an override that added or removed a CRLF
		// field, including for names net/textproto declines to tokenise.
		lines := strings.Split(headerBlock, "\r\n")
		if len(lines) != len(want) {
			t.Fatalf("header block has %d fields, want %d (name=%q value=%q):\n%s",
				len(lines), len(want), name, value, headerBlock)
		}
		for _, line := range lines {
			key, _, ok := strings.Cut(line, ":")
			if !ok {
				t.Fatalf("header line %q has no colon (name=%q value=%q)", line, name, value)
			}
			if !want[textproto.CanonicalMIMEHeaderKey(key)] {
				t.Fatalf("unexpected header %q (name=%q value=%q):\n%s", key, name, value, headerBlock)
			}
		}

		// Parser-level check: net/textproto also treats a bare LF as a line
		// break, so this catches structure a CRLF split would miss.
		hdr, perr := readHeaderBlock(headerBlock)
		if perr != nil {
			// ValidateHeaderName accepts the full RFC 5322 ftext set, which is
			// wider than the token set net/textproto parses. Tolerate that, but
			// only after proving the field name alone is the cause: if a lone
			// "<name>: x" block parses, the failure came from the message we
			// built and is a genuine defect.
			if _, probeErr := readHeaderBlock(canonical + ": x"); probeErr == nil {
				t.Fatalf("header block failed to parse (name=%q value=%q): %v\n%s",
					name, value, perr, headerBlock)
			}
			return
		}
		if len(hdr) != len(want) {
			t.Fatalf("parsed %d distinct headers, want %d (name=%q value=%q):\n%s",
				len(hdr), len(want), name, value, headerBlock)
		}
		for key, values := range hdr {
			if !want[key] {
				t.Fatalf("parsed unexpected header %q (name=%q value=%q):\n%s",
					key, name, value, headerBlock)
			}
			if len(values) != 1 {
				t.Fatalf("header %q has %d values, want 1 (name=%q value=%q):\n%s",
					key, len(values), name, value, headerBlock)
			}
		}
	})
}

// readHeaderBlock parses a CRLF-delimited header block (without its trailing
// blank line) the way an MTA would.
func readHeaderBlock(block string) (textproto.MIMEHeader, error) {
	return textproto.NewReader(bufio.NewReader(strings.NewReader(block + "\r\n\r\n"))).ReadMIMEHeader()
}
