package domain

import (
	"fmt"
	"path"

	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
)

func ManagedMCPDocumentFile() basespec.Locator {
	return documentTopology.MustDefaultDocumentFile(
		documentTopology.DocumentUseManagedMCP,
	)
}

func ManagedMCPPolicyDocumentFile() basespec.Locator {
	return documentTopology.MustDefaultDocumentFile(
		documentTopology.DocumentUseManagedMCPPolicy,
	)
}

func ManagedPackageAddressForMCP(
	name basespec.LogicalName,
	version basespec.LogicalVersion,
) (source.ManagedPackageAddress, error) {
	if version == "" {
		version = documentTopology.UnversionedPackageVersion()
	}
	return source.NewManagedPackageAddress(
		ManagedMCPPackageKind,
		name,
		version,
	)
}

func ManagedPackageLocatorForMCP(
	address source.ManagedPackageAddress,
) (basespec.Locator, error) {
	if err := validateManagedMCPPackageAddress(address); err != nil {
		return "", err
	}
	return address.FileLocator(ManagedMCPDocumentFile())
}

func ManagedPackageAddressFromMCPLocator(
	locator basespec.Locator,
) (source.ManagedPackageAddress, error) {
	if err := locator.ValidatePortable(false); err != nil {
		return source.ManagedPackageAddress{}, err
	}
	if path.Base(string(locator)) != string(ManagedMCPDocumentFile()) {
		return source.ManagedPackageAddress{}, fmt.Errorf(
			"%w: MCP locator %q is not %q",
			basespec.ErrInvalid,
			locator,
			ManagedMCPDocumentFile(),
		)
	}

	address, err := source.ParseManagedPackageAddressDirectory(
		basespec.Locator(path.Dir(string(locator))),
	)
	if err != nil {
		return source.ManagedPackageAddress{}, err
	}
	if err := validateManagedMCPPackageAddress(address); err != nil {
		return source.ManagedPackageAddress{}, err
	}
	return address, nil
}

func ManagedPackageAddressFromMCPPolicyLocator(
	locator basespec.Locator,
) (source.ManagedPackageAddress, error) {
	if err := locator.ValidatePortable(false); err != nil {
		return source.ManagedPackageAddress{}, err
	}
	if path.Base(string(locator)) != string(ManagedMCPPolicyDocumentFile()) {
		return source.ManagedPackageAddress{}, fmt.Errorf(
			"%w: MCP Policy locator %q is not %q",
			basespec.ErrInvalid,
			locator,
			ManagedMCPPolicyDocumentFile(),
		)
	}

	address, err := source.ParseManagedPackageAddressDirectory(
		basespec.Locator(path.Dir(string(locator))),
	)
	if err != nil {
		return source.ManagedPackageAddress{}, err
	}
	if address.Kind != ManagedMCPPolicyPackageKind {
		return source.ManagedPackageAddress{}, fmt.Errorf(
			"%w: MCP Policy package kind must be %q",
			basespec.ErrInvalid,
			ManagedMCPPolicyPackageKind,
		)
	}
	return address, nil
}

func validateManagedMCPPackageAddress(
	address source.ManagedPackageAddress,
) error {
	if err := address.Validate(); err != nil {
		return err
	}
	if address.Kind != ManagedMCPPackageKind {
		return fmt.Errorf(
			"%w: MCP package kind must be %q",
			basespec.ErrInvalid,
			ManagedMCPPackageKind,
		)
	}
	return nil
}
