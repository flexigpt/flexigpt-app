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
	var header struct {
		Schema string `json:"$schema"`
		ID     string `json:"$id"`
	}
	if err := json.Unmarshal(raw, &header); err != nil {
		panic(fmt.Sprintf("decode embedded JSON Schema: %v", err))
	}
	if header.Schema == "" || header.ID == "" {
		panic("embedded JSON Schema requires $schema and $id")
	}

	compiler := jsonschema.NewCompiler()

	resource, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		panic(fmt.Sprintf("decode embedded JSON Schema resource: %v", err))
	}
	if err := compiler.AddResource(header.ID, resource); err != nil {
		panic(fmt.Sprintf(
			"register embedded JSON Schema resource %q: %v",
			header.ID,
			err,
		))
	}

	compiled, err := compiler.Compile(header.ID)
	if err != nil {
		panic(fmt.Sprintf(
			"compile embedded JSON Schema resource %q: %v",
			header.ID,
			err,
		))
	}
	return compiled
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
