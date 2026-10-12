package ingest

import (
	"fmt"
	"sort"

	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func NormalizeDecoders(
	values []Decoder,
) ([]Decoder, error) {
	output := append([]Decoder(nil), values...)
	seen := make(map[storeSpec.DecoderID]struct{}, len(output))

	for index, decoder := range output {
		if decoder == nil {
			return nil, fmt.Errorf(
				"%w: decoder %d is nil",
				storeSpec.ErrInvalid,
				index,
			)
		}

		id := decoder.ID()
		if err := id.Validate(); err != nil {
			return nil, err
		}
		if err := storeSpec.ValidateRequiredText(
			"decoder revision",
			decoder.Revision(),
			storeSpec.MaxVersionBytes,
		); err != nil {
			return nil, err
		}
		if _, duplicate := seen[id]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate decoder %q",
				storeSpec.ErrConflict,
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
