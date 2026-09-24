package consumerapi

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type BuiltInPackageInstallRequest struct {
	RootID              root.RootID                  `json:"rootID"`
	SourceID            source.SourceID              `json:"sourceID"`
	Package             source.ManagedPackageAddress `json:"package"`
	DocumentFile        basespec.Locator             `json:"documentFile"`
	PackageFiles        []source.ManagedPackageFile  `json:"packageFiles"`
	ExpectedKind        artifact.ArtifactKind        `json:"expectedKind"`
	ExpectedLogicalName basespec.LogicalName         `json:"expectedLogicalName"`
	ExpectedDefinition  cryptoutil.Digest            `json:"expectedDefinition"`
}

func (r BuiltInPackageInstallRequest) Validate() error {
	if err := r.RootID.Validate(); err != nil {
		return err
	}
	if err := r.SourceID.Validate(); err != nil {
		return err
	}
	if err := r.Package.Validate(); err != nil {
		return err
	}
	if err := r.DocumentFile.ValidatePortable(false); err != nil {
		return err
	}
	if _, err := source.NormalizeManagedPackageFiles(
		r.PackageFiles,
	); err != nil {
		return err
	}
	if err := r.ExpectedKind.Validate(); err != nil {
		return err
	}
	if err := r.ExpectedLogicalName.Validate(); err != nil {
		return err
	}
	return cryptoutil.ValidateDigest(r.ExpectedDefinition)
}
