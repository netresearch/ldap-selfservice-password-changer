// Package static embeds static web assets (CSS, JS, images) for the application.
package static

import "embed"

// Static embeds all static web assets including CSS, JavaScript, images, and manifest files.
//
//go:embed *.css js/*.js *.png *.ico *.svg *.webp site.webmanifest browserconfig.xml
var Static embed.FS
