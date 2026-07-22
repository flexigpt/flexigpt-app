package provider

import (
	"context"
	"errors"
	"strings"

	"github.com/flexigpt/agentskills-go"
	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/record"
	"github.com/flexigpt/flexigpt-app/internal/workspace/skilladapter"
)

const workspaceIdentityPrefix = "workspace/"

type Workspace struct {
	adapter *skilladapter.Adapter
}

func NewWorkspace(
	adapter *skilladapter.Adapter,
) (*Workspace, error) {
	if adapter == nil {
		return nil, errors.New("Workspace Skill adapter is nil")
	}
	return &Workspace{adapter: adapter}, nil
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
		document := value.AgentSkillDocument()
		if value.MarkdownBody == "" {
			document.MarkdownBody = "Workspace Skill body is loaded only at runtime."
		}
		if err := agentskills.ValidateSkillDocument(document); err != nil {
			value.Diagnostics = artifactstore.AppendDiagnostics(
				value.Diagnostics,
				artifactstore.Diagnostic{
					Severity: artifactstore.DiagnosticError,
					Code:     "skill.provider.projection-invalid",
					Message:  err.Error(),
					Location: &artifactstore.DiagnosticLocation{
						Locator: value.Locator,
					},
				},
			)
		}
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
			RuntimeAllowed:    value.RuntimeAllowed,
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
	rootID, recordID, err := parseWorkspaceIdentity(request.Identity)
	if err != nil {
		return RenderedSkill{}, err
	}
	if request.Scope.WorkspaceRootID != "" &&
		request.Scope.WorkspaceRootID != rootID {
		return RenderedSkill{}, errors.New("Workspace Skill belongs to another scope")
	}
	plan, err := p.adapter.Load(
		ctx,
		rootID,
		[]artifactstore.RecordID{recordID},
	)
	if err != nil {
		return RenderedSkill{}, err
	}
	if len(plan.Skills) != 1 {
		return RenderedSkill{
			Available:   false,
			Diagnostics: artifactstore.CloneDiagnostics(plan.Diagnostics),
		}, nil
	}
	value := plan.Skills[0]
	document := value.AgentSkillDocument()
	if err := agentskills.ValidateSkillDocument(document); err != nil {
		return RenderedSkill{}, err
	}
	rendered, err := agentskills.RenderSkillDocument(
		document,
		request.Arguments,
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
		Diagnostics:      artifactstore.CloneDiagnostics(plan.Diagnostics),
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

var _ Provider = (*Workspace)(nil)
