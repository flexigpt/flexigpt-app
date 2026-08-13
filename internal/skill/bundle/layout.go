package bundle

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

func BuiltInCollectionPackageAddress(
	name basespec.LogicalName,
	version basespec.LogicalVersion,
) (source.ManagedPackageAddress, error) {
	if version == "" {
		version = artifactbuiltin.UnversionedPackageVersion
	}
	return source.NewManagedPackageAddress(
		artifactbuiltin.SkillBundlePackageKind,
		name,
		version,
	)
}
