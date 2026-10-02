package internal

import (
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider"
	schema "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type DecoderRegistry struct {
	decoders    []provider.Decoder
	byID        map[spec.DecoderID]provider.Decoder
	fingerprint cryptoutil.Digest
}

func NewDecoderRegistry(
	codecs []provider.SchemaCodec,
	decoders ...provider.Decoder,
) (*DecoderRegistry, error) {
	byID := make(map[spec.DecoderID]provider.Decoder, len(decoders))
	ordered := make([]provider.Decoder, 0, len(decoders))
	for _, decoder := range decoders {
		if decoder == nil {
			return nil, fmt.Errorf("%w: decoder is nil", spec.ErrInvalid)
		}
		id := decoder.ID()
		if err := id.Validate(); err != nil {
			return nil, err
		}
		if err := spec.ValidateRequiredText(
			"decoder revision",
			decoder.Revision(),
			spec.MaxVersionBytes,
		); err != nil {
			return nil, err
		}
		if _, duplicate := byID[id]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate decoder %q",
				spec.ErrConflict,
				id,
			)
		}
		byID[id] = decoder
		ordered = append(ordered, decoder)
	}
	sort.Slice(ordered, func(left, right int) bool {
		return ordered[left].ID() < ordered[right].ID()
	})
	fingerprint, err := registryFingerprint(codecs, ordered)
	if err != nil {
		return nil, err
	}
	return &DecoderRegistry{
		decoders:    ordered,
		byID:        byID,
		fingerprint: fingerprint,
	}, nil
}

func (r *DecoderRegistry) Fingerprint() (cryptoutil.Digest, error) {
	if r == nil {
		return "", fmt.Errorf(
			"%w: decoder registry is nil",
			spec.ErrInvalid,
		)
	}
	return r.fingerprint, nil
}

func registryFingerprint(
	codecs []provider.SchemaCodec,
	decoders []provider.Decoder,
) (cryptoutil.Digest, error) {
	type descriptor struct {
		ID       spec.DecoderID `json:"id"`
		Revision string         `json:"revision"`
	}
	values := make([]descriptor, 0, len(decoders))
	for _, decoder := range decoders {
		values = append(values, descriptor{
			ID:       decoder.ID(),
			Revision: decoder.Revision(),
		})
	}
	type schemaDescriptor struct {
		Key    schema.Key        `json:"key"`
		Digest cryptoutil.Digest `json:"digest"`
	}
	schemas := make([]schemaDescriptor, 0, len(codecs))
	for _, codec := range codecs {
		if codec == nil {
			return "", fmt.Errorf("%w: nil schema codec", spec.ErrInvalid)
		}
		raw, err := jsonutil.Canonicalize(codec.JSONSchema())
		if err != nil {
			return "", err
		}
		schemas = append(schemas, schemaDescriptor{
			Key:    codec.Key(),
			Digest: cryptoutil.DigestBytes(raw),
		})
	}
	sort.Slice(schemas, func(i, j int) bool {
		left, right := schemas[i].Key, schemas[j].Key
		if left.Entity != right.Entity {
			return left.Entity < right.Entity
		}
		if left.Kind != right.Kind {
			return left.Kind < right.Kind
		}
		if left.SchemaID != right.SchemaID {
			return left.SchemaID < right.SchemaID
		}
		return left.SchemaVersion < right.SchemaVersion
	})
	raw, err := jsonutil.MarshalCanonicalObject(map[string]any{
		"format":   "artifact-decoder-registry/v1",
		"decoders": values,
		"schemas":  schemas,
	}, spec.MaxDefinitionBytes)
	if err != nil {
		return "", err
	}
	return cryptoutil.DigestBytes(raw), nil
}

func (r *DecoderRegistry) find(
	id spec.DecoderID,
) (provider.Decoder, bool) {
	if r == nil {
		return nil, false
	}
	value, exists := r.byID[id]
	return value, exists
}

func (r *DecoderRegistry) registered() []provider.Decoder {
	if r == nil {
		return nil
	}
	return append([]provider.Decoder(nil), r.decoders...)
}
