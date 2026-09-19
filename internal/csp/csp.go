// Package csp builds the Content-Security-Policy header the application serves.
package csp

import "strings"

// TurnstileOrigin is the origin the Cloudflare Turnstile widget loads from and
// talks to.
const TurnstileOrigin = "https://challenges.cloudflare.com"

// base is the policy served when no optional third-party integration is
// enabled. Every variant is derived from it, so a change here reaches all of
// them.
const base = "default-src 'self'; " +
	"script-src 'self'; " +
	"style-src 'self' 'unsafe-inline'; " + // unsafe-inline needed for browser password managers (Bitwarden etc.)
	"img-src 'self' data:; " +
	"font-src 'self'; " +
	"connect-src 'self'; " +
	"frame-ancestors 'none'; " +
	"base-uri 'self'; " +
	"form-action 'self'"

const directiveSeparator = "; "

// Build returns the policy for the given configuration. With Turnstile enabled
// the widget's origin is added to the directives Cloudflare requires
// (script-src, connect-src and frame-src); everything else stays as in base.
func Build(turnstileEnabled bool) string {
	if !turnstileEnabled {
		return base
	}

	directives := strings.Split(base, directiveSeparator)
	out := make([]string, 0, len(directives)+1)

	for _, directive := range directives {
		switch {
		case strings.HasPrefix(directive, "script-src "), strings.HasPrefix(directive, "connect-src "):
			directive += " " + TurnstileOrigin
		case strings.HasPrefix(directive, "frame-ancestors "):
			// Turnstile renders its challenge in an iframe. frame-src has no
			// fallback of its own here beyond default-src 'self', so it has to
			// be added rather than extended.
			out = append(out, "frame-src "+TurnstileOrigin)
		}

		out = append(out, directive)
	}

	return strings.Join(out, directiveSeparator)
}
