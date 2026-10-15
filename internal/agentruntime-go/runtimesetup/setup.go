package runtimesetup

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	inferenceRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/inference"
	mcpConnection "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/connection"
	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/invocation"
	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
	skillRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/skill"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/modelruntime"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/skillruntime"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/toolruntime"
	agentAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent"
	textAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/text"
	toolAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool"
	workspaceAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace"
)

const (
	typeStr = "type"

	resolutionItemKindTool        = "tool"
	resolutionItemKindSkill       = "skill"
	resolutionItemKindInstruction = "instruction"
	resolutionItemKindMCP         = "mcp"

	inputRoleTool    = resolutionItemKindTool
	toolChoicePrefix = resolutionItemKindTool

	jsonSchemaTypeObject = "object"

	skillToolLoadName         = "skills-load"
	skillToolUnloadName       = "skills-unload"
	skillToolReadResourceName = "skills-readresource"
	skillToolRunScriptName    = "skills-runscript"
)

// ArtifactSnapshotReader is the narrow generic Artifact read capability used
// by runtime setup to enforce explicit Session selection pins. Session and
// inference do not receive this dependency.
type ArtifactSnapshotReader interface {
	GetMany(
		ctx context.Context,
		refs []artifactModel.ArtifactRef,
	) ([]artifactModel.Artifact, error)
}

type Dependencies struct {
	Checkpoints sessionRuntime.CheckpointSink
	Recovery    sessionRuntime.RecoverySource
	Events      sessionRuntime.EventPublisher

	Providers         *inferenceRuntime.ProviderSetAPI
	Models            *modelruntime.RuntimeAdapter
	ArtifactSnapshots ArtifactSnapshotReader

	Agents     *agentAPI.Service
	Workspaces *workspaceAPI.Service
	Texts      *textAPI.Service
	Tools      *toolAPI.Service

	ToolRuntime *toolruntime.Adapter

	SkillResolver *skillruntime.RuntimeAdapter
	SkillRuntime  *skillRuntime.Service

	MCPRuntime *mcpConnection.MCPRuntimeManager
	MCPBridge  *invocation.ToolBridge
}

func New(
	ctx context.Context,
	dependencies Dependencies,
) (*sessionRuntime.Runtime, error) {
	if err := dependencies.validate(); err != nil {
		return nil, err
	}

	attachments := newReferenceMaterializer()
	coordinator := newCoordinator(dependencies, attachments)

	goExecutor, err := newGoToolExecutor(
		dependencies.Tools,
		dependencies.ToolRuntime,
	)
	if err != nil {
		return nil, err
	}

	mcpExecutor, err := newMCPToolExecutor(dependencies.MCPBridge)
	if err != nil {
		return nil, err
	}

	skillExecutor, err := newSkillToolExecutor(coordinator)
	if err != nil {
		return nil, err
	}

	router, err := sessionRuntime.NewToolRouter(
		map[sessionRuntime.ToolBindingKind]sessionRuntime.ToolExecutor{
			sessionRuntime.ToolBindingKindGo:    goExecutor,
			sessionRuntime.ToolBindingKindMCP:   mcpExecutor,
			sessionRuntime.ToolBindingKindSkill: skillExecutor,
		},
	)
	if err != nil {
		return nil, err
	}

	runtimeValue, err := sessionRuntime.NewRuntime(
		sessionRuntime.RuntimeDependencies{
			Preparation:       coordinator,
			Operations:        coordinator,
			Templates:         coordinator,
			InstructionSkills: coordinator,
			Model:             coordinator,
			Tools:             router,
			Skills:            skillExecutor,
			Apps:              coordinator,
			Attachments:       attachments,
			Events:            dependencies.Events,
			Checkpoints:       dependencies.Checkpoints,
			Diagnostics:       runtimeDiagnosticLogger{},
			Recovery:          dependencies.Recovery,
		},
	)
	if err != nil {
		return nil, err
	}

	if err := runtimeValue.Restore(ctx); err != nil {
		//nolint:contextcheck // Runtime.Close owns and cancels its lifecycle internally.
		runtimeValue.Close()
		return nil, fmt.Errorf("restore Session runtime: %w", err)
	}

	return runtimeValue, nil
}

func (d Dependencies) validate() error {
	switch {
	case d.Checkpoints == nil:
		return errors.New(
			"runtime setup requires a Session checkpoint sink",
		)
	case d.Recovery == nil:
		return errors.New(
			"runtime setup requires a Session recovery source",
		)
	case d.Providers == nil:
		return errors.New(
			"runtime setup requires an inference provider runtime",
		)
	case d.Models == nil:
		return errors.New("runtime setup requires a Model runtime")
	case d.ArtifactSnapshots == nil:
		return errors.New("runtime setup requires an Artifact snapshot reader")
	case d.Agents == nil:
		return errors.New("runtime setup requires an Agent service")
	case d.Workspaces == nil:
		return errors.New("runtime setup requires a Workspace service")
	case d.Texts == nil:
		return errors.New("runtime setup requires a Text service")
	case d.Tools == nil:
		return errors.New("runtime setup requires a Tool service")
	case d.ToolRuntime == nil:
		return errors.New("runtime setup requires a Go Tool runtime")
	case d.SkillResolver == nil:
		return errors.New(
			"runtime setup requires an Artifact Skill resolver",
		)
	case d.SkillRuntime == nil:
		return errors.New("runtime setup requires a Skill runtime")
	case d.MCPRuntime == nil:
		return errors.New("runtime setup requires an MCP runtime")
	case d.MCPBridge == nil:
		return errors.New(
			"runtime setup requires an MCP invocation bridge",
		)
	default:
		return nil
	}
}

type runtimeDiagnosticLogger struct{}

func (runtimeDiagnosticLogger) ReportRuntimeDiagnostic(
	_ context.Context,
	diagnostic sessionRuntime.RuntimeDiagnostic,
) {
	slog.Warn(
		"session runtime diagnostic",
		"sessionID", diagnostic.SessionID,
		"runID", diagnostic.RunID,
		"operation", diagnostic.Operation,
		"message", diagnostic.Message,
		"cause", diagnostic.Cause,
	)
}
