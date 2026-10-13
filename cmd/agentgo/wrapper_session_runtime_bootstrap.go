package main

import (
	"context"
	"errors"
	"fmt"

	runtimesetupRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/runtimesetup"
	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/modelruntime"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/skillruntime"
)

func InitComposedSessionRuntime(
	ctx context.Context,
	wrapper *SessionRuntimeWrapper,
	checkpoints sessionRuntime.CheckpointSink,
	recovery sessionRuntime.RecoverySource,
	completion *CompletionWrapper,
	modelRuntime *modelruntime.RuntimeAdapter,
	artifactSnapshots runtimesetupRuntime.ArtifactSnapshotReader,
	tools *ToolStoreWrapper,
	mcpWrapper *MCPRuntimeWrapper,
	skillResolver *skillruntime.RuntimeAdapter,
	skillRuntimeWrapper *SkillRuntimeWrapper,
	agents *AgentStoreWrapper,
	workspaces *WorkspaceStoreWrapper,
	texts *TextStoreWrapper,
) error {
	if wrapper == nil ||
		completion == nil ||
		modelRuntime == nil ||
		artifactSnapshots == nil ||
		tools == nil ||
		mcpWrapper == nil ||
		skillResolver == nil ||
		skillRuntimeWrapper == nil ||
		agents == nil ||
		workspaces == nil ||
		texts == nil {
		return errors.New("session runtime composition dependencies are incomplete")
	}

	providers, err := completion.providerSet()
	if err != nil {
		return err
	}
	if tools.api == nil ||
		tools.runtime == nil ||
		mcpWrapper.runtime == nil ||
		mcpWrapper.toolBridge == nil ||
		skillRuntimeWrapper.service == nil ||
		agents.api == nil ||
		workspaces.api == nil ||
		texts.service == nil {
		return errors.New("session runtime composition dependency is closed")
	}

	runtimeValue, err := runtimesetupRuntime.New(
		ctx,
		runtimesetupRuntime.Dependencies{
			Checkpoints:       checkpoints,
			Recovery:          recovery,
			Providers:         providers,
			Models:            modelRuntime,
			ArtifactSnapshots: artifactSnapshots,
			Agents:            agents.api,
			Workspaces:        workspaces.api,
			Texts:             texts.service,
			Tools:             tools.api,
			ToolRuntime:       tools.runtime,
			SkillResolver:     skillResolver,
			SkillRuntime:      skillRuntimeWrapper.service,
			MCPRuntime:        mcpWrapper.runtime,
			MCPBridge:         mcpWrapper.toolBridge,
		},
	)
	if err != nil {
		return fmt.Errorf("construct Session runtime: %w", err)
	}

	if err := InitSessionRuntimeWrapper(wrapper, runtimeValue); err != nil {
		runtimeValue.Close()
		return err
	}
	return nil
}
