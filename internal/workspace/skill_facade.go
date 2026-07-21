package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/record"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/fsdir"
	"github.com/flexigpt/flexigpt-app/internal/skill/spec"
	skillStore "github.com/flexigpt/flexigpt-app/internal/skill/store"
)

type WorkspaceSkill struct {
	RootID            artifactstore.RootID     `json:"rootID"`
	RecordID          artifactstore.RecordID   `json:"recordID"`
	DefinitionDigest  artifactstore.Digest     `json:"definitionDigest"`
	SourceID          artifactstore.SourceID   `json:"sourceID"`
	Locator           artifactstore.Locator    `json:"locator"`
	Skill             spec.Skill               `json:"skill"`
	MarkdownBody      string                   `json:"markdownBody"`
	RuntimeDefinition agentskillsSpec.SkillDef `json:"runtimeDefinition"`
}

type SkillLoadPlan struct {
	RootID          artifactstore.RootID       `json:"rootID"`
	CatalogRevision uint64                     `json:"catalogRevision"`
	Skills          []WorkspaceSkill           `json:"skills"`
	Diagnostics     []artifactstore.Diagnostic `json:"diagnostics,omitempty"`
}

type SkillFacade struct {
	query *QueryService
}

func NewSkillFacade(
	query *QueryService,
) (*SkillFacade, error) {
	if query == nil {
		return nil, fmt.Errorf(
			"%w: Workspace Skill facade query is nil",
			ErrInvalidWorkspace,
		)
	}
	return &SkillFacade{query: query}, nil
}

func (f *SkillFacade) List(
	ctx context.Context,
	rootID artifactstore.RootID,
) ([]WorkspaceSkill, error) {
	view, err := f.query.Catalog(ctx, rootID)
	if err != nil {
		return nil, err
	}
	output := make([]WorkspaceSkill, 0)
	for _, resourceValue := range view.Resources {
		if resourceValue.Definition.Kind != SkillKind ||
			resourceValue.Definition.SchemaID != SkillSchemaID ||
			resourceValue.Record.State != record.StateAvailable {
			continue
		}
		value, err := projectWorkspaceSkill(rootID, resourceValue)
		if err != nil {
			return nil, err
		}
		output = append(output, value)
	}
	sortWorkspaceSkills(output)
	return output, nil
}

func (f *SkillFacade) Load(
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
		if item.Definition.Kind != SkillKind ||
			item.Definition.SchemaID != SkillSchemaID {
			return SkillLoadPlan{}, fmt.Errorf(
				"%w: record %q is not a Workspace Skill",
				ErrInvalidWorkspace,
				item.Record.ID,
			)
		}
		resourceValue := Resource{
			Record:     item.Record,
			Definition: item.Definition,
			Source:     item.Source,
		}
		projected, err := projectWorkspaceSkill(rootID, resourceValue)
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
	resourceValue Resource,
) (WorkspaceSkill, error) {
	body, err := decodeDefinitionBody[SkillDefinition](
		resourceValue.Definition.Body,
	)
	if err != nil {
		return WorkspaceSkill{}, err
	}
	location, err := workspaceSkillLocation(
		resourceValue.Source.Kind,
		resourceValue.Source.Config,
		resourceValue.Record.Occurrence.Locator,
	)
	if err != nil {
		return WorkspaceSkill{}, err
	}

	arguments := make([]spec.SkillArgument, 0, len(body.Arguments))
	for _, value := range body.Arguments {
		arguments = append(arguments, spec.SkillArgument{
			Name:        value.Name,
			Description: value.Description,
			Default:     value.Default,
		})
	}
	insert := spec.SkillInsert(body.Insert)
	projected := spec.Skill{
		SchemaVersion:  spec.SkillSchemaVersion,
		ID:             spec.SkillID(resourceValue.Record.ID),
		Slug:           spec.SkillSlug(body.Name),
		Type:           spec.SkillTypeFS,
		Location:       location,
		Name:           body.Name,
		DisplayName:    body.DisplayName,
		Description:    body.Description,
		Tags:           append([]string(nil), body.Tags...),
		Insert:         insert,
		Arguments:      arguments,
		RawFrontmatter: cloneMap(body.RawFrontmatter),
		Presence: &spec.SkillPresence{
			Status: spec.SkillPresencePresent,
		},
		IsEnabled:  resourceValue.Record.Enabled,
		IsBuiltIn:  false,
		CreatedAt:  resourceValue.Record.CreatedAt,
		ModifiedAt: resourceValue.Record.ModifiedAt,
	}
	if err := skillStore.ValidateSkill(&projected); err != nil {
		return WorkspaceSkill{}, fmt.Errorf(
			"%w: project Workspace Skill: %w",
			ErrInvalidWorkspace,
			err,
		)
	}

	return WorkspaceSkill{
		RootID:           rootID,
		RecordID:         resourceValue.Record.ID,
		DefinitionDigest: resourceValue.Definition.Digest,
		SourceID:         resourceValue.Source.ID,
		Locator:          resourceValue.Record.Occurrence.Locator,
		Skill:            projected,
		MarkdownBody:     body.MarkdownBody,
		RuntimeDefinition: agentskillsSpec.SkillDef{
			Type:     "fs",
			Name:     body.Name,
			Location: location,
		},
	}, nil
}

func workspaceSkillLocation(
	kind artifactstore.SourceKind,
	config json.RawMessage,
	locator artifactstore.Locator,
) (string, error) {
	if kind != fsdir.Kind {
		return "", fmt.Errorf(
			"%w: Workspace Skill source kind %q cannot provide a filesystem runtime location",
			artifactstore.ErrUnsupported,
			kind,
		)
	}
	var value fsdir.Config
	if err := json.Unmarshal(config, &value); err != nil {
		return "", err
	}
	if !filepath.IsAbs(value.RootPath) {
		return "", fmt.Errorf(
			"%w: Workspace Skill source root is not absolute",
			ErrInvalidWorkspace,
		)
	}

	relativeDirectory := path.Dir(string(locator))
	location := filepath.Clean(filepath.Join(
		value.RootPath,
		filepath.FromSlash(relativeDirectory),
	))
	relative, err := filepath.Rel(value.RootPath, location)
	if err != nil {
		return "", err
	}
	if relative == ".." ||
		strings.HasPrefix(relative, ".."+string(filepath.Separator)) ||
		filepath.IsAbs(relative) {
		return "", fmt.Errorf(
			"%w: Workspace Skill location escapes source root",
			ErrInvalidWorkspace,
		)
	}
	return location, nil
}

func sortWorkspaceSkills(values []WorkspaceSkill) {
	sort.Slice(values, func(left, right int) bool {
		if values[left].Skill.Name != values[right].Skill.Name {
			return values[left].Skill.Name < values[right].Skill.Name
		}
		return values[left].RecordID < values[right].RecordID
	})
}

func cloneMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return nil
	}
	var output map[string]any
	if err := json.Unmarshal(raw, &output); err != nil {
		return nil
	}
	return output
}
