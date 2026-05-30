package store

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
	"github.com/flexigpt/mapstore-go"
	"github.com/flexigpt/mapstore-go/jsonencdec"
)

const builtInSnapshotMaxAge = 24 * time.Hour

type storeSchema struct {
	SchemaVersion      string                                                                 `json:"schemaVersion"`
	Bundles            map[bundleitemutils.BundleID]spec.MCPBundle                            `json:"bundles"`
	Servers            map[bundleitemutils.BundleID]map[spec.MCPServerID]spec.MCPServerConfig `json:"servers"`
	LastKnownSnapshots map[spec.MCPServerID]spec.MCPDiscoverySnapshot                         `json:"lastKnownSnapshots,omitempty"`
}

type Store struct {
	baseDir     string
	file        *mapstore.MapFileStore
	builtinData *BuiltInData
	mu          sync.RWMutex
}

func NewMCPStore(ctx context.Context, baseDir string) (*Store, error) {
	if baseDir == "" {
		return nil, fmt.Errorf("%w: baseDir is empty", spec.ErrMCPInvalidRequest)
	}
	baseDir = filepath.Clean(baseDir)
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, err
	}

	builtInData, err := NewBuiltInData(ctx, baseDir, builtInSnapshotMaxAge)
	if err != nil {
		return nil, err
	}

	def, err := jsonencdec.StructWithJSONTagsToMap(storeSchema{
		SchemaVersion:      spec.MCPSchemaVersion,
		Bundles:            map[bundleitemutils.BundleID]spec.MCPBundle{},
		Servers:            map[bundleitemutils.BundleID]map[spec.MCPServerID]spec.MCPServerConfig{},
		LastKnownSnapshots: map[spec.MCPServerID]spec.MCPDiscoverySnapshot{},
	})
	if err != nil {
		_ = builtInData.Close()
		return nil, err
	}

	file, err := mapstore.NewMapFileStore(
		filepath.Join(baseDir, spec.MCPStoreFileName),
		def,
		jsonencdec.JSONEncoderDecoder{},
		mapstore.WithCreateIfNotExists(true),
		mapstore.WithFileAutoFlush(true),
		mapstore.WithFileLogger(slog.Default()),
	)
	if err != nil {
		return nil, err
	}

	st := &Store{baseDir: baseDir, file: file, builtinData: builtInData}
	if err := st.ensureBaseBundleHydrated(); err != nil {
		_ = file.Close()
		_ = builtInData.Close()
		return nil, err
	}
	return st, nil
}

func (s *Store) Close() error {
	if s == nil || s.file == nil {
		return nil
	}
	if s.builtinData != nil {
		_ = s.builtinData.Close()
	}
	return s.file.Close()
}

func (s *Store) PutMCPBundle(
	ctx context.Context,
	req *spec.PutMCPBundleRequest,
) (*spec.PutMCPBundleResponse, error) {
	if req == nil || req.Body == nil || req.BundleID == "" {
		return nil, fmt.Errorf("%w: bundleID and body required", spec.ErrMCPInvalidRequest)
	}
	if isBaseMCPBundleID(req.BundleID) {
		return nil, fmt.Errorf("%w: bundleID %q", spec.ErrMCPReservedBundleReadOnly, req.BundleID)
	}
	if isBaseMCPBundleSlug(req.Body.Slug) {
		return nil, fmt.Errorf("%w: bundle slug %q is reserved", spec.ErrMCPConflict, req.Body.Slug)
	}
	if s.builtinData != nil {
		if _, err := s.builtinData.GetBuiltInBundle(ctx, req.BundleID); err == nil {
			return nil, fmt.Errorf("%w: bundleID %q", spec.ErrMCPBuiltInReadOnly, req.BundleID)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sc, err := s.readAll(false)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	created := now
	if old, ok := sc.Bundles[req.BundleID]; ok && !old.CreatedAt.IsZero() {
		created = old.CreatedAt
	}

	b := spec.MCPBundle{
		SchemaVersion: spec.MCPSchemaVersion,
		ID:            req.BundleID,
		Slug:          req.Body.Slug,
		DisplayName:   req.Body.DisplayName,
		Description:   req.Body.Description,
		IsEnabled:     req.Body.IsEnabled,
		CreatedAt:     created,
		ModifiedAt:    now,
		IsBuiltIn:     false,
	}
	if err := validateBundle(&b); err != nil {
		return nil, fmt.Errorf("%w: %w", spec.ErrMCPInvalidRequest, err)
	}

	sc.Bundles[req.BundleID] = cloneBundle(b)
	if sc.Servers[req.BundleID] == nil {
		sc.Servers[req.BundleID] = map[spec.MCPServerID]spec.MCPServerConfig{}
	}
	if err := s.writeAll(sc); err != nil {
		return nil, err
	}
	return &spec.PutMCPBundleResponse{}, nil
}

func (s *Store) PatchMCPBundle(
	ctx context.Context,
	req *spec.PatchMCPBundleRequest,
) (*spec.PatchMCPBundleResponse, error) {
	if req == nil || req.Body == nil || req.BundleID == "" {
		return nil, fmt.Errorf("%w: bundleID and body required", spec.ErrMCPInvalidRequest)
	}
	if isBaseMCPBundleID(req.BundleID) {
		return nil, fmt.Errorf("%w: bundleID %q", spec.ErrMCPReservedBundleReadOnly, req.BundleID)
	}

	if s.builtinData != nil {
		if _, err := s.builtinData.GetBuiltInBundle(ctx, req.BundleID); err == nil {
			if _, err := s.builtinData.SetBundleEnabled(ctx, req.BundleID, req.Body.IsEnabled); err != nil {
				return nil, err
			}
			return &spec.PatchMCPBundleResponse{}, nil
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sc, err := s.readAll(false)
	if err != nil {
		return nil, err
	}
	b, ok := sc.Bundles[req.BundleID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", spec.ErrMCPBundleNotFound, req.BundleID)
	}
	if isBundleSoftDeleted(b) {
		return nil, fmt.Errorf("%w: %s", spec.ErrMCPBundleDeleting, req.BundleID)
	}

	b.IsEnabled = req.Body.IsEnabled
	b.ModifiedAt = time.Now().UTC()
	if err := validateBundle(&b); err != nil {
		return nil, fmt.Errorf("%w: %w", spec.ErrMCPInvalidRequest, err)
	}
	sc.Bundles[req.BundleID] = cloneBundle(b)
	if err := s.writeAll(sc); err != nil {
		return nil, err
	}
	return &spec.PatchMCPBundleResponse{}, nil
}

func (s *Store) DeleteMCPBundle(
	ctx context.Context,
	req *spec.DeleteMCPBundleRequest,
) (*spec.DeleteMCPBundleResponse, error) {
	if req == nil || req.BundleID == "" {
		return nil, fmt.Errorf("%w: bundleID required", spec.ErrMCPInvalidRequest)
	}
	if isBaseMCPBundleID(req.BundleID) {
		return nil, fmt.Errorf("%w: bundleID %q", spec.ErrMCPReservedBundleReadOnly, req.BundleID)
	}
	if s.builtinData != nil {
		if _, err := s.builtinData.GetBuiltInBundle(ctx, req.BundleID); err == nil {
			return nil, fmt.Errorf("%w: bundleID %q", spec.ErrMCPBuiltInReadOnly, req.BundleID)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sc, err := s.readAll(false)
	if err != nil {
		return nil, err
	}
	b, ok := sc.Bundles[req.BundleID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", spec.ErrMCPBundleNotFound, req.BundleID)
	}
	if isBundleSoftDeleted(b) {
		return nil, fmt.Errorf("%w: %s", spec.ErrMCPBundleDeleting, req.BundleID)
	}
	for _, cfg := range sc.Servers[req.BundleID] {
		if !isServerSoftDeleted(&cfg) {
			return nil, fmt.Errorf("%w: %s", spec.ErrMCPBundleNotEmpty, req.BundleID)
		}
	}

	now := time.Now().UTC()
	b.IsEnabled = false
	b.SoftDeletedAt = &now
	b.ModifiedAt = now
	sc.Bundles[req.BundleID] = cloneBundle(b)
	if err := s.writeAll(sc); err != nil {
		return nil, err
	}
	return &spec.DeleteMCPBundleResponse{}, nil
}

func (s *Store) ListMCPBundles(
	ctx context.Context,
	req *spec.ListMCPBundlesRequest,
) (*spec.ListMCPBundlesResponse, error) {
	pageSize, cursorAt, cursorID, includeDisabled, filterIDs, err := parseBundleListPage(req)
	if err != nil {
		return nil, err
	}

	want := map[bundleitemutils.BundleID]struct{}{}
	for _, id := range filterIDs {
		want[id] = struct{}{}
	}

	items := make([]spec.MCPBundle, 0)
	if s.builtinData != nil {
		bundles, _, _ := s.builtinData.ListBuiltInData(ctx)
		for _, b := range bundles {
			items = append(items, cloneBundle(b))
		}
	}

	s.mu.RLock()
	sc, err := s.readAll(false)
	s.mu.RUnlock()
	if err != nil {
		return nil, err
	}
	for _, b := range sc.Bundles {
		if isBundleSoftDeleted(b) {
			continue
		}
		items = append(items, cloneBundle(b))
	}

	items = filterBundles(items, want, includeDisabled)
	sortBundles(items)

	start := bundleCursorStart(items, cursorAt, cursorID)
	end := min(start+pageSize, len(items))
	next := nextBundlePageToken(items, end, pageSize, filterIDs, includeDisabled)

	return &spec.ListMCPBundlesResponse{
		Body: &spec.ListMCPBundlesResponseBody{
			Bundles:       items[start:end],
			NextPageToken: next,
		},
	}, nil
}

func (s *Store) PutMCPServer(
	ctx context.Context,
	req *spec.PutMCPServerRequest,
) (*spec.PutMCPServerResponse, error) {
	if req == nil || req.Body == nil || req.ServerID == "" {
		return nil, fmt.Errorf("%w: serverID and body required", spec.ErrMCPInvalidRequest)
	}
	if req.BundleID == "" {
		return nil, fmt.Errorf("%w: bundleID required", spec.ErrMCPInvalidRequest)
	}

	bundle, isBuiltInBundle, err := s.getAnyBundle(ctx, req.BundleID)
	if err != nil {
		return nil, err
	}
	if isBuiltInBundle {
		return nil, fmt.Errorf("%w: bundleID %q", spec.ErrMCPBuiltInReadOnly, req.BundleID)
	}
	if !bundle.IsEnabled {
		return nil, fmt.Errorf("%w: %s", spec.ErrMCPBundleDisabled, req.BundleID)
	}
	if s.builtinData != nil {
		if _, err := s.builtinData.GetBuiltInServer(ctx, "", req.ServerID); err == nil {
			return nil, fmt.Errorf("%w: serverID %q belongs to a built-in server", spec.ErrMCPConflict, req.ServerID)
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	sc, err := s.readAll(false)
	if err != nil {
		return nil, err
	}

	if sc.Servers[req.BundleID] == nil {
		sc.Servers[req.BundleID] = map[spec.MCPServerID]spec.MCPServerConfig{}
	}

	if existingBundle, ok := findUserServerBundle(sc, req.ServerID); ok && existingBundle != req.BundleID {
		return nil, fmt.Errorf(
			"%w: serverID %q already exists in bundle %q",
			spec.ErrMCPConflict,
			req.ServerID,
			existingBundle,
		)
	}

	now := time.Now().UTC()
	created := now
	if ex, ok := sc.Servers[req.BundleID][req.ServerID]; ok && !ex.CreatedAt.IsZero() {
		created = ex.CreatedAt
	}

	policy := spec.DefaultMCPServerPolicy()
	if req.Body.DefaultPolicy != nil {
		policy = *req.Body.DefaultPolicy
	}

	cfg := spec.MCPServerConfig{
		SchemaVersion:  spec.MCPSchemaVersion,
		BundleID:       req.BundleID,
		ID:             req.ServerID,
		DisplayName:    req.Body.DisplayName,
		Enabled:        req.Body.Enabled,
		Transport:      req.Body.Transport,
		Stdio:          req.Body.Stdio,
		StreamableHTTP: req.Body.StreamableHTTP,
		Availability:   req.Body.Availability,
		TrustLevel:     req.Body.TrustLevel,
		DefaultPolicy:  policy,
		ToolPolicies:   req.Body.ToolPolicies,
		AppsPolicy:     req.Body.AppsPolicy,

		IsBuiltIn:  false,
		CreatedAt:  created,
		ModifiedAt: now,
	}

	if err := validateServerConfig(&cfg); err != nil {
		return nil, fmt.Errorf("%w: %w", spec.ErrMCPInvalidRequest, err)
	}

	if old, ok := sc.Servers[req.BundleID][req.ServerID]; ok && serverConnectionMaterialChanged(old, cfg) {
		// A changed endpoint/process invalidates the live/discovered view.
		// Runtime disconnect is handled by the wrapper.
		delete(sc.LastKnownSnapshots, req.ServerID)
	}

	sc.Servers[req.BundleID][req.ServerID] = cloneServerConfig(cfg)
	if err := s.writeAll(sc); err != nil {
		return nil, err
	}
	return &spec.PutMCPServerResponse{}, nil
}

func (s *Store) GetMCPServer(
	ctx context.Context,
	req *spec.GetMCPServerRequest,
) (*spec.GetMCPServerResponse, error) {
	if req == nil || req.ServerID == "" {
		return nil, fmt.Errorf("%w: serverID required", spec.ErrMCPInvalidRequest)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	cfg, _, bundle, ok, err := s.getAnyServerLocked(ctx, req.BundleID, req.ServerID, req.IncludeDeleted)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("%w: %s", spec.ErrMCPServerNotFound, req.ServerID)
	}

	cfg = cloneServerConfig(cfg)
	cfg.Enabled = cfg.Enabled && bundle.IsEnabled
	return &spec.GetMCPServerResponse{Body: &cfg}, nil
}

func (s *Store) ListMCPServers(
	ctx context.Context,
	req *spec.ListMCPServersRequest,
) (*spec.ListMCPServersResponse, error) {
	var (
		pageSize        = spec.DefaultMCPPageSize
		cursorAt        time.Time
		cursorID        spec.MCPServerID
		enabled         *bool
		includeDisabled bool
		filterIDs       []spec.MCPServerID
		filterBundleIDs []bundleitemutils.BundleID
	)

	if req != nil && req.PageToken != "" {
		tok, err := jsonutil.Base64JSONDecode[spec.MCPPageToken](req.PageToken)
		if err != nil {
			return nil, fmt.Errorf("%w: bad pageToken", spec.ErrMCPInvalidRequest)
		}
		pageSize = tok.PageSize
		if pageSize <= 0 || pageSize > spec.MaxMCPServerPageSize {
			pageSize = spec.DefaultMCPPageSize
		}
		enabled = tok.Enabled
		includeDisabled = tok.IncludeDisabled
		filterIDs = slices.Clone(tok.IDs)
		filterBundleIDs = slices.Clone(tok.BundleIDs)

		if tok.CursorAt != "" {
			cursorAt, err = time.Parse(time.RFC3339Nano, tok.CursorAt)
			if err != nil {
				return nil, fmt.Errorf("%w: bad cursor time", spec.ErrMCPInvalidRequest)
			}
			cursorID = tok.CursorID
		}
	} else if req != nil {
		if req.PageSize > 0 && req.PageSize <= spec.MaxMCPServerPageSize {
			pageSize = req.PageSize
		}
		enabled = req.Enabled
		includeDisabled = req.IncludeDisabled

		filterIDs = slices.Clone(req.ServerIDs)
		filterBundleIDs = slices.Clone(req.BundleIDs)

	}

	want := map[spec.MCPServerID]struct{}{}
	for _, id := range filterIDs {
		want[id] = struct{}{}
	}

	wantBundles := map[bundleitemutils.BundleID]struct{}{}
	for _, id := range filterBundleIDs {
		wantBundles[id] = struct{}{}
	}

	s.mu.RLock()
	sc, err := s.readAll(false)
	s.mu.RUnlock()
	if err != nil {
		return nil, err
	}

	items := make([]spec.MCPServerConfig, 0)
	add := func(bundle spec.MCPBundle, cfg spec.MCPServerConfig) {
		if isServerSoftDeleted(&cfg) || isBundleSoftDeleted(bundle) {
			return
		}
		if len(wantBundles) > 0 {
			if _, ok := wantBundles[cfg.BundleID]; !ok {
				return
			}
		}
		effectiveEnabled := cfg.Enabled && bundle.IsEnabled
		if !includeDisabled && !effectiveEnabled {
			return
		}
		if len(want) > 0 {
			if _, ok := want[cfg.ID]; !ok {
				return
			}
		}
		if enabled != nil && effectiveEnabled != *enabled {
			return
		}
		cfg = cloneServerConfig(cfg)
		cfg.Enabled = effectiveEnabled
		items = append(items, cloneServerConfig(cfg))
	}

	if s.builtinData != nil {
		bundles, servers, _ := s.builtinData.ListBuiltInData(ctx)
		for bid, sm := range servers {
			b := bundles[bid]
			for _, cfg := range sm {
				add(b, cfg)
			}
		}
	}
	for bid, sm := range sc.Servers {
		b := sc.Bundles[bid]
		for _, cfg := range sm {
			add(b, cfg)
		}
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].ModifiedAt.Equal(items[j].ModifiedAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].ModifiedAt.After(items[j].ModifiedAt)
	})

	start := 0
	if !cursorAt.IsZero() {
		start = sort.Search(len(items), func(i int) bool {
			it := items[i]
			if it.ModifiedAt.Before(cursorAt) {
				return true
			}
			return it.ModifiedAt.Equal(cursorAt) && it.ID > cursorID
		})
	}

	end := min(start+pageSize, len(items))
	var nextTok *string
	if end < len(items) {
		slices.Sort(filterIDs)
		next := jsonutil.Base64JSONEncode(spec.MCPPageToken{
			PageSize:        pageSize,
			CursorAt:        items[end-1].ModifiedAt.Format(time.RFC3339Nano),
			CursorID:        items[end-1].ID,
			Enabled:         enabled,
			IDs:             filterIDs,
			BundleIDs:       filterBundleIDs,
			IncludeDisabled: includeDisabled,
		})
		nextTok = &next
	}

	return &spec.ListMCPServersResponse{
		Body: &spec.ListMCPServersResponseBody{
			Servers:       items[start:end],
			NextPageToken: nextTok,
		},
	}, nil
}

func (s *Store) PatchMCPServerEnabled(
	ctx context.Context,
	req *spec.PatchMCPServerEnabledRequest,
) (*spec.PatchMCPServerEnabledResponse, error) {
	if req == nil || req.Body == nil || req.ServerID == "" {
		return nil, fmt.Errorf("%w: serverID and body required", spec.ErrMCPInvalidRequest)
	}

	if s.builtinData != nil {
		if _, err := s.builtinData.GetBuiltInServer(ctx, req.BundleID, req.ServerID); err == nil {
			if _, err := s.builtinData.SetServerEnabled(ctx, req.BundleID, req.ServerID, req.Body.Enabled); err != nil {
				return nil, err
			}
			return &spec.PatchMCPServerEnabledResponse{}, nil
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sc, err := s.readAll(false)
	if err != nil {
		return nil, err
	}
	bid, cfg, ok := findUserServer(sc, req.BundleID, req.ServerID)

	if !ok {
		return nil, fmt.Errorf("%w: %s", spec.ErrMCPServerNotFound, req.ServerID)
	}
	if isServerSoftDeleted(&cfg) {
		return nil, fmt.Errorf("%w: %s", spec.ErrMCPServerDeleting, req.ServerID)
	}
	cfg.BundleID = bid
	cfg.Enabled = req.Body.Enabled
	cfg.ModifiedAt = time.Now().UTC()
	if err := validateServerConfig(&cfg); err != nil {
		return nil, fmt.Errorf("%w: %w", spec.ErrMCPInvalidRequest, err)
	}
	sc.Servers[bid][req.ServerID] = cloneServerConfig(cfg)
	if err := s.writeAll(sc); err != nil {
		return nil, err
	}
	return &spec.PatchMCPServerEnabledResponse{}, nil
}

func (s *Store) PatchMCPServerPolicy(
	ctx context.Context,
	req *spec.PatchMCPServerPolicyRequest,
) (*spec.PatchMCPServerPolicyResponse, error) {
	if req == nil || req.Body == nil || req.ServerID == "" {
		return nil, fmt.Errorf("%w: serverID and body required", spec.ErrMCPInvalidRequest)
	}

	if s.builtinData != nil {
		if _, err := s.builtinData.GetBuiltInServer(ctx, req.BundleID, req.ServerID); err == nil {
			return nil, fmt.Errorf("%w: serverID %q", spec.ErrMCPBuiltInReadOnly, req.ServerID)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sc, err := s.readAll(false)
	if err != nil {
		return nil, err
	}
	bid, cfg, ok := findUserServer(sc, req.BundleID, req.ServerID)
	if !ok {
		return nil, fmt.Errorf("%w: %s", spec.ErrMCPServerNotFound, req.ServerID)
	}
	if isServerSoftDeleted(&cfg) {
		return nil, fmt.Errorf("%w: %s", spec.ErrMCPServerDeleting, req.ServerID)
	}

	if req.Body.DefaultPolicy != nil {
		cfg.DefaultPolicy = *req.Body.DefaultPolicy
	}
	if req.Body.ToolPolicies != nil {
		cfg.ToolPolicies = req.Body.ToolPolicies
	}
	if req.Body.AppsPolicy != nil {
		cfg.AppsPolicy = req.Body.AppsPolicy
	}
	cfg.ModifiedAt = time.Now().UTC()

	if err := validateServerConfig(&cfg); err != nil {
		return nil, fmt.Errorf("%w: %w", spec.ErrMCPInvalidRequest, err)
	}

	sc.Servers[bid][req.ServerID] = cloneServerConfig(cfg)

	if err := s.writeAll(sc); err != nil {
		return nil, err
	}
	return &spec.PatchMCPServerPolicyResponse{}, nil
}

func (s *Store) DeleteMCPServer(
	ctx context.Context,
	req *spec.DeleteMCPServerRequest,
) (*spec.DeleteMCPServerResponse, error) {
	if req == nil || req.ServerID == "" {
		return nil, fmt.Errorf("%w: serverID required", spec.ErrMCPInvalidRequest)
	}

	if s.builtinData != nil {
		if _, err := s.builtinData.GetBuiltInServer(ctx, req.BundleID, req.ServerID); err == nil {
			return nil, fmt.Errorf("%w: serverID %q", spec.ErrMCPBuiltInReadOnly, req.ServerID)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sc, err := s.readAll(false)
	if err != nil {
		return nil, err
	}
	bid, cfg, ok := findUserServer(sc, req.BundleID, req.ServerID)

	if !ok {
		return nil, fmt.Errorf("%w: %s", spec.ErrMCPServerNotFound, req.ServerID)
	}

	now := time.Now().UTC()
	cfg.Enabled = false
	cfg.SoftDeletedAt = &now
	cfg.ModifiedAt = now
	sc.Servers[bid][req.ServerID] = cloneServerConfig(cfg)
	delete(sc.LastKnownSnapshots, req.ServerID)

	if err := s.writeAll(sc); err != nil {
		return nil, err
	}
	return &spec.DeleteMCPServerResponse{}, nil
}

func (s *Store) SaveLastKnownSnapshot(ctx context.Context, snap spec.MCPDiscoverySnapshot) error {
	if snap.ServerID == "" {
		return fmt.Errorf("%w: serverID required", spec.ErrMCPInvalidRequest)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sc, err := s.readAll(false)
	if err != nil {
		return err
	}
	if _, _, ok := findUserServer(sc, snap.BundleID, snap.ServerID); !ok {
		if s.builtinData == nil {
			return fmt.Errorf("%w: %s", spec.ErrMCPServerNotFound, snap.ServerID)
		}
		if _, err := s.builtinData.GetBuiltInServer(ctx, snap.BundleID, snap.ServerID); err != nil {
			return fmt.Errorf("%w: %s", spec.ErrMCPServerNotFound, snap.ServerID)
		}
	}
	sc.LastKnownSnapshots[snap.ServerID] = cloneDiscoverySnapshot(snap)
	return s.writeAll(sc)
}

func (s *Store) GetLastKnownSnapshot(
	ctx context.Context,
	serverID spec.MCPServerID,
) (spec.MCPDiscoverySnapshot, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sc, err := s.readAll(false)
	if err != nil {
		return spec.MCPDiscoverySnapshot{}, false, err
	}
	snap, ok := sc.LastKnownSnapshots[serverID]
	return cloneDiscoverySnapshot(snap), ok, nil
}

func (s *Store) readAll(force bool) (storeSchema, error) {
	raw, err := s.file.GetAll(force)
	if err != nil {
		return storeSchema{}, err
	}

	var sc storeSchema
	if err := jsonencdec.MapToStructWithJSONTags(raw, &sc); err != nil {
		return storeSchema{}, err
	}
	if sc.SchemaVersion == "" {
		sc.SchemaVersion = spec.MCPSchemaVersion
	}
	if sc.SchemaVersion != spec.MCPSchemaVersion {
		return storeSchema{}, fmt.Errorf("mcp store schemaVersion %q != %q", sc.SchemaVersion, spec.MCPSchemaVersion)
	}
	if sc.Bundles == nil {
		sc.Bundles = map[bundleitemutils.BundleID]spec.MCPBundle{}
	}

	if sc.Servers == nil {
		sc.Servers = map[bundleitemutils.BundleID]map[spec.MCPServerID]spec.MCPServerConfig{}
	}
	if sc.LastKnownSnapshots == nil {
		sc.LastKnownSnapshots = map[spec.MCPServerID]spec.MCPDiscoverySnapshot{}
	}

	for id, b := range sc.Bundles {
		if b.ID != id {
			return storeSchema{}, fmt.Errorf("bundle key %q != bundle.id %q", id, b.ID)
		}
		if err := validateBundle(&b); err != nil {
			return storeSchema{}, fmt.Errorf("invalid mcp bundle %q: %w", id, err)
		}
		sc.Bundles[id] = cloneBundle(b)
		if sc.Servers[id] == nil {
			sc.Servers[id] = map[spec.MCPServerID]spec.MCPServerConfig{}
		}
	}

	seen := map[spec.MCPServerID]bundleitemutils.BundleID{}
	for bid, servers := range sc.Servers {
		if _, ok := sc.Bundles[bid]; !ok {
			return storeSchema{}, fmt.Errorf("servers contain unknown bundle %q", bid)
		}

		for id, cfg := range servers {
			if cfg.ID != id {
				return storeSchema{}, fmt.Errorf("server key %q != server.id %q", id, cfg.ID)
			}
			if cfg.BundleID != bid {
				return storeSchema{}, fmt.Errorf("server %q bundleID %q != parent %q", id, cfg.BundleID, bid)
			}
			if prev, dup := seen[id]; dup && prev != bid {
				return storeSchema{}, fmt.Errorf("server id %q appears in multiple bundles", id)
			}
			seen[id] = bid
			if err := validateServerConfig(&cfg); err != nil {
				return storeSchema{}, fmt.Errorf("invalid mcp server %q: %w", id, err)
			}
			sc.Servers[bid][id] = cloneServerConfig(cfg)
		}
	}
	return sc, nil
}

func (s *Store) writeAll(sc storeSchema) error {
	sc.SchemaVersion = spec.MCPSchemaVersion
	if sc.Bundles == nil {
		sc.Bundles = map[bundleitemutils.BundleID]spec.MCPBundle{}
	}
	if sc.Servers == nil {
		sc.Servers = map[bundleitemutils.BundleID]map[spec.MCPServerID]spec.MCPServerConfig{}
	}
	for bid := range sc.Bundles {
		if sc.Servers[bid] == nil {
			sc.Servers[bid] = map[spec.MCPServerID]spec.MCPServerConfig{}
		}
	}
	if sc.LastKnownSnapshots == nil {
		sc.LastKnownSnapshots = map[spec.MCPServerID]spec.MCPDiscoverySnapshot{}
	}

	mp, err := jsonencdec.StructWithJSONTagsToMap(sc)
	if err != nil {
		return err
	}
	return s.file.SetAll(mp)
}

func (s *Store) ensureBaseBundleHydrated() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sc, err := s.readAll(false)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	if b, ok := sc.Bundles[spec.BaseMCPBundleID]; ok {
		changed := false
		if b.SchemaVersion == "" {
			b.SchemaVersion = spec.MCPSchemaVersion
			changed = true
		}
		if b.Slug == "" {
			b.Slug = spec.BaseMCPBundleSlug
			changed = true
		}
		if b.DisplayName == "" {
			b.DisplayName = spec.BaseMCPBundleDisplayName
			changed = true
		}
		if b.Description == "" {
			b.Description = spec.BaseMCPBundleDescription
			changed = true
		}
		if b.CreatedAt.IsZero() {
			b.CreatedAt = now
			changed = true
		}
		if !b.IsEnabled {
			b.IsEnabled = true
			changed = true
		}
		if b.SoftDeletedAt != nil {
			b.SoftDeletedAt = nil
			changed = true
		}
		if b.IsBuiltIn {
			b.IsBuiltIn = false
			changed = true
		}
		if !changed {
			return nil
		}
		b.ModifiedAt = now
		sc.Bundles[spec.BaseMCPBundleID] = b
		return s.writeAll(sc)
	}

	sc.Bundles[spec.BaseMCPBundleID] = spec.MCPBundle{
		SchemaVersion: spec.MCPSchemaVersion,
		ID:            spec.BaseMCPBundleID,
		Slug:          spec.BaseMCPBundleSlug,
		DisplayName:   spec.BaseMCPBundleDisplayName,
		Description:   spec.BaseMCPBundleDescription,
		IsEnabled:     true,
		CreatedAt:     now,
		ModifiedAt:    now,
	}
	sc.Servers[spec.BaseMCPBundleID] = map[spec.MCPServerID]spec.MCPServerConfig{}
	return s.writeAll(sc)
}

func (s *Store) getAnyBundle(ctx context.Context, id bundleitemutils.BundleID) (spec.MCPBundle, bool, error) {
	if id == "" {
		return spec.MCPBundle{}, false, fmt.Errorf("%w: bundleID required", spec.ErrMCPInvalidRequest)
	}
	if s.builtinData != nil {
		if b, err := s.builtinData.GetBuiltInBundle(ctx, id); err == nil {
			return b, true, nil
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	sc, err := s.readAll(false)
	if err != nil {
		return spec.MCPBundle{}, false, err
	}
	b, ok := sc.Bundles[id]
	if !ok {
		return spec.MCPBundle{}, false, fmt.Errorf("%w: %s", spec.ErrMCPBundleNotFound, id)
	}
	if isBundleSoftDeleted(b) {
		return spec.MCPBundle{}, false, fmt.Errorf("%w: %s", spec.ErrMCPBundleDeleting, id)
	}
	return cloneBundle(b), false, nil
}

func (s *Store) getAnyServerLocked(
	ctx context.Context,
	bundleID bundleitemutils.BundleID,
	serverID spec.MCPServerID,
	includeDeleted bool,
) (cfg spec.MCPServerConfig, builtIn bool, bundle spec.MCPBundle, ok bool, err error) {
	if s.builtinData != nil {
		if cfg, err = s.builtinData.GetBuiltInServer(ctx, bundleID, serverID); err == nil {
			bundle, err = s.builtinData.GetBuiltInBundle(ctx, cfg.BundleID)
			if err != nil {
				return spec.MCPServerConfig{}, false, spec.MCPBundle{}, false, err
			}
			return cfg, true, bundle, true, nil
		}
	}

	sc, err := s.readAll(false)
	if err != nil {
		return spec.MCPServerConfig{}, false, spec.MCPBundle{}, false, err
	}
	bid, cfg, ok := findUserServer(sc, bundleID, serverID)
	if !ok {
		return spec.MCPServerConfig{}, false, spec.MCPBundle{}, false, nil
	}
	if isServerSoftDeleted(&cfg) && !includeDeleted {
		return spec.MCPServerConfig{}, false, spec.MCPBundle{}, false, fmt.Errorf(
			"%w: %s",
			spec.ErrMCPServerDeleting,
			serverID,
		)
	}
	bundle = sc.Bundles[bid]
	if isBundleSoftDeleted(bundle) && !includeDeleted {
		return spec.MCPServerConfig{}, false, spec.MCPBundle{}, false, fmt.Errorf(
			"%w: %s",
			spec.ErrMCPBundleDeleting,
			bid,
		)
	}
	return cfg, false, bundle, true, nil
}

func findUserServer(
	sc storeSchema,
	bundleID bundleitemutils.BundleID,
	serverID spec.MCPServerID,
) (bundleitemutils.BundleID, spec.MCPServerConfig, bool) {
	if bundleID != "" {
		if cfg, ok := sc.Servers[bundleID][serverID]; ok {
			return bundleID, cfg, true
		}
		return "", spec.MCPServerConfig{}, false
	}
	for bid, servers := range sc.Servers {
		if cfg, ok := servers[serverID]; ok {
			return bid, cfg, true
		}
	}
	return "", spec.MCPServerConfig{}, false
}

func findUserServerBundle(sc storeSchema, serverID spec.MCPServerID) (bundleitemutils.BundleID, bool) {
	bid, _, ok := findUserServer(sc, "", serverID)
	return bid, ok
}

//nolint:gocritic // page results are large.
func parseBundleListPage(req *spec.ListMCPBundlesRequest) (
	pageSize int,
	cursorAt time.Time,
	cursorID bundleitemutils.BundleID,
	includeDisabled bool,
	filterIDs []bundleitemutils.BundleID,
	err error,
) {
	pageSize = spec.DefaultMCPPageSize
	if req != nil && req.PageToken != "" {
		tok, decErr := jsonutil.Base64JSONDecode[spec.MCPBundlePageToken](req.PageToken)
		if decErr != nil {
			err = fmt.Errorf("%w: bad pageToken", spec.ErrMCPInvalidRequest)
			return pageSize, cursorAt, cursorID, includeDisabled, filterIDs, err
		}
		pageSize = tok.PageSize
		if pageSize <= 0 || pageSize > spec.MaxMCPServerPageSize {
			pageSize = spec.DefaultMCPPageSize
		}
		includeDisabled = tok.IncludeDisabled
		filterIDs = slices.Clone(tok.BundleIDs)
		if tok.CursorMod != "" {
			cursorAt, err = time.Parse(time.RFC3339Nano, tok.CursorMod)
			cursorID = tok.CursorID
		}
		return pageSize, cursorAt, cursorID, includeDisabled, filterIDs, err
	}
	if req != nil {
		if req.PageSize > 0 && req.PageSize <= spec.MaxMCPServerPageSize {
			pageSize = req.PageSize
		}
		includeDisabled = req.IncludeDisabled
		filterIDs = slices.Clone(req.BundleIDs)
	}
	return pageSize, cursorAt, cursorID, includeDisabled, filterIDs, err
}

func filterBundles(
	items []spec.MCPBundle,
	want map[bundleitemutils.BundleID]struct{},
	includeDisabled bool,
) []spec.MCPBundle {
	out := items[:0]
	for _, b := range items {
		if len(want) > 0 {
			if _, ok := want[b.ID]; !ok {
				continue
			}
		}
		if !includeDisabled && !b.IsEnabled {
			continue
		}
		out = append(out, b)
	}
	return out
}

func sortBundles(items []spec.MCPBundle) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].ModifiedAt.Equal(items[j].ModifiedAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].ModifiedAt.After(items[j].ModifiedAt)
	})
}

func bundleCursorStart(
	items []spec.MCPBundle,
	cursorAt time.Time,
	cursorID bundleitemutils.BundleID,
) int {
	if cursorAt.IsZero() && cursorID == "" {
		return 0
	}
	return sort.Search(len(items), func(i int) bool {
		it := items[i]
		if it.ModifiedAt.Before(cursorAt) {
			return true
		}
		return it.ModifiedAt.Equal(cursorAt) && it.ID > cursorID
	})
}

func nextBundlePageToken(
	items []spec.MCPBundle,
	end int,
	pageSize int,
	filterIDs []bundleitemutils.BundleID,
	includeDisabled bool,
) *string {
	if end >= len(items) {
		return nil
	}
	slices.Sort(filterIDs)
	next := jsonutil.Base64JSONEncode(spec.MCPBundlePageToken{
		PageSize:        pageSize,
		CursorMod:       items[end-1].ModifiedAt.Format(time.RFC3339Nano),
		CursorID:        items[end-1].ID,
		BundleIDs:       filterIDs,
		IncludeDisabled: includeDisabled,
	})
	return &next
}

func isBaseMCPBundleID(id bundleitemutils.BundleID) bool {
	return id == spec.BaseMCPBundleID
}

func isBaseMCPBundleSlug(slug bundleitemutils.BundleSlug) bool {
	return slug == spec.BaseMCPBundleSlug
}

func serverConnectionMaterialChanged(a, b spec.MCPServerConfig) bool {
	return a.Transport != b.Transport ||
		!reflect.DeepEqual(a.Stdio, b.Stdio) ||
		!reflect.DeepEqual(a.StreamableHTTP, b.StreamableHTTP)
}
