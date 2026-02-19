package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/setting/spec"
	"github.com/ppipada/mapstore-go"
	"github.com/ppipada/mapstore-go/jsonencdec"
	"github.com/ppipada/mapstore-go/keyringencdec"
)

type SettingStore struct {
	store      *mapstore.MapFileStore
	encEncrypt mapstore.IOEncoderDecoder
}

const (
	keyringServiceName = "FlexiGPTKeyRingEncDec"
	keyringUserName    = "user"
)

func NewSettingStore(baseDir string) (*SettingStore, error) {
	encoderDecoder, err := keyringencdec.NewEncryptedStringValueEncoderDecoder(keyringServiceName, keyringUserName)
	if err != nil {
		return nil, fmt.Errorf("could not get keyring encoder/decoder: %w", err)
	}
	st := &SettingStore{
		encEncrypt: encoderDecoder,
	}

	defaultMap, err := jsonencdec.StructWithJSONTagsToMap(DefaultSettingsData)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal default settings: %w", err)
	}

	file := filepath.Join(baseDir, spec.SettingsFile)
	fs, err := mapstore.NewMapFileStore(
		file,
		defaultMap,
		jsonencdec.JSONEncoderDecoder{},
		mapstore.WithCreateIfNotExists(true),
		mapstore.WithFileAutoFlush(true),
		mapstore.WithValueEncDecGetter(st.valueEncDecGetter),
		mapstore.WithFileLogger(slog.Default()),
	)
	if err != nil {
		return nil, fmt.Errorf("file store init failed: %w", err)
	}

	st.store = fs
	if err := st.Migrate(context.Background()); err != nil {
		return nil, fmt.Errorf("settings migration failed: %w", err)
	}
	slog.Info("settings store ready", "file", file)
	return st, nil
}

func (s *SettingStore) Close() error {
	if s == nil {
		return nil
	}

	if s.store != nil {
		_ = s.store.Close()
	}
	return nil
}

// Migrate ensures the store is up-to-date with built-in data.
// - Adds missing built-in auth keys as empty entries.
// - Updates schemaVersion if changed.
func (s *SettingStore) Migrate(ctx context.Context) error {
	// Force re-read from disk to be safe during startup.
	raw, err := s.store.GetAll(true)
	if err != nil {
		return fmt.Errorf("migrate: read store: %w", err)
	}

	var schema spec.SettingsSchema
	if err := jsonencdec.MapToStructWithJSONTags(raw, &schema); err != nil {
		return fmt.Errorf("migrate: decode: %w", err)
	}

	added := 0

	// Ensure the authKeys map and the required nested maps exist,
	// by using SetKey to create them on demand, but check via the decoded schema.
	for t, names := range BuiltInAuthKeys {
		for _, name := range names {
			exists := false
			if schema.AuthKeys != nil {
				if m, ok := schema.AuthKeys[t]; ok {
					if _, ok := m[name]; ok {
						exists = true
					}
				}
			}
			if exists {
				continue
			}

			// Create an empty key: encrypted secret "", sha of "", nonEmpty false.
			secret := ""
			sha := computeSHA(secret)

			if err := s.store.SetKey([]string{
				"authKeys", string(t), string(name), "secret",
			}, secret); err != nil {
				return fmt.Errorf("settings migrate: set secret for %s/%s: %w", t, name, err)
			}

			if err := s.store.SetKey([]string{
				"authKeys", string(t), string(name), "sha256",
			}, sha); err != nil {
				return fmt.Errorf("settings migrate: set sha256 for %s/%s: %w", t, name, err)
			}

			if err := s.store.SetKey([]string{
				"authKeys", string(t), string(name), "nonEmpty",
			}, false); err != nil {
				return fmt.Errorf("settings migrate: set nonEmpty for %s/%s: %w", t, name, err)
			}

			added++
		}
	}

	// Optionally bump schemaVersion if changed or missing.
	if schema.SchemaVersion != spec.SchemaVersion {
		if err := s.store.SetKey([]string{"schemaVersion"}, spec.SchemaVersion); err != nil {
			return fmt.Errorf("migrate: update schemaVersion: %w", err)
		}
	}

	if added > 0 {
		slog.Info("settings migration complete: added missing built-in auth keys", "added", added)
	} else {
		slog.Info("settings migration: no changes needed")
	}

	return nil
}

// SetAppTheme validates and persists a new theme.
func (s *SettingStore) SetAppTheme(
	_ context.Context,
	req *spec.SetAppThemeRequest,
) (*spec.SetAppThemeResponse, error) {
	if req == nil || req.Body == nil {
		return nil, spec.ErrInvalidArgument
	}

	theme := &spec.AppTheme{Type: req.Body.Type, Name: req.Body.Name}
	if err := validateTheme(theme); err != nil {
		return nil, err
	}

	val, _ := jsonencdec.StructWithJSONTagsToMap(theme)
	if err := s.store.SetKey([]string{"appTheme"}, val); err != nil {
		return nil, err
	}

	slog.Info("appTheme updated", "type", theme.Type, "name", theme.Name)
	return &spec.SetAppThemeResponse{}, nil
}

// SetAuthKey inserts or updates one auth-key.
func (s *SettingStore) SetAuthKey(
	_ context.Context,
	req *spec.SetAuthKeyRequest,
) (*spec.SetAuthKeyResponse, error) {
	if req == nil || req.Body == nil || req.Type == "" || req.KeyName == "" {
		return nil, spec.ErrInvalidArgument
	}

	nonEmptySecret := req.Body.Secret != ""

	// Build AuthKey record.
	newAk := spec.AuthKey{
		Secret:   req.Body.Secret,
		SHA256:   computeSHA(req.Body.Secret),
		NonEmpty: nonEmptySecret,
	}

	// Persist secret (encrypted) then sha (plain).
	secretPath := []string{"authKeys", string(req.Type), string(req.KeyName), "secret"}
	if err := s.store.SetKey(secretPath, newAk.Secret); err != nil {
		return nil, err
	}
	shaPath := []string{"authKeys", string(req.Type), string(req.KeyName), "sha256"}
	if err := s.store.SetKey(shaPath, newAk.SHA256); err != nil {
		return nil, err
	}
	nonEmptyPath := []string{
		"authKeys",
		string(req.Type),
		string(req.KeyName),
		"nonEmpty",
	}
	if err := s.store.SetKey(nonEmptyPath, newAk.NonEmpty); err != nil {
		return nil, err
	}

	slog.Info("authKey set",
		"type", req.Type, "keyName", req.KeyName,
		"builtIn", isBuiltInKey(req.Type, req.KeyName))
	return &spec.SetAuthKeyResponse{}, nil
}

// DeleteAuthKey removes a key unless it is marked built-in.
func (s *SettingStore) DeleteAuthKey(
	_ context.Context,
	req *spec.DeleteAuthKeyRequest,
) (*spec.DeleteAuthKeyResponse, error) {
	if req == nil || req.Type == "" || req.KeyName == "" {
		return nil, spec.ErrInvalidArgument
	}
	if isBuiltInKey(req.Type, req.KeyName) {
		return nil, spec.ErrBuiltInAuthKeyReadOnly
	}

	// Delete the key map entirely (secret + sha).
	keyPath := []string{"authKeys", string(req.Type), string(req.KeyName)}
	if err := s.store.DeleteKey(keyPath); err != nil {
		return nil, err
	}

	// If the type map is now empty, delete it too.
	raw, err := s.store.GetAll(false)
	if err != nil {
		return nil, err
	}

	if akRaw, ok := raw["authKeys"].(map[string]any); ok {
		if typRaw, ok := akRaw[string(req.Type)].(map[string]any); ok && len(typRaw) == 0 {
			_ = s.store.DeleteKey([]string{"authKeys", string(req.Type)})
		}
	}

	slog.Info("authKey deleted", "type", req.Type, "keyName", req.KeyName)
	return &spec.DeleteAuthKeyResponse{}, nil
}

// GetAuthKey returns the decrypted secret for one key.
func (s *SettingStore) GetAuthKey(
	_ context.Context,
	req *spec.GetAuthKeyRequest,
) (*spec.GetAuthKeyResponse, error) {
	if req == nil || req.Type == "" || req.KeyName == "" {
		return nil, spec.ErrInvalidArgument
	}

	raw, err := s.store.GetAll(false)
	if err != nil {
		return nil, err
	}

	var schema spec.SettingsSchema
	if err := jsonencdec.MapToStructWithJSONTags(raw, &schema); err != nil {
		return nil, err
	}

	typData, ok := schema.AuthKeys[req.Type]
	if !ok {
		return nil, spec.ErrAuthKeyNotFound
	}
	ak, ok := typData[req.KeyName]
	if !ok {
		return nil, spec.ErrAuthKeyNotFound
	}

	return &spec.GetAuthKeyResponse{
		Body: &spec.GetAuthKeyResponseBody{
			Secret:   ak.Secret,
			SHA256:   ak.SHA256,
			NonEmpty: ak.NonEmpty,
		},
	}, nil
}

// GetSettings returns the current settings without secrets.
func (s *SettingStore) GetSettings(
	_ context.Context,
	req *spec.GetSettingsRequest,
) (*spec.GetSettingsResponse, error) {
	force := false
	if req != nil {
		force = req.ForceFetch
	}

	raw, err := s.store.GetAll(force)
	if err != nil {
		return nil, err
	}

	var schema spec.SettingsSchema
	if err := jsonencdec.MapToStructWithJSONTags(raw, &schema); err != nil {
		return nil, err
	}

	// Convert to DTO (secrets stripped).
	out := spec.GetSettingsResponse{
		Body: &spec.GetSettingsResponseBody{
			AppTheme: schema.AppTheme,
			AuthKeys: []spec.AuthKeyMeta{},
		},
	}
	for t, m := range schema.AuthKeys {
		for n, ak := range m {
			out.Body.AuthKeys = append(out.Body.AuthKeys, spec.AuthKeyMeta{
				Type:     t,
				KeyName:  n,
				SHA256:   ak.SHA256,
				NonEmpty: ak.NonEmpty,
			})
		}
	}

	// Sort by Type, then by KeyName.
	sort.Slice(out.Body.AuthKeys, func(i, j int) bool {
		if out.Body.AuthKeys[i].Type == out.Body.AuthKeys[j].Type {
			return out.Body.AuthKeys[i].KeyName < out.Body.AuthKeys[j].KeyName
		}
		return out.Body.AuthKeys[i].Type < out.Body.AuthKeys[j].Type
	})

	return &out, nil
}

// valueEncDecGetter returns the encoder/decoder to encrypt secrets.
func (s *SettingStore) valueEncDecGetter(path []string) mapstore.IOEncoderDecoder {
	// AuthKeys / <type> / <keyName> / secret.
	if len(path) == 4 && path[0] == "authKeys" && path[3] == "secret" {
		return s.encEncrypt
	}
	return nil
}

// computeSHA returns the hex SHA-256 of the given string.
func computeSHA(in string) string {
	sum := sha256.Sum256([]byte(in))
	return hex.EncodeToString(sum[:])
}
