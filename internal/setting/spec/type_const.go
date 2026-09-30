package spec

import "errors"

const (
	SchemaVersion = "2026-03-28"
	SettingsFile  = "settings.json"
)

var (
	ErrInvalidArgument      = errors.New("invalid argument")
	ErrInvalidTheme         = errors.New("invalid app theme")
	ErrInvalidDebugSettings = errors.New("invalid debug settings")
)

type ThemeType string

const (
	ThemeSystem ThemeType = "system"
	ThemeLight  ThemeType = "light"
	ThemeDark   ThemeType = "dark"
	ThemeOther  ThemeType = "other"
)

const (
	ThemeNameSystem = "system"
	ThemeNameLight  = "applight"
	ThemeNameDark   = "appdark"
)

type AppTheme struct {
	Type ThemeType `json:"type"`
	Name string    `json:"name"`
}

type DebugLogLevel string

const (
	DebugLogLevelDebug DebugLogLevel = "debug"
	DebugLogLevelInfo  DebugLogLevel = "info"
	DebugLogLevelWarn  DebugLogLevel = "warn"
	DebugLogLevelError DebugLogLevel = "error"
)

type DebugSettings struct {
	LogLLMReqResp           bool          `json:"logLLMReqResp"`
	DisableContentStripping bool          `json:"disableContentStripping"`
	LogLevel                DebugLogLevel `json:"logLevel"`
}

type SettingsSchema struct {
	SchemaVersion string        `json:"schemaVersion"`
	AppTheme      AppTheme      `json:"appTheme"`
	Debug         DebugSettings `json:"debug"`
}
