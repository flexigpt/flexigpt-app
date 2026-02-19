package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"            // POSIX for embed.FS
	"path/filepath" // Native paths
	"sync"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/builtin"
	"github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	"github.com/flexigpt/flexigpt-app/internal/overlay"
	inferencegoSpec "github.com/flexigpt/inference-go/spec"
)

type builtInProviderKey inferencegoSpec.ProviderName

func (builtInProviderKey) Group() overlay.GroupID { return "providers" }
func (k builtInProviderKey) ID() overlay.KeyID    { return overlay.KeyID(k) }

type builtInModelKey spec.ModelPresetID

func (builtInModelKey) Group() overlay.GroupID { return "models" }
func (k builtInModelKey) ID() overlay.KeyID    { return overlay.KeyID(k) }

type builtInProviderDefaultModelIDKey inferencegoSpec.ProviderName

func (builtInProviderDefaultModelIDKey) Group() overlay.GroupID { return "providerDefaultModelIDs" }
func (k builtInProviderDefaultModelIDKey) ID() overlay.KeyID    { return overlay.KeyID(k) }

// BuiltInPresets loads built-in preset assets and maintains an overlay store.
type BuiltInPresets struct {
	// Immutable original data.
	defaultProvider inferencegoSpec.ProviderName
	providers       map[inferencegoSpec.ProviderName]spec.ProviderPreset
	models          map[inferencegoSpec.ProviderName]map[spec.ModelPresetID]spec.ModelPreset

	// View after overlay application, guarded by mu.
	mu         sync.RWMutex
	viewProv   map[inferencegoSpec.ProviderName]spec.ProviderPreset
	viewModels map[inferencegoSpec.ProviderName]map[spec.ModelPresetID]spec.ModelPreset

	// IO.
	presetsFS      fs.FS
	presetsDir     string
	overlayBaseDir string

	store                              *overlay.Store
	providerOverlayFlags               *overlay.TypedGroup[builtInProviderKey, bool]
	modelOverlayFlags                  *overlay.TypedGroup[builtInModelKey, bool]
	providerDefaultModelIDOverlayFlags *overlay.TypedGroup[builtInProviderDefaultModelIDKey, spec.ModelPresetID]

	rebuilder *builtin.AsyncRebuilder
}

type PresetStoreOption func(*BuiltInPresets)

// WithModelPresetsFS sets a custom FS + root for tests.
func WithModelPresetsFS(fsys fs.FS, root string) PresetStoreOption {
	return func(b *BuiltInPresets) {
		b.presetsFS = fsys
		b.presetsDir = root
	}
}

// NewBuiltInPresets prepares the presets store and loads a first snapshot.
func NewBuiltInPresets(
	ctx context.Context,
	overlayBaseDir string,
	maxSnapshotAge time.Duration,
	opts ...PresetStoreOption,
) (*BuiltInPresets, error) {
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

	providerOverlayFlags, err := overlay.NewTypedGroup[builtInProviderKey, bool](ctx, store)
	if err != nil {
		return nil, err
	}
	modelOverlayFlags, err := overlay.NewTypedGroup[builtInModelKey, bool](ctx, store)
	if err != nil {
		return nil, err
	}

	providerDefaultModelIDOverlayFlags, err := overlay.NewTypedGroup[
		builtInProviderDefaultModelIDKey, spec.ModelPresetID](ctx, store)
	if err != nil {
		return nil, err
	}

	b := &BuiltInPresets{
		presetsFS:                          builtin.BuiltInModelPresetsFS,
		presetsDir:                         builtin.BuiltInModelPresetsRootDir,
		overlayBaseDir:                     overlayBaseDir,
		store:                              store,
		providerOverlayFlags:               providerOverlayFlags,
		modelOverlayFlags:                  modelOverlayFlags,
		providerDefaultModelIDOverlayFlags: providerDefaultModelIDOverlayFlags,
	}
	for _, o := range opts {
		o(b)
	}
	if err := b.loadFromFS(ctx); err != nil {
		return nil, err
	}

	b.rebuilder = builtin.NewAsyncRebuilder(
		maxSnapshotAge,
		func() error { //nolint:contextcheck // Cannot pass app context to async builder.
			b.mu.Lock()
			defer b.mu.Unlock()
			return b.rebuildSnapshot(context.Background())
		},
	)
	b.rebuilder.MarkFresh()
	return b, nil
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
	providerPresets map[inferencegoSpec.ProviderName]spec.ProviderPreset,
	modelPresets map[inferencegoSpec.ProviderName]map[spec.ModelPresetID]spec.ModelPreset,
	err error,
) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return maps.Clone(b.viewProv), cloneModels(b.viewModels), nil
}

// GetBuiltInDefaultProviderName fetches the default provider name in builtin.
func (b *BuiltInPresets) GetBuiltInDefaultProviderName(
	ctx context.Context,
) (inferencegoSpec.ProviderName, error) {
	defaultProvider := b.defaultProvider

	if defaultProvider == "" {
		defaultProvider = builtin.ProviderNameOpenAIChatCompletions
	}
	return defaultProvider, nil
}

// GetBuiltInProvider fetches a provider from the snapshot.
func (b *BuiltInPresets) GetBuiltInProvider(
	ctx context.Context,
	name inferencegoSpec.ProviderName,
) (spec.ProviderPreset, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	p, ok := b.viewProv[name]
	if !ok {
		return spec.ProviderPreset{}, spec.ErrProviderNotFound
	}
	return p, nil
}

// SetProviderEnabled toggles a provider.
func (b *BuiltInPresets) SetProviderEnabled(
	ctx context.Context,
	name inferencegoSpec.ProviderName,
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
	b.mu.Unlock()

	b.rebuilder.Trigger()
	return pp, nil
}

// SetModelPresetEnabled toggles a model preset.
func (b *BuiltInPresets) SetModelPresetEnabled(
	ctx context.Context,
	provider inferencegoSpec.ProviderName,
	modelID spec.ModelPresetID,
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
		pp.ModelPresets = map[spec.ModelPresetID]spec.ModelPreset{}
	}
	pp.ModelPresets[modelID] = mp
	b.viewProv[provider] = pp
	b.mu.Unlock()

	b.rebuilder.Trigger()
	return mp, nil
}

// GetBuiltInModelPreset fetches a model preset.
func (b *BuiltInPresets) GetBuiltInModelPreset(
	ctx context.Context,
	provider inferencegoSpec.ProviderName,
	modelID spec.ModelPresetID,
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
	return mp, nil
}

func (b *BuiltInPresets) SetDefaultModelPreset(
	ctx context.Context,
	provider inferencegoSpec.ProviderName,
	modelID spec.ModelPresetID,
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
		ctx, builtInProviderDefaultModelIDKey(provider), modelID)
	if err != nil {
		return spec.ProviderPreset{}, err
	}

	// Update hot snapshot.
	b.mu.Lock()
	pp := b.viewProv[provider]
	pp.DefaultModelPresetID = modelID
	pp.ModifiedAt = flag.ModifiedAt
	b.viewProv[provider] = pp
	b.mu.Unlock()

	b.rebuilder.Trigger()
	return pp, nil
}

func (b *BuiltInPresets) loadFromFS(ctx context.Context) error {
	subFS, err := resolvePresetsFS(b.presetsFS, b.presetsDir)
	if err != nil {
		return err
	}
	raw, err := fs.ReadFile(subFS, builtin.BuiltInModelPresetsJSON)
	if err != nil {
		return err
	}

	var schema spec.PresetsSchema
	if err := json.Unmarshal(raw, &schema); err != nil {
		return err
	}
	if schema.SchemaVersion != spec.SchemaVersion {
		return fmt.Errorf("schemaVersion %q not equal to %q",
			schema.SchemaVersion, spec.SchemaVersion)
	}
	if schema.DefaultProvider == "" {
		return errors.New("no default provider in builtin")
	}
	if len(schema.ProviderPresets) == 0 {
		return fmt.Errorf("%s contains no providers", builtin.BuiltInModelPresetsJSON)
	}

	// Parse + validate.
	prov := make(map[inferencegoSpec.ProviderName]spec.ProviderPreset, len(schema.ProviderPresets))
	models := make(map[inferencegoSpec.ProviderName]map[spec.ModelPresetID]spec.ModelPreset)

	for name, pp := range schema.ProviderPresets {
		if err := validateProviderPreset(&pp); err != nil {
			return err
		}
		pp.IsBuiltIn = true
		prov[name] = pp

		sub := make(map[spec.ModelPresetID]spec.ModelPreset, len(pp.ModelPresets))
		for mid, mp := range pp.ModelPresets {
			mp.IsBuiltIn = true
			sub[mid] = mp
		}
		models[name] = sub
	}

	if _, ok := prov[schema.DefaultProvider]; !ok {
		return errors.New("default provider not present in presets")
	}

	b.defaultProvider = schema.DefaultProvider
	b.providers = prov
	b.models = models

	b.mu.Lock()
	defer b.mu.Unlock()
	return b.rebuildSnapshot(ctx)
}

// rebuildSnapshot applies overlay flags onto the immutable base sets.
// Caller must hold write lock.
func (b *BuiltInPresets) rebuildSnapshot(ctx context.Context) error {
	newProv := make(map[inferencegoSpec.ProviderName]spec.ProviderPreset, len(b.providers))
	newModels := make(map[inferencegoSpec.ProviderName]map[spec.ModelPresetID]spec.ModelPreset, len(b.models))

	for pname, mm := range b.models {
		sub := make(map[spec.ModelPresetID]spec.ModelPreset, len(mm))
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
			ctx, builtInProviderDefaultModelIDKey(pname)); err != nil {
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

func cloneModels(
	src map[inferencegoSpec.ProviderName]map[spec.ModelPresetID]spec.ModelPreset,
) map[inferencegoSpec.ProviderName]map[spec.ModelPresetID]spec.ModelPreset {
	dst := make(map[inferencegoSpec.ProviderName]map[spec.ModelPresetID]spec.ModelPreset, len(src))
	for pname, inner := range src {
		dst[pname] = maps.Clone(inner)
	}
	return dst
}

func resolvePresetsFS(fsys fs.FS, dir string) (fs.FS, error) {
	if dir == "" || dir == "." {
		return fsys, nil
	}
	return fs.Sub(fsys, dir)
}

func getModelKey(pName inferencegoSpec.ProviderName, modelID spec.ModelPresetID) builtInModelKey {
	return builtInModelKey(fmt.Sprintf("%s::%s", pName, modelID))
}
