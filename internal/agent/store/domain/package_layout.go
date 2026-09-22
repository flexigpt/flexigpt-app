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

// ManagedAgentEntryPayload projects one concrete canonical Agent Entry into
// managed package bytes and an immutable Definition.
//
// Strict managed-import profile validation belongs to the managed import
// domain. This package owns the invariant that a managed package root is a
// concrete Agent declaration rather than a source-selected alias.
func ManagedAgentEntryPayload(
	entry declaration.Entry,
) ([]byte, definition.Definition, error) {
	if err := declaration.ValidateEntryType(
		entry,
		declaration.TypeAgent,
	); err != nil {
		return nil, definition.Definition{}, err
	}

	document, err := agentv1.DecodeAgentEntry(entry)
	if err != nil {
		return nil, definition.Definition{}, err
	}
	if document.Locator != nil {
		return nil, definition.Definition{}, fmt.Errorf(
			"%w: managed Agent declaration cannot be a source-selected alias",
			basespec.ErrUnsupported,
		)
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
