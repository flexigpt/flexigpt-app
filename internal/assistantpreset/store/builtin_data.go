package store

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/assistantpreset/spec"
	"github.com/flexigpt/flexigpt-app/internal/builtin"
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/fsutil"
	"github.com/flexigpt/flexigpt-app/internal/overlay"
)

type builtInBundleID bundleitemutils.BundleID

func (builtInBundleID) Group() overlay.GroupID { return "bundles" }
func (b builtInBundleID) ID() overlay.KeyID    { return overlay.KeyID(b) }

type builtInAssistantPresetID bundleitemutils.ItemID

func (builtInAssistantPresetID) Group() overlay.GroupID { return "assistantpresets" }
func (p builtInAssistantPresetID) ID() overlay.KeyID    { return overlay.KeyID(p) }

type BuiltInData struct {
	bundlesFS      fs.FS
	bundlesDir     string
	overlayBaseDir string
	lookups        ReferenceLookups

	bundles map[bundleitemutils.BundleID]spec.AssistantPresetBundle
	presets map[bundleitemutils.BundleID]map[bundleitemutils.ItemID]spec.AssistantPreset

	store              *overlay.Store
	bundleOverlayFlags *overlay.TypedGroup[builtInBundleID, bool]
	presetOverlayFlags *overlay.TypedGroup[builtInAssistantPresetID, bool]

	mu          sync.RWMutex
	viewBundles map[bundleitemutils.BundleID]spec.AssistantPresetBundle
	viewPresets map[bundleitemutils.BundleID]map[bundleitemutils.ItemID]spec.AssistantPreset

	rebuilder *builtin.AsyncRebuilder
}

type BuiltInDataOption func(*BuiltInData)

// WithBundlesFS overrides the default embedded built-in assistant preset FS.
func WithBundlesFS(fsys fs.FS, rootDir string) BuiltInDataOption {
	return func(d *BuiltInData) {
		d.bundlesFS = fsys
		d.bundlesDir = rootDir
	}
}

func NewBuiltInData(
	ctx context.Context,
	overlayBaseDir string,
	snapshotMaxAge time.Duration,
	lookups ReferenceLookups,
	opts ...BuiltInDataOption,
) (data *BuiltInData, err error) {
	if snapshotMaxAge <= 0 {
		snapshotMaxAge = time.Hour
	}
	if overlayBaseDir == "" {
		return nil, fmt.Errorf("%w: overlayBaseDir", spec.ErrInvalidDir)
	}
	if err := os.MkdirAll(overlayBaseDir, 0o755); err != nil {
		return nil, err
	}

	store, err := overlay.NewOverlayStore(
		ctx,
		filepath.Join(overlayBaseDir, spec.AssistantPresetBuiltInOverlayDBFileName),
		overlay.WithKeyType[builtInBundleID](),
		overlay.WithKeyType[builtInAssistantPresetID](),
	)
	if err != nil {
		return nil, err
	}

	data = &BuiltInData{
		bundlesFS:      builtin.BuiltInAssistantPresetBundlesFS,
		bundlesDir:     builtin.BuiltInAssistantPresetBundlesRootDir,
		overlayBaseDir: overlayBaseDir,
		lookups:        lookups,
		store:          store,
	}

	defer func() {
		if err != nil && data != nil {
			_ = data.Close()
			data = nil
		}
	}()

	data.bundleOverlayFlags, err = overlay.NewTypedGroup[builtInBundleID, bool](ctx, store)
	if err != nil {
		return nil, err
	}

	data.presetOverlayFlags, err = overlay.NewTypedGroup[builtInAssistantPresetID, bool](ctx, store)
	if err != nil {
		return nil, err
	}

	for _, opt := range opts {
		opt(data)
	}

	if err := data.populateDataFromFS(ctx); err != nil {
		return nil, err
	}

	data.rebuilder = builtin.NewAsyncRebuilder(
		snapshotMaxAge,
		func() error { //nolint:contextcheck // background rebuilder cannot reuse caller ctx
			data.mu.Lock()
			defer data.mu.Unlock()
			return data.rebuildSnapshot(context.Background())
		},
	)
	data.rebuilder.MarkFresh()

	return data, nil
}

func (d *BuiltInData) Close() error {
	if d == nil {
		return nil
	}
	if d.rebuilder != nil {
		d.rebuilder.Close()
	}
	if d.store != nil {
		return d.store.Close()
	}
	return nil
}

// ListBuiltInData returns deep-copied snapshots.
func (d *BuiltInData) ListBuiltInData(
	ctx context.Context,
) (
	bundles map[bundleitemutils.BundleID]spec.AssistantPresetBundle,
	presets map[bundleitemutils.BundleID]map[bundleitemutils.ItemID]spec.AssistantPreset,
	err error,
) {
	_ = ctx

	d.mu.RLock()
	defer d.mu.RUnlock()

	bundles = maps.Clone(d.viewBundles)
	presets = cloneAllAssistantPresets(d.viewPresets)
	return bundles, presets, nil
}

func (d *BuiltInData) SetAssistantPresetBundleEnabled(
	ctx context.Context,
	id bundleitemutils.BundleID,
	enabled bool,
) (spec.AssistantPresetBundle, error) {
	if _, ok := d.bundles[id]; !ok {
		return spec.AssistantPresetBundle{}, fmt.Errorf(
			"bundleID: %q, err: %w",
			id,
			spec.ErrBuiltInBundleNotFound,
		)
	}

	flag, err := d.bundleOverlayFlags.SetFlag(ctx, builtInBundleID(id), enabled)
	if err != nil {
		return spec.AssistantPresetBundle{}, err
	}

	d.mu.Lock()
	b := d.viewBundles[id]
	b.IsEnabled = enabled
	b.ModifiedAt = flag.ModifiedAt
	d.viewBundles[id] = b
	d.mu.Unlock()

	if d.rebuilder != nil {
		d.rebuilder.Trigger()
	}

	return b, nil
}

func (d *BuiltInData) GetBuiltInBundle(
	ctx context.Context,
	id bundleitemutils.BundleID,
) (spec.AssistantPresetBundle, error) {
	_ = ctx

	d.mu.RLock()
	defer d.mu.RUnlock()

	b, ok := d.viewBundles[id]
	if !ok {
		return spec.AssistantPresetBundle{}, spec.ErrBundleNotFound
	}
	return b, nil
}

func (d *BuiltInData) SetAssistantPresetEnabled(
	ctx context.Context,
	bundleID bundleitemutils.BundleID,
	slug bundleitemutils.ItemSlug,
	version bundleitemutils.ItemVersion,
	enabled bool,
) (spec.AssistantPreset, error) {
	preset, err := d.GetBuiltInAssistantPreset(ctx, bundleID, slug, version)
	if err != nil {
		return spec.AssistantPreset{}, err
	}

	flag, err := d.presetOverlayFlags.SetFlag(
		ctx,
		getAssistantPresetKey(bundleID, preset.ID),
		enabled,
	)
	if err != nil {
		return spec.AssistantPreset{}, err
	}

	d.mu.Lock()
	preset.IsEnabled = enabled
	preset.ModifiedAt = flag.ModifiedAt
	d.viewPresets[bundleID][preset.ID] = preset
	d.mu.Unlock()

	if d.rebuilder != nil {
		d.rebuilder.Trigger()
	}

	return cloneAssistantPreset(preset), nil
}

func (d *BuiltInData) GetBuiltInAssistantPreset(
	ctx context.Context,
	bundleID bundleitemutils.BundleID,
	slug bundleitemutils.ItemSlug,
	version bundleitemutils.ItemVersion,
) (spec.AssistantPreset, error) {
	_ = ctx

	d.mu.RLock()
	defer d.mu.RUnlock()

	presets, ok := d.viewPresets[bundleID]
	if !ok {
		return spec.AssistantPreset{}, spec.ErrBundleNotFound
	}

	for _, preset := range presets {
		if preset.Slug == slug && preset.Version == version {
			return cloneAssistantPreset(preset), nil
		}
	}

	return spec.AssistantPreset{}, fmt.Errorf(
		"%w: bundleID=%s, slug=%s, version=%s",
		spec.ErrAssistantPresetNotFound,
		bundleID,
		slug,
		version,
	)
}

func (d *BuiltInData) populateDataFromFS(ctx context.Context) error {
	bundlesFS, err := fsutil.ResolveFS(d.bundlesFS, d.bundlesDir)
	if err != nil {
		return err
	}

	rawManifest, err := fs.ReadFile(bundlesFS, builtin.BuiltInAssistantPresetBundlesJSON)
	if err != nil {
		return err
	}

	var manifest spec.AllBundles
	if err := json.Unmarshal(rawManifest, &manifest); err != nil {
		return err
	}

	bundleMap := make(
		map[bundleitemutils.BundleID]spec.AssistantPresetBundle,
		len(manifest.Bundles),
	)
	presetMap := make(
		map[bundleitemutils.BundleID]map[bundleitemutils.ItemID]spec.AssistantPreset,
		len(manifest.Bundles),
	)

	for id, bundle := range manifest.Bundles {
		bundle.IsBuiltIn = true
		if err := validateAssistantPresetBundle(&bundle); err != nil {
			return fmt.Errorf("manifest bundle %s invalid: %w", id, err)
		}
		bundleMap[id] = bundle
		presetMap[id] = make(map[bundleitemutils.ItemID]spec.AssistantPreset)
	}

	if len(bundleMap) == 0 {
		// Keep internal state consistent and explicit.
		d.bundles = map[bundleitemutils.BundleID]spec.AssistantPresetBundle{}
		d.presets = map[bundleitemutils.BundleID]map[bundleitemutils.ItemID]spec.AssistantPreset{}

		d.mu.Lock()
		defer d.mu.Unlock()
		return d.rebuildSnapshot(ctx)
	}

	seenPresetPerBundle := make(map[bundleitemutils.BundleID]map[bundleitemutils.ItemID]string)

	err = fs.WalkDir(
		bundlesFS,
		".",
		func(inPath string, de fs.DirEntry, _ error) error {
			if de.IsDir() || path.Ext(inPath) != ".json" {
				return nil
			}

			fn := path.Base(inPath)
			if fn == builtin.BuiltInAssistantPresetBundlesJSON ||
				fn == spec.AssistantPresetBuiltInOverlayDBFileName {
				return nil
			}

			dir := path.Base(path.Dir(inPath))
			dirInfo, derr := bundleitemutils.ParseBundleDir(dir)
			if derr != nil {
				return fmt.Errorf("%s: %w", inPath, derr)
			}
			bundleID := dirInfo.ID

			bundleDef, ok := bundleMap[bundleID]
			if !ok {
				return fmt.Errorf(
					"%s: bundle dir %q not in %s",
					inPath,
					bundleID,
					builtin.BuiltInAssistantPresetBundlesJSON,
				)
			}
			if dirInfo.Slug != bundleDef.Slug {
				return fmt.Errorf(
					"%s: dir slug %q not equal to manifest slug %q",
					inPath,
					dirInfo.Slug,
					bundleDef.Slug,
				)
			}

			raw, err := fs.ReadFile(bundlesFS, inPath)
			if err != nil {
				return err
			}

			var preset spec.AssistantPreset
			if err := json.Unmarshal(raw, &preset); err != nil {
				return fmt.Errorf("%s: %w", inPath, err)
			}
			preset.IsBuiltIn = true

			if err := validateAssistantPreset(ctx, &preset, d.lookups); err != nil {
				return fmt.Errorf("%s: invalid assistant preset: %w", inPath, err)
			}

			info, err := bundleitemutils.ParseItemFileName(fn)
			if err != nil {
				return fmt.Errorf("%s: %w", inPath, err)
			}
			if info.Slug != preset.Slug || info.Version != preset.Version {
				return fmt.Errorf(
					"%s: filename (slug=%q,ver=%q) not equal to JSON (slug=%q,ver=%q)",
					inPath,
					info.Slug,
					info.Version,
					preset.Slug,
					preset.Version,
				)
			}

			if seenPresetPerBundle[bundleID] == nil {
				seenPresetPerBundle[bundleID] = make(map[bundleitemutils.ItemID]string)
			}
			if prev := seenPresetPerBundle[bundleID][preset.ID]; prev != "" {
				return fmt.Errorf(
					"%s: duplicate assistant preset ID %s within bundle %s (also %s)",
					inPath,
					preset.ID,
					bundleID,
					prev,
				)
			}
			seenPresetPerBundle[bundleID][preset.ID] = inPath

			presetMap[bundleID][preset.ID] = preset
			return nil
		},
	)
	if err != nil {
		return err
	}

	for id, presets := range presetMap {
		if len(presets) == 0 {
			return fmt.Errorf("built-in data: bundle %s has no assistant presets", id)
		}
	}

	d.bundles = bundleMap
	d.presets = presetMap

	d.mu.Lock()
	if err := d.rebuildSnapshot(ctx); err != nil {
		d.mu.Unlock()
		return err
	}
	d.mu.Unlock()

	return nil
}

// rebuildSnapshot assumes d.mu is already locked.
func (d *BuiltInData) rebuildSnapshot(ctx context.Context) error {
	newBundles := make(
		map[bundleitemutils.BundleID]spec.AssistantPresetBundle,
		len(d.bundles),
	)
	newPresets := make(
		map[bundleitemutils.BundleID]map[bundleitemutils.ItemID]spec.AssistantPreset,
		len(d.presets),
	)

	for id, bundle := range d.bundles {
		flag, ok, err := d.bundleOverlayFlags.GetFlag(ctx, builtInBundleID(id))
		if err != nil {
			return err
		}
		if ok {
			bundle.IsEnabled = flag.Value
			bundle.ModifiedAt = flag.ModifiedAt
		}
		newBundles[id] = bundle
	}

	for bid, presets := range d.presets {
		sub := make(map[bundleitemutils.ItemID]spec.AssistantPreset, len(presets))
		for pid, preset := range presets {
			flag, ok, err := d.presetOverlayFlags.GetFlag(ctx, getAssistantPresetKey(bid, pid))
			if err != nil {
				return err
			}
			if ok {
				preset.IsEnabled = flag.Value
				preset.ModifiedAt = flag.ModifiedAt
			}
			sub[pid] = preset
		}
		newPresets[bid] = sub
	}

	d.viewBundles = newBundles
	d.viewPresets = newPresets
	return nil
}

func getAssistantPresetKey(
	bundleID bundleitemutils.BundleID,
	presetID bundleitemutils.ItemID,
) builtInAssistantPresetID {
	return builtInAssistantPresetID(fmt.Sprintf("%s::%s", bundleID, presetID))
}
