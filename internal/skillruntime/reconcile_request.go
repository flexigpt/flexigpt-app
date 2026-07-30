package skillruntime

import (
	"context"
	"log/slog"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
)

type collectionReconcileRequest struct {
	Generation uint64
	Remove     bool
}

// RequestCollectionResync schedules nonblocking, coalesced and fail-closed
// runtime reconciliation for one Artifact Store Collection.
func (s *SkillRuntime) RequestCollectionResync(
	ref collection.CollectionRef,
) {
	s.requestCollectionReconciliation(ref, false)
}

// RequestCollectionRemoval schedules removal of a retired or purged Collection
// from process-local Agent Skills state.
func (s *SkillRuntime) RequestCollectionRemoval(
	ref collection.CollectionRef,
) {
	s.requestCollectionReconciliation(ref, true)
}

func (s *SkillRuntime) requestCollectionReconciliation(
	ref collection.CollectionRef,
	remove bool,
) {
	if s == nil || ref.Validate() != nil {
		return
	}
	parent, started := s.beginBackground()
	if !started {
		return
	}

	s.collectionRequestMu.Lock()
	if current, exists := s.collectionRequests[ref]; exists {
		current.Generation++
		current.Remove = remove
		s.collectionRequests[ref] = current
		s.collectionRequestMu.Unlock()
		s.endBackground()
		return
	}
	s.collectionRequests[ref] = collectionReconcileRequest{
		Generation: 1,
		Remove:     remove,
	}
	s.collectionRequestMu.Unlock()

	go s.runCollectionReconciliation(parent, ref)
}

func (s *SkillRuntime) runCollectionReconciliation(
	parent context.Context,
	ref collection.CollectionRef,
) {
	defer s.endBackground()

	attempt := 0
	for {
		if parent.Err() != nil {
			s.collectionRequestMu.Lock()
			delete(s.collectionRequests, ref)
			s.collectionRequestMu.Unlock()
			return
		}

		s.collectionRequestMu.Lock()
		request, exists := s.collectionRequests[ref]
		s.collectionRequestMu.Unlock()
		if !exists {
			return
		}

		ctx, cancel := context.WithTimeout(parent, runtimeResyncTimeout)
		var err error
		if request.Remove {
			err = s.RemoveCollection(ctx, ref)
		} else {
			err = s.ResyncCollection(ctx, ref)
		}
		cancel()

		s.collectionRequestMu.Lock()
		current, exists := s.collectionRequests[ref]
		if !exists {
			s.collectionRequestMu.Unlock()
			return
		}
		if current.Generation != request.Generation {
			attempt = 0
			s.collectionRequestMu.Unlock()
			continue
		}
		if err == nil {
			delete(s.collectionRequests, ref)
			s.collectionRequestMu.Unlock()
			return
		}
		s.collectionRequestMu.Unlock()

		attempt++
		action := "resync"
		if request.Remove {
			action = "remove"
		}
		if attempt == collectionReconcileLogAfterAttempts {
			slog.Error(
				"skill runtime collection reconciliation remains pending",
				"action", action,
				"rootID", ref.RootID,
				"collectionID", ref.CollectionID,
				"attempts", attempt,
				"error", err,
			)
		}

		delay := collectionReconcileDelay(attempt)
		slog.Warn(
			"retry Skill runtime collection reconciliation",
			"action", action,
			"rootID", ref.RootID,
			"collectionID", ref.CollectionID,
			"attempt", attempt,
			"retryIn", delay,
			"error", err,
		)

		timer := time.NewTimer(delay)
		select {
		case <-parent.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return
		case <-timer.C:
		}
	}
}

func collectionReconcileDelay(attempt int) time.Duration {
	delay := collectionReconcileInitialDelay
	for index := 1; index < attempt; index++ {
		delay *= 2
		if delay >= collectionReconcileMaximumDelay {
			return collectionReconcileMaximumDelay
		}
	}
	return delay
}
