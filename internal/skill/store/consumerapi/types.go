package consumerapi

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
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

type ManagedSkillCreateRequest struct {
	RootID   root.RootID     `json:"rootID"`
	SourceID source.SourceID `json:"sourceID"`

	SkillName string                      `json:"skillName"`
	SKILLMD   []byte                      `json:"skillMD,omitempty"`
	Files     []source.ManagedPackageFile `json:"files,omitempty"`
	Enabled   bool                        `json:"enabled"`
}

type BuiltInSkillArtifactExpectation struct {
	Subresource      basespec.SubresourceLocator `json:"subresource"`
	Kind             artifact.ArtifactKind       `json:"kind"`
	LogicalName      basespec.LogicalName        `json:"logicalName"`
	DefinitionDigest cryptoutil.Digest           `json:"definitionDigest"`
	Enabled          bool                        `json:"enabled"`
}

type BuiltInSkillPackageInstallRequest struct {
	RootID         root.RootID                       `json:"rootID"`
	SourceID       source.SourceID                   `json:"sourceID"`
	PackageAddress source.ManagedPackageAddress      `json:"packageAddress"`
	DocumentFile   basespec.Locator                  `json:"documentFile"`
	PackageFiles   []source.ManagedPackageFile       `json:"packageFiles"`
	Expectations   []BuiltInSkillArtifactExpectation `json:"expectations"`
}

type ManagedSkillCreateResult struct {
	Artifact artifact.Artifact        `json:"artifact"`
	Address  artifact.ArtifactAddress `json:"address"`
}
