package source

import (
	"fmt"
	"path"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

type PackageKind string

func (v PackageKind) Validate() error {
	return basespec.ValidateIdentifier("package kind", string(v), basespec.MaxKindBytes)
}

// ManagedPackageAddress is the generic semantic address of one complete
// managed package.
//
// Artifact Store owns only this three-segment address shape:
//
//	<kind>/<name>/<version>
//
// Artifact families own the values of Kind, Name, Version, all primary file
// names, and all package-relative resource conventions.
type ManagedPackageAddress struct {
	Kind    PackageKind             `json:"kind"`
	Name    basespec.LogicalName    `json:"name"`
	Version basespec.LogicalVersion `json:"version"`
}

func NewManagedPackageAddress(
	kind PackageKind,
	name basespec.LogicalName,
	version basespec.LogicalVersion,
) (ManagedPackageAddress, error) {
	value := ManagedPackageAddress{
		Kind:    kind,
		Name:    name,
		Version: version,
	}
	if err := value.Validate(); err != nil {
		return ManagedPackageAddress{}, err
	}
	return value, nil
}

// ParseManagedPackageAddressDirectory decodes an address previously derived
// through Directory. It accepts no extra namespace or implementation segments.
func ParseManagedPackageAddressDirectory(
	directory basespec.Locator,
) (ManagedPackageAddress, error) {
	if err := directory.ValidatePortable(false); err != nil {
		return ManagedPackageAddress{}, err
	}

	segments := strings.Split(string(directory), "/")
	if len(segments) != 3 {
		return ManagedPackageAddress{}, fmt.Errorf(
			"%w: managed package directory %q must contain kind, name, and version",
			basespec.ErrInvalid,
			directory,
		)
	}

	return NewManagedPackageAddress(
		PackageKind(segments[0]),
		basespec.LogicalName(segments[1]),
		basespec.LogicalVersion(segments[2]),
	)
}

func (a ManagedPackageAddress) Validate() error {
	if err := a.Kind.Validate(); err != nil {
		return err
	}
	if err := basespec.ValidatePortableName("package name", string(a.Name)); err != nil {
		return err
	}
	return basespec.ValidatePortableName("package version", string(a.Version))
}

// Directory returns the source-relative directory used by MapStore and normal
// filesystem users. It is derived from semantic package identity and never
// caller-supplied as an arbitrary directory.
func (a ManagedPackageAddress) Directory() (basespec.Locator, error) {
	if err := a.Validate(); err != nil {
		return "", err
	}
	value := basespec.Locator(path.Join(
		string(a.Kind),
		string(a.Name),
		string(a.Version),
	))
	if err := value.ValidatePortable(false); err != nil {
		return "", err
	}
	return value, nil
}

// FileLocator returns a source-relative locator for one package-relative
// regular file.
func (a ManagedPackageAddress) FileLocator(
	relative basespec.Locator,
) (basespec.Locator, error) {
	if err := relative.ValidatePortable(false); err != nil {
		return "", err
	}
	directory, err := a.Directory()
	if err != nil {
		return "", err
	}
	return basespec.Locator(
		string(directory) + "/" + string(relative),
	), nil
}
