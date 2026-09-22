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

const (
	WorkspaceDirectorySourceStorageKey  = "workspace-directory"
	WorkspaceBasePolicySourceStorageKey = "workspace-base-policy"
	WorkspaceRootStorageKeyPrefix       = "workspace-directory-root-"
)

type WorkspaceDirectoryRef struct {
	RootID root.RootID `json:"rootID"`
}

type WorkspaceDirectoryOrigin string

const (
	WorkspaceDirectoryOriginDefault  WorkspaceDirectoryOrigin = "default"
	WorkspaceDirectoryOriginManifest WorkspaceDirectoryOrigin = "manifest"
)

type WorkspaceDirectoryWorkspace struct {
	Workspace       workspaceDomain.WorkspaceView `json:"workspace"`
	Origin          WorkspaceDirectoryOrigin      `json:"origin"`
	ManifestLocator basespec.Locator              `json:"manifestLocator,omitempty"`
}

type WorkspaceDirectoryView struct {
	Ref             WorkspaceDirectoryRef         `json:"ref"`
	Root            root.Root                     `json:"root"`
	DirectorySource source.Summary                `json:"directorySource"`
	Enabled         bool                          `json:"enabled"`
	PolicyID        string                        `json:"policyID"`
	PolicyVersion   string                        `json:"policyVersion"`
	PolicyDigest    cryptoutil.Digest             `json:"policyDigest"`
	Workspaces      []WorkspaceDirectoryWorkspace `json:"workspaces"`
	Diagnostics     []diagnostic.Diagnostic       `json:"diagnostics,omitempty"`
}

type WorkspacePageRequest struct {
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

type WorkspacePage struct {
	Items      []WorkspaceDirectoryView `json:"items"`
	NextCursor string                   `json:"nextCursor,omitempty"`
}

type WorkspaceDefaultPolicyView struct {
	ID      string            `json:"policyID"`
	Version string            `json:"policyVersion"`
	Digest  cryptoutil.Digest `json:"policyDigest"`
	YAML    string            `json:"yaml"`
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
	Instructions  string                        `json:"instructions"`
	UserMessage   string                        `json:"userMessage"`
	Diagnostics   []diagnostic.Diagnostic       `json:"diagnostics,omitempty"`
	Decisions     []WorkspacePromptDecision     `json:"decisions"`
}

type WorkspaceSkill struct {
	Artifact         artifact.ArtifactRef     `json:"artifact"`
	ArtifactRevision uint64                   `json:"artifactRevision"`
	DefinitionDigest cryptoutil.Digest        `json:"definitionDigest"`
	Name             string                   `json:"name"`
	DisplayName      string                   `json:"displayName,omitempty"`
	Insert           declaration.InsertTarget `json:"insert,omitempty"`
	Locator          basespec.Locator         `json:"locator,omitempty"`
	Version          string                   `json:"version"`
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
}
