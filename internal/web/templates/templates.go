// Package templates provides HTML template rendering for web pages.
package templates

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/options"
)

//go:embed index.html
var rawIndex string

//go:embed forgot-password.html
var rawForgotPassword string

//go:embed reset-password.html
var rawResetPassword string

//go:embed atoms/button-primary.html
var rawButtonPrimary string

//go:embed atoms/button-secondary.html
var rawButtonSecondary string

//go:embed atoms/link.html
var rawLink string

//go:embed atoms/button-toggle.html
var rawButtonToggle string

//go:embed atoms/icons.html
var rawIcons string

//go:embed molecules/input-field.html
var rawInputField string

//go:embed molecules/theme-init-script.html
var rawThemeInitScript string

//go:embed molecules/density-init-script.html
var rawDensityInitScript string

//go:embed molecules/html-head.html
var rawHTMLHead string

//go:embed molecules/page-header.html
var rawPageHeader string

//go:embed molecules/page-footer.html
var rawPageFooter string

//go:embed molecules/toggle-buttons.html
var rawToggleButtons string

//go:embed molecules/form-submit.html
var rawFormSubmit string

//go:embed molecules/success-message.html
var rawSuccessMessage string

//go:embed molecules/page-title.html
var rawPageTitle string

// InputOpts holds configuration for rendering HTML input fields with validation and accessibility attributes.
type InputOpts struct {
	Name         string
	Placeholder  string
	Type         string
	Autocomplete string
	Help         string
}

// MakeInputOpts creates an InputOpts configuration for rendering form input fields.
func MakeInputOpts(name, placeholder, inputType, autocomplete, help string) InputOpts {
	if inputType != "password" && inputType != "text" {
		panic("InputOpts type must be either `password` or `text`")
	}

	return InputOpts{
		name,
		placeholder,
		inputType,
		autocomplete,
		help,
	}
}

func makeSlice(args ...interface{}) []interface{} {
	return args
}

// parseCommonTemplates parses all shared atom and molecule templates.
func parseCommonTemplates(tpl *template.Template) error {
	templates := []string{
		rawIcons,
		rawButtonPrimary,
		rawButtonSecondary,
		rawLink,
		rawButtonToggle,
		rawInputField,
		rawThemeInitScript,
		rawDensityInitScript,
		rawHTMLHead,
		rawPageHeader,
		rawPageFooter,
		rawToggleButtons,
		rawFormSubmit,
		rawSuccessMessage,
		rawPageTitle,
	}

	for _, tmpl := range templates {
		if _, err := tpl.Parse(tmpl); err != nil {
			return fmt.Errorf("failed to parse common template: %w", err)
		}
	}

	return nil
}

// renderTemplate renders a template with common setup logic.
func renderTemplate(templateName, rawTemplate string, data any) ([]byte, error) {
	funcs := template.FuncMap{
		"InputOpts": MakeInputOpts,
		"slice":     makeSlice,
	}

	// Parse dependencies first, BEFORE parsing main template
	tpl := template.New(templateName).Funcs(funcs)

	// Parse common atom and molecule templates
	if err := parseCommonTemplates(tpl); err != nil {
		return nil, err
	}

	// NOW parse the main template after all dependencies are defined
	if _, err := tpl.Parse(rawTemplate); err != nil {
		return nil, fmt.Errorf("failed to parse %s template: %w", templateName, err)
	}

	var buf bytes.Buffer
	if err := tpl.ExecuteTemplate(&buf, templateName, data); err != nil {
		return nil, fmt.Errorf("failed to execute %s template: %w", templateName, err)
	}

	return buf.Bytes(), nil
}

// RenderIndex renders the main password change page with the provided configuration options.
func RenderIndex(opts *options.Opts) ([]byte, error) {
	data := map[string]any{
		"opts": opts,
	}
	return renderTemplate("index", rawIndex, data)
}

// RenderForgotPassword renders the password reset request page.
func RenderForgotPassword() ([]byte, error) {
	return renderTemplate("forgot-password", rawForgotPassword, nil)
}

// RenderResetPassword renders the password reset completion page with the provided configuration options.
func RenderResetPassword(opts *options.Opts) ([]byte, error) {
	data := map[string]any{
		"opts": opts,
	}
	return renderTemplate("reset-password", rawResetPassword, data)
}
