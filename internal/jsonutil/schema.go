package jsonutil

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

var errInvalid = errors.New("invalid jsonschema")

const (
	anonymousSchemaID    = "https://schemas.flexigpt.dev/internal/jsonschema/anonymous"
	draft202012SchemaURI = "https://json-schema.org/draft/2020-12/schema"
)

// MustCompileJSONSchema compiles an embedded schema during package
// initialization. Embedded source-controlled schemas are expected to be valid.
func MustCompileJSONSchema(raw []byte) *jsonschema.Schema {
	compiled, err := CompileJSONSchema(raw)
	if err != nil {
		panic(fmt.Sprintf("compile embedded JSON Schema: %v", err))
	}
	return compiled
}

// CompileJSONSchema compiles one JSON Schema.
//
// Published schemas normally provide both $schema and $id. Schema-valued
// declaration fields may instead contain a valid JSON Schema fragment such as
// {"type":"object"} or {"const":true}; those receive private Draft 2020-12
// metadata before compilation.
func CompileJSONSchema(
	raw []byte,
) (*jsonschema.Schema, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil, fmt.Errorf("%w: JSON Schema is empty", errInvalid)
	}

	var header struct {
		Schema string `json:"$schema"`
		ID     string `json:"$id"`
	}

	objectSchema := raw[0] == '{'
	switch {
	case objectSchema:
		if err := json.Unmarshal(raw, &header); err != nil {
			return nil, fmt.Errorf(
				"decode JSON Schema header: %w",
				err,
			)
		}

	case bytes.Equal(raw, []byte("true")),
		bytes.Equal(raw, []byte("false")):
		// Boolean schemas are valid JSON Schemas and need a wrapper because
		// they cannot carry $schema or $id members directly.

	default:
		return nil, fmt.Errorf(
			"%w: JSON Schema must be an object or boolean",
			errInvalid,
		)
	}

	var resourceRaw []byte
	resourceID := header.ID
	if objectSchema {
		var object map[string]json.RawMessage
		if err := json.Unmarshal(raw, &object); err != nil {
			return nil, fmt.Errorf(
				"decode JSON Schema object: %w",
				err,
			)
		}
		if header.Schema == "" {
			value, err := json.Marshal(draft202012SchemaURI)
			if err != nil {
				return nil, err
			}
			object["$schema"] = value
		}
		if resourceID == "" {
			value, err := json.Marshal(anonymousSchemaID)
			if err != nil {
				return nil, err
			}
			object["$id"] = value
			resourceID = anonymousSchemaID
		}
		var err error
		resourceRaw, err = json.Marshal(object)
		if err != nil {
			return nil, fmt.Errorf("encode JSON Schema object: %w", err)
		}
	} else {
		resourceID = anonymousSchemaID
		wrapped, err := json.Marshal(struct {
			Schema string            `json:"$schema"`
			ID     string            `json:"$id"`
			AllOf  []json.RawMessage `json:"allOf"`
		}{
			Schema: draft202012SchemaURI,
			ID:     resourceID,
			AllOf: []json.RawMessage{
				json.RawMessage(append([]byte(nil), raw...)),
			},
		})
		if err != nil {
			return nil, fmt.Errorf("wrap boolean JSON Schema: %w", err)
		}
		resourceRaw = wrapped
	}

	compiler := jsonschema.NewCompiler()

	resource, err := jsonschema.UnmarshalJSON(
		bytes.NewReader(resourceRaw),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"decode JSON Schema resource: %w",
			err,
		)
	}
	if err := compiler.AddResource(resourceID, resource); err != nil {
		return nil, fmt.Errorf(
			"%w: register JSON Schema resource %q: %w",
			errInvalid,
			resourceID,
			err,
		)
	}

	compiled, err := compiler.Compile(resourceID)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: compile JSON Schema resource %q: %w",
			errInvalid,
			resourceID,
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
