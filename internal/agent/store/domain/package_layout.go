package domain

import (
	"fmt"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/agentv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/decoder"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
)

func ManagedAgentDocumentFile() basespec.Locator {
	return documentTopology.MustDefaultDocumentFile(
		documentTopology.DocumentUseManagedAgent,
	)
}

func IsAgentDeclarationDocument(locator basespec.Locator) bool {
	return documentTopology.IsAgentDeclarationDocument(locator)
}

func ManagedPackageAddressForAgent(
	name basespec.LogicalName,
) (source.ManagedPackageAddress, error) {
	if err := name.Validate(); err != nil {
		return source.ManagedPackageAddress{}, err
	}

	return source.NewManagedPackageAddress(
		ManagedAgentPackageKind,
		name,
		documentTopology.UnversionedPackageVersion(),
	)
}

func ManagedPackageLocatorForAgent(
	address source.ManagedPackageAddress,
) (basespec.Locator, error) {
	if err := ValidateManagedAgentPackageAddress(address); err != nil {
		return "", err
	}
	return address.FileLocator(ManagedAgentDocumentFile())
}

func ManagedPackageAddressFromAgentLocator(
	locator basespec.Locator,
) (source.ManagedPackageAddress, error) {
	if err := locator.ValidatePortable(false); err != nil {
		return source.ManagedPackageAddress{}, err
	}
	if path.Base(string(locator)) != string(ManagedAgentDocumentFile()) {
		return source.ManagedPackageAddress{}, fmt.Errorf(
			"%w: Agent locator %q is not %q",
			basespec.ErrInvalid,
			locator,
			ManagedAgentDocumentFile(),
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
	if address.Version != documentTopology.UnversionedPackageVersion() {
		return fmt.Errorf(
			"%w: Agent package version must be %q",
			basespec.ErrInvalid,
			documentTopology.UnversionedPackageVersion(),
		)
	}
	return nil
}

// ManagedAgentDocumentPayload validates one concrete managed Agent declaration
// and projects it into its canonical source bytes and immutable Definition.
//
// Managed Agent packages must contain concrete Agent declarations. A
// source-selected Agent alias is a valid portable declaration, but it is not
// a managed Agent package body because the package would not own the selected
// declaration occurrence.
func ManagedAgentDocumentPayload(
	document agentv1.AgentDocument,
) ([]byte, definition.Definition, error) {
	if err := document.Validate(); err != nil {
		return nil, definition.Definition{}, err
	}
	if document.Locator != nil {
		return nil, definition.Definition{}, fmt.Errorf(
			"%w: managed Agent declaration cannot be a source-selected alias",
			basespec.ErrUnsupported,
		)
	}

	entry, err := declaration.NewEntry(document)
	if err != nil {
		return nil, definition.Definition{}, err
	}
	raw, err := entry.CanonicalJSON()
	if err != nil {
		return nil, definition.Definition{}, err
	}

	value, err := decoder.DefinitionForEntry(entry)
	if err != nil {
		return nil, definition.Definition{}, err
	}
	if value.Kind != AgentArtifactKind ||
		value.LogicalName != basespec.LogicalName(document.Name) ||
		value.LogicalVersion != "" {
		return nil, definition.Definition{}, fmt.Errorf(
			"%w: managed Agent Definition identity is invalid",
			basespec.ErrInvalid,
		)
	}

	return raw, value, nil
}
