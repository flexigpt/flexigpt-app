import { useCallback, useEffect, useRef, useState } from 'react';

import { ensureMakeID, getUUIDv7 } from '@/lib/uuid_utils';

export type AutoExecPhase = 'idle' | 'queued' | 'running';

export type AutoExecBatchOutcome = 'no-batch' | 'submitted' | 'manual' | 'cancelled' | 'replaced' | 'submit-failed';

export interface AutoExecBatchResult {
	batchId: string;
	outcome: AutoExecBatchOutcome;
	error?: string;
}

export interface AutoExecRuntimeSnapshot {
	hasPendingRunnableToolCalls: boolean;
	hasRunningRunnableToolCalls: boolean;
	hasFailedRunnableToolCalls: boolean;
}

interface ActiveAutoExecBatch {
	id: string;
	remainingCallIds: string[];
	runningCallId: string | null;
	autoSubmitAllowed: boolean;
	settled: boolean;
	resolve: (result: AutoExecBatchResult) => void;
}

export interface AutoExecMachineState {
	phase: AutoExecPhase;
	batchId: string | null;
	runningCallId: string | null;
	remainingCallIds: string[];
	lastResult: AutoExecBatchResult | null;
}

interface UseToolAutoExecMachineArgs {
	isBlocked: () => boolean;
	runCallSequentially: (callId: string) => Promise<void>;
	getRuntimeSnapshot: () => AutoExecRuntimeSnapshot;
	onAutoSubmitReady?: () => Promise<void> | void;
}

function getErrorMessage(error: unknown, fallback: string): string {
	if (error instanceof Error && error.message.trim().length > 0) {
		return error.message;
	}
	return fallback;
}

function makeBatchID(): string {
	try {
		return getUUIDv7();
	} catch {
		return ensureMakeID();
	}
}

export function useToolAutoExecMachine({
	isBlocked,
	runCallSequentially,
	getRuntimeSnapshot,
	onAutoSubmitReady,
}: UseToolAutoExecMachineArgs): {
	state: AutoExecMachineState;
	enqueueAutoExecBatch: (callIds: string[]) => Promise<AutoExecBatchResult>;
	resumeAutoExecBatch: () => void;
	removeCallFromAutoExecBatch: (callId: string) => void;
	clearAutoExecBatch: () => void;
} {
	const [state, setState] = useState<AutoExecMachineState>({
		phase: 'idle',
		batchId: null,
		runningCallId: null,
		remainingCallIds: [],
		lastResult: null,
	});

	const isMountedRef = useRef(true);
	const activeBatchRef = useRef<ActiveAutoExecBatch | null>(null);
	const isPumpingRef = useRef(false);
	const scheduledFrameRef = useRef<number | null>(null);

	useEffect(() => {
		return () => {
			isMountedRef.current = false;
			if (scheduledFrameRef.current !== null) {
				window.cancelAnimationFrame(scheduledFrameRef.current);
				scheduledFrameRef.current = null;
			}
		};
	}, []);

	const syncViewFromBatch = useCallback((phase: AutoExecPhase, batch: ActiveAutoExecBatch | null) => {
		setState(prev => {
			const next: AutoExecMachineState = {
				phase,
				batchId: batch?.id ?? null,
				runningCallId: batch?.runningCallId ?? null,
				remainingCallIds: batch ? [...batch.remainingCallIds] : [],
				lastResult: prev.lastResult,
			};

			if (
				prev.phase === next.phase &&
				prev.batchId === next.batchId &&
				prev.runningCallId === next.runningCallId &&
				prev.lastResult === next.lastResult &&
				prev.remainingCallIds.length === next.remainingCallIds.length &&
				prev.remainingCallIds.every((id, idx) => id === next.remainingCallIds[idx])
			) {
				return prev;
			}

			return next;
		});
	}, []);

	const settleBatch = useCallback((batchId: string, result: AutoExecBatchResult) => {
		const batch = activeBatchRef.current;
		if (!batch || batch.id !== batchId || batch.settled) return;

		batch.settled = true;
		activeBatchRef.current = null;
		batch.resolve(result);

		setState({
			phase: 'idle',
			batchId: null,
			runningCallId: null,
			remainingCallIds: [],
			lastResult: result,
		});
	}, []);

	const replaceActiveBatch = useCallback(() => {
		const current = activeBatchRef.current;
		if (!current) return;

		settleBatch(current.id, {
			batchId: current.id,
			outcome: 'replaced',
		});
	}, [settleBatch]);

	const pump = useCallback(async () => {
		if (isPumpingRef.current) return;

		const initialBatch = activeBatchRef.current;
		if (!initialBatch) return;

		if (isBlocked()) {
			syncViewFromBatch('queued', initialBatch);
			return;
		}

		isPumpingRef.current = true;

		try {
			while (true) {
				const batch = activeBatchRef.current;
				if (!batch) return;

				if (isBlocked()) {
					syncViewFromBatch('queued', batch);
					return;
				}

				const nextCallId = batch.remainingCallIds[0];

				if (!nextCallId) {
					if (!batch.autoSubmitAllowed) {
						settleBatch(batch.id, {
							batchId: batch.id,
							outcome: 'manual',
						});
						return;
					}

					const snapshot = getRuntimeSnapshot();
					const canAutoSubmit =
						!snapshot.hasPendingRunnableToolCalls &&
						!snapshot.hasRunningRunnableToolCalls &&
						!snapshot.hasFailedRunnableToolCalls;

					if (!canAutoSubmit) {
						settleBatch(batch.id, {
							batchId: batch.id,
							outcome: 'manual',
						});
						return;
					}

					try {
						await onAutoSubmitReady?.();
						settleBatch(batch.id, {
							batchId: batch.id,
							outcome: 'submitted',
						});
					} catch (error) {
						settleBatch(batch.id, {
							batchId: batch.id,
							outcome: 'submit-failed',
							error: getErrorMessage(error, 'Auto-submit failed.'),
						});
					}
					return;
				}

				batch.runningCallId = nextCallId;
				syncViewFromBatch('running', batch);

				try {
					await runCallSequentially(nextCallId);
				} catch {
					// The per-tool runner is responsible for marking failed state.
				}

				const latest = activeBatchRef.current;
				if (!latest || latest.id !== batch.id) return;

				latest.runningCallId = null;
				latest.remainingCallIds = latest.remainingCallIds.filter(id => id !== nextCallId);
			}
		} finally {
			isPumpingRef.current = false;

			const current = activeBatchRef.current;
			if (current && !isBlocked() && scheduledFrameRef.current === null) {
				scheduledFrameRef.current = window.requestAnimationFrame(() => {
					scheduledFrameRef.current = null;
					void pump();
				});
			}
		}
	}, [getRuntimeSnapshot, isBlocked, onAutoSubmitReady, runCallSequentially, settleBatch, syncViewFromBatch]);

	const schedulePump = useCallback(() => {
		if (!isMountedRef.current) return;
		if (scheduledFrameRef.current !== null) return;

		scheduledFrameRef.current = window.requestAnimationFrame(() => {
			scheduledFrameRef.current = null;
			void pump();
		});
	}, [pump]);

	const enqueueAutoExecBatch = useCallback(
		(callIds: string[]): Promise<AutoExecBatchResult> => {
			const uniqueCallIds = [...new Set(callIds.filter(Boolean))];

			if (uniqueCallIds.length === 0) {
				return Promise.resolve({
					batchId: makeBatchID(),
					outcome: 'no-batch',
				});
			}

			replaceActiveBatch();

			return new Promise<AutoExecBatchResult>(resolve => {
				const batch: ActiveAutoExecBatch = {
					id: makeBatchID(),
					remainingCallIds: [...uniqueCallIds],
					runningCallId: null,
					autoSubmitAllowed: true,
					settled: false,
					resolve,
				};

				activeBatchRef.current = batch;
				syncViewFromBatch(isBlocked() ? 'queued' : 'queued', batch);
				schedulePump();
			});
		},
		[isBlocked, replaceActiveBatch, schedulePump, syncViewFromBatch]
	);

	const resumeAutoExecBatch = useCallback(() => {
		const batch = activeBatchRef.current;
		if (!batch) return;
		schedulePump();
	}, [schedulePump]);

	const removeCallFromAutoExecBatch = useCallback(
		(callId: string) => {
			const batch = activeBatchRef.current;
			if (!batch) return;

			const hadCall = batch.remainingCallIds.includes(callId) || batch.runningCallId === callId;

			if (!hadCall) return;

			batch.autoSubmitAllowed = false;
			batch.remainingCallIds = batch.remainingCallIds.filter(id => id !== callId);

			if (batch.remainingCallIds.length === 0 && batch.runningCallId === null) {
				settleBatch(batch.id, {
					batchId: batch.id,
					outcome: 'cancelled',
				});
				return;
			}

			syncViewFromBatch(batch.runningCallId ? 'running' : 'queued', batch);
		},
		[settleBatch, syncViewFromBatch]
	);

	const clearAutoExecBatch = useCallback(() => {
		const batch = activeBatchRef.current;
		if (!batch) return;

		settleBatch(batch.id, {
			batchId: batch.id,
			outcome: 'cancelled',
		});
	}, [settleBatch]);

	return {
		state,
		enqueueAutoExecBatch,
		resumeAutoExecBatch,
		removeCallFromAutoExecBatch,
		clearAutoExecBatch,
	};
}
