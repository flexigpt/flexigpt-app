package skill

import (
	"context"
	"fmt"

	"github.com/flexigpt/agentskills-go/document"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/skill/store/materialize"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/workspace/store/domain"
)

type WorkspaceSkill struct {
	Artifact         artifactModel.ArtifactRef `json:"-"`
	ArtifactRevision uint64                    `json:"-"`
	DefinitionDigest cryptoutil.Digest         `json:"-"`
	SourceID         string                    `json:"-"`
	Locator          spec.Locator              `json:"-"`

	Document        document.SkillDocument `json:"-"`
	RuntimeLocation string                 `json:"-"`
	Version         string                 `json:"-"`
}

type LoadPlan struct {
	Workspace artifactModel.ArtifactRef `json:"-"`
	Skills    []WorkspaceSkill          `json:"-"`
}

type Adapter struct {
	artifacts artifact.API
	resources resource.API
}

func New(
	artifacts artifact.API,
	resources resource.API,
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

// LoadSelected loads exactly the capability-authorized refs in order.
// Protected built-in ArtifactRefs may belong to another Root. An empty list
// means no Skills rather than every available Skill.
func (a *Adapter) LoadSelected(
	ctx context.Context,
	workspace workspaceDomain.Workspace,
	refs []artifactModel.ArtifactRef,
) (LoadPlan, error) {
	if a == nil || a.artifacts == nil || a.resources == nil {
		return LoadPlan{}, spec.ErrClosed
	}
	return a.loadSelected(ctx, workspace, refs)
}

func (a *Adapter) loadSelected(
	ctx context.Context,
	workspace workspaceDomain.Workspace,
	refs []artifactModel.ArtifactRef,
) (LoadPlan, error) {
	output := LoadPlan{
		Workspace: workspace.Ref(),
		Skills:    make([]WorkspaceSkill, 0, len(refs)),
	}
	for _, ref := range refs {
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
	record artifactModel.Artifact,
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
