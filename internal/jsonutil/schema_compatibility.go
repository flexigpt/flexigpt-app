package jsonutil

import (
	"bytes"
	"encoding/json"
	"errors"
)

const (
	jsonSchemaTypeArray   = "array"
	jsonSchemaTypeBoolean = "boolean"
	jsonSchemaTypeInteger = "integer"
	jsonSchemaTypeNull    = "null"
	jsonSchemaTypeNumber  = "number"
	jsonSchemaTypeObject  = "object"
	jsonSchemaTypeString  = "string"
)

// JSONSchemaTypesMayIntersect performs the intentionally limited compatibility
// check used by declaration and runtime boundaries. It proves only disjoint
// primitive JSON types. Full JSON Schema implication remains out of scope.
func JSONSchemaTypesMayIntersect(
	left []byte,
	right []byte,
) (bool, error) {
	leftTypes, leftImpossible, err := jsonSchemaTypes(left)
	if err != nil {
		return false, err
	}
	rightTypes, rightImpossible, err := jsonSchemaTypes(right)
	if err != nil {
		return false, err
	}
	if leftImpossible || rightImpossible {
		return false, nil
	}
	if len(leftTypes) == 0 || len(rightTypes) == 0 {
		return true, nil
	}
	for value := range leftTypes {
		if _, found := rightTypes[value]; found {
			return true, nil
		}
	}
	return false, nil
}

func jsonSchemaTypes(
	raw []byte,
) (t map[string]struct{}, i bool, err error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil, false, nil
	}

	var boolean bool
	if err := json.Unmarshal(raw, &boolean); err == nil {
		return nil, !boolean, nil
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return nil, false, errors.New(
			"json schema must be an object or boolean",
		)
	}

	typeValue, found := fields["type"]
	if !found {
		return nil, false, nil
	}

	output := make(map[string]struct{})
	var single string
	if err := json.Unmarshal(typeValue, &single); err == nil {
		if err := addJSONSchemaType(output, single); err != nil {
			return nil, false, err
		}
		return output, false, nil
	}

	var multiple []string
	if err := json.Unmarshal(typeValue, &multiple); err != nil || len(multiple) == 0 {
		return nil, false, errors.New("json schema type is invalid")
	}
	for _, value := range multiple {
		if err := addJSONSchemaType(output, value); err != nil {
			return nil, false, err
		}
	}
	return output, false, nil
}

func addJSONSchemaType(
	values map[string]struct{},
	value string,
) error {
	switch value {
	case jsonSchemaTypeArray,
		jsonSchemaTypeBoolean,
		jsonSchemaTypeNull,
		jsonSchemaTypeObject,
		jsonSchemaTypeString:
		values[value] = struct{}{}
	case jsonSchemaTypeInteger:
		values[jsonSchemaTypeInteger] = struct{}{}
		values[jsonSchemaTypeNumber] = struct{}{}
	case jsonSchemaTypeNumber:
		values[jsonSchemaTypeNumber] = struct{}{}
		values[jsonSchemaTypeInteger] = struct{}{}
	default:
		return errors.New("json schema type is invalid")
	}
	return nil
}
