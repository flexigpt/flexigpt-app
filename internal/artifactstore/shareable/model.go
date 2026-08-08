package shareable

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type EntityType string

const EntityCollection EntityType = "collection"

type SchemaKey struct {
	Entity        EntityType              `json:"entity"`
	Kind          basespec.CollectionKind `json:"kind"`
	SchemaID      basespec.SchemaID       `json:"schemaID"`
	SchemaVersion string                  `json:"schemaVersion"`
}

func (k SchemaKey) Validate() error {
	if k.Entity != EntityCollection {
		return fmt.Errorf(
			"%w: unsupported shareable entity type %q",
			basespec.ErrInvalid,
			k.Entity,
		)
	}
	if err := basespec.ValidateCollectionKind(k.Kind); err != nil {
		return err
	}
	if err := basespec.ValidateSchemaID(k.SchemaID); err != nil {
		return err
	}
	return basespec.ValidateRequiredText(
		"shareable schema version",
		k.SchemaVersion,
		basespec.MaxVersionBytes,
	)
}

type ParsedDocument struct {
	Key    SchemaKey
	Digest cryptoutil.Digest
	Raw    json.RawMessage
}

func (d ParsedDocument) Validate() error {
	if err := d.Key.Validate(); err != nil {
		return err
	}
	if err := cryptoutil.ValidateDigest(d.Digest); err != nil {
		return err
	}
	if len(d.Raw) == 0 {
		return fmt.Errorf("%w: shareable document is empty", basespec.ErrInvalid)
	}
	return nil
}

func (d ParsedDocument) Clone() ParsedDocument {
	output := d
	output.Raw = append(json.RawMessage(nil), d.Raw...)
	return output
}

type CollectionDocumentBinding struct {
	Collection collection.CollectionRef `json:"collection"`
	Key        SchemaKey                `json:"schema"`
	Digest     cryptoutil.Digest        `json:"digest"`
}

func (b CollectionDocumentBinding) Validate() error {
	if err := b.Collection.Validate(); err != nil {
		return err
	}
	if err := b.Key.Validate(); err != nil {
		return err
	}
	return cryptoutil.ValidateDigest(b.Digest)
}

type CollectionDocument struct {
	Binding CollectionDocumentBinding `json:"binding"`
	Raw     json.RawMessage           `json:"raw"`
}

func (d CollectionDocument) Clone() CollectionDocument {
	output := d
	output.Raw = append(json.RawMessage(nil), d.Raw...)
	return output
}

type Codec interface {
	Key() SchemaKey
	JSONSchema() []byte

	// Canonicalize must strictly decode, semantically validate, calculate or
	// verify the portable digest, and return canonical JSON containing the
	// resulting digest.
	Canonicalize(
		ctx context.Context,
		raw []byte,
	) (ParsedDocument, error)
}

type DocumentRepository interface {
	Put(
		ctx context.Context,
		rootID basespec.RootID,
		digest cryptoutil.Digest,
		raw json.RawMessage,
	) error

	Get(
		ctx context.Context,
		rootID basespec.RootID,
		digest cryptoutil.Digest,
	) (json.RawMessage, error)
}

type CollectionBindingRepository interface {
	PutCollectionDocument(
		ctx context.Context,
		value CollectionDocumentBinding,
	) error

	GetCollectionDocument(
		ctx context.Context,
		ref collection.CollectionRef,
	) (CollectionDocumentBinding, error)
}
