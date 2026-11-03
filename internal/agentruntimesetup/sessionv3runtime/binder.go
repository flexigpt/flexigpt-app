package sessionv3runtime

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/inference"
	mcpConnection "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/connection"
	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/invocation"
	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	skillRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/skill"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/modelruntime"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/skillruntime"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/toolruntime"
	agentAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent"
	textAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/text"
	toolAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool"
	workspaceAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace"
)

type Dependencies struct {
	Providers     *inference.ProviderSetAPI
	Models        *modelruntime.RuntimeAdapter
	Agents        *agentAPI.Service
	Workspaces    *workspaceAPI.Service
	Texts         *textAPI.Service
	Tools         *toolAPI.Service
	ToolRuntime   *toolruntime.Adapter
	SkillResolver *skillruntime.RuntimeAdapter
	SkillRuntime  *skillRuntime.Service
	MCPRuntime    *mcpConnection.MCPRuntimeManager
	MCPBridge     *invocation.ToolBridge
}

type Binder struct {
	deps          Dependencies
	skills        *skillSessions
	deterministic sessionv3.DeterministicExecutor
}

func New(deps Dependencies) (*Binder, error) {
	if deps.Providers == nil || deps.Models == nil {
		return nil, errors.New("sessionv3 binder requires providers and models")
	}
	if deps.Tools == nil || deps.ToolRuntime == nil {
		return nil, errors.New("sessionv3 binder requires tool service and runtime")
	}
	if deps.SkillResolver == nil || deps.SkillRuntime == nil {
		return nil, errors.New("sessionv3 binder requires skill resolver and runtime")
	}
	if deps.MCPRuntime == nil || deps.MCPBridge == nil {
		return nil, errors.New("sessionv3 binder requires MCP runtime and bridge")
	}
	return &Binder{deps: deps, skills: newSkillSessions()}, nil
}

func (b *Binder) BindWork(ctx context.Context, req sessionv3.WorkDispatchRequest) (sessionv3.BindResult, error) {
	if result, handled, err := b.deterministic.Bind(ctx, req); handled || err != nil {
		return result, err
	}

	switch req.WorkItem.Type {
	case sessionv3.WorkItemTypeLLM:
		return b.bindLLM(ctx, req)
	case sessionv3.WorkItemTypeTool:
		return b.bindTool(ctx, req)
	default:
		return sessionv3.BindResult{}, fmt.Errorf(
			"unsupported sessionv3 WorkItem type %q",
			req.WorkItem.Type,
		)
	}
}

func (b *Binder) ExecuteWork(
	ctx context.Context,
	req sessionv3.WorkDispatchRequest,
	bound sessionv3.BoundWork,
) (sessionv3.WorkExecutionResult, error) {
	if result, handled, err := b.deterministic.Execute(ctx, req, bound); handled || err != nil {
		return result, err
	}

	if v, ok := bound.Private.(privateLLM); ok {
		return b.executeLLM(ctx, req, v)
	}
	if v, ok := bound.Private.(privateTool); ok {
		return b.executeTool(ctx, req, bound, v)
	}
	return sessionv3.WorkExecutionResult{}, fmt.Errorf("unknown bound work type %T", bound.Private)
}

func (b *Binder) ResolveWait(
	ctx context.Context,
	req sessionv3.WaitResolutionRequest,
) (sessionv3.WaitResolutionResult, error) {
	if result, handled, err := b.deterministic.ResolveWait(ctx, req); handled || err != nil {
		return result, err
	}

	if v, ok := req.Private.(privateToolWait); ok {
		return b.resolveToolWait(ctx, req, v)
	}
	return sessionv3.WaitResolutionResult{}, errors.New(
		"wait has no registered sessionv3 resolver",
	)
}

func (b *Binder) CloseExecutionBinding(_ context.Context, _ sessionv3.ExecutionID) error {
	return nil
}

func boundNow() time.Time {
	return time.Now().UTC()
}
