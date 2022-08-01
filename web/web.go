// Embed templates

package web

import (
	"embed"
)

//go:embed static template
var Files embed.FS
