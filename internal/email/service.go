// Package email provides SMTP email functionality for sending password reset tokens.
package email

import (
	"fmt"
	"net/smtp"
	"regexp"
	"strings"
)

// emailRegex is compiled once for performance.
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// Config holds the configuration for the email service.
type Config struct {
	SMTPHost     string // SMTP server hostname (e.g., smtp.gmail.com)
	SMTPPort     int    // SMTP server port (e.g., 587 for STARTTLS)
	SMTPUsername string // SMTP authentication username
	SMTPPassword string // SMTP authentication password
	FromAddress  string // Email sender address (also the SMTP envelope sender)
	FromName     string // Optional From display name
	ReplyTo      string // Optional Reply-To address
	BaseURL      string // Base URL for reset links (e.g., https://password.example.com)

	ExpiryMinutes uint // Token validity in minutes, surfaced to templates

	SubjectTemplate  string            // Inline subject template; empty => default
	TemplateHTMLPath string            // Path to custom HTML body template; empty => embedded default
	TemplateTextPath string            // Path to custom text body template; empty => embedded default
	HeaderOverrides  map[string]string // Raw header overrides (name => verbatim value)
}

// Service handles sending password reset emails.
type Service struct {
	config   Config
	renderer *renderer
}

// NewService creates an email service, loading and validating templates.
// It fails fast: a missing, unparseable, or field-invalid template returns an
// error rather than deferring the failure to the first send.
func NewService(config *Config) (*Service, error) {
	r, err := newRenderer(config)
	if err != nil {
		return nil, fmt.Errorf("initialize email templates: %w", err)
	}
	return &Service{config: *config, renderer: r}, nil
}

// SendResetEmail renders and sends a password reset email with a token link.
func (s *Service) SendResetEmail(to, token string) error {
	if !ValidateEmailAddress(to) {
		return fmt.Errorf("invalid email address: %s", to)
	}

	data := resetEmailData{
		ResetLink:     s.buildResetLink(token),
		Token:         token,
		BaseURL:       strings.TrimSuffix(s.config.BaseURL, "/"),
		Recipient:     to,
		ExpiryMinutes: s.config.ExpiryMinutes,
	}

	subject, textBody, htmlBody, err := s.renderer.render(data)
	if err != nil {
		return fmt.Errorf("render reset email: %w", err)
	}

	msg, err := s.buildMIMEMessage(to, subject, textBody, htmlBody)
	if err != nil {
		return fmt.Errorf("build reset email: %w", err)
	}

	return s.sendEmail(to, msg)
}

// sendEmail sends a pre-built message via SMTP.
func (s *Service) sendEmail(to string, msg []byte) error {
	addr := fmt.Sprintf("%s:%d", s.config.SMTPHost, s.config.SMTPPort)

	var auth smtp.Auth
	if s.config.SMTPUsername != "" {
		auth = smtp.PlainAuth("", s.config.SMTPUsername, s.config.SMTPPassword, s.config.SMTPHost)
	}

	if err := smtp.SendMail(addr, auth, s.config.FromAddress, []string{to}, msg); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// buildResetLink constructs the password reset URL with token.
func (s *Service) buildResetLink(token string) string {
	baseURL := strings.TrimSuffix(s.config.BaseURL, "/")
	return fmt.Sprintf("%s/reset-password?token=%s", baseURL, token)
}

// ValidateEmailAddress performs basic email validation.
func ValidateEmailAddress(email string) bool {
	if email == "" {
		return false
	}
	return emailRegex.MatchString(email)
}
