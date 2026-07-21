package fsrepo

import (
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/jsoncanon"
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
			artifactstore.ErrInvalid,
			f.Format,
		)
	}
	return f.Definition.Validate()
}

func encodeFile(value definition.Definition) ([]byte, error) {
	canonical, err := definition.Canonicalize(value)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(file{
		Format:     fileFormatV1,
		Definition: canonical,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal definition file: %w", err)
	}
	return jsoncanon.Canonicalize(raw)
}

func decodeFile(raw []byte) (definition.Definition, error) {
	canonicalFile, err := jsoncanon.Canonicalize(raw)
	if err != nil {
		return definition.Definition{}, err
	}
	var value file
	if err := json.Unmarshal(canonicalFile, &value); err != nil {
		return definition.Definition{}, fmt.Errorf("decode definition file: %w", err)
	}
	if err := value.validate(); err != nil {
		return definition.Definition{}, err
	}
	return definition.Canonicalize(value.Definition)
}
