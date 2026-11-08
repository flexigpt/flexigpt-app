package sessionv3

import (
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type CanonicalValue = jsonutil.CanonicalValue

func MustCanonicalValue(value any) CanonicalValue {
	return jsonutil.MustCanonicalValue(value)
}

func DecodeCanonicalValue[T any](value CanonicalValue) (T, error) {
	return jsonutil.DecodeCanonicalValue[T](value)
}

func CanonicalValueFrom(raw []byte) (CanonicalValue, error) {
	return jsonutil.CanonicalValueFrom(raw)
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
