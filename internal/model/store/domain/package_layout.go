package domain

import (
	"fmt"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	source "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func ModelProviderPackageAddress(
	name spec.LogicalName,
) (source.ManagedPackageAddress, error) {
	return source.NewManagedPackageAddress(
		ModelProviderPackageKind,
		name,
		topology.UnversionedPackageVersion(),
	)
}

func ModelPackageAddress(
	name spec.LogicalName,
) (source.ManagedPackageAddress, error) {
	return source.NewManagedPackageAddress(
		ModelPackageKind,
		name,
		topology.UnversionedPackageVersion(),
	)
}

func ModelProviderPackageLocator(
	address source.ManagedPackageAddress,
) (spec.Locator, error) {
	if err := validateModelProviderPackageAddress(address); err != nil {
		return "", err
	}
	return address.FileLocator(ModelProviderDocumentFile())
}

func ModelPackageLocator(
	address source.ManagedPackageAddress,
) (spec.Locator, error) {
	if err := validateModelPackageAddress(address); err != nil {
		return "", err
	}
	return address.FileLocator(ModelDocumentFile())
}

func ModelProviderPackageAddressFromLocator(
	locator spec.Locator,
) (source.ManagedPackageAddress, error) {
	return packageAddressFromLocator(
		ModelProviderPackageKind,
		ModelProviderDocumentFile(),
		locator,
	)
}

func ModelPackageAddressFromLocator(
	locator spec.Locator,
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
	address source.ManagedPackageAddress,
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
	kind source.PackageKind,
	documentFile spec.Locator,
	locator spec.Locator,
) (source.ManagedPackageAddress, error) {
	if err := locator.ValidatePortable(false); err != nil {
		return source.ManagedPackageAddress{}, err
	}
	if path.Base(string(locator)) != string(documentFile) {
		return source.ManagedPackageAddress{}, fmt.Errorf(
			"%w: managed Model package locator %q is not %q",
			spec.ErrInvalid,
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
			"%w: managed Model package kind is %q, expected %q",
			spec.ErrInvalid,
			address.Kind,
			kind,
		)
	}
	if address.Version != topology.UnversionedPackageVersion() {
		return source.ManagedPackageAddress{}, fmt.Errorf(
			"%w: managed Model packages must use the unversioned layout",
			spec.ErrInvalid,
		)
	}
	return address, nil
}
