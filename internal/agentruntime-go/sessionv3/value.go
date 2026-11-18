package sessionv3

import (
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

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
