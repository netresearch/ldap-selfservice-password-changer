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

	service := NewService(&config)
	if service == nil {
		t.Fatal("NewService() returned nil")
		return
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

	service := NewService(&config)
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
	service := NewService(&config)
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
		{"invalid multiple @", "user@@example.com", false},
		{"invalid multiple @ separated", "user@domain@example.com", false},
		{"invalid no TLD", "user@localhost", false},
		{"invalid single letter TLD", "user@example.c", false},
		{"invalid just @", "@", false},
		{"invalid only domain", "example.com", false},
		{"leading dot (permissive)", ".user@example.com", true},               // Regex allows this
		{"trailing dot in local (permissive)", "user.@example.com", true},     // Regex allows this
		{"leading hyphen in domain (permissive)", "user@-example.com", true},  // Regex allows this
		{"trailing hyphen in domain (permissive)", "user@example-.com", true}, // Regex allows this
		{"invalid special chars", "user!#$%@example.com", false},
		{"invalid unicode", "üser@example.com", false},
		{"valid with hyphen", "user@my-domain.com", true},
		{"valid with numbers", "user123@example456.com", true},
		{"valid with dots", "first.last@example.com", true},
		{"valid with underscore", "user_name@example.com", true},
		{"valid with multiple subdomains", "user@mail.corp.example.com", true},
		{"very long local part (63 chars)", "a" + strings.Repeat("x", 62) + "@example.com", true},
		{"very long local part (64 chars)", strings.Repeat("x", 64) + "@example.com", true},
		{"very long local part (65+ chars)", strings.Repeat("x", 65) + "@example.com", true},
		{"very long domain", "user@" + strings.Repeat("a", 250) + ".com", true},
		{"maximum valid TLD length", "user@example." + strings.Repeat("a", 10), true},
		{"empty local part", "@example.com", false},
		{"empty domain part", "user@", false},
		{"whitespace in email", "user name@example.com", false},
		{"tab in email", "user\t@example.com", false},
		{"newline in email", "user\n@example.com", false},
		{"null byte", "user\x00@example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := ValidateEmailAddress(tt.email)
			if valid != tt.valid {
				t.Errorf("ValidateEmailAddress(%q) = %v, want %v", tt.email, valid, tt.valid)
			}
		})
	}
}

func TestBuildResetLink(t *testing.T) {
	config := Config{
		BaseURL: "https://example.com",
	}
	service := NewService(&config)
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
	service := NewService(&config)
	token := "test-token-123"

	link := service.buildResetLink(token)

	expected := "https://example.com/reset-password?token=test-token-123"
	if link != expected {
		t.Errorf("buildResetLink() = %s, want %s", link, expected)
	}
}

func TestBuildEmailMessage(t *testing.T) {
	config := Config{
		FromAddress: "noreply@example.com",
	}
	service := NewService(&config)

	msg := service.buildEmailMessage("user@example.com", "Test Subject", "Test Body")
	msgStr := string(msg)

	// Verify RFC 5322 compliance
	requiredHeaders := []string{
		"From: noreply@example.com",
		"To: user@example.com",
		"Subject: Test Subject",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
	}

	for _, header := range requiredHeaders {
		if !strings.Contains(msgStr, header) {
			t.Errorf("Email message missing header: %s", header)
		}
	}

	// Verify body is included
	if !strings.Contains(msgStr, "Test Body") {
		t.Error("Email message missing body content")
	}

	// Verify CRLF line endings
	if !strings.Contains(msgStr, "\r\n") {
		t.Error("Email message should use CRLF line endings")
	}
}

func TestSendResetEmailValidation(t *testing.T) {
	config := Config{
		SMTPHost:    "localhost",
		SMTPPort:    1025,
		FromAddress: "noreply@example.com",
		BaseURL:     "https://example.com",
	}
	service := NewService(&config)

	// Check if MailHog/SMTP is actually available by trying to send
	smtpAvailable := false
	testService := NewService(&config)
	if err := testService.SendResetEmail("probe@example.com", "probe"); err == nil {
		smtpAvailable = true
	}

	tests := []struct {
		name         string
		email        string
		token        string
		invalidEmail bool // true if email format is invalid
	}{
		{"valid email", "user@example.com", "token123", false},
		{"empty email", "", "token123", true},
		{"invalid email", "not-an-email", "token123", true},
		{"email with spaces", "user @example.com", "token123", true},
		{"valid complex email", "user+tag@sub.example.com", "token123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.SendResetEmail(tt.email, tt.token)

			switch {
			case tt.invalidEmail:
				// Invalid emails should always error with validation error
				if err == nil {
					t.Errorf("SendResetEmail should error for invalid email %q", tt.email)
				} else if !strings.Contains(err.Error(), "invalid email") {
					t.Errorf("Expected validation error for %q, got: %v", tt.email, err)
				}
			case smtpAvailable:
				// Valid emails should succeed when SMTP is available
				if err != nil {
					t.Errorf("SendResetEmail should succeed with SMTP server for %q, got: %v", tt.email, err)
				}
			default:
				// Valid emails should error when no SMTP server
				if err == nil {
					t.Error("SendResetEmail should error without SMTP server")
				}
			}
		})
	}
}

func TestEmailBodyContainsSecurityWarning(t *testing.T) {
	config := Config{
		BaseURL: "https://example.com",
	}
	service := NewService(&config)
	body := service.buildResetEmailBody("token123")

	// Verify security warning is present
	securityPhrases := []string{
		"If you didn't request",
		"safely ignore",
		"will not be changed",
	}

	for _, phrase := range securityPhrases {
		if !strings.Contains(body, phrase) {
			t.Errorf("Email body missing security phrase: %s", phrase)
		}
	}
}

func TestEmailBodyContainsExpirationInfo(t *testing.T) {
	config := Config{
		BaseURL: "https://example.com",
	}
	service := NewService(&config)
	body := service.buildResetEmailBody("token123")

	if !strings.Contains(body, "15 minutes") {
		t.Error("Email body should mention token expiration time")
	}
}

func TestCaseSensitivityHandling(t *testing.T) {
	// Email addresses should be treated case-insensitively for validation
	tests := []struct {
		email string
		valid bool
	}{
		{"User@Example.COM", true},
		{"USER@EXAMPLE.COM", true},
		{"user@example.com", true},
		{"User@example.COM", true},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			valid := ValidateEmailAddress(tt.email)
			if valid != tt.valid {
				t.Errorf("ValidateEmailAddress(%q) = %v, want %v", tt.email, valid, tt.valid)
			}
		})
	}
}

func TestDomainValidationEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		email string
		valid bool
	}{
		{"domain with numbers", "user@123.com", true},
		{"domain starts with number", "user@1example.com", true},
		{"all numeric domain", "user@123.456", false},                            // No TLD
		{"domain with consecutive dots (permissive)", "user@example..com", true}, // Regex allows
		{"domain ends with dot", "user@example.com.", false},
		{"domain starts with dot (permissive)", "user@.example.com", true}, // Regex allows
		{"IP address as domain", "user@192.168.1.1", false},                // Not in our regex
		{"domain too short (single char TLD)", "user@a.b", false},          // Requires 2+ char TLD
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := ValidateEmailAddress(tt.email)
			if valid != tt.valid {
				t.Errorf("ValidateEmailAddress(%q) = %v, want %v", tt.email, valid, tt.valid)
			}
		})
	}
}
