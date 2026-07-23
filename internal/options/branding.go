package options

import (
	"errors"
	"fmt"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/web/static"
)

// DefaultProductName is the wordmark shown when BRANDING_PRODUCT_NAME is unset.
const DefaultProductName = "GopherPass"

// defaultPageTitleSuffix is appended to the product name when no explicit page
// title is configured.
const defaultPageTitleSuffix = " — Self-service password change & reset"

// ErrNoBrandName is returned when neither a wordmark nor logo alt text is
// configured, leaving the page header with nothing but a decorative image.
var ErrNoBrandName = errors.New(
	"BRANDING_PRODUCT_NAME is empty, so the logo is the only branding left: " +
		"set BRANDING_LOGO_ALT to the name it depicts, otherwise the header conveys the brand " +
		"to sighted users only",
)

// Branding holds the customisable presentation of the web UI. Build it with
// NewBranding — the derived fields and the accessibility guarantees do not
// hold for a hand-assembled value.
type Branding struct {
	// Dir is the operator-supplied directory layered over the embedded static
	// assets. Empty means no overrides.
	Dir string

	// ProductName is the wordmark rendered next to the logo. Empty renders no
	// wordmark, which is only permitted alongside a non-empty LogoAlt.
	ProductName string

	// PageTitle is the browser tab title of the start page. The reset pages
	// keep their own prefix and append Name instead.
	PageTitle string

	// LogoAlt describes the logo for deployments that show no wordmark. It is
	// deliberately ignored while a wordmark is rendered — see LogoAltText.
	LogoAlt string

	// ShowAttribution controls the "Built by Netresearch" footer line.
	ShowAttribution bool

	// HasDarkLogo reports whether the branding directory supplies a dark-mode
	// logo variant.
	HasDarkLogo bool
}

// NewBranding derives and validates a branding configuration.
//
// The only rejected combination is an empty wordmark with no logo alt text:
// the header would then consist of a decorative image alone, so the brand
// would reach sighted users only. Everything else has a sensible default.
func NewBranding(dir, productName, pageTitle, logoAlt string, showAttribution bool) (Branding, error) {
	b := Branding{
		Dir:             dir,
		ProductName:     productName,
		PageTitle:       pageTitle,
		LogoAlt:         logoAlt,
		ShowAttribution: showAttribution,
	}

	if productName == "" && logoAlt == "" {
		return b, ErrNoBrandName
	}

	if b.PageTitle == "" {
		b.PageTitle = b.Name() + defaultPageTitleSuffix
	}

	hasDarkLogo, err := static.ValidateOverlay(dir)
	if err != nil {
		return b, fmt.Errorf("branding directory: %w", err)
	}
	b.HasDarkLogo = hasDarkLogo

	return b, nil
}

// DefaultBranding is the configuration a deployment that sets nothing ends up
// with. Tests that assemble an Opts by hand should start from this rather than
// from a zero Branding, which would render an empty title.
func DefaultBranding() Branding {
	b, err := NewBranding("", DefaultProductName, "", "", true)
	if err != nil {
		panic("default branding must be valid: " + err.Error())
	}

	return b
}

// Name returns the deployment's display name: the wordmark when one is shown,
// otherwise the logo's alternative text. NewBranding guarantees at least one
// of the two is set, so the result is never empty and sub-page titles never
// degrade to a dangling separator.
func (b Branding) Name() string {
	if b.ProductName != "" {
		return b.ProductName
	}

	return b.LogoAlt
}

// LogoAltText returns the alt attribute for the logo image.
//
// The logo is decorative whenever the wordmark is rendered: the wordmark
// already carries the name, and repeating it in alt text makes a screen reader
// announce the brand twice. LogoAlt therefore takes effect only when no
// wordmark is shown — configuring both is accepted, and the alt text is the
// part that yields.
func (b Branding) LogoAltText() string {
	if b.ProductName != "" {
		return ""
	}

	return b.LogoAlt
}

// buildBranding adapts NewBranding to the collecting error style used while
// parsing options, so a bad value joins the other configuration errors
// instead of aborting on the first one.
func buildBranding(dir, productName, pageTitle, logoAlt string, showAttribution bool, errs *ConfigError) Branding {
	b, err := NewBranding(dir, productName, pageTitle, logoAlt, showAttribution)
	if err != nil {
		if errors.Is(err, ErrNoBrandName) {
			errs.Add(err.Error())
		} else {
			errs.Add(fmt.Sprintf("invalid value for BRANDING_DIR: %v", err))
		}
	}

	return b
}
