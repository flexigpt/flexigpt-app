package bundle

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

const (
	BuiltInCollectionPackageKind basespec.PackageKind = "skill.bundle"

	builtInCollectionUnversionedVersion basespec.LogicalVersion = "unversioned"
)

func BuiltInCollectionPackageAddress(
	name basespec.LogicalName,
	version basespec.LogicalVersion,
) (source.ManagedPackageAddress, error) {
	if version == "" {
		version = builtInCollectionUnversionedVersion
	}
	return source.NewManagedPackageAddress(
		BuiltInCollectionPackageKind,
		name,
		version,
	)
}
