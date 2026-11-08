package jsonutil

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

const MaxCanonicalValueBytes = 2 << 20

// CanonicalValue is a bounded canonical JSON value for actual arbitrary-JSON
// boundaries. Typed domain models must not use it as their primary model.
type CanonicalValue json.RawMessage

func CanonicalValueFrom(raw []byte) (CanonicalValue, error) {
	if len(raw) > MaxCanonicalValueBytes {
		return nil, fmt.Errorf(
			"canonical value exceeds %d bytes",
			MaxCanonicalValueBytes,
		)
	}

	canonical, err := Canonicalize(raw)
	if err != nil {
		return nil, err
	}
	return CanonicalValue(append([]byte(nil), canonical...)), nil
}

func MustCanonicalValue(value any) CanonicalValue {
	raw, err := EncodeToJSONRaw(value)
	if err != nil {
		panic(err)
	}

	canonical, err := CanonicalValueFrom(raw)
	if err != nil {
		panic(err)
	}
	return canonical
}

func (v CanonicalValue) Clone() CanonicalValue {
	if len(v) == 0 {
		return nil
	}
	return CanonicalValue(append([]byte(nil), v...))
}

func (v CanonicalValue) Valid() bool {
	if len(v) == 0 {
		return true
	}

	canonical, err := CanonicalValueFrom(v)
	return err == nil && bytes.Equal(v, canonical)
}

func (v CanonicalValue) MarshalJSON() ([]byte, error) {
	if len(v) == 0 {
		return []byte("null"), nil
	}
	return append([]byte(nil), v...), nil
}

func (v *CanonicalValue) UnmarshalJSON(raw []byte) error {
	if v == nil {
		return errors.New("canonical value target is nil")
	}

	value, err := CanonicalValueFrom(raw)
	if err != nil {
		return err
	}
	*v = value
	return nil
}

// DecodeCanonicalValue validates the canonical representation before decoding
// a typed value.
func DecodeCanonicalValue[T any](value CanonicalValue) (T, error) {
	var zero T
	if len(value) == 0 {
		return zero, nil
	}
	if !value.Valid() {
		return zero, errors.New("canonical value is not canonical JSON")
	}

	return DecodeJSONRaw[T](
		json.RawMessage(append([]byte(nil), value...)),
	)
}
