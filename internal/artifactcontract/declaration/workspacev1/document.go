package workspacev1

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	WorkspaceType          = declaration.TypeWorkspace
	WorkspaceSchemaID      = "artifact.workspace.v1"
	WorkspaceSchemaVersion = declaration.APIVersionV1
)

//go:embed workspace-v1.schema.json
var schemaJSON []byte

var compiledWorkspaceSchema = jsonutil.MustCompileJSONSchema(schemaJSON)

var WorkspaceSchemaKey = schema.ArtifactKey(
	artifact.ArtifactKind(WorkspaceType),
	schema.SchemaID(WorkspaceSchemaID),
	WorkspaceSchemaVersion,
)

type DeclarationScan struct {
	Base    declaration.Locator `json:"base"`
	Include []string            `json:"include"`
	Exclude []string            `json:"exclude,omitempty"`
}

// DeclarationSource is either a Locator, an inline declaration Entry, or an
// independent Workspace declaration scan.
type DeclarationSource struct {
	raw json.RawMessage
}

type WorkspaceDocument struct {
	declaration.Header

	Declarations []DeclarationSource `json:"declarations,omitempty"`
	Roots        []declaration.Entry `json:"roots,omitempty"`
}

func WorkspaceJSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
}

func NewLocatorDeclarationSource(
	value declaration.Locator,
) (DeclarationSource, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return DeclarationSource{}, err
	}
	return newCanonicalDeclarationSource(raw)
}

func NewEntryDeclarationSource(
	value declaration.Entry,
) (DeclarationSource, error) {
	raw, err := value.CanonicalJSON()
	if err != nil {
		return DeclarationSource{}, err
	}
	return newCanonicalDeclarationSource(raw)
}

func NewScanDeclarationSource(
	value DeclarationScan,
) (DeclarationSource, error) {
	raw, err := jsonutil.MarshalCanonicalObject(
		value,
		basespec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return DeclarationSource{}, err
	}
	return newCanonicalDeclarationSource(raw)
}

func newCanonicalDeclarationSource(
	raw []byte,
) (DeclarationSource, error) {
	value := DeclarationSource{
		raw: append(json.RawMessage(nil), raw...),
	}
	if err := value.Validate(); err != nil {
		return DeclarationSource{}, err
	}
	return value, nil
}

func (s DeclarationSource) Clone() DeclarationSource {
	return DeclarationSource{
		raw: append(json.RawMessage(nil), s.raw...),
	}
}

func (s DeclarationSource) CanonicalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return append([]byte(nil), s.raw...), nil
}

func (s DeclarationSource) MarshalJSON() ([]byte, error) {
	return s.CanonicalJSON()
}

func (s *DeclarationSource) UnmarshalJSON(raw []byte) error {
	if s == nil {
		return fmt.Errorf(
			"%w: declaration source target is nil",
			basespec.ErrInvalid,
		)
	}
	canonical, err := jsonutil.Canonicalize(raw)
	if err != nil {
		return err
	}
	value, err := newCanonicalDeclarationSource(canonical)
	if err != nil {
		return err
	}

	*s = value
	return nil
}

func (s DeclarationSource) Validate() error {
	trimmed := bytes.TrimSpace(s.raw)
	if len(trimmed) == 0 {
		return fmt.Errorf(
			"%w: declaration source is empty",
			basespec.ErrInvalid,
		)
	}
	if trimmed[0] == '"' {
		var locator declaration.Locator
		if err := json.Unmarshal(trimmed, &locator); err != nil {
			return err
		}
		return locator.Validate()
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &fields); err != nil {
		return fmt.Errorf(
			"%w: declaration source must be a Locator, declaration, or scan",
			basespec.ErrInvalid,
		)
	}
	if _, found := fields["type"]; found {
		entry, err := declaration.DecodeCanonicalEntryJSON(trimmed)
		if err != nil {
			return err
		}
		return entry.Validate()
	}
	if _, found := fields["kind"]; found {
		var locator declaration.Locator
		if err := json.Unmarshal(trimmed, &locator); err != nil {
			return err
		}
		return locator.Validate()
	}

	var scan DeclarationScan
	if err := jsonutil.DecodeCanonicalObjectBytesInto(
		trimmed,
		&scan,
		basespec.MaxDefinitionBodyBytes,
	); err != nil {
		return err
	}
	if err := scan.Base.Validate(); err != nil {
		return fmt.Errorf("declaration scan base: %w", err)
	}
	if len(scan.Include) == 0 {
		return fmt.Errorf(
			"%w: declaration scan requires include patterns",
			basespec.ErrInvalid,
		)
	}
	if err := basespec.ValidatePathPatterns(
		"declaration scan include",
		scan.Include,
	); err != nil {
		return err
	}
	return basespec.ValidatePathPatterns(
		"declaration scan exclude",
		scan.Exclude,
	)
}

func DecodeWorkspaceJSON(raw []byte) (WorkspaceDocument, error) {
	return decodeWorkspace(raw)
}

func DecodeWorkspaceEntry(
	entry declaration.Entry,
) (WorkspaceDocument, error) {
	var value WorkspaceDocument
	if err := declaration.DecodeEntryDocumentInto(
		entry,
		compiledWorkspaceSchema,
		&value,
	); err != nil {
		return WorkspaceDocument{}, err
	}
	if err := value.validateFields(); err != nil {
		return WorkspaceDocument{}, err
	}
	return value, nil
}

func decodeWorkspace(
	raw []byte,
) (WorkspaceDocument, error) {
	var value WorkspaceDocument
	if err := declaration.DecodeDocumentInto(
		raw,
		compiledWorkspaceSchema,
		&value,
	); err != nil {
		return WorkspaceDocument{}, err
	}
	if err := value.validateFields(); err != nil {
		return WorkspaceDocument{}, err
	}
	return value, nil
}

func (v WorkspaceDocument) Clone() (
	WorkspaceDocument,
	error,
) {
	return jsonutil.CloneJSON(v)
}

func (v WorkspaceDocument) Canonicalize() (
	WorkspaceDocument,
	error,
) {
	return v.Clone()
}

func (v WorkspaceDocument) CanonicalJSON() ([]byte, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	return declaration.CanonicalDocumentJSON(v)
}

func (v WorkspaceDocument) CalculatedDigest() (
	cryptoutil.Digest,
	error,
) {
	return declaration.DocumentDigest(v)
}

func (v WorkspaceDocument) Validate() error {
	return v.validate()
}

func (v WorkspaceDocument) ValidateEntry() error {
	return v.validate()
}

func (v WorkspaceDocument) validate() error {
	if err := declaration.ValidateDocument(
		compiledWorkspaceSchema,
		v,
	); err != nil {
		return fmt.Errorf("workspace schema: %w", err)
	}
	return v.validateFields()
}

func (v WorkspaceDocument) validateFields() error {
	if err := v.Header.Validate(declaration.HeaderValidation{
		ExpectedType: WorkspaceType,
		APIVersion:   WorkspaceSchemaVersion,
	}); err != nil {
		return err
	}
	for index, declaration := range v.Declarations {
		if err := declaration.Validate(); err != nil {
			return fmt.Errorf(
				"workspace declarations[%d]: %w",
				index,
				err,
			)
		}
	}
	for index, rootEntry := range v.Roots {
		if err := rootEntry.Validate(); err != nil {
			return fmt.Errorf(
				"workspace roots[%d]: %w",
				index,
				err,
			)
		}
	}
	return nil
}
