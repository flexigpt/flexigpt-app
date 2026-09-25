package domain

import (
	"fmt"
	"path"

	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
)

func ToolPackageAddress(
	name basespec.LogicalName,
	version basespec.LogicalVersion,
) (source.ManagedPackageAddress, error) {
	return source.NewManagedPackageAddress(
		ToolPackageKind,
		name,
		version,
	)
}

func ToolCollectionPackageAddress(
	name basespec.LogicalName,
) (source.ManagedPackageAddress, error) {
	return source.NewManagedPackageAddress(
		ToolCollectionPackageKind,
		name,
		documentTopology.UnversionedPackageVersion(),
	)
}

func ToolPackageAddressFromLocator(
	locator basespec.Locator,
) (source.ManagedPackageAddress, error) {
	return packageAddressFromLocator(
		ToolPackageKind,
		ToolDocumentFile(),
		locator,
	)
}

func ToolCollectionPackageAddressFromLocator(
	locator basespec.Locator,
) (source.ManagedPackageAddress, error) {
	return packageAddressFromLocator(
		ToolCollectionPackageKind,
		ToolCollectionDocumentFile(),
		locator,
	)
}

func packageAddressFromLocator(
	kind source.PackageKind,
	documentFile basespec.Locator,
	locator basespec.Locator,
) (source.ManagedPackageAddress, error) {
	if err := locator.ValidatePortable(false); err != nil {
		return source.ManagedPackageAddress{}, err
	}
	if path.Base(string(locator)) != string(documentFile) {
		return source.ManagedPackageAddress{}, fmt.Errorf(
			"%w: package locator %q is not %q",
			basespec.ErrUnsupported,
			locator,
			documentFile,
		)
	}

	address, err := source.ParseManagedPackageAddressDirectory(
		basespec.Locator(path.Dir(string(locator))),
	)
	if err != nil {
		return source.ManagedPackageAddress{}, err
	}
	if address.Kind != kind {
		return source.ManagedPackageAddress{}, fmt.Errorf(
			"%w: package kind is %q, expected %q",
			basespec.ErrUnsupported,
			address.Kind,
			kind,
		)
	}
	return address, nil
}
