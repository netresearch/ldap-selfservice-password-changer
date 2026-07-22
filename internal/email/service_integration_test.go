//go:build integration

package email_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/email"
)

// testToken is the reset token used by the integration tests.
const testToken = "test-reset-token-12345"

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
	ID   string `json:"ID"`
	From struct {
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

// integrationEmailConfig builds the base email config pointing at the MailHog
// container. It skips the test when SMTP_HOST is not configured.
func integrationEmailConfig(t *testing.T) *email.Config {
	smtpHost := getEnvOrSkip(t, "SMTP_HOST")

	return &email.Config{
		SMTPHost:     smtpHost,
		SMTPPort:     1025, // MailHog SMTP port
		SMTPUsername: "",
		SMTPPassword: "",
		FromAddress:  "noreply@example.com",
		BaseURL:      "http://localhost:3000",
	}
}

// waitForMessage polls MailHog's API for a message addressed to `to`.
// Returns nil when no matching message was found.
func waitForMessage(t *testing.T, to string) *MailHogMessage {
	smtpHost := getEnvOrSkip(t, "SMTP_HOST")

	// Wait briefly for MailHog to receive the email
	time.Sleep(500 * time.Millisecond)

	mailhogURL := fmt.Sprintf("http://%s:8025/api/v2/messages", smtpHost)
	resp, err := http.Get(mailhogURL)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var messages MailHogMessages
	err = json.Unmarshal(body, &messages)
	require.NoError(t, err)

	for i, msg := range messages.Items {
		for _, recipient := range msg.To {
			if fmt.Sprintf("%s@%s", recipient.Mailbox, recipient.Domain) == to {
				return &messages.Items[i]
			}
		}
	}

	return nil
}

// TestIntegration_SendResetEmail tests sending an actual email via MailHog.
func TestIntegration_SendResetEmail(t *testing.T) {
	config := integrationEmailConfig(t)
	service, err := email.NewService(config)
	require.NoError(t, err)

	// Generate unique email to avoid conflicts
	uniqueEmail := fmt.Sprintf("testuser-%d@example.com", time.Now().UnixNano())

	// Send email
	err = service.SendResetEmail(uniqueEmail, testToken)
	require.NoError(t, err)

	foundMessage := waitForMessage(t, uniqueEmail)
	require.NotNil(t, foundMessage, "Email not found in MailHog")

	// Verify email content
	assert.Contains(t, foundMessage.Content.Body, testToken, "Email should contain reset token")
	assert.Contains(t, foundMessage.Content.Body, "reset", "Email should mention password reset")

	// Verify headers
	subjects := foundMessage.Content.Headers["Subject"]
	require.NotEmpty(t, subjects)
	assert.Contains(t, subjects[0], "Password Reset", "Subject should mention password reset")

	contentTypes := foundMessage.Content.Headers["Content-Type"]
	require.NotEmpty(t, contentTypes)
	assert.Contains(t, contentTypes[0], "multipart/alternative", "reset email should be multipart")

	// Parse the body and assert exactly two parts: text/plain then text/html.
	_, params, err := mime.ParseMediaType(contentTypes[0])
	require.NoError(t, err)
	require.NotEmpty(t, params["boundary"])

	mr := multipart.NewReader(strings.NewReader(foundMessage.Content.Body), params["boundary"])
	wantTypes := []string{"text/plain", "text/html"}
	var gotTypes []string
	for {
		p, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)

		mt, _, err := mime.ParseMediaType(p.Header.Get("Content-Type"))
		require.NoError(t, err)
		gotTypes = append(gotTypes, mt)

		decoded, err := io.ReadAll(quotedprintable.NewReader(p))
		require.NoError(t, err)
		assert.NotEmpty(t, decoded, "part %d body should not be empty", len(gotTypes)-1)
	}
	assert.Equal(t, wantTypes, gotTypes, "reset email should have text/plain then text/html")
}

// TestIntegration_CustomSubjectAndHeaderOverride verifies that a custom subject
// template and a header override round-trip to the received message.
func TestIntegration_CustomSubjectAndHeaderOverride(t *testing.T) {
	cfg := integrationEmailConfig(t)
	cfg.SubjectTemplate = "[ACME] Reset for {{.Recipient}}"
	cfg.HeaderOverrides = map[string]string{"X-HelpDesk-Topic": "password-reset"}

	service, err := email.NewService(cfg)
	require.NoError(t, err)

	to := "custom-headers@example.com"
	require.NoError(t, service.SendResetEmail(to, testToken))

	foundMessage := waitForMessage(t, to)
	require.NotNil(t, foundMessage)

	subjects := foundMessage.Content.Headers["Subject"]
	require.NotEmpty(t, subjects)
	assert.Equal(t, "[ACME] Reset for "+to, subjects[0], "custom subject should round-trip")

	// Canonical key: applyHeaderOverrides emits header names via
	// textproto.CanonicalMIMEHeaderKey, so the wire header is X-Helpdesk-Topic.
	topics := foundMessage.Content.Headers["X-Helpdesk-Topic"]
	require.NotEmpty(t, topics, "override header should be present on the received message")
	assert.Equal(t, "password-reset", topics[0])
}

// TestIntegration_SendResetEmailInvalidAddress tests with invalid email.
func TestIntegration_SendResetEmailInvalidAddress(t *testing.T) {
	config := integrationEmailConfig(t)
	service, err := email.NewService(config)
	require.NoError(t, err)

	// Invalid email address should fail validation
	err = service.SendResetEmail("not-an-email", "token123")
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
	service, err := email.NewService(config)
	require.NoError(t, err)

	err = service.SendResetEmail("test@example.com", "token123")
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "failed to send") ||
		strings.Contains(err.Error(), "connection refused"),
		"Expected connection error, got: %v", err)
}
