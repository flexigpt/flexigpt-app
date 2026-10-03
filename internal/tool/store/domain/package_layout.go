package domain

import (
	"fmt"
	"path"

	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func ToolPackageAddress(
	name spec.LogicalName,
	version spec.LogicalVersion,
) (sourceModel.ManagedPackageAddress, error) {
	return sourceModel.NewManagedPackageAddress(
		ToolPackageKind,
		name,
		version,
	)
}

func ToolCollectionPackageAddress(
	name spec.LogicalName,
) (sourceModel.ManagedPackageAddress, error) {
	return sourceModel.NewManagedPackageAddress(
		ToolCollectionPackageKind,
		name,
		documentTopology.UnversionedPackageVersion(),
	)
}

func ToolPackageAddressFromLocator(
	locator spec.Locator,
) (sourceModel.ManagedPackageAddress, error) {
	return packageAddressFromLocator(
		ToolPackageKind,
		ToolDocumentFile(),
		locator,
	)
}

func ToolCollectionPackageAddressFromLocator(
	locator spec.Locator,
) (sourceModel.ManagedPackageAddress, error) {
	return packageAddressFromLocator(
		ToolCollectionPackageKind,
		ToolCollectionDocumentFile(),
		locator,
	)
}

func packageAddressFromLocator(
	kind sourceModel.PackageKind,
	documentFile spec.Locator,
	locator spec.Locator,
) (sourceModel.ManagedPackageAddress, error) {
	if err := locator.ValidatePortable(false); err != nil {
		return sourceModel.ManagedPackageAddress{}, err
	}
	if path.Base(string(locator)) != string(documentFile) {
		return sourceModel.ManagedPackageAddress{}, fmt.Errorf(
			"%w: package locator %q is not %q",
			spec.ErrUnsupported,
			locator,
			documentFile,
		)
	}

	address, err := sourceModel.ParseManagedPackageAddressDirectory(
		spec.Locator(path.Dir(string(locator))),
	)
	if err != nil {
		return sourceModel.ManagedPackageAddress{}, err
	}
	if address.Kind != kind {
		return sourceModel.ManagedPackageAddress{}, fmt.Errorf(
			"%w: package kind is %q, expected %q",
			spec.ErrUnsupported,
			address.Kind,
			kind,
		)
	}
	return address, nil
}
