package workflowruntime

import (
	"errors"
	"fmt"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
)

// ConfigureSessionV3Runtime installs the current typed Workflow interpreter,
// admission boundary, and Workflow binder into Session Runtime v3.
//
// The caller supplies an already-constructed domain binder as base. This
// function does not create Artifact services, mutate Session state, or start
// the Runtime. Bootstrap remains responsible for calling sessionv3.NewRuntime
// with the returned dependency value.
func ConfigureSessionV3Runtime(
	dependencies sessionv3Runtime.RuntimeDependencies,
	admission *Admission,
	binder *Binder,
) (sessionv3Runtime.RuntimeDependencies, error) {
	if admission == nil || binder == nil {
		return sessionv3Runtime.RuntimeDependencies{}, errors.New(
			"workflow Runtime v3 admission and binder are required",
		)
	}

	interpreter, err := NewInterpreter()
	if err != nil {
		return sessionv3Runtime.RuntimeDependencies{}, err
	}

	if len(dependencies.Interpreters) == 0 {
		dependencies.Interpreters = sessionv3Runtime.NewBuiltinInterpreters()
	}
	for _, current := range dependencies.Interpreters {
		if current == nil {
			continue
		}
		if current.ID() == interpreter.ID() &&
			current.Version() == interpreter.Version() {
			return sessionv3Runtime.RuntimeDependencies{}, fmt.Errorf(
				"workflow interpreter %q is already registered",
				sessionv3Runtime.ProgramInterpreterKey(
					interpreter.ID(),
					interpreter.Version(),
				),
			)
		}
	}

	dependencies.Binder = binder
	dependencies.Workflow = admission
	dependencies.Interpreters = append(
		dependencies.Interpreters,
		interpreter,
	)
	return dependencies, nil
}
