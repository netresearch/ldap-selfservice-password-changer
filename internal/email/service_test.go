package email

import (
	"strings"
	"testing"
)

func TestNewService(t *testing.T) {
	config := Config{
		SMTPHost:     "smtp.gmail.com",
		SMTPPort:     587,
		SMTPUsername: "test@example.com",
		SMTPPassword: "testpass",
		FromAddress:  "noreply@example.com",
		BaseURL:      "https://password.example.com",
	}

	service := NewService(config)
	if service == nil {
		t.Fatal("NewService() returned nil")
	}
	if service.config.SMTPHost != config.SMTPHost {
		t.Errorf("SMTPHost = %s, want %s", service.config.SMTPHost, config.SMTPHost)
	}
}

func TestSendResetEmail(t *testing.T) {
	// This test requires a mock SMTP server or skip in CI
	// For now, test template rendering logic
	config := Config{
		SMTPHost:     "localhost",
		SMTPPort:     1025, // MailHog
		SMTPUsername: "",
		SMTPPassword: "",
		FromAddress:  "test@example.com",
		BaseURL:      "https://test.example.com",
	}

	service := NewService(config)
	token := "test-token-abc123"
	to := "user@example.com"

	// Test that email body is generated correctly
	body := service.buildResetEmailBody(token)
	if !strings.Contains(body, token) {
		t.Error("Email body does not contain token")
	}
	if !strings.Contains(body, config.BaseURL) {
		t.Error("Email body does not contain base URL")
	}
	if !strings.Contains(body, "reset the password") {
		t.Error("Email body does not contain expected reset text")
	}

	// Skip actual SMTP test in unit tests
	// Integration tests will validate actual sending
	_ = to
}

func TestBuildResetEmailBody(t *testing.T) {
	config := Config{
		BaseURL: "https://example.com",
	}
	service := NewService(config)
	token := "abc123token"

	body := service.buildResetEmailBody(token)

	// Verify required elements in email
	requiredElements := []string{
		"reset the password",
		token,
		config.BaseURL,
		"reset-password",
		"15 minutes",
	}

	for _, element := range requiredElements {
		if !strings.Contains(body, element) {
			t.Errorf("Email body missing required element: %s", element)
		}
	}
}

func TestValidateEmailAddress(t *testing.T) {
	tests := []struct {
		name  string
		email string
		valid bool
	}{
		{"valid email", "user@example.com", true},
		{"valid with subdomain", "user@mail.example.com", true},
		{"valid with plus", "user+tag@example.com", true},
		{"invalid no @", "userexample.com", false},
		{"invalid no domain", "user@", false},
		{"invalid no user", "@example.com", false},
		{"invalid empty", "", false},
		{"invalid spaces", "user @example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := validateEmailAddress(tt.email)
			if valid != tt.valid {
				t.Errorf("validateEmailAddress(%q) = %v, want %v", tt.email, valid, tt.valid)
			}
		})
	}
}

func TestBuildResetLink(t *testing.T) {
	config := Config{
		BaseURL: "https://example.com",
	}
	service := NewService(config)
	token := "test-token-123"

	link := service.buildResetLink(token)

	expected := "https://example.com/reset-password?token=test-token-123"
	if link != expected {
		t.Errorf("buildResetLink() = %s, want %s", link, expected)
	}
}

func TestBuildResetLinkWithTrailingSlash(t *testing.T) {
	config := Config{
		BaseURL: "https://example.com/", // trailing slash
	}
	service := NewService(config)
	token := "test-token-123"

	link := service.buildResetLink(token)

	expected := "https://example.com/reset-password?token=test-token-123"
	if link != expected {
		t.Errorf("buildResetLink() = %s, want %s", link, expected)
	}
}
