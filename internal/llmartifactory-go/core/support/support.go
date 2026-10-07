// Package support contains application-supplied, immutable support values used
// by LLM artifact families. It deliberately contains no Source driver,
// persistence, runtime, topology, built-in content, or Wails dependency.
package support

import (
	"encoding/json"
	"fmt"
	"path"
	"strings"

	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// Document identifies one application-selected package-root declaration file
// and the decoder that must be used for that file.
type Document struct {
	Locator   spec.Locator
	DecoderID spec.DecoderID
}

func (d Document) Validate() error {
	if err := d.Locator.ValidatePortable(false); err != nil {
		return err
	}
	if path.Base(string(d.Locator)) != string(d.Locator) {
		return fmt.Errorf(
			"%w: supported document %q must be a package-root filename",
			spec.ErrInvalid,
			d.Locator,
		)
	}
	return d.DecoderID.Validate()
}

func (d Document) Matches(locator spec.Locator) bool {
	return path.Base(string(locator)) == string(d.Locator)
}

// NamedFile derives a portable export filename from an application-selected
// package document filename. For example, document "agent.yaml" yields
// "<name>.agent.yaml"; the family does not hard-code that suffix.
func (d Document) NamedFile(name spec.LogicalName) (string, error) {
	if err := d.Validate(); err != nil {
		return "", err
	}
	if err := name.Validate(); err != nil {
		return "", err
	}

	value := string(name) + "." + string(d.Locator)
	if err := spec.Locator(value).ValidatePortable(false); err != nil {
		return "", err
	}
	return value, nil
}

// Documents is the family-selected set of equivalent source document names.
// Default is used for managed publication and export; Files is used for
// package inspection. Patterns extends source-format matching to declared
// suffix forms such as "reviewer.agent.yaml" without teaching an artifact
// family which filename conventions the application selected.
type Documents struct {
	Default  Document
	Files    []spec.Locator
	Patterns []string
}

func (d Documents) Validate() error {
	if err := d.Default.Validate(); err != nil {
		return err
	}

	if len(d.Files) == 0 {
		return fmt.Errorf(
			"%w: supported document set is empty",
			spec.ErrInvalid,
		)
	}

	seen := make(map[spec.Locator]struct{}, len(d.Files))
	defaultFound := false
	for _, value := range d.Files {
		if err := value.ValidatePortable(false); err != nil {
			return err
		}
		if path.Base(string(value)) != string(value) {
			return fmt.Errorf(
				"%w: supported document %q must be a package-root filename",
				spec.ErrInvalid,
				value,
			)
		}
		if _, duplicate := seen[value]; duplicate {
			return fmt.Errorf(
				"%w: supported document %q is repeated",
				spec.ErrConflict,
				value,
			)
		}
		seen[value] = struct{}{}
		if value == d.Default.Locator {
			defaultFound = true
		}
	}
	if !defaultFound {
		return fmt.Errorf(
			"%w: default document %q is not in the supported document set",
			spec.ErrInvalid,
			d.Default.Locator,
		)
	}

	if len(d.Patterns) != 0 {
		if err := spec.ValidatePathPatterns(
			"supported document patterns",
			d.Patterns,
		); err != nil {
			return err
		}

		seenPatterns := make(map[string]struct{}, len(d.Patterns))
		for _, pattern := range d.Patterns {
			key := strings.ToLower(pattern)
			if _, duplicate := seenPatterns[key]; duplicate {
				return fmt.Errorf(
					"%w: supported document pattern %q is repeated",
					spec.ErrConflict,
					pattern,
				)
			}
			seenPatterns[key] = struct{}{}
		}
	}
	return nil
}

func (d Documents) Matches(locator spec.Locator) bool {
	value := strings.ToLower(string(locator))
	name := strings.ToLower(path.Base(value))

	for _, file := range d.Files {
		if strings.EqualFold(string(file), name) {
			return true
		}
	}
	for _, pattern := range d.Patterns {
		pattern = strings.ToLower(pattern)
		if matched, err := spec.MatchPathPattern(pattern, value); err == nil &&
			matched {
			return true
		}
		if matched, err := spec.MatchPathPattern(pattern, name); err == nil &&
			matched {
			return true
		}
	}
	return false
}

func (d Documents) Clone() Documents {
	output := d
	output.Files = append([]spec.Locator(nil), d.Files...)
	output.Patterns = append([]string(nil), d.Patterns...)
	return output
}

// PackageLayout describes application-selected physical package conventions.
// The artifact family owns how this layout is used; application setup owns the
// actual package kind, version policy, document filename, and decoder choice.
type PackageLayout struct {
	Kind           managedpackageModel.PackageKind
	DefaultVersion spec.LogicalVersion
	FixedVersion   bool
	Document       Document
}

func (p PackageLayout) Validate() error {
	if err := p.Kind.Validate(); err != nil {
		return err
	}
	if err := p.DefaultVersion.Validate(false); err != nil {
		return err
	}
	return p.Document.Validate()
}

func (p PackageLayout) Address(
	name spec.LogicalName,
	version spec.LogicalVersion,
) (managedpackageModel.ManagedPackageAddress, error) {
	if err := p.Validate(); err != nil {
		return managedpackageModel.ManagedPackageAddress{}, err
	}
	if version == "" {
		version = p.DefaultVersion
	}
	if p.FixedVersion && version != p.DefaultVersion {
		return managedpackageModel.ManagedPackageAddress{}, fmt.Errorf(
			"%w: package kind %q requires version %q",
			spec.ErrInvalid,
			p.Kind,
			p.DefaultVersion,
		)
	}
	return managedpackageModel.NewManagedPackageAddress(
		p.Kind,
		name,
		version,
	)
}

func (p PackageLayout) Locator(
	address managedpackageModel.ManagedPackageAddress,
) (spec.Locator, error) {
	if err := p.Validate(); err != nil {
		return "", err
	}
	if err := address.Validate(); err != nil {
		return "", err
	}
	if address.Kind != p.Kind {
		return "", fmt.Errorf(
			"%w: package kind is %q, expected %q",
			spec.ErrInvalid,
			address.Kind,
			p.Kind,
		)
	}
	if p.FixedVersion && address.Version != p.DefaultVersion {
		return "", fmt.Errorf(
			"%w: package version is %q, expected %q",
			spec.ErrInvalid,
			address.Version,
			p.DefaultVersion,
		)
	}
	return address.FileLocator(p.Document.Locator)
}

func (p PackageLayout) AddressFromLocator(
	locator spec.Locator,
) (managedpackageModel.ManagedPackageAddress, error) {
	if err := p.Validate(); err != nil {
		return managedpackageModel.ManagedPackageAddress{}, err
	}
	if err := locator.ValidatePortable(false); err != nil {
		return managedpackageModel.ManagedPackageAddress{}, err
	}
	if path.Base(string(locator)) != string(p.Document.Locator) {
		return managedpackageModel.ManagedPackageAddress{}, fmt.Errorf(
			"%w: package locator %q is not %q",
			spec.ErrUnsupported,
			locator,
			p.Document.Locator,
		)
	}

	address, err := managedpackageModel.ParseManagedPackageAddressDirectory(
		spec.Locator(path.Dir(string(locator))),
	)
	if err != nil {
		return managedpackageModel.ManagedPackageAddress{}, err
	}
	if address.Kind != p.Kind {
		return managedpackageModel.ManagedPackageAddress{}, fmt.Errorf(
			"%w: package kind is %q, expected %q",
			spec.ErrUnsupported,
			address.Kind,
			p.Kind,
		)
	}
	if p.FixedVersion && address.Version != p.DefaultVersion {
		return managedpackageModel.ManagedPackageAddress{}, fmt.Errorf(
			"%w: package version is %q, expected %q",
			spec.ErrUnsupported,
			address.Version,
			p.DefaultVersion,
		)
	}
	return address, nil
}

// SourceProfile is a product-selected managed Source declaration. It is not a
// Source driver and does not open, mutate, or inspect storage by itself.
type SourceProfile struct {
	StorageKey  spec.StorageKey
	Kind        sourceModel.SourceKind
	DisplayName string
	Config      json.RawMessage
}

func (p SourceProfile) Validate() error {
	if err := p.StorageKey.Validate(); err != nil {
		return err
	}
	if err := spec.ValidateIdentifier(
		"managed Source kind",
		string(p.Kind),
		spec.MaxKindBytes,
	); err != nil {
		return err
	}
	if err := spec.ValidateRequiredText(
		"managed Source display name",
		p.DisplayName,
		spec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if len(p.Config) == 0 {
		return nil
	}
	_, err := jsonutil.CanonicalizeObject(
		p.Config,
		spec.MaxDefinitionBodyBytes,
	)
	return err
}

func (p SourceProfile) Draft(
	id sourceModel.SourceID,
) sourceModel.Draft {
	config := append(json.RawMessage(nil), p.Config...)
	if len(config) == 0 {
		config = json.RawMessage(`{}`)
	}
	return sourceModel.Draft{
		ID:          id,
		StorageKey:  p.StorageKey,
		Kind:        p.Kind,
		DisplayName: p.DisplayName,
		Enabled:     true,
		Config:      config,
	}
}

func (p SourceProfile) Matches(
	value sourceModel.Summary,
) bool {
	return value.Kind == p.Kind &&
		value.StorageKey == p.StorageKey
}

func (p SourceProfile) MatchesSourceMetadata(
	value catalogModel.SourceMetadata,
) bool {
	return value.Kind == p.Kind &&
		value.StorageKey == p.StorageKey
}

// Candidates describes source filenames and patterns accepted by one decoder.
// The application support catalog supplies these values; decoder packages do
// not infer support from hard-coded extensions.
type Candidates struct {
	Patterns []string
}

func (c Candidates) Validate() error {
	if len(c.Patterns) == 0 {
		return fmt.Errorf(
			"%w: decoder candidate support is empty",
			spec.ErrInvalid,
		)
	}
	if err := spec.ValidatePathPatterns(
		"decoder candidate patterns",
		c.Patterns,
	); err != nil {
		return err
	}

	seen := make(map[string]struct{}, len(c.Patterns))
	for _, value := range c.Patterns {
		value = strings.ToLower(value)
		if _, duplicate := seen[value]; duplicate {
			return fmt.Errorf(
				"%w: decoder candidate pattern %q is repeated",
				spec.ErrConflict,
				value,
			)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func (c Candidates) Matches(
	locator spec.Locator,
) bool {
	value := strings.ToLower(string(locator))
	name := strings.ToLower(path.Base(value))
	for _, pattern := range c.Patterns {
		pattern = strings.ToLower(pattern)
		if matched, err := spec.MatchPathPattern(pattern, value); err == nil && matched {
			return true
		}
		if matched, err := spec.MatchPathPattern(pattern, name); err == nil && matched {
			return true
		}
	}
	return false
}

func (c Candidates) Clone() Candidates {
	return Candidates{
		Patterns: append([]string(nil), c.Patterns...),
	}
}
