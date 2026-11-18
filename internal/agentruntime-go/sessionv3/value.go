package sessionv3

import (
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// CanonicalValue is the Session V3 durable canonical JSON value. It is a
// named domain type, not an alias of an infrastructure type.
type CanonicalValue jsonutil.CanonicalValue

func MustCanonicalValue(value any) CanonicalValue {
	return CanonicalValue(jsonutil.MustCanonicalValue(value))
}

func (v CanonicalValue) Clone() CanonicalValue {
	return CanonicalValue(jsonutil.CanonicalValue(v).Clone())
}

func (v CanonicalValue) Valid() bool {
	return jsonutil.CanonicalValue(v).Valid()
}

func (v CanonicalValue) IsZero() bool {
	return len(v) == 0
}

func (v CanonicalValue) MarshalJSON() ([]byte, error) {
	if len(v) == 0 {
		return []byte("null"), nil
	}
	return append([]byte(nil), v...), nil
}

func (v *CanonicalValue) UnmarshalJSON(raw []byte) error {
	value, err := jsonutil.CanonicalValueFrom(raw)
	if err != nil {
		return err
	}
	*v = CanonicalValue(value)
	return nil
}

func DecodeCanonicalValue[T any](value CanonicalValue) (T, error) {
	return jsonutil.DecodeCanonicalValue[T](jsonutil.CanonicalValue(value))
}

func CanonicalValueFrom(raw []byte) (CanonicalValue, error) {
	value, err := jsonutil.CanonicalValueFrom(raw)
	return CanonicalValue(value), err
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

func cloneSessionTombstone(
	value SessionTombstone,
) SessionTombstone {
	return jsonutil.CloneJSONInValue(value)
}

func cloneFrame(value ProgramFrame) ProgramFrame {
	return jsonutil.CloneJSONInValue(value)
}
