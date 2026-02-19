package store

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"

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

// withUserWriteSaga runs a strict foreground saga under the store write locks:
//  1. read user schema
//  2. fn does runtime strict validation/mutation + mutates the schema in-memory
//  3. commit schema to store
//  4. on store write failure: rollback runtime strictly to store
//  5. after successful commit: best-effort runtime resync
//
// NOTE: fn is called while holding (sweepMu + mu). Keep fn bounded and avoid external blocking.
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

	s.sweepMu.Lock()
	s.mu.Lock()

	var (
		outcome        userWriteSagaOutcome
		committed      bool
		rollbackReason string
		rollbackErr    error
		resyncReason   string
	)

	defer func() {
		// Always unlock first.
		s.mu.Unlock()
		s.sweepMu.Unlock()

		// Then do post-actions without holding store locks.
		if rollbackErr != nil {
			s.runtimeRollbackToStoreStrict(rollbackReason, rollbackErr)
			return
		}
		if committed && resyncReason != "" {
			s.bestEffortRuntimeResync(ctx, resyncReason)
		}
	}()

	sc, err := s.readAllUser(false)
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

	if err := s.writeAllUser(sc); err != nil {
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
