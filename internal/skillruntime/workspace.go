package skillruntime

import (
	"context"
	"errors"
	"maps"
	"strings"

	"github.com/flexigpt/agentskills-go"
	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/record"

	"github.com/flexigpt/flexigpt-app/internal/workspace/skilladapter"
)

const workspaceIdentityPrefix = "workspace/"

type Workspace struct {
	runtime *SkillRuntime
	adapter *skilladapter.Adapter
}

func NewWorkspace(
	runtime *SkillRuntime,
) (*Workspace, error) {
	if runtime == nil || runtime.workspaceSkills == nil {
		return nil, errors.New("Workspace Skill adapter is nil")
	}
	return &Workspace{
		runtime: runtime,
		adapter: runtime.workspaceSkills,
	}, nil
}

func (*Workspace) Owns(identity string) bool {
	return strings.HasPrefix(identity, workspaceIdentityPrefix)
}

func (p *Workspace) List(
	ctx context.Context,
	scope Scope,
) ([]Skill, error) {
	if scope.WorkspaceRootID == "" {
		return []Skill{}, nil
	}
	values, err := p.adapter.List(ctx, scope.WorkspaceRootID)
	if err != nil {
		return nil, err
	}
	output := make([]Skill, 0, len(values))
	for _, value := range values {
		arguments := make(
			[]agentskillsSpec.SkillArgument,
			0,
			len(value.Skill.Arguments),
		)
		for _, argument := range value.Skill.Arguments {
			arguments = append(arguments, agentskillsSpec.SkillArgument{
				Name:        argument.Name,
				Description: argument.Description,
				Default:     argument.Default,
			})
		}
		projected := Skill{
			Identity:          workspaceIdentity(value.RootID, value.RecordID),
			Origin:            OriginWorkspace,
			WorkspaceRootID:   value.RootID,
			WorkspaceRecordID: value.RecordID,
			RecordRevision:    value.RecordRevision,
			Name:              value.Skill.Name,
			DisplayName:       value.Skill.DisplayName,
			Description:       value.Skill.Description,
			Insert:            agentskillsSpec.SkillInsert(value.Skill.Insert),
			Arguments:         arguments,
			Tags:              append([]string(nil), value.Skill.Tags...),
			Enabled:           value.Skill.IsEnabled,
			Available:         value.State == record.StateAvailable,
			RuntimeAllowed:    !value.RuntimeDisabled,
			Priority:          value.Priority,
			CatalogCurrent:    value.CatalogCurrent,
			State:             string(value.State),
			DefinitionDigest:  string(value.DefinitionDigest),
			SourceID:          value.SourceID,
			Locator:           value.Locator,
			Diagnostics:       artifactstore.CloneDiagnostics(value.Diagnostics),
			CreatedAt:         value.Skill.CreatedAt,
			ModifiedAt:        value.Skill.ModifiedAt,
		}
		if err := projected.Validate(); err != nil {
			return nil, err
		}
		output = append(output, projected)
	}
	return output, nil
}

func (p *Workspace) Render(
	ctx context.Context,
	request RenderRequest,
) (RenderedSkill, error) {
	rootID, _, err := parseWorkspaceIdentity(request.Identity)
	if err != nil {
		return RenderedSkill{}, err
	}
	if request.Scope.WorkspaceRootID != "" &&
		request.Scope.WorkspaceRootID != rootID {
		return RenderedSkill{}, errors.New("Workspace Skill belongs to another scope")
	}
	definition, found := p.runtime.workspaceDefinitionForIdentity(
		ctx,
		request.Identity,
	)
	if !found {
		return RenderedSkill{Available: false}, nil
	}
	rendered, err := p.runtime.runtime.RenderSkill(
		ctx,
		agentskills.RenderSkillParams{
			Def:       definition,
			Arguments: request.Arguments,
		},
	)
	if err != nil {
		return RenderedSkill{}, err
	}
	list, err := p.List(ctx, Scope{WorkspaceRootID: rootID})
	if err != nil {
		return RenderedSkill{}, err
	}
	var projected Skill
	for _, item := range list {
		if item.Identity == request.Identity {
			projected = item
			break
		}
	}
	return RenderedSkill{
		Skill:            projected,
		Available:        true,
		Text:             rendered.Text,
		Insert:           rendered.Insert,
		Arguments:        append([]agentskillsSpec.SkillArgument(nil), rendered.Arguments...),
		AppliedArguments: cloneStrings(rendered.AppliedArguments),
		Diagnostics:      artifactstore.CloneDiagnostics(projected.Diagnostics),
	}, nil
}

func workspaceIdentity(
	rootID artifactstore.RootID,
	recordID artifactstore.RecordID,
) string {
	return workspaceIdentityPrefix + string(rootID) + "/" + string(recordID)
}

func parseWorkspaceIdentity(
	value string,
) (artifactstore.RootID, artifactstore.RecordID, error) {
	relative, found := strings.CutPrefix(value, workspaceIdentityPrefix)
	if !found {
		return "", "", errors.New("identity is not a Workspace Skill")
	}
	parts := strings.Split(relative, "/")
	if len(parts) != 2 {
		return "", "", errors.New("Workspace Skill identity is invalid")
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

func cloneStrings(value map[string]string) map[string]string {
	if value == nil {
		return nil
	}
	output := make(map[string]string, len(value))
	maps.Copy(output, value)
	return output
}
