package shareable

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type registeredCodec struct {
	codec  Codec
	schema *jsonschema.Schema
}

type Registry struct {
	codecs map[SchemaKey]registeredCodec
	keys   []SchemaKey
}

func NewRegistry(codecs ...Codec) (*Registry, error) {
	values := make(map[SchemaKey]registeredCodec, len(codecs))
	keys := make([]SchemaKey, 0, len(codecs))

	for _, codec := range codecs {
		if codec == nil {
			return nil, fmt.Errorf("%w: shareable schema codec is nil", basespec.ErrInvalid)
		}
		key := codec.Key()
		if err := key.Validate(); err != nil {
			return nil, err
		}
		compiled, err := compilePublishedJSONSchema(codec.JSONSchema())
		if err != nil {
			return nil, err
		}
		if _, exists := values[key]; exists {
			return nil, fmt.Errorf(
				"%w: duplicate shareable schema %q/%q/%q",
				basespec.ErrConflict,
				key.Kind,
				key.SchemaID,
				key.SchemaVersion,
			)
		}
		values[key] = registeredCodec{
			codec:  codec,
			schema: compiled,
		}
		keys = append(keys, key)
	}

	sort.Slice(keys, func(left, right int) bool {
		if keys[left].Kind != keys[right].Kind {
			return keys[left].Kind < keys[right].Kind
		}
		if keys[left].SchemaID != keys[right].SchemaID {
			return keys[left].SchemaID < keys[right].SchemaID
		}
		return keys[left].SchemaVersion < keys[right].SchemaVersion
	})
	return &Registry{codecs: values, keys: keys}, nil
}

func (r *Registry) Keys() []SchemaKey {
	if r == nil {
		return nil
	}
	return append([]SchemaKey(nil), r.keys...)
}

func (r *Registry) Canonicalize(
	ctx context.Context,
	raw []byte,
) (ParsedDocument, error) {
	if r == nil {
		return ParsedDocument{}, basespec.ErrClosed
	}
	if ctx == nil {
		return ParsedDocument{}, fmt.Errorf(
			"%w: shareable document context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return ParsedDocument{}, err
	}

	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxDefinitionBytes,
	)
	if err != nil {
		return ParsedDocument{}, err
	}

	var header struct {
		Kind          basespec.CollectionKind `json:"kind"`
		SchemaID      basespec.SchemaID       `json:"schemaID"`
		SchemaVersion string                  `json:"schemaVersion"`
	}
	if err := json.Unmarshal(canonical, &header); err != nil {
		return ParsedDocument{}, fmt.Errorf(
			"%w: decode shareable document header: %w",
			basespec.ErrInvalid,
			err,
		)
	}

	key := SchemaKey{
		Entity:        EntityCollection,
		Kind:          header.Kind,
		SchemaID:      header.SchemaID,
		SchemaVersion: header.SchemaVersion,
	}
	if err := key.Validate(); err != nil {
		return ParsedDocument{}, err
	}
	registered, found := r.codecs[key]
	if !found {
		return ParsedDocument{}, fmt.Errorf(
			"%w: shareable collection schema %q/%q/%q",
			basespec.ErrUnsupported,
			key.Kind,
			key.SchemaID,
			key.SchemaVersion,
		)
	}

	if err := validateJSONSchemaInstance(registered.schema, canonical); err != nil {
		return ParsedDocument{}, err
	}

	value, err := registered.codec.Canonicalize(ctx, canonical)
	if err != nil {
		return ParsedDocument{}, err
	}
	if value.Key != key {
		return ParsedDocument{}, fmt.Errorf(
			"%w: shareable codec returned another schema key",
			basespec.ErrInvalid,
		)
	}
	if err := validateJSONSchemaInstance(registered.schema, value.Raw); err != nil {
		return ParsedDocument{}, err
	}
	if err := validateCodecOutput(key, value); err != nil {
		return ParsedDocument{}, err
	}
	return value.Clone(), nil
}

func validateCodecOutput(
	expected SchemaKey,
	value ParsedDocument,
) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if value.Key != expected {
		return fmt.Errorf(
			"%w: shareable codec returned another schema key",
			basespec.ErrInvalid,
		)
	}

	canonical, err := jsonutil.CanonicalizeObject(
		value.Raw,
		basespec.MaxDefinitionBytes,
	)
	if err != nil {
		return err
	}
	if !bytes.Equal(canonical, value.Raw) {
		return fmt.Errorf(
			"%w: shareable codec returned non-canonical JSON",
			basespec.ErrInvalid,
		)
	}

	var header struct {
		Kind          basespec.CollectionKind `json:"kind"`
		SchemaID      basespec.SchemaID       `json:"schemaID"`
		SchemaVersion string                  `json:"schemaVersion"`
		Digest        string                  `json:"digest"`
	}
	if err := json.Unmarshal(canonical, &header); err != nil {
		return fmt.Errorf(
			"%w: decode canonical shareable document header: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	actual := SchemaKey{
		Entity:        EntityCollection,
		Kind:          header.Kind,
		SchemaID:      header.SchemaID,
		SchemaVersion: header.SchemaVersion,
	}
	if actual != expected || header.Digest != string(value.Digest) {
		return fmt.Errorf(
			"%w: shareable codec output does not match its metadata",
			basespec.ErrDigestMismatch,
		)
	}
	return nil
}

func compilePublishedJSONSchema(raw []byte) (*jsonschema.Schema, error) {
	var value struct {
		Schema string `json:"$schema"`
		ID     string `json:"$id"`
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf(
			"%w: published JSON Schema is invalid JSON: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if value.Schema == "" || value.ID == "" {
		return nil, fmt.Errorf(
			"%w: published JSON Schema requires $schema and $id",
			basespec.ErrInvalid,
		)
	}

	compiler := jsonschema.NewCompiler()
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	if err := compiler.AddResource(value.ID, inst); err != nil {
		return nil, fmt.Errorf(
			"%w: register published JSON Schema %q: %w",
			basespec.ErrInvalid,
			value.ID,
			err,
		)
	}
	compiled, err := compiler.Compile(value.ID)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: compile published JSON Schema %q: %w",
			basespec.ErrInvalid,
			value.ID,
			err,
		)
	}
	return compiled, nil
}

func validateJSONSchemaInstance(
	schema *jsonschema.Schema,
	raw []byte,
) error {
	// Using jsonschema.UnmarshalJSON ensures proper float/integer typing alignment for v6.
	value, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf(
			"%w: decode shareable JSON for schema validation: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if err := schema.Validate(value); err != nil {
		return fmt.Errorf(
			"%w: shareable document does not satisfy its JSON Schema: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	return nil
}
