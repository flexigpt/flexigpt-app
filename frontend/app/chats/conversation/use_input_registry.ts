import { useCallback, useEffect, useLayoutEffect, useRef } from 'react';

import type { AttachmentsDroppedPayload } from '@/spec/attachment';
import type { Conversation } from '@/spec/conversation';
import type { SessionConfiguration } from '@/spec/session';

import type { ComposerBoxHandle } from '@/chats/composer/composer_box';
import type { ChatWorkflowStarter } from '@/chats/conversation/starter_intent';
import { deriveRestorableConversationContextFromMessages } from '@/chats/conversation/hydration_helper';

interface PendingDrop {
	tabId: string;
	payload: AttachmentsDroppedPayload;
}

interface PendingSessionConfiguration {
	tabId: string;
	configuration: SessionConfiguration;
}

interface UseInputRegistryArgs {
	tabExists: (tabId: string) => boolean;
}

export function useInputRegistry({ tabExists }: UseInputRegistryArgs) {
	const inputRefs = useRef(new Map<string, ComposerBoxHandle | null>());
	const pendingDropsRef = useRef<PendingDrop[]>([]);
	const pendingConversationContextsRef = useRef(new Map<string, Conversation>());
	const pendingSessionConfigurationsRef = useRef(new Map<string, PendingSessionConfiguration>());
	const pendingWorkflowStartersRef = useRef(new Map<string, ChatWorkflowStarter>());
	const workflowStarterLoadsRef = useRef(new Map<string, ChatWorkflowStarter>());
	const inputRefCallbacksRef = useRef(new Map<string, (inst: ComposerBoxHandle | null) => void>());
	const pendingInputFlushTimerRef = useRef<number | null>(null);
	const flushPendingDropsRef = useRef<() => void>(() => {});
	const flushPendingConversationContextsRef = useRef<() => void>(() => {});
	const flushPendingWorkflowStartersRef = useRef<() => void>(() => {});
	const flushPendingSessionConfigurationsRef = useRef<() => void>(() => {});

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

	const tryApplySessionConfigurationToTab = useCallback(
		async (tabId: string, configuration: SessionConfiguration): Promise<boolean> => {
			const input = inputRefs.current.get(tabId);
			if (!input) {
				return false;
			}

			await input.restoreSessionConfiguration(configuration);
			return true;
		},
		[]
	);

	const flushPendingSessionConfigurations = useCallback(() => {
		for (const [tabId, pending] of pendingSessionConfigurationsRef.current.entries()) {
			if (!tabExists(tabId)) {
				pendingSessionConfigurationsRef.current.delete(tabId);
				continue;
			}

			void tryApplySessionConfigurationToTab(tabId, pending.configuration)
				.then(applied => {
					if (applied && pendingSessionConfigurationsRef.current.get(tabId) === pending) {
						pendingSessionConfigurationsRef.current.delete(tabId);
					}
				})
				.catch(error => {
					console.error('Failed to restore Session Composer configuration:', error);
				});
		}
	}, [tabExists, tryApplySessionConfigurationToTab]);

	const restoreSessionConfiguration = useCallback(
		async (tabId: string, configuration: SessionConfiguration) => {
			if (await tryApplySessionConfigurationToTab(tabId, configuration)) {
				pendingSessionConfigurationsRef.current.delete(tabId);
				return;
			}

			pendingSessionConfigurationsRef.current.set(tabId, { tabId, configuration });
			schedulePendingInputFlush();
		},
		[schedulePendingInputFlush, tryApplySessionConfigurationToTab]
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

			// Loading a workflow starter is asynchronous. A composer mount or
			// another pending-input flush must not start the same load again.
			if (workflowStarterLoadsRef.current.has(tabId)) {
				continue;
			}

			workflowStarterLoadsRef.current.set(tabId, starter);
			void tryApplyWorkflowStarterToTab(tabId, starter)
				.then(applied => {
					if (applied && pendingWorkflowStartersRef.current.get(tabId) === starter) {
						pendingWorkflowStartersRef.current.delete(tabId);
					}
				})
				.catch((error: unknown) => {
					console.error('Failed to apply pending workflow starter:', error);
				})
				.finally(() => {
					if (workflowStarterLoadsRef.current.get(tabId) === starter) {
						workflowStarterLoadsRef.current.delete(tabId);
					}

					// A newer starter may have replaced this one while the
					// asynchronous load was running.
					const queuedStarter = pendingWorkflowStartersRef.current.get(tabId);
					if (queuedStarter && queuedStarter !== starter) {
						flushPendingWorkflowStartersRef.current();
					}
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
		flushPendingWorkflowStartersRef.current = flushPendingWorkflowStarters;
	}, [flushPendingWorkflowStarters]);

	useLayoutEffect(() => {
		flushPendingSessionConfigurationsRef.current = flushPendingSessionConfigurations;
	}, [flushPendingSessionConfigurations]);

	const schedulePendingInputFlush = useCallback(() => {
		if (pendingInputFlushTimerRef.current !== null) {
			return;
		}

		pendingInputFlushTimerRef.current = window.setTimeout(() => {
			pendingInputFlushTimerRef.current = null;
			flushPendingDropsRef.current();
			flushPendingConversationContextsRef.current();
			flushPendingWorkflowStartersRef.current();
			flushPendingSessionConfigurationsRef.current();
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
			if (!tabExists(tabId)) {
				return false;
			}
			if (await tryApplyWorkflowStarterToTab(tabId, starter)) {
				return true;
			}

			pendingWorkflowStartersRef.current.set(tabId, starter);
			// The composer may have mounted after the failed direct attempt but
			// before the starter was inserted into the pending map.
			schedulePendingInputFlush();
			return true;
		},
		[schedulePendingInputFlush, tabExists, tryApplyWorkflowStarterToTab]
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
	const openWorkspaceMenu = useCallback((tabId: string) => inputRefs.current.get(tabId)?.openWorkspaceMenu(), []);
	const requestStopResponse = useCallback((tabId: string) => inputRefs.current.get(tabId)?.requestStopResponse(), []);

	const disposeInputRuntime = useCallback((tabId: string) => {
		inputRefs.current.delete(tabId);
		pendingWorkflowStartersRef.current.delete(tabId);
		workflowStarterLoadsRef.current.delete(tabId);
		pendingConversationContextsRef.current.delete(tabId);
		pendingSessionConfigurationsRef.current.delete(tabId);
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
		restoreSessionConfiguration,
		focusInput,
		openTemplateMenu,
		openToolMenu,
		openAttachmentMenu,
		openSystemPromptMenu,
		openSkillsMenu,
		openMCPMenu,
		openWorkspaceMenu,
		requestStopResponse,
		disposeInputRuntime,
		applyWorkflowStarterToComposer,
	};
}
