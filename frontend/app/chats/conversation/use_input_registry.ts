import { useCallback, useEffect, useRef } from 'react';

import type { AttachmentsDroppedPayload } from '@/spec/attachment';
import type { Conversation } from '@/spec/conversation';

import type { ComposerBoxHandle } from '@/chats/composer/composer_box';
import {
	deriveHydratedLastAssistantToolCalls,
	deriveRestorableConversationContextFromMessages,
} from '@/chats/conversation/hydration_helper';
import type { ChatWorkflowStarter } from '@/chats/conversation/starter_intent';

type PendingDrop = {
	tabId: string;
	payload: AttachmentsDroppedPayload;
};

type UseInputRegistryArgs = {
	tabExists: (tabId: string) => boolean;
};

export function useInputRegistry({ tabExists }: UseInputRegistryArgs) {
	const inputRefs = useRef(new Map<string, ComposerBoxHandle | null>());
	const pendingDropsRef = useRef<PendingDrop[]>([]);
	const pendingWorkflowStartersRef = useRef(new Map<string, ChatWorkflowStarter>());
	const inputRefCallbacksRef = useRef(new Map<string, (inst: ComposerBoxHandle | null) => void>());
	const pendingInputFlushTimerRef = useRef<number | null>(null);
	const flushPendingDropsRef = useRef<() => void>(() => {});
	const flushPendingWorkflowStartersRef = useRef<() => void>(() => {});

	const tryApplyDropToTab = useCallback((tabId: string, payload: AttachmentsDroppedPayload): boolean => {
		const input = inputRefs.current.get(tabId);
		if (!input) return false;

		input.applyAttachmentsDrop(payload);
		return true;
	}, []);

	const queuePendingDrop = useCallback((tabId: string, payload: AttachmentsDroppedPayload) => {
		pendingDropsRef.current.push({ tabId, payload });
	}, []);

	const flushPendingDrops = useCallback(() => {
		const pending = pendingDropsRef.current;
		if (!pending || pending.length === 0) return;

		const remaining: PendingDrop[] = [];

		for (const item of pending) {
			if (!tabExists(item.tabId)) continue;
			if (!tryApplyDropToTab(item.tabId, item.payload)) {
				remaining.push(item);
			}
		}

		pendingDropsRef.current = remaining;
	}, [tabExists, tryApplyDropToTab]);

	const tryApplyWorkflowStarterToTab = useCallback(
		async (tabId: string, starter: ChatWorkflowStarter): Promise<boolean> => {
			const input = inputRefs.current.get(tabId);
			if (!input) return false;

			await input.loadWorkflowStarter(starter);
			return true;
		},
		[]
	);

	const flushPendingWorkflowStarters = useCallback(() => {
		const pending = pendingWorkflowStartersRef.current;
		if (pending.size === 0) return;

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

	// eslint-disable-next-line react-hooks/refs
	flushPendingDropsRef.current = flushPendingDrops;
	// eslint-disable-next-line react-hooks/refs
	flushPendingWorkflowStartersRef.current = flushPendingWorkflowStarters;

	const schedulePendingInputFlush = useCallback(() => {
		if (pendingInputFlushTimerRef.current !== null) return;

		pendingInputFlushTimerRef.current = window.setTimeout(() => {
			pendingInputFlushTimerRef.current = null;
			flushPendingDropsRef.current();
			flushPendingWorkflowStartersRef.current();
		}, 0);
	}, []);

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
			if (!tabExists(tabId)) return false;
			if (await tryApplyWorkflowStarterToTab(tabId, starter)) return true;

			pendingWorkflowStartersRef.current.set(tabId, starter);
			return true;
		},
		[tabExists, tryApplyWorkflowStarterToTab]
	);

	const setInputRef = useCallback(
		(tabId: string) => {
			let callback = inputRefCallbacksRef.current.get(tabId);
			if (callback) return callback;

			callback = (inst: ComposerBoxHandle | null) => {
				const previous = inputRefs.current.get(tabId) ?? null;
				if (previous === inst) return;
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

	const syncComposerFromConversation = useCallback((tabId: string, conversation: Conversation) => {
		const input = inputRefs.current.get(tabId);
		if (!input) return;

		input.resetEditor();
		input.restoreConversationContext(deriveRestorableConversationContextFromMessages(conversation.messages));
		// Match the normal live-chat behavior: if the last hydrated message is an
		// assistant tool-call turn, restore runnable tool calls into the composer.
		// Do not auto-execute restored calls; they remain manual until the user acts.
		const hydratedToolCalls = deriveHydratedLastAssistantToolCalls(conversation);
		if (hydratedToolCalls.length > 0) {
			input.loadToolCalls(hydratedToolCalls);
			input.finishAssistantTurn({ loadedRunnableToolCallCount: hydratedToolCalls.length });
		}
	}, []);

	const resetComposerForNewConversation = useCallback(async (tabId: string) => {
		const input = inputRefs.current.get(tabId);
		if (!input) return;
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
	};
}
