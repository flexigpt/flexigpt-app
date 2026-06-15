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

type builtInMCPServerSetupID string

func (builtInMCPServerSetupID) Group() overlay.GroupID { return "serverSetups" }
func (s builtInMCPServerSetupID) ID() overlay.KeyID    { return overlay.KeyID(s) }

type BuiltInData struct {
	bundlesFS      fs.FS
	bundlesDir     string
	overlayBaseDir string

	bundles map[bundleitemutils.BundleID]spec.MCPBundle
	servers map[bundleitemutils.BundleID]map[spec.MCPServerID]spec.MCPServerConfig

	store              *overlay.Store
	bundleOverlayFlags *overlay.TypedGroup[builtInMCPBundleID, bool]
	serverOverlayFlags *overlay.TypedGroup[builtInMCPServerID, bool]
	serverSetups       *overlay.TypedGroup[builtInMCPServerSetupID, spec.MCPBuiltInServerOverlay]

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
		overlay.WithKeyType[builtInMCPServerSetupID](),
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
	data.serverSetups, err = overlay.NewTypedGroup[builtInMCPServerSetupID, spec.MCPBuiltInServerOverlay](ctx, store)
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
	if d == nil {
		return spec.MCPBundle{}, fmt.Errorf("%w: built-in data unavailable", spec.ErrMCPBundleNotFound)
	}
	if err := requireMCPBundleID(id); err != nil {
		return spec.MCPBundle{}, err
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
	if d == nil {
		return spec.MCPServerConfig{}, fmt.Errorf("%w: built-in data unavailable", spec.ErrMCPServerNotFound)
	}
	if err := requireMCPBundleServerIDs(bundleID, serverID); err != nil {
		return spec.MCPServerConfig{}, err
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
	if d == nil {
		return spec.MCPServerConfig{}, fmt.Errorf("%w: built-in data unavailable", spec.ErrMCPServerNotFound)
	}
	if serverID == "" {
		return spec.MCPServerConfig{}, fmt.Errorf("%w: serverID required", spec.ErrMCPInvalidRequest)
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
	if err := requireMCPBundleID(id); err != nil {
		return spec.MCPBundle{}, err
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
	if err := requireMCPBundleServerIDs(bundleID, serverID); err != nil {
		return spec.MCPServerConfig{}, err
	}
	if _, ok := d.servers[bundleID]; !ok {
		return spec.MCPServerConfig{}, fmt.Errorf("%w: %s", spec.ErrMCPBundleNotFound, bundleID)
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

// ApplyServerSetupOverlay merges a setup overlay fragment into the stored
// overlay (or starts fresh when reset is true), validates the resulting config,
// persists, and returns the rebuilt config.
func (d *BuiltInData) ApplyServerSetupOverlay(
	ctx context.Context,
	bundleID bundleitemutils.BundleID,
	serverID spec.MCPServerID,
	patch spec.MCPBuiltInServerOverlay,
	reset bool,
) (spec.MCPServerConfig, error) {
	if d == nil {
		return spec.MCPServerConfig{}, fmt.Errorf("%w: built-in data unavailable", spec.ErrMCPServerNotFound)
	}
	if err := requireMCPBundleServerIDs(bundleID, serverID); err != nil {
		return spec.MCPServerConfig{}, err
	}

	d.mu.RLock()
	base, ok := d.servers[bundleID][serverID]
	d.mu.RUnlock()
	if !ok {
		return spec.MCPServerConfig{}, fmt.Errorf("%w: %s", spec.ErrMCPServerNotFound, serverID)
	}

	current := spec.MCPBuiltInServerOverlay{}
	if !reset {
		if flag, ok, err := d.serverSetups.GetFlag(ctx, builtInServerSetupKey(bundleID, serverID)); err != nil {
			return spec.MCPServerConfig{}, err
		} else if ok {
			current = flag.Value
		}
	}
	merged := mergeBuiltInServerOverlay(current, patch)

	candidate, err := applyServerOverlay(base, merged)
	if err != nil {
		return spec.MCPServerConfig{}, err
	}
	if err := validateServerConfig(&candidate); err != nil {
		return spec.MCPServerConfig{}, fmt.Errorf("%w: %w", spec.ErrMCPInvalidRequest, err)
	}

	if _, err := d.serverSetups.SetFlag(ctx, builtInServerSetupKey(bundleID, serverID), merged); err != nil {
		return spec.MCPServerConfig{}, err
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.rebuildSnapshot(ctx); err != nil {
		return spec.MCPServerConfig{}, err
	}
	return cloneServerConfig(d.viewServers[bundleID][serverID]), nil
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
			if flag, ok, err := d.serverSetups.GetFlag(ctx, builtInServerSetupKey(bid, sid)); err != nil {
				return err
			} else if ok {
				cfg, err = applyServerOverlay(cfg, flag.Value)
				if err != nil {
					return fmt.Errorf("built-in mcp server setup %s/%s: %w", bid, sid, err)
				}
				if flag.ModifiedAt.After(cfg.ModifiedAt) {
					cfg.ModifiedAt = flag.ModifiedAt
				}
			}
			sub[sid] = cfg
		}
		newServers[bid] = sub
	}

	d.viewBundles = newBundles
	d.viewServers = newServers
	return nil
}

func builtInServerSetupKey(
	bundleID bundleitemutils.BundleID,
	serverID spec.MCPServerID,
) builtInMCPServerSetupID {
	return builtInMCPServerSetupID(fmt.Sprintf("%s::%s", bundleID, serverID))
}

func mergeBuiltInServerOverlay(dst, src spec.MCPBuiltInServerOverlay) spec.MCPBuiltInServerOverlay {
	out := dst
	if src.Stdio != nil {
		if out.Stdio == nil {
			out.Stdio = &spec.MCPStdioConfigOverlay{}
		}
		out.Stdio.Env = mergeStringMap(out.Stdio.Env, src.Stdio.Env)
		out.Stdio.SecretEnvRefs = mergeStringMap(out.Stdio.SecretEnvRefs, src.Stdio.SecretEnvRefs)
	}
	if src.StreamableHTTP != nil {
		if out.StreamableHTTP == nil {
			out.StreamableHTTP = &spec.MCPStreamableHTTPConfigOverlay{}
		}
		if src.StreamableHTTP.URL != nil {
			out.StreamableHTTP.URL = src.StreamableHTTP.URL
		}
		if src.StreamableHTTP.TimeoutMS != nil {
			out.StreamableHTTP.TimeoutMS = src.StreamableHTTP.TimeoutMS
		}
		if src.StreamableHTTP.ClientCredentialRef != nil {
			out.StreamableHTTP.ClientCredentialRef = src.StreamableHTTP.ClientCredentialRef
		}
		if src.StreamableHTTP.ClientIDMetadataDocumentURL != nil {
			out.StreamableHTTP.ClientIDMetadataDocumentURL = src.StreamableHTTP.ClientIDMetadataDocumentURL
		}
		out.StreamableHTTP.Headers = mergeStringMap(out.StreamableHTTP.Headers, src.StreamableHTTP.Headers)
		out.StreamableHTTP.SecretHeaderRefs = mergeStringMap(
			out.StreamableHTTP.SecretHeaderRefs,
			src.StreamableHTTP.SecretHeaderRefs,
		)
	}
	return out
}

func applyServerOverlay(
	cfg spec.MCPServerConfig,
	ov spec.MCPBuiltInServerOverlay,
) (spec.MCPServerConfig, error) {
	out := cloneServerConfig(cfg)

	if ov.Stdio != nil {
		if out.Transport != spec.MCPTransportStdio {
			return spec.MCPServerConfig{}, fmt.Errorf(
				"%w: stdio overlay on %s server",
				spec.ErrMCPInvalidRequest,
				out.Transport,
			)
		}
		if out.Stdio == nil {
			out.Stdio = &spec.MCPStdioConfig{}
		}
		out.Stdio.Env = mergeStringMap(out.Stdio.Env, ov.Stdio.Env)
		out.Stdio.SecretEnvRefs = mergeStringMap(out.Stdio.SecretEnvRefs, ov.Stdio.SecretEnvRefs)
	}

	if ov.StreamableHTTP != nil {
		if out.Transport != spec.MCPTransportStreamableHTTP {
			return spec.MCPServerConfig{}, fmt.Errorf(
				"%w: streamableHttp overlay on %s server",
				spec.ErrMCPInvalidRequest,
				out.Transport,
			)
		}
		if out.StreamableHTTP == nil {
			out.StreamableHTTP = &spec.MCPStreamableHTTPConfig{}
		}
		if ov.StreamableHTTP.URL != nil {
			out.StreamableHTTP.URL = *ov.StreamableHTTP.URL
		}
		if ov.StreamableHTTP.TimeoutMS != nil {
			out.StreamableHTTP.TimeoutMS = *ov.StreamableHTTP.TimeoutMS
		}
		if ov.StreamableHTTP.ClientCredentialRef != nil {
			out.StreamableHTTP.ClientCredentialRef = *ov.StreamableHTTP.ClientCredentialRef
		}
		if ov.StreamableHTTP.ClientIDMetadataDocumentURL != nil {
			out.StreamableHTTP.ClientIDMetadataDocumentURL = *ov.StreamableHTTP.ClientIDMetadataDocumentURL
		}
		out.StreamableHTTP.Headers = mergeStringMap(out.StreamableHTTP.Headers, ov.StreamableHTTP.Headers)
		out.StreamableHTTP.SecretHeaderRefs = mergeStringMap(
			out.StreamableHTTP.SecretHeaderRefs,
			ov.StreamableHTTP.SecretHeaderRefs,
		)
	}
	return out, nil
}

func mergeStringMap(dst, src map[string]string) map[string]string {
	if len(src) == 0 {
		return dst
	}
	if dst == nil {
		dst = make(map[string]string, len(src))
	}
	maps.Copy(dst, src)
	return dst
}

func builtInServerKey(bundleID bundleitemutils.BundleID, serverID spec.MCPServerID) builtInMCPServerID {
	return builtInMCPServerID(fmt.Sprintf("%s::%s", bundleID, serverID))
}
