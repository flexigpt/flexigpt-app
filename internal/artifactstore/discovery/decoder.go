package discovery

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type DecoderRegistry struct {
	decoders []providerapi.Decoder
	byID     map[basespec.DecoderID]providerapi.Decoder
}

func NewDecoderRegistry(
	decoders ...providerapi.Decoder,
) (*DecoderRegistry, error) {
	byID := make(map[basespec.DecoderID]providerapi.Decoder, len(decoders))
	ordered := make([]providerapi.Decoder, 0, len(decoders))
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
) (providerapi.Decoder, bool) {
	if r == nil {
		return nil, false
	}
	value, exists := r.byID[id]
	return value, exists
}

func (r *DecoderRegistry) registered() []providerapi.Decoder {
	if r == nil {
		return nil
	}
	return append([]providerapi.Decoder(nil), r.decoders...)
}
