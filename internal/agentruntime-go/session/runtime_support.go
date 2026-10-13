package session

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const (
	defaultRunMaxModelOperations = 32
	defaultRunMaxToolAttempts    = 64
	maxRunMaxModelOperations     = 256
	maxRunMaxToolAttempts        = 512
)

func (r *Runtime) prepareSubmission(
	ctx context.Context,
	sessionID SessionID,
	configuration SessionConfiguration,
	input UserInput,
	persistConfiguration bool,
) (PreparedSubmission, error) {
	if err := ctx.Err(); err != nil {
		return PreparedSubmission{}, err
	}

	input = cloneJSON(input)
	attachments, err := r.prepareAttachments(ctx, input.Attachments)
	if err != nil {
		return PreparedSubmission{}, err
	}
	input.Attachments = attachments

	prepared := PreparedSubmission{
		Configuration:        cloneJSON(configuration),
		Input:                input,
		PersistConfiguration: persistConfiguration,
	}

	if r.deps.Preparation == nil {
		return prepared, nil
	}

	prepared, err = r.deps.Preparation.PrepareSubmission(
		ctx,
		SubmissionPreparationRequest{
			SessionID:            sessionID,
			Configuration:        cloneJSON(configuration),
			Input:                cloneJSON(input),
			PersistConfiguration: persistConfiguration,
		},
	)
	if err != nil {
		return PreparedSubmission{}, err
	}

	// Persistence is a command decision. A preparer may normalize the
	// configuration, but it must not silently change whether it becomes the
	// Session default.
	prepared.PersistConfiguration = persistConfiguration
	return prepared, nil
}

func (r *Runtime) prepareAttachments(
	ctx context.Context,
	references []AttachmentReference,
) ([]AttachmentReference, error) {
	if len(references) == 0 {
		return nil, nil
	}
	if r.deps.Attachments == nil {
		return nil, errors.New(
			"attachments were supplied but no attachment preparer is configured",
		)
	}

	values, err := r.deps.Attachments.PrepareAttachments(
		ctx,
		cloneJSON(references),
	)
	if err != nil {
		return nil, fmt.Errorf("prepare attachments: %w", err)
	}
	return values, nil
}

func normalizeRunLimits(input *RunLimits) (RunLimits, error) {
	output := RunLimits{
		MaxModelOperations: defaultRunMaxModelOperations,
		MaxToolAttempts:    defaultRunMaxToolAttempts,
	}
	if input == nil {
		return output, nil
	}

	output = cloneJSON(*input)
	if output.MaxModelOperations < 0 ||
		output.MaxToolAttempts < 0 {
		return RunLimits{}, errors.New("run limits cannot be negative")
	}
	if output.MaxModelOperations == 0 {
		output.MaxModelOperations = defaultRunMaxModelOperations
	}
	if output.MaxToolAttempts == 0 {
		output.MaxToolAttempts = defaultRunMaxToolAttempts
	}
	if output.MaxModelOperations > maxRunMaxModelOperations {
		return RunLimits{}, fmt.Errorf(
			"run model-operation limit cannot exceed %d",
			maxRunMaxModelOperations,
		)
	}
	if output.MaxToolAttempts > maxRunMaxToolAttempts {
		return RunLimits{}, fmt.Errorf(
			"run tool-attempt limit cannot exceed %d",
			maxRunMaxToolAttempts,
		)
	}
	return output, nil
}

func uniqueArtifactReferences(
	values []ArtifactReference,
) []ArtifactReference {
	seen := make(map[string]struct{}, len(values))
	output := make([]ArtifactReference, 0, len(values))

	for _, value := range values {
		key := value.RootID + "\x00" + value.ArtifactID
		if value.RootID == "" || value.ArtifactID == "" {
			continue
		}
		if _, found := seen[key]; found {
			continue
		}
		seen[key] = struct{}{}
		output = append(output, value)
	}

	return output
}

func hasPendingRecoveryReview(run *Run) bool {
	if run == nil {
		return false
	}
	for _, wait := range run.WaitingRequests {
		if wait.Kind != WaitKindRecovery {
			continue
		}
		if wait.Status == WaitStatusPending ||
			wait.Status == WaitStatusResolving {
			return true
		}
	}
	return false
}

func toolMutationMessage(run *Run, call *ToolCall) string {
	if run == nil || call == nil {
		return toolCallNotFound
	}
	if run.State.Terminal() || run.Controls.CancellationRequested {
		return "terminal or cancelling runs cannot change tool calls"
	}
	if hasPendingRecoveryReview(run) {
		return "recovery review must be resolved before changing tool calls"
	}
	if run.ActiveModelOperationID != "" {
		return "tool calls cannot change while a model operation is active"
	}
	if call.ConsumedByOperationID != "" {
		return "the selected tool result has already been consumed"
	}

	latest := latestBatchOperationID(run)
	if latest != "" && call.OriginOperationID != latest {
		return "the tool call does not belong to the current tool batch"
	}

	switch call.State {
	case ToolCallStateEvaluationQueued,
		ToolCallStateEvaluating,
		ToolCallStateExecutionQueued,
		ToolCallStateWaiting,
		ToolCallStateRunning:
		return "tool call is active and cannot be changed manually"
	default:
		return ""
	}
}

func (r *Runtime) cancelWaitExpiryLocked(waitID WaitID) {
	if cancel, found := r.waitExpirers[waitID]; found {
		delete(r.waitExpirers, waitID)
		cancel()
	}
}

func (r *Runtime) clearWaitReferenceLocked(waitID WaitID) {
	r.cancelWaitExpiryLocked(waitID)
	delete(r.waitReferences, waitID)
}

func (r *Runtime) cancelUndispatchedLocked(run *Run) {
	if run == nil {
		return
	}

	now := r.now()

	for index := range run.ModelOperations {
		operation := &run.ModelOperations[index]
		switch operation.State {
		case OperationStateQueued, OperationStatePreparing:
			operation.State = OperationStateCancelled
			operation.FinishedAt = now
			operation.Error = &OperationError{
				Code:    string(RunStateCancelled),
				Message: "cancelled before model invocation",
			}
			if run.ActiveModelOperationID == operation.ID {
				run.ActiveModelOperationID = ""
			}
		default:
		}
	}

	for index := range run.WaitingRequests {
		wait := &run.WaitingRequests[index]
		if wait.Status != WaitStatusPending &&
			wait.Status != WaitStatusResolving {
			continue
		}
		wait.Status = WaitStatusCancelled
		wait.Error = "run was cancelled"
		wait.UpdatedAt = now
		r.clearWaitReferenceLocked(wait.ID)
	}

	for index := range run.ToolCalls {
		call := &run.ToolCalls[index]
		switch call.State {
		case ToolCallStatePending,
			ToolCallStateEvaluationQueued,
			ToolCallStateEvaluating,
			ToolCallStateReady,
			ToolCallStateExecutionQueued,
			ToolCallStateWaiting:
			call.State = ToolCallStateCancelled
			call.Authorization = nil
			call.WaitID = ""
		default:
		}
	}

	run.RequestedCallIDs = nil
}

func hasActiveExternalWork(run *Run) bool {
	if run == nil {
		return false
	}

	for _, operation := range run.ModelOperations {
		if operation.State == OperationStateRunning {
			return true
		}
	}
	for _, call := range run.ToolCalls {
		if call.State == ToolCallStateRunning {
			return true
		}
	}
	return false
}

func (r *Runtime) releaseQueuedToolWorkLocked(
	sessionValue *Session,
	run *Run,
	call *ToolCall,
) bool {
	if sessionValue == nil || run == nil || call == nil {
		return false
	}

	switch call.State {
	case ToolCallStateEvaluationQueued:
		call.State = ToolCallStatePending
	case ToolCallStateExecutionQueued:
		call.State = ToolCallStateReady
	default:
		return false
	}

	r.touchRunLocked(sessionValue, run)
	return true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
