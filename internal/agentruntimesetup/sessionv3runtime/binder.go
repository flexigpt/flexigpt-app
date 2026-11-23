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
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/attachmentruntime"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/contributionruntime"
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

	goTools       *toolruntime.V3Executor
	mcpTools      *mcpruntime.V3Executor
	skillTools    *skillruntime.V3Executor
	attachments   *attachmentruntime.SessionV3Admitter
	contributions *contributionruntime.Preparer

	deterministic sessionv3Runtime.DeterministicExecutor
}

type binderArtifactPinVerifier struct {
	binder *Binder
}

func (v binderArtifactPinVerifier) VerifyArtifactPin(ctx context.Context, pin sessionv3Runtime.ArtifactPin) error {
	if v.binder == nil {
		return errors.New("session v3 binder is unavailable")
	}
	return contributionruntime.VerifyArtifactPin(ctx, v.binder.deps.Artifacts, pin)
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
		return nil, errors.New("sessionv3 binder requires mcp runtime and bridge")
	}

	if deps.Artifacts == nil {
		return nil, errors.New("sessionv3 binder requires artifact runtime")
	}

	output := &Binder{
		deps: deps,
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
		binderArtifactPinVerifier{binder: output},
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
	contributions, err := contributionruntime.New(
		deps.Artifacts,
		deps.Agents,
		deps.Workspaces,
		deps.Texts,
		contributionruntime.WithInstructionSkillRenderer(skillTools),
		contributionruntime.WithMCPContributionPreparer(mcpTools),
	)
	if err != nil {
		return nil, err
	}

	attachments := attachmentruntime.NewSessionV3Admitter()

	output.goTools = goTools
	output.mcpTools = mcpTools
	output.skillTools = skillTools
	output.attachments = attachments
	output.contributions = contributions
	return output, nil
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
			"unsupported sessionv3 work item type %q",
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

func (b *Binder) CloseProgramFrameBinding(
	ctx context.Context,
	executionID sessionv3Runtime.ExecutionID,
	frameID sessionv3Runtime.ProgramFrameID,
) error {
	if b == nil || b.skillTools == nil {
		return nil
	}
	return b.skillTools.CloseProgramFrame(
		ctx,
		executionID,
		frameID,
	)
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

func (b *Binder) PrepareUserMessageSkillTemplate(
	ctx context.Context,
	request sessionv3Runtime.UserMessageSkillTemplatePreparationRequest,
) (sessionv3Runtime.UserMessageSkillTemplateResult, error) {
	if b == nil || b.skillTools == nil {
		return sessionv3Runtime.UserMessageSkillTemplateResult{}, errors.New(
			"session v3 skill runtime is unavailable",
		)
	}
	return b.skillTools.PrepareUserMessageTemplate(
		ctx,
		request.Skill,
		request.Arguments,
	)
}

func (b *Binder) PreviewInstructionSkill(
	ctx context.Context,
	request sessionv3Runtime.InstructionSkillPreviewRequest,
) (sessionv3Runtime.InstructionSkillPreviewResult, error) {
	if b == nil || b.skillTools == nil {
		return sessionv3Runtime.InstructionSkillPreviewResult{}, errors.New(
			"session v3 skill runtime is unavailable",
		)
	}
	return b.skillTools.PreviewInstruction(ctx, request.Skill, request.Arguments)
}

func (b *Binder) PrepareAppTool(
	ctx context.Context,
	request sessionv3Runtime.AppToolPreparationRequest,
) (sessionv3Runtime.ToolCall, error) {
	if b == nil || b.mcpTools == nil {
		return sessionv3Runtime.ToolCall{}, errors.New(
			"session v3 mcp app runtime is unavailable",
		)
	}
	return b.mcpTools.PrepareAppTool(ctx, request)
}

func (b *Binder) ReadAppResource(
	ctx context.Context,
	request sessionv3Runtime.AppResourceReadRequest,
) (sessionv3Runtime.AppResourceReadResult, error) {
	if b == nil || b.mcpTools == nil {
		return sessionv3Runtime.AppResourceReadResult{}, errors.New(
			"session v3 mcp app runtime is unavailable",
		)
	}
	return b.mcpTools.ReadAppResource(ctx, request)
}

func (b *Binder) ValidateAppContext(
	ctx context.Context,
	request sessionv3Runtime.AppContextStagingRequest,
) (sessionv3Runtime.AppContextUpdate, error) {
	if b == nil || b.mcpTools == nil {
		return sessionv3Runtime.AppContextUpdate{}, errors.New(
			"session v3 mcp app runtime is unavailable",
		)
	}
	return b.mcpTools.ValidateAppContext(ctx, request)
}

func (b *Binder) ValidatePortPlans(
	_ context.Context,
	request sessionv3Runtime.WorkDispatchRequest,
) error {
	if len(request.WorkItem.Ports) == 0 {
		return nil
	}

	switch request.WorkItem.Type {
	case sessionv3Runtime.WorkItemTypeLLM:
		return validateLLMPortTransport(request.WorkItem)

	case sessionv3Runtime.WorkItemTypeTool:
		return validateToolPortTransport(request.WorkItem)

	case sessionv3Runtime.WorkItemTypeService,
		sessionv3Runtime.WorkItemTypeActivity:
		return errors.New(
			"service and activity transport must be validated by the workflow binding layer",
		)

	default:
		return errors.New(
			"port adaptation is unavailable for this work item",
		)
	}
}
