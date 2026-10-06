package skill

import (
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	pluginAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
)

type SkillDirectoryRegistration struct {
	RootID            rootModel.RootID `json:"rootID"`
	RootPath          string           `json:"rootPath"`
	SourceDisplayName string           `json:"sourceDisplayName"`
}

type SkillPathRegistration struct {
	RootID            rootModel.RootID `json:"rootID"`
	Path              string           `json:"path"`
	SourceDisplayName string           `json:"sourceDisplayName,omitempty"`
	Enabled           bool             `json:"enabled"`
}

type SkillPathRegistrationResult struct {
	Source   sourceModel.Summary    `json:"source"`
	Artifact artifactModel.Artifact `json:"artifact"`
}

type ListSkillsRequest struct {
	RootID rootModel.RootID `json:"rootID"`

	Enabled *bool `json:"enabled,omitempty"`
}

// SkillListItem is declaration metadata. Runtime registration, source files,
// arguments, resources, and execution state are intentionally not listing
// fields.
type SkillListItem struct {
	Ref artifactModel.ArtifactRef `json:"ref"`

	Name        spec.LogicalName `json:"name"`
	DisplayName string           `json:"displayName"`
	Description string           `json:"description,omitempty"`

	State            artifactModel.State `json:"state"`
	Enabled          bool                `json:"enabled"`
	Revision         uint64              `json:"revision"`
	DefinitionDigest cryptoutil.Digest   `json:"definitionDigest,omitempty"`

	BuiltIn bool `json:"builtIn"`
	Managed bool `json:"managed"`
}

type ManagedSkillCreateRequest struct {
	Plugin                 artifactModel.ArtifactRef `json:"plugin"`
	ExpectedPluginRevision uint64                    `json:"expectedPluginRevision"`
	SkillName              string                    `json:"skillName"`

	// SKILLMD and Files describe the logical Agent Skill directory supplied by
	// the caller. Files must contain SKILL.md at the logical package root.
	// The Store owns physical managed-package layout beneath that directory.
	SKILLMD []byte                                   `json:"skillMD,omitempty"`
	Files   []managedpackageModel.ManagedPackageFile `json:"files,omitempty"`
	Enabled bool                                     `json:"enabled"`
}

type ManagedSkillCreateResult struct {
	Artifact          artifactModel.Artifact        `json:"artifact"`
	Address           artifactModel.ArtifactAddress `json:"address"`
	Plugin            pluginAPI.PluginView          `json:"plugin"`
	MembershipCreated bool                          `json:"membershipCreated"`
}

type ManagedSkillReplaceRequest struct {
	Plugin                   artifactModel.ArtifactRef `json:"plugin"`
	ExpectedPluginRevision   uint64                    `json:"expectedPluginRevision"`
	Artifact                 artifactModel.ArtifactRef `json:"artifact"`
	ExpectedArtifactRevision uint64                    `json:"expectedArtifactRevision"`
	SkillName                string                    `json:"skillName"`

	// SKILLMD and Files describe the logical Agent Skill directory supplied by
	// the caller. Files must contain SKILL.md at the logical package root.
	// The Store owns physical managed-package layout beneath that directory.
	SKILLMD []byte                                   `json:"skillMD,omitempty"`
	Files   []managedpackageModel.ManagedPackageFile `json:"files,omitempty"`
	Enabled bool                                     `json:"enabled"`
}

type ManagedSkillReplaceResult struct {
	Artifact artifactModel.Artifact        `json:"artifact"`
	Address  artifactModel.ArtifactAddress `json:"address"`
	Plugin   pluginAPI.PluginView          `json:"plugin"`
}
