package workflowruntime

import (
	"context"
	"encoding/json"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
)

// DeterministicExecutor executes semantic local Workflow WorkItems. It has no
// transport, credentials, native paths, client handles, callbacks, or tokens.
type DeterministicExecutor struct{}

type deterministicPrivate struct{}

func (DeterministicExecutor) Bind(
	_ context.Context,
	request sessionv3.WorkDispatchRequest,
) (sessionv3.BindResult, bool, error) {
	item := request.WorkItem
	binding := sessionv3.SemanticBinding{
		Version:           "workflow-deterministic/v1",
		TargetFingerprint: string(item.Type) + ":" + item.ProgramPath,
		BoundAt:           time.Now().UTC(),
	}
	switch item.Type {
	case sessionv3.WorkItemTypeTransform,
		sessionv3.WorkItemTypeCondition,
		sessionv3.WorkItemTypeStateUpdate,
		sessionv3.WorkItemTypeFork,
		sessionv3.WorkItemTypeJoin,
		sessionv3.WorkItemTypeEmit,
		sessionv3.WorkItemTypeCompensation:
		return sessionv3.BindResult{SemanticBinding: binding, Private: deterministicPrivate{}}, true, nil
	case sessionv3.WorkItemTypeWait:
		return sessionv3.BindResult{
			SemanticBinding: binding,
			Wait:            &item.Input.Wait.Request,
			Private:         deterministicPrivate{},
		}, true, nil
	default:
		return sessionv3.BindResult{}, false, nil
	}
}

func (DeterministicExecutor) Execute(
	_ context.Context,
	request sessionv3.WorkDispatchRequest,
	_ sessionv3.BoundWork,
) (sessionv3.WorkExecutionResult, bool, error) {
	item := request.WorkItem
	var value sessionv3.CanonicalValue
	switch item.Type {
	case sessionv3.WorkItemTypeTransform:
		value = item.Input.Transform.Input.Clone()
	case sessionv3.WorkItemTypeCondition:
		value = normalizeCondition(item.Input.Condition.Input)
	case sessionv3.WorkItemTypeStateUpdate:
		value = stateUpdateOutput(item.Input.StateUpdate)
	case sessionv3.WorkItemTypeFork:
		value = sessionv3.MustCanonicalValue(item.Input.Fork)
	case sessionv3.WorkItemTypeJoin:
		value = sessionv3.MustCanonicalValue(item.Input.Join)
	case sessionv3.WorkItemTypeEmit:
		value = item.Input.Emit.Data.Clone()
	case sessionv3.WorkItemTypeCompensation:
		value = item.Input.Compensation.Input.Clone()
	default:
		return sessionv3.WorkExecutionResult{}, false, nil
	}
	return sessionv3.WorkExecutionResult{
		Output: &sessionv3.WorkItemOutput{
			Value: value,
			Receipt: &sessionv3.ExternalReceipt{
				System:      "workflow-deterministic",
				Reference:   string(item.ID),
				ConfirmedAt: time.Now().UTC(),
			},
		},
		Confirmed: true,
	}, true, nil
}

func (DeterministicExecutor) ResolveWait(
	_ context.Context,
	_ sessionv3.WaitResolutionRequest,
) (sessionv3.WaitResolutionResult, bool, error) {
	return sessionv3.WaitResolutionResult{
		Resume: true,
		Bound: &sessionv3.BoundWork{
			SemanticBinding: sessionv3.SemanticBinding{
				Version:           "workflow-wait-resolution/v1",
				TargetFingerprint: "workflow-wait-resolution",
				BoundAt:           time.Now().UTC(),
			},
			Private: deterministicPrivate{},
		},
	}, true, nil
}

func normalizeCondition(value sessionv3.CanonicalValue) sessionv3.CanonicalValue {
	var boolean bool
	if err := json.Unmarshal(value, &boolean); err == nil {
		return sessionv3.MustCanonicalValue(boolean)
	}
	var wire struct {
		Value bool `json:"value"`
	}
	if err := json.Unmarshal(value, &wire); err == nil {
		return sessionv3.MustCanonicalValue(wire.Value)
	}
	return sessionv3.MustCanonicalValue(true)
}

func stateUpdateOutput(value *sessionv3.StateUpdateWork) sessionv3.CanonicalValue {
	if value == nil {
		return sessionv3.MustCanonicalValue(map[string]any{})
	}
	output := make(map[string]json.RawMessage, len(value.Assignments))
	for _, assignment := range value.Assignments {
		c := assignment.Value.Clone()
		output[assignment.Path] = json.RawMessage(c)
	}
	for _, path := range value.RemovePaths {
		delete(output, path)
	}
	return sessionv3.MustCanonicalValue(output)
}
