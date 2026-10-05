package domain

import (
	"fmt"
	"path"

	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
)

func ToolPackageAddress(
	name spec.LogicalName,
	version spec.LogicalVersion,
) (managedpackageModel.ManagedPackageAddress, error) {
	return managedpackageModel.NewManagedPackageAddress(
		ToolPackageKind,
		name,
		version,
	)
}

func ToolPluginPackageAddress(
	name spec.LogicalName,
) (managedpackageModel.ManagedPackageAddress, error) {
	return managedpackageModel.NewManagedPackageAddress(
		ToolPluginPackageKind,
		name,
		topology.UnversionedPackageVersion(),
	)
}

func ToolPackageAddressFromLocator(
	locator spec.Locator,
) (managedpackageModel.ManagedPackageAddress, error) {
	return packageAddressFromLocator(
		ToolPackageKind,
		ToolDocumentFile(),
		locator,
	)
}

func ToolPluginPackageAddressFromLocator(
	locator spec.Locator,
) (managedpackageModel.ManagedPackageAddress, error) {
	return packageAddressFromLocator(
		ToolPluginPackageKind,
		ToolPluginDocumentFile(),
		locator,
	)
}

func packageAddressFromLocator(
	kind managedpackageModel.PackageKind,
	documentFile spec.Locator,
	locator spec.Locator,
) (managedpackageModel.ManagedPackageAddress, error) {
	if err := locator.ValidatePortable(false); err != nil {
		return managedpackageModel.ManagedPackageAddress{}, err
	}
	if path.Base(string(locator)) != string(documentFile) {
		return managedpackageModel.ManagedPackageAddress{}, fmt.Errorf(
			"%w: package locator %q is not %q",
			spec.ErrUnsupported,
			locator,
			documentFile,
		)
	}

	address, err := managedpackageModel.ParseManagedPackageAddressDirectory(
		spec.Locator(path.Dir(string(locator))),
	)
	if err != nil {
		return managedpackageModel.ManagedPackageAddress{}, err
	}
	if address.Kind != kind {
		return managedpackageModel.ManagedPackageAddress{}, fmt.Errorf(
			"%w: package kind is %q, expected %q",
			spec.ErrUnsupported,
			address.Kind,
			kind,
		)
	}
	return address, nil
}
