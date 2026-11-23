package workflowruntime

import (
	"errors"
	"fmt"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
)

// ConfigureSessionV3Runtime installs the typed Workflow interpreter and
// Workflow binder. Execution preparation is invoked by setup before Runtime
// receives a ProgramSnapshot.
//
// This function does not install a Workflow preparation port in Runtime v3.
func ConfigureSessionV3Runtime(
	dependencies sessionv3Runtime.RuntimeDependencies,
	binder *Binder,
) (sessionv3Runtime.RuntimeDependencies, error) {
	if binder == nil {
		return sessionv3Runtime.RuntimeDependencies{}, errors.New(
			"workflow runtime v3 binder is required",
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
	dependencies.Interpreters = append(
		dependencies.Interpreters,
		interpreter,
	)
	return dependencies, nil
}
