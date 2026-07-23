//nolint:testpackage // consistent with templates_test.go
package templates

import (
	"strings"
	"testing"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/options"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/web/static"
)

// branding builds a configuration through the real constructor, so the tests
// render what a running server would rather than a hand-assembled struct.
func branding(t *testing.T, productName, pageTitle, logoAlt string, showAttribution bool) options.Branding {
	t.Helper()

	b, err := options.NewBranding("", productName, pageTitle, logoAlt, showAttribution)
	if err != nil {
		t.Fatalf("NewBranding: %v", err)
	}

	return b
}

// renderAllPages renders every page with the same branding so a regression in
// one template cannot hide behind another.
func renderAllPages(t *testing.T, b options.Branding) map[string]string {
	t.Helper()

	opts := &options.Opts{Branding: b, PasswordResetEnabled: true}

	pages := map[string]func(*options.Opts) ([]byte, error){
		"index":           RenderIndex,
		"forgot-password": RenderForgotPassword,
		"reset-password":  RenderResetPassword,
	}

	out := make(map[string]string, len(pages))
	for name, render := range pages {
		raw, err := render(opts)
		if err != nil {
			t.Fatalf("render %s: %v", name, err)
		}
		out[name] = string(raw)
	}

	return out
}

func TestBranding_CustomProductNameReachesWordmarkAndTitle(t *testing.T) {
	pages := renderAllPages(t, branding(t, "Acme Passwords", "Acme — password self service", "", true))

	if !strings.Contains(pages["index"], "<title>Acme — password self service</title>") {
		t.Error("index should use the configured page title")
	}
	for name, html := range pages {
		if !strings.Contains(html, "Acme Passwords") {
			t.Errorf("%s should render the configured wordmark", name)
		}
		if strings.Contains(html, "GopherPass —") {
			t.Errorf("%s still contains the stock title", name)
		}
	}

	if !strings.Contains(pages["forgot-password"], "<title>Forgot Password — Acme Passwords</title>") {
		t.Error("forgot-password should combine its own prefix with the configured name")
	}
	if !strings.Contains(pages["reset-password"], "<title>Reset Password — Acme Passwords</title>") {
		t.Error("reset-password should combine its own prefix with the configured name")
	}
}

func TestBranding_AttributionCanBeHidden(t *testing.T) {
	shown := renderAllPages(t, branding(t, "Acme", "", "", true))
	hidden := renderAllPages(t, branding(t, "Acme", "", "", false))

	for name := range shown {
		if !strings.Contains(shown[name], "Netresearch DTT GmbH") {
			t.Errorf("%s should show the attribution when enabled", name)
		}
		if strings.Contains(hidden[name], "Netresearch DTT GmbH") {
			t.Errorf("%s should hide the attribution when disabled", name)
		}
		// The whole landmark goes, not just its text: an empty contentinfo
		// region would still be announced by a screen reader.
		if strings.Contains(hidden[name], `role="contentinfo"`) {
			t.Errorf("%s should omit the footer landmark entirely when the attribution is hidden", name)
		}
	}
}

func TestBranding_DarkLogoOnlyRenderedWhenSupplied(t *testing.T) {
	base := branding(t, "Acme", "", "", true)
	withDark := base
	withDark.HasDarkLogo = true

	without := renderAllPages(t, base)
	with := renderAllPages(t, withDark)

	// The URL must be derived from the constant the overlay allowlists. If the
	// two drift apart, the browser requests a name the overlay refuses and the
	// dark-mode logo silently breaks.
	wantSrc := `src="/static/` + static.DarkLogo + `"`

	for name := range without {
		if strings.Contains(without[name], static.DarkLogo) {
			t.Errorf("%s must not reference a dark logo that was never supplied", name)
		}
		if strings.Contains(without[name], "dark:hidden") {
			t.Errorf("%s must not hide the only logo in dark mode", name)
		}

		if !strings.Contains(with[name], wantSrc) {
			t.Errorf("%s should reference %s", name, wantSrc)
		}
		// Class-based toggling, not a prefers-color-scheme media query, so the
		// variant follows the in-page theme switch.
		if !strings.Contains(with[name], "dark:hidden") || !strings.Contains(with[name], "dark:block") {
			t.Errorf("%s should toggle the logo variants with Tailwind dark classes", name)
		}
	}
}

// Only the light logo is fetched eagerly: the hidden variant would otherwise
// compete with the LCP candidate on every light-mode load, and browsers fetch
// display:none images regardless.
func TestBranding_OnlyTheVisibleLogoIsHighPriority(t *testing.T) {
	withDark := branding(t, "Acme", "", "", true)
	withDark.HasDarkLogo = true

	for name, html := range renderAllPages(t, withDark) {
		if got := strings.Count(html, `fetchpriority="high"`); got != 1 {
			t.Errorf("%s has %d high-priority images, want exactly 1", name, got)
		}
	}
}

// A logo-only deployment must still name the brand and title every page.
func TestBranding_LogoOnlyDeploymentKeepsNamesAndTitles(t *testing.T) {
	pages := renderAllPages(t, branding(t, "", "", "Acme Corporation", true))

	for name, html := range pages {
		if !strings.Contains(html, `alt="Acme Corporation"`) {
			t.Errorf("%s should render the configured logo alt text", name)
		}
		// No wordmark element at all — an empty <p> would leave a layout gap
		// and an empty node in the accessibility tree.
		if strings.Contains(html, `class="text-2xl font-semibold`) {
			t.Errorf("%s should omit the wordmark element when no product name is set", name)
		}
	}

	if !strings.Contains(pages["index"], "<title>Acme Corporation — ") {
		t.Error("index title should be derived from the logo alt text")
	}
	if !strings.Contains(pages["forgot-password"], "<title>Forgot Password — Acme Corporation</title>") {
		t.Error("forgot-password title must not degrade to a dangling separator")
	}
	if !strings.Contains(pages["reset-password"], "<title>Reset Password — Acme Corporation</title>") {
		t.Error("reset-password title must not degrade to a dangling separator")
	}
}

// With a wordmark present the logo stays decorative — including when the
// operator also filled in the alt text, which would otherwise make a screen
// reader announce the brand twice.
func TestBranding_LogoStaysDecorativeAlongsideAWordmark(t *testing.T) {
	for _, tt := range []struct {
		name string
		b    options.Branding
	}{
		{"wordmark only", branding(t, "Acme", "", "", true)},
		{"wordmark and alt text", branding(t, "Acme", "", "Acme Corporation", true)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for page, html := range renderAllPages(t, tt.b) {
				if !strings.Contains(html, `alt=""`) {
					t.Errorf("%s should keep the logo decorative while a wordmark is shown", page)
				}
				if strings.Contains(html, `alt="Acme Corporation"`) {
					t.Errorf("%s announces the brand twice: wordmark plus logo alt text", page)
				}
			}
		})
	}
}

// Branding values are operator-supplied; html/template must escape them
// rather than letting markup through.
func TestBranding_ValuesAreEscaped(t *testing.T) {
	pages := renderAllPages(t, branding(t, `Acme<script>alert(1)</script>`, "", "", true))

	for name, html := range pages {
		if strings.Contains(html, "<script>alert(1)</script>") {
			t.Errorf("%s rendered unescaped markup from the product name", name)
		}
	}
}
