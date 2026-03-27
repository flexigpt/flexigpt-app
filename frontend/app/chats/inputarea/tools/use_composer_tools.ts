import {
	type Dispatch,
	type SetStateAction,
	useCallback,
	useEffect,
	useLayoutEffect,
	useMemo,
	useRef,
	useState,
} from 'react';

import type { UIToolCall, UIToolOutput } from '@/spec/inference';
import type { SkillRef } from '@/spec/skill';
import { type Tool, type ToolArgsTarget, type ToolStoreChoice, ToolStoreChoiceType } from '@/spec/tool';

import { resolveStateUpdate } from '@/lib/hook_utils';
import { ensureMakeID, getUUIDv7 } from '@/lib/uuid_utils';

import { skillStoreAPI, toolRuntimeAPI, toolStoreAPI } from '@/apis/baseapi';

import { useOpenToolArgs } from '@/chats/inputarea/events/open_attached_toolargs';
import type { AttachedToolEntry } from '@/chats/inputarea/platedoc/tool_document_ops';
import {
	type ConversationToolStateEntry,
	initConversationToolsStateFromChoices,
} from '@/chats/inputarea/tools/conversation_tool_utils';
import type { ToolDetailsState } from '@/chats/inputarea/tools/tool_details_modal';
import {
	type AutoExecMachineState,
	type AutoExecRuntimeSnapshot,
	useToolAutoExecMachine,
} from '@/chats/inputarea/tools/use_tool_auto_execute_machine';
import {
	normalizeWebSearchChoiceTemplates,
	type WebSearchChoiceTemplate,
	webSearchTemplateFromChoice,
} from '@/chats/inputarea/tools/websearch_utils';
import { formatToolOutputSummary } from '@/tools/lib/tool_output_utils';
import { computeToolUserArgsStatus } from '@/tools/lib/tool_userargs_utils';

interface ComposerToolRuntimeState {
	toolCalls: UIToolCall[];
	toolOutputs: UIToolOutput[];
}

interface UseComposerToolsArgs {
	isBusy: boolean;
	isSubmitting: boolean;
	templateBlocked: boolean;
	ensureSkillSession: () => Promise<string | null>;
	listActiveSkillRefs: (sid: string) => Promise<SkillRef[]>;
	setActiveSkillRefs: Dispatch<SetStateAction<SkillRef[]>>;
	getCurrentSkillSessionID: () => string | null;
	toolArgsEventTarget?: EventTarget | null;
	onAutoSubmitReady?: () => Promise<void> | void;
	externalAutoExecuteBlocked?: boolean;
	getAttachedToolEntries: (uniqueByIdentity?: boolean) => AttachedToolEntry[];
}

interface UseComposerToolsResult {
	toolCalls: UIToolCall[];
	toolOutputs: UIToolOutput[];
	setToolCalls: Dispatch<SetStateAction<UIToolCall[]>>;
	setToolOutputs: Dispatch<SetStateAction<UIToolOutput[]>>;
	conversationToolsState: ConversationToolStateEntry[];
	setConversationToolsState: Dispatch<SetStateAction<ConversationToolStateEntry[]>>;
	webSearchTemplates: WebSearchChoiceTemplate[];
	setWebSearchTemplates: Dispatch<SetStateAction<WebSearchChoiceTemplate[]>>;
	toolDetailsState: ToolDetailsState;
	setToolDetailsState: Dispatch<SetStateAction<ToolDetailsState>>;
	toolArgsTarget: ToolArgsTarget | null;
	setToolArgsTarget: Dispatch<SetStateAction<ToolArgsTarget | null>>;
	toolArgsBlocked: boolean;
	hasPendingToolCalls: boolean;
	hasRunningToolCalls: boolean;
	recomputeAttachedToolArgsBlocked: () => void;
	runAllPendingToolCalls: () => Promise<UIToolOutput[]>;
	handleRunSingleToolCall: (id: string) => Promise<void>;
	handleDiscardToolCall: (id: string) => void;
	handleRemoveToolOutput: (id: string) => void;
	handleRetryErroredOutput: (output: UIToolOutput) => void;
	handleAttachedToolsChanged: () => void;
	applyConversationToolsFromChoices: (tools: ToolStoreChoice[]) => void;
	applyWebSearchFromChoices: (tools: ToolStoreChoice[]) => void;
	loadToolCalls: (toolCalls: UIToolCall[]) => void;
	handleOpenToolOutput: (output: UIToolOutput) => void;
	handleOpenToolCallDetails: (call: UIToolCall) => void;
	handleOpenConversationToolDetails: (entry: ConversationToolStateEntry) => void;
	handleOpenAttachedToolDetails: (entry: AttachedToolEntry) => void;
	clearComposerToolsState: () => void;
	getToolRuntimeSnapshot: () => ComposerToolRuntimeState;
	resumeAutoExecBatch: () => void;
	autoExecState: AutoExecMachineState;
}

const TOOL_CALL_TIMEOUT_MS = 90_000;

function withTimeout<T>(promise: Promise<T>, ms: number, timeoutMessage: string): Promise<T> {
	return new Promise<T>((resolve, reject) => {
		const timer = window.setTimeout(() => {
			reject(new Error(timeoutMessage));
		}, ms);

		promise.then(
			value => {
				window.clearTimeout(timer);
				resolve(value);
			},
			(error: unknown) => {
				window.clearTimeout(timer);
				reject(error);
			}
		);
	});
}

const conversationToolHydrationKey = (entry: ConversationToolStateEntry): string => {
	return `${entry.toolStoreChoice.bundleID}::${entry.toolStoreChoice.toolSlug}::${entry.toolStoreChoice.toolVersion}`;
};

function isRunnableComposerToolCall(toolCall: UIToolCall): boolean {
	return toolCall.type === ToolStoreChoiceType.Function || toolCall.type === ToolStoreChoiceType.Custom;
}

function getPendingRunnableToolCalls(toolCalls: UIToolCall[], autoExecuteOnly = false): UIToolCall[] {
	return toolCalls.filter(toolCall => {
		if (toolCall.status !== 'pending') return false;
		if (!isRunnableComposerToolCall(toolCall)) return false;
		if (autoExecuteOnly && !toolCall.toolStoreChoice?.autoExecute) return false;
		return true;
	});
}

function getPendingAutoExecuteToolCalls(toolCalls: UIToolCall[]): UIToolCall[] {
	return getPendingRunnableToolCalls(toolCalls, true);
}

function hasRunningRunnableToolCalls(toolCalls: UIToolCall[]): boolean {
	return toolCalls.some(toolCall => toolCall.status === 'running' && isRunnableComposerToolCall(toolCall));
}

function hasFailedRunnableToolCalls(toolCalls: UIToolCall[]): boolean {
	return toolCalls.some(toolCall => toolCall.status === 'failed' && isRunnableComposerToolCall(toolCall));
}

function getConversationToolArgsBlocked(entries: ConversationToolStateEntry[]): boolean {
	for (const entry of entries) {
		if (!entry.enabled) continue;

		const status = entry.argStatus;
		if (status?.hasSchema && !status.isSatisfied) {
			return true;
		}
	}
	return false;
}

export function useComposerTools({
	isBusy,
	isSubmitting,
	templateBlocked,
	ensureSkillSession,
	listActiveSkillRefs,
	setActiveSkillRefs,
	getCurrentSkillSessionID,
	toolArgsEventTarget,
	onAutoSubmitReady,
	externalAutoExecuteBlocked = false,
	getAttachedToolEntries,
}: UseComposerToolsArgs): UseComposerToolsResult {
	const isMountedRef = useRef(true);
	useEffect(() => {
		return () => {
			isMountedRef.current = false;
		};
	}, []);

	// Tool-call chips (assistant-suggested) + tool outputs attached to the next user message.
	const [toolRuntimeState, setToolRuntimeStateRaw] = useState<ComposerToolRuntimeState>({
		toolCalls: [],
		toolOutputs: [],
	});
	const toolRuntimeStateRef = useRef<ComposerToolRuntimeState>({
		toolCalls: [],
		toolOutputs: [],
	});

	const toolCalls = toolRuntimeState.toolCalls;
	const toolOutputs = toolRuntimeState.toolOutputs;
	const [toolDetailsState, setToolDetailsState] = useState<ToolDetailsState>(null);
	const [toolArgsTarget, setToolArgsTarget] = useState<ToolArgsTarget | null>(null);

	const [conversationToolsState, setConversationToolsStateRaw] = useState<ConversationToolStateEntry[]>([]);
	const conversationToolsStateRef = useRef<ConversationToolStateEntry[]>([]);
	const isBusyRef = useRef(isBusy);
	const isSubmittingRef = useRef(isSubmitting);
	const templateBlockedRef = useRef(templateBlocked);
	const externalAutoExecuteBlockedRef = useRef(externalAutoExecuteBlocked);

	// eslint-disable-next-line @typescript-eslint/no-unnecessary-type-arguments
	const conversationToolDefsCacheRef = useRef<Map<string, Tool>>(new Map());
	const toolCallAttemptKeyByIdRef = useRef(new Map<string, string>());

	const hydratingConversationToolKeysRef = useRef(new Set());

	// Arg-blocking state, split by attached-vs-conversation tools.
	const [attachedToolArgsBlocked, setAttachedToolArgsBlocked] = useState(false);
	const attachedToolArgsBlockedRef = useRef(false);

	const [webSearchTemplates, setWebSearchTemplatesRaw] = useState<WebSearchChoiceTemplate[]>([]);
	const webSearchTemplatesRef = useRef<WebSearchChoiceTemplate[]>([]);

	const conversationToolArgsBlocked = useMemo(
		() => getConversationToolArgsBlocked(conversationToolsState),
		[conversationToolsState]
	);

	const toolArgsBlocked = attachedToolArgsBlocked || conversationToolArgsBlocked;
	const toolArgsBlockedRef = useRef(false);

	useEffect(() => {
		toolRuntimeStateRef.current = toolRuntimeState;
	}, [toolRuntimeState]);

	useLayoutEffect(() => {
		isBusyRef.current = isBusy;
	}, [isBusy]);

	useLayoutEffect(() => {
		isSubmittingRef.current = isSubmitting;
	}, [isSubmitting]);

	useLayoutEffect(() => {
		templateBlockedRef.current = templateBlocked;
	}, [templateBlocked]);

	useLayoutEffect(() => {
		externalAutoExecuteBlockedRef.current = externalAutoExecuteBlocked;
	}, [externalAutoExecuteBlocked]);

	useLayoutEffect(() => {
		attachedToolArgsBlockedRef.current = attachedToolArgsBlocked;
		toolArgsBlockedRef.current =
			attachedToolArgsBlocked || getConversationToolArgsBlocked(conversationToolsStateRef.current);
	}, [attachedToolArgsBlocked]);

	useLayoutEffect(() => {
		toolArgsBlockedRef.current = attachedToolArgsBlockedRef.current || conversationToolArgsBlocked;
	}, [conversationToolArgsBlocked]);

	const updateToolRuntimeState = useCallback((update: SetStateAction<ComposerToolRuntimeState>) => {
		const prev = toolRuntimeStateRef.current;
		const next = resolveStateUpdate(update, prev);
		toolRuntimeStateRef.current = next;

		setToolRuntimeStateRaw(current => (current === next ? current : next));
	}, []);

	const setToolCalls = useCallback<Dispatch<SetStateAction<UIToolCall[]>>>(
		update => {
			updateToolRuntimeState(prev => {
				const nextToolCalls = resolveStateUpdate(update, prev.toolCalls);
				if (nextToolCalls === prev.toolCalls) return prev;
				return { ...prev, toolCalls: nextToolCalls };
			});
		},
		[updateToolRuntimeState]
	);

	const setToolOutputs = useCallback<Dispatch<SetStateAction<UIToolOutput[]>>>(
		update => {
			updateToolRuntimeState(prev => {
				const nextToolOutputs = resolveStateUpdate(update, prev.toolOutputs);
				if (nextToolOutputs === prev.toolOutputs) return prev;
				return { ...prev, toolOutputs: nextToolOutputs };
			});
		},
		[updateToolRuntimeState]
	);

	useOpenToolArgs(target => {
		setToolArgsTarget(target);
	}, toolArgsEventTarget);

	useEffect(() => {
		conversationToolsStateRef.current = conversationToolsState;
	}, [conversationToolsState]);

	const setWebSearchTemplates = useCallback<Dispatch<SetStateAction<WebSearchChoiceTemplate[]>>>(update => {
		const prev = webSearchTemplatesRef.current;
		const requested = resolveStateUpdate(update, prev);
		const next = normalizeWebSearchChoiceTemplates(requested);

		if (
			prev.length === next.length &&
			prev.every(
				(item, idx) =>
					item.bundleID === next[idx]?.bundleID &&
					item.toolSlug === next[idx]?.toolSlug &&
					item.toolVersion === next[idx]?.toolVersion &&
					item.userArgSchemaInstance === next[idx]?.userArgSchemaInstance
			)
		) {
			return;
		}

		webSearchTemplatesRef.current = next;
		setWebSearchTemplatesRaw(next);
	}, []);

	const hasPendingToolCalls = useMemo(() => toolCalls.some(c => c.status === 'pending'), [toolCalls]);
	const hasRunningToolCalls = useMemo(() => toolCalls.some(c => c.status === 'running'), [toolCalls]);

	useEffect(() => {
		const callIds = new Set(toolCalls.map(toolCall => toolCall.id));
		for (const callId of toolCallAttemptKeyByIdRef.current.keys()) {
			// eslint-disable-next-line react-you-might-not-need-an-effect/no-event-handler
			if (!callIds.has(callId)) {
				toolCallAttemptKeyByIdRef.current.delete(callId);
			}
		}
	}, [toolCalls]);

	const isSkillsToolName = useCallback((name: string | undefined): boolean => {
		const n = (name ?? '').trim();
		return n.startsWith('skills.');
	}, []);

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

	const runToolCallInternal = useCallback(
		async (toolCall: UIToolCall): Promise<UIToolOutput | null> => {
			if (toolCall.type !== ToolStoreChoiceType.Function && toolCall.type !== ToolStoreChoiceType.Custom) {
				const errMsg = 'This tool call type cannot be executed from the composer.';
				setToolCalls(prev =>
					prev.map(c => (c.id === toolCall.id ? { ...c, status: 'failed', errorMessage: errMsg } : c))
				);
				return null;
			}

			const args = toolCall.arguments && toolCall.arguments.trim().length > 0 ? toolCall.arguments : undefined;

			// ---- Skills tool path (runtime-injected tools: skills.*) ----
			if (isSkillsToolName(toolCall.name)) {
				let sid = getCurrentSkillSessionID();
				if (!sid) {
					try {
						sid = await ensureSkillSession();
					} catch {
						sid = null;
					}
				}

				if (!sid) {
					const errMsg = 'No active skills session. Enable skills and resend, or run again after a session is created.';
					setToolCalls(prev =>
						prev.map(c => (c.id === toolCall.id ? { ...c, status: 'failed', errorMessage: errMsg } : c))
					);
					return null;
				}

				if (toolCallAttemptKeyByIdRef.current.has(toolCall.id)) {
					return null;
				}

				const attemptKey = beginToolCallAttempt(toolCall.id);

				// Mark as running (allow retry after failure by overwriting previous status).
				setToolCalls(prev =>
					prev.map(c => (c.id === toolCall.id ? { ...c, status: 'running', errorMessage: undefined } : c))
				);

				try {
					const resp = await withTimeout(
						skillStoreAPI.invokeSkillTool(sid, toolCall.name, args),
						TOOL_CALL_TIMEOUT_MS,
						`Tool call "${toolCall.name}" timed out after ${Math.round(TOOL_CALL_TIMEOUT_MS / 1000)} seconds.`
					);

					if (!isCurrentToolCallAttempt(toolCall.id, attemptKey)) {
						return null;
					}

					const isError = !!resp.isError;
					const errorMessage =
						resp.errorMessage ||
						(isError ? 'Skill tool reported an error. Inspect the output for details.' : undefined);

					clearToolCallAttempt(toolCall.id, attemptKey);

					const output: UIToolOutput = {
						id: toolCall.id,
						callID: toolCall.callID,
						name: toolCall.name,
						choiceID: toolCall.choiceID,
						type: toolCall.type,
						summary: isError
							? `Error: ${formatToolOutputSummary(toolCall.name)}`
							: formatToolOutputSummary(toolCall.name),
						toolOutputs: resp.outputs,
						isError,
						errorMessage,
						arguments: toolCall.arguments,
						webSearchToolCallItems: toolCall.webSearchToolCallItems,
						toolStoreChoice: toolCall.toolStoreChoice,
					};

					updateToolRuntimeState(prev => ({
						...(() => {
							const current = prev.toolCalls.find(c => c.id === toolCall.id);
							if (!current || current.status !== 'running') return prev;
							return {
								toolCalls: prev.toolCalls.filter(c => c.id !== toolCall.id),
								toolOutputs: [...prev.toolOutputs, output],
							};
						})(),
					}));

					// Refresh active skills after load/unload.
					void (async () => {
						try {
							const nextActive = await listActiveSkillRefs(sid);
							if (getCurrentSkillSessionID() !== sid) return;
							setActiveSkillRefs(nextActive);
						} catch {
							// ignore
						}
					})();

					return output;
				} catch (err) {
					if (!isCurrentToolCallAttempt(toolCall.id, attemptKey)) {
						return null;
					}
					const msg = (err as Error)?.message || 'Skill tool invocation failed.';
					clearToolCallAttempt(toolCall.id, attemptKey);
					setToolCalls(prev =>
						prev.map(c => (c.id === toolCall.id ? { ...c, status: 'failed', errorMessage: msg } : c))
					);
					return null;
				}
			}

			// Resolve identity using toolStoreChoice when available; fall back to name parsing.
			let bundleID: string | undefined;
			let toolSlug: string | undefined;
			let version: string | undefined;

			if (toolCall.toolStoreChoice) {
				bundleID = toolCall.toolStoreChoice.bundleID;
				toolSlug = toolCall.toolStoreChoice.toolSlug;
				version = toolCall.toolStoreChoice.toolVersion;
			}

			if (!bundleID || !toolSlug || !version) {
				const errMsg = 'Cannot resolve tool identity for this call.';
				setToolCalls(prev =>
					prev.map(c => (c.id === toolCall.id ? { ...c, status: 'failed', errorMessage: errMsg } : c))
				);
				return null;
			}

			// Mark as running (allow retry after failure by overwriting previous status).
			setToolCalls(prev =>
				prev.map(c => (c.id === toolCall.id ? { ...c, status: 'running', errorMessage: undefined } : c))
			);

			try {
				const resp = await toolRuntimeAPI.invokeTool(bundleID, toolSlug, version, args);
				const isError = !!resp.isError;
				const errorMessage =
					resp.errorMessage || (isError ? 'Tool reported an error. Inspect the output for details.' : undefined);

				const output: UIToolOutput = {
					id: toolCall.id,
					callID: toolCall.callID,
					name: toolCall.name,
					choiceID: toolCall.choiceID,
					type: toolCall.type,
					summary: isError
						? `Error: ${formatToolOutputSummary(toolCall.name)}`
						: formatToolOutputSummary(toolCall.name),
					toolOutputs: resp.outputs,
					isError,
					errorMessage,
					arguments: toolCall.arguments,
					webSearchToolCallItems: toolCall.webSearchToolCallItems,
					toolStoreChoice: toolCall.toolStoreChoice,
				};

				// Remove the call chip & append the output.
				updateToolRuntimeState(prev => ({
					toolCalls: prev.toolCalls.filter(c => c.id !== toolCall.id),
					toolOutputs: [...prev.toolOutputs, output],
				}));

				return output;
			} catch (err) {
				const msg = (err as Error)?.message || 'Tool invocation failed.';
				setToolCalls(prev => prev.map(c => (c.id === toolCall.id ? { ...c, status: 'failed', errorMessage: msg } : c)));
				return null;
			}
		},
		[
			beginToolCallAttempt,
			clearToolCallAttempt,
			ensureSkillSession,
			getCurrentSkillSessionID,
			isCurrentToolCallAttempt,
			isSkillsToolName,
			listActiveSkillRefs,
			setActiveSkillRefs,
			setToolCalls,
			updateToolRuntimeState,
		]
	);

	const getAutoExecBlocked = useCallback(() => {
		return (
			isBusyRef.current ||
			isSubmittingRef.current ||
			templateBlockedRef.current ||
			toolArgsBlockedRef.current ||
			externalAutoExecuteBlockedRef.current
		);
	}, []);

	const getAutoExecRuntimeSnapshot = useCallback((): AutoExecRuntimeSnapshot => {
		const currentToolCalls = toolRuntimeStateRef.current.toolCalls;
		return {
			hasPendingRunnableToolCalls: getPendingRunnableToolCalls(currentToolCalls).length > 0,
			hasRunningRunnableToolCalls: hasRunningRunnableToolCalls(currentToolCalls),
			hasFailedRunnableToolCalls: hasFailedRunnableToolCalls(currentToolCalls),
		};
	}, []);

	const runAutoExecCallById = useCallback(
		async (callId: string) => {
			const toolCall = toolRuntimeStateRef.current.toolCalls.find(
				c =>
					c.id === callId && c.status === 'pending' && c.toolStoreChoice?.autoExecute && isRunnableComposerToolCall(c)
			);
			if (!toolCall) return;
			await runToolCallInternal(toolCall);
		},
		[runToolCallInternal]
	);

	const {
		state: autoExecState,
		enqueueAutoExecBatch,
		resumeAutoExecBatch,
		removeCallFromAutoExecBatch,
		clearAutoExecBatch,
	} = useToolAutoExecMachine({
		isBlocked: getAutoExecBlocked,
		runCallSequentially: runAutoExecCallById,
		getRuntimeSnapshot: getAutoExecRuntimeSnapshot,
		onAutoSubmitReady,
	});

	const drainPendingToolCalls = useCallback(
		async (autoExecuteOnly: boolean): Promise<UIToolOutput[]> => {
			const produced: UIToolOutput[] = [];

			while (true) {
				const pending = getPendingRunnableToolCalls(toolRuntimeStateRef.current.toolCalls, autoExecuteOnly);
				if (pending.length === 0) break;

				for (const toolCall of pending) {
					const latest = toolRuntimeStateRef.current.toolCalls.find(current => current.id === toolCall.id);
					if (!latest || latest.status !== 'pending') continue;
					if (autoExecuteOnly && !latest.toolStoreChoice?.autoExecute) continue;

					const out = await runToolCallInternal(latest);
					if (out) produced.push(out);
				}
			}

			return produced;
		},
		[runToolCallInternal]
	);

	/**
	 * Run all currently pending tool calls (in sequence) and return the
	 * UIToolOutput objects produced in this pass.
	 */
	const runAllPendingToolCalls = useCallback(async (): Promise<UIToolOutput[]> => {
		return drainPendingToolCalls(false);
	}, [drainPendingToolCalls]);

	const handleRunSingleToolCall = useCallback(
		async (id: string) => {
			removeCallFromAutoExecBatch(id);
			const chip = toolRuntimeStateRef.current.toolCalls.find(
				c => c.id === id && (c.status === 'pending' || c.status === 'failed')
			);
			if (!chip) return;
			await runToolCallInternal(chip);
		},
		[removeCallFromAutoExecBatch, runToolCallInternal]
	);

	const handleDiscardToolCall = useCallback(
		(id: string) => {
			removeCallFromAutoExecBatch(id);
			clearToolCallAttempt(id);
			setToolCalls(prev => {
				const next = prev.filter(c => c.id !== id);
				return next.length === prev.length ? prev : next;
			});
		},
		[clearToolCallAttempt, removeCallFromAutoExecBatch, setToolCalls]
	);

	const handleRemoveToolOutput = useCallback(
		(id: string) => {
			setToolOutputs(prev => {
				const next = prev.filter(o => o.id !== id);
				return next.length === prev.length ? prev : next;
			});
			setToolDetailsState(current =>
				current && current.kind === 'output' && current.output.id === id ? null : current
			);
		},
		[setToolOutputs]
	);

	const handleRetryErroredOutput = useCallback(
		(output: UIToolOutput) => {
			// Only support retry when we still know arguments. For non-skills tools we
			// also require toolStoreChoice; for skills.* we route via skill session.
			if (!output.isError || !output.arguments) return;

			const isSkills = isSkillsToolName(output.name);

			if (!isSkills) {
				if (!output.toolStoreChoice) return;
				if (output.type !== ToolStoreChoiceType.Function && output.type !== ToolStoreChoiceType.Custom) return;

				const { bundleID, toolSlug, toolVersion } = output.toolStoreChoice;
				if (!bundleID || !toolSlug || !toolVersion) return;
			} else {
				// skills.* are expected to be Function or Custom in the inference schema;
				// if something else slips through, don't retry.
				if (output.type !== ToolStoreChoiceType.Function && output.type !== ToolStoreChoiceType.Custom) return;
			}

			let newId: string;
			try {
				newId = getUUIDv7();
			} catch {
				newId = ensureMakeID();
			}

			const chip: UIToolCall = {
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

			updateToolRuntimeState(prev => ({
				toolCalls: [...prev.toolCalls, chip],
				toolOutputs: prev.toolOutputs.filter(o => o.id !== output.id),
			}));
			if (chip.toolStoreChoice?.autoExecute) {
				void enqueueAutoExecBatch([chip.id]);
			}
		},
		[enqueueAutoExecBatch, isSkillsToolName, updateToolRuntimeState]
	);

	const primeConversationToolsFromCache = useCallback((entries: ConversationToolStateEntry[]) => {
		let changed = false;

		const next = entries.map(entry => {
			const cacheKey = conversationToolHydrationKey(entry);
			const def = entry.toolDefinition ?? conversationToolDefsCacheRef.current.get(cacheKey);
			if (!def) return entry;

			const argStatus = computeToolUserArgsStatus(def.userArgSchema, entry.toolStoreChoice.userArgSchemaInstance);
			if (entry.toolDefinition === def && entry.argStatus === argStatus) return entry;

			changed = true;
			return { ...entry, toolDefinition: def, argStatus };
		});

		return changed ? next : entries;
	}, []);

	const hydrateConversationToolsIfNeeded = useCallback(
		(entries: ConversationToolStateEntry[]) => {
			const inFlight = hydratingConversationToolKeysRef.current;
			const cache = conversationToolDefsCacheRef.current;

			const missing = entries.filter(entry => {
				const cacheKey = conversationToolHydrationKey(entry);
				return !entry.toolDefinition && !cache.has(cacheKey) && !inFlight.has(cacheKey);
			});

			if (!missing.length) return;

			const requestedKeys = new Set<string>();
			for (const entry of missing) {
				const cacheKey = conversationToolHydrationKey(entry);
				inFlight.add(cacheKey);
				requestedKeys.add(cacheKey);
			}

			void Promise.all(
				missing.map(async entry => {
					const cacheKey = conversationToolHydrationKey(entry);
					try {
						const def = await toolStoreAPI.getTool(
							entry.toolStoreChoice.bundleID,
							entry.toolStoreChoice.toolSlug,
							entry.toolStoreChoice.toolVersion
						);
						return def ? { cacheKey, def } : null;
					} catch {
						return null;
					}
				})
			)
				.then(results => {
					if (!isMountedRef.current) return;

					let loadedAny = false;
					for (const result of results) {
						if (!result) continue;
						cache.set(result.cacheKey, result.def);
						loadedAny = true;
					}

					if (!loadedAny) return;

					// Use the full setter so primeConversationToolsFromCache runs
					// and any further missing definitions are queued for hydration.
					setConversationToolsStateRaw(prev => {
						const next = primeConversationToolsFromCache(prev);

						conversationToolsStateRef.current = next;

						return next;
					});
				})
				.finally(() => {
					for (const key of requestedKeys) {
						inFlight.delete(key);
					}
				});
		},
		[primeConversationToolsFromCache]
	);

	const setConversationToolsState = useCallback<Dispatch<SetStateAction<ConversationToolStateEntry[]>>>(
		update => {
			const prev = conversationToolsStateRef.current;
			const requested = resolveStateUpdate(update, prev);
			const next = primeConversationToolsFromCache(requested);

			conversationToolsStateRef.current = next;

			setConversationToolsStateRaw(next);
			hydrateConversationToolsIfNeeded(next);
			resumeAutoExecBatch();
		},
		[primeConversationToolsFromCache, hydrateConversationToolsIfNeeded, resumeAutoExecBatch]
	);

	const recomputeAttachedToolArgsBlocked = useCallback(() => {
		const toolEntries = getAttachedToolEntries(false);
		let nextBlocked = false;

		for (const entry of toolEntries) {
			const schema = entry.toolSnapshot?.userArgSchema;
			const status = computeToolUserArgsStatus(schema, entry.userArgSchemaInstance);
			if (status.hasSchema && !status.isSatisfied) {
				nextBlocked = true;
				break;
			}
		}

		attachedToolArgsBlockedRef.current = nextBlocked;
		setAttachedToolArgsBlocked(prev => (prev === nextBlocked ? prev : nextBlocked));
	}, [getAttachedToolEntries]);

	const handleAttachedToolsChanged = useCallback(() => {
		recomputeAttachedToolArgsBlocked();
		resumeAutoExecBatch();
	}, [recomputeAttachedToolArgsBlocked, resumeAutoExecBatch]);

	const handleOpenToolOutput = useCallback((output: UIToolOutput) => {
		setToolDetailsState({ kind: 'output', output });
	}, []);

	const handleOpenToolCallDetails = useCallback((call: UIToolCall) => {
		setToolDetailsState({ kind: 'call', call });
	}, []);

	const handleOpenConversationToolDetails = useCallback((entry: ConversationToolStateEntry) => {
		setToolDetailsState({ kind: 'choice', choice: entry.toolStoreChoice });
	}, []);

	const handleOpenAttachedToolDetails = useCallback((entry: AttachedToolEntry) => {
		const choice: ToolStoreChoice = {
			choiceID: entry.choiceID,
			bundleID: entry.bundleID,
			bundleSlug: entry.bundleSlug,
			toolSlug: entry.toolSlug,
			toolVersion: entry.toolVersion,
			displayName: entry.overrides?.displayName ?? entry.toolSnapshot?.displayName ?? entry.toolSlug,
			description: entry.overrides?.description ?? entry.toolSnapshot?.description ?? entry.toolSlug,
			toolID: entry.toolSnapshot?.id,
			toolType: entry.toolType,
			autoExecute: entry.autoExecute,
			userArgSchemaInstance: entry.userArgSchemaInstance,
		};
		setToolDetailsState({ kind: 'choice', choice });
	}, []);

	const applyConversationToolsFromChoices = useCallback(
		(tools: ToolStoreChoice[]) => {
			setConversationToolsState(initConversationToolsStateFromChoices(tools));
		},
		[setConversationToolsState]
	);

	const applyWebSearchFromChoices = useCallback(
		(tools: ToolStoreChoice[]) => {
			const ws = normalizeWebSearchChoiceTemplates(
				(tools ?? []).filter(t => t.toolType === ToolStoreChoiceType.WebSearch).map(webSearchTemplateFromChoice)
			);
			setWebSearchTemplates(ws);
		},
		[setWebSearchTemplates]
	);

	const loadToolCalls = useCallback(
		async (nextToolCalls: UIToolCall[]) => {
			clearAutoExecBatch();
			setToolCalls(nextToolCalls);

			const autoExecIds = getPendingAutoExecuteToolCalls(nextToolCalls).map(toolCall => toolCall.id);
			await enqueueAutoExecBatch(autoExecIds);
		},
		[clearAutoExecBatch, enqueueAutoExecBatch, setToolCalls]
	);

	const clearComposerToolsState = useCallback(() => {
		clearAutoExecBatch();
		toolCallAttemptKeyByIdRef.current.clear();

		updateToolRuntimeState(prev =>
			prev.toolCalls.length === 0 && prev.toolOutputs.length === 0 ? prev : { toolCalls: [], toolOutputs: [] }
		);
		setToolDetailsState(null);
		setToolArgsTarget(null);
		attachedToolArgsBlockedRef.current = false;
		toolArgsBlockedRef.current = false;
		setAttachedToolArgsBlocked(false);
	}, [clearAutoExecBatch, updateToolRuntimeState]);

	const getToolRuntimeSnapshot = useCallback((): ComposerToolRuntimeState => {
		return toolRuntimeStateRef.current;
	}, []);

	return {
		toolCalls,
		toolOutputs,
		setToolCalls,
		setToolOutputs,
		conversationToolsState,
		setConversationToolsState,
		webSearchTemplates,
		setWebSearchTemplates,
		toolDetailsState,
		setToolDetailsState,
		toolArgsTarget,
		setToolArgsTarget,
		toolArgsBlocked,
		hasPendingToolCalls,
		hasRunningToolCalls,
		recomputeAttachedToolArgsBlocked,
		runAllPendingToolCalls,
		handleRunSingleToolCall,
		handleDiscardToolCall,
		handleRemoveToolOutput,
		handleRetryErroredOutput,
		handleAttachedToolsChanged,
		applyConversationToolsFromChoices,
		applyWebSearchFromChoices,
		loadToolCalls,
		handleOpenToolOutput,
		handleOpenToolCallDetails,
		handleOpenConversationToolDetails,
		handleOpenAttachedToolDetails,
		clearComposerToolsState,
		getToolRuntimeSnapshot,
		resumeAutoExecBatch,
		autoExecState,
	};
}
