package consumerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/collection"
	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/policy"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
)

// WorkspaceServerResolver is the only MCP capability that Workspace runtime
// planning needs. It intentionally exposes no installation, policy mutation,
// secret, overlay, or collection APIs.
type WorkspaceServerResolver struct {
	api *API
}

func NewWorkspaceServerResolver(
	api *API,
) (*WorkspaceServerResolver, error) {
	if api == nil {
		return nil, fmt.Errorf(
			"%w: Workspace MCP resolver requires an API",
			basespec.ErrInvalid,
		)
	}
	return &WorkspaceServerResolver{api: api}, nil
}

func (r *WorkspaceServerResolver) ResolveMCPServer(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (mcpDomainServer.Resolved, error) {
	if r == nil || r.api == nil {
		return mcpDomainServer.Resolved{}, basespec.ErrClosed
	}
	return r.api.resolveMCPServer(ctx, ref)
}

type builtinStore struct {
	api *API
}

func NewBuiltinStore(api *API) (BuiltinStore, error) {
	if api == nil {
		return nil, fmt.Errorf(
			"%w: MCP built-in Store requires an API",
			basespec.ErrInvalid,
		)
	}
	return &builtinStore{api: api}, nil
}

func (s *builtinStore) InstallBuiltInPackage(
	ctx context.Context,
	request BuiltInPackageInstallRequest,
) ([]artifact.Artifact, error) {
	if s == nil || s.api == nil {
		return nil, basespec.ErrClosed
	}
	return s.api.installBuiltInPackage(ctx, request)
}

func (s *builtinStore) RemoveBuiltInPackage(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	address source.ManagedPackageAddress,
) error {
	if s == nil || s.api == nil {
		return basespec.ErrClosed
	}
	return s.api.removeBuiltInPackage(
		ctx, rootID, sourceID, address,
	)
}

func (s *builtinStore) EnsureBuiltInSourceCurrent(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
) error {
	if s == nil || s.api == nil {
		return basespec.ErrClosed
	}
	return s.api.ensureBuiltInSourceCurrent(
		ctx, rootID, sourceID,
	)
}

type BaselineEnsurer interface {
	EnsureMCPBaselineCollection(
		ctx context.Context,
		rootID root.RootID,
	) (collection.CollectionView, error)
}

type baselineEnsurer struct {
	api *API
}

func NewBaselineEnsurer(api *API) (BaselineEnsurer, error) {
	if api == nil || api.collections == nil {
		return nil, fmt.Errorf(
			"%w: MCP baseline ensurer requires collections",
			basespec.ErrInvalid,
		)
	}
	return &baselineEnsurer{api: api}, nil
}

func (s *baselineEnsurer) EnsureMCPBaselineCollection(
	ctx context.Context,
	rootID root.RootID,
) (collection.CollectionView, error) {
	if s == nil || s.api == nil {
		return collection.CollectionView{}, basespec.ErrClosed
	}
	return s.api.ensureMCPBaselineCollection(ctx, rootID)
}

// ManagementStoreFacade is passed to runtime and aggregate orchestration.
// It is intentionally separate from the Wails-facing MCP consumer API.
type ManagementStoreFacade struct {
	api *API
}

func NewManagementStore(
	api *API,
) (*ManagementStoreFacade, error) {
	if api == nil {
		return nil, fmt.Errorf(
			"%w: MCP management Store requires an API",
			basespec.ErrInvalid,
		)
	}
	return &ManagementStoreFacade{api: api}, nil
}

func (s *ManagementStoreFacade) ResolveMCPServer(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (mcpDomainServer.Resolved, error) {
	if s == nil || s.api == nil {
		return mcpDomainServer.Resolved{}, basespec.ErrClosed
	}
	return s.api.resolveMCPServer(ctx, ref)
}

func (s *ManagementStoreFacade) GetServerInstallation(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ServerInstallationView, error) {
	return s.api.GetServerInstallation(ctx, ref)
}

func (s *ManagementStoreFacade) GetMCPPolicy(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (PolicyView, error) {
	return s.api.GetMCPPolicy(ctx, ref)
}

func (s *ManagementStoreFacade) GetMCPEffectivePolicy(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (mcpPolicy.Effective, error) {
	return s.api.GetMCPEffectivePolicy(ctx, ref)
}

func (s *ManagementStoreFacade) ListMCPServersReferencingPolicy(
	ctx context.Context,
	rootID root.RootID,
	policyName basespec.LogicalName,
) ([]artifact.ArtifactRef, error) {
	return s.api.ListMCPServersReferencingPolicy(ctx, rootID, policyName)
}

func (s *ManagementStoreFacade) CreateManagedMCP(
	ctx context.Context,
	request ManagedMCPCreateRequest,
) (ManagedMCPCreateResult, error) {
	return s.api.CreateManagedMCP(ctx, request)
}

func (s *ManagementStoreFacade) ReplaceManagedMCP(
	ctx context.Context,
	request ManagedMCPReplaceRequest,
) (ManagedMCPReplaceResult, error) {
	return s.api.ReplaceManagedMCP(ctx, request)
}

func (s *ManagementStoreFacade) PurgeManagedMCP(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
) error {
	return s.api.PurgeManagedMCP(ctx, ref, expectedRevision)
}

func (s *ManagementStoreFacade) UpsertManagedMCPPolicy(
	ctx context.Context,
	request ManagedMCPPolicyUpsertRequest,
) (ManagedMCPPolicyUpsertResult, error) {
	return s.api.UpsertManagedMCPPolicy(ctx, request)
}

func (s *ManagementStoreFacade) PurgeManagedMCPPolicy(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
) error {
	return s.api.PurgeManagedMCPPolicy(ctx, ref, expectedRevision)
}

// CatalogStore is the narrow cross-Root management query port consumed by
// mcp/management.Service.
type CatalogStore struct {
	api *API
}

func NewCatalogStore(api *API) (*CatalogStore, error) {
	if api == nil {
		return nil, fmt.Errorf(
			"%w: MCP catalog Store requires an API",
			basespec.ErrInvalid,
		)
	}
	return &CatalogStore{api: api}, nil
}

func (s *CatalogStore) ListMCPCollections(
	ctx context.Context,
	rootID root.RootID,
) ([]collection.CollectionView, error) {
	if s == nil || s.api == nil {
		return nil, basespec.ErrClosed
	}
	return s.api.ListMCPCollections(ctx, rootID)
}

func (s *CatalogStore) ListServers(
	ctx context.Context,
	rootID root.RootID,
) ([]artifact.Artifact, error) {
	if s == nil || s.api == nil {
		return nil, basespec.ErrClosed
	}
	return s.api.ListServers(ctx, rootID)
}
