package consumerapi

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/adapter/mcp"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/adapter/prompt"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/adapter/skill"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/workspace/store/domain"
)

type WorkspaceRef = workspaceDomain.WorkspaceRef

type FilesystemSourceRegistration struct {
	RootID            root.RootID `json:"rootID"`
	RootPath          string      `json:"rootPath"`
	SourceDisplayName string      `json:"sourceDisplayName"`
}

type ManagedSourceRegistration struct {
	RootID            root.RootID `json:"rootID"`
	SourceDisplayName string      `json:"sourceDisplayName,omitempty"`
}

type WorkspacePathRegistration struct {
	RootID            root.RootID          `json:"rootID"`
	Path              string               `json:"path"`
	SourceDisplayName string               `json:"sourceDisplayName,omitempty"`
	WorkspaceName     basespec.LogicalName `json:"workspaceName,omitempty"`
}

type WorkspaceLoad struct {
	Workspace workspaceDomain.Workspace
	Members   []declaration.Entry

	resolved *resolve.ResolvedWorkspace
}

func (v WorkspaceLoad) ResolvedWorkspace() *resolve.ResolvedWorkspace {
	return v.resolved
}

type WorkspaceRefresh struct {
	Workspace WorkspaceRef
}

type WorkspacePathRegistrationResult struct {
	Source    source.Summary            `json:"source"`
	Workspace workspaceDomain.Workspace `json:"workspace"`
	Load      WorkspaceLoad             `json:"load"`
}

// WorkspaceCapabilityOccurrence preserves one Workspace member or nested
// composition occurrence, including unavailable and ambiguous relationships.
type WorkspaceCapabilityOccurrence = resolve.CapabilityOccurrence

// WorkspaceCapabilityPlan is the Artifact-backed projection of resolved
// Workspace members. It is a consumer plan, not a Store entity.
type WorkspaceCapabilityPlan struct {
	Workspace       WorkspaceRef                    `json:"workspace"`
	Occurrences     []WorkspaceCapabilityOccurrence `json:"occurrences"`
	PromptArtifacts []artifact.ArtifactRef          `json:"promptArtifacts"`
	SkillArtifacts  []artifact.ArtifactRef          `json:"skillArtifacts"`
	MCPArtifacts    []artifact.ArtifactRef          `json:"mcpArtifacts"`
	Complete        bool                            `json:"complete"`
}

type WorkspaceRuntimeSelection struct {
	PromptArtifacts []artifact.ArtifactRef `json:"promptArtifacts,omitempty"`
	SkillArtifacts  []artifact.ArtifactRef `json:"skillArtifacts,omitempty"`
	MCPArtifacts    []artifact.ArtifactRef `json:"mcpArtifacts,omitempty"`
	RequireComplete bool                   `json:"requireComplete,omitempty"`
}

// WorkspaceRuntimePlan contains source-verified material ready for existing
// prompt, Skill, and MCP runtime consumers. Execution remains outside this
// plan and outside Artifact Store.
type WorkspaceRuntimePlan struct {
	Workspace    workspaceDomain.Workspace `json:"workspace"`
	Capabilities WorkspaceCapabilityPlan   `json:"capabilities"`
	Prompt       prompt.Plan               `json:"prompt"`
	Skills       skill.LoadPlan            `json:"skills"`
	MCPServers   mcp.LoadPlan              `json:"mcpServers"`
}

type WorkspaceArtifactView struct {
	Artifact           artifact.ArtifactRef        `json:"artifact"`
	Revision           uint64                      `json:"revision"`
	DisplayName        string                      `json:"displayName"`
	Kind               artifact.ArtifactKind       `json:"kind"`
	LogicalName        basespec.LogicalName        `json:"logicalName"`
	LogicalVersion     basespec.LogicalVersion     `json:"logicalVersion,omitempty"`
	Enabled            bool                        `json:"enabled"`
	State              artifact.State              `json:"state"`
	SourceID           source.SourceID             `json:"sourceID"`
	Locator            basespec.Locator            `json:"locator"`
	SubresourceLocator basespec.SubresourceLocator `json:"subresourceLocator,omitempty"`
	RuntimeDisabled    bool                        `json:"runtimeDisabled"`
}
