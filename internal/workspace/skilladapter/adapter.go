package skilladapter

import (
	"context"
	"fmt"
	"sort"
	"time"

	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/record"
	"github.com/flexigpt/flexigpt-app/internal/workspace/engine"
)

type SkillArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Default     string `json:"default,omitempty"`
}

type SkillSummary struct {
	SchemaVersion string                 `json:"schemaVersion"`
	ID            artifactstore.RecordID `json:"id"`
	Slug          string                 `json:"slug"`
	Name          string                 `json:"name"`
	DisplayName   string                 `json:"displayName"`
	Description   string                 `json:"description"`
	Tags          []string               `json:"tags,omitempty"`
	Insert        string                 `json:"insert"`
	Arguments     []SkillArgument        `json:"arguments,omitempty"`
	IsEnabled     bool                   `json:"isEnabled"`
	CreatedAt     time.Time              `json:"createdAt"`
	ModifiedAt    time.Time              `json:"modifiedAt"`
}

type WorkspaceSkill struct {
	RootID           artifactstore.RootID       `json:"rootID"`
	RecordID         artifactstore.RecordID     `json:"recordID"`
	DefinitionDigest artifactstore.Digest       `json:"definitionDigest"`
	SourceID         artifactstore.SourceID     `json:"sourceID"`
	Locator          artifactstore.Locator      `json:"locator"`
	Skill            SkillSummary               `json:"skill"`
	MarkdownBody     string                     `json:"markdownBody,omitempty"`
	Priority         int                        `json:"priority"`
	RecordRevision   uint64                     `json:"recordRevision"`
	State            record.State               `json:"state"`
	CatalogCurrent   bool                       `json:"catalogCurrent"`
	RuntimeAllowed   bool                       `json:"runtimeAllowed"`
	Diagnostics      []artifactstore.Diagnostic `json:"diagnostics,omitempty"`
}

type SkillLoadPlan struct {
	RootID          artifactstore.RootID       `json:"rootID"`
	CatalogRevision uint64                     `json:"catalogRevision"`
	Skills          []WorkspaceSkill           `json:"skills"`
	Diagnostics     []artifactstore.Diagnostic `json:"diagnostics,omitempty"`
}

type Adapter struct {
	query         *engine.QueryService
	runtimePolicy engine.SourceUsePolicy
}

func NewAdapter(
	query *engine.QueryService,
	runtimePolicy engine.SourceUsePolicy,
) (*Adapter, error) {
	if query == nil || runtimePolicy == nil {
		return nil, fmt.Errorf(
			"%w: Workspace Skill adapter dependencies are incomplete",
			engine.ErrInvalidWorkspace,
		)
	}
	return &Adapter{
		query:         query,
		runtimePolicy: runtimePolicy,
	}, nil
}

func (f *Adapter) List(
	ctx context.Context,
	rootID artifactstore.RootID,
) ([]WorkspaceSkill, error) {
	view, err := f.query.Catalog(ctx, rootID)
	if err != nil {
		return nil, err
	}
	priorities := attachmentPriorities(view.Workspace)
	output := make([]WorkspaceSkill, 0)
	for _, resourceValue := range view.Resources {
		if resourceValue.Definition.Kind != skillKind ||
			resourceValue.Definition.SchemaID != skillSchemaID {
			continue
		}
		value, err := projectWorkspaceSkill(
			rootID,
			resourceValue,
			priorities[resourceValue.Source.ID],
			false,
		)
		if err != nil {
			value.Diagnostics = artifactstore.AppendDiagnostics(
				value.Diagnostics,
				skillProjectionDiagnostic(resourceValue.Record, err),
			)
		}
		output = append(output, value)
	}
	sortWorkspaceSkills(output)
	return output, nil
}

func (f *Adapter) Load(
	ctx context.Context,
	rootID artifactstore.RootID,
	recordIDs []artifactstore.RecordID,
) (SkillLoadPlan, error) {
	loadPlan, err := f.query.ComposeLoadPlan(ctx, rootID, recordIDs)
	if err != nil {
		return SkillLoadPlan{}, err
	}
	workspaceValue, err := f.query.GetWorkspace(ctx, rootID)
	if err != nil {
		return SkillLoadPlan{}, err
	}
	priorities := attachmentPriorities(workspaceValue)
	output := SkillLoadPlan{
		RootID:          rootID,
		CatalogRevision: loadPlan.CatalogRevision,
		Diagnostics:     loadPlan.Diagnostics,
	}
	for _, item := range loadPlan.Items {
		if err := ValidateSkillDefinition(item.Definition); err != nil {
			output.Diagnostics = artifactstore.AppendDiagnostics(
				output.Diagnostics,
				skillProjectionDiagnostic(item.Record, err),
			)
			continue
		}
		decision := f.runtimePolicy.Decide(ctx, engine.RuntimePolicyRequest{
			Use:              engine.RuntimeUseSkill,
			Workspace:        workspaceValue,
			Record:           item.Record,
			DefinitionDigest: item.Definition.Digest,
			SourceID:         item.Source.ID,
			TrustReference:   workspaceValue.Data.TrustReference,
		})
		if err := decision.Validate(); err != nil {
			return SkillLoadPlan{}, err
		}
		if decision.Disposition != engine.RuntimeAllowed {
			output.Diagnostics = artifactstore.AppendDiagnostics(
				output.Diagnostics,
				engine.RuntimeDecisionDiagnostic(decision, item.Record),
			)
			continue
		}
		resourceValue := engine.Resource{
			Record:          item.Record,
			Definition:      item.Definition,
			Source:          item.Source,
			ProjectionValid: true,
		}
		projected, err := projectWorkspaceSkill(
			rootID,
			resourceValue,
			priorities[item.Source.ID],
			true,
		)
		if err != nil {
			output.Diagnostics = artifactstore.AppendDiagnostics(
				output.Diagnostics,
				skillProjectionDiagnostic(item.Record, err),
			)
			continue
		}
		output.Skills = append(output.Skills, projected)
	}
	sortWorkspaceSkills(output.Skills)
	return output, nil
}

func projectWorkspaceSkill(
	rootID artifactstore.RootID,
	resourceValue engine.Resource,
	priority int,
	includeMarkdown bool,
) (WorkspaceSkill, error) {
	runtimeAllowed, dataErr := engine.RecordRuntimeAllowed(resourceValue.Record)
	output := WorkspaceSkill{
		RootID:           rootID,
		RecordID:         resourceValue.Record.ID,
		RecordRevision:   resourceValue.Record.Revision,
		DefinitionDigest: resourceValue.Definition.Digest,
		SourceID:         resourceValue.Source.ID,
		Locator:          resourceValue.Record.Occurrence.Locator,
		Priority:         priority,
		State:            resourceValue.Record.State,
		CatalogCurrent:   resourceValue.CatalogCurrent,
		RuntimeAllowed:   runtimeAllowed,
		Diagnostics: artifactstore.AppendDiagnostics(
			resourceValue.Record.Diagnostics,
			resourceValue.Diagnostics...,
		),
	}
	if dataErr != nil {
		return output, dataErr
	}
	if err := ValidateSkillDefinition(resourceValue.Definition); err != nil {
		return output, err
	}
	body, err := engine.DecodeDefinitionBody[skillDefinition](
		resourceValue.Definition.Body,
	)
	if err != nil {
		return output, err
	}
	markdownBody := ""
	if includeMarkdown {
		markdownBody = body.MarkdownBody
	}
	output.Skill = skillSummary(resourceValue.Record, body)
	output.MarkdownBody = markdownBody
	return output, nil
}

func skillSummary(
	recordValue record.Record,
	value skillDefinition,
) SkillSummary {
	arguments := make([]SkillArgument, 0, len(value.Arguments))
	for _, argument := range value.Arguments {
		arguments = append(arguments, SkillArgument(argument))
	}
	return SkillSummary{
		SchemaVersion: workspaceSkillsSchemaVersionV1,

		ID:   recordValue.ID,
		Slug: value.Name,

		Name:        value.Name,
		DisplayName: value.DisplayName,
		Description: value.Description,
		Tags:        append([]string(nil), value.Tags...),
		Insert:      value.Insert,
		Arguments:   arguments,
		IsEnabled:   recordValue.Enabled,
		CreatedAt:   recordValue.CreatedAt,
		ModifiedAt:  recordValue.ModifiedAt,
	}
}

func sortWorkspaceSkills(values []WorkspaceSkill) {
	sort.Slice(values, func(left, right int) bool {
		if values[left].Skill.Name != values[right].Skill.Name {
			return values[left].Skill.Name < values[right].Skill.Name
		}
		return values[left].RecordID < values[right].RecordID
	})
}

func attachmentPriorities(
	value engine.Workspace,
) map[artifactstore.SourceID]int {
	output := make(map[artifactstore.SourceID]int, len(value.Attachments))
	for _, attachment := range value.Attachments {
		if attachment.Enabled {
			output[attachment.SourceID] = attachment.Priority
		}
	}
	return output
}

func (s WorkspaceSkill) AgentSkillDocument() agentskillsSpec.SkillDocument {
	arguments := make([]agentskillsSpec.SkillArgument, 0, len(s.Skill.Arguments))
	for _, argument := range s.Skill.Arguments {
		arguments = append(arguments, agentskillsSpec.SkillArgument{
			Name:        argument.Name,
			Description: argument.Description,
			Default:     argument.Default,
		})
	}
	return agentskillsSpec.SkillDocument{
		Name:         s.Skill.Name,
		DisplayName:  s.Skill.DisplayName,
		Description:  s.Skill.Description,
		Insert:       agentskillsSpec.SkillInsert(s.Skill.Insert),
		Arguments:    arguments,
		Tags:         append([]string(nil), s.Skill.Tags...),
		MarkdownBody: s.MarkdownBody,
	}
}

func skillProjectionDiagnostic(
	value record.Record,
	err error,
) artifactstore.Diagnostic {
	return artifactstore.Diagnostic{
		Severity: artifactstore.DiagnosticError,
		Code:     engine.DiagnosticCodeProjectionInvalid,
		Message:  err.Error(),
		Location: &artifactstore.DiagnosticLocation{
			Locator:            value.Occurrence.Locator,
			SubresourceLocator: value.Occurrence.SubresourceLocator,
		},
	}
}
