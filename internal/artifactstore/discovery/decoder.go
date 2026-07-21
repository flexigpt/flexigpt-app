package discovery

import (
	"context"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

type Recognition int

const (
	RecognitionNone Recognition = iota
	RecognitionPossible
	RecognitionPreferred
)

type Candidate struct {
	Source              source.Source
	Locator             artifactstore.Locator
	SourceContentDigest artifactstore.Digest
	Content             []byte
}

type Decoded struct {
	SubresourceLocator artifactstore.SubresourceLocator
	Definition         definition.Definition
}

type Decoder interface {
	ID() artifactstore.DecoderID
	Recognize(ctx context.Context, candidate Candidate) Recognition
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

func (r *DecoderRegistry) Get(
	id artifactstore.DecoderID,
) (Decoder, bool) {
	if r == nil {
		return nil, false
	}
	value, exists := r.byID[id]
	return value, exists
}

func (r *DecoderRegistry) All() []Decoder {
	if r == nil {
		return nil
	}
	return append([]Decoder(nil), r.decoders...)
}
