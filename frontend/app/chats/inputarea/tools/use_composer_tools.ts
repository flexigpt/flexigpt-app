import {
	type Dispatch,
	type RefObject,
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
	isSubmittingRef: RefObject<boolean>;
	templateBlocked: boolean;
	ensureSkillSession: () => Promise<string | null>;
	listActiveSkillRefs: (sid: string) => Promise<SkillRef[]>;
	setActiveSkillRefs: Dispatch<SetStateAction<SkillRef[]>>;
	getCurrentSkillSessionID: () => string | null;
	toolArgsEventTarget?: EventTarget | null;
	onAutoSubmitRequest?: () => void;
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

function hasPendingConversationToolDefinitions(
	entries: ConversationToolStateEntry[],
	cache: Map<string, Tool>
): boolean {
	for (const entry of entries) {
		if (!entry.enabled) continue;
		if (entry.toolDefinition) continue;
		if (!cache.has(conversationToolHydrationKey(entry))) {
			return true;
		}
	}

	return false;
}

export function useComposerTools({
	isBusy,
	isSubmittingRef,
	templateBlocked,
	ensureSkillSession,
	listActiveSkillRefs,
	setActiveSkillRefs,
	getCurrentSkillSessionID,
	toolArgsEventTarget,
	onAutoSubmitRequest,
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

	// eslint-disable-next-line @typescript-eslint/no-unnecessary-type-arguments
	const conversationToolDefsCacheRef = useRef<Map<string, Tool>>(new Map());

	const hydratingConversationToolKeysRef = useRef(new Set());

	// Arg-blocking state, split by attached-vs-conversation tools.
	const [attachedToolArgsBlocked, setAttachedToolArgsBlocked] = useState(false);
	const attachedToolArgsBlockedRef = useRef(false);
	const isBusyRef = useRef(isBusy);
	const externalAutoExecuteBlockedRef = useRef(externalAutoExecuteBlocked);
	const onAutoSubmitRequestRef = useRef(onAutoSubmitRequest ?? null);
	const templateBlockedRef = useRef(templateBlocked);
	const [webSearchTemplates, setWebSearchTemplatesRaw] = useState<WebSearchChoiceTemplate[]>([]);
	const webSearchTemplatesRef = useRef<WebSearchChoiceTemplate[]>([]);

	const conversationToolArgsBlocked = useMemo(
		() => getConversationToolArgsBlocked(conversationToolsState),
		[conversationToolsState]
	);

	const toolArgsBlocked = attachedToolArgsBlocked || conversationToolArgsBlocked;
	const toolArgsBlockedRef = useRef(toolArgsBlocked);

	const lastAutoExecuteAttemptKeyRef = useRef<string | null>(null);
	const autoExecuteCheckTimeoutRef = useRef<number | null>(null);
	const autoExecuteInFlightRef = useRef(false);
	const autoExecuteRequestedRef = useRef(false);
	const runAutoExecutePendingToolCallsCheckRef = useRef<() => void>(() => {});

	useEffect(() => {
		toolRuntimeStateRef.current = toolRuntimeState;
	}, [toolRuntimeState]);

	useLayoutEffect(() => {
		onAutoSubmitRequestRef.current = onAutoSubmitRequest ?? null;
	}, [onAutoSubmitRequest]);

	const kickAutoExecutePendingToolCalls = useCallback(() => {
		if (!autoExecuteRequestedRef.current) return;
		if (
			isBusyRef.current ||
			isSubmittingRef.current ||
			templateBlockedRef.current ||
			toolArgsBlockedRef.current ||
			externalAutoExecuteBlockedRef.current
		) {
			return;
		}
		if (autoExecuteCheckTimeoutRef.current != null) return;
		autoExecuteCheckTimeoutRef.current = window.setTimeout(() => {
			autoExecuteCheckTimeoutRef.current = null;
			runAutoExecutePendingToolCallsCheckRef.current();
		}, 0);
	}, [isSubmittingRef]);

	const syncAutoExecutePendingToolCallsRequest = useCallback(
		(nextToolCalls: UIToolCall[]) => {
			const pendingAutoExecuteCalls = getPendingAutoExecuteToolCalls(nextToolCalls);

			if (pendingAutoExecuteCalls.length === 0) {
				autoExecuteRequestedRef.current = false;
				lastAutoExecuteAttemptKeyRef.current = null;
				return;
			}

			autoExecuteRequestedRef.current = true;
			kickAutoExecutePendingToolCalls();
		},
		[kickAutoExecutePendingToolCalls]
	);

	const updateToolRuntimeState = useCallback(
		(update: SetStateAction<ComposerToolRuntimeState>) => {
			const prev = toolRuntimeStateRef.current;
			const next = resolveStateUpdate(update, prev);
			toolRuntimeStateRef.current = next;

			if (next.toolCalls !== prev.toolCalls) {
				syncAutoExecutePendingToolCallsRequest(next.toolCalls);
			}

			setToolRuntimeStateRaw(current => (current === next ? current : next));
		},
		[syncAutoExecutePendingToolCallsRequest]
	);

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

	const setWebSearchTemplates = useCallback<Dispatch<SetStateAction<WebSearchChoiceTemplate[]>>>(
		update => {
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
			if (toolRuntimeStateRef.current.toolCalls.length > 0) {
				kickAutoExecutePendingToolCalls();
			}
		},
		[kickAutoExecutePendingToolCalls]
	);

	const hasPendingToolCalls = useMemo(() => toolCalls.some(c => c.status === 'pending'), [toolCalls]);
	const hasRunningToolCalls = useMemo(() => toolCalls.some(c => c.status === 'running'), [toolCalls]);

	useEffect(() => {
		isBusyRef.current = isBusy;
		if (!isBusy) {
			kickAutoExecutePendingToolCalls();
		}
	}, [isBusy, kickAutoExecutePendingToolCalls]);

	useEffect(() => {
		templateBlockedRef.current = templateBlocked;
		if (!templateBlocked) {
			kickAutoExecutePendingToolCalls();
		}
	}, [templateBlocked, kickAutoExecutePendingToolCalls]);

	useEffect(() => {
		toolArgsBlockedRef.current = toolArgsBlocked;
	}, [toolArgsBlocked]);

	useEffect(() => {
		externalAutoExecuteBlockedRef.current = externalAutoExecuteBlocked;
		if (!externalAutoExecuteBlocked) {
			kickAutoExecutePendingToolCalls();
		}
	}, [externalAutoExecuteBlocked, kickAutoExecutePendingToolCalls]);

	useEffect(() => {
		return () => {
			if (autoExecuteCheckTimeoutRef.current != null) {
				window.clearTimeout(autoExecuteCheckTimeoutRef.current);
			}
		};
	}, []);

	const isSkillsToolName = useCallback((name: string | undefined): boolean => {
		const n = (name ?? '').trim();
		return n.startsWith('skills.');
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

				// Mark as running (allow retry after failure by overwriting previous status).
				setToolCalls(prev =>
					prev.map(c => (c.id === toolCall.id ? { ...c, status: 'running', errorMessage: undefined } : c))
				);

				try {
					const resp = await skillStoreAPI.invokeSkillTool(sid, toolCall.name, args);

					const isError = !!resp.isError;
					const errorMessage =
						resp.errorMessage ||
						(isError ? 'Skill tool reported an error. Inspect the output for details.' : undefined);

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
						toolCalls: prev.toolCalls.filter(c => c.id !== toolCall.id),
						toolOutputs: [...prev.toolOutputs, output],
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
					const msg = (err as Error)?.message || 'Skill tool invocation failed.';
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
			ensureSkillSession,
			getCurrentSkillSessionID,
			isSkillsToolName,
			listActiveSkillRefs,
			setActiveSkillRefs,
			setToolCalls,
			updateToolRuntimeState,
		]
	);

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
			const chip = toolRuntimeStateRef.current.toolCalls.find(
				c => c.id === id && (c.status === 'pending' || c.status === 'failed')
			);
			if (!chip) return;
			await runToolCallInternal(chip);
		},
		[runToolCallInternal]
	);

	const handleDiscardToolCall = useCallback(
		(id: string) => {
			setToolCalls(prev => {
				const next = prev.filter(c => c.id !== id);
				return next.length === prev.length ? prev : next;
			});
		},
		[setToolCalls]
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
		},
		[isSkillsToolName, updateToolRuntimeState]
	);

	const runAutoExecutePendingToolCallsCheck = useCallback(() => {
		if (autoExecuteInFlightRef.current) return;

		const currentToolCalls = toolRuntimeStateRef.current.toolCalls;
		const pendingAutoExecuteCalls = getPendingAutoExecuteToolCalls(currentToolCalls);

		if (pendingAutoExecuteCalls.length === 0) {
			autoExecuteRequestedRef.current = false;
			lastAutoExecuteAttemptKeyRef.current = null;
			return;
		}

		const hasRunningCalls = currentToolCalls.some(toolCall => toolCall.status === 'running');
		if (
			isBusyRef.current ||
			isSubmittingRef.current ||
			hasRunningCalls ||
			templateBlockedRef.current ||
			externalAutoExecuteBlockedRef.current ||
			toolArgsBlockedRef.current
		) {
			return;
		}

		const nextKey = pendingAutoExecuteCalls.map(toolCall => toolCall.id).join('|');
		if (nextKey && lastAutoExecuteAttemptKeyRef.current === nextKey) {
			return;
		}

		lastAutoExecuteAttemptKeyRef.current = nextKey;
		autoExecuteInFlightRef.current = true;

		void (async () => {
			try {
				await drainPendingToolCalls(true);

				const postRunToolCalls = toolRuntimeStateRef.current.toolCalls;
				const remainingPendingRunnable = getPendingRunnableToolCalls(postRunToolCalls);
				const hasRunningRunnable = hasRunningRunnableToolCalls(postRunToolCalls);
				const hasFailedRunnable = hasFailedRunnableToolCalls(postRunToolCalls);
				if (remainingPendingRunnable.length > 0 || hasRunningRunnable) return;

				// Mixed case:
				// - auto-exec calls have already run
				// - manual calls are still pending
				// => stop here and let the user decide what to run/send next.
				if (remainingPendingRunnable.length > 0) return;

				// If any auto-exec invocation failed at runtime, keep the failed chip
				// in the composer and do not auto-send.
				if (hasFailedRunnable || externalAutoExecuteBlockedRef.current) return;

				// Request the parent submit path; with no pending calls left, this becomes a plain send.
				onAutoSubmitRequestRef.current?.();
			} finally {
				autoExecuteInFlightRef.current = false;

				const remainingAutoExecuteKey =
					getPendingAutoExecuteToolCalls(toolRuntimeStateRef.current.toolCalls)
						.map(toolCall => toolCall.id)
						.join('|') || null;

				autoExecuteRequestedRef.current = !!remainingAutoExecuteKey;
				lastAutoExecuteAttemptKeyRef.current = remainingAutoExecuteKey;

				if (remainingAutoExecuteKey) {
					kickAutoExecutePendingToolCalls();
				}
			}
		})();
	}, [drainPendingToolCalls, isSubmittingRef, kickAutoExecutePendingToolCalls]);

	useLayoutEffect(() => {
		runAutoExecutePendingToolCallsCheckRef.current = runAutoExecutePendingToolCallsCheck;
	}, [runAutoExecutePendingToolCallsCheck]);

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
						const nextConversationToolArgsBlocked = getConversationToolArgsBlocked(next);
						const nextConversationToolDefsPending = hasPendingConversationToolDefinitions(next, cache);

						conversationToolsStateRef.current = next;
						toolArgsBlockedRef.current =
							attachedToolArgsBlockedRef.current || nextConversationToolArgsBlocked || nextConversationToolDefsPending;
						return next;
					});

					// Re-check auto-execute now that definitions have loaded.
					// This is critical: pending tool calls may have been blocked
					// by while definitions were in flight.
					if (toolRuntimeStateRef.current.toolCalls.length > 0) {
						kickAutoExecutePendingToolCalls();
					}
				})
				.finally(() => {
					for (const key of requestedKeys) {
						inFlight.delete(key);
					}
					if (isMountedRef.current) {
						// When the last hydration task finishes, re-check auto-execute.
						// The earlier timeout-based check may have bailed because definitions
						// were still loading.
						if (toolRuntimeStateRef.current.toolCalls.length > 0) {
							kickAutoExecutePendingToolCalls();
						}
					}
				});
		},
		[kickAutoExecutePendingToolCalls, primeConversationToolsFromCache]
	);

	const setConversationToolsState = useCallback<Dispatch<SetStateAction<ConversationToolStateEntry[]>>>(
		update => {
			const prev = conversationToolsStateRef.current;
			const requested = resolveStateUpdate(update, prev);
			const next = primeConversationToolsFromCache(requested);
			const nextConversationToolArgsBlocked = getConversationToolArgsBlocked(next);
			const nextConversationToolDefsPending = hasPendingConversationToolDefinitions(
				next,
				conversationToolDefsCacheRef.current
			);

			conversationToolsStateRef.current = next;
			toolArgsBlockedRef.current =
				attachedToolArgsBlockedRef.current || nextConversationToolArgsBlocked || nextConversationToolDefsPending;

			setConversationToolsStateRaw(next);
			hydrateConversationToolsIfNeeded(next);
			if (toolRuntimeStateRef.current.toolCalls.length > 0) {
				kickAutoExecutePendingToolCalls();
			}
		},
		[hydrateConversationToolsIfNeeded, kickAutoExecutePendingToolCalls, primeConversationToolsFromCache]
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
		toolArgsBlockedRef.current =
			nextBlocked ||
			getConversationToolArgsBlocked(conversationToolsStateRef.current) ||
			hasPendingConversationToolDefinitions(conversationToolsStateRef.current, conversationToolDefsCacheRef.current);
		setAttachedToolArgsBlocked(prev => (prev === nextBlocked ? prev : nextBlocked));
	}, [getAttachedToolEntries]);

	const handleAttachedToolsChanged = useCallback(() => {
		recomputeAttachedToolArgsBlocked();
		if (toolRuntimeStateRef.current.toolCalls.length > 0) {
			kickAutoExecutePendingToolCalls();
		}
	}, [kickAutoExecutePendingToolCalls, recomputeAttachedToolArgsBlocked]);

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
		(nextToolCalls: UIToolCall[]) => {
			setToolCalls(nextToolCalls);
		},
		[setToolCalls]
	);

	const clearComposerToolsState = useCallback(() => {
		if (autoExecuteCheckTimeoutRef.current != null) {
			window.clearTimeout(autoExecuteCheckTimeoutRef.current);
			autoExecuteCheckTimeoutRef.current = null;
		}
		autoExecuteRequestedRef.current = false;
		lastAutoExecuteAttemptKeyRef.current = null;

		updateToolRuntimeState(prev =>
			prev.toolCalls.length === 0 && prev.toolOutputs.length === 0 ? prev : { toolCalls: [], toolOutputs: [] }
		);
		setToolDetailsState(null);
		setToolArgsTarget(null);
		attachedToolArgsBlockedRef.current = false;
		toolArgsBlockedRef.current = false;
		setAttachedToolArgsBlocked(false);
	}, [updateToolRuntimeState]);

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
	};
}
