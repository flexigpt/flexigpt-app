package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
	"github.com/flexigpt/flexigpt-app/internal/mcp/store"
)

type ClientFactory interface {
	Connect(
		ctx context.Context,
		cfg spec.MCPServerConfig,
		resolved ResolvedTransportAuth,
		events ClientNotificationSink,
	) (ClientSession, error)
}

type ClientSession interface {
	Close(ctx context.Context) error
	Ping(ctx context.Context) error
	Discover(
		ctx context.Context,
		serverID spec.MCPServerID,
		policy spec.MCPServerPolicy,
		trustLevel spec.MCPTrustLevel,
	) (spec.MCPDiscoverySnapshot, error)

	CallTool(ctx context.Context, toolName string, args map[string]any) (*spec.InvokeMCPToolResponseBody, error)
	ReadResource(ctx context.Context, uri string) (*spec.MCPReadResourceResponseBody, error)
	GetPrompt(ctx context.Context, name string, args map[string]string) (*spec.MCPGetPromptResponseBody, error)
	Complete(ctx context.Context, req spec.MCPCompleteArgumentRequestBody) (*spec.MCPCompletionResult, error)
}

type sessionState struct {
	serverID        spec.MCPServerID
	status          spec.MCPServerStatus
	client          ClientSession
	snapshot        spec.MCPDiscoverySnapshot
	lastError       string
	lastConnectedAt time.Time
	lastSyncedAt    time.Time
}

type RuntimeManager struct {
	store   *store.Store
	auth    *AuthManager
	factory ClientFactory

	mu                        sync.RWMutex
	sessions                  map[spec.MCPServerID]*sessionState
	generations               map[spec.MCPServerID]uint64
	notificationRefreshTimers map[spec.MCPServerID]*time.Timer
	shuttingDown              bool
}

func NewRuntimeManager(st *store.Store, authMgr *AuthManager, factory ClientFactory) *RuntimeManager {
	return &RuntimeManager{
		store:                     st,
		auth:                      authMgr,
		factory:                   factory,
		sessions:                  map[spec.MCPServerID]*sessionState{},
		generations:               map[spec.MCPServerID]uint64{},
		notificationRefreshTimers: map[spec.MCPServerID]*time.Timer{},
	}
}

func (m *RuntimeManager) Close(ctx context.Context) error {
	m.mu.Lock()
	m.shuttingDown = true
	states := make([]*sessionState, 0, len(m.sessions))
	for _, st := range m.sessions {
		states = append(states, st)
	}
	timers := make([]*time.Timer, 0, len(m.notificationRefreshTimers))
	for _, timer := range m.notificationRefreshTimers {
		timers = append(timers, timer)
	}
	m.sessions = map[spec.MCPServerID]*sessionState{}
	m.notificationRefreshTimers = map[spec.MCPServerID]*time.Timer{}

	for id := range m.generations {
		m.generations[id]++
	}
	m.mu.Unlock()
	for _, timer := range timers {
		timer.Stop()
	}
	var first error
	for _, st := range states {
		if st.client != nil {
			if err := st.client.Close(ctx); err != nil && first == nil {
				first = err
			}
		}
	}
	return first
}

func (m *RuntimeManager) Connect(
	ctx context.Context,
	req *spec.ConnectMCPServerRequest,
) (*spec.ConnectMCPServerResponse, error) {
	if req == nil || req.ServerID == "" {
		return nil, fmt.Errorf("%w: serverID required", spec.ErrMCPInvalidRequest)
	}

	m.mu.Lock()
	if m.shuttingDown {
		m.mu.Unlock()
		return nil, fmt.Errorf("%w: runtime is shutting down", spec.ErrMCPRuntimeNotReady)
	}
	state := m.getOrCreateLocked(req.ServerID)
	generation := m.bumpGenerationLocked(req.ServerID)

	state.status = spec.MCPServerStatusConnecting
	state.lastError = ""
	m.mu.Unlock()

	cfgResp, err := m.store.GetMCPServer(ctx, &spec.GetMCPServerRequest{ServerID: req.ServerID})
	if err != nil {
		m.setErrorIfCurrent(req.ServerID, generation, err)
		return nil, err
	}
	cfg := *cfgResp.Body
	if !cfg.Enabled {
		err := fmt.Errorf("%w: %s", spec.ErrMCPServerDisabled, cfg.ID)
		m.setErrorIfCurrent(req.ServerID, generation, err)
		return nil, err
	}

	resolved, err := m.auth.PrepareTransportAuth(ctx, cfg)
	_ = m.store.SaveAuthStatus(ctx, resolved.Status)
	if err != nil {
		m.setErrorIfCurrent(req.ServerID, generation, err)
		return nil, err
	}

	connectTimeout := time.Duration(spec.DefaultConnectTimeoutMS) * time.Millisecond
	if cfg.Transport == spec.MCPTransportStreamableHTTP && cfg.StreamableHTTP != nil &&
		cfg.StreamableHTTP.TimeoutMS > 0 {
		connectTimeout = time.Duration(cfg.StreamableHTTP.TimeoutMS) * time.Millisecond
	}
	if cfg.Transport == spec.MCPTransportStdio && cfg.Stdio != nil && cfg.Stdio.StartupTimeoutMS > 0 {
		connectTimeout = time.Duration(cfg.Stdio.StartupTimeoutMS) * time.Millisecond
	}

	cctx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	client, err := m.factory.Connect(cctx, cfg, resolved, m)
	if err != nil {
		m.setErrorIfCurrent(req.ServerID, generation, err)
		return nil, err
	}

	snap, err := client.Discover(cctx, cfg.ID, cfg.DefaultPolicy, cfg.TrustLevel)
	if err != nil {
		_ = client.Close(ctx)
		m.setErrorIfCurrent(req.ServerID, generation, err)
		return nil, err
	}

	normalizeSnapshot(&snap)
	if err := m.store.SaveLastKnownSnapshot(ctx, snap); err != nil {
		slog.Warn("mcp: save last known snapshot failed", "serverID", req.ServerID, "err", err)
	}

	now := time.Now().UTC()
	var oldClient ClientSession

	m.mu.Lock()
	if m.shuttingDown || m.generations[req.ServerID] != generation {
		m.mu.Unlock()
		_ = client.Close(ctx)
		return nil, fmt.Errorf("%w: connection superseded for server %s", spec.ErrMCPRuntimeNotReady, req.ServerID)
	}

	old := m.sessions[req.ServerID]
	if old != nil && old.client != nil && old.client != client {
		oldClient = old.client
	}

	state = m.getOrCreateLocked(req.ServerID)
	state.client = client
	state.status = spec.MCPServerStatusReady
	state.snapshot = cloneDiscoverySnapshot(snap)
	state.lastConnectedAt = now
	state.lastSyncedAt = now
	state.lastError = ""
	m.mu.Unlock()

	if oldClient != nil {
		_ = oldClient.Close(ctx)
	}

	return &spec.ConnectMCPServerResponse{Body: m.snapshotFromState(req.ServerID)}, nil
}

func (m *RuntimeManager) Disconnect(
	ctx context.Context,
	req *spec.DisconnectMCPServerRequest,
) (*spec.DisconnectMCPServerResponse, error) {
	if req == nil || req.ServerID == "" {
		return nil, fmt.Errorf("%w: serverID required", spec.ErrMCPInvalidRequest)
	}

	m.mu.Lock()
	m.bumpGenerationLocked(req.ServerID)

	state := m.sessions[req.ServerID]
	delete(m.sessions, req.ServerID)
	m.mu.Unlock()

	if state != nil && state.client != nil {
		if err := state.client.Close(ctx); err != nil {
			return nil, err
		}
	}
	return &spec.DisconnectMCPServerResponse{}, nil
}

func (m *RuntimeManager) Refresh(
	ctx context.Context,
	req *spec.RefreshMCPServerRequest,
) (*spec.RefreshMCPServerResponse, error) {
	if req == nil || req.ServerID == "" {
		return nil, fmt.Errorf("%w: serverID required", spec.ErrMCPInvalidRequest)
	}
	client, cfg, err := m.readyClient(ctx, req.ServerID)
	if err != nil {
		return nil, err
	}

	rctx, cancel := context.WithTimeout(ctx, time.Duration(spec.DefaultRequestTimeoutMS)*time.Millisecond)
	defer cancel()

	snap, err := client.Discover(rctx, req.ServerID, cfg.DefaultPolicy, cfg.TrustLevel)
	if err != nil {
		m.setError(req.ServerID, err)
		return nil, err
	}
	normalizeSnapshot(&snap)
	_ = m.store.SaveLastKnownSnapshot(ctx, snap)

	now := time.Now().UTC()
	m.mu.Lock()
	state := m.sessions[req.ServerID]
	if state == nil || state.client != client || state.status != spec.MCPServerStatusReady {
		m.mu.Unlock()
		return nil, fmt.Errorf("%w: server %s is no longer connected", spec.ErrMCPRuntimeNotReady, req.ServerID)
	}
	state.snapshot = cloneDiscoverySnapshot(snap)
	state.lastSyncedAt = now
	state.status = spec.MCPServerStatusReady
	state.lastError = ""
	m.mu.Unlock()

	return &spec.RefreshMCPServerResponse{Body: m.snapshotFromState(req.ServerID)}, nil
}

func (m *RuntimeManager) Status(
	ctx context.Context,
	req *spec.GetMCPServerStatusRequest,
) (*spec.GetMCPServerStatusResponse, error) {
	if req == nil || req.ServerID == "" {
		return nil, fmt.Errorf("%w: serverID required", spec.ErrMCPInvalidRequest)
	}
	return &spec.GetMCPServerStatusResponse{Body: m.snapshotFromState(req.ServerID)}, nil
}

func (m *RuntimeManager) OnClientNotification(ctx context.Context, event ClientNotification) {
	if m == nil || event.ServerID == "" {
		return
	}

	switch event.Kind {
	case ClientNotificationToolListChanged,
		ClientNotificationResourceListChanged,
		ClientNotificationPromptListChanged:
		m.scheduleNotificationRefresh(ctx, event.ServerID, string(event.Kind))

	case ClientNotificationResourceUpdated:
		slog.Info(
			"mcp resource updated notification received",
			"serverID", event.ServerID,
			"uri", event.ResourceURI,
		)

	case ClientNotificationLoggingMessage:
		slog.Info(
			"mcp server log notification received",
			"serverID", event.ServerID,
			"logger", event.LoggerName,
			"level", event.LoggingLevel,
			"data", event.LogData,
		)

	case ClientNotificationProgress:
		slog.Debug(
			"mcp progress notification received",
			"serverID", event.ServerID,
			"progress", event.Progress,
			"total", event.Total,
			"message", event.Message,
		)
	}
}

func (m *RuntimeManager) ListTools(
	ctx context.Context,
	req *spec.ListMCPServerToolsRequest,
) (*spec.ListMCPServerToolsResponse, error) {
	if req == nil || req.ServerID == "" {
		return nil, fmt.Errorf("%w: serverID required", spec.ErrMCPInvalidRequest)
	}
	snap, err := m.currentSnapshot(ctx, req.ServerID)
	if err != nil {
		return nil, err
	}
	return &spec.ListMCPServerToolsResponse{
		Body: &spec.ListMCPServerToolsResponseBody{Tools: snap.Tools},
	}, nil
}

func (m *RuntimeManager) ListResources(
	ctx context.Context,
	req *spec.ListMCPServerResourcesRequest,
) (*spec.ListMCPServerResourcesResponse, error) {
	if req == nil || req.ServerID == "" {
		return nil, fmt.Errorf("%w: serverID required", spec.ErrMCPInvalidRequest)
	}
	snap, err := m.currentSnapshot(ctx, req.ServerID)
	if err != nil {
		return nil, err
	}
	return &spec.ListMCPServerResourcesResponse{
		Body: &spec.ListMCPServerResourcesResponseBody{Resources: snap.Resources},
	}, nil
}

func (m *RuntimeManager) ListResourceTemplates(
	ctx context.Context,
	req *spec.ListMCPServerResourceTemplatesRequest,
) (*spec.ListMCPServerResourceTemplatesResponse, error) {
	if req == nil || req.ServerID == "" {
		return nil, fmt.Errorf("%w: serverID required", spec.ErrMCPInvalidRequest)
	}
	snap, err := m.currentSnapshot(ctx, req.ServerID)
	if err != nil {
		return nil, err
	}
	return &spec.ListMCPServerResourceTemplatesResponse{
		Body: &spec.ListMCPServerResourceTemplatesResponseBody{ResourceTemplates: snap.ResourceTemplates},
	}, nil
}

func (m *RuntimeManager) ListPrompts(
	ctx context.Context,
	req *spec.ListMCPServerPromptsRequest,
) (*spec.ListMCPServerPromptsResponse, error) {
	if req == nil || req.ServerID == "" {
		return nil, fmt.Errorf("%w: serverID required", spec.ErrMCPInvalidRequest)
	}
	snap, err := m.currentSnapshot(ctx, req.ServerID)
	if err != nil {
		return nil, err
	}
	return &spec.ListMCPServerPromptsResponse{
		Body: &spec.ListMCPServerPromptsResponseBody{Prompts: snap.Prompts},
	}, nil
}

func (m *RuntimeManager) ReadResource(
	ctx context.Context,
	req *spec.MCPReadResourceRequest,
) (*spec.MCPReadResourceResponse, error) {
	if req == nil || req.Body == nil || req.Body.ServerID == "" || req.Body.URI == "" {
		return nil, fmt.Errorf("%w: serverID and uri required", spec.ErrMCPInvalidRequest)
	}
	client, _, err := m.readyClient(ctx, req.Body.ServerID)
	if err != nil {
		return nil, err
	}
	rctx, cancel := withDefaultRequestTimeout(ctx)
	defer cancel()

	body, err := client.ReadResource(rctx, req.Body.URI)
	if err != nil {
		return nil, err
	}
	body.ServerID = req.Body.ServerID
	body.URI = req.Body.URI
	return &spec.MCPReadResourceResponse{Body: body}, nil
}

func (m *RuntimeManager) GetPrompt(
	ctx context.Context,
	req *spec.MCPGetPromptRequest,
) (*spec.MCPGetPromptResponse, error) {
	if req == nil || req.Body == nil || req.Body.ServerID == "" || req.Body.PromptName == "" {
		return nil, fmt.Errorf("%w: serverID and promptName required", spec.ErrMCPInvalidRequest)
	}
	client, _, err := m.readyClient(ctx, req.Body.ServerID)
	if err != nil {
		return nil, err
	}
	rctx, cancel := withDefaultRequestTimeout(ctx)
	defer cancel()

	body, err := client.GetPrompt(rctx, req.Body.PromptName, req.Body.Arguments)
	if err != nil {
		return nil, err
	}
	body.ServerID = req.Body.ServerID

	return &spec.MCPGetPromptResponse{Body: body}, nil
}

func (m *RuntimeManager) Complete(
	ctx context.Context,
	req *spec.MCPCompleteArgumentRequest,
) (*spec.MCPCompletionResult, error) {
	if req == nil || req.Body == nil || req.Body.ServerID == "" {
		return nil, fmt.Errorf("%w: serverID required", spec.ErrMCPInvalidRequest)
	}
	client, _, err := m.readyClient(ctx, req.Body.ServerID)
	if err != nil {
		return nil, err
	}
	rctx, cancel := withDefaultRequestTimeout(ctx)
	defer cancel()
	return client.Complete(rctx, *req.Body)
}

func (m *RuntimeManager) CallTool(
	ctx context.Context,
	req spec.InvokeMCPToolRequestBody,
) (*spec.InvokeMCPToolResponseBody, spec.MCPServerConfig, spec.MCPToolCapability, error) {
	client, cfg, err := m.readyClient(ctx, req.ServerID)
	if err != nil {
		return nil, spec.MCPServerConfig{}, spec.MCPToolCapability{}, err
	}

	snap, err := m.currentSnapshot(ctx, req.ServerID)
	if err != nil {
		return nil, spec.MCPServerConfig{}, spec.MCPToolCapability{}, err
	}

	var tool spec.MCPToolCapability
	found := false
	for _, t := range snap.Tools {
		if t.ToolName == req.ToolName {
			tool = t
			found = true
			break
		}
	}
	if !found {
		return nil, cfg, spec.MCPToolCapability{}, fmt.Errorf("%w: tool %s", spec.ErrMCPInvalidRequest, req.ToolName)
	}
	if req.ToolDigest != "" && req.ToolDigest != tool.Digest {
		return nil, cfg, tool, fmt.Errorf("%w: tool digest changed", spec.ErrMCPStaleReference)
	}

	rctx, cancel := withDefaultRequestTimeout(ctx)
	defer cancel()

	body, err := client.CallTool(rctx, req.ToolName, req.Arguments)
	if err != nil {
		return nil, cfg, tool, err
	}

	body.ServerID = req.ServerID
	body.ToolName = req.ToolName
	body.ProviderToolName = req.ProviderToolName
	body.Provenance.ServerID = req.ServerID
	body.Provenance.ServerDisplayName = cfg.DisplayName
	body.Provenance.ToolName = req.ToolName
	body.Provenance.ProviderToolName = req.ProviderToolName
	body.Provenance.ToolDigest = tool.Digest
	body.Provenance.ToolUseID = req.ToolUseID
	return body, cfg, tool, nil
}

func (m *RuntimeManager) CallToolDryRun(
	ctx context.Context,
	req spec.InvokeMCPToolRequestBody,
) (*spec.InvokeMCPToolResponseBody, spec.MCPServerConfig, spec.MCPToolCapability, error) {
	_, cfg, err := m.readyClient(ctx, req.ServerID)
	if err != nil {
		return nil, spec.MCPServerConfig{}, spec.MCPToolCapability{}, err
	}

	snap, err := m.currentSnapshot(ctx, req.ServerID)
	if err != nil {
		return nil, spec.MCPServerConfig{}, spec.MCPToolCapability{}, err
	}

	var tool spec.MCPToolCapability
	found := false
	for _, t := range snap.Tools {
		if t.ToolName == req.ToolName {
			tool = t
			found = true
			break
		}
	}
	if !found {
		return nil, cfg, spec.MCPToolCapability{}, fmt.Errorf("%w: tool %s", spec.ErrMCPInvalidRequest, req.ToolName)
	}
	if req.ToolDigest != "" && req.ToolDigest != tool.Digest {
		return nil, cfg, tool, fmt.Errorf("%w: tool digest changed", spec.ErrMCPStaleReference)
	}

	return &spec.InvokeMCPToolResponseBody{
		ServerID:         req.ServerID,
		ToolName:         req.ToolName,
		ProviderToolName: req.ProviderToolName,
	}, cfg, tool, nil
}

func (m *RuntimeManager) scheduleNotificationRefresh(ctx context.Context, serverID spec.MCPServerID, reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shuttingDown {
		return
	}

	if timer := m.notificationRefreshTimers[serverID]; timer != nil {
		timer.Reset(notificationRefreshDebounce)
		return
	}

	var timer *time.Timer
	timer = time.AfterFunc(notificationRefreshDebounce, func() {
		m.refreshFromNotification(ctx, serverID, reason, timer)
	})
	m.notificationRefreshTimers[serverID] = timer
}

func (m *RuntimeManager) refreshFromNotification(
	ctx context.Context,
	serverID spec.MCPServerID,
	reason string,
	timer *time.Timer,
) {
	ctx, cancel := context.WithTimeout(
		ctx,
		time.Duration(spec.DefaultRequestTimeoutMS)*time.Millisecond,
	)
	defer cancel()

	m.mu.Lock()
	if current := m.notificationRefreshTimers[serverID]; current == timer {
		delete(m.notificationRefreshTimers, serverID)
	}
	if m.shuttingDown {
		m.mu.Unlock()
		return
	}

	state := m.sessions[serverID]
	if state == nil || state.status != spec.MCPServerStatusReady || state.client == nil {
		m.mu.Unlock()
		return
	}

	client := state.client
	generation := m.generations[serverID]
	m.mu.Unlock()

	cfgResp, err := m.store.GetMCPServer(ctx, &spec.GetMCPServerRequest{ServerID: serverID})
	if err != nil {
		m.setErrorIfCurrent(serverID, generation, err)
		return
	}
	cfg := *cfgResp.Body
	if !cfg.Enabled {
		return
	}

	snap, err := client.Discover(ctx, serverID, cfg.DefaultPolicy, cfg.TrustLevel)
	if err != nil {
		m.setErrorIfCurrent(serverID, generation, err)
		return
	}

	normalizeSnapshot(&snap)
	if err := m.store.SaveLastKnownSnapshot(ctx, snap); err != nil {
		slog.Warn("mcp: save notification-refreshed snapshot failed", "serverID", serverID, "err", err)
	}

	now := time.Now().UTC()

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shuttingDown || m.generations[serverID] != generation {
		return
	}

	state = m.sessions[serverID]
	if state == nil || state.client != client || state.status != spec.MCPServerStatusReady {
		return
	}

	state.snapshot = cloneDiscoverySnapshot(snap)
	state.lastSyncedAt = now
	state.lastError = ""

	slog.Info("mcp discovery refreshed from notification", "serverID", serverID, "reason", reason)
}

func (m *RuntimeManager) currentSnapshot(
	ctx context.Context,
	serverID spec.MCPServerID,
) (spec.MCPDiscoverySnapshot, error) {
	m.mu.RLock()
	if st := m.sessions[serverID]; st != nil && st.status == spec.MCPServerStatusReady {
		snap := cloneDiscoverySnapshot(st.snapshot)
		m.mu.RUnlock()
		return snap, nil
	}
	m.mu.RUnlock()

	snap, ok, err := m.store.GetLastKnownSnapshot(ctx, serverID)
	if err != nil {
		return spec.MCPDiscoverySnapshot{}, err
	}
	if !ok {
		return spec.MCPDiscoverySnapshot{}, fmt.Errorf("%w: no runtime snapshot", spec.ErrMCPRuntimeNotReady)
	}
	return cloneDiscoverySnapshot(snap), nil
}

func (m *RuntimeManager) readyClient(
	ctx context.Context,
	serverID spec.MCPServerID,
) (ClientSession, spec.MCPServerConfig, error) {
	cfgResp, err := m.store.GetMCPServer(ctx, &spec.GetMCPServerRequest{ServerID: serverID})
	if err != nil {
		return nil, spec.MCPServerConfig{}, err
	}
	cfg := *cfgResp.Body
	if !cfg.Enabled {
		return nil, cfg, fmt.Errorf("%w: %s", spec.ErrMCPServerDisabled, serverID)
	}

	m.mu.RLock()
	st := m.sessions[serverID]
	if st != nil && st.status == spec.MCPServerStatusReady && st.client != nil {
		client := st.client
		m.mu.RUnlock()
		return client, cfg, nil
	}
	m.mu.RUnlock()

	return nil, cfg, fmt.Errorf("%w: server %s not connected", spec.ErrMCPRuntimeNotReady, serverID)
}

func (m *RuntimeManager) getOrCreateLocked(id spec.MCPServerID) *sessionState {
	st := m.sessions[id]
	if st == nil {
		st = &sessionState{serverID: id, status: spec.MCPServerStatusDisconnected}
		m.sessions[id] = st
	}
	return st
}

func (m *RuntimeManager) setError(id spec.MCPServerID, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.getOrCreateLocked(id)
	st.status = spec.MCPServerStatusError
	if err != nil {
		st.lastError = err.Error()
	}
}

func (m *RuntimeManager) setErrorIfCurrent(id spec.MCPServerID, generation uint64, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.generations[id] != generation {
		return
	}
	st := m.getOrCreateLocked(id)
	st.status = spec.MCPServerStatusError
	if err != nil {
		st.lastError = err.Error()
	}
}

func (m *RuntimeManager) bumpGenerationLocked(id spec.MCPServerID) uint64 {
	m.generations[id]++
	return m.generations[id]
}

func (m *RuntimeManager) snapshotFromState(id spec.MCPServerID) *spec.MCPServerRuntimeSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	st := m.sessions[id]
	if st == nil {
		return &spec.MCPServerRuntimeSnapshot{ServerID: id, Status: spec.MCPServerStatusDisconnected}
	}

	out := &spec.MCPServerRuntimeSnapshot{
		ServerID:                  id,
		Status:                    st.status,
		LastError:                 st.lastError,
		NegotiatedProtocolVersion: st.snapshot.NegotiatedProtocolVersion,
		ServerInfo:                clonePtr(st.snapshot.ServerInfo),
		ServerCapabilities:        cloneCapabilitiesSummary(st.snapshot.ServerCapabilities),
		Instructions:              st.snapshot.Instructions,
		ToolCount:                 len(st.snapshot.Tools),
		ResourceCount:             len(st.snapshot.Resources),
		ResourceTemplateCount:     len(st.snapshot.ResourceTemplates),
		PromptCount:               len(st.snapshot.Prompts),
		SnapshotDigest:            st.snapshot.Digest,
	}
	if !st.lastConnectedAt.IsZero() {
		out.LastConnectedAt = st.lastConnectedAt.Format(time.RFC3339Nano)
	}
	if !st.lastSyncedAt.IsZero() {
		out.LastSyncedAt = st.lastSyncedAt.Format(time.RFC3339Nano)
	}
	return out
}

func normalizeSnapshot(snap *spec.MCPDiscoverySnapshot) {
	sort.Slice(snap.Tools, func(i, j int) bool { return snap.Tools[i].ToolName < snap.Tools[j].ToolName })
	sort.Slice(snap.Resources, func(i, j int) bool { return snap.Resources[i].URI < snap.Resources[j].URI })
	sort.Slice(snap.ResourceTemplates, func(i, j int) bool {
		return snap.ResourceTemplates[i].URITemplate < snap.ResourceTemplates[j].URITemplate
	})
	sort.Slice(snap.Prompts, func(i, j int) bool { return snap.Prompts[i].PromptName < snap.Prompts[j].PromptName })

	if snap.SyncedAt == "" {
		snap.SyncedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	if snap.Digest == "" {
		raw, _ := json.Marshal(struct {
			Tools             []spec.MCPToolCapability      `json:"tools"`
			Resources         []spec.MCPResourceRef         `json:"resources"`
			ResourceTemplates []spec.MCPResourceTemplateRef `json:"resourceTemplates"`
			Prompts           []spec.MCPPromptRef           `json:"prompts"`
		}{
			Tools:             snap.Tools,
			Resources:         snap.Resources,
			ResourceTemplates: snap.ResourceTemplates,
			Prompts:           snap.Prompts,
		})
		sum := sha256.Sum256(raw)
		snap.Digest = hex.EncodeToString(sum[:])
	}
}

func withDefaultRequestTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, time.Duration(spec.DefaultRequestTimeoutMS)*time.Millisecond)
}

func cloneDiscoverySnapshot(in spec.MCPDiscoverySnapshot) spec.MCPDiscoverySnapshot {
	out := in
	out.ServerInfo = clonePtr(in.ServerInfo)
	out.ServerCapabilities = cloneCapabilitiesSummary(in.ServerCapabilities)

	out.Tools = slices.Clone(in.Tools)
	for i := range out.Tools {
		out.Tools[i] = cloneToolCapability(out.Tools[i])
	}

	out.Resources = slices.Clone(in.Resources)
	for i := range out.Resources {
		out.Resources[i].Annotations = maps.Clone(out.Resources[i].Annotations)
	}

	out.ResourceTemplates = slices.Clone(in.ResourceTemplates)
	for i := range out.ResourceTemplates {
		out.ResourceTemplates[i].Arguments = maps.Clone(out.ResourceTemplates[i].Arguments)
		out.ResourceTemplates[i].Annotations = maps.Clone(out.ResourceTemplates[i].Annotations)
	}

	out.Prompts = slices.Clone(in.Prompts)
	for i := range out.Prompts {
		out.Prompts[i].Arguments = maps.Clone(out.Prompts[i].Arguments)
	}

	return out
}

func cloneToolCapability(in spec.MCPToolCapability) spec.MCPToolCapability {
	out := in
	out.InputSchema = maps.Clone(in.InputSchema)
	out.OutputSchema = maps.Clone(in.OutputSchema)
	out.Annotations = clonePtr(in.Annotations)
	if in.App != nil {
		cp := *in.App
		cp.Visibility = slices.Clone(in.App.Visibility)
		out.App = &cp
	}
	return out
}

func cloneCapabilitiesSummary(in *spec.MCPServerCapabilitiesSummary) *spec.MCPServerCapabilitiesSummary {
	if in == nil {
		return nil
	}
	out := *in
	out.Experimental = maps.Clone(in.Experimental)
	out.Extensions = maps.Clone(in.Extensions)
	return &out
}

func clonePtr[T any](in *T) *T {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}
