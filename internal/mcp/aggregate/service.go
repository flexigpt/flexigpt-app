package aggregate

import (
	"context"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	mcpAuth "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/auth"
	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/policy"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/server"
	mcpConsumerAPI "github.com/flexigpt/flexigpt-app/internal/mcp/store/consumerapi"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
)

type AuthState interface {
	ClearAuthStatus(server mcpServer.ServerID)

	BuildAuthHealth(
		ctx context.Context,
		config mcpServer.RuntimeConfig,
	) mcpAuth.MCPAuthHealth
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
	ref artifact.ArtifactRef,
) (MCPServerDetails, error) {
	if err := s.ready(); err != nil {
		return MCPServerDetails{}, err
	}

	settings, err := s.store.GetServerSettings(ctx, ref)
	if err != nil {
		return MCPServerDetails{}, err
	}
	policy, err := s.store.GetServerEffectivePolicy(ctx, ref)
	if err != nil {
		return MCPServerDetails{}, err
	}
	authorization, err := s.serverAuthHealth(ctx, ref)
	if err != nil {
		return MCPServerDetails{}, err
	}

	serverID, err := runtimeServerIDForArtifact(ref)
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
		Settings:      settings,
		Policy:        policy,
		Authorization: authorization,
		Connection:    *connection,
	}, nil
}

func (s *Service) GetMCPServerForRuntimeServer(
	ctx context.Context,
	serverID mcpServer.ServerID,
) (MCPServerDetails, error) {
	if err := s.ready(); err != nil {
		return MCPServerDetails{}, err
	}
	ref, err := artifactRefForRuntimeServerID(serverID)
	if err != nil {
		return MCPServerDetails{}, err
	}
	return s.GetMCPServer(ctx, ref)
}

func (s *Service) SaveMCPServerSettings(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedSettingsRevision uint64,
	data mcpDomainServer.ServerData,
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
	ref artifact.ArtifactRef,
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
	if err := request.Collection.Validate(); err != nil {
		return mcpConsumerAPI.ManagedMCPPolicyUpsertResult{}, err
	}
	if err := request.Name.Validate(); err != nil {
		return mcpConsumerAPI.ManagedMCPPolicyUpsertResult{}, err
	}

	affected, err := s.store.ListMCPServersReferencingPolicy(
		ctx,
		request.Collection.RootID,
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
	ref artifact.ArtifactRef,
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

func retainServerSecretBindings(
	current mcpDomainServer.ServerData,
	next mcpDomainServer.ServerData,
	document mcpDomainServer.ServerDocument,
) mcpDomainServer.ServerData {
	output := next.Clone()
	if output.Inputs == nil {
		output.Inputs = map[string]mcpDomainServer.InputBinding{}
	}

	for name, declaration := range document.Configuration.Install.Inputs {
		switch declaration.Kind {
		case mcpDomainServer.InputSecret,
			mcpDomainServer.InputOAuthClientCredentials:
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
	ref artifact.ArtifactRef,
	expectedSettingsRevision uint64,
	data mcpDomainServer.ServerData,
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
	ref artifact.ArtifactRef,
) (mcpAuth.MCPAuthHealth, error) {
	config, resolved, err := s.source.InspectRuntimeConfig(ctx, ref)
	if err == nil {
		return s.auth.BuildAuthHealth(ctx, config), nil
	}
	if resolved.Server != ref {
		return mcpAuth.MCPAuthHealth{}, err
	}

	serverID, idErr := runtimeServerIDForArtifact(ref)
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

func (s *Service) clearServerAuthStatus(ref artifact.ArtifactRef) {
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
