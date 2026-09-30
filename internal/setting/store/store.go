package store

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/flexigpt/flexigpt-app/internal/setting/spec"
	"github.com/flexigpt/mapstore-go"
	"github.com/flexigpt/mapstore-go/jsonencdec"
)

type DebugSettingsApplier func(context.Context, spec.DebugSettings) error

type SettingStore struct {
	store                *mapstore.MapFileStore
	debugSettingsApplier DebugSettingsApplier
}

func ErrClosed() error {
	return basespecClosedError()
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

	file := filepath.Join(baseDir, spec.SettingsFile)
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
	if s == nil {
		return basespecClosedError()
	}

	response, err := s.GetSettings(
		ctx,
		&spec.GetSettingsRequest{
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
	if s == nil || s.store == nil {
		return basespecClosedError()
	}
	if ctx == nil {
		return errors.New("settings migration context is nil")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	raw, err := s.store.GetAll(true)
	if err != nil {
		return fmt.Errorf("read settings: %w", err)
	}

	var schema spec.SettingsSchema
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

	if schema.SchemaVersion != spec.SchemaVersion {
		if err := s.store.SetKey(
			[]string{settingKeySchemaVersion},
			spec.SchemaVersion,
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
	request *spec.SetAppThemeRequest,
) (*spec.SetAppThemeResponse, error) {
	if s == nil || s.store == nil {
		return nil, basespecClosedError()
	}
	if request == nil || request.Body == nil {
		return nil, spec.ErrInvalidArgument
	}

	theme := spec.AppTheme{
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
	return &spec.SetAppThemeResponse{}, nil
}

func (s *SettingStore) SetDebugSettings(
	ctx context.Context,
	request *spec.SetDebugSettingsRequest,
) (*spec.SetDebugSettingsResponse, error) {
	if s == nil || s.store == nil {
		return nil, basespecClosedError()
	}
	if request == nil || request.Body == nil {
		return nil, spec.ErrInvalidArgument
	}

	settings := spec.DebugSettings{
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
	return &spec.SetDebugSettingsResponse{}, nil
}

func (s *SettingStore) GetSettings(
	_ context.Context,
	request *spec.GetSettingsRequest,
) (*spec.GetSettingsResponse, error) {
	if s == nil || s.store == nil {
		return nil, basespecClosedError()
	}

	forceFetch := false
	if request != nil {
		forceFetch = request.ForceFetch
	}

	raw, err := s.store.GetAll(forceFetch)
	if err != nil {
		return nil, err
	}

	var schema spec.SettingsSchema
	if err := jsonencdec.MapToStructWithJSONTags(raw, &schema); err != nil {
		return nil, err
	}
	schema.Debug, _ = normalizeDebugSettings(schema.Debug)

	return &spec.GetSettingsResponse{
		Body: &spec.GetSettingsResponseBody{
			AppTheme: schema.AppTheme,
			Debug:    schema.Debug,
		},
	}, nil
}

func (s *SettingStore) applyDebugSettings(
	ctx context.Context,
	settings spec.DebugSettings,
) error {
	if s == nil || s.debugSettingsApplier == nil {
		return nil
	}
	return s.debugSettingsApplier(ctx, settings)
}

// basespecClosedError deliberately keeps Setting Store independent from
// Artifact Store packages.
func basespecClosedError() error {
	return errors.New("settings store: closed")
}
