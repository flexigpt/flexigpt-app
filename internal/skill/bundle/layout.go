package bundle

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	builtinSchema "github.com/flexigpt/flexigpt-app/internal/builtin/schema"
)

const BuiltInCollectionPackageKind = builtinSchema.SkillBundlePackageKind

func BuiltInCollectionPackageAddress(
	name basespec.LogicalName,
	version basespec.LogicalVersion,
) (source.ManagedPackageAddress, error) {
	if version == "" {
		version = builtinSchema.UnversionedPackageVersion
	}
	return source.NewManagedPackageAddress(
		BuiltInCollectionPackageKind,
		name,
		version,
	)
}
