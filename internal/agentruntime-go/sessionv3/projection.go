package sessionv3

import (
	"time"
)

// SessionRuntimeProjection is the transport-neutral projection for normal
// Session behavior. It is derived only from the durable Session document.
type SessionRuntimeProjection struct {
	SchemaVersion   string    `json:"schemaVersion"`
	SessionID       SessionID `json:"sessionID"`
	SessionRevision Revision  `json:"sessionRevision"`
	Title           string    `json:"title,omitempty"`

	Activities      []Activity                 `json:"activities"`
	ModelOperations []ModelOperationProjection `json:"modelOperations"`
	ToolCalls       []ToolCallProjection       `json:"toolCalls"`
	Attempts        []ToolAttemptProjection    `json:"attempts"`
	Waits           []WaitProjection           `json:"waits"`
	Apps            []AppInstance              `json:"apps"`
	Usage           []ExecutionUsageProjection `json:"usage"`
	Protocol        []ProtocolOutputProjection `json:"protocol"`
	Debugger        []DebuggerProjection       `json:"debugger"`
}

type ModelOperationProjection struct {
	ExecutionID ExecutionID     `json:"executionID"`
	FrameID     ProgramFrameID  `json:"frameID"`
	WorkItemID  WorkItemID      `json:"workItemID"`
	State       WorkItemState   `json:"state"`
	StartedAt   *time.Time      `json:"startedAt,omitempty"`
	CompletedAt *time.Time      `json:"completedAt,omitempty"`
	Stream      *LLMStreamState `json:"stream,omitempty"`
	Output      *LLMOutput      `json:"output,omitempty"`
	Error       *SanitizedError `json:"error,omitempty"`
}

type ToolCallProjection struct {
	ExecutionID ExecutionID     `json:"executionID"`
	FrameID     ProgramFrameID  `json:"frameID"`
	WorkItemID  WorkItemID      `json:"workItemID"`
	State       WorkItemState   `json:"state"`
	Call        ToolCall        `json:"call"`
	Output      *ToolOutput     `json:"output,omitempty"`
	Effect      EffectState     `json:"effectState,omitempty"`
	Error       *SanitizedError `json:"error,omitempty"`
}

type ToolAttemptProjection struct {
	ExecutionID ExecutionID `json:"executionID"`
	Attempt     Attempt     `json:"attempt"`
}

type WaitProjection struct {
	ExecutionID ExecutionID `json:"executionID"`
	Wait        Wait        `json:"wait"`
}

type ExecutionUsageProjection struct {
	ExecutionID ExecutionID          `json:"executionID"`
	Usage       ExecutionBudgetUsage `json:"usage"`
}

type ProtocolOutputProjection struct {
	ExecutionID ExecutionID      `json:"executionID"`
	WorkItemID  WorkItemID       `json:"workItemID"`
	Outputs     []CanonicalValue `json:"outputs,omitempty"`
}

type DebuggerProjection struct {
	ExecutionID ExecutionID    `json:"executionID"`
	WorkItemID  WorkItemID     `json:"workItemID"`
	Details     CanonicalValue `json:"details,omitempty"`
}

// BuildSessionRuntimeProjection builds a stable projection from a durable
// Session snapshot. It does not inspect provider clients, private bindings,
// callbacks, or process-local App routes.
func BuildSessionRuntimeProjection(
	session Session,
) SessionRuntimeProjection {
	value := cloneSession(session)
	output := SessionRuntimeProjection{
		SchemaVersion:   SchemaVersion,
		SessionID:       value.ID,
		SessionRevision: value.Revision,
		Title:           value.Title,
		Activities:      append([]Activity(nil), value.Timeline...),
		ModelOperations: []ModelOperationProjection{},
		ToolCalls:       []ToolCallProjection{},
		Attempts:        []ToolAttemptProjection{},
		Waits:           []WaitProjection{},
		Apps:            []AppInstance{},
		Usage:           []ExecutionUsageProjection{},
		Protocol:        []ProtocolOutputProjection{},
		Debugger:        []DebuggerProjection{},
	}

	for _, execution := range value.Executions {
		output.Usage = append(output.Usage, ExecutionUsageProjection{
			ExecutionID: execution.ID,
			Usage:       execution.Usage,
		})
		for _, app := range execution.Apps {
			output.Apps = append(output.Apps, cloneAppInstance(app))
		}
		for _, attempt := range execution.Attempts {
			output.Attempts = append(output.Attempts, ToolAttemptProjection{
				ExecutionID: execution.ID,
				Attempt:     attempt,
			})
		}
		for _, wait := range execution.Waits {
			output.Waits = append(output.Waits, WaitProjection{
				ExecutionID: execution.ID,
				Wait:        wait,
			})
		}
		for _, item := range execution.WorkItems {
			switch item.Type {
			case WorkItemTypeLLM:
				var llm *LLMOutput
				if item.Output != nil && item.Output.LLM != nil {
					copyValue := *item.Output.LLM
					copyValue.ToolCalls = append(
						[]ToolCall(nil),
						item.Output.LLM.ToolCalls...,
					)
					copyValue.ProtocolOutputs = cloneCanonicalValues(
						item.Output.LLM.ProtocolOutputs,
					)
					copyValue.DebugDetails = item.Output.LLM.DebugDetails.Clone()
					llm = &copyValue
				}
				output.ModelOperations = append(
					output.ModelOperations,
					ModelOperationProjection{
						ExecutionID: execution.ID,
						FrameID:     item.FrameID,
						WorkItemID:  item.ID,
						State:       item.State,
						StartedAt:   cloneTime(item.DispatchedAt),
						CompletedAt: cloneTime(item.CompletedAt),
						Stream:      cloneProjectionStream(item.LLMStream),
						Output:      llm,
						Error:       cloneWorkflowSanitizedError(item.Error),
					},
				)
				if llm != nil && len(llm.ProtocolOutputs) != 0 {
					output.Protocol = append(output.Protocol, ProtocolOutputProjection{
						ExecutionID: execution.ID,
						WorkItemID:  item.ID,
						Outputs:     cloneCanonicalValues(llm.ProtocolOutputs),
					})
				}
				if llm != nil && len(llm.DebugDetails) != 0 {
					output.Debugger = append(output.Debugger, DebuggerProjection{
						ExecutionID: execution.ID,
						WorkItemID:  item.ID,
						Details:     llm.DebugDetails.Clone(),
					})
				}

			case WorkItemTypeTool:
				if item.Input.Tool == nil {
					continue
				}
				var tool *ToolOutput
				if item.Output != nil && item.Output.Tool != nil {
					copyValue := *item.Output.Tool
					copyValue.Structured = item.Output.Tool.Structured.Clone()
					tool = &copyValue
				}
				output.ToolCalls = append(output.ToolCalls, ToolCallProjection{
					ExecutionID: execution.ID,
					FrameID:     item.FrameID,
					WorkItemID:  item.ID,
					State:       item.State,
					Call:        cloneProjectionToolCall(item.Input.Tool.Call),
					Output:      tool,
					Effect:      item.EffectState,
					Error:       cloneWorkflowSanitizedError(item.Error),
				})
			default:
			}
		}
	}
	return output
}

// BuildWorkflowProjection derives Workflow activity, execution, and typed step
// outcome views only from a durable Session document.
func BuildWorkflowProjection(
	session Session,
) WorkflowProjection {
	value := cloneSession(session)
	output := WorkflowProjection{
		SchemaVersion:   SchemaVersion,
		SessionID:       value.ID,
		SessionRevision: value.Revision,
		Activities:      []Activity{},
		Executions:      []Execution{},
		StepOutcomes:    []WorkflowStepOutcomeProjection{},
	}
	for _, activity := range value.Timeline {
		switch activity.Kind {
		case ActivityKindWorkflow, ActivityKindAction, ActivityKindWait:
			output.Activities = append(output.Activities, activity)
		default:
		}
	}
	for _, execution := range value.Executions {
		workflowExecution := false
		for _, frame := range execution.ProgramFrames {
			if frame.Program.InterpreterID != WorkflowInterpreterID {
				continue
			}
			workflowExecution = true
			state, err := DecodeCanonicalValue[WorkflowState](frame.State.Data)
			if err != nil {
				continue
			}
			output.StepOutcomes = append(
				output.StepOutcomes,
				WorkflowStepOutcomeProjection{
					ExecutionID: execution.ID,
					FrameID:     frame.ID,
					Outcomes:    cloneWorkflowStepOutcomes(state.Outcomes),
				},
			)
		}
		if workflowExecution {
			output.Executions = append(output.Executions, execution)
		}
	}
	return output
}

func cloneCanonicalValues(values []CanonicalValue) []CanonicalValue {
	if len(values) == 0 {
		return nil
	}
	output := make([]CanonicalValue, len(values))
	for index := range values {
		output[index] = values[index].Clone()
	}
	return output
}

func cloneProjectionStream(value *LLMStreamState) *LLMStreamState {
	if value == nil {
		return nil
	}
	output := *value
	return &output
}

func cloneProjectionToolCall(value ToolCall) ToolCall {
	output := value
	output.Arguments = value.Arguments.Clone()
	output.Binding = cloneAppToolBinding(value.Binding)
	return output
}
