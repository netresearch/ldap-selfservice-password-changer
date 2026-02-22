//go:build integration

package rpchandler_test

import (
	"os"
	"testing"
	"time"

	ldap "github.com/netresearch/simple-ldap-go"
	"github.com/stretchr/testify/require"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/email"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/options"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/ratelimit"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/resettoken"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/rpchandler"
)

// getEnvOrSkip returns an environment variable or skips the test.
func getEnvOrSkip(t *testing.T, key string) string {
	value := os.Getenv(key)
	if value == "" {
		t.Skipf("Skipping test: %s not set", key)
	}
	return value
}

// TestIntegration_NewHandler tests creating a handler with real LDAP.
func TestIntegration_NewHandler(t *testing.T) {
	ldapServer := getEnvOrSkip(t, "LDAP_SERVER")
	baseDN := getEnvOrSkip(t, "LDAP_BASE_DN")
	readonlyUser := getEnvOrSkip(t, "LDAP_READONLY_USER")
	readonlyPassword := getEnvOrSkip(t, "LDAP_READONLY_PASSWORD")

	opts := &options.Opts{
		LDAP: ldap.Config{
			Server: ldapServer,
			BaseDN: baseDN,
		},
		ReadonlyUser:     readonlyUser,
		ReadonlyPassword: readonlyPassword,
	}

	handler, err := rpchandler.New(opts)
	require.NoError(t, err)
	require.NotNil(t, handler)
}

// TestIntegration_NewWithServices tests creating a handler with all services.
func TestIntegration_NewWithServices(t *testing.T) {
	ldapServer := getEnvOrSkip(t, "LDAP_SERVER")
	baseDN := getEnvOrSkip(t, "LDAP_BASE_DN")
	readonlyUser := getEnvOrSkip(t, "LDAP_READONLY_USER")
	readonlyPassword := getEnvOrSkip(t, "LDAP_READONLY_PASSWORD")
	smtpHost := getEnvOrSkip(t, "SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	if smtpPort == "" {
		smtpPort = "1025"
	}

	opts := &options.Opts{
		LDAP: ldap.Config{
			Server: ldapServer,
			BaseDN: baseDN,
		},
		ReadonlyUser:            readonlyUser,
		ReadonlyPassword:        readonlyPassword,
		ResetTokenExpiryMinutes: 15,
	}

	tokenStore := resettoken.NewStore()
	emailConfig := &email.Config{
		SMTPHost:     smtpHost,
		SMTPPort:     1025,
		SMTPUsername: "",
		SMTPPassword: "",
		FromAddress:  "test@example.com",
		BaseURL:      "http://localhost:3000",
	}
	emailService := email.NewService(emailConfig)
	rateLimiter := ratelimit.NewLimiter(10, 60*time.Minute)
	ipLimiter := ratelimit.NewIPLimiter()

	handler, err := rpchandler.NewWithServices(opts, tokenStore, emailService, rateLimiter, ipLimiter)
	require.NoError(t, err)
	require.NotNil(t, handler)
}

// TestIntegration_UserLookup tests LDAP user lookup.
func TestIntegration_UserLookup(t *testing.T) {
	ldapServer := getEnvOrSkip(t, "LDAP_SERVER")
	baseDN := getEnvOrSkip(t, "LDAP_BASE_DN")
	readonlyUser := getEnvOrSkip(t, "LDAP_READONLY_USER")
	readonlyPassword := getEnvOrSkip(t, "LDAP_READONLY_PASSWORD")

	client, err := ldap.New(
		ldap.Config{
			Server: ldapServer,
			BaseDN: baseDN,
		},
		readonlyUser,
		readonlyPassword,
	)
	require.NoError(t, err)

	// Try to find a user (this will fail if the test user doesn't exist)
	user, err := client.FindUserByMail("testuser@example.com")
	if err != nil {
		t.Logf("User lookup failed (expected if test user doesn't exist): %v", err)
	} else {
		t.Logf("Found user: %s", user.SAMAccountName)
	}
}
