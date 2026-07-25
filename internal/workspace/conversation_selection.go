package workspace

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
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
	RecordID         artifactstore.RecordID `json:"recordID"`
	Name             string                 `json:"name,omitempty"`
	Locator          artifactstore.Locator  `json:"locator,omitempty"`
	DefinitionDigest artifactstore.Digest   `json:"definitionDigest,omitempty"`
	RecordRevision   uint64                 `json:"recordRevision,omitempty"`
}

type ConversationSkillSelectionRef struct {
	ConversationResourceSelectionRef

	Identity    string               `json:"identity"`
	DisplayName string               `json:"displayName,omitempty"`
	Insert      WorkspaceSkillInsert `json:"insert,omitempty"`
}

type ConversationSelection struct {
	RootID            artifactstore.RootID               `json:"rootID"`
	DisplayName       string                             `json:"displayName,omitempty"`
	WorkspaceRevision uint64                             `json:"workspaceRevision,omitempty"`
	CatalogRevision   uint64                             `json:"catalogRevision,omitempty"`
	ContextRefs       []ConversationResourceSelectionRef `json:"contextRefs,omitempty"`
	SkillRefs         []ConversationSkillSelectionRef    `json:"skillRefs,omitempty"`
}

type ConversationContextUsage struct {
	RecordID                 artifactstore.RecordID         `json:"recordID"`
	Name                     string                         `json:"name,omitempty"`
	Locator                  artifactstore.Locator          `json:"locator,omitempty"`
	SelectedDefinitionDigest artifactstore.Digest           `json:"selectedDefinitionDigest,omitempty"`
	UsedDefinitionDigest     artifactstore.Digest           `json:"usedDefinitionDigest,omitempty"`
	Status                   ConversationContextUsageStatus `json:"status"`
	Code                     string                         `json:"code,omitempty"`
	OriginalBytes            int                            `json:"originalBytes,omitempty"`
	IncludedBytes            int                            `json:"includedBytes,omitempty"`
	Changed                  bool                           `json:"changed,omitempty"`
	Diagnostics              []artifactstore.Diagnostic     `json:"diagnostics,omitempty"`
}

type ConversationSkillUsage struct {
	RecordID                 artifactstore.RecordID       `json:"recordID"`
	Identity                 string                       `json:"identity"`
	Name                     string                       `json:"name,omitempty"`
	DisplayName              string                       `json:"displayName,omitempty"`
	Locator                  artifactstore.Locator        `json:"locator,omitempty"`
	SelectedDefinitionDigest artifactstore.Digest         `json:"selectedDefinitionDigest,omitempty"`
	UsedDefinitionDigest     artifactstore.Digest         `json:"usedDefinitionDigest,omitempty"`
	Status                   ConversationSkillUsageStatus `json:"status"`
	Changed                  bool                         `json:"changed,omitempty"`
	SessionAvailable         bool                         `json:"sessionAvailable,omitempty"`
	Active                   bool                         `json:"active,omitempty"`
	Advertised               bool                         `json:"advertised,omitempty"`
	Diagnostics              []artifactstore.Diagnostic   `json:"diagnostics,omitempty"`
}

type ConversationUsage struct {
	RootID            artifactstore.RootID        `json:"rootID"`
	DisplayName       string                      `json:"displayName,omitempty"`
	WorkspaceRevision uint64                      `json:"workspaceRevision,omitempty"`
	CatalogRevision   uint64                      `json:"catalogRevision,omitempty"`
	Status            ConversationSelectionStatus `json:"status"`
	Contexts          []ConversationContextUsage  `json:"contexts,omitempty"`
	Skills            []ConversationSkillUsage    `json:"skills,omitempty"`
	Diagnostics       []artifactstore.Diagnostic  `json:"diagnostics,omitempty"`
}

type ConversationResolution struct {
	Usage  ConversationUsage
	Prompt string
}

const WorkspaceSkillIdentityPrefix = "workspace/"

func WorkspaceSkillIdentity(
	rootID artifactstore.RootID,
	recordID artifactstore.RecordID,
) string {
	return WorkspaceSkillIdentityPrefix + string(rootID) + "/" + string(recordID)
}

func ParseWorkspaceSkillIdentity(
	identity string,
) (artifactstore.RootID, artifactstore.RecordID, error) {
	value := strings.TrimSpace(identity)
	if value == "" || value != identity {
		return "", "", errors.New("workspace Skill identity must be non-empty and trimmed")
	}

	relative, found := strings.CutPrefix(value, WorkspaceSkillIdentityPrefix)
	if !found {
		return "", "", fmt.Errorf("identity %q is not a Workspace Skill", identity)
	}

	parts := strings.Split(relative, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("workspace Skill identity %q is invalid", identity)
	}

	rootID := artifactstore.RootID(parts[0])
	recordID := artifactstore.RecordID(parts[1])
	if err := artifactstore.ValidateRootID(rootID); err != nil {
		return "", "", err
	}
	if err := artifactstore.ValidateRecordID(recordID); err != nil {
		return "", "", err
	}
	return rootID, recordID, nil
}

func (a *API) ResolveConversationSelection(
	ctx context.Context,
	selection ConversationSelection,
) (ConversationResolution, error) {
	if err := a.ready(); err != nil {
		return ConversationResolution{}, err
	}
	if err := artifactstore.ValidateRootID(selection.RootID); err != nil {
		return ConversationResolution{}, err
	}

	workspaceValue, err := a.GetWorkspace(ctx, &GetWorkspaceRequest{
		RootID: selection.RootID,
	})
	if err != nil {
		return ConversationResolution{
			Usage: unresolvedConversationUsage(selection, err),
		}, err
	}
	if workspaceValue == nil || workspaceValue.Body == nil {
		err := fmt.Errorf(
			"%w: Workspace %q returned an empty view",
			artifactstore.ErrNotFound,
			selection.RootID,
		)
		return ConversationResolution{
			Usage: unresolvedConversationUsage(selection, err),
		}, err
	}

	if !workspaceValue.Body.Enabled {
		err := fmt.Errorf("selected Workspace %q is disabled", selection.RootID)
		return ConversationResolution{
			Usage: unresolvedConversationUsage(selection, err),
		}, err
	}

	usage := ConversationUsage{
		RootID:            selection.RootID,
		DisplayName:       workspaceValue.Body.DisplayName,
		WorkspaceRevision: workspaceValue.Body.Revision,
		CatalogRevision:   selection.CatalogRevision,
		Status:            ConversationSelectionReady,
	}

	if usage.DisplayName == "" {
		usage.DisplayName = selection.DisplayName
	}

	contextUsageByID := make(map[artifactstore.RecordID]int, len(selection.ContextRefs))
	contextRecordIDs := make([]artifactstore.RecordID, 0, len(selection.ContextRefs))

	for _, ref := range selection.ContextRefs {
		if err := artifactstore.ValidateRecordID(ref.RecordID); err != nil {
			return ConversationResolution{
				Usage: unresolvedConversationUsage(selection, err),
			}, err
		}

		contextUsageByID[ref.RecordID] = len(usage.Contexts)
		contextRecordIDs = append(contextRecordIDs, ref.RecordID)
		usage.Contexts = append(usage.Contexts, ConversationContextUsage{
			RecordID:                 ref.RecordID,
			Name:                     ref.Name,
			Locator:                  ref.Locator,
			SelectedDefinitionDigest: ref.DefinitionDigest,
			Status:                   ConversationContextUsageUnavailable,
		})
	}

	prompt := ""
	if len(contextRecordIDs) > 0 {
		contextPlan, composeErr := a.ComposeWorkspaceContext(
			ctx,
			&ComposeWorkspaceContextRequest{
				RootID: selection.RootID,
				Body: &ComposeWorkspaceContextRequestBody{
					RecordIDs: contextRecordIDs,
				},
			},
		)
		if composeErr != nil {
			usage.Status = ConversationSelectionUnavailable
			usage.Diagnostics = artifactstore.AppendDiagnostics(
				usage.Diagnostics,
				conversationSelectionDiagnostic(
					"workspace.conversation.context-unavailable",
					composeErr.Error(),
				),
			)
			return ConversationResolution{
				Usage: usage,
			}, composeErr
		}

		if contextPlan != nil && contextPlan.Body != nil {
			usage.CatalogRevision = contextPlan.Body.CatalogRevision
			usage.Diagnostics = artifactstore.AppendDiagnostics(
				usage.Diagnostics,
				contextPlan.Body.Diagnostics...,
			)
			prompt = contextPlan.Body.Prompt

			for _, contribution := range contextPlan.Body.Contributions {
				index, found := contextUsageByID[contribution.RecordID]
				if !found {
					continue
				}

				current := &usage.Contexts[index]
				current.Name = contribution.Name
				current.Locator = contribution.Locator
				current.UsedDefinitionDigest = contribution.DefinitionDigest
				current.Changed = current.SelectedDefinitionDigest != "" &&
					current.SelectedDefinitionDigest != contribution.DefinitionDigest
			}

			for _, decision := range contextPlan.Body.Decisions {
				index, found := contextUsageByID[decision.RecordID]
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

	skillUsageByID := make(map[artifactstore.RecordID]int, len(selection.SkillRefs))
	skillRecordIDs := make([]artifactstore.RecordID, 0, len(selection.SkillRefs))

	for _, ref := range selection.SkillRefs {
		if err := artifactstore.ValidateRecordID(ref.RecordID); err != nil {
			return ConversationResolution{
				Usage: unresolvedConversationUsage(selection, err),
			}, err
		}

		expectedIdentity := WorkspaceSkillIdentity(selection.RootID, ref.RecordID)
		if strings.TrimSpace(ref.Identity) != expectedIdentity {
			err := fmt.Errorf(
				"%w: Workspace Skill identity %q does not match Workspace %q record %q",
				artifactstore.ErrInvalid,
				ref.Identity,
				selection.RootID,
				ref.RecordID,
			)
			return ConversationResolution{
				Usage: unresolvedConversationUsage(selection, err),
			}, err
		}

		skillUsageByID[ref.RecordID] = len(usage.Skills)
		skillRecordIDs = append(skillRecordIDs, ref.RecordID)
		usage.Skills = append(usage.Skills, ConversationSkillUsage{
			RecordID:                 ref.RecordID,
			Identity:                 ref.Identity,
			Name:                     ref.Name,
			DisplayName:              ref.DisplayName,
			Locator:                  ref.Locator,
			SelectedDefinitionDigest: ref.DefinitionDigest,
			Status:                   ConversationSkillUsageUnavailable,
		})
	}

	if len(skillRecordIDs) > 0 {
		skillPlan, loadErr := a.LoadWorkspaceSkills(
			ctx,
			&LoadWorkspaceSkillsRequest{
				RootID: selection.RootID,
				Body: &LoadWorkspaceSkillsRequestBody{
					RecordIDs: skillRecordIDs,
				},
			},
		)
		if loadErr != nil {
			usage.Diagnostics = artifactstore.AppendDiagnostics(
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
			usage.Diagnostics = artifactstore.AppendDiagnostics(
				usage.Diagnostics,
				skillPlan.Body.Diagnostics...,
			)

			for _, skill := range skillPlan.Body.Skills {
				index, found := skillUsageByID[skill.RecordID]
				if !found {
					continue
				}

				current := &usage.Skills[index]
				current.Name = skill.Skill.Name
				current.DisplayName = skill.Skill.DisplayName
				current.Locator = skill.Locator
				current.UsedDefinitionDigest = skill.DefinitionDigest
				current.Changed = current.SelectedDefinitionDigest != "" &&
					current.SelectedDefinitionDigest != skill.DefinitionDigest

				if skill.Skill.Insert != WorkspaceSkillInsertInstructions {
					current.Status = ConversationSkillUsageUnavailable
					current.Diagnostics = artifactstore.AppendDiagnostics(
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
				current.Diagnostics = artifactstore.AppendDiagnostics(
					current.Diagnostics,
					skill.Diagnostics...,
				)
			}
		}
	}

	ResolveConversationUsageStatus(&usage)
	if usage.Status == ConversationSelectionUnavailable &&
		len(usage.Contexts)+len(usage.Skills) > 0 {
		return ConversationResolution{
				Usage: usage,
			}, errors.New(
				"selected Workspace has no currently usable Context or Skills",
			)
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
		RootID:            selection.RootID,
		DisplayName:       selection.DisplayName,
		WorkspaceRevision: selection.WorkspaceRevision,
		CatalogRevision:   selection.CatalogRevision,
		Status:            ConversationSelectionUnavailable,
		Diagnostics: []artifactstore.Diagnostic{
			conversationSelectionDiagnostic(
				"workspace.conversation.unavailable",
				message,
			),
		},
	}

	for _, ref := range selection.ContextRefs {
		usage.Contexts = append(usage.Contexts, ConversationContextUsage{
			RecordID:                 ref.RecordID,
			Name:                     ref.Name,
			Locator:                  ref.Locator,
			SelectedDefinitionDigest: ref.DefinitionDigest,
			Status:                   ConversationContextUsageUnavailable,
		})
	}

	for _, ref := range selection.SkillRefs {
		usage.Skills = append(usage.Skills, ConversationSkillUsage{
			RecordID:                 ref.RecordID,
			Identity:                 ref.Identity,
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
) artifactstore.Diagnostic {
	return artifactstore.Diagnostic{
		Severity: artifactstore.DiagnosticError,
		Code:     code,
		Message:  artifactstore.BoundedDiagnosticMessage(message),
	}
}
