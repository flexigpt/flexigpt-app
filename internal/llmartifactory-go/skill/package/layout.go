package skillpackage

import (
	"fmt"
	"path"

	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/support"
)

func ManagedPackageAddressForSkill(
	layout support.PackageLayout,
	name storeSpec.LogicalName,
	version storeSpec.LogicalVersion,
) (managedpackageModel.ManagedPackageAddress, error) {
	return layout.Address(name, version)
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
	layout support.PackageLayout,
	address managedpackageModel.ManagedPackageAddress,
) (storeSpec.Locator, error) {
	if err := validateManagedSkillPackageAddress(layout, address); err != nil {
		return "", err
	}

	packageDirectory, err := address.Directory()
	if err != nil {
		return "", err
	}
	directory := storeSpec.Locator(path.Join(
		string(packageDirectory),
		string(address.Name),
	))
	if err := directory.ValidatePortable(false); err != nil {
		return "", err
	}
	return directory, nil
}

func ManagedPackageLocatorForSkill(
	layout support.PackageLayout,
	address managedpackageModel.ManagedPackageAddress,
) (storeSpec.Locator, error) {
	directory, err := ManagedSkillDirectoryLocator(layout, address)
	if err != nil {
		return "", err
	}
	locator := storeSpec.Locator(path.Join(
		string(directory),
		string(layout.Document.Locator),
	))
	if err := locator.ValidatePortable(false); err != nil {
		return "", err
	}
	return locator, nil
}

func ManagedPackageAddressFromSkillLocator(
	layout support.PackageLayout,
	locator storeSpec.Locator,
) (managedpackageModel.ManagedPackageAddress, error) {
	if err := locator.ValidatePortable(false); err != nil {
		return managedpackageModel.ManagedPackageAddress{}, err
	}
	if path.Base(string(locator)) != string(layout.Document.Locator) {
		return managedpackageModel.ManagedPackageAddress{}, fmt.Errorf(
			"%w: Skill locator %q is not a configured Skill package document",
			storeSpec.ErrInvalid,
			locator,
		)
	}

	skillDirectory := storeSpec.Locator(path.Dir(string(locator)))
	packageDirectory := storeSpec.Locator(path.Dir(
		string(skillDirectory),
	))

	address, err := managedpackageModel.ParseManagedPackageAddressDirectory(
		packageDirectory,
	)
	if err != nil {
		return managedpackageModel.ManagedPackageAddress{}, err
	}
	if err := validateManagedSkillPackageAddress(layout, address); err != nil {
		return managedpackageModel.ManagedPackageAddress{}, err
	}

	if path.Base(string(skillDirectory)) != string(address.Name) {
		return managedpackageModel.ManagedPackageAddress{}, fmt.Errorf(
			"%w: managed Skill directory %q must match Skill name %q",
			storeSpec.ErrInvalid,
			skillDirectory,
			address.Name,
		)
	}

	expected, err := ManagedPackageLocatorForSkill(layout, address)
	if err != nil {
		return managedpackageModel.ManagedPackageAddress{}, err
	}
	if locator != expected {
		return managedpackageModel.ManagedPackageAddress{}, fmt.Errorf(
			"%w: managed Skill locator %q must be %q",
			storeSpec.ErrInvalid,
			locator,
			expected,
		)
	}
	return address, nil
}

func validateManagedSkillPackageAddress(
	layout support.PackageLayout,
	address managedpackageModel.ManagedPackageAddress,
) error {
	if err := layout.Validate(); err != nil {
		return err
	}
	if err := address.Validate(); err != nil {
		return err
	}
	if address.Kind != layout.Kind {
		return fmt.Errorf(
			"%w: Skill package kind must be %q",
			storeSpec.ErrInvalid,
			layout.Kind,
		)
	}
	return nil
}
