package domain

import (
	"fmt"
	"path"

	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func ManagedPackageAddressForSkill(
	name spec.LogicalName,
	version spec.LogicalVersion,
) (sourceModel.ManagedPackageAddress, error) {
	if version == "" {
		version = documentTopology.UnversionedPackageVersion()
	}
	return sourceModel.NewManagedPackageAddress(
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
	address sourceModel.ManagedPackageAddress,
) (spec.Locator, error) {
	if err := validateManagedSkillPackageAddress(address); err != nil {
		return "", err
	}

	packageDirectory, err := address.Directory()
	if err != nil {
		return "", err
	}
	directory := spec.Locator(path.Join(
		string(packageDirectory),
		string(address.Name),
	))
	if err := directory.ValidatePortable(false); err != nil {
		return "", err
	}
	return directory, nil
}

func ManagedPackageLocatorForSkill(
	address sourceModel.ManagedPackageAddress,
) (spec.Locator, error) {
	directory, err := ManagedSkillDirectoryLocator(address)
	if err != nil {
		return "", err
	}
	locator := spec.Locator(path.Join(
		string(directory),
		string(SkillDefinitionFileName()),
	))
	if err := locator.ValidatePortable(false); err != nil {
		return "", err
	}
	return locator, nil
}

func ManagedPackageAddressFromSkillLocator(
	locator spec.Locator,
) (sourceModel.ManagedPackageAddress, error) {
	if err := locator.ValidatePortable(false); err != nil {
		return sourceModel.ManagedPackageAddress{}, err
	}
	if !IsSkillDefinitionFile(locator) {
		return sourceModel.ManagedPackageAddress{}, fmt.Errorf(
			"%w: Skill locator %q is not a configured Skill package document",
			spec.ErrInvalid,
			locator,
		)
	}

	skillDirectory := spec.Locator(path.Dir(string(locator)))
	packageDirectory := spec.Locator(path.Dir(
		string(skillDirectory),
	))

	address, err := sourceModel.ParseManagedPackageAddressDirectory(
		packageDirectory,
	)
	if err != nil {
		return sourceModel.ManagedPackageAddress{}, err
	}
	if err := validateManagedSkillPackageAddress(address); err != nil {
		return sourceModel.ManagedPackageAddress{}, err
	}

	if path.Base(string(skillDirectory)) != string(address.Name) {
		return sourceModel.ManagedPackageAddress{}, fmt.Errorf(
			"%w: managed Skill directory %q must match Skill name %q",
			spec.ErrInvalid,
			skillDirectory,
			address.Name,
		)
	}

	expected, err := ManagedPackageLocatorForSkill(address)
	if err != nil {
		return sourceModel.ManagedPackageAddress{}, err
	}
	if locator != expected {
		return sourceModel.ManagedPackageAddress{}, fmt.Errorf(
			"%w: managed Skill locator %q must be %q",
			spec.ErrInvalid,
			locator,
			expected,
		)
	}
	return address, nil
}

func validateManagedSkillPackageAddress(
	address sourceModel.ManagedPackageAddress,
) error {
	if err := address.Validate(); err != nil {
		return err
	}
	if address.Kind != ManagedSkillPackageKind {
		return fmt.Errorf(
			"%w: Skill package kind must be %q",
			spec.ErrInvalid,
			ManagedSkillPackageKind,
		)
	}
	return nil
}
