package store

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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

func (s *SkillStore) runtimeRemoveForegroundStrict(ctx context.Context, def agentskillsSpec.SkillDef) error {
	if s == nil || s.runtime == nil {
		return fmt.Errorf("%w: runtime not configured", spec.ErrSkillInvalidRequest)
	}
	ctx, cancel := context.WithTimeout(ctx, runtimeForegroundValidateTimeout)
	defer cancel()

	_, err := s.runtime.RemoveSkill(ctx, def)
	if err == nil {
		return nil
	}
	if errors.Is(err, agentskillsSpec.ErrSkillNotFound) {
		return nil
	}
	return err
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
