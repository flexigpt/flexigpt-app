package sessionv3runtime

import (
	"context"
	"errors"
	"fmt"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
)

// boundTool is process-private setup routing state. Session Runtime persists
// only the semantic binding returned by the owning domain executor.
type boundTool struct {
	Call          sessionv3Runtime.ToolCall
	DomainPrivate any
}

// boundToolWait retains a domain-owned wait resolver value. It is never
// serialized and is discarded during Session Runtime recovery.
type boundToolWait struct {
	Call          sessionv3Runtime.ToolCall
	DomainPrivate any
}

func (b *Binder) bindTool(
	ctx context.Context,
	req sessionv3Runtime.WorkDispatchRequest,
) (sessionv3Runtime.BindResult, error) {
	payload := req.WorkItem.Input.Tool
	if payload == nil {
		return sessionv3Runtime.BindResult{}, errors.New("tool payload is missing")
	}
	if err := rejectUnsupportedWorkflowToolTransport(
		req.WorkItem,
		payload,
	); err != nil {
		return sessionv3Runtime.BindResult{}, err
	}

	call := payload.Call
	if call.Binding.Kind == sessionv3Runtime.ToolBindingKindUnresolved {
		switch {
		case payload.MCP != nil:
			if b.mcpTools == nil {
				return sessionv3Runtime.BindResult{}, errors.New(
					"session v3 MCP Tool runtime is unavailable",
				)
			}
			value, err := b.mcpTools.BindWorkflowTool(ctx, payload)
			if err != nil {
				return sessionv3Runtime.BindResult{}, err
			}
			call = value

		case payload.Target != nil:
			if b.goTools == nil {
				return sessionv3Runtime.BindResult{}, errors.New(
					"session v3 Tool runtime is unavailable",
				)
			}
			value, err := b.goTools.BindWorkflowTool(ctx, payload)
			if err != nil {
				return sessionv3Runtime.BindResult{}, err
			}
			call = value

		default:
			return sessionv3Runtime.BindResult{}, fmt.Errorf(
				"unresolved Tool WorkItem %q has no typed target",
				req.WorkItem.ID,
			)
		}
	}
	if err := call.Validate(); err != nil {
		return sessionv3Runtime.BindResult{}, err
	}
	if call.Binding.Kind == sessionv3Runtime.ToolBindingKindMCP {
		if call.Binding.MCP == nil ||
			call.Binding.MCP.ServerArtifact == nil {
			return sessionv3Runtime.BindResult{}, errors.New(
				"mcp tool binding has no server artifact pin",
			)
		}
		if err := b.verifyArtifactPin(
			ctx,
			*call.Binding.MCP.ServerArtifact,
		); err != nil {
			return sessionv3Runtime.BindResult{}, err
		}
	}

	effectiveRequest, err := requestWithBoundTool(req, call)
	if err != nil {
		return sessionv3Runtime.BindResult{}, err
	}

	switch call.Binding.Kind {
	case sessionv3Runtime.ToolBindingKindGo:
		if b.goTools == nil {
			return sessionv3Runtime.BindResult{}, errors.New(
				"session v3 Tool runtime is unavailable",
			)
		}
		semantic, err := b.goTools.BindGo(ctx, effectiveRequest.WorkItem.Input.Tool)
		if err != nil {
			return sessionv3Runtime.BindResult{}, err
		}
		return sessionv3Runtime.BindResult{
			SemanticBinding: semantic,
			Private:         boundTool{Call: call},
		}, nil

	case sessionv3Runtime.ToolBindingKindMCP:
		if b.mcpTools == nil {
			return sessionv3Runtime.BindResult{}, errors.New(
				"session v3 MCP Tool runtime is unavailable",
			)
		}
		result, handled, err := b.mcpTools.Bind(ctx, effectiveRequest)
		if err != nil {
			return sessionv3Runtime.BindResult{}, err
		}
		if !handled {
			return sessionv3Runtime.BindResult{}, errors.New(
				"MCP Tool runtime declined an MCP Tool binding",
			)
		}
		return wrapMCPBindResult(result, call)

	case sessionv3Runtime.ToolBindingKindSkill:
		if b.skillTools == nil {
			return sessionv3Runtime.BindResult{}, errors.New(
				"session v3 Skill runtime is unavailable",
			)
		}
		semantic, err := b.skillTools.Bind(
			ctx,
			effectiveRequest,
			effectiveRequest.WorkItem.Input.Tool,
		)
		if err != nil {
			return sessionv3Runtime.BindResult{}, err
		}
		return sessionv3Runtime.BindResult{
			SemanticBinding: semantic,
			Private:         boundTool{Call: call},
		}, nil

	case sessionv3Runtime.ToolBindingKindProvider:
		return sessionv3Runtime.BindResult{}, fmt.Errorf(
			"provider-managed Tool %q requires a typed Model execution plan",
			call.Name,
		)

	default:
		return sessionv3Runtime.BindResult{}, fmt.Errorf(
			"unsupported Tool binding kind %q",
			call.Binding.Kind,
		)
	}
}

func (b *Binder) executeTool(
	ctx context.Context,
	req sessionv3Runtime.WorkDispatchRequest,
	bound sessionv3Runtime.BoundWork,
) (sessionv3Runtime.WorkExecutionResult, error) {
	private, found := bound.Private.(boundTool)
	if !found {
		return sessionv3Runtime.WorkExecutionResult{}, errors.New(
			"tool workItem has no typed domain binding",
		)
	}

	effectiveRequest, err := requestWithBoundTool(req, private.Call)
	if err != nil {
		return sessionv3Runtime.WorkExecutionResult{}, err
	}
	payload := effectiveRequest.WorkItem.Input.Tool

	switch private.Call.Binding.Kind {
	case sessionv3Runtime.ToolBindingKindGo:
		if b.goTools == nil {
			return sessionv3Runtime.WorkExecutionResult{}, errors.New(
				"session v3 Tool runtime is unavailable",
			)
		}
		result, executeErr := b.goTools.ExecuteGo(ctx, payload)
		if executeErr != nil {
			return sessionv3Runtime.WorkExecutionResult{}, executeErr
		}
		return result, nil

	case sessionv3Runtime.ToolBindingKindMCP:
		if b.mcpTools == nil {
			return sessionv3Runtime.WorkExecutionResult{}, errors.New(
				"session v3 MCP Tool runtime is unavailable",
			)
		}
		result, handled, err := b.mcpTools.Execute(
			ctx,
			effectiveRequest,
			sessionv3Runtime.BoundWork{
				SemanticBinding: bound.SemanticBinding,
				Private:         private.DomainPrivate,
			},
		)
		if err != nil {
			return sessionv3Runtime.WorkExecutionResult{}, err
		}
		if !handled {
			return sessionv3Runtime.WorkExecutionResult{}, errors.New(
				"MCP Tool runtime declined an MCP Tool execution",
			)
		}
		return result, nil

	case sessionv3Runtime.ToolBindingKindSkill:
		if b.skillTools == nil {
			return sessionv3Runtime.WorkExecutionResult{}, errors.New(
				"session v3 Skill runtime is unavailable",
			)
		}
		result, handled, err := b.skillTools.Execute(
			ctx,
			effectiveRequest,
			payload,
		)
		if err != nil {
			return sessionv3Runtime.WorkExecutionResult{}, err
		}
		if !handled {
			return sessionv3Runtime.WorkExecutionResult{}, errors.New(
				"skill runtime declined a Skill Tool execution",
			)
		}
		return result, nil

	case sessionv3Runtime.ToolBindingKindProvider:
		return sessionv3Runtime.WorkExecutionResult{}, errors.New(
			"provider-managed Tool cannot execute as a local WorkItem",
		)

	default:
		return sessionv3Runtime.WorkExecutionResult{}, fmt.Errorf(
			"unsupported Tool binding kind %q",
			private.Call.Binding.Kind,
		)
	}
}

func rejectUnsupportedWorkflowToolTransport(
	item sessionv3Runtime.WorkItem,
	payload *sessionv3Runtime.ToolWork,
) error {
	if payload == nil {
		return errors.New("tool payload is missing")
	}
	if len(payload.Context) != 0 {
		return errors.New("workflow tool context transport is unavailable")
	}
	if len(item.PortInputs) != 0 {
		return errors.New("workflow tool input port transport is unavailable")
	}
	for _, plan := range item.Ports {
		if plan.Direction == sessionv3Runtime.WorkflowPortDirectionOutput &&
			plan.Stream {
			return errors.New("workflow tool streamed port transport is unavailable")
		}
	}
	return nil
}

func (b *Binder) resolveToolWait(
	ctx context.Context,
	req sessionv3Runtime.WaitResolutionRequest,
) (sessionv3Runtime.WaitResolutionResult, error) {
	private, found := req.Private.(boundToolWait)
	if !found {
		return sessionv3Runtime.WaitResolutionResult{}, errors.New(
			"wait has no registered session v3 domain resolver",
		)
	}
	if b.mcpTools == nil {
		return sessionv3Runtime.WaitResolutionResult{}, errors.New(
			"session v3 MCP Tool runtime is unavailable",
		)
	}

	effectiveRequest, err := requestWithBoundTool(
		sessionv3Runtime.WorkDispatchRequest{
			Session:   req.Session,
			Execution: req.Execution,
			WorkItem:  req.WorkItem,
		},
		private.Call,
	)
	if err != nil {
		return sessionv3Runtime.WaitResolutionResult{}, err
	}

	domainRequest := req
	domainRequest.WorkItem = effectiveRequest.WorkItem
	domainRequest.Private = private.DomainPrivate
	result, handled, err := b.mcpTools.ResolveWait(ctx, domainRequest)
	if err != nil {
		return sessionv3Runtime.WaitResolutionResult{}, err
	}
	if !handled {
		return sessionv3Runtime.WaitResolutionResult{}, errors.New(
			"MCP Tool runtime declined an MCP approval resolution",
		)
	}
	if result.Resume {
		if result.Bound == nil || result.Bound.Private == nil {
			return sessionv3Runtime.WaitResolutionResult{}, errors.New(
				"MCP approval resolution returned no execution binding",
			)
		}
		result.Bound.Private = boundTool{
			Call:          private.Call,
			DomainPrivate: result.Bound.Private,
		}
	}
	return result, nil
}

func requestWithBoundTool(
	request sessionv3Runtime.WorkDispatchRequest,
	call sessionv3Runtime.ToolCall,
) (sessionv3Runtime.WorkDispatchRequest, error) {
	if request.WorkItem.Input.Tool == nil {
		return sessionv3Runtime.WorkDispatchRequest{}, errors.New(
			"tool workItem payload is missing",
		)
	}

	payload := *request.WorkItem.Input.Tool
	payload.Call = call
	input := request.WorkItem.Input
	input.Tool = &payload
	request.WorkItem.Input = input
	return request, nil
}

func wrapMCPBindResult(
	result sessionv3Runtime.BindResult,
	call sessionv3Runtime.ToolCall,
) (sessionv3Runtime.BindResult, error) {
	switch {
	case result.Wait != nil:
		if result.Private == nil {
			return sessionv3Runtime.BindResult{}, errors.New(
				"MCP Tool wait has no private resolver state",
			)
		}
		result.Private = boundToolWait{
			Call:          call,
			DomainPrivate: result.Private,
		}

	case result.ImmediateOutput != nil:
		result.Private = nil

	default:
		if result.Private == nil {
			return sessionv3Runtime.BindResult{}, errors.New(
				"MCP Tool binding has no private execution state",
			)
		}
		result.Private = boundTool{
			Call:          call,
			DomainPrivate: result.Private,
		}
	}
	return result, nil
}
