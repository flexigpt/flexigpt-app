package skill

import (
	"context"
	"fmt"

	"github.com/flexigpt/agentskills-go/document"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/skill/store/materialize"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/workspace/store/domain"
)

type WorkspaceSkill struct {
	Artifact         artifact.ArtifactRef `json:"-"`
	ArtifactRevision uint64               `json:"-"`
	DefinitionDigest cryptoutil.Digest    `json:"-"`
	SourceID         string               `json:"-"`
	Locator          basespec.Locator     `json:"-"`

	Document        document.SkillDocument `json:"-"`
	RuntimeLocation string                 `json:"-"`
	Version         string                 `json:"-"`
}

type LoadPlan struct {
	Workspace artifact.ArtifactRef `json:"-"`
	Skills    []WorkspaceSkill     `json:"-"`
}

type Adapter struct {
	artifacts compositionapi.ArtifactAPI
	resources compositionapi.ResourceAPI
}

func New(
	artifacts compositionapi.ArtifactAPI,
	resources compositionapi.ResourceAPI,
) (*Adapter, error) {
	if artifacts == nil || resources == nil {
		return nil, fmt.Errorf(
			"%w: Workspace Skill adapter dependencies are incomplete",
			workspaceDomain.ErrInvalidWorkspace,
		)
	}
	return &Adapter{
		artifacts: artifacts,
		resources: resources,
	}, nil
}

// LoadSelected loads exactly refs in order. An empty list means no Skills,
// rather than every available Skill in the Workspace Root.
func (a *Adapter) LoadSelected(
	ctx context.Context,
	workspace workspaceDomain.Workspace,
	refs []artifact.ArtifactRef,
) (LoadPlan, error) {
	if a == nil || a.artifacts == nil || a.resources == nil {
		return LoadPlan{}, basespec.ErrClosed
	}
	if err := workspace.Validate(); err != nil {
		return LoadPlan{}, err
	}
	return a.loadSelected(ctx, workspace, refs)
}

func (a *Adapter) loadSelected(
	ctx context.Context,
	workspace workspaceDomain.Workspace,
	refs []artifact.ArtifactRef,
) (LoadPlan, error) {
	seen := make(map[artifact.ArtifactID]struct{}, len(refs))
	output := LoadPlan{
		Workspace: workspace.Ref(),
		Skills:    make([]WorkspaceSkill, 0, len(refs)),
	}
	for _, ref := range refs {
		if err := ref.Validate(); err != nil {
			return LoadPlan{}, err
		}
		if ref.RootID != workspace.Artifact.RootID {
			return LoadPlan{}, fmt.Errorf(
				"%w: selected Skill belongs to another Root",
				workspaceDomain.ErrReferenceUnresolved,
			)
		}
		if _, duplicate := seen[ref.ArtifactID]; duplicate {
			return LoadPlan{}, fmt.Errorf(
				"%w: duplicate selected Workspace Skill",
				workspaceDomain.ErrInvalidWorkspace,
			)
		}
		seen[ref.ArtifactID] = struct{}{}

		record, err := a.artifacts.Get(ctx, ref)
		if err != nil {
			return LoadPlan{}, err
		}
		if !record.Enabled {
			continue
		}
		value, err := a.resolve(ctx, workspace, record)
		if err != nil {
			return LoadPlan{}, err
		}
		output.Skills = append(output.Skills, value)
	}

	return output, nil
}

func (a *Adapter) resolve(
	ctx context.Context,
	_ workspaceDomain.Workspace,
	record artifact.Artifact,
) (WorkspaceSkill, error) {
	value, err := materialize.Resolve(ctx, a.resources, record)
	if err != nil {
		return WorkspaceSkill{}, err
	}

	return WorkspaceSkill{
		Artifact:         value.Artifact,
		ArtifactRevision: value.ArtifactRevision,
		DefinitionDigest: value.DefinitionDigest,
		SourceID:         string(value.SourceID),
		Locator:          value.Locator,
		Document:         value.Document,
		RuntimeLocation:  value.RuntimeLocation,
		Version: "workspace-skill:" + string(
			value.VersionDigest,
		),
	}, nil
}
