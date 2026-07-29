package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type Recognition int

const (
	RecognitionNone Recognition = iota
	RecognitionPossible
	RecognitionPreferred
)

type Candidate struct {
	SourceID            basespec.SourceID
	SourceKind          basespec.SourceKind
	Locator             basespec.Locator
	SourceContentDigest cryptoutil.Digest
	Content             []byte
	RequestedDecoderIDs []basespec.DecoderID
}

func (c Candidate) RequestsDecoder(id basespec.DecoderID) bool {
	return slices.Contains(c.RequestedDecoderIDs, id)
}

type Decoded struct {
	SubresourceLocator basespec.SubresourceLocator
	Definition         definition.Definition
	Diagnostics        []diagnostic.Diagnostic
}

type Decoder interface {
	ID() basespec.DecoderID
	Revision() string
	Recognize(ctx context.Context, candidate Candidate) Recognition

	// Decode returns candidate-level diagnostics as its second result.
	//
	// Diagnostics attached to Decoded apply only to that emitted subresource.
	Decode(
		ctx context.Context,
		candidate Candidate,
	) ([]Decoded, []diagnostic.Diagnostic)
}

type DecoderRegistry struct {
	decoders []Decoder
	byID     map[basespec.DecoderID]Decoder
}

func NewDecoderRegistry(
	decoders ...Decoder,
) (*DecoderRegistry, error) {
	byID := make(map[basespec.DecoderID]Decoder, len(decoders))
	ordered := make([]Decoder, 0, len(decoders))
	for _, decoder := range decoders {
		if decoder == nil {
			return nil, fmt.Errorf("%w: decoder is nil", basespec.ErrInvalid)
		}
		id := decoder.ID()
		if err := basespec.ValidateDecoderID(id); err != nil {
			return nil, err
		}
		if err := basespec.ValidateRequiredText(
			"decoder revision",
			decoder.Revision(),
			basespec.MaxVersionBytes,
		); err != nil {
			return nil, err
		}
		if _, duplicate := byID[id]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate decoder %q",
				basespec.ErrConflict,
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

func (r *DecoderRegistry) Fingerprint() (cryptoutil.Digest, error) {
	if r == nil {
		return "", fmt.Errorf(
			"%w: decoder registry is nil",
			basespec.ErrInvalid,
		)
	}
	type descriptor struct {
		ID       basespec.DecoderID `json:"id"`
		Revision string             `json:"revision"`
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
	canonical, err := jsonutil.Canonicalize(raw)
	if err != nil {
		return "", err
	}
	return cryptoutil.DigestBytes(canonical), nil
}

func (r *DecoderRegistry) find(
	id basespec.DecoderID,
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
