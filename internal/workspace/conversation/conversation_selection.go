package conversation

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/flexigpt/agentskills-go/document"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	workspaceRuntime "github.com/flexigpt/flexigpt-app/internal/workspace/runtime"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/adapter/prompt"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/adapter/skill"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/workspace/store/domain"
)

type ConversationSelectionStatus string

const (
	ConversationSelectionReady       ConversationSelectionStatus = "ready"
	ConversationSelectionPartial     ConversationSelectionStatus = "partial"
	ConversationSelectionUnavailable ConversationSelectionStatus = "unavailable"
)

type ConversationContextUsageStatus string

const (
	ConversationContextUsageIncluded    ConversationContextUsageStatus = "included"
	ConversationContextUsageTruncated   ConversationContextUsageStatus = "truncated"
	ConversationContextUsageExcluded    ConversationContextUsageStatus = "excluded"
	ConversationContextUsageDenied      ConversationContextUsageStatus = "denied"
	ConversationContextUsageUnavailable ConversationContextUsageStatus = "unavailable"
)

type ConversationSkillUsageStatus string

const (
	ConversationSkillUsageAvailable   ConversationSkillUsageStatus = "available"
	ConversationSkillUsageUnavailable ConversationSkillUsageStatus = "unavailable"
)

type ConversationResourceSelectionRef struct {
	Artifact         artifact.ArtifactRef `json:"artifact"`
	Name             string               `json:"name,omitempty"`
	Locator          basespec.Locator     `json:"locator,omitempty"`
	DefinitionDigest cryptoutil.Digest    `json:"definitionDigest,omitempty"`
	ArtifactRevision uint64               `json:"artifactRevision,omitempty"`
}

// ConversationSelection stores one user-selected Workspace Artifact and the
// explicitly selected Root-scoped Artifact resources for one conversation
// turn. No Collection or Catalog identity is persisted.
type ConversationSelection struct {
	Workspace         artifact.ArtifactRef               `json:"workspace"`
	DisplayName       string                             `json:"displayName,omitempty"`
	WorkspaceRevision uint64                             `json:"workspaceRevision,omitempty"`
	ContextRefs       []ConversationResourceSelectionRef `json:"contextRefs,omitempty"`
	SkillRefs         []ConversationResourceSelectionRef `json:"skillRefs,omitempty"`
}

type ConversationContextUsage struct {
	Artifact                 artifact.ArtifactRef           `json:"artifact"`
	Name                     string                         `json:"name,omitempty"`
	Locator                  basespec.Locator               `json:"locator,omitempty"`
	SelectedDefinitionDigest cryptoutil.Digest              `json:"selectedDefinitionDigest,omitempty"`
	UsedDefinitionDigest     cryptoutil.Digest              `json:"usedDefinitionDigest,omitempty"`
	UsedArtifactRevision     uint64                         `json:"usedArtifactRevision,omitempty"`
	Status                   ConversationContextUsageStatus `json:"status"`
	Code                     string                         `json:"code,omitempty"`
	OriginalBytes            int                            `json:"originalBytes,omitempty"`
	IncludedBytes            int                            `json:"includedBytes,omitempty"`
	Changed                  bool                           `json:"changed,omitempty"`
	Diagnostics              []diagnostic.Diagnostic        `json:"diagnostics,omitempty"`
}

type ConversationSkillUsage struct {
	Artifact                 artifact.ArtifactRef         `json:"artifact"`
	Name                     string                       `json:"name,omitempty"`
	DisplayName              string                       `json:"displayName,omitempty"`
	Locator                  basespec.Locator             `json:"locator,omitempty"`
	SelectedDefinitionDigest cryptoutil.Digest            `json:"selectedDefinitionDigest,omitempty"`
	UsedDefinitionDigest     cryptoutil.Digest            `json:"usedDefinitionDigest,omitempty"`
	UsedArtifactRevision     uint64                       `json:"usedArtifactRevision,omitempty"`
	Status                   ConversationSkillUsageStatus `json:"status"`
	Changed                  bool                         `json:"changed,omitempty"`
	SessionAvailable         bool                         `json:"sessionAvailable,omitempty"`
	Active                   bool                         `json:"active,omitempty"`
	Advertised               bool                         `json:"advertised,omitempty"`
	Diagnostics              []diagnostic.Diagnostic      `json:"diagnostics,omitempty"`
}

type ConversationUsage struct {
	Workspace         artifact.ArtifactRef        `json:"workspace"`
	DisplayName       string                      `json:"displayName,omitempty"`
	WorkspaceRevision uint64                      `json:"workspaceRevision,omitempty"`
	Status            ConversationSelectionStatus `json:"status"`
	Contexts          []ConversationContextUsage  `json:"contexts,omitempty"`
	Skills            []ConversationSkillUsage    `json:"skills,omitempty"`
	Diagnostics       []diagnostic.Diagnostic     `json:"diagnostics,omitempty"`
}

type ConversationResolution struct {
	Usage        ConversationUsage
	Instructions string
	UserMessage  string
}

// WorkspaceSource is the narrow Root-scoped Workspace consumer port used by
// conversation inference hydration.
type WorkspaceSource interface {
	ResolveWorkspace(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (workspaceDomain.Workspace, error)

	ComposeWorkspacePromptForRuntime(
		ctx context.Context,
		workspace artifact.ArtifactRef,
		artifacts []artifact.ArtifactRef,
	) (prompt.Plan, error)

	LoadWorkspaceSkillsForRuntime(
		ctx context.Context,
		workspace artifact.ArtifactRef,
		artifacts []artifact.ArtifactRef,
	) (skill.LoadPlan, error)
}

type ConversationResolver struct {
	workspaceAPI WorkspaceSource
}

func NewConversationResolver(
	workspaceAPI WorkspaceSource,
) (*ConversationResolver, error) {
	if workspaceAPI == nil {
		return nil, errors.New("workspace conversation source is required")
	}
	return &ConversationResolver{
		workspaceAPI: workspaceAPI,
	}, nil
}

func (r *ConversationResolver) ResolveConversationSelection(
	ctx context.Context,
	selection ConversationSelection,
) (ConversationResolution, error) {
	if r == nil || r.workspaceAPI == nil {
		return ConversationResolution{}, errors.New(
			"workspace conversation source is unavailable",
		)
	}
	if err := selection.Workspace.Validate(); err != nil {
		return ConversationResolution{}, err
	}

	workspace, err := r.workspaceAPI.ResolveWorkspace(
		ctx,
		selection.Workspace,
	)
	if err != nil {
		return ConversationResolution{
			Usage: unresolvedConversationUsage(selection, err),
		}, err
	}

	usage := ConversationUsage{
		Workspace:         selection.Workspace,
		DisplayName:       workspace.Artifact.DisplayName,
		WorkspaceRevision: workspace.Artifact.Revision,
		Status:            ConversationSelectionReady,
	}
	if usage.DisplayName == "" {
		usage.DisplayName = selection.DisplayName
	}

	contextRefs, contextIndex, err := initializeContextUsage(
		selection,
		&usage,
	)
	if err != nil {
		return ConversationResolution{
			Usage: unresolvedConversationUsage(selection, err),
		}, err
	}

	instructions := ""
	userMessage := ""
	if len(contextRefs) != 0 {
		plan, composeErr := r.workspaceAPI.ComposeWorkspacePromptForRuntime(
			ctx,
			selection.Workspace,
			contextRefs,
		)
		if composeErr != nil {
			usage.Diagnostics = diagnostic.Append(
				usage.Diagnostics,
				conversationDiagnostic(
					"workspace.conversation.context-unavailable",
					composeErr.Error(),
				),
			)
		} else {
			instructions = plan.Instructions
			userMessage = plan.UserMessage
			applyContextPlan(
				&usage,
				selection,
				plan,
				contextIndex,
			)
		}
	}

	skillRefs, skillIndex, err := initializeSkillUsage(
		selection,
		&usage,
	)
	if err != nil {
		return ConversationResolution{
			Usage: unresolvedConversationUsage(selection, err),
		}, err
	}
	if len(skillRefs) != 0 {
		plan, loadErr := r.workspaceAPI.LoadWorkspaceSkillsForRuntime(
			ctx,
			selection.Workspace,
			skillRefs,
		)
		if loadErr != nil {
			usage.Diagnostics = diagnostic.Append(
				usage.Diagnostics,
				conversationDiagnostic(
					"workspace.conversation.skills-unavailable",
					loadErr.Error(),
				),
			)
		} else {
			applySkillPlan(
				&usage,
				selection,
				plan,
				skillIndex,
			)
		}
	}

	ResolveConversationUsageStatus(&usage)
	if usage.Status == ConversationSelectionUnavailable &&
		len(usage.Contexts)+len(usage.Skills) != 0 {
		return ConversationResolution{Usage: usage}, errors.New(
			"selected Workspace has no currently usable Context, Instruction, or Skill Artifacts",
		)
	}
	return ConversationResolution{
		Usage:        usage,
		Instructions: instructions,
		UserMessage:  userMessage,
	}, nil
}

func initializeContextUsage(
	selection ConversationSelection,
	usage *ConversationUsage,
) ([]artifact.ArtifactRef, map[artifact.ArtifactRef]int, error) {
	index := make(map[artifact.ArtifactRef]int, len(selection.ContextRefs))
	refs := make([]artifact.ArtifactRef, 0, len(selection.ContextRefs))
	for _, selected := range selection.ContextRefs {
		if err := selected.Artifact.Validate(); err != nil {
			return nil, nil, err
		}
		if _, duplicate := index[selected.Artifact]; duplicate {
			return nil, nil, fmt.Errorf(
				"%w: duplicate selected Context Artifact",
				basespec.ErrInvalid,
			)
		}
		index[selected.Artifact] = len(usage.Contexts)
		refs = append(refs, selected.Artifact)
		usage.Contexts = append(usage.Contexts, ConversationContextUsage{
			Artifact:                 selected.Artifact,
			Name:                     selected.Name,
			Locator:                  selected.Locator,
			SelectedDefinitionDigest: selected.DefinitionDigest,
			Status:                   ConversationContextUsageUnavailable,
		})
	}
	return refs, index, nil
}

func initializeSkillUsage(
	selection ConversationSelection,
	usage *ConversationUsage,
) ([]artifact.ArtifactRef, map[artifact.ArtifactRef]int, error) {
	index := make(map[artifact.ArtifactRef]int, len(selection.SkillRefs))
	refs := make([]artifact.ArtifactRef, 0, len(selection.SkillRefs))
	for _, selected := range selection.SkillRefs {
		if err := selected.Artifact.Validate(); err != nil {
			return nil, nil, err
		}
		if _, duplicate := index[selected.Artifact]; duplicate {
			return nil, nil, fmt.Errorf(
				"%w: duplicate selected Skill Artifact",
				basespec.ErrInvalid,
			)
		}
		index[selected.Artifact] = len(usage.Skills)
		refs = append(refs, selected.Artifact)
		usage.Skills = append(usage.Skills, ConversationSkillUsage{
			Artifact:                 selected.Artifact,
			Name:                     selected.Name,
			Locator:                  selected.Locator,
			SelectedDefinitionDigest: selected.DefinitionDigest,
			Status:                   ConversationSkillUsageUnavailable,
		})
	}
	return refs, index, nil
}

func applyContextPlan(
	usage *ConversationUsage,
	selection ConversationSelection,
	plan prompt.Plan,
	index map[artifact.ArtifactRef]int,
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
		current.Changed = conversationResourceChanged(
			current.SelectedDefinitionDigest,
			current.UsedDefinitionDigest,
			selection.ContextRefs[position].ArtifactRevision,
			current.UsedArtifactRevision,
			selection.ContextRefs[position].Locator,
			current.Locator,
		)
		current.Status = ConversationContextUsageIncluded
		if contribution.Truncated {
			current.Status = ConversationContextUsageTruncated
		}
	}
	for _, decision := range plan.Decisions {
		position, found := index[decision.Artifact]
		if !found {
			continue
		}
		current := &usage.Contexts[position]
		current.Status = contextUsageStatus(decision.Status)
		current.Code = decision.Code
		current.OriginalBytes = decision.OriginalBytes
		current.IncludedBytes = decision.IncludedBytes
	}
}

func applySkillPlan(
	usage *ConversationUsage,
	selection ConversationSelection,
	plan skill.LoadPlan,
	index map[artifact.ArtifactRef]int,
) {
	for _, skill := range plan.Skills {
		position, found := index[skill.Artifact]
		if !found {
			continue
		}
		current := &usage.Skills[position]
		current.Name = skill.Document.Name
		current.DisplayName = skill.Document.DisplayName
		current.Locator = skill.Locator
		current.UsedDefinitionDigest = skill.DefinitionDigest
		current.UsedArtifactRevision = skill.ArtifactRevision
		current.Changed = conversationResourceChanged(
			current.SelectedDefinitionDigest,
			current.UsedDefinitionDigest,
			selection.SkillRefs[position].ArtifactRevision,
			current.UsedArtifactRevision,
			selection.SkillRefs[position].Locator,
			current.Locator,
		)
		if skill.Document.Insert != document.SkillInsertInstructions {
			current.Diagnostics = diagnostic.Append(
				current.Diagnostics,
				conversationDiagnostic(
					"workspace.conversation.skill-ineligible",
					"only Skills with insert=\"instructions\" can enter a conversation Skill session",
				),
			)
			continue
		}
		current.Status = ConversationSkillUsageAvailable
	}
}

func unresolvedConversationUsage(
	selection ConversationSelection,
	cause error,
) ConversationUsage {
	message := "the selected Workspace is unavailable"
	if cause != nil && strings.TrimSpace(cause.Error()) != "" {
		message = cause.Error()
	}
	usage := ConversationUsage{
		Workspace:         selection.Workspace,
		DisplayName:       selection.DisplayName,
		WorkspaceRevision: selection.WorkspaceRevision,
		Status:            ConversationSelectionUnavailable,
		Diagnostics: []diagnostic.Diagnostic{
			conversationDiagnostic(
				"workspace.conversation.unavailable",
				message,
			),
		},
	}
	for _, selected := range selection.ContextRefs {
		usage.Contexts = append(usage.Contexts, ConversationContextUsage{
			Artifact:                 selected.Artifact,
			Name:                     selected.Name,
			Locator:                  selected.Locator,
			SelectedDefinitionDigest: selected.DefinitionDigest,
			Status:                   ConversationContextUsageUnavailable,
		})
	}
	for _, selected := range selection.SkillRefs {
		usage.Skills = append(usage.Skills, ConversationSkillUsage{
			Artifact:                 selected.Artifact,
			Name:                     selected.Name,
			Locator:                  selected.Locator,
			SelectedDefinitionDigest: selected.DefinitionDigest,
			Status:                   ConversationSkillUsageUnavailable,
		})
	}
	return usage
}

func contextUsageStatus(
	status workspaceRuntime.CompositionStatus,
) ConversationContextUsageStatus {
	switch status {
	case workspaceRuntime.CompositionIncluded:
		return ConversationContextUsageIncluded
	case workspaceRuntime.CompositionTruncated:
		return ConversationContextUsageTruncated
	case workspaceRuntime.CompositionExcluded:
		return ConversationContextUsageExcluded
	case workspaceRuntime.CompositionDenied:
		return ConversationContextUsageDenied
	default:
		return ConversationContextUsageUnavailable
	}
}

func conversationResourceChanged(
	selectedDigest cryptoutil.Digest,
	usedDigest cryptoutil.Digest,
	selectedRevision uint64,
	usedRevision uint64,
	selectedLocator basespec.Locator,
	usedLocator basespec.Locator,
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

func ResolveConversationUsageStatus(
	usage *ConversationUsage,
) {
	if usage == nil {
		return
	}
	total := len(usage.Contexts) + len(usage.Skills)
	if total == 0 {
		usage.Status = ConversationSelectionReady
		return
	}

	usable := 0
	for _, value := range usage.Contexts {
		if value.Status == ConversationContextUsageIncluded ||
			value.Status == ConversationContextUsageTruncated {
			usable++
		}
	}
	for _, value := range usage.Skills {
		if value.Status == ConversationSkillUsageAvailable {
			usable++
		}
	}
	switch {
	case usable == total:
		usage.Status = ConversationSelectionReady
	case usable != 0:
		usage.Status = ConversationSelectionPartial
	default:
		usage.Status = ConversationSelectionUnavailable
	}
}

func conversationDiagnostic(
	code string,
	message string,
) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{
		Severity: diagnostic.SeverityWarning,
		Code:     code,
		Message:  diagnostic.BoundedMessage(message),
	}
}
