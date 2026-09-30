package store

import "github.com/flexigpt/flexigpt-app/internal/setting/spec"

// DefaultDebugSettingsData is written to disk on first start.
var DefaultDebugSettingsData = spec.DebugSettings{
	LogLLMReqResp:           false,
	DisableContentStripping: false,
	LogLevel:                spec.DebugLogLevelInfo,
}

// DefaultSettingsData is written to disk on first start.
var DefaultSettingsData = spec.SettingsSchema{
	SchemaVersion: spec.SchemaVersion,
	AppTheme: spec.AppTheme{
		Type: spec.ThemeSystem,
		Name: spec.ThemeNameSystem,
	},
	Debug: DefaultDebugSettingsData,
}
