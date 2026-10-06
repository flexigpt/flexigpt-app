package spec

import (
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type WorkspaceConversationSelectionStatus string

const (
	WorkspaceConversationSelectionReady       WorkspaceConversationSelectionStatus = "ready"
	WorkspaceConversationSelectionPartial     WorkspaceConversationSelectionStatus = "partial"
	WorkspaceConversationSelectionUnavailable WorkspaceConversationSelectionStatus = "unavailable"
)

type WorkspaceConversationContextUsageStatus string

const (
	WorkspaceConversationContextUsageIncluded    WorkspaceConversationContextUsageStatus = "included"
	WorkspaceConversationContextUsageTruncated   WorkspaceConversationContextUsageStatus = "truncated"
	WorkspaceConversationContextUsageExcluded    WorkspaceConversationContextUsageStatus = "excluded"
	WorkspaceConversationContextUsageDenied      WorkspaceConversationContextUsageStatus = "denied"
	WorkspaceConversationContextUsageUnavailable WorkspaceConversationContextUsageStatus = "unavailable"
)

type WorkspaceConversationSkillUsageStatus string

const (
	WorkspaceConversationSkillUsageAvailable   WorkspaceConversationSkillUsageStatus = "available"
	WorkspaceConversationSkillUsageUnavailable WorkspaceConversationSkillUsageStatus = "unavailable"
)

type WorkspaceConversationResourceSelectionRef struct {
	Artifact         artifactModel.ArtifactRef `json:"artifact"`
	Name             string                    `json:"name,omitempty"`
	Locator          spec.Locator              `json:"locator,omitempty"`
	DefinitionDigest cryptoutil.Digest         `json:"definitionDigest,omitempty"`
	ArtifactRevision uint64                    `json:"artifactRevision,omitempty"`
}

// WorkspaceConversationSelection stores one user-selected Workspace Artifact and the
// explicitly selected Root-scoped Artifact resources for one conversation
// turn. No Plugin or Catalog identity is persisted.
type WorkspaceConversationSelection struct {
	Workspace         artifactModel.ArtifactRef                   `json:"workspace"`
	DisplayName       string                                      `json:"displayName,omitempty"`
	WorkspaceRevision uint64                                      `json:"workspaceRevision,omitempty"`
	ContextRefs       []WorkspaceConversationResourceSelectionRef `json:"contextRefs,omitempty"`
	SkillRefs         []WorkspaceConversationResourceSelectionRef `json:"skillRefs,omitempty"`
}

type WorkspaceConversationContextUsage struct {
	Artifact                 artifactModel.ArtifactRef               `json:"artifact"`
	Name                     string                                  `json:"name,omitempty"`
	Locator                  spec.Locator                            `json:"locator,omitempty"`
	SelectedDefinitionDigest cryptoutil.Digest                       `json:"selectedDefinitionDigest,omitempty"`
	UsedDefinitionDigest     cryptoutil.Digest                       `json:"usedDefinitionDigest,omitempty"`
	UsedArtifactRevision     uint64                                  `json:"usedArtifactRevision,omitempty"`
	Status                   WorkspaceConversationContextUsageStatus `json:"status"`
	Code                     string                                  `json:"code,omitempty"`
	OriginalBytes            int                                     `json:"originalBytes,omitempty"`
	IncludedBytes            int                                     `json:"includedBytes,omitempty"`
	Changed                  bool                                    `json:"changed,omitempty"`
	Diagnostics              []diagnostic.Diagnostic                 `json:"diagnostics,omitempty"`
}

type WorkspaceConversationSkillUsage struct {
	Artifact                 artifactModel.ArtifactRef             `json:"artifact"`
	Name                     string                                `json:"name,omitempty"`
	DisplayName              string                                `json:"displayName,omitempty"`
	Locator                  spec.Locator                          `json:"locator,omitempty"`
	SelectedDefinitionDigest cryptoutil.Digest                     `json:"selectedDefinitionDigest,omitempty"`
	UsedDefinitionDigest     cryptoutil.Digest                     `json:"usedDefinitionDigest,omitempty"`
	UsedArtifactRevision     uint64                                `json:"usedArtifactRevision,omitempty"`
	Status                   WorkspaceConversationSkillUsageStatus `json:"status"`
	Changed                  bool                                  `json:"changed,omitempty"`
	SessionAvailable         bool                                  `json:"sessionAvailable,omitempty"`
	Active                   bool                                  `json:"active,omitempty"`
	Advertised               bool                                  `json:"advertised,omitempty"`
	Diagnostics              []diagnostic.Diagnostic               `json:"diagnostics,omitempty"`
}

type WorkspaceConversationUsage struct {
	Workspace         artifactModel.ArtifactRef            `json:"workspace"`
	DisplayName       string                               `json:"displayName,omitempty"`
	WorkspaceRevision uint64                               `json:"workspaceRevision,omitempty"`
	Status            WorkspaceConversationSelectionStatus `json:"status"`
	Contexts          []WorkspaceConversationContextUsage  `json:"contexts,omitempty"`
	Skills            []WorkspaceConversationSkillUsage    `json:"skills,omitempty"`
	Diagnostics       []diagnostic.Diagnostic              `json:"diagnostics,omitempty"`
}

type WorkspaceConversationResolution struct {
	Usage        WorkspaceConversationUsage
	Instructions string
	UserMessage  string
}
