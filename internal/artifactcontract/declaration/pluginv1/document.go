package pluginv1

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
	PluginType          = declaration.TypePlugin
	PluginSchemaID      = "artifact.plugin.v1"
	PluginSchemaVersion = declaration.SchemaVersionV1
)

//go:embed plugin-v1.schema.json
var schemaJSON []byte

var compiledPluginSchema = jsonutil.MustCompileJSONSchema(schemaJSON)

var PluginSchemaKey = schema.ArtifactKey(
	artifact.ArtifactKind(PluginType),
	schema.SchemaID(PluginSchemaID),
	PluginSchemaVersion,
)

type PluginDocument struct {
	declaration.Header

	Members []declaration.Entry `json:"members,omitempty"`
}

func PluginJSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
}

func DecodePluginJSON(raw []byte) (PluginDocument, error) {
	var value PluginDocument
	if err := declaration.DecodeDocumentInto(
		raw,
		compiledPluginSchema,
		&value,
	); err != nil {
		return PluginDocument{}, err
	}
	if err := value.validateFields(); err != nil {
		return PluginDocument{}, err
	}
	return value, nil
}

func DecodePluginEntry(entry declaration.Entry) (PluginDocument, error) {
	var value PluginDocument
	if err := declaration.DecodeEntryDocumentInto(
		entry,
		compiledPluginSchema,
		&value,
	); err != nil {
		return PluginDocument{}, err
	}
	if err := value.validateFields(); err != nil {
		return PluginDocument{}, err
	}
	return value, nil
}

func (v PluginDocument) Clone() (PluginDocument, error) {
	return jsonutil.CloneJSON(v)
}

func (v PluginDocument) Canonicalize() (PluginDocument, error) {
	return v.Clone()
}

func (v PluginDocument) CanonicalJSON() ([]byte, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	return declaration.CanonicalDocumentJSON(v)
}

func (v PluginDocument) CalculatedDigest() (cryptoutil.Digest, error) {
	return declaration.DocumentDigest(v)
}

func (v PluginDocument) Validate() error {
	if err := declaration.ValidateDocument(
		compiledPluginSchema,
		v,
	); err != nil {
		return fmt.Errorf("plugin schema: %w", err)
	}
	return v.validateFields()
}

func (v PluginDocument) ValidateEntry() error {
	return v.Validate()
}

func (v PluginDocument) validateFields() error {
	if err := v.Header.Validate(declaration.HeaderValidation{
		ExpectedType: PluginType,
		RequireName:  true,
	}); err != nil {
		return err
	}
	if err := declaration.ValidateDeclarationLocatorExclusivity(
		"Plugin",
		v.Locator,
		v.Members != nil,
	); err != nil {
		return err
	}
	return declaration.ValidateMemberUniqueness(
		"Plugin members",
		v.Members,
	)
}
