package agent

import (
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

type AgentView struct {
	Ref artifactModel.ArtifactRef `json:"ref"`

	Name        spec.LogicalName `json:"name"`
	DisplayName string           `json:"displayName"`
	Description string           `json:"description,omitempty"`

	State            artifactModel.State `json:"state"`
	Enabled          bool                `json:"enabled"`
	Revision         uint64              `json:"revision"`
	DefinitionDigest cryptoutil.Digest   `json:"definitionDigest,omitempty"`
	BuiltIn          bool                `json:"builtIn"`
	Managed          bool                `json:"managed"`
}

// AgentListItem is the consumer-facing Agent list projection. Declaration
// documents remain an internal storage and export concern.
type AgentListItem struct {
	AgentView
}

type AgentSkillUseMode string

const (
	AgentSkillUseModeAvailable    AgentSkillUseMode = "available"
	AgentSkillUseModeActive       AgentSkillUseMode = "active"
	AgentSkillUseModeInstructions AgentSkillUseMode = "instructions"
)

type AgentCapabilityOccurrence struct {
	Path     string                       `json:"path"`
	Type     declaration.Type             `json:"type"`
	Name     spec.LogicalName             `json:"name,omitempty"`
	Status   composition.ResolutionStatus `json:"status"`
	Required bool                         `json:"required"`

	Target              *composition.CapabilityTarget `json:"target,omitempty"`
	AutoExecute         *bool                         `json:"autoExecute,omitempty"`
	IncludeSystemPrompt *bool                         `json:"includeSystemPrompt,omitempty"`
	SkillUseMode        AgentSkillUseMode             `json:"skillUseMode,omitempty"`
	Code                string                        `json:"code,omitempty"`
	Message             string                        `json:"message,omitempty"`
}

type AgentCapabilityPlan struct {
	Occurrences []AgentCapabilityOccurrence `json:"occurrences"`
	Complete    bool                        `json:"complete"`
}

type AgentResolution struct {
	Agent        AgentView           `json:"agent"`
	Capabilities AgentCapabilityPlan `json:"capabilities"`
}

type ListAgentsRequest struct {
	RootID rootModel.RootID `json:"rootID"`

	LogicalNames []spec.LogicalName `json:"logicalNames,omitempty"`

	// Plugin limits the result to currently available direct Agent
	// relationships selected by one Agent Plugin Plugin.
	Plugin *artifactModel.ArtifactRef `json:"plugin,omitempty"`

	// IncludeBuiltin appends Agent Artifacts from the protected built-in Root
	// when RootID is not already the protected Root.
	IncludeBuiltin bool `json:"includeBuiltin,omitempty"`

	// Enabled filters universal Artifact.Enabled metadata when non-nil.
	Enabled *bool `json:"enabled,omitempty"`
}

type ManagedAgentDeleteRequest struct {
	Agent            artifactModel.ArtifactRef `json:"agent"`
	ExpectedRevision uint64                    `json:"expectedRevision"`
}
