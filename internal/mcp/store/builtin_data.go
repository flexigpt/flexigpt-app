package store

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/builtin"
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/fsutil"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
	"github.com/flexigpt/flexigpt-app/internal/overlay"
)

type builtInMCPBundleID bundleitemutils.BundleID

func (builtInMCPBundleID) Group() overlay.GroupID { return "bundles" }
func (b builtInMCPBundleID) ID() overlay.KeyID    { return overlay.KeyID(b) }

type builtInMCPServerID string

func (builtInMCPServerID) Group() overlay.GroupID { return "servers" }
func (s builtInMCPServerID) ID() overlay.KeyID    { return overlay.KeyID(s) }

type BuiltInData struct {
	bundlesFS      fs.FS
	bundlesDir     string
	overlayBaseDir string

	bundles map[bundleitemutils.BundleID]spec.MCPBundle
	servers map[bundleitemutils.BundleID]map[spec.MCPServerID]spec.MCPServerConfig

	store              *overlay.Store
	bundleOverlayFlags *overlay.TypedGroup[builtInMCPBundleID, bool]
	serverOverlayFlags *overlay.TypedGroup[builtInMCPServerID, bool]

	mu          sync.RWMutex
	viewBundles map[bundleitemutils.BundleID]spec.MCPBundle
	viewServers map[bundleitemutils.BundleID]map[spec.MCPServerID]spec.MCPServerConfig

	rebuilder *builtin.AsyncRebuilder
}

type BuiltInDataOption func(*BuiltInData)

func WithMCPBundlesFS(fsys fs.FS, rootDir string) BuiltInDataOption {
	return func(d *BuiltInData) {
		d.bundlesFS = fsys
		d.bundlesDir = rootDir
	}
}

func NewBuiltInData(
	ctx context.Context,
	overlayBaseDir string,
	builtInSnapshotMaxAge time.Duration,
	opts ...BuiltInDataOption,
) (data *BuiltInData, err error) {
	if builtInSnapshotMaxAge <= 0 {
		builtInSnapshotMaxAge = time.Hour
	}
	if overlayBaseDir == "" {
		return nil, fmt.Errorf("%w: overlayBaseDir", spec.ErrMCPInvalidRequest)
	}
	if err := os.MkdirAll(overlayBaseDir, 0o755); err != nil {
		return nil, err
	}

	store, err := overlay.NewOverlayStore(
		ctx,
		filepath.Join(overlayBaseDir, spec.MCPBuiltInOverlayDBFileName),
		overlay.WithKeyType[builtInMCPBundleID](),
		overlay.WithKeyType[builtInMCPServerID](),
	)
	if err != nil {
		return nil, err
	}

	data = &BuiltInData{
		bundlesFS:      builtin.BuiltInMCPBundlesFS,
		bundlesDir:     builtin.BuiltInMCPBundlesRootDir,
		overlayBaseDir: overlayBaseDir,
		store:          store,
	}

	defer func() {
		if err != nil && data != nil {
			_ = data.Close()
			data = nil
		}
	}()

	data.bundleOverlayFlags, err = overlay.NewTypedGroup[builtInMCPBundleID, bool](ctx, store)
	if err != nil {
		return nil, err
	}
	data.serverOverlayFlags, err = overlay.NewTypedGroup[builtInMCPServerID, bool](ctx, store)
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
		builtInSnapshotMaxAge,
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

func (d *BuiltInData) ListBuiltInData(ctx context.Context) (
	bundleMap map[bundleitemutils.BundleID]spec.MCPBundle,
	serverMap map[bundleitemutils.BundleID]map[spec.MCPServerID]spec.MCPServerConfig,
	err error,
) {
	if d == nil {
		return nil, nil, nil
	}
	d.mu.RLock()
	defer d.mu.RUnlock()

	return cloneBundleMap(d.viewBundles), cloneAllServerMaps(d.viewServers), nil
}

func (d *BuiltInData) GetBuiltInBundle(
	ctx context.Context,
	id bundleitemutils.BundleID,
) (spec.MCPBundle, error) {
	if d == nil || id == "" {
		return spec.MCPBundle{}, fmt.Errorf("%w: %s", spec.ErrMCPBundleNotFound, id)
	}
	d.mu.RLock()
	defer d.mu.RUnlock()

	b, ok := d.viewBundles[id]
	if !ok {
		return spec.MCPBundle{}, fmt.Errorf("%w: %s", spec.ErrMCPBundleNotFound, id)
	}
	return cloneBundle(b), nil
}

func (d *BuiltInData) GetBuiltInServer(
	ctx context.Context,
	bundleID bundleitemutils.BundleID,
	serverID spec.MCPServerID,
) (spec.MCPServerConfig, error) {
	if d == nil || bundleID == "" || serverID == "" {
		return spec.MCPServerConfig{}, fmt.Errorf("%w: %s", spec.ErrMCPServerNotFound, serverID)
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	if servers := d.viewServers[bundleID]; servers != nil {
		if cfg, ok := servers[serverID]; ok {
			return cloneServerConfig(cfg), nil
		}
	}
	return spec.MCPServerConfig{}, fmt.Errorf("%w: %s", spec.ErrMCPServerNotFound, serverID)
}

func (d *BuiltInData) FindBuiltInServerByID(
	ctx context.Context,
	serverID spec.MCPServerID,
) (spec.MCPServerConfig, error) {
	if d == nil || serverID == "" {
		return spec.MCPServerConfig{}, fmt.Errorf("%w: %s", spec.ErrMCPServerNotFound, serverID)
	}

	d.mu.RLock()
	defer d.mu.RUnlock()
	for _, servers := range d.viewServers {
		if cfg, ok := servers[serverID]; ok {
			return cloneServerConfig(cfg), nil
		}
	}

	return spec.MCPServerConfig{}, fmt.Errorf("%w: %s", spec.ErrMCPServerNotFound, serverID)
}

func (d *BuiltInData) SetBundleEnabled(
	ctx context.Context,
	id bundleitemutils.BundleID,
	enabled bool,
) (spec.MCPBundle, error) {
	if d == nil {
		return spec.MCPBundle{}, fmt.Errorf("%w: built-in data unavailable", spec.ErrMCPBundleNotFound)
	}
	if _, ok := d.bundles[id]; !ok {
		return spec.MCPBundle{}, fmt.Errorf("%w: %s", spec.ErrMCPBundleNotFound, id)
	}

	flag, err := d.bundleOverlayFlags.SetFlag(ctx, builtInMCPBundleID(id), enabled)
	if err != nil {
		return spec.MCPBundle{}, err
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

	return cloneBundle(b), nil
}

func (d *BuiltInData) SetServerEnabled(
	ctx context.Context,
	bundleID bundleitemutils.BundleID,
	serverID spec.MCPServerID,
	enabled bool,
) (spec.MCPServerConfig, error) {
	if d == nil {
		return spec.MCPServerConfig{}, fmt.Errorf("%w: built-in data unavailable", spec.ErrMCPServerNotFound)
	}

	if bundleID == "" {
		return spec.MCPServerConfig{}, fmt.Errorf("%w: bundleID required", spec.ErrMCPInvalidRequest)
	}

	if d.servers[bundleID] == nil {
		return spec.MCPServerConfig{}, fmt.Errorf("%w: %s", spec.ErrMCPServerNotFound, serverID)
	}
	if _, ok := d.servers[bundleID][serverID]; !ok {
		return spec.MCPServerConfig{}, fmt.Errorf("%w: %s", spec.ErrMCPServerNotFound, serverID)
	}

	flag, err := d.serverOverlayFlags.SetFlag(ctx, builtInServerKey(bundleID, serverID), enabled)
	if err != nil {
		return spec.MCPServerConfig{}, err
	}

	d.mu.Lock()
	cfg := d.viewServers[bundleID][serverID]
	cfg.Enabled = enabled
	cfg.ModifiedAt = flag.ModifiedAt
	d.viewServers[bundleID][serverID] = cfg
	d.mu.Unlock()

	if d.rebuilder != nil {
		d.rebuilder.Trigger()
	}

	return cloneServerConfig(cfg), nil
}

func (d *BuiltInData) populateDataFromFS(ctx context.Context) error {
	bundlesFS, err := fsutil.ResolveFS(d.bundlesFS, d.bundlesDir)
	if err != nil {
		return err
	}

	rawBundles, err := fs.ReadFile(bundlesFS, builtin.BuiltInMCPBundlesJSON)
	if err != nil {
		return err
	}

	var manifest spec.AllMCPBundles
	if err := json.Unmarshal(rawBundles, &manifest); err != nil {
		return err
	}
	if len(manifest.Bundles) == 0 {
		return fmt.Errorf("built-in mcp data: %s contains no bundles", builtin.BuiltInMCPBundlesJSON)
	}

	bundleMap := make(map[bundleitemutils.BundleID]spec.MCPBundle, len(manifest.Bundles))
	serverMap := make(map[bundleitemutils.BundleID]map[spec.MCPServerID]spec.MCPServerConfig, len(manifest.Bundles))

	for id, b := range manifest.Bundles {
		b.ID = id
		b.IsBuiltIn = true
		if err := validateBundle(&b); err != nil {
			return fmt.Errorf("built-in mcp bundle %s: %w", id, err)
		}
		bundleMap[id] = b
		serverMap[id] = map[spec.MCPServerID]spec.MCPServerConfig{}
	}

	seenServers := map[spec.MCPServerID]string{}

	err = fs.WalkDir(
		bundlesFS,
		".",
		func(inPath string, de fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if de.IsDir() || path.Ext(inPath) != ".json" {
				return nil
			}
			if path.Base(inPath) == builtin.BuiltInMCPBundlesJSON {
				return nil
			}

			dir := path.Base(path.Dir(inPath))
			dirInfo, err := bundleitemutils.ParseBundleDir(dir)
			if err != nil {
				return fmt.Errorf("%s: %w", inPath, err)
			}

			bundleDef, ok := bundleMap[dirInfo.ID]
			if !ok {
				return fmt.Errorf("%s: bundle %q not in %s", inPath, dirInfo.ID, builtin.BuiltInMCPBundlesJSON)
			}
			if bundleDef.Slug != dirInfo.Slug {
				return fmt.Errorf("%s: dir slug %q != manifest slug %q", inPath, dirInfo.Slug, bundleDef.Slug)
			}

			raw, err := fs.ReadFile(bundlesFS, inPath)
			if err != nil {
				return err
			}

			var cfg spec.MCPServerConfig
			if err := json.Unmarshal(raw, &cfg); err != nil {
				return fmt.Errorf("%s: %w", inPath, err)
			}

			if cfg.BundleID != "" && cfg.BundleID != dirInfo.ID {
				return fmt.Errorf(
					"%s: bundleID %q does not match parent bundle %q",
					inPath,
					cfg.BundleID,
					dirInfo.ID,
				)
			}
			cfg.BundleID = dirInfo.ID
			cfg.IsBuiltIn = true
			cfg.SoftDeletedAt = nil

			if err := validateServerConfig(&cfg); err != nil {
				return fmt.Errorf("%s: invalid mcp server: %w", inPath, err)
			}

			if prev := seenServers[cfg.ID]; prev != "" {
				return fmt.Errorf("%s: duplicate built-in mcp server id %s, also in %s", inPath, cfg.ID, prev)
			}
			seenServers[cfg.ID] = inPath

			serverMap[dirInfo.ID][cfg.ID] = cfg
			return nil
		},
	)
	if err != nil {
		return err
	}

	for id, servers := range serverMap {
		if len(servers) == 0 {
			return fmt.Errorf("built-in mcp bundle %s has no servers", id)
		}
	}

	d.bundles = bundleMap
	d.servers = serverMap

	d.mu.Lock()
	defer d.mu.Unlock()

	return d.rebuildSnapshot(ctx)
}

func (d *BuiltInData) rebuildSnapshot(ctx context.Context) error {
	newBundles := make(map[bundleitemutils.BundleID]spec.MCPBundle, len(d.bundles))
	newServers := make(map[bundleitemutils.BundleID]map[spec.MCPServerID]spec.MCPServerConfig, len(d.servers))

	for id, b := range d.bundles {
		flag, ok, err := d.bundleOverlayFlags.GetFlag(ctx, builtInMCPBundleID(id))
		if err != nil {
			return err
		}
		if ok {
			b.IsEnabled = flag.Value
			b.ModifiedAt = flag.ModifiedAt
		}
		newBundles[id] = b
	}

	for bid, servers := range d.servers {
		sub := make(map[spec.MCPServerID]spec.MCPServerConfig, len(servers))
		for sid, cfg := range servers {
			flag, ok, err := d.serverOverlayFlags.GetFlag(ctx, builtInServerKey(bid, sid))
			if err != nil {
				return err
			}
			if ok {
				cfg.Enabled = flag.Value
				cfg.ModifiedAt = flag.ModifiedAt
			}
			sub[sid] = cfg
		}
		newServers[bid] = sub
	}

	d.viewBundles = newBundles
	d.viewServers = newServers
	return nil
}

func builtInServerKey(bundleID bundleitemutils.BundleID, serverID spec.MCPServerID) builtInMCPServerID {
	return builtInMCPServerID(fmt.Sprintf("%s::%s", bundleID, serverID))
}
