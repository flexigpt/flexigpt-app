package domain

import (
	"fmt"
	"path"

	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	source "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func ManagedMCPDocumentFile() spec.Locator {
	return documentTopology.MustDefaultDocumentFile(
		documentTopology.DocumentUseManagedMCP,
	)
}

func ManagedMCPPolicyDocumentFile() spec.Locator {
	return documentTopology.MustDefaultDocumentFile(
		documentTopology.DocumentUseManagedMCPPolicy,
	)
}

func ManagedPackageAddressForMCP(
	name spec.LogicalName,
	version spec.LogicalVersion,
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
) (spec.Locator, error) {
	if err := validateManagedMCPPackageAddress(address); err != nil {
		return "", err
	}
	return address.FileLocator(ManagedMCPDocumentFile())
}

func ManagedPackageAddressFromMCPLocator(
	locator spec.Locator,
) (source.ManagedPackageAddress, error) {
	if err := locator.ValidatePortable(false); err != nil {
		return source.ManagedPackageAddress{}, err
	}
	if path.Base(string(locator)) != string(ManagedMCPDocumentFile()) {
		return source.ManagedPackageAddress{}, fmt.Errorf(
			"%w: MCP locator %q is not %q",
			spec.ErrInvalid,
			locator,
			ManagedMCPDocumentFile(),
		)
	}

	address, err := source.ParseManagedPackageAddressDirectory(
		spec.Locator(path.Dir(string(locator))),
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
	locator spec.Locator,
) (source.ManagedPackageAddress, error) {
	if err := locator.ValidatePortable(false); err != nil {
		return source.ManagedPackageAddress{}, err
	}
	if path.Base(string(locator)) != string(ManagedMCPPolicyDocumentFile()) {
		return source.ManagedPackageAddress{}, fmt.Errorf(
			"%w: MCP Policy locator %q is not %q",
			spec.ErrInvalid,
			locator,
			ManagedMCPPolicyDocumentFile(),
		)
	}

	address, err := source.ParseManagedPackageAddressDirectory(
		spec.Locator(path.Dir(string(locator))),
	)
	if err != nil {
		return source.ManagedPackageAddress{}, err
	}
	if address.Kind != ManagedMCPPolicyPackageKind {
		return source.ManagedPackageAddress{}, fmt.Errorf(
			"%w: MCP Policy package kind must be %q",
			spec.ErrInvalid,
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
			spec.ErrInvalid,
			ManagedMCPPackageKind,
		)
	}
	return nil
}
