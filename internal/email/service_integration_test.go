//go:build integration

package email_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/email"
)

// getEnvOrSkip returns an environment variable or skips the test.
func getEnvOrSkip(t *testing.T, key string) string {
	value := os.Getenv(key)
	if value == "" {
		t.Skipf("Skipping test: %s not set", key)
	}
	return value
}

// MailHogMessage represents a message in MailHog's API.
type MailHogMessage struct {
	ID      string `json:"ID"`
	From    struct {
		Mailbox string `json:"Mailbox"`
		Domain  string `json:"Domain"`
	} `json:"From"`
	To []struct {
		Mailbox string `json:"Mailbox"`
		Domain  string `json:"Domain"`
	} `json:"To"`
	Content struct {
		Headers map[string][]string `json:"Headers"`
		Body    string              `json:"Body"`
	} `json:"Content"`
}

// MailHogMessages represents the response from MailHog's messages API.
type MailHogMessages struct {
	Total int              `json:"total"`
	Items []MailHogMessage `json:"items"`
}

// TestIntegration_SendResetEmail tests sending an actual email via MailHog.
func TestIntegration_SendResetEmail(t *testing.T) {
	smtpHost := getEnvOrSkip(t, "SMTP_HOST")

	// Create email service
	config := &email.Config{
		SMTPHost:     smtpHost,
		SMTPPort:     1025, // MailHog SMTP port
		SMTPUsername: "",
		SMTPPassword: "",
		FromAddress:  "noreply@example.com",
		BaseURL:      "http://localhost:3000",
	}
	service := email.NewService(config)

	// Generate unique email to avoid conflicts
	uniqueEmail := fmt.Sprintf("testuser-%d@example.com", time.Now().UnixNano())
	testToken := "test-reset-token-12345"

	// Send email
	err := service.SendResetEmail(uniqueEmail, testToken)
	require.NoError(t, err)

	// Wait briefly for MailHog to receive the email
	time.Sleep(500 * time.Millisecond)

	// Verify email was received via MailHog API
	mailhogURL := fmt.Sprintf("http://%s:8025/api/v2/messages", smtpHost)
	resp, err := http.Get(mailhogURL)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var messages MailHogMessages
	err = json.Unmarshal(body, &messages)
	require.NoError(t, err)

	// Find our email
	var foundMessage *MailHogMessage
	for i, msg := range messages.Items {
		for _, to := range msg.To {
			if fmt.Sprintf("%s@%s", to.Mailbox, to.Domain) == uniqueEmail {
				foundMessage = &messages.Items[i]
				break
			}
		}
		if foundMessage != nil {
			break
		}
	}

	require.NotNil(t, foundMessage, "Email not found in MailHog")

	// Verify email content
	assert.Contains(t, foundMessage.Content.Body, testToken, "Email should contain reset token")
	assert.Contains(t, foundMessage.Content.Body, "reset", "Email should mention password reset")

	// Verify headers
	subjects := foundMessage.Content.Headers["Subject"]
	require.NotEmpty(t, subjects)
	assert.Contains(t, subjects[0], "Password Reset", "Subject should mention password reset")
}

// TestIntegration_SendResetEmailInvalidAddress tests with invalid email.
func TestIntegration_SendResetEmailInvalidAddress(t *testing.T) {
	smtpHost := getEnvOrSkip(t, "SMTP_HOST")

	config := &email.Config{
		SMTPHost:     smtpHost,
		SMTPPort:     1025,
		SMTPUsername: "",
		SMTPPassword: "",
		FromAddress:  "noreply@example.com",
		BaseURL:      "http://localhost:3000",
	}
	service := email.NewService(config)

	// Invalid email address should fail validation
	err := service.SendResetEmail("not-an-email", "token123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid email")
}

// TestIntegration_SendResetEmailConnectionFailure tests SMTP connection failure.
func TestIntegration_SendResetEmailConnectionFailure(t *testing.T) {
	// Use a non-existent SMTP server
	config := &email.Config{
		SMTPHost:     "localhost",
		SMTPPort:     9999, // Non-existent port
		SMTPUsername: "",
		SMTPPassword: "",
		FromAddress:  "noreply@example.com",
		BaseURL:      "http://localhost:3000",
	}
	service := email.NewService(config)

	err := service.SendResetEmail("test@example.com", "token123")
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "failed to send") ||
		strings.Contains(err.Error(), "connection refused"),
		"Expected connection error, got: %v", err)
}
