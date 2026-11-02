package main

import (
	"context"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/inference"
	mcpConnection "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/connection"
	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/invocation"
	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3/binder"
	skillRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/skill"
	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/workflowruntime"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/modelruntime"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/skillruntime"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/toolruntime"
	agentAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent"
	textAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/text"
	toolAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool"
	workspaceAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace"
	"github.com/flexigpt/flexigpt-app/internal/sessionstorev3"
)

type SessionV3Params struct {
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
	Events        sessionv3.EventPublisher
	Workflow      sessionv3.WorkflowAdmission
}

var (
	errSessionV3Deps        = errors.New("session v3 wrapper dependencies are incomplete")
	errSessionV3AlreadyInit = errors.New("session v3 wrapper is already initialized")
	errSessionV3Closed      = errors.New("session v3 wrapper is closed")
)

func withSessionV3[T any](wrapper *SessionV3Wrapper, fn func(*sessionv3.Runtime) (T, error)) (T, error) {
	var zero T
	if wrapper == nil {
		return zero, errSessionV3Closed
	}
	wrapper.mu.RLock()
	runtime := wrapper.runtime
	wrapper.mu.RUnlock()
	if runtime == nil {
		return zero, errSessionV3Closed
	}
	return fn(runtime)
}

func InitComposedSessionV3Runtime(
	ctx context.Context,
	wrapper *SessionV3Wrapper,
	store *sessionstorev3.Store,
	params SessionV3Params,
) error {
	if wrapper == nil || store == nil {
		return errSessionV3Deps
	}
	base, err := binder.New(
		binder.Dependencies{
			Providers:     params.Providers,
			Models:        params.Models,
			Agents:        params.Agents,
			Workspaces:    params.Workspaces,
			Texts:         params.Texts,
			Tools:         params.Tools,
			ToolRuntime:   params.ToolRuntime,
			SkillResolver: params.SkillResolver,
			SkillRuntime:  params.SkillRuntime,
			MCPRuntime:    params.MCPRuntime,
			MCPBridge:     params.MCPBridge,
		},
	)
	if err != nil {
		return err
	}
	registry := workflowruntime.NewBindingRegistry()
	composed, err := workflowruntime.NewBinder(base, registry)
	if err != nil {
		return err
	}
	interpreters := sessionv3.NewBuiltinInterpreters()
	interpreters = append(interpreters, workflowruntime.NewInterpreter())
	runtime, err := sessionv3.NewRuntime(
		sessionv3.RuntimeDependencies{
			Binder:       composed,
			Checkpoints:  store,
			Recovery:     store,
			Events:       params.Events,
			Workflow:     params.Workflow,
			Interpreters: interpreters,
		},
	)
	if err != nil {
		return err
	}
	if err := runtime.Restore(ctx); err != nil {
		runtime.Close()
		return err
	}
	return InitSessionV3Wrapper(wrapper, runtime, store)
}
