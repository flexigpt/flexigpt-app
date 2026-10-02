package domain

import (
	"fmt"
	"path"

	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	source "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func ToolPackageAddress(
	name spec.LogicalName,
	version spec.LogicalVersion,
) (source.ManagedPackageAddress, error) {
	return source.NewManagedPackageAddress(
		ToolPackageKind,
		name,
		version,
	)
}

func ToolCollectionPackageAddress(
	name spec.LogicalName,
) (source.ManagedPackageAddress, error) {
	return source.NewManagedPackageAddress(
		ToolCollectionPackageKind,
		name,
		documentTopology.UnversionedPackageVersion(),
	)
}

func ToolPackageAddressFromLocator(
	locator spec.Locator,
) (source.ManagedPackageAddress, error) {
	return packageAddressFromLocator(
		ToolPackageKind,
		ToolDocumentFile(),
		locator,
	)
}

func ToolCollectionPackageAddressFromLocator(
	locator spec.Locator,
) (source.ManagedPackageAddress, error) {
	return packageAddressFromLocator(
		ToolCollectionPackageKind,
		ToolCollectionDocumentFile(),
		locator,
	)
}

func packageAddressFromLocator(
	kind source.PackageKind,
	documentFile spec.Locator,
	locator spec.Locator,
) (source.ManagedPackageAddress, error) {
	if err := locator.ValidatePortable(false); err != nil {
		return source.ManagedPackageAddress{}, err
	}
	if path.Base(string(locator)) != string(documentFile) {
		return source.ManagedPackageAddress{}, fmt.Errorf(
			"%w: package locator %q is not %q",
			spec.ErrUnsupported,
			locator,
			documentFile,
		)
	}

	address, err := source.ParseManagedPackageAddressDirectory(
		spec.Locator(path.Dir(string(locator))),
	)
	if err != nil {
		return source.ManagedPackageAddress{}, err
	}
	if address.Kind != kind {
		return source.ManagedPackageAddress{}, fmt.Errorf(
			"%w: package kind is %q, expected %q",
			spec.ErrUnsupported,
			address.Kind,
			kind,
		)
	}
	return address, nil
}
