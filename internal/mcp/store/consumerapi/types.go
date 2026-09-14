package consumerapi

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/policy"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
)

type ServerInstallationView struct {
	Artifact             artifact.Artifact              `json:"artifact"`
	Definition           definition.Definition          `json:"definition"`
	Document             mcpDomainServer.ServerDocument `json:"document"`
	Installation         mcpDomainServer.ServerData     `json:"installation"`
	InstallationRevision uint64                         `json:"installationRevision"`
	InstallationEnabled  bool                           `json:"installationEnabled"`
	RuntimeEnabled       bool                           `json:"runtimeEnabled"`
	BuiltIn              bool                           `json:"builtIn"`
}

type PolicyView struct {
	Artifact         artifact.Artifact     `json:"artifact"`
	Definition       definition.Definition `json:"definition"`
	Body             mcpPolicy.MCPPolicy   `json:"body"`
	EffectiveEnabled bool                  `json:"effectiveEnabled"`
	BuiltIn          bool                  `json:"builtIn"`
}

type BuiltInArtifactExpectation struct {
	Subresource      basespec.SubresourceLocator `json:"subresource"`
	Kind             artifact.ArtifactKind       `json:"kind"`
	LogicalName      basespec.LogicalName        `json:"logicalName"`
	DefinitionDigest cryptoutil.Digest           `json:"definitionDigest"`
	Enabled          bool                        `json:"enabled"`
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

	InspectMCPServerForRuntime(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (mcpDomainServer.Resolved, error)

	GetServerInstallation(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (ServerInstallationView, error)
}

type BuiltinStore interface {
	InstallBuiltInPackage(
		ctx context.Context,
		request BuiltInPackageInstallRequest,
	) ([]artifact.Artifact, error)

	EnsureBuiltInSourceCurrent(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
	) error
}
