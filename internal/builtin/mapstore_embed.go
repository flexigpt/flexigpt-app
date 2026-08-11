package builtin

import (
	"embed"
)

//go:embed tools
var BuiltInToolBundlesFS embed.FS

//go:embed assistantpresets
var BuiltInAssistantPresetBundlesFS embed.FS

const (
	BuiltInToolBundlesRootDir = "tools"
	BuiltInToolBundlesJSON    = "tools.bundles.json"

	BuiltInAssistantPresetBundlesRootDir = "assistantpresets"
	BuiltInAssistantPresetBundlesJSON    = "assistantpresets.bundles.json"
)
