package email

import (
	"fmt"
	"net/smtp"
	"regexp"
	"strings"
)

// emailRegex is compiled once for performance
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// Config holds the configuration for the email service.
type Config struct {
	SMTPHost     string // SMTP server hostname (e.g., smtp.gmail.com)
	SMTPPort     int    // SMTP server port (e.g., 587 for STARTTLS)
	SMTPUsername string // SMTP authentication username
	SMTPPassword string // SMTP authentication password
	FromAddress  string // Email sender address
	BaseURL      string // Base URL for reset links (e.g., https://password.example.com)
}

// Service handles sending password reset emails.
type Service struct {
	config Config
}

// NewService creates a new email service with the given configuration.
func NewService(config Config) *Service {
	return &Service{
		config: config,
	}
}

// SendResetEmail sends a password reset email with a token link.
func (s *Service) SendResetEmail(to, token string) error {
	// Validate email address
	if !validateEmailAddress(to) {
		return fmt.Errorf("invalid email address: %s", to)
	}

	// Build email content
	subject := "Password Reset Request"
	body := s.buildResetEmailBody(token)

	// Send email
	return s.sendEmail(to, subject, body)
}

// sendEmail sends an email via SMTP.
func (s *Service) sendEmail(to, subject, body string) error {
	// Build email message
	msg := s.buildEmailMessage(to, subject, body)

	// Connect to SMTP server
	addr := fmt.Sprintf("%s:%d", s.config.SMTPHost, s.config.SMTPPort)

	// Authenticate if credentials provided
	var auth smtp.Auth
	if s.config.SMTPUsername != "" {
		auth = smtp.PlainAuth("", s.config.SMTPUsername, s.config.SMTPPassword, s.config.SMTPHost)
	}

	// Send email
	err := smtp.SendMail(addr, auth, s.config.FromAddress, []string{to}, msg)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// buildEmailMessage constructs the RFC 5322 email message.
func (s *Service) buildEmailMessage(to, subject, body string) []byte {
	msg := fmt.Sprintf("From: %s\r\n", s.config.FromAddress)
	msg += fmt.Sprintf("To: %s\r\n", to)
	msg += fmt.Sprintf("Subject: %s\r\n", subject)
	msg += "MIME-Version: 1.0\r\n"
	msg += "Content-Type: text/plain; charset=UTF-8\r\n"
	msg += "\r\n"
	msg += body

	return []byte(msg)
}

// buildResetEmailBody generates the email body for password reset.
func (s *Service) buildResetEmailBody(token string) string {
	resetLink := s.buildResetLink(token)

	body := `Hi,

We received a request to reset the password for your account. If you made this request, click the link below to continue:

` + resetLink + `

This link will expire in 15 minutes.

If you didn't request a password reset, you can safely ignore this email. Your password will not be changed.

--
LDAP Selfservice Password Changer`

	return body
}

// buildResetLink constructs the password reset URL with token.
func (s *Service) buildResetLink(token string) string {
	baseURL := strings.TrimSuffix(s.config.BaseURL, "/")
	return fmt.Sprintf("%s/reset-password?token=%s", baseURL, token)
}

// validateEmailAddress performs basic email validation.
func validateEmailAddress(email string) bool {
	if email == "" {
		return false
	}

	// Use package-level compiled regex for performance
	// Matches: user@example.com, user+tag@sub.example.com
	return emailRegex.MatchString(email)
}
