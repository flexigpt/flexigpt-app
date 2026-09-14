package jsonutil

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

var errInvalid = errors.New("invalid jsonschema")

// MustCompileJSONSchema compiles an embedded schema during package
// initialization. Embedded source-controlled schemas are expected to be valid.
func MustCompileJSONSchema(raw []byte) *jsonschema.Schema {
	compiled, err := CompileJSONSchema(raw)
	if err != nil {
		panic(fmt.Sprintf("compile embedded JSON Schema: %v", err))
	}
	return compiled
}

// CompileJSONSchema compiles one published JSON Schema. The caller owns input
// size limits when schemas are not embedded and source-controlled.
func CompileJSONSchema(
	raw []byte,
) (*jsonschema.Schema, error) {
	var header struct {
		Schema string `json:"$schema"`
		ID     string `json:"$id"`
	}
	if err := json.Unmarshal(raw, &header); err != nil {
		return nil, fmt.Errorf(
			"decode JSON Schema header: %w",
			err,
		)
	}
	if header.Schema == "" || header.ID == "" {
		return nil, fmt.Errorf(
			"%w: JSON Schema requires $schema and $id",
			errInvalid,
		)
	}

	compiler := jsonschema.NewCompiler()

	resource, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf(
			"decode JSON Schema resource: %w",
			err,
		)
	}
	if err := compiler.AddResource(header.ID, resource); err != nil {
		return nil, fmt.Errorf(
			"%w: register JSON Schema resource %q: %w",
			errInvalid,
			header.ID,
			err,
		)
	}

	compiled, err := compiler.Compile(header.ID)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: compile JSON Schema resource %q: %w",
			errInvalid,
			header.ID,
			err,
		)
	}
	return compiled, nil
}

// ValidateJSONSchema validates a typed document against an already-compiled
// JSON Schema. It does not canonicalize or mutate the value.
func ValidateJSONSchema(
	schema *jsonschema.Schema,
	value any,
	maximumBytes int,
) error {
	if schema == nil {
		return fmt.Errorf("%w: compiled JSON Schema is nil", errInvalid)
	}
	if maximumBytes <= 0 {
		return fmt.Errorf("%w: JSON byte limit is invalid", errInvalid)
	}

	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal JSON Schema instance: %w", err)
	}
	if len(raw) > maximumBytes {
		return fmt.Errorf(
			"%w: JSON Schema instance exceeds %d bytes",
			errInvalid,
			maximumBytes,
		)
	}

	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf(
			"%w: decode JSON Schema instance: %w",
			errInvalid,
			err,
		)
	}
	if err := schema.Validate(instance); err != nil {
		return fmt.Errorf(
			"%w: document does not satisfy JSON Schema: %w",
			errInvalid,
			err,
		)
	}
	return nil
}
