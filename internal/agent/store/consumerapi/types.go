package consumerapi

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/agentv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type AgentView struct {
	Ref artifact.ArtifactRef `json:"ref"`

	Name        basespec.LogicalName `json:"name"`
	DisplayName string               `json:"displayName"`
	Description string               `json:"description,omitempty"`

	State            artifact.State    `json:"state"`
	Enabled          bool              `json:"enabled"`
	Revision         uint64            `json:"revision"`
	DefinitionDigest cryptoutil.Digest `json:"definitionDigest,omitempty"`
	BuiltIn          bool              `json:"builtIn"`
	Managed          bool              `json:"managed"`
}

// AgentListItem is the lightweight Agent list response. Document is omitted
// unless ListAgentsRequest.IncludeDocument is true.
type AgentListItem struct {
	AgentView

	Document *agentv1.AgentDocument `json:"document,omitempty"`
}

type AgentResolution struct {
	Agent        AgentView              `json:"agent"`
	Capabilities resolve.CapabilityPlan `json:"capabilities"`
}

type AgentTextMaterialization struct {
	Artifact         artifact.ArtifactRef     `json:"artifact"`
	ArtifactRevision uint64                   `json:"artifactRevision"`
	DefinitionDigest cryptoutil.Digest        `json:"definitionDigest"`
	Name             basespec.LogicalName     `json:"name"`
	Insert           declaration.InsertTarget `json:"insert"`
	MediaType        string                   `json:"mediaType,omitempty"`
	Content          string                   `json:"content"`
	Locator          basespec.Locator         `json:"locator"`
	BuiltIn          bool                     `json:"builtIn"`
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

	IncludeDocument bool `json:"includeDocument,omitempty"`
}

type ManagedAgentDeleteRequest struct {
	Agent            artifact.ArtifactRef `json:"agent"`
	ExpectedRevision uint64               `json:"expectedRevision"`
}
