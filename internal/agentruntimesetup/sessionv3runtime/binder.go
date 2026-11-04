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
	toolAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool"
)

type Dependencies struct {
	Providers     *inferenceRuntime.ProviderSetAPI
	Models        *modelruntime.SourceResolver
	Tools         *toolAPI.Service
	ToolCaller    toolruntime.GoToolCaller
	SkillResolver *skillruntime.Resolver
	SkillRuntime  *skillRuntime.Service
	MCPRuntime    *mcpConnection.MCPRuntimeManager
	MCPBridge     *invocation.ToolBridge
}

type Binder struct {
	deps Dependencies

	goTools    *toolruntime.V3Executor
	mcpTools   *mcpruntime.V3Executor
	skillTools *skillruntime.V3Executor

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

	return &Binder{
		deps:       deps,
		goTools:    goTools,
		mcpTools:   mcpTools,
		skillTools: skillTools,
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
