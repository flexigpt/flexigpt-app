package consumerapi

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/collection"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type AgentView struct {
	Artifact artifact.Artifact `json:"artifact"`

	Name        basespec.LogicalName `json:"name"`
	DisplayName string               `json:"displayName"`
	Description string               `json:"description,omitempty"`

	BuiltIn bool `json:"builtIn"`
	Managed bool `json:"managed"`
}

type AgentResolution struct {
	Agent        AgentView              `json:"agent"`
	Capabilities resolve.CapabilityPlan `json:"capabilities"`
}

type ListAgentsRequest struct {
	RootID root.RootID `json:"rootID"`

	LogicalNames []basespec.LogicalName `json:"logicalNames,omitempty"`

	// Collection limits the result to currently available direct Agent
	// relationships selected by one Agent Collection Plugin.
	Collection *artifact.ArtifactRef `json:"collection,omitempty"`

	// IncludeBuiltin appends Agent Artifacts from the protected built-in Root
	// when RootID is not already the protected Root.
	IncludeBuiltin bool `json:"includeBuiltin,omitempty"`

	// Enabled filters universal Artifact.Enabled metadata when non-nil.
	Enabled *bool `json:"enabled,omitempty"`
}

type AttachAgentToCollectionRequest struct {
	Collection       artifact.ArtifactRef `json:"collection"`
	ExpectedRevision uint64               `json:"expectedRevision"`
	Agent            artifact.ArtifactRef `json:"agent"`
}

type ManagedAgentCreateRequest struct {
	Collection                 artifact.ArtifactRef `json:"collection"`
	ExpectedCollectionRevision uint64               `json:"expectedCollectionRevision"`

	Document ManagedAgentDocument `json:"document"`
	Enabled  bool                 `json:"enabled"`
}

type ManagedAgentCreateResult struct {
	Agent             artifact.Artifact         `json:"agent"`
	Address           artifact.ArtifactAddress  `json:"address"`
	Collection        collection.CollectionView `json:"collection"`
	MembershipCreated bool                      `json:"membershipCreated"`
}

type ManagedAgentReplaceRequest struct {
	Agent            artifact.ArtifactRef `json:"agent"`
	ExpectedRevision uint64               `json:"expectedRevision"`
	Document         ManagedAgentDocument `json:"document"`
}

type ManagedAgentReplaceResult struct {
	Agent   artifact.Artifact        `json:"agent"`
	Address artifact.ArtifactAddress `json:"address"`
}

type ManagedAgentDeleteRequest struct {
	Agent            artifact.ArtifactRef `json:"agent"`
	ExpectedRevision uint64               `json:"expectedRevision"`
}

type BuiltInAgentArtifactExpectation struct {
	Locator          basespec.Locator            `json:"locator"`
	Subresource      basespec.SubresourceLocator `json:"subresource,omitempty"`
	Kind             artifact.ArtifactKind       `json:"kind"`
	LogicalName      basespec.LogicalName        `json:"logicalName"`
	LogicalVersion   basespec.LogicalVersion     `json:"logicalVersion,omitempty"`
	DefinitionDigest cryptoutil.Digest           `json:"definitionDigest"`
}

type BuiltInAgentPackageInstallRequest struct {
	RootID root.RootID `json:"rootID"`

	SourceID source.SourceID `json:"sourceID"`

	PackageAddress source.ManagedPackageAddress `json:"packageAddress"`

	PluginDocumentFile basespec.Locator `json:"pluginDocumentFile"`

	PackageFiles []source.ManagedPackageFile `json:"packageFiles"`

	Expectations []BuiltInAgentArtifactExpectation `json:"expectations"`
}
