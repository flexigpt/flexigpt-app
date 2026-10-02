package consumerapi

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/collection"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type SkillDirectoryRegistration struct {
	RootID            root.RootID `json:"rootID"`
	RootPath          string      `json:"rootPath"`
	SourceDisplayName string      `json:"sourceDisplayName"`
}

type SkillPathRegistration struct {
	RootID            root.RootID `json:"rootID"`
	Path              string      `json:"path"`
	SourceDisplayName string      `json:"sourceDisplayName,omitempty"`
	Enabled           bool        `json:"enabled"`
}

type SkillPathRegistrationResult struct {
	Source   source.Summary    `json:"source"`
	Artifact artifact.Artifact `json:"artifact"`
}

type ListSkillsRequest struct {
	RootID root.RootID `json:"rootID"`

	Enabled *bool `json:"enabled,omitempty"`
}

// SkillListItem is declaration metadata. Runtime registration, source files,
// arguments, resources, and execution state are intentionally not listing
// fields.
type SkillListItem struct {
	Ref artifact.ArtifactRef `json:"ref"`

	Name        basespec.LogicalName `json:"name"`
	DisplayName string               `json:"displayName"`
	Description string               `json:"description,omitempty"`

	State            artifact.State    `json:"state"`
	Enabled          bool              `json:"enabled"`
	Revision         uint64            `json:"revision"`
	DefinitionDigest cryptoutil.Digest `json:"definitionDigest,omitempty"`

	BuiltIn bool `json:"builtIn"`
	Managed bool `json:"managed"`
}

type ManagedSkillCreateRequest struct {
	Collection                 artifact.ArtifactRef `json:"collection"`
	ExpectedCollectionRevision uint64               `json:"expectedCollectionRevision"`
	SkillName                  string               `json:"skillName"`

	// SKILLMD and Files describe the logical Agent Skill directory supplied by
	// the caller. Files must contain SKILL.md at the logical package root.
	// The Store owns physical managed-package layout beneath that directory.
	SKILLMD []byte                      `json:"skillMD,omitempty"`
	Files   []source.ManagedPackageFile `json:"files,omitempty"`
	Enabled bool                        `json:"enabled"`
}

type ManagedSkillCreateResult struct {
	Artifact          artifact.Artifact         `json:"artifact"`
	Address           artifact.ArtifactAddress  `json:"address"`
	Collection        collection.CollectionView `json:"collection"`
	MembershipCreated bool                      `json:"membershipCreated"`
}

type ManagedSkillReplaceRequest struct {
	Collection                 artifact.ArtifactRef `json:"collection"`
	ExpectedCollectionRevision uint64               `json:"expectedCollectionRevision"`
	Artifact                   artifact.ArtifactRef `json:"artifact"`
	ExpectedArtifactRevision   uint64               `json:"expectedArtifactRevision"`
	SkillName                  string               `json:"skillName"`

	// SKILLMD and Files describe the logical Agent Skill directory supplied by
	// the caller. Files must contain SKILL.md at the logical package root.
	// The Store owns physical managed-package layout beneath that directory.
	SKILLMD []byte                      `json:"skillMD,omitempty"`
	Files   []source.ManagedPackageFile `json:"files,omitempty"`
	Enabled bool                        `json:"enabled"`
}

type ManagedSkillReplaceResult struct {
	Artifact   artifact.Artifact         `json:"artifact"`
	Address    artifact.ArtifactAddress  `json:"address"`
	Collection collection.CollectionView `json:"collection"`
}
