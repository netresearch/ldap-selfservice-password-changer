package email

import (
	"errors"
	"fmt"
	"mime"
	"net/mail"
	"net/textproto"
	"regexp"
	"sort"
	"strings"
)

// ValidateConfiguredAddress reports whether an operator-supplied address (the
// sender or Reply-To) is a usable RFC 5322 addr-spec.
//
// It is deliberately more permissive than ValidateEmailAddress. That regex
// requires a dotted TLD, which rejects senders SMTP delivers perfectly well:
// noreply@localhost, gopherpass@intranet, and IP-literal domains are all
// routine for a containerised app relaying through a local MTA. Applying the
// strict regex here would refuse to start deployments that work today.
//
// Recipient addresses keep the stricter regex: they derive from directory data
// rather than operator config, so narrowing what is accepted there limits
// attack surface rather than breaking a working deployment.
func ValidateConfiguredAddress(addr string) error {
	if _, err := mail.ParseAddress(addr); err != nil {
		return fmt.Errorf("not a valid email address: %w", err)
	}
	return nil
}

// headerNameRegex matches an RFC 5322 field name: 1*ftext, where ftext is any
// printable US-ASCII char (33-126) except ':' (58). No spaces, no controls.
// The bounds are spelled as hex escapes so the two sub-ranges read as the
// deliberate "0x21..0x39, 0x3b..0x7e" split around ':' (0x3a) rather than as an
// accidental mixed-class range. Simple character class + '+', so no
// catastrophic backtracking (Sonar S5852).
var headerNameRegex = regexp.MustCompile(`^[\x21-\x39\x3b-\x7e]+$`)

// ValidateHeaderName reports whether name is a syntactically valid RFC 5322
// header field name.
func ValidateHeaderName(name string) error {
	if name == "" {
		return errors.New("empty header name")
	}
	if !headerNameRegex.MatchString(name) {
		return fmt.Errorf("invalid header name %q: must be printable ASCII without spaces or ':'", name)
	}
	return nil
}

// ValidateHeaderValue rejects values that would break message structure or
// smuggle bytes into the SMTP DATA stream. A raw CR or LF enables header/body
// injection, so it is never permitted. Other C0 controls and DEL are rejected
// too: MTA handling of them is undefined (truncation at NUL, or a 5xx for the
// whole message). HTAB is allowed — it is legal folding whitespace.
func ValidateHeaderValue(value string) error {
	for i := range len(value) {
		c := value[i]
		if c == '\t' {
			continue
		}
		if c < 0x20 || c == 0x7f {
			return fmt.Errorf(
				"header value must not contain control characters (found 0x%02x at offset %d)", c, i)
		}
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

// formatFrom builds the From header value. With both a display name and an
// address it uses net/mail so the name is quoted/RFC 2047-encoded correctly;
// with only an address it emits the bare address (unchanged from prior
// behavior).
//
// An empty address drops the display name and yields an empty value, so the
// caller emits a bare empty From exactly as the pre-template code did. There is
// no valid header to build from a name alone: mail.Address{Name: "ACME IT"}.String()
// renders "ACME IT" <@>, which is not an RFC 5322 addr-spec — net/mail itself
// refuses to parse it back. Startup warns about this configuration; see
// warnEmptySender in main.go.
func formatFrom(name, address string) string {
	if name == "" || address == "" {
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
