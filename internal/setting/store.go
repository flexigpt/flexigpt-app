package setting

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/flexigpt/mapstore-go"
	"github.com/flexigpt/mapstore-go/jsonencdec"
)

type DebugSettingsApplier func(context.Context, DebugSettings) error

type SettingStore struct {
	store                *mapstore.MapFileStore
	debugSettingsApplier DebugSettingsApplier
}

const (
	settingKeyDebug         = "debug"
	settingKeySchemaVersion = "schemaVersion"
	settingKeyAppTheme      = "appTheme"
)

func NewSettingStore(
	baseDir string,
) (*SettingStore, error) {
	defaultMap, err := jsonencdec.StructWithJSONTagsToMap(
		DefaultSettingsData,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"cannot marshal default settings: %w",
			err,
		)
	}

	file := filepath.Join(baseDir, SettingsFile)
	fileStore, err := mapstore.NewMapFileStore(
		file,
		defaultMap,
		jsonencdec.JSONEncoderDecoder{},
		mapstore.WithCreateIfNotExists(true),
		mapstore.WithFileAutoFlush(true),
		mapstore.WithFileLogger(slog.Default()),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"settings file store initialization failed: %w",
			err,
		)
	}

	store := &SettingStore{
		store: fileStore,
	}
	if err := store.Migrate(context.Background()); err != nil {
		_ = store.Close()
		return nil, fmt.Errorf(
			"settings initialization failed: %w",
			err,
		)
	}

	slog.Info("settings store ready", "file", file)
	return store, nil
}

func (s *SettingStore) Close() error {
	if s == nil || s.store == nil {
		return nil
	}
	return s.store.Close()
}

func (s *SettingStore) SetDebugSettingsApplier(
	applier DebugSettingsApplier,
) {
	if s == nil {
		return
	}
	s.debugSettingsApplier = applier
}

func (s *SettingStore) ApplyCurrentDebugSettings(
	ctx context.Context,
	forceFetch bool,
) error {
	response, err := s.GetSettings(
		ctx,
		&GetSettingsRequest{
			ForceFetch: forceFetch,
		},
	)
	if err != nil {
		return err
	}
	if response == nil || response.Body == nil {
		return errors.New("get settings: empty response body")
	}
	return s.applyDebugSettings(ctx, response.Body.Debug)
}

// Migrate now normalizes only the non-secret Settings document.
//
// Auth-key migration is intentionally absent. Artifact Store owns Model and
// MCP overlays, secret refs, SHA metadata, and encrypted secret values.
func (s *SettingStore) Migrate(
	ctx context.Context,
) error {
	raw, err := s.store.GetAll(true)
	if err != nil {
		return fmt.Errorf("read settings: %w", err)
	}

	var schema SettingsSchema
	if err := jsonencdec.MapToStructWithJSONTags(raw, &schema); err != nil {
		return fmt.Errorf("decode settings: %w", err)
	}

	normalizedDebug, debugChanged := normalizeDebugSettings(schema.Debug)
	if debugChanged {
		value, err := jsonencdec.StructWithJSONTagsToMap(normalizedDebug)
		if err != nil {
			return fmt.Errorf(
				"encode normalized debug settings: %w",
				err,
			)
		}
		if err := s.store.SetKey(
			[]string{settingKeyDebug},
			value,
		); err != nil {
			return fmt.Errorf(
				"persist normalized debug settings: %w",
				err,
			)
		}
	}

	if schema.SchemaVersion != SchemaVersion {
		if err := s.store.SetKey(
			[]string{settingKeySchemaVersion},
			SchemaVersion,
		); err != nil {
			return fmt.Errorf(
				"persist settings schema version: %w",
				err,
			)
		}
	}

	return nil
}

func (s *SettingStore) SetAppTheme(
	_ context.Context,
	request *SetAppThemeRequest,
) (*SetAppThemeResponse, error) {
	if request == nil || request.Body == nil {
		return nil, ErrInvalidArgument
	}

	theme := AppTheme{
		Type: request.Body.Type,
		Name: request.Body.Name,
	}
	if err := validateTheme(&theme); err != nil {
		return nil, err
	}

	value, err := jsonencdec.StructWithJSONTagsToMap(theme)
	if err != nil {
		return nil, err
	}
	if err := s.store.SetKey(
		[]string{settingKeyAppTheme},
		value,
	); err != nil {
		return nil, err
	}

	slog.Info(
		"app theme updated",
		"type", theme.Type,
		"name", theme.Name,
	)
	return &SetAppThemeResponse{}, nil
}

func (s *SettingStore) SetDebugSettings(
	ctx context.Context,
	request *SetDebugSettingsRequest,
) (*SetDebugSettingsResponse, error) {
	if request == nil || request.Body == nil {
		return nil, ErrInvalidArgument
	}

	settings := DebugSettings{
		LogLLMReqResp:           request.Body.LogLLMReqResp,
		DisableContentStripping: request.Body.DisableContentStripping,
		LogLevel:                request.Body.LogLevel,
	}
	if err := validateDebugSettings(&settings); err != nil {
		return nil, err
	}

	value, err := jsonencdec.StructWithJSONTagsToMap(settings)
	if err != nil {
		return nil, err
	}
	if err := s.store.SetKey(
		[]string{settingKeyDebug},
		value,
	); err != nil {
		return nil, err
	}
	if err := s.applyDebugSettings(ctx, settings); err != nil {
		return nil, fmt.Errorf(
			"debug settings saved but runtime apply failed: %w",
			err,
		)
	}

	slog.Info(
		"debug settings updated",
		"logLLMReqResp", settings.LogLLMReqResp,
		"disableContentStripping", settings.DisableContentStripping,
		"logLevel", settings.LogLevel,
	)
	return &SetDebugSettingsResponse{}, nil
}

func (s *SettingStore) GetSettings(
	_ context.Context,
	request *GetSettingsRequest,
) (*GetSettingsResponse, error) {
	forceFetch := false
	if request != nil {
		forceFetch = request.ForceFetch
	}

	raw, err := s.store.GetAll(forceFetch)
	if err != nil {
		return nil, err
	}

	var schema SettingsSchema
	if err := jsonencdec.MapToStructWithJSONTags(raw, &schema); err != nil {
		return nil, err
	}
	schema.Debug, _ = normalizeDebugSettings(schema.Debug)

	return &GetSettingsResponse{
		Body: &GetSettingsResponseBody{
			AppTheme: schema.AppTheme,
			Debug:    schema.Debug,
		},
	}, nil
}

func (s *SettingStore) applyDebugSettings(
	ctx context.Context,
	settings DebugSettings,
) error {
	if s == nil || s.debugSettingsApplier == nil {
		return nil
	}
	return s.debugSettingsApplier(ctx, settings)
}

// validateTheme checks a theme for correctness.
func validateTheme(th *AppTheme) error {
	if th == nil {
		return ErrInvalidTheme
	}
	switch th.Type {
	case ThemeSystem:
		if th.Name != ThemeNameSystem {
			return fmt.Errorf(
				"%w: type and name required. input - type %s, name %s",
				ErrInvalidTheme,
				th.Type,
				th.Name,
			)
		}
		return nil
	case ThemeLight:
		if th.Name != ThemeNameLight {
			return fmt.Errorf(
				"%w: type and name required. input - type %s, name %s",
				ErrInvalidTheme,
				th.Type,
				th.Name,
			)
		}
		return nil
	case ThemeDark:
		if th.Name != ThemeNameDark {
			return fmt.Errorf(
				"%w: type and name required. input - type %s, name %s",
				ErrInvalidTheme,
				th.Type,
				th.Name,
			)
		}
		return nil
	case ThemeOther:
		if th.Name == "" {
			return fmt.Errorf("%w: name required", ErrInvalidTheme)
		}
		return nil
	default:
		return ErrInvalidTheme
	}
}

func normalizeDebugSettings(cfg DebugSettings) (DebugSettings, bool) {
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
func validateDebugSettings(cfg *DebugSettings) error {
	if cfg == nil {
		return ErrInvalidDebugSettings
	}

	switch cfg.LogLevel {
	case DebugLogLevelDebug,
		DebugLogLevelInfo,
		DebugLogLevelWarn,
		DebugLogLevelError:
		return nil
	default:
		return fmt.Errorf("%w: unsupported logLevel %q", ErrInvalidDebugSettings, cfg.LogLevel)
	}
}
