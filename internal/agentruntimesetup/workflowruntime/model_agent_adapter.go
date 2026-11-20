package workflowruntime

import (
	"context"
	"errors"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/modelruntime"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	agentAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	workspaceAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace"
)

// This is an adapter capability name, not Workflow source grammar.
const workflowInvocationContextInstructions = "instructions"

// ModelAgentInvocationAdapter is Workflow setup composition over existing
// Model and Agent APIs. It does not add a Workflow-specific Model or Agent
// domain request contract.
type ModelAgentInvocationAdapter struct {
	models     *modelruntime.SourceResolver
	agents     *agentAPI.Service
	artifacts  artifact.API
	workspaces *workspaceAPI.Service
}

func NewModelAgentInvocationAdapter(
	models *modelruntime.SourceResolver,
	agents *agentAPI.Service,
	artifacts artifact.API,
	workspaces *workspaceAPI.Service,
) (*ModelAgentInvocationAdapter, error) {
	if models == nil || agents == nil || artifacts == nil || workspaces == nil {
		return nil, errors.New(
			"workflow model and agent invocation adapter dependencies are incomplete",
		)
	}
	return &ModelAgentInvocationAdapter{
		models:     models,
		agents:     agents,
		artifacts:  artifacts,
		workspaces: workspaces,
	}, nil
}

func (*ModelAgentInvocationAdapter) ModelInvocationCapabilities() WorkflowInvocationTargetCapabilities {
	return WorkflowInvocationTargetCapabilities{
		Modes: []sessionv3Runtime.WorkflowCallMode{
			sessionv3Runtime.WorkflowCallModeInvoke,
			sessionv3Runtime.WorkflowCallModeStart,
		},
		ContextForms: []string{
			workflowInvocationContextInstructions,
			"messages",
			"defaults",
		},
		OutputPorts:       true,
		StreamOutputPorts: true,
	}
}

func (*ModelAgentInvocationAdapter) AgentInvocationCapabilities() WorkflowInvocationTargetCapabilities {
	return WorkflowInvocationTargetCapabilities{
		Modes: []sessionv3Runtime.WorkflowCallMode{
			sessionv3Runtime.WorkflowCallModeInvoke,
			sessionv3Runtime.WorkflowCallModeStart,
			sessionv3Runtime.WorkflowCallModeHandoff,
		},
		ContextForms: []string{
			workflowInvocationContextInstructions,
			"messages",
			"defaults",
		},
		OutputPorts:       true,
		StreamOutputPorts: true,
	}
}

// AdmitWorkflowModelTarget resolves the current source-backed Model through
// the existing Model runtime resolver. Credential and provider availability
// remain Model-domain concerns.
func (a *ModelAgentInvocationAdapter) AdmitWorkflowModelTarget(
	ctx context.Context,
	pin sessionv3Runtime.ArtifactPin,
) error {
	ref := workflowInvocationArtifactRef(pin)
	_, use, err := a.models.ResolveRuntimeModelWithRequestPatch(
		ctx,
		ref,
		nil,
	)
	if err != nil {
		return err
	}
	if use == nil ||
		use.RootID != pin.RootID ||
		use.ArtifactID != pin.ArtifactID ||
		use.Revision != pin.Revision ||
		use.Digest != pin.Digest {
		return errors.New(
			"workflow model target changed during invocation admission",
		)
	}
	return nil
}

// AdmitWorkflowAgentTarget resolves the Agent through the existing Agent API,
// then resolves the first currently usable Agent Model capability through the
// existing Model API. The Workflow adapter does not inspect Agent source or
// build a second Agent request contract.
func (a *ModelAgentInvocationAdapter) AdmitWorkflowAgentTarget(
	ctx context.Context,
	pin sessionv3Runtime.ArtifactPin,
) error {
	_, err := a.ResolveWorkflowAgentOwnerModel(ctx, pin)
	return err
}

// ResolveWorkflowAgentOwnerModel resolves and pins the Model selected by an
// Artifact-backed Agent. Provider SDK Tools use this owner when a Workflow
// invocation supplies an Agent but no explicit Model.
func (a *ModelAgentInvocationAdapter) ResolveWorkflowAgentOwnerModel(
	ctx context.Context,
	pin sessionv3Runtime.ArtifactPin,
) (sessionv3Runtime.ArtifactPin, error) {
	ref := workflowInvocationArtifactRef(pin)
	resolution, err := a.agents.ResolveAgent(ctx, ref)
	if err != nil {
		return sessionv3Runtime.ArtifactPin{}, err
	}
	if resolution.Agent.Ref != ref ||
		resolution.Agent.Revision != pin.Revision ||
		string(resolution.Agent.DefinitionDigest) != pin.Digest {
		return sessionv3Runtime.ArtifactPin{}, errors.New(
			"workflow agent target changed during invocation admission",
		)
	}

	var unavailable error
	for _, occurrence := range resolution.Capabilities.Occurrences {
		if occurrence.Type != declaration.TypeModel {
			continue
		}
		if occurrence.Status != composition.ResolutionAvailable ||
			occurrence.Target == nil {
			if occurrence.Required {
				if occurrence.Message != "" {
					return sessionv3Runtime.ArtifactPin{}, errors.New(occurrence.Message)
				}
				return sessionv3Runtime.ArtifactPin{}, errors.New(
					"required agent model capability is unavailable",
				)
			}
			continue
		}

		if occurrence.Target.Form != composition.TargetFormArtifact ||
			occurrence.Target.Artifact == nil {
			unavailable = errors.New(
				"workflow agent model capability is not artifact-backed",
			)
			if occurrence.Required {
				return sessionv3Runtime.ArtifactPin{}, unavailable
			}
			continue
		}

		_, use, resolveErr := a.models.ResolveRuntimeModelWithRequestPatch(
			ctx,
			*occurrence.Target.Artifact,
			nil,
		)
		if resolveErr != nil {
			unavailable = resolveErr

			if occurrence.Required {
				return sessionv3Runtime.ArtifactPin{}, resolveErr
			}
			continue
		}
		if use == nil || use.RootID == "" || use.ArtifactID == "" ||
			use.Revision == 0 || use.Digest == "" {
			return sessionv3Runtime.ArtifactPin{}, errors.New(
				"workflow agent model resolution returned an incomplete Artifact pin",
			)
		}
		return sessionv3Runtime.ArtifactPin{
			RootID:     use.RootID,
			ArtifactID: use.ArtifactID,
			Revision:   use.Revision,
			Digest:     use.Digest,
		}, nil
	}

	if unavailable != nil {
		return sessionv3Runtime.ArtifactPin{}, unavailable
	}
	return sessionv3Runtime.ArtifactPin{}, errors.New(
		"workflow agent target has no available model capability",
	)
}

func (a *ModelAgentInvocationAdapter) PrepareWorkflowModelInvocation(
	_ context.Context,
	model sessionv3Runtime.ArtifactPin,
) (sessionv3Runtime.WorkflowPreparedInvocation, error) {
	value := sessionv3Runtime.WorkflowPreparedInvocation{
		Model: &model,
	}
	return value, value.Validate()
}

func (a *ModelAgentInvocationAdapter) PrepareWorkflowAgentInvocation(
	ctx context.Context,
	agent sessionv3Runtime.ArtifactPin,
) (sessionv3Runtime.WorkflowPreparedInvocation, error) {
	ref := workflowInvocationArtifactRef(agent)
	resolution, err := a.agents.ResolveAgent(ctx, ref)
	if err != nil {
		return sessionv3Runtime.WorkflowPreparedInvocation{}, err
	}
	if resolution.Agent.Ref != ref ||
		resolution.Agent.Revision != agent.Revision ||
		string(resolution.Agent.DefinitionDigest) != agent.Digest {
		return sessionv3Runtime.WorkflowPreparedInvocation{}, errors.New(
			"workflow Agent changed during invocation preparation",
		)
	}

	output := sessionv3Runtime.WorkflowPreparedInvocation{
		Agent: &agent,
	}
	if err := a.appendAgentInvocationCapabilities(
		ctx,
		&output,
		resolution.Capabilities,
		map[string]struct{}{},
	); err != nil {
		return sessionv3Runtime.WorkflowPreparedInvocation{}, err
	}
	if output.Model == nil {
		model, modelErr := a.ResolveWorkflowAgentOwnerModel(ctx, agent)
		if modelErr != nil {
			return sessionv3Runtime.WorkflowPreparedInvocation{}, modelErr
		}
		output.Model = &model
	}
	return output, output.Validate()
}

func (a *ModelAgentInvocationAdapter) appendAgentInvocationCapabilities(
	ctx context.Context,
	output *sessionv3Runtime.WorkflowPreparedInvocation,
	plan agentAPI.AgentCapabilityPlan,
	visited map[string]struct{},
) error {
	if output == nil {
		return errors.New("workflow invocation contribution output is unavailable")
	}

	for _, occurrence := range plan.Occurrences {
		if occurrence.Status != composition.ResolutionAvailable ||
			occurrence.Target == nil {
			if occurrence.Required {
				return errors.New("required workflow invocation capability is unavailable")
			}
			continue
		}
		if occurrence.Target.Form != composition.TargetFormArtifact ||
			occurrence.Target.Artifact == nil {
			if occurrence.Required {
				return errors.New("workflow invocation capability is not artifact-backed")
			}
			continue
		}
		pin, err := a.pinInvocationArtifact(ctx, *occurrence.Target.Artifact)
		if err != nil {
			return err
		}
		switch occurrence.Type {
		case declaration.TypeModel:
			if output.Model == nil {
				output.Model = &pin
			}
			if occurrence.IncludeSystemPrompt != nil {
				value := *occurrence.IncludeSystemPrompt
				output.IncludeModelSystemPrompt = &value
			}
		case declaration.TypeText:
			appendPreparedWorkflowText(output, pin)
		case declaration.TypeTool:
			var automatic *bool
			if occurrence.AutoExecute != nil {
				value := *occurrence.AutoExecute
				automatic = &value
			}
			appendPreparedWorkflowTool(output, pin, automatic)
		case declaration.TypeSkill:
			appendPreparedWorkflowSkill(
				output,
				pin,
				occurrence.SkillUseMode == agentAPI.AgentSkillUseModeActive,
				occurrence.SkillUseMode == agentAPI.AgentSkillUseModeInstructions,
			)
		case declaration.TypeMCP:
			appendPreparedWorkflowMCP(output, pin)
		case declaration.TypeWorkspace:
			if output.Workspace == nil {
				value := pin
				output.Workspace = &value
			}
			if err := a.appendWorkspaceInvocationForPin(
				ctx,
				output,
				pin,
				visited,
			); err != nil {
				return err
			}
		default:
		}
	}
	return nil
}

func (a *ModelAgentInvocationAdapter) appendWorkspaceInvocationForPin(
	ctx context.Context,
	output *sessionv3Runtime.WorkflowPreparedInvocation,
	pin sessionv3Runtime.ArtifactPin,
	visited map[string]struct{},
) error {
	key := pin.RootID + "\x00" + pin.ArtifactID
	if _, found := visited[key]; found {
		return nil
	}
	visited[key] = struct{}{}

	workspaceRef := workflowInvocationArtifactRef(pin)
	plan, err := a.workspaces.ResolveWorkspaceCapabilities(
		ctx,
		workspaceRef,
	)
	if err != nil {
		return err
	}
	return a.appendWorkspaceInvocationCapabilities(
		ctx,
		output,
		plan,
		visited,
	)
}

// Workspace capability plans use generic composition occurrences. They do not
// own Agent-only declaration facts such as IncludeSystemPrompt, AutoExecute,
// or SkillUseMode. Keep those interpretations in the Agent-specific path.
func (a *ModelAgentInvocationAdapter) appendWorkspaceInvocationCapabilities(
	ctx context.Context,
	output *sessionv3Runtime.WorkflowPreparedInvocation,
	plan composition.CapabilityPlan,
	visited map[string]struct{},
) error {
	if output == nil {
		return errors.New("workflow invocation contribution output is unavailable")
	}

	for _, occurrence := range plan.Occurrences {
		if occurrence.Status != composition.ResolutionAvailable ||
			occurrence.Target == nil ||
			occurrence.Target.Form != composition.TargetFormArtifact ||
			occurrence.Target.Artifact == nil {
			continue
		}

		pin, err := a.pinInvocationArtifact(
			ctx,
			*occurrence.Target.Artifact,
		)
		if err != nil {
			return err
		}

		switch occurrence.Type {
		case declaration.TypeModel:
			if output.Model == nil {
				value := pin
				output.Model = &value
			}
		case declaration.TypeText:
			appendPreparedWorkflowText(output, pin)
		case declaration.TypeTool:
			appendPreparedWorkflowTool(output, pin, nil)
		case declaration.TypeSkill:
			appendPreparedWorkflowSkill(output, pin, false, false)
		case declaration.TypeMCP:
			appendPreparedWorkflowMCP(output, pin)
		case declaration.TypeWorkspace:
			if output.Workspace == nil {
				value := pin
				output.Workspace = &value
			}
			if err := a.appendWorkspaceInvocationForPin(
				ctx,
				output,
				pin,
				visited,
			); err != nil {
				return err
			}
		default:
		}
	}
	return nil
}

func appendPreparedWorkflowText(
	output *sessionv3Runtime.WorkflowPreparedInvocation,
	pin sessionv3Runtime.ArtifactPin,
) {
	for _, value := range output.Texts {
		if value.Artifact.RootID == pin.RootID &&
			value.Artifact.ArtifactID == pin.ArtifactID {
			return
		}
	}
	output.Texts = append(output.Texts, sessionv3Runtime.WorkflowPreparedText{
		Artifact: pin,
	})
}

func appendPreparedWorkflowTool(
	output *sessionv3Runtime.WorkflowPreparedInvocation,
	pin sessionv3Runtime.ArtifactPin,
	autoExecute *bool,
) {
	for _, value := range output.Tools {
		if value.Artifact.RootID == pin.RootID &&
			value.Artifact.ArtifactID == pin.ArtifactID {
			return
		}
	}
	output.Tools = append(output.Tools, sessionv3Runtime.WorkflowPreparedTool{
		Artifact:    pin,
		AutoExecute: autoExecute,
	})
}

func appendPreparedWorkflowSkill(
	output *sessionv3Runtime.WorkflowPreparedInvocation,
	pin sessionv3Runtime.ArtifactPin,
	active bool,
	instructions bool,
) {
	for _, value := range output.Skills {
		if value.Artifact.RootID == pin.RootID &&
			value.Artifact.ArtifactID == pin.ArtifactID {
			return
		}
	}
	output.Skills = append(output.Skills, sessionv3Runtime.WorkflowPreparedSkill{
		Artifact:     pin,
		Active:       active,
		Instructions: instructions,
	})
}

func appendPreparedWorkflowMCP(
	output *sessionv3Runtime.WorkflowPreparedInvocation,
	pin sessionv3Runtime.ArtifactPin,
) {
	for _, value := range output.MCP {
		if value.Server.RootID == pin.RootID &&
			value.Server.ArtifactID == pin.ArtifactID {
			return
		}
	}
	output.MCP = append(output.MCP, sessionv3Runtime.WorkflowPreparedMCP{
		Server:                    pin,
		IncludeServerInstructions: true,
		ToolExposure:              sessionv3Runtime.MCPToolExposureAll,
	})
}

func (a *ModelAgentInvocationAdapter) pinInvocationArtifact(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (sessionv3Runtime.ArtifactPin, error) {
	values, err := a.artifacts.GetMany(ctx, []artifactModel.ArtifactRef{ref})
	if err != nil {
		return sessionv3Runtime.ArtifactPin{}, err
	}
	if len(values) != 1 ||
		values[0].Ref() != ref ||
		values[0].ResolvedDefinition == nil {
		return sessionv3Runtime.ArtifactPin{}, errors.New(
			"workflow invocation artifact pin is unavailable",
		)
	}
	return sessionv3Runtime.ArtifactPin{
		RootID:     string(values[0].RootID),
		ArtifactID: string(values[0].ID),
		Revision:   values[0].Revision,
		Digest:     string(*values[0].ResolvedDefinition),
	}, nil
}

func workflowInvocationArtifactRef(
	pin sessionv3Runtime.ArtifactPin,
) artifactModel.ArtifactRef {
	return artifactModel.ArtifactRef{
		RootID:     rootModel.RootID(pin.RootID),
		ArtifactID: artifactModel.ArtifactID(pin.ArtifactID),
	}
}
