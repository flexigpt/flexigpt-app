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

type ManagedSkillCreateRequest struct {
	RootID   root.RootID     `json:"rootID"`
	SourceID source.SourceID `json:"sourceID"`

	SkillName string                      `json:"skillName"`
	SKILLMD   []byte                      `json:"skillMD,omitempty"`
	Files     []source.ManagedPackageFile `json:"files,omitempty"`
	Enabled   bool                        `json:"enabled"`
}

type ManagedSkillCreateResult struct {
	Artifact artifact.Artifact        `json:"artifact"`
	Address  artifact.ArtifactAddress `json:"address"`
}

type BuiltInSkillInstallRequest struct {
	RootID              root.RootID                  `json:"rootID"`
	SourceID            source.SourceID              `json:"sourceID"`
	PackageAddress      source.ManagedPackageAddress `json:"packageAddress"`
	PackageFiles        []source.ManagedPackageFile  `json:"packageFiles"`
	ExpectedLogicalName basespec.LogicalName         `json:"expectedLogicalName"`
	ExpectedDefinition  cryptoutil.Digest            `json:"expectedDefinition"`
	Enabled             bool                         `json:"enabled"`
}
