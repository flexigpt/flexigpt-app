package builtin

import (
	"embed"

	"github.com/flexigpt/inference-go/modelpreset"
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

// IMPORTANT: keep these stable (match tools.bundles.json).
const (
	BuiltinBundleIDLLMToolsFS    = "018fe0f4-b8cd-7e55-82d5-9df0bd70e400"
	BuiltinBundleIDLLMToolsImage = "018fe0f4-b8cd-7e55-82d5-9df0bd70e401"
	BuiltinBundleIDLLMToolsExec  = "019c0415-40f7-70cb-9200-d804c9388a57"
	BuiltinBundleIDLLMToolsText  = "019c0da2-02d2-7e71-81d7-35f2e5a0bebf"
	BuiltinBundleIDLLMToolsGit   = "019f27b6-10ee-7858-8ad0-a20081c1a88d"
	BuiltinBundleIDLLMToolsWeb   = "019f27b6-bce1-742f-8ff9-d3a12ce4361c"
)

const (
	ProviderNameAnthropic       = string(modelpreset.ProviderAnthropic)
	ProviderNameLocalAI         = string(modelpreset.ProviderLocalAI)
	ProviderNameLMStudio        = string(modelpreset.ProviderLMStudio)
	ProviderNameGoogleGemini    = string(modelpreset.ProviderGoogleGemini)
	ProviderNameHuggingFace     = string(modelpreset.ProviderHuggingFace)
	ProviderNameLlamaCPP        = string(modelpreset.ProviderLlamaCPP)
	ProviderNameMistral         = string(modelpreset.ProviderMistral)
	ProviderNameOllama          = string(modelpreset.ProviderOllama)
	ProviderNameOpenAIChat      = string(modelpreset.ProviderOpenAIChat)
	ProviderNameOpenAIResponses = string(modelpreset.ProviderOpenAIResponses)
	ProviderNameOpenRouter      = string(modelpreset.ProviderOpenRouter)
	ProviderNameSGLang          = string(modelpreset.ProviderSGLang)
	ProviderNameVLLM            = string(modelpreset.ProviderVLLM)
	ProviderNameXAI             = string(modelpreset.ProviderXAI)
)

var BuiltInProviderNames = []string{
	ProviderNameAnthropic,
	ProviderNameLocalAI,
	ProviderNameLMStudio,
	ProviderNameGoogleGemini,
	ProviderNameHuggingFace,
	ProviderNameLlamaCPP,
	ProviderNameMistral,
	ProviderNameOllama,
	ProviderNameOpenAIChat,
	ProviderNameOpenAIResponses,
	ProviderNameOpenRouter,
	ProviderNameSGLang,
	ProviderNameVLLM,
	ProviderNameXAI,
}
