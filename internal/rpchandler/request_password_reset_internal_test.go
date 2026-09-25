package rpchandler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	ldap "github.com/netresearch/simple-ldap-go"
	"github.com/stretchr/testify/require"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/options"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/ratelimit"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/resettoken"
)

// Mock email service for testing.
type mockEmailService struct {
	lastTo    string
	lastToken string
	sendError error
}

func (m *mockEmailService) SendResetEmail(to, token string) error {
	m.lastTo = to
	m.lastToken = token
	return m.sendError
}

// Mock LDAP client for testing.
type mockLDAPClient struct {
	findUserByMailError error
	findUserBySAMError  error
	users               map[string]*ldap.User // keyed by email
	usersBySAM          map[string]*ldap.User // keyed by sAMAccountName
}

func (m *mockLDAPClient) FindUserByMail(mail string) (*ldap.User, error) {
	if m.findUserByMailError != nil {
		return nil, m.findUserByMailError
	}
	if user, ok := m.users[mail]; ok {
		return user, nil
	}
	return nil, errors.New("user not found")
}

func (m *mockLDAPClient) FindUserBySAMAccountName(sAMAccountName string) (*ldap.User, error) {
	if m.findUserBySAMError != nil {
		return nil, m.findUserBySAMError
	}
	if user, ok := m.usersBySAM[sAMAccountName]; ok {
		return user, nil
	}
	return nil, errors.New("user not found")
}

func (m *mockLDAPClient) ChangePasswordForSAMAccountName(_, _, _ string) error {
	return nil
}

func (m *mockLDAPClient) ResetPasswordForSAMAccountName(_, _ string) error {
	return nil
}

func (m *mockLDAPClient) UnlockUserForSAMAccountName(_ string) error {
	return nil
}

func TestRequestPasswordResetValidEmail(t *testing.T) {
	// Setup
	tokenStore := resettoken.NewStore()
	limiter := ratelimit.NewLimiter(3, 60*time.Minute)
	mockEmail := &mockEmailService{}
	mockLDAP := &mockLDAPClient{
		users: map[string]*ldap.User{
			"test@example.com": {
				SAMAccountName: "testuser",
			},
		},
	}

	handler := &Handler{
		ldap:         mockLDAP,
		tokenStore:   tokenStore,
		emailService: mockEmail,
		rateLimiter:  limiter,
		opts: &options.Opts{
			ResetTokenExpiryMinutes: 15,
		},
	}

	// Test valid email request
	params := []string{"test@example.com"}
	result, err := handler.requestPasswordReset(params)
	require.NoError(t, err)

	// Should return generic success message
	if len(result) != 1 {
		t.Errorf("Expected 1 result, got %d", len(result))
	}

	expectedMsg := "If an account exists, a reset email has been sent"
	if result[0] != expectedMsg {
		t.Errorf("Result = %q, want %q", result[0], expectedMsg)
	}

	// Verify email was sent
	if mockEmail.lastTo != "test@example.com" {
		t.Errorf("Email sent to %q, want test@example.com", mockEmail.lastTo)
	}

	// Verify token was generated and stored
	if mockEmail.lastToken == "" {
		t.Error("No token generated")
	}

	// Verify token exists in store
	token, err := tokenStore.Get(mockEmail.lastToken)
	require.NoError(t, err)

	if token.Email != "test@example.com" {
		t.Errorf("Token email = %q, want test@example.com", token.Email)
	}
}

func TestRequestPasswordResetInvalidEmail(t *testing.T) {
	tokenStore := resettoken.NewStore()
	limiter := ratelimit.NewLimiter(3, 60*time.Minute)
	mockEmail := &mockEmailService{}
	mockLDAP := &mockLDAPClient{
		users: map[string]*ldap.User{},
	}

	handler := &Handler{
		ldap:         mockLDAP,
		tokenStore:   tokenStore,
		emailService: mockEmail,
		rateLimiter:  limiter,
		opts: &options.Opts{
			ResetTokenExpiryMinutes: 15,
		},
	}

	tests := []struct {
		name  string
		email string
	}{
		{"empty", ""},
		{"no @", "notanemail"},
		{"no domain", "user@"},
		{"no user", "@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := []string{tt.email}
			result, err := handler.requestPasswordReset(params)
			// Should still return generic success (don't reveal invalid email)
			if err != nil {
				t.Errorf("Should not error on invalid email, got: %v", err)
			}

			if len(result) != 1 {
				t.Errorf("Expected 1 result, got %d", len(result))
			}
		})
	}
}

func TestRequestPasswordResetRateLimit(t *testing.T) {
	tokenStore := resettoken.NewStore()
	limiter := ratelimit.NewLimiter(2, 60*time.Minute) // Only 2 requests allowed
	mockEmail := &mockEmailService{}
	mockLDAP := &mockLDAPClient{
		users: map[string]*ldap.User{
			"ratelimit@example.com": {
				SAMAccountName: "ratelimituser",
			},
		},
	}

	handler := &Handler{
		ldap:         mockLDAP,
		tokenStore:   tokenStore,
		emailService: mockEmail,
		rateLimiter:  limiter,
		opts: &options.Opts{
			ResetTokenExpiryMinutes: 15,
		},
	}

	email := "ratelimit@example.com"

	// First 2 requests should succeed
	for i := range 2 {
		params := []string{email}
		_, err := handler.requestPasswordReset(params)
		require.NoError(t, err, "Request %d failed", i+1)
	}

	// 3rd request should be rate limited but still return success
	params := []string{email}
	result, err := handler.requestPasswordReset(params)
	if err != nil {
		t.Errorf("Rate limited request should not error, got: %v", err)
	}

	// Should still return generic success message
	if len(result) != 1 {
		t.Errorf("Expected 1 result, got %d", len(result))
	}

	// But email should NOT have been sent
	if mockEmail.lastTo == email {
		// This is tricky - we need to track how many emails were sent
		// For now, we'll check that token count is only 2
		if tokenStore.Count() > 2 {
			t.Errorf("Too many tokens generated: %d, want 2", tokenStore.Count())
		}
	}
}

func TestRequestPasswordResetInvalidArgumentCount(t *testing.T) {
	tokenStore := resettoken.NewStore()
	limiter := ratelimit.NewLimiter(3, 60*time.Minute)
	mockEmail := &mockEmailService{}
	mockLDAP := &mockLDAPClient{
		users: map[string]*ldap.User{},
	}

	handler := &Handler{
		ldap:         mockLDAP,
		tokenStore:   tokenStore,
		emailService: mockEmail,
		rateLimiter:  limiter,
		opts: &options.Opts{
			ResetTokenExpiryMinutes: 15,
		},
	}

	tests := []struct {
		name   string
		params []string
	}{
		{"no params", []string{}},
		{"too many params", []string{"email@example.com", "extra"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := handler.requestPasswordReset(tt.params)
			if !errors.Is(err, ErrInvalidArgumentCount) {
				t.Errorf("Expected ErrInvalidArgumentCount, got: %v", err)
			}
		})
	}
}

func TestRequestPasswordResetTokenExpiration(t *testing.T) {
	tokenStore := resettoken.NewStore()
	limiter := ratelimit.NewLimiter(3, 60*time.Minute)
	mockEmail := &mockEmailService{}
	mockLDAP := &mockLDAPClient{
		users: map[string]*ldap.User{
			"test@example.com": {
				SAMAccountName: "testuser",
			},
		},
	}

	handler := &Handler{
		ldap:         mockLDAP,
		tokenStore:   tokenStore,
		emailService: mockEmail,
		rateLimiter:  limiter,
		opts: &options.Opts{
			ResetTokenExpiryMinutes: 15,
		},
	}

	params := []string{"test@example.com"}
	_, err := handler.requestPasswordReset(params)
	require.NoError(t, err)

	// Verify token has expiration set (15 minutes from now)
	token, err := tokenStore.Get(mockEmail.lastToken)
	require.NoError(t, err)

	expiryDuration := token.ExpiresAt.Sub(token.CreatedAt)
	expectedDuration := 15 * time.Minute

	// Allow 1 second tolerance for test execution time
	if expiryDuration < expectedDuration-time.Second || expiryDuration > expectedDuration+time.Second {
		t.Errorf("Token expiry duration = %v, want ~%v", expiryDuration, expectedDuration)
	}
}

func TestRequestPasswordResetEmailFailure(t *testing.T) {
	tokenStore := resettoken.NewStore()
	limiter := ratelimit.NewLimiter(3, 60*time.Minute)
	mockEmail := &mockEmailService{
		sendError: errors.New("SMTP connection failed"),
	}
	mockLDAP := &mockLDAPClient{
		users: map[string]*ldap.User{
			"test@example.com": {
				SAMAccountName: "testuser",
			},
		},
	}

	handler := &Handler{
		ldap:         mockLDAP,
		tokenStore:   tokenStore,
		emailService: mockEmail,
		rateLimiter:  limiter,
		opts: &options.Opts{
			ResetTokenExpiryMinutes: 15,
		},
	}

	params := []string{"test@example.com"}
	result, err := handler.requestPasswordReset(params)
	// Should still return success (don't reveal SMTP failure to user)
	if err != nil {
		t.Errorf("Should not error on email failure, got: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("Expected 1 result, got %d", len(result))
	}

	// Token should still be created (for retry purposes)
	if tokenStore.Count() != 1 {
		t.Errorf("Token count = %d, want 1", tokenStore.Count())
	}
}

func TestRequestPasswordResetIPRateLimitingIntegration(t *testing.T) {
	tokenStore := resettoken.NewStore()
	emailLimiter := ratelimit.NewLimiter(10, 60*time.Minute)
	ipLimiter := ratelimit.NewIPLimiter() // 10 requests per IP per 60 minutes
	mockEmail := &mockEmailService{}
	mockLDAP := &mockLDAPClient{
		users: map[string]*ldap.User{
			"user1@example.com":  {SAMAccountName: "user1"},
			"user2@example.com":  {SAMAccountName: "user2"},
			"user3@example.com":  {SAMAccountName: "user3"},
			"user4@example.com":  {SAMAccountName: "user4"},
			"user5@example.com":  {SAMAccountName: "user5"},
			"user6@example.com":  {SAMAccountName: "user6"},
			"user7@example.com":  {SAMAccountName: "user7"},
			"user8@example.com":  {SAMAccountName: "user8"},
			"user9@example.com":  {SAMAccountName: "user9"},
			"user10@example.com": {SAMAccountName: "user10"},
			"user11@example.com": {SAMAccountName: "user11"},
		},
	}

	handler := &Handler{
		ldap:         mockLDAP,
		tokenStore:   tokenStore,
		emailService: mockEmail,
		rateLimiter:  emailLimiter,
		ipLimiter:    ipLimiter,
		opts: &options.Opts{
			ResetTokenExpiryMinutes: 15,
		},
	}

	// The per-IP limiter is a guard, so the requests go through Handle.
	app := fiber.New()
	app.Post("/api/rpc", handler.Handle)

	// Requests within the IP limit are served, one token each.
	for i := 1; i <= 10; i++ {
		served := postResetRequest(t, app, fmt.Sprintf("user%d@example.com", i))
		if served.status != http.StatusOK {
			t.Fatalf("request %d: status = %d, want %d (body: %s)", i, served.status, http.StatusOK, served.body)
		}
	}

	if tokenStore.Count() != 10 {
		t.Errorf("Token count = %d, want 10", tokenStore.Count())
	}

	// The 11th is stopped by the IP guard, and answers exactly like a served
	// request so that the caller cannot tell the difference.
	got := postResetRequest(t, app, "user11@example.com")
	if got.status != http.StatusOK {
		t.Errorf("status = %d, want %d (body: %s)", got.status, http.StatusOK, got.body)
	}
	if !strings.Contains(got.body, msgResetEmailSent) {
		t.Errorf("body = %s, want the generic success message", got.body)
	}

	if tokenStore.Count() != 10 {
		t.Errorf("After IP rate limit: token count = %d, want 10", tokenStore.Count())
	}
}

func TestRequestPasswordResetIPRateLimitCheckedBeforeEmail(t *testing.T) {
	tokenStore := resettoken.NewStore()
	emailLimiter := ratelimit.NewLimiter(10, 60*time.Minute)
	ipLimiter := ratelimit.NewIPLimiter()
	mockEmail := &mockEmailService{}
	mockLDAP := &mockLDAPClient{
		users: map[string]*ldap.User{
			"test@example.com": {SAMAccountName: "testuser"},
		},
	}

	handler := &Handler{
		ldap:         mockLDAP,
		tokenStore:   tokenStore,
		emailService: mockEmail,
		rateLimiter:  emailLimiter,
		ipLimiter:    ipLimiter,
		opts: &options.Opts{
			ResetTokenExpiryMinutes: 15,
		},
	}

	app := fiber.New()
	app.Post("/api/rpc", handler.Handle)

	// Exhaust the per-IP limit.
	for i := 1; i <= 10; i++ {
		email := fmt.Sprintf("user%d@example.com", i)
		mockLDAP.users[email] = &ldap.User{SAMAccountName: fmt.Sprintf("user%d", i)}

		if served := postResetRequest(t, app, email); served.status != http.StatusOK {
			t.Fatalf("request %d: status = %d, want %d (body: %s)", i, served.status, http.StatusOK, served.body)
		}
	}

	// An identifier that has its own untouched bucket: if the guards ran in the
	// other order, this request would consume it.
	newEmail := "completely-new-email@example.com"
	mockLDAP.users[newEmail] = &ldap.User{SAMAccountName: "newuser"}

	initialEmailLimiterCount := emailLimiter.Count()

	got := postResetRequest(t, app, newEmail)
	if got.status != http.StatusOK {
		t.Errorf("status = %d, want %d (body: %s)", got.status, http.StatusOK, got.body)
	}
	if !strings.Contains(got.body, msgResetEmailSent) {
		t.Errorf("body = %s, want the generic success message", got.body)
	}

	if emailLimiter.Count() != initialEmailLimiterCount {
		t.Errorf("the per-identifier limiter was consulted although the per-IP guard had already stopped the request")
	}
}

func TestRequestPasswordResetEmailTooLong(t *testing.T) {
	tokenStore := resettoken.NewStore()
	limiter := ratelimit.NewLimiter(3, 60*time.Minute)
	mockEmail := &mockEmailService{}
	mockLDAP := &mockLDAPClient{
		users: map[string]*ldap.User{},
	}

	handler := &Handler{
		ldap:         mockLDAP,
		tokenStore:   tokenStore,
		emailService: mockEmail,
		rateLimiter:  limiter,
		opts: &options.Opts{
			ResetTokenExpiryMinutes: 15,
		},
	}

	// The length check is a guard, so the request goes through Handle: calling
	// the method directly would pass for want of a matching user rather than
	// for the length.
	app := fiber.New()
	app.Post("/api/rpc", handler.Handle)

	// One character over RFC 5321's maximum, and one exactly at it.
	tooLong := strings.Repeat("a", maxIdentifierLength+1)
	atMaximum := strings.Repeat("b", maxIdentifierLength-len("@example.com")) + "@example.com"

	got := postResetRequest(t, app, tooLong)
	if got.status != http.StatusOK {
		t.Errorf("status = %d, want %d (body: %s)", got.status, http.StatusOK, got.body)
	}
	if !strings.Contains(got.body, msgResetEmailSent) {
		t.Errorf("body = %s, want the generic success message", got.body)
	}
	if mockEmail.lastTo != "" {
		t.Errorf("Email should not be sent for too-long email address")
	}
	if limiter.Count() != 0 {
		t.Errorf("the per-identifier limiter tracks %d identifiers after an impossible one was refused", limiter.Count())
	}

	// The boundary itself is allowed through the guard: it reaches the method,
	// which then finds no such user and answers the same way. The limiter
	// having been consulted is what distinguishes the two.
	got = postResetRequest(t, app, atMaximum)
	if got.status != http.StatusOK {
		t.Errorf("status = %d, want %d (body: %s)", got.status, http.StatusOK, got.body)
	}
	if limiter.Count() == 0 {
		t.Errorf("an identifier of exactly %d characters was refused by the length guard", maxIdentifierLength)
	}
}

func TestRequestPasswordResetTokenStoreError(t *testing.T) {
	mockTokenStore := &mockFailingTokenStore{}
	limiter := ratelimit.NewLimiter(3, 60*time.Minute)
	mockEmail := &mockEmailService{}
	mockLDAP := &mockLDAPClient{
		users: map[string]*ldap.User{
			"test@example.com": {SAMAccountName: "testuser"},
		},
	}

	handler := &Handler{
		ldap:         mockLDAP,
		tokenStore:   mockTokenStore,
		emailService: mockEmail,
		rateLimiter:  limiter,
		opts: &options.Opts{
			ResetTokenExpiryMinutes: 15,
		},
	}

	params := []string{"test@example.com"}
	result, err := handler.requestPasswordReset(params)
	// Should return generic success without error (don't reveal store failure)
	if err != nil {
		t.Errorf("Should not error on token store failure, got: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("Expected 1 result, got %d", len(result))
	}
}

// mockFailingTokenStore is a token store that always fails on Store.
type mockFailingTokenStore struct{}

func (m *mockFailingTokenStore) Store(_ *resettoken.ResetToken) error {
	return errors.New("token store failed")
}

func (m *mockFailingTokenStore) Get(_ string) (*resettoken.ResetToken, error) {
	return nil, errors.New("not found")
}

func (m *mockFailingTokenStore) MarkUsed(_ string) error {
	return nil
}

func (m *mockFailingTokenStore) Delete(_ string) error {
	return nil
}

func (m *mockFailingTokenStore) CleanupExpired() int {
	return 0
}

func (m *mockFailingTokenStore) Count() int {
	return 0
}
