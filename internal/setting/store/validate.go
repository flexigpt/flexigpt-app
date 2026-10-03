package store

import (
	"fmt"

	settingSpec "github.com/flexigpt/flexigpt-app/internal/setting/spec"
)

// validateTheme checks a theme for correctness.
func validateTheme(th *settingSpec.AppTheme) error {
	if th == nil {
		return settingSpec.ErrInvalidTheme
	}
	switch th.Type {
	case settingSpec.ThemeSystem:
		if th.Name != settingSpec.ThemeNameSystem {
			return fmt.Errorf(
				"%w: type and name required. input - type %s, name %s",
				settingSpec.ErrInvalidTheme,
				th.Type,
				th.Name,
			)
		}
		return nil
	case settingSpec.ThemeLight:
		if th.Name != settingSpec.ThemeNameLight {
			return fmt.Errorf(
				"%w: type and name required. input - type %s, name %s",
				settingSpec.ErrInvalidTheme,
				th.Type,
				th.Name,
			)
		}
		return nil
	case settingSpec.ThemeDark:
		if th.Name != settingSpec.ThemeNameDark {
			return fmt.Errorf(
				"%w: type and name required. input - type %s, name %s",
				settingSpec.ErrInvalidTheme,
				th.Type,
				th.Name,
			)
		}
		return nil
	case settingSpec.ThemeOther:
		if th.Name == "" {
			return fmt.Errorf("%w: name required", settingSpec.ErrInvalidTheme)
		}
		return nil
	default:
		return settingSpec.ErrInvalidTheme
	}
}

func normalizeDebugSettings(cfg settingSpec.DebugSettings) (settingSpec.DebugSettings, bool) {
	normalized := cfg
	changed := false

	if normalized.LogLevel == "" {
		normalized.LogLevel = DefaultDebugSettingsData.LogLevel
		changed = true
	}

	if err := validateDebugSettings(&normalized); err != nil {
		normalized = DefaultDebugSettingsData
		changed = true
	}

	return normalized, changed
}

// validateDebugSettings checks whether debug settings are supported.
func validateDebugSettings(cfg *settingSpec.DebugSettings) error {
	if cfg == nil {
		return settingSpec.ErrInvalidDebugSettings
	}

	switch cfg.LogLevel {
	case settingSpec.DebugLogLevelDebug,
		settingSpec.DebugLogLevelInfo,
		settingSpec.DebugLogLevelWarn,
		settingSpec.DebugLogLevelError:
		return nil
	default:
		return fmt.Errorf("%w: unsupported logLevel %q", settingSpec.ErrInvalidDebugSettings, cfg.LogLevel)
	}
}
