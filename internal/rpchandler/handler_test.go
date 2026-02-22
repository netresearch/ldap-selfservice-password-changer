package rpchandler_test

import (
	"testing"
	"time"

	ldap "github.com/netresearch/simple-ldap-go"
	"github.com/stretchr/testify/assert"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/email"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/options"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/ratelimit"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/resettoken"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/rpchandler"
)

// TestNewHandlerInvalidLDAP tests New with invalid LDAP configuration.
// This test is slow (~10s) as it waits for LDAP connection timeout.
func TestNewHandlerInvalidLDAP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow LDAP timeout test in short mode")
	}

	opts := &options.Opts{
		LDAP: ldap.Config{
			Server: "ldap://nonexistent-server:389",
			BaseDN: "dc=example,dc=com",
		},
		ReadonlyUser:     "cn=readonly,dc=example,dc=com",
		ReadonlyPassword: "password",
	}

	handler, err := rpchandler.New(opts)
	// Should fail because LDAP connection will fail
	assert.Error(t, err)
	assert.Nil(t, handler)
	assert.Contains(t, err.Error(), "failed to initialize LDAP connection")
}

// TestNewWithServicesInvalidLDAP tests NewWithServices with invalid LDAP.
// This test is slow (~10s) as it waits for LDAP connection timeout.
func TestNewWithServicesInvalidLDAP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow LDAP timeout test in short mode")
	}

	opts := &options.Opts{
		LDAP: ldap.Config{
			Server: "ldap://nonexistent-server:389",
			BaseDN: "dc=example,dc=com",
		},
		ReadonlyUser:     "cn=readonly,dc=example,dc=com",
		ReadonlyPassword: "password",
	}

	tokenStore := resettoken.NewStore()
	emailConfig := &email.Config{
		SMTPHost:     "smtp.example.com",
		SMTPPort:     587,
		SMTPUsername: "user",
		SMTPPassword: "pass",
		FromAddress:  "noreply@example.com",
		BaseURL:      "https://pwd.example.com",
	}
	emailService := email.NewService(emailConfig)
	rateLimiter := ratelimit.NewLimiter(3, 60*time.Minute)
	ipLimiter := ratelimit.NewIPLimiter()

	handler, err := rpchandler.NewWithServices(opts, tokenStore, emailService, rateLimiter, ipLimiter)
	assert.Error(t, err)
	assert.Nil(t, handler)
	assert.Contains(t, err.Error(), "failed to initialize LDAP connection")
}
