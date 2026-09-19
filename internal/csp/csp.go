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

// turnstileExtends names the directives the Cloudflare origin is appended to.
// frame-src is in the list for a base that carries one; where it does not,
// Build inserts the directive instead.
var turnstileExtends = []string{"script-src", "connect-src", "frame-src"}

// isDirective reports whether the directive is the named one. The name is
// matched in full so that script-src-elem is not taken for script-src.
func isDirective(directive, name string) bool {
	return directive == name || strings.HasPrefix(directive, name+" ")
}

func matchesAny(directive string, names []string) bool {
	for _, name := range names {
		if isDirective(directive, name) {
			return true
		}
	}

	return false
}

// Build returns the policy for the given configuration. With Turnstile enabled
// the widget's origin is added to the directives Cloudflare requires
// (script-src, connect-src and frame-src); every other directive, frame-ancestors
// included, is carried over unchanged.
func Build(turnstileEnabled bool) string {
	if !turnstileEnabled {
		return base
	}

	directives := strings.Split(base, directiveSeparator)
	frameSrc := "frame-src " + TurnstileOrigin

	// A base that carries its own frame-src is extended in place; otherwise the
	// directive is added. It cannot be left out: frame-src has no fallback
	// beyond default-src 'self', which would block the challenge iframe.
	hasFrameSrc := false
	for _, directive := range directives {
		if isDirective(directive, "frame-src") {
			hasFrameSrc = true

			break
		}
	}

	out := make([]string, 0, len(directives)+1)
	for _, directive := range directives {
		if !hasFrameSrc && isDirective(directive, "frame-ancestors") {
			out = append(out, frameSrc)
			hasFrameSrc = true
		}

		if matchesAny(directive, turnstileExtends) {
			directive += " " + TurnstileOrigin
		}

		out = append(out, directive)
	}

	if !hasFrameSrc {
		out = append(out, frameSrc)
	}

	return strings.Join(out, directiveSeparator)
}
