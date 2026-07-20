package rpchandler

import (
	"testing"
	"time"

	ldap "github.com/netresearch/simple-ldap-go"
	"github.com/stretchr/testify/require"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/options"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/ratelimit"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/resettoken"
)

// newResetHandler builds a handler wired with the given mock LDAP client and
// identifier mode for the reset-request tests.
func newResetHandler(
	mode options.ResetIdentifierMode,
	mockLDAP LDAPClient,
	mockEmail *mockEmailService,
	tokenStore TokenStore,
) *Handler {
	return &Handler{
		ldap:         mockLDAP,
		tokenStore:   tokenStore,
		emailService: mockEmail,
		rateLimiter:  ratelimit.NewLimiter(10, 60*time.Minute),
		opts: &options.Opts{
			ResetIdentifierMode:     mode,
			ResetTokenExpiryMinutes: 15,
		},
	}
}

func strptr(s string) *string { return &s }

// TestRequestPasswordResetUsernameModeSendsToRegisteredMail is the security-
// critical regression: when looked up by username, the reset link must go to
// the account's LDAP-registered address, never to the typed identifier.
func TestRequestPasswordResetUsernameModeSendsToRegisteredMail(t *testing.T) {
	tokenStore := resettoken.NewStore()
	mockEmail := &mockEmailService{}
	mockLDAP := &mockLDAPClient{
		usersBySAM: map[string]*ldap.User{
			"jdoe": {SAMAccountName: "jdoe", Mail: strptr("john.doe@example.com")},
		},
	}
	handler := newResetHandler(options.ResetIdentifierUsername, mockLDAP, mockEmail, tokenStore)

	result, err := handler.requestPasswordReset([]string{"jdoe"})
	require.NoError(t, err)
	require.Equal(t, []string{msgResetEmailSent}, result)

	require.Equal(t, "john.doe@example.com", mockEmail.lastTo,
		"reset link must be sent to the LDAP-registered address, not the typed username")
	require.NotEqual(t, "jdoe", mockEmail.lastTo, "must never send to the typed identifier")

	token, err := tokenStore.Get(mockEmail.lastToken)
	require.NoError(t, err)
	require.Equal(t, "jdoe", token.Username)
	require.Equal(t, "john.doe@example.com", token.Email)
}

// TestRequestPasswordResetUsernameModeNoMail: a resolved account without a
// registered mail cannot receive a link — generic success, nothing sent.
func TestRequestPasswordResetUsernameModeNoMail(t *testing.T) {
	tokenStore := resettoken.NewStore()
	mockEmail := &mockEmailService{}
	mockLDAP := &mockLDAPClient{
		usersBySAM: map[string]*ldap.User{
			"jdoe": {SAMAccountName: "jdoe"}, // Mail is nil
		},
	}
	handler := newResetHandler(options.ResetIdentifierUsername, mockLDAP, mockEmail, tokenStore)

	result, err := handler.requestPasswordReset([]string{"jdoe"})
	require.NoError(t, err)
	require.Equal(t, []string{msgResetEmailSent}, result)
	require.Empty(t, mockEmail.lastTo, "no mail address means no email is sent")
	require.Zero(t, tokenStore.Count(), "no token is stored when no link can be sent")
}

// TestRequestPasswordResetBothModeRouting: the combined field routes by the
// presence of "@" and sends to the correct address in each case.
func TestRequestPasswordResetBothModeRouting(t *testing.T) {
	t.Run("email input", func(t *testing.T) {
		tokenStore := resettoken.NewStore()
		mockEmail := &mockEmailService{}
		mockLDAP := &mockLDAPClient{
			users: map[string]*ldap.User{
				"john.doe@example.com": {SAMAccountName: "jdoe", Mail: strptr("john.doe@example.com")},
			},
		}
		handler := newResetHandler(options.ResetIdentifierBoth, mockLDAP, mockEmail, tokenStore)

		_, err := handler.requestPasswordReset([]string{"john.doe@example.com"})
		require.NoError(t, err)
		require.Equal(t, "john.doe@example.com", mockEmail.lastTo)
	})

	t.Run("username input", func(t *testing.T) {
		tokenStore := resettoken.NewStore()
		mockEmail := &mockEmailService{}
		mockLDAP := &mockLDAPClient{
			usersBySAM: map[string]*ldap.User{
				"jdoe": {SAMAccountName: "jdoe", Mail: strptr("john.doe@example.com")},
			},
		}
		handler := newResetHandler(options.ResetIdentifierBoth, mockLDAP, mockEmail, tokenStore)

		_, err := handler.requestPasswordReset([]string{"jdoe"})
		require.NoError(t, err)
		require.Equal(t, "john.doe@example.com", mockEmail.lastTo)
	})
}

// TestRequestPasswordResetEmailModeRejectsUsername: in email-only mode a bare
// username (no "@") is rejected without any LDAP lookup — generic success.
func TestRequestPasswordResetEmailModeRejectsUsername(t *testing.T) {
	tokenStore := resettoken.NewStore()
	mockEmail := &mockEmailService{}
	mockLDAP := &mockLDAPClient{
		usersBySAM: map[string]*ldap.User{
			"jdoe": {SAMAccountName: "jdoe", Mail: strptr("john.doe@example.com")},
		},
	}
	handler := newResetHandler(options.ResetIdentifierEmail, mockLDAP, mockEmail, tokenStore)

	result, err := handler.requestPasswordReset([]string{"jdoe"})
	require.NoError(t, err)
	require.Equal(t, []string{msgResetEmailSent}, result)
	require.Empty(t, mockEmail.lastTo, "username must not resolve in email-only mode")
	require.Zero(t, tokenStore.Count())
}

// TestRequestPasswordResetDuplicatedMail: a non-unique email stays a generic
// success with nothing sent (those users use the username path).
func TestRequestPasswordResetDuplicatedMail(t *testing.T) {
	tokenStore := resettoken.NewStore()
	mockEmail := &mockEmailService{}
	mockLDAP := &mockLDAPClient{
		findUserByMailError: ldap.ErrMailDuplicated,
	}
	handler := newResetHandler(options.ResetIdentifierBoth, mockLDAP, mockEmail, tokenStore)

	result, err := handler.requestPasswordReset([]string{"shared@example.com"})
	require.NoError(t, err)
	require.Equal(t, []string{msgResetEmailSent}, result)
	require.Empty(t, mockEmail.lastTo)
	require.Zero(t, tokenStore.Count())
}

// TestRequestPasswordResetAccountRateLimitAcrossIdentifiers: in "both" mode
// the email and username spellings of one account must share a rate-limit
// budget — the typed-string buckets alone would double the reset emails
// deliverable to a single mailbox.
func TestRequestPasswordResetAccountRateLimitAcrossIdentifiers(t *testing.T) {
	tokenStore := resettoken.NewStore()
	mockEmail := &mockEmailService{}
	user := &ldap.User{SAMAccountName: "jdoe", Mail: strptr("john.doe@example.com")}
	mockLDAP := &mockLDAPClient{
		users:      map[string]*ldap.User{"john.doe@example.com": user},
		usersBySAM: map[string]*ldap.User{"jdoe": user},
	}
	handler := &Handler{
		ldap:         mockLDAP,
		tokenStore:   tokenStore,
		emailService: mockEmail,
		rateLimiter:  ratelimit.NewLimiter(3, 60*time.Minute),
		opts: &options.Opts{
			ResetIdentifierMode:     options.ResetIdentifierBoth,
			ResetTokenExpiryMinutes: 15,
		},
	}

	// Exhaust the account's budget via the email spelling.
	for i := range 3 {
		_, err := handler.requestPasswordReset([]string{"john.doe@example.com"})
		require.NoError(t, err, "request %d", i+1)
	}
	require.Equal(t, 3, tokenStore.Count())

	// The username spelling hits a fresh typed-string bucket but must be
	// stopped by the shared per-account bucket: no fourth email.
	result, err := handler.requestPasswordReset([]string{"jdoe"})
	require.NoError(t, err)
	require.Equal(t, []string{msgResetEmailSent}, result, "response stays enumeration-safe")
	require.Equal(t, 3, tokenStore.Count(), "no additional token past the account budget")
}

// TestRequestPasswordResetEmptyModeDefaultsToEmail: an unset mode behaves like
// email-only (backward compatible).
func TestRequestPasswordResetEmptyModeDefaultsToEmail(t *testing.T) {
	tokenStore := resettoken.NewStore()
	mockEmail := &mockEmailService{}
	mockLDAP := &mockLDAPClient{
		users: map[string]*ldap.User{
			"john.doe@example.com": {SAMAccountName: "jdoe", Mail: strptr("john.doe@example.com")},
		},
		usersBySAM: map[string]*ldap.User{
			"jdoe": {SAMAccountName: "jdoe", Mail: strptr("john.doe@example.com")},
		},
	}
	// Zero-value mode ("") must default to email.
	handler := newResetHandler("", mockLDAP, mockEmail, tokenStore)

	// Email resolves.
	_, err := handler.requestPasswordReset([]string{"john.doe@example.com"})
	require.NoError(t, err)
	require.Equal(t, "john.doe@example.com", mockEmail.lastTo)

	// Username does not.
	mockEmail.lastTo = ""
	_, err = handler.requestPasswordReset([]string{"jdoe"})
	require.NoError(t, err)
	require.Empty(t, mockEmail.lastTo)
}
