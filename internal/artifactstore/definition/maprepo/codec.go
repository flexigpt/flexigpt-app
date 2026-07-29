package maprepo

import (
	"fmt"

	"github.com/flexigpt/mapstore-go/jsonencdec"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
)

const fileFormatV1 = "artifact-definition/v1"

type file struct {
	Format     string                `json:"format"`
	Definition definition.Definition `json:"definition"`
}

func (f file) validate() error {
	if f.Format != fileFormatV1 {
		return fmt.Errorf(
			"%w: unsupported definition file format %q",
			basespec.ErrInvalid,
			f.Format,
		)
	}
	return f.Definition.Validate()
}

func encodeFile(
	value definition.Definition,
) (map[string]any, error) {
	canonical, err := definition.Canonicalize(value)
	if err != nil {
		return nil, err
	}
	return jsonencdec.StructWithJSONTagsToMap(file{
		Format:     fileFormatV1,
		Definition: canonical,
	})
}

func decodeFile(
	raw map[string]any,
) (definition.Definition, error) {
	var value file
	if err := jsonencdec.MapToStructWithJSONTags(raw, &value); err != nil {
		return definition.Definition{}, fmt.Errorf(
			"decode definition MapStore file: %w",
			err,
		)
	}
	if err := value.validate(); err != nil {
		return definition.Definition{}, err
	}
	canonical, err := definition.Canonicalize(value.Definition)
	if err != nil {
		return definition.Definition{}, err
	}
	if canonical.Digest != value.Definition.Digest {
		return definition.Definition{}, fmt.Errorf(
			"%w: definition file digest changed during canonicalization",
			basespec.ErrDigestMismatch,
		)
	}
	return canonical, nil
}
