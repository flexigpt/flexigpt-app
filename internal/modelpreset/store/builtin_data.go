package store

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/flexigpt/inference-go/capabilityoverride"
	"github.com/flexigpt/inference-go/modelpreset"
	inferenceSpec "github.com/flexigpt/inference-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/builtin"
	"github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	"github.com/flexigpt/flexigpt-app/internal/overlay"
)

type builtInProviderKey inferenceSpec.ProviderName

func (builtInProviderKey) Group() overlay.GroupID { return "providers" }
func (k builtInProviderKey) ID() overlay.KeyID    { return overlay.KeyID(k) }

type builtInModelKey modelpreset.ModelPresetID

func (builtInModelKey) Group() overlay.GroupID { return "models" }
func (k builtInModelKey) ID() overlay.KeyID    { return overlay.KeyID(k) }

type builtInProviderDefaultModelIDKey inferenceSpec.ProviderName

func (builtInProviderDefaultModelIDKey) Group() overlay.GroupID { return "providerDefaultModelIDs" }
func (k builtInProviderDefaultModelIDKey) ID() overlay.KeyID    { return overlay.KeyID(k) }

// BuiltInPresets loads built-in preset assets and maintains an overlay store.
type BuiltInPresets struct {
	// Immutable original data.
	defaultProvider inferenceSpec.ProviderName
	providers       map[inferenceSpec.ProviderName]spec.ProviderPreset
	models          map[inferenceSpec.ProviderName]map[modelpreset.ModelPresetID]spec.ModelPreset

	// View after overlay application, guarded by mu.
	mu         sync.RWMutex
	viewProv   map[inferenceSpec.ProviderName]spec.ProviderPreset
	viewModels map[inferenceSpec.ProviderName]map[modelpreset.ModelPresetID]spec.ModelPreset

	// IO.
	overlayBaseDir string

	store                              *overlay.Store
	providerOverlayFlags               *overlay.TypedGroup[builtInProviderKey, bool]
	modelOverlayFlags                  *overlay.TypedGroup[builtInModelKey, bool]
	providerDefaultModelIDOverlayFlags *overlay.TypedGroup[builtInProviderDefaultModelIDKey, modelpreset.ModelPresetID]

	rebuilder *builtin.AsyncRebuilder
}

type PresetStoreOption func(*BuiltInPresets)

// NewBuiltInPresets prepares the presets store and loads a first snapshot.
func NewBuiltInPresets(
	ctx context.Context,
	overlayBaseDir string,
	maxSnapshotAge time.Duration,
	opts ...PresetStoreOption,
) (bi *BuiltInPresets, err error) {
	if overlayBaseDir == "" {
		return nil, fmt.Errorf("%w: overlayBaseDir", spec.ErrInvalidDir)
	}
	if maxSnapshotAge <= 0 {
		maxSnapshotAge = time.Hour
	}
	if err := os.MkdirAll(overlayBaseDir, 0o755); err != nil {
		return nil, err
	}

	store, err := overlay.NewOverlayStore(ctx,
		filepath.Join(overlayBaseDir, spec.ModelPresetsBuiltInOverlayDBFileName),
		overlay.WithKeyType[builtInProviderKey](),
		overlay.WithKeyType[builtInModelKey](),
		overlay.WithKeyType[builtInProviderDefaultModelIDKey](),
	)
	if err != nil {
		return nil, err
	}

	bi = &BuiltInPresets{
		overlayBaseDir: overlayBaseDir,
		store:          store,
	}
	defer func() {
		if err != nil && bi != nil {
			_ = bi.Close()
			bi = nil
		}
	}()

	providerOverlayFlags, err := overlay.NewTypedGroup[builtInProviderKey, bool](ctx, store)
	if err != nil {
		return nil, err
	}
	modelOverlayFlags, err := overlay.NewTypedGroup[builtInModelKey, bool](ctx, store)
	if err != nil {
		return nil, err
	}

	providerDefaultModelIDOverlayFlags, err := overlay.NewTypedGroup[
		builtInProviderDefaultModelIDKey, modelpreset.ModelPresetID](ctx, store)
	if err != nil {
		return nil, err
	}

	bi.providerOverlayFlags = providerOverlayFlags
	bi.modelOverlayFlags = modelOverlayFlags
	bi.providerDefaultModelIDOverlayFlags = providerDefaultModelIDOverlayFlags

	for _, o := range opts {
		o(bi)
	}
	if err := bi.populateDataFromInferenceCatalog(ctx); err != nil {
		return nil, err
	}

	bi.rebuilder = builtin.NewAsyncRebuilder(
		maxSnapshotAge,
		func() error { //nolint:contextcheck // Cannot pass app context to async builder.
			bi.mu.Lock()
			defer bi.mu.Unlock()
			return bi.rebuildSnapshot(context.Background())
		},
	)
	bi.rebuilder.MarkFresh()
	return bi, nil
}

func (b *BuiltInPresets) Close() error {
	if b == nil {
		return nil
	}

	// Stop async rebuilder if it exposes a Stop or Close method.
	if b.rebuilder != nil {
		b.rebuilder.Close()
	}

	// Close overlay store if it supports Close().
	if b.store != nil {
		_ = b.store.Close()
	}
	return nil
}

// ListBuiltInPresets returns deep-copied snapshots.
func (b *BuiltInPresets) ListBuiltInPresets(ctx context.Context) (
	providerPresets map[inferenceSpec.ProviderName]spec.ProviderPreset,
	modelPresets map[inferenceSpec.ProviderName]map[modelpreset.ModelPresetID]spec.ModelPreset,
	err error,
) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return cloneProviderPresetMap(b.viewProv), cloneModelPresetNestedMap(b.viewModels), nil
}

// GetBuiltInDefaultProviderName fetches the default provider name in builtin.
func (b *BuiltInPresets) GetBuiltInDefaultProviderName(
	ctx context.Context,
) (inferenceSpec.ProviderName, error) {
	defaultProvider := b.defaultProvider

	if defaultProvider == "" {
		defaultProvider = modelpreset.ProviderOpenAIResponses
	}
	return defaultProvider, nil
}

// GetBuiltInProvider fetches a provider from the snapshot.
func (b *BuiltInPresets) GetBuiltInProvider(
	ctx context.Context,
	name inferenceSpec.ProviderName,
) (spec.ProviderPreset, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	p, ok := b.viewProv[name]
	if !ok {
		return spec.ProviderPreset{}, spec.ErrProviderNotFound
	}
	return cloneProviderPreset(p), nil
}

// SetProviderEnabled toggles a provider.
func (b *BuiltInPresets) SetProviderEnabled(
	ctx context.Context,
	name inferenceSpec.ProviderName,
	enabled bool,
) (spec.ProviderPreset, error) {
	if _, ok := b.providers[name]; !ok {
		return spec.ProviderPreset{}, spec.ErrBuiltInProviderAbsent
	}
	flag, err := b.providerOverlayFlags.SetFlag(ctx, builtInProviderKey(name), enabled)
	if err != nil {
		return spec.ProviderPreset{}, err
	}

	b.mu.Lock()
	pp := b.viewProv[name]
	pp.IsEnabled = enabled
	pp.ModifiedAt = flag.ModifiedAt
	b.viewProv[name] = pp
	cloned := cloneProviderPreset(pp)
	b.mu.Unlock()

	b.rebuilder.Trigger()
	return cloned, nil
}

// SetModelPresetEnabled toggles a model preset.
func (b *BuiltInPresets) SetModelPresetEnabled(
	ctx context.Context,
	provider inferenceSpec.ProviderName,
	modelID modelpreset.ModelPresetID,
	enabled bool,
) (spec.ModelPreset, error) {
	mp, err := b.GetBuiltInModelPreset(ctx, provider, modelID)
	if err != nil {
		return mp, err
	}
	flag, err := b.modelOverlayFlags.SetFlag(ctx, getModelKey(provider, modelID), enabled)
	if err != nil {
		return spec.ModelPreset{}, err
	}

	b.mu.Lock()
	mp.IsEnabled = enabled
	mp.ModifiedAt = flag.ModifiedAt
	b.viewModels[provider][modelID] = mp

	// Keep provider snapshot consistent for immediate reads.
	pp := b.viewProv[provider]
	if pp.ModelPresets == nil {
		pp.ModelPresets = map[modelpreset.ModelPresetID]spec.ModelPreset{}
	}
	pp.ModelPresets[modelID] = mp
	b.viewProv[provider] = pp
	b.mu.Unlock()

	b.rebuilder.Trigger()
	return cloneModelPreset(mp), nil
}

// GetBuiltInModelPreset fetches a model preset.
func (b *BuiltInPresets) GetBuiltInModelPreset(
	ctx context.Context,
	provider inferenceSpec.ProviderName,
	modelID modelpreset.ModelPresetID,
) (spec.ModelPreset, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	pm, ok := b.viewModels[provider]
	if !ok {
		return spec.ModelPreset{}, spec.ErrProviderNotFound
	}
	mp, ok := pm[modelID]
	if !ok {
		return spec.ModelPreset{}, spec.ErrModelPresetNotFound
	}
	return cloneModelPreset(mp), nil
}

func (b *BuiltInPresets) SetDefaultModelPreset(
	ctx context.Context,
	provider inferenceSpec.ProviderName,
	modelID modelpreset.ModelPresetID,
) (spec.ProviderPreset, error) {
	// Validate provider existence.
	pm, ok := b.models[provider]
	if !ok {
		return spec.ProviderPreset{}, spec.ErrProviderNotFound
	}
	// Validate model existence.
	if _, ok := pm[modelID]; !ok {
		return spec.ProviderPreset{}, spec.ErrModelPresetNotFound
	}

	// Persist in overlay.
	flag, err := b.providerDefaultModelIDOverlayFlags.SetFlag(
		ctx, builtInProviderDefaultModelIDKey(provider), modelID,
	)
	if err != nil {
		return spec.ProviderPreset{}, err
	}

	// Update hot snapshot.
	b.mu.Lock()
	pp := b.viewProv[provider]
	pp.DefaultModelPresetID = modelID
	pp.ModifiedAt = flag.ModifiedAt
	b.viewProv[provider] = pp
	cloned := cloneProviderPreset(pp)
	b.mu.Unlock()

	b.rebuilder.Trigger()
	return cloned, nil
}

// rebuildSnapshot applies overlay flags onto the immutable base sets.
// Caller must hold write lock.
func (b *BuiltInPresets) rebuildSnapshot(ctx context.Context) error {
	newProv := make(map[inferenceSpec.ProviderName]spec.ProviderPreset, len(b.providers))
	newModels := make(map[inferenceSpec.ProviderName]map[modelpreset.ModelPresetID]spec.ModelPreset, len(b.models))

	for pname, mm := range b.models {
		sub := make(map[modelpreset.ModelPresetID]spec.ModelPreset, len(mm))
		for mid, m := range mm {
			if flag, ok, err := b.modelOverlayFlags.GetFlag(ctx, getModelKey(pname, mid)); err != nil {
				return err
			} else if ok {
				m.IsEnabled = flag.Value
				m.ModifiedAt = flag.ModifiedAt
			}
			sub[mid] = m
		}
		newModels[pname] = sub
	}

	for pname, p := range b.providers {
		if flag, ok, err := b.providerDefaultModelIDOverlayFlags.GetFlag(
			ctx, builtInProviderDefaultModelIDKey(pname),
		); err != nil {
			return err
		} else if ok {
			p.DefaultModelPresetID = flag.Value
			p.ModifiedAt = flag.ModifiedAt
		}

		if flag, ok, err := b.providerOverlayFlags.GetFlag(ctx, builtInProviderKey(pname)); err != nil {
			return err
		} else if ok {
			p.IsEnabled = flag.Value
			if flag.ModifiedAt.After(p.ModifiedAt) {
				p.ModifiedAt = flag.ModifiedAt
			}
		}
		// Need to apply the overlayed model presets.
		p.ModelPresets = newModels[pname]

		newProv[pname] = p
	}

	b.viewProv = newProv
	b.viewModels = newModels
	return nil
}

func (b *BuiltInPresets) populateDataFromInferenceCatalog(ctx context.Context) error {
	catalog := modelpreset.DefaultCatalog()
	if len(catalog.Providers) == 0 {
		return errors.New("inference model preset catalog contains no providers")
	}
	if err := modelpreset.ValidateCatalog(catalog); err != nil {
		return fmt.Errorf("invalid inference model preset catalog: %w", err)
	}
	if err := validateBuiltInOverlayCoverage(catalog); err != nil {
		return fmt.Errorf("invalid built-in model preset overlay configuration: %w", err)
	}

	providers := make(map[inferenceSpec.ProviderName]spec.ProviderPreset, len(catalog.Providers))
	models := make(
		map[inferenceSpec.ProviderName]map[modelpreset.ModelPresetID]spec.ModelPreset,
		len(catalog.Providers),
	)

	for providerName, inferenceProvider := range catalog.Providers {
		ts, err := builtInTimestampForProvider(providerName)
		if err != nil {
			return err
		}

		appModels := make(map[modelpreset.ModelPresetID]spec.ModelPreset, len(inferenceProvider.ModelPresets))
		for _, inferenceModel := range inferenceProvider.ModelPresets {
			appModel := appModelPresetFromInference(providerName, inferenceModel, ts)
			appModels[appModel.ID] = appModel
		}

		defaultModelID, ok := builtin.BuiltInDefaultModelPresetIDs[providerName]
		if !ok || defaultModelID == "" {
			return fmt.Errorf(
				"provider %q has no explicit built-in defaultModelPresetID",
				providerName,
			)
		}
		if _, ok := appModels[defaultModelID]; !ok {
			return fmt.Errorf(
				"provider %q defaultModelPresetID %q not present: %w",
				providerName,
				defaultModelID,
				spec.ErrModelPresetNotFound,
			)
		}

		appProvider := appProviderPresetFromInference(
			inferenceProvider,
			appModels,
			defaultModelID,
			ts,
		)
		if err := validateProviderPreset(&appProvider); err != nil {
			return err
		}

		providers[providerName] = appProvider
		models[providerName] = appModels
	}

	if _, ok := providers[builtin.DefaultBuiltInProvider]; !ok {
		return fmt.Errorf("default provider %q not present in inference catalog", builtin.DefaultBuiltInProvider)
	}

	b.defaultProvider = builtin.DefaultBuiltInProvider
	b.providers = providers
	b.models = models

	b.mu.Lock()
	defer b.mu.Unlock()
	return b.rebuildSnapshot(ctx)
}

func appProviderPresetFromInference(
	in modelpreset.ProviderPreset,
	models map[modelpreset.ModelPresetID]spec.ModelPreset,
	defaultModelID modelpreset.ModelPresetID,
	ts time.Time,
) spec.ProviderPreset {
	headers := maps.Clone(in.DefaultHeaders)
	if extra := builtin.BuiltInProviderDefaultHeaderOverlays[in.Name]; len(extra) > 0 {
		if headers == nil {
			headers = map[string]string{}
		}
		maps.Copy(headers, extra)
	}

	return spec.ProviderPreset{
		SchemaVersion:            spec.SchemaVersion,
		Name:                     in.Name,
		DisplayName:              in.DisplayName,
		SDKType:                  in.SDKType,
		IsEnabled:                true,
		CreatedAt:                ts,
		ModifiedAt:               ts,
		IsBuiltIn:                true,
		Origin:                   in.Origin,
		ChatCompletionPathPrefix: in.ChatCompletionPathPrefix,
		APIKeyHeaderKey:          in.APIKeyHeaderKey,
		DefaultHeaders:           headers,
		CapabilitiesOverride:     capabilityoverride.CloneModelCapabilitiesOverride(in.CapabilitiesOverride),
		DefaultModelPresetID:     defaultModelID,
		ModelPresets:             cloneModelPresetMap(models),
	}
}

func appModelPresetFromInference(
	provider inferenceSpec.ProviderName,
	in modelpreset.ModelPreset,
	ts time.Time,
) spec.ModelPreset {
	modelID := in.ID
	modelParam := in.ModelParam

	var stopSequences *[]string
	if len(modelParam.StopSequences) > 0 {
		s := slices.Clone(modelParam.StopSequences)
		stopSequences = &s
	}

	systemPrompt := modelParam.SystemPrompt

	return spec.ModelPreset{
		Stream:                      new(modelParam.Stream),
		MaxPromptLength:             new(modelParam.MaxPromptLength),
		MaxOutputLength:             new(modelParam.MaxOutputLength),
		Temperature:                 cloneFloat64Ptr(modelParam.Temperature),
		Reasoning:                   cloneReasoningParam(modelParam.Reasoning),
		SystemPrompt:                &systemPrompt,
		Timeout:                     new(modelParam.Timeout),
		CacheControl:                cloneCacheControl(modelParam.CacheControl),
		OutputParam:                 cloneOutputParam(modelParam.OutputParam),
		StopSequences:               stopSequences,
		AdditionalParametersRawJSON: cloneStringPtr(modelParam.AdditionalParametersRawJSON),
		CapabilitiesOverride:        capabilityoverride.CloneModelCapabilitiesOverride(in.CapabilitiesOverride),
		SchemaVersion:               spec.SchemaVersion,
		ID:                          modelID,
		Name:                        modelParam.Name,
		DisplayName:                 in.DisplayName,
		Slug:                        spec.ModelSlug(modelID),
		IsEnabled:                   builtInModelPresetEnabled(provider, modelID),
		CreatedAt:                   ts,
		ModifiedAt:                  ts,
		IsBuiltIn:                   true,
	}
}

func validateBuiltInOverlayCoverage(catalog modelpreset.Catalog) error {
	var errs []error

	if _, ok := catalog.Providers[builtin.DefaultBuiltInProvider]; !ok {
		errs = append(errs, fmt.Errorf(
			"default built-in provider %q is absent from the inference catalog",
			builtin.DefaultBuiltInProvider,
		))
	}

	for providerName, provider := range catalog.Providers {
		if ts, ok := builtin.BuiltInProviderTimestamps[providerName]; !ok {
			errs = append(errs, fmt.Errorf(
				"provider %q is missing a built-in timestamp",
				providerName,
			))
		} else if ts.IsZero() {
			errs = append(errs, fmt.Errorf(
				"provider %q has a zero built-in timestamp",
				providerName,
			))
		}

		defaultModelID, ok := builtin.BuiltInDefaultModelPresetIDs[providerName]
		if !ok || defaultModelID == "" {
			errs = append(errs, fmt.Errorf(
				"provider %q is missing an explicit built-in defaultModelPresetID",
				providerName,
			))
		} else if _, ok := provider.ModelPresets[defaultModelID]; !ok {
			errs = append(errs, fmt.Errorf(
				"provider %q default model %q is absent from the inference catalog",
				providerName,
				defaultModelID,
			))
		}
	}

	for providerName := range builtin.BuiltInProviderTimestamps {
		if _, ok := catalog.Providers[providerName]; !ok {
			errs = append(errs, fmt.Errorf(
				"built-in timestamp references unknown provider %q",
				providerName,
			))
		}
	}

	for providerName := range builtin.BuiltInDefaultModelPresetIDs {
		if _, ok := catalog.Providers[providerName]; !ok {
			errs = append(errs, fmt.Errorf(
				"built-in default model references unknown provider %q",
				providerName,
			))
		}
	}

	for providerName, disabledModels := range builtin.BuiltInDisabledModelPresetIDs {
		provider, ok := catalog.Providers[providerName]
		if !ok {
			errs = append(errs, fmt.Errorf(
				"disabled model overlay references unknown provider %q",
				providerName,
			))
			continue
		}

		for modelID := range disabledModels {
			if _, ok := provider.ModelPresets[modelID]; !ok {
				errs = append(errs, fmt.Errorf(
					"disabled model overlay references unknown model %q/%q",
					providerName,
					modelID,
				))
			}
		}
	}

	for providerName, headers := range builtin.BuiltInProviderDefaultHeaderOverlays {
		if _, ok := catalog.Providers[providerName]; !ok {
			errs = append(errs, fmt.Errorf(
				"default-header overlay references unknown provider %q",
				providerName,
			))
		}
		for key, value := range headers {
			if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
				errs = append(errs, fmt.Errorf(
					"default-header overlay for provider %q contains an empty key or value",
					providerName,
				))
			}
		}
	}

	return errors.Join(errs...)
}

func builtInTimestampForProvider(provider inferenceSpec.ProviderName) (time.Time, error) {
	ts, ok := builtin.BuiltInProviderTimestamps[provider]
	if !ok {
		return time.Time{}, fmt.Errorf(
			"provider %q has no explicit built-in timestamp",
			provider,
		)
	}
	return ts, nil
}

func builtInModelPresetEnabled(
	provider inferenceSpec.ProviderName,
	modelID modelpreset.ModelPresetID,
) bool {
	disabled, ok := builtin.BuiltInDisabledModelPresetIDs[provider]
	if !ok {
		return true
	}
	_, isDisabled := disabled[modelID]
	return !isDisabled
}

func getModelKey(pName inferenceSpec.ProviderName, modelID modelpreset.ModelPresetID) builtInModelKey {
	return builtInModelKey(fmt.Sprintf("%s::%s", pName, modelID))
}
