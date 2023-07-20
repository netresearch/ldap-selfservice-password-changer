package templates

import (
	"bytes"
	_ "embed"
	"html/template"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/options"
)

//go:embed index.html
var rawIndex string

type InputOpts struct {
	Name         string
	Placeholder  string
	Type         string
	Autocomplete string
}

func MakeInputOpts(name, placeholder, type_, autocomplete string) InputOpts {
	if type_ != "password" && type_ != "text" {
		panic("InputOpts type must be either `password` or `text`")
	}

	return InputOpts{
		name,
		placeholder,
		type_,
		autocomplete,
	}
}

func RenderIndex(opts *options.Opts) ([]byte, error) {
	funcs := template.FuncMap{"InputOpts": MakeInputOpts}

	tpl, err := template.New("index").Funcs(funcs).Parse(rawIndex)
	if err != nil {
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
