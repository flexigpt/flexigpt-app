package main

import (
	"context"
	"errors"

	inferenceRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/inference"
	mcpConnection "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/connection"
	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/invocation"
	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	skillRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/skill"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/modelruntime"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/sessionv3runtime"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/skillruntime"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/toolruntime"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/workflowruntime"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	toolAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool"
	workflowAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow"
	"github.com/flexigpt/flexigpt-app/internal/sessionstorev3"
)

type SessionV3Params struct {
	Providers        *inferenceRuntime.ProviderSetAPI
	Models           *modelruntime.SourceResolver
	Tools            *toolAPI.Service
	ToolCaller       toolruntime.GoToolCaller
	SkillResolver    *skillruntime.Resolver
	SkillRuntime     *skillRuntime.Service
	MCPRuntime       *mcpConnection.MCPRuntimeManager
	MCPBridge        *invocation.ToolBridge
	Events           sessionv3Runtime.EventPublisher
	WorkflowBindings *workflowruntime.BindingRegistry
	Workflow         *workflowruntime.Admission
}

var (
	errSessionV3Deps        = errors.New("session v3 wrapper dependencies are incomplete")
	errSessionV3AlreadyInit = errors.New("session v3 wrapper is already initialized")
	errSessionV3Closed      = errors.New("session v3 wrapper is closed")
)

func withSessionV3[T any](wrapper *SessionV3Wrapper, fn func(*sessionv3Runtime.Runtime) (T, error)) (T, error) {
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

func newSessionV3WorkflowAdmission(
	artifacts artifact.API,
	resolver *composition.Resolver,
) (
	*workflowruntime.Admission,
	*workflowruntime.BindingRegistry,
	error,
) {
	if artifacts == nil || resolver == nil {
		return nil, nil, errSessionV3Deps
	}

	workflows, err := workflowAPI.NewService(resolver)
	if err != nil {
		return nil, nil, err
	}

	bindings := workflowruntime.NewBindingRegistry()
	admission, err := workflowruntime.NewAdmission(
		workflows,
		artifacts,
		bindings,
	)
	if err != nil {
		return nil, nil, err
	}
	return admission, bindings, nil
}

func InitComposedSessionV3Runtime(
	ctx context.Context,
	wrapper *SessionV3Wrapper,
	store *sessionstorev3.Store,
	params SessionV3Params,
) error {
	if wrapper == nil || store == nil ||
		params.Workflow == nil ||
		params.WorkflowBindings == nil {
		return errSessionV3Deps
	}
	base, err := sessionv3runtime.New(
		sessionv3runtime.Dependencies{
			Providers:     params.Providers,
			Models:        params.Models,
			Tools:         params.Tools,
			ToolCaller:    params.ToolCaller,
			SkillResolver: params.SkillResolver,
			SkillRuntime:  params.SkillRuntime,
			MCPRuntime:    params.MCPRuntime,
			MCPBridge:     params.MCPBridge,
		},
	)
	if err != nil {
		return err
	}
	composed, err := workflowruntime.NewBinder(
		base,
		params.WorkflowBindings,
		params.Workflow,
	)
	if err != nil {
		return err
	}

	events := params.Events
	if events == nil {
		events = wrapper
	}
	dependencies, err := workflowruntime.ConfigureSessionV3Runtime(
		sessionv3Runtime.RuntimeDependencies{
			Binder:      composed,
			Checkpoints: store,
			Recovery:    store,
			Events:      events,
		},
		params.Workflow,
		composed,
	)
	if err != nil {
		return err
	}

	runtime, err := sessionv3Runtime.NewRuntime(dependencies)
	if err != nil {
		return err
	}
	if err := runtime.Restore(ctx); err != nil {
		runtime.Close()
		return err
	}
	return InitSessionV3Wrapper(wrapper, runtime, store)
}
