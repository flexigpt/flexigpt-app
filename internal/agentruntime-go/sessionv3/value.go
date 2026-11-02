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
		return nil, nil
	}
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
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	var output Session
	if err := json.Unmarshal(raw, &output); err != nil {
		panic(err)
	}
	return output
}

func cloneExecution(value Execution) Execution {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	var output Execution
	if err := json.Unmarshal(raw, &output); err != nil {
		panic(err)
	}
	return output
}

func cloneFrame(value ProgramFrame) ProgramFrame {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	var output ProgramFrame
	if err := json.Unmarshal(raw, &output); err != nil {
		panic(err)
	}
	return output
}
