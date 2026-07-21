package discovery

import (
	"context"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
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
}

type Decoded struct {
	SubresourceLocator artifactstore.SubresourceLocator
	Definition         definition.Definition
	Diagnostics        []artifactstore.Diagnostic
}

type Decoder interface {
	ID() artifactstore.DecoderID
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
