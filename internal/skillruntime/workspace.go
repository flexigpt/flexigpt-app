package skillruntime

import (
	"context"
	"errors"
	"maps"

	"github.com/flexigpt/agentskills-go"
	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	skillruntimeSpec "github.com/flexigpt/flexigpt-app/internal/skillruntime/spec"
	"github.com/flexigpt/flexigpt-app/internal/workspace/skilladapter"
)

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

func (*Workspace) Owns(ref skillruntimeSpec.SkillRef) bool {
	return ref.Artifact != nil
}

func (p *Workspace) List(
	ctx context.Context,
	scope Scope,
) ([]Skill, error) {
	if scope.Workspace == nil {
		return []Skill{}, nil
	}
	if err := scope.Workspace.Validate(); err != nil {
		return nil, err
	}
	values, err := p.adapter.List(ctx, *scope.Workspace)
	if err != nil {
		return nil, err
	}
	output := make([]Skill, 0, len(values))
	for _, value := range values {
		if !value.ProjectionValid {
			continue
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
		diagnostics := artifactstore.CloneDiagnostics(value.Diagnostics)
		if !value.WorkspaceEnabled {
			diagnostics = artifactstore.AppendDiagnostics(
				diagnostics,
				unavailableDiagnostic(
					"skill.provider.workspace-disabled",
					"the Workspace containing this Skill is disabled",
				),
			)
		}
		runtimeAllowed := value.WorkspaceEnabled &&
			value.Skill.IsEnabled &&
			value.State == artifact.StateAvailable &&
			value.CatalogCurrent &&
			!value.RuntimeDisabled
		artifactRef := value.Artifact
		workspaceRef := value.Workspace
		projected := Skill{
			Ref: skillruntimeSpec.SkillRef{
				Artifact: &artifactRef,
			},
			Origin:           OriginWorkspace,
			Workspace:        &workspaceRef,
			ArtifactRevision: value.ArtifactRevision,
			Name:             value.Skill.Name,
			DisplayName:      value.Skill.DisplayName,
			Description:      value.Skill.Description,
			Insert:           agentskillsSpec.SkillInsert(value.Skill.Insert),
			Arguments:        arguments,
			Tags:             append([]string(nil), value.Skill.Tags...),
			Enabled:          value.Skill.IsEnabled,
			Available:        value.WorkspaceEnabled && value.State == artifact.StateAvailable,
			RuntimeAllowed:   runtimeAllowed,
			CatalogCurrent:   value.CatalogCurrent,
			State:            string(value.State),
			DefinitionDigest: string(value.DefinitionDigest),
			SourceID:         value.SourceID,
			Locator:          value.Locator,
			Diagnostics:      diagnostics,
			CreatedAt:        value.Skill.CreatedAt,
			ModifiedAt:       value.Skill.ModifiedAt,
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
	if request.Ref.Artifact == nil {
		return RenderedSkill{}, errors.New(
			"Workspace Skill render request has no ArtifactRef",
		)
	}
	if err := request.Ref.Artifact.Validate(); err != nil {
		return RenderedSkill{}, err
	}
	if request.Scope.Workspace != nil {
		if err := request.Scope.Workspace.Validate(); err != nil {
			return RenderedSkill{}, err
		}
	}
	definition, workspaceRef, found := p.runtime.workspaceDefinitionForArtifact(
		ctx,
		*request.Ref.Artifact,
	)
	if !found {
		return RenderedSkill{
			Available: false,
			Diagnostics: []artifactstore.Diagnostic{
				unavailableDiagnostic(
					"skill.provider.workspace-unavailable",
					"the Workspace Skill is unavailable or no longer current",
				),
			},
		}, nil
	}
	if request.Scope.Workspace != nil &&
		*request.Scope.Workspace != workspaceRef {
		return RenderedSkill{}, errors.New(
			"Workspace Skill belongs to another Workspace scope",
		)
	}
	rendered, err := p.runtime.runtime.RenderSkill(
		ctx,
		agentskills.RenderSkillParams{
			Def:       definition,
			Arguments: request.Arguments,
		},
	)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return RenderedSkill{}, ctxErr
		}
		return RenderedSkill{
			Available: false,
			Diagnostics: []artifactstore.Diagnostic{
				unavailableDiagnostic(
					"skill.provider.render-unavailable",
					"the Workspace Skill could not be rendered",
				),
			},
		}, nil
	}
	list, err := p.List(ctx, Scope{Workspace: &workspaceRef})
	if err != nil {
		return RenderedSkill{}, err
	}
	var projected Skill
	for _, item := range list {
		if item.Ref.Artifact != nil &&
			*item.Ref.Artifact == *request.Ref.Artifact {
			projected = item
			break
		}
	}
	if projected.Ref.Artifact == nil {
		return RenderedSkill{
			Available: false,
			Diagnostics: []artifactstore.Diagnostic{
				unavailableDiagnostic(
					"skill.provider.workspace-unavailable",
					"the Workspace Skill became unavailable while it was rendered",
				),
			},
		}, nil
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

func cloneStrings(value map[string]string) map[string]string {
	if value == nil {
		return nil
	}
	output := make(map[string]string, len(value))
	maps.Copy(output, value)
	return output
}
