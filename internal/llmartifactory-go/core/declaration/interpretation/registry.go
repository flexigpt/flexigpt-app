// Package interpretation owns the registration boundary between shared
// declaration grammar and family-owned declaration semantics.
package interpretation

import (
	"fmt"
	"sort"
	"strings"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition"
	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

// Registration is one family-owned interpretation registration. It does not
// carry source drivers, runtime adapters, content inventory, persistence, or
// application topology.
type Registration struct {
	DeclarationType declaration.Type
	SchemaKey       schemaModel.Key

	ValidateEntry func(declaration.Entry) error

	// ValidateAdmittedEntry validates family semantics of an Entry whose
	// enclosing Definition has already passed schema admission. It must not
	// execute the family JSON Schema again.
	ValidateAdmittedEntry func(declaration.Entry) error

	// Relationships returns direct family-owned relationship facts. The core
	// traverses these facts but does not interpret family document fields.
	Relationships func(declaration.Entry) ([]Relationship, error)

	// LogicalVersion supplies family-specific Artifact identity qualification.
	// Text uses its insertion target; most families return an empty version.
	LogicalVersion func(declaration.Entry) (spec.LogicalVersion, error)

	// SelectorEligible identifies target families that can participate in
	// generic selector expansion.
	SelectorEligible bool

	// DeclarationAliasEligible identifies declaration types whose locator is
	// a source-selected alias rather than resource material.
	DeclarationAliasEligible bool

	// LocatorCandidates permits a family to expand a source-relative locator
	// into family-owned candidate declaration entries. Skill uses this for
	// directory locators and SKILL.md.
	LocatorCandidates func(spec.Locator) ([]spec.Locator, error)
}

func (r Registration) Validate() error {
	if err := r.DeclarationType.Validate(); err != nil {
		return err
	}
	if err := r.SchemaKey.Validate(); err != nil {
		return err
	}
	if r.SchemaKey.Entity != schemaModel.EntityArtifact ||
		r.SchemaKey.Kind != schemaModel.Kind(r.DeclarationType) {
		return fmt.Errorf(
			"%w: declaration interpretation schema key does not match %q",
			spec.ErrInvalid,
			r.DeclarationType,
		)
	}
	if r.ValidateEntry == nil {
		return fmt.Errorf(
			"%w: declaration interpretation %q has no semantic validator",
			spec.ErrInvalid,
			r.DeclarationType,
		)
	}
	if r.ValidateAdmittedEntry == nil {
		return fmt.Errorf(
			"%w: declaration interpretation %q has no admitted semantic validator",
			spec.ErrInvalid,
			r.DeclarationType,
		)
	}
	return nil
}

type Registry struct {
	bySchema map[schemaModel.Key]Registration
	byType   map[declaration.Type][]Registration
	keys     []schemaModel.Key
}

func NewRegistry(
	registrations ...Registration,
) (*Registry, error) {
	if len(registrations) == 0 {
		return nil, fmt.Errorf(
			"%w: declaration interpretation registry is empty",
			spec.ErrInvalid,
		)
	}

	output := &Registry{
		bySchema: make(map[schemaModel.Key]Registration, len(registrations)),
		byType:   make(map[declaration.Type][]Registration),
		keys:     make([]schemaModel.Key, 0, len(registrations)),
	}
	for index, registration := range registrations {
		if err := registration.Validate(); err != nil {
			return nil, fmt.Errorf(
				"declaration interpretation %d: %w",
				index,
				err,
			)
		}
		if _, duplicate := output.bySchema[registration.SchemaKey]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate declaration interpretation schema %q/%q/%q",
				spec.ErrConflict,
				registration.SchemaKey.Kind,
				registration.SchemaKey.SchemaID,
				registration.SchemaKey.SchemaVersion,
			)
		}

		output.bySchema[registration.SchemaKey] = registration
		output.byType[registration.DeclarationType] = append(
			output.byType[registration.DeclarationType],
			registration,
		)
		output.keys = append(output.keys, registration.SchemaKey)
	}
	sort.Slice(output.keys, func(left, right int) bool {
		a := output.keys[left]
		b := output.keys[right]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.SchemaID != b.SchemaID {
			return a.SchemaID < b.SchemaID
		}
		return a.SchemaVersion < b.SchemaVersion
	})
	return output, nil
}

func (r *Registry) SchemaKeys() []schemaModel.Key {
	if r == nil {
		return nil
	}
	return append([]schemaModel.Key(nil), r.keys...)
}

func (r *Registry) SupportsType(
	value declaration.Type,
) bool {
	if r == nil {
		return false
	}
	_, found := r.byType[value]
	return found
}

func (r *Registry) SelectorEligible(
	value declaration.Type,
) bool {
	values := r.byType[value]
	for _, registration := range values {
		if registration.SelectorEligible {
			return true
		}
	}
	return false
}

func (r *Registry) DeclarationAliasEligible(
	value declaration.Type,
) bool {
	values := r.byType[value]
	for _, registration := range values {
		if registration.DeclarationAliasEligible {
			return true
		}
	}
	return false
}

func (r *Registry) LocatorCandidates(
	kind artifactModel.ArtifactKind,
	target spec.Locator,
) ([]spec.Locator, error) {
	if r == nil {
		return nil, spec.ErrClosed
	}
	declarationType := declaration.Type(kind)
	values := r.byType[declarationType]
	if len(values) == 0 {
		return nil, fmt.Errorf(
			"%w: no declaration interpretation for %q",
			spec.ErrUnsupported,
			kind,
		)
	}
	if len(values) != 1 {
		return nil, fmt.Errorf(
			"%w: declaration type %q has multiple schema versions",
			spec.ErrUnsupported,
			declarationType,
		)
	}
	if values[0].LocatorCandidates == nil {
		return []spec.Locator{target}, nil
	}
	return values[0].LocatorCandidates(target)
}

func (r *Registry) Relationships(
	entry declaration.Entry,
) ([]Relationship, error) {
	registration, err := r.registrationForEntry(entry)
	if err != nil {
		return nil, err
	}
	if err := registration.ValidateEntry(entry); err != nil {
		return nil, err
	}
	return r.relationshipsForValidatedEntry(registration, entry)
}

// AdmittedRelationships extracts relationship facts from an Entry that was
// reconstructed from an admitted Definition. It intentionally invokes only
// family semantic validation and never repeats schema execution.
func (r *Registry) AdmittedRelationships(
	entry declaration.Entry,
) ([]Relationship, error) {
	registration, err := r.registrationForEntry(entry)
	if err != nil {
		return nil, err
	}
	if err := registration.ValidateAdmittedEntry(entry); err != nil {
		return nil, err
	}
	return r.relationshipsForValidatedEntry(registration, entry)
}

func (r *Registry) ValidateTree(
	root declaration.Entry,
) error {
	_, err := r.walkNamedEntries(root)
	return err
}

func (r *Registry) WalkNamedEntries(
	root declaration.Entry,
) ([]NamedEntry, error) {
	return r.walkNamedEntries(root)
}

// DefinitionForSchema reconstructs the immutable generic Definition from a
// family-owned validated declaration. It does not rerun JSON Schema execution.
func (r *Registry) DefinitionForSchema(
	key schemaModel.Key,
	entry declaration.Entry,
) (definitionModel.Definition, error) {
	registration, err := r.registrationForSchema(key)
	if err != nil {
		return definitionModel.Definition{}, err
	}
	if err := registration.ValidateEntry(entry); err != nil {
		return definitionModel.Definition{}, err
	}
	return r.definitionForValidatedSchema(
		registration,
		key,
		entry,
	)
}

func (r *Registry) DefinitionForEntry(
	entry declaration.Entry,
) (definitionModel.Definition, error) {
	registration, err := r.registrationForEntry(entry)
	if err != nil {
		return definitionModel.Definition{}, err
	}
	if err := registration.ValidateEntry(entry); err != nil {
		return definitionModel.Definition{}, err
	}
	return r.definitionForValidatedSchema(
		registration,
		registration.SchemaKey,
		entry,
	)
}

func (r *Registry) DefinitionsForDocument(
	key schemaModel.Key,
	raw []byte,
) ([]AdmittedEntry, error) {
	return r.definitionsForDocument(key, raw, false)
}

// DefinitionsForSchemaValidatedDocument reconstructs Definitions after the
// generic expected-key schema catalog has already validated the root
// declaration. Nested contained declarations still receive their owning
// family schema validation during tree admission.
func (r *Registry) DefinitionsForSchemaValidatedDocument(
	key schemaModel.Key,
	raw []byte,
) ([]AdmittedEntry, error) {
	return r.definitionsForDocument(key, raw, true)
}

// EntryFromDefinition performs family-owned typed reconstruction preparation.
// It verifies the admitted Definition linkage and family semantic invariants
// without executing the admission schema or recalculating the digest.
func (r *Registry) EntryFromDefinition(
	value definitionModel.Definition,
) (declaration.Entry, error) {
	if err := definitionModel.ValidateAdmitted(value); err != nil {
		return declaration.Entry{}, err
	}
	key := schemaModel.ArtifactKey(
		value.Kind,
		value.SchemaID,
		value.SchemaVersion,
	)
	registration, err := r.registrationForSchema(key)
	if err != nil {
		return declaration.Entry{}, err
	}
	entry, err := declaration.DecodeCanonicalEntryJSON(value.Body)
	if err != nil {
		return declaration.Entry{}, err
	}
	if entry.Header().Type != registration.DeclarationType ||
		entry.Header().Name != string(value.LogicalName) {
		return declaration.Entry{}, fmt.Errorf(
			"%w: Definition body identity differs from Definition metadata",
			spec.ErrDigestMismatch,
		)
	}
	if err := registration.ValidateAdmittedEntry(entry); err != nil {
		return declaration.Entry{}, err
	}
	if registration.LogicalVersion != nil {
		version, err := registration.LogicalVersion(entry)
		if err != nil {
			return declaration.Entry{}, err
		}
		if version != value.LogicalVersion {
			return declaration.Entry{}, fmt.Errorf(
				"%w: Definition logical version differs from family identity",
				spec.ErrDigestMismatch,
			)
		}
	}
	return entry, nil
}

func (r *Registry) definitionForValidatedSchema(
	registration Registration,
	key schemaModel.Key,
	entry declaration.Entry,
) (definitionModel.Definition, error) {
	if entry.Header().Type != registration.DeclarationType {
		return definitionModel.Definition{}, fmt.Errorf(
			"%w: declaration type differs from schema interpretation",
			spec.ErrInvalid,
		)
	}

	version := spec.LogicalVersion("")
	var err error
	if registration.LogicalVersion != nil {
		version, err = registration.LogicalVersion(entry)
		if err != nil {
			return definitionModel.Definition{}, err
		}
	}
	if err := version.Validate(true); err != nil {
		return definitionModel.Definition{}, err
	}

	header := entry.Header()
	body, err := entry.CanonicalJSON()
	if err != nil {
		return definitionModel.Definition{}, err
	}
	displayName := header.DisplayName
	if displayName == "" {
		displayName = header.Name
	}

	return definition.Admit(definitionModel.Definition{
		Kind:           artifactModel.ArtifactKind(registration.DeclarationType),
		SchemaID:       key.SchemaID,
		SchemaVersion:  key.SchemaVersion,
		LogicalName:    spec.LogicalName(header.Name),
		LogicalVersion: version,
		DisplayName:    displayName,
		Description:    header.Description,
		Labels:         declaration.CloneStringMap(header.Labels),
		Body:           body,
	})
}

func (r *Registry) walkNamedEntries(
	root declaration.Entry,
) ([]NamedEntry, error) {
	return r.walkNamedEntriesWithRootValidation(root, false)
}

func (r *Registry) walkNamedEntriesWithRootValidation(
	root declaration.Entry,
	rootSchemaValidated bool,
) ([]NamedEntry, error) {
	if r == nil {
		return nil, spec.ErrClosed
	}
	if err := root.Validate(); err != nil {
		return nil, err
	}
	if root.Header().Name == "" {
		return nil, fmt.Errorf(
			"%w: declaration root requires a name",
			spec.ErrInvalid,
		)
	}

	output := make([]NamedEntry, 0)
	seen := make(map[spec.SubresourceLocator]struct{})

	var walk func(
		declaration.Entry,
		[]string,
		int,
		bool,
	) error
	walk = func(
		entry declaration.Entry,
		path []string,
		depth int,
		schemaValidated bool,
	) error {
		if depth > spec.MaxDiscoveryDepth {
			return fmt.Errorf(
				"%w: declaration nesting exceeds depth %d",
				spec.ErrInvalid,
				spec.MaxDiscoveryDepth,
			)
		}

		registration, err := r.registrationForEntry(entry)
		if err != nil {
			return err
		}
		if schemaValidated {
			if err := registration.ValidateAdmittedEntry(entry); err != nil {
				return err
			}
		} else {
			if err := registration.ValidateEntry(entry); err != nil {
				return err
			}
		}

		subresource := spec.SubresourceLocator(strings.Join(path, "/"))
		if err := subresource.Validate(); err != nil {
			return err
		}
		if _, duplicate := seen[subresource]; duplicate {
			return fmt.Errorf(
				"%w: declaration emits duplicate subresource %q",
				spec.ErrIdentityConflict,
				subresource,
			)
		}
		seen[subresource] = struct{}{}
		output = append(output, NamedEntry{
			SubresourceLocator: subresource,
			Entry:              entry.Clone(),
		})

		relationships, err := r.relationshipsForValidatedEntry(
			registration,
			entry,
		)
		if err != nil {
			return err
		}
		for _, relationship := range relationships {
			if relationship.Form != declaration.MemberContained {
				continue
			}
			target, err := relationship.Declared.ContainedDeclaration()
			if err != nil {
				return err
			}
			if err := walk(
				target,
				append([]string(nil), relationship.ContainedPath...),
				depth+1,
				false,
			); err != nil {
				return err
			}
		}
		return nil
	}

	if err := walk(root, nil, 0, rootSchemaValidated); err != nil {
		return nil, err
	}
	return output, nil
}

func (r *Registry) relationshipsForValidatedEntry(
	registration Registration,
	entry declaration.Entry,
) ([]Relationship, error) {
	if registration.Relationships == nil {
		return []Relationship{}, nil
	}

	values, err := registration.Relationships(entry)
	if err != nil {
		return nil, err
	}
	output := make([]Relationship, len(values))
	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		if err := value.Validate(); err != nil {
			return nil, fmt.Errorf(
				"declaration relationship %d: %w",
				index,
				err,
			)
		}
		key := strings.Join(value.Path, "\x00")
		if _, duplicate := seen[key]; duplicate {
			return nil, fmt.Errorf(
				"%w: declaration relationship path %q is repeated",
				spec.ErrIdentityConflict,
				strings.Join(value.Path, "/"),
			)
		}
		seen[key] = struct{}{}
		output[index] = value.Clone()
	}
	sort.SliceStable(output, func(left, right int) bool {
		return strings.Join(output[left].Path, "\x00") <
			strings.Join(output[right].Path, "\x00")
	})
	return output, nil
}

func (r *Registry) definitionsForDocument(
	key schemaModel.Key,
	raw []byte,
	rootSchemaValidated bool,
) ([]AdmittedEntry, error) {
	registration, err := r.registrationForSchema(key)
	if err != nil {
		return nil, err
	}
	root, err := declaration.DecodeCanonicalEntryJSON(raw)
	if err != nil {
		return nil, err
	}
	if root.Header().Type != registration.DeclarationType {
		return nil, fmt.Errorf(
			"%w: declaration header does not match expected schema",
			spec.ErrInvalid,
		)
	}

	entries, err := r.walkNamedEntriesWithRootValidation(
		root,
		rootSchemaValidated,
	)
	if err != nil {
		return nil, err
	}
	output := make([]AdmittedEntry, 0, len(entries))
	for index, entry := range entries {
		var definitionValue definitionModel.Definition
		if index == 0 {
			definitionValue, err = r.definitionForValidatedSchema(
				registration,
				key,
				entry.Entry,
			)
		} else {
			nested, nestedErr := r.registrationForEntry(entry.Entry)
			if nestedErr != nil {
				return nil, nestedErr
			}
			definitionValue, err = r.definitionForValidatedSchema(
				nested,
				nested.SchemaKey,
				entry.Entry,
			)
		}
		if err != nil {
			return nil, err
		}
		output = append(output, AdmittedEntry{
			SubresourceLocator: entry.SubresourceLocator,
			Definition:         definitionValue,
		})
	}
	return output, nil
}

func (r *Registry) registrationForSchema(
	key schemaModel.Key,
) (Registration, error) {
	if r == nil {
		return Registration{}, spec.ErrClosed
	}
	value, found := r.bySchema[key]
	if !found {
		return Registration{}, fmt.Errorf(
			"%w: declaration schema %q/%q/%q",
			spec.ErrUnsupported,
			key.Kind,
			key.SchemaID,
			key.SchemaVersion,
		)
	}
	return value, nil
}

func (r *Registry) registrationForEntry(
	entry declaration.Entry,
) (Registration, error) {
	if r == nil {
		return Registration{}, spec.ErrClosed
	}
	header := entry.Header()
	values := r.byType[header.Type]
	if len(values) == 0 {
		return Registration{}, fmt.Errorf(
			"%w: declaration type %q",
			spec.ErrUnsupported,
			header.Type,
		)
	}
	if len(values) != 1 {
		return Registration{}, fmt.Errorf(
			"%w: declaration type %q requires an explicit schema key",
			spec.ErrUnsupported,
			header.Type,
		)
	}
	return values[0], nil
}

type NamedEntry struct {
	SubresourceLocator spec.SubresourceLocator
	Entry              declaration.Entry
}

func (e NamedEntry) Clone() NamedEntry {
	return NamedEntry{
		SubresourceLocator: e.SubresourceLocator,
		Entry:              e.Entry.Clone(),
	}
}

type AdmittedEntry struct {
	SubresourceLocator spec.SubresourceLocator
	Definition         definitionModel.Definition
}

func (e AdmittedEntry) Clone() AdmittedEntry {
	return AdmittedEntry{
		SubresourceLocator: e.SubresourceLocator,
		Definition:         e.Definition.Clone(),
	}
}

func shortIdentity(
	entry declaration.Entry,
) (string, error) {
	raw, err := declaration.MemberIdentityJSON(entry)
	if err != nil {
		return "", err
	}
	digest := strings.TrimPrefix(
		string(cryptoutil.DigestBytes(raw)),
		cryptoutil.DigestSHA256Prefix,
	)
	if len(digest) < 16 {
		return "", fmt.Errorf(
			"%w: declaration member identity digest is invalid",
			spec.ErrInvalid,
		)
	}
	return digest[:16], nil
}
