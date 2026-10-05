package aggregate

import (
	"context"
	"errors"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	mcpConsumerAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/consumerapi"
	serverMCPDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/domain/server"
	mcpAuth "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/auth"
	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/policy"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/server"
)

type AuthState interface {
	ClearAuthStatus(server mcpServer.ServerID)

	BuildAuthHealth(
		ctx context.Context,
		config mcpServer.RuntimeConfig,
	) mcpAuth.MCPAuthHealth

	GetAuthStatus(
		server mcpServer.ServerID,
	) (mcpAuth.MCPAuthStatus, bool)

	PendingAuthorizations() []mcpAuth.MCPOAuthAuthorization
}

type RuntimeStatusReader interface {
	Status(
		ctx context.Context,
		server mcpServer.ServerID,
	) (*mcpServer.MCPServerRuntimeSnapshot, error)
}

// SecretStore is deliberately narrow. Aggregate owns secret writes and
// runtime invalidation while the application supplies persistence.
type SecretStore interface {
	ResolveSecret(ctx context.Context, ref string) (string, error)

	SetMCPSecret(
		ctx context.Context,
		ref string,
		value string,
	) (hash string, nonEmpty bool, err error)

	DeleteSecret(ctx context.Context, ref string) error
}

type Dependencies struct {
	Lifecycle *Lifecycle
	Servers   *ArtifactServerResolver
	Source    *RuntimeServerSource
	Store     mcpConsumerAPI.ManagementStore
	Runtime   RuntimeStatusReader
	Auth      AuthState
	Secrets   SecretStore
}

type Service struct {
	lifecycle *Lifecycle
	servers   *ArtifactServerResolver
	source    *RuntimeServerSource
	store     mcpConsumerAPI.ManagementStore
	runtime   RuntimeStatusReader
	auth      AuthState
	secrets   SecretStore
}

// MCPServerDetails is the one normal server read used by management callers.
// It intentionally composes existing Store, policy, auth, and runtime views.
type MCPServerDetails struct {
	Settings      mcpConsumerAPI.ServerInstallationView `json:"settings"`
	Policy        mcpPolicy.Effective                   `json:"policy"`
	Authorization mcpAuth.MCPAuthHealth                 `json:"authorization"`
	Connection    mcpServer.MCPServerRuntimeSnapshot    `json:"connection"`
}

// MCPServerRuntimeDetails contains only process-local runtime/auth observations.
// Configuration health and effective policy belong to the verified detail read.
type MCPServerRuntimeDetails struct {
	Ref                  artifactModel.ArtifactRef          `json:"ref"`
	Connection           mcpServer.MCPServerRuntimeSnapshot `json:"connection"`
	Authorization        *mcpAuth.MCPAuthStatus             `json:"authorization,omitempty"`
	PendingAuthorization *mcpAuth.MCPOAuthAuthorization     `json:"pendingAuthorization,omitempty"`
}

func NewService(dependencies Dependencies) (*Service, error) {
	if dependencies.Lifecycle == nil ||
		dependencies.Servers == nil ||
		dependencies.Source == nil ||
		dependencies.Store == nil ||
		dependencies.Runtime == nil ||
		dependencies.Auth == nil ||
		dependencies.Secrets == nil {
		return nil, errors.New("MCP aggregate dependencies are incomplete")
	}

	return &Service{
		lifecycle: dependencies.Lifecycle,
		servers:   dependencies.Servers,
		source:    dependencies.Source,
		store:     dependencies.Store,
		runtime:   dependencies.Runtime,
		auth:      dependencies.Auth,
		secrets:   dependencies.Secrets,
	}, nil
}

func (s *Service) GetMCPServer(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (MCPServerDetails, error) {
	if err := s.ready(); err != nil {
		return MCPServerDetails{}, err
	}

	read, err := s.store.ResolveMCPServer(ctx, ref)
	if err != nil {
		return MCPServerDetails{}, err
	}
	return s.serverDetails(ctx, read)
}

func (s *Service) ListMCPCollectionServers(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) ([]MCPServerDetails, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	reads, err := s.store.ListMCPCollectionServers(ctx, ref)
	if err != nil {
		return nil, err
	}
	output := make([]MCPServerDetails, 0, len(reads))
	for _, read := range reads {
		details, err := s.serverDetails(ctx, read)
		if err != nil {
			return nil, err
		}
		output = append(output, details)
	}
	return output, nil
}

func (s *Service) GetMCPServersForRuntimeServers(
	ctx context.Context,
	servers []mcpServer.ServerID,
) ([]MCPServerRuntimeDetails, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	if len(servers) > mcpServer.MaxMCPServerPageSize {
		return nil, fmt.Errorf(
			"%w: too many MCP runtime status requests",
			mcpServer.ErrInvalid,
		)
	}
	refs := make(map[mcpServer.ServerID]artifactModel.ArtifactRef, len(servers))
	for _, server := range servers {
		ref, err := artifactRefForRuntimeServerID(server)
		if err != nil {
			return nil, err
		}
		refs[server] = ref
	}
	pending := make(map[mcpServer.ServerID]mcpAuth.MCPOAuthAuthorization)
	for _, value := range s.auth.PendingAuthorizations() {
		pending[value.Server] = value
	}

	output := make([]MCPServerRuntimeDetails, 0, len(refs))
	seen := make(map[mcpServer.ServerID]struct{}, len(refs))
	for _, server := range servers {
		if _, duplicate := seen[server]; duplicate {
			continue
		}
		seen[server] = struct{}{}
		connection, err := s.runtime.Status(ctx, server)
		if err != nil {
			return nil, err
		}
		if connection == nil {
			return nil, mcpServer.ErrClosed
		}
		value := MCPServerRuntimeDetails{
			Ref:        refs[server],
			Connection: *connection,
		}
		if status, found := s.auth.GetAuthStatus(server); found {
			value.Authorization = &status
		}
		if authorization, found := pending[server]; found {
			value.PendingAuthorization = &authorization
		}
		output = append(output, value)
	}
	return output, nil
}

func (s *Service) SaveMCPServerSettings(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedSettingsRevision uint64,
	data serverMCPDomain.ServerData,
) (MCPServerDetails, error) {
	if err := s.ready(); err != nil {
		return MCPServerDetails{}, err
	}
	resolved, err := s.servers.InspectMCPServer(ctx, ref)
	if err != nil {
		return MCPServerDetails{}, err
	}
	data = retainServerSecretBindings(
		resolved.Installation,
		data,
		resolved.Document,
	)
	return s.saveMCPServerSettings(
		ctx,
		ref,
		expectedSettingsRevision,
		data,
	)
}

func (s *Service) CreateMCPServer(
	ctx context.Context,
	request mcpConsumerAPI.ManagedMCPCreateRequest,
) (mcpConsumerAPI.ManagedMCPCreateResult, error) {
	if err := s.ready(); err != nil {
		return mcpConsumerAPI.ManagedMCPCreateResult{}, err
	}

	result, err := s.store.CreateMCPServer(ctx, request)
	if err != nil {
		return mcpConsumerAPI.ManagedMCPCreateResult{}, err
	}
	if err := s.lifecycle.InvalidateServer(ctx, result.Artifact.Ref()); err != nil {
		return mcpConsumerAPI.ManagedMCPCreateResult{}, err
	}
	s.clearServerAuthStatus(result.Artifact.Ref())
	return result, nil
}

func (s *Service) UpdateMCPServer(
	ctx context.Context,
	request mcpConsumerAPI.ManagedMCPReplaceRequest,
) (mcpConsumerAPI.ManagedMCPReplaceResult, error) {
	if err := s.ready(); err != nil {
		return mcpConsumerAPI.ManagedMCPReplaceResult{}, err
	}
	if err := s.lifecycle.InvalidateServer(ctx, request.Artifact); err != nil {
		return mcpConsumerAPI.ManagedMCPReplaceResult{}, err
	}
	s.clearServerAuthStatus(request.Artifact)
	return s.store.UpdateMCPServer(ctx, request)
}

func (s *Service) DeleteMCPServer(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
) error {
	if err := s.ready(); err != nil {
		return err
	}
	if err := s.lifecycle.InvalidateServer(ctx, ref); err != nil {
		return err
	}
	s.clearServerAuthStatus(ref)
	return s.store.DeleteMCPServer(ctx, ref, expectedRevision)
}

func (s *Service) SaveMCPPolicy(
	ctx context.Context,
	request mcpConsumerAPI.ManagedMCPPolicyUpsertRequest,
) (mcpConsumerAPI.ManagedMCPPolicyUpsertResult, error) {
	if err := s.ready(); err != nil {
		return mcpConsumerAPI.ManagedMCPPolicyUpsertResult{}, err
	}
	if err := request.Plugin.Validate(); err != nil {
		return mcpConsumerAPI.ManagedMCPPolicyUpsertResult{}, err
	}
	if err := request.Name.Validate(); err != nil {
		return mcpConsumerAPI.ManagedMCPPolicyUpsertResult{}, err
	}

	affected, err := s.store.ListMCPServersReferencingPolicy(
		ctx,
		request.Plugin.RootID,
		request.Name,
	)
	if err != nil {
		return mcpConsumerAPI.ManagedMCPPolicyUpsertResult{}, err
	}
	if err := s.lifecycle.InvalidateServers(ctx, affected); err != nil {
		return mcpConsumerAPI.ManagedMCPPolicyUpsertResult{}, err
	}
	return s.store.SaveMCPPolicy(ctx, request)
}

func (s *Service) DeleteMCPPolicy(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
) error {
	if err := s.ready(); err != nil {
		return err
	}

	policy, err := s.store.GetMCPPolicy(ctx, ref)
	if err != nil {
		return err
	}
	affected, err := s.store.ListMCPServersReferencingPolicy(
		ctx,
		policy.Artifact.RootID,
		policy.Artifact.LogicalName,
	)
	if err != nil {
		return err
	}
	if err := s.lifecycle.InvalidateServers(ctx, affected); err != nil {
		return err
	}
	return s.store.DeleteMCPPolicy(ctx, ref, expectedRevision)
}

func (s *Service) serverDetails(
	ctx context.Context,
	read mcpConsumerAPI.ServerRead,
) (MCPServerDetails, error) {
	authorization, err := s.serverAuthHealth(ctx, read.Resolved)
	if err != nil {
		return MCPServerDetails{}, err
	}

	serverID, err := runtimeServerIDForArtifact(read.Resolved.Server)
	if err != nil {
		return MCPServerDetails{}, err
	}
	connection, err := s.runtime.Status(ctx, serverID)
	if err != nil {
		return MCPServerDetails{}, err
	}
	if connection == nil {
		return MCPServerDetails{}, mcpServer.ErrClosed
	}

	return MCPServerDetails{
		Settings:      read.Settings,
		Policy:        read.Resolved.Policy,
		Authorization: authorization,
		Connection:    *connection,
	}, nil
}

func retainServerSecretBindings(
	current serverMCPDomain.ServerData,
	next serverMCPDomain.ServerData,
	document serverMCPDomain.ServerDocument,
) serverMCPDomain.ServerData {
	output := next.Clone()
	if output.Inputs == nil {
		output.Inputs = map[string]serverMCPDomain.InputBinding{}
	}

	for name, declaration := range document.Configuration.Install.Inputs {
		switch declaration.Kind {
		case serverMCPDomain.InputSecret,
			serverMCPDomain.InputOAuthClientCredentials:
		default:
			continue
		}

		if binding, found := current.Inputs[name]; found {
			output.Inputs[name] = binding
			continue
		}
		delete(output.Inputs, name)
	}

	return output
}

func (s *Service) saveMCPServerSettings(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedSettingsRevision uint64,
	data serverMCPDomain.ServerData,
) (MCPServerDetails, error) {
	if err := s.lifecycle.SaveServerSettings(
		ctx,
		ref,
		expectedSettingsRevision,
		data,
	); err != nil {
		return MCPServerDetails{}, err
	}
	s.clearServerAuthStatus(ref)
	return s.GetMCPServer(ctx, ref)
}

func (s *Service) serverAuthHealth(
	ctx context.Context,
	resolved serverMCPDomain.Resolved,
) (mcpAuth.MCPAuthHealth, error) {
	config, err := s.source.InspectRuntimeConfig(ctx, resolved)
	if err == nil {
		return s.auth.BuildAuthHealth(ctx, config), nil
	}
	if ctx.Err() != nil {
		return mcpAuth.MCPAuthHealth{}, ctx.Err()
	}

	serverID, idErr := runtimeServerIDForArtifact(resolved.Server)
	if idErr != nil {
		return mcpAuth.MCPAuthHealth{}, idErr
	}
	mode, modeErr := runtimeHTTPAuthMode(
		resolved.Document.Configuration.Auth.Mode,
	)
	if modeErr != nil {
		return mcpAuth.MCPAuthHealth{}, modeErr
	}
	return mcpAuth.MCPAuthHealth{
		Server:     serverID,
		AuthMode:   mode,
		State:      mcpAuth.MCPAuthHealthStateNotConfigured,
		Configured: false,
		LastError:  "required MCP installation input is not configured",
	}, nil
}

func (s *Service) clearServerAuthStatus(ref artifactModel.ArtifactRef) {
	serverID, err := runtimeServerIDForArtifact(ref)
	if err == nil {
		s.auth.ClearAuthStatus(serverID)
	}
}

func (s *Service) ready() error {
	if s == nil ||
		s.lifecycle == nil ||
		s.servers == nil ||
		s.source == nil ||
		s.store == nil ||
		s.runtime == nil ||
		s.auth == nil ||
		s.secrets == nil {
		return mcpServer.ErrClosed
	}
	return nil
}
