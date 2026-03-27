import { useCallback, useRef } from 'react';

import type { AttachmentsDroppedPayload } from '@/spec/attachment';
import type { Conversation } from '@/spec/conversation';

import type { ComposerBoxHandle } from '@/chats/composer/composer_box';
import { deriveRestorableConversationContextFromMessages } from '@/chats/conversation/hydration_helper';

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

	const setInputRef = useCallback(
		(tabId: string) => {
			return (inst: ComposerBoxHandle | null) => {
				inputRefs.current.set(tabId, inst);

				if (inst) {
					window.setTimeout(() => {
						flushPendingDrops();
					}, 0);
				}
			};
		},
		[flushPendingDrops]
	);

	const syncComposerFromConversation = useCallback((tabId: string, conversation: Conversation) => {
		const input = inputRefs.current.get(tabId);
		if (!input) return;

		input.resetEditor();
		input.restoreConversationContext(deriveRestorableConversationContextFromMessages(conversation.messages));
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

	const disposeInputRuntime = useCallback((tabId: string) => {
		inputRefs.current.delete(tabId);
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
		disposeInputRuntime,
	};
}
