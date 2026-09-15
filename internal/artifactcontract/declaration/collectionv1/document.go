package collectionv1

import (
	_ "embed"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	CollectionType          = declaration.TypeCollection
	CollectionSchemaID      = "artifact.collection.v1"
	CollectionSchemaVersion = declaration.APIVersionV1
)

//go:embed collection-v1.schema.json
var schemaJSON []byte

var compiledCollectionSchema = jsonutil.MustCompileJSONSchema(schemaJSON)

var CollectionSchemaKey = schema.ArtifactKey(
	artifact.ArtifactKind(CollectionType),
	schema.SchemaID(CollectionSchemaID),
	CollectionSchemaVersion,
)

type CollectionDocument struct {
	declaration.Header

	Version string              `json:"version,omitempty"`
	Members []declaration.Entry `json:"members,omitempty"`
}

func CollectionJSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
}

func DecodeCollectionJSON(raw []byte) (CollectionDocument, error) {
	return decodeCollection(raw)
}

func DecodeCollectionEntry(
	entry declaration.Entry,
) (CollectionDocument, error) {
	var value CollectionDocument
	if err := declaration.DecodeEntryDocumentInto(
		entry,
		compiledCollectionSchema,
		&value,
	); err != nil {
		return CollectionDocument{}, err
	}
	if err := value.validateFields(); err != nil {
		return CollectionDocument{}, err
	}
	return value, nil
}

func decodeCollection(
	raw []byte,
) (CollectionDocument, error) {
	var value CollectionDocument
	if err := declaration.DecodeDocumentInto(
		raw,
		compiledCollectionSchema,
		&value,
	); err != nil {
		return CollectionDocument{}, err
	}
	if err := value.validateFields(); err != nil {
		return CollectionDocument{}, err
	}
	return value, nil
}

func (v CollectionDocument) Clone() (
	CollectionDocument,
	error,
) {
	return jsonutil.CloneJSON(v)
}

func (v CollectionDocument) Canonicalize() (
	CollectionDocument,
	error,
) {
	return v.Clone()
}

func (v CollectionDocument) CanonicalJSON() ([]byte, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	return declaration.CanonicalDocumentJSON(v)
}

func (v CollectionDocument) CalculatedDigest() (
	cryptoutil.Digest,
	error,
) {
	return declaration.DocumentDigest(v)
}

func (v CollectionDocument) Validate() error {
	return v.validate()
}

func (v CollectionDocument) ValidateEntry() error {
	return v.validate()
}

func (v CollectionDocument) validate() error {
	if err := declaration.ValidateDocument(
		compiledCollectionSchema,
		v,
	); err != nil {
		return fmt.Errorf("collection schema: %w", err)
	}
	return v.validateFields()
}

func (v CollectionDocument) validateFields() error {
	if err := v.Header.Validate(declaration.HeaderValidation{
		ExpectedType: CollectionType,
		APIVersion:   CollectionSchemaVersion,
	}); err != nil {
		return err
	}
	if err := declaration.ValidateDeclarationLocatorExclusivity(
		"Collection",
		v.Locator,
		v.Version != "" || v.Members != nil,
	); err != nil {
		return err
	}
	if v.Version != "" {
		if err := basespec.LogicalVersion(v.Version).Validate(false); err != nil {
			return err
		}
	}
	for index, member := range v.Members {
		if err := member.Validate(); err != nil {
			return fmt.Errorf("collection members[%d]: %w", index, err)
		}
	}
	return nil
}
