package skilladapter

import (
	"context"
	"fmt"
	"sort"
	"time"

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
	RootID           artifactstore.RootID   `json:"rootID"`
	RecordID         artifactstore.RecordID `json:"recordID"`
	DefinitionDigest artifactstore.Digest   `json:"definitionDigest"`
	SourceID         artifactstore.SourceID `json:"sourceID"`
	Locator          artifactstore.Locator  `json:"locator"`
	Skill            SkillSummary           `json:"skill"`
	MarkdownBody     string                 `json:"markdownBody,omitempty"`
}

type SkillLoadPlan struct {
	RootID          artifactstore.RootID       `json:"rootID"`
	CatalogRevision uint64                     `json:"catalogRevision"`
	Skills          []WorkspaceSkill           `json:"skills"`
	Diagnostics     []artifactstore.Diagnostic `json:"diagnostics,omitempty"`
}

type Adapter struct {
	query *engine.QueryService
}

func NewAdapter(
	query *engine.QueryService,
) (*Adapter, error) {
	if query == nil {
		return nil, fmt.Errorf(
			"%w: Workspace Skill adapter query is nil",
			engine.ErrInvalidWorkspace,
		)
	}
	return &Adapter{query: query}, nil
}

func (f *Adapter) List(
	ctx context.Context,
	rootID artifactstore.RootID,
) ([]WorkspaceSkill, error) {
	view, err := f.query.Catalog(ctx, rootID)
	if err != nil {
		return nil, err
	}
	output := make([]WorkspaceSkill, 0)
	for _, resourceValue := range view.Resources {
		if resourceValue.Definition.Kind != skillKind ||
			resourceValue.Definition.SchemaID != skillSchemaID ||
			resourceValue.Record.State != record.StateAvailable ||
			!resourceValue.Record.Enabled {
			continue
		}
		value, err := projectWorkspaceSkill(rootID, resourceValue, false)
		if err != nil {
			return nil, err
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
	output := SkillLoadPlan{
		RootID:          rootID,
		CatalogRevision: loadPlan.CatalogRevision,
		Diagnostics:     loadPlan.Diagnostics,
	}
	for _, item := range loadPlan.Items {
		if item.Definition.Kind != skillKind ||
			item.Definition.SchemaID != skillSchemaID {
			return SkillLoadPlan{}, fmt.Errorf(
				"%w: record %q is not a Workspace Skill",
				engine.ErrInvalidWorkspace,
				item.Record.ID,
			)
		}
		resourceValue := engine.Resource{
			Record:     item.Record,
			Definition: item.Definition,
			Source:     item.Source,
		}
		projected, err := projectWorkspaceSkill(rootID, resourceValue, true)
		if err != nil {
			return SkillLoadPlan{}, err
		}
		output.Skills = append(output.Skills, projected)
	}
	sortWorkspaceSkills(output.Skills)
	return output, nil
}

func projectWorkspaceSkill(
	rootID artifactstore.RootID,
	resourceValue engine.Resource,
	includeMarkdown bool,
) (WorkspaceSkill, error) {
	body, err := engine.DecodeDefinitionBody[skillDefinition](
		resourceValue.Definition.Body,
	)
	if err != nil {
		return WorkspaceSkill{}, err
	}
	markdownBody := ""
	if includeMarkdown {
		markdownBody = body.MarkdownBody
	}

	return WorkspaceSkill{
		RootID:           rootID,
		RecordID:         resourceValue.Record.ID,
		DefinitionDigest: resourceValue.Definition.Digest,
		SourceID:         resourceValue.Source.ID,
		Locator:          resourceValue.Record.Occurrence.Locator,
		Skill: skillSummary(
			resourceValue.Record,
			body,
		),
		MarkdownBody: markdownBody,
	}, nil
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
