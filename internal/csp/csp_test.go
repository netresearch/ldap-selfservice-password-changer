package csp_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/csp"
)

const (
	wantDisabled = "default-src 'self'; " +
		"script-src 'self'; " +
		"style-src 'self' 'unsafe-inline'; " +
		"img-src 'self' data:; " +
		"font-src 'self'; " +
		"connect-src 'self'; " +
		"frame-ancestors 'none'; " +
		"base-uri 'self'; " +
		"form-action 'self'"

	wantEnabled = "default-src 'self'; " +
		"script-src 'self' https://challenges.cloudflare.com; " +
		"style-src 'self' 'unsafe-inline'; " +
		"img-src 'self' data:; " +
		"font-src 'self'; " +
		"connect-src 'self' https://challenges.cloudflare.com; " +
		"frame-src https://challenges.cloudflare.com; " +
		"frame-ancestors 'none'; " +
		"base-uri 'self'; " +
		"form-action 'self'"
)

func TestBuildDisabled(t *testing.T) {
	got := csp.Build(false)

	assert.Equal(t, wantDisabled, got)
	assert.NotContains(t, got, csp.TurnstileOrigin)
}

func TestBuildEnabled(t *testing.T) {
	assert.Equal(t, wantEnabled, csp.Build(true))
}

// TestBuildEnabledKeepsEveryBaseDirective is the guard against the variant
// drifting away from the base: every directive of the disabled policy must
// still be present with Turnstile on, either unchanged or extended by the
// Cloudflare origin.
func TestBuildEnabledKeepsEveryBaseDirective(t *testing.T) {
	enabled := strings.Split(csp.Build(true), "; ")

	for _, directive := range strings.Split(csp.Build(false), "; ") {
		name, _, found := strings.Cut(directive, " ")
		assert.True(t, found, "directive %q has no value", directive)

		var match string
		for _, candidate := range enabled {
			if candidate == directive || candidate == directive+" "+csp.TurnstileOrigin {
				match = candidate
				break
			}
		}

		assert.NotEmpty(t, match, "directive %q (%s) is missing or rewritten in the Turnstile policy", directive, name)
	}
}
