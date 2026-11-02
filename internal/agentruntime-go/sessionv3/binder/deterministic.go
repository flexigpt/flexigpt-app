package binder

import (
	"time"

	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
)

type deterministicPrivate struct{}

func (b *Binder) bindDeterministic(req sessionv3.WorkDispatchRequest) (sessionv3.BindResult, error) {
	item := req.WorkItem
	semantic := sessionv3.SemanticBinding{
		Version:           "workflow-deterministic/v1",
		TargetFingerprint: string(item.Type) + ":" + item.ProgramPath,
		BoundAt:           time.Now().UTC(),
	}
	switch item.Type {
	case sessionv3.WorkItemTypeWait:
		if item.Input.Wait == nil {
			return sessionv3.BindResult{SemanticBinding: semantic, Private: deterministicPrivate{}}, nil
		}
		return sessionv3.BindResult{
			SemanticBinding: semantic,
			Wait:            &item.Input.Wait.Request,
			Private:         deterministicPrivate{},
		}, nil
	default:
		return sessionv3.BindResult{SemanticBinding: semantic, Private: deterministicPrivate{}}, nil
	}
}

func (b *Binder) executeDeterministic(req sessionv3.WorkDispatchRequest) (sessionv3.WorkExecutionResult, error) {
	item := req.WorkItem
	var value sessionv3.CanonicalValue
	switch item.Type {
	case sessionv3.WorkItemTypeTransform:
		value = item.Input.Transform.Input.Clone()
	case sessionv3.WorkItemTypeCondition:
		value = sessionv3.MustCanonicalValue(true)
	case sessionv3.WorkItemTypeStateUpdate:
		value = sessionv3.MustCanonicalValue(map[string]any{})
	case sessionv3.WorkItemTypeFork:
		value = sessionv3.MustCanonicalValue(item.Input.Fork)
	case sessionv3.WorkItemTypeJoin:
		value = sessionv3.MustCanonicalValue(item.Input.Join)
	case sessionv3.WorkItemTypeEmit:
		value = item.Input.Emit.Data.Clone()
	case sessionv3.WorkItemTypeCompensation:
		value = item.Input.Compensation.Input.Clone()
	default:
		value = sessionv3.MustCanonicalValue(map[string]any{})
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
	}, nil
}
