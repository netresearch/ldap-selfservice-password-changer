//nolint:testpackage // tests internal functions
package email

import (
	"testing"
)

// FuzzValidateEmailAddress fuzzes the ValidateEmailAddress function.
func FuzzValidateEmailAddress(f *testing.F) {
	// Seed corpus with various email patterns
	seeds := []string{
		"",
		"user@example.com",
		"user",
		"@example.com",
		"user@",
		"user@domain",
		"user@domain.com",
		"user+tag@example.com",
		"user.name@sub.example.com",
		"very-long-email-address-that-exceeds-reasonable-length@very-long-domain-name-that-exceeds-reasonable-length.com",
		"user@localhost",
		"user@127.0.0.1",
		"user@[127.0.0.1]",
		"<script>@example.com",
		"' OR '1'='1@example.com",
		"user\x00@example.com",
		"user\n@example.com",
		"user\t@example.com",
		"用户@例子.公司",
		"🔐@emoji.com",
		"user@例子.com",
		"a@b.co",
		"a@b.c",
	}

	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, email string) {
		// The function should not panic for any input
		result := ValidateEmailAddress(email)

		// Verify basic contract: empty email should always be invalid
		if email == "" && result {
			t.Error("Empty email should be invalid")
		}

		// If valid, it should contain @ and .
		if result {
			hasAt := false
			hasDot := false
			for _, c := range email {
				if c == '@' {
					hasAt = true
				}
				if c == '.' {
					hasDot = true
				}
			}
			if !hasAt {
				t.Errorf("Valid email %q should contain @", email)
			}
			if !hasDot {
				t.Errorf("Valid email %q should contain .", email)
			}
		}
	})
}
