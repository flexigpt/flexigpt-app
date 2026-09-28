package consumerapi

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
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

// AgentListItem is the consumer-facing Agent list projection. Declaration
// documents remain an internal storage and export concern.
type AgentListItem struct {
	AgentView
}

type AgentCapabilityOccurrence struct {
	Path     string                   `json:"path"`
	Type     declaration.Type         `json:"type"`
	Name     basespec.LogicalName     `json:"name,omitempty"`
	Status   resolve.ResolutionStatus `json:"status"`
	Required bool                     `json:"required"`

	Artifact *artifact.ArtifactRef `json:"artifact,omitempty"`
	Code     string                `json:"code,omitempty"`
	Message  string                `json:"message,omitempty"`
}

type AgentCapabilityPlan struct {
	Occurrences []AgentCapabilityOccurrence `json:"occurrences"`
	Complete    bool                        `json:"complete"`
}

type AgentResolution struct {
	Agent        AgentView           `json:"agent"`
	Capabilities AgentCapabilityPlan `json:"capabilities"`
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
}

type ManagedAgentDeleteRequest struct {
	Agent            artifact.ArtifactRef `json:"agent"`
	ExpectedRevision uint64               `json:"expectedRevision"`
}
