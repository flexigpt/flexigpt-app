package domain

import (
	"fmt"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func ModelProviderPackageAddress(
	name spec.LogicalName,
) (sourceModel.ManagedPackageAddress, error) {
	return sourceModel.NewManagedPackageAddress(
		ModelProviderPackageKind,
		name,
		topology.UnversionedPackageVersion(),
	)
}

func ModelPackageAddress(
	name spec.LogicalName,
) (sourceModel.ManagedPackageAddress, error) {
	return sourceModel.NewManagedPackageAddress(
		ModelPackageKind,
		name,
		topology.UnversionedPackageVersion(),
	)
}

func ModelProviderPackageLocator(
	address sourceModel.ManagedPackageAddress,
) (spec.Locator, error) {
	if err := validateModelProviderPackageAddress(address); err != nil {
		return "", err
	}
	return address.FileLocator(ModelProviderDocumentFile())
}

func ModelPackageLocator(
	address sourceModel.ManagedPackageAddress,
) (spec.Locator, error) {
	if err := validateModelPackageAddress(address); err != nil {
		return "", err
	}
	return address.FileLocator(ModelDocumentFile())
}

func ModelProviderPackageAddressFromLocator(
	locator spec.Locator,
) (sourceModel.ManagedPackageAddress, error) {
	return packageAddressFromLocator(
		ModelProviderPackageKind,
		ModelProviderDocumentFile(),
		locator,
	)
}

func ModelPackageAddressFromLocator(
	locator spec.Locator,
) (sourceModel.ManagedPackageAddress, error) {
	return packageAddressFromLocator(
		ModelPackageKind,
		ModelDocumentFile(),
		locator,
	)
}

func validateModelProviderPackageAddress(
	address sourceModel.ManagedPackageAddress,
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
	address sourceModel.ManagedPackageAddress,
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
	kind sourceModel.PackageKind,
	documentFile spec.Locator,
	locator spec.Locator,
) (sourceModel.ManagedPackageAddress, error) {
	if err := locator.ValidatePortable(false); err != nil {
		return sourceModel.ManagedPackageAddress{}, err
	}
	if path.Base(string(locator)) != string(documentFile) {
		return sourceModel.ManagedPackageAddress{}, fmt.Errorf(
			"%w: managed Model package locator %q is not %q",
			spec.ErrInvalid,
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
			"%w: managed Model package kind is %q, expected %q",
			spec.ErrInvalid,
			address.Kind,
			kind,
		)
	}
	if address.Version != topology.UnversionedPackageVersion() {
		return sourceModel.ManagedPackageAddress{}, fmt.Errorf(
			"%w: managed Model packages must use the unversioned layout",
			spec.ErrInvalid,
		)
	}
	return address, nil
}
