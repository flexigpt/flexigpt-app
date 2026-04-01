import { type Dispatch, type SetStateAction, useCallback, useEffect, useMemo, useReducer, useRef } from 'react';

import type { UIToolCall, UIToolOutput } from '@/spec/inference';
import type { SkillRef } from '@/spec/skill';
import { ToolStoreChoiceType } from '@/spec/tool';

import { resolveStateUpdate } from '@/lib/hook_utils';
import { ensureMakeID, getUUIDv7 } from '@/lib/uuid_utils';

import { executeComposerToolCall } from '@/chats/composer/toolruntime/execute_tool_call';
import {
	getPendingRunnableToolCalls,
	isRunnableComposerToolCall,
} from '@/chats/composer/toolruntime/tool_runtime_utils';
import { isSkillsToolName } from '@/skills/lib/skill_identity_utils';

export interface ComposerToolRuntimeState {
	toolCalls: UIToolCall[];
	toolOutputs: UIToolOutput[];
}

interface UseComposerToolRuntimeArgs {
	ensureSkillSession: () => Promise<string | null>;
	listActiveSkillRefs: (sid: string) => Promise<SkillRef[]>;
	setActiveSkillRefs: Dispatch<SetStateAction<SkillRef[]>>;
	getCurrentSkillSessionID: () => string | null;
}

type Action =
	| { type: 'set-calls'; toolCalls: UIToolCall[] }
	| { type: 'set-outputs'; toolOutputs: UIToolOutput[] }
	| { type: 'mark-running'; callId: string }
	| { type: 'mark-failed'; callId: string; errorMessage: string }
	| { type: 'succeed'; callId: string; output: UIToolOutput }
	| { type: 'discard'; callId: string }
	| { type: 'remove-output'; outputId: string }
	| { type: 'add-call-and-remove-output'; call: UIToolCall; outputId: string }
	| { type: 'clear' };

function runtimeReducer(state: ComposerToolRuntimeState, action: Action): ComposerToolRuntimeState {
	switch (action.type) {
		case 'set-calls':
			return state.toolCalls === action.toolCalls ? state : { ...state, toolCalls: action.toolCalls };

		case 'set-outputs':
			return state.toolOutputs === action.toolOutputs ? state : { ...state, toolOutputs: action.toolOutputs };

		case 'mark-running': {
			let changed = false;
			const nextToolCalls = state.toolCalls.map(toolCall => {
				if (toolCall.id !== action.callId) return toolCall;
				if (toolCall.status === 'running' && toolCall.errorMessage === undefined) return toolCall;
				changed = true;
				return { ...toolCall, status: 'running', errorMessage: undefined } as UIToolCall;
			});
			return changed ? { ...state, toolCalls: nextToolCalls } : state;
		}

		case 'mark-failed': {
			let changed = false;
			const nextToolCalls = state.toolCalls.map(toolCall => {
				if (toolCall.id !== action.callId) return toolCall;
				if (toolCall.status === 'failed' && toolCall.errorMessage === action.errorMessage) return toolCall;
				changed = true;
				return { ...toolCall, status: 'failed', errorMessage: action.errorMessage } as UIToolCall;
			});
			return changed ? { ...state, toolCalls: nextToolCalls } : state;
		}

		case 'succeed': {
			const exists = state.toolCalls.some(toolCall => toolCall.id === action.callId);
			if (!exists) return state;

			return {
				toolCalls: state.toolCalls.filter(toolCall => toolCall.id !== action.callId),
				toolOutputs: [...state.toolOutputs, action.output],
			};
		}

		case 'discard': {
			const nextToolCalls = state.toolCalls.filter(toolCall => toolCall.id !== action.callId);
			return nextToolCalls.length === state.toolCalls.length ? state : { ...state, toolCalls: nextToolCalls };
		}

		case 'remove-output': {
			const nextToolOutputs = state.toolOutputs.filter(output => output.id !== action.outputId);
			return nextToolOutputs.length === state.toolOutputs.length ? state : { ...state, toolOutputs: nextToolOutputs };
		}

		case 'add-call-and-remove-output':
			return {
				toolCalls: [...state.toolCalls, action.call],
				toolOutputs: state.toolOutputs.filter(output => output.id !== action.outputId),
			};

		case 'clear':
			return state.toolCalls.length === 0 && state.toolOutputs.length === 0
				? state
				: { toolCalls: [], toolOutputs: [] };

		default:
			return state;
	}
}

export function useComposerToolRuntime({
	ensureSkillSession,
	listActiveSkillRefs,
	setActiveSkillRefs,
	getCurrentSkillSessionID,
}: UseComposerToolRuntimeArgs): {
	toolCalls: UIToolCall[];
	toolOutputs: UIToolOutput[];
	setToolOutputs: Dispatch<SetStateAction<UIToolOutput[]>>;
	hasPendingToolCalls: boolean;
	hasRunningToolCalls: boolean;
	loadToolCalls: (toolCalls: UIToolCall[]) => void;
	runToolCall: (id: string) => Promise<UIToolOutput | null>;
	runAllPendingToolCalls: () => Promise<UIToolOutput[]>;
	discardToolCall: (id: string) => void;
	removeToolOutput: (id: string) => void;
	retryErroredOutput: (output: UIToolOutput) => void;
	clearToolRuntime: () => void;
	getToolRuntimeSnapshot: () => ComposerToolRuntimeState;
} {
	const [state, dispatch] = useReducer(runtimeReducer, {
		toolCalls: [],
		toolOutputs: [],
	});

	const stateRef = useRef(state);
	const isMountedRef = useRef(true);
	const toolCallAttemptKeyByIdRef = useRef(new Map<string, string>());

	useEffect(() => {
		stateRef.current = state;
	}, [state]);

	useEffect(() => {
		return () => {
			isMountedRef.current = false;
			// eslint-disable-next-line react-hooks/exhaustive-deps
			toolCallAttemptKeyByIdRef.current.clear();
		};
	}, []);

	useEffect(() => {
		const callIds = new Set(state.toolCalls.map(toolCall => toolCall.id));
		for (const callId of toolCallAttemptKeyByIdRef.current.keys()) {
			if (!callIds.has(callId)) {
				toolCallAttemptKeyByIdRef.current.delete(callId);
			}
		}
	}, [state.toolCalls]);

	const beginToolCallAttempt = useCallback((toolCallId: string) => {
		let attemptKey: string;
		try {
			attemptKey = getUUIDv7();
		} catch {
			attemptKey = ensureMakeID();
		}
		toolCallAttemptKeyByIdRef.current.set(toolCallId, attemptKey);
		return attemptKey;
	}, []);

	const isCurrentToolCallAttempt = useCallback((toolCallId: string, attemptKey: string) => {
		return toolCallAttemptKeyByIdRef.current.get(toolCallId) === attemptKey;
	}, []);

	const clearToolCallAttempt = useCallback((toolCallId: string, attemptKey?: string) => {
		const currentAttemptKey = toolCallAttemptKeyByIdRef.current.get(toolCallId);
		if (attemptKey === undefined || currentAttemptKey === attemptKey) {
			toolCallAttemptKeyByIdRef.current.delete(toolCallId);
		}
	}, []);

	const refreshActiveSkillRefs = useCallback(
		async (sid: string) => {
			try {
				const nextActive = await listActiveSkillRefs(sid);
				if (!isMountedRef.current) return;
				if (getCurrentSkillSessionID() !== sid) return;
				setActiveSkillRefs(nextActive);
			} catch {
				// ignore
			}
		},
		[getCurrentSkillSessionID, listActiveSkillRefs, setActiveSkillRefs]
	);

	const setToolOutputs = useCallback<Dispatch<SetStateAction<UIToolOutput[]>>>(update => {
		const next = resolveStateUpdate(update, stateRef.current.toolOutputs);
		dispatch({ type: 'set-outputs', toolOutputs: next });
	}, []);

	const loadToolCalls = useCallback((toolCalls: UIToolCall[]) => {
		toolCallAttemptKeyByIdRef.current.clear();
		dispatch({ type: 'set-calls', toolCalls });
	}, []);

	const runToolCall = useCallback(
		async (id: string): Promise<UIToolOutput | null> => {
			const toolCall = stateRef.current.toolCalls.find(current => current.id === id);
			if (!toolCall) return null;
			if (toolCall.status !== 'pending' && toolCall.status !== 'failed') return null;

			if (!isRunnableComposerToolCall(toolCall)) {
				const errorMessage = 'This tool call type cannot be executed from the composer.';
				dispatch({ type: 'mark-failed', callId: id, errorMessage });
				return null;
			}

			if (toolCallAttemptKeyByIdRef.current.has(id)) {
				return null;
			}

			const attemptKey = beginToolCallAttempt(id);
			dispatch({ type: 'mark-running', callId: id });

			try {
				const result = await executeComposerToolCall({
					toolCall,
					ensureSkillSession,
					getCurrentSkillSessionID,
				});

				if (!isCurrentToolCallAttempt(id, attemptKey)) {
					return null;
				}

				clearToolCallAttempt(id, attemptKey);

				if (!result.ok) {
					dispatch({ type: 'mark-failed', callId: id, errorMessage: result.errorMessage });
					return null;
				}

				dispatch({ type: 'succeed', callId: id, output: result.output });

				if (result.refreshActiveSkillRefsForSessionID) {
					void refreshActiveSkillRefs(result.refreshActiveSkillRefsForSessionID);
				}

				return result.output;
			} catch (err) {
				if (!isCurrentToolCallAttempt(id, attemptKey)) {
					return null;
				}

				clearToolCallAttempt(id, attemptKey);

				dispatch({
					type: 'mark-failed',
					callId: id,
					errorMessage: (err as Error)?.message || 'Tool invocation failed.',
				});

				return null;
			}
		},
		[
			beginToolCallAttempt,
			clearToolCallAttempt,
			ensureSkillSession,
			getCurrentSkillSessionID,
			isCurrentToolCallAttempt,
			refreshActiveSkillRefs,
		]
	);

	const runAllPendingToolCalls = useCallback(async (): Promise<UIToolOutput[]> => {
		const produced: UIToolOutput[] = [];

		while (true) {
			const pending = getPendingRunnableToolCalls(stateRef.current.toolCalls);
			if (pending.length === 0) break;

			for (const toolCall of pending) {
				const latest = stateRef.current.toolCalls.find(current => current.id === toolCall.id);
				if (!latest || latest.status !== 'pending') continue;

				const output = await runToolCall(latest.id);
				if (output) {
					produced.push(output);
				}
			}
		}

		return produced;
	}, [runToolCall]);

	const discardToolCall = useCallback(
		(id: string) => {
			clearToolCallAttempt(id);
			dispatch({ type: 'discard', callId: id });
		},
		[clearToolCallAttempt]
	);

	const removeToolOutput = useCallback((id: string) => {
		dispatch({ type: 'remove-output', outputId: id });
	}, []);

	const retryErroredOutput = useCallback((output: UIToolOutput) => {
		if (!output.isError || !output.arguments) return;

		const isSkills = isSkillsToolName(output.name);

		if (!isSkills) {
			if (!output.toolStoreChoice) return;
			if (output.type !== ToolStoreChoiceType.Function && output.type !== ToolStoreChoiceType.Custom) return;

			const { bundleID, toolSlug, toolVersion } = output.toolStoreChoice;
			if (!bundleID || !toolSlug || !toolVersion) return;
		} else {
			if (output.type !== ToolStoreChoiceType.Function && output.type !== ToolStoreChoiceType.Custom) return;
		}

		let newId: string;
		try {
			newId = getUUIDv7();
		} catch {
			newId = ensureMakeID();
		}

		const call: UIToolCall = {
			id: newId,
			callID: output.callID || newId,
			name: output.name,
			arguments: output.arguments,
			webSearchToolCallItems: output.webSearchToolCallItems,
			choiceID: output.choiceID,
			type: output.type,
			status: 'pending',
			toolStoreChoice: output.toolStoreChoice,
		};

		dispatch({
			type: 'add-call-and-remove-output',
			call,
			outputId: output.id,
		});
	}, []);

	const clearToolRuntime = useCallback(() => {
		toolCallAttemptKeyByIdRef.current.clear();
		dispatch({ type: 'clear' });
	}, []);

	const getToolRuntimeSnapshot = useCallback((): ComposerToolRuntimeState => {
		return stateRef.current;
	}, []);

	const hasPendingToolCalls = useMemo(
		() => state.toolCalls.some(toolCall => toolCall.status === 'pending'),
		[state.toolCalls]
	);
	const hasRunningToolCalls = useMemo(
		() => state.toolCalls.some(toolCall => toolCall.status === 'running'),
		[state.toolCalls]
	);

	return {
		toolCalls: state.toolCalls,
		toolOutputs: state.toolOutputs,
		setToolOutputs,
		hasPendingToolCalls,
		hasRunningToolCalls,
		loadToolCalls,
		runToolCall,
		runAllPendingToolCalls,
		discardToolCall,
		removeToolOutput,
		retryErroredOutput,
		clearToolRuntime,
		getToolRuntimeSnapshot,
	};
}
