package domain

import (
	"fmt"
	"path"

	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/source"
)

func ToolPackageAddress(
	name model.LogicalName,
	version model.LogicalVersion,
) (source.ManagedPackageAddress, error) {
	return source.NewManagedPackageAddress(
		ToolPackageKind,
		name,
		version,
	)
}

func ToolCollectionPackageAddress(
	name model.LogicalName,
) (source.ManagedPackageAddress, error) {
	return source.NewManagedPackageAddress(
		ToolCollectionPackageKind,
		name,
		documentTopology.UnversionedPackageVersion(),
	)
}

func ToolPackageAddressFromLocator(
	locator model.Locator,
) (source.ManagedPackageAddress, error) {
	return packageAddressFromLocator(
		ToolPackageKind,
		ToolDocumentFile(),
		locator,
	)
}

func ToolCollectionPackageAddressFromLocator(
	locator model.Locator,
) (source.ManagedPackageAddress, error) {
	return packageAddressFromLocator(
		ToolCollectionPackageKind,
		ToolCollectionDocumentFile(),
		locator,
	)
}

func packageAddressFromLocator(
	kind source.PackageKind,
	documentFile model.Locator,
	locator model.Locator,
) (source.ManagedPackageAddress, error) {
	if err := locator.ValidatePortable(false); err != nil {
		return source.ManagedPackageAddress{}, err
	}
	if path.Base(string(locator)) != string(documentFile) {
		return source.ManagedPackageAddress{}, fmt.Errorf(
			"%w: package locator %q is not %q",
			model.ErrUnsupported,
			locator,
			documentFile,
		)
	}

	address, err := source.ParseManagedPackageAddressDirectory(
		model.Locator(path.Dir(string(locator))),
	)
	if err != nil {
		return source.ManagedPackageAddress{}, err
	}
	if address.Kind != kind {
		return source.ManagedPackageAddress{}, fmt.Errorf(
			"%w: package kind is %q, expected %q",
			model.ErrUnsupported,
			address.Kind,
			kind,
		)
	}
	return address, nil
}
