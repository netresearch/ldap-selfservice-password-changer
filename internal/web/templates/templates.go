package templates

import (
	"bytes"
	_ "embed"
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
var rawHtmlHead string

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

type InputOpts struct {
	Name         string
	Placeholder  string
	Type         string
	Autocomplete string
	Help         string
}

func MakeInputOpts(name, placeholder, type_, autocomplete, help string) InputOpts {
	if type_ != "password" && type_ != "text" {
		panic("InputOpts type must be either `password` or `text`")
	}

	return InputOpts{
		name,
		placeholder,
		type_,
		autocomplete,
		help,
	}
}

func makeSlice(args ...interface{}) []interface{} {
	return args
}

func RenderIndex(opts *options.Opts) ([]byte, error) {
	funcs := template.FuncMap{
		"InputOpts": MakeInputOpts,
		"slice":     makeSlice,
	}

	// Parse dependencies first, BEFORE parsing main template
	tpl := template.New("index").Funcs(funcs)
	var err error

	// Parse atom and molecule templates
	if _, err = tpl.Parse(rawIcons); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawButtonPrimary); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawButtonSecondary); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawLink); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawButtonToggle); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawInputField); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawThemeInitScript); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawDensityInitScript); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawHtmlHead); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawPageHeader); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawPageFooter); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawToggleButtons); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawFormSubmit); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawSuccessMessage); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawPageTitle); err != nil {
		return nil, err
	}

	// NOW parse the main template after all dependencies are defined
	if _, err = tpl.Parse(rawIndex); err != nil {
		return nil, err
	}

	data := map[string]any{
		"opts": opts,
	}

	var buf bytes.Buffer
	if err = tpl.ExecuteTemplate(&buf, "index", data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func RenderForgotPassword() ([]byte, error) {
	funcs := template.FuncMap{
		"InputOpts": MakeInputOpts,
		"slice":     makeSlice,
	}

	// Parse dependencies first, BEFORE parsing main template
	tpl := template.New("forgot-password").Funcs(funcs)
	var err error

	// Parse atom and molecule templates
	if _, err = tpl.Parse(rawIcons); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawButtonPrimary); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawButtonSecondary); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawLink); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawButtonToggle); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawInputField); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawThemeInitScript); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawDensityInitScript); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawHtmlHead); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawPageHeader); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawPageFooter); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawToggleButtons); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawFormSubmit); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawSuccessMessage); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawPageTitle); err != nil {
		return nil, err
	}

	// NOW parse the main template after all dependencies are defined
	if _, err = tpl.Parse(rawForgotPassword); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err = tpl.ExecuteTemplate(&buf, "forgot-password", nil); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func RenderResetPassword(opts *options.Opts) ([]byte, error) {
	funcs := template.FuncMap{
		"InputOpts": MakeInputOpts,
		"slice":     makeSlice,
	}

	// Parse dependencies first, BEFORE parsing main template
	tpl := template.New("reset-password").Funcs(funcs)
	var err error

	// Parse atom and molecule templates
	if _, err = tpl.Parse(rawIcons); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawButtonPrimary); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawButtonSecondary); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawLink); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawButtonToggle); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawInputField); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawThemeInitScript); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawDensityInitScript); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawHtmlHead); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawPageHeader); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawPageFooter); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawToggleButtons); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawFormSubmit); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawSuccessMessage); err != nil {
		return nil, err
	}
	if _, err = tpl.Parse(rawPageTitle); err != nil {
		return nil, err
	}

	// NOW parse the main template after all dependencies are defined
	if _, err = tpl.Parse(rawResetPassword); err != nil {
		return nil, err
	}

	data := map[string]any{
		"opts": opts,
	}

	var buf bytes.Buffer
	if err = tpl.ExecuteTemplate(&buf, "reset-password", data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
