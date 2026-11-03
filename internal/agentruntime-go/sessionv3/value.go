package sessionv3

import (
	"encoding/json"
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
	raw, err := json.Marshal(value)
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

func cloneSession(value Session) Session {
	return jsonutil.CloneJSONInValue(value)
}

func cloneExecution(value Execution) Execution {
	return jsonutil.CloneJSONInValue(value)
}

func cloneFrame(value ProgramFrame) ProgramFrame {
	return jsonutil.CloneJSONInValue(value)
}
