package consumerapi

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/collection"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/policy"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
)

type ServerInstallationView struct {
	Artifact     artifact.Artifact              `json:"artifact"`
	Document     mcpDomainServer.ServerDocument `json:"document"`
	Installation mcpDomainServer.ServerData     `json:"installation"`

	// InstallationRevision is the optimistic-concurrency token for a write
	// through UpdateServerInstallation or UpdateProtectedServerInstallation.
	//
	// Mutable servers use their Artifact revision. Protected servers use the
	// persisted overlay revision, which is zero when no overlay exists yet.
	InstallationRevision uint64 `json:"installationRevision"`
	BuiltIn              bool   `json:"builtIn"`
}

// MCPCollectionServerView is the management-list projection for one available
// MCP Server reachable from a Collection. It deliberately excludes secrets,
// materialized connection values, runtime status, and discovery payloads.
type MCPCollectionServerView struct {
	Installation ServerInstallationView `json:"installation"`
	Policy       mcpPolicy.Effective    `json:"policy"`
}

type PolicyView struct {
	Artifact artifact.Artifact   `json:"artifact"`
	Body     mcpPolicy.MCPPolicy `json:"body"`
	BuiltIn  bool                `json:"builtIn"`
}

type ManagedMCPPolicyUpsertRequest struct {
	Collection                 artifact.ArtifactRef `json:"collection"`
	ExpectedCollectionRevision uint64               `json:"expectedCollectionRevision"`
	Name                       basespec.LogicalName `json:"name"`
	Description                string               `json:"description,omitempty"`
	Policy                     mcpPolicy.MCPPolicy  `json:"policy"`
	Enabled                    bool                 `json:"enabled"`
}

type ManagedMCPPolicyUpsertResult struct {
	Artifact          artifact.Artifact         `json:"artifact"`
	Address           artifact.ArtifactAddress  `json:"address"`
	Collection        collection.CollectionView `json:"collection"`
	MembershipCreated bool                      `json:"membershipCreated"`
}

type ManagedMCPCreateRequest struct {
	Collection                 artifact.ArtifactRef           `json:"collection"`
	ExpectedCollectionRevision uint64                         `json:"expectedCollectionRevision"`
	Document                   mcpDomainServer.ServerDocument `json:"document"`
	Enabled                    bool                           `json:"enabled"`
}

type ManagedMCPCreateResult struct {
	Artifact          artifact.Artifact         `json:"artifact"`
	Address           artifact.ArtifactAddress  `json:"address"`
	Collection        collection.CollectionView `json:"collection"`
	MembershipCreated bool                      `json:"membershipCreated"`
}

type ManagedMCPReplaceRequest struct {
	Collection                 artifact.ArtifactRef           `json:"collection"`
	ExpectedCollectionRevision uint64                         `json:"expectedCollectionRevision"`
	Artifact                   artifact.ArtifactRef           `json:"artifact"`
	ExpectedArtifactRevision   uint64                         `json:"expectedArtifactRevision"`
	Document                   mcpDomainServer.ServerDocument `json:"document"`
	Enabled                    bool                           `json:"enabled"`
}

type ManagedMCPReplaceResult struct {
	Artifact   artifact.Artifact         `json:"artifact"`
	Address    artifact.ArtifactAddress  `json:"address"`
	Collection collection.CollectionView `json:"collection"`
}

type BuiltInArtifactExpectation struct {
	Locator          basespec.Locator            `json:"locator"`
	Subresource      basespec.SubresourceLocator `json:"subresource"`
	Kind             artifact.ArtifactKind       `json:"kind"`
	LogicalName      basespec.LogicalName        `json:"logicalName"`
	DefinitionDigest cryptoutil.Digest           `json:"definitionDigest"`
}

type BuiltInPackageInstallRequest struct {
	RootID         root.RootID                  `json:"rootID"`
	SourceID       source.SourceID              `json:"sourceID"`
	PackageAddress source.ManagedPackageAddress `json:"packageAddress"`
	DocumentFile   basespec.Locator             `json:"documentFile"`
	PackageFiles   []source.ManagedPackageFile  `json:"packageFiles"`
	Expectations   []BuiltInArtifactExpectation `json:"expectations"`
}

type ServerStore interface {
	ResolveMCPServer(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (mcpDomainServer.Resolved, error)

	GetServerInstallation(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (ServerInstallationView, error)
}

// ManagementStore is the aggregate-facing MCP persistence port. It includes
// mutation operations whose callers must coordinate runtime invalidation.
type ManagementStore interface {
	ServerStore

	GetMCPPolicy(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (PolicyView, error)

	GetMCPEffectivePolicy(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (mcpPolicy.Effective, error)

	ListMCPServersReferencingPolicy(
		ctx context.Context,
		rootID root.RootID,
		policyName basespec.LogicalName,
	) ([]artifact.ArtifactRef, error)

	CreateManagedMCP(
		ctx context.Context,
		request ManagedMCPCreateRequest,
	) (ManagedMCPCreateResult, error)

	ReplaceManagedMCP(
		ctx context.Context,
		request ManagedMCPReplaceRequest,
	) (ManagedMCPReplaceResult, error)

	PurgeManagedMCP(ctx context.Context, ref artifact.ArtifactRef, expectedRevision uint64) error
	UpsertManagedMCPPolicy(
		ctx context.Context,
		request ManagedMCPPolicyUpsertRequest,
	) (ManagedMCPPolicyUpsertResult, error)
	PurgeManagedMCPPolicy(ctx context.Context, ref artifact.ArtifactRef, expectedRevision uint64) error
}

type BuiltinStore interface {
	InstallBuiltInPackage(
		ctx context.Context,
		request BuiltInPackageInstallRequest,
	) ([]artifact.Artifact, error)

	RemoveBuiltInPackage(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
		address source.ManagedPackageAddress,
	) error

	EnsureBuiltInSourceCurrent(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
	) error
}
