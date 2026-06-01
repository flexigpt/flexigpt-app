package runtime

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/mcp/auth"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
	"github.com/flexigpt/flexigpt-app/internal/mcp/store"
)

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
type ClientFactory interface {
	Connect(
		ctx context.Context,
		cfg spec.MCPServerConfig,
		resolved auth.ResolvedTransportAuth,
		events ClientNotificationSink,
	) (ClientSession, error)
}

type sessionState struct {
	bundleID        bundleitemutils.BundleID
	serverID        spec.MCPServerID
	status          spec.MCPServerStatus
	client          ClientSession
	snapshot        spec.MCPDiscoverySnapshot
	lastError       string
	lastConnectedAt time.Time
	lastSyncedAt    time.Time
}

type MCPRuntimeManager struct {
	store   *store.Store
	auth    *auth.AuthManager
	factory ClientFactory

	mu                        sync.RWMutex
	sessions                  map[spec.MCPServerID]*sessionState
	generations               map[spec.MCPServerID]uint64
	notificationRefreshTimers map[spec.MCPServerID]*time.Timer
	shuttingDown              bool
}

func NewMCPRuntimeManager(st *store.Store, authMgr *auth.AuthManager, factory ClientFactory) *MCPRuntimeManager {
	return &MCPRuntimeManager{
		store:                     st,
		auth:                      authMgr,
		factory:                   factory,
		sessions:                  map[spec.MCPServerID]*sessionState{},
		generations:               map[spec.MCPServerID]uint64{},
		notificationRefreshTimers: map[spec.MCPServerID]*time.Timer{},
	}
}

func (m *MCPRuntimeManager) Close(ctx context.Context) error {
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
	if m.auth != nil {
		m.auth.ClearAuthStatuses()
	}
	return first
}

func (m *MCPRuntimeManager) Connect(
	ctx context.Context,
	req *spec.ConnectMCPServerRequest,
) (*spec.ConnectMCPServerResponse, error) {
	if req == nil || req.BundleID == "" || req.ServerID == "" {
		return nil, fmt.Errorf("%w: bundleID and serverID required", spec.ErrMCPInvalidRequest)
	}

	m.mu.Lock()
	if m.shuttingDown {
		m.mu.Unlock()
		return nil, fmt.Errorf("%w: runtime is shutting down", spec.ErrMCPRuntimeNotReady)
	}
	state := m.getOrCreateLocked(req.ServerID)
	state.bundleID = req.BundleID
	generation := m.bumpGenerationLocked(req.ServerID)

	state.status = spec.MCPServerStatusConnecting
	state.lastError = ""
	m.mu.Unlock()

	cfgResp, err := m.store.GetMCPServer(ctx, &spec.GetMCPServerRequest{
		BundleID: req.BundleID,
		ServerID: req.ServerID,
	})
	if err != nil {
		m.setErrorIfCurrent(req.ServerID, generation, err)
		return nil, err
	}
	if cfgResp == nil || cfgResp.Body == nil {
		err := fmt.Errorf("%w: empty server config response", spec.ErrMCPRuntimeNotReady)
		m.setErrorIfCurrent(req.ServerID, generation, err)
		return nil, err
	}
	cfg := *cfgResp.Body
	if !cfg.Enabled {
		if m.auth != nil {
			m.auth.ClearAuthStatus(req.BundleID, req.ServerID)
		}
		err := fmt.Errorf("%w: %s", spec.ErrMCPServerDisabled, cfg.ID)
		m.setStatusIfCurrent(req.ServerID, generation, spec.MCPServerStatusDisabled, "")
		return nil, err
	}

	resolved := auth.ResolvedTransportAuth{Env: map[string]string{}}
	if cfg.Transport == spec.MCPTransportStdio && cfg.Stdio != nil && len(cfg.Stdio.Env) > 0 {
		resolved.Env = maps.Clone(cfg.Stdio.Env)
	}

	if m.auth != nil {
		prepared, err := m.auth.PrepareTransportAuth(ctx, cfg)
		if err != nil {
			m.setErrorIfCurrent(req.ServerID, generation, err)
			return nil, err
		}
		mergeResolvedTransportAuth(&resolved, prepared)
	}

	connectTimeout := time.Duration(spec.DefaultConnectTimeoutMS) * time.Millisecond
	if cfg.Transport == spec.MCPTransportStreamableHTTP && cfg.StreamableHTTP != nil &&
		cfg.StreamableHTTP.TimeoutMS > 0 {
		connectTimeout = time.Duration(cfg.StreamableHTTP.TimeoutMS) * time.Millisecond
	}
	if cfg.Transport == spec.MCPTransportStdio && cfg.Stdio != nil && cfg.Stdio.StartupTimeoutMS > 0 {
		connectTimeout = time.Duration(cfg.Stdio.StartupTimeoutMS) * time.Millisecond
	}
	// Only the interactive authorization-code flow needs an extended connect window.
	// "client_credentials" is a single non-interactive token-endpoint call and uses the regular connect timeout.
	if cfg.Transport == spec.MCPTransportStreamableHTTP &&
		cfg.StreamableHTTP != nil &&
		cfg.StreamableHTTP.AuthMode == spec.MCPHTTPAuthOAuth &&
		connectTimeout < spec.DefaultInteractiveOAuthTimeout {
		connectTimeout = spec.DefaultInteractiveOAuthTimeout
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

	hydrateSnapshotIdentity(&snap, cfg.BundleID, cfg.ID)
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
	state.bundleID = cfg.BundleID
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

	return &spec.ConnectMCPServerResponse{Body: m.snapshotFromState(req.BundleID, req.ServerID)}, nil
}

func (m *MCPRuntimeManager) Disconnect(
	ctx context.Context,
	req *spec.DisconnectMCPServerRequest,
) (*spec.DisconnectMCPServerResponse, error) {
	if req == nil || req.BundleID == "" || req.ServerID == "" {
		return nil, fmt.Errorf("%w: bundleID and serverID required", spec.ErrMCPInvalidRequest)
	}

	m.mu.Lock()
	m.bumpGenerationLocked(req.ServerID)

	state := m.sessions[req.ServerID]
	if state != nil && state.bundleID != "" && state.bundleID != req.BundleID {
		m.mu.Unlock()
		return nil, fmt.Errorf(
			"%w: server %s is connected under bundle %s, not %s",
			spec.ErrMCPInvalidRequest,
			req.ServerID,
			state.bundleID,
			req.BundleID,
		)
	}
	delete(m.sessions, req.ServerID)
	timer := m.notificationRefreshTimers[req.ServerID]
	delete(m.notificationRefreshTimers, req.ServerID)
	m.mu.Unlock()
	if timer != nil {
		timer.Stop()
	}
	if state != nil && state.client != nil {
		if err := state.client.Close(ctx); err != nil {
			return nil, err
		}
	}
	if m.auth != nil {
		m.auth.ClearAuthStatus(req.BundleID, req.ServerID)
	}

	return &spec.DisconnectMCPServerResponse{}, nil
}

func (m *MCPRuntimeManager) Refresh(
	ctx context.Context,
	req *spec.RefreshMCPServerRequest,
) (*spec.RefreshMCPServerResponse, error) {
	if req == nil || req.BundleID == "" || req.ServerID == "" {
		return nil, fmt.Errorf("%w: bundleID and serverID required", spec.ErrMCPInvalidRequest)
	}
	client, cfg, err := m.readyClient(ctx, req.BundleID, req.ServerID)
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
	hydrateSnapshotIdentity(&snap, cfg.BundleID, cfg.ID)
	normalizeSnapshot(&snap)
	if err := m.store.SaveLastKnownSnapshot(ctx, snap); err != nil {
		slog.Warn("mcp: save refreshed snapshot failed", "serverID", req.ServerID, "err", err)
	}
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

	return &spec.RefreshMCPServerResponse{Body: m.snapshotFromState(req.BundleID, req.ServerID)}, nil
}

func (m *MCPRuntimeManager) Status(
	ctx context.Context,
	req *spec.GetMCPServerStatusRequest,
) (*spec.GetMCPServerStatusResponse, error) {
	if req == nil || req.BundleID == "" || req.ServerID == "" {
		return nil, fmt.Errorf("%w: bundleID and serverID required", spec.ErrMCPInvalidRequest)
	}
	cfgResp, err := m.store.GetMCPServer(ctx, &spec.GetMCPServerRequest{
		BundleID: req.BundleID,
		ServerID: req.ServerID,
	})
	if err != nil {
		return nil, err
	}
	snap := m.snapshotFromState(req.BundleID, req.ServerID)

	if cfgResp != nil && cfgResp.Body != nil {
		snap.BundleID = cfgResp.Body.BundleID
		if !cfgResp.Body.Enabled {
			snap.Status = spec.MCPServerStatusDisabled
			snap.LastError = ""
		}
	}
	return &spec.GetMCPServerStatusResponse{Body: snap}, nil
}

func (m *MCPRuntimeManager) OnClientNotification(ctx context.Context, event ClientNotification) {
	if m == nil || event.ServerID == "" {
		return
	}

	switch event.Kind {
	case ClientNotificationToolListChanged,
		ClientNotificationResourceListChanged,
		ClientNotificationPromptListChanged:
		m.scheduleNotificationRefresh(ctx, event.BundleID, event.ServerID, string(event.Kind))

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

func (m *MCPRuntimeManager) ListTools(
	ctx context.Context,
	req *spec.ListMCPServerToolsRequest,
) (*spec.ListMCPServerToolsResponse, error) {
	if req == nil || req.BundleID == "" || req.ServerID == "" {
		return nil, fmt.Errorf("%w: bundleID and serverID required", spec.ErrMCPInvalidRequest)
	}
	snap, err := m.currentSnapshot(ctx, req.BundleID, req.ServerID)
	if err != nil {
		return nil, err
	}
	cfgResp, err := m.store.GetMCPServer(ctx, &spec.GetMCPServerRequest{
		BundleID: req.BundleID,
		ServerID: req.ServerID,
	})
	if err != nil {
		// Best effort: keep the snapshot if the config read fails.
		slog.Warn("mcp: tool policy overlay skipped", "serverID", req.ServerID, "err", err)
	}

	sort.Slice(snap.Tools, func(i, j int) bool {
		return snap.Tools[i].ToolName < snap.Tools[j].ToolName
	})

	if cfgResp != nil && cfgResp.Body != nil {
		for i := range snap.Tools {
			snap.Tools[i] = applyToolPolicyOverlay(snap.Tools[i], *cfgResp.Body)
		}
	}

	digest := computeDiscoverySnapshotDigest(snap)
	tools, next, err := paginateDiscoveryItems(
		req.BundleID, req.ServerID, digest, discoveryPageKindTools, snap.Tools, req.PageSize, req.PageToken,
	)
	if err != nil {
		return nil, err
	}
	return &spec.ListMCPServerToolsResponse{
		Body: &spec.ListMCPServerToolsResponseBody{
			Tools:         tools,
			NextPageToken: next,
		},
	}, nil
}

func (m *MCPRuntimeManager) ListResources(
	ctx context.Context,
	req *spec.ListMCPServerResourcesRequest,
) (*spec.ListMCPServerResourcesResponse, error) {
	if req == nil || req.BundleID == "" || req.ServerID == "" {
		return nil, fmt.Errorf("%w: bundleID and serverID required", spec.ErrMCPInvalidRequest)
	}
	snap, err := m.currentSnapshot(ctx, req.BundleID, req.ServerID)
	if err != nil {
		return nil, err
	}
	sort.Slice(snap.Resources, func(i, j int) bool {
		return snap.Resources[i].URI < snap.Resources[j].URI
	})
	digest := computeDiscoverySnapshotDigest(snap)
	resources, next, err := paginateDiscoveryItems(
		req.BundleID, req.ServerID, digest, discoveryPageKindResources, snap.Resources, req.PageSize, req.PageToken,
	)
	if err != nil {
		return nil, err
	}
	return &spec.ListMCPServerResourcesResponse{
		Body: &spec.ListMCPServerResourcesResponseBody{
			Resources:     resources,
			NextPageToken: next,
		},
	}, nil
}

func (m *MCPRuntimeManager) ListResourceTemplates(
	ctx context.Context,
	req *spec.ListMCPServerResourceTemplatesRequest,
) (*spec.ListMCPServerResourceTemplatesResponse, error) {
	if req == nil || req.BundleID == "" || req.ServerID == "" {
		return nil, fmt.Errorf("%w: bundleID and serverID required", spec.ErrMCPInvalidRequest)
	}
	snap, err := m.currentSnapshot(ctx, req.BundleID, req.ServerID)
	if err != nil {
		return nil, err
	}
	sort.Slice(snap.ResourceTemplates, func(i, j int) bool {
		return snap.ResourceTemplates[i].URITemplate < snap.ResourceTemplates[j].URITemplate
	})
	digest := computeDiscoverySnapshotDigest(snap)
	templates, next, err := paginateDiscoveryItems(
		req.BundleID,
		req.ServerID,
		digest,
		discoveryPageKindResourceTemplates,
		snap.ResourceTemplates,
		req.PageSize,
		req.PageToken,
	)
	if err != nil {
		return nil, err
	}
	return &spec.ListMCPServerResourceTemplatesResponse{
		Body: &spec.ListMCPServerResourceTemplatesResponseBody{
			ResourceTemplates: templates,
			NextPageToken:     next,
		},
	}, nil
}

func (m *MCPRuntimeManager) ListPrompts(
	ctx context.Context,
	req *spec.ListMCPServerPromptsRequest,
) (*spec.ListMCPServerPromptsResponse, error) {
	if req == nil || req.BundleID == "" || req.ServerID == "" {
		return nil, fmt.Errorf("%w: bundleID and serverID required", spec.ErrMCPInvalidRequest)
	}
	snap, err := m.currentSnapshot(ctx, req.BundleID, req.ServerID)
	if err != nil {
		return nil, err
	}
	sort.Slice(snap.Prompts, func(i, j int) bool {
		return snap.Prompts[i].PromptName < snap.Prompts[j].PromptName
	})
	digest := computeDiscoverySnapshotDigest(snap)
	prompts, next, err := paginateDiscoveryItems(
		req.BundleID, req.ServerID, digest, discoveryPageKindPrompts, snap.Prompts, req.PageSize, req.PageToken,
	)
	if err != nil {
		return nil, err
	}
	return &spec.ListMCPServerPromptsResponse{
		Body: &spec.ListMCPServerPromptsResponseBody{
			Prompts:       prompts,
			NextPageToken: next,
		},
	}, nil
}

func (m *MCPRuntimeManager) ReadResource(
	ctx context.Context,
	req *spec.MCPReadResourceRequest,
) (*spec.MCPReadResourceResponse, error) {
	if req == nil || req.Body == nil || req.BundleID == "" || req.ServerID == "" || req.Body.URI == "" {
		return nil, fmt.Errorf("%w: bundleID serverID and uri required", spec.ErrMCPInvalidRequest)
	}
	client, cfg, err := m.readyClient(ctx, req.BundleID, req.ServerID)
	if err != nil {
		return nil, err
	}
	rctx, cancel := withDefaultRequestTimeout(ctx)
	defer cancel()

	body, err := client.ReadResource(rctx, req.Body.URI)
	if err != nil {
		return nil, err
	}
	if body == nil {
		return nil, fmt.Errorf("%w: resource read returned nil response", spec.ErrMCPRuntimeNotReady)
	}

	body.BundleID = cfg.BundleID
	body.ServerID = req.ServerID
	body.URI = req.Body.URI
	return &spec.MCPReadResourceResponse{Body: body}, nil
}

func (m *MCPRuntimeManager) GetPrompt(
	ctx context.Context,
	req *spec.MCPGetPromptRequest,
) (*spec.MCPGetPromptResponse, error) {
	if req == nil || req.Body == nil || req.BundleID == "" || req.ServerID == "" ||

		req.Body.PromptName == "" {
		return nil, fmt.Errorf("%w: bundleID serverID and promptName required", spec.ErrMCPInvalidRequest)
	}
	client, cfg, err := m.readyClient(ctx, req.BundleID, req.ServerID)
	if err != nil {
		return nil, err
	}
	rctx, cancel := withDefaultRequestTimeout(ctx)
	defer cancel()

	body, err := client.GetPrompt(rctx, req.Body.PromptName, req.Body.Arguments)
	if err != nil {
		return nil, err
	}
	if body == nil {
		return nil, fmt.Errorf("%w: prompt read returned nil response", spec.ErrMCPRuntimeNotReady)
	}

	body.BundleID = cfg.BundleID
	body.ServerID = req.ServerID

	return &spec.MCPGetPromptResponse{Body: body}, nil
}

func (m *MCPRuntimeManager) Complete(
	ctx context.Context,
	req *spec.MCPCompleteArgumentRequest,
) (*spec.MCPCompletionResult, error) {
	if req == nil || req.Body == nil || req.BundleID == "" || req.ServerID == "" {
		return nil, fmt.Errorf("%w: bundleID serverID required", spec.ErrMCPInvalidRequest)
	}
	client, _, err := m.readyClient(ctx, req.BundleID, req.ServerID)
	if err != nil {
		return nil, err
	}
	rctx, cancel := withDefaultRequestTimeout(ctx)
	defer cancel()
	res, err := client.Complete(rctx, *req.Body)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, fmt.Errorf("%w: completion returned nil response", spec.ErrMCPRuntimeNotReady)
	}

	return res, nil
}

func (m *MCPRuntimeManager) CallTool(
	ctx context.Context,
	bundleID bundleitemutils.BundleID,
	serverID spec.MCPServerID,
	req spec.InvokeMCPToolRequestBody,
) (*spec.InvokeMCPToolResponseBody, spec.MCPServerConfig, spec.MCPToolCapability, error) {
	client, cfg, err := m.readyClient(ctx, bundleID, serverID)
	if err != nil {
		return nil, spec.MCPServerConfig{}, spec.MCPToolCapability{}, err
	}

	snap, err := m.currentSnapshot(ctx, bundleID, serverID)
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
	if !tool.Enabled || tool.TaskSupport == spec.MCPTaskSupportRequired {
		return nil, cfg, tool, fmt.Errorf(
			"%w: tool %s is disabled or unsupported", spec.ErrMCPPolicyDenied, req.ToolName,
		)
	}
	if req.ToolDigest != "" && req.ToolDigest != tool.Digest {
		if ov, ok := cfg.ToolPolicies[tool.ToolName]; !ok || !ov.AllowStaleDigest {
			return nil, cfg, tool, fmt.Errorf("%w: tool digest changed", spec.ErrMCPStaleReference)
		}
	}
	rctx, cancel := withDefaultRequestTimeout(ctx)
	defer cancel()

	body, err := client.CallTool(rctx, req.ToolName, req.Arguments)
	if err != nil {
		return nil, cfg, tool, err
	}

	if body == nil {
		return nil, cfg, tool, fmt.Errorf("%w: tool call returned nil response", spec.ErrMCPRuntimeNotReady)
	}

	body.BundleID = cfg.BundleID
	body.ServerID = serverID
	body.ToolName = req.ToolName
	body.ProviderToolName = req.ProviderToolName
	body.Provenance.BundleID = cfg.BundleID
	body.Provenance.ServerID = serverID
	body.Provenance.ServerDisplayName = cfg.DisplayName
	body.Provenance.ToolName = req.ToolName
	body.Provenance.ProviderToolName = req.ProviderToolName
	body.Provenance.ToolDigest = tool.Digest
	body.Provenance.ToolUseID = req.ToolUseID
	body.Provenance.ApprovalID = req.ApprovalID

	return body, cfg, tool, nil
}

func (m *MCPRuntimeManager) CallToolDryRun(
	ctx context.Context,
	bundleID bundleitemutils.BundleID,
	serverID spec.MCPServerID,
	req spec.InvokeMCPToolRequestBody,
) (*spec.InvokeMCPToolResponseBody, spec.MCPServerConfig, spec.MCPToolCapability, error) {
	_, cfg, err := m.readyClient(ctx, bundleID, serverID)
	if err != nil {
		return nil, spec.MCPServerConfig{}, spec.MCPToolCapability{}, err
	}

	snap, err := m.currentSnapshot(ctx, bundleID, serverID)
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
	tool = applyToolPolicyOverlay(tool, cfg)
	if req.ToolDigest != "" && req.ToolDigest != tool.Digest {
		tool.Stale = true
	}
	return &spec.InvokeMCPToolResponseBody{
		BundleID:         cfg.BundleID,
		ServerID:         serverID,
		ToolName:         req.ToolName,
		ProviderToolName: req.ProviderToolName,
	}, cfg, tool, nil
}

func (m *MCPRuntimeManager) scheduleNotificationRefresh(
	ctx context.Context,
	bundleID bundleitemutils.BundleID,
	serverID spec.MCPServerID,
	reason string,
) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shuttingDown {
		return
	}

	if timer := m.notificationRefreshTimers[serverID]; timer != nil {
		timer.Reset(spec.NotificationRefreshDebounce)
		return
	}

	var timer *time.Timer
	timer = time.AfterFunc(spec.NotificationRefreshDebounce, func() {
		m.refreshFromNotification(ctx, bundleID, serverID, reason, timer)
	})
	m.notificationRefreshTimers[serverID] = timer
}

func (m *MCPRuntimeManager) refreshFromNotification(
	ctx context.Context,
	bundleID bundleitemutils.BundleID,
	serverID spec.MCPServerID,
	reason string,
	timer *time.Timer,
) {
	ctx = context.WithoutCancel(ctx)
	ctx, cancel := context.WithTimeout(ctx, time.Duration(spec.DefaultRequestTimeoutMS)*time.Millisecond)
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

	cfgResp, err := m.store.GetMCPServer(ctx, &spec.GetMCPServerRequest{BundleID: bundleID, ServerID: serverID})
	if err != nil {
		m.setErrorIfCurrent(serverID, generation, err)
		return
	}
	if cfgResp == nil || cfgResp.Body == nil {
		m.setErrorIfCurrent(
			serverID,
			generation,
			fmt.Errorf("%w: empty server config response", spec.ErrMCPRuntimeNotReady),
		)
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

	hydrateSnapshotIdentity(&snap, cfg.BundleID, serverID)
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

func (m *MCPRuntimeManager) currentSnapshot(
	ctx context.Context,
	bundleID bundleitemutils.BundleID,
	serverID spec.MCPServerID,
) (spec.MCPDiscoverySnapshot, error) {
	m.mu.RLock()
	if st := m.sessions[serverID]; st != nil &&
		st.status == spec.MCPServerStatusReady &&
		st.bundleID == bundleID {
		snap := cloneDiscoverySnapshot(st.snapshot)
		m.mu.RUnlock()
		return snap, nil
	}
	m.mu.RUnlock()

	snap, ok, err := m.store.GetLastKnownSnapshot(ctx, bundleID, serverID)
	if err != nil {
		return spec.MCPDiscoverySnapshot{}, err
	}
	if !ok {
		return spec.MCPDiscoverySnapshot{}, fmt.Errorf("%w: no runtime snapshot", spec.ErrMCPRuntimeNotReady)
	}
	return cloneDiscoverySnapshot(snap), nil
}

func (m *MCPRuntimeManager) readyClient(
	ctx context.Context,
	bundleID bundleitemutils.BundleID,
	serverID spec.MCPServerID,
) (ClientSession, spec.MCPServerConfig, error) {
	cfgResp, err := m.store.GetMCPServer(ctx, &spec.GetMCPServerRequest{BundleID: bundleID, ServerID: serverID})
	if err != nil {
		return nil, spec.MCPServerConfig{}, err
	}
	if cfgResp == nil || cfgResp.Body == nil {
		return nil, spec.MCPServerConfig{}, fmt.Errorf("%w: empty server config response", spec.ErrMCPRuntimeNotReady)
	}
	cfg := *cfgResp.Body
	if !cfg.Enabled {
		return nil, cfg, fmt.Errorf("%w: %s", spec.ErrMCPServerDisabled, serverID)
	}

	m.mu.RLock()
	st := m.sessions[serverID]
	if st != nil && st.status == spec.MCPServerStatusReady &&
		st.bundleID == bundleID &&
		st.client != nil {
		client := st.client
		m.mu.RUnlock()
		return client, cfg, nil
	}
	m.mu.RUnlock()

	return nil, cfg, fmt.Errorf("%w: server %s not connected", spec.ErrMCPRuntimeNotReady, serverID)
}

func (m *MCPRuntimeManager) setError(id spec.MCPServerID, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.getOrCreateLocked(id)
	st.status = spec.MCPServerStatusError
	if err != nil {
		st.lastError = err.Error()
	}
}

func (m *MCPRuntimeManager) setStatusIfCurrent(
	id spec.MCPServerID,
	generation uint64,
	status spec.MCPServerStatus,
	lastErr string,
) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.generations[id] != generation {
		return
	}
	st := m.getOrCreateLocked(id)
	st.status = status
	st.lastError = lastErr
}

func (m *MCPRuntimeManager) setErrorIfCurrent(id spec.MCPServerID, generation uint64, err error) {
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

func (m *MCPRuntimeManager) getOrCreateLocked(id spec.MCPServerID) *sessionState {
	st := m.sessions[id]
	if st == nil {
		st = &sessionState{serverID: id, status: spec.MCPServerStatusDisconnected}
		m.sessions[id] = st
	}
	return st
}

func (m *MCPRuntimeManager) bumpGenerationLocked(id spec.MCPServerID) uint64 {
	m.generations[id]++
	return m.generations[id]
}

func (m *MCPRuntimeManager) snapshotFromState(
	bundleID bundleitemutils.BundleID,
	id spec.MCPServerID,
) *spec.MCPServerRuntimeSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	st := m.sessions[id]
	if st == nil {
		return &spec.MCPServerRuntimeSnapshot{
			BundleID: bundleID,
			ServerID: id,
			Status:   spec.MCPServerStatusDisconnected,
		}
	}

	snapshotDigest := st.snapshot.Digest
	if snapshotDigest == "" {
		snapshotDigest = computeDiscoverySnapshotDigest(st.snapshot)
	}
	out := &spec.MCPServerRuntimeSnapshot{
		BundleID:                  firstBundleID(st.snapshot.BundleID, st.bundleID, bundleID),
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
		SnapshotDigest:            snapshotDigest,
	}
	if !st.lastConnectedAt.IsZero() {
		out.LastConnectedAt = st.lastConnectedAt.Format(time.RFC3339Nano)
	}
	if !st.lastSyncedAt.IsZero() {
		out.LastSyncedAt = st.lastSyncedAt.Format(time.RFC3339Nano)
	}
	return out
}

func firstBundleID(values ...bundleitemutils.BundleID) bundleitemutils.BundleID {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func applyToolPolicyOverlay(tool spec.MCPToolCapability, cfg spec.MCPServerConfig) spec.MCPToolCapability {
	if ov, ok := cfg.ToolPolicies[tool.ToolName]; ok {
		if ov.ApprovalRule != nil {
			tool.ApprovalRule = *ov.ApprovalRule
		}
		if ov.ExecutionMode != nil {
			tool.ExecutionMode = *ov.ExecutionMode
		}
		if ov.ExpectedDigest != "" && ov.ExpectedDigest != tool.Digest {
			tool.Stale = true
		}
	}
	return tool
}

func mergeResolvedTransportAuth(dst *auth.ResolvedTransportAuth, src auth.ResolvedTransportAuth) {
	if dst == nil {
		return
	}
	maps.Copy(dst.Env, src.Env)
	dst.SensitiveValues = append(dst.SensitiveValues, src.SensitiveValues...)
	if src.OAuthHandler != nil {
		dst.OAuthHandler = src.OAuthHandler
	}
	if src.Status.ServerID != "" {
		dst.Status = src.Status
	}
}

func hydrateSnapshotIdentity(
	snap *spec.MCPDiscoverySnapshot,
	bundleID bundleitemutils.BundleID,
	serverID spec.MCPServerID,
) {
	if snap == nil {
		return
	}
	snap.BundleID = bundleID
	snap.ServerID = serverID
	for i := range snap.Tools {
		snap.Tools[i].BundleID = bundleID
		snap.Tools[i].ServerID = serverID
	}
	for i := range snap.Resources {
		snap.Resources[i].BundleID = bundleID
		snap.Resources[i].ServerID = serverID
	}
	for i := range snap.ResourceTemplates {
		snap.ResourceTemplates[i].BundleID = bundleID
		snap.ResourceTemplates[i].ServerID = serverID
	}
	for i := range snap.Prompts {
		snap.Prompts[i].BundleID = bundleID
		snap.Prompts[i].ServerID = serverID
	}
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
	snap.Digest = computeDiscoverySnapshotDigest(*snap)
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
