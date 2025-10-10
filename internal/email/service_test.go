package email_test

import (
	"testing"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/email"
)

func TestNewService(t *testing.T) {
	config := email.Config{
		SMTPHost:     "smtp.gmail.com",
		SMTPPort:     587,
		SMTPUsername: "test@example.com",
		SMTPPassword: "testpass",
		FromAddress:  "noreply@example.com",
		BaseURL:      "https://password.example.com",
	}

	service := email.NewService(&config)
	if service == nil {
		t.Fatal("NewService() returned nil")
	}
}

func TestSendResetEmailValidation(t *testing.T) {
	config := email.Config{
		SMTPHost:    "localhost",
		SMTPPort:    1025,
		FromAddress: "noreply@example.com",
		BaseURL:     "https://example.com",
	}
	service := email.NewService(&config)

	tests := []struct {
		name      string
		emailAddr string
		token     string
		shouldErr bool
	}{
		{"valid email", "user@example.com", "token123", true}, // Will error (no SMTP server)
		{"empty email", "", "token123", true},
		{"invalid email", "not-an-email", "token123", true},
		{"email with spaces", "user @example.com", "token123", true},
		{"valid complex email", "user+tag@sub.example.com", "token123", true}, // Will error (no SMTP server)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.SendResetEmail(tt.emailAddr, tt.token)
			// All should error because no SMTP server is running
			if err == nil {
				t.Error("SendResetEmail should error without SMTP server")
			}
		})
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
		{"invalid no TLD", "user@localhost", false},
		{"valid with hyphen", "user@my-domain.com", true},
		{"valid with numbers", "user123@example456.com", true},
		{"valid with dots", "first.last@example.com", true},
		{"valid with underscore", "user_name@example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := email.ValidateEmailAddress(tt.email)
			if valid != tt.valid {
				t.Errorf("ValidateEmailAddress(%q) = %v, want %v", tt.email, valid, tt.valid)
			}
		})
	}
}

func TestCaseSensitivityHandling(t *testing.T) {
	// Email addresses should be treated case-insensitively for validation.
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
			valid := email.ValidateEmailAddress(tt.email)
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
		{"all numeric domain", "user@123.456", false}, // No TLD
		{"domain ends with dot", "user@example.com.", false},
		{"IP address as domain", "user@192.168.1.1", false},       // Not in our regex
		{"domain too short (single char TLD)", "user@a.b", false}, // Requires 2+ char TLD
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := email.ValidateEmailAddress(tt.email)
			if valid != tt.valid {
				t.Errorf("ValidateEmailAddress(%q) = %v, want %v", tt.email, valid, tt.valid)
			}
		})
	}
}
