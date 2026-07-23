package options

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/web/static"
)

func TestBuildBranding_DefaultsReproduceTheStockAppearance(t *testing.T) {
	errs := &ConfigError{}

	got := buildBranding("", DefaultProductName, "", "", true, errs)

	if errs.HasErrors() {
		t.Fatalf("unexpected errors: %v", errs.Errors)
	}
	if got.ProductName != DefaultProductName {
		t.Errorf("ProductName = %q, want %q", got.ProductName, DefaultProductName)
	}
	if want := DefaultProductName + defaultPageTitleSuffix; got.PageTitle != want {
		t.Errorf("PageTitle = %q, want %q", got.PageTitle, want)
	}
	if got.LogoAlt != "" {
		t.Errorf("LogoAlt = %q, want empty so the logo stays decorative", got.LogoAlt)
	}
	if !got.ShowAttribution {
		t.Error("ShowAttribution = false, want true by default")
	}
}

func TestBuildBranding_ExplicitPageTitleIsKept(t *testing.T) {
	errs := &ConfigError{}

	got := buildBranding("", "Acme Passwords", "Reset your Acme password", "", true, errs)

	if errs.HasErrors() {
		t.Fatalf("unexpected errors: %v", errs.Errors)
	}
	if got.PageTitle != "Reset your Acme password" {
		t.Errorf("PageTitle = %q, want the configured value", got.PageTitle)
	}
}

// Dropping the wordmark leaves the logo as the only branding. Without
// alternative text the page would have no accessible name at all, which a
// sighted operator testing their own rebrand would never notice.
func TestBuildBranding_EmptyProductNameWithoutLogoAltIsRejected(t *testing.T) {
	errs := &ConfigError{}

	buildBranding("", "", "", "", true, errs)

	if !errs.HasErrors() {
		t.Fatal("expected an error when both the wordmark and the logo alt text are empty")
	}
	if !strings.Contains(strings.Join(errs.Errors, "\n"), "BRANDING_LOGO_ALT") {
		t.Errorf("error should name BRANDING_LOGO_ALT, got %v", errs.Errors)
	}
}

func TestBuildBranding_LogoOnlyBrandingDerivesTitleFromAltText(t *testing.T) {
	errs := &ConfigError{}

	got := buildBranding("", "", "", "Acme Corporation", true, errs)

	if errs.HasErrors() {
		t.Fatalf("unexpected errors: %v", errs.Errors)
	}
	if want := "Acme Corporation" + defaultPageTitleSuffix; got.PageTitle != want {
		t.Errorf("PageTitle = %q, want %q", got.PageTitle, want)
	}
	if got.Name() != "Acme Corporation" {
		t.Errorf("Name() = %q, want the logo alt text", got.Name())
	}
}

func TestBuildBranding_InvalidDirectoryIsReported(t *testing.T) {
	errs := &ConfigError{}

	buildBranding(filepath.Join(t.TempDir(), "absent"), DefaultProductName, "", "", true, errs)

	if !errs.HasErrors() {
		t.Fatal("expected a missing branding directory to be reported")
	}
	if !strings.Contains(strings.Join(errs.Errors, "\n"), "BRANDING_DIR") {
		t.Errorf("error should name BRANDING_DIR, got %v", errs.Errors)
	}
}

func TestBuildBranding_DetectsDarkLogo(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, static.DarkLogo), []byte("dark"), 0o600); err != nil {
		t.Fatalf("write dark logo: %v", err)
	}
	errs := &ConfigError{}

	got := buildBranding(dir, DefaultProductName, "", "", true, errs)

	if errs.HasErrors() {
		t.Fatalf("unexpected errors: %v", errs.Errors)
	}
	if !got.HasDarkLogo {
		t.Error("HasDarkLogo = false, want true")
	}
}

// Regression: an explicitly empty BRANDING_PRODUCT_NAME must clear the
// wordmark. The shared envStringOrDefault helper treats set-but-empty as
// unset, which silently restored "GopherPass" and made the logo-only layout
// unreachable for container deployments, where environment variables are the
// only configuration channel. Testing buildBranding alone does not catch this
// — the defect lives in the environment-parsing layer above it.
func TestParseArgs_EmptyProductNameClearsTheWordmark(t *testing.T) {
	t.Setenv("LDAP_SERVER", "ldaps://ldap.example.com:636")
	t.Setenv("LDAP_BASE_DN", "dc=example,dc=com")
	t.Setenv("LDAP_READONLY_USER", "cn=readonly,dc=example,dc=com")
	t.Setenv("LDAP_READONLY_PASSWORD", "secret")
	t.Setenv("BRANDING_PRODUCT_NAME", "")
	t.Setenv("BRANDING_LOGO_ALT", "Acme Corporation")

	opts, err := ParseArgs(nil)
	if err != nil {
		t.Fatalf("expected the logo-only configuration to be accepted, got %v", err)
	}

	if opts.Branding.ProductName != "" {
		t.Errorf("ProductName = %q, want it cleared by the empty environment variable", opts.Branding.ProductName)
	}
	if opts.Branding.Name() != "Acme Corporation" {
		t.Errorf("Name() = %q, want the logo alt text", opts.Branding.Name())
	}
}

// The same empty value without alternative text must still be refused: it is
// the combination that strips the page of any accessible name.
func TestParseArgs_EmptyProductNameWithoutAltIsRefused(t *testing.T) {
	t.Setenv("LDAP_SERVER", "ldaps://ldap.example.com:636")
	t.Setenv("LDAP_BASE_DN", "dc=example,dc=com")
	t.Setenv("LDAP_READONLY_USER", "cn=readonly,dc=example,dc=com")
	t.Setenv("LDAP_READONLY_PASSWORD", "secret")
	t.Setenv("BRANDING_PRODUCT_NAME", "")

	if _, err := ParseArgs(nil); err == nil {
		t.Fatal("expected startup to fail when nothing carries the accessible name")
	}
}

func TestBranding_Normalized(t *testing.T) {
	tests := []struct {
		name            string
		in              Branding
		wantProductName string
		wantPageTitle   string
		wantAttribution bool
	}{
		{
			name:            "zero value gains the stock name and a title",
			in:              Branding{},
			wantProductName: DefaultProductName,
			wantPageTitle:   DefaultProductName + defaultPageTitleSuffix,
			wantAttribution: false,
		},
		{
			name:            "configured values are left alone",
			in:              Branding{ProductName: "Acme", PageTitle: "Acme SSO", ShowAttribution: true},
			wantProductName: "Acme",
			wantPageTitle:   "Acme SSO",
			wantAttribution: true,
		},
		{
			name:            "deliberate logo-only branding keeps the empty wordmark",
			in:              Branding{LogoAlt: "Acme", ShowAttribution: true},
			wantProductName: "",
			wantPageTitle:   "Acme" + defaultPageTitleSuffix,
			wantAttribution: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.in.Normalized()

			if got.ProductName != tt.wantProductName {
				t.Errorf("ProductName = %q, want %q", got.ProductName, tt.wantProductName)
			}
			if got.PageTitle != tt.wantPageTitle {
				t.Errorf("PageTitle = %q, want %q", got.PageTitle, tt.wantPageTitle)
			}
			if got.ShowAttribution != tt.wantAttribution {
				t.Errorf("ShowAttribution = %v, want %v", got.ShowAttribution, tt.wantAttribution)
			}
		})
	}
}
