package spec

import (
	"context"
	"errors"
	"fmt"
	"strings"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	workspaceAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/contextengine"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/domain"
)

type WorkspaceConversationSelectionStatus string

const (
	WorkspaceConversationSelectionReady       WorkspaceConversationSelectionStatus = "ready"
	WorkspaceConversationSelectionPartial     WorkspaceConversationSelectionStatus = "partial"
	WorkspaceConversationSelectionUnavailable WorkspaceConversationSelectionStatus = "unavailable"
)

type WorkspaceConversationContextUsageStatus string

const (
	WorkspaceConversationContextUsageIncluded    WorkspaceConversationContextUsageStatus = "included"
	WorkspaceConversationContextUsageTruncated   WorkspaceConversationContextUsageStatus = "truncated"
	WorkspaceConversationContextUsageExcluded    WorkspaceConversationContextUsageStatus = "excluded"
	WorkspaceConversationContextUsageDenied      WorkspaceConversationContextUsageStatus = "denied"
	WorkspaceConversationContextUsageUnavailable WorkspaceConversationContextUsageStatus = "unavailable"
)

type WorkspaceConversationSkillUsageStatus string

const (
	WorkspaceConversationSkillUsageAvailable   WorkspaceConversationSkillUsageStatus = "available"
	WorkspaceConversationSkillUsageUnavailable WorkspaceConversationSkillUsageStatus = "unavailable"
)

type WorkspaceConversationResourceSelectionRef struct {
	Artifact         artifactModel.ArtifactRef `json:"artifact"`
	Name             string                    `json:"name,omitempty"`
	Locator          spec.Locator              `json:"locator,omitempty"`
	DefinitionDigest cryptoutil.Digest         `json:"definitionDigest,omitempty"`
	ArtifactRevision uint64                    `json:"artifactRevision,omitempty"`
}

// WorkspaceConversationSelection stores one user-selected Workspace Artifact and the
// explicitly selected Root-scoped Artifact resources for one conversation
// turn. No Plugin or Catalog identity is persisted.
type WorkspaceConversationSelection struct {
	Workspace         artifactModel.ArtifactRef                   `json:"workspace"`
	DisplayName       string                                      `json:"displayName,omitempty"`
	WorkspaceRevision uint64                                      `json:"workspaceRevision,omitempty"`
	ContextRefs       []WorkspaceConversationResourceSelectionRef `json:"contextRefs,omitempty"`
	SkillRefs         []WorkspaceConversationResourceSelectionRef `json:"skillRefs,omitempty"`
}

type WorkspaceConversationContextUsage struct {
	Artifact                 artifactModel.ArtifactRef               `json:"artifact"`
	Name                     string                                  `json:"name,omitempty"`
	Locator                  spec.Locator                            `json:"locator,omitempty"`
	SelectedDefinitionDigest cryptoutil.Digest                       `json:"selectedDefinitionDigest,omitempty"`
	UsedDefinitionDigest     cryptoutil.Digest                       `json:"usedDefinitionDigest,omitempty"`
	UsedArtifactRevision     uint64                                  `json:"usedArtifactRevision,omitempty"`
	Status                   WorkspaceConversationContextUsageStatus `json:"status"`
	Code                     string                                  `json:"code,omitempty"`
	OriginalBytes            int                                     `json:"originalBytes,omitempty"`
	IncludedBytes            int                                     `json:"includedBytes,omitempty"`
	Changed                  bool                                    `json:"changed,omitempty"`
	Diagnostics              []diagnostic.Diagnostic                 `json:"diagnostics,omitempty"`
}

type WorkspaceConversationSkillUsage struct {
	Artifact                 artifactModel.ArtifactRef             `json:"artifact"`
	Name                     string                                `json:"name,omitempty"`
	DisplayName              string                                `json:"displayName,omitempty"`
	Locator                  spec.Locator                          `json:"locator,omitempty"`
	SelectedDefinitionDigest cryptoutil.Digest                     `json:"selectedDefinitionDigest,omitempty"`
	UsedDefinitionDigest     cryptoutil.Digest                     `json:"usedDefinitionDigest,omitempty"`
	UsedArtifactRevision     uint64                                `json:"usedArtifactRevision,omitempty"`
	Status                   WorkspaceConversationSkillUsageStatus `json:"status"`
	Changed                  bool                                  `json:"changed,omitempty"`
	SessionAvailable         bool                                  `json:"sessionAvailable,omitempty"`
	Active                   bool                                  `json:"active,omitempty"`
	Advertised               bool                                  `json:"advertised,omitempty"`
	Diagnostics              []diagnostic.Diagnostic               `json:"diagnostics,omitempty"`
}

type WorkspaceConversationUsage struct {
	Workspace         artifactModel.ArtifactRef            `json:"workspace"`
	DisplayName       string                               `json:"displayName,omitempty"`
	WorkspaceRevision uint64                               `json:"workspaceRevision,omitempty"`
	Status            WorkspaceConversationSelectionStatus `json:"status"`
	Contexts          []WorkspaceConversationContextUsage  `json:"contexts,omitempty"`
	Skills            []WorkspaceConversationSkillUsage    `json:"skills,omitempty"`
	Diagnostics       []diagnostic.Diagnostic              `json:"diagnostics,omitempty"`
}

type WorkspaceConversationResolution struct {
	Usage        WorkspaceConversationUsage
	Instructions string
	UserMessage  string
}

// WorkspaceSource is the narrow Root-scoped Workspace consumer port used by
// conversation inference hydration.
type WorkspaceSource interface {
	ResolveWorkspace(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
	) (workspaceDomain.WorkspaceView, error)

	ComposeWorkspacePrompt(
		ctx context.Context,
		workspace artifactModel.ArtifactRef,
		artifacts []artifactModel.ArtifactRef,
	) (workspaceAPI.WorkspacePromptPlan, error)

	LoadWorkspaceSkills(
		ctx context.Context,
		workspace artifactModel.ArtifactRef,
		artifacts []artifactModel.ArtifactRef,
	) (workspaceAPI.WorkspaceSkillLoadPlan, error)
}

type WorkspaceConversationResolver struct {
	workspaceAPI WorkspaceSource
}

func NewWorkspaceConversationResolver(
	wpSource WorkspaceSource,
) (*WorkspaceConversationResolver, error) {
	if wpSource == nil {
		return nil, errors.New("workspace conversation source is required")
	}
	return &WorkspaceConversationResolver{
		workspaceAPI: wpSource,
	}, nil
}

func (r *WorkspaceConversationResolver) ResolveWorkspaceConversationSelection(
	ctx context.Context,
	selection WorkspaceConversationSelection,
) (WorkspaceConversationResolution, error) {
	if r == nil || r.workspaceAPI == nil {
		return WorkspaceConversationResolution{}, errors.New(
			"workspace conversation source is unavailable",
		)
	}
	if err := selection.Workspace.Validate(); err != nil {
		return WorkspaceConversationResolution{}, err
	}

	workspace, err := r.workspaceAPI.ResolveWorkspace(
		ctx,
		selection.Workspace,
	)
	if err != nil {
		return WorkspaceConversationResolution{
			Usage: unresolvedWorkspaceConversationUsage(selection, err),
		}, err
	}

	usage := WorkspaceConversationUsage{
		Workspace:         selection.Workspace,
		DisplayName:       workspace.Artifact.DisplayName,
		WorkspaceRevision: workspace.Artifact.Revision,
		Status:            WorkspaceConversationSelectionReady,
	}
	if usage.DisplayName == "" {
		usage.DisplayName = selection.DisplayName
	}

	contextRefs, contextIndex, err := initializeWorkspaceContextUsage(
		selection,
		&usage,
	)
	if err != nil {
		return WorkspaceConversationResolution{
			Usage: unresolvedWorkspaceConversationUsage(selection, err),
		}, err
	}

	instructions := ""
	userMessage := ""
	if len(contextRefs) != 0 {
		plan, composeErr := r.workspaceAPI.ComposeWorkspacePrompt(
			ctx,
			selection.Workspace,
			contextRefs,
		)
		if composeErr != nil {
			usage.Diagnostics = diagnostic.Append(
				usage.Diagnostics,
				workspaceConversationDiagnostic(
					"workspace.conversation.context-unavailable",
					composeErr.Error(),
				),
			)
		} else {
			instructions = plan.Instructions
			userMessage = plan.UserMessage
			applyWorkspaceContextPlan(
				&usage,
				selection,
				plan,
				contextIndex,
			)
		}
	}

	skillRefs, skillIndex, err := initializeWorkspaceSkillUsage(
		selection,
		&usage,
	)
	if err != nil {
		return WorkspaceConversationResolution{
			Usage: unresolvedWorkspaceConversationUsage(selection, err),
		}, err
	}
	if len(skillRefs) != 0 {
		plan, loadErr := r.workspaceAPI.LoadWorkspaceSkills(
			ctx,
			selection.Workspace,
			skillRefs,
		)
		if loadErr != nil {
			usage.Diagnostics = diagnostic.Append(
				usage.Diagnostics,
				workspaceConversationDiagnostic(
					"workspace.conversation.skills-unavailable",
					loadErr.Error(),
				),
			)
		} else {
			applyWorkspaceSkillPlan(
				&usage,
				selection,
				plan,
				skillIndex,
			)
		}
	}

	ResolveWorkspaceConversationUsageStatus(&usage)
	if usage.Status == WorkspaceConversationSelectionUnavailable &&
		len(usage.Contexts)+len(usage.Skills) != 0 {
		return WorkspaceConversationResolution{Usage: usage}, errors.New(
			"selected Workspace has no currently usable Context, Instruction, or Skill Artifacts",
		)
	}
	return WorkspaceConversationResolution{
		Usage:        usage,
		Instructions: instructions,
		UserMessage:  userMessage,
	}, nil
}

func initializeWorkspaceContextUsage(
	selection WorkspaceConversationSelection,
	usage *WorkspaceConversationUsage,
) ([]artifactModel.ArtifactRef, map[artifactModel.ArtifactRef]int, error) {
	index := make(map[artifactModel.ArtifactRef]int, len(selection.ContextRefs))
	refs := make([]artifactModel.ArtifactRef, 0, len(selection.ContextRefs))
	for _, selected := range selection.ContextRefs {
		if err := selected.Artifact.Validate(); err != nil {
			return nil, nil, err
		}
		if _, duplicate := index[selected.Artifact]; duplicate {
			return nil, nil, fmt.Errorf(
				"%w: duplicate selected Context Artifact",
				spec.ErrInvalid,
			)
		}
		index[selected.Artifact] = len(usage.Contexts)
		refs = append(refs, selected.Artifact)
		usage.Contexts = append(usage.Contexts, WorkspaceConversationContextUsage{
			Artifact:                 selected.Artifact,
			Name:                     selected.Name,
			Locator:                  selected.Locator,
			SelectedDefinitionDigest: selected.DefinitionDigest,
			Status:                   WorkspaceConversationContextUsageUnavailable,
		})
	}
	return refs, index, nil
}

func initializeWorkspaceSkillUsage(
	selection WorkspaceConversationSelection,
	usage *WorkspaceConversationUsage,
) ([]artifactModel.ArtifactRef, map[artifactModel.ArtifactRef]int, error) {
	index := make(map[artifactModel.ArtifactRef]int, len(selection.SkillRefs))
	refs := make([]artifactModel.ArtifactRef, 0, len(selection.SkillRefs))
	for _, selected := range selection.SkillRefs {
		if err := selected.Artifact.Validate(); err != nil {
			return nil, nil, err
		}
		if _, duplicate := index[selected.Artifact]; duplicate {
			return nil, nil, fmt.Errorf(
				"%w: duplicate selected Skill Artifact",
				spec.ErrInvalid,
			)
		}
		index[selected.Artifact] = len(usage.Skills)
		refs = append(refs, selected.Artifact)
		usage.Skills = append(usage.Skills, WorkspaceConversationSkillUsage{
			Artifact:                 selected.Artifact,
			Name:                     selected.Name,
			Locator:                  selected.Locator,
			SelectedDefinitionDigest: selected.DefinitionDigest,
			Status:                   WorkspaceConversationSkillUsageUnavailable,
		})
	}
	return refs, index, nil
}

func applyWorkspaceContextPlan(
	usage *WorkspaceConversationUsage,
	selection WorkspaceConversationSelection,
	plan workspaceAPI.WorkspacePromptPlan,
	index map[artifactModel.ArtifactRef]int,
) {
	usage.Diagnostics = diagnostic.Append(
		usage.Diagnostics,
		plan.Diagnostics...,
	)

	for _, contribution := range plan.Contributions {
		position, found := index[contribution.Artifact]
		if !found {
			continue
		}
		current := &usage.Contexts[position]
		current.Name = contribution.Name
		current.Locator = contribution.Locator
		current.UsedDefinitionDigest = contribution.DefinitionDigest
		current.UsedArtifactRevision = contribution.ArtifactRevision
		current.OriginalBytes = contribution.OriginalBytes
		current.IncludedBytes = contribution.IncludedBytes
		current.Changed = workspaceConversationResourceChanged(
			current.SelectedDefinitionDigest,
			current.UsedDefinitionDigest,
			selection.ContextRefs[position].ArtifactRevision,
			current.UsedArtifactRevision,
			selection.ContextRefs[position].Locator,
			current.Locator,
		)
		current.Status = WorkspaceConversationContextUsageIncluded
		if contribution.Truncated {
			current.Status = WorkspaceConversationContextUsageTruncated
		}
	}
	for _, decision := range plan.Decisions {
		position, found := index[decision.Artifact]
		if !found {
			continue
		}
		current := &usage.Contexts[position]
		current.Status = workspaceContextUsageStatus(decision.Status)
		current.Code = decision.Code
		current.OriginalBytes = decision.OriginalBytes
		current.IncludedBytes = decision.IncludedBytes
	}
}

func applyWorkspaceSkillPlan(
	usage *WorkspaceConversationUsage,
	selection WorkspaceConversationSelection,
	plan workspaceAPI.WorkspaceSkillLoadPlan,
	index map[artifactModel.ArtifactRef]int,
) {
	for _, skill := range plan.Skills {
		position, found := index[skill.Artifact]
		if !found {
			continue
		}
		current := &usage.Skills[position]
		current.Name = skill.Name
		current.DisplayName = skill.DisplayName
		current.Locator = skill.Locator
		current.UsedDefinitionDigest = skill.DefinitionDigest
		current.UsedArtifactRevision = skill.ArtifactRevision
		current.Changed = workspaceConversationResourceChanged(
			current.SelectedDefinitionDigest,
			current.UsedDefinitionDigest,
			selection.SkillRefs[position].ArtifactRevision,
			current.UsedArtifactRevision,
			selection.SkillRefs[position].Locator,
			current.Locator,
		)
		if skill.Insert != declaration.InsertInstructions {
			current.Diagnostics = diagnostic.Append(
				current.Diagnostics,
				workspaceConversationDiagnostic(
					"workspace.conversation.skill-ineligible",
					"only Skills with insert=\"instructions\" can enter a conversation Skill session",
				),
			)
			continue
		}
		current.Status = WorkspaceConversationSkillUsageAvailable
	}
}

func unresolvedWorkspaceConversationUsage(
	selection WorkspaceConversationSelection,
	cause error,
) WorkspaceConversationUsage {
	message := "the selected Workspace is unavailable"
	if cause != nil && strings.TrimSpace(cause.Error()) != "" {
		message = cause.Error()
	}
	usage := WorkspaceConversationUsage{
		Workspace:         selection.Workspace,
		DisplayName:       selection.DisplayName,
		WorkspaceRevision: selection.WorkspaceRevision,
		Status:            WorkspaceConversationSelectionUnavailable,
		Diagnostics: []diagnostic.Diagnostic{
			workspaceConversationDiagnostic(
				"workspace.conversation.unavailable",
				message,
			),
		},
	}
	for _, selected := range selection.ContextRefs {
		usage.Contexts = append(usage.Contexts, WorkspaceConversationContextUsage{
			Artifact:                 selected.Artifact,
			Name:                     selected.Name,
			Locator:                  selected.Locator,
			SelectedDefinitionDigest: selected.DefinitionDigest,
			Status:                   WorkspaceConversationContextUsageUnavailable,
		})
	}
	for _, selected := range selection.SkillRefs {
		usage.Skills = append(usage.Skills, WorkspaceConversationSkillUsage{
			Artifact:                 selected.Artifact,
			Name:                     selected.Name,
			Locator:                  selected.Locator,
			SelectedDefinitionDigest: selected.DefinitionDigest,
			Status:                   WorkspaceConversationSkillUsageUnavailable,
		})
	}
	return usage
}

func workspaceContextUsageStatus(
	status contextengine.CompositionStatus,
) WorkspaceConversationContextUsageStatus {
	switch status {
	case contextengine.CompositionIncluded:
		return WorkspaceConversationContextUsageIncluded
	case contextengine.CompositionTruncated:
		return WorkspaceConversationContextUsageTruncated
	case contextengine.CompositionExcluded:
		return WorkspaceConversationContextUsageExcluded
	case contextengine.CompositionDenied:
		return WorkspaceConversationContextUsageDenied
	default:
		return WorkspaceConversationContextUsageUnavailable
	}
}

func workspaceConversationResourceChanged(
	selectedDigest cryptoutil.Digest,
	usedDigest cryptoutil.Digest,
	selectedRevision uint64,
	usedRevision uint64,
	selectedLocator spec.Locator,
	usedLocator spec.Locator,
) bool {
	if selectedDigest != "" && selectedDigest != usedDigest {
		return true
	}
	if selectedRevision != 0 && selectedRevision != usedRevision {
		return true
	}
	return selectedLocator != "" &&
		selectedLocator != usedLocator
}

func ResolveWorkspaceConversationUsageStatus(
	usage *WorkspaceConversationUsage,
) {
	if usage == nil {
		return
	}
	total := len(usage.Contexts) + len(usage.Skills)
	if total == 0 {
		usage.Status = WorkspaceConversationSelectionReady
		return
	}

	usable := 0
	for _, value := range usage.Contexts {
		if value.Status == WorkspaceConversationContextUsageIncluded ||
			value.Status == WorkspaceConversationContextUsageTruncated {
			usable++
		}
	}
	for _, value := range usage.Skills {
		if value.Status == WorkspaceConversationSkillUsageAvailable {
			usable++
		}
	}
	switch {
	case usable == total:
		usage.Status = WorkspaceConversationSelectionReady
	case usable != 0:
		usage.Status = WorkspaceConversationSelectionPartial
	default:
		usage.Status = WorkspaceConversationSelectionUnavailable
	}
}

func workspaceConversationDiagnostic(
	code string,
	message string,
) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{
		Severity: diagnostic.SeverityWarning,
		Code:     code,
		Message:  diagnostic.BoundedMessage(message),
	}
}
