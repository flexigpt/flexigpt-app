import { type RefObject, useCallback } from 'react';

import type { Conversation, ConversationMessage } from '@/spec/conversation';
import {
	ContentItemKind,
	type InferenceError,
	type ModelParam,
	OutputKind,
	type OutputUnion,
	RoleEnum,
	Status,
	type UIToolCall,
} from '@/spec/inference';
import { type ModelPresetRef, type UIChatOption } from '@/spec/modelpreset';
import { type ToolStoreChoice, ToolStoreChoiceType } from '@/spec/tool';

import { ensureMakeID, getUUIDv7 } from '@/lib/uuid_utils';

import type { ComposerBoxHandle } from '@/chats/composer/composer_box';
import type { EditorExternalMessage, EditorSubmitPayload } from '@/chats/composer/editor/editor_types';
import { sliceMessagesForSend } from '@/chats/composer/previousmessages/previous_messages_helper';
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
import { appendSystemPromptParts } from '@/prompts/lib/system_prompt_utils';

type UseSendMessageArgs = {
	tabsRef: RefObject<ChatTabState[]>;
	selectedTabIdRef: RefObject<string>;
	updateTab: (tabId: string, updater: (tab: ChatTabState) => ChatTabState) => void;
	saveUpdatedConversation: (tabId: string, updatedConv: Conversation, titleWasExternallyChanged?: boolean) => void;
	scrollTabToBottomSoon: (tabId: string) => void;

	tabExists: (tabId: string) => boolean;
	getAbortRef: (tabId: string) => { current: AbortController | null };
	requestIdByTabRef: RefObject<Map<string, string | null>>;
	tokensReceivedByTabRef: RefObject<Map<string, boolean | null>>;
	clearStreamBuffer: (tabId: string) => void;
	notifyStreamNow: (tabId: string) => void;
	notifyStreamSoon: (tabId: string) => void;
	getStreamBuffer: (tabId: string) => StreamBuffer;
	getFullStreamTextForTab: (tabId: string) => string;
	getFullStreamThinkingForTab: (tabId: string) => string;

	inputRefs: RefObject<Map<string, ComposerBoxHandle | null>>;
};

function buildTerminalAssistantOutputs(assistantMessageId: string, status: Status, uiContent: string): OutputUnion[] {
	return [
		{
			kind: OutputKind.OutputMessage,
			outputMessage: {
				id: assistantMessageId,
				role: RoleEnum.Assistant,
				status,
				contents: [
					{
						kind: ContentItemKind.Text,
						textItem: {
							text: uiContent,
						},
					},
				],
			},
		},
	];
}

function buildTerminalAssistantMessage(args: {
	assistantPlaceholder: ConversationMessage;
	status: Status;
	partialText: string;
	terminalLine: string;
	error?: InferenceError;
	debugDetails?: unknown;
	uiDebugDetails?: string;
}): ConversationMessage {
	const baseText = args.partialText.trimEnd();
	const uiContent = baseText ? `${baseText}\n\n${args.terminalLine}` : args.terminalLine;

	return {
		...args.assistantPlaceholder,
		status: args.status,
		error: args.error,
		debugDetails: args.debugDetails,
		uiDebugDetails: args.uiDebugDetails,
		uiContent,
		outputs: buildTerminalAssistantOutputs(args.assistantPlaceholder.id, args.status, uiContent),
	};
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
	tokensReceivedByTabRef,
	clearStreamBuffer,
	notifyStreamNow,
	notifyStreamSoon,
	getStreamBuffer,
	getFullStreamTextForTab,
	getFullStreamThinkingForTab,
	inputRefs,
}: UseSendMessageArgs) {
	const updateStreamingMessage = useCallback(
		async (tabId: string, updatedChatWithUserMessage: Conversation, options: UIChatOption, skillSessionID?: string) => {
			if (!tabExists(tabId)) return;

			const abortRef = getAbortRef(tabId);
			let queuedRunnableToolCalls: UIToolCall[] = [];

			abortRef.current?.abort();
			tokensReceivedByTabRef.current.set(tabId, false);

			updateTab(tabId, tab => ({ ...tab, isBusy: true }));

			let reqId: string;
			try {
				reqId = getUUIDv7();
			} catch {
				reqId = ensureMakeID();
			}

			requestIdByTabRef.current.set(tabId, reqId);

			const controller = new AbortController();
			abortRef.current = controller;

			const allMessages = sliceMessagesForSend(updatedChatWithUserMessage.messages, options.includePreviousMessages);
			if (allMessages.length === 0) {
				updateTab(tabId, tab => ({ ...tab, isBusy: false }));
				return;
			}

			const currentUserMsg = allMessages[allMessages.length - 1];
			const history = allMessages.slice(0, allMessages.length - 1);
			const effectiveCurrentUserMsg = {
				...currentUserMsg,
				attachments: dedupeAttachmentsByRef(currentUserMsg.attachments),
			};

			clearStreamBuffer(tabId);
			notifyStreamNow(tabId);

			const assistantPlaceholder = initConversationMessage(RoleEnum.Assistant);
			const chatWithPlaceholder: Conversation = {
				...updatedChatWithUserMessage,
				messages: [...updatedChatWithUserMessage.messages, assistantPlaceholder],
				modifiedAt: new Date(),
			};

			updateTab(tabId, tab => ({
				...tab,
				conversation: { ...chatWithPlaceholder, messages: [...chatWithPlaceholder.messages] },
			}));

			if (selectedTabIdRef.current === tabId) {
				scrollTabToBottomSoon(tabId);
			}

			const onStreamTextData = (textData: string) => {
				if (!textData) return;
				if (requestIdByTabRef.current.get(tabId) !== reqId) return;

				tokensReceivedByTabRef.current.set(tabId, true);
				getStreamBuffer(tabId).text.chunks.push(textData);
				notifyStreamSoon(tabId);
			};

			const onStreamThinkingData = (thinkingData: string) => {
				if (!thinkingData) return;
				if (requestIdByTabRef.current.get(tabId) !== reqId) return;

				tokensReceivedByTabRef.current.set(tabId, true);
				getStreamBuffer(tabId).thinking.chunks.push(thinkingData);
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

				const latestUser = updatedChatWithUserMessage.messages
					.slice()
					.reverse()
					.find(message => message.role === RoleEnum.User);

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
					assistantPlaceholder,
					skillSessionID,
					reqId,
					controller.signal,
					onStreamTextData,
					onStreamThinkingData
				);

				if (!tabExists(tabId)) return;
				if (requestIdByTabRef.current.get(tabId) !== reqId) return;
				const partialText = getFullStreamTextForTab(tabId);
				const partialThinking = getFullStreamThinkingForTab(tabId);
				const hasPartialStream = partialText.trim().length > 0 || partialThinking.trim().length > 0;

				if (responseMessage) {
					let finalResponseMessage = responseMessage;

					const isErrorResponseWithoutOutputs =
						finalResponseMessage.status === Status.Failed &&
						!!rawResponse?.inferenceResponse?.error &&
						(rawResponse?.inferenceResponse?.outputs?.length ?? 0) === 0;

					// If backend streamed something but finalized as an error without
					// outputs, keep the streamed text and only show thinking as a
					// transient UI overlay.
					if (hasPartialStream && isErrorResponseWithoutOutputs) {
						const backendErrorMessage =
							typeof finalResponseMessage.error?.message === 'string' &&
							finalResponseMessage.error.message.trim().length > 0
								? finalResponseMessage.error.message.trim()
								: 'Got error in API processing.';

						finalResponseMessage = buildTerminalAssistantMessage({
							assistantPlaceholder,
							status: Status.Failed,
							partialText,
							terminalLine: `> Error: ${backendErrorMessage}`,
							error: finalResponseMessage.error,
							debugDetails: finalResponseMessage.debugDetails,
							uiDebugDetails: finalResponseMessage.uiDebugDetails,
						});
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

					if (rawResponse?.hydratedCurrentInputs && currentUserMsg.id) {
						const hydratedInputs = rawResponse.hydratedCurrentInputs;
						finalChat = {
							...finalChat,
							messages: finalChat.messages.map(message =>
								message.id === currentUserMsg.id
									? {
											...message,
											inputs: hydratedInputs,
										}
									: message
							),
						};
					}

					saveUpdatedConversation(tabId, finalChat);

					if (persistedAssistantMessage.uiToolCalls && persistedAssistantMessage.uiToolCalls.length > 0) {
						queuedRunnableToolCalls = persistedAssistantMessage.uiToolCalls.filter(
							call => call.type === ToolStoreChoiceType.Function || call.type === ToolStoreChoiceType.Custom
						);
					}
				} else {
					const partialText = getFullStreamTextForTab(tabId);
					const partialThinking = getFullStreamThinkingForTab(tabId);
					const hasPartialStream = partialText.trim().length > 0 || partialThinking.trim().length > 0;

					const fallbackBase = hasPartialStream
						? buildTerminalAssistantMessage({
								assistantPlaceholder,
								status: Status.Failed,
								partialText,
								terminalLine: '> Error: No response was returned by the backend.',
								error: {
									code: 'unknown',
									message: 'No response was returned by the backend.',
								} as InferenceError,
							})
						: {
								...assistantPlaceholder,
								status: Status.Failed,
								error: {
									code: 'unknown',
									message: 'No response was returned by the backend.',
								} as InferenceError,
								uiContent: '> Error: No response was returned by the backend.',
								outputs: buildTerminalAssistantOutputs(
									assistantPlaceholder.id,
									Status.Failed,
									'> Error: No response was returned by the backend.'
								),
							};

					const fallbackMsg = applyAssistantPersistenceContext(
						fallbackBase,
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
			} catch (error) {
				if (!tabExists(tabId)) return;
				if (requestIdByTabRef.current.get(tabId) !== reqId) return;

				if ((error as DOMException).name === 'AbortError') {
					const tokensReceived = tokensReceivedByTabRef.current.get(tabId);

					if (!tokensReceived) {
						updateTab(tabId, tab => {
							const idx = tab.conversation.messages.findIndex(message => message.id === assistantPlaceholder.id);
							if (idx === -1) return tab;

							const messages = tab.conversation.messages.filter((_, i) => i !== idx);
							return {
								...tab,
								conversation: { ...tab.conversation, messages, modifiedAt: new Date() },
							};
						});
					} else {
						const partialText = getFullStreamTextForTab(tabId);

						const partialMsg = applyAssistantPersistenceContext(
							buildTerminalAssistantMessage({
								assistantPlaceholder,
								status: Status.Completed,
								partialText,
								terminalLine: '> API aborted after partial response...',
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

					const fallbackMsg = applyAssistantPersistenceContext(
						buildTerminalAssistantMessage({
							assistantPlaceholder,
							status: Status.Failed,
							partialText,
							terminalLine: `> Error: ${errorMessage}`,
							error: {
								code: 'unknown',
								message: errorMessage,
							} as InferenceError,
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
				if (tabExists(tabId) && requestIdByTabRef.current.get(tabId) === reqId) {
					clearStreamBuffer(tabId);
					updateTab(tabId, tab => ({ ...tab, isBusy: false }));

					if (queuedRunnableToolCalls.length > 0) {
						requestAnimationFrame(() => {
							if (!tabExists(tabId)) return;
							if (requestIdByTabRef.current.get(tabId) !== reqId) return;
							inputRefs.current.get(tabId)?.loadToolCalls(queuedRunnableToolCalls);
						});
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
			inputRefs,
			notifyStreamNow,
			notifyStreamSoon,
			requestIdByTabRef,
			saveUpdatedConversation,
			scrollTabToBottomSoon,
			selectedTabIdRef,
			tabExists,
			tokensReceivedByTabRef,
			updateTab,
		]
	);

	const sendMessageForTab = useCallback(
		async (tabId: string, payload: EditorSubmitPayload, options: UIChatOption) => {
			const tab = tabsRef.current.find(t => t.tabId === tabId);
			if (!tab) return;
			if (tab.isBusy || tab.isHydrating) return;

			const trimmed = payload.text.trim();
			const hasNonEmptyText = trimmed.length > 0;
			const hasToolOutputs = payload.toolOutputs.length > 0;
			const hasAttachments = payload.attachments.length > 0;

			if (!hasNonEmptyText && !hasToolOutputs && !hasAttachments) return;

			let sendOptions: UIChatOption = {
				...options,
				systemPrompt: payload.resolvedSystemPrompt?.trim() || '',
			};
			const editingId = tab.editingMessageId ?? undefined;
			const templateSystemPrompt = payload.templateSystemPrompt?.trim() || undefined;

			if (templateSystemPrompt) {
				sendOptions = {
					...sendOptions,
					systemPrompt: appendSystemPromptParts(sendOptions.systemPrompt ?? '', [templateSystemPrompt]),
				};
			}

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
			if (!tab) return;
			if (tab.isBusy || tab.isHydrating) return;

			const message = tab.conversation.messages.find(m => m.id === id);
			if (!message) return;
			if (message.role !== RoleEnum.User) return;

			const external: EditorExternalMessage = {
				text: message.uiContent ?? '',
				attachments: message.attachments,
				toolChoices: message.toolStoreChoices,
				toolOutputs: message.uiToolOutputs,
				enabledSkillRefs: message.enabledSkillRefs,
				activeSkillRefs: message.activeSkillRefs,
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
