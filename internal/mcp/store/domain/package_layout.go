package domain

import (
	"fmt"
	"path"

	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
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
) (sourceModel.ManagedPackageAddress, error) {
	if version == "" {
		version = documentTopology.UnversionedPackageVersion()
	}
	return sourceModel.NewManagedPackageAddress(
		ManagedMCPPackageKind,
		name,
		version,
	)
}

func ManagedPackageLocatorForMCP(
	address sourceModel.ManagedPackageAddress,
) (spec.Locator, error) {
	if err := validateManagedMCPPackageAddress(address); err != nil {
		return "", err
	}
	return address.FileLocator(ManagedMCPDocumentFile())
}

func ManagedPackageAddressFromMCPLocator(
	locator spec.Locator,
) (sourceModel.ManagedPackageAddress, error) {
	if err := locator.ValidatePortable(false); err != nil {
		return sourceModel.ManagedPackageAddress{}, err
	}
	if path.Base(string(locator)) != string(ManagedMCPDocumentFile()) {
		return sourceModel.ManagedPackageAddress{}, fmt.Errorf(
			"%w: MCP locator %q is not %q",
			spec.ErrInvalid,
			locator,
			ManagedMCPDocumentFile(),
		)
	}

	address, err := sourceModel.ParseManagedPackageAddressDirectory(
		spec.Locator(path.Dir(string(locator))),
	)
	if err != nil {
		return sourceModel.ManagedPackageAddress{}, err
	}
	if err := validateManagedMCPPackageAddress(address); err != nil {
		return sourceModel.ManagedPackageAddress{}, err
	}
	return address, nil
}

func ManagedPackageAddressFromMCPPolicyLocator(
	locator spec.Locator,
) (sourceModel.ManagedPackageAddress, error) {
	if err := locator.ValidatePortable(false); err != nil {
		return sourceModel.ManagedPackageAddress{}, err
	}
	if path.Base(string(locator)) != string(ManagedMCPPolicyDocumentFile()) {
		return sourceModel.ManagedPackageAddress{}, fmt.Errorf(
			"%w: MCP Policy locator %q is not %q",
			spec.ErrInvalid,
			locator,
			ManagedMCPPolicyDocumentFile(),
		)
	}

	address, err := sourceModel.ParseManagedPackageAddressDirectory(
		spec.Locator(path.Dir(string(locator))),
	)
	if err != nil {
		return sourceModel.ManagedPackageAddress{}, err
	}
	if address.Kind != ManagedMCPPolicyPackageKind {
		return sourceModel.ManagedPackageAddress{}, fmt.Errorf(
			"%w: MCP Policy package kind must be %q",
			spec.ErrInvalid,
			ManagedMCPPolicyPackageKind,
		)
	}
	return address, nil
}

func validateManagedMCPPackageAddress(
	address sourceModel.ManagedPackageAddress,
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
