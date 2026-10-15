package runtimesetup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/flexigpt/agentskills-go/document"
	"github.com/flexigpt/agentskills-go/provider"
	agentskillsRuntime "github.com/flexigpt/agentskills-go/runtime"
	inferenceSpec "github.com/flexigpt/inference-go/spec"

	inferenceRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/inference"
	inferenceRuntimeSpec "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/inference/spec"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/server"
	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
	skillRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/skill"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/modelruntime"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/skillruntime"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	agentAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	modelDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model/domain"
	textAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/text"
	toolAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool"
	workspaceAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace"
)

const operationResolutionVersion = uint32(1)

type coordinator struct {
	providers         *inferenceRuntime.ProviderSetAPI
	models            *modelruntime.RuntimeAdapter
	artifactSnapshots ArtifactSnapshotReader

	agents     *agentAPI.Service
	workspaces *workspaceAPI.Service
	texts      textMaterializer
	tools      *toolAPI.Service

	skillResolver *skillruntime.RuntimeAdapter
	skillRuntime  *skillRuntime.Service

	mcpRuntime *mcpRuntimeAdapter

	attachments *referenceMaterializer

	skillMu       sync.RWMutex
	skillSessions map[string]preparedSkillSession
}

type textMaterializer interface {
	Materialize(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
	) (textAPI.Materialization, error)
}

type mcpRuntimeAdapter struct {
	runtime interface {
		Status(
			ctx context.Context,
			serverID mcpServer.ServerID,
		) (*mcpServer.MCPServerRuntimeSnapshot, error)
		ListTools(
			ctx context.Context,
			serverID mcpServer.ServerID,
		) ([]mcpServer.MCPToolCapability, error)
		ListResources(
			ctx context.Context,
			serverID mcpServer.ServerID,
		) ([]mcpServer.MCPResourceRef, error)
		ListResourceTemplates(
			ctx context.Context,
			serverID mcpServer.ServerID,
		) ([]mcpServer.MCPResourceTemplateRef, error)
		ListPrompts(
			ctx context.Context,
			serverID mcpServer.ServerID,
		) ([]mcpServer.MCPPromptRef, error)
		ReadResource(
			ctx context.Context,
			serverID mcpServer.ServerID,
			uri string,
		) (*mcpServer.MCPReadResourceResponseBody, error)
		GetPrompt(
			ctx context.Context,
			serverID mcpServer.ServerID,
			name string,
			arguments map[string]string,
		) (*mcpServer.MCPGetPromptResponseBody, error)
	}
}

func newCoordinator(
	dependencies Dependencies,
	attachments *referenceMaterializer,
) *coordinator {
	return &coordinator{
		providers:         dependencies.Providers,
		models:            dependencies.Models,
		artifactSnapshots: dependencies.ArtifactSnapshots,
		agents:            dependencies.Agents,
		workspaces:        dependencies.Workspaces,
		texts:             dependencies.Texts,
		tools:             dependencies.Tools,
		skillResolver:     dependencies.SkillResolver,
		skillRuntime:      dependencies.SkillRuntime,
		mcpRuntime: &mcpRuntimeAdapter{
			runtime: dependencies.MCPRuntime,
		},
		attachments:   attachments,
		skillSessions: make(map[string]preparedSkillSession),
	}
}

type preparedSkillSession struct {
	byDefinition map[provider.SkillDef]sessionRuntime.ArtifactReference
}

type preparedOperation struct {
	configuration inferenceRuntimeSpec.RuntimeConfiguration
	inputs        []inferenceSpec.InputUnion
	currentInputs []inferenceSpec.InputUnion
	toolChoices   []inferenceSpec.ToolChoice
	bindings      preparedBindings
	skills        preparedSkills
}

type preparedBindings struct {
	byChoice map[string]sessionRuntime.ToolCallProposal
	byName   map[string]sessionRuntime.ToolCallProposal
}

type preparedSkills struct {
	definitions  []provider.SkillDef
	byArtifact   map[string]provider.SkillDef
	byDefinition map[provider.SkillDef]sessionRuntime.ArtifactReference
}

type operationPlan struct {
	summary      sessionRuntime.OperationResolutionSummary
	model        *sessionRuntime.ArtifactSelection
	modelOwner   sessionRuntime.ConfigurationOwner
	instructions []instructionContribution
	tools        []toolContribution
	skills       []skillContribution
	activeSkills map[string]struct{}
	mcp          []sessionRuntime.MCPServerSelection
	mcpUses      map[string]*sessionRuntime.ResolvedArtifact
	mcpOwners    map[string]sessionRuntime.ConfigurationOwner
	userContexts []string
}

type instructionContribution struct {
	key   string
	owner sessionRuntime.ConfigurationOwner
	label string
	text  string
	item  sessionRuntime.ResolutionItem
}

type toolContribution struct {
	key         string
	owner       sessionRuntime.ConfigurationOwner
	target      *composition.CapabilityTarget
	selection   *sessionRuntime.ToolSelectionIntent
	autoExecute *bool
	item        sessionRuntime.ResolutionItem
}

type skillContribution struct {
	key         string
	owner       sessionRuntime.ConfigurationOwner
	selection   sessionRuntime.ArtifactSelection
	item        sessionRuntime.ResolutionItem
	instruction bool
}

func (c *coordinator) PrepareSubmission(
	ctx context.Context,
	request sessionRuntime.SubmissionPreparationRequest,
) (sessionRuntime.PreparedSubmission, error) {
	if err := normalizeConfigurationIntent(&request.Configuration); err != nil {
		return sessionRuntime.PreparedSubmission{}, err
	}
	if err := normalizeAppContextUpdates(request.Input.MCPAppContext); err != nil {
		return sessionRuntime.PreparedSubmission{}, err
	}

	return sessionRuntime.PreparedSubmission{
		Configuration:        cloneConfiguration(request.Configuration),
		Input:                cloneUserInput(request.Input),
		PersistConfiguration: request.PersistConfiguration,
	}, nil
}

func (c *coordinator) PrepareModelOperation(
	ctx context.Context,
	request sessionRuntime.ModelOperationPreparationRequest,
) (sessionRuntime.OperationPreparation, error) {
	plan := operationPlan{
		summary: sessionRuntime.OperationResolutionSummary{
			Version: operationResolutionVersion,
		},
		activeSkills: make(map[string]struct{}),
		mcpUses:      make(map[string]*sessionRuntime.ResolvedArtifact),
		mcpOwners:    make(map[string]sessionRuntime.ConfigurationOwner),
	}

	if err := c.resolveAgent(ctx, request.Configuration.Agent, &plan); err != nil {
		return sessionRuntime.OperationPreparation{}, err
	}
	if err := c.resolveWorkspace(ctx, request.Configuration.Workspace, &plan); err != nil {
		return sessionRuntime.OperationPreparation{}, err
	}
	if err := c.applySessionOverrides(
		ctx,
		request.Configuration,
		&plan,
	); err != nil {
		return sessionRuntime.OperationPreparation{}, err
	}

	if plan.model == nil {
		return sessionRuntime.OperationPreparation{}, operationPreparationError(
			"model_selection_missing",
			"no effective Model selection is available",
			plan.summary,
			nil,
		)
	}

	modelOwner := plan.modelOwner
	if modelOwner == "" {
		modelOwner = sessionRuntime.ConfigurationOwnerSession
	}

	modelRef, err := artifactRef(*plan.model)
	if err != nil {
		return sessionRuntime.OperationPreparation{}, operationPreparationError(
			"model_selection_invalid",
			err.Error(),
			plan.summary,
			err,
		)
	}

	modelPatch, err := modelRequestPatch(request.Configuration.ModelOverrides)
	if err != nil {
		return sessionRuntime.OperationPreparation{}, operationPreparationError(
			"model_override_invalid",
			err.Error(),
			plan.summary,
			err,
		)
	}

	modelConfig, usedModel, err := c.models.ResolveRuntimeModelWithRequestPatch(
		ctx,
		modelRef,
		modelPatch,
	)
	if err != nil {
		return sessionRuntime.OperationPreparation{}, operationPreparationError(
			"model_unavailable",
			err.Error(),
			plan.summary,
			err,
		)
	}
	usedModelArtifact := resolvedArtifactFromModelUse(usedModel)
	if err := requireSelectionPin(*plan.model, usedModelArtifact); err != nil {
		plan.summary.Model = &sessionRuntime.ResolutionItem{
			Key:      plan.model.CanonicalKey(),
			Kind:     "model",
			Owner:    modelOwner,
			State:    sessionRuntime.ResolutionStateStale,
			Selected: plan.model,
			Used:     usedModelArtifact,
			Reason:   err.Error(),
		}
		return sessionRuntime.OperationPreparation{}, operationPreparationError(
			"model_selection_stale",
			err.Error(),
			plan.summary,
			err,
		)
	}
	plan.summary.Model = &sessionRuntime.ResolutionItem{
		Key:      plan.model.CanonicalKey(),
		Kind:     "model",
		Owner:    modelOwner,
		State:    sessionRuntime.ResolutionStateIncluded,
		Selected: plan.model,
		Used:     usedModelArtifact,
	}
	skills, effectiveSkills, activeSkills, skillInstructions, err := c.prepareSkills(
		ctx,
		plan.skills,
		plan.activeSkills,
		&plan.summary,
	)
	if err != nil {
		return sessionRuntime.OperationPreparation{}, err
	}

	plan.instructions = append(plan.instructions, skillInstructions...)

	systemParts := make([]string, 0, len(plan.instructions)+1)
	if strings.TrimSpace(modelConfig.ModelParam.SystemPrompt) != "" {
		systemParts = append(systemParts, modelConfig.ModelParam.SystemPrompt)
	}
	for _, contribution := range plan.instructions {
		if strings.TrimSpace(contribution.text) == "" {
			continue
		}
		systemParts = append(systemParts, contribution.text)
		plan.summary.Instructions = append(
			plan.summary.Instructions,
			contribution.item,
		)
	}
	modelConfig.ModelParam.SystemPrompt = strings.Join(systemParts, "\n\n")

	toolChoices, bindings, err := c.prepareTools(ctx, plan.tools, &plan.summary)
	if err != nil {
		return sessionRuntime.OperationPreparation{}, err
	}

	mcpChoices, mcpBindings, mcpInputs, err := c.prepareMCP(
		ctx,
		plan.mcp,
		plan.mcpUses,
		plan.mcpOwners,
		&plan.summary,
	)
	if err != nil {
		return sessionRuntime.OperationPreparation{}, err
	}
	toolChoices, bindings, err = mergeToolBindings(
		toolChoices,
		bindings,
		mcpChoices,
		mcpBindings,
	)
	if err != nil {
		return sessionRuntime.OperationPreparation{}, err
	}

	inputs, currentInputs, err := c.buildInputs(
		ctx,
		request.History,
		request.Current,
		append(plan.userContexts, mcpInputs...),
	)
	if err != nil {
		return sessionRuntime.OperationPreparation{}, operationPreparationError(
			"input_preparation_failed",
			err.Error(),
			plan.summary,
			err,
		)
	}

	for index := range request.Current.Attachments {
		plan.summary.Attachments = append(
			plan.summary.Attachments,
			sessionRuntime.ResolutionItem{
				Key:   attachmentReferenceKey(request.Current.Attachments[index]),
				Kind:  "attachment",
				Owner: sessionRuntime.ConfigurationOwnerUser,
				State: sessionRuntime.ResolutionStateIncluded,
			},
		)
	}

	data := &preparedOperation{
		configuration: cloneRuntimeConfiguration(modelConfig),
		inputs:        cloneInputUnions(inputs),
		currentInputs: cloneInputUnions(currentInputs),
		toolChoices:   cloneToolChoices(toolChoices),
		bindings:      bindings,
		skills:        skills,
	}

	fingerprint, err := preparedOperationFingerprint(
		request,
		plan.summary,
		effectiveSkills,
		toolChoices,
	)
	if err != nil {
		return sessionRuntime.OperationPreparation{}, err
	}

	return sessionRuntime.OperationPreparation{
		State:           sessionRuntime.PreparationStatePrepared,
		Fingerprint:     fingerprint,
		PreparedAt:      time.Now().UTC(),
		Resolution:      plan.summary,
		EffectiveSkills: effectiveSkills,
		ActiveSkills:    activeSkills,
		Private:         data,
	}, nil
}

func (c *coordinator) ExecuteModel(
	ctx context.Context,
	request sessionRuntime.ModelExecutionRequest,
	stream sessionRuntime.ModelStream,
) (sessionRuntime.ModelExecutionResult, error) {
	if err := sessionRuntime.RequirePreparedOperation(request.Preparation); err != nil {
		//nolint:nilerr // Session persists model-operation failures in the result.
		return sessionRuntime.ModelExecutionResult{
			Error: &sessionRuntime.OperationError{
				Code:    "prepared_operation_unavailable",
				Message: err.Error(),
			},
		}, nil
	}

	data, ok := request.Preparation.Private.(*preparedOperation)
	if !ok || data == nil {
		return sessionRuntime.ModelExecutionResult{
			Error: &sessionRuntime.OperationError{
				Code:    "prepared_operation_unavailable",
				Message: "prepared operation belongs to another runtime adapter",
			},
		}, nil
	}

	configuration := cloneRuntimeConfiguration(data.configuration)
	inputs := cloneInputUnions(data.inputs)
	currentInputs := cloneInputUnions(data.currentInputs)
	toolChoices := cloneToolChoices(data.toolChoices)
	bindings := clonePreparedBindings(data.bindings)

	if len(data.skills.definitions) != 0 {
		skillPrompt, skillChoices, skillBindings, err := c.hydrateSkillSession(
			ctx,
			request.SkillSessionID,
			data.skills,
		)
		if err != nil {
			//nolint:nilerr // Session persists model-operation failures in the result.
			return sessionRuntime.ModelExecutionResult{
				Error: &sessionRuntime.OperationError{
					Code:    "skill_session_unavailable",
					Message: err.Error(),
				},
			}, nil
		}
		if strings.TrimSpace(skillPrompt) != "" {
			configuration.ModelParam.SystemPrompt = appendSystemPrompt(
				configuration.ModelParam.SystemPrompt,
				skillPrompt,
			)
		}
		toolChoices, bindings, err = mergeToolBindings(
			toolChoices,
			bindings,
			skillChoices,
			skillBindings,
		)
		if err != nil {
			//nolint:nilerr // Session persists model-operation failures in the result.
			return sessionRuntime.ModelExecutionResult{
				Error: &sessionRuntime.OperationError{
					Code:    "prepared_tool_collision",
					Message: err.Error(),
				},
			}, nil
		}
	}

	response, executeErr := c.providers.FetchCompletion(
		ctx,
		&inferenceRuntimeSpec.CompletionRequest{
			Configuration:    configuration,
			Inputs:           inputs,
			ToolChoices:      toolChoices,
			OnStreamText:     stream.OnText,
			OnStreamThinking: stream.OnThinking,
		},
	)

	result := sessionRuntime.ModelExecutionResult{
		HydratedCurrentInputs: marshalInputUnions(currentInputs),
		AttachmentsHydrated:   len(request.Current.Attachments) != 0,
	}
	if response == nil || response.Body == nil ||
		response.Body.InferenceResponse == nil {
		if executeErr != nil {
			result.Error = &sessionRuntime.OperationError{
				Code:    "completion_failed",
				Message: executeErr.Error(),
			}
			//nolint:nilerr // Provider failures are represented as operation results.
			return result, nil
		}
		result.Error = &sessionRuntime.OperationError{
			Code:    "empty_completion_response",
			Message: "provider execution returned no completion response",
		}
		return result, nil
	}

	providerResponse := response.Body.InferenceResponse
	result.ProtocolOutputs = marshalOutputUnions(providerResponse.Outputs)
	if providerResponse.Error != nil {
		result.Error = &sessionRuntime.OperationError{
			Code:    providerResponse.Error.Code,
			Message: providerResponse.Error.Message,
		}
	}

	text, thinking, calls := decodeCompletionOutputs(
		result.ProtocolOutputs,
		bindings,
	)
	result.Text = text
	result.Thinking = thinking
	result.ToolCalls = calls
	result.Usable = strings.TrimSpace(text) != "" ||
		len(calls) != 0 ||
		hasUsableProtocolOutput(result.ProtocolOutputs)

	if executeErr != nil && result.Error == nil {
		result.Error = &sessionRuntime.OperationError{
			Code:    "completion_failed",
			Message: executeErr.Error(),
		}
	}

	return result, nil
}

// PrepareUserMessageSkillTemplate is the only runtime-setup path used by
// frontend Composer template insertion. The frontend never calls Skill Runtime
// rendering directly.
func (c *coordinator) PrepareUserMessageSkillTemplate(
	ctx context.Context,
	request sessionRuntime.UserMessageSkillTemplatePreparationRequest,
) (sessionRuntime.UserMessageSkillTemplateResult, error) {
	skill, text, err := c.renderSelectedSkill(
		ctx,
		request.Skill,
		request.Arguments,
		document.SkillInsertUserMessage,
	)
	if err != nil {
		return sessionRuntime.UserMessageSkillTemplateResult{}, err
	}
	return sessionRuntime.UserMessageSkillTemplateResult{
		Skill: skill,
		Text:  text,
	}, nil
}

func (c *coordinator) PreviewInstructionSkill(
	ctx context.Context,
	request sessionRuntime.InstructionSkillPreviewRequest,
) (sessionRuntime.InstructionSkillPreviewResult, error) {
	skill, text, err := c.renderSelectedSkill(
		ctx,
		request.Skill,
		request.Arguments,
		document.SkillInsertInstructions,
	)
	if err != nil {
		return sessionRuntime.InstructionSkillPreviewResult{}, err
	}
	return sessionRuntime.InstructionSkillPreviewResult{
		Skill: skill,
		Text:  text,
	}, nil
}

func (c *coordinator) renderSelectedSkill(
	ctx context.Context,
	selection sessionRuntime.ArtifactSelection,
	arguments map[string]string,
	expectedInsert document.SkillInsert,
) (sessionRuntime.ArtifactReference, string, error) {
	ref, err := artifactRef(selection)
	if err != nil {
		return sessionRuntime.ArtifactReference{}, "", err
	}

	resolved, err := c.skillResolver.ResolveSkills(
		ctx,
		[]artifactModel.ArtifactRef{ref},
	)
	if err != nil {
		return sessionRuntime.ArtifactReference{}, "", err
	}
	if len(resolved.Values) != 1 {
		return sessionRuntime.ArtifactReference{}, "", errors.New(
			"skill resolver did not return exactly one Skill",
		)
	}

	value := resolved.Values[0]
	if value.Artifact != ref {
		return sessionRuntime.ArtifactReference{}, "", errors.New(
			"skill resolver returned another Artifact",
		)
	}
	if !value.Enabled {
		return sessionRuntime.ArtifactReference{}, "", errors.New(
			"selected user-message Skill is disabled",
		)
	}

	used := resolvedArtifact(
		value.Artifact,
		value.Revision,
		cryptoutil.Digest(value.DefinitionDigest),
	)
	if err := requireSelectionPin(selection, used); err != nil {
		return sessionRuntime.ArtifactReference{}, "", err
	}

	records, err := c.skillRuntime.ListAgentSkills(
		ctx,
		&agentskillsRuntime.SkillListFilter{
			AllowSkills: []provider.SkillDef{value.Definition},
		},
	)
	if err != nil {
		return sessionRuntime.ArtifactReference{}, "", err
	}

	found := false
	for _, record := range records {
		if record.Def != value.Definition {
			continue
		}

		insert, supported := document.NormalizeSkillInsert(record.Insert)
		if !supported || insert != expectedInsert {
			return sessionRuntime.ArtifactReference{}, "", errors.New(
				"selected Skill has a different insertion behavior",
			)
		}
		found = true
		break
	}
	if !found {
		return sessionRuntime.ArtifactReference{}, "", errors.New(
			"skill runtime did not index the selected Skill",
		)
	}

	rendered, err := c.skillRuntime.RenderSkill(
		ctx,
		&skillRuntime.RenderSkillRequest{
			Body: &skillRuntime.RenderSkillRequestBody{
				Definition: value.Definition,
				Arguments:  maps.Clone(arguments),
			},
		},
	)
	if err != nil {
		return sessionRuntime.ArtifactReference{}, "", err
	}
	if rendered == nil || rendered.Body == nil {
		return sessionRuntime.ArtifactReference{}, "", errors.New(
			"skill runtime returned no rendered Skill",
		)
	}

	return sessionArtifactReference(value.Artifact), rendered.Body.Text, nil
}

func cloneConfiguration(
	value sessionRuntime.SessionConfiguration,
) sessionRuntime.SessionConfiguration {
	return jsonutil.CloneJSONInValue(value)
}

func cloneUserInput(value sessionRuntime.UserInput) sessionRuntime.UserInput {
	return jsonutil.CloneJSONInValue(value)
}

func normalizeConfigurationIntent(
	configuration *sessionRuntime.SessionConfiguration,
) error {
	if configuration == nil {
		return errors.New("session configuration is required")
	}
	if err := normalizeArtifactSelection(configuration.Agent); err != nil {
		return fmt.Errorf("agent selection: %w", err)
	}
	if err := normalizeArtifactSelection(configuration.Model); err != nil {
		return fmt.Errorf("model selection: %w", err)
	}
	if configuration.Workspace != nil {
		if err := normalizeArtifactSelection(&configuration.Workspace.Workspace); err != nil {
			return fmt.Errorf("workspace selection: %w", err)
		}
		if err := normalizeArtifactOverrides(configuration.Workspace.Context); err != nil {
			return fmt.Errorf("workspace context overrides: %w", err)
		}
		if err := normalizeArtifactOverrides(configuration.Workspace.Skills); err != nil {
			return fmt.Errorf("workspace skill overrides: %w", err)
		}
		if err := normalizeMCPOverrides(configuration.Workspace.MCP); err != nil {
			return fmt.Errorf("workspace MCP overrides: %w", err)
		}
	}
	if err := normalizeInstructionOverrides(configuration.Instructions); err != nil {
		return err
	}
	if err := normalizeToolOverrides(configuration.Tools); err != nil {
		return err
	}
	if err := normalizeArtifactOverrides(configuration.Skills); err != nil {
		return err
	}
	if err := normalizeActiveSkillOverrides(configuration.ActiveSkills); err != nil {
		return err
	}
	if err := normalizeMCPOverrides(configuration.MCP); err != nil {
		return err
	}
	if err := normalizeModelOverrides(&configuration.ModelOverrides); err != nil {
		return err
	}

	canonical, err := canonicalRaw(configuration.Metadata)
	if err != nil {
		return fmt.Errorf("session metadata: %w", err)
	}
	configuration.Metadata = canonical
	return nil
}

func normalizeArtifactSelection(selection *sessionRuntime.ArtifactSelection) error {
	if selection == nil {
		return nil
	}
	if strings.TrimSpace(selection.RootID) == "" ||
		strings.TrimSpace(selection.ArtifactID) == "" ||
		strings.TrimSpace(selection.RootID) != selection.RootID ||
		strings.TrimSpace(selection.ArtifactID) != selection.ArtifactID {
		return errors.New("artifact identity is incomplete")
	}
	return nil
}

func normalizeArtifactOverrides(
	overrides []sessionRuntime.ArtifactOverride,
) error {
	seen := make(map[string]struct{}, len(overrides))
	for index := range overrides {
		override := &overrides[index]
		if strings.TrimSpace(override.Key) == "" {
			return errors.New("artifact override key is required")
		}
		if _, duplicate := seen[override.Key]; duplicate {
			return fmt.Errorf("duplicate Artifact override key %q", override.Key)
		}
		seen[override.Key] = struct{}{}
		switch override.Mode {
		case sessionRuntime.CollectionOverrideInherit,
			sessionRuntime.CollectionOverrideRemove:
			if override.Value != nil {
				return errors.New("inherit or remove Artifact overrides cannot contain a value")
			}
		case sessionRuntime.CollectionOverrideAdd,
			sessionRuntime.CollectionOverrideReplace:
			if override.Value == nil {
				return errors.New("add or replace Artifact overrides require a value")
			}
			if err := normalizeArtifactSelection(override.Value); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported Artifact override mode %q", override.Mode)
		}
	}
	return nil
}

func normalizeInstructionOverrides(
	overrides []sessionRuntime.InstructionOverride,
) error {
	seen := make(map[string]struct{}, len(overrides))
	for index := range overrides {
		override := &overrides[index]
		if strings.TrimSpace(override.Key) == "" {
			return errors.New("instruction override key is required")
		}
		if _, duplicate := seen[override.Key]; duplicate {
			return fmt.Errorf("duplicate instruction override key %q", override.Key)
		}
		seen[override.Key] = struct{}{}

		switch override.Mode {
		case sessionRuntime.CollectionOverrideInherit,
			sessionRuntime.CollectionOverrideRemove:
			if override.Value != nil {
				return errors.New("inherit or remove instruction overrides cannot contain a value")
			}
		case sessionRuntime.CollectionOverrideAdd,
			sessionRuntime.CollectionOverrideReplace:
			if override.Value == nil {
				return errors.New("add or replace instruction overrides require a value")
			}
			if override.Value.Artifact != nil {
				if err := normalizeArtifactSelection(override.Value.Artifact); err != nil {
					return err
				}
			}
			if override.Value.Artifact == nil &&
				strings.TrimSpace(override.Value.Text) == "" {
				return errors.New("inline instruction text or Artifact source is required")
			}
		default:
			return fmt.Errorf("unsupported instruction override mode %q", override.Mode)
		}
	}
	return nil
}

func normalizeToolOverrides(
	overrides []sessionRuntime.ToolOverride,
) error {
	seen := make(map[string]struct{}, len(overrides))
	for index := range overrides {
		override := &overrides[index]
		if strings.TrimSpace(override.Key) == "" {
			return errors.New("tool override key is required")
		}
		if _, duplicate := seen[override.Key]; duplicate {
			return fmt.Errorf("duplicate Tool override key %q", override.Key)
		}
		seen[override.Key] = struct{}{}

		switch override.Mode {
		case sessionRuntime.CollectionOverrideInherit,
			sessionRuntime.CollectionOverrideRemove:
			if override.Value != nil {
				return errors.New("inherit or remove Tool overrides cannot contain a value")
			}
		case sessionRuntime.CollectionOverrideAdd,
			sessionRuntime.CollectionOverrideReplace:
			if override.Value == nil {
				return errors.New("add or replace Tool overrides require a value")
			}
			if err := normalizeCapabilitySelection(override.Value.Capability); err != nil {
				return err
			}
			canonical, err := canonicalRaw(override.Value.UserArguments)
			if err != nil {
				return fmt.Errorf("tool arguments: %w", err)
			}
			override.Value.UserArguments = canonical
		default:
			return fmt.Errorf("unsupported Tool override mode %q", override.Mode)
		}
	}
	return nil
}

func normalizeCapabilitySelection(
	selection sessionRuntime.CapabilitySelection,
) error {
	switch {
	case selection.Artifact != nil && selection.Direct != nil:
		return errors.New("capability selection cannot contain both Artifact and direct targets")
	case selection.Artifact != nil:
		return normalizeArtifactSelection(selection.Artifact)
	case selection.Direct != nil:
		if strings.TrimSpace(selection.Direct.ProviderIdentity) == "" ||
			strings.TrimSpace(selection.Direct.ProviderLocalID) == "" ||
			strings.TrimSpace(selection.Direct.Name) == "" ||
			strings.TrimSpace(selection.Direct.Type) == "" {
			return errors.New("direct capability identity is incomplete")
		}
		return nil
	default:
		return errors.New("capability selection is required")
	}
}

func normalizeActiveSkillOverrides(
	overrides []sessionRuntime.ActiveSkillOverride,
) error {
	seen := make(map[string]struct{}, len(overrides))
	for index := range overrides {
		override := &overrides[index]
		if strings.TrimSpace(override.Key) == "" {
			return errors.New("active Skill override key is required")
		}
		if _, duplicate := seen[override.Key]; duplicate {
			return fmt.Errorf("duplicate active Skill override key %q", override.Key)
		}
		seen[override.Key] = struct{}{}

		switch override.Mode {
		case sessionRuntime.ActivityOverrideInherit,
			sessionRuntime.ActivityOverrideDeactivate:
			if override.Value != nil {
				return errors.New("inherit or deactivate overrides cannot contain a value")
			}
		case sessionRuntime.ActivityOverrideActivate:
			if override.Value == nil {
				return errors.New("activate override requires a Skill Artifact")
			}
			if err := normalizeArtifactSelection(override.Value); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported active Skill override mode %q", override.Mode)
		}
	}
	return nil
}

func normalizeMCPOverrides(
	overrides []sessionRuntime.MCPServerOverride,
) error {
	seen := make(map[string]struct{}, len(overrides))
	for index := range overrides {
		override := &overrides[index]
		if strings.TrimSpace(override.Key) == "" {
			return errors.New("MCP override key is required")
		}
		if _, duplicate := seen[override.Key]; duplicate {
			return fmt.Errorf("duplicate MCP override key %q", override.Key)
		}
		seen[override.Key] = struct{}{}

		switch override.Mode {
		case sessionRuntime.CollectionOverrideInherit,
			sessionRuntime.CollectionOverrideRemove:
			if override.Value != nil {
				return errors.New("inherit or remove MCP overrides cannot contain a value")
			}
		case sessionRuntime.CollectionOverrideAdd,
			sessionRuntime.CollectionOverrideReplace:
			if override.Value == nil {
				return errors.New("add or replace MCP overrides require a value")
			}
			if err := normalizeMCPSelection(*override.Value); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported MCP override mode %q", override.Mode)
		}
	}
	return nil
}

func normalizeMCPSelection(
	selection sessionRuntime.MCPServerSelection,
) error {
	if err := normalizeArtifactSelection(&selection.Server); err != nil {
		return err
	}
	switch selection.ToolExposure {
	case sessionRuntime.MCPToolExposureNone,
		sessionRuntime.MCPToolExposureAll,
		sessionRuntime.MCPToolExposureSelected:
	default:
		return fmt.Errorf("unsupported MCP Tool exposure %q", selection.ToolExposure)
	}
	if selection.ToolExposure == sessionRuntime.MCPToolExposureSelected &&
		len(selection.SelectedTools) == 0 {
		return errors.New("selected MCP Tool exposure requires selected Tools")
	}
	return nil
}

func normalizeModelOverrides(
	overrides *sessionRuntime.ModelDefaultsOverride,
) error {
	if overrides == nil {
		return nil
	}
	seen := make(map[sessionRuntime.ModelDefaultField]struct{}, len(overrides.Clear))
	for _, field := range overrides.Clear {
		if _, duplicate := seen[field]; duplicate {
			return fmt.Errorf("duplicate Model default clear field %q", field)
		}
		seen[field] = struct{}{}
	}
	for key, raw := range overrides.AdapterParameters {
		if strings.TrimSpace(key) == "" {
			return errors.New("model adapter parameter key is empty")
		}
		canonical, err := canonicalRaw(raw)
		if err != nil {
			return fmt.Errorf("model adapter parameter %q: %w", key, err)
		}
		overrides.AdapterParameters[key] = canonical
	}
	if overrides.Output != nil &&
		overrides.Output.Format != nil &&
		overrides.Output.Format.JSONSchema != nil {
		value := overrides.Output.Format.JSONSchema
		canonical, err := canonicalRaw(value.Schema)
		if err != nil {
			return fmt.Errorf("model output JSON schema: %w", err)
		}
		value.Schema = canonical
	}
	return nil
}

func normalizeAppContextUpdates(
	updates []sessionRuntime.AppContextUpdate,
) error {
	for index := range updates {
		update := &updates[index]
		if strings.TrimSpace(string(update.AppInstanceID)) == "" ||
			strings.TrimSpace(string(update.ViewID)) == "" ||
			strings.TrimSpace(update.ServerID) == "" {
			return errors.New("mcp app context update scope is incomplete")
		}
		canonical, err := canonicalRaw(update.StructuredContent)
		if err != nil {
			return fmt.Errorf("MCP App context update %d: %w", index, err)
		}
		update.StructuredContent = canonical
	}
	return nil
}

func canonicalRaw(value json.RawMessage) (json.RawMessage, error) {
	if strings.TrimSpace(string(value)) == "" {
		return nil, nil
	}
	canonical, err := jsonutil.Canonicalize(value)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(canonical), nil
}

func artifactRef(selection sessionRuntime.ArtifactSelection) (artifactModel.ArtifactRef, error) {
	value := artifactModel.ArtifactRef{
		RootID:     rootModel.RootID(selection.RootID),
		ArtifactID: artifactModel.ArtifactID(selection.ArtifactID),
	}
	if err := value.Validate(); err != nil {
		return artifactModel.ArtifactRef{}, err
	}
	return value, nil
}

func (c *coordinator) resolveArtifactSelection(
	ctx context.Context,
	selection sessionRuntime.ArtifactSelection,
) (*sessionRuntime.ResolvedArtifact, error) {
	ref, err := artifactRef(selection)
	if err != nil {
		return nil, err
	}

	values, err := c.artifactSnapshots.GetMany(
		ctx,
		[]artifactModel.ArtifactRef{ref},
	)
	if err != nil {
		return nil, err
	}
	if len(values) != 1 || values[0].Ref() != ref {
		return nil, errors.New(
			"artifact snapshot reader returned an unexpected Artifact",
		)
	}

	return resolvedArtifact(
		ref,
		values[0].Revision,
		digestFromArtifact(values[0]),
	), nil
}

func resolvedArtifactFromModelUse(
	value *modelruntime.RuntimeModelUse,
) *sessionRuntime.ResolvedArtifact {
	if value == nil {
		return nil
	}
	return &sessionRuntime.ResolvedArtifact{
		RootID:     value.RootID,
		ArtifactID: value.ArtifactID,
		Revision:   sessionRuntime.Revision(value.Revision),
		Digest:     value.Digest,
	}
}

func attachmentReferenceKey(
	value sessionRuntime.AttachmentReference,
) string {
	switch value.Kind {
	case sessionRuntime.AttachmentReferenceFile:
		if value.File != nil {
			return "file:" + value.File.OriginalPath
		}
	case sessionRuntime.AttachmentReferenceImage:
		if value.Image != nil {
			return "image:" + value.Image.OriginalPath
		}
	case sessionRuntime.AttachmentReferenceURL:
		if value.URL != nil {
			return "url:" + value.URL.Original
		}
	case sessionRuntime.AttachmentReferenceDirectory,
		sessionRuntime.AttachmentReferenceGeneric:
		return string(value.Kind) + ":" + value.Label
	}
	return string(value.Kind) + ":" + value.Label
}

func sessionArtifactReference(
	value artifactModel.ArtifactRef,
) sessionRuntime.ArtifactReference {
	return sessionRuntime.ArtifactReference{
		RootID:     string(value.RootID),
		ArtifactID: string(value.ArtifactID),
	}
}

func resolvedArtifact(
	value artifactModel.ArtifactRef,
	revision uint64,
	digest cryptoutil.Digest,
) *sessionRuntime.ResolvedArtifact {
	return &sessionRuntime.ResolvedArtifact{
		RootID:     string(value.RootID),
		ArtifactID: string(value.ArtifactID),
		Revision:   sessionRuntime.Revision(revision),
		Digest:     string(digest),
	}
}

func requireSelectionPin(
	selection sessionRuntime.ArtifactSelection,
	used *sessionRuntime.ResolvedArtifact,
) error {
	if used == nil {
		return errors.New("resolved Artifact identity is unavailable")
	}
	if selection.ExpectedRevision != 0 &&
		selection.ExpectedRevision != used.Revision {
		return fmt.Errorf(
			"selected Artifact revision changed: expected %d, current %d",
			selection.ExpectedRevision,
			used.Revision,
		)
	}
	if selection.ExpectedDigest != "" &&
		selection.ExpectedDigest != used.Digest {
		return errors.New("selected Artifact definition changed")
	}
	return nil
}

func modelRequestPatch(
	input sessionRuntime.ModelDefaultsOverride,
) (*modelruntime.RuntimeRequestPatch, error) {
	raw, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	var defaults modelDomain.DefaultsPatch
	if err := json.Unmarshal(raw, &defaults); err != nil {
		return nil, err
	}

	clearFields := make(
		[]modelruntime.RuntimeDefaultField,
		0,
		len(input.Clear),
	)
	for _, field := range input.Clear {
		clearFields = append(
			clearFields,
			modelruntime.RuntimeDefaultField(field),
		)
	}

	if isZeroModelDefaults(input) && len(clearFields) == 0 {
		//nolint:nilnil // A nil patch explicitly means no request-level override.
		return nil, nil
	}
	return &modelruntime.RuntimeRequestPatch{
		Defaults: &defaults,
		Clear:    clearFields,
	}, nil
}

func isZeroModelDefaults(value sessionRuntime.ModelDefaultsOverride) bool {
	return value.Stream == nil &&
		value.MaxPromptTokens == nil &&
		value.MaxOutputTokens == nil &&
		value.Temperature == nil &&
		value.SystemPrompt == nil &&
		value.TimeoutMS == nil &&
		value.Reasoning == nil &&
		value.CacheControl == nil &&
		value.Output == nil &&
		value.StopSequences == nil &&
		len(value.AdapterParameters) == 0
}

func appendSystemPrompt(base string, values ...string) string {
	parts := make([]string, 0, len(values)+1)
	if strings.TrimSpace(base) != "" {
		parts = append(parts, strings.TrimSpace(base))
	}
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			parts = append(parts, strings.TrimSpace(value))
		}
	}
	return strings.Join(parts, "\n\n")
}

func preparedOperationFingerprint(
	request sessionRuntime.ModelOperationPreparationRequest,
	summary sessionRuntime.OperationResolutionSummary,
	skills []sessionRuntime.ArtifactReference,
	tools []inferenceSpec.ToolChoice,
) (string, error) {
	raw, err := jsonutil.MarshalCanonicalObject(
		struct {
			SessionID            sessionRuntime.SessionID                  `json:"sessionID"`
			RunID                sessionRuntime.RunID                      `json:"runID"`
			OperationID          sessionRuntime.OperationID                `json:"operationID"`
			ContinuationRevision sessionRuntime.Revision                   `json:"continuationRevision"`
			Summary              sessionRuntime.OperationResolutionSummary `json:"summary"`
			Skills               []sessionRuntime.ArtifactReference        `json:"skills"`
			Tools                []inferenceSpec.ToolChoice                `json:"tools"`
		}{
			SessionID:            request.SessionID,
			RunID:                request.RunID,
			OperationID:          request.OperationID,
			ContinuationRevision: request.ContinuationRevision,
			Summary:              summary,
			Skills:               skills,
			Tools:                tools,
		},
		storeSpec.MaxDefinitionBytes,
	)
	if err != nil {
		return "", err
	}
	return string(cryptoutil.DigestBytes(raw)), nil
}

func cloneRuntimeConfiguration(
	value inferenceRuntimeSpec.RuntimeConfiguration,
) inferenceRuntimeSpec.RuntimeConfiguration {
	output := value
	output.ProviderParam.DefaultHeaders = maps.Clone(
		value.ProviderParam.DefaultHeaders,
	)
	output.CapabilityOverrides = slices.Clone(
		value.CapabilityOverrides,
	)
	return output
}

func operationPreparationError(
	code string,
	message string,
	summary sessionRuntime.OperationResolutionSummary,
	cause error,
) error {
	return &sessionRuntime.OperationPreparationFailureError{
		Code:    code,
		Message: message,
		Summary: summary,
		Cause:   cause,
	}
}
