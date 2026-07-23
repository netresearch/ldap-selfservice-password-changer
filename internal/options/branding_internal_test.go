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

// The wordmark and the logo alt text describe the same brand. Rendering both
// makes a screen reader announce it twice, so the alt text is the part that
// yields — the combination is accepted rather than refused, because an
// operator who later clears the wordmark should not have to remember to put
// the alt text back.
func TestBranding_LogoAltTextYieldsToTheWordmark(t *testing.T) {
	tests := []struct {
		name        string
		productName string
		logoAlt     string
		wantAlt     string
	}{
		{"wordmark only", "Acme", "", ""},
		{"logo only", "", "Acme Corporation", "Acme Corporation"},
		{"both configured", "Acme", "Acme Corporation", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := NewBranding("", tt.productName, "", tt.logoAlt, true)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := b.LogoAltText(); got != tt.wantAlt {
				t.Errorf("LogoAltText() = %q, want %q", got, tt.wantAlt)
			}
		})
	}
}

// DefaultBranding must be exactly what a deployment configuring nothing gets,
// since tests build their Opts from it.
func TestDefaultBranding_MatchesAnUnconfiguredDeployment(t *testing.T) {
	t.Setenv("LDAP_SERVER", "ldaps://ldap.example.com:636")
	t.Setenv("LDAP_BASE_DN", "dc=example,dc=com")
	t.Setenv("LDAP_READONLY_USER", "cn=readonly,dc=example,dc=com")
	t.Setenv("LDAP_READONLY_PASSWORD", "secret")
	t.Setenv("BRANDING_DIR", "")
	t.Setenv("BRANDING_PRODUCT_NAME", DefaultProductName)
	t.Setenv("BRANDING_PAGE_TITLE", "")
	t.Setenv("BRANDING_LOGO_ALT", "")
	t.Setenv("BRANDING_SHOW_ATTRIBUTION", "true")

	opts, err := ParseArgs(nil)
	if err != nil {
		t.Fatalf("ParseArgs: %v", err)
	}

	if opts.Branding != DefaultBranding() {
		t.Errorf("ParseArgs branding %+v differs from DefaultBranding() %+v", opts.Branding, DefaultBranding())
	}
}

// Every branding variable must survive the environment-parsing layer. Testing
// NewBranding alone leaves the flag wiring uncovered, which is exactly where a
// silently-ignored setting hides.
func TestParseArgs_BrandingVariablesReachOpts(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, static.DarkLogo), []byte("dark"), 0o600); err != nil {
		t.Fatalf("write dark logo: %v", err)
	}

	t.Setenv("LDAP_SERVER", "ldaps://ldap.example.com:636")
	t.Setenv("LDAP_BASE_DN", "dc=example,dc=com")
	t.Setenv("LDAP_READONLY_USER", "cn=readonly,dc=example,dc=com")
	t.Setenv("LDAP_READONLY_PASSWORD", "secret")
	t.Setenv("BRANDING_DIR", dir)
	t.Setenv("BRANDING_PRODUCT_NAME", "Acme Passwords")
	t.Setenv("BRANDING_PAGE_TITLE", "Acme portal")
	t.Setenv("BRANDING_LOGO_ALT", "Acme Corporation")
	t.Setenv("BRANDING_SHOW_ATTRIBUTION", "false")

	opts, err := ParseArgs(nil)
	if err != nil {
		t.Fatalf("ParseArgs: %v", err)
	}

	got := opts.Branding
	if got.Dir != dir {
		t.Errorf("Dir = %q, want %q", got.Dir, dir)
	}
	if got.ProductName != "Acme Passwords" {
		t.Errorf("ProductName = %q", got.ProductName)
	}
	if got.PageTitle != "Acme portal" {
		t.Errorf("PageTitle = %q", got.PageTitle)
	}
	if got.LogoAlt != "Acme Corporation" {
		t.Errorf("LogoAlt = %q", got.LogoAlt)
	}
	if got.ShowAttribution {
		t.Error("ShowAttribution = true, want the configured false")
	}
	if !got.HasDarkLogo {
		t.Error("HasDarkLogo = false, want the dark logo in BRANDING_DIR to be detected")
	}
}

// A branding directory the overlay refuses must stop startup, not be ignored.
func TestParseArgs_RejectsAnInvalidBrandingDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "styles.css"), []byte("body{}"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	t.Setenv("LDAP_SERVER", "ldaps://ldap.example.com:636")
	t.Setenv("LDAP_BASE_DN", "dc=example,dc=com")
	t.Setenv("LDAP_READONLY_USER", "cn=readonly,dc=example,dc=com")
	t.Setenv("LDAP_READONLY_PASSWORD", "secret")
	t.Setenv("BRANDING_DIR", dir)

	if _, err := ParseArgs(nil); err == nil {
		t.Fatal("expected a branding directory holding a non-overridable file to abort startup")
	}
}
