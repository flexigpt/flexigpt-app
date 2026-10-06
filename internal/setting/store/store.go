package store

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"

	settingSpec "github.com/flexigpt/flexigpt-app/internal/setting/spec"
	"github.com/flexigpt/mapstore-go"
	"github.com/flexigpt/mapstore-go/jsonencdec"
)

type DebugSettingsApplier func(context.Context, settingSpec.DebugSettings) error

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

	file := filepath.Join(baseDir, settingSpec.SettingsFile)
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
		&settingSpec.GetSettingsRequest{
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

	var schema settingSpec.SettingsSchema
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

	if schema.SchemaVersion != settingSpec.SchemaVersion {
		if err := s.store.SetKey(
			[]string{settingKeySchemaVersion},
			settingSpec.SchemaVersion,
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
	request *settingSpec.SetAppThemeRequest,
) (*settingSpec.SetAppThemeResponse, error) {
	if s == nil || s.store == nil {
		return nil, basespecClosedError()
	}
	if request == nil || request.Body == nil {
		return nil, settingSpec.ErrInvalidArgument
	}

	theme := settingSpec.AppTheme{
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
	return &settingSpec.SetAppThemeResponse{}, nil
}

func (s *SettingStore) SetDebugSettings(
	ctx context.Context,
	request *settingSpec.SetDebugSettingsRequest,
) (*settingSpec.SetDebugSettingsResponse, error) {
	if s == nil || s.store == nil {
		return nil, basespecClosedError()
	}
	if request == nil || request.Body == nil {
		return nil, settingSpec.ErrInvalidArgument
	}

	settings := settingSpec.DebugSettings{
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
	return &settingSpec.SetDebugSettingsResponse{}, nil
}

func (s *SettingStore) GetSettings(
	_ context.Context,
	request *settingSpec.GetSettingsRequest,
) (*settingSpec.GetSettingsResponse, error) {
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

	var schema settingSpec.SettingsSchema
	if err := jsonencdec.MapToStructWithJSONTags(raw, &schema); err != nil {
		return nil, err
	}
	schema.Debug, _ = normalizeDebugSettings(schema.Debug)

	return &settingSpec.GetSettingsResponse{
		Body: &settingSpec.GetSettingsResponseBody{
			AppTheme: schema.AppTheme,
			Debug:    schema.Debug,
		},
	}, nil
}

func (s *SettingStore) applyDebugSettings(
	ctx context.Context,
	settings settingSpec.DebugSettings,
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
