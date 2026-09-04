package shareable

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type registeredCodec struct {
	codec  providerapi.SchemaCodec
	schema *jsonschema.Schema
}

type Registry struct {
	codecs map[providerapi.SchemaKey]registeredCodec
	keys   []providerapi.SchemaKey
}

func NewRegistry(codecs ...providerapi.SchemaCodec) (*Registry, error) {
	values := make(map[providerapi.SchemaKey]registeredCodec, len(codecs))
	keys := make([]providerapi.SchemaKey, 0, len(codecs))

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
		if keys[left].Entity != keys[right].Entity {
			return keys[left].Entity < keys[right].Entity
		}
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

func (r *Registry) Keys() []providerapi.SchemaKey {
	if r == nil {
		return nil
	}
	return append([]providerapi.SchemaKey(nil), r.keys...)
}

func (r *Registry) Canonicalize(
	ctx context.Context,
	raw []byte,
) (providerapi.ParsedDocument, error) {
	return r.CanonicalizeEntity(ctx, providerapi.EntityCollection, raw)
}

// CanonicalizeExpected canonicalizes raw content through the Artifact Store
// registry and requires the resulting schema key to match expected.
//
// Feature services use this boundary for known document inputs. The registry
// remains the only owner of JSON Schema execution, codec selection, canonical
// JSON enforcement, and canonical output validation.
func (r *Registry) CanonicalizeExpected(
	ctx context.Context,
	expected providerapi.SchemaKey,
	raw []byte,
) (providerapi.ParsedDocument, error) {
	if err := expected.Validate(); err != nil {
		return providerapi.ParsedDocument{}, err
	}

	value, err := r.CanonicalizeEntity(ctx, expected.Entity, raw)
	if err != nil {
		return providerapi.ParsedDocument{}, err
	}
	if value.Key != expected {
		return providerapi.ParsedDocument{}, fmt.Errorf(
			"%w: expected shareable schema %q/%q/%q, got %q/%q/%q",
			basespec.ErrInvalid,
			expected.Kind,
			expected.SchemaID,
			expected.SchemaVersion,
			value.Key.Kind,
			value.Key.SchemaID,
			value.Key.SchemaVersion,
		)
	}
	return value.Clone(), nil
}

func (r *Registry) CanonicalizeEntity(
	ctx context.Context,
	entity providerapi.EntityType,
	raw []byte,
) (providerapi.ParsedDocument, error) {
	if r == nil {
		return providerapi.ParsedDocument{}, basespec.ErrClosed
	}
	if entity != providerapi.EntityCollection && entity != providerapi.EntityArtifact {
		return providerapi.ParsedDocument{}, fmt.Errorf(
			"%w: unsupported shareable entity %q",
			basespec.ErrInvalid,
			entity,
		)
	}
	if ctx == nil {
		return providerapi.ParsedDocument{}, fmt.Errorf(
			"%w: shareable document context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return providerapi.ParsedDocument{}, err
	}

	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxDefinitionBytes,
	)
	if err != nil {
		return providerapi.ParsedDocument{}, err
	}

	var header struct {
		Kind          string            `json:"kind"`
		SchemaID      basespec.SchemaID `json:"schemaID"`
		SchemaVersion string            `json:"schemaVersion"`
	}
	if err := json.Unmarshal(canonical, &header); err != nil {
		return providerapi.ParsedDocument{}, fmt.Errorf(
			"%w: decode shareable document header: %w",
			basespec.ErrInvalid,
			err,
		)
	}

	key := providerapi.SchemaKey{
		Entity:        entity,
		Kind:          providerapi.SchemaKind(header.Kind),
		SchemaID:      header.SchemaID,
		SchemaVersion: header.SchemaVersion,
	}
	if err := key.Validate(); err != nil {
		return providerapi.ParsedDocument{}, err
	}
	registered, found := r.codecs[key]
	if !found {
		return providerapi.ParsedDocument{}, fmt.Errorf(
			"%w: shareable %s schema %q/%q/%q",
			basespec.ErrUnsupported,
			entity,
			key.Kind,
			key.SchemaID,
			key.SchemaVersion,
		)
	}

	if err := validateJSONSchemaInstance(registered.schema, canonical); err != nil {
		return providerapi.ParsedDocument{}, err
	}

	value, err := registered.codec.Canonicalize(ctx, canonical)
	if err != nil {
		return providerapi.ParsedDocument{}, err
	}
	if value.Key != key {
		return providerapi.ParsedDocument{}, fmt.Errorf(
			"%w: shareable codec returned another schema key",
			basespec.ErrInvalid,
		)
	}
	if err := validateJSONSchemaInstance(registered.schema, value.Raw); err != nil {
		return providerapi.ParsedDocument{}, err
	}
	if err := validateCodecOutput(key, value); err != nil {
		return providerapi.ParsedDocument{}, err
	}
	return value.Clone(), nil
}

func validateCodecOutput(
	expected providerapi.SchemaKey,
	value providerapi.ParsedDocument,
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
		Kind          string            `json:"kind"`
		SchemaID      basespec.SchemaID `json:"schemaID"`
		SchemaVersion string            `json:"schemaVersion"`
		Digest        string            `json:"digest"`
	}
	if err := json.Unmarshal(canonical, &header); err != nil {
		return fmt.Errorf(
			"%w: decode canonical shareable document header: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	actual := providerapi.SchemaKey{
		Entity:        expected.Entity,
		Kind:          providerapi.SchemaKind(header.Kind),
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
