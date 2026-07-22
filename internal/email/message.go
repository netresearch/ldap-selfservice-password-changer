package email

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"mime/quotedprintable"
	"net/textproto"
)

// Header field names written by the message builder, plus the MIME-Version
// value. headerMIMEVersion is the wire spelling; textproto canonicalizes it to
// "Mime-Version", which is what reservedMIMEHeader is keyed by.
const (
	headerFrom                    = "From"
	headerTo                      = "To"
	headerSubject                 = "Subject"
	headerReplyTo                 = "Reply-To"
	headerMIMEVersion             = "MIME-Version"
	headerContentType             = "Content-Type"
	headerContentTransferEncoding = "Content-Transfer-Encoding"

	mimeVersionValue = "1.0"
)

// reservedMIMEHeader lists structural headers the message builder owns; an
// override of these would duplicate or corrupt the MIME structure. Keys are
// canonicalized so lookups against textproto.CanonicalMIMEHeaderKey output
// match ("MIME-Version" canonicalizes to "Mime-Version").
var reservedMIMEHeader = map[string]bool{
	textproto.CanonicalMIMEHeaderKey(headerMIMEVersion):             true,
	textproto.CanonicalMIMEHeaderKey(headerContentType):             true,
	textproto.CanonicalMIMEHeaderKey(headerContentTransferEncoding): true,
}

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
		{key: headerFrom, value: formatFrom(s.config.FromName, s.config.FromAddress)},
		{key: headerTo, value: to},
		{key: headerSubject, value: encodeSubject(subject)},
	}
	if s.config.ReplyTo != "" {
		fields = append(fields, headerField{key: headerReplyTo, value: s.config.ReplyTo})
	}
	// Defense in depth: the email package must not depend on the options layer
	// for correctness. A reserved structural header is dropped (the builder
	// owns those, and the wired path already hard-errors on them); a malformed
	// name or a value carrying CR/LF or other control bytes is a header
	// injection vector, so it fails the send loudly rather than being smuggled
	// into the message.
	overrides := make(map[string]string, len(s.config.HeaderOverrides))
	for name, value := range s.config.HeaderOverrides {
		if reservedMIMEHeader[textproto.CanonicalMIMEHeaderKey(name)] {
			continue
		}
		if err := ValidateHeaderName(name); err != nil {
			return nil, fmt.Errorf("header override %q: %w", name, err)
		}
		if err := ValidateHeaderValue(value); err != nil {
			return nil, fmt.Errorf("header override %q: %w", name, err)
		}
		overrides[name] = value
	}

	fields = applyHeaderOverrides(fields, overrides)
	fields = append(fields,
		headerField{key: headerMIMEVersion, value: mimeVersionValue},
		headerField{key: headerContentType, value: `multipart/alternative; boundary="` + mw.Boundary() + `"`},
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
	h.Set(headerContentType, contentType)
	h.Set(headerContentTransferEncoding, "quoted-printable")
	pw, err := mw.CreatePart(h)
	if err != nil {
		return fmt.Errorf("create %s part: %w", contentType, err)
	}
	qw := quotedprintable.NewWriter(pw)
	if _, err := qw.Write([]byte(content)); err != nil {
		return fmt.Errorf("write quoted-printable body of %s part: %w", contentType, err)
	}
	if err := qw.Close(); err != nil {
		return fmt.Errorf("flush quoted-printable body of %s part: %w", contentType, err)
	}
	return nil
}
