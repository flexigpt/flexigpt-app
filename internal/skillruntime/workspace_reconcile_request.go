package skillruntime

import (
	"context"
	"log/slog"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

type workspaceReconcileRequest struct {
	Generation uint64
	Remove     bool
}

// RequestWorkspaceResync schedules one coalesced rebuild of a Workspace's
// derived Agent Skills runtime partition. Durable Workspace mutations must not
// wait for this rebuild or report its transient failures as metadata failures.
func (s *SkillRuntime) RequestWorkspaceResync(
	workspace basespec.CollectionRef,
) {
	s.requestWorkspaceReconciliation(workspace, false)
}

// RequestWorkspaceRemoval schedules removal of one retired Workspace's
// derived Agent Skills runtime partition.
func (s *SkillRuntime) RequestWorkspaceRemoval(
	workspace basespec.CollectionRef,
) {
	s.requestWorkspaceReconciliation(workspace, true)
}

func (s *SkillRuntime) requestWorkspaceReconciliation(
	workspace basespec.CollectionRef,
	remove bool,
) {
	if s == nil || workspace.Validate() != nil {
		return
	}
	parent, started := s.beginBackground()
	if !started {
		return
	}

	s.workspaceRequestMu.Lock()
	if request, exists := s.workspaceRequests[workspace]; exists {
		request.Generation++
		request.Remove = remove
		s.workspaceRequests[workspace] = request
		s.workspaceRequestMu.Unlock()
		s.endBackground()
		return
	}

	s.workspaceRequests[workspace] = workspaceReconcileRequest{
		Generation: 1,
		Remove:     remove,
	}
	s.workspaceRequestMu.Unlock()

	go s.runWorkspaceReconciliation(parent, workspace)
}

func (s *SkillRuntime) runWorkspaceReconciliation(
	parent context.Context,
	workspace basespec.CollectionRef,
) {
	defer s.endBackground()

	attempt := 0
	for {
		if parent.Err() != nil {
			s.workspaceRequestMu.Lock()
			delete(s.workspaceRequests, workspace)
			s.workspaceRequestMu.Unlock()
			return
		}

		s.workspaceRequestMu.Lock()
		request, exists := s.workspaceRequests[workspace]
		s.workspaceRequestMu.Unlock()
		if !exists {
			return
		}

		ctx, cancel := context.WithTimeout(parent, runtimeResyncTimeout)
		var err error
		if request.Remove {
			err = s.RemoveWorkspace(ctx, workspace)
		} else {
			err = s.ResyncWorkspace(ctx, workspace)
		}
		cancel()

		s.workspaceRequestMu.Lock()
		current, exists := s.workspaceRequests[workspace]
		if !exists {
			s.workspaceRequestMu.Unlock()
			return
		}
		if current.Generation != request.Generation {
			attempt = 0
			s.workspaceRequestMu.Unlock()
			continue
		}
		if err == nil {
			delete(s.workspaceRequests, workspace)
			s.workspaceRequestMu.Unlock()
			return
		}
		s.workspaceRequestMu.Unlock()

		attempt++
		action := "resync"
		if request.Remove {
			action = "remove"
		}
		if attempt >= workspaceReconcileMaxAttempts {
			s.workspaceRequestMu.Lock()
			current, exists := s.workspaceRequests[workspace]
			if exists && current.Generation == request.Generation {
				delete(s.workspaceRequests, workspace)
			}
			s.workspaceRequestMu.Unlock()
			slog.Error(
				"reconcile Workspace Skills exhausted retries",
				"action", action,
				"rootID", workspace.RootID,
				"collectionID", workspace.CollectionID,
				"attempts", attempt,
				"error", err,
			)
			return
		}

		delay := workspaceReconcileDelay(attempt)
		slog.Warn(
			"retry Workspace Skill reconciliation",
			"action", action,
			"rootID", workspace.RootID,
			"collectionID", workspace.CollectionID,
			"attempt", attempt,
			"retryIn", delay,
			"error", err,
		)
		timer := time.NewTimer(delay)
		select {
		case <-parent.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
		}
	}
}

func workspaceReconcileDelay(attempt int) time.Duration {
	delay := workspaceReconcileInitialDelay
	for index := 1; index < attempt; index++ {
		delay *= 2
		if delay >= workspaceReconcileMaximumDelay {
			return workspaceReconcileMaximumDelay
		}
	}
	return delay
}
