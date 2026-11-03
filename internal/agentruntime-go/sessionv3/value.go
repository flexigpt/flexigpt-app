package sessionv3

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const maxCanonicalValueBytes = 2 << 20

type CanonicalValue json.RawMessage

func CanonicalValueFrom(raw []byte) (CanonicalValue, error) {
	if len(raw) > maxCanonicalValueBytes {
		return nil, fmt.Errorf("canonical value exceeds %d bytes", maxCanonicalValueBytes)
	}
	canonical, err := jsonutil.Canonicalize(raw)
	if err != nil {
		return nil, err
	}
	return CanonicalValue(append([]byte(nil), canonical...)), nil
}

func MustCanonicalValue(value any) CanonicalValue {
	raw, err := jsonutil.EncodeToJSONRaw(value)
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
	if err != nil {
		return false
	}
	return string(canonical) == string(v)
}

// DecodeCanonicalValue verifies canonical representation before decoding a
// typed value. It is the only decoding helper for durable v3 JSON values.
func DecodeCanonicalValue[T any](value CanonicalValue) (T, error) {
	var zero T
	if len(value) == 0 {
		return zero, nil
	}
	if !value.Valid() {
		return zero, errors.New("canonical value is not canonical JSON")
	}

	decoded, err := jsonutil.DecodeJSONRaw[T](
		json.RawMessage(append([]byte(nil), value...)),
	)
	if err != nil {
		return zero, err
	}
	return decoded, nil
}

func cloneSession(value Session) Session {
	return jsonutil.CloneJSONInValue(value)
}

func cloneSessionDefaults(value SessionDefaults) SessionDefaults {
	return jsonutil.CloneJSONInValue(value)
}

func cloneExecution(value Execution) Execution {
	return jsonutil.CloneJSONInValue(value)
}

func cloneFrame(value ProgramFrame) ProgramFrame {
	return jsonutil.CloneJSONInValue(value)
}
