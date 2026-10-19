import type { SessionRun } from '@/spec/session';
import { SessionRunState, SessionToolCallState, SessionWaitStatus } from '@/spec/session_base';

/**
 * A waiting Run is still active, but is not necessarily busy generating.
 * Keep continuation eligibility in one place so rendering and submission
 * enforce the same rules.
 */
export function getSessionRunComposerState(run: SessionRun | undefined, hasUnconsumedToolResults = false) {
	const isActive = Boolean(
		run &&
		run.state !== SessionRunState.Completed &&
		run.state !== SessionRunState.Failed &&
		run.state !== SessionRunState.Cancelled
	);
	const pendingWait = isActive
		? run?.waitingRequests?.find(wait => wait.status === SessionWaitStatus.Pending)
		: undefined;
	const calls = isActive ? (run?.toolCalls ?? []) : [];
	const pendingExecutableToolCallIDs = calls
		.filter(
			call =>
				call.state === SessionToolCallState.Proposed ||
				call.state === SessionToolCallState.Pending ||
				call.state === SessionToolCallState.Ready
		)
		.map(call => call.id);

	// Failed/running/approval-waiting calls must not be mistaken for a
	// completed manual batch merely because they are not executable yet.
	const hasUnresolvedToolCalls = calls.some(
		call => call.state !== SessionToolCallState.Resolved && call.state !== SessionToolCallState.Observed
	);
	const isWaitingForUser = run?.state === SessionRunState.Waiting && !pendingWait;
	const hasStagedContinuation = Boolean(
		isActive && run && run.continuation.revision > (run.continuation.consumedRevision ?? 0)
	);
	const canComposeContinuationInput = isWaitingForUser && !hasUnresolvedToolCalls;
	const canContinueWithoutDraft =
		canComposeContinuationInput &&
		(hasStagedContinuation || hasUnconsumedToolResults || run?.continuationRequested === true);

	return {
		isActive,
		pendingWait,
		pendingExecutableToolCallIDs,
		hasStagedContinuation,
		canComposeContinuationInput,
		canContinueWithoutDraft,
		canRunPendingToolCalls: isWaitingForUser && pendingExecutableToolCallIDs.length > 0,
	};
}
