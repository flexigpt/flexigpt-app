package store

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"

	skillSpec "github.com/flexigpt/flexigpt-app/internal/skill/spec"
)

const (
	runtimeRollbackAttempts = 3
	runtimeRollbackBackoff  = 150 * time.Millisecond
)

func (s *SkillStore) runtimeRemoveForegroundStrict(ctx context.Context, def agentskillsSpec.SkillDef) error {
	if s == nil || s.runtime == nil {
		return fmt.Errorf("%w: runtime not configured", skillSpec.ErrSkillInvalidRequest)
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
