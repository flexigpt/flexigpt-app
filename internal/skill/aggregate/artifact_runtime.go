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

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	skillRuntime "github.com/flexigpt/flexigpt-app/internal/skill/runtime"
	skillStore "github.com/flexigpt/flexigpt-app/internal/skill/store"
)

// Service is an internal Artifact Store to Agent Skills bridge.
//
// Wails runtime APIs belong to SkillRuntimeWrapper. This service exists only
// where backend callers hold ArtifactRef identities and need them translated
// into runtime-owned SkillDef identities.
type Service struct {
	resolver *skillStore.ArtifactRouter
	runtime  *skillRuntime.Service

	lifecycleMu sync.RWMutex
	closed      bool
}

func New(
	resolver *skillStore.ArtifactRouter,
	runtimeService *skillRuntime.Service,
) (*Service, error) {
	if resolver == nil {
		return nil, errors.New("artifact skill router is required")
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

// GetArtifactSkillsPrompt resolves Artifact Store allow-list entries before
// delegating prompt generation to the Agent Skills runtime.
//
// An empty AllowArtifacts list intentionally preserves native runtime
// semantics: all currently registered Skills are eligible.
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

// ListArtifactSkillRefs returns Artifact identities corresponding to the
// selected runtime view. An explicit Artifact allow-list is required because
// the Agent Skills runtime deliberately does not know Artifact identities.
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
		ref, found := resolved.DefToArtifacts[record.Def]
		if found {
			output = append(output, ref)
		}
	}
	sort.Slice(output, func(left, right int) bool {
		return artifactRefKey(output[left]) < artifactRefKey(output[right])
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

	resolved, found := s.resolveArtifactSkill(ctx, ref)
	if !found {
		if _, err := s.resolver.ResolveArtifactSkill(ctx, ref); err != nil {
			return ArtifactSkillSummary{}, fmt.Errorf(
				"artifact skill %q is unavailable: %w",
				ref.ArtifactID,
				err,
			)
		}
		return ArtifactSkillSummary{}, fmt.Errorf(
			"%w: artifact skill %q could not be registered",
			basespec.ErrReferenceUnresolved,
			ref.ArtifactID,
		)
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
		"%w: runtime did not index artifact skill %q",
		basespec.ErrReferenceUnresolved,
		ref.ArtifactID,
	)
}

func (s *Service) ensureConfigured() error {
	if s == nil {
		return errors.New("artifact skill runtime bridge is not configured")
	}

	s.lifecycleMu.RLock()
	closed := s.closed
	configured := s.resolver != nil && s.runtime != nil
	s.lifecycleMu.RUnlock()

	if closed || !configured {
		return errors.New("artifact skill runtime bridge is not configured")
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
	RefToDef       map[string]provider.SkillDef
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
		DefToArtifacts: map[provider.SkillDef]artifact.ArtifactRef{},
		RefToDef:       map[string]provider.SkillDef{},
	}
	resynced := map[collection.CollectionRef]error{}
	unavailable := make([]artifact.ArtifactRef, 0)

	for _, ref := range refs {
		value, found := s.resolveArtifactSkillWithResync(
			ctx,
			ref,
			resynced,
		)
		if !found {
			unavailable = append(unavailable, ref)
			continue
		}

		if previous, exists := output.DefToArtifacts[value.Definition]; exists && previous != value.Artifact {
			return resolvedArtifactSkills{}, fmt.Errorf(
				"%w: artifact skills %q and %q resolve to the same runtime definition",
				basespec.ErrConflict,
				previous.ArtifactID,
				value.Artifact.ArtifactID,
			)
		}

		output.DefToArtifacts[value.Definition] = value.Artifact
		output.RefToDef[artifactRefKey(value.Artifact)] = value.Definition
		output.AllowDefs = append(output.AllowDefs, value.Definition)
	}

	if len(unavailable) != 0 {
		return resolvedArtifactSkills{}, unavailableArtifactSkillsError(unavailable)
	}

	sortSkillDefs(output.AllowDefs)
	return output, nil
}

func (s *Service) resolveArtifactSkill(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (skillStore.ResolvedArtifactSkill, bool) {
	return s.resolveArtifactSkillWithResync(ctx, ref, nil)
}

func (s *Service) resolveArtifactSkillWithResync(
	ctx context.Context,
	ref artifact.ArtifactRef,
	resynced map[collection.CollectionRef]error,
) (skillStore.ResolvedArtifactSkill, bool) {
	if resynced == nil {
		resynced = map[collection.CollectionRef]error{}
	}

	collectionRef, err := s.resolver.CollectionForArtifact(ctx, ref)
	if err != nil {
		return skillStore.ResolvedArtifactSkill{}, false
	}

	if previous, found := resynced[collectionRef]; found {
		if previous != nil {
			return skillStore.ResolvedArtifactSkill{}, false
		}
	} else {
		err := s.resyncCollection(ctx, collectionRef)
		resynced[collectionRef] = err
		if err != nil {
			return skillStore.ResolvedArtifactSkill{}, false
		}
	}

	value, err := s.resolver.ResolveArtifactSkill(ctx, ref)
	if err != nil || value.Collection != collectionRef {
		return skillStore.ResolvedArtifactSkill{}, false
	}
	if !s.runtime.IsRegistered(skillRuntime.SkillRegistration{
		Definition: value.Definition,
		Revision:   value.Version,
	}) {
		return skillStore.ResolvedArtifactSkill{}, false
	}
	return value, true
}

func unavailableArtifactSkillsError(
	refs []artifact.ArtifactRef,
) error {
	sort.Slice(refs, func(left, right int) bool {
		return artifactRefKey(refs[left]) < artifactRefKey(refs[right])
	})

	values := make([]string, 0, len(refs))
	for _, ref := range refs {
		values = append(values, artifactRefKey(ref))
	}
	return fmt.Errorf(
		"%w: unavailable artifact skills: %s",
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

func containsInstructionInsert(
	values []document.SkillInsert,
) bool {
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
