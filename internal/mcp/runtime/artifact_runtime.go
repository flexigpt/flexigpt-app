package runtime

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
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
	"github.com/flexigpt/flexigpt-app/internal/mcp/server"
)

const (
	runtimeSnapshotTTL               = time.Hour
	defaultMCPConnectTimeout         = 30 * time.Second
	defaultInteractiveMCPAuthTimeout = 10 * time.Minute
	defaultMCPRefreshTimeout         = time.Minute
)

const (
	artifactDiscoveryPageKindTools             = "tools"
	artifactDiscoveryPageKindResources         = "resources"
	artifactDiscoveryPageKindResourceTemplates = "resourceTemplates"
	artifactDiscoveryPageKindPrompts           = "prompts"
)

type discoverySnapshotDigestPayload struct {
	Server                    artifact.ArtifactRef          `json:"server"`
	NegotiatedProtocolVersion string                        `json:"negotiatedProtocolVersion,omitempty"`
	ServerInfo                *MCPImplementationInfo        `json:"serverInfo,omitempty"`
	ServerCapabilities        *MCPServerCapabilitiesSummary `json:"serverCapabilities,omitempty"`
	Instructions              string                        `json:"instructions,omitempty"`
	Tools                     []MCPToolCapability           `json:"tools,omitempty"`
	Resources                 []MCPResourceRef              `json:"resources,omitempty"`
	ResourceTemplates         []MCPResourceTemplateRef      `json:"resourceTemplates,omitempty"`
	Prompts                   []MCPPromptRef                `json:"prompts,omitempty"`
}

type ClientSession interface {
	Close(ctx context.Context) error

	Discover(
		ctx context.Context,
		config server.RuntimeConfig,
	) (MCPDiscoverySnapshot, error)

	CallTool(
		ctx context.Context,
		toolName string,
		arguments map[string]any,
	) (*InvokeMCPToolResponseBody, error)

	ReadResource(
		ctx context.Context,
		uri string,
	) (*MCPReadResourceResponseBody, error)

	GetPrompt(
		ctx context.Context,
		name string,
		arguments map[string]string,
	) (*MCPGetPromptResponseBody, error)

	Complete(
		ctx context.Context,
		request MCPCompleteArgumentRequestBody,
	) (*MCPCompletionResult, error)
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
	generation uint64
	config     server.RuntimeConfig

	status MCPServerStatus
	client ClientSession

	snapshot          MCPDiscoverySnapshot
	lastError         string
	lastConnectedAt   time.Time
	lastSyncedAt      time.Time
	snapshotExpiresAt time.Time
}

type connectionAttempt struct {
	generation uint64
	cancel     context.CancelFunc
}

type MCPRuntimeManager struct {
	resolver    server.Resolver
	secrets     server.SecretResolver
	environment server.EnvironmentResolver
	authorizer  auth.ConnectionAuthorizer
	factory     ClientFactory

	lifecycleContext context.Context
	cancelLifecycle  context.CancelFunc

	mu          sync.RWMutex
	approvals   *ApprovalManager
	sessions    map[artifact.ArtifactRef]*sessionState
	generations map[artifact.ArtifactRef]uint64
	timers      map[artifact.ArtifactRef]*time.Timer
	attempts    map[artifact.ArtifactRef]connectionAttempt
	closed      bool
}

func NewMCPRuntimeManager(
	resolver server.Resolver,
	secrets server.SecretResolver,
	environment server.EnvironmentResolver,
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

	lifecycleContext, cancelLifecycle := context.WithCancel(context.Background())

	return &MCPRuntimeManager{
		resolver:         resolver,
		secrets:          secrets,
		environment:      environment,
		authorizer:       authorizer,
		factory:          factory,
		lifecycleContext: lifecycleContext,
		cancelLifecycle:  cancelLifecycle,
		sessions:         make(map[artifact.ArtifactRef]*sessionState),
		generations:      make(map[artifact.ArtifactRef]uint64),
		timers:           make(map[artifact.ArtifactRef]*time.Timer),
		attempts:         make(map[artifact.ArtifactRef]connectionAttempt),
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
) (*MCPServerRuntimeSnapshot, error) {
	if err := validateRuntimeRef(ctx, ref); err != nil {
		return nil, err
	}

	attemptContext, generation, previous, timer, err := m.beginConnection(
		ctx,
		ref,
	)
	if err != nil {
		return nil, err
	}
	defer m.finishConnectionAttempt(ref, generation)

	return m.connect(
		attemptContext,
		ref,
		generation,
		previous,
		timer,
	)
}

// StartConnect starts one managed connection attempt and immediately returns
// the Connecting snapshot. OAuth browser interaction and discovery continue in
// the runtime-owned context and can be cancelled by Disconnect or Invalidate.

//nolint:contextcheck // Background check.
func (m *MCPRuntimeManager) StartConnect(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (*MCPServerRuntimeSnapshot, error) {
	if err := validateRuntimeRef(ctx, ref); err != nil {
		return nil, err
	}

	attemptContext, generation, previous, timer, err := m.beginConnection(
		m.lifecycleContext,
		ref,
	)
	if err != nil {
		return nil, err
	}

	go func() {
		defer m.finishConnectionAttempt(ref, generation)

		if _, connectErr := m.connect(
			attemptContext,
			ref,
			generation,
			previous,
			timer,
		); connectErr != nil && !errors.Is(connectErr, context.Canceled) {
			slog.Debug("background MCP connection finished with an error", "server", ref, "error", connectErr)
		}
	}()

	return m.Status(ctx, ref)
}

func (m *MCPRuntimeManager) Disconnect(
	ctx context.Context,
	ref artifact.ArtifactRef,
) error {
	if err := validateRuntimeRef(ctx, ref); err != nil {
		return err
	}
	state, timer, cancelAttempt := m.disconnectSession(ref)

	if cancelAttempt != nil {
		cancelAttempt()
	}
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
) ([]MCPToolCapability, error) {
	if err := validateRuntimeRef(ctx, ref); err != nil {
		return nil, err
	}
	snapshot, err := m.currentSnapshot(ref)
	if err != nil {
		return nil, err
	}
	output := append([]MCPToolCapability{}, snapshot.Tools...)
	sort.Slice(output, func(left, right int) bool {
		return output[left].ToolName < output[right].ToolName
	})
	return output, nil
}

func (m *MCPRuntimeManager) ListToolsPage(
	ctx context.Context,
	ref artifact.ArtifactRef,
	pageSize int,
	pageToken string,
) ([]MCPToolCapability, *string, error) {
	if err := validateRuntimeRef(ctx, ref); err != nil {
		return nil, nil, err
	}
	snapshot, err := m.currentSnapshot(ref)
	if err != nil {
		return nil, nil, err
	}
	values := make([]MCPToolCapability, len(snapshot.Tools))
	for index, value := range snapshot.Tools {
		values[index] = cloneTool(value)
	}
	sort.Slice(values, func(left, right int) bool {
		return values[left].ToolName < values[right].ToolName
	})
	return paginateArtifactDiscoveryItems(
		ref,
		snapshot.Digest,
		artifactDiscoveryPageKindTools,
		values,
		pageSize,
		pageToken,
	)
}

func (m *MCPRuntimeManager) ListResources(
	ctx context.Context,
	ref artifact.ArtifactRef,
) ([]MCPResourceRef, error) {
	if err := validateRuntimeRef(ctx, ref); err != nil {
		return nil, err
	}
	snapshot, err := m.currentSnapshot(ref)
	if err != nil {
		return nil, err
	}
	output := append([]MCPResourceRef{}, snapshot.Resources...)
	sort.Slice(output, func(left, right int) bool {
		return output[left].URI < output[right].URI
	})
	return output, nil
}

func (m *MCPRuntimeManager) ListResourcesPage(
	ctx context.Context,
	ref artifact.ArtifactRef,
	pageSize int,
	pageToken string,
) ([]MCPResourceRef, *string, error) {
	if err := validateRuntimeRef(ctx, ref); err != nil {
		return nil, nil, err
	}
	snapshot, err := m.currentSnapshot(ref)
	if err != nil {
		return nil, nil, err
	}
	values := append([]MCPResourceRef(nil), snapshot.Resources...)
	sort.Slice(values, func(left, right int) bool {
		return values[left].URI < values[right].URI
	})
	return paginateArtifactDiscoveryItems(
		ref, snapshot.Digest, artifactDiscoveryPageKindResources,
		values, pageSize, pageToken,
	)
}

func (m *MCPRuntimeManager) ListResourceTemplates(
	ctx context.Context,
	ref artifact.ArtifactRef,
) ([]MCPResourceTemplateRef, error) {
	if err := validateRuntimeRef(ctx, ref); err != nil {
		return nil, err
	}
	snapshot, err := m.currentSnapshot(ref)
	if err != nil {
		return nil, err
	}
	output := append(
		[]MCPResourceTemplateRef{},
		snapshot.ResourceTemplates...,
	)
	sort.Slice(output, func(left, right int) bool {
		return output[left].URITemplate < output[right].URITemplate
	})
	return output, nil
}

func (m *MCPRuntimeManager) ListResourceTemplatesPage(
	ctx context.Context,
	ref artifact.ArtifactRef,
	pageSize int,
	pageToken string,
) ([]MCPResourceTemplateRef, *string, error) {
	if err := validateRuntimeRef(ctx, ref); err != nil {
		return nil, nil, err
	}
	snapshot, err := m.currentSnapshot(ref)
	if err != nil {
		return nil, nil, err
	}
	values := append(
		[]MCPResourceTemplateRef(nil),
		snapshot.ResourceTemplates...,
	)
	sort.Slice(values, func(left, right int) bool {
		return values[left].URITemplate < values[right].URITemplate
	})
	return paginateArtifactDiscoveryItems(
		ref, snapshot.Digest, artifactDiscoveryPageKindResourceTemplates,
		values, pageSize, pageToken,
	)
}

func (m *MCPRuntimeManager) ListPrompts(
	ctx context.Context,
	ref artifact.ArtifactRef,
) ([]MCPPromptRef, error) {
	if err := validateRuntimeRef(ctx, ref); err != nil {
		return nil, err
	}
	snapshot, err := m.currentSnapshot(ref)
	if err != nil {
		return nil, err
	}
	output := append([]MCPPromptRef{}, snapshot.Prompts...)
	sort.Slice(output, func(left, right int) bool {
		return output[left].PromptName < output[right].PromptName
	})
	return output, nil
}

func (m *MCPRuntimeManager) ListPromptsPage(
	ctx context.Context,
	ref artifact.ArtifactRef,
	pageSize int,
	pageToken string,
) ([]MCPPromptRef, *string, error) {
	if err := validateRuntimeRef(ctx, ref); err != nil {
		return nil, nil, err
	}
	snapshot, err := m.currentSnapshot(ref)
	if err != nil {
		return nil, nil, err
	}
	values := append([]MCPPromptRef(nil), snapshot.Prompts...)
	sort.Slice(values, func(left, right int) bool {
		return values[left].PromptName < values[right].PromptName
	})
	return paginateArtifactDiscoveryItems(
		ref, snapshot.Digest, artifactDiscoveryPageKindPrompts,
		values, pageSize, pageToken,
	)
}

func (m *MCPRuntimeManager) ReadResource(
	ctx context.Context,
	ref artifact.ArtifactRef,
	uri string,
) (*MCPReadResourceResponseBody, error) {
	if err := validateRuntimeRef(ctx, ref); err != nil {
		return nil, err
	}
	if uri == "" {
		return nil, fmt.Errorf(
			"%w: MCP resource URI is required",
			ErrMCPInvalidRuntimeRequest,
		)
	}

	state, err := m.readySession(ref)
	if err != nil {
		return nil, err
	}
	body, err := state.client.ReadResource(ctx, uri)
	if err != nil {
		return nil, redactRuntimeError(
			err,
			state.config.SensitiveValues,
		)
	}
	if body == nil {
		return nil, fmt.Errorf(
			"%w: MCP resource read returned no response",
			ErrMCPRuntimeNotReady,
		)
	}
	return body, nil
}

func (m *MCPRuntimeManager) GetPrompt(
	ctx context.Context,
	ref artifact.ArtifactRef,
	name string,
	arguments map[string]string,
) (*MCPGetPromptResponseBody, error) {
	if err := validateRuntimeRef(ctx, ref); err != nil {
		return nil, err
	}
	if name == "" {
		return nil, fmt.Errorf(
			"%w: MCP prompt name is required",
			ErrMCPInvalidRuntimeRequest,
		)
	}

	state, err := m.readySession(ref)
	if err != nil {
		return nil, err
	}
	body, err := state.client.GetPrompt(
		ctx,
		name,
		arguments,
	)
	if err != nil {
		return nil, redactRuntimeError(
			err,
			state.config.SensitiveValues,
		)
	}
	if body == nil {
		return nil, fmt.Errorf(
			"%w: MCP prompt read returned no response",
			ErrMCPRuntimeNotReady,
		)
	}
	return body, nil
}

func (m *MCPRuntimeManager) Complete(
	ctx context.Context,
	ref artifact.ArtifactRef,
	request MCPCompleteArgumentRequestBody,
) (*MCPCompletionResult, error) {
	if err := validateRuntimeRef(ctx, ref); err != nil {
		return nil, err
	}

	state, err := m.readySession(ref)
	if err != nil {
		return nil, err
	}
	body, err := state.client.Complete(ctx, request)
	if err != nil {
		return nil, redactRuntimeError(
			err,
			state.config.SensitiveValues,
		)
	}
	if body == nil {
		return nil, fmt.Errorf(
			"%w: MCP completion returned no response",
			ErrMCPRuntimeNotReady,
		)
	}
	return body, nil
}

func (m *MCPRuntimeManager) CallToolDryRun(
	ctx context.Context,
	ref artifact.ArtifactRef,
	request InvokeMCPToolRequestBody,
) (server.RuntimeConfig, MCPToolCapability, error) {
	if err := validateRuntimeRef(ctx, ref); err != nil {
		return server.RuntimeConfig{}, MCPToolCapability{}, err
	}
	if request.ToolName == "" {
		return server.RuntimeConfig{},
			MCPToolCapability{},
			fmt.Errorf("%w: MCP tool name is required", ErrMCPInvalidRuntimeRequest)
	}

	state, err := m.readySession(ref)
	if err != nil {
		return server.RuntimeConfig{}, MCPToolCapability{}, err
	}

	tool, err := toolByName(state.snapshot, request.ToolName)
	if err != nil {
		return server.RuntimeConfig{}, MCPToolCapability{}, err
	}
	if request.ToolDigest != "" &&
		request.ToolDigest != tool.Digest {
		override := state.config.ToolPolicies[tool.ToolName]
		if !override.AllowStaleDigest {
			return server.RuntimeConfig{},
				MCPToolCapability{},
				fmt.Errorf(
					"%w: MCP tool digest changed",
					ErrMCPStaleReference,
				)
		}
	}

	return cloneRuntimeConfig(state.config), cloneTool(tool), nil
}

func (m *MCPRuntimeManager) CallTool(
	ctx context.Context,
	ref artifact.ArtifactRef,
	request InvokeMCPToolRequestBody,
) (*InvokeMCPToolResponseBody, error) {
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
	if !tool.Enabled || tool.TaskSupport == MCPTaskSupportRequired {
		return nil, fmt.Errorf(
			"%w: MCP tool %q is disabled or unsupported",
			ErrMCPPolicyDenied,
			tool.ToolName,
		)
	}
	if request.ToolDigest != "" && request.ToolDigest != tool.Digest {
		override := state.config.ToolPolicies[tool.ToolName]
		if !override.AllowStaleDigest {
			return nil, fmt.Errorf(
				"%w: MCP tool digest changed",
				ErrMCPStaleReference,
			)
		}
	}

	body, err := state.client.CallTool(
		ctx,
		request.ToolName,
		request.Arguments,
	)
	if err != nil {
		return nil, redactRuntimeError(
			err,
			state.config.SensitiveValues,
		)
	}
	if body == nil {
		return nil, fmt.Errorf(
			"%w: MCP tool call returned no response",
			ErrMCPRuntimeNotReady,
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
	body.Provenance.ChoiceID = request.ChoiceID
	body.Provenance.ApprovalID = request.ApprovalID
	body.Provenance.AppInstanceID = request.AppInstanceID

	if state.config.AppsPolicy.Enabled &&
		tool.App != nil &&
		tool.App.ResourceURI != "" {
		body.Provenance.AppResourceURI = tool.App.ResourceURI
		if body.App == nil {
			body.App = &MCPToolAppRenderInfo{
				ResourceURI: tool.App.ResourceURI,
			}
		} else if body.App.ResourceURI == "" {
			body.App.ResourceURI = tool.App.ResourceURI
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
	case ClientNotificationResourceUpdated:
		slog.Debug(
			"MCP resource update notification received",
			"server", event.Server,
			"uri", event.ResourceURI,
		)

	case ClientNotificationProgress:
		slog.Debug(
			"MCP progress notification received",
			"server", event.Server,
			"progress", event.Progress,
			"total", event.Total,
			"message", event.Message,
		)
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

	cancelLifecycle := m.cancelLifecycle
	m.cancelLifecycle = nil
	approvals := m.approvals
	m.approvals = nil
	states := make([]*sessionState, 0, len(m.sessions))
	for ref, state := range m.sessions {
		m.generations[ref]++
		states = append(states, state)
	}
	timers := make([]*time.Timer, 0, len(m.timers))
	for _, timer := range m.timers {
		timers = append(timers, timer)
	}
	attemptCancels := make([]context.CancelFunc, 0, len(m.attempts))
	for _, attempt := range m.attempts {
		if attempt.cancel != nil {
			attemptCancels = append(attemptCancels, attempt.cancel)
		}
	}
	m.sessions = make(map[artifact.ArtifactRef]*sessionState)
	m.timers = make(map[artifact.ArtifactRef]*time.Timer)
	m.attempts = make(map[artifact.ArtifactRef]connectionAttempt)
	m.mu.Unlock()

	if approvals != nil {
		approvals.Clear()
	}
	if cancelLifecycle != nil {
		cancelLifecycle()
	}
	for _, cancelAttempt := range attemptCancels {
		cancelAttempt()
	}
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
) (*MCPServerRuntimeSnapshot, error) {
	if err := validateRuntimeRef(ctx, ref); err != nil {
		return nil, err
	}

	state, found := m.session(ref)
	if !found || state.client == nil {
		return nil, fmt.Errorf(
			"%w: MCP server is not connected",
			ErrMCPRuntimeNotReady,
		)
	}

	refreshCtx, cancel := withMCPRefreshTimeout(ctx)
	defer cancel()

	resolved, config, _, err := m.resolveForConnection(refreshCtx, ref)
	if err != nil {
		m.setErrorIfCurrent(ref, state.generation, err)
		return nil, err
	}
	if resolved.Version != state.version {
		_ = m.Invalidate(context.WithoutCancel(ctx), ref)
		return nil, fmt.Errorf(
			"%w: MCP server version changed; reconnect is required",
			ErrMCPStaleReference,
		)
	}

	snapshot, err := state.client.Discover(refreshCtx, config)
	if err != nil {
		err = redactRuntimeError(err, config.SensitiveValues)
		m.setErrorIfCurrent(ref, state.generation, err)
		return nil, err
	}

	now := time.Now().UTC()
	normalizeSnapshot(&snapshot, ref, now)

	m.mu.Lock()
	current := m.sessions[ref]
	if current == nil ||
		m.generations[ref] != state.generation ||
		current.generation != state.generation ||
		current.client != state.client ||
		current.version != resolved.Version {
		m.mu.Unlock()
		return nil, fmt.Errorf(
			"%w: MCP session changed during refresh",
			ErrMCPRuntimeNotReady,
		)
	}
	current.config = cloneRuntimeConfig(config)
	current.snapshot = cloneSnapshot(snapshot)
	current.status = MCPServerStatusReady
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

	state, timer, cancelAttempt := m.removeSession(ref)
	if cancelAttempt != nil {
		cancelAttempt()
	}
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
) (*MCPServerRuntimeSnapshot, error) {
	if err := validateRuntimeRef(ctx, ref); err != nil {
		return nil, err
	}

	state, found := m.session(ref)
	if !found {
		return &MCPServerRuntimeSnapshot{
			Server: ref,
			Status: MCPServerStatusDisconnected,
		}, nil
	}

	return runtimeSnapshot(state), nil
}

func (m *MCPRuntimeManager) connect(
	ctx context.Context,
	ref artifact.ArtifactRef,
	generation uint64,
	previous *sessionState,
	timer *time.Timer,
) (*MCPServerRuntimeSnapshot, error) {
	if timer != nil {
		timer.Stop()
	}
	if previous != nil && previous.client != nil {
		_ = closeMCPClient(ctx, previous.client)
	}

	resolved, config, authConn, err := m.resolveForConnection(ctx, ref)
	if err != nil {
		m.setErrorIfCurrent(ref, generation, err)
		return nil, err
	}
	m.setConnectingCollectionIfCurrent(ref, generation, resolved.Collection)
	if !m.connectionCurrent(ref, generation) {
		return nil, fmt.Errorf(
			"%w: MCP connection was superseded",
			ErrMCPRuntimeNotReady,
		)
	}

	connectCtx, cancel := withMCPConnectTimeout(ctx, config)
	defer cancel()

	client, err := m.factory.Connect(connectCtx, config, authConn, m)
	if err == nil && client == nil {
		err = fmt.Errorf(
			"%w: MCP client factory returned no session",
			ErrMCPRuntimeNotReady,
		)
	}
	if err != nil {
		err = redactRuntimeError(
			err,
			config.SensitiveValues,
			authConn.SensitiveValues,
		)
		if m.setErrorIfCurrent(ref, generation, err) {
			m.authorizer.ConnectionFailed(context.WithoutCancel(ctx), config, err)
		}
		return nil, err
	}

	snapshot, err := client.Discover(connectCtx, config)
	if err != nil {
		closeErr := closeMCPClient(ctx, client)
		err = redactRuntimeError(
			errors.Join(err, closeErr),
			config.SensitiveValues,
			authConn.SensitiveValues,
		)
		if m.setErrorIfCurrent(ref, generation, err) {
			m.authorizer.ConnectionFailed(context.WithoutCancel(ctx), config, err)
		}
		return nil, err
	}

	now := time.Now().UTC()
	normalizeSnapshot(&snapshot, ref, now)
	if !m.commitConnection(
		ref,
		generation,
		resolved,
		config,
		client,
		snapshot,
		now,
	) {
		_ = closeMCPClient(ctx, client)
		return nil, fmt.Errorf(
			"%w: MCP connection was superseded",
			ErrMCPRuntimeNotReady,
		)
	}

	m.authorizer.ConnectionSucceeded(
		context.WithoutCancel(ctx),
		config,
	)
	return m.Status(ctx, ref)
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
) (MCPDiscoverySnapshot, error) {
	state, found := m.session(ref)
	if !found {
		return MCPDiscoverySnapshot{}, fmt.Errorf(
			"%w: MCP server is not connected",
			ErrMCPRuntimeNotReady,
		)
	}
	if state.status == MCPServerStatusReady && state.client != nil {
		return cloneSnapshot(state.snapshot), nil
	}
	if state.snapshotStillValid(time.Now().UTC()) {
		return cloneSnapshot(state.snapshot), nil
	}
	return MCPDiscoverySnapshot{}, fmt.Errorf(
		"%w: MCP server has no current discovery snapshot",
		ErrMCPRuntimeNotReady,
	)
}

func (m *MCPRuntimeManager) readySession(
	ref artifact.ArtifactRef,
) (*sessionState, error) {
	state, found := m.session(ref)
	if !found ||
		state.status != MCPServerStatusReady ||
		state.client == nil {
		return nil, fmt.Errorf(
			"%w: MCP server is not connected",
			ErrMCPRuntimeNotReady,
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

func (m *MCPRuntimeManager) beginConnection(
	parent context.Context,
	ref artifact.ArtifactRef,
) (context.Context, uint64, *sessionState, *time.Timer, error) {
	attemptContext, cancelAttempt := context.WithCancel(parent)
	stopLifecycle := context.AfterFunc(m.lifecycleContext, cancelAttempt)
	cancelConnection := func() {
		stopLifecycle()
		cancelAttempt()
	}

	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		cancelConnection()
		return nil, 0, nil, nil, basespec.ErrClosed
	}

	previousAttempt := m.attempts[ref]
	m.generations[ref]++
	generation := m.generations[ref]
	previous := m.sessions[ref]
	timer := m.timers[ref]

	m.attempts[ref] = connectionAttempt{
		generation: generation,
		cancel:     cancelConnection,
	}
	delete(m.timers, ref)
	next := &sessionState{
		server:     ref,
		generation: generation,
		status:     MCPServerStatusConnecting,
	}
	if previous != nil {
		next.collection = previous.collection
		if previous.snapshotStillValid(time.Now().UTC()) {
			next.snapshot = cloneSnapshot(previous.snapshot)
			next.lastSyncedAt = previous.lastSyncedAt
			next.snapshotExpiresAt = previous.snapshotExpiresAt
		}
	}
	m.sessions[ref] = next
	m.mu.Unlock()

	if previousAttempt.cancel != nil {
		previousAttempt.cancel()
	}
	m.clearApprovalSession(ref)

	return attemptContext, generation, previous, timer, nil
}

func (m *MCPRuntimeManager) finishConnectionAttempt(
	ref artifact.ArtifactRef,
	generation uint64,
) {
	var cancelAttempt context.CancelFunc

	m.mu.Lock()
	attempt := m.attempts[ref]
	if attempt.generation == generation {
		cancelAttempt = attempt.cancel
		delete(m.attempts, ref)
	}
	m.mu.Unlock()

	if cancelAttempt != nil {
		cancelAttempt()
	}
}

// disconnectSession preserves a bounded process-local discovery snapshot for
// read-only capability display. Lifecycle mutation uses removeSession through
// Invalidate and therefore always drops the snapshot.
func (m *MCPRuntimeManager) disconnectSession(
	ref artifact.ArtifactRef,
) (*sessionState, *time.Timer, context.CancelFunc) {
	m.mu.Lock()
	m.generations[ref]++
	state := m.sessions[ref]
	timer := m.timers[ref]
	attempt := m.attempts[ref]
	delete(m.timers, ref)
	delete(m.attempts, ref)
	if state == nil {
		m.mu.Unlock()
		m.clearApprovalSession(ref)
		return nil, timer, attempt.cancel
	}

	closed := cloneSessionState(state)
	state.client = nil
	state.generation = m.generations[ref]
	state.status = MCPServerStatusDisconnected
	state.lastError = ""
	if !state.snapshotStillValid(time.Now().UTC()) {
		state.snapshot = MCPDiscoverySnapshot{Server: ref}
		state.lastSyncedAt = time.Time{}
		state.snapshotExpiresAt = time.Time{}
	}
	m.mu.Unlock()

	m.clearApprovalSession(ref)
	return closed, timer, attempt.cancel
}

func (m *MCPRuntimeManager) setConnectingCollectionIfCurrent(
	ref artifact.ArtifactRef,
	generation uint64,
	collectionRef collection.CollectionRef,
) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed || m.generations[ref] != generation {
		return
	}
	state := m.sessions[ref]
	if state == nil || state.generation != generation {
		return
	}
	state.collection = collectionRef
}

func (m *MCPRuntimeManager) connectionCurrent(
	ref artifact.ArtifactRef,
	generation uint64,
) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state := m.sessions[ref]
	return !m.closed &&
		state != nil &&
		state.generation == generation &&
		m.generations[ref] == generation
}

func (m *MCPRuntimeManager) commitConnection(
	ref artifact.ArtifactRef,
	generation uint64,
	resolved server.Resolved,
	config server.RuntimeConfig,
	client ClientSession,
	snapshot MCPDiscoverySnapshot,
	now time.Time,
) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	state := m.sessions[ref]
	if m.closed ||
		state == nil ||
		state.generation != generation ||
		m.generations[ref] != generation {
		return false
	}

	m.sessions[ref] = &sessionState{
		server:            ref,
		collection:        resolved.Collection,
		version:           resolved.Version,
		generation:        generation,
		config:            cloneRuntimeConfig(config),
		status:            MCPServerStatusReady,
		client:            client,
		snapshot:          cloneSnapshot(snapshot),
		lastConnectedAt:   now,
		lastSyncedAt:      now,
		snapshotExpiresAt: now.Add(runtimeSnapshotTTL),
	}
	return true
}

func (m *MCPRuntimeManager) removeSession(
	ref artifact.ArtifactRef,
) (*sessionState, *time.Timer, context.CancelFunc) {
	m.mu.Lock()
	m.generations[ref]++
	state := m.sessions[ref]
	timer := m.timers[ref]
	attempt := m.attempts[ref]
	delete(m.sessions, ref)
	delete(m.timers, ref)
	delete(m.attempts, ref)
	m.mu.Unlock()
	m.clearApprovalSession(ref)
	return state, timer, attempt.cancel
}

func (m *MCPRuntimeManager) bindApprovalManager(
	approvals *ApprovalManager,
) {
	if m == nil {
		return
	}

	m.mu.Lock()
	previous := m.approvals
	m.approvals = approvals
	closed := m.closed
	m.mu.Unlock()

	if previous != nil && previous != approvals {
		previous.Clear()
	}
	if closed && approvals != nil {
		approvals.Clear()
	}
}

func (m *MCPRuntimeManager) clearApprovalSession(
	ref artifact.ArtifactRef,
) {
	if m == nil {
		return
	}

	m.mu.RLock()
	approvals := m.approvals
	m.mu.RUnlock()

	if approvals != nil {
		approvals.ClearServer(ref)
	}
}

func (m *MCPRuntimeManager) setErrorIfCurrent(
	ref artifact.ArtifactRef,
	generation uint64,
	err error,
) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed || m.generations[ref] != generation {
		return false
	}
	state := m.sessions[ref]
	if state == nil {
		state = &sessionState{
			server:     ref,
			generation: generation,
		}
		m.sessions[ref] = state
	}
	if state.generation != generation {
		return false
	}
	state.status = MCPServerStatusError
	if err != nil {
		state.lastError = err.Error()
	}
	return true
}

func withMCPConnectTimeout(
	ctx context.Context,
	config server.RuntimeConfig,
) (context.Context, context.CancelFunc) {
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		return ctx, func() {}
	}

	timeout := defaultMCPConnectTimeout
	if config.Stdio != nil && config.Stdio.StartupTimeoutMS > 0 {
		timeout = time.Duration(config.Stdio.StartupTimeoutMS) *
			time.Millisecond
	}
	if config.StreamableHTTP != nil &&
		config.StreamableHTTP.TimeoutMS > 0 {
		timeout = time.Duration(config.StreamableHTTP.TimeoutMS) *
			time.Millisecond
	}
	if config.StreamableHTTP != nil &&
		config.StreamableHTTP.AuthMode == server.MCPHTTPAuthOAuth {
		timeout = max(timeout, defaultInteractiveMCPAuthTimeout)
	}
	return context.WithTimeout(ctx, timeout)
}

func withMCPRefreshTimeout(
	ctx context.Context,
) (context.Context, context.CancelFunc) {
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, defaultMCPRefreshTimeout)
}

func closeMCPClient(ctx context.Context, client ClientSession) error {
	if client == nil {
		return nil
	}
	closeCtx, cancel := context.WithTimeout(
		context.WithoutCancel(ctx),
		5*time.Second,
	)
	defer cancel()
	return client.Close(closeCtx)
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
		timer.Reset(NotificationRefreshDebounce)
		return
	}

	var timer *time.Timer
	timer = time.AfterFunc(NotificationRefreshDebounce, func() {
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
) *MCPServerRuntimeSnapshot {
	snapshot := state.snapshot
	if state.status != MCPServerStatusReady &&
		!state.snapshotStillValid(time.Now().UTC()) {
		snapshot = MCPDiscoverySnapshot{Server: state.server}
	}
	output := &MCPServerRuntimeSnapshot{
		Server:                    state.server,
		Collection:                state.collection,
		Status:                    state.status,
		LastError:                 state.lastError,
		NegotiatedProtocolVersion: snapshot.NegotiatedProtocolVersion,
		ServerInfo:                cloneImplementationInfo(snapshot.ServerInfo),
		ServerCapabilities:        cloneCapabilities(snapshot.ServerCapabilities),
		Instructions:              snapshot.Instructions,
		ToolCount:                 len(snapshot.Tools),
		ResourceCount:             len(snapshot.Resources),
		ResourceTemplateCount:     len(snapshot.ResourceTemplates),
		PromptCount:               len(snapshot.Prompts),
		SnapshotDigest:            snapshot.Digest,
	}
	if !state.lastConnectedAt.IsZero() {
		output.LastConnectedAt = state.lastConnectedAt.Format(time.RFC3339Nano)
	}
	if !state.lastSyncedAt.IsZero() {
		output.LastSyncedAt = state.lastSyncedAt.Format(time.RFC3339Nano)
	}
	return output
}

func (s *sessionState) snapshotStillValid(now time.Time) bool {
	if s == nil ||
		s.snapshot.Server == (artifact.ArtifactRef{}) ||
		s.snapshot.Digest == "" ||
		s.snapshotExpiresAt.IsZero() {
		return false
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return now.Before(s.snapshotExpiresAt)
}

func normalizeSnapshot(
	snapshot *MCPDiscoverySnapshot,
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

func computeDiscoverySnapshotDigest(snap MCPDiscoverySnapshot) string {
	raw, err := json.Marshal(discoverySnapshotDigestPayload{
		Server:                    snap.Server,
		NegotiatedProtocolVersion: snap.NegotiatedProtocolVersion,
		ServerInfo:                snap.ServerInfo,
		ServerCapabilities:        snap.ServerCapabilities,
		Instructions:              snap.Instructions,
		Tools:                     snap.Tools,
		Resources:                 snap.Resources,
		ResourceTemplates:         snap.ResourceTemplates,
		Prompts:                   snap.Prompts,
	})
	if err != nil {
		return ""
	}
	return string(cryptoutil.DigestBytes(raw))
}

func toolByName(
	snapshot MCPDiscoverySnapshot,
	name string,
) (MCPToolCapability, error) {
	for _, tool := range snapshot.Tools {
		if tool.ToolName == name {
			return cloneTool(tool), nil
		}
	}
	return MCPToolCapability{}, fmt.Errorf(
		"%w: MCP tool %q was not found",
		ErrMCPInvalidRuntimeRequest,
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

func paginateArtifactDiscoveryItems[T any](
	srv artifact.ArtifactRef,
	snapshotDigest string,
	kind string,
	items []T,
	pageSize int,
	pageToken string,
) (out []T, next *string, err error) {
	start := 0

	if pageToken != "" {
		token, err := decodeArtifactDiscoveryPageToken(pageToken)
		if err != nil {
			return nil, nil, fmt.Errorf(
				"%w: invalid MCP discovery page token",
				ErrMCPInvalidRuntimeRequest,
			)
		}
		if token.Server != srv ||
			token.SnapshotDigest != snapshotDigest ||
			token.Kind != kind {
			return nil, nil, fmt.Errorf(
				"%w: stale MCP discovery page token",
				ErrMCPStaleReference,
			)
		}
		if token.PageSize <= 0 ||
			token.PageSize > MaxMCPServerPageSize ||
			token.Index < 0 ||
			token.Index > len(items) {
			return nil, nil, fmt.Errorf(
				"%w: invalid MCP discovery page token",
				ErrMCPInvalidRuntimeRequest,
			)
		}

		start = token.Index
		pageSize = token.PageSize
	} else {
		if pageSize <= 0 {
			pageSize = DefaultMCPPageSize
		}
		if pageSize > MaxMCPServerPageSize {
			pageSize = MaxMCPServerPageSize
		}
	}

	end := min(start+pageSize, len(items))
	out = append([]T(nil), items[start:end]...)
	if out == nil {
		out = []T{}
	}

	if end < len(items) {
		token, err := encodeArtifactDiscoveryPageToken(
			MCPDiscoveryPageToken{
				Server:         srv,
				SnapshotDigest: snapshotDigest,
				Kind:           kind,
				PageSize:       pageSize,
				Index:          end,
			},
		)
		if err != nil {
			return nil, nil, err
		}
		next = &token
	}

	return out, next, nil
}

func encodeArtifactDiscoveryPageToken(
	token MCPDiscoveryPageToken,
) (string, error) {
	raw, err := json.Marshal(token)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodeArtifactDiscoveryPageToken(
	value string,
) (MCPDiscoveryPageToken, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return MCPDiscoveryPageToken{}, err
	}

	var token MCPDiscoveryPageToken
	if err := json.Unmarshal(raw, &token); err != nil {
		return MCPDiscoveryPageToken{}, err
	}
	return token, nil
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
	input MCPDiscoverySnapshot,
) MCPDiscoverySnapshot {
	output := input
	output.ServerInfo = cloneImplementationInfo(input.ServerInfo)
	output.ServerCapabilities = cloneCapabilities(input.ServerCapabilities)

	output.Tools = make([]MCPToolCapability, len(input.Tools))
	for index, value := range input.Tools {
		output.Tools[index] = cloneTool(value)
	}

	output.Resources = append([]MCPResourceRef(nil), input.Resources...)
	for index := range output.Resources {
		output.Resources[index].Annotations = maps.Clone(
			input.Resources[index].Annotations,
		)
	}

	output.ResourceTemplates = append(
		[]MCPResourceTemplateRef(nil),
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

	output.Prompts = append([]MCPPromptRef(nil), input.Prompts...)
	for index := range output.Prompts {
		output.Prompts[index].Arguments = maps.Clone(
			input.Prompts[index].Arguments,
		)
	}
	return output
}

func cloneTool(input MCPToolCapability) MCPToolCapability {
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
	input *MCPImplementationInfo,
) *MCPImplementationInfo {
	if input == nil {
		return nil
	}
	output := *input
	return &output
}

func cloneCapabilities(
	input *MCPServerCapabilitiesSummary,
) *MCPServerCapabilitiesSummary {
	if input == nil {
		return nil
	}
	output := *input
	output.Experimental = maps.Clone(input.Experimental)
	output.Extensions = maps.Clone(input.Extensions)
	return &output
}
