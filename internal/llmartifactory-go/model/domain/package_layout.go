package domain

import (
	"fmt"
	"path"

	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
)

func ModelProviderPackageAddress(
	name spec.LogicalName,
) (managedpackageModel.ManagedPackageAddress, error) {
	return managedpackageModel.NewManagedPackageAddress(
		ModelProviderPackageKind,
		name,
		topology.UnversionedPackageVersion(),
	)
}

func ModelPackageAddress(
	name spec.LogicalName,
) (managedpackageModel.ManagedPackageAddress, error) {
	return managedpackageModel.NewManagedPackageAddress(
		ModelPackageKind,
		name,
		topology.UnversionedPackageVersion(),
	)
}

func ModelProviderPackageLocator(
	address managedpackageModel.ManagedPackageAddress,
) (spec.Locator, error) {
	if err := validateModelProviderPackageAddress(address); err != nil {
		return "", err
	}
	return address.FileLocator(ModelProviderDocumentFile())
}

func ModelPackageLocator(
	address managedpackageModel.ManagedPackageAddress,
) (spec.Locator, error) {
	if err := validateModelPackageAddress(address); err != nil {
		return "", err
	}
	return address.FileLocator(ModelDocumentFile())
}

func ModelProviderPackageAddressFromLocator(
	locator spec.Locator,
) (managedpackageModel.ManagedPackageAddress, error) {
	return packageAddressFromLocator(
		ModelProviderPackageKind,
		ModelProviderDocumentFile(),
		locator,
	)
}

func ModelPackageAddressFromLocator(
	locator spec.Locator,
) (managedpackageModel.ManagedPackageAddress, error) {
	return packageAddressFromLocator(
		ModelPackageKind,
		ModelDocumentFile(),
		locator,
	)
}

func validateModelProviderPackageAddress(
	address managedpackageModel.ManagedPackageAddress,
) error {
	if err := address.Validate(); err != nil {
		return err
	}
	if address.Kind != ModelProviderPackageKind {
		return fmt.Errorf(
			"%w: Model Provider package kind is %q, expected %q",
			spec.ErrInvalid,
			address.Kind,
			ModelProviderPackageKind,
		)
	}
	if address.Version != topology.UnversionedPackageVersion() {
		return fmt.Errorf(
			"%w: Model Provider packages must use the unversioned layout",
			spec.ErrInvalid,
		)
	}
	return nil
}

func validateModelPackageAddress(
	address managedpackageModel.ManagedPackageAddress,
) error {
	if err := address.Validate(); err != nil {
		return err
	}
	if address.Kind != ModelPackageKind {
		return fmt.Errorf(
			"%w: Model package kind is %q, expected %q",
			spec.ErrInvalid,
			address.Kind,
			ModelPackageKind,
		)
	}
	if address.Version != topology.UnversionedPackageVersion() {
		return fmt.Errorf(
			"%w: Model packages must use the unversioned layout",
			spec.ErrInvalid,
		)
	}
	return nil
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
			"%w: managed Model package locator %q is not %q",
			spec.ErrInvalid,
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
			"%w: managed Model package kind is %q, expected %q",
			spec.ErrInvalid,
			address.Kind,
			kind,
		)
	}
	if address.Version != topology.UnversionedPackageVersion() {
		return managedpackageModel.ManagedPackageAddress{}, fmt.Errorf(
			"%w: managed Model packages must use the unversioned layout",
			spec.ErrInvalid,
		)
	}
	return address, nil
}
