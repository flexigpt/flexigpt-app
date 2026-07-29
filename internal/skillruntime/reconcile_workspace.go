package skillruntime

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/flexigpt/agentskills-go/fsskillprovider"
	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/workspace/skilladapter"
)

func (s *SkillRuntime) ResyncWorkspace(
	ctx context.Context,
	workspace collection.CollectionRef,
) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	if s.workspaceSkills == nil {
		return errors.New("Workspace Skill adapter is not configured")
	}
	if err := workspace.Validate(); err != nil {
		return err
	}
	s.rtResyncMu.Lock()
	defer s.rtResyncMu.Unlock()

	values, err := s.workspaceSkills.List(ctx, workspace)
	if err != nil {
		//nolint:gocritic // Dont want switch.
		if errors.Is(err, basespec.ErrCatalogUnavailable) {
			values = nil
		} else if errors.Is(err, basespec.ErrCollectionNotFound) ||
			errors.Is(err, basespec.ErrRootNotFound) {
			workspaces := cloneWorkspaceDesiredViews(s.managedWorkspaces)
			delete(workspaces, workspace)
			return s.reconcilePartitionsLocked(
				ctx,
				cloneRuntimeDesiredView(s.managedInstalled),
				workspaces,
				runtimeApplyBestEffort,
			)
		} else {
			return s.failClosedWorkspaceLocked(
				ctx,
				workspace,
				err,
			)
		}
	}

	desired := newRuntimeDesiredView()
	artifactRefs := make([]artifact.ArtifactRef, 0, len(values))
	for _, value := range values {
		if !value.ProjectionValid ||
			!value.RuntimePathBacked ||
			!value.WorkspaceEnabled ||
			!value.Skill.IsEnabled ||
			!value.CatalogCurrent ||
			value.RuntimeDisabled ||
			value.State != artifact.StateAvailable {
			continue
		}
		artifactRefs = append(artifactRefs, value.Artifact)
	}

	if len(artifactRefs) != 0 {
		plan, err := s.workspaceSkills.Load(
			ctx,
			workspace,
			artifactRefs,
		)
		if err != nil {
			return s.failClosedWorkspaceLocked(
				ctx,
				workspace,
				err,
			)
		}

		for _, value := range plan.Skills {
			if value.Workspace != workspace {
				return fmt.Errorf(
					"%w: Workspace Skill load plan returned a Skill from another Workspace",
					basespec.ErrInvalid,
				)
			}

			definition, err := workspaceRuntimeDefinition(value)
			if err != nil {
				return s.failClosedWorkspaceLocked(
					ctx,
					workspace,
					err,
				)
			}
			desired.add(
				definition,
				workspaceRuntimeVersion(value),
			)
		}
	}

	workspaces := cloneWorkspaceDesiredViews(s.managedWorkspaces)
	workspaces[workspace] = desired
	return s.reconcilePartitionsLocked(
		ctx,
		cloneRuntimeDesiredView(s.managedInstalled),
		workspaces,
		runtimeApplyStrict,
	)
}

func (s *SkillRuntime) failClosedWorkspaceLocked(
	ctx context.Context,
	workspace collection.CollectionRef,
	cause error,
) error {
	workspaces := cloneWorkspaceDesiredViews(s.managedWorkspaces)
	workspaces[workspace] = newRuntimeDesiredView()

	cleanupContext, cancel := context.WithTimeout(
		context.WithoutCancel(ctx),
		runtimeForegroundValidateTimeout,
	)
	defer cancel()
	cleanupErr := s.reconcilePartitionsLocked(
		cleanupContext,
		cloneRuntimeDesiredView(s.managedInstalled),
		workspaces,
		runtimeApplyBestEffort,
	)
	if cleanupErr != nil {
		return errors.Join(
			cause,
			fmt.Errorf(
				"fail-closed Workspace runtime cleanup: %w",
				cleanupErr,
			),
		)
	}
	return cause
}

func (s *SkillRuntime) RemoveWorkspace(
	ctx context.Context,
	workspace collection.CollectionRef,
) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	if err := workspace.Validate(); err != nil {
		return err
	}
	s.rtResyncMu.Lock()
	defer s.rtResyncMu.Unlock()
	workspaces := cloneWorkspaceDesiredViews(s.managedWorkspaces)

	delete(workspaces, workspace)
	return s.reconcilePartitionsLocked(
		ctx,
		cloneRuntimeDesiredView(s.managedInstalled),
		workspaces,
		runtimeApplyStrict,
	)
}

func (s *SkillRuntime) workspaceDefinitionForArtifact(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (agentskillsSpec.SkillDef, collection.CollectionRef, bool) {
	if s.workspaceSkills == nil {
		return agentskillsSpec.SkillDef{}, collection.CollectionRef{}, false
	}
	value, err := s.workspaceSkills.LoadArtifact(ctx, ref)
	if err != nil {
		return agentskillsSpec.SkillDef{}, collection.CollectionRef{}, false
	}
	if err := s.ResyncWorkspace(ctx, value.Workspace); err != nil {
		return agentskillsSpec.SkillDef{}, collection.CollectionRef{}, false
	}
	value, err = s.workspaceSkills.LoadArtifact(ctx, ref)
	if err != nil {
		return agentskillsSpec.SkillDef{}, collection.CollectionRef{}, false
	}
	definition, err := workspaceRuntimeDefinition(value)
	if err != nil {
		return agentskillsSpec.SkillDef{}, collection.CollectionRef{}, false
	}
	return definition, value.Workspace, true
}

// workspaceRuntimeDefinition intentionally uses the ordinary filesystem
// provider. Workspace identity remains outside Agent Skills as
// ArtifactRef; the runtime definition is only an ephemeral local projection
// for a selected, approved filesystem package.
func workspaceRuntimeVersion(value skilladapter.WorkspaceSkill) string {
	input := string(value.DefinitionDigest) + "\x00" +
		string(value.SourceContentDigest) + "\x00" +
		value.SourceGeneration
	return "workspace:" + string(
		cryptoutil.DigestBytes([]byte(input)),
	)
}

func workspaceRuntimeDefinition(
	value skilladapter.WorkspaceSkill,
) (agentskillsSpec.SkillDef, error) {
	if !value.ProjectionValid ||
		!value.RuntimePathBacked ||
		!value.WorkspaceEnabled ||
		!value.CatalogCurrent ||
		!value.Skill.IsEnabled ||
		value.RuntimeDisabled ||
		value.State != artifact.StateAvailable {
		return agentskillsSpec.SkillDef{}, fmt.Errorf(
			"%w: Workspace Skill is not eligible for runtime registration",
			basespec.ErrCatalogStale,
		)
	}

	location := strings.TrimSpace(value.RuntimeLocation)
	if location == "" || !filepath.IsAbs(location) {
		return agentskillsSpec.SkillDef{}, fmt.Errorf(
			"%w: Workspace Skill runtime location is not an absolute filesystem path",
			basespec.ErrInvalid,
		)
	}
	return agentskillsSpec.SkillDef{
		Type:     fsskillprovider.Type,
		Name:     value.Skill.Name,
		Location: location,
	}, nil
}
