package store

import (
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/builtin"
	"github.com/flexigpt/flexigpt-app/internal/setting/spec"
)

// BuiltInAuthKeys lists every key that must never be deleted.
var BuiltInAuthKeys = func() map[spec.AuthKeyType][]spec.AuthKeyName {
	m := map[spec.AuthKeyType][]spec.AuthKeyName{
		spec.AuthKeyTypeProvider: {},
	}
	for _, p := range builtin.BuiltInProviderNames {
		m[spec.AuthKeyTypeProvider] = append(
			m[spec.AuthKeyTypeProvider],
			spec.AuthKeyName(p),
		)
	}
	return m
}()

// DefaultDebugSettingsData is written to disk on first start and used for migrations.
var DefaultDebugSettingsData = spec.DebugSettings{
	LogLLMReqResp:           false,
	DisableContentStripping: false,
	LogLevel:                spec.DebugLogLevelInfo,
}

// DefaultSettingsData is written to disk on first start.
var DefaultSettingsData = func() spec.SettingsSchema {
	ak := spec.AuthKeysSchema{
		spec.AuthKeyTypeProvider: map[spec.AuthKeyName]spec.AuthKey{},
	}
	for _, p := range builtin.BuiltInProviderNames {
		ak[spec.AuthKeyTypeProvider][spec.AuthKeyName(p)] = spec.AuthKey{
			Secret:   "",
			SHA256:   computeSHA(""),
			NonEmpty: false,
		}
	}
	return spec.SettingsSchema{
		SchemaVersion: spec.SchemaVersion,
		AppTheme:      spec.AppTheme{Type: spec.ThemeSystem, Name: string(spec.ThemeSystem)},
		Debug:         DefaultDebugSettingsData,
		AuthKeys:      ak,
	}
}()

// isBuiltInKey reports whether type/keyName is part of BuiltInAuthKeys.
func isBuiltInKey(t spec.AuthKeyType, name spec.AuthKeyName) bool {
	if names, ok := BuiltInAuthKeys[t]; ok {
		return slices.Contains(names, name)
	}
	return false
}
