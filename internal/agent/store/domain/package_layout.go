package domain

import (
	"fmt"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
)

func ManagedPackageAddressForAgent(
	name basespec.LogicalName,
) (source.ManagedPackageAddress, error) {
	if err := name.Validate(); err != nil {
		return source.ManagedPackageAddress{}, err
	}

	return source.NewManagedPackageAddress(
		ManagedAgentPackageKind,
		name,
		builtin.UnversionedPackageVersion,
	)
}

func ManagedPackageLocatorForAgent(
	address source.ManagedPackageAddress,
) (basespec.Locator, error) {
	if err := ValidateManagedAgentPackageAddress(address); err != nil {
		return "", err
	}
	return address.FileLocator(ManagedAgentDocumentFile)
}

func ManagedPackageAddressFromAgentLocator(
	locator basespec.Locator,
) (source.ManagedPackageAddress, error) {
	if err := locator.ValidatePortable(false); err != nil {
		return source.ManagedPackageAddress{}, err
	}
	if path.Base(string(locator)) != string(ManagedAgentDocumentFile) {
		return source.ManagedPackageAddress{}, fmt.Errorf(
			"%w: Agent locator %q is not %q",
			basespec.ErrInvalid,
			locator,
			ManagedAgentDocumentFile,
		)
	}

	address, err := source.ParseManagedPackageAddressDirectory(
		basespec.Locator(path.Dir(string(locator))),
	)
	if err != nil {
		return source.ManagedPackageAddress{}, err
	}
	if err := ValidateManagedAgentPackageAddress(address); err != nil {
		return source.ManagedPackageAddress{}, err
	}
	return address, nil
}

func ValidateManagedAgentPackageAddress(
	address source.ManagedPackageAddress,
) error {
	if err := address.Validate(); err != nil {
		return err
	}
	if address.Kind != ManagedAgentPackageKind {
		return fmt.Errorf(
			"%w: Agent package kind must be %q",
			basespec.ErrInvalid,
			ManagedAgentPackageKind,
		)
	}
	if address.Version != builtin.UnversionedPackageVersion {
		return fmt.Errorf(
			"%w: Agent package version must be %q",
			basespec.ErrInvalid,
			builtin.UnversionedPackageVersion,
		)
	}
	return nil
}
