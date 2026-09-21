package consumerapi

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	workspaceRuntime "github.com/flexigpt/flexigpt-app/internal/workspace/runtime"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/workspace/store/domain"
)

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
	Workspace    workspaceDomain.WorkspaceView `json:"workspace"`
	Capabilities resolve.CapabilityPlan        `json:"capabilities"`
}

type WorkspaceRefresh struct {
	Workspace artifact.ArtifactRef `json:"workspace"`
}

type WorkspacePathRegistrationResult struct {
	Source    source.Summary                `json:"source"`
	Workspace workspaceDomain.WorkspaceView `json:"workspace"`
	Load      WorkspaceLoad                 `json:"load"`
}

type WorkspaceRuntimeSelection struct {
	PromptArtifacts []artifact.ArtifactRef `json:"promptArtifacts,omitempty"`
	SkillArtifacts  []artifact.ArtifactRef `json:"skillArtifacts,omitempty"`
	MCPArtifacts    []artifact.ArtifactRef `json:"mcpArtifacts,omitempty"`
	RequireComplete bool                   `json:"requireComplete,omitempty"`
}

type WorkspacePromptContribution struct {
	Artifact         artifact.ArtifactRef     `json:"artifact"`
	ArtifactRevision uint64                   `json:"artifactRevision"`
	DefinitionDigest cryptoutil.Digest        `json:"definitionDigest"`
	Kind             artifact.ArtifactKind    `json:"kind"`
	Name             string                   `json:"name"`
	Insert           declaration.InsertTarget `json:"insert"`
	MediaType        string                   `json:"mediaType,omitempty"`
	Locator          basespec.Locator         `json:"locator,omitempty"`
	OriginalBytes    int                      `json:"originalBytes"`
	IncludedBytes    int                      `json:"includedBytes"`
	Truncated        bool                     `json:"truncated"`
}

type WorkspacePromptDecision struct {
	Artifact      artifact.ArtifactRef               `json:"artifact"`
	Status        workspaceRuntime.CompositionStatus `json:"status"`
	Code          string                             `json:"code,omitempty"`
	OriginalBytes int                                `json:"originalBytes"`
	IncludedBytes int                                `json:"includedBytes"`
}

type WorkspacePromptPlan struct {
	Workspace     artifact.ArtifactRef          `json:"workspace"`
	Contributions []WorkspacePromptContribution `json:"contributions"`
	Prompt        string                        `json:"prompt"`
	Diagnostics   []diagnostic.Diagnostic       `json:"diagnostics,omitempty"`
	Decisions     []WorkspacePromptDecision     `json:"decisions"`
}

type WorkspaceSkill struct {
	Artifact         artifact.ArtifactRef `json:"artifact"`
	ArtifactRevision uint64               `json:"artifactRevision"`
	DefinitionDigest cryptoutil.Digest    `json:"definitionDigest"`
	Name             string               `json:"name"`
	DisplayName      string               `json:"displayName,omitempty"`
	Locator          basespec.Locator     `json:"locator,omitempty"`
	Version          string               `json:"version"`
	RuntimeDisabled  bool                 `json:"runtimeDisabled"`
}

type WorkspaceSkillLoadPlan struct {
	Workspace artifact.ArtifactRef `json:"workspace"`
	Skills    []WorkspaceSkill     `json:"skills"`
}

type WorkspaceMCPServer struct {
	Artifact         artifact.ArtifactRef `json:"artifact"`
	ArtifactRevision uint64               `json:"artifactRevision"`
	DefinitionDigest cryptoutil.Digest    `json:"definitionDigest"`
	Name             basespec.LogicalName `json:"name"`
	DisplayName      string               `json:"displayName,omitempty"`
	BuiltIn          bool                 `json:"builtIn"`
	Version          cryptoutil.Digest    `json:"version"`
}

type WorkspaceMCPServerLoadPlan struct {
	Workspace artifact.ArtifactRef `json:"workspace"`
	Servers   []WorkspaceMCPServer `json:"servers"`
}

// WorkspaceRuntimePlan contains consumer-safe runtime planning output.
// Runtime-only paths, source bytes, MCP installation state, secret
// references, and resolver graphs are deliberately excluded.
type WorkspaceRuntimePlan struct {
	Workspace    workspaceDomain.WorkspaceView `json:"workspace"`
	Capabilities resolve.CapabilityPlan        `json:"capabilities"`
	Prompt       WorkspacePromptPlan           `json:"prompt"`
	Skills       WorkspaceSkillLoadPlan        `json:"skills"`
	MCPServers   WorkspaceMCPServerLoadPlan    `json:"mcpServers"`
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
