package collectionv1

import (
	_ "embed"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	CollectionType          = artifactcontract.TypeCollection
	CollectionSchemaID      = "artifact.collection.v1"
	CollectionSchemaVersion = artifactcontract.APIVersionV1
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
	artifactcontract.Header

	Version string                   `json:"version,omitempty"`
	Members []artifactcontract.Entry `json:"members,omitempty"`
}

func CollectionJSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
}

func DecodeCollectionJSON(raw []byte) (CollectionDocument, error) {
	return decodeCollection(raw, true)
}

func DecodeCollectionEntry(
	entry artifactcontract.Entry,
) (CollectionDocument, error) {
	raw, err := entry.CanonicalJSON()
	if err != nil {
		return CollectionDocument{}, err
	}
	return decodeCollection(raw, false)
}

func decodeCollection(
	raw []byte,
	requireName bool,
) (CollectionDocument, error) {
	var value CollectionDocument
	if err := artifactcontract.DecodeDocumentInto(
		raw,
		compiledCollectionSchema,
		&value,
	); err != nil {
		return CollectionDocument{}, err
	}
	if err := value.validate(requireName); err != nil {
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
	return artifactcontract.CanonicalDocumentJSON(v)
}

func (v CollectionDocument) CalculatedDigest() (
	cryptoutil.Digest,
	error,
) {
	return artifactcontract.DocumentDigest(v)
}

func (v CollectionDocument) Validate() error {
	return v.validate(true)
}

func (v CollectionDocument) ValidateEntry() error {
	return v.validate(false)
}

func (v CollectionDocument) validate(requireName bool) error {
	if err := artifactcontract.ValidateDocument(
		compiledCollectionSchema,
		v,
	); err != nil {
		return fmt.Errorf("collection schema: %w", err)
	}
	if err := v.Header.Validate(artifactcontract.HeaderValidation{
		ExpectedType: CollectionType,
		APIVersion:   CollectionSchemaVersion,
		RequireName:  requireName,
	}); err != nil {
		return err
	}
	if v.Version != "" {
		if err := basespec.LogicalVersion(v.Version).Validate(false); err != nil {
			return err
		}
	}
	if len(v.Members) > basespec.MaxDefinitionDependencies {
		return fmt.Errorf(
			"%w: collection members exceed %d entries",
			basespec.ErrInvalid,
			basespec.MaxDefinitionDependencies,
		)
	}
	for index, member := range v.Members {
		if err := member.Validate(); err != nil {
			return fmt.Errorf("collection members[%d]: %w", index, err)
		}
	}
	return nil
}
