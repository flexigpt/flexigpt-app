package bundle

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

const BuiltInCollectionPackageKind = artifactbuiltin.SkillBundlePackageKind

func BuiltInCollectionPackageAddress(
	name basespec.LogicalName,
	version basespec.LogicalVersion,
) (source.ManagedPackageAddress, error) {
	if version == "" {
		version = artifactbuiltin.UnversionedPackageVersion
	}
	return source.NewManagedPackageAddress(
		BuiltInCollectionPackageKind,
		name,
		version,
	)
}
