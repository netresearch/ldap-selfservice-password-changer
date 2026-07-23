package options

import (
	"fmt"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/web/static"
)

// DefaultProductName is the wordmark shown when BRANDING_PRODUCT_NAME is unset.
const DefaultProductName = "GopherPass"

// defaultPageTitleSuffix is appended to the product name when no explicit page
// title is configured.
const defaultPageTitleSuffix = " — Self-service password change & reset"

// Branding holds the customisable presentation of the web UI. The zero value
// is not usable directly; build it with buildBranding so the derived fields
// and accessibility guarantees hold.
type Branding struct {
	// Dir is the operator-supplied directory layered over the embedded static
	// assets. Empty means no overrides.
	Dir string

	// ProductName is the wordmark rendered next to the logo. Empty renders no
	// wordmark, which is only permitted alongside a non-empty LogoAlt.
	ProductName string

	// PageTitle is the browser tab title.
	PageTitle string

	// LogoAlt is the logo's alternative text. Empty marks the logo decorative,
	// which is correct while a wordmark carries the accessible name.
	LogoAlt string

	// ShowAttribution controls the "Built by Netresearch" footer line.
	ShowAttribution bool

	// HasDarkLogo reports whether the branding directory supplies a dark-mode
	// logo variant.
	HasDarkLogo bool
}

// Name returns the deployment's display name: the wordmark when one is shown,
// otherwise the logo's alternative text. buildBranding guarantees at least one
// of the two is set, so the result is never empty and sub-page titles never
// degrade to a dangling separator.
func (b Branding) Name() string {
	if b.ProductName != "" {
		return b.ProductName
	}

	return b.LogoAlt
}

// Normalized fills in empty presentation fields so that rendering an Opts
// built by hand — in tests, or by a future caller that bypasses LoadOptions —
// still yields a titled page with an accessible name instead of an empty
// <title> and an unnamed logo.
//
// It only ever fills empty strings; it never overrides a configured value and
// never flips ShowAttribution, so an operator who deliberately clears the
// wordmark in favor of a logo keeps exactly what they asked for.
func (b Branding) Normalized() Branding {
	if b.ProductName == "" && b.LogoAlt == "" {
		b.ProductName = DefaultProductName
	}
	if b.PageTitle == "" {
		b.PageTitle = b.Name() + defaultPageTitleSuffix
	}

	return b
}

// buildBranding derives the branding configuration and records any problem on
// errs so startup fails before the first request rather than serving a broken
// or inaccessible page.
func buildBranding(dir, productName, pageTitle, logoAlt string, showAttribution bool, errs *ConfigError) Branding {
	// The logo is decorative (alt="") because the wordmark beside it carries
	// the accessible name. Removing the wordmark without supplying alternative
	// text would leave the page with no accessible name at all — a WCAG 1.1.1
	// failure that is invisible to a sighted operator testing their own
	// rebrand, so it is refused rather than warned about.
	if productName == "" && logoAlt == "" {
		errs.Add(
			"BRANDING_PRODUCT_NAME is empty, so the logo is the only branding left: " +
				"set BRANDING_LOGO_ALT to the name it depicts, otherwise the page has no accessible name",
		)
	}

	if pageTitle == "" {
		if productName == "" {
			pageTitle = logoAlt + defaultPageTitleSuffix
		} else {
			pageTitle = productName + defaultPageTitleSuffix
		}
	}

	hasDarkLogo, err := static.ValidateOverlay(dir)
	if err != nil {
		errs.Add(fmt.Sprintf("invalid value for BRANDING_DIR: %v", err))
	}

	return Branding{
		Dir:             dir,
		ProductName:     productName,
		PageTitle:       pageTitle,
		LogoAlt:         logoAlt,
		ShowAttribution: showAttribution,
		HasDarkLogo:     hasDarkLogo,
	}
}
