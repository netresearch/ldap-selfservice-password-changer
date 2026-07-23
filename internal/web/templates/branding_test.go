//nolint:testpackage // consistent with templates_test.go
package templates

import (
	"strings"
	"testing"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/options"
)

// renderAllPages renders every page with the same branding so a regression in
// one template cannot hide behind another.
func renderAllPages(t *testing.T, branding options.Branding) map[string]string {
	t.Helper()

	opts := &options.Opts{Branding: branding, PasswordResetEnabled: true}

	pages := map[string]func(*options.Opts) ([]byte, error){
		"index":           RenderIndex,
		"forgot-password": RenderForgotPassword,
		"reset-password":  RenderResetPassword,
	}

	out := make(map[string]string, len(pages))
	for name, render := range pages {
		b, err := render(opts)
		if err != nil {
			t.Fatalf("render %s: %v", name, err)
		}
		out[name] = string(b)
	}

	return out
}

func TestBranding_CustomProductNameReachesWordmarkAndTitle(t *testing.T) {
	pages := renderAllPages(t, options.Branding{
		ProductName:     "Acme Passwords",
		PageTitle:       "Acme — password self service",
		ShowAttribution: true,
	})

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

	// Sub-pages keep their own prefix but adopt the configured name.
	if !strings.Contains(pages["forgot-password"], "<title>Forgot Password — Acme Passwords</title>") {
		t.Error("forgot-password should combine its own prefix with the configured name")
	}
	if !strings.Contains(pages["reset-password"], "<title>Reset Password — Acme Passwords</title>") {
		t.Error("reset-password should combine its own prefix with the configured name")
	}
}

func TestBranding_AttributionCanBeHidden(t *testing.T) {
	shown := renderAllPages(t, options.Branding{ProductName: "Acme", ShowAttribution: true})
	hidden := renderAllPages(t, options.Branding{ProductName: "Acme", ShowAttribution: false})

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
	without := renderAllPages(t, options.Branding{ProductName: "Acme", ShowAttribution: true})
	with := renderAllPages(t, options.Branding{ProductName: "Acme", ShowAttribution: true, HasDarkLogo: true})

	for name := range without {
		if strings.Contains(without[name], "logo-dark.webp") {
			t.Errorf("%s must not reference a dark logo that was never supplied", name)
		}
		if strings.Contains(without[name], "dark:hidden") {
			t.Errorf("%s must not hide the only logo in dark mode", name)
		}

		if !strings.Contains(with[name], "logo-dark.webp") {
			t.Errorf("%s should reference the dark logo when supplied", name)
		}
		// Class-based toggling, not a prefers-color-scheme media query, so the
		// variant follows the in-page theme switch.
		if !strings.Contains(with[name], "dark:hidden") || !strings.Contains(with[name], "dark:block") {
			t.Errorf("%s should toggle the logo variants with Tailwind dark classes", name)
		}
	}
}

// A logo-only deployment must name the logo, otherwise the page has no
// accessible name. options refuses that combination; this guards the template
// half of the contract.
func TestBranding_LogoAltIsRenderedWhenNoWordmarkIsShown(t *testing.T) {
	pages := renderAllPages(t, options.Branding{LogoAlt: "Acme Corporation", ShowAttribution: true})

	for name, html := range pages {
		if !strings.Contains(html, `alt="Acme Corporation"`) {
			t.Errorf("%s should render the configured logo alt text", name)
		}
	}
}

// With a wordmark present the logo stays decorative: announcing the name twice
// is a WCAG 1.1.1 nuisance rather than a help.
func TestBranding_LogoStaysDecorativeAlongsideAWordmark(t *testing.T) {
	pages := renderAllPages(t, options.Branding{ProductName: "Acme", ShowAttribution: true})

	for name, html := range pages {
		if !strings.Contains(html, `alt=""`) {
			t.Errorf("%s should keep the logo decorative while a wordmark is shown", name)
		}
	}
}

// Branding values are attacker-irrelevant but operator-supplied; html/template
// must still escape them rather than letting markup through.
func TestBranding_ValuesAreEscaped(t *testing.T) {
	pages := renderAllPages(t, options.Branding{
		ProductName:     `Acme<script>alert(1)</script>`,
		ShowAttribution: true,
	})

	for name, html := range pages {
		if strings.Contains(html, "<script>alert(1)</script>") {
			t.Errorf("%s rendered unescaped markup from the product name", name)
		}
	}
}
