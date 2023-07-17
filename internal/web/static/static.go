package static

import "embed"

//go:embed *.css js/*.js *.png *.ico *.svg *.webp site.webmanifest browserconfig.xml
var Static embed.FS
