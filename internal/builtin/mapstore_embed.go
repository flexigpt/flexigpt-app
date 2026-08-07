package builtin

import (
	"embed"
)

//go:embed tools
var BuiltInToolBundlesFS embed.FS

//go:embed skills
var BuiltInSkillBundlesFS embed.FS

//go:embed assistantpresets
var BuiltInAssistantPresetBundlesFS embed.FS

//go:embed mcp
var BuiltInMCPBundlesFS embed.FS

const (
	BuiltInToolBundlesRootDir = "tools"
	BuiltInToolBundlesJSON    = "tools.bundles.json"

	BuiltInSkillBundlesRootDir = "skills"
	BuiltInSkillBundlesJSON    = "skills.json"

	BuiltInAssistantPresetBundlesRootDir = "assistantpresets"
	BuiltInAssistantPresetBundlesJSON    = "assistantpresets.bundles.json"

	BuiltInMCPBundlesRootDir = "mcp"
	BuiltInMCPBundlesJSON    = "mcp.bundles.json"
)
