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
