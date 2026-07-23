package skillruntime

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/flexigpt/agentskills-go/fsskillprovider"
	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/record"
	"github.com/flexigpt/flexigpt-app/internal/workspace/skilladapter"
)

func (s *SkillRuntime) ResyncWorkspace(
	ctx context.Context,
	rootID artifactstore.RootID,
) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	if s.workspaceSkills == nil {
		return errors.New("Workspace Skill adapter is not configured")
	}
	if err := artifactstore.ValidateRootID(rootID); err != nil {
		return err
	}

	values, err := s.workspaceSkills.List(ctx, rootID)
	if err != nil {
		if errors.Is(err, artifactstore.ErrCatalogUnavailable) {
			values = nil
		} else {
			return err
		}
	}

	desired := newRuntimeDesiredView()
	recordIDs := make([]artifactstore.RecordID, 0, len(values))
	for _, value := range values {
		if !value.Skill.IsEnabled ||
			!value.CatalogCurrent ||
			!value.RuntimeAllowed ||
			value.State != record.StateAvailable {
			continue
		}
		recordIDs = append(recordIDs, value.RecordID)
	}
	if len(recordIDs) != 0 {
		plan, err := s.workspaceSkills.Load(ctx, rootID, recordIDs)
		if err != nil {
			return err
		}
		for _, value := range plan.Skills {
			definition, err := workspaceRuntimeDefinition(value)
			if err != nil {
				return err
			}
			desired.add(
				definition,
				"workspace:"+string(value.DefinitionDigest),
			)
		}
	}

	s.rtResyncMu.Lock()
	defer s.rtResyncMu.Unlock()
	workspaces := cloneWorkspaceDesiredViews(s.managedWorkspaces)
	workspaces[rootID] = desired
	return s.reconcilePartitionsLocked(
		ctx,
		cloneRuntimeDesiredView(s.managedInstalled),
		workspaces,
		runtimeApplyStrict,
	)
}

func (s *SkillRuntime) RemoveWorkspace(
	ctx context.Context,
	rootID artifactstore.RootID,
) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	if err := artifactstore.ValidateRootID(rootID); err != nil {
		return err
	}
	s.rtResyncMu.Lock()
	defer s.rtResyncMu.Unlock()
	workspaces := cloneWorkspaceDesiredViews(s.managedWorkspaces)
	delete(workspaces, rootID)
	return s.reconcilePartitionsLocked(
		ctx,
		cloneRuntimeDesiredView(s.managedInstalled),
		workspaces,
		runtimeApplyStrict,
	)
}

func (s *SkillRuntime) workspaceDefinitionForIdentity(
	ctx context.Context,
	identity string,
) (agentskillsSpec.SkillDef, bool) {
	rootID, recordID, err := parseWorkspaceIdentity(identity)
	if err != nil {
		return agentskillsSpec.SkillDef{}, false
	}
	if err := s.ResyncWorkspace(ctx, rootID); err != nil {
		return agentskillsSpec.SkillDef{}, false
	}
	if s.workspaceSkills == nil {
		return agentskillsSpec.SkillDef{}, false
	}
	plan, err := s.workspaceSkills.Load(
		ctx,
		rootID,
		[]artifactstore.RecordID{recordID},
	)
	if err != nil || len(plan.Skills) != 1 {
		return agentskillsSpec.SkillDef{}, false
	}
	definition, err := workspaceRuntimeDefinition(plan.Skills[0])
	if err != nil {
		return agentskillsSpec.SkillDef{}, false
	}
	return definition, true
}

// workspaceRuntimeDefinition intentionally uses the ordinary filesystem
// provider. Workspace identity remains outside Agent Skills as
// workspace/<rootID>/<recordID>; the runtime definition is only an ephemeral
// local projection for a selected, approved filesystem package.
func workspaceRuntimeDefinition(
	value skilladapter.WorkspaceSkill,
) (agentskillsSpec.SkillDef, error) {
	location := strings.TrimSpace(value.RuntimeLocation)
	if location == "" || !filepath.IsAbs(location) {
		return agentskillsSpec.SkillDef{}, fmt.Errorf(
			"%w: Workspace Skill runtime location is not an absolute filesystem path",
			artifactstore.ErrInvalid,
		)
	}
	return agentskillsSpec.SkillDef{
		Type:     fsskillprovider.Type,
		Name:     value.Skill.Name,
		Location: location,
	}, nil
}
