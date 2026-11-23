// Package contributionruntime prepares source-pinned Agent and Workspace
// contributions for Session V3 and Workflow setup. It owns no Runtime state,
// scheduler, provider client, Artifact Store execution API, or native request
// envelope.
package contributionruntime

import (
	"context"
	"errors"
	"fmt"
	"strings"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	agentAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	textAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/text"
	workspaceAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace"
)

const preparedContributionKeySeparator = "\x00"

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
	Agent              *sessionv3Runtime.ArtifactSelection
	Workspace          *sessionv3Runtime.ArtifactSelection
	WorkspaceOverrides sessionv3Runtime.WorkspaceSelectionOverrides
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

func (p *Preparer) Prepare(
	ctx context.Context,
	request Request,
) (sessionv3Runtime.PreparedContributionPlan, error) {
	if p == nil {
		return sessionv3Runtime.PreparedContributionPlan{}, errors.New(
			"prepared contribution runtime is unavailable",
		)
	}

	output := sessionv3Runtime.PreparedContributionPlan{}
	var inheritedWorkspace *sessionv3Runtime.ArtifactSelection
	if request.Agent != nil {
		workspace, err := p.prepareAgent(ctx, &output, *request.Agent)
		if err != nil {
			return sessionv3Runtime.PreparedContributionPlan{}, err
		}
		inheritedWorkspace = workspace
	}

	workspace := request.Workspace
	if workspace == nil {
		workspace = inheritedWorkspace
	}
	if workspace != nil {
		if err := p.prepareWorkspace(
			ctx,
			&output,
			*workspace,
			request.WorkspaceOverrides,
		); err != nil {
			return sessionv3Runtime.PreparedContributionPlan{}, err
		}
	}
	if output.Agent != nil {
		output.OwnerProvenance = sessionv3Runtime.ConfigurationOwnerAgent
	} else if output.Workspace != nil {
		output.OwnerProvenance = sessionv3Runtime.ConfigurationOwnerWorkspace
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
) (*sessionv3Runtime.ArtifactSelection, error) {
	pin, err := p.pinSelection(ctx, selection)
	if err != nil {
		return nil, err
	}
	ref := contributionArtifactRef(selection)
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
			if output.Model == nil {
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
			}
			if occurrence.IncludeSystemPrompt != nil {
				value := *occurrence.IncludeSystemPrompt
				output.IncludeModelSystemPrompt = &value
			}

		case declaration.TypeText:
			if err := p.appendText(
				ctx,
				output,
				targetPin,
				sessionv3Runtime.ConfigurationOwnerAgent,
				true,
			); err != nil {
				return nil, err
			}

		case declaration.TypeTool:
			appendPreparedTool(
				output,
				targetPin,
				occurrence.AutoExecute,
				sessionv3Runtime.ConfigurationOwnerAgent,
			)

		case declaration.TypeSkill:
			instructions := occurrence.SkillUseMode == agentAPI.AgentSkillUseModeInstructions
			instructionContent := ""
			if instructions {
				if p.skillInstructions == nil {
					return nil, errors.New(
						"instruction skill renderer is unavailable",
					)
				}
				rendered, renderErr := p.skillInstructions.Instruction(
					ctx,
					selectionForPin(targetPin),
				)
				if renderErr != nil {
					return nil, renderErr
				}
				if strings.TrimSpace(rendered) == "" {
					return nil, errors.New(
						"instruction skill rendered no content",
					)
				}
				instructionContent = rendered
			}
			appendPreparedSkill(
				output,
				targetPin,
				occurrence.SkillUseMode == agentAPI.AgentSkillUseModeActive,
				instructions,
				instructionContent,
				sessionv3Runtime.ConfigurationOwnerAgent,
			)

		case declaration.TypeMCP:
			if err := p.appendMCP(
				ctx,
				output,
				targetPin,
				sessionv3Runtime.MCPServerSelection{
					Server:                    selectionForPin(targetPin),
					IncludeServerInstructions: true,
					ToolExposure:              sessionv3Runtime.MCPToolExposureAll,
				},
				sessionv3Runtime.ConfigurationOwnerAgent,
			); err != nil {
				return nil, err
			}

		case declaration.TypeWorkspace:
			if workspace == nil {
				value := selectionForPin(targetPin)
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
) error {
	workspacePin, err := p.pinSelection(ctx, selection)
	if err != nil {
		return err
	}
	workspaceRef := contributionArtifactRef(selection)
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
	skillSelections, err := applyArtifactOverrides(
		capabilitySelections(capabilities, declaration.TypeSkill),
		overrides.Skills,
	)
	if err != nil {
		return err
	}
	mcpSelections, err := applyMCPOverrides(
		workspaceMCPSelections(capabilities),
		overrides.MCP,
	)
	if err != nil {
		return err
	}

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
			if contribution.Artifact != contributionArtifactRef(promptSelection) {
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
			if skill.Artifact != contributionArtifactRef(skillSelection) {
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
			if server.Artifact != contributionArtifactRef(mcpSelection.Server) {
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
			mcpSelection.Server = selectionForPin(pin)
			if err := p.appendMCP(
				ctx,
				output,
				pin,
				mcpSelection,
				sessionv3Runtime.ConfigurationOwnerWorkspace,
			); err != nil {
				return err
			}
		}
	}

	for _, toolSelection := range capabilitySelections(
		capabilities,
		declaration.TypeTool,
	) {
		pin, pinErr := p.pinSelection(ctx, toolSelection)
		if pinErr != nil {
			return pinErr
		}
		appendPreparedTool(
			output,
			pin,
			nil,
			sessionv3Runtime.ConfigurationOwnerWorkspace,
		)
	}

	prompt := &sessionv3Runtime.WorkflowPreparedWorkspacePrompt{
		Instructions: runtimePlan.Prompt.Instructions,
		UserContext:  runtimePlan.Prompt.UserMessage,
	}
	output.WorkspacePrompt = prompt
	output.WorkspacePromptPrepared = true
	if strings.TrimSpace(prompt.Instructions) != "" {
		output.WorkspaceInstructions = append(
			output.WorkspaceInstructions,
			sessionv3Runtime.Instruction{
				ID: "workspace:" + workspacePin.RootID + ":" +
					workspacePin.ArtifactID + ":instructions",
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

func (p *Preparer) appendText(
	ctx context.Context,
	output *sessionv3Runtime.PreparedContributionPlan,
	pin sessionv3Runtime.ArtifactPin,
	owner sessionv3Runtime.ConfigurationOwner,
	included bool,
) error {
	for _, current := range output.Texts {
		if current.Artifact.RootID == pin.RootID &&
			current.Artifact.ArtifactID == pin.ArtifactID {
			return nil
		}
	}

	value := sessionv3Runtime.WorkflowPreparedText{
		Artifact: pin,
		Owner:    owner,
		Included: included,
	}
	if included {
		ref := contributionArtifactRef(selectionForPin(pin))
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
		value.Insert = insert
		value.Content = materialized.Content
		value.Label = string(materialized.Name)
	} else {
		value.Insert = sessionv3Runtime.TextDispatchInsertInstructions
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
	return p.pinArtifactRef(ctx, *target.Artifact)
}

func (p *Preparer) pinSelection(
	ctx context.Context,
	selection sessionv3Runtime.ArtifactSelection,
) (sessionv3Runtime.ArtifactPin, error) {
	pin, err := p.pinArtifactRef(ctx, contributionArtifactRef(selection))
	if err != nil {
		return sessionv3Runtime.ArtifactPin{}, err
	}
	if selection.ExpectedRevision != 0 &&
		selection.ExpectedRevision != pin.Revision {
		return sessionv3Runtime.ArtifactPin{}, errors.New(
			"selected contribution artifact revision changed",
		)
	}
	if selection.ExpectedDigest != "" &&
		selection.ExpectedDigest != pin.Digest {
		return sessionv3Runtime.ArtifactPin{}, errors.New(
			"selected contribution artifact definition changed",
		)
	}
	return pin, nil
}

func (p *Preparer) pinArtifactRef(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (sessionv3Runtime.ArtifactPin, error) {
	if err := ref.Validate(); err != nil {
		return sessionv3Runtime.ArtifactPin{}, err
	}
	values, err := p.artifacts.GetMany(ctx, []artifactModel.ArtifactRef{ref})
	if err != nil {
		return sessionv3Runtime.ArtifactPin{}, err
	}
	if len(values) != 1 || values[0].Ref() != ref ||
		values[0].ResolvedDefinition == nil ||
		values[0].State != artifactModel.StateAvailable ||
		!values[0].Enabled {
		return sessionv3Runtime.ArtifactPin{}, errors.New(
			"selected contribution artifact is unavailable",
		)
	}
	return sessionv3Runtime.ArtifactPin{
		RootID:     string(values[0].RootID),
		ArtifactID: string(values[0].ID),
		Revision:   values[0].Revision,
		Digest:     string(*values[0].ResolvedDefinition),
	}, nil
}

func appendPreparedTool(
	output *sessionv3Runtime.PreparedContributionPlan,
	pin sessionv3Runtime.ArtifactPin,
	autoExecute *bool,
	owner sessionv3Runtime.ConfigurationOwner,
) {
	for _, current := range output.Tools {
		if current.Artifact.RootID == pin.RootID &&
			current.Artifact.ArtifactID == pin.ArtifactID {
			return
		}
	}
	var automatic *bool
	if autoExecute != nil {
		value := *autoExecute
		automatic = &value
	}
	output.Tools = append(output.Tools, sessionv3Runtime.WorkflowPreparedTool{
		Artifact:    pin,
		AutoExecute: automatic,
		Owner:       owner,
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

func appendPreparedSkill(
	output *sessionv3Runtime.PreparedContributionPlan,
	pin sessionv3Runtime.ArtifactPin,
	active bool,
	instructions bool,
	instructionContent string,
	owner sessionv3Runtime.ConfigurationOwner,
) {
	for _, current := range output.Skills {
		if current.Artifact.RootID == pin.RootID &&
			current.Artifact.ArtifactID == pin.ArtifactID {
			return
		}
	}
	output.Skills = append(output.Skills, sessionv3Runtime.WorkflowPreparedSkill{
		Artifact:           pin,
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

func (p *Preparer) appendMCP(
	ctx context.Context,
	output *sessionv3Runtime.PreparedContributionPlan,
	pin sessionv3Runtime.ArtifactPin,
	selection sessionv3Runtime.MCPServerSelection,
	owner sessionv3Runtime.ConfigurationOwner,
) error {
	for _, current := range output.MCP {
		if current.Server.RootID == pin.RootID &&
			current.Server.ArtifactID == pin.ArtifactID {
			return nil
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
	output.MCP = append(output.MCP, prepared)
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
	output.ContributionOrder = append(
		output.ContributionOrder,
		sessionv3Runtime.PreparedContributionOrder{
			Kind:  kind,
			Key:   key,
			Owner: owner,
		},
	)
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

func contributionArtifactRef(
	selection sessionv3Runtime.ArtifactSelection,
) artifactModel.ArtifactRef {
	return artifactModel.ArtifactRef{
		RootID:     rootModel.RootID(selection.RootID),
		ArtifactID: artifactModel.ArtifactID(selection.ArtifactID),
	}
}

func selectionForPin(
	pin sessionv3Runtime.ArtifactPin,
) sessionv3Runtime.ArtifactSelection {
	return sessionv3Runtime.ArtifactSelection{
		RootID:           pin.RootID,
		ArtifactID:       pin.ArtifactID,
		ExpectedRevision: pin.Revision,
		ExpectedDigest:   pin.Digest,
	}
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
			key == "artifact:"+value.RootID+":"+value.ArtifactID {
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
			key == "artifact:"+value.Server.RootID+":"+value.Server.ArtifactID {
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
		output = append(output, contributionArtifactRef(value))
	}
	return output
}

func contributionMCPRefs(
	values []sessionv3Runtime.MCPServerSelection,
) []artifactModel.ArtifactRef {
	output := make([]artifactModel.ArtifactRef, 0, len(values))
	for _, value := range values {
		output = append(output, contributionArtifactRef(value.Server))
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
