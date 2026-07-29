package selection

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/workspace"
	"github.com/flexigpt/flexigpt-app/internal/workspace/contextadapter"
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

type ConversationSkillSelectionRef struct {
	ConversationResourceSelectionRef

	DisplayName string                         `json:"displayName,omitempty"`
	Insert      workspace.WorkspaceSkillInsert `json:"insert,omitempty"`
}

type ConversationSelection struct {
	Workspace         workspace.WorkspaceRef             `json:"workspace"`
	DisplayName       string                             `json:"displayName,omitempty"`
	WorkspaceRevision uint64                             `json:"workspaceRevision,omitempty"`
	CatalogRevision   uint64                             `json:"catalogRevision,omitempty"`
	ContextRefs       []ConversationResourceSelectionRef `json:"contextRefs,omitempty"`
	SkillRefs         []ConversationSkillSelectionRef    `json:"skillRefs,omitempty"`
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
	Workspace         workspace.WorkspaceRef      `json:"workspace"`
	DisplayName       string                      `json:"displayName,omitempty"`
	WorkspaceRevision uint64                      `json:"workspaceRevision,omitempty"`
	CatalogRevision   uint64                      `json:"catalogRevision,omitempty"`
	Status            ConversationSelectionStatus `json:"status"`
	Contexts          []ConversationContextUsage  `json:"contexts,omitempty"`
	Skills            []ConversationSkillUsage    `json:"skills,omitempty"`
	Diagnostics       []diagnostic.Diagnostic     `json:"diagnostics,omitempty"`
}

type ConversationResolution struct {
	Usage  ConversationUsage
	Prompt string
}

type ConversationResolver struct {
	workspaceAPI *workspace.API
}

func NewConversationResolver(workspaceAPI *workspace.API) (*ConversationResolver, error) {
	if workspaceAPI == nil {
		return nil, errors.New("nil workspaceAPI providerd")
	}
	return &ConversationResolver{workspaceAPI: workspaceAPI}, nil
}

func (cr *ConversationResolver) ResolveConversationSelection(
	ctx context.Context,
	selection ConversationSelection,
) (ConversationResolution, error) {
	if cr.workspaceAPI == nil {
		return ConversationResolution{}, errors.New("invalid workspaceAPI")
	}
	if err := cr.workspaceAPI.Ready(); err != nil {
		return ConversationResolution{}, err
	}
	if err := selection.Workspace.Validate(); err != nil {
		return ConversationResolution{}, err
	}

	workspaceValue, err := cr.workspaceAPI.GetWorkspace(ctx, &workspace.GetWorkspaceRequest{
		Workspace: selection.Workspace,
	})
	if err != nil {
		return ConversationResolution{
			Usage: unresolvedConversationUsage(selection, err),
		}, err
	}
	if workspaceValue == nil || workspaceValue.Body == nil {
		err := fmt.Errorf(
			"%w: Workspace %q returned an empty view",
			basespec.ErrNotFound,
			selection.Workspace.CollectionID,
		)
		return ConversationResolution{
			Usage: unresolvedConversationUsage(selection, err),
		}, err
	}

	if !workspaceValue.Body.Enabled {
		err := fmt.Errorf("selected Workspace %q is disabled", selection.Workspace.CollectionID)
		return ConversationResolution{
			Usage: unresolvedConversationUsage(selection, err),
		}, err
	}

	usage := ConversationUsage{
		Workspace:         selection.Workspace,
		DisplayName:       workspaceValue.Body.DisplayName,
		WorkspaceRevision: workspaceValue.Body.Revision,
		CatalogRevision:   selection.CatalogRevision,
		Status:            ConversationSelectionReady,
	}

	if usage.DisplayName == "" {
		usage.DisplayName = selection.DisplayName
	}

	contextUsageByID := make(map[basespec.ArtifactID]int, len(selection.ContextRefs))
	contextArtifactRefs := make([]artifact.ArtifactRef, 0, len(selection.ContextRefs))

	for _, ref := range selection.ContextRefs {
		if err := ref.Artifact.Validate(); err != nil {
			return ConversationResolution{
				Usage: unresolvedConversationUsage(selection, err),
			}, err
		}
		if ref.Artifact.RootID != selection.Workspace.RootID {
			err := fmt.Errorf(
				"%w: selected Context Artifact belongs to another Root",
				basespec.ErrInvalid,
			)
			return ConversationResolution{
				Usage: unresolvedConversationUsage(selection, err),
			}, err
		}
		if _, duplicate := contextUsageByID[ref.Artifact.ArtifactID]; duplicate {
			err := fmt.Errorf("%w: duplicate selected Context Artifact", basespec.ErrInvalid)
			return ConversationResolution{
				Usage: unresolvedConversationUsage(selection, err),
			}, err
		}

		contextUsageByID[ref.Artifact.ArtifactID] = len(usage.Contexts)
		contextArtifactRefs = append(contextArtifactRefs, ref.Artifact)
		usage.Contexts = append(usage.Contexts, ConversationContextUsage{
			Artifact:                 ref.Artifact,
			Name:                     ref.Name,
			Locator:                  ref.Locator,
			SelectedDefinitionDigest: ref.DefinitionDigest,
			Status:                   ConversationContextUsageUnavailable,
		})
	}

	prompt := ""
	if len(contextArtifactRefs) > 0 {
		contextPlan, composeErr := cr.workspaceAPI.ComposeWorkspaceContext(
			ctx,
			&workspace.ComposeWorkspaceContextRequest{
				Workspace: selection.Workspace,
				Body: &workspace.ComposeWorkspaceContextRequestBody{
					Artifacts: contextArtifactRefs,
				},
			},
		)
		if composeErr != nil {
			usage.Status = ConversationSelectionUnavailable
			usage.Diagnostics = diagnostic.AppendDiagnostics(
				usage.Diagnostics,
				conversationSelectionDiagnostic(
					"workspace.conversation.context-unavailable",
					composeErr.Error(),
				),
			)
		} else if contextPlan != nil && contextPlan.Body != nil {
			usage.CatalogRevision = contextPlan.Body.CatalogRevision
			usage.Diagnostics = diagnostic.AppendDiagnostics(
				usage.Diagnostics,
				contextPlan.Body.Diagnostics...,
			)

			prompt = contextPlan.Body.Prompt

			for _, contribution := range contextPlan.Body.Contributions {
				index, found := contextUsageByID[contribution.Artifact.ArtifactID]
				if !found {
					continue
				}

				current := &usage.Contexts[index]
				current.Name = contribution.Name
				current.Locator = contribution.Locator
				current.UsedDefinitionDigest = contribution.DefinitionDigest
				current.UsedArtifactRevision = contribution.RecordRevision
				current.Changed = conversationResourceChanged(
					current.SelectedDefinitionDigest,
					current.UsedDefinitionDigest,
					selection.ContextRefs[index].ArtifactRevision,
					current.UsedArtifactRevision,
					selection.ContextRefs[index].Locator,
					current.Locator,
				)
			}

			for _, decision := range contextPlan.Body.Decisions {
				index, found := contextUsageByID[decision.Artifact.ArtifactID]
				if !found {
					continue
				}

				current := &usage.Contexts[index]
				current.Status = conversationContextUsageStatusOf(decision.Status)
				current.Code = decision.Code
				current.OriginalBytes = decision.OriginalBytes
				current.IncludedBytes = decision.IncludedBytes
			}
		}
	}

	skillUsageByID := make(map[basespec.ArtifactID]int, len(selection.SkillRefs))
	skillArtifactRefs := make([]artifact.ArtifactRef, 0, len(selection.SkillRefs))

	for _, ref := range selection.SkillRefs {
		if err := ref.Artifact.Validate(); err != nil {
			return ConversationResolution{
				Usage: unresolvedConversationUsage(selection, err),
			}, err
		}
		if ref.Artifact.RootID != selection.Workspace.RootID {
			err := fmt.Errorf(
				"%w: selected Workspace Skill belongs to another Root",
				basespec.ErrInvalid,
			)
			return ConversationResolution{
				Usage: unresolvedConversationUsage(selection, err),
			}, err
		}
		if _, duplicate := skillUsageByID[ref.Artifact.ArtifactID]; duplicate {
			err := fmt.Errorf("%w: duplicate selected Workspace Skill Artifact", basespec.ErrInvalid)
			return ConversationResolution{
				Usage: unresolvedConversationUsage(selection, err),
			}, err
		}
		skillUsageByID[ref.Artifact.ArtifactID] = len(usage.Skills)
		skillArtifactRefs = append(skillArtifactRefs, ref.Artifact)
		usage.Skills = append(usage.Skills, ConversationSkillUsage{
			Artifact:                 ref.Artifact,
			Name:                     ref.Name,
			DisplayName:              ref.DisplayName,
			Locator:                  ref.Locator,
			SelectedDefinitionDigest: ref.DefinitionDigest,
			Status:                   ConversationSkillUsageUnavailable,
		})
	}

	if len(skillArtifactRefs) > 0 {
		skillPlan, loadErr := cr.workspaceAPI.LoadWorkspaceSkills(
			ctx,
			&workspace.LoadWorkspaceSkillsRequest{
				Workspace: selection.Workspace,
				Body: &workspace.LoadWorkspaceSkillsRequestBody{
					Artifacts: skillArtifactRefs,
				},
			},
		)
		if loadErr != nil {
			usage.Diagnostics = diagnostic.AppendDiagnostics(
				usage.Diagnostics,
				conversationSelectionDiagnostic(
					"workspace.conversation.skills-unavailable",
					loadErr.Error(),
				),
			)
		} else if skillPlan != nil && skillPlan.Body != nil {
			if skillPlan.Body.CatalogRevision > usage.CatalogRevision {
				usage.CatalogRevision = skillPlan.Body.CatalogRevision
			}
			usage.Diagnostics = diagnostic.AppendDiagnostics(
				usage.Diagnostics,
				skillPlan.Body.Diagnostics...,
			)

			for _, skill := range skillPlan.Body.Skills {
				index, found := skillUsageByID[skill.Artifact.ArtifactID]
				if !found {
					continue
				}

				current := &usage.Skills[index]
				current.Name = skill.Skill.Name
				current.DisplayName = skill.Skill.DisplayName
				current.Locator = skill.Locator
				current.UsedDefinitionDigest = skill.DefinitionDigest
				current.UsedArtifactRevision = skill.RecordRevision
				current.Changed = conversationResourceChanged(
					current.SelectedDefinitionDigest,
					current.UsedDefinitionDigest,
					selection.SkillRefs[index].ArtifactRevision,
					current.UsedArtifactRevision,
					selection.SkillRefs[index].Locator,
					current.Locator,
				)

				if skill.Skill.Insert != workspace.WorkspaceSkillInsertInstructions {
					current.Status = ConversationSkillUsageUnavailable
					current.Diagnostics = diagnostic.AppendDiagnostics(
						current.Diagnostics,
						conversationSelectionDiagnostic(
							"workspace.conversation.skill-ineligible",
							"only Workspace Skills with insert=\"instructions\" can enter a conversation Skill session",
						),
					)
					// A persisted selection can become ineligible after a
					// Workspace refresh. Keep it in usage as unavailable, but
					// allow any other valid selected Context or Skills to make
					// this a partial completion.
					continue
				}

				current.Status = ConversationSkillUsageAvailable
				current.Diagnostics = diagnostic.AppendDiagnostics(
					current.Diagnostics,
					skill.Diagnostics...,
				)
			}
		}
	}

	ResolveConversationUsageStatus(&usage)
	if usage.Status == ConversationSelectionUnavailable &&
		len(usage.Contexts)+len(usage.Skills) > 0 {
		cause := errors.New(
			"selected Workspace has no currently usable Context or Skills",
		)
		return ConversationResolution{
			Usage: usage,
		}, cause
	}

	return ConversationResolution{
		Usage:  usage,
		Prompt: prompt,
	}, nil
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
		CatalogRevision:   selection.CatalogRevision,
		Status:            ConversationSelectionUnavailable,
		Diagnostics: []diagnostic.Diagnostic{
			conversationSelectionDiagnostic(
				"workspace.conversation.unavailable",
				message,
			),
		},
	}

	for _, ref := range selection.ContextRefs {
		usage.Contexts = append(usage.Contexts, ConversationContextUsage{
			Artifact:                 ref.Artifact,
			Name:                     ref.Name,
			Locator:                  ref.Locator,
			SelectedDefinitionDigest: ref.DefinitionDigest,
			Status:                   ConversationContextUsageUnavailable,
		})
	}

	for _, ref := range selection.SkillRefs {
		usage.Skills = append(usage.Skills, ConversationSkillUsage{
			Artifact:                 ref.Artifact,
			Name:                     ref.Name,
			DisplayName:              ref.DisplayName,
			Locator:                  ref.Locator,
			SelectedDefinitionDigest: ref.DefinitionDigest,
			Status:                   ConversationSkillUsageUnavailable,
		})
	}

	return usage
}

func conversationContextUsageStatusOf(
	status contextadapter.CompositionStatus,
) ConversationContextUsageStatus {
	switch status {
	case contextadapter.CompositionIncluded:
		return ConversationContextUsageIncluded
	case contextadapter.CompositionTruncated:
		return ConversationContextUsageTruncated
	case contextadapter.CompositionExcluded:
		return ConversationContextUsageExcluded
	case contextadapter.CompositionDenied:
		return ConversationContextUsageDenied
	case contextadapter.CompositionUnavailable:
		return ConversationContextUsageUnavailable
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
	if selectedLocator != "" && selectedLocator != usedLocator {
		return true
	}
	return false
}

// ResolveConversationUsageStatus recomputes the aggregate status after an
// adapter or Skill Runtime adds authoritative send-time information.
func ResolveConversationUsageStatus(usage *ConversationUsage) {
	if usage == nil {
		return
	}

	total := len(usage.Contexts) + len(usage.Skills)
	if total == 0 {
		usage.Status = ConversationSelectionReady
		return
	}

	usable := 0
	for _, contextUsage := range usage.Contexts {
		if contextUsage.Status == ConversationContextUsageIncluded ||
			contextUsage.Status == ConversationContextUsageTruncated {
			usable++
		}
	}
	for _, skillUsage := range usage.Skills {
		if skillUsage.Status == ConversationSkillUsageAvailable {
			usable++
		}
	}

	switch {
	case usable == total:
		usage.Status = ConversationSelectionReady
	case usable > 0:
		usage.Status = ConversationSelectionPartial
	default:
		usage.Status = ConversationSelectionUnavailable
	}
}

func conversationSelectionDiagnostic(
	code string,
	message string,
) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{
		Severity: diagnostic.DiagnosticError,
		Code:     code,
		Message:  diagnostic.BoundedDiagnosticMessage(message),
	}
}
