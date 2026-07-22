package email

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"mime/quotedprintable"
	"net/textproto"
)

// reservedMIMEHeader lists structural headers the message builder owns; an
// override of these would duplicate or corrupt the MIME structure.
var reservedMIMEHeader = map[string]bool{
	"Mime-Version":              true,
	"Content-Type":              true,
	"Content-Transfer-Encoding": true,
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
		{key: "From", value: formatFrom(s.config.FromName, s.config.FromAddress)},
		{key: "To", value: to},
		{key: "Subject", value: encodeSubject(subject)},
	}
	if s.config.ReplyTo != "" {
		fields = append(fields, headerField{key: "Reply-To", value: s.config.ReplyTo})
	}
	// Defence in depth: never let an override emit a duplicate structural
	// header. The wired path already rejects these at config-parse time, but
	// the email package must not depend on the options layer for correctness.
	overrides := make(map[string]string, len(s.config.HeaderOverrides))
	for name, value := range s.config.HeaderOverrides {
		if reservedMIMEHeader[textproto.CanonicalMIMEHeaderKey(name)] {
			continue
		}
		overrides[name] = value
	}

	fields = applyHeaderOverrides(fields, overrides)
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
