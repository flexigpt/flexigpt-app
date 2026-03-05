package store

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/skill/spec"
)

const (
	runtimeRollbackAttempts = 3
	runtimeRollbackBackoff  = 150 * time.Millisecond
)

type userWriteSagaOutcome struct {
	// ResyncReason triggers a best-effort resync after a successful store commit.
	ResyncReason string

	// RollbackReason/RollbackErr trigger a strict rollback *when returning an error*,
	// for cases where runtime may have been mutated before the error.
	// If RollbackErr is nil, the returned error is used.
	RollbackReason string
	RollbackErr    error
}

// runtimeApplyUserBundleEnabledDelta applies the strict foreground runtime delta for a bundle enable/disable.
// Caller is expected to run this BEFORE committing the bundle enabled state to the store.
func (s *SkillStore) runtimeApplyUserBundleEnabledDelta(
	ctx context.Context,
	sc *skillStoreSchema,
	bundleID bundleitemutils.BundleID,
	oldEnabled, newEnabled bool,
) error {
	if oldEnabled == newEnabled {
		return nil
	}

	bundleDefCounts, err := s.enabledDefCountsInUserBundle(sc, bundleID)
	if err != nil {
		return err
	}

	// ENABLE: validate/add all enabled skills in this bundle.
	if newEnabled {
		for def := range bundleDefCounts {
			if _, rtErr := s.runtimeTryAddForeground(ctx, def); rtErr != nil {
				return fmt.Errorf("runtime rejected bundle enable: %w", rtErr)
			}
		}
		return nil
	}

	// DISABLE: remove SkillDefs only if they become undesired globally (duplicate-safe).
	desiredCounts, err := s.runtimeDesiredDefCountsForSnapshot(ctx, *sc)
	if err != nil {
		return err
	}
	for def, n := range bundleDefCounts {
		after := desiredCounts[def] - n
		if after <= 0 {
			if rtErr := s.runtimeRemoveForegroundStrict(ctx, def); rtErr != nil {
				return fmt.Errorf("runtime remove failed: %w", rtErr)
			}
		}
	}
	return nil
}

func (s *SkillStore) runtimeRemoveForegroundStrict(ctx context.Context, def agentskillsSpec.SkillDef) error {
	if s == nil || s.runtime == nil {
		return fmt.Errorf("%w: runtime not configured", spec.ErrSkillInvalidRequest)
	}
	s.rtResyncMu.Lock()
	defer s.rtResyncMu.Unlock()
	return s.runtimeRemoveForegroundStrictLocked(ctx, def)
}

// runtimeRemoveForegroundStrictLocked is runtimeRemoveForegroundStrict but requires rtResyncMu held.
func (s *SkillStore) runtimeRemoveForegroundStrictLocked(ctx context.Context, def agentskillsSpec.SkillDef) error {
	ctx, cancel := context.WithTimeout(ctx, runtimeForegroundValidateTimeout)
	defer cancel()

	_, err := s.runtime.RemoveSkill(ctx, def)
	if err == nil || errors.Is(err, agentskillsSpec.ErrSkillNotFound) {
		return nil
	}
	return err
}

// runtimeTryAddForeground attempts to add/index a skill in runtime for strict foreground validation.
// Returns (addedByUs=true) if we successfully added; false if it already existed.
func (s *SkillStore) runtimeTryAddForeground(ctx context.Context, def agentskillsSpec.SkillDef) (bool, error) {
	if s == nil || s.runtime == nil {
		return false, fmt.Errorf("%w: runtime not configured", spec.ErrSkillInvalidRequest)
	}
	s.rtResyncMu.Lock()
	defer s.rtResyncMu.Unlock()
	return s.runtimeTryAddForegroundLocked(ctx, def)
}

// runtimeTryAddForegroundLocked is runtimeTryAddForeground but requires rtResyncMu held.
func (s *SkillStore) runtimeTryAddForegroundLocked(ctx context.Context, def agentskillsSpec.SkillDef) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, runtimeForegroundValidateTimeout)
	defer cancel()

	if _, err := s.runtime.AddSkill(ctx, def); err != nil {
		if errors.Is(err, agentskillsSpec.ErrSkillAlreadyExists) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// enabledDefCountsInUserBundle returns enabled SkillDef counts for all enabled skills in a bundle.
// Counts are used to make runtime removals duplicate-safe (same SkillDef may appear in multiple places).
func (s *SkillStore) enabledDefCountsInUserBundle(
	sc *skillStoreSchema,
	bundleID bundleitemutils.BundleID,
) (map[agentskillsSpec.SkillDef]int, error) {
	out := map[agentskillsSpec.SkillDef]int{}
	if sc == nil {
		return out, nil
	}
	sm := sc.Skills[bundleID]
	if sm == nil {
		return out, nil
	}
	for _, sk := range sm {
		if !sk.IsEnabled {
			continue
		}
		def, err := runtimeDefForUserSkill(sk)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", spec.ErrSkillInvalidRequest, err)
		}
		out[def]++
	}
	return out, nil
}

// withUserWriteSaga runs a strict foreground saga under the store write locks:
//  1. serialize writers (writeMu)
//  2. read user schema (mu.RLock)
//  3. fn does runtime strict validation/mutation + mutates schema in-memory (NO mu held)
//  4. commit schema to store (mu.Lock)
//  5. on store write failure: rollback runtime strictly to store
//  6. after successful commit: best-effort runtime resync.
//
// NOTE: fn is called while holding writeMu (so no other writer can change the store snapshot),
// but without holding mu (so readers are not blocked by runtime validation).
func (s *SkillStore) withUserWriteSaga(
	ctx context.Context,
	op string,
	fn func(sc *skillStoreSchema) (userWriteSagaOutcome, error),
) (err error) {
	if s == nil {
		return fmt.Errorf("%w: nil store", spec.ErrSkillInvalidRequest)
	}
	if fn == nil {
		return fmt.Errorf("%w: nil saga func", spec.ErrSkillInvalidRequest)
	}

	s.writeMu.Lock()

	var (
		outcome        userWriteSagaOutcome
		committed      bool
		rollbackReason string
		rollbackErr    error
		resyncReason   string
	)

	defer func() {
		// Always unlock writer serialization first.
		s.writeMu.Unlock()

		// Then do post-actions without holding store locks.
		if rollbackErr != nil {
			s.runtimeRollbackToStoreStrict(rollbackReason, rollbackErr)
			return
		}
		if committed && resyncReason != "" {
			s.bestEffortRuntimeResync(ctx, resyncReason)
		}
	}()

	// Read snapshot under mu, but keep mu held for as short as possible.
	s.mu.RLock()
	sc, err := s.readAllUser(false)
	s.mu.RUnlock()
	if err != nil {
		return err
	}

	outcome, err = fn(&sc)
	if err != nil {
		if outcome.RollbackReason != "" {
			rollbackReason = outcome.RollbackReason
			rollbackErr = outcome.RollbackErr
			if rollbackErr == nil {
				rollbackErr = err
			}
		}
		return err
	}

	// Commit under mu (exclusive), again held briefly.
	s.mu.Lock()
	err = s.writeAllUser(sc)
	s.mu.Unlock()
	if err != nil {
		rollbackReason = op + "(store-failed)"
		rollbackErr = err
		return err
	}

	committed = true
	resyncReason = outcome.ResyncReason
	return nil
}

// bestEffortRuntimeResync reconciles the runtime catalog to match the store's enabled skills.
// It must never panic and must never abort store initialization or CRUD.
//
// IMPORTANT:
//   - It does NOT persist any runtime-produced canonicalization.
//   - It uses only SkillDef (type/name/location) lifecycle selectors.
//   - It is safe to call frequently; it serializes internally.
func (s *SkillStore) bestEffortRuntimeResync(ctx context.Context, reason string) {
	if s == nil || s.runtime == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			slog.Error("runtime resync: panic", "reason", reason, "panic", r)
		}
	}()

	ctx, cancel := context.WithTimeout(ctx, runtimeResyncTimeout)
	defer cancel()

	if err := s.runtimeResyncFromStore(ctx); err != nil {
		slog.Error("runtime resync failed", "reason", reason, "err", err)
	}
}

// runtimeRollbackToStoreStrict retries a strict runtime reconcile to match the actual store.
// This is used when store write fails after runtime was mutated.
//
//nolint:contextcheck,nolintlint // Rollback is without req context.
func (s *SkillStore) runtimeRollbackToStoreStrict(
	reason string,
	storeErr error,
) {
	if s == nil || s.runtime == nil {
		return
	}

	// Don't depend on request context cancellation for rollback.
	baseCtx := context.Background()

	var lastErr error
	for attempt := 1; attempt <= runtimeRollbackAttempts; attempt++ {
		ctx, cancel := context.WithTimeout(baseCtx, runtimeResyncTimeout)
		err := s.runtimeResyncStrictFromStore(ctx)
		cancel()

		if err == nil {
			if attempt > 1 {
				slog.Warn("runtime rollback succeeded after retries", "reason", reason, "attempt", attempt)
			}
			return
		}
		lastErr = err
		slog.Warn(
			"runtime rollback attempt failed",
			"reason", reason,
			"attempt", attempt,
			"err", err,
			"storeErr", storeErr,
		)
		time.Sleep(runtimeRollbackBackoff)
	}

	slog.Error(
		"runtime rollback failed (giving up; restart will resync)",
		"reason", reason,
		"err", lastErr,
		"storeErr", storeErr,
	)
}

// runtimeResyncStrictFromStore performs a STRICT reconcile: any add/remove failure returns an error.
// This is used for rollback after store write failures.
func (s *SkillStore) runtimeResyncStrictFromStore(ctx context.Context) error {
	if s == nil || s.runtime == nil {
		return nil
	}
	// Strict rollback should ensure built-in hydration is present.
	s.rtResyncMu.Lock()
	if err := s.hydrateBuiltInEmbeddedFS(ctx); err != nil {
		s.rtResyncMu.Unlock()
		return err
	}
	s.rtResyncMu.Unlock()

	view, err := s.runtimeDesiredViewFromStore(ctx, runtimeDesiredViewOpts{
		WantByTypeName: true,
		LogInvalid:     true,
	})
	if err != nil {
		return err
	}
	s.rtResyncMu.Lock()
	defer s.rtResyncMu.Unlock()
	return s.runtimeApplyDesiredStrict(ctx, view.Set, view.ByTypeName)
}

// runtimeApplyDesiredStrict reconciles runtime to match desired, failing fast on add/remove errors.
func (s *SkillStore) runtimeApplyDesiredStrict(
	ctx context.Context,
	desired map[agentskillsSpec.SkillDef]struct{},
	desiredByTypeName map[string][]agentskillsSpec.SkillDef,
) (err error) {
	if s == nil || s.runtime == nil {
		return nil
	}

	defer func() {
		if r := recover(); r != nil {
			slog.Error("runtimeApplyDesiredStrict: panic", "panic", r)
			err = fmt.Errorf("runtimeApplyDesiredStrict panic: %v", r)

		}
	}()

	return s.runtimeApplyDesired(ctx, desired, desiredByTypeName, runtimeApplyStrict)
}

func (s *SkillStore) runtimeResyncFromStore(ctx context.Context) error {
	if s == nil || s.runtime == nil {
		return nil
	}

	// Best effort: attempt hydration but don't abort resync of user skills.
	s.rtResyncMu.Lock()
	if err := s.hydrateBuiltInEmbeddedFS(ctx); err != nil {
		slog.Error("runtime resync: embeddedfs hydration failed", "err", err)
	}
	s.rtResyncMu.Unlock()

	view, err := s.runtimeDesiredViewFromStore(ctx, runtimeDesiredViewOpts{
		WantByTypeName: true,
		LogInvalid:     true,
	})
	if err != nil {
		// Hard stop: do not mutate runtime if we couldn't build the desired view safely.
		return err
	}

	s.rtResyncMu.Lock()
	defer s.rtResyncMu.Unlock()
	return s.runtimeApplyDesired(ctx, view.Set, view.ByTypeName, runtimeApplyBestEffort)
}

type runtimeDesiredView struct {
	// Set is the desired unique SkillDefs (deduped).
	Set map[agentskillsSpec.SkillDef]struct{}

	// ByTypeName groups desired SkillDefs by (type,name) for replacement-safety rules.
	// Optional (only built when requested).
	ByTypeName map[string][]agentskillsSpec.SkillDef

	// Counts tracks how many enabled skills resolve to the same SkillDef (duplicate-safe removals).
	Counts map[agentskillsSpec.SkillDef]int
}

type runtimeDesiredViewOpts struct {
	WantByTypeName bool
	LogInvalid     bool
}

func (s *SkillStore) runtimeDesiredViewFromStore(
	ctx context.Context,
	opts runtimeDesiredViewOpts,
) (runtimeDesiredView, error) {
	s.mu.RLock()
	user, err := s.readAllUser(false)
	s.mu.RUnlock()
	if err != nil {
		return runtimeDesiredView{}, err
	}
	return s.runtimeDesiredViewForSnapshot(ctx, user, opts)
}

// runtimeDesiredDefCountsForSnapshot builds desired counts for enabled skills across builtin + user snapshot.
// Used for duplicate-safe runtime removals in foreground paths.
func (s *SkillStore) runtimeDesiredDefCountsForSnapshot(
	ctx context.Context,
	user skillStoreSchema,
) (map[agentskillsSpec.SkillDef]int, error) {
	view, err := s.runtimeDesiredViewForSnapshot(ctx, user, runtimeDesiredViewOpts{
		WantByTypeName: false,
		LogInvalid:     false,
	})
	if err != nil {
		return nil, err
	}
	return view.Counts, nil
}

// runtimeDesiredViewForSnapshot computes a consistent desired view across built-in + user snapshot.
func (s *SkillStore) runtimeDesiredViewForSnapshot(
	ctx context.Context,
	user skillStoreSchema,
	opts runtimeDesiredViewOpts,
) (runtimeDesiredView, error) {
	view := runtimeDesiredView{
		Set:    map[agentskillsSpec.SkillDef]struct{}{},
		Counts: map[agentskillsSpec.SkillDef]int{},
	}
	if opts.WantByTypeName {
		view.ByTypeName = map[string][]agentskillsSpec.SkillDef{}
	}

	add := func(def agentskillsSpec.SkillDef) {
		view.Set[def] = struct{}{}
		view.Counts[def]++
		if opts.WantByTypeName {
			k := typeNameKey(def.Type, def.Name)
			view.ByTypeName[k] = append(view.ByTypeName[k], def)
		}
	}

	// Built-ins (overlay view).
	if s.builtin != nil {
		bundles, skills, err := s.builtin.ListBuiltInSkills(ctx)
		if err != nil {
			return runtimeDesiredView{}, err
		}
		for bid, b := range bundles {
			if !b.IsEnabled {
				continue
			}
			for _, sk := range skills[bid] {
				if !sk.IsEnabled {
					continue
				}
				def, err := s.runtimeDefForStoreSkill(sk)
				if err != nil {
					if opts.LogInvalid {
						slog.Error(
							"runtime desired (builtin) invalid def",
							"bundleID", bid,
							"skill", sk.Slug,
							"err", err,
						)
					}
					continue
				}
				add(def)
			}
		}
	}

	// Users (snapshot provided).
	for bid, b := range user.Bundles {
		if isSoftDeletedSkillBundle(b) || !b.IsEnabled {
			continue
		}
		for _, sk := range user.Skills[bid] {
			if !sk.IsEnabled {
				continue
			}
			def, err := s.runtimeDefForStoreSkill(sk)
			if err != nil {
				if opts.LogInvalid {
					slog.Error(
						"runtime desired (user) invalid def",
						"bundleID", bid,
						"skill", sk.Slug,
						"err", err,
					)
				}
				continue
			}
			add(def)
		}
	}

	return view, nil
}

// runtimeApplyDesired reconciles runtime to match desired.
//   - In strict mode: add/remove errors fail fast (but ErrAlreadyExists/ErrNotFound are tolerated).
//   - In best-effort mode: add/remove errors are logged and ignored.
//
// Replacement safety rule (both modes):
// If a to-be-removed def has (type,name) replacements in desired, we only remove it if at least one
// desired replacement is known-present after the add phase.
func (s *SkillStore) runtimeApplyDesired(
	ctx context.Context,
	desired map[agentskillsSpec.SkillDef]struct{},
	desiredByTypeName map[string][]agentskillsSpec.SkillDef,
	mode runtimeApplyMode,
) error {
	if s == nil || s.runtime == nil {
		return nil
	}

	currentRecs, err := s.runtime.ListSkills(ctx, nil)
	if err != nil {
		return err
	}
	currentSet := make(map[agentskillsSpec.SkillDef]struct{}, len(currentRecs))
	for _, r := range currentRecs {
		currentSet[r.Def] = struct{}{}
	}

	// Determine additions.
	var toAdd []agentskillsSpec.SkillDef
	for def := range desired {
		if _, ok := currentSet[def]; !ok {
			toAdd = append(toAdd, def)
		}
	}
	sortSkillDefs(toAdd)

	// "presentAfterAdds" tracks which defs are known present after the add phase.
	// This is used by the replacement-safety rule for removals.
	presentAfterAdds := make(map[agentskillsSpec.SkillDef]struct{}, len(currentSet)+len(toAdd))
	for def := range currentSet {
		presentAfterAdds[def] = struct{}{}
	}

	for _, def := range toAdd {
		if _, err := s.runtime.AddSkill(ctx, def); err != nil {
			if errors.Is(err, agentskillsSpec.ErrSkillAlreadyExists) {
				presentAfterAdds[def] = struct{}{}
				continue
			}
			if mode == runtimeApplyStrict {
				return err
			}
			slog.Error("runtime add failed", "type", def.Type, "name", def.Name, "location", def.Location, "err", err)
			continue
		}
		presentAfterAdds[def] = struct{}{}
	}

	// Build replacement safety index (type+name that is desired and known present).
	desiredPresentTypeName := map[string]bool{}
	for def := range presentAfterAdds {
		if _, ok := desired[def]; ok {
			desiredPresentTypeName[typeNameKey(def.Type, def.Name)] = true
		}
	}

	// Determine removals.
	var toRemove []agentskillsSpec.SkillDef
	for def := range currentSet {
		if _, ok := desired[def]; !ok {
			toRemove = append(toRemove, def)
		}
	}
	sortSkillDefs(toRemove)

	for _, def := range toRemove {
		tn := typeNameKey(def.Type, def.Name)

		// Safety rule: if this looks like a "replacement" and none of the desired
		// replacements are present, skip removing the old one.
		if _, hasReplacement := desiredByTypeName[tn]; hasReplacement && !desiredPresentTypeName[tn] {
			if mode == runtimeApplyBestEffort {
				slog.Warn(
					"runtime remove skipped (replacement missing)",
					"type", def.Type,
					"name", def.Name,
					"location", def.Location,
				)
			}
			// Strict mode: silent skip (keeps previous behavior).
			continue
		}

		if _, err := s.runtime.RemoveSkill(ctx, def); err != nil {
			if errors.Is(err, agentskillsSpec.ErrSkillNotFound) {
				continue
			}
			if mode == runtimeApplyStrict {
				return err
			}
			slog.Error(
				"runtime remove failed",
				"type", def.Type,
				"name", def.Name,
				"location", def.Location,
				"err", err,
			)
		}
	}

	return nil
}

// runtimeDefForStoreSkill ensures ALL store skills resolve to the runtime lifecycle selector SkillDef.
func (s *SkillStore) runtimeDefForStoreSkill(sk spec.Skill) (agentskillsSpec.SkillDef, error) {
	if sk.IsBuiltIn {
		return s.runtimeDefForBuiltInSkill(sk)
	}
	return runtimeDefForUserSkill(sk)
}

func runtimeDefForUserSkill(sk spec.Skill) (agentskillsSpec.SkillDef, error) {
	if strings.TrimSpace(sk.Name) == "" || strings.TrimSpace(sk.Location) == "" {
		return agentskillsSpec.SkillDef{}, fmt.Errorf("%w: empty name/location", agentskillsSpec.ErrInvalidArgument)
	}
	return agentskillsSpec.SkillDef{
		Type:     string(sk.Type),
		Name:     sk.Name,
		Location: sk.Location,
	}, nil
}

func sortSkillDefs(defs []agentskillsSpec.SkillDef) {
	sort.Slice(defs, func(i, j int) bool {
		if defs[i].Type != defs[j].Type {
			return defs[i].Type < defs[j].Type
		}
		if defs[i].Name != defs[j].Name {
			return defs[i].Name < defs[j].Name
		}
		return defs[i].Location < defs[j].Location
	})
}

func typeNameKey(typ, name string) string {
	return typ + "\x00" + name
}
