package schema

import (
	"fmt"

	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func NormalizeCodecs(
	values []Codec,
) ([]Codec, error) {
	output := append([]Codec(nil), values...)
	seen := make(map[schemaModel.Key]struct{}, len(output))

	for index, codec := range output {
		if codec == nil {
			return nil, fmt.Errorf(
				"%w: schema codec %d is nil",
				spec.ErrInvalid,
				index,
			)
		}

		key := codec.Key()
		if err := key.Validate(); err != nil {
			return nil, fmt.Errorf(
				"schema codec %d: %w",
				index,
				err,
			)
		}
		if _, duplicate := seen[key]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate schema %q/%q/%q",
				spec.ErrConflict,
				key.Kind,
				key.SchemaID,
				key.SchemaVersion,
			)
		}
		seen[key] = struct{}{}
	}

	return output, nil
}
