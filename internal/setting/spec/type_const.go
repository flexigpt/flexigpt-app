package spec

import "errors"

const (
	SchemaVersion = "2026-03-27"
	SettingsFile  = "settings.json"
)

var (
	ErrInvalidArgument        = errors.New("invalid argument")
	ErrInvalidTheme           = errors.New("invalid app theme")
	ErrInvalidAuthKey         = errors.New("invalid auth key")
	ErrInvalidDebugSettings   = errors.New("invalid debug settings")
	ErrAuthKeyNotFound        = errors.New("auth key not found")
	ErrBuiltInAuthKeyReadOnly = errors.New("built-in auth key is read-only")
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

// AuthKeyType groups keys (e.g. "provider", "github").
type AuthKeyType string

const AuthKeyTypeProvider AuthKeyType = "provider"

// AuthKeyName is the unique key within its type.
type AuthKeyName string

// AuthKey holds an encrypted secret plus its SHA-256 hash.
type AuthKey struct {
	Secret   string `json:"secret"` // encrypted on disk
	SHA256   string `json:"sha256"` // plain text
	NonEmpty bool   `json:"nonEmpty"`
}

type AuthKeysSchema map[AuthKeyType]map[AuthKeyName]AuthKey

type SettingsSchema struct {
	SchemaVersion string         `json:"schemaVersion"`
	AppTheme      AppTheme       `json:"appTheme"`
	Debug         DebugSettings  `json:"debug"`
	AuthKeys      AuthKeysSchema `json:"authKeys"`
}
