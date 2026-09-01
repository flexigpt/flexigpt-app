package workspaceadapter

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/flexigpt/agentskills-go/document"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	skillArtifact "github.com/flexigpt/flexigpt-app/internal/skill/artifact"
	"github.com/flexigpt/flexigpt-app/internal/workspace/artifactadapter"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

type SkillArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Default     string `json:"default,omitempty"`
}

type SkillSummary struct {
	SchemaVersion string              `json:"schemaVersion"`
	ID            basespec.ArtifactID `json:"id"`
	Slug          string              `json:"slug"`
	Name          string              `json:"name"`
	DisplayName   string              `json:"displayName"`
	Description   string              `json:"description"`
	Tags          []string            `json:"tags,omitempty"`
	Insert        string              `json:"insert"`
	Arguments     []SkillArgument     `json:"arguments,omitempty"`
	IsEnabled     bool                `json:"isEnabled"`
	CreatedAt     time.Time           `json:"createdAt"`
	ModifiedAt    time.Time           `json:"modifiedAt"`
}

type WorkspaceSkill struct {
	Workspace        collection.CollectionRef `json:"workspace"`
	Artifact         artifact.ArtifactRef     `json:"artifact"`
	DefinitionDigest cryptoutil.Digest        `json:"definitionDigest"`
	SourceID         basespec.SourceID        `json:"sourceID"`
	Locator          basespec.Locator         `json:"locator"`
	Skill            SkillSummary             `json:"skill"`
	MarkdownBody     string                   `json:"markdownBody,omitempty"`
	ArtifactRevision uint64                   `json:"artifactRevision"`
	State            artifact.State           `json:"state"`
	CatalogCurrent   bool                     `json:"catalogCurrent"`
	WorkspaceEnabled bool                     `json:"-"`
	RuntimeDisabled  bool                     `json:"runtimeDisabled"`
	Diagnostics      []diagnostic.Diagnostic  `json:"diagnostics,omitempty"`

	ProjectionValid     bool              `json:"-"`
	RuntimePathBacked   bool              `json:"-"`
	SourceContentDigest cryptoutil.Digest `json:"-"`
	SourceGeneration    string            `json:"-"`
	RuntimeLocation     string            `json:"-"`
}

type SkillLoadPlan struct {
	Workspace       collection.CollectionRef `json:"workspace"`
	CatalogRevision uint64                   `json:"catalogRevision"`
	Skills          []WorkspaceSkill         `json:"skills"`
	Diagnostics     []diagnostic.Diagnostic  `json:"diagnostics,omitempty"`
}

type Adapter struct {
	query         *artifactadapter.QueryService
	runtimePolicy artifactadapter.SourceUsePolicy
	sourceRuntime source.Runtime
}

func NewAdapter(
	query *artifactadapter.QueryService,
	runtimePolicy artifactadapter.SourceUsePolicy,
	sourceRuntime source.Runtime,
) (*Adapter, error) {
	if query == nil || runtimePolicy == nil || sourceRuntime == nil {
		return nil, fmt.Errorf(
			"%w: Workspace Skill adapter dependencies are incomplete",
			spec.ErrInvalidWorkspace,
		)
	}
	return &Adapter{
		query:         query,
		runtimePolicy: runtimePolicy,
		sourceRuntime: sourceRuntime,
	}, nil
}

func (f *Adapter) List(
	ctx context.Context,
	workspace collection.CollectionRef,
) ([]WorkspaceSkill, error) {
	if err := workspace.Validate(); err != nil {
		return nil, err
	}
	view, err := f.query.Catalog(ctx, workspace)
	if err != nil {
		return nil, err
	}
	output := make([]WorkspaceSkill, 0)
	for _, resourceValue := range view.Resources {
		if resourceValue.Definition.Kind != artifactbuiltin.AgentSkillArtifactKind ||
			resourceValue.Definition.SchemaID != artifactbuiltin.AgentSkillSchemaID {
			continue
		}
		value, err := projectWorkspaceSkill(
			workspace,
			resourceValue,
			view.Workspace.Collection.Enabled,
			false,
			f.supportsRuntimePath(resourceValue.Source.Kind),
		)
		if err != nil {
			value.Diagnostics = diagnostic.AppendDiagnostics(
				value.Diagnostics,
				skillProjectionDiagnostic(resourceValue.Artifact, err),
			)
		}
		output = append(output, value)
	}
	sortWorkspaceSkills(output)
	return output, nil
}

func (f *Adapter) LoadArtifact(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (WorkspaceSkill, error) {
	workspaceValue, resourceValue, err := f.query.ResolveArtifact(ctx, ref)
	if err != nil {
		return WorkspaceSkill{}, err
	}
	workspace := workspaceValue.Collection.Ref()
	if resourceValue.Definition.Kind != artifactbuiltin.AgentSkillArtifactKind ||
		resourceValue.Definition.SchemaID != artifactbuiltin.AgentSkillSchemaID {
		return WorkspaceSkill{}, fmt.Errorf(
			"%w: Artifact %q is not an Agent Skill",
			spec.ErrReferenceUnresolved,
			ref.ArtifactID,
		)
	}
	plan, err := f.Load(ctx, workspace, []artifact.ArtifactRef{ref})
	if err != nil {
		return WorkspaceSkill{}, err
	}
	if len(plan.Skills) != 1 {
		return WorkspaceSkill{}, fmt.Errorf(
			"%w: Artifact %q is unavailable for runtime loading",
			spec.ErrReferenceUnresolved,
			ref.ArtifactID,
		)
	}
	return plan.Skills[0], nil
}

func (f *Adapter) Load(
	ctx context.Context,
	workspace collection.CollectionRef,
	artifactRefs []artifact.ArtifactRef,
) (SkillLoadPlan, error) {
	seen := make(map[artifact.ArtifactRef]struct{}, len(artifactRefs))
	for _, ref := range artifactRefs {
		if err := ref.Validate(); err != nil {
			return SkillLoadPlan{}, err
		}
		if ref.RootID != workspace.RootID {
			return SkillLoadPlan{}, fmt.Errorf(
				"%w: Workspace Skill belongs to another Root",
				spec.ErrReferenceUnresolved,
			)
		}
		if _, duplicate := seen[ref]; duplicate {
			return SkillLoadPlan{}, fmt.Errorf(
				"%w: duplicate Workspace Skill Artifact %q",
				spec.ErrInvalidWorkspace,
				ref.ArtifactID,
			)
		}
		seen[ref] = struct{}{}
	}
	return f.loadLocal(ctx, workspace, artifactRefs)
}

func (f *Adapter) loadLocal(
	ctx context.Context,
	workspace collection.CollectionRef,
	artifactRefs []artifact.ArtifactRef,
) (SkillLoadPlan, error) {
	if err := workspace.Validate(); err != nil {
		return SkillLoadPlan{}, err
	}
	loadPlan, err := f.query.ComposeLoadPlan(
		ctx,
		workspace,
		artifactRefs,
	)
	if err != nil {
		return SkillLoadPlan{}, err
	}
	workspaceValue, err := f.query.GetWorkspace(ctx, workspace)
	if err != nil {
		return SkillLoadPlan{}, err
	}

	output := SkillLoadPlan{
		Workspace:       workspace,
		CatalogRevision: loadPlan.CatalogRevision,
		Diagnostics:     diagnostic.CloneDiagnostics(loadPlan.Diagnostics),
	}

	for _, item := range loadPlan.Items {
		resourceValue := spec.Resource{
			Artifact:        item.Artifact,
			Definition:      item.Definition,
			Source:          item.Source,
			CatalogCurrent:  item.CatalogCurrent,
			ProjectionValid: true,
		}
		projected, err := projectWorkspaceSkill(
			workspace,
			resourceValue,
			workspaceValue.Collection.Enabled,
			true,
			f.supportsRuntimePath(item.Source.Kind),
		)
		if err != nil {
			output.Diagnostics = diagnostic.AppendDiagnostics(
				output.Diagnostics,
				skillProjectionDiagnostic(item.Artifact, err),
			)
			continue
		}
		decision := f.runtimePolicy.Decide(ctx, artifactadapter.RuntimePolicyRequest{
			Use:              artifactadapter.RuntimeUseSkill,
			Workspace:        workspaceValue,
			Artifact:         item.Artifact,
			DefinitionDigest: item.Definition.Digest,
			SourceID:         item.Source.ID,
		})
		if err := decision.Validate(); err != nil {
			return SkillLoadPlan{}, err
		}
		if decision.Disposition != artifactadapter.RuntimeAllowed {
			output.Diagnostics = diagnostic.AppendDiagnostics(
				output.Diagnostics,
				artifactadapter.RuntimeDecisionDiagnostic(decision, item.Artifact),
			)
			continue
		}
		projected.SourceContentDigest = item.SourceContentDigest
		projected.SourceGeneration = item.SourceGeneration
		sourceValue, err := f.runtimeSource(ctx, item)
		if err != nil {
			output.Diagnostics = diagnostic.AppendDiagnostics(
				output.Diagnostics,
				runtimeLocationDiagnostic(item.Artifact, err),
			)
			continue
		}
		runtimeLocation, err := f.resolveRuntimeLocation(
			ctx,
			item,
			sourceValue,
		)
		if err != nil {
			output.Diagnostics = diagnostic.AppendDiagnostics(
				output.Diagnostics,
				runtimeLocationDiagnostic(item.Artifact, err),
			)
			continue
		}
		projected.RuntimeLocation = runtimeLocation
		output.Skills = append(output.Skills, projected)
	}
	sortWorkspaceSkills(output.Skills)
	return output, nil
}

func projectWorkspaceSkill(
	workspace collection.CollectionRef,
	resourceValue spec.Resource,
	workspaceEnabled bool,
	includeMarkdown bool,
	runtimePathBacked bool,
) (WorkspaceSkill, error) {
	runtimeDisabled, dataErr := artifactadapter.ArtifactRuntimeDisabled(
		resourceValue.Artifact,
	)
	output := WorkspaceSkill{
		Workspace:        workspace,
		Artifact:         resourceValue.Artifact.Ref(),
		ArtifactRevision: resourceValue.Artifact.Revision,
		DefinitionDigest: resourceValue.Definition.Digest,
		SourceID:         resourceValue.Source.ID,
		Locator:          resourceValue.Artifact.Binding.Locator,
		State:            resourceValue.Artifact.State,
		CatalogCurrent:   resourceValue.CatalogCurrent,
		RuntimeDisabled:  runtimeDisabled,
		WorkspaceEnabled: workspaceEnabled,
		Diagnostics: diagnostic.AppendDiagnostics(
			resourceValue.Artifact.Diagnostics,
			resourceValue.Diagnostics...,
		),
		RuntimePathBacked: runtimePathBacked,
	}
	if dataErr != nil {
		return output, dataErr
	}
	doc, err := skillArtifact.DocumentFromDefinition(
		resourceValue.Definition,
	)
	if err != nil {
		return output, err
	}
	markdownBody := ""
	if includeMarkdown {
		markdownBody = doc.MarkdownBody
	}
	output.Skill = skillSummary(resourceValue.Artifact, doc)
	output.MarkdownBody = markdownBody
	if resourceValue.Occurrence != nil &&
		resourceValue.Occurrence.SourceContentDigest != nil {
		output.SourceContentDigest = *resourceValue.Occurrence.SourceContentDigest
	}
	output.ProjectionValid = true
	return output, nil
}

func skillSummary(
	artifactValue artifact.Artifact,
	value document.SkillDocument,
) SkillSummary {
	arguments := make([]SkillArgument, 0, len(value.Arguments))
	for _, argument := range value.Arguments {
		arguments = append(arguments, SkillArgument{
			Name:        argument.Name,
			Description: argument.Description,
			Default:     argument.Default,
		})
	}
	return SkillSummary{
		SchemaVersion: artifactbuiltin.AgentSkillSchemaVersion,
		ID:            artifactValue.ID,
		Slug:          value.Name,
		Name:          value.Name,
		DisplayName:   value.DisplayName,
		Description:   value.Description,
		Tags:          append([]string(nil), value.Tags...),
		Insert:        string(value.Insert),
		Arguments:     arguments,
		IsEnabled:     artifactValue.Enabled,
		CreatedAt:     artifactValue.CreatedAt,
		ModifiedAt:    artifactValue.ModifiedAt,
	}
}

func sortWorkspaceSkills(values []WorkspaceSkill) {
	sort.Slice(values, func(left, right int) bool {
		if values[left].Skill.Name != values[right].Skill.Name {
			return values[left].Skill.Name < values[right].Skill.Name
		}
		return values[left].Artifact.ArtifactID < values[right].Artifact.ArtifactID
	})
}

// resolveRuntimeLocation performs the Workspace-owned handoff from a selected
// source-linked record to a native filesystem skill package. It does not
// register or execute the skill. Agent Skills runtime lifecycle remains in
// skillRuntime.
func (f *Adapter) runtimeSource(
	ctx context.Context,
	item spec.LoadPlanItem,
) (source.Source, error) {
	// Artifact Store owns Source state. Do not retain a feature-owned source
	// cache merely to avoid repeated reads during one load operation.
	sourceValue, err := f.sourceRuntime.Get(
		ctx,
		item.Artifact.RootID,
		item.Source.ID,
	)
	if err != nil {
		return source.Source{}, err
	}
	if sourceValue.ID != item.Source.ID ||
		sourceValue.Revision != item.Source.Revision ||
		sourceValue.Kind != item.Source.Kind {
		err := fmt.Errorf(
			"%w: Workspace Skill source changed after load-plan composition",
			basespec.ErrCatalogStale,
		)
		return source.Source{}, err
	}
	return sourceValue, nil
}

func (f *Adapter) resolveRuntimeLocation(
	ctx context.Context,
	item spec.LoadPlanItem,
	sourceValue source.Source,
) (string, error) {
	if item.SourceContentDigest == "" ||
		item.OccurrenceDefinitionDigest == "" {
		return "", fmt.Errorf(
			"%w: Workspace Skill has no current source occurrence",
			basespec.ErrCatalogStale,
		)
	}
	if item.OccurrenceDefinitionDigest != item.Definition.Digest {
		return "", fmt.Errorf(
			"%w: selected Workspace Skill definition does not match its current source occurrence",
			basespec.ErrCatalogStale,
		)
	}

	location, err := skillArtifact.ResolveRuntimePackage(
		ctx,
		f.sourceRuntime,
		sourceValue,
		item.Artifact.Binding.Locator,
		item.Artifact.Binding.SubresourceLocator,
		item.SourceGeneration,
		item.SourceContentDigest,
	)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", ctxErr
		}
		return "", err
	}
	return location, nil
}

func (f *Adapter) supportsRuntimePath(
	kind basespec.SourceKind,
) bool {
	localPaths, supported := f.sourceRuntime.(source.LocalPathRuntime)
	if !supported {
		return false
	}
	return localPaths.SupportsLocalPath(kind)
}

func runtimeLocationDiagnostic(
	value artifact.Artifact,
	err error,
) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{
		Severity: diagnostic.DiagnosticError,
		Code:     artifactadapter.DiagnosticCodeRuntimeUnavailable,
		Message:  diagnostic.BoundedDiagnosticMessage(err.Error()),
		Location: &diagnostic.DiagnosticLocation{
			Locator:            value.Binding.Locator,
			SubresourceLocator: value.Binding.SubresourceLocator,
		},
	}
}

func skillProjectionDiagnostic(
	value artifact.Artifact,
	err error,
) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{
		Severity: diagnostic.DiagnosticError,
		Code:     artifactadapter.DiagnosticCodeProjectionInvalid,
		Message:  diagnostic.BoundedDiagnosticMessage(err.Error()),
		Location: &diagnostic.DiagnosticLocation{
			Locator:            value.Binding.Locator,
			SubresourceLocator: value.Binding.SubresourceLocator,
		},
	}
}
