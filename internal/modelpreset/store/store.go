// Package store implements the provider / model-preset storage layer.
// It offers CRUD operations for both providers and model-presets, integrates
// read-only built-in data, performs structural validation and supports
// paged listing with opaque tokens.
package store

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	inferencegoSpec "github.com/flexigpt/inference-go/spec"
	"github.com/ppipada/mapstore-go"
	"github.com/ppipada/mapstore-go/jsonencdec"
)

// ModelPresetStore is the main storage façade for provider / model-preset data.
type ModelPresetStore struct {
	baseDir string

	// User-modifiable provider / model presets.
	userStore *mapstore.MapFileStore

	// Read-only built-ins with overlay enable/disable flags.
	builtinData *BuiltInPresets

	mu sync.RWMutex // Guards userStore modifications.
}

// NewModelPresetStore initialises the storage in baseDir.
// Built-in data are automatically loaded and overlaid.
func NewModelPresetStore(baseDir string) (*ModelPresetStore, error) {
	s := &ModelPresetStore{baseDir: filepath.Clean(baseDir)}
	ctx := context.Background()
	bi, err := NewBuiltInPresets(ctx, baseDir, spec.BuiltInSnapshotMaxAge)
	if err != nil {
		return nil, err
	}
	s.builtinData = bi
	var defaultProvider inferencegoSpec.ProviderName = ""
	if s.builtinData != nil {
		defaultProvider, err = s.builtinData.GetBuiltInDefaultProviderName(ctx)
		if err != nil {
			return nil, err
		}
	}

	def, err := jsonencdec.StructWithJSONTagsToMap(spec.PresetsSchema{
		SchemaVersion:   spec.SchemaVersion,
		DefaultProvider: defaultProvider,
		ProviderPresets: map[inferencegoSpec.ProviderName]spec.ProviderPreset{},
	})
	if err != nil {
		return nil, err
	}
	s.userStore, err = mapstore.NewMapFileStore(
		filepath.Join(baseDir, spec.ModelPresetsFile),
		def,
		jsonencdec.JSONEncoderDecoder{},
		mapstore.WithCreateIfNotExists(true),
		mapstore.WithFileAutoFlush(true),
		mapstore.WithFileLogger(slog.Default()),
	)
	if err != nil {
		return nil, err
	}

	slog.Info("model-preset store ready", "baseDir", s.baseDir)
	return s, nil
}

func (s *ModelPresetStore) Close() error {
	if s == nil {
		return nil
	}
	if s.builtinData != nil {
		if err := s.builtinData.Close(); err != nil {
			slog.Error("builtinData close failed", "err", err)
		}
		s.builtinData = nil
	}
	if s.userStore != nil {
		if err := s.userStore.Close(); err != nil {
			slog.Error("userStore close failed", "err", err)
		}
		s.userStore = nil
	}
	return nil
}

func (s *ModelPresetStore) GetDefaultProvider(
	ctx context.Context, req *spec.GetDefaultProviderRequest,
) (*spec.GetDefaultProviderResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	all, err := s.readAllUserPresets(false)
	if err != nil {
		return nil, err
	}
	defaultProvider := all.DefaultProvider
	if defaultProvider == "" {
		defaultProvider, err = s.builtinData.GetBuiltInDefaultProviderName(ctx)
		if err != nil {
			return nil, err
		}
	}
	return &spec.GetDefaultProviderResponse{
		Body: &spec.GetDefaultProviderResponseBody{
			DefaultProvider: defaultProvider,
		},
	}, nil
}

func (s *ModelPresetStore) PatchDefaultProvider(
	ctx context.Context, req *spec.PatchDefaultProviderRequest,
) (*spec.PatchDefaultProviderResponse, error) {
	if req == nil || req.Body == nil || req.Body.DefaultProvider == "" {
		return nil, fmt.Errorf("%w: providerName required", spec.ErrProviderNotFound)
	}

	providerName := req.Body.DefaultProvider

	found := false
	if s.builtinData != nil {
		if _, err := s.builtinData.GetBuiltInProvider(ctx, providerName); err == nil {
			found = true
		}
	}
	if !found {
		s.mu.RLock()
		all, err := s.readAllUserPresets(false)
		s.mu.RUnlock()
		if err != nil {
			return nil, err
		}
		if _, ok := all.ProviderPresets[providerName]; ok {
			found = true
		}
	}

	if !found {
		return nil, fmt.Errorf(
			"%w: providerName %q not found",
			spec.ErrProviderNotFound,
			providerName,
		)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	all, err := s.readAllUserPresets(false)
	if err != nil {
		return nil, err
	}
	all.DefaultProvider = providerName
	if err := s.writeAllUserPresets(all); err != nil {
		return nil, err
	}

	slog.Info("patchDefaultProvider", "defaultProvider", providerName)
	return &spec.PatchDefaultProviderResponse{}, nil
}

// PutProviderPreset creates or replaces a provider preset.
func (s *ModelPresetStore) PutProviderPreset(
	ctx context.Context, req *spec.PutProviderPresetRequest,
) (*spec.PutProviderPresetResponse, error) {
	if req == nil || req.Body == nil || req.ProviderName == "" {
		return nil, fmt.Errorf("%w: providerName & body required", spec.ErrInvalidDir)
	}

	// Reject built-ins.
	if _, err := s.builtinData.GetBuiltInProvider(ctx, req.ProviderName); err == nil {
		return nil, fmt.Errorf("%w: providerName: %q",
			spec.ErrBuiltInReadOnly, req.ProviderName)
	}

	now := time.Now().UTC()

	// Build object - keep CreatedAt if provider existed.
	pp := spec.ProviderPreset{
		SchemaVersion:            spec.SchemaVersion,
		Name:                     req.ProviderName,
		DisplayName:              req.Body.DisplayName,
		SDKType:                  req.Body.SDKType,
		IsEnabled:                req.Body.IsEnabled,
		CreatedAt:                now,
		ModifiedAt:               now,
		IsBuiltIn:                false,
		Origin:                   req.Body.Origin,
		ChatCompletionPathPrefix: req.Body.ChatCompletionPathPrefix,
		APIKeyHeaderKey:          req.Body.APIKeyHeaderKey,
		DefaultHeaders:           req.Body.DefaultHeaders,
		ModelPresets:             map[spec.ModelPresetID]spec.ModelPreset{},
	}

	// Validate.
	if err := validateProviderPreset(&pp); err != nil {
		return nil, err
	}

	// Persist.
	s.mu.Lock()
	defer s.mu.Unlock()

	all, err := s.readAllUserPresets(false)
	if err != nil {
		return nil, err
	}
	if existing, ok := all.ProviderPresets[req.ProviderName]; ok {
		pp.CreatedAt = existing.CreatedAt
	}
	pp.ModifiedAt = now
	all.ProviderPresets[req.ProviderName] = pp
	if err := s.writeAllUserPresets(all); err != nil {
		return nil, err
	}
	slog.Info("putProviderPreset", "provider", req.ProviderName)
	return &spec.PutProviderPresetResponse{}, nil
}

// PatchProviderPreset updates a provider preset.
// It can (independently or simultaneously)
//   - enable / disable the provider (body.isEnabled)
//   - change the provider-level default model-preset (body.defaultModelPresetID).
//
// At least one of the two fields must be supplied.
func (s *ModelPresetStore) PatchProviderPreset(
	ctx context.Context, req *spec.PatchProviderPresetRequest,
) (*spec.PatchProviderPresetResponse, error) {
	if req == nil || req.Body == nil || req.ProviderName == "" {
		return nil, fmt.Errorf("%w: providerName required", spec.ErrInvalidDir)
	}
	if req.Body.IsEnabled == nil && req.Body.DefaultModelPresetID == nil {
		return nil, fmt.Errorf("%w: either isEnabled or defaultModelPresetID must be supplied",
			spec.ErrInvalidDir)
	}
	if req.Body.DefaultModelPresetID != nil {
		if err := validateModelPresetID(*req.Body.DefaultModelPresetID); err != nil {
			return nil, err
		}
	}

	if _, err := s.builtinData.GetBuiltInProvider(ctx, req.ProviderName); err == nil {

		// Enable / disable.
		if req.Body.IsEnabled != nil {
			if _, err := s.builtinData.SetProviderEnabled(ctx,
				req.ProviderName, *req.Body.IsEnabled,
			); err != nil {
				return nil, err
			}
			slog.Info("patchProviderPreset.builtin",
				"provider", req.ProviderName,
				"isEnabled", *req.Body.IsEnabled)
		}

		// Change default model-preset (placeholder - to be implemented later).
		if req.Body.DefaultModelPresetID != nil {
			if _, err := s.builtinData.SetDefaultModelPreset(
				ctx, req.ProviderName, *req.Body.DefaultModelPresetID,
			); err != nil {
				return nil, err
			}
			slog.Info("patchProviderPreset.builtin",
				"provider", req.ProviderName,
				"defaultModelPresetID", *req.Body.DefaultModelPresetID)
		}

		return &spec.PatchProviderPresetResponse{}, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	all, err := s.readAllUserPresets(false)
	if err != nil {
		return nil, err
	}

	pp, ok := all.ProviderPresets[req.ProviderName]
	if !ok {
		return nil, fmt.Errorf("%w: %s", spec.ErrProviderNotFound, req.ProviderName)
	}

	changed := false

	// Enable / disable.
	if req.Body.IsEnabled != nil && pp.IsEnabled != *req.Body.IsEnabled {
		pp.IsEnabled = *req.Body.IsEnabled
		changed = true
	}

	// Default model-preset.
	if req.Body.DefaultModelPresetID != nil &&
		pp.DefaultModelPresetID != *req.Body.DefaultModelPresetID {

		if _, ok := pp.ModelPresets[*req.Body.DefaultModelPresetID]; !ok {
			return nil, fmt.Errorf("%w: %s",
				spec.ErrModelPresetNotFound, *req.Body.DefaultModelPresetID)
		}
		pp.DefaultModelPresetID = *req.Body.DefaultModelPresetID
		changed = true
	}

	if !changed {
		// Nothing to do - silently succeed.
		return &spec.PatchProviderPresetResponse{}, nil
	}

	pp.ModifiedAt = time.Now().UTC()
	all.ProviderPresets[req.ProviderName] = pp

	if err := s.writeAllUserPresets(all); err != nil {
		return nil, err
	}

	slog.Info("patchProviderPreset",
		"provider", req.ProviderName,
		"isEnabled", pp.IsEnabled,
		"defaultModelPresetID", pp.DefaultModelPresetID)

	return &spec.PatchProviderPresetResponse{}, nil
}

// DeleteProviderPreset removes a provider if it has no model presets.
func (s *ModelPresetStore) DeleteProviderPreset(
	ctx context.Context, req *spec.DeleteProviderPresetRequest,
) (*spec.DeleteProviderPresetResponse, error) {
	if req == nil || req.ProviderName == "" {
		return nil, fmt.Errorf("%w: providerName required", spec.ErrInvalidDir)
	}
	// Built-ins are read-only.
	if _, err := s.builtinData.GetBuiltInProvider(ctx, req.ProviderName); err == nil {
		return nil, fmt.Errorf("%w: providerName: %q",
			spec.ErrBuiltInReadOnly, req.ProviderName)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	all, err := s.readAllUserPresets(true)
	if err != nil {
		return nil, err
	}
	pp, ok := all.ProviderPresets[req.ProviderName]
	if !ok {
		return nil, fmt.Errorf("%w: %s", spec.ErrProviderNotFound, req.ProviderName)
	}
	if len(pp.ModelPresets) != 0 {
		return nil, fmt.Errorf("provider %q is not empty", req.ProviderName)
	}
	delete(all.ProviderPresets, req.ProviderName)

	if err := s.writeAllUserPresets(all); err != nil {
		return nil, err
	}
	slog.Info("deleteProviderPreset", "provider", req.ProviderName)
	return &spec.DeleteProviderPresetResponse{}, nil
}

// ListProviderPresets lists provider presets with optional filters and paging.
func (s *ModelPresetStore) ListProviderPresets(
	ctx context.Context, req *spec.ListProviderPresetsRequest,
) (*spec.ListProviderPresetsResponse, error) {
	// Resolve parameters - defaults first.
	pageSize := spec.DefaultPageSize
	includeDisabled := false
	want := map[inferencegoSpec.ProviderName]struct{}{}
	cursor := inferencegoSpec.ProviderName("")

	// Token overrides everything.
	if req != nil && req.PageToken != "" {
		if tok, err := jsonutil.Base64JSONDecode[spec.ProviderPageToken](req.PageToken); err == nil {
			pageSize = tok.PageSize
			if pageSize <= 0 || pageSize > spec.MaxPageSize {
				pageSize = spec.DefaultPageSize
			}
			includeDisabled = tok.IncludeDisabled
			cursor = tok.CursorSlug
			for _, n := range tok.Names {
				want[n] = struct{}{}
			}
		}
	} else if req != nil {
		if req.PageSize > 0 && req.PageSize <= spec.DefaultPageSize {
			pageSize = req.PageSize
		}
		includeDisabled = req.IncludeDisabled
		for _, n := range req.Names {
			want[n] = struct{}{}
		}
	}

	// Collect built-ins.
	all := make([]spec.ProviderPreset, 0)
	if s.builtinData != nil {
		bi, _, _ := s.builtinData.ListBuiltInPresets(ctx)
		for _, p := range bi {
			all = append(all, p)
		}
	}
	// Collect user.
	user, err := s.readAllUserPresets(false)
	if err != nil {
		return nil, err
	}
	for _, p := range user.ProviderPresets {
		all = append(all, p)
	}

	// Filtering.
	filtered := make([]spec.ProviderPreset, 0, len(all))
	for _, p := range all {
		if len(want) != 0 {
			if _, ok := want[p.Name]; !ok {
				continue
			}
		}
		if !includeDisabled && !p.IsEnabled {
			continue
		}
		filtered = append(filtered, p)
	}

	// Ordering.
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].ModifiedAt.Equal(filtered[j].ModifiedAt) {
			return filtered[i].Name < filtered[j].Name
		}
		return filtered[i].ModifiedAt.After(filtered[j].ModifiedAt)
	})

	// Cursor.
	start := 0
	if cursor != "" {
		for i, p := range filtered {
			if p.Name == cursor {
				start = i + 1
				break
			}
		}
	}

	end := min(start+pageSize, len(filtered))
	var nextToken *string
	if end < len(filtered) {
		// Preserve filter parameters in token.
		names := make([]inferencegoSpec.ProviderName, 0, len(want))
		for n := range want {
			names = append(names, n)
		}
		slices.Sort(names)

		tok := spec.ProviderPageToken{
			Names:           names,
			IncludeDisabled: includeDisabled,
			PageSize:        pageSize,
			CursorSlug:      filtered[end-1].Name,
		}
		ns := jsonutil.Base64JSONEncode(tok)
		nextToken = &ns
	}

	return &spec.ListProviderPresetsResponse{
		Body: &spec.ListProviderPresetsResponseBody{
			Providers:     filtered[start:end],
			NextPageToken: nextToken,
		},
	}, nil
}

// PutModelPreset creates or replaces a model preset on a user provider.
func (s *ModelPresetStore) PutModelPreset(
	ctx context.Context, req *spec.PutModelPresetRequest,
) (*spec.PutModelPresetResponse, error) {
	if req == nil || req.Body == nil ||
		req.ProviderName == "" || req.ModelPresetID == "" {
		return nil, fmt.Errorf("%w: providerName & modelPresetID required", spec.ErrInvalidDir)
	}
	if err := validateModelPresetID(req.ModelPresetID); err != nil {
		return nil, err
	}
	if err := validateModelSlug(req.Body.Slug); err != nil {
		return nil, err
	}
	// Reject built-ins.
	if _, err := s.builtinData.GetBuiltInProvider(ctx, req.ProviderName); err == nil {
		return nil, fmt.Errorf("%w: providerName: %q",
			spec.ErrBuiltInReadOnly, req.ProviderName)
	}

	now := time.Now().UTC()

	// Build model preset.
	mp := spec.ModelPreset{
		SchemaVersion:               spec.SchemaVersion,
		ID:                          req.ModelPresetID,
		Name:                        req.Body.Name,
		DisplayName:                 req.Body.DisplayName,
		Slug:                        req.Body.Slug,
		IsEnabled:                   req.Body.IsEnabled,
		Stream:                      req.Body.Stream,
		MaxPromptLength:             req.Body.MaxPromptLength,
		MaxOutputLength:             req.Body.MaxOutputLength,
		Temperature:                 req.Body.Temperature,
		Reasoning:                   req.Body.Reasoning,
		SystemPrompt:                req.Body.SystemPrompt,
		Timeout:                     req.Body.Timeout,
		AdditionalParametersRawJSON: req.Body.AdditionalParametersRawJSON,
		CreatedAt:                   now,
		ModifiedAt:                  now,
		IsBuiltIn:                   false,
	}
	if err := validateModelPreset(&mp); err != nil {
		return nil, err
	}

	// Persist.
	s.mu.Lock()
	defer s.mu.Unlock()

	all, err := s.readAllUserPresets(false)
	if err != nil {
		return nil, err
	}
	pp, ok := all.ProviderPresets[req.ProviderName]
	if !ok {
		return nil, fmt.Errorf("%w: %s", spec.ErrProviderNotFound, req.ProviderName)
	}
	// Keep createdAt if overwriting.
	if old, ok := pp.ModelPresets[req.ModelPresetID]; ok {
		mp.CreatedAt = old.CreatedAt
	}
	if pp.ModelPresets == nil {
		pp.ModelPresets = map[spec.ModelPresetID]spec.ModelPreset{}
	}
	mp.ModifiedAt = now
	pp.ModelPresets[req.ModelPresetID] = mp
	pp.ModifiedAt = now
	all.ProviderPresets[req.ProviderName] = pp

	if err := s.writeAllUserPresets(all); err != nil {
		return nil, err
	}
	slog.Info("putModelPreset",
		"provider", req.ProviderName, "modelPresetID", req.ModelPresetID)
	return &spec.PutModelPresetResponse{}, nil
}

// PatchModelPreset enables or disables a model preset.
func (s *ModelPresetStore) PatchModelPreset(
	ctx context.Context, req *spec.PatchModelPresetRequest,
) (*spec.PatchModelPresetResponse, error) {
	if req == nil || req.Body == nil ||
		req.ProviderName == "" || req.ModelPresetID == "" {
		return nil, fmt.Errorf("%w: providerName & modelPresetID required", spec.ErrInvalidDir)
	}

	// Built-in branch.
	if _, err := s.builtinData.GetBuiltInProvider(ctx, req.ProviderName); err == nil {
		if _, err := s.builtinData.SetModelPresetEnabled(
			ctx,
			req.ProviderName, req.ModelPresetID, req.Body.IsEnabled,
		); err != nil {
			return nil, err
		}
		slog.Info("patchModelPreset.builtin",
			"provider", req.ProviderName, "modelPresetID", req.ModelPresetID,
			"enabled", req.Body.IsEnabled)
		return &spec.PatchModelPresetResponse{}, nil
	}

	// User branch.
	s.mu.Lock()
	defer s.mu.Unlock()

	all, err := s.readAllUserPresets(false)
	if err != nil {
		return nil, err
	}
	pp, ok := all.ProviderPresets[req.ProviderName]
	if !ok {
		return nil, fmt.Errorf("%w: %s", spec.ErrProviderNotFound, req.ProviderName)
	}
	mp, ok := pp.ModelPresets[req.ModelPresetID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", spec.ErrModelPresetNotFound, req.ModelPresetID)
	}
	mp.IsEnabled = req.Body.IsEnabled
	mp.ModifiedAt = time.Now().UTC()
	pp.ModelPresets[req.ModelPresetID] = mp
	pp.ModifiedAt = mp.ModifiedAt
	all.ProviderPresets[req.ProviderName] = pp

	if err := s.writeAllUserPresets(all); err != nil {
		return nil, err
	}
	slog.Info("patchModelPreset",
		"provider", req.ProviderName, "modelPresetID", req.ModelPresetID,
		"enabled", req.Body.IsEnabled)
	return &spec.PatchModelPresetResponse{}, nil
}

// DeleteModelPreset removes a model preset.
func (s *ModelPresetStore) DeleteModelPreset(
	ctx context.Context, req *spec.DeleteModelPresetRequest,
) (*spec.DeleteModelPresetResponse, error) {
	if req == nil || req.ProviderName == "" || req.ModelPresetID == "" {
		return nil, fmt.Errorf("%w: providerName & modelPresetID required", spec.ErrInvalidDir)
	}
	// Built-in are read-only.
	if _, err := s.builtinData.GetBuiltInProvider(ctx, req.ProviderName); err == nil {
		return nil, fmt.Errorf("%w: providerName: %q",
			spec.ErrBuiltInReadOnly, req.ProviderName)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	all, err := s.readAllUserPresets(false)
	if err != nil {
		return nil, err
	}
	pp, ok := all.ProviderPresets[req.ProviderName]
	if !ok {
		return nil, fmt.Errorf("%w: %s", spec.ErrProviderNotFound, req.ProviderName)
	}
	if _, ok := pp.ModelPresets[req.ModelPresetID]; !ok {
		return nil, fmt.Errorf("%w: %s", spec.ErrModelPresetNotFound, req.ModelPresetID)
	}
	delete(pp.ModelPresets, req.ModelPresetID)
	// Reset default if it pointed to the deleted model.
	if pp.DefaultModelPresetID == req.ModelPresetID {
		pp.DefaultModelPresetID = ""
	}
	pp.ModifiedAt = time.Now().UTC()
	all.ProviderPresets[req.ProviderName] = pp

	if err := s.writeAllUserPresets(all); err != nil {
		return nil, err
	}
	slog.Info("deleteModelPreset",
		"provider", req.ProviderName, "modelPresetID", req.ModelPresetID)
	return &spec.DeleteModelPresetResponse{}, nil
}

func (s *ModelPresetStore) readAllUserPresets(force bool) (spec.PresetsSchema, error) {
	raw, err := s.userStore.GetAll(force)
	if err != nil {
		return spec.PresetsSchema{}, err
	}
	var ps spec.PresetsSchema
	if err := jsonencdec.MapToStructWithJSONTags(raw, &ps); err != nil {
		return ps, err
	}
	return ps, nil
}

func (s *ModelPresetStore) writeAllUserPresets(ps spec.PresetsSchema) error {
	mp, _ := jsonencdec.StructWithJSONTagsToMap(ps)
	return s.userStore.SetAll(mp)
}
