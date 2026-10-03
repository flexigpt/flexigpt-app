package store

import settingSpec "github.com/flexigpt/flexigpt-app/internal/setting/spec"

// DefaultDebugSettingsData is written to disk on first start.
var DefaultDebugSettingsData = settingSpec.DebugSettings{
	LogLLMReqResp:           false,
	DisableContentStripping: false,
	LogLevel:                settingSpec.DebugLogLevelInfo,
}

// DefaultSettingsData is written to disk on first start.
var DefaultSettingsData = settingSpec.SettingsSchema{
	SchemaVersion: settingSpec.SchemaVersion,
	AppTheme: settingSpec.AppTheme{
		Type: settingSpec.ThemeSystem,
		Name: settingSpec.ThemeNameSystem,
	},
	Debug: DefaultDebugSettingsData,
}
