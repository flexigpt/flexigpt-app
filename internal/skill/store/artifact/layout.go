package artifact

import (
	"fmt"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

func ManagedPackageAddressForSkill(
	name basespec.LogicalName,
	version basespec.LogicalVersion,
) (source.ManagedPackageAddress, error) {
	if version == "" {
		version = artifactbuiltin.UnversionedPackageVersion
	}
	return source.NewManagedPackageAddress(
		artifactbuiltin.AgentSkillPackageKind,
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
	return address.FileLocator(artifactbuiltin.AgentSkillDefinitionFileName)
}

func ManagedPackageAddressFromSkillLocator(
	locator basespec.Locator,
) (source.ManagedPackageAddress, error) {
	if err := basespec.ValidatePortableLocator(locator, false); err != nil {
		return source.ManagedPackageAddress{}, err
	}
	if path.Base(string(locator)) != string(artifactbuiltin.AgentSkillDefinitionFileName) {
		return source.ManagedPackageAddress{}, fmt.Errorf(
			"%w: Skill locator %q is not %q",
			basespec.ErrInvalid,
			locator,
			artifactbuiltin.AgentSkillDefinitionFileName,
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
	if address.Kind != artifactbuiltin.AgentSkillPackageKind {
		return fmt.Errorf(
			"%w: Skill package kind must be %q",
			basespec.ErrInvalid,
			artifactbuiltin.AgentSkillPackageKind,
		)
	}
	return nil
}
