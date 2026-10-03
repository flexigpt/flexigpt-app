package consumerapi

import (
	"context"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/collection"
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
			spec.ErrInvalid,
		)
	}
	return &WorkspaceServerResolver{api: api}, nil
}

func (r *WorkspaceServerResolver) ResolveMCPServer(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (mcpDomainServer.Resolved, error) {
	if r == nil || r.api == nil {
		return mcpDomainServer.Resolved{}, spec.ErrClosed
	}
	read, err := r.api.resolveMCPServer(ctx, ref)
	if err != nil {
		return mcpDomainServer.Resolved{}, err
	}
	return read.Resolved, nil
}

type BaselineEnsurer interface {
	EnsureMCPBaselineCollection(
		ctx context.Context,
		rootID rootModel.RootID,
	) (collection.CollectionView, error)
}

type baselineEnsurer struct {
	api *API
}

func NewBaselineEnsurer(api *API) (BaselineEnsurer, error) {
	if api == nil || api.collections == nil {
		return nil, fmt.Errorf(
			"%w: MCP baseline ensurer requires collections",
			spec.ErrInvalid,
		)
	}
	return &baselineEnsurer{api: api}, nil
}

func (s *baselineEnsurer) EnsureMCPBaselineCollection(
	ctx context.Context,
	rootID rootModel.RootID,
) (collection.CollectionView, error) {
	if s == nil || s.api == nil {
		return collection.CollectionView{}, spec.ErrClosed
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
			spec.ErrInvalid,
		)
	}
	return &ManagementStoreFacade{api: api}, nil
}

func (s *ManagementStoreFacade) ResolveMCPServer(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (ServerRead, error) {
	if s == nil || s.api == nil {
		return ServerRead{}, spec.ErrClosed
	}
	return s.api.resolveMCPServer(ctx, ref)
}

func (s *ManagementStoreFacade) GetServerSettings(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (ServerInstallationView, error) {
	return s.api.GetServerSettings(ctx, ref)
}

func (s *ManagementStoreFacade) GetMCPPolicy(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (PolicyView, error) {
	return s.api.GetMCPPolicy(ctx, ref)
}

func (s *ManagementStoreFacade) ListMCPCollectionServers(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) ([]ServerRead, error) {
	return s.api.ListMCPCollectionServers(ctx, ref)
}

func (s *ManagementStoreFacade) ListMCPServersReferencingPolicy(
	ctx context.Context,
	rootID rootModel.RootID,
	policyName spec.LogicalName,
) ([]artifactModel.ArtifactRef, error) {
	return s.api.ListMCPServersReferencingPolicy(ctx, rootID, policyName)
}

func (s *ManagementStoreFacade) CreateMCPServer(
	ctx context.Context,
	request ManagedMCPCreateRequest,
) (ManagedMCPCreateResult, error) {
	return s.api.CreateMCPServer(ctx, request)
}

func (s *ManagementStoreFacade) UpdateMCPServer(
	ctx context.Context,
	request ManagedMCPReplaceRequest,
) (ManagedMCPReplaceResult, error) {
	return s.api.UpdateMCPServer(ctx, request)
}

func (s *ManagementStoreFacade) DeleteMCPServer(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
) error {
	return s.api.DeleteMCPServer(ctx, ref, expectedRevision)
}

func (s *ManagementStoreFacade) SaveMCPPolicy(
	ctx context.Context,
	request ManagedMCPPolicyUpsertRequest,
) (ManagedMCPPolicyUpsertResult, error) {
	return s.api.SaveMCPPolicy(ctx, request)
}

func (s *ManagementStoreFacade) DeleteMCPPolicy(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
) error {
	return s.api.DeleteMCPPolicy(ctx, ref, expectedRevision)
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
			spec.ErrInvalid,
		)
	}
	return &CatalogStore{api: api}, nil
}

func (s *CatalogStore) ListMCPCollections(
	ctx context.Context,
	rootID rootModel.RootID,
) ([]collection.ListItem, error) {
	if s == nil || s.api == nil {
		return nil, spec.ErrClosed
	}
	return s.api.ListMCPCollections(ctx, rootID)
}

func (s *CatalogStore) ListServers(
	ctx context.Context,
	rootID rootModel.RootID,
) ([]ServerListItem, error) {
	if s == nil || s.api == nil {
		return nil, spec.ErrClosed
	}
	return s.api.ListServers(ctx, ListServersRequest{RootID: rootID})
}
