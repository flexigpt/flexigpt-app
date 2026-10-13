package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strings"
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
	conversationSpec "github.com/flexigpt/flexigpt-app/internal/conversation/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
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
	workspaceSource inferenceRuntime.WorkspaceSource,
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
		skillRuntimeWrapper.service == nil ||
		workspaceSource == nil {
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
		Attachments: attachmentMaterializer,
		Checkpoints: checkpoints,
		Workspace: &agentgoWorkspaceSkillResolver{
			source: workspaceSource,
		},
		Recovery: recovery,
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

type agentgoWorkspaceSkillResolver struct {
	source inferenceRuntime.WorkspaceSource
}

func (r *agentgoWorkspaceSkillResolver) ResolveWorkspaceSkills(
	ctx context.Context,
	request sessionRuntime.WorkspaceSkillResolutionRequest,
) (sessionRuntime.WorkspaceSkillResolution, error) {
	if r == nil || r.source == nil {
		return sessionRuntime.WorkspaceSkillResolution{}, errors.New(
			"workspace skill resolver is unavailable",
		)
	}
	if len(request.RequestedWorkspaceSkills) == 0 {
		return sessionRuntime.WorkspaceSkillResolution{}, nil
	}

	rawSelection := strings.TrimSpace(
		string(request.Configuration.WorkspaceSelection),
	)
	if rawSelection == "" || rawSelection == "null" {
		return sessionRuntime.WorkspaceSkillResolution{}, errors.New(
			"workspace skills require a Workspace selection",
		)
	}

	var selection conversationSpec.WorkspaceConversationSelection
	if err := json.Unmarshal(
		request.Configuration.WorkspaceSelection,
		&selection,
	); err != nil {
		return sessionRuntime.WorkspaceSkillResolution{}, fmt.Errorf(
			"decode Workspace selection for Skill session: %w",
			err,
		)
	}

	requested := make(
		map[string]sessionRuntime.ArtifactReference,
		len(request.RequestedWorkspaceSkills),
	)
	for _, ref := range request.RequestedWorkspaceSkills {
		if ref.RootID == "" || ref.ArtifactID == "" {
			return sessionRuntime.WorkspaceSkillResolution{}, errors.New(
				"workspace skill artifact reference is incomplete",
			)
		}
		requested[ref.RootID+"\x00"+ref.ArtifactID] = ref
	}

	artifactRefs, err := sessionArtifactRefsToArtifactRefs(
		request.RequestedWorkspaceSkills,
	)
	if err != nil {
		return sessionRuntime.WorkspaceSkillResolution{}, err
	}

	plan, err := r.source.LoadWorkspaceSkills(
		ctx,
		selection.Workspace,
		artifactRefs,
	)
	if err != nil {
		return sessionRuntime.WorkspaceSkillResolution{}, err
	}

	enabled := make([]sessionRuntime.ArtifactReference, 0, len(plan.Skills))
	seen := make(map[string]struct{}, len(plan.Skills))
	for _, skill := range plan.Skills {
		if skill.Insert != declaration.InsertInstructions {
			continue
		}
		key := string(skill.Artifact.RootID) +
			"\x00" +
			string(skill.Artifact.ArtifactID)
		ref, found := requested[key]
		if !found {
			continue
		}
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		enabled = append(enabled, ref)
	}

	return sessionRuntime.WorkspaceSkillResolution{
		EnabledWorkspaceSkills: enabled,
	}, nil
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

	enabledDefinitions, byDefinition, err := b.resolveSkillDefinitions(
		ctx,
		request.Enabled,
	)
	if err != nil {
		return skillRuntime.ArtifactSessionResult{}, err
	}

	activeDefinitions, activeByDefinition, err := b.resolveSkillDefinitions(
		ctx,
		request.Active,
	)
	if err != nil {
		return skillRuntime.ArtifactSessionResult{}, err
	}

	for key := range activeByDefinition {
		if _, allowed := byDefinition[key]; !allowed {
			return skillRuntime.ArtifactSessionResult{}, errors.New(
				"active Skill is not included in the Run allow-list",
			)
		}
	}

	response, err := b.runtime.service.CreateSkillSession(
		ctx,
		&skillRuntime.CreateSkillSessionRequest{
			Body: &skillRuntime.CreateSkillSessionRequestBody{
				CloseSessionID: agentskillsRuntimeSpec.SessionID(
					request.PreviousSessionID,
				),
				AllowedSkills: enabledDefinitions,
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

// resolveSkillDefinitions resolves one Artifact at a time intentionally.
//
// ResolveSkills is allowed to reorder, deduplicate, filter, or expand its
// result. Mapping the returned Definitions slice by positional index to the
// requested ArtifactReference slice is therefore unsafe: it can mark a
// different Artifact active than the one actually resolved by the runtime.
func (b *agentgoArtifactSkillSessionBridge) resolveSkillDefinitions(
	ctx context.Context,
	references []sessionRuntime.ArtifactReference,
) (
	[]provider.SkillDef,
	map[string]sessionRuntime.ArtifactReference,
	error,
) {
	definitions := make([]provider.SkillDef, 0, len(references))
	byDefinition := make(
		map[string]sessionRuntime.ArtifactReference,
		len(references),
	)
	seenReferences := make(map[string]struct{}, len(references))

	for _, reference := range references {
		if reference.RootID == "" || reference.ArtifactID == "" {
			return nil, nil, errors.New(
				"skill artifact reference is incomplete",
			)
		}

		referenceKey := reference.RootID + "\x00" + reference.ArtifactID
		if _, duplicate := seenReferences[referenceKey]; duplicate {
			continue
		}
		seenReferences[referenceKey] = struct{}{}

		artifactRefs, err := sessionArtifactRefsToArtifactRefs(
			[]sessionRuntime.ArtifactReference{reference},
		)
		if err != nil {
			return nil, nil, err
		}

		resolved, err := b.aggregate.resolver.ResolveSkills(
			ctx,
			artifactRefs,
		)
		if err != nil {
			return nil, nil, err
		}
		if len(resolved.Definitions) != 1 {
			return nil, nil, fmt.Errorf(
				"skill artifact %q must resolve to exactly one Skill definition",
				reference.ArtifactID,
			)
		}

		definition := resolved.Definitions[0]
		definitionKey, err := definitionKey(definition)
		if err != nil {
			return nil, nil, err
		}
		if previous, found := byDefinition[definitionKey]; found &&
			previous != reference {
			return nil, nil, errors.New("multiple Skill Artifacts resolved to one runtime Skill definition")
		}

		byDefinition[definitionKey] = reference
		definitions = append(definitions, definition)
	}

	return definitions, byDefinition, nil
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
