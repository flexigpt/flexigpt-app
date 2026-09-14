package shareable

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type registeredCodec struct {
	codec  providerapi.SchemaCodec
	schema *jsonschema.Schema
}

type typeVersionKey struct {
	Entity     schema.EntityType
	Type       schema.Kind
	APIVersion string
}

type Registry struct {
	codecs        map[schema.Key]registeredCodec
	byTypeVersion map[typeVersionKey]schema.Key
	byType        map[schema.Kind][]schema.Key
	keys          []schema.Key
}

func NewRegistry(codecs ...providerapi.SchemaCodec) (*Registry, error) {
	values := make(map[schema.Key]registeredCodec, len(codecs))
	byTypeVersion := make(
		map[typeVersionKey]schema.Key,
		len(codecs),
	)
	byType := make(map[schema.Kind][]schema.Key)
	keys := make([]schema.Key, 0, len(codecs))

	for _, codec := range codecs {
		if codec == nil {
			return nil, fmt.Errorf("%w: shareable schema codec is nil", basespec.ErrInvalid)
		}
		key := codec.Key()
		if err := key.Validate(); err != nil {
			return nil, err
		}
		compiled, err := jsonutil.CompileJSONSchema(
			codec.JSONSchema(),
		)
		if err != nil {
			return nil, fmt.Errorf(
				"%w: compile published JSON Schema: %w",
				basespec.ErrInvalid,
				err,
			)
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

		if key.Entity == schema.EntityArtifact {
			portableKey := typeVersionKey{
				Entity:     key.Entity,
				Type:       key.Kind,
				APIVersion: key.SchemaVersion,
			}
			if previous, duplicate := byTypeVersion[portableKey]; duplicate {
				return nil, fmt.Errorf(
					"%w: schemas %q and %q both dispatch from type %q and apiVersion %q",
					basespec.ErrConflict,
					previous.SchemaID,
					key.SchemaID,
					key.Kind,
					key.SchemaVersion,
				)
			}
			byTypeVersion[portableKey] = key
			byType[key.Kind] = append(byType[key.Kind], key)
		}
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
	for kind := range byType {
		sort.Slice(byType[kind], func(left, right int) bool {
			if byType[kind][left].SchemaVersion !=
				byType[kind][right].SchemaVersion {
				return byType[kind][left].SchemaVersion <
					byType[kind][right].SchemaVersion
			}
			return byType[kind][left].SchemaID <
				byType[kind][right].SchemaID
		})
	}
	return &Registry{
		codecs:        values,
		byTypeVersion: byTypeVersion,
		byType:        byType,
		keys:          keys,
	}, nil
}

func (r *Registry) Keys() []schema.Key {
	if r == nil {
		return nil
	}
	return append([]schema.Key(nil), r.keys...)
}

func (r *Registry) Canonicalize(
	ctx context.Context,
	raw []byte,
) (schema.ParsedDocument, error) {
	return r.CanonicalizeEntity(ctx, schema.EntityArtifact, raw)
}

// CanonicalizeExpected canonicalizes raw content through the Artifact Store
// registry and requires the resulting schema key to match expected.
//
// Feature services use this boundary for known document inputs. The registry
// remains the only owner of JSON Schema execution, codec selection, canonical
// JSON enforcement, and canonical output validation.
func (r *Registry) CanonicalizeExpected(
	ctx context.Context,
	expected schema.Key,
	raw []byte,
) (schema.ParsedDocument, error) {
	if err := expected.Validate(); err != nil {
		return schema.ParsedDocument{}, err
	}

	value, err := r.CanonicalizeEntity(ctx, expected.Entity, raw)
	if err != nil {
		return schema.ParsedDocument{}, err
	}
	if value.Key != expected {
		return schema.ParsedDocument{}, fmt.Errorf(
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
	entity schema.EntityType,
	raw []byte,
) (schema.ParsedDocument, error) {
	if r == nil {
		return schema.ParsedDocument{}, basespec.ErrClosed
	}
	if entity != schema.EntityArtifact {
		return schema.ParsedDocument{}, fmt.Errorf(
			"%w: unsupported schema entity %q",
			basespec.ErrInvalid,
			entity,
		)
	}

	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxDefinitionBytes,
	)
	if err != nil {
		return schema.ParsedDocument{}, err
	}

	key, headerStyle, err := r.dispatchKey(
		entity,
		canonical,
	)
	if err != nil {
		return schema.ParsedDocument{}, err
	}
	registered, found := r.codecs[key]
	if !found {
		return schema.ParsedDocument{}, fmt.Errorf(
			"%w: shareable %s schema %q/%q/%q",
			basespec.ErrUnsupported,
			entity,
			key.Kind,
			key.SchemaID,
			key.SchemaVersion,
		)
	}

	if err := jsonutil.ValidateJSONSchema(
		registered.schema,
		json.RawMessage(canonical),
		basespec.MaxDefinitionBytes,
	); err != nil {
		return schema.ParsedDocument{}, fmt.Errorf(
			"%w: shareable document does not satisfy its JSON Schema: %w",
			basespec.ErrInvalid,
			err,
		)
	}

	value, err := registered.codec.Canonicalize(ctx, canonical)
	if err != nil {
		return schema.ParsedDocument{}, err
	}
	if value.Key != key {
		return schema.ParsedDocument{}, fmt.Errorf(
			"%w: shareable codec returned another schema key",
			basespec.ErrInvalid,
		)
	}
	can, err := validateCodecOutput(key, value, headerStyle)
	if err != nil {
		return schema.ParsedDocument{}, err
	}
	if err := jsonutil.ValidateJSONSchema(
		registered.schema,
		json.RawMessage(can),
		basespec.MaxDefinitionBytes,
	); err != nil {
		return schema.ParsedDocument{}, fmt.Errorf(
			"%w: canonical codec output does not satisfy its JSON Schema: %w",
			basespec.ErrInvalid,
			err,
		)
	}

	value.Raw = json.RawMessage(can)
	return value.Clone(), nil
}

type documentHeaderStyle int

const (
	documentHeaderLegacy documentHeaderStyle = iota + 1
	documentHeaderTypeVersion
)

func (r *Registry) dispatchKey(
	entity schema.EntityType,
	canonical []byte,
) (schema.Key, documentHeaderStyle, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(canonical, &fields); err != nil {
		return schema.Key{}, 0, fmt.Errorf(
			"%w: decode shareable document header: %w",
			basespec.ErrInvalid,
			err,
		)
	}

	_, hasType := fields["type"]
	_, hasAPIVersion := fields["apiVersion"]
	_, hasKind := fields["kind"]
	_, hasSchemaID := fields["schemaID"]
	_, hasSchemaVersion := fields["schemaVersion"]

	hasTypeHeader := hasType || hasAPIVersion
	hasLegacyHeader := hasKind ||
		hasSchemaID ||
		hasSchemaVersion
	if hasTypeHeader && hasLegacyHeader {
		return schema.Key{}, 0, fmt.Errorf(
			"%w: shareable document mixes type/apiVersion and kind/schemaID/schemaVersion headers",
			basespec.ErrInvalid,
		)
	}

	if hasTypeHeader {
		if entity != schema.EntityArtifact {
			return schema.Key{}, 0, fmt.Errorf(
				"%w: type/apiVersion documents are Artifact declarations",
				basespec.ErrInvalid,
			)
		}

		var header struct {
			Type       schema.Kind `json:"type"`
			APIVersion string      `json:"apiVersion"`
		}
		if err := json.Unmarshal(canonical, &header); err != nil {
			return schema.Key{}, 0, fmt.Errorf(
				"%w: decode type/apiVersion header: %w",
				basespec.ErrInvalid,
				err,
			)
		}
		if err := artifactKindFromSchemaKind(header.Type).Validate(); err != nil {
			return schema.Key{}, 0, err
		}

		if header.APIVersion != "" {
			key, found := r.byTypeVersion[typeVersionKey{
				Entity:     entity,
				Type:       header.Type,
				APIVersion: header.APIVersion,
			}]
			if !found {
				return schema.Key{}, 0, fmt.Errorf(
					"%w: shareable Artifact type %q apiVersion %q",
					basespec.ErrUnsupported,
					header.Type,
					header.APIVersion,
				)
			}
			return key, documentHeaderTypeVersion, nil
		}

		candidates := r.byType[header.Type]
		if len(candidates) != 1 {
			return schema.Key{}, 0, fmt.Errorf(
				"%w: Artifact type %q without apiVersion resolves to %d registered schemas",
				basespec.ErrUnsupported,
				header.Type,
				len(candidates),
			)
		}
		return candidates[0], documentHeaderTypeVersion, nil
	}

	var header struct {
		Kind          string          `json:"kind"`
		SchemaID      schema.SchemaID `json:"schemaID"`
		SchemaVersion string          `json:"schemaVersion"`
	}
	if err := json.Unmarshal(canonical, &header); err != nil {
		return schema.Key{}, 0, fmt.Errorf(
			"%w: decode legacy shareable document header: %w",
			basespec.ErrInvalid,
			err,
		)
	}

	key := schema.Key{
		Entity:        entity,
		Kind:          schema.Kind(header.Kind),
		SchemaID:      header.SchemaID,
		SchemaVersion: header.SchemaVersion,
	}
	if err := key.Validate(); err != nil {
		return schema.Key{}, 0, err
	}
	return key, documentHeaderLegacy, nil
}

func validateCodecOutput(
	expected schema.Key,
	value schema.ParsedDocument,
	headerStyle documentHeaderStyle,
) ([]byte, error) {
	if err := value.Validate(); err != nil {
		return nil, err
	}
	if value.Key != expected {
		return nil, fmt.Errorf(
			"%w: shareable codec returned another schema key",
			basespec.ErrInvalid,
		)
	}

	canonical, err := jsonutil.CanonicalizeObject(
		value.Raw,
		basespec.MaxDefinitionBytes,
	)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(canonical, value.Raw) {
		return nil, fmt.Errorf(
			"%w: shareable codec returned non-canonical JSON",
			basespec.ErrInvalid,
		)
	}
	if headerStyle == documentHeaderTypeVersion {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(canonical, &fields); err != nil {
			return nil, err
		}
		if _, found := fields["kind"]; found {
			return nil, fmt.Errorf(
				"%w: type/apiVersion codec emitted a legacy kind",
				basespec.ErrInvalid,
			)
		}
		if _, found := fields["schemaID"]; found {
			return nil, fmt.Errorf(
				"%w: type/apiVersion codec emitted a legacy schemaID",
				basespec.ErrInvalid,
			)
		}
		if _, found := fields["schemaVersion"]; found {
			return nil, fmt.Errorf(
				"%w: type/apiVersion codec emitted a legacy schemaVersion",
				basespec.ErrInvalid,
			)
		}

		var header struct {
			Type       schema.Kind `json:"type"`
			APIVersion string      `json:"apiVersion"`
		}
		if err := json.Unmarshal(canonical, &header); err != nil {
			return nil, err
		}
		if header.Type != expected.Kind ||
			(header.APIVersion != "" &&
				header.APIVersion != expected.SchemaVersion) ||
			value.Digest != cryptoutil.DigestBytes(canonical) {
			return nil, fmt.Errorf(
				"%w: type/apiVersion codec output does not match its metadata",
				basespec.ErrDigestMismatch,
			)
		}
		return canonical, nil
	}

	var header struct {
		Kind          string          `json:"kind"`
		SchemaID      schema.SchemaID `json:"schemaID"`
		SchemaVersion string          `json:"schemaVersion"`
		Digest        string          `json:"digest"`
	}
	if err := json.Unmarshal(canonical, &header); err != nil {
		return nil, fmt.Errorf(
			"%w: decode canonical shareable document header: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	actual := schema.Key{
		Entity:        expected.Entity,
		Kind:          schema.Kind(header.Kind),
		SchemaID:      header.SchemaID,
		SchemaVersion: header.SchemaVersion,
	}
	if actual != expected || header.Digest != string(value.Digest) {
		return nil, fmt.Errorf(
			"%w: shareable codec output does not match its metadata",
			basespec.ErrDigestMismatch,
		)
	}
	return canonical, nil
}

func artifactKindFromSchemaKind(value schema.Kind) artifact.ArtifactKind {
	return artifact.ArtifactKind(value)
}
