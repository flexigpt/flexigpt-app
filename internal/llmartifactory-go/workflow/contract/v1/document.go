// Package workflowv1 owns the Workflow v1 declaration language.
//
// This package validates declarations. It does not evaluate expressions,
// invoke targets, schedule work, persist invocations, or execute workflows.
package workflowv1

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"maps"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

const (
	WorkflowType          declaration.Type = declaration.TypeWorkflow
	WorkflowSchemaID                       = "artifact.workflow.v1"
	WorkflowSchemaVersion                  = declaration.SchemaVersionV1

	WorkflowSourceVersionRequired = true
)

//go:embed workflow-v1.schema.json
var schemaJSON []byte

var (
	definitionSchemaJSON   = makeDefinitionSchema()
	compiledWorkflowSchema = jsonutil.MustCompileJSONSchema(definitionSchemaJSON)

	WorkflowSchemaKey = schemaModel.ArtifactKey(
		artifactModel.ArtifactKind(WorkflowType),
		schemaModel.SchemaID(WorkflowSchemaID),
		WorkflowSchemaVersion,
	)
)

// Value is a portable value-construction expression or JSON literal.
//
// A single-key {"$expr":"..."} object is CEL. A single-key {"$literal":...}
// object suppresses directive interpretation for its entire value.
type Value json.RawMessage

func (v Value) MarshalJSON() ([]byte, error) {
	return marshalRaw(v)
}

func (v *Value) UnmarshalJSON(raw []byte) error {
	if v == nil {
		return fmt.Errorf("%w: nil Workflow value target", storeSpec.ErrInvalid)
	}
	canonical, err := boundedJSON(raw)
	if err != nil {
		return err
	}
	*v = Value(canonical)
	return nil
}

// Schema is a JSON Schema Draft 2020-12 object or boolean.
type Schema json.RawMessage

func (s Schema) MarshalJSON() ([]byte, error) {
	return marshalRaw(s)
}

func (s *Schema) UnmarshalJSON(raw []byte) error {
	if s == nil {
		return fmt.Errorf("%w: nil Workflow schema target", storeSpec.ErrInvalid)
	}
	canonical, err := boundedJSON(raw)
	if err != nil {
		return err
	}
	*s = Schema(canonical)
	return nil
}

func marshalRaw(raw []byte) ([]byte, error) {
	if len(raw) == 0 {
		return []byte("null"), nil
	}
	return boundedJSON(raw)
}

func boundedJSON(raw []byte) ([]byte, error) {
	if len(raw) > storeSpec.MaxDefinitionBodyBytes {
		return nil, fmt.Errorf("%w: Workflow JSON value exceeds its size limit", storeSpec.ErrInvalid)
	}
	return jsonutil.Canonicalize(raw)
}

// WorkflowDocument is the canonical Definition representation.
//
// "apiVersion" is mandatory in standalone source documents. It is dispatch
// metadata and is not stored in Definition.Body. Contained declarations inherit
// the enclosing v1 admission context and use this canonical representation.
type WorkflowDocument struct {
	declaration.Header
	Block

	InputSchema  Schema `json:"inputSchema"`
	OutputSchema Schema `json:"outputSchema"`

	Schemas      map[string]Schema        `json:"schemas,omitempty"`
	Targets      map[string]Target        `json:"targets,omitempty"`
	TargetSets   map[string]TargetSet     `json:"targetSets,omitempty"`
	Events       map[string]EventContract `json:"events,omitempty"`
	Channels     map[string]Channel       `json:"channels,omitempty"`
	Ports        map[string]PortBinding   `json:"ports,omitempty"`
	Triggers     map[string]Trigger       `json:"triggers,omitempty"`
	Requirements map[string]Target        `json:"requirements,omitempty"`
	Resources    map[string]Resource      `json:"resources,omitempty"`
	Limits       *Limits                  `json:"limits,omitempty"`
}

// WorkflowJSONSchema returns the complete standalone source schema.
// Standalone source requires apiVersion: v1.
func WorkflowJSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
}

// WorkflowDefinitionJSONSchema returns the same grammar for canonical
// Definition bodies, after dispatch-only apiVersion removal.
func WorkflowDefinitionJSONSchema() []byte {
	return append([]byte(nil), definitionSchemaJSON...)
}

func makeDefinitionSchema() []byte {
	var value map[string]json.RawMessage
	if err := json.Unmarshal(schemaJSON, &value); err != nil {
		panic(err)
	}
	delete(value, "properties")
	delete(value, "required")
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return raw
}

func DecodeWorkflowJSON(raw []byte) (WorkflowDocument, error) {
	var value WorkflowDocument
	if err := declaration.DecodeSourceDocumentInto(
		raw,
		WorkflowSchemaKey,
		WorkflowSourceVersionRequired,
		compiledWorkflowSchema,
		&value,
	); err != nil {
		return WorkflowDocument{}, err
	}
	if err := value.validateFields(); err != nil {
		return WorkflowDocument{}, err
	}
	return value, nil
}

func DecodeWorkflowEntry(entry declaration.Entry) (WorkflowDocument, error) {
	var value WorkflowDocument
	if err := declaration.DecodeEntryDocumentInto(entry, compiledWorkflowSchema, &value); err != nil {
		return WorkflowDocument{}, err
	}
	if err := value.validateFields(); err != nil {
		return WorkflowDocument{}, err
	}
	return value, nil
}

func DecodeAdmittedWorkflowJSON(raw []byte) (WorkflowDocument, error) {
	entry, err := declaration.DecodeCanonicalEntryJSON(raw)
	if err != nil {
		return WorkflowDocument{}, err
	}
	return DecodeAdmittedWorkflowEntry(entry)
}

func DecodeAdmittedWorkflowEntry(entry declaration.Entry) (WorkflowDocument, error) {
	var value WorkflowDocument
	if err := declaration.DecodeAdmittedEntryInto(entry, &value); err != nil {
		return WorkflowDocument{}, err
	}
	if err := value.validateFields(); err != nil {
		return WorkflowDocument{}, err
	}
	return value, nil
}

func (v WorkflowDocument) Validate() error {
	if err := declaration.ValidateDocument(compiledWorkflowSchema, v); err != nil {
		return fmt.Errorf("workflow schema: %w", err)
	}
	return v.validateFields()
}

func (v WorkflowDocument) ValidateEntry() error {
	return v.Validate()
}

func (v WorkflowDocument) Clone() (WorkflowDocument, error) {
	return jsonutil.CloneJSON(v)
}

func (v WorkflowDocument) Canonicalize() (WorkflowDocument, error) {
	return v.Clone()
}

func (v WorkflowDocument) CanonicalJSON() ([]byte, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	return declaration.CanonicalDocumentJSON(v)
}

// SourceJSON returns a standalone declaration, including mandatory apiVersion.
func (v WorkflowDocument) SourceJSON() ([]byte, error) {
	raw, err := v.CanonicalJSON()
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	version, err := json.Marshal(WorkflowSchemaVersion)
	if err != nil {
		return nil, err
	}
	fields["apiVersion"] = version
	return jsonutil.MarshalCanonicalObject(fields, storeSpec.MaxDefinitionBytes)
}

func (v WorkflowDocument) CalculatedDigest() (cryptoutil.Digest, error) {
	raw, err := v.CanonicalJSON()
	if err != nil {
		return "", err
	}
	return cryptoutil.DigestBytes(raw), nil
}

func DefinitionForDocument(document WorkflowDocument) (definitionModel.Definition, error) {
	body, err := document.CanonicalJSON()
	if err != nil {
		return definitionModel.Definition{}, err
	}
	displayName := document.DisplayName
	if displayName == "" {
		displayName = document.Name
	}
	return definitionModel.Canonicalize(definitionModel.Definition{
		Kind:          artifactModel.ArtifactKind(WorkflowType),
		SchemaID:      WorkflowSchemaKey.SchemaID,
		SchemaVersion: WorkflowSchemaKey.SchemaVersion,
		LogicalName:   storeSpec.LogicalName(document.Name),
		DisplayName:   displayName,
		Description:   document.Description,
		Labels:        declaration.CloneStringMap(document.Labels),
		Body:          body,
	})
}

// FromDefinition reconstructs admitted meaning without executing the family
// admission schema or recalculating the immutable Definition digest.
func FromDefinition(value definitionModel.Definition) (WorkflowDocument, error) {
	if err := definitionModel.ValidateAdmitted(value); err != nil {
		return WorkflowDocument{}, err
	}
	if value.Kind != artifactModel.ArtifactKind(WorkflowType) ||
		value.SchemaID != WorkflowSchemaKey.SchemaID ||
		value.SchemaVersion != WorkflowSchemaKey.SchemaVersion ||
		value.LogicalVersion != "" {
		return WorkflowDocument{}, fmt.Errorf("%w: unsupported Workflow Definition", storeSpec.ErrUnsupported)
	}
	document, err := DecodeAdmittedWorkflowJSON(value.Body)
	if err != nil {
		return WorkflowDocument{}, err
	}
	displayName := document.DisplayName
	if displayName == "" {
		displayName = document.Name
	}
	if document.Name != string(value.LogicalName) ||
		document.Description != value.Description ||
		displayName != value.DisplayName ||
		!maps.Equal(document.Labels, value.Labels) {
		return WorkflowDocument{}, fmt.Errorf(
			"%w: Workflow body differs from Definition metadata",
			storeSpec.ErrDigestMismatch,
		)
	}
	return document, nil
}
