package ingest

import (
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema"
	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type DecoderRegistry struct {
	decoders    []Decoder
	byID        map[spec.DecoderID]Decoder
	fingerprint cryptoutil.Digest
}

func NewDecoderRegistry(
	codecs []schema.Codec,
	decoders ...Decoder,
) (*DecoderRegistry, error) {
	ordered, err := NormalizeDecoders(decoders)
	if err != nil {
		return nil, err
	}

	byID := make(map[spec.DecoderID]Decoder, len(ordered))
	for _, decoder := range ordered {
		byID[decoder.ID()] = decoder
	}

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

func (r *DecoderRegistry) Fingerprint() (
	cryptoutil.Digest,
	error,
) {
	if r == nil {
		return "", fmt.Errorf(
			"%w: decoder registry is nil",
			spec.ErrInvalid,
		)
	}
	return r.fingerprint, nil
}

func registryFingerprint(
	codecs []schema.Codec,
	decoders []Decoder,
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
		Key    schemaModel.Key   `json:"key"`
		Digest cryptoutil.Digest `json:"digest"`
	}
	schemas := make([]schemaDescriptor, 0, len(codecs))
	for _, codec := range codecs {
		if codec == nil {
			return "", fmt.Errorf(
				"%w: nil schema codec",
				spec.ErrInvalid,
			)
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

	sort.Slice(schemas, func(left, right int) bool {
		leftKey := schemas[left].Key
		rightKey := schemas[right].Key
		if leftKey.Entity != rightKey.Entity {
			return leftKey.Entity < rightKey.Entity
		}
		if leftKey.Kind != rightKey.Kind {
			return leftKey.Kind < rightKey.Kind
		}
		if leftKey.SchemaID != rightKey.SchemaID {
			return leftKey.SchemaID < rightKey.SchemaID
		}
		return leftKey.SchemaVersion < rightKey.SchemaVersion
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
) (Decoder, bool) {
	if r == nil {
		return nil, false
	}
	value, found := r.byID[id]
	return value, found
}

func (r *DecoderRegistry) registered() []Decoder {
	if r == nil {
		return nil
	}
	return append([]Decoder(nil), r.decoders...)
}
