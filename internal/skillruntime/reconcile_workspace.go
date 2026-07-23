package skillruntime

import (
	"context"
	"errors"
	"strings"

	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
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
		return err
	}
	recordIDs := make([]artifactstore.RecordID, 0, len(values))
	for _, value := range values {
		if !value.Skill.IsEnabled ||
			!value.CatalogCurrent ||
			!value.RuntimeAllowed {
			continue
		}
		recordIDs = append(recordIDs, value.RecordID)
	}

	desired := runtimeDesiredView{
		set:        map[agentskillsSpec.SkillDef]struct{}{},
		byTypeName: map[string][]agentskillsSpec.SkillDef{},
	}
	if len(recordIDs) > 0 {
		plan, err := s.workspaceSkills.Load(ctx, rootID, recordIDs)
		if err != nil {
			return err
		}
		for _, value := range plan.Skills {
			definition := workspaceRuntimeDefinition(value)
			desired.set[definition] = struct{}{}
			key := typeNameKey(definition.Type, definition.Name)
			desired.byTypeName[key] = append(
				desired.byTypeName[key],
				definition,
			)
		}
	}

	prefix := workspaceIdentityPrefix + string(rootID) + "/"
	s.rtResyncMu.Lock()
	defer s.rtResyncMu.Unlock()
	return s.runtimeApplyDesired(
		ctx,
		desired.set,
		desired.byTypeName,
		func(definition agentskillsSpec.SkillDef) bool {
			return definition.Type == workspaceSkillProviderType &&
				strings.HasPrefix(definition.Location, prefix)
		},
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
	prefix := workspaceIdentityPrefix + string(rootID) + "/"
	s.rtResyncMu.Lock()
	defer s.rtResyncMu.Unlock()
	return s.runtimeApplyDesired(
		ctx,
		map[agentskillsSpec.SkillDef]struct{}{},
		map[string][]agentskillsSpec.SkillDef{},
		func(definition agentskillsSpec.SkillDef) bool {
			return definition.Type == workspaceSkillProviderType &&
				strings.HasPrefix(definition.Location, prefix)
		},
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
	plan, err := s.workspaceSkills.Load(
		ctx,
		rootID,
		[]artifactstore.RecordID{recordID},
	)
	if err != nil || len(plan.Skills) != 1 {
		return agentskillsSpec.SkillDef{}, false
	}
	return workspaceRuntimeDefinition(plan.Skills[0]), true
}
