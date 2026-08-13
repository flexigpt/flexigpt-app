package source

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

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
	Kind    basespec.PackageKind    `json:"kind"`
	Name    basespec.LogicalName    `json:"name"`
	Version basespec.LogicalVersion `json:"version"`
}

func NewManagedPackageAddress(
	kind basespec.PackageKind,
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

func (a ManagedPackageAddress) Validate() error {
	if err := basespec.ValidatePackageKind(a.Kind); err != nil {
		return err
	}
	if err := basespec.ValidatePackageName(a.Name); err != nil {
		return err
	}
	return basespec.ValidatePackageVersion(a.Version)
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
	if err := basespec.ValidatePortableLocator(value, false); err != nil {
		return "", err
	}
	return value, nil
}

// FileLocator returns a source-relative locator for one package-relative
// regular file.
func (a ManagedPackageAddress) FileLocator(
	relative basespec.Locator,
) (basespec.Locator, error) {
	if err := basespec.ValidatePortableLocator(relative, false); err != nil {
		return "", err
	}
	directory, err := a.Directory()
	if err != nil {
		return "", err
	}
	value := basespec.Locator(path.Join(
		string(directory),
		string(relative),
	))
	if err := basespec.ValidatePortableLocator(value, false); err != nil {
		return "", err
	}
	return value, nil
}

// ParseManagedPackageAddressDirectory decodes an address previously derived
// through Directory. It accepts no extra namespace or implementation segments.
func ParseManagedPackageAddressDirectory(
	directory basespec.Locator,
) (ManagedPackageAddress, error) {
	if err := basespec.ValidatePortableLocator(directory, false); err != nil {
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
		basespec.PackageKind(segments[0]),
		basespec.LogicalName(segments[1]),
		basespec.LogicalVersion(segments[2]),
	)
}

// ManagedPackageFile is one regular file relative to a managed package root.
// Empty directories are deliberately not represented.
type ManagedPackageFile struct {
	Locator basespec.Locator `json:"locator"`
	Content []byte           `json:"content"`
}

// NormalizeManagedPackageFiles validates and returns an independently owned,
// deterministic package file set. It is separate from package addressing so a
// caller can validate package contents before it has derived the final semantic
// package address.
func NormalizeManagedPackageFiles(
	input []ManagedPackageFile,
) ([]ManagedPackageFile, error) {
	if len(input) == 0 {
		return nil, fmt.Errorf(
			"%w: managed package must contain at least one file",
			basespec.ErrInvalid,
		)
	}
	if len(input) > basespec.MaxDiscoveryEntries {
		return nil, fmt.Errorf(
			"%w: managed package exceeds the file count limit",
			basespec.ErrInvalid,
		)
	}

	output := make([]ManagedPackageFile, len(input))
	seen := make(map[basespec.Locator]struct{}, len(input))
	filesByIdentity := make(map[string]basespec.Locator, len(input))
	directoriesByIdentity := make(map[string]basespec.Locator)

	var total int64
	for index, file := range input {
		if err := basespec.ValidatePortableLocator(file.Locator, false); err != nil {
			return nil, fmt.Errorf(
				"managed package files[%d]: %w",
				index,
				err,
			)
		}
		if _, duplicate := seen[file.Locator]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate managed package file %q",
				basespec.ErrInvalid,
				file.Locator,
			)
		}
		seen[file.Locator] = struct{}{}

		identity, err := basespec.PortableLocatorIdentity(
			file.Locator,
			false,
		)
		if err != nil {
			return nil, err
		}
		if previous, collision := filesByIdentity[identity]; collision {
			return nil, fmt.Errorf(
				"%w: managed package file %q collides with %q",
				basespec.ErrInvalid,
				file.Locator,
				previous,
			)
		}
		if child, conflict := directoriesByIdentity[identity]; conflict {
			return nil, fmt.Errorf(
				"%w: managed package file %q conflicts with child path %q",
				basespec.ErrInvalid,
				file.Locator,
				child,
			)
		}

		for parent := path.Dir(string(file.Locator)); parent != "."; parent = path.Dir(parent) {
			parentIdentity, err := basespec.PortableLocatorIdentity(
				basespec.Locator(parent),
				false,
			)
			if err != nil {
				return nil, err
			}
			if parentFile, conflict := filesByIdentity[parentIdentity]; conflict {
				return nil, fmt.Errorf(
					"%w: managed package path %q is below file %q",
					basespec.ErrInvalid,
					file.Locator,
					parentFile,
				)
			}
			directoriesByIdentity[parentIdentity] = file.Locator
		}
		filesByIdentity[identity] = file.Locator

		if int64(len(file.Content)) > basespec.MaxScanBytes-total {
			return nil, fmt.Errorf(
				"%w: managed package exceeds the byte limit",
				basespec.ErrInvalid,
			)
		}
		total += int64(len(file.Content))

		output[index] = ManagedPackageFile{
			Locator: file.Locator,
			Content: append([]byte(nil), file.Content...),
		}
	}

	sort.Slice(output, func(left, right int) bool {
		return output[left].Locator < output[right].Locator
	})
	return output, nil
}

// ManagedPackagePublication atomically publishes one package at one semantic
// address. ExpectedGeneration is optional for creation and required by callers
// that intentionally replace existing package content.
type ManagedPackagePublication struct {
	Address            ManagedPackageAddress `json:"address"`
	ExpectedGeneration string                `json:"expectedGeneration,omitempty"`
	Files              []ManagedPackageFile  `json:"files"`
}

// NormalizeManagedPackagePublication validates both the semantic address and
// package files and returns fully owned deterministic data.
func NormalizeManagedPackagePublication(
	input ManagedPackagePublication,
) (ManagedPackagePublication, error) {
	if err := input.Address.Validate(); err != nil {
		return ManagedPackagePublication{}, err
	}
	if input.ExpectedGeneration != "" {
		if err := basespec.ValidateSourceGeneration(input.ExpectedGeneration); err != nil {
			return ManagedPackagePublication{}, err
		}
	}

	files, err := NormalizeManagedPackageFiles(input.Files)
	if err != nil {
		return ManagedPackagePublication{}, err
	}

	output := input
	output.Files = files
	return output, nil
}
