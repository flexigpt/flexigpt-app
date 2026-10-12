package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"sync"

	"github.com/flexigpt/agentskills-go/provider"
	agentskillsRuntime "github.com/flexigpt/agentskills-go/runtime"
	agentskillsRuntimeSpec "github.com/flexigpt/agentskills-go/runtime/spec"

	attachmentRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/attachment"
	inferenceRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/inference"
	mcpRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp"
	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
	skillRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/skill"
	toolRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/tool"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
)

func InitComposedSessionRuntime(
	ctx context.Context,
	wrapper *SessionRuntimeWrapper,
	checkpoints sessionRuntime.CheckpointSink,
	recovery sessionRuntime.RecoverySource,
	completion *CompletionWrapper,
	tools *ToolStoreWrapper,
	mcpWrapper *MCPRuntimeWrapper,
	skillAggregate *SkillAggregateWrapper,
	skillRuntimeWrapper *SkillRuntimeWrapper,
) error {
	if wrapper == nil ||
		completion == nil ||
		completion.providersetAPI == nil ||
		tools == nil ||
		tools.api == nil ||
		tools.runtime == nil ||
		mcpWrapper == nil ||
		mcpWrapper.toolBridge == nil ||
		skillAggregate == nil ||
		skillAggregate.resolver == nil ||
		skillRuntimeWrapper == nil ||
		skillRuntimeWrapper.service == nil {
		return errors.New("session runtime composition dependencies are incomplete")
	}

	attachmentMaterializer := attachmentRuntime.NewJSONReferenceMaterializer()
	modelExecutor, err := inferenceRuntime.NewAdapter(
		completion.providersetAPI,
		attachmentMaterializer,
	)
	if err != nil {
		return err
	}

	goExecutor, err := toolRuntime.NewAdapter(tools)
	if err != nil {
		return err
	}

	mcpExecutor, err := mcpRuntime.NewAdapter(mcpWrapper.toolBridge)
	if err != nil {
		return err
	}

	skillBridge := &agentgoArtifactSkillSessionBridge{
		aggregate:   skillAggregate,
		runtime:     skillRuntimeWrapper,
		sessionRefs: make(map[string]map[string]sessionRuntime.ArtifactReference),
	}
	skillExecutor, err := skillRuntime.NewAdapter(skillBridge)
	if err != nil {
		return err
	}

	toolRouter, err := sessionRuntime.NewToolRouter(
		map[sessionRuntime.ToolBindingKind]sessionRuntime.ToolExecutor{
			sessionRuntime.ToolBindingKindGo:    goExecutor,
			sessionRuntime.ToolBindingKindMCP:   mcpExecutor,
			sessionRuntime.ToolBindingKindSkill: skillExecutor,
		},
	)
	if err != nil {
		return err
	}

	runtimeValue, err := sessionRuntime.NewRuntime(sessionRuntime.RuntimeDependencies{
		Model:       modelExecutor,
		Tools:       toolRouter,
		Skills:      skillExecutor,
		Checkpoints: checkpoints,
		Recovery:    recovery,
	})
	if err != nil {
		return err
	}

	if err := runtimeValue.Restore(ctx); err != nil {
		runtimeValue.Close()
		return fmt.Errorf("restore session runtime: %w", err)
	}

	return InitSessionRuntimeWrapper(wrapper, runtimeValue)
}

type agentgoArtifactSkillSessionBridge struct {
	aggregate *SkillAggregateWrapper
	runtime   *SkillRuntimeWrapper

	mu          sync.RWMutex
	sessionRefs map[string]map[string]sessionRuntime.ArtifactReference
}

func (b *agentgoArtifactSkillSessionBridge) EnsureArtifactSkillSession(
	ctx context.Context,
	request skillRuntime.ArtifactSessionRequest,
) (skillRuntime.ArtifactSessionResult, error) {
	if b == nil ||
		b.aggregate == nil ||
		b.aggregate.resolver == nil ||
		b.runtime == nil ||
		b.runtime.service == nil {
		return skillRuntime.ArtifactSessionResult{}, errors.New(
			"artifact skill-session bridge is unavailable",
		)
	}

	enabled, err := sessionArtifactRefsToArtifactRefs(request.Enabled)
	if err != nil {
		return skillRuntime.ArtifactSessionResult{}, err
	}
	active, err := sessionArtifactRefsToArtifactRefs(request.Active)
	if err != nil {
		return skillRuntime.ArtifactSessionResult{}, err
	}

	resolvedEnabled, err := b.aggregate.resolver.ResolveSkills(ctx, enabled)
	if err != nil {
		return skillRuntime.ArtifactSessionResult{}, err
	}

	var activeDefinitions []provider.SkillDef
	if len(active) != 0 {
		resolvedActive, resolveErr := b.aggregate.resolver.ResolveSkills(ctx, active)
		if resolveErr != nil {
			return skillRuntime.ArtifactSessionResult{}, resolveErr
		}
		activeDefinitions = resolvedActive.Definitions
	}

	response, err := b.runtime.service.CreateSkillSession(
		ctx,
		&skillRuntime.CreateSkillSessionRequest{
			Body: &skillRuntime.CreateSkillSessionRequestBody{
				CloseSessionID: agentskillsRuntimeSpec.SessionID(
					request.PreviousSessionID,
				),
				AllowedSkills: resolvedEnabled.Definitions,
				ActiveSkills:  activeDefinitions,
			},
		},
	)
	if err != nil {
		return skillRuntime.ArtifactSessionResult{}, err
	}
	if response == nil || response.Body == nil {
		return skillRuntime.ArtifactSessionResult{}, errors.New(
			"skill runtime returned no session body",
		)
	}

	byDefinition := make(map[string]sessionRuntime.ArtifactReference)
	for index, definition := range resolvedEnabled.Definitions {
		if index >= len(request.Enabled) {
			break
		}
		key, keyErr := definitionKey(definition)
		if keyErr == nil {
			byDefinition[key] = request.Enabled[index]
		}
	}

	activeReferences := make([]sessionRuntime.ArtifactReference, 0)
	for _, definition := range response.Body.ActiveSkills {
		key, keyErr := definitionKey(definition)
		if keyErr != nil {
			continue
		}
		if reference, found := byDefinition[key]; found {
			activeReferences = append(activeReferences, reference)
		}
	}

	b.mu.Lock()
	if request.PreviousSessionID != "" &&
		request.PreviousSessionID != string(response.Body.SessionID) {
		delete(b.sessionRefs, request.PreviousSessionID)
	}
	b.sessionRefs[string(response.Body.SessionID)] = cloneSessionDefinitionMap(
		byDefinition,
	)
	b.mu.Unlock()

	return skillRuntime.ArtifactSessionResult{
		SessionID: string(response.Body.SessionID),
		Active:    activeReferences,
	}, nil
}

func (b *agentgoArtifactSkillSessionBridge) CloseArtifactSkillSession(
	ctx context.Context,
	sessionID string,
) error {
	if b == nil || b.runtime == nil || b.runtime.service == nil {
		return errors.New("artifact skill-session bridge is unavailable")
	}

	err := b.runtime.service.CloseSession(
		ctx,
		agentskillsRuntimeSpec.SessionID(sessionID),
	)
	if err != nil {
		return err
	}

	b.mu.Lock()
	delete(b.sessionRefs, sessionID)
	b.mu.Unlock()

	return nil
}

func (b *agentgoArtifactSkillSessionBridge) ListActiveArtifactSkills(
	ctx context.Context,
	sessionID string,
) ([]sessionRuntime.ArtifactReference, error) {
	if b == nil || b.runtime == nil || b.runtime.service == nil {
		return nil, errors.New("artifact skill-session bridge is unavailable")
	}

	b.mu.RLock()
	byDefinition := cloneSessionDefinitionMap(b.sessionRefs[sessionID])
	b.mu.RUnlock()

	if len(byDefinition) == 0 {
		return []sessionRuntime.ArtifactReference{}, nil
	}

	records, err := b.runtime.service.ListAgentSkills(
		ctx,
		&agentskillsRuntime.SkillListFilter{
			SessionID: agentskillsRuntimeSpec.SessionID(sessionID),
			Activity:  agentskillsRuntimeSpec.SkillActivity("active"),
		},
	)
	if err != nil {
		return nil, err
	}

	output := make([]sessionRuntime.ArtifactReference, 0, len(records))
	seen := make(map[string]struct{}, len(records))
	for _, record := range records {
		key, keyErr := definitionKey(record.Def)
		if keyErr != nil {
			continue
		}
		reference, found := byDefinition[key]
		if !found {
			continue
		}
		referenceKey := reference.RootID + "\x00" + reference.ArtifactID
		if _, duplicate := seen[referenceKey]; duplicate {
			continue
		}
		seen[referenceKey] = struct{}{}
		output = append(output, reference)
	}

	return output, nil
}

func (b *agentgoArtifactSkillSessionBridge) InvokeArtifactSkillTool(
	ctx context.Context,
	sessionID string,
	toolName string,
	args json.RawMessage,
) (skillRuntime.ToolInvocationResult, error) {
	if b == nil || b.runtime == nil || b.runtime.service == nil {
		return skillRuntime.ToolInvocationResult{}, errors.New(
			"artifact skill-session bridge is unavailable",
		)
	}

	response, err := b.runtime.service.InvokeSkillTool(
		ctx,
		&skillRuntime.InvokeSkillToolRequest{
			Body: &skillRuntime.InvokeSkillToolRequestBody{
				SessionID: agentskillsRuntimeSpec.SessionID(sessionID),
				ToolName:  toolName,
				Args:      string(args),
			},
		},
	)
	if err != nil {
		return skillRuntime.ToolInvocationResult{}, err
	}
	if response == nil || response.Body == nil {
		return skillRuntime.ToolInvocationResult{}, errors.New(
			"skill tool invocation returned no body",
		)
	}

	outputs, err := json.Marshal(response.Body.Outputs)
	if err != nil {
		return skillRuntime.ToolInvocationResult{}, err
	}
	meta, err := json.Marshal(response.Body.Meta)
	if err != nil {
		return skillRuntime.ToolInvocationResult{}, err
	}

	return skillRuntime.ToolInvocationResult{
		Outputs: outputs,
		Meta:    meta,
		IsError: response.Body.IsError,
		Error:   response.Body.ErrorMessage,
	}, nil
}

func sessionArtifactRefsToArtifactRefs(
	references []sessionRuntime.ArtifactReference,
) ([]artifactModel.ArtifactRef, error) {
	if len(references) == 0 {
		return nil, nil
	}

	raw, err := json.Marshal(references)
	if err != nil {
		return nil, err
	}

	var output []artifactModel.ArtifactRef
	if err := json.Unmarshal(raw, &output); err != nil {
		return nil, fmt.Errorf(
			"decode Session artifact references: %w",
			err,
		)
	}

	return output, nil
}

func cloneSessionDefinitionMap(
	input map[string]sessionRuntime.ArtifactReference,
) map[string]sessionRuntime.ArtifactReference {
	output := make(map[string]sessionRuntime.ArtifactReference, len(input))
	maps.Copy(output, input)
	return output
}

func definitionKey(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
