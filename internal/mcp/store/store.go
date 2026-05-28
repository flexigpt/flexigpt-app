package store

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
	"github.com/flexigpt/mapstore-go"
	"github.com/flexigpt/mapstore-go/jsonencdec"
)

type storeSchema struct {
	SchemaVersion      string                                         `json:"schemaVersion"`
	Servers            map[spec.MCPServerID]spec.MCPServerConfig      `json:"servers"`
	LastKnownSnapshots map[spec.MCPServerID]spec.MCPDiscoverySnapshot `json:"lastKnownSnapshots,omitempty"`
	AuthStatuses       map[spec.MCPServerID]spec.MCPAuthStatus        `json:"authStatuses,omitempty"`
}

type Store struct {
	baseDir string
	file    *mapstore.MapFileStore
	mu      sync.RWMutex
}

func NewStore(baseDir string) (*Store, error) {
	if baseDir == "" {
		return nil, fmt.Errorf("%w: baseDir is empty", spec.ErrMCPInvalidRequest)
	}
	baseDir = filepath.Clean(baseDir)
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, err
	}

	def, err := jsonencdec.StructWithJSONTagsToMap(storeSchema{
		SchemaVersion:      spec.MCPSchemaVersion,
		Servers:            map[spec.MCPServerID]spec.MCPServerConfig{},
		LastKnownSnapshots: map[spec.MCPServerID]spec.MCPDiscoverySnapshot{},
		AuthStatuses:       map[spec.MCPServerID]spec.MCPAuthStatus{},
	})
	if err != nil {
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

	return &Store{baseDir: baseDir, file: file}, nil
}

func (s *Store) Close() error {
	if s == nil || s.file == nil {
		return nil
	}
	return s.file.Close()
}

func (s *Store) PutMCPServer(
	ctx context.Context,
	req *spec.PutMCPServerRequest,
) (*spec.PutMCPServerResponse, error) {
	if req == nil || req.Body == nil || req.ServerID == "" {
		return nil, fmt.Errorf("%w: serverID and body required", spec.ErrMCPInvalidRequest)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sc, err := s.readAll(false)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	created := now
	if ex, ok := sc.Servers[req.ServerID]; ok && !ex.CreatedAt.IsZero() {
		created = ex.CreatedAt
	}

	policy := spec.DefaultMCPServerPolicy()
	if req.Body.DefaultPolicy != nil {
		policy = *req.Body.DefaultPolicy
	}

	cfg := spec.MCPServerConfig{
		SchemaVersion:  spec.MCPSchemaVersion,
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
		AuthRef:        req.Body.AuthRef,
		CreatedAt:      created,
		ModifiedAt:     now,
	}

	if err := validateServerConfig(&cfg); err != nil {
		return nil, fmt.Errorf("%w: %w", spec.ErrMCPInvalidRequest, err)
	}

	sc.Servers[req.ServerID] = cloneServerConfig(cfg)
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

	sc, err := s.readAll(false)
	if err != nil {
		return nil, err
	}
	cfg, ok := sc.Servers[req.ServerID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", spec.ErrMCPServerNotFound, req.ServerID)
	}
	if isSoftDeleted(&cfg) && !req.IncludeDeleted {
		return nil, fmt.Errorf("%w: %s", spec.ErrMCPServerDeleting, req.ServerID)
	}
	cfg = cloneServerConfig(cfg)
	return &spec.GetMCPServerResponse{Body: &cfg}, nil
}

func (s *Store) ListMCPServers(
	ctx context.Context,
	req *spec.ListMCPServersRequest,
) (*spec.ListMCPServersResponse, error) {
	var (
		pageSize  = spec.DefaultMCPPageSize
		cursorAt  time.Time
		cursorID  spec.MCPServerID
		enabled   *bool
		filterIDs []spec.MCPServerID
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
		filterIDs = slices.Clone(tok.IDs)
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
		filterIDs = slices.Clone(req.ServerIDs)
	}

	want := map[spec.MCPServerID]struct{}{}
	for _, id := range filterIDs {
		want[id] = struct{}{}
	}

	s.mu.RLock()
	sc, err := s.readAll(false)
	s.mu.RUnlock()
	if err != nil {
		return nil, err
	}

	items := make([]spec.MCPServerConfig, 0, len(sc.Servers))
	for _, cfg := range sc.Servers {
		if isSoftDeleted(&cfg) {
			continue
		}
		if len(want) > 0 {
			if _, ok := want[cfg.ID]; !ok {
				continue
			}
		}
		if enabled != nil && cfg.Enabled != *enabled {
			continue
		}
		items = append(items, cloneServerConfig(cfg))
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
			PageSize: pageSize,
			CursorAt: items[end-1].ModifiedAt.Format(time.RFC3339Nano),
			CursorID: items[end-1].ID,
			Enabled:  enabled,
			IDs:      filterIDs,
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

	s.mu.Lock()
	defer s.mu.Unlock()

	sc, err := s.readAll(false)
	if err != nil {
		return nil, err
	}
	cfg, ok := sc.Servers[req.ServerID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", spec.ErrMCPServerNotFound, req.ServerID)
	}
	if isSoftDeleted(&cfg) {
		return nil, fmt.Errorf("%w: %s", spec.ErrMCPServerDeleting, req.ServerID)
	}

	cfg.Enabled = req.Body.Enabled
	cfg.ModifiedAt = time.Now().UTC()
	if err := validateServerConfig(&cfg); err != nil {
		return nil, fmt.Errorf("%w: %w", spec.ErrMCPInvalidRequest, err)
	}
	sc.Servers[req.ServerID] = cloneServerConfig(cfg)
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

	s.mu.Lock()
	defer s.mu.Unlock()

	sc, err := s.readAll(false)
	if err != nil {
		return nil, err
	}
	cfg, ok := sc.Servers[req.ServerID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", spec.ErrMCPServerNotFound, req.ServerID)
	}
	if isSoftDeleted(&cfg) {
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

	sc.Servers[req.ServerID] = cloneServerConfig(cfg)
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

	s.mu.Lock()
	defer s.mu.Unlock()

	sc, err := s.readAll(false)
	if err != nil {
		return nil, err
	}
	cfg, ok := sc.Servers[req.ServerID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", spec.ErrMCPServerNotFound, req.ServerID)
	}

	now := time.Now().UTC()
	cfg.Enabled = false
	cfg.SoftDeletedAt = &now
	cfg.ModifiedAt = now
	sc.Servers[req.ServerID] = cloneServerConfig(cfg)
	delete(sc.LastKnownSnapshots, req.ServerID)
	delete(sc.AuthStatuses, req.ServerID)

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
	if _, ok := sc.Servers[snap.ServerID]; !ok {
		return fmt.Errorf("%w: %s", spec.ErrMCPServerNotFound, snap.ServerID)
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

func (s *Store) SaveAuthStatus(ctx context.Context, st spec.MCPAuthStatus) error {
	if st.ServerID == "" {
		return fmt.Errorf("%w: serverID required", spec.ErrMCPInvalidRequest)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sc, err := s.readAll(false)
	if err != nil {
		return err
	}
	sc.AuthStatuses[st.ServerID] = st
	return s.writeAll(sc)
}

func (s *Store) GetAuthStatus(ctx context.Context, serverID spec.MCPServerID) (spec.MCPAuthStatus, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sc, err := s.readAll(false)
	if err != nil {
		return spec.MCPAuthStatus{}, false, err
	}
	st, ok := sc.AuthStatuses[serverID]
	return st, ok, nil
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
	if sc.Servers == nil {
		sc.Servers = map[spec.MCPServerID]spec.MCPServerConfig{}
	}
	if sc.LastKnownSnapshots == nil {
		sc.LastKnownSnapshots = map[spec.MCPServerID]spec.MCPDiscoverySnapshot{}
	}
	if sc.AuthStatuses == nil {
		sc.AuthStatuses = map[spec.MCPServerID]spec.MCPAuthStatus{}
	}

	for id, cfg := range sc.Servers {
		if cfg.ID != id {
			return storeSchema{}, fmt.Errorf("server key %q != server.id %q", id, cfg.ID)
		}
		if err := validateServerConfig(&cfg); err != nil {
			return storeSchema{}, fmt.Errorf("invalid mcp server %q: %w", id, err)
		}
		sc.Servers[id] = cloneServerConfig(cfg)
	}
	return sc, nil
}

func (s *Store) writeAll(sc storeSchema) error {
	sc.SchemaVersion = spec.MCPSchemaVersion
	if sc.Servers == nil {
		sc.Servers = map[spec.MCPServerID]spec.MCPServerConfig{}
	}
	if sc.LastKnownSnapshots == nil {
		sc.LastKnownSnapshots = map[spec.MCPServerID]spec.MCPDiscoverySnapshot{}
	}
	if sc.AuthStatuses == nil {
		sc.AuthStatuses = map[spec.MCPServerID]spec.MCPAuthStatus{}
	}

	mp, err := jsonencdec.StructWithJSONTagsToMap(sc)
	if err != nil {
		return err
	}
	return s.file.SetAll(mp)
}
