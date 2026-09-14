package aggregate

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/flexigpt/agentskills-go/document"
	"github.com/flexigpt/agentskills-go/provider"
	agentskillsRuntime "github.com/flexigpt/agentskills-go/runtime"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	skillRuntime "github.com/flexigpt/flexigpt-app/internal/skill/runtime"
)

type Service struct {
	resolver *ArtifactRouter
	runtime  *skillRuntime.Service

	lifecycleMu sync.RWMutex
	closed      bool
}

func New(
	resolver *ArtifactRouter,
	runtimeService *skillRuntime.Service,
) (*Service, error) {
	if resolver == nil {
		return nil, errors.New("artifact Skill router is required")
	}
	if runtimeService == nil {
		return nil, errors.New("skill runtime service is required")
	}
	return &Service{
		resolver: resolver,
		runtime:  runtimeService,
	}, nil
}

func (s *Service) RunScriptsEnabled() bool {
	if s == nil || s.isClosed() {
		return false
	}
	return s.runtime.SupportsRunScript()
}

func (s *Service) Close() {
	if s == nil {
		return
	}
	s.lifecycleMu.Lock()
	s.closed = true
	s.lifecycleMu.Unlock()
}

func (s *Service) ResolveArtifactSkill(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ResolvedArtifactSkill, error) {
	if err := s.ensureConfigured(); err != nil {
		return ResolvedArtifactSkill{}, err
	}
	if err := ref.Validate(); err != nil {
		return ResolvedArtifactSkill{}, err
	}
	rootID, err := s.resolver.RootForArtifact(ctx, ref)
	if err != nil {
		return ResolvedArtifactSkill{}, err
	}
	if err := s.resyncRoot(ctx, rootID); err != nil {
		return ResolvedArtifactSkill{}, err
	}
	value, err := s.resolver.ResolveArtifactSkill(ctx, ref)
	if err != nil {
		return ResolvedArtifactSkill{}, err
	}
	if !s.runtime.IsRegistered(skillRuntime.SkillRegistration{
		Definition: value.Definition,
		Revision:   value.Version,
	}) {
		return ResolvedArtifactSkill{}, fmt.Errorf(
			"%w: runtime did not register Artifact Skill %q",
			basespec.ErrReferenceUnresolved,
			ref.ArtifactID,
		)
	}
	return value, nil
}

func (s *Service) GetArtifactSkillsPrompt(
	ctx context.Context,
	filter ArtifactSkillFilter,
) (string, error) {
	if err := s.ensureConfigured(); err != nil {
		return "", err
	}
	resolved, err := s.resolveArtifactSkills(
		ctx,
		filter.AllowArtifacts,
	)
	if err != nil {
		return "", err
	}
	if len(filter.Inserts) != 0 &&
		!containsInstructionInsert(filter.Inserts) {
		return "", nil
	}
	return s.runtime.SkillsPrompt(ctx, &agentskillsRuntime.SkillFilter{
		Types:          append([]string(nil), filter.Types...),
		NamePrefix:     filter.NamePrefix,
		LocationPrefix: filter.LocationPrefix,
		AllowSkills:    resolved.AllowDefs,
		SessionID:      filter.SessionID,
		Activity:       filter.Activity,
	})
}

func (s *Service) ListArtifactSkillRefs(
	ctx context.Context,
	filter ArtifactSkillFilter,
) ([]artifact.ArtifactRef, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, err
	}
	if len(filter.AllowArtifacts) == 0 {
		return nil, ErrArtifactSkillSelectionRequired
	}
	resolved, err := s.resolveArtifactSkills(
		ctx,
		filter.AllowArtifacts,
	)
	if err != nil {
		return nil, err
	}

	records, err := s.runtime.ListAgentSkills(
		ctx,
		&agentskillsRuntime.SkillListFilter{
			Types:          append([]string(nil), filter.Types...),
			NamePrefix:     filter.NamePrefix,
			LocationPrefix: filter.LocationPrefix,
			AllowSkills:    resolved.AllowDefs,
			Inserts:        append([]document.SkillInsert(nil), filter.Inserts...),
			SessionID:      filter.SessionID,
			Activity:       filter.Activity,
		},
	)
	if err != nil {
		return nil, err
	}

	output := make([]artifact.ArtifactRef, 0, len(records))
	for _, record := range records {
		if ref, found := resolved.DefToArtifacts[record.Def]; found {
			output = append(output, ref)
		}
	}
	sort.Slice(output, func(left, right int) bool {
		return artifactRefKey(output[left]) <
			artifactRefKey(output[right])
	})
	return output, nil
}

func (s *Service) DescribeArtifactSkill(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ArtifactSkillSummary, error) {
	if err := s.ensureConfigured(); err != nil {
		return ArtifactSkillSummary{}, err
	}
	if err := ref.Validate(); err != nil {
		return ArtifactSkillSummary{}, err
	}

	resolved, err := s.ResolveArtifactSkill(ctx, ref)
	if err != nil {
		return ArtifactSkillSummary{}, err
	}
	records, err := s.runtime.ListAgentSkills(
		ctx,
		&agentskillsRuntime.SkillListFilter{
			AllowSkills: []provider.SkillDef{resolved.Definition},
		},
	)
	if err != nil {
		return ArtifactSkillSummary{}, err
	}
	for _, record := range records {
		if record.Def != resolved.Definition {
			continue
		}
		return ArtifactSkillSummary{
			Artifact:     ref,
			IsEnabled:    true,
			Insert:       record.Insert,
			HasArguments: len(record.Arguments) != 0,
			HasResources: record.Resources.HasResources,
		}, nil
	}
	return ArtifactSkillSummary{}, fmt.Errorf(
		"%w: runtime did not index Artifact Skill %q",
		basespec.ErrReferenceUnresolved,
		ref.ArtifactID,
	)
}

func (s *Service) resyncRoot(
	ctx context.Context,
	rootID root.RootID,
) error {
	catalogID, err := RootCatalogID(rootID)
	if err != nil {
		return err
	}
	return s.runtime.SyncCatalog(ctx, catalogID)
}

func (s *Service) ensureConfigured() error {
	if s == nil {
		return errors.New("artifact Skill runtime bridge is not configured")
	}
	s.lifecycleMu.RLock()
	closed := s.closed
	configured := s.resolver != nil && s.runtime != nil
	s.lifecycleMu.RUnlock()
	if closed || !configured {
		return errors.New("artifact Skill runtime bridge is not configured")
	}
	return nil
}

func (s *Service) isClosed() bool {
	s.lifecycleMu.RLock()
	defer s.lifecycleMu.RUnlock()
	return s.closed
}

type resolvedArtifactSkills struct {
	DefToArtifacts map[provider.SkillDef]artifact.ArtifactRef
	AllowDefs      []provider.SkillDef
}

func (s *Service) resolveArtifactSkills(
	ctx context.Context,
	refs []artifact.ArtifactRef,
) (resolvedArtifactSkills, error) {
	if err := validateArtifactRefs(refs); err != nil {
		return resolvedArtifactSkills{}, err
	}
	output := resolvedArtifactSkills{
		DefToArtifacts: make(
			map[provider.SkillDef]artifact.ArtifactRef,
		),
	}
	resynced := map[root.RootID]error{}
	unavailable := make([]artifact.ArtifactRef, 0)

	for _, ref := range refs {
		rootID, err := s.resolver.RootForArtifact(ctx, ref)
		if err != nil {
			unavailable = append(unavailable, ref)
			continue
		}
		if previous, found := resynced[rootID]; found {
			if previous != nil {
				unavailable = append(unavailable, ref)
				continue
			}
		} else {
			err := s.resyncRoot(ctx, rootID)
			resynced[rootID] = err
			if err != nil {
				unavailable = append(unavailable, ref)
				continue
			}
		}

		value, err := s.resolver.ResolveArtifactSkill(ctx, ref)
		if err != nil ||
			!s.runtime.IsRegistered(skillRuntime.SkillRegistration{
				Definition: value.Definition,
				Revision:   value.Version,
			}) {
			unavailable = append(unavailable, ref)
			continue
		}
		if previous, exists := output.DefToArtifacts[value.Definition]; exists &&
			previous != value.Artifact {
			return resolvedArtifactSkills{}, fmt.Errorf(
				"%w: Artifacts %q and %q resolve to one runtime Skill",
				basespec.ErrConflict,
				previous.ArtifactID,
				value.Artifact.ArtifactID,
			)
		}
		output.DefToArtifacts[value.Definition] = value.Artifact
		output.AllowDefs = append(output.AllowDefs, value.Definition)
	}
	if len(unavailable) != 0 {
		return resolvedArtifactSkills{}, unavailableArtifactSkillsError(
			unavailable,
		)
	}
	sortSkillDefs(output.AllowDefs)
	return output, nil
}

func unavailableArtifactSkillsError(
	refs []artifact.ArtifactRef,
) error {
	sort.Slice(refs, func(left, right int) bool {
		return artifactRefKey(refs[left]) <
			artifactRefKey(refs[right])
	})
	values := make([]string, 0, len(refs))
	for _, ref := range refs {
		values = append(values, artifactRefKey(ref))
	}
	return fmt.Errorf(
		"%w: unavailable Artifact Skills: %s",
		basespec.ErrReferenceUnresolved,
		strings.Join(values, ", "),
	)
}

func validateArtifactRefs(values []artifact.ArtifactRef) error {
	seen := map[string]struct{}{}
	for _, value := range values {
		if err := value.Validate(); err != nil {
			return err
		}
		key := artifactRefKey(value)
		if _, duplicate := seen[key]; duplicate {
			return errors.New("duplicate ArtifactRef")
		}
		seen[key] = struct{}{}
	}
	return nil
}

func containsInstructionInsert(values []document.SkillInsert) bool {
	for _, value := range values {
		insert, supported := document.NormalizeSkillInsert(value)
		if supported && insert == document.SkillInsertInstructions {
			return true
		}
	}
	return false
}

func sortSkillDefs(values []provider.SkillDef) {
	sort.Slice(values, func(left, right int) bool {
		if values[left].Type != values[right].Type {
			return values[left].Type < values[right].Type
		}
		if values[left].Name != values[right].Name {
			return values[left].Name < values[right].Name
		}
		return values[left].Location < values[right].Location
	})
}

func artifactRefKey(ref artifact.ArtifactRef) string {
	return string(ref.RootID) + "\x00" + string(ref.ArtifactID)
}
