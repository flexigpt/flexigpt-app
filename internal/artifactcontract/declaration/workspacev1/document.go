package workspacev1

import (
	_ "embed"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	WorkspaceType          = declaration.TypeWorkspace
	WorkspaceSchemaID      = "artifact.workspace.v1"
	WorkspaceSchemaVersion = declaration.SchemaVersionV1
)

//go:embed workspace-v1.schema.json
var schemaJSON []byte

var compiledWorkspaceSchema = jsonutil.MustCompileJSONSchema(schemaJSON)

var WorkspaceSchemaKey = schema.ArtifactKey(
	artifact.ArtifactKind(WorkspaceType),
	schema.SchemaID(WorkspaceSchemaID),
	WorkspaceSchemaVersion,
)

type WorkspaceDocument struct {
	declaration.Header

	Members []declaration.Entry `json:"members,omitempty"`
}

func WorkspaceJSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
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
		RequireName:  true,
	}); err != nil {
		return err
	}
	if err := declaration.ValidateDeclarationLocatorExclusivity(
		"Workspace",
		v.Locator,
		v.Members != nil,
	); err != nil {
		return err
	}
	return declaration.ValidateMemberUniqueness(
		"Workspace members",
		v.Members,
	)
}
