package sessionv3

import (
	"errors"
	"time"
)

type WorkflowChannelMessageSnapshot struct {
	ID          string         `json:"id"`
	Sequence    uint64         `json:"sequence"`
	Kind        string         `json:"kind,omitempty"`
	Data        CanonicalValue `json:"data"`
	Correlation CanonicalValue `json:"correlation,omitempty"`
	Time        time.Time      `json:"time"`
}

// captureWorkflowMessages records the authorized channel snapshot used by the
// current evaluation boundary. It does not consume messages or expose channels
// that the current Workflow path cannot read.
func captureWorkflowMessages(
	program WorkflowProgram,
	state *WorkflowState,
	execution Execution,
) error {
	if state == nil {
		return errors.New("workflow state is unavailable")
	}

	path := state.PendingPath
	if path == "" && state.Index < len(program.Root.Steps) {
		path = "step/" + program.Root.Steps[state.Index].ID
	}

	output := make(map[string][]WorkflowChannelMessageSnapshot)
	for _, declaration := range program.Channels {
		if !workflowChannelAllows(declaration.Readers, path) {
			continue
		}
		for _, channel := range execution.Channels {
			if channel.Name != declaration.Name {
				continue
			}
			values := make(
				[]WorkflowChannelMessageSnapshot,
				0,
				len(channel.Messages),
			)
			for _, message := range channel.Messages {
				values = append(values, WorkflowChannelMessageSnapshot{
					ID:          message.ID,
					Sequence:    message.Sequence,
					Kind:        message.Kind,
					Data:        message.Data.Clone(),
					Correlation: message.Correlation.Clone(),
					Time:        message.CreatedAt,
				})
			}
			output[declaration.Name] = values
			break
		}
	}
	value, err := canonicalWorkflowValue(output)
	if err != nil {
		return err
	}
	state.Messages = value
	return nil
}

// workflowActivationContext reserves context.time for the Runtime logical
// clock. Call context remains otherwise unchanged. This does not feed generic
// Workflow JSON into a model request; it is only CEL evaluation data.
func workflowActivationContext(
	value *CanonicalValue,
	logicalTime time.Time,
) (any, error) {
	contextValues := map[string]CanonicalValue{}
	if value != nil && len(*value) != 0 {
		decoded, err := DecodeCanonicalValue[map[string]CanonicalValue](*value)
		if err != nil {
			return nil, errors.New("workflow context must be an object")
		}
		for key, current := range decoded {
			contextValues[key] = current.Clone()
		}
	}
	contextValues["time"] = MustCanonicalValue(logicalTime.UTC().Format(time.RFC3339Nano))
	return workflowActivationValue(MustCanonicalValue(contextValues))
}
