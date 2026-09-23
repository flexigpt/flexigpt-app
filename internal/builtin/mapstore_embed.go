package builtin

import (
	"embed"
)

//go:embed tools
var BuiltInToolBundlesFS embed.FS

const (
	BuiltInToolBundlesRootDir = "tools"
	BuiltInToolBundlesJSON    = "tools.bundles.json"
)
