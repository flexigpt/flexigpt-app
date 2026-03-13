import {
	type Dispatch,
	type RefObject,
	type SetStateAction,
	useCallback,
	useEffect,
	useMemo,
	useRef,
	useState,
} from 'react';

import type { PlateEditor } from 'platejs/react';

import type { UIToolCall, UIToolOutput } from '@/spec/inference';
import type { SkillRef } from '@/spec/skill';
import { type Tool, type ToolStoreChoice, ToolStoreChoiceType } from '@/spec/tool';

import { resolveStateUpdate } from '@/lib/hook_utils';
import { ensureMakeID, getUUIDv7 } from '@/lib/uuid_utils';

import { skillStoreAPI, toolRuntimeAPI, toolStoreAPI } from '@/apis/baseapi';

import { useOpenToolArgs } from '@/chats/events/open_attached_toolargs';
import {
	type ConversationToolStateEntry,
	initConversationToolsStateFromChoices,
} from '@/chats/tools/conversation_tool_utils';
import type { ToolDetailsState } from '@/chats/tools/tool_details_modal';
import {
	computeToolUserArgsStatus,
	formatToolOutputSummary,
	getToolNodesWithPath,
	type ToolSelectionElementNode,
} from '@/chats/tools/tool_editor_utils';
import type { ToolArgsTarget } from '@/chats/tools/tool_user_args_modal';
import { type WebSearchChoiceTemplate, webSearchTemplateFromChoice } from '@/chats/tools/websearch_utils';

interface ComposerToolRuntimeState {
	toolCalls: UIToolCall[];
	toolOutputs: UIToolOutput[];
}

interface UseComposerToolsArgs {
	editor: PlateEditor;
	isBusy: boolean;
	isSubmittingRef: RefObject<boolean>;
	templateBlocked: boolean;
	submitPendingToolsAndSendRef: RefObject<(() => void | Promise<void>) | null>;
	ensureSkillSession: () => Promise<string | null>;
	listActiveSkillRefs: (sid: string) => Promise<SkillRef[]>;
	setActiveSkillRefs: Dispatch<SetStateAction<SkillRef[]>>;
	getCurrentSkillSessionID: () => string | null;
	skillSessionID: string | null;
}

interface UseComposerToolsResult {
	toolCalls: UIToolCall[];
	toolOutputs: UIToolOutput[];
	setToolCalls: Dispatch<SetStateAction<UIToolCall[]>>;
	setToolOutputs: Dispatch<SetStateAction<UIToolOutput[]>>;
	conversationToolsState: ConversationToolStateEntry[];
	setConversationToolsState: Dispatch<SetStateAction<ConversationToolStateEntry[]>>;
	setConversationToolsStateAndMaybeAutoExecute: Dispatch<SetStateAction<ConversationToolStateEntry[]>>;
	webSearchTemplates: WebSearchChoiceTemplate[];
	setWebSearchTemplates: Dispatch<SetStateAction<WebSearchChoiceTemplate[]>>;
	setWebSearchTemplatesAndMaybeAutoExecute: Dispatch<SetStateAction<WebSearchChoiceTemplate[]>>;
	toolDetailsState: ToolDetailsState;
	setToolDetailsState: Dispatch<SetStateAction<ToolDetailsState>>;
	toolArgsTarget: ToolArgsTarget | null;
	setToolArgsTarget: Dispatch<SetStateAction<ToolArgsTarget | null>>;
	toolsDefLoading: boolean;
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
	handleOpenAttachedToolDetails: (node: ToolSelectionElementNode) => void;
	clearComposerToolsState: () => void;
}

const conversationToolHydrationKey = (entry: ConversationToolStateEntry): string => {
	return `${entry.toolStoreChoice.bundleID}::${entry.toolStoreChoice.toolSlug}::${entry.toolStoreChoice.toolVersion}`;
};

export function useComposerTools({
	editor,
	isBusy,
	isSubmittingRef,
	templateBlocked,
	submitPendingToolsAndSendRef,
	ensureSkillSession,
	listActiveSkillRefs,
	setActiveSkillRefs,
	getCurrentSkillSessionID,
	skillSessionID,
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
	const updateToolRuntimeState = useCallback((update: SetStateAction<ComposerToolRuntimeState>) => {
		setToolRuntimeStateRaw(prev => {
			const next = resolveStateUpdate(update, prev);
			toolRuntimeStateRef.current = next;
			return next === prev ? prev : next;
		});
	}, []);
	useEffect(() => {
		toolRuntimeStateRef.current = toolRuntimeState;
	}, [toolRuntimeState]);

	const toolCalls = toolRuntimeState.toolCalls;
	const toolOutputs = toolRuntimeState.toolOutputs;

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

	const [toolDetailsState, setToolDetailsState] = useState<ToolDetailsState>(null);
	const [toolArgsTarget, setToolArgsTarget] = useState<ToolArgsTarget | null>(null);

	useOpenToolArgs(target => {
		setToolArgsTarget(target);
	});

	const [conversationToolsState, setConversationToolsStateRaw] = useState<ConversationToolStateEntry[]>([]);
	const conversationToolsStateRef = useRef<ConversationToolStateEntry[]>([]);
	useEffect(() => {
		conversationToolsStateRef.current = conversationToolsState;
	}, [conversationToolsState]);
	const conversationToolDefsCacheRef = useRef<Map<string, Tool>>(new Map());

	// Count of in-flight conversation tool-definition hydration tasks.
	// Used to gate sending while schemas are still loading.
	const [toolsHydratingCount, setToolsHydratingCount] = useState(0);
	const hydratingConversationToolKeysRef = useRef<Set<string>>(new Set());

	const toolsDefLoading = toolsHydratingCount > 0;

	// Arg-blocking state, split by attached-vs-conversation tools.
	const [attachedToolArgsBlocked, setAttachedToolArgsBlocked] = useState(false);
	const conversationToolArgsBlocked = useMemo(() => {
		for (const entry of conversationToolsState) {
			if (!entry.enabled) continue;
			const status = entry.argStatus;
			if (status?.hasSchema && !status.isSatisfied) {
				return true;
			}
		}
		return false;
	}, [conversationToolsState]);
	const toolArgsBlocked = attachedToolArgsBlocked || conversationToolArgsBlocked;

	const [webSearchTemplates, setWebSearchTemplatesRaw] = useState<WebSearchChoiceTemplate[]>([]);
	const setWebSearchTemplates = useCallback<Dispatch<SetStateAction<WebSearchChoiceTemplate[]>>>(update => {
		setWebSearchTemplatesRaw(update);
	}, []);

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

			setToolsHydratingCount(c => c + 1);

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
					if (isMountedRef.current) {
						setToolsHydratingCount(c => Math.max(0, c - 1));
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
		},
		[hydrateConversationToolsIfNeeded, primeConversationToolsFromCache]
	);

	const hasPendingToolCalls = useMemo(() => toolCalls.some(c => c.status === 'pending'), [toolCalls]);
	const hasRunningToolCalls = useMemo(() => toolCalls.some(c => c.status === 'running'), [toolCalls]);

	const isBusyRef = useRef(isBusy);
	const templateBlockedRef = useRef(templateBlocked);
	const toolArgsBlockedRef = useRef(toolArgsBlocked);
	const toolsDefLoadingRef = useRef(toolsDefLoading);

	useEffect(() => {
		isBusyRef.current = isBusy;
	}, [isBusy]);

	useEffect(() => {
		templateBlockedRef.current = templateBlocked;
	}, [templateBlocked]);

	useEffect(() => {
		toolArgsBlockedRef.current = toolArgsBlocked;
	}, [toolArgsBlocked]);

	useEffect(() => {
		toolsDefLoadingRef.current = toolsDefLoading;
	}, [toolsDefLoading]);

	const lastAutoExecuteAttemptKeyRef = useRef<string | null>(null);
	const autoExecuteCheckTimeoutRef = useRef<number | null>(null);
	const autoExecuteRetryBudgetRef = useRef(0);

	const runAutoExecutePendingToolCallsCheck = useCallback(() => {
		function check(): void {
			const currentToolCalls = toolRuntimeStateRef.current.toolCalls;
			const pendingRunnable = currentToolCalls.filter(
				c =>
					c.status === 'pending' && (c.type === ToolStoreChoiceType.Function || c.type === ToolStoreChoiceType.Custom)
			);

			if (pendingRunnable.length === 0) {
				lastAutoExecuteAttemptKeyRef.current = null;
				autoExecuteRetryBudgetRef.current = 0;
				return;
			}

			const shouldAutoExecute = pendingRunnable.every(c => c.toolStoreChoice?.autoExecute);
			if (!shouldAutoExecute) {
				autoExecuteRetryBudgetRef.current = 0;
				return;
			}

			const hasRunningCalls = currentToolCalls.some(c => c.status === 'running');
			if (
				isBusyRef.current ||
				isSubmittingRef.current ||
				hasRunningCalls ||
				templateBlockedRef.current ||
				toolArgsBlockedRef.current ||
				toolsDefLoadingRef.current
			) {
				if (autoExecuteRetryBudgetRef.current > 0) {
					autoExecuteRetryBudgetRef.current -= 1;
					autoExecuteCheckTimeoutRef.current = window.setTimeout(() => {
						autoExecuteCheckTimeoutRef.current = null;
						check();
					}, 80);
				}
				return;
			}

			const nextKey = pendingRunnable.map(c => c.id).join('|');
			if (nextKey && lastAutoExecuteAttemptKeyRef.current === nextKey) {
				autoExecuteRetryBudgetRef.current = 0;
				return;
			}

			lastAutoExecuteAttemptKeyRef.current = nextKey;
			autoExecuteRetryBudgetRef.current = 0;
			void submitPendingToolsAndSendRef.current?.();
		}

		check();
	}, [isSubmittingRef, submitPendingToolsAndSendRef]);

	const kickAutoExecutePendingToolCalls = useCallback(
		(retryBudget = 0) => {
			autoExecuteRetryBudgetRef.current = Math.max(autoExecuteRetryBudgetRef.current, retryBudget);
			if (autoExecuteCheckTimeoutRef.current != null) return;
			autoExecuteCheckTimeoutRef.current = window.setTimeout(() => {
				autoExecuteCheckTimeoutRef.current = null;
				runAutoExecutePendingToolCallsCheck();
			}, 0);
		},
		[runAutoExecutePendingToolCallsCheck]
	);

	useEffect(() => {
		return () => {
			if (autoExecuteCheckTimeoutRef.current != null) {
				window.clearTimeout(autoExecuteCheckTimeoutRef.current);
			}
		};
	}, []);

	const setConversationToolsStateAndMaybeAutoExecute = useCallback<
		Dispatch<SetStateAction<ConversationToolStateEntry[]>>
	>(
		update => {
			setConversationToolsState(update);
			if (toolRuntimeStateRef.current.toolCalls.length > 0) {
				kickAutoExecutePendingToolCalls(1);
			}
		},
		[kickAutoExecutePendingToolCalls, setConversationToolsState]
	);

	const setWebSearchTemplatesAndMaybeAutoExecute = useCallback<Dispatch<SetStateAction<WebSearchChoiceTemplate[]>>>(
		update => {
			setWebSearchTemplates(update);
			if (toolRuntimeStateRef.current.toolCalls.length > 0) {
				kickAutoExecutePendingToolCalls(1);
			}
		},
		[kickAutoExecutePendingToolCalls, setWebSearchTemplates]
	);

	const recomputeAttachedToolArgsBlocked = useCallback(() => {
		const toolEntries = getToolNodesWithPath(editor, false);
		let nextBlocked = false;

		for (const [node] of toolEntries) {
			const schema = node.toolSnapshot?.userArgSchema;
			const status = computeToolUserArgsStatus(schema, node.userArgSchemaInstance);
			if (status.hasSchema && !status.isSatisfied) {
				nextBlocked = true;
				break;
			}
		}

		setAttachedToolArgsBlocked(prev => (prev === nextBlocked ? prev : nextBlocked));
	}, [editor]);

	const handleAttachedToolsChanged = useCallback(() => {
		recomputeAttachedToolArgsBlocked();
		if (toolRuntimeStateRef.current.toolCalls.length > 0) {
			kickAutoExecutePendingToolCalls(1);
		}
	}, [kickAutoExecutePendingToolCalls, recomputeAttachedToolArgsBlocked]);

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

	/**
	 * Run all currently pending tool calls (in sequence) and return the
	 * UIToolOutput objects produced in this pass.
	 */
	const runAllPendingToolCalls = useCallback(async (): Promise<UIToolOutput[]> => {
		const pending = toolCalls.filter(c => c.status === 'pending');
		if (pending.length === 0) return [];

		const produced: UIToolOutput[] = [];
		for (const chip of pending) {
			if (chip.type === ToolStoreChoiceType.Function || chip.type === ToolStoreChoiceType.Custom) {
				const out = await runToolCallInternal(chip);
				if (out) produced.push(out);
			}
		}
		return produced;
	}, [toolCalls, runToolCallInternal]);

	const handleRunSingleToolCall = useCallback(
		async (id: string) => {
			const chip = toolCalls.find(c => c.id === id && (c.status === 'pending' || c.status === 'failed'));
			if (!chip) return;
			await runToolCallInternal(chip);
		},
		[toolCalls, runToolCallInternal]
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
				// Skills retry requires an active session.
				if (!skillSessionID) return;
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
			kickAutoExecutePendingToolCalls(20);
		},
		[isSkillsToolName, kickAutoExecutePendingToolCalls, skillSessionID, updateToolRuntimeState]
	);

	const handleOpenToolOutput = useCallback((output: UIToolOutput) => {
		setToolDetailsState({ kind: 'output', output });
	}, []);

	const handleOpenToolCallDetails = useCallback((call: UIToolCall) => {
		setToolDetailsState({ kind: 'call', call });
	}, []);

	const handleOpenConversationToolDetails = useCallback((entry: ConversationToolStateEntry) => {
		setToolDetailsState({ kind: 'choice', choice: entry.toolStoreChoice });
	}, []);

	const handleOpenAttachedToolDetails = useCallback((node: ToolSelectionElementNode) => {
		const choice: ToolStoreChoice = {
			choiceID: node.choiceID,
			bundleID: node.bundleID,
			bundleSlug: node.bundleSlug,
			toolSlug: node.toolSlug,
			toolVersion: node.toolVersion,
			displayName: node.overrides?.displayName ?? node.toolSnapshot?.displayName ?? node.toolSlug,
			description: node.overrides?.description ?? node.toolSnapshot?.description ?? node.toolSlug,
			toolID: node.toolSnapshot?.id,
			toolType: node.toolType,
			autoExecute: node.autoExecute,
			userArgSchemaInstance: node.userArgSchemaInstance,
		};
		setToolDetailsState({ kind: 'choice', choice });
	}, []);

	const applyConversationToolsFromChoices = useCallback(
		(tools: ToolStoreChoice[]) => {
			setConversationToolsStateAndMaybeAutoExecute(initConversationToolsStateFromChoices(tools));
		},
		[setConversationToolsStateAndMaybeAutoExecute]
	);

	const applyWebSearchFromChoices = useCallback(
		(tools: ToolStoreChoice[]) => {
			const ws = (tools ?? []).filter(t => t.toolType === ToolStoreChoiceType.WebSearch);
			setWebSearchTemplatesAndMaybeAutoExecute(ws.map(webSearchTemplateFromChoice));
		},
		[setWebSearchTemplatesAndMaybeAutoExecute]
	);

	const loadToolCalls = useCallback(
		(nextToolCalls: UIToolCall[]) => {
			setToolCalls(nextToolCalls);
			kickAutoExecutePendingToolCalls(20);
		},
		[kickAutoExecutePendingToolCalls, setToolCalls]
	);

	const clearComposerToolsState = useCallback(() => {
		updateToolRuntimeState(prev =>
			prev.toolCalls.length === 0 && prev.toolOutputs.length === 0 ? prev : { toolCalls: [], toolOutputs: [] }
		);
		setToolDetailsState(null);
		setToolArgsTarget(null);
		setAttachedToolArgsBlocked(false);
	}, [updateToolRuntimeState]);

	return {
		toolCalls,
		toolOutputs,
		setToolCalls,
		setToolOutputs,
		conversationToolsState,
		setConversationToolsState,
		setConversationToolsStateAndMaybeAutoExecute,
		webSearchTemplates,
		setWebSearchTemplates,
		setWebSearchTemplatesAndMaybeAutoExecute,
		toolDetailsState,
		setToolDetailsState,
		toolArgsTarget,
		setToolArgsTarget,
		toolsDefLoading,
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
	};
}
