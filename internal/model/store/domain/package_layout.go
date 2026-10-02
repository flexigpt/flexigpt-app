package domain

import (
	"fmt"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec/source"
)

func ModelProviderPackageAddress(
	name basespec.LogicalName,
) (source.ManagedPackageAddress, error) {
	return source.NewManagedPackageAddress(
		ModelProviderPackageKind,
		name,
		topology.UnversionedPackageVersion(),
	)
}

func ModelPackageAddress(
	name basespec.LogicalName,
) (source.ManagedPackageAddress, error) {
	return source.NewManagedPackageAddress(
		ModelPackageKind,
		name,
		topology.UnversionedPackageVersion(),
	)
}

func ModelProviderPackageLocator(
	address source.ManagedPackageAddress,
) (basespec.Locator, error) {
	if err := validateModelProviderPackageAddress(address); err != nil {
		return "", err
	}
	return address.FileLocator(ModelProviderDocumentFile())
}

func ModelPackageLocator(
	address source.ManagedPackageAddress,
) (basespec.Locator, error) {
	if err := validateModelPackageAddress(address); err != nil {
		return "", err
	}
	return address.FileLocator(ModelDocumentFile())
}

func ModelProviderPackageAddressFromLocator(
	locator basespec.Locator,
) (source.ManagedPackageAddress, error) {
	return packageAddressFromLocator(
		ModelProviderPackageKind,
		ModelProviderDocumentFile(),
		locator,
	)
}

func ModelPackageAddressFromLocator(
	locator basespec.Locator,
) (source.ManagedPackageAddress, error) {
	return packageAddressFromLocator(
		ModelPackageKind,
		ModelDocumentFile(),
		locator,
	)
}

func validateModelProviderPackageAddress(
	address source.ManagedPackageAddress,
) error {
	if err := address.Validate(); err != nil {
		return err
	}
	if address.Kind != ModelProviderPackageKind {
		return fmt.Errorf(
			"%w: Model Provider package kind is %q, expected %q",
			basespec.ErrInvalid,
			address.Kind,
			ModelProviderPackageKind,
		)
	}
	if address.Version != topology.UnversionedPackageVersion() {
		return fmt.Errorf(
			"%w: Model Provider packages must use the unversioned layout",
			basespec.ErrInvalid,
		)
	}
	return nil
}

func validateModelPackageAddress(
	address source.ManagedPackageAddress,
) error {
	if err := address.Validate(); err != nil {
		return err
	}
	if address.Kind != ModelPackageKind {
		return fmt.Errorf(
			"%w: Model package kind is %q, expected %q",
			basespec.ErrInvalid,
			address.Kind,
			ModelPackageKind,
		)
	}
	if address.Version != topology.UnversionedPackageVersion() {
		return fmt.Errorf(
			"%w: Model packages must use the unversioned layout",
			basespec.ErrInvalid,
		)
	}
	return nil
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
			"%w: managed Model package locator %q is not %q",
			basespec.ErrInvalid,
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
			"%w: managed Model package kind is %q, expected %q",
			basespec.ErrInvalid,
			address.Kind,
			kind,
		)
	}
	if address.Version != topology.UnversionedPackageVersion() {
		return source.ManagedPackageAddress{}, fmt.Errorf(
			"%w: managed Model packages must use the unversioned layout",
			basespec.ErrInvalid,
		)
	}
	return address, nil
}
