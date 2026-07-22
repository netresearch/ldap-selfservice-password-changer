package email

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"strings"
	"testing"
	"time"
)

// testClockInstant is the instant pinned into every service built by
// newClockedService.
//
// The zone is deliberately obscure: a Date rendered with a zone *name* rather
// than a numeric offset still round-trips through mail.ParseDate whenever the
// name happens to match the machine's own zone, which would make the assertions
// on the parsed instant pass or fail depending on where the suite runs. "NPT"
// resolves nowhere in Go's tables, so a name-based rendering is caught.
var testClockInstant = time.Date(2026, time.March, 14, 9, 26, 53, 0, time.FixedZone("NPT", 5*3600+45*60))

// wantDateHeader is the RFC 5322 rendering of testClockInstant, written out
// literally rather than re-derived with time.Format so the test pins the wire
// format instead of mirroring whatever the builder does.
const wantDateHeader = "Sat, 14 Mar 2026 09:26:53 +0545"

// newClockedService builds a Service with a pinned clock, standing in for
// NewService in tests that exercise the message builder without templates.
// config is taken by pointer for the same reason newTestService does: Config is
// large enough that copying it per call is wasteful.
func newClockedService(config *Config) *Service {
	return &Service{config: *config, now: func() time.Time { return testClockInstant }}
}

// parseMessage splits raw message bytes into headers + a multipart reader.
func parseMessage(t *testing.T, raw []byte) (textproto.MIMEHeader, *multipart.Reader) {
	t.Helper()
	r := bufio.NewReader(bytes.NewReader(raw))
	tp := textproto.NewReader(r)
	hdr, err := tp.ReadMIMEHeader()
	if err != nil && !errors.Is(err, io.EOF) {
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

// TestBuildMIMEMessage_RejectsInjectedOverride is a regression test for a
// confirmed header-injection defect: buildMIMEMessage filtered reserved header
// names but passed override *values* through verbatim, so a value containing
// CRLF smuggled arbitrary extra headers into the message. The options layer
// rejects such values at startup, but the email package must not depend on it.
func TestBuildMIMEMessage_RejectsInjectedOverride(t *testing.T) {
	cases := map[string]map[string]string{
		"crlf in value":    {"X-Evil": "ok\r\nBcc: attacker@evil.com"},
		"lone lf in value": {"X-Evil": "ok\nBcc: attacker@evil.com"},
		"nul in value":     {"X-Evil": "a\x00b"},
		"space in name":    {"X Bad": "value"},
		"colon in name":    {"X:Bad": "value"},
	}

	for name, overrides := range cases {
		t.Run(name, func(t *testing.T) {
			s := newClockedService(&Config{FromAddress: "noreply@acme.com", HeaderOverrides: overrides})
			raw, err := s.buildMIMEMessage("user@x.com", "Sub", "t", "<p>h</p>")
			if err == nil {
				t.Fatalf("expected an error; got a message:\n%s", raw)
			}
			if raw != nil {
				t.Errorf("expected nil message on error, got %d bytes", len(raw))
			}
		})
	}
}

// TestBuildMIMEMessage_DropsReservedOverride confirms the builder keeps
// ownership of the structural MIME headers: an override naming one is dropped
// rather than emitted a second time, which would corrupt the multipart parse.
func TestBuildMIMEMessage_DropsReservedOverride(t *testing.T) {
	s := newClockedService(&Config{
		FromAddress:     "noreply@acme.com",
		HeaderOverrides: map[string]string{"Content-Type": "text/plain", "Mime-Version": "9.9"},
	})
	raw, err := s.buildMIMEMessage("user@x.com", "Sub", "t", "<p>h</p>")
	if err != nil {
		t.Fatalf("buildMIMEMessage: %v", err)
	}

	hdr, _ := parseMessage(t, raw) // fails the test if Content-Type is not multipart/alternative
	if v := hdr.Get("Mime-Version"); v != "1.0" {
		t.Errorf("MIME-Version = %q, want 1.0 (override must not win)", v)
	}
	if n := strings.Count(string(raw), "Content-Type: multipart/alternative"); n != 1 {
		t.Errorf("found %d top-level multipart Content-Type headers, want exactly 1", n)
	}
}

// TestBuildMIMEMessage_LineEndings guards the RFC 5322 wire format. The
// pre-refactor TestBuildEmailMessage asserted MIME-Version and CRLF endings;
// textproto.ReadMIMEHeader tolerates bare LF, so parsing alone cannot catch a
// regression here.
func TestBuildMIMEMessage_LineEndings(t *testing.T) {
	s := newClockedService(&Config{FromAddress: "noreply@acme.com"})
	raw, err := s.buildMIMEMessage("user@x.com", "Sub", "t", "<p>h</p>")
	if err != nil {
		t.Fatalf("buildMIMEMessage: %v", err)
	}

	hdr, _ := parseMessage(t, raw)
	if v := hdr.Get("Mime-Version"); v != "1.0" {
		t.Errorf("MIME-Version = %q, want 1.0", v)
	}

	// Every LF in the header block must be preceded by CR.
	head, _, _ := strings.Cut(string(raw), "\r\n\r\n")
	for i := range len(head) {
		if head[i] == '\n' && (i == 0 || head[i-1] != '\r') {
			t.Fatalf("bare LF at offset %d in header block:\n%q", i, head)
		}
	}
}

func TestBuildMIMEMessage_Structure(t *testing.T) {
	s := newClockedService(&Config{FromAddress: "noreply@acme.com"})
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
	// NextRawPart, not NextPart: NextPart hides the "Content-Transfer-Encoding:
	// quoted-printable" header and transparently decodes the body, which would
	// make both the CTE and the encoding assertions below vacuous.
	for i := 0; ; i++ {
		p, err := mr.NextRawPart()
		if errors.Is(err, io.EOF) {
			if i != 2 {
				t.Fatalf("got %d parts, want 2", i)
			}
			break
		}
		if err != nil {
			t.Fatalf("next part: %v", err)
		}
		mt, _, err := mime.ParseMediaType(p.Header.Get("Content-Type"))
		if err != nil {
			t.Fatalf("part %d: parse content-type %q: %v", i, p.Header.Get("Content-Type"), err)
		}
		if mt != wantTypes[i] {
			t.Errorf("part %d type = %q, want %q", i, mt, wantTypes[i])
		}
		if enc := p.Header.Get("Content-Transfer-Encoding"); enc != "quoted-printable" {
			t.Errorf("part %d CTE = %q, want quoted-printable", i, enc)
		}
		decoded, err := io.ReadAll(quotedprintable.NewReader(p))
		if err != nil {
			t.Fatalf("part %d: decode quoted-printable body: %v", i, err)
		}
		if !strings.Contains(string(decoded), wantBodies[i]) {
			t.Errorf("part %d body missing %q; got %q", i, wantBodies[i], decoded)
		}
	}
}

func TestBuildMIMEMessage_FromNameAndReplyTo(t *testing.T) {
	s := newClockedService(&Config{FromAddress: "noreply@acme.com", FromName: "ACME IT", ReplyTo: "help@acme.com"})
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
	s := newClockedService(&Config{
		FromAddress:     "noreply@acme.com",
		FromName:        "ACME IT",
		HeaderOverrides: map[string]string{"From": "Custom <c@acme.com>", "X-HelpDesk-Topic": "reset"},
	})
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

func TestBuildMIMEMessage_OverrideReplacesReplyTo(t *testing.T) {
	s := newClockedService(&Config{
		FromAddress:     "noreply@acme.com",
		ReplyTo:         "help@acme.com",
		HeaderOverrides: map[string]string{"Reply-To": "other@acme.com"},
	})
	raw, err := s.buildMIMEMessage("user@x.com", "Sub", "t", "<p>h</p>")
	if err != nil {
		t.Fatalf("buildMIMEMessage: %v", err)
	}
	hdr, _ := parseMessage(t, raw)
	if got := hdr.Get("Reply-To"); got != "other@acme.com" {
		t.Errorf("Reply-To = %q, want override value", got)
	}
	if got := hdr["Reply-To"]; len(got) != 1 {
		t.Errorf("Reply-To appears %d times, want 1", len(got))
	}
}

// TestBuildMIMEMessage_DateHeader covers the RFC 5322 section 3.6 orig-date
// requirement: exactly one Date field, carrying the service clock's instant in
// a form net/mail can parse back, numeric zone offset included.
func TestBuildMIMEMessage_DateHeader(t *testing.T) {
	s := newClockedService(&Config{FromAddress: "noreply@acme.com"})
	raw, err := s.buildMIMEMessage("user@x.com", "Sub", "t", "<p>h</p>")
	if err != nil {
		t.Fatalf("buildMIMEMessage: %v", err)
	}

	hdr, _ := parseMessage(t, raw)
	got := hdr.Get("Date")
	if got == "" {
		t.Fatalf("no Date header; RFC 5322 3.6 requires one:\n%s", raw)
	}
	if n := len(hdr["Date"]); n != 1 {
		t.Errorf("Date appears %d times, want exactly 1", n)
	}

	if got != wantDateHeader {
		t.Errorf("Date = %q, want %q", got, wantDateHeader)
	}

	parsed, err := mail.ParseDate(got)
	if err != nil {
		t.Fatalf("Date %q is not a parseable date-time: %v", got, err)
	}
	if !parsed.Equal(testClockInstant) {
		t.Errorf("Date %q parsed to %v, want the pinned instant %v", got, parsed, testClockInstant)
	}
	// Equal compares instants, so it also holds for a value normalised to UTC.
	// The offset check is what requires a numeric zone carrying the sender's
	// local time, as RFC 5322 section 3.3 specifies.
	_, wantOffset := testClockInstant.Zone()
	if _, gotOffset := parsed.Zone(); gotOffset != wantOffset {
		t.Errorf("Date %q zone offset = %d seconds, want %d", got, gotOffset, wantOffset)
	}
}

// TestBuildMIMEMessage_OverrideReplacesDate pins the ordering decision: Date is
// written with the originator fields, before the overrides are applied, so an
// operator who sets it wins and the field is still emitted only once.
func TestBuildMIMEMessage_OverrideReplacesDate(t *testing.T) {
	const override = "Tue, 01 Jan 2030 00:00:00 +0000"
	s := newClockedService(&Config{
		FromAddress:     "noreply@acme.com",
		HeaderOverrides: map[string]string{"Date": override},
	})
	raw, err := s.buildMIMEMessage("user@x.com", "Sub", "t", "<p>h</p>")
	if err != nil {
		t.Fatalf("buildMIMEMessage: %v", err)
	}

	hdr, _ := parseMessage(t, raw)
	if got := hdr.Get("Date"); got != override {
		t.Errorf("Date = %q, want override value %q", got, override)
	}
	if n := len(hdr["Date"]); n != 1 {
		t.Errorf("Date appears %d times, want exactly 1", n)
	}
	head, _, _ := strings.Cut(string(raw), "\r\n\r\n")
	if n := strings.Count(head, "Date: "); n != 1 {
		t.Errorf("found %d Date lines in the header block, want exactly 1:\n%q", n, head)
	}
}
