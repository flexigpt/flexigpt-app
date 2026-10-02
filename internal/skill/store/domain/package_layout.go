package domain

import (
	"fmt"
	"path"

	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/source"
)

func ManagedPackageAddressForSkill(
	name model.LogicalName,
	version model.LogicalVersion,
) (source.ManagedPackageAddress, error) {
	if version == "" {
		version = documentTopology.UnversionedPackageVersion()
	}
	return source.NewManagedPackageAddress(
		ManagedSkillPackageKind,
		name,
		version,
	)
}

// ManagedSkillDirectoryLocator returns the physical Skill directory inside a
// managed package. Agent Skills requires the directory containing SKILL.md to
// have the same name as the document frontmatter name.
//
// The generic package address remains:
//
//	skill/<skill-name>/<version>
//
// The runtime-facing Skill directory is:
//
//	skill/<skill-name>/<version>/<skill-name>
func ManagedSkillDirectoryLocator(
	address source.ManagedPackageAddress,
) (model.Locator, error) {
	if err := validateManagedSkillPackageAddress(address); err != nil {
		return "", err
	}

	packageDirectory, err := address.Directory()
	if err != nil {
		return "", err
	}
	directory := model.Locator(path.Join(
		string(packageDirectory),
		string(address.Name),
	))
	if err := directory.ValidatePortable(false); err != nil {
		return "", err
	}
	return directory, nil
}

func ManagedPackageLocatorForSkill(
	address source.ManagedPackageAddress,
) (model.Locator, error) {
	directory, err := ManagedSkillDirectoryLocator(address)
	if err != nil {
		return "", err
	}
	locator := model.Locator(path.Join(
		string(directory),
		string(SkillDefinitionFileName()),
	))
	if err := locator.ValidatePortable(false); err != nil {
		return "", err
	}
	return locator, nil
}

func ManagedPackageAddressFromSkillLocator(
	locator model.Locator,
) (source.ManagedPackageAddress, error) {
	if err := locator.ValidatePortable(false); err != nil {
		return source.ManagedPackageAddress{}, err
	}
	if !IsSkillDefinitionFile(locator) {
		return source.ManagedPackageAddress{}, fmt.Errorf(
			"%w: Skill locator %q is not a configured Skill package document",
			model.ErrInvalid,
			locator,
		)
	}

	skillDirectory := model.Locator(path.Dir(string(locator)))
	packageDirectory := model.Locator(path.Dir(
		string(skillDirectory),
	))

	address, err := source.ParseManagedPackageAddressDirectory(
		packageDirectory,
	)
	if err != nil {
		return source.ManagedPackageAddress{}, err
	}
	if err := validateManagedSkillPackageAddress(address); err != nil {
		return source.ManagedPackageAddress{}, err
	}

	if path.Base(string(skillDirectory)) != string(address.Name) {
		return source.ManagedPackageAddress{}, fmt.Errorf(
			"%w: managed Skill directory %q must match Skill name %q",
			model.ErrInvalid,
			skillDirectory,
			address.Name,
		)
	}

	expected, err := ManagedPackageLocatorForSkill(address)
	if err != nil {
		return source.ManagedPackageAddress{}, err
	}
	if locator != expected {
		return source.ManagedPackageAddress{}, fmt.Errorf(
			"%w: managed Skill locator %q must be %q",
			model.ErrInvalid,
			locator,
			expected,
		)
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
			model.ErrInvalid,
			ManagedSkillPackageKind,
		)
	}
	return nil
}
