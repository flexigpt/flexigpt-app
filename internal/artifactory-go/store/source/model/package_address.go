package source

import (
	"fmt"
	"path"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

type PackageKind string

func (v PackageKind) Validate() error {
	return spec.ValidateIdentifier("package kind", string(v), spec.MaxKindBytes)
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
	Kind    PackageKind         `json:"kind"`
	Name    spec.LogicalName    `json:"name"`
	Version spec.LogicalVersion `json:"version"`
}

func NewManagedPackageAddress(
	kind PackageKind,
	name spec.LogicalName,
	version spec.LogicalVersion,
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
	directory spec.Locator,
) (ManagedPackageAddress, error) {
	if err := directory.ValidatePortable(false); err != nil {
		return ManagedPackageAddress{}, err
	}

	segments := strings.Split(string(directory), "/")
	if len(segments) != 3 {
		return ManagedPackageAddress{}, fmt.Errorf(
			"%w: managed package directory %q must contain kind, name, and version",
			spec.ErrInvalid,
			directory,
		)
	}

	return NewManagedPackageAddress(
		PackageKind(segments[0]),
		spec.LogicalName(segments[1]),
		spec.LogicalVersion(segments[2]),
	)
}

func (a ManagedPackageAddress) Validate() error {
	if err := a.Kind.Validate(); err != nil {
		return err
	}
	if err := spec.ValidatePortableName("package name", string(a.Name)); err != nil {
		return err
	}
	return spec.ValidatePortableName("package version", string(a.Version))
}

// Directory returns the source-relative directory used by MapStore and normal
// filesystem users. It is derived from semantic package identity and never
// caller-supplied as an arbitrary directory.
func (a ManagedPackageAddress) Directory() (spec.Locator, error) {
	if err := a.Validate(); err != nil {
		return "", err
	}
	return spec.Locator(path.Join(
		string(a.Kind),
		string(a.Name),
		string(a.Version),
	)), nil
}

// FileLocator returns a source-relative locator for one package-relative
// regular file.
func (a ManagedPackageAddress) FileLocator(
	relative spec.Locator,
) (spec.Locator, error) {
	if err := relative.ValidatePortable(false); err != nil {
		return "", err
	}
	directory, err := a.Directory()
	if err != nil {
		return "", err
	}
	return spec.Locator(
		string(directory) + "/" + string(relative),
	), nil
}
