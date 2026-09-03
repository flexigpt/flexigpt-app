package jsonutil

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

type JSONSchema = json.RawMessage

type JSONRawString string

func (value JSONRawString) MarshalJSON() ([]byte, error) {
	raw := bytes.TrimSpace([]byte(value))
	if len(raw) == 0 {
		return []byte("null"), nil
	}
	if !json.Valid(raw) {
		return nil, errors.New("invalid raw JSON")
	}
	return append([]byte(nil), raw...), nil
}

func (value *JSONRawString) UnmarshalJSON(raw []byte) error {
	if value == nil {
		return errors.New("nil raw JSON target")
	}
	if !json.Valid(raw) {
		return errors.New("invalid raw JSON")
	}
	*value = JSONRawString(append([]byte(nil), raw...))
	return nil
}

// EncodeToJSONRaw encodes any value to json.RawMessage.
// No typed method here as value being of a type doesnt really affect its functionality.
func EncodeToJSONRaw(value any) (json.RawMessage, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode JSON: %w", err)
	}
	return json.RawMessage(data), nil
}

// DecodeJSONRaw decodes a json.RawMessage into a typed value T, disallowing unknown fields and rejecting trailing data.
// If raw is empty, or only whitespace, it returns the zero value of T.
func DecodeJSONRaw[T any](raw json.RawMessage) (T, error) {
	var zero T
	if isBlankJSON(raw) {
		return zero, nil
	}

	var v T
	if err := decodeBytes(raw, &v, true, true); err != nil {
		return zero, err
	}
	return v, nil
}

func CloneJSONInValue[T any](in T) T {
	out, err := CloneJSON(in)
	if err != nil {
		return in
	}
	return out
}

// CloneJSON returns a copy of input produced by a JSON marshal/unmarshal
// round trip. It intentionally does not use DecodeJSONRaw, because clone
// semantics should match json.Unmarshal rather than reject unknown fields.
func CloneJSON[T any](input T) (T, error) {
	var output T

	raw, err := EncodeToJSONRaw(input)
	if err != nil {
		return output, err
	}
	if err := json.Unmarshal([]byte(raw), &output); err != nil {
		return output, fmt.Errorf("unmarshal JSON clone: %w", err)
	}
	return output, nil
}

// decodeBytes decodes JSON bytes into out with options:
// - disallowUnknown: Disallow unknown fields if true.
// - requireEOF: Reject trailing JSON after the first value if true.
func decodeBytes(data []byte, out any, disallowUnknown, requireEOF bool) error {
	dec := newDecoder(bytes.NewReader(data), disallowUnknown)
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}
	if requireEOF {
		if err := requireNoTrailing(dec); err != nil {
			return err
		}
	}
	return nil
}

func newDecoder(r io.Reader, disallowUnknown bool) *json.Decoder {
	dec := json.NewDecoder(r)
	if disallowUnknown {
		dec.DisallowUnknownFields()
	}
	return dec
}

// requireNoTrailing ensures there is no trailing data after the first JSON value.
func requireNoTrailing(dec *json.Decoder) error {
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("unexpected trailing data after JSON value")
		}
		return fmt.Errorf("trailing data validation: %w", err)
	}
	return nil
}

func isBlankJSON(b []byte) bool {
	return len(bytes.TrimSpace(b)) == 0
}
