package domain

import (
	"fmt"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
)

func ManagedPackageAddressForSkill(
	name basespec.LogicalName,
	version basespec.LogicalVersion,
) (source.ManagedPackageAddress, error) {
	if version == "" {
		version = builtin.UnversionedPackageVersion
	}
	return source.NewManagedPackageAddress(
		ManagedSkillPackageKind,
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
	return address.FileLocator(SkillDefinitionFileName)
}

func ManagedPackageAddressFromSkillLocator(
	locator basespec.Locator,
) (source.ManagedPackageAddress, error) {
	if err := locator.ValidatePortable(false); err != nil {
		return source.ManagedPackageAddress{}, err
	}
	if path.Base(string(locator)) != string(SkillDefinitionFileName) {
		return source.ManagedPackageAddress{}, fmt.Errorf(
			"%w: Skill locator %q is not %q",
			basespec.ErrInvalid,
			locator,
			SkillDefinitionFileName,
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
	if address.Kind != ManagedSkillPackageKind {
		return fmt.Errorf(
			"%w: Skill package kind must be %q",
			basespec.ErrInvalid,
			ManagedSkillPackageKind,
		)
	}
	return nil
}
