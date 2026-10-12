package jsonschema

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema"
	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type registeredCodec struct {
	codec  schema.Codec
	schema jsonutil.JSONSchema
}

// Registry executes schemas selected by their complete expected key.
//
// Document-language dispatch is deliberately absent. Different schema IDs may
// register the same kind and version; only a repeated complete key conflicts.
type Registry struct {
	codecs map[schemaModel.Key]registeredCodec
	keys   []schemaModel.Key
}

func NewRegistry(codecs ...schema.Codec) (*Registry, error) {
	values := make(map[schemaModel.Key]registeredCodec, len(codecs))
	keys := make([]schemaModel.Key, 0, len(codecs))

	for index, codec := range codecs {
		if codec == nil {
			return nil, fmt.Errorf("%w: schema codec %d is nil", storeSpec.ErrInvalid, index)
		}
		key := codec.Key()
		if err := key.Validate(); err != nil {
			return nil, err
		}
		if _, exists := values[key]; exists {
			return nil, fmt.Errorf(
				"%w: duplicate schema %q/%q/%q/%q",
				storeSpec.ErrConflict,
				key.Entity,
				key.Kind,
				key.SchemaID,
				key.SchemaVersion,
			)
		}

		compiled, err := jsonutil.CompileJSONSchema(codec.JSONSchema())
		if err != nil {
			return nil, fmt.Errorf(
				"%w: compile schema %q/%q/%q: %w",
				storeSpec.ErrInvalid,
				key.Kind,
				key.SchemaID,
				key.SchemaVersion,
				err,
			)
		}
		values[key] = registeredCodec{codec: codec, schema: compiled}
		keys = append(keys, key)
	}

	sort.Slice(keys, func(left, right int) bool {
		a, b := keys[left], keys[right]
		switch {
		case a.Entity != b.Entity:
			return a.Entity < b.Entity
		case a.Kind != b.Kind:
			return a.Kind < b.Kind
		case a.SchemaID != b.SchemaID:
			return a.SchemaID < b.SchemaID
		default:
			return a.SchemaVersion < b.SchemaVersion
		}
	})

	return &Registry{codecs: values, keys: keys}, nil
}

func (r *Registry) Keys() []schemaModel.Key {
	return append([]schemaModel.Key(nil), r.keys...)
}

func (r *Registry) CanonicalizeExpected(
	ctx context.Context,
	expected schemaModel.Key,
	raw []byte,
) (schemaModel.ParsedDocument, error) {
	if err := expected.Validate(); err != nil {
		return schemaModel.ParsedDocument{}, err
	}

	registered, found := r.codecs[expected]
	if !found {
		return schemaModel.ParsedDocument{}, fmt.Errorf(
			"%w: schema %q/%q/%q/%q",
			storeSpec.ErrUnsupported,
			expected.Entity,
			expected.Kind,
			expected.SchemaID,
			expected.SchemaVersion,
		)
	}

	canonical, err := jsonutil.CanonicalizeObject(raw, storeSpec.MaxDefinitionBytes)
	if err != nil {
		return schemaModel.ParsedDocument{}, err
	}
	if err := jsonutil.ValidateJSONSchema(
		registered.schema,
		json.RawMessage(canonical),
		storeSpec.MaxDefinitionBytes,
	); err != nil {
		return schemaModel.ParsedDocument{}, fmt.Errorf(
			"%w: document does not satisfy its expected JSON Schema: %w",
			storeSpec.ErrInvalid,
			err,
		)
	}

	// Keep the validated input independent of plugin-owned mutable bytes.
	// Otherwise a codec could mutate its input and defeat the changed-output
	// comparison below.
	value, err := registered.codec.Canonicalize(
		ctx,
		append([]byte(nil), canonical...),
	)
	if err != nil {
		return schemaModel.ParsedDocument{}, err
	}
	if value.Key != expected {
		return schemaModel.ParsedDocument{}, fmt.Errorf(
			"%w: schema codec returned another schema key",
			storeSpec.ErrInvalid,
		)
	}

	output := append([]byte(nil), value.Raw...)
	if !bytes.Equal(output, canonical) {
		normalized, err := jsonutil.CanonicalizeObject(output, storeSpec.MaxDefinitionBytes)
		if err != nil {
			return schemaModel.ParsedDocument{}, err
		}
		if !bytes.Equal(normalized, output) {
			return schemaModel.ParsedDocument{}, fmt.Errorf(
				"%w: schema codec returned non-canonical JSON",
				storeSpec.ErrInvalid,
			)
		}
		if err := jsonutil.ValidateJSONSchema(
			registered.schema,
			json.RawMessage(output),
			storeSpec.MaxDefinitionBytes,
		); err != nil {
			return schemaModel.ParsedDocument{}, fmt.Errorf(
				"%w: changed codec output does not satisfy its expected JSON Schema: %w",
				storeSpec.ErrInvalid,
				err,
			)
		}
	}

	if value.Digest != cryptoutil.DigestBytes(output) {
		return schemaModel.ParsedDocument{}, fmt.Errorf(
			"%w: schema codec output does not match its document digest",
			storeSpec.ErrDigestMismatch,
		)
	}

	return schemaModel.ParsedDocument{
		Key:    expected,
		Digest: value.Digest,
		Raw:    output,
	}, nil
}
