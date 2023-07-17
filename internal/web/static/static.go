package static

import "embed"

//go:embed *.css js/*.js
var Static embed.FS
