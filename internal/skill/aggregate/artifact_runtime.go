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

	lifecycleMu    sync.RWMutex
	closed         bool
	catalogSyncMu  sync.Mutex
	catalogSyncing map[root.RootID]*rootCatalogSync
}

type rootCatalogSync struct {
	done chan struct{}
	err  error
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
		resolver:       resolver,
		runtime:        runtimeService,
		catalogSyncing: make(map[root.RootID]*rootCatalogSync),
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
	if err := ref.Validate(); err != nil {
		return ResolvedArtifactSkill{}, err
	}
	values, err := s.ResolveArtifactSkills(
		ctx,
		[]artifact.ArtifactRef{ref},
	)
	if err != nil {
		return ResolvedArtifactSkill{}, err
	}
	if len(values) != 1 {
		return ResolvedArtifactSkill{}, fmt.Errorf(
			"%w: expected one resolved Artifact Skill",
			basespec.ErrReferenceUnresolved,
		)
	}
	return values[0], nil
}

// ResolveArtifactSkills synchronizes each owning Root at most once for this
// request, then maps all requested durable Artifact references to runtime
// definitions. Callers that need several Skills must use this instead of
// repeatedly calling ResolveArtifactSkill.
func (s *Service) ResolveArtifactSkills(
	ctx context.Context,
	refs []artifact.ArtifactRef,
) ([]ResolvedArtifactSkill, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, err
	}

	resolved, err := s.resolveArtifactSkills(ctx, refs)
	if err != nil {
		return nil, err
	}
	return append([]ResolvedArtifactSkill(nil), resolved.Values...), nil
}

// SyncRootCatalog warms or reconciles one Root's runtime catalog. It is
// useful during startup after protected topology hydration has completed.
func (s *Service) SyncRootCatalog(
	ctx context.Context,
	rootID root.RootID,
) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	if err := rootID.Validate(); err != nil {
		return err
	}
	return s.resyncRoot(ctx, rootID)
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
			IsEnabled:    resolved.Enabled,
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
	return s.syncRootCatalog(ctx, rootID, catalogID)
}

// syncRootCatalog prevents a startup warmup and a foreground page request
// from independently materializing the same Root at the same time.
func (s *Service) syncRootCatalog(
	ctx context.Context,
	rootID root.RootID,
	catalogID skillRuntime.CatalogID,
) error {
	s.catalogSyncMu.Lock()
	if current, found := s.catalogSyncing[rootID]; found {
		s.catalogSyncMu.Unlock()
		select {
		case <-current.done:
			return current.err
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	current := &rootCatalogSync{
		done: make(chan struct{}),
	}
	s.catalogSyncing[rootID] = current
	s.catalogSyncMu.Unlock()

	err := s.runtime.SyncCatalog(ctx, catalogID)

	s.catalogSyncMu.Lock()
	current.err = err
	delete(s.catalogSyncing, rootID)
	close(current.done)
	s.catalogSyncMu.Unlock()

	return err
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
	Values         []ResolvedArtifactSkill
}

type unavailableArtifactSkill struct {
	ref   artifact.ArtifactRef
	cause error
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
	unavailable := make([]unavailableArtifactSkill, 0)
	readyRefs := make([]artifact.ArtifactRef, 0, len(refs))

	recordUnavailable := func(
		ref artifact.ArtifactRef,
		cause error,
	) {
		unavailable = append(unavailable, unavailableArtifactSkill{
			ref:   ref,
			cause: cause,
		})
	}

	for _, ref := range refs {
		rootID, err := s.resolver.RootForArtifact(ctx, ref)
		if err != nil {
			recordUnavailable(
				ref,
				fmt.Errorf("resolve Artifact Root: %w", err),
			)
			continue
		}
		if previous, found := resynced[rootID]; found {
			if previous != nil {
				recordUnavailable(
					ref,
					fmt.Errorf(
						"synchronize Root catalog %q: %w",
						rootID,
						previous,
					),
				)
				continue
			}
		} else {
			err := s.resyncRoot(ctx, rootID)
			resynced[rootID] = err
			if err != nil {
				recordUnavailable(
					ref,
					fmt.Errorf(
						"synchronize Root catalog %q: %w",
						rootID,
						err,
					),
				)
				continue
			}
		}

		readyRefs = append(readyRefs, ref)
	}

	values, err := s.resolver.ResolveArtifactSkills(ctx, readyRefs)
	if err != nil {
		return resolvedArtifactSkills{}, err
	}
	if len(values) != len(readyRefs) {
		return resolvedArtifactSkills{}, fmt.Errorf(
			"%w: Artifact Skill router returned an unexpected result count",
			basespec.ErrInvalid,
		)
	}

	for index, value := range values {
		ref := readyRefs[index]
		if value.Artifact != ref {
			return resolvedArtifactSkills{}, fmt.Errorf(
				"%w: Artifact Skill router resolved another Artifact",
				basespec.ErrRefreshRequired,
			)
		}
		if !value.Enabled {
			recordUnavailable(
				ref,
				errors.New("artifact skill is disabled"),
			)
			continue
		}

		registration := skillRuntime.SkillRegistration{
			Definition: value.Definition,
			Revision:   value.Version,
		}
		if !s.runtime.IsRegistered(registration) {
			recordUnavailable(
				ref,
				fmt.Errorf(
					"runtime catalog did not register Skill revision %q",
					value.Version,
				),
			)
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
		output.Values = append(output.Values, value)
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
	unavailable []unavailableArtifactSkill,
) error {
	sort.Slice(unavailable, func(left, right int) bool {
		return artifactRefKey(unavailable[left].ref) <
			artifactRefKey(unavailable[right].ref)
	})

	refs := make([]string, 0, len(unavailable))
	causes := make([]error, 0, len(unavailable))
	for _, value := range unavailable {
		key := artifactRefKey(value.ref)
		refs = append(refs, key)
		if value.cause == nil {
			continue
		}
		causes = append(
			causes,
			fmt.Errorf("%s: %w", key, value.cause),
		)
	}

	summary := fmt.Errorf(
		"%w: unavailable Artifact Skills: %s",
		basespec.ErrReferenceUnresolved,
		strings.Join(refs, ", "),
	)
	if len(causes) == 0 {
		return summary
	}
	return errors.Join(
		append([]error{summary}, causes...)...,
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
