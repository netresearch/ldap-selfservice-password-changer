package templates

import "embed"

//go:embed *.html
var Templates embed.FS

type InputOpts struct {
	Name        string
	Placeholder string
	Type        string
}

func MakeInputOpts(name, placeholder, type_ string) InputOpts {
	if type_ != "password" && type_ != "text" {
		panic("InputOpts type must be either `password` or `text`")
	}

	return InputOpts{
		name,
		placeholder,
		type_,
	}
}
