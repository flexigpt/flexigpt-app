package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/jsoncanon"
)

type Recognition int

const (
	RecognitionNone Recognition = iota
	RecognitionPossible
	RecognitionPreferred
)

type Candidate struct {
	SourceID            artifactstore.SourceID
	SourceKind          artifactstore.SourceKind
	Locator             artifactstore.Locator
	SourceContentDigest artifactstore.Digest
	Content             []byte
	RequestedDecoderIDs []artifactstore.DecoderID
}

func (c Candidate) RequestsDecoder(id artifactstore.DecoderID) bool {
	return slices.Contains(c.RequestedDecoderIDs, id)
}

type Decoded struct {
	SubresourceLocator artifactstore.SubresourceLocator
	Definition         definition.Definition
	Diagnostics        []artifactstore.Diagnostic
}

type Decoder interface {
	ID() artifactstore.DecoderID
	Revision() string
	Recognize(ctx context.Context, candidate Candidate) Recognition

	// Decode returns candidate-level diagnostics as its second result.
	//
	// Diagnostics attached to Decoded apply only to that emitted subresource.
	Decode(
		ctx context.Context,
		candidate Candidate,
	) ([]Decoded, []artifactstore.Diagnostic)
}

type DecoderRegistry struct {
	decoders []Decoder
	byID     map[artifactstore.DecoderID]Decoder
}

func NewDecoderRegistry(
	decoders ...Decoder,
) (*DecoderRegistry, error) {
	byID := make(map[artifactstore.DecoderID]Decoder, len(decoders))
	ordered := make([]Decoder, 0, len(decoders))
	for _, decoder := range decoders {
		if decoder == nil {
			return nil, fmt.Errorf("%w: decoder is nil", artifactstore.ErrInvalid)
		}
		id := decoder.ID()
		if err := artifactstore.ValidateDecoderID(id); err != nil {
			return nil, err
		}
		if err := artifactstore.ValidateRequiredText(
			"decoder revision",
			decoder.Revision(),
			artifactstore.MaxVersionBytes,
		); err != nil {
			return nil, err
		}
		if _, duplicate := byID[id]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate decoder %q",
				artifactstore.ErrConflict,
				id,
			)
		}
		byID[id] = decoder
		ordered = append(ordered, decoder)
	}
	sort.Slice(ordered, func(left, right int) bool {
		return ordered[left].ID() < ordered[right].ID()
	})
	return &DecoderRegistry{
		decoders: ordered,
		byID:     byID,
	}, nil
}

func (r *DecoderRegistry) Fingerprint() (artifactstore.Digest, error) {
	if r == nil {
		return "", fmt.Errorf(
			"%w: decoder registry is nil",
			artifactstore.ErrInvalid,
		)
	}
	type descriptor struct {
		ID       artifactstore.DecoderID `json:"id"`
		Revision string                  `json:"revision"`
	}
	values := make([]descriptor, 0, len(r.decoders))
	for _, decoder := range r.decoders {
		values = append(values, descriptor{
			ID:       decoder.ID(),
			Revision: decoder.Revision(),
		})
	}
	raw, err := json.Marshal(values)
	if err != nil {
		return "", err
	}
	canonical, err := jsoncanon.Canonicalize(raw)
	if err != nil {
		return "", err
	}
	return artifactstore.DigestBytes(canonical), nil
}

func (r *DecoderRegistry) find(
	id artifactstore.DecoderID,
) (Decoder, bool) {
	if r == nil {
		return nil, false
	}
	value, exists := r.byID[id]
	return value, exists
}

func (r *DecoderRegistry) registered() []Decoder {
	if r == nil {
		return nil
	}
	return append([]Decoder(nil), r.decoders...)
}
