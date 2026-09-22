//nolint:testpackage // exercises the same template renderers as branding_test.go
package templates

import (
	"strings"
	"testing"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/options"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/web/static"
)

func TestKeyholderProgressiveEnhancement(t *testing.T) {
	opts := &options.Opts{Branding: options.DefaultBranding()}
	page, err := RenderIndex(opts)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		`character="keyholder"`, `src="/static/logo.webp"`,
		`src="/static/companion/keyholder-login.js"`, `autocomplete="current-password"`,
	} {
		if !strings.Contains(string(page), required) {
			t.Errorf("missing login enhancement or fallback: %s", required)
		}
	}
	for _, asset := range []string{
		"avatar.css", "keyholder-login.js", "scormiq-avatar.js", "gopher-rigs.js",
		"vendor/three.module.js", "assets/logos/netresearch-symbol-only.svg",
	} {
		if data, readErr := static.Static.ReadFile("companion/" + asset); readErr != nil || len(data) == 0 {
			t.Errorf("companion asset %s is not embedded: %v", asset, readErr)
		}
	}
}

func TestKeyholderPreservesOperatorBranding(t *testing.T) {
	brand, err := options.NewBranding(t.TempDir(), "Acme", "", "", true)
	if err != nil {
		t.Fatal(err)
	}
	page, err := RenderIndex(&options.Opts{Branding: brand})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(page), "scormiq-avatar") || !strings.Contains(string(page), `src="/static/logo.webp"`) {
		t.Fatal("operator branding must keep its logo without loading the companion")
	}
}
