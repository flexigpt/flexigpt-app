package workspace

import (
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/contextengine"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/domain"
)

const (
	WorkspaceDirectorySourceStorageKey  = "workspace-directory"
	WorkspaceBasePolicySourceStorageKey = "workspace-base-policy"
	WorkspaceRootStorageKeyPrefix       = "workspace-directory-root-"
)

type WorkspaceDirectoryRef struct {
	RootID rootModel.RootID `json:"rootID"`
}

type WorkspaceDirectoryOrigin string

const (
	WorkspaceDirectoryOriginDefault  WorkspaceDirectoryOrigin = "default"
	WorkspaceDirectoryOriginManifest WorkspaceDirectoryOrigin = "manifest"
)

type WorkspaceDirectoryWorkspace struct {
	Workspace       workspaceDomain.WorkspaceView `json:"workspace"`
	Origin          WorkspaceDirectoryOrigin      `json:"origin"`
	ManifestLocator spec.Locator                  `json:"manifestLocator,omitempty"`
}

type WorkspaceDirectoryView struct {
	Ref             WorkspaceDirectoryRef         `json:"ref"`
	Root            rootModel.Root                `json:"root"`
	DirectorySource sourceModel.Summary           `json:"directorySource"`
	Enabled         bool                          `json:"enabled"`
	PolicyID        string                        `json:"policyID"`
	PolicyVersion   string                        `json:"policyVersion"`
	PolicyDigest    cryptoutil.Digest             `json:"policyDigest"`
	Workspaces      []WorkspaceDirectoryWorkspace `json:"workspaces"`
	Diagnostics     []diagnostic.Diagnostic       `json:"diagnostics,omitempty"`
}

// WorkspaceDirectoryListItem is the lightweight directory-management list
// projection. It intentionally does not resolve Workspace manifests or build
// runtime composition plans.
type WorkspaceDirectoryListItem struct {
	Ref WorkspaceDirectoryRef `json:"ref"`

	RootID          rootModel.RootID `json:"rootID"`
	RootDisplayName string           `json:"rootDisplayName"`

	Enabled bool `json:"enabled"`

	DirectorySourceID       sourceModel.SourceID `json:"directorySourceID"`
	DirectorySourceRevision uint64               `json:"directorySourceRevision"`

	PolicyID      string            `json:"policyID"`
	PolicyVersion string            `json:"policyVersion"`
	PolicyDigest  cryptoutil.Digest `json:"policyDigest"`
}

type WorkspacePageRequest struct {
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

type WorkspacePage struct {
	Items      []WorkspaceDirectoryListItem `json:"items"`
	NextCursor string                       `json:"nextCursor,omitempty"`
}

type WorkspaceDefaultPolicyView struct {
	ID      string            `json:"policyID"`
	Version string            `json:"policyVersion"`
	Digest  cryptoutil.Digest `json:"policyDigest"`
	YAML    string            `json:"yaml"`
}

type WorkspaceRuntimeSelection struct {
	PromptArtifacts []artifactModel.ArtifactRef `json:"promptArtifacts,omitempty"`
	SkillArtifacts  []artifactModel.ArtifactRef `json:"skillArtifacts,omitempty"`
	MCPArtifacts    []artifactModel.ArtifactRef `json:"mcpArtifacts,omitempty"`
	RequireComplete bool                        `json:"requireComplete,omitempty"`
}

type WorkspacePromptContribution struct {
	Artifact         artifactModel.ArtifactRef  `json:"artifact"`
	ArtifactRevision uint64                     `json:"artifactRevision"`
	DefinitionDigest cryptoutil.Digest          `json:"definitionDigest"`
	Kind             artifactModel.ArtifactKind `json:"kind"`
	Name             string                     `json:"name"`
	Insert           declaration.InsertTarget   `json:"insert"`
	MediaType        string                     `json:"mediaType,omitempty"`
	Locator          spec.Locator               `json:"locator,omitempty"`
	OriginalBytes    int                        `json:"originalBytes"`
	IncludedBytes    int                        `json:"includedBytes"`
	Truncated        bool                       `json:"truncated"`
}

type WorkspacePromptDecision struct {
	Artifact      artifactModel.ArtifactRef       `json:"artifact"`
	Status        contextengine.CompositionStatus `json:"status"`
	Code          string                          `json:"code,omitempty"`
	OriginalBytes int                             `json:"originalBytes"`
	IncludedBytes int                             `json:"includedBytes"`
}

type WorkspacePromptPlan struct {
	Workspace     artifactModel.ArtifactRef     `json:"workspace"`
	Contributions []WorkspacePromptContribution `json:"contributions"`
	Instructions  string                        `json:"instructions"`
	UserMessage   string                        `json:"userMessage"`
	Diagnostics   []diagnostic.Diagnostic       `json:"diagnostics,omitempty"`
	Decisions     []WorkspacePromptDecision     `json:"decisions"`
}

type WorkspaceSkill struct {
	Artifact         artifactModel.ArtifactRef `json:"artifact"`
	ArtifactRevision uint64                    `json:"artifactRevision"`
	DefinitionDigest cryptoutil.Digest         `json:"definitionDigest"`
	Name             string                    `json:"name"`
	DisplayName      string                    `json:"displayName,omitempty"`
	Insert           declaration.InsertTarget  `json:"insert,omitempty"`
	Locator          spec.Locator              `json:"locator,omitempty"`
	Version          string                    `json:"version"`
}

type WorkspaceSkillLoadPlan struct {
	Workspace artifactModel.ArtifactRef `json:"workspace"`
	Skills    []WorkspaceSkill          `json:"skills"`
}

type WorkspaceMCPServer struct {
	Artifact         artifactModel.ArtifactRef `json:"artifact"`
	ArtifactRevision uint64                    `json:"artifactRevision"`
	DefinitionDigest cryptoutil.Digest         `json:"definitionDigest"`
	Name             spec.LogicalName          `json:"name"`
	DisplayName      string                    `json:"displayName,omitempty"`
	BuiltIn          bool                      `json:"builtIn"`
	Version          cryptoutil.Digest         `json:"version"`
}

type WorkspaceMCPServerLoadPlan struct {
	Workspace artifactModel.ArtifactRef `json:"workspace"`
	Servers   []WorkspaceMCPServer      `json:"servers"`
}

// WorkspaceRuntimePlan contains consumer-safe runtime planning output.
// Runtime-only paths, source bytes, MCP installation state, secret
// references, and resolver graphs are deliberately excluded.
type WorkspaceRuntimePlan struct {
	Workspace    workspaceDomain.WorkspaceView `json:"workspace"`
	Capabilities composition.CapabilityPlan    `json:"capabilities"`
	Prompt       WorkspacePromptPlan           `json:"prompt"`
	Skills       WorkspaceSkillLoadPlan        `json:"skills"`
	MCPServers   WorkspaceMCPServerLoadPlan    `json:"mcpServers"`
}

type WorkspaceArtifactView struct {
	Artifact           artifactModel.ArtifactRef  `json:"artifact"`
	Revision           uint64                     `json:"revision"`
	DisplayName        string                     `json:"displayName"`
	Kind               artifactModel.ArtifactKind `json:"kind"`
	LogicalName        spec.LogicalName           `json:"logicalName"`
	LogicalVersion     spec.LogicalVersion        `json:"logicalVersion,omitempty"`
	Enabled            bool                       `json:"enabled"`
	State              artifactModel.State        `json:"state"`
	SourceID           sourceModel.SourceID       `json:"sourceID"`
	Locator            spec.Locator               `json:"locator"`
	SubresourceLocator spec.SubresourceLocator    `json:"subresourceLocator,omitempty"`
}
