package artifact

import (
	"fmt"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	builtinSchema "github.com/flexigpt/flexigpt-app/internal/builtin/schema"
)

const (
	ManagedPackageKind = builtinSchema.AgentSkillPackageKind
	DefinitionFileName = builtinSchema.AgentSkillDefinitionFileName
)

func ManagedPackageAddressForSkill(
	name basespec.LogicalName,
	version basespec.LogicalVersion,
) (source.ManagedPackageAddress, error) {
	if version == "" {
		version = builtinSchema.UnversionedPackageVersion
	}
	return source.NewManagedPackageAddress(
		ManagedPackageKind,
		name,
		version,
	)
}

func ManagedPackageLocatorForSkill(
	address source.ManagedPackageAddress,
) (basespec.Locator, error) {
	if err := validateManagedSkillPackageAddress(address); err != nil {
		return "", err
	}
	return address.FileLocator(DefinitionFileName)
}

func ManagedPackageAddressFromSkillLocator(
	locator basespec.Locator,
) (source.ManagedPackageAddress, error) {
	if err := basespec.ValidatePortableLocator(locator, false); err != nil {
		return source.ManagedPackageAddress{}, err
	}
	if path.Base(string(locator)) != string(DefinitionFileName) {
		return source.ManagedPackageAddress{}, fmt.Errorf(
			"%w: Skill locator %q is not %q",
			basespec.ErrInvalid,
			locator,
			DefinitionFileName,
		)
	}

	address, err := source.ParseManagedPackageAddressDirectory(
		basespec.Locator(path.Dir(string(locator))),
	)
	if err != nil {
		return source.ManagedPackageAddress{}, err
	}
	if err := validateManagedSkillPackageAddress(address); err != nil {
		return source.ManagedPackageAddress{}, err
	}
	return address, nil
}

func validateManagedSkillPackageAddress(
	address source.ManagedPackageAddress,
) error {
	if err := address.Validate(); err != nil {
		return err
	}
	if address.Kind != ManagedPackageKind {
		return fmt.Errorf(
			"%w: Skill package kind must be %q",
			basespec.ErrInvalid,
			ManagedPackageKind,
		)
	}
	return nil
}
