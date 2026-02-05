package rpc

import (
	"testing"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/options"
)

// FuzzValidateNewPassword fuzzes the ValidateNewPassword function with random inputs.
func FuzzValidateNewPassword(f *testing.F) {
	// Seed corpus with known inputs
	seeds := []struct {
		password string
		username string
	}{
		{"", ""},
		{"password", "user"},
		{"Password123!", "admin"},
		{"短い", "日本語"},
		{"Aa1!Bb2@Cc3#", "testuser"},
		{string(make([]byte, 200)), "user"},
		{"<script>alert('xss')</script>", "attacker"},
		{"' OR '1'='1", "sqlinjection"},
		{"\x00\xff\xfe", "binary"},
		{"password\nwith\nnewlines", "user"},
		{"password\twith\ttabs", "user"},
		{"🔐Secure123!", "emoji"},
	}

	for _, s := range seeds {
		f.Add(s.password, s.username)
	}

	opts := &options.Opts{
		MinLength:                  8,
		MinNumbers:                 1,
		MinSymbols:                 1,
		MinUppercase:               1,
		MinLowercase:               1,
		PasswordCanIncludeUsername: false,
	}

	f.Fuzz(func(t *testing.T, password, username string) {
		// The function should not panic for any input
		_ = ValidateNewPassword(password, username, opts)
	})
}

// FuzzPluralize fuzzes the pluralize function.
func FuzzPluralize(f *testing.F) {
	// Seed corpus
	f.Add("number", uint(0))
	f.Add("number", uint(1))
	f.Add("number", uint(2))
	f.Add("", uint(0))
	f.Add("letter", uint(100))
	f.Add("symbol", ^uint(0)) // Max uint

	f.Fuzz(func(t *testing.T, word string, amount uint) {
		// The function should not panic
		result := pluralize(word, amount)

		// Verify basic contract
		if amount == 1 {
			if result != word {
				t.Errorf("Expected %q for singular, got %q", word, result)
			}
		} else {
			if result != word+"s" {
				t.Errorf("Expected %qs for plural, got %q", word, result)
			}
		}
	})
}
