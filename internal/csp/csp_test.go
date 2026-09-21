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
// drifting away from the base: every base directive must still be there with
// Turnstile on, unchanged unless it is one of the three Cloudflare needs.
// The origin count at the end is what additionally rules out a Build that
// returns the base unchanged for both arguments.
func TestBuildEnabledKeepsEveryBaseDirective(t *testing.T) {
	extended := map[string]bool{"script-src": true, "connect-src": true, "frame-src": true}

	enabled := strings.Split(csp.Build(true), "; ")
	byName := make(map[string]string, len(enabled))
	for _, directive := range enabled {
		name, _, _ := strings.Cut(directive, " ")
		byName[name] = directive
	}

	for directive := range strings.SplitSeq(csp.Build(false), "; ") {
		name, _, _ := strings.Cut(directive, " ")

		want := directive
		if extended[name] {
			want = directive + " " + csp.TurnstileOrigin
		}

		assert.Equal(t, want, byName[name], "directive %s differs between the two policies", name)
	}

	// frame-src is the one directive the variant adds, and the origin must
	// appear in exactly the three directives Cloudflare needs — no more, so a
	// source can never land beside frame-ancestors 'none', and no fewer, so a
	// Build that returned the base unchanged fails here.
	assert.Equal(t, "frame-src "+csp.TurnstileOrigin, byName["frame-src"])
	assert.Equal(t, 3, strings.Count(csp.Build(true), csp.TurnstileOrigin))
	assert.NotContains(t, byName["frame-ancestors"], csp.TurnstileOrigin)
}
