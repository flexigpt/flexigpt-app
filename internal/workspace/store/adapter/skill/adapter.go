package skill

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/flexigpt/agentskills-go/document"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/resource"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	skillDomain "github.com/flexigpt/flexigpt-app/internal/skill/store/domain"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/workspace/store/domain"
)

type WorkspaceSkill struct {
	Artifact         artifact.ArtifactRef
	ArtifactRevision uint64
	DefinitionDigest cryptoutil.Digest
	SourceID         string
	Locator          basespec.Locator

	Document        document.SkillDocument
	RuntimeLocation string
	Version         string
	RuntimeDisabled bool
}

type LoadPlan struct {
	Workspace workspaceDomain.WorkspaceRef
	Skills    []WorkspaceSkill
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

func (a *Adapter) List(
	ctx context.Context,
	workspace workspaceDomain.Workspace,
) ([]WorkspaceSkill, error) {
	if err := workspace.Validate(); err != nil {
		return nil, err
	}
	records, err := a.artifacts.ListByRoot(
		ctx,
		workspace.Artifact.RootID,
	)
	if err != nil {
		return nil, err
	}

	output := make([]WorkspaceSkill, 0)
	for _, record := range records {
		if !skillDomain.IsSkillKind(record.Kind) ||
			!record.Enabled ||
			record.State != artifact.StateAvailable {
			continue
		}
		value, err := a.resolve(ctx, workspace, record)
		if err != nil {
			return nil, err
		}
		output = append(output, value)
	}
	sort.Slice(output, func(left, right int) bool {
		if output[left].Document.Name != output[right].Document.Name {
			return output[left].Document.Name <
				output[right].Document.Name
		}
		return output[left].Artifact.ArtifactID <
			output[right].Artifact.ArtifactID
	})
	return output, nil
}

func (a *Adapter) Load(
	ctx context.Context,
	workspace workspaceDomain.Workspace,
	refs []artifact.ArtifactRef,
) (LoadPlan, error) {
	if err := workspace.Validate(); err != nil {
		return LoadPlan{}, err
	}
	if len(refs) == 0 {
		values, err := a.List(ctx, workspace)
		if err != nil {
			return LoadPlan{}, err
		}
		return LoadPlan{
			Workspace: workspace.Ref(),
			Skills:    values,
		}, nil
	}

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
	workspace workspaceDomain.Workspace,
	record artifact.Artifact,
) (WorkspaceSkill, error) {
	if record.RootID != workspace.Artifact.RootID ||
		!skillDomain.IsSkillKind(record.Kind) ||
		!record.Enabled ||
		record.State != artifact.StateAvailable ||
		record.ResolvedDefinition == nil ||
		record.SourceContentDigest == nil {
		return WorkspaceSkill{}, fmt.Errorf(
			"%w: Skill Artifact %q is unavailable",
			workspaceDomain.ErrReferenceUnresolved,
			record.ID,
		)
	}

	settings, err := workspaceDomain.DecodeArtifactData(record.Data)
	if err != nil {
		return WorkspaceSkill{}, err
	}
	resolved, err := a.resources.ResolveArtifact(
		ctx,
		record.Ref(),
		resource.ResolveOptions{
			VerifySourceContent: true,
		},
	)
	if err != nil {
		return WorkspaceSkill{}, err
	}
	if err := skillDomain.ValidateDefinition(resolved.Definition); err != nil {
		return WorkspaceSkill{}, err
	}

	sourceEntry, err := a.resources.ReadSourceEntry(
		ctx,
		record.RootID,
		record.Binding.SourceID,
		record.Binding.Locator,
		basespec.MaxCandidateBytes,
	)
	if err != nil {
		return WorkspaceSkill{}, err
	}
	if sourceEntry.Digest != *record.SourceContentDigest ||
		sourceEntry.SourceGeneration !=
			resolved.RefreshState.SourceGeneration {
		return WorkspaceSkill{}, fmt.Errorf(
			"%w: Skill Source changed during Workspace resolution",
			basespec.ErrRefreshRequired,
		)
	}
	documentValue, _, err := skillDomain.ParseSkillDocument(
		sourceEntry.Content,
		string(record.LogicalName),
	)
	if err != nil {
		return WorkspaceSkill{}, err
	}

	packageLocator, err := skillDomain.RuntimePackageLocator(
		record.Binding.Locator,
		record.Binding.SubresourceLocator,
	)
	if err != nil {
		return WorkspaceSkill{}, err
	}
	location, err := a.resources.ResolveVerifiedLocalPath(
		ctx,
		resolved,
		packageLocator,
	)
	if err != nil {
		return WorkspaceSkill{}, err
	}

	versionInput := string(resolved.Definition.Digest) + "\x00" +
		string(*record.SourceContentDigest) + "\x00" +
		resolved.RefreshState.SourceGeneration + "\x00" +
		strconv.FormatUint(record.Revision, 10)

	return WorkspaceSkill{
		Artifact:         record.Ref(),
		ArtifactRevision: record.Revision,
		DefinitionDigest: resolved.Definition.Digest,
		SourceID:         string(record.Binding.SourceID),
		Locator:          record.Binding.Locator,
		Document:         documentValue,
		RuntimeLocation:  location,
		Version: "workspace-skill:" + string(
			cryptoutil.DigestBytes([]byte(versionInput)),
		),
		RuntimeDisabled: settings.RuntimeDisabled,
	}, nil
}
