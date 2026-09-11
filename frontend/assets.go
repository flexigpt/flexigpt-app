package frontend

import (
	"embed"
)

//go:embed all:dist/client
var Assets embed.FS

//go:embed public/icon.png
var Icon []byte
