import { useCallback, useEffect, useRef, useState } from 'react';

import type { UIToolCall } from '@/spec/inference';

import { getNextPendingAutoExecutableToolCall } from '@/chats/composer/toolruntime/tool_runtime_utils';

export type AutoExecPhase = 'idle' | 'blocked' | 'running';

export interface AutoExecState {
	phase: AutoExecPhase;
	runningCallId: string | null;
}

interface UseToolAutoExecDrainerArgs {
	toolCalls: UIToolCall[];
	isBlocked: boolean;
	runToolCall: (id: string) => Promise<unknown>;
}

export function useToolAutoExecDrainer({ toolCalls, isBlocked, runToolCall }: UseToolAutoExecDrainerArgs): {
	state: AutoExecState;
} {
	const [state, setState] = useState<AutoExecState>({
		phase: 'idle',
		runningCallId: null,
	});

	const isMountedRef = useRef(true);
	const isBlockedRef = useRef(isBlocked);
	const toolCallsRef = useRef(toolCalls);
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

	const syncState = useCallback((phase: AutoExecPhase, runningCallId: string | null) => {
		if (!isMountedRef.current) return;

		setState(prev => {
			if (prev.phase === phase && prev.runningCallId === runningCallId) {
				return prev;
			}
			return { phase, runningCallId };
		});
	}, []);

	const getNextCall = useCallback(() => {
		return getNextPendingAutoExecutableToolCall(toolCallsRef.current);
	}, []);

	const pump = useCallback(async () => {
		if (isPumpingRef.current) return;

		const first = getNextCall();
		if (!first) {
			syncState('idle', null);
			return;
		}

		if (isBlockedRef.current) {
			syncState('blocked', null);
			return;
		}

		isPumpingRef.current = true;

		try {
			while (true) {
				const nextCall = getNextCall();

				if (!nextCall) {
					syncState('idle', null);
					return;
				}

				if (isBlockedRef.current) {
					syncState('blocked', null);
					return;
				}

				syncState('running', nextCall.id);

				try {
					await runToolCall(nextCall.id);
				} catch {
					// per-call runner owns failure state
				}
			}
		} finally {
			isPumpingRef.current = false;

			const nextCall = getNextCall();
			if (!nextCall) {
				syncState('idle', null);
				// eslint-disable-next-line no-unsafe-finally
				return;
			}

			if (isBlockedRef.current) {
				syncState('blocked', null);
				// eslint-disable-next-line no-unsafe-finally
				return;
			}

			if (scheduledFrameRef.current === null && isMountedRef.current) {
				scheduledFrameRef.current = window.requestAnimationFrame(() => {
					scheduledFrameRef.current = null;
					void pump();
				});
			}
		}
	}, [getNextCall, runToolCall, syncState]);

	useEffect(() => {
		toolCallsRef.current = toolCalls;
		isBlockedRef.current = isBlocked;

		if (scheduledFrameRef.current !== null) {
			window.cancelAnimationFrame(scheduledFrameRef.current);
			scheduledFrameRef.current = null;
		}

		void pump();
	}, [isBlocked, pump, toolCalls]);

	return { state };
}
