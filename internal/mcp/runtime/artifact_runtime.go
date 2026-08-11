package runtime

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/mcp/auth"
	"github.com/flexigpt/flexigpt-app/internal/mcp/installation"
	"github.com/flexigpt/flexigpt-app/internal/mcp/server"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

const runtimeSnapshotTTL = time.Hour

type ClientSession interface {
	Close(ctx context.Context) error
	Ping(ctx context.Context) error

	Discover(
		ctx context.Context,
		config server.RuntimeConfig,
	) (spec.MCPDiscoverySnapshot, error)

	CallTool(
		ctx context.Context,
		toolName string,
		arguments map[string]any,
	) (*spec.InvokeMCPToolResponseBody, error)

	ReadResource(
		ctx context.Context,
		uri string,
	) (*spec.MCPReadResourceResponseBody, error)

	GetPrompt(
		ctx context.Context,
		name string,
		arguments map[string]string,
	) (*spec.MCPGetPromptResponseBody, error)

	Complete(
		ctx context.Context,
		request spec.MCPCompleteArgumentRequestBody,
	) (*spec.MCPCompletionResult, error)
}

type ClientFactory interface {
	Connect(
		ctx context.Context,
		config server.RuntimeConfig,
		auth auth.ResolvedTransportAuth,
		notifications ClientNotificationSink,
	) (ClientSession, error)
}

type sessionState struct {
	server     artifact.ArtifactRef
	collection collection.CollectionRef
	version    cryptoutil.Digest
	config     server.RuntimeConfig

	status spec.MCPServerStatus
	client ClientSession

	snapshot          spec.MCPDiscoverySnapshot
	lastError         string
	lastConnectedAt   time.Time
	lastSyncedAt      time.Time
	snapshotExpiresAt time.Time
}

type MCPRuntimeManager struct {
	resolver    server.Resolver
	secrets     installation.SecretResolver
	environment installation.EnvironmentResolver
	authorizer  auth.ConnectionAuthorizer
	factory     ClientFactory

	mu       sync.RWMutex
	sessions map[artifact.ArtifactRef]*sessionState
	timers   map[artifact.ArtifactRef]*time.Timer
	closed   bool
}

func NewMCPRuntimeManager(
	resolver server.Resolver,
	secrets installation.SecretResolver,
	environment installation.EnvironmentResolver,
	authorizer auth.ConnectionAuthorizer,
	factory ClientFactory,
) (*MCPRuntimeManager, error) {
	if resolver == nil ||
		authorizer == nil ||
		factory == nil {
		return nil, fmt.Errorf(
			"%w: MCP runtime dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}

	return &MCPRuntimeManager{
		resolver:    resolver,
		secrets:     secrets,
		environment: environment,
		authorizer:  authorizer,
		factory:     factory,
		sessions:    map[artifact.ArtifactRef]*sessionState{},
		timers:      map[artifact.ArtifactRef]*time.Timer{},
	}, nil
}

// Connect resolves and verifies the full Artifact-backed server state once,
// materializes the runtime configuration, establishes the client session, and
// captures the session version.
//
// Tool calls do not repeat Source snapshot verification. Mutations explicitly
// invalidate sessions, while explicit Refresh performs a new full resolution.
func (m *MCPRuntimeManager) Connect(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (*spec.MCPServerRuntimeSnapshot, error) {
	if err := validateRuntimeRef(ctx, ref); err != nil {
		return nil, err
	}

	resolved, config, authConn, err := m.resolveForConnection(ctx, ref)
	if err != nil {
		return nil, err
	}

	client, err := m.factory.Connect(ctx, config, authConn, m)
	if err != nil {
		err = redactRuntimeError(
			err,
			config.SensitiveValues,
			authConn.SensitiveValues,
		)
		m.authorizer.ConnectionFailed(
			context.WithoutCancel(ctx),
			config,
			err,
		)
		m.setError(ref, err)
		return nil, err
	}

	snapshot, err := client.Discover(ctx, config)
	if err != nil {
		closeErr := client.Close(context.WithoutCancel(ctx))
		err = redactRuntimeError(
			errors.Join(err, closeErr),
			config.SensitiveValues,
			authConn.SensitiveValues,
		)
		m.authorizer.ConnectionFailed(
			context.WithoutCancel(ctx),
			config,
			err,
		)
		m.setError(ref, err)
		return nil, err
	}

	now := time.Now().UTC()
	normalizeSnapshot(&snapshot, ref, now)

	var previous ClientSession

	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		_ = client.Close(context.WithoutCancel(ctx))
		return nil, basespec.ErrClosed
	}

	if current := m.sessions[ref]; current != nil {
		previous = current.client
	}

	m.sessions[ref] = &sessionState{
		server:            ref,
		collection:        resolved.Collection,
		version:           resolved.Version,
		config:            cloneRuntimeConfig(config),
		status:            spec.MCPServerStatusReady,
		client:            client,
		snapshot:          cloneSnapshot(snapshot),
		lastConnectedAt:   now,
		lastSyncedAt:      now,
		snapshotExpiresAt: now.Add(runtimeSnapshotTTL),
	}
	m.mu.Unlock()

	if previous != nil && previous != client {
		_ = previous.Close(context.WithoutCancel(ctx))
	}

	m.authorizer.ConnectionSucceeded(
		context.WithoutCancel(ctx),
		config,
	)
	return m.Status(ctx, ref)
}

func (m *MCPRuntimeManager) Disconnect(
	ctx context.Context,
	ref artifact.ArtifactRef,
) error {
	if err := validateRuntimeRef(ctx, ref); err != nil {
		return err
	}

	state, timer := m.removeSession(ref)
	if timer != nil {
		timer.Stop()
	}
	if state == nil || state.client == nil {
		return nil
	}
	return state.client.Close(ctx)
}

func (m *MCPRuntimeManager) InvalidateCollection(
	ctx context.Context,
	ref collection.CollectionRef,
) error {
	if err := validateRuntimeCollectionRef(ctx, ref); err != nil {
		return err
	}

	m.mu.RLock()
	refs := make([]artifact.ArtifactRef, 0)
	for serverRef, state := range m.sessions {
		if state.collection == ref {
			refs = append(refs, serverRef)
		}
	}
	m.mu.RUnlock()

	sort.Slice(refs, func(left, right int) bool {
		if refs[left].RootID != refs[right].RootID {
			return refs[left].RootID < refs[right].RootID
		}
		return refs[left].ArtifactID < refs[right].ArtifactID
	})

	var output error
	for _, serverRef := range refs {
		output = errors.Join(output, m.Invalidate(ctx, serverRef))
	}
	return output
}

func (m *MCPRuntimeManager) ListTools(
	ctx context.Context,
	ref artifact.ArtifactRef,
) ([]spec.MCPToolCapability, error) {
	if err := validateRuntimeRef(ctx, ref); err != nil {
		return nil, err
	}
	snapshot, err := m.currentSnapshot(ref)
	if err != nil {
		return nil, err
	}
	output := append([]spec.MCPToolCapability(nil), snapshot.Tools...)
	sort.Slice(output, func(left, right int) bool {
		return output[left].ToolName < output[right].ToolName
	})
	return output, nil
}

func (m *MCPRuntimeManager) ListResources(
	ctx context.Context,
	ref artifact.ArtifactRef,
) ([]spec.MCPResourceRef, error) {
	if err := validateRuntimeRef(ctx, ref); err != nil {
		return nil, err
	}
	snapshot, err := m.currentSnapshot(ref)
	if err != nil {
		return nil, err
	}
	output := append([]spec.MCPResourceRef(nil), snapshot.Resources...)
	sort.Slice(output, func(left, right int) bool {
		return output[left].URI < output[right].URI
	})
	return output, nil
}

func (m *MCPRuntimeManager) ListResourceTemplates(
	ctx context.Context,
	ref artifact.ArtifactRef,
) ([]spec.MCPResourceTemplateRef, error) {
	if err := validateRuntimeRef(ctx, ref); err != nil {
		return nil, err
	}
	snapshot, err := m.currentSnapshot(ref)
	if err != nil {
		return nil, err
	}
	output := append(
		[]spec.MCPResourceTemplateRef(nil),
		snapshot.ResourceTemplates...,
	)
	sort.Slice(output, func(left, right int) bool {
		return output[left].URITemplate < output[right].URITemplate
	})
	return output, nil
}

func (m *MCPRuntimeManager) ListPrompts(
	ctx context.Context,
	ref artifact.ArtifactRef,
) ([]spec.MCPPromptRef, error) {
	if err := validateRuntimeRef(ctx, ref); err != nil {
		return nil, err
	}
	snapshot, err := m.currentSnapshot(ref)
	if err != nil {
		return nil, err
	}
	output := append([]spec.MCPPromptRef(nil), snapshot.Prompts...)
	sort.Slice(output, func(left, right int) bool {
		return output[left].PromptName < output[right].PromptName
	})
	return output, nil
}

func (m *MCPRuntimeManager) ReadResource(
	ctx context.Context,
	ref artifact.ArtifactRef,
	uri string,
) (*spec.MCPReadResourceResponseBody, error) {
	if err := validateRuntimeRef(ctx, ref); err != nil {
		return nil, err
	}
	if uri == "" {
		return nil, fmt.Errorf(
			"%w: MCP resource URI is required",
			spec.ErrMCPInvalidRequest,
		)
	}

	state, err := m.readySession(ref)
	if err != nil {
		return nil, err
	}
	return state.client.ReadResource(ctx, uri)
}

func (m *MCPRuntimeManager) GetPrompt(
	ctx context.Context,
	ref artifact.ArtifactRef,
	name string,
	arguments map[string]string,
) (*spec.MCPGetPromptResponseBody, error) {
	if err := validateRuntimeRef(ctx, ref); err != nil {
		return nil, err
	}
	if name == "" {
		return nil, fmt.Errorf(
			"%w: MCP prompt name is required",
			spec.ErrMCPInvalidRequest,
		)
	}

	state, err := m.readySession(ref)
	if err != nil {
		return nil, err
	}
	return state.client.GetPrompt(ctx, name, arguments)
}

func (m *MCPRuntimeManager) Complete(
	ctx context.Context,
	ref artifact.ArtifactRef,
	request spec.MCPCompleteArgumentRequestBody,
) (*spec.MCPCompletionResult, error) {
	if err := validateRuntimeRef(ctx, ref); err != nil {
		return nil, err
	}

	state, err := m.readySession(ref)
	if err != nil {
		return nil, err
	}
	return state.client.Complete(ctx, request)
}

func (m *MCPRuntimeManager) CallToolDryRun(
	ctx context.Context,
	ref artifact.ArtifactRef,
	request spec.InvokeMCPToolRequestBody,
) (server.RuntimeConfig, spec.MCPToolCapability, error) {
	if err := validateRuntimeRef(ctx, ref); err != nil {
		return server.RuntimeConfig{}, spec.MCPToolCapability{}, err
	}
	if request.ToolName == "" {
		return server.RuntimeConfig{},
			spec.MCPToolCapability{},
			fmt.Errorf("%w: MCP tool name is required", spec.ErrMCPInvalidRequest)
	}

	state, err := m.readySession(ref)
	if err != nil {
		return server.RuntimeConfig{}, spec.MCPToolCapability{}, err
	}

	tool, err := toolByName(state.snapshot, request.ToolName)
	if err != nil {
		return server.RuntimeConfig{}, spec.MCPToolCapability{}, err
	}
	if request.ToolDigest != "" &&
		request.ToolDigest != tool.Digest {
		override := state.config.ToolPolicies[tool.ToolName]
		if !override.AllowStaleDigest {
			return server.RuntimeConfig{},
				spec.MCPToolCapability{},
				fmt.Errorf(
					"%w: MCP tool digest changed",
					spec.ErrMCPStaleReference,
				)
		}
	}

	return cloneRuntimeConfig(state.config), cloneTool(tool), nil
}

func (m *MCPRuntimeManager) CallTool(
	ctx context.Context,
	ref artifact.ArtifactRef,
	request spec.InvokeMCPToolRequestBody,
) (*spec.InvokeMCPToolResponseBody, error) {
	if err := validateRuntimeRef(ctx, ref); err != nil {
		return nil, err
	}

	state, err := m.readySession(ref)
	if err != nil {
		return nil, err
	}
	tool, err := toolByName(state.snapshot, request.ToolName)
	if err != nil {
		return nil, err
	}
	if !tool.Enabled || tool.TaskSupport == spec.MCPTaskSupportRequired {
		return nil, fmt.Errorf(
			"%w: MCP tool %q is disabled or unsupported",
			spec.ErrMCPPolicyDenied,
			tool.ToolName,
		)
	}
	if request.ToolDigest != "" && request.ToolDigest != tool.Digest {
		override := state.config.ToolPolicies[tool.ToolName]
		if !override.AllowStaleDigest {
			return nil, fmt.Errorf(
				"%w: MCP tool digest changed",
				spec.ErrMCPStaleReference,
			)
		}
	}

	body, err := state.client.CallTool(
		ctx,
		request.ToolName,
		request.Arguments,
	)
	if err != nil {
		return nil, err
	}
	if body == nil {
		return nil, fmt.Errorf(
			"%w: MCP tool call returned no response",
			spec.ErrMCPRuntimeNotReady,
		)
	}

	body.Server = ref
	body.ToolName = request.ToolName
	body.ProviderToolName = request.ProviderToolName
	body.Provenance.Server = ref
	body.Provenance.Collection = state.collection
	body.Provenance.ServerDisplayName = state.config.DisplayName
	body.Provenance.ToolName = request.ToolName
	body.Provenance.ProviderToolName = request.ProviderToolName
	body.Provenance.ToolDigest = tool.Digest
	body.Provenance.ToolUseID = request.ToolUseID
	body.Provenance.ApprovalID = request.ApprovalID
	body.Provenance.AppInstanceID = request.AppInstanceID

	if state.config.AppsPolicy.Enabled &&
		tool.App != nil &&
		tool.App.ResourceURI != "" {
		body.Provenance.AppResourceURI = tool.App.ResourceURI
		body.App = &spec.MCPToolAppRenderInfo{
			ResourceURI: tool.App.ResourceURI,
		}
	}
	return body, nil
}

func (m *MCPRuntimeManager) OnClientNotification(
	ctx context.Context,
	event ClientNotification,
) {
	if m == nil || event.Server == (artifact.ArtifactRef{}) {
		return
	}

	switch event.Kind {
	case ClientNotificationToolListChanged,
		ClientNotificationResourceListChanged,
		ClientNotificationPromptListChanged:
		m.scheduleRefresh(ctx, event.Server)
	default:
	}
}

func (m *MCPRuntimeManager) Close(ctx context.Context) error {
	if m == nil {
		return nil
	}

	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	m.closed = true

	states := make([]*sessionState, 0, len(m.sessions))
	for _, state := range m.sessions {
		states = append(states, state)
	}
	timers := make([]*time.Timer, 0, len(m.timers))
	for _, timer := range m.timers {
		timers = append(timers, timer)
	}
	m.sessions = map[artifact.ArtifactRef]*sessionState{}
	m.timers = map[artifact.ArtifactRef]*time.Timer{}
	m.mu.Unlock()

	for _, timer := range timers {
		timer.Stop()
	}

	var output error
	for _, state := range states {
		if state.client == nil {
			continue
		}
		output = errors.Join(output, state.client.Close(ctx))
	}
	return output
}

func (m *MCPRuntimeManager) Refresh(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (*spec.MCPServerRuntimeSnapshot, error) {
	if err := validateRuntimeRef(ctx, ref); err != nil {
		return nil, err
	}

	state, found := m.session(ref)
	if !found || state.client == nil {
		return nil, fmt.Errorf(
			"%w: MCP server is not connected",
			spec.ErrMCPRuntimeNotReady,
		)
	}

	resolved, config, _, err := m.resolveForConnection(ctx, ref)
	if err != nil {
		m.setError(ref, err)
		return nil, err
	}
	if resolved.Version != state.version {
		_ = m.Invalidate(context.WithoutCancel(ctx), ref)
		return nil, fmt.Errorf(
			"%w: MCP server version changed; reconnect is required",
			spec.ErrMCPStaleReference,
		)
	}

	snapshot, err := state.client.Discover(ctx, config)
	if err != nil {
		m.setError(ref, err)
		return nil, err
	}

	now := time.Now().UTC()
	normalizeSnapshot(&snapshot, ref, now)

	m.mu.Lock()
	current := m.sessions[ref]
	if current == nil ||
		current.client != state.client ||
		current.version != resolved.Version {
		m.mu.Unlock()
		return nil, fmt.Errorf(
			"%w: MCP session changed during refresh",
			spec.ErrMCPRuntimeNotReady,
		)
	}
	current.config = cloneRuntimeConfig(config)
	current.snapshot = cloneSnapshot(snapshot)
	current.status = spec.MCPServerStatusReady
	current.lastError = ""
	current.lastSyncedAt = now
	current.snapshotExpiresAt = now.Add(runtimeSnapshotTTL)
	m.mu.Unlock()

	return m.Status(ctx, ref)
}

// Invalidate removes the derived runtime session after an MCP lifecycle,
// policy, installation-overlay, or secret-binding mutation.
//
// It deliberately does not resolve the server again. The mutation path has
// already established that the prior runtime version is obsolete.
func (m *MCPRuntimeManager) Invalidate(
	ctx context.Context,
	ref artifact.ArtifactRef,
) error {
	if err := validateRuntimeRef(ctx, ref); err != nil {
		return err
	}

	state, timer := m.removeSession(ref)
	if timer != nil {
		timer.Stop()
	}
	if state == nil || state.client == nil {
		return nil
	}

	closeCtx, cancel := context.WithTimeout(
		context.WithoutCancel(ctx),
		5*time.Second,
	)
	defer cancel()
	return state.client.Close(closeCtx)
}

func (m *MCPRuntimeManager) Status(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (*spec.MCPServerRuntimeSnapshot, error) {
	if err := validateRuntimeRef(ctx, ref); err != nil {
		return nil, err
	}

	state, found := m.session(ref)
	if !found {
		return &spec.MCPServerRuntimeSnapshot{
			Server: ref,
			Status: spec.MCPServerStatusDisconnected,
		}, nil
	}

	return runtimeSnapshot(state), nil
}

func (m *MCPRuntimeManager) resolveForConnection(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (server.Resolved, server.RuntimeConfig, auth.ResolvedTransportAuth, error) {
	resolved, err := m.resolver.ResolveMCPServer(ctx, ref)
	if err != nil {
		return server.Resolved{},
			server.RuntimeConfig{},
			auth.ResolvedTransportAuth{},
			err
	}
	config, err := resolved.MaterializeTrusted(
		ctx,
		m.secrets,
		m.environment,
	)
	if err != nil {
		return server.Resolved{},
			server.RuntimeConfig{},
			auth.ResolvedTransportAuth{},
			err
	}
	authConn, err := m.authorizer.PrepareConnection(ctx, config)
	if err != nil {
		return server.Resolved{},
			server.RuntimeConfig{},
			auth.ResolvedTransportAuth{},
			err
	}
	authConn.SensitiveValues = mergeSensitiveValues(
		config.SensitiveValues,
		authConn.SensitiveValues,
	)
	return resolved, config, authConn, nil
}

func (m *MCPRuntimeManager) currentSnapshot(
	ref artifact.ArtifactRef,
) (spec.MCPDiscoverySnapshot, error) {
	state, err := m.readySession(ref)
	if err != nil {
		return spec.MCPDiscoverySnapshot{}, err
	}
	return cloneSnapshot(state.snapshot), nil
}

func (m *MCPRuntimeManager) readySession(
	ref artifact.ArtifactRef,
) (*sessionState, error) {
	state, found := m.session(ref)
	if !found ||
		state.status != spec.MCPServerStatusReady ||
		state.client == nil {
		return nil, fmt.Errorf(
			"%w: MCP server is not connected",
			spec.ErrMCPRuntimeNotReady,
		)
	}
	return state, nil
}

func (m *MCPRuntimeManager) session(
	ref artifact.ArtifactRef,
) (*sessionState, bool) {
	m.mu.RLock()
	state := m.sessions[ref]
	m.mu.RUnlock()
	if state == nil {
		return nil, false
	}
	return cloneSessionState(state), true
}

func (m *MCPRuntimeManager) removeSession(
	ref artifact.ArtifactRef,
) (*sessionState, *time.Timer) {
	m.mu.Lock()
	state := m.sessions[ref]
	timer := m.timers[ref]
	delete(m.sessions, ref)
	delete(m.timers, ref)
	m.mu.Unlock()
	return state, timer
}

func (m *MCPRuntimeManager) setError(
	ref artifact.ArtifactRef,
	err error,
) {
	m.mu.Lock()
	defer m.mu.Unlock()

	state := m.sessions[ref]
	if state == nil {
		state = &sessionState{
			server: ref,
		}
		m.sessions[ref] = state
	}
	state.status = spec.MCPServerStatusError
	if err != nil {
		state.lastError = err.Error()
	}
}

func (m *MCPRuntimeManager) scheduleRefresh(
	ctx context.Context,
	ref artifact.ArtifactRef,
) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return
	}
	if timer := m.timers[ref]; timer != nil {
		timer.Reset(spec.NotificationRefreshDebounce)
		return
	}

	var timer *time.Timer
	timer = time.AfterFunc(spec.NotificationRefreshDebounce, func() {
		refreshCtx, cancel := context.WithTimeout(
			context.WithoutCancel(ctx),
			time.Minute,
		)
		defer cancel()

		m.mu.Lock()
		if m.timers[ref] == timer {
			delete(m.timers, ref)
		}
		m.mu.Unlock()

		_, _ = m.Refresh(refreshCtx, ref)
	})
	m.timers[ref] = timer
}

func validateRuntimeRef(
	ctx context.Context,
	ref artifact.ArtifactRef,
) error {
	if ctx == nil {
		return fmt.Errorf(
			"%w: MCP runtime context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return ref.Validate()
}

func validateRuntimeCollectionRef(
	ctx context.Context,
	ref collection.CollectionRef,
) error {
	if ctx == nil {
		return fmt.Errorf(
			"%w: MCP runtime context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return ref.Validate()
}

func runtimeSnapshot(
	state *sessionState,
) *spec.MCPServerRuntimeSnapshot {
	output := &spec.MCPServerRuntimeSnapshot{
		Server:                    state.server,
		Collection:                state.collection,
		Status:                    state.status,
		LastError:                 state.lastError,
		NegotiatedProtocolVersion: state.snapshot.NegotiatedProtocolVersion,
		ServerInfo:                cloneImplementationInfo(state.snapshot.ServerInfo),
		ServerCapabilities:        cloneCapabilities(state.snapshot.ServerCapabilities),
		Instructions:              state.snapshot.Instructions,
		ToolCount:                 len(state.snapshot.Tools),
		ResourceCount:             len(state.snapshot.Resources),
		ResourceTemplateCount:     len(state.snapshot.ResourceTemplates),
		PromptCount:               len(state.snapshot.Prompts),
		SnapshotDigest:            state.snapshot.Digest,
	}
	if !state.lastConnectedAt.IsZero() {
		output.LastConnectedAt = state.lastConnectedAt.Format(time.RFC3339Nano)
	}
	if !state.lastSyncedAt.IsZero() {
		output.LastSyncedAt = state.lastSyncedAt.Format(time.RFC3339Nano)
	}
	return output
}

func normalizeSnapshot(
	snapshot *spec.MCPDiscoverySnapshot,
	ref artifact.ArtifactRef,
	now time.Time,
) {
	if snapshot == nil {
		return
	}
	sort.Slice(snapshot.Tools, func(left, right int) bool {
		return snapshot.Tools[left].ToolName < snapshot.Tools[right].ToolName
	})
	sort.Slice(snapshot.Resources, func(left, right int) bool {
		return snapshot.Resources[left].URI < snapshot.Resources[right].URI
	})
	sort.Slice(snapshot.ResourceTemplates, func(left, right int) bool {
		return snapshot.ResourceTemplates[left].URITemplate <
			snapshot.ResourceTemplates[right].URITemplate
	})
	sort.Slice(snapshot.Prompts, func(left, right int) bool {
		return snapshot.Prompts[left].PromptName < snapshot.Prompts[right].PromptName
	})
	snapshot.Server = ref
	for index := range snapshot.Tools {
		snapshot.Tools[index].Server = ref
	}
	for index := range snapshot.Resources {
		snapshot.Resources[index].Server = ref
	}
	for index := range snapshot.ResourceTemplates {
		snapshot.ResourceTemplates[index].Server = ref
	}
	for index := range snapshot.Prompts {
		snapshot.Prompts[index].Server = ref
	}
	snapshot.Digest = computeDiscoverySnapshotDigest(*snapshot)
	snapshot.SyncedAt = now.UTC().Format(time.RFC3339Nano)
}

func toolByName(
	snapshot spec.MCPDiscoverySnapshot,
	name string,
) (spec.MCPToolCapability, error) {
	for _, tool := range snapshot.Tools {
		if tool.ToolName == name {
			return cloneTool(tool), nil
		}
	}
	return spec.MCPToolCapability{}, fmt.Errorf(
		"%w: MCP tool %q was not found",
		spec.ErrMCPInvalidRequest,
		name,
	)
}

func redactRuntimeError(err error, values ...[]string) error {
	return auth.RedactError(err, mergeSensitiveValues(values...))
}

func mergeSensitiveValues(groups ...[]string) []string {
	seen := make(map[string]struct{})
	values := make([]string, 0)

	for _, group := range groups {
		for _, value := range group {
			if value == "" || strings.TrimSpace(value) == "" {
				continue
			}
			if _, found := seen[value]; found {
				continue
			}
			seen[value] = struct{}{}
			values = append(values, value)
		}
	}

	sort.Strings(values)
	return values
}

func cloneSessionState(input *sessionState) *sessionState {
	if input == nil {
		return nil
	}
	output := *input
	output.config = cloneRuntimeConfig(input.config)
	output.snapshot = cloneSnapshot(input.snapshot)
	return &output
}

func cloneRuntimeConfig(input server.RuntimeConfig) server.RuntimeConfig {
	output := input
	if input.Stdio != nil {
		value := *input.Stdio
		value.Args = append([]string(nil), input.Stdio.Args...)
		value.Env = maps.Clone(input.Stdio.Env)
		output.Stdio = &value
	}
	if input.StreamableHTTP != nil {
		value := *input.StreamableHTTP
		value.Headers = maps.Clone(input.StreamableHTTP.Headers)
		output.StreamableHTTP = &value
	}
	output.ToolPolicies = maps.Clone(input.ToolPolicies)
	output.SensitiveValues = append([]string(nil), input.SensitiveValues...)
	return output
}

func cloneSnapshot(
	input spec.MCPDiscoverySnapshot,
) spec.MCPDiscoverySnapshot {
	output := input
	output.ServerInfo = cloneImplementationInfo(input.ServerInfo)
	output.ServerCapabilities = cloneCapabilities(input.ServerCapabilities)

	output.Tools = make([]spec.MCPToolCapability, len(input.Tools))
	for index, value := range input.Tools {
		output.Tools[index] = cloneTool(value)
	}

	output.Resources = append([]spec.MCPResourceRef(nil), input.Resources...)
	for index := range output.Resources {
		output.Resources[index].Annotations = maps.Clone(
			input.Resources[index].Annotations,
		)
	}

	output.ResourceTemplates = append(
		[]spec.MCPResourceTemplateRef(nil),
		input.ResourceTemplates...,
	)
	for index := range output.ResourceTemplates {
		output.ResourceTemplates[index].Arguments = maps.Clone(
			input.ResourceTemplates[index].Arguments,
		)
		output.ResourceTemplates[index].Annotations = maps.Clone(
			input.ResourceTemplates[index].Annotations,
		)
	}

	output.Prompts = append([]spec.MCPPromptRef(nil), input.Prompts...)
	for index := range output.Prompts {
		output.Prompts[index].Arguments = maps.Clone(
			input.Prompts[index].Arguments,
		)
	}
	return output
}

func cloneTool(input spec.MCPToolCapability) spec.MCPToolCapability {
	output := input
	output.InputSchema = maps.Clone(input.InputSchema)
	output.OutputSchema = maps.Clone(input.OutputSchema)
	if input.Annotations != nil {
		value := *input.Annotations
		output.Annotations = &value
	}
	if input.App != nil {
		value := *input.App
		value.Visibility = append([]string(nil), input.App.Visibility...)
		output.App = &value
	}
	return output
}

func cloneImplementationInfo(
	input *spec.MCPImplementationInfo,
) *spec.MCPImplementationInfo {
	if input == nil {
		return nil
	}
	output := *input
	return &output
}

func cloneCapabilities(
	input *spec.MCPServerCapabilitiesSummary,
) *spec.MCPServerCapabilitiesSummary {
	if input == nil {
		return nil
	}
	output := *input
	output.Experimental = maps.Clone(input.Experimental)
	output.Extensions = maps.Clone(input.Extensions)
	return &output
}
