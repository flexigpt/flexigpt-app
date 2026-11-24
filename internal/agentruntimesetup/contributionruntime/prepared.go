// Package contributionruntime prepares source-pinned Agent and Workspace
// contributions for Session V3 and Workflow setup. It owns no Runtime state,
// scheduler, provider client, Artifact Store execution API, or native request
// envelope.
package contributionruntime

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	agentAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	textAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/text"
	workspaceAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace"
)

const (
	preparedContributionKeySeparator = "\x00"
	canonicalSelectionKeyPrefix      = "artifact:"
	canonicalSelectionSeparator      = ":"
	workspaceInstructionsSuffix      = ":instructions"
	workspaceInstructionPrefix       = "workspace:"
)

// InstructionSkillRenderer is the Skill-domain boundary used during
// contribution preparation to render instruction-mode Skills.
type InstructionSkillRenderer interface {
	Instruction(
		ctx context.Context,
		selection sessionv3Runtime.ArtifactSelection,
	) (string, error)
}

// MCPContributionPreparer is the MCP-domain boundary used during contribution
// preparation to capture immutable MCP prompt context, discovery identity,
// policy, and provider Tool plans.
type MCPContributionPreparer interface {
	PrepareWorkflowMCPContribution(
		ctx context.Context,
		server sessionv3Runtime.ArtifactPin,
		selection sessionv3Runtime.MCPServerSelection,
		owner sessionv3Runtime.ConfigurationOwner,
	) (sessionv3Runtime.WorkflowPreparedMCP, error)
}

type Option func(*Preparer)

func WithInstructionSkillRenderer(renderer InstructionSkillRenderer) Option {
	return func(p *Preparer) {
		if p != nil {
			p.skillInstructions = renderer
		}
	}
}

func WithMCPContributionPreparer(preparer MCPContributionPreparer) Option {
	return func(p *Preparer) {
		if p != nil {
			p.mcpPreparer = preparer
		}
	}
}

type Request struct {
	Model              *sessionv3Runtime.ArtifactSelection
	Agent              *sessionv3Runtime.ArtifactSelection
	Workspace          *sessionv3Runtime.ArtifactSelection
	WorkspaceOverrides sessionv3Runtime.WorkspaceSelectionOverrides
	Defaults           *sessionv3Runtime.SessionDefaults
	DefaultsOverride   *sessionv3Runtime.SessionDefaultsOverride
	SpecificOwner      sessionv3Runtime.ConfigurationOwner
}

type Preparer struct {
	artifacts         artifact.API
	agents            *agentAPI.Service
	workspaces        *workspaceAPI.Service
	texts             *textAPI.Service
	skillInstructions InstructionSkillRenderer
	mcpPreparer       MCPContributionPreparer
}

func New(
	artifacts artifact.API,
	agents *agentAPI.Service,
	workspaces *workspaceAPI.Service,
	texts *textAPI.Service,
	options ...Option,
) (*Preparer, error) {
	if artifacts == nil || agents == nil || workspaces == nil || texts == nil {
		return nil, errors.New(
			"prepared contribution dependencies are incomplete",
		)
	}
	preparer := &Preparer{
		artifacts:  artifacts,
		agents:     agents,
		workspaces: workspaces,
		texts:      texts,
	}
	for _, option := range options {
		if option != nil {
			option(preparer)
		}
	}
	return preparer, nil
}

// SetDefaultMCPPreparer installs an MCP contribution preparer only when one
// was not explicitly supplied during construction.
func (p *Preparer) SetDefaultMCPPreparer(preparer MCPContributionPreparer) {
	if p != nil && p.mcpPreparer == nil {
		p.mcpPreparer = preparer
	}
}

func ArtifactRefForSelection(
	selection sessionv3Runtime.ArtifactSelection,
) artifactModel.ArtifactRef {
	return artifactModel.ArtifactRef{
		RootID:     rootModel.RootID(selection.RootID),
		ArtifactID: artifactModel.ArtifactID(selection.ArtifactID),
	}
}

func ArtifactRefForPin(
	pin sessionv3Runtime.ArtifactPin,
) artifactModel.ArtifactRef {
	return artifactModel.ArtifactRef{
		RootID:     rootModel.RootID(pin.RootID),
		ArtifactID: artifactModel.ArtifactID(pin.ArtifactID),
	}
}

func PinArtifactRef(
	ctx context.Context,
	artifacts artifact.API,
	ref artifactModel.ArtifactRef,
) (sessionv3Runtime.ArtifactPin, error) {
	if artifacts == nil {
		return sessionv3Runtime.ArtifactPin{}, errors.New(
			"artifact store reader is unavailable",
		)
	}
	if err := ref.Validate(); err != nil {
		return sessionv3Runtime.ArtifactPin{}, err
	}
	values, err := artifacts.GetMany(ctx, []artifactModel.ArtifactRef{ref})
	if err != nil {
		return sessionv3Runtime.ArtifactPin{}, err
	}
	if len(values) != 1 || values[0].Ref() != ref ||
		values[0].ResolvedDefinition == nil ||
		strings.TrimSpace(string(*values[0].ResolvedDefinition)) == "" ||
		values[0].State != artifactModel.StateAvailable ||
		!values[0].Enabled {
		return sessionv3Runtime.ArtifactPin{}, errors.New(
			"selected artifact is unavailable",
		)
	}
	return sessionv3Runtime.ArtifactPin{
		RootID:     string(values[0].RootID),
		ArtifactID: string(values[0].ID),
		Revision:   values[0].Revision,
		Digest:     string(*values[0].ResolvedDefinition),
	}, nil
}

func PinArtifactSelection(
	ctx context.Context,
	artifacts artifact.API,
	selection sessionv3Runtime.ArtifactSelection,
) (sessionv3Runtime.ArtifactPin, error) {
	pin, err := PinArtifactRef(ctx, artifacts, ArtifactRefForSelection(selection))
	if err != nil {
		return sessionv3Runtime.ArtifactPin{}, err
	}
	if selection.ExpectedRevision != 0 &&
		selection.ExpectedRevision != pin.Revision {
		return sessionv3Runtime.ArtifactPin{}, errors.New(
			"selected artifact revision changed",
		)
	}
	if selection.ExpectedDigest != "" &&
		selection.ExpectedDigest != pin.Digest {
		return sessionv3Runtime.ArtifactPin{}, errors.New(
			"selected artifact definition changed",
		)
	}
	return pin, nil
}

func VerifyArtifactPin(
	ctx context.Context,
	artifacts artifact.API,
	pin sessionv3Runtime.ArtifactPin,
) error {
	current, err := PinArtifactRef(ctx, artifacts, ArtifactRefForPin(pin))
	if err != nil {
		return err
	}
	if current != pin {
		return errors.New("artifact pin changed since preparation")
	}
	return nil
}

func (p *Preparer) Prepare(
	ctx context.Context,
	request Request,
) (sessionv3Runtime.PreparedContributionPlan, error) {
	if p == nil {
		return sessionv3Runtime.PreparedContributionPlan{}, errors.New(
			"prepared contribution runtime is unavailable",
		)
	}

	specific, hasSpecific, workspaceCleared, err := resolveSpecificDefaults(request)
	if err != nil {
		return sessionv3Runtime.PreparedContributionPlan{}, err
	}
	specificOwner := request.SpecificOwner
	if specificOwner == "" {
		if request.DefaultsOverride != nil && request.Defaults == nil {
			specificOwner = sessionv3Runtime.ConfigurationOwnerInvocation
		} else {
			specificOwner = sessionv3Runtime.ConfigurationOwnerSession
		}
	}
	specificMCPByKey := make(
		map[string]sessionv3Runtime.MCPServerSelection,
		len(specific.MCP),
	)
	handledSpecificMCP := make(map[string]struct{}, len(specific.MCP))
	for _, item := range specific.MCP {
		if isArtifactRemoved(
			specific.CollectionPolicies.MCP,
			sessionv3Runtime.PreparedContributionKindMCP,
			item.Server.RootID,
			item.Server.ArtifactID,
		) {
			continue
		}
		normalized := jsonutil.CloneJSONInValue(item)
		if normalized.ToolExposure == "" {
			normalized.ToolExposure = sessionv3Runtime.MCPToolExposureAll
		}
		specificMCPByKey[contributionSelectionKey(normalized.Server)] = normalized
	}

	selectedModel := request.Model
	if selectedModel == nil && specific.Model != nil {
		value := *specific.Model
		selectedModel = &value
	}
	selectedAgent := request.Agent
	if selectedAgent == nil && specific.Agent != nil {
		value := *specific.Agent
		selectedAgent = &value
	}

	output := sessionv3Runtime.PreparedContributionPlan{}
	if selectedModel != nil {
		modelPin, modelErr := p.pinSelection(ctx, *selectedModel)
		if modelErr != nil {
			return sessionv3Runtime.PreparedContributionPlan{}, modelErr
		}
		output.Model = &modelPin
		appendContributionOrder(
			&output,
			sessionv3Runtime.PreparedContributionKindModel,
			sessionv3Runtime.PreparedContributionKey(
				sessionv3Runtime.PreparedContributionKindModel,
				modelPin,
			),
			sessionv3Runtime.ConfigurationOwnerModel,
		)
	}

	var inheritedWorkspace *sessionv3Runtime.ArtifactSelection
	allowInheritedWorkspace := !workspaceCleared &&
		request.Workspace == nil &&
		specific.Workspace == nil
	if selectedAgent != nil {
		workspace, err := p.prepareAgent(
			ctx,
			&output,
			*selectedAgent,
			specific.CollectionPolicies,
			specificMCPByKey,
			handledSpecificMCP,
			specificOwner,
			allowInheritedWorkspace,
		)
		if err != nil {
			return sessionv3Runtime.PreparedContributionPlan{}, err
		}
		inheritedWorkspace = workspace
	}

	var workspace *sessionv3Runtime.ArtifactSelection
	switch {
	case specific.Workspace != nil:
		ws := *specific.Workspace
		workspace = &ws
	case request.Workspace != nil && !workspaceCleared:
		ws := *request.Workspace
		workspace = &ws
	case !workspaceCleared:
		workspace = inheritedWorkspace
	}

	workspaceOverrides := request.WorkspaceOverrides
	if hasSpecific {
		workspaceOverrides = specific.WorkspaceOverrides
	}
	if workspace != nil {
		if err := p.prepareWorkspace(
			ctx,
			&output,
			*workspace,
			workspaceOverrides,
			specific.CollectionPolicies,
			specificMCPByKey,
			handledSpecificMCP,
			specificOwner,
		); err != nil {
			return sessionv3Runtime.PreparedContributionPlan{}, err
		}
	}
	if hasSpecific {
		if err := p.applySpecificSelections(
			ctx,
			&output,
			specific,
			specificMCPByKey,
			handledSpecificMCP,
			specificOwner,
		); err != nil {
			return sessionv3Runtime.PreparedContributionPlan{}, err
		}
	}
	//nolint:gocritic // If else ok.
	if output.Agent != nil {
		output.OwnerProvenance = sessionv3Runtime.ConfigurationOwnerAgent
	} else if output.Workspace != nil {
		output.OwnerProvenance = sessionv3Runtime.ConfigurationOwnerWorkspace
	} else if output.Model != nil {
		output.OwnerProvenance = sessionv3Runtime.ConfigurationOwnerModel
	}
	if err := output.Validate(); err != nil {
		return sessionv3Runtime.PreparedContributionPlan{}, err
	}
	return output, nil
}

func (p *Preparer) prepareAgent(
	ctx context.Context,
	output *sessionv3Runtime.PreparedContributionPlan,
	selection sessionv3Runtime.ArtifactSelection,
	policies sessionv3Runtime.SessionCollectionPolicies,
	specificMCPByKey map[string]sessionv3Runtime.MCPServerSelection,
	handledSpecificMCP map[string]struct{},
	specificOwner sessionv3Runtime.ConfigurationOwner,
	allowInheritedWorkspace bool,
) (*sessionv3Runtime.ArtifactSelection, error) {
	pin, err := p.pinSelection(ctx, selection)
	if err != nil {
		return nil, err
	}
	ref := ArtifactRefForSelection(selection)
	resolved, err := p.agents.ResolveAgent(ctx, ref)
	if err != nil {
		return nil, err
	}
	if resolved.Agent.Ref != ref ||
		resolved.Agent.Revision != pin.Revision ||
		string(resolved.Agent.DefinitionDigest) != pin.Digest {
		return nil, errors.New("prepared agent changed during contribution preparation")
	}

	output.Agent = &pin
	appendContributionOrder(
		output,
		sessionv3Runtime.PreparedContributionKindAgent,
		sessionv3Runtime.PreparedContributionKey(
			sessionv3Runtime.PreparedContributionKindAgent,
			pin,
		),
		sessionv3Runtime.ConfigurationOwnerAgent,
	)

	var workspace *sessionv3Runtime.ArtifactSelection
	for _, occurrence := range resolved.Capabilities.Occurrences {
		if occurrence.Type == declaration.TypeModel && output.Model != nil {
			continue
		}
		if occurrence.Type == declaration.TypeWorkspace &&
			!allowInheritedWorkspace {
			continue
		}
		if preparedAgentOccurrenceExcluded(
			occurrence,
			policies,
		) {
			continue
		}
		if occurrence.Status != composition.ResolutionAvailable ||
			occurrence.Target == nil {
			if occurrence.Required {
				return nil, errors.New(
					"required agent contribution is unavailable",
				)
			}
			continue
		}
		targetPin, targetErr := p.pinCapabilityTarget(ctx, occurrence.Target)
		if targetErr != nil {
			if occurrence.Required {
				return nil, targetErr
			}
			continue
		}

		switch occurrence.Type {
		case declaration.TypeModel:
			value := targetPin
			output.Model = &value
			appendContributionOrder(
				output,
				sessionv3Runtime.PreparedContributionKindModel,
				sessionv3Runtime.PreparedContributionKey(
					sessionv3Runtime.PreparedContributionKindModel,
					targetPin,
				),
				sessionv3Runtime.ConfigurationOwnerAgent,
			)
			if occurrence.IncludeSystemPrompt != nil {
				value := *occurrence.IncludeSystemPrompt
				output.IncludeModelSystemPrompt = &value
			}

		case declaration.TypeText:
			if policies.SystemInstructions.Mode == sessionv3Runtime.CollectionOverrideReplace ||
				isArtifactRemoved(
					policies.SystemInstructions,
					sessionv3Runtime.PreparedContributionKindText,
					targetPin.RootID,
					targetPin.ArtifactID,
				) ||
				isArtifactRemoved(
					policies.InstructionArtifacts,
					sessionv3Runtime.PreparedContributionKindText,
					targetPin.RootID,
					targetPin.ArtifactID,
				) {
				continue
			}
			if err := p.appendText(
				ctx,
				output,
				targetPin,
				sessionv3Runtime.ConfigurationOwnerAgent,
				true,
				false,
			); err != nil {
				if occurrence.Required {
					return nil, err
				}
				continue
			}

		case declaration.TypeTool:
			if policies.ToolSelections.Mode == sessionv3Runtime.CollectionOverrideReplace ||
				isArtifactRemoved(
					policies.ToolSelections,
					sessionv3Runtime.PreparedContributionKindTool,
					targetPin.RootID,
					targetPin.ArtifactID,
				) {
				continue
			}
			appendPreparedTool(
				output,
				targetPin,
				nil,
				occurrence.AutoExecute,
				sessionv3Runtime.ConfigurationOwnerAgent,
			)

		case declaration.TypeSkill:
			instructions := occurrence.SkillUseMode == agentAPI.AgentSkillUseModeInstructions
			instructionContent := ""
			if instructions {
				if policies.SystemInstructions.Mode == sessionv3Runtime.CollectionOverrideReplace ||
					isArtifactRemoved(
						policies.SystemInstructions,
						sessionv3Runtime.PreparedContributionKindSkill,
						targetPin.RootID,
						targetPin.ArtifactID,
					) ||
					isArtifactRemoved(
						policies.Skills,
						sessionv3Runtime.PreparedContributionKindSkill,
						targetPin.RootID,
						targetPin.ArtifactID,
					) {
					continue
				}
				if p.skillInstructions == nil {
					if occurrence.Required {
						return nil, errors.New(
							"instruction skill renderer is unavailable",
						)
					}
					continue
				}
				rendered, renderErr := p.skillInstructions.Instruction(
					ctx,
					targetPin.Selection(),
				)
				if renderErr != nil {
					if occurrence.Required {
						return nil, renderErr
					}
					continue
				}
				if strings.TrimSpace(rendered) == "" {
					if occurrence.Required {
						return nil, errors.New(
							"instruction skill rendered no content",
						)
					}
					continue
				}
				instructionContent = rendered
			} else if policies.Skills.Mode == sessionv3Runtime.CollectionOverrideReplace ||
				isArtifactRemoved(
					policies.Skills,
					sessionv3Runtime.PreparedContributionKindSkill,
					targetPin.RootID,
					targetPin.ArtifactID,
				) {
				continue
			}
			active := occurrence.SkillUseMode == agentAPI.AgentSkillUseModeActive
			if active &&
				(policies.ActiveSkills.Mode == sessionv3Runtime.CollectionOverrideReplace ||
					isArtifactRemoved(
						policies.ActiveSkills,
						sessionv3Runtime.PreparedContributionKindSkill,
						targetPin.RootID,
						targetPin.ArtifactID,
					)) {
				active = false
			}
			appendPreparedSkill(
				output,
				targetPin,
				!instructions,
				active,
				instructions,
				instructionContent,
				sessionv3Runtime.ConfigurationOwnerAgent,
			)

		case declaration.TypeMCP:
			serverKey := targetPin.RootID + preparedContributionKeySeparator + targetPin.ArtifactID
			if overrideSelection, overridden := specificMCPByKey[serverKey]; overridden {
				overrideSelection.Server = targetPin.Selection()
				if err := p.appendMCP(
					ctx,
					output,
					targetPin,
					overrideSelection,
					specificOwner,
				); err != nil {
					return nil, err
				}
				handledSpecificMCP[serverKey] = struct{}{}
				continue
			}
			if policies.MCP.Mode == sessionv3Runtime.CollectionOverrideReplace ||
				isArtifactRemoved(
					policies.MCP,
					sessionv3Runtime.PreparedContributionKindMCP,
					targetPin.RootID,
					targetPin.ArtifactID,
				) {
				continue
			}
			if err := p.appendMCP(
				ctx,
				output,
				targetPin,
				sessionv3Runtime.MCPServerSelection{
					Server:                    targetPin.Selection(),
					IncludeServerInstructions: true,
					ToolExposure:              sessionv3Runtime.MCPToolExposureAll,
				},
				sessionv3Runtime.ConfigurationOwnerAgent,
			); err != nil {
				if occurrence.Required {
					return nil, err
				}
				continue
			}

		case declaration.TypeWorkspace:
			if workspace == nil {
				value := targetPin.Selection()
				workspace = &value
			}
		default:
		}
	}
	return workspace, nil
}

func (p *Preparer) prepareWorkspace(
	ctx context.Context,
	output *sessionv3Runtime.PreparedContributionPlan,
	selection sessionv3Runtime.ArtifactSelection,
	overrides sessionv3Runtime.WorkspaceSelectionOverrides,
	policies sessionv3Runtime.SessionCollectionPolicies,
	specificMCPByKey map[string]sessionv3Runtime.MCPServerSelection,
	handledSpecificMCP map[string]struct{},
	specificOwner sessionv3Runtime.ConfigurationOwner,
) error {
	workspacePin, err := p.pinSelection(ctx, selection)
	if err != nil {
		return err
	}
	workspaceRef := ArtifactRefForSelection(selection)
	capabilities, err := p.workspaces.ResolveWorkspaceCapabilities(
		ctx,
		workspaceRef,
	)
	if err != nil {
		return err
	}

	promptSelections, err := applyArtifactOverrides(
		capabilitySelections(capabilities, declaration.TypeText),
		overrides.Context,
	)
	if err != nil {
		return err
	}
	promptSelections = filterSelectionsByPolicy(
		promptSelections,
		policies.SystemInstructions,
		sessionv3Runtime.PreparedContributionKindText,
	)
	promptSelections = filterSelectionsRemoved(
		promptSelections,
		policies.InstructionArtifacts,
		sessionv3Runtime.PreparedContributionKindText,
	)
	skillSelections, err := applyArtifactOverrides(
		capabilitySelections(capabilities, declaration.TypeSkill),
		overrides.Skills,
	)
	if err != nil {
		return err
	}
	skillSelections = filterSelectionsByPolicy(
		skillSelections,
		policies.Skills,
		sessionv3Runtime.PreparedContributionKindSkill,
	)
	mcpSelections, err := applyMCPOverrides(
		workspaceMCPSelections(capabilities),
		overrides.MCP,
	)
	if err != nil {
		return err
	}
	mcpSelections = filterMCPSelectionsByPolicy(
		mcpSelections,
		policies.MCP,
		specificMCPByKey,
	)

	runtimePlan, err := p.workspaces.ResolveWorkspaceRuntimePlan(
		ctx,
		workspaceRef,
		workspaceAPI.WorkspaceRuntimeSelection{
			PromptArtifacts: contributionRefs(promptSelections),
			SkillArtifacts:  contributionRefs(skillSelections),
			MCPArtifacts:    contributionMCPRefs(mcpSelections),
		},
	)
	if err != nil {
		return err
	}
	if runtimePlan.Workspace.Artifact.Ref() != workspaceRef ||
		runtimePlan.Workspace.Artifact.Revision != workspacePin.Revision ||
		runtimePlan.Workspace.Artifact.ResolvedDefinition == nil ||
		string(*runtimePlan.Workspace.Artifact.ResolvedDefinition) !=
			workspacePin.Digest {
		return errors.New(
			"workspace runtime plan differs from its source-pinned workspace",
		)
	}

	output.Workspace = &workspacePin
	appendContributionOrder(
		output,
		sessionv3Runtime.PreparedContributionKindWorkspace,
		sessionv3Runtime.PreparedContributionKey(
			sessionv3Runtime.PreparedContributionKindWorkspace,
			workspacePin,
		),
		sessionv3Runtime.ConfigurationOwnerWorkspace,
	)
	appendContributionOrder(
		output,
		sessionv3Runtime.PreparedContributionKindWorkspacePrompt,
		sessionv3Runtime.PreparedWorkspacePromptKey(workspacePin),
		sessionv3Runtime.ConfigurationOwnerWorkspace,
	)

	for _, promptSelection := range promptSelections {
		pin, pinErr := p.pinSelection(ctx, promptSelection)
		if pinErr != nil {
			return pinErr
		}
		included := false
		for _, contribution := range runtimePlan.Prompt.Contributions {
			if contribution.Artifact != ArtifactRefForSelection(promptSelection) {
				continue
			}
			if contribution.ArtifactRevision != pin.Revision ||
				string(contribution.DefinitionDigest) != pin.Digest {
				return errors.New(
					"workspace prompt contribution differs from its source pin",
				)
			}
			included = true
			break
		}
		if err := p.appendText(
			ctx,
			output,
			pin,
			sessionv3Runtime.ConfigurationOwnerWorkspace,
			included,
			false,
		); err != nil {
			return err
		}
	}

	for _, skillSelection := range skillSelections {
		pin, pinErr := p.pinSelection(ctx, skillSelection)
		if pinErr != nil {
			return pinErr
		}
		included := false
		for _, skill := range runtimePlan.Skills.Skills {
			if skill.Artifact != ArtifactRefForSelection(skillSelection) {
				continue
			}
			if skill.ArtifactRevision != pin.Revision ||
				string(skill.DefinitionDigest) != pin.Digest {
				return errors.New(
					"workspace skill contribution differs from its source pin",
				)
			}
			included = true
			break
		}
		if included {
			appendPreparedSkill(
				output,
				pin,
				true,
				false,
				false,
				"",
				sessionv3Runtime.ConfigurationOwnerWorkspace,
			)
		}
	}

	for _, mcpSelection := range mcpSelections {
		pin, pinErr := p.pinSelection(ctx, mcpSelection.Server)
		if pinErr != nil {
			return pinErr
		}
		included := false
		for _, server := range runtimePlan.MCPServers.Servers {
			if server.Artifact != ArtifactRefForSelection(mcpSelection.Server) {
				continue
			}
			if server.ArtifactRevision != pin.Revision ||
				string(server.DefinitionDigest) != pin.Digest {
				return errors.New(
					"workspace mcp contribution differs from its source pin",
				)
			}
			included = true
			break
		}
		if included {
			serverKey := pin.RootID + preparedContributionKeySeparator + pin.ArtifactID
			owner := sessionv3Runtime.ConfigurationOwnerWorkspace
			if overrideSelection, overridden := specificMCPByKey[serverKey]; overridden {
				mcpSelection = overrideSelection
				owner = specificOwner
				handledSpecificMCP[serverKey] = struct{}{}
			}
			mcpSelection.Server = pin.Selection()
			if err := p.appendMCP(
				ctx,
				output,
				pin,
				mcpSelection,
				owner,
			); err != nil {
				return err
			}
		}
	}

	for _, toolSelection := range capabilitySelections(
		capabilities,
		declaration.TypeTool,
	) {
		if policies.ToolSelections.Mode == sessionv3Runtime.CollectionOverrideReplace ||
			isArtifactRemoved(
				policies.ToolSelections,
				sessionv3Runtime.PreparedContributionKindTool,
				toolSelection.RootID,
				toolSelection.ArtifactID,
			) {
			continue
		}
		pin, pinErr := p.pinSelection(ctx, toolSelection)
		if pinErr != nil {
			return pinErr
		}
		appendPreparedTool(
			output,
			pin,
			nil,
			nil,
			sessionv3Runtime.ConfigurationOwnerWorkspace,
		)
	}

	instructions := runtimePlan.Prompt.Instructions
	if policies.SystemInstructions.Mode == sessionv3Runtime.CollectionOverrideReplace ||
		slices.Contains(
			policies.SystemInstructions.Removed,
			sessionv3Runtime.PreparedWorkspacePromptKey(workspacePin),
		) {
		instructions = ""
	}
	prompt := &sessionv3Runtime.WorkflowPreparedWorkspacePrompt{
		Instructions: instructions,
		UserContext:  runtimePlan.Prompt.UserMessage,
	}
	output.WorkspacePrompt = prompt
	output.WorkspacePromptPrepared = true
	if strings.TrimSpace(prompt.Instructions) != "" {
		output.WorkspaceInstructions = append(
			output.WorkspaceInstructions,
			sessionv3Runtime.Instruction{
				ID: workspaceInstructionPrefix + workspacePin.RootID + canonicalSelectionSeparator +
					workspacePin.ArtifactID + workspaceInstructionsSuffix,
				Content:   prompt.Instructions,
				Owner:     sessionv3Runtime.ConfigurationOwnerWorkspace,
				SourceKey: sessionv3Runtime.PreparedWorkspacePromptKey(workspacePin),
			},
		)
	}
	if strings.TrimSpace(prompt.UserContext) != "" {
		output.WorkspaceContext = append(
			output.WorkspaceContext,
			prompt.UserContext,
		)
	}
	return nil
}

func (p *Preparer) applySpecificSelections(
	ctx context.Context,
	output *sessionv3Runtime.PreparedContributionPlan,
	specific sessionv3Runtime.SessionDefaults,
	specificMCPByKey map[string]sessionv3Runtime.MCPServerSelection,
	handledSpecificMCP map[string]struct{},
	specificOwner sessionv3Runtime.ConfigurationOwner,
) error {
	for _, selection := range specific.InstructionArtifacts {
		if isArtifactRemoved(
			specific.CollectionPolicies.InstructionArtifacts,
			sessionv3Runtime.PreparedContributionKindText,
			selection.RootID,
			selection.ArtifactID,
		) {
			continue
		}
		pin, err := p.pinSelection(ctx, selection)
		if err != nil {
			return err
		}
		if err := p.appendText(
			ctx,
			output,
			pin,
			specificOwner,
			true,
			true,
		); err != nil {
			return err
		}
	}

	for _, selection := range sessionv3Runtime.EffectiveToolSelections(specific) {
		if isArtifactRemoved(
			specific.CollectionPolicies.ToolSelections,
			sessionv3Runtime.PreparedContributionKindTool,
			selection.Artifact.RootID,
			selection.Artifact.ArtifactID,
		) {
			continue
		}
		pin, err := p.pinSelection(ctx, selection.Artifact)
		if err != nil {
			return err
		}
		replaceOrAppendPreparedTool(
			output,
			pin,
			selection.UserArguments,
			selection.AutoExecute,
			specificOwner,
		)
	}

	for _, selection := range specific.Skills {
		if isArtifactRemoved(
			specific.CollectionPolicies.Skills,
			sessionv3Runtime.PreparedContributionKindSkill,
			selection.RootID,
			selection.ArtifactID,
		) {
			continue
		}
		pin, err := p.pinSelection(ctx, selection)
		if err != nil {
			return err
		}
		enableOrAppendPreparedSkill(output, pin, specificOwner)
	}

	for _, selection := range specific.ActiveSkills {
		if isArtifactRemoved(
			specific.CollectionPolicies.ActiveSkills,
			sessionv3Runtime.PreparedContributionKindSkill,
			selection.RootID,
			selection.ArtifactID,
		) {
			continue
		}
		pin, err := p.pinSelection(ctx, selection)
		if err != nil {
			return err
		}
		if !activatePreparedSkill(output, pin) {
			return errors.New("active skill is not part of enabled skills")
		}
	}

	for _, selection := range specific.MCP {
		key := contributionSelectionKey(selection.Server)
		normalized, ok := specificMCPByKey[key]
		if !ok {
			continue
		}
		if _, alreadyHandled := handledSpecificMCP[key]; alreadyHandled {
			continue
		}
		pin, err := p.pinSelection(ctx, normalized.Server)
		if err != nil {
			return err
		}
		normalized.Server = pin.Selection()
		if err := p.appendMCP(
			ctx,
			output,
			pin,
			normalized,
			specificOwner,
		); err != nil {
			return err
		}
		handledSpecificMCP[key] = struct{}{}
	}
	return nil
}

func (p *Preparer) appendText(
	ctx context.Context,
	output *sessionv3Runtime.PreparedContributionPlan,
	pin sessionv3Runtime.ArtifactPin,
	owner sessionv3Runtime.ConfigurationOwner,
	included bool,
	requireInstructionsInsert bool,
) error {
	replaceIndex := -1
	for index := range output.Texts {
		current := &output.Texts[index]
		if current.Artifact.RootID == pin.RootID &&
			current.Artifact.ArtifactID == pin.ArtifactID {
			if current.Artifact != pin {
				return errors.New("text contribution differs from its source pin")
			}
			if requireInstructionsInsert &&
				current.Insert != sessionv3Runtime.TextDispatchInsertInstructions {
				return fmt.Errorf(
					"instruction artifact %q does not insert instructions",
					pin.ArtifactID,
				)
			}
			if included &&
				(!current.Included ||
					contributionOwnerRank(owner) >
						contributionOwnerRank(current.Owner)) {
				replaceIndex = index
				break
			}
			return nil
		}
	}

	value := sessionv3Runtime.WorkflowPreparedText{
		Artifact: pin,
		Owner:    owner,
		Included: included,
	}
	if included {
		ref := ArtifactRefForPin(pin)
		materialized, err := p.texts.Materialize(ctx, ref)
		if err != nil {
			return err
		}
		if materialized.Artifact != ref ||
			materialized.ArtifactRevision != pin.Revision ||
			string(materialized.DefinitionDigest) != pin.Digest {
			return errors.New(
				"text contribution differs from its source pin",
			)
		}
		insert, err := preparedTextInsert(string(materialized.Insert))
		if err != nil {
			return err
		}
		if requireInstructionsInsert &&
			insert != sessionv3Runtime.TextDispatchInsertInstructions {
			return fmt.Errorf(
				"instruction artifact %q does not insert instructions",
				pin.ArtifactID,
			)
		}
		value.Insert = insert
		value.Content = materialized.Content
		value.Label = string(materialized.Name)
	} else {
		value.Insert = sessionv3Runtime.TextDispatchInsertInstructions
	}
	if replaceIndex >= 0 {
		output.Texts[replaceIndex] = value
		replaceContributionOrder(output, sessionv3Runtime.PreparedContributionKindText,
			sessionv3Runtime.PreparedContributionKey(sessionv3Runtime.PreparedContributionKindText, pin), owner)
		return nil
	}
	output.Texts = append(output.Texts, value)
	appendContributionOrder(
		output,
		sessionv3Runtime.PreparedContributionKindText,
		sessionv3Runtime.PreparedContributionKey(
			sessionv3Runtime.PreparedContributionKindText,
			pin,
		),
		owner,
	)
	return nil
}

func (p *Preparer) pinCapabilityTarget(
	ctx context.Context,
	target *composition.CapabilityTarget,
) (sessionv3Runtime.ArtifactPin, error) {
	if target == nil ||
		target.Form != composition.TargetFormArtifact ||
		target.Artifact == nil {
		return sessionv3Runtime.ArtifactPin{}, errors.New(
			"prepared contribution target is not artifact-backed",
		)
	}
	return PinArtifactRef(ctx, p.artifacts, *target.Artifact)
}

func (p *Preparer) pinSelection(
	ctx context.Context,
	selection sessionv3Runtime.ArtifactSelection,
) (sessionv3Runtime.ArtifactPin, error) {
	return PinArtifactSelection(ctx, p.artifacts, selection)
}

func appendPreparedTool(
	output *sessionv3Runtime.PreparedContributionPlan,
	pin sessionv3Runtime.ArtifactPin,
	userArguments jsonutil.CanonicalValue,
	autoExecute *bool,
	owner sessionv3Runtime.ConfigurationOwner,
) {
	for _, current := range output.Tools {
		if current.Artifact.RootID == pin.RootID &&
			current.Artifact.ArtifactID == pin.ArtifactID {
			if contributionOwnerRank(owner) > contributionOwnerRank(current.Owner) {
				replaceOrAppendPreparedTool(
					output,
					pin,
					userArguments,
					autoExecute,
					owner,
				)
			}
			return
		}
	}
	var automatic *bool
	if autoExecute != nil {
		value := *autoExecute
		automatic = &value
	}
	output.Tools = append(output.Tools, sessionv3Runtime.WorkflowPreparedTool{
		Artifact:      pin,
		UserArguments: userArguments.Clone(),
		AutoExecute:   automatic,
		Owner:         owner,
	})
	appendContributionOrder(
		output,
		sessionv3Runtime.PreparedContributionKindTool,
		sessionv3Runtime.PreparedContributionKey(
			sessionv3Runtime.PreparedContributionKindTool,
			pin,
		),
		owner,
	)
}

func replaceOrAppendPreparedTool(
	output *sessionv3Runtime.PreparedContributionPlan,
	pin sessionv3Runtime.ArtifactPin,
	userArguments jsonutil.CanonicalValue,
	autoExecute *bool,
	owner sessionv3Runtime.ConfigurationOwner,
) {
	var automatic *bool
	if autoExecute != nil {
		value := *autoExecute
		automatic = &value
	}
	for index := range output.Tools {
		if output.Tools[index].Artifact.RootID == pin.RootID &&
			output.Tools[index].Artifact.ArtifactID == pin.ArtifactID {
			output.Tools[index] = sessionv3Runtime.WorkflowPreparedTool{
				Artifact:      pin,
				UserArguments: userArguments.Clone(),
				AutoExecute:   automatic,
				Owner:         owner,
			}
			replaceContributionOrder(
				output,
				sessionv3Runtime.PreparedContributionKindTool,
				sessionv3Runtime.PreparedContributionKey(sessionv3Runtime.PreparedContributionKindTool, pin),
				owner,
			)
			return
		}
	}
	appendPreparedTool(output, pin, userArguments, automatic, owner)
}

func appendPreparedSkill(
	output *sessionv3Runtime.PreparedContributionPlan,
	pin sessionv3Runtime.ArtifactPin,
	enabled bool,
	active bool,
	instructions bool,
	instructionContent string,
	owner sessionv3Runtime.ConfigurationOwner,
) {
	for _, current := range output.Skills {
		if current.Artifact.RootID == pin.RootID &&
			current.Artifact.ArtifactID == pin.ArtifactID {
			if contributionOwnerRank(owner) > contributionOwnerRank(current.Owner) {
				enableOrAppendPreparedSkill(output, pin, owner)
				output.Skills[len(output.Skills)-1].Instructions = instructions
				output.Skills[len(output.Skills)-1].InstructionContent = instructionContent
				output.Skills[len(output.Skills)-1].Active = active
			}
			return
		}
	}
	output.Skills = append(output.Skills, sessionv3Runtime.WorkflowPreparedSkill{
		Artifact:           pin,
		Enabled:            enabled,
		Active:             active,
		Instructions:       instructions,
		InstructionContent: instructionContent,
		Owner:              owner,
	})
	appendContributionOrder(
		output,
		sessionv3Runtime.PreparedContributionKindSkill,
		sessionv3Runtime.PreparedContributionKey(
			sessionv3Runtime.PreparedContributionKindSkill,
			pin,
		),
		owner,
	)
}

func enableOrAppendPreparedSkill(
	output *sessionv3Runtime.PreparedContributionPlan,
	pin sessionv3Runtime.ArtifactPin,
	owner sessionv3Runtime.ConfigurationOwner,
) {
	for index := range output.Skills {
		if output.Skills[index].Artifact.RootID != pin.RootID ||
			output.Skills[index].Artifact.ArtifactID != pin.ArtifactID {
			continue
		}
		output.Skills[index].Artifact = pin
		output.Skills[index].Enabled = true
		output.Skills[index].Owner = owner
		replaceContributionOrder(
			output,
			sessionv3Runtime.PreparedContributionKindSkill,
			sessionv3Runtime.PreparedContributionKey(sessionv3Runtime.PreparedContributionKindSkill, pin),
			owner,
		)
		return
	}
	appendPreparedSkill(output, pin, true, false, false, "", owner)
}

func activatePreparedSkill(
	output *sessionv3Runtime.PreparedContributionPlan,
	pin sessionv3Runtime.ArtifactPin,
) bool {
	for index := range output.Skills {
		current := &output.Skills[index]
		if current.Artifact.RootID == pin.RootID &&
			current.Artifact.ArtifactID == pin.ArtifactID &&
			(!current.Instructions || current.Enabled) {
			current.Active = true
			return true
		}
	}
	return false
}

func (p *Preparer) appendMCP(
	ctx context.Context,
	output *sessionv3Runtime.PreparedContributionPlan,
	pin sessionv3Runtime.ArtifactPin,
	selection sessionv3Runtime.MCPServerSelection,
	owner sessionv3Runtime.ConfigurationOwner,
) error {
	replaceIndex := -1
	for index, current := range output.MCP {
		if current.Server.RootID == pin.RootID &&
			current.Server.ArtifactID == pin.ArtifactID {
			if contributionOwnerRank(owner) <= contributionOwnerRank(current.Owner) {
				return nil
			}
			replaceIndex = index
			break
		}
	}
	if p == nil || p.mcpPreparer == nil {
		return errors.New("mcp contribution preparer is unavailable")
	}
	prepared, err := p.mcpPreparer.PrepareWorkflowMCPContribution(
		ctx,
		pin,
		selection,
		owner,
	)
	if err != nil {
		return err
	}
	if replaceIndex >= 0 {
		output.MCP[replaceIndex] = prepared
		replaceContributionOrder(
			output,
			sessionv3Runtime.PreparedContributionKindMCP,
			sessionv3Runtime.PreparedContributionKey(sessionv3Runtime.PreparedContributionKindMCP, pin),
			owner,
		)
	} else {
		output.MCP = append(output.MCP, prepared)
	}
	appendContributionOrder(
		output,
		sessionv3Runtime.PreparedContributionKindMCP,
		sessionv3Runtime.PreparedContributionKey(
			sessionv3Runtime.PreparedContributionKindMCP,
			pin,
		),
		owner,
	)
	return nil
}

func contributionOwnerRank(value sessionv3Runtime.ConfigurationOwner) int {
	switch value {
	case sessionv3Runtime.ConfigurationOwnerInvocation:
		return 5
	case sessionv3Runtime.ConfigurationOwnerSession:
		return 4
	case sessionv3Runtime.ConfigurationOwnerWorkspace:
		return 3
	case sessionv3Runtime.ConfigurationOwnerAgent:
		return 2
	case sessionv3Runtime.ConfigurationOwnerModel:
		return 1
	default:
		return 0
	}
}

func replaceContributionOrder(
	output *sessionv3Runtime.PreparedContributionPlan,
	kind sessionv3Runtime.PreparedContributionKind,
	key string,
	owner sessionv3Runtime.ConfigurationOwner,
) {
	for index := range output.ContributionOrder {
		if output.ContributionOrder[index].Kind != kind ||
			output.ContributionOrder[index].Key != key {
			continue
		}
		output.ContributionOrder = append(
			output.ContributionOrder[:index],
			output.ContributionOrder[index+1:]...,
		)
		break
	}
	output.ContributionOrder = append(output.ContributionOrder,
		sessionv3Runtime.PreparedContributionOrder{Kind: kind, Key: key, Owner: owner})
}

func appendContributionOrder(
	output *sessionv3Runtime.PreparedContributionPlan,
	kind sessionv3Runtime.PreparedContributionKind,
	key string,
	owner sessionv3Runtime.ConfigurationOwner,
) {
	if output == nil {
		return
	}
	for _, current := range output.ContributionOrder {
		if current.Kind == kind && current.Key == key {
			return
		}
	}
	replaceContributionOrder(output, kind, key, owner)
}

func preparedAgentOccurrenceExcluded(
	occurrence agentAPI.AgentCapabilityOccurrence,
	policies sessionv3Runtime.SessionCollectionPolicies,
) bool {
	if occurrence.Target == nil ||
		occurrence.Target.Form != composition.TargetFormArtifact ||
		occurrence.Target.Artifact == nil {
		return false
	}
	rootID := string(occurrence.Target.Artifact.RootID)
	artifactID := string(occurrence.Target.Artifact.ArtifactID)
	switch occurrence.Type {
	case declaration.TypeText:
		return policies.SystemInstructions.Mode == sessionv3Runtime.CollectionOverrideReplace ||
			policies.InstructionArtifacts.Mode == sessionv3Runtime.CollectionOverrideReplace ||
			isArtifactRemoved(
				policies.SystemInstructions,
				sessionv3Runtime.PreparedContributionKindText,
				rootID,
				artifactID,
			) ||
			isArtifactRemoved(
				policies.InstructionArtifacts,
				sessionv3Runtime.PreparedContributionKindText,
				rootID,
				artifactID,
			)
	case declaration.TypeTool:
		return policies.ToolSelections.Mode == sessionv3Runtime.CollectionOverrideReplace ||
			isArtifactRemoved(
				policies.ToolSelections,
				sessionv3Runtime.PreparedContributionKindTool,
				rootID,
				artifactID,
			)
	case declaration.TypeSkill:
		if occurrence.SkillUseMode == agentAPI.AgentSkillUseModeInstructions {
			return policies.SystemInstructions.Mode == sessionv3Runtime.CollectionOverrideReplace ||
				isArtifactRemoved(
					policies.SystemInstructions,
					sessionv3Runtime.PreparedContributionKindSkill,
					rootID,
					artifactID,
				)
		}
		return policies.Skills.Mode == sessionv3Runtime.CollectionOverrideReplace ||
			isArtifactRemoved(policies.Skills, sessionv3Runtime.PreparedContributionKindSkill, rootID, artifactID)
	case declaration.TypeMCP:
		return policies.MCP.Mode == sessionv3Runtime.CollectionOverrideReplace ||
			isArtifactRemoved(policies.MCP, sessionv3Runtime.PreparedContributionKindMCP, rootID, artifactID)
	default:
		return false
	}
}

func capabilitySelections(
	plan composition.CapabilityPlan,
	kind declaration.Type,
) []sessionv3Runtime.ArtifactSelection {
	output := make([]sessionv3Runtime.ArtifactSelection, 0)
	seen := make(map[string]struct{})
	for _, occurrence := range plan.Occurrences {
		if occurrence.Status != composition.ResolutionAvailable ||
			occurrence.Type != kind ||
			occurrence.Target == nil ||
			occurrence.Target.Form != composition.TargetFormArtifact ||
			occurrence.Target.Artifact == nil {
			continue
		}
		value := sessionv3Runtime.ArtifactSelection{
			RootID:     string(occurrence.Target.Artifact.RootID),
			ArtifactID: string(occurrence.Target.Artifact.ArtifactID),
		}
		key := contributionSelectionKey(value)
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		output = append(output, value)
	}
	return output
}

func workspaceMCPSelections(
	plan composition.CapabilityPlan,
) []sessionv3Runtime.MCPServerSelection {
	values := capabilitySelections(plan, declaration.TypeMCP)
	output := make([]sessionv3Runtime.MCPServerSelection, 0, len(values))
	for _, value := range values {
		output = append(output, sessionv3Runtime.MCPServerSelection{
			Server:       value,
			ToolExposure: sessionv3Runtime.MCPToolExposureAll,
		})
	}
	return output
}

func applyArtifactOverrides(
	baseline []sessionv3Runtime.ArtifactSelection,
	overrides []sessionv3Runtime.ArtifactOverride,
) ([]sessionv3Runtime.ArtifactSelection, error) {
	output := append([]sessionv3Runtime.ArtifactSelection(nil), baseline...)
	replaced := false
	for _, override := range overrides {
		switch override.Mode {
		case sessionv3Runtime.CollectionOverrideInherit:
		case sessionv3Runtime.CollectionOverrideRemove:
			output = removeArtifactSelection(output, override.Key)
		case sessionv3Runtime.CollectionOverrideReplace:
			if !replaced {
				output = nil
				replaced = true
			}
			fallthrough
		case sessionv3Runtime.CollectionOverrideAdd:
			if override.Value == nil {
				return nil, errors.New("workspace contribution override has no value")
			}
			if !containsArtifactSelection(output, *override.Value) {
				output = append(output, *override.Value)
			}
		default:
			return nil, errors.New("workspace contribution override mode is invalid")
		}
	}
	return output, nil
}

func applyMCPOverrides(
	baseline []sessionv3Runtime.MCPServerSelection,
	overrides []sessionv3Runtime.MCPServerOverride,
) ([]sessionv3Runtime.MCPServerSelection, error) {
	output := append([]sessionv3Runtime.MCPServerSelection(nil), baseline...)
	replaced := false
	for _, override := range overrides {
		switch override.Mode {
		case sessionv3Runtime.CollectionOverrideInherit:
		case sessionv3Runtime.CollectionOverrideRemove:
			output = removeMCPSelection(output, override.Key)
		case sessionv3Runtime.CollectionOverrideReplace:
			if !replaced {
				output = nil
				replaced = true
			}
			fallthrough
		case sessionv3Runtime.CollectionOverrideAdd:
			if override.Value == nil {
				return nil, errors.New("workspace mcp contribution override has no value")
			}
			if !containsMCPSelection(output, *override.Value) {
				output = append(output, *override.Value)
			}
		default:
			return nil, errors.New("workspace mcp contribution override mode is invalid")
		}
	}
	return output, nil
}

func contributionSelectionKey(value sessionv3Runtime.ArtifactSelection) string {
	return value.RootID + preparedContributionKeySeparator + value.ArtifactID
}

func containsArtifactSelection(
	values []sessionv3Runtime.ArtifactSelection,
	want sessionv3Runtime.ArtifactSelection,
) bool {
	for _, value := range values {
		if contributionSelectionKey(value) == contributionSelectionKey(want) {
			return true
		}
	}
	return false
}

func removeArtifactSelection(
	values []sessionv3Runtime.ArtifactSelection,
	key string,
) []sessionv3Runtime.ArtifactSelection {
	output := make([]sessionv3Runtime.ArtifactSelection, 0, len(values))
	for _, value := range values {
		if key == contributionSelectionKey(value) ||
			key == canonicalSelectionKeyPrefix+value.RootID+canonicalSelectionSeparator+value.ArtifactID {
			continue
		}
		output = append(output, value)
	}
	return output
}

func containsMCPSelection(
	values []sessionv3Runtime.MCPServerSelection,
	want sessionv3Runtime.MCPServerSelection,
) bool {
	for _, value := range values {
		if contributionSelectionKey(value.Server) ==
			contributionSelectionKey(want.Server) {
			return true
		}
	}
	return false
}

func removeMCPSelection(
	values []sessionv3Runtime.MCPServerSelection,
	key string,
) []sessionv3Runtime.MCPServerSelection {
	output := make([]sessionv3Runtime.MCPServerSelection, 0, len(values))
	for _, value := range values {
		if key == contributionSelectionKey(value.Server) ||
			key == canonicalSelectionKeyPrefix+value.Server.RootID+canonicalSelectionSeparator+value.Server.ArtifactID {
			continue
		}
		output = append(output, value)
	}
	return output
}

func contributionRefs(
	values []sessionv3Runtime.ArtifactSelection,
) []artifactModel.ArtifactRef {
	output := make([]artifactModel.ArtifactRef, 0, len(values))
	for _, value := range values {
		output = append(output, ArtifactRefForSelection(value))
	}
	return output
}

func contributionMCPRefs(
	values []sessionv3Runtime.MCPServerSelection,
) []artifactModel.ArtifactRef {
	output := make([]artifactModel.ArtifactRef, 0, len(values))
	for _, value := range values {
		output = append(output, ArtifactRefForSelection(value.Server))
	}
	return output
}

func resolveSpecificDefaults(
	request Request,
) (specific sessionv3Runtime.SessionDefaults, hasSpecific, worlspaceCleared bool, err error) {
	if request.Defaults == nil && request.DefaultsOverride == nil {
		return sessionv3Runtime.SessionDefaults{}, false, false, nil
	}
	base := sessionv3Runtime.SessionDefaults{}
	if request.Defaults != nil {
		base = jsonutil.CloneJSONInValue(*request.Defaults)
	}
	workspaceCleared := false
	if request.DefaultsOverride != nil {
		workspaceCleared = slices.Contains(
			request.DefaultsOverride.Clear,
			sessionv3Runtime.SessionDefaultFieldWorkspace,
		)
	}
	resolved, err := sessionv3Runtime.ResolveSessionInvocationDefaults(
		base,
		request.DefaultsOverride,
	)
	if err != nil {
		return sessionv3Runtime.SessionDefaults{}, false, false, err
	}
	return resolved, true, workspaceCleared, nil
}

func isArtifactRemoved(
	policy sessionv3Runtime.CollectionPolicy,
	kind sessionv3Runtime.PreparedContributionKind,
	rootID string,
	artifactID string,
) bool {
	if len(policy.Removed) == 0 {
		return false
	}
	pin := sessionv3Runtime.ArtifactPin{RootID: rootID, ArtifactID: artifactID}
	rawKey := rootID + preparedContributionKeySeparator + artifactID
	canonicalKey := canonicalSelectionKeyPrefix + rootID + canonicalSelectionSeparator + artifactID
	contributionKey := sessionv3Runtime.PreparedContributionKey(kind, pin)
	for _, removed := range policy.Removed {
		if removed == rawKey || removed == canonicalKey || removed == contributionKey {
			return true
		}
	}
	return false
}

func filterSelectionsByPolicy(
	values []sessionv3Runtime.ArtifactSelection,
	policy sessionv3Runtime.CollectionPolicy,
	kind sessionv3Runtime.PreparedContributionKind,
) []sessionv3Runtime.ArtifactSelection {
	if policy.Mode == sessionv3Runtime.CollectionOverrideReplace {
		return nil
	}
	return filterSelectionsRemoved(values, policy, kind)
}

func filterSelectionsRemoved(
	values []sessionv3Runtime.ArtifactSelection,
	policy sessionv3Runtime.CollectionPolicy,
	kind sessionv3Runtime.PreparedContributionKind,
) []sessionv3Runtime.ArtifactSelection {
	if len(policy.Removed) == 0 {
		return values
	}
	output := make([]sessionv3Runtime.ArtifactSelection, 0, len(values))
	for _, value := range values {
		if !isArtifactRemoved(policy, kind, value.RootID, value.ArtifactID) {
			output = append(output, value)
		}
	}
	return output
}

func filterMCPSelectionsByPolicy(
	values []sessionv3Runtime.MCPServerSelection,
	policy sessionv3Runtime.CollectionPolicy,
	specificMCPByKey map[string]sessionv3Runtime.MCPServerSelection,
) []sessionv3Runtime.MCPServerSelection {
	if policy.Mode == sessionv3Runtime.CollectionOverrideReplace {
		return nil
	}
	output := make([]sessionv3Runtime.MCPServerSelection, 0, len(values))
	for _, value := range values {
		if isArtifactRemoved(
			policy,
			sessionv3Runtime.PreparedContributionKindMCP,
			value.Server.RootID,
			value.Server.ArtifactID,
		) {
			continue
		}
		if override, ok := specificMCPByKey[contributionSelectionKey(value.Server)]; ok {
			value = override
		}
		output = append(output, value)
	}
	return output
}

func preparedTextInsert(
	value string,
) (sessionv3Runtime.TextDispatchInsert, error) {
	switch value {
	case string(sessionv3Runtime.TextDispatchInsertInstructions):
		return sessionv3Runtime.TextDispatchInsertInstructions, nil
	case string(sessionv3Runtime.TextDispatchInsertUserMessage):
		return sessionv3Runtime.TextDispatchInsertUserMessage, nil
	default:
		return "", fmt.Errorf("prepared text insert %q is invalid", value)
	}
}
