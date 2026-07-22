package email

import (
	"bytes"
	_ "embed"
	"fmt"
	htmltemplate "html/template"
	"io"
	"os"
	texttemplate "text/template"
)

//go:embed templates/reset.txt.tmpl
var defaultTextTemplate string

//go:embed templates/reset.html.tmpl
var defaultHTMLTemplate string

// defaultSubjectTemplate is used when Config.SubjectTemplate is empty.
const defaultSubjectTemplate = "Password Reset Request"

// resetEmailData is the context passed to every reset-email template.
type resetEmailData struct {
	ResetLink     string
	Token         string
	BaseURL       string
	Recipient     string
	ExpiryMinutes uint
}

// sampleResetData dry-runs templates at construction so a broken template
// fails at startup, not when the first reset email is sent.
var sampleResetData = resetEmailData{
	ResetLink:     "https://example.com/reset-password?token=sample",
	Token:         "sample",
	BaseURL:       "https://example.com",
	Recipient:     "user@example.com",
	ExpiryMinutes: 15,
}

// renderer holds the parsed subject/text/html templates.
type renderer struct {
	subject *texttemplate.Template
	text    *texttemplate.Template
	html    *htmltemplate.Template
}

// newRenderer loads, parses and dry-runs the templates. A configured path
// overrides the embedded default; an unset path uses it. Any failure is
// returned (fail-fast). Templates use missingkey=error so an undefined field
// surfaces during the dry-run rather than silently rendering "<no value>".
func newRenderer(cfg *Config) (*renderer, error) {
	subjectSrc := cfg.SubjectTemplate
	if subjectSrc == "" {
		subjectSrc = defaultSubjectTemplate
	}
	subjectTmpl, err := texttemplate.New("subject").Option("missingkey=error").Parse(subjectSrc)
	if err != nil {
		return nil, fmt.Errorf("parse subject template: %w", err)
	}

	textSrc, err := loadTemplateSource(cfg.TemplateTextPath, defaultTextTemplate)
	if err != nil {
		return nil, fmt.Errorf("text template: %w", err)
	}
	textTmpl, err := texttemplate.New("text").Option("missingkey=error").Parse(textSrc)
	if err != nil {
		return nil, fmt.Errorf("parse text template: %w", err)
	}

	htmlSrc, err := loadTemplateSource(cfg.TemplateHTMLPath, defaultHTMLTemplate)
	if err != nil {
		return nil, fmt.Errorf("html template: %w", err)
	}
	htmlTmpl, err := htmltemplate.New("html").Option("missingkey=error").Parse(htmlSrc)
	if err != nil {
		return nil, fmt.Errorf("parse html template: %w", err)
	}

	r := &renderer{subject: subjectTmpl, text: textTmpl, html: htmlTmpl}

	if _, _, _, err := r.render(sampleResetData); err != nil {
		return nil, fmt.Errorf("template dry-run: %w", err)
	}

	return r, nil
}

// maxTemplateBytes caps a configured template file. An email template is a few
// kilobytes; the cap keeps a mistyped path from stalling or OOM-ing startup.
const maxTemplateBytes = 1 << 20 // 1 MiB

// loadTemplateSource returns the file content at path, or fallback when path
// is empty. A configured-but-unreadable path is an error. The path must be a
// regular file within the size cap: pointing at a device or stream such as
// /dev/zero would otherwise hang the process before it ever binds a listener,
// which in a memory-limited container is an undiagnosable crash-loop.
func loadTemplateSource(path, fallback string) (string, error) {
	if path == "" {
		return fallback, nil
	}

	fi, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat %q: %w", path, err)
	}
	if !fi.Mode().IsRegular() {
		return "", fmt.Errorf("template %q is not a regular file (mode %s)", path, fi.Mode())
	}
	if fi.Size() > maxTemplateBytes {
		return "", fmt.Errorf("template %q is %d bytes, exceeding the %d byte limit", path, fi.Size(), maxTemplateBytes)
	}

	f, err := os.Open(path) //#nosec G304 -- operator-controlled config path, intentional
	if err != nil {
		return "", fmt.Errorf("open %q: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	b, err := io.ReadAll(io.LimitReader(f, maxTemplateBytes))
	if err != nil {
		return "", fmt.Errorf("read %q: %w", path, err)
	}
	return string(b), nil
}

// render executes all three templates against data.
func (r *renderer) render(data resetEmailData) (subject, text, html string, err error) {
	var sb bytes.Buffer
	if err = r.subject.Execute(&sb, data); err != nil {
		return "", "", "", fmt.Errorf("render subject: %w", err)
	}
	var tb bytes.Buffer
	if err = r.text.Execute(&tb, data); err != nil {
		return "", "", "", fmt.Errorf("render text body: %w", err)
	}
	var hb bytes.Buffer
	if err = r.html.Execute(&hb, data); err != nil {
		return "", "", "", fmt.Errorf("render html body: %w", err)
	}
	return sb.String(), tb.String(), hb.String(), nil
}
