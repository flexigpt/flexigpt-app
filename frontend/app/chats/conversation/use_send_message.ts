import type { RefObject } from 'react';
import { useCallback } from 'react';

import type { Conversation, ConversationMessage } from '@/spec/conversation';
import type { InferenceError, ModelParam, UIToolCall } from '@/spec/inference';
import { RoleEnum, Status } from '@/spec/inference';
import type { ModelPresetRef, UIChatOption } from '@/spec/modelpreset';
import type { ToolStoreChoice } from '@/spec/tool';

import { ensureMakeID, getUUIDv7 } from '@/lib/uuid_utils';

import type { ComposerBoxHandle } from '@/chats/composer/composer_box';
import type {
	AssistantTurnFinishedPayload,
	EditorExternalMessage,
	EditorSubmitPayload,
} from '@/chats/composer/editor/editor_types';
import { sliceMessagesForSend } from '@/chats/composer/previousmessages/previous_messages_helper';
import {
	applyCompletionMetadata,
	buildTerminalAssistantMessage,
	getFailureTerminalLine,
	getInferenceFailureMessage,
	hasNonTextAssistantOutcome,
	reconcileAssistantTextWithStream,
	streamHasData,
} from '@/chats/conversation/completion_finalizer';
import { HandleCompletion } from '@/chats/conversation/completion_helper';
import {
	applyAssistantPersistenceContext,
	buildUserConversationMessageFromEditor,
	dedupeAttachmentsByRef,
	initConversationMessage,
	shouldPersistAssistantModelParam,
} from '@/chats/conversation/hydration_helper';
import type { StreamBuffer } from '@/chats/conversation/use_streaming_runtime';
import type { ChatTabState } from '@/chats/tabs/tabs_model';
import { isRunnableComposerToolCall } from '@/tools/lib/tool_call_utils';

interface UseSendMessageArgs {
	tabsRef: RefObject<ChatTabState[]>;
	selectedTabIdRef: RefObject<string>;
	updateTab: (tabId: string, updater: (tab: ChatTabState) => ChatTabState) => void;
	saveUpdatedConversation: (tabId: string, updatedConv: Conversation, titleWasExternallyChanged?: boolean) => void;
	scrollTabToBottomSoon: (tabId: string) => void;

	tabExists: (tabId: string) => boolean;
	getAbortRef: (tabId: string) => { current: AbortController | null };
	requestIdByTabRef: RefObject<Map<string, string | null>>;
	clearStreamBuffer: (tabId: string) => void;
	notifyStreamNow: (tabId: string) => void;
	notifyStreamSoon: (tabId: string) => void;
	getStreamBuffer: (tabId: string) => StreamBuffer;
	getFullStreamTextForTab: (tabId: string) => string;
	getFullStreamThinkingForTab: (tabId: string) => string;

	inputRefs: RefObject<Map<string, ComposerBoxHandle | null>>;
	loadAssistantTurnForTab: (
		tabId: string,
		toolCalls: UIToolCall[],
		finishPayload: AssistantTurnFinishedPayload
	) => boolean;
}

function findLatestUserMessage(messages: ConversationMessage[]): ConversationMessage | undefined {
	for (let index = messages.length - 1; index >= 0; index -= 1) {
		const message = messages[index];
		if (message.role === RoleEnum.User) {
			return message;
		}
	}
	return undefined;
}

export function useSendMessage({
	tabsRef,
	selectedTabIdRef,
	updateTab,
	saveUpdatedConversation,
	scrollTabToBottomSoon,
	tabExists,
	getAbortRef,
	requestIdByTabRef,
	clearStreamBuffer,
	notifyStreamNow,
	notifyStreamSoon,
	getStreamBuffer,
	getFullStreamTextForTab,
	getFullStreamThinkingForTab,
	inputRefs,
	loadAssistantTurnForTab,
}: UseSendMessageArgs) {
	const updateStreamingMessage = useCallback(
		async (tabId: string, updatedChatWithUserMessage: Conversation, options: UIChatOption, skillSessionID?: string) => {
			if (!tabExists(tabId)) {
				return;
			}

			const abortRef = getAbortRef(tabId);
			let queuedRunnableToolCalls: UIToolCall[] = [];

			abortRef.current?.abort();
			const allMessages = sliceMessagesForSend(updatedChatWithUserMessage.messages, options.includePreviousMessages);
			if (allMessages.length === 0) {
				return;
			}

			let reqId: string;
			try {
				reqId = getUUIDv7();
			} catch {
				reqId = ensureMakeID();
			}

			requestIdByTabRef.current.set(tabId, reqId);

			const controller = new AbortController();
			abortRef.current = controller;

			const currentUserMsg = allMessages.at(-1);
			const history = allMessages.slice(0, allMessages.length - 1);
			const effectiveCurrentUserMsg = {
				...currentUserMsg,
				attachments: dedupeAttachmentsByRef(currentUserMsg?.attachments),
			} as ConversationMessage;

			clearStreamBuffer(tabId);
			notifyStreamNow(tabId);
			const streamBuffer = getStreamBuffer(tabId);

			const assistantPlaceholder = initConversationMessage(RoleEnum.Assistant);
			const chatWithPlaceholder: Conversation = {
				...updatedChatWithUserMessage,
				messages: [...updatedChatWithUserMessage.messages, assistantPlaceholder],
				modifiedAt: new Date(),
			};

			updateTab(tabId, tab => ({
				...tab,
				isBusy: true,
				conversation: { ...chatWithPlaceholder, messages: [...chatWithPlaceholder.messages] },
			}));

			if (selectedTabIdRef.current === tabId) {
				scrollTabToBottomSoon(tabId);
			}

			const onStreamTextData = (textData: string) => {
				// Empty chunks are valid no-ops, never an end-of-stream signal.
				if (typeof textData !== 'string' || textData.length === 0) {
					return;
				}
				if (requestIdByTabRef.current.get(tabId) !== reqId) {
					return;
				}

				streamBuffer.text.chunks.push(textData);
				notifyStreamSoon(tabId);
			};

			const onStreamThinkingData = (thinkingData: string) => {
				if (typeof thinkingData !== 'string' || thinkingData.length === 0) {
					return;
				}
				if (requestIdByTabRef.current.get(tabId) !== reqId) {
					return;
				}

				streamBuffer.thinking.chunks.push(thinkingData);
				notifyStreamSoon(tabId);
			};

			const inputParams: ModelParam = {
				name: options.name,
				temperature: options.temperature,
				stream: options.stream,
				maxPromptLength: options.maxPromptLength,
				cacheControl: options.cacheControl,
				maxOutputLength: options.maxOutputLength,
				reasoning: options.reasoning,
				systemPrompt: options.systemPrompt,
				timeout: options.timeout,
				outputParam: options.outputParam,
				stopSequences: options.stopSequences,
				additionalParametersRawJSON: options.additionalParametersRawJSON,
			};

			const effectiveModelPresetRef: ModelPresetRef = {
				providerName: options.providerName,
				modelPresetID: options.modelPresetID,
			};

			const persistedAssistantModelParam = shouldPersistAssistantModelParam(
				updatedChatWithUserMessage.messages,
				inputParams
			)
				? inputParams
				: undefined;

			try {
				let toolStoreChoices: ToolStoreChoice[] | undefined;

				const latestUser = findLatestUserMessage(updatedChatWithUserMessage.messages);

				if (latestUser?.toolStoreChoices && latestUser.toolStoreChoices.length > 0) {
					toolStoreChoices = latestUser.toolStoreChoices;
				}

				const { responseMessage, rawResponse } = await HandleCompletion(
					options.providerName,
					options.modelPresetID,
					inputParams,
					effectiveCurrentUserMsg,
					history,
					toolStoreChoices,
					effectiveCurrentUserMsg.mcpContext,
					assistantPlaceholder,
					skillSessionID,
					reqId,
					controller.signal,
					onStreamTextData,
					onStreamThinkingData
				);

				// Yield only to already-queued bridge microtasks. This does not add
				// a timer or frame delay during normal streaming.
				await Promise.resolve();

				if (!tabExists(tabId)) {
					return;
				}
				if (requestIdByTabRef.current.get(tabId) !== reqId) {
					return;
				}

				const partialText = getFullStreamTextForTab(tabId);
				const partialThinking = getFullStreamThinkingForTab(tabId);
				const hasPartialText = streamHasData(partialText);
				const hasPartialStream = hasPartialText || streamHasData(partialThinking);

				if (responseMessage) {
					const responseError = responseMessage.error ?? rawResponse?.inferenceResponse?.error;
					const responseFailed = responseMessage.status === Status.Failed || !!responseError;
					let finalResponseMessage: ConversationMessage;

					if (responseFailed) {
						const rawOutputs = rawResponse?.inferenceResponse?.outputs ?? [];
						const baseText = rawOutputs.length > 0 ? (responseMessage.uiContent ?? '') : '';
						const failureError =
							responseError ??
							({
								code: 'unknown',
								message: 'The API returned an incomplete response.',
							} as InferenceError);
						const hasContentBeforeFailure = hasPartialStream || streamHasData(baseText);

						finalResponseMessage = buildTerminalAssistantMessage({
							baseMessage: responseMessage,
							status: Status.Failed,
							partialText,
							partialThinking,
							baseText,
							error: failureError,
							terminalLine: getFailureTerminalLine(
								getInferenceFailureMessage(failureError, 'The API returned an incomplete response.'),
								hasContentBeforeFailure
							),
							preferStreamedText: hasPartialText,
							includeStreamedThinking: true,
						});
					} else {
						// Inspect the backend's final payload before reconciling it
						// with streamed text. Otherwise an empty final payload would
						// incorrectly look successful merely because streaming had
						// already produced visible content.
						const finalPayloadHasOutcome =
							/\S/.test(responseMessage.uiContent ?? '') || hasNonTextAssistantOutcome(responseMessage);

						if (!finalPayloadHasOutcome) {
							const noResponseError = {
								code: 'empty_completion_response',
								message: 'The backend ended without a usable final response.',
							} as InferenceError;

							finalResponseMessage = buildTerminalAssistantMessage({
								baseMessage: responseMessage,
								status: Status.Failed,
								partialText,
								partialThinking,
								error: noResponseError,
								terminalLine: getFailureTerminalLine(noResponseError.message, hasPartialStream),
								preferStreamedText: true,
								includeStreamedThinking: true,
							});
						} else {
							finalResponseMessage = reconcileAssistantTextWithStream(responseMessage, partialText);
						}
					}

					const persistedAssistantMessage = applyAssistantPersistenceContext(
						finalResponseMessage,
						effectiveModelPresetRef,
						persistedAssistantModelParam
					);

					let finalChat: Conversation = {
						...chatWithPlaceholder,
						messages: [...chatWithPlaceholder.messages.slice(0, -1), persistedAssistantMessage],
						modifiedAt: new Date(),
					};

					finalChat = applyCompletionMetadata(finalChat, currentUserMsg?.id, rawResponse);

					saveUpdatedConversation(tabId, finalChat);

					if (persistedAssistantMessage.uiToolCalls && persistedAssistantMessage.uiToolCalls.length > 0) {
						queuedRunnableToolCalls = persistedAssistantMessage.uiToolCalls.filter(call =>
							isRunnableComposerToolCall(call)
						);
					}
				} else {
					const fallbackError = {
						code: 'unknown',
						message: 'No response was returned by the backend.',
					} as InferenceError;
					const fallbackMsg = applyAssistantPersistenceContext(
						buildTerminalAssistantMessage({
							baseMessage: assistantPlaceholder,
							status: Status.Failed,
							partialText,
							partialThinking,
							error: fallbackError,
							terminalLine: getFailureTerminalLine(fallbackError.message, hasPartialStream),
							preferStreamedText: true,
							includeStreamedThinking: true,
						}),
						effectiveModelPresetRef,
						persistedAssistantModelParam
					);

					let finalChat: Conversation = {
						...chatWithPlaceholder,
						messages: [...chatWithPlaceholder.messages.slice(0, -1), fallbackMsg],
						modifiedAt: new Date(),
					};
					finalChat = applyCompletionMetadata(finalChat, currentUserMsg?.id, rawResponse);

					saveUpdatedConversation(tabId, finalChat);
				}
			} catch (error) {
				if (!tabExists(tabId)) {
					return;
				}
				if (requestIdByTabRef.current.get(tabId) !== reqId) {
					return;
				}

				if ((error as DOMException).name === 'AbortError') {
					const partialText = getFullStreamTextForTab(tabId);
					const partialThinking = getFullStreamThinkingForTab(tabId);
					const hasPartialStream = streamHasData(partialText) || streamHasData(partialThinking);

					if (!hasPartialStream) {
						updateTab(tabId, tab => {
							const idx = tab.conversation.messages.findIndex(message => message.id === assistantPlaceholder.id);
							if (idx === -1) {
								return tab;
							}

							const messages = tab.conversation.messages.filter((_, i) => i !== idx);
							return {
								...tab,
								conversation: { ...tab.conversation, messages, modifiedAt: new Date() },
							};
						});
					} else {
						const partialMsg = applyAssistantPersistenceContext(
							buildTerminalAssistantMessage({
								baseMessage: assistantPlaceholder,
								status: Status.Completed,
								partialText,
								partialThinking,
								terminalLine: '> Generation stopped before the API returned a final response.',
								includeStreamedThinking: true,
							}),
							effectiveModelPresetRef,
							persistedAssistantModelParam
						);

						const finalChat: Conversation = {
							...chatWithPlaceholder,
							messages: [...chatWithPlaceholder.messages.slice(0, -1), partialMsg],
							modifiedAt: new Date(),
						};

						saveUpdatedConversation(tabId, finalChat);
					}
				} else {
					console.error(error);

					const errorMessage =
						error instanceof Error && error.message.trim().length > 0
							? error.message
							: 'Unexpected error while processing this request.';

					const partialText = getFullStreamTextForTab(tabId);
					const partialThinking = getFullStreamThinkingForTab(tabId);
					const fallbackError = {
						code: 'unknown',
						message: errorMessage,
					} as InferenceError;
					const hasPartialStream = streamHasData(partialText) || streamHasData(partialThinking);

					const fallbackMsg = applyAssistantPersistenceContext(
						buildTerminalAssistantMessage({
							baseMessage: assistantPlaceholder,
							status: Status.Failed,
							partialText,
							partialThinking,
							terminalLine: getFailureTerminalLine(errorMessage, hasPartialStream),
							error: fallbackError,
							preferStreamedText: true,
							includeStreamedThinking: true,
						}),
						effectiveModelPresetRef,
						persistedAssistantModelParam
					);

					const finalChat: Conversation = {
						...chatWithPlaceholder,
						messages: [...chatWithPlaceholder.messages.slice(0, -1), fallbackMsg],
						modifiedAt: new Date(),
					};

					saveUpdatedConversation(tabId, finalChat);
				}
			} finally {
				if (abortRef.current === controller) {
					abortRef.current = null;
				}

				if (tabExists(tabId) && requestIdByTabRef.current.get(tabId) === reqId) {
					updateTab(tabId, tab => (tab.isBusy ? { ...tab, isBusy: false } : tab));

					const finishPayload: AssistantTurnFinishedPayload = {
						loadedRunnableToolCallCount: queuedRunnableToolCalls.length,
					};

					// Keep the last stream snapshot until the next request or
					// explicit tab cleanup. Clearing it before React commits the
					// final message can make the live card briefly render empty.
					// The next request clears this buffer before installing its
					// placeholder, so stale text cannot enter another turn.

					const deliverAssistantTurn = () => {
						if (!tabExists(tabId)) {
							return;
						}
						if (requestIdByTabRef.current.get(tabId) !== reqId) {
							return;
						}
						loadAssistantTurnForTab(tabId, queuedRunnableToolCalls, finishPayload);
					};

					// The runtime itself remains blocked by the tab's busy prop until
					// React commits the completion transition. No animation frame is
					// needed, and background windows must not pause this delivery.
					deliverAssistantTurn();

					if (requestIdByTabRef.current.get(tabId) === reqId) {
						requestIdByTabRef.current.set(tabId, null);
					}
				}
			}
		},
		[
			clearStreamBuffer,
			getAbortRef,
			getFullStreamTextForTab,
			getFullStreamThinkingForTab,
			getStreamBuffer,
			loadAssistantTurnForTab,
			notifyStreamNow,
			notifyStreamSoon,
			requestIdByTabRef,
			saveUpdatedConversation,
			scrollTabToBottomSoon,
			selectedTabIdRef,
			tabExists,
			updateTab,
		]
	);

	const sendMessageForTab = useCallback(
		async (tabId: string, payload: EditorSubmitPayload, options: UIChatOption) => {
			const tab = tabsRef.current.find(t => t.tabId === tabId);
			if (!tab) {
				return;
			}
			if (tab.isBusy || tab.isHydrating) {
				return;
			}

			const trimmed = payload.text.trim();
			const hasNonEmptyText = trimmed.length > 0;
			const hasToolOutputs = payload.toolOutputs.length > 0;
			const hasAttachments = payload.attachments.length > 0;
			const hasMCPContext = (payload.mcpContext?.servers?.length ?? 0) > 0;
			const hasMCPAppContext = (payload.mcpAppContextUpdates?.length ?? 0) > 0;

			if (!hasNonEmptyText && !hasToolOutputs && !hasAttachments && !hasMCPContext && !hasMCPAppContext) {
				return;
			}

			const sendOptions: UIChatOption = {
				...options,
				systemPrompt: payload.resolvedSystemPrompt?.trim() || '',
			};
			const editingId = tab.editingMessageId ?? undefined;

			const modelPresetRef: ModelPresetRef = {
				providerName: sendOptions.providerName,
				modelPresetID: sendOptions.modelPresetID,
			};

			const userMsg = buildUserConversationMessageFromEditor(payload, editingId, modelPresetRef);

			if (tab.editingMessageId) {
				const idx = tab.conversation.messages.findIndex(message => message.id === tab.editingMessageId);
				if (idx !== -1) {
					const oldMessages = tab.conversation.messages.slice(0, idx);
					const messages = [...oldMessages, userMsg];

					const updatedChat: Conversation = {
						...tab.conversation,
						messages,
						modifiedAt: new Date(),
					};

					updateTab(tabId, current => ({ ...current, editingMessageId: null }));
					saveUpdatedConversation(tabId, updatedChat);
					void updateStreamingMessage(tabId, updatedChat, sendOptions, payload.skillSessionID).catch(console.error);
					return;
				}

				updateTab(tabId, current => ({ ...current, editingMessageId: null }));
			}

			const updatedChat: Conversation = {
				...tab.conversation,
				messages: [...tab.conversation.messages, userMsg],
				modifiedAt: new Date(),
			};

			saveUpdatedConversation(tabId, updatedChat);

			if (selectedTabIdRef.current === tabId) {
				scrollTabToBottomSoon(tabId);
			}

			void updateStreamingMessage(tabId, updatedChat, sendOptions, payload.skillSessionID).catch(console.error);
		},
		[selectedTabIdRef, saveUpdatedConversation, scrollTabToBottomSoon, tabsRef, updateStreamingMessage, updateTab]
	);

	const beginEditMessageForTab = useCallback(
		(tabId: string, id: string) => {
			const tab = tabsRef.current.find(t => t.tabId === tabId);
			if (!tab) {
				return;
			}
			if (tab.isBusy || tab.isHydrating) {
				return;
			}

			const message = tab.conversation.messages.find(m => m.id === id);
			if (!message) {
				return;
			}
			if (message.role !== RoleEnum.User) {
				return;
			}

			const external: EditorExternalMessage = {
				text: message.uiContent ?? '',
				attachments: message.attachments,
				toolChoices: message.toolStoreChoices,
				mcpContext: message.mcpContext,
				mcpAppContextUpdates: message.mcpAppContextUpdates,
				toolOutputs: message.uiToolOutputs,
				enabledSkillRefs: message.enabledSkillRefs,
				activeSkillRefs: message.activeSkillRefs,
				workspaceSelection: message.workspaceSelection,
			};

			inputRefs.current.get(tabId)?.loadExternalMessage(external);
			updateTab(tabId, current => ({ ...current, editingMessageId: id }));
		},
		[inputRefs, tabsRef, updateTab]
	);

	const cancelEditingForTab = useCallback(
		(tabId: string) => {
			updateTab(tabId, current => ({ ...current, editingMessageId: null }));
		},
		[updateTab]
	);

	return {
		sendMessageForTab,
		beginEditMessageForTab,
		cancelEditingForTab,
	};
}
