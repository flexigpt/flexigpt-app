package ingest

import (
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func NormalizeDecoders(
	values []Decoder,
) ([]Decoder, error) {
	output := append([]Decoder(nil), values...)
	seen := make(map[spec.DecoderID]struct{}, len(output))

	for index, decoder := range output {
		if decoder == nil {
			return nil, fmt.Errorf(
				"%w: decoder %d is nil",
				spec.ErrInvalid,
				index,
			)
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
		if _, duplicate := seen[id]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate decoder %q",
				spec.ErrConflict,
				id,
			)
		}
		seen[id] = struct{}{}
	}

	sort.Slice(output, func(left, right int) bool {
		return output[left].ID() < output[right].ID()
	})

	return output, nil
}
