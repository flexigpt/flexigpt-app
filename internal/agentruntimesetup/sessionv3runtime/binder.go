package sessionv3runtime

import (
	"context"
	"errors"
	"fmt"

	inferenceRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/inference"
	mcpConnection "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/connection"
	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/invocation"
	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	skillRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/skill"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/mcpruntime"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/modelruntime"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/skillruntime"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/toolruntime"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	agentAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent"
	textAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/text"
	toolAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool"
	workspaceAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace"
)

type Dependencies struct {
	Artifacts     artifact.API
	Providers     *inferenceRuntime.ProviderSetAPI
	Models        *modelruntime.SourceResolver
	Agents        *agentAPI.Service
	Workspaces    *workspaceAPI.Service
	Texts         *textAPI.Service
	Tools         *toolAPI.Service
	ToolCaller    toolruntime.GoToolCaller
	SkillResolver *skillruntime.Resolver
	SkillRuntime  *skillRuntime.Service
	MCPRuntime    *mcpConnection.MCPRuntimeManager
	MCPBridge     *invocation.ToolBridge
}

type Binder struct {
	deps Dependencies

	goTools     *toolruntime.V3Executor
	mcpTools    *mcpruntime.V3Executor
	skillTools  *skillruntime.V3Executor
	attachments *V3AttachmentAdmitter

	deterministic sessionv3Runtime.DeterministicExecutor
}

func New(deps Dependencies) (*Binder, error) {
	if deps.Providers == nil || deps.Models == nil {
		return nil, errors.New("sessionv3 binder requires providers and models")
	}
	if deps.Tools == nil || deps.ToolCaller == nil {
		return nil, errors.New("sessionv3 binder requires tool service and runtime")
	}
	if deps.SkillResolver == nil || deps.SkillRuntime == nil {
		return nil, errors.New("sessionv3 binder requires skill resolver and runtime")
	}
	if deps.MCPRuntime == nil || deps.MCPBridge == nil {
		return nil, errors.New("sessionv3 binder requires MCP runtime and bridge")
	}

	if deps.Artifacts == nil {
		return nil, errors.New("sessionv3 binder requires artifact runtime")
	}

	goTools, err := toolruntime.NewV3Executor(
		deps.Tools,
		deps.ToolCaller,
	)
	if err != nil {
		return nil, err
	}
	mcpTools, err := mcpruntime.NewV3Executor(
		deps.MCPRuntime,
		deps.MCPBridge,
	)
	if err != nil {
		return nil, err
	}
	skillTools, err := skillruntime.NewV3Executor(
		deps.SkillResolver,
		deps.SkillRuntime,
	)
	if err != nil {
		return nil, err
	}

	attachments := NewV3AttachmentAdmitter()

	return &Binder{
		deps:        deps,
		goTools:     goTools,
		mcpTools:    mcpTools,
		skillTools:  skillTools,
		attachments: attachments,
	}, nil
}

func (b *Binder) BindWork(
	ctx context.Context,
	req sessionv3Runtime.WorkDispatchRequest,
) (sessionv3Runtime.BindResult, error) {
	if result, handled, err := b.deterministic.Bind(ctx, req); handled || err != nil {
		return result, err
	}

	switch req.WorkItem.Type {
	case sessionv3Runtime.WorkItemTypeLLM:
		return b.bindLLM(ctx, req)
	case sessionv3Runtime.WorkItemTypeTool:
		return b.bindTool(ctx, req)
	default:
		return sessionv3Runtime.BindResult{}, fmt.Errorf(
			"unsupported sessionv3 WorkItem type %q",
			req.WorkItem.Type,
		)
	}
}

func (b *Binder) ExecuteWork(
	ctx context.Context,
	req sessionv3Runtime.WorkDispatchRequest,
	bound sessionv3Runtime.BoundWork,
) (sessionv3Runtime.WorkExecutionResult, error) {
	if result, handled, err := b.deterministic.Execute(ctx, req, bound); handled || err != nil {
		return result, err
	}

	if v, ok := bound.Private.(privateLLM); ok {
		return b.executeLLM(ctx, req, v)
	}
	if req.WorkItem.Type == sessionv3Runtime.WorkItemTypeTool {
		return b.executeTool(ctx, req, bound)
	}
	return sessionv3Runtime.WorkExecutionResult{}, fmt.Errorf(
		"unknown bound work type %T",
		bound.Private,
	)
}

func (b *Binder) ResolveWait(
	ctx context.Context,
	req sessionv3Runtime.WaitResolutionRequest,
) (sessionv3Runtime.WaitResolutionResult, error) {
	if result, handled, err := b.deterministic.ResolveWait(ctx, req); handled || err != nil {
		return result, err
	}

	return b.resolveToolWait(ctx, req)
}

func (b *Binder) CloseExecutionBinding(
	ctx context.Context,
	executionID sessionv3Runtime.ExecutionID,
) error {
	if b == nil || b.skillTools == nil {
		return nil
	}
	return b.skillTools.CloseExecution(ctx, executionID)
}

func (b *Binder) AdmitAttachments(
	ctx context.Context,
	request sessionv3Runtime.AttachmentAdmissionRequest,
) ([]sessionv3Runtime.Attachment, error) {
	if b == nil || b.attachments == nil {
		return nil, errors.New("session v3 attachment runtime is unavailable")
	}
	return b.attachments.AdmitAttachments(ctx, request)
}

func (b *Binder) ValidatePortPlans(
	_ context.Context,
	request sessionv3Runtime.WorkDispatchRequest,
) error {
	if len(request.WorkItem.Ports) == 0 {
		return nil
	}

	switch request.WorkItem.Type {
	case sessionv3Runtime.WorkItemTypeLLM,
		sessionv3Runtime.WorkItemTypeService,
		sessionv3Runtime.WorkItemTypeActivity:
		return nil
	case sessionv3Runtime.WorkItemTypeTool:
		return validateV3ToolPortPlans(request.WorkItem.Ports)
	default:
		return errors.New(
			"port adaptation is unavailable for this work item",
		)
	}
}

func validateV3ToolPortPlans(
	plans []sessionv3Runtime.PortPlan,
) error {
	for _, plan := range plans {
		switch plan.Direction {
		case sessionv3Runtime.WorkflowPortDirectionInput:
			return errors.New(
				"tool input port adaptation is unavailable",
			)

		case sessionv3Runtime.WorkflowPortDirectionOutput:
			continue

		default:
			return errors.New("tool port direction is invalid")
		}
	}
	return nil
}
