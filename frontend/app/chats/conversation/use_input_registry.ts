import { useCallback, useEffect, useLayoutEffect, useRef } from 'react';

import type { AttachmentsDroppedPayload } from '@/spec/attachment';
import type { Conversation } from '@/spec/conversation';
import type { UIToolCall } from '@/spec/inference';

import type { ComposerBoxHandle } from '@/chats/composer/composer_box';
import type { AssistantTurnFinishedPayload } from '@/chats/composer/editor/editor_types';
import {
	deriveHydratedLastAssistantToolCalls,
	deriveRestorableConversationContextFromMessages,
} from '@/chats/conversation/hydration_helper';
import type { ChatWorkflowStarter } from '@/chats/conversation/starter_intent';

interface PendingDrop {
	tabId: string;
	payload: AttachmentsDroppedPayload;
}

interface PendingAssistantTurn {
	tabId: string;
	toolCalls: UIToolCall[];
	finishPayload: AssistantTurnFinishedPayload;
}

interface UseInputRegistryArgs {
	tabExists: (tabId: string) => boolean;
}

export function useInputRegistry({ tabExists }: UseInputRegistryArgs) {
	const inputRefs = useRef(new Map<string, ComposerBoxHandle | null>());
	const pendingDropsRef = useRef<PendingDrop[]>([]);
	const pendingAssistantTurnsRef = useRef(new Map<string, PendingAssistantTurn>());
	const pendingConversationContextsRef = useRef(new Map<string, Conversation>());
	const pendingWorkflowStartersRef = useRef(new Map<string, ChatWorkflowStarter>());
	const inputRefCallbacksRef = useRef(new Map<string, (inst: ComposerBoxHandle | null) => void>());
	const pendingInputFlushTimerRef = useRef<number | null>(null);
	const flushPendingDropsRef = useRef<() => void>(() => {});
	const flushPendingConversationContextsRef = useRef<() => void>(() => {});
	const flushPendingAssistantTurnsRef = useRef<() => void>(() => {});
	const flushPendingWorkflowStartersRef = useRef<() => void>(() => {});

	const tryApplyDropToTab = useCallback((tabId: string, payload: AttachmentsDroppedPayload): boolean => {
		const input = inputRefs.current.get(tabId);
		if (!input) {
			return false;
		}

		input.applyAttachmentsDrop(payload);
		return true;
	}, []);

	const queuePendingDrop = useCallback((tabId: string, payload: AttachmentsDroppedPayload) => {
		pendingDropsRef.current.push({ tabId, payload });
	}, []);

	const flushPendingDrops = useCallback(() => {
		const pending = pendingDropsRef.current;
		if (!pending || pending.length === 0) {
			return;
		}

		const remaining: PendingDrop[] = [];

		for (const item of pending) {
			if (!tabExists(item.tabId)) {
				continue;
			}
			if (!tryApplyDropToTab(item.tabId, item.payload)) {
				remaining.push(item);
			}
		}

		pendingDropsRef.current = remaining;
	}, [tabExists, tryApplyDropToTab]);

	const applyConversationContextToInput = useCallback((input: ComposerBoxHandle, conversation: Conversation) => {
		input.resetEditor();
		input.restoreConversationContext(deriveRestorableConversationContextFromMessages(conversation.messages));

		const hydratedToolCalls = deriveHydratedLastAssistantToolCalls(conversation);
		if (hydratedToolCalls.length > 0) {
			input.loadToolCalls(hydratedToolCalls);
			input.finishAssistantTurn({ loadedRunnableToolCallCount: hydratedToolCalls.length });
		}
	}, []);

	const tryApplyConversationContextToTab = useCallback(
		(tabId: string, conversation: Conversation): boolean => {
			const input = inputRefs.current.get(tabId);
			if (!input) {
				return false;
			}

			applyConversationContextToInput(input, conversation);
			return true;
		},
		[applyConversationContextToInput]
	);

	const flushPendingConversationContexts = useCallback(() => {
		for (const [tabId, conversation] of pendingConversationContextsRef.current.entries()) {
			if (!tabExists(tabId)) {
				pendingConversationContextsRef.current.delete(tabId);
				continue;
			}

			if (tryApplyConversationContextToTab(tabId, conversation)) {
				pendingConversationContextsRef.current.delete(tabId);
			}
		}
	}, [tabExists, tryApplyConversationContextToTab]);

	const tryApplyAssistantTurnToTab = useCallback((tabId: string, turn: PendingAssistantTurn): boolean => {
		const input = inputRefs.current.get(tabId);
		if (!input) {
			return false;
		}

		if (turn.toolCalls.length > 0) {
			input.loadToolCalls(turn.toolCalls);
		}
		input.finishAssistantTurn(turn.finishPayload);
		return true;
	}, []);

	const flushPendingAssistantTurns = useCallback(() => {
		const pending = pendingAssistantTurnsRef.current;
		if (pending.size === 0) {
			return;
		}

		for (const [tabId, turn] of pending.entries()) {
			if (!tabExists(tabId)) {
				pending.delete(tabId);
				continue;
			}

			if (tryApplyAssistantTurnToTab(tabId, turn)) {
				pending.delete(tabId);
			}
		}
	}, [tabExists, tryApplyAssistantTurnToTab]);

	const tryApplyWorkflowStarterToTab = useCallback(
		async (tabId: string, starter: ChatWorkflowStarter): Promise<boolean> => {
			const input = inputRefs.current.get(tabId);
			if (!input) {
				return false;
			}

			await input.loadWorkflowStarter(starter);
			return true;
		},
		[]
	);

	const flushPendingWorkflowStarters = useCallback(() => {
		const pending = pendingWorkflowStartersRef.current;
		if (pending.size === 0) {
			return;
		}

		for (const [tabId, starter] of pending.entries()) {
			if (!tabExists(tabId)) {
				pending.delete(tabId);
				continue;
			}

			void tryApplyWorkflowStarterToTab(tabId, starter)
				.then(applied => {
					if (applied && pendingWorkflowStartersRef.current.get(tabId) === starter) {
						pendingWorkflowStartersRef.current.delete(tabId);
					}
				})
				.catch((error: unknown) => {
					console.error('Failed to apply pending workflow starter:', error);
				});
		}
	}, [tabExists, tryApplyWorkflowStarterToTab]);

	useLayoutEffect(() => {
		flushPendingDropsRef.current = flushPendingDrops;
	}, [flushPendingDrops]);

	useLayoutEffect(() => {
		flushPendingConversationContextsRef.current = flushPendingConversationContexts;
	}, [flushPendingConversationContexts]);

	useLayoutEffect(() => {
		flushPendingAssistantTurnsRef.current = flushPendingAssistantTurns;
	}, [flushPendingAssistantTurns]);

	useLayoutEffect(() => {
		flushPendingWorkflowStartersRef.current = flushPendingWorkflowStarters;
	}, [flushPendingWorkflowStarters]);

	const schedulePendingInputFlush = useCallback(() => {
		if (pendingInputFlushTimerRef.current !== null) {
			return;
		}

		pendingInputFlushTimerRef.current = window.setTimeout(() => {
			pendingInputFlushTimerRef.current = null;
			flushPendingDropsRef.current();
			flushPendingConversationContextsRef.current();
			flushPendingAssistantTurnsRef.current();
			flushPendingWorkflowStartersRef.current();
		}, 0);
	}, []);

	const loadAssistantTurnForTab = useCallback(
		(tabId: string, toolCalls: UIToolCall[], finishPayload: AssistantTurnFinishedPayload): boolean => {
			if (!tabExists(tabId)) {
				return false;
			}

			const turn: PendingAssistantTurn = {
				tabId,
				toolCalls: [...toolCalls],
				finishPayload: { ...finishPayload },
			};

			if (tryApplyAssistantTurnToTab(tabId, turn)) {
				return true;
			}

			pendingAssistantTurnsRef.current.set(tabId, turn);
			schedulePendingInputFlush();
			return false;
		},
		[schedulePendingInputFlush, tabExists, tryApplyAssistantTurnToTab]
	);

	useEffect(() => {
		return () => {
			if (pendingInputFlushTimerRef.current !== null) {
				window.clearTimeout(pendingInputFlushTimerRef.current);
				pendingInputFlushTimerRef.current = null;
			}
		};
	}, []);

	const applyWorkflowStarterToComposer = useCallback(
		async (tabId: string, starter: ChatWorkflowStarter): Promise<boolean> => {
			if (!tabExists(tabId)) {
				return false;
			}
			if (await tryApplyWorkflowStarterToTab(tabId, starter)) {
				return true;
			}

			pendingWorkflowStartersRef.current.set(tabId, starter);
			return true;
		},
		[tabExists, tryApplyWorkflowStarterToTab]
	);

	const setInputRef = useCallback(
		(tabId: string) => {
			let callback = inputRefCallbacksRef.current.get(tabId);
			if (callback) {
				return callback;
			}

			callback = (inst: ComposerBoxHandle | null) => {
				const previous = inputRefs.current.get(tabId) ?? null;
				if (previous === inst) {
					return;
				}
				inputRefs.current.set(tabId, inst);

				if (inst) {
					schedulePendingInputFlush();
				}
			};
			inputRefCallbacksRef.current.set(tabId, callback);
			return callback;
		},
		[schedulePendingInputFlush]
	);

	const syncComposerFromConversation = useCallback(
		(tabId: string, conversation: Conversation) => {
			pendingAssistantTurnsRef.current.delete(tabId);
			if (tryApplyConversationContextToTab(tabId, conversation)) {
				pendingConversationContextsRef.current.delete(tabId);
				return;
			}

			pendingConversationContextsRef.current.set(tabId, conversation);
			schedulePendingInputFlush();
		},
		[schedulePendingInputFlush, tryApplyConversationContextToTab]
	);

	const resetComposerForNewConversation = useCallback(async (tabId: string) => {
		const input = inputRefs.current.get(tabId);
		pendingAssistantTurnsRef.current.delete(tabId);
		pendingConversationContextsRef.current.delete(tabId);
		if (!input) {
			return;
		}
		await input.resetForNewConversation();
	}, []);

	const focusInput = useCallback((tabId: string) => inputRefs.current.get(tabId)?.focus(), []);
	const openTemplateMenu = useCallback((tabId: string) => inputRefs.current.get(tabId)?.openTemplateMenu(), []);
	const openToolMenu = useCallback((tabId: string) => inputRefs.current.get(tabId)?.openToolMenu(), []);
	const openAttachmentMenu = useCallback((tabId: string) => inputRefs.current.get(tabId)?.openAttachmentMenu(), []);
	const openSystemPromptMenu = useCallback((tabId: string) => inputRefs.current.get(tabId)?.openSystemPromptMenu(), []);
	const openSkillsMenu = useCallback((tabId: string) => inputRefs.current.get(tabId)?.openSkillsMenu(), []);
	const openMCPMenu = useCallback((tabId: string) => inputRefs.current.get(tabId)?.openMCPMenu(), []);
	const requestStopResponse = useCallback((tabId: string) => inputRefs.current.get(tabId)?.requestStopResponse(), []);

	const disposeInputRuntime = useCallback((tabId: string) => {
		inputRefs.current.delete(tabId);
		pendingWorkflowStartersRef.current.delete(tabId);
		pendingConversationContextsRef.current.delete(tabId);
		pendingAssistantTurnsRef.current.delete(tabId);
		inputRefCallbacksRef.current.delete(tabId);
	}, []);

	return {
		inputRefs,
		setInputRef,
		tryApplyDropToTab,
		queuePendingDrop,
		flushPendingDrops,
		syncComposerFromConversation,
		resetComposerForNewConversation,
		focusInput,
		openTemplateMenu,
		openToolMenu,
		openAttachmentMenu,
		openSystemPromptMenu,
		openSkillsMenu,
		openMCPMenu,
		requestStopResponse,
		disposeInputRuntime,
		applyWorkflowStarterToComposer,
		loadAssistantTurnForTab,
	};
}
