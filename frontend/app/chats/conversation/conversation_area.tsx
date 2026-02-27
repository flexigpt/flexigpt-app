import {
	forwardRef,
	useCallback,
	useEffect,
	useImperativeHandle,
	useLayoutEffect,
	useMemo,
	useRef,
	useSyncExternalStore,
} from 'react';

import type { AttachmentsDroppedPayload } from '@/spec/attachment';
import type { Conversation, ConversationMessage } from '@/spec/conversation';
import { ContentItemKind, type ModelParam, OutputKind, type OutputUnion, RoleEnum, Status } from '@/spec/inference';
import { type UIChatOption } from '@/spec/modelpreset';
import { type ToolStoreChoice, ToolStoreChoiceType } from '@/spec/tool';

import type { ShortcutConfig } from '@/lib/keyboard_shortcuts';
import { ensureMakeID, getUUIDv7 } from '@/lib/uuid_utils';

import { useAtTopBottom } from '@/hooks/use_at_top_bottom';

import { attachmentsDropAPI } from '@/apis/baseapi';

import { ButtonScrollToBottom, ButtonScrollToTop } from '@/components/button_scroll_top_bottom';

import type { ChatTabState } from '@/chats/chat_tabs_persist';
import { HandleCompletion } from '@/chats/conversation/completion_helper';
import {
	buildUserConversationMessageFromEditor,
	dedupeAttachmentsByRef,
	deriveConversationToolsFromMessages,
	deriveWebSearchChoiceFromMessages,
	initConversationMessage,
} from '@/chats/conversation/hydration_helper';
import type { InputBoxHandle } from '@/chats/inputarea/input_box';
import type { EditorExternalMessage, EditorSubmitPayload } from '@/chats/inputarea/input_editor_utils';
import { InputPane } from '@/chats/inputarea/input_pane';
import { ChatMessage } from '@/chats/messages/message';

type StreamChannelBuffer = { chunks: string[]; flushedIdx: number; display: string };
type StreamBuffer = { text: StreamChannelBuffer; thinking: StreamChannelBuffer };

function StreamingLastMessage(props: {
	message: ConversationMessage;
	rowIsBusy: boolean;

	isEditing: boolean;
	onEdit: () => void;
	subscribe: (cb: () => void) => () => void;
	getSnapshot: () => number;
	getStreamText: () => string;
	getStreamThinking: () => string;
}) {
	// Only this component re-renders on stream updates (not the whole ConversationArea).
	useSyncExternalStore(props.subscribe, props.getSnapshot, () => 0);
	const streamedText = props.rowIsBusy ? props.getStreamText() : '';
	const streamedThinking = props.rowIsBusy ? props.getStreamThinking() : '';
	return (
		<ChatMessage
			message={props.message}
			onEdit={props.onEdit}
			streamedText={streamedText}
			streamedThinking={streamedThinking}
			// IMPORTANT: only the streaming row sees busy=true (prevents EnhancedMarkdown remounts for all other rows)
			isBusy={props.rowIsBusy}
			isEditing={props.isEditing}
		/>
	);
}

export type ConversationAreaHandle = {
	disposeTabRuntime: (tabId: string) => void;
	clearStreamForTab: (tabId: string) => void;
	syncComposerFromConversation: (tabId: string, conv: Conversation) => void;

	focusInput: (tabId: string) => void;
	openTemplateMenu: (tabId: string) => void;
	openToolMenu: (tabId: string) => void;
	openAttachmentMenu: (tabId: string) => void;

	setScrollTopForTab: (tabId: string, top: number) => void;
	resetScrollToTop: (tabId: string) => void;
	getScrollTopByTabSnapshot: () => Record<string, number>;
};

type ConversationAreaProps = {
	tabs: ChatTabState[];
	selectedTabId: string;

	// Used by InputPane (unchanged API)
	shortcutConfig: ShortcutConfig;

	// Persisted scroll restore seed (from storage)
	initialScrollTopByTab?: Record<string, number>;

	// State mutations remain in the page; this component owns "conversation runtime":
	// streaming buffers, abort controllers, input refs, scroll restoration, send/edit.
	updateTab: (tabId: string, updater: (t: ChatTabState) => ChatTabState) => void;
	saveUpdatedConversation: (tabId: string, updatedConv: Conversation, titleWasExternallyChanged?: boolean) => void;
};

export const ConversationArea = forwardRef<ConversationAreaHandle, ConversationAreaProps>(function ConversationArea(
	{ tabs, selectedTabId, shortcutConfig, initialScrollTopByTab, updateTab, saveUpdatedConversation },
	ref
) {
	// ---------------- Tabs ref (for async safety) ----------------
	const tabsRef = useRef(tabs);
	useEffect(() => {
		tabsRef.current = tabs;
	}, [tabs]);

	const selectedTabIdRef = useRef(selectedTabId);
	useEffect(() => {
		selectedTabIdRef.current = selectedTabId;
	}, [selectedTabId]);

	const activeTab = useMemo(() => tabs.find(t => t.tabId === selectedTabId) ?? tabs[0], [tabs, selectedTabId]);

	// PERF: O(1) existence checks (used in async paths)
	const tabIdSetRef = useRef<Set<string>>(new Set());
	useEffect(() => {
		tabIdSetRef.current = new Set(tabs.map(t => t.tabId));
	}, [tabs]);
	const tabExists = useCallback((tabId: string) => tabIdSetRef.current.has(tabId), []);

	// ---------------- Per-tab runtime refs ----------------
	const abortRefs = useRef(new Map<string, { current: AbortController | null }>());
	const requestIdByTab = useRef(new Map<string, string | null>());
	const tokensReceivedByTab = useRef(new Map<string, boolean | null>());

	const getAbortRef = useCallback((tabId: string) => {
		let refObj = abortRefs.current.get(tabId);
		if (!refObj) {
			refObj = { current: null };
			abortRefs.current.set(tabId, refObj);
		}
		return refObj;
	}, []);

	// Stream buffers (chunked to avoid expensive repeated string concatenation)
	const streamBuffersRef = useRef(new Map<string, StreamBuffer>());
	const streamVersionRef = useRef(new Map<string, number>());

	const getStreamBuffer = useCallback((tabId: string) => {
		let buf = streamBuffersRef.current.get(tabId);
		if (!buf) {
			buf = {
				text: { chunks: [], flushedIdx: 0, display: '' },
				thinking: { chunks: [], flushedIdx: 0, display: '' },
			};
			streamBuffersRef.current.set(tabId, buf);
		}
		return buf;
	}, []);

	const clearStreamBuffer = useCallback(
		(tabId: string) => {
			const buf = getStreamBuffer(tabId);
			buf.text.chunks = [];
			buf.text.flushedIdx = 0;
			buf.text.display = '';

			buf.thinking.chunks = [];
			buf.thinking.flushedIdx = 0;
			buf.thinking.display = '';
		},
		[getStreamBuffer]
	);

	const flushStreamForTab = useCallback(
		(tabId: string) => {
			const buf = getStreamBuffer(tabId);
			const flushChannel = (ch: StreamChannelBuffer) => {
				if (ch.flushedIdx < ch.chunks.length) {
					ch.display += ch.chunks.slice(ch.flushedIdx).join('');
					ch.flushedIdx = ch.chunks.length;
				}
			};
			flushChannel(buf.text);
			flushChannel(buf.thinking);
		},
		[getStreamBuffer]
	);

	const getFullStreamTextForTab = useCallback((tabId: string) => {
		const buf = streamBuffersRef.current.get(tabId);
		if (!buf) return '';
		// We flush on notify; treat display as authoritative.
		return buf.text.display;
	}, []);

	const getFullStreamThinkingForTab = useCallback((tabId: string) => {
		const buf = streamBuffersRef.current.get(tabId);
		if (!buf) return '';
		return buf.thinking.display;
	}, []);
	const bumpStreamVersion = useCallback((tabId: string) => {
		const v = (streamVersionRef.current.get(tabId) ?? 0) + 1;
		streamVersionRef.current.set(tabId, v);
	}, []);
	const getStreamVersionSnapshot = useCallback((tabId: string) => {
		return streamVersionRef.current.get(tabId) ?? 0;
	}, []);

	// External-store style streaming subscriptions (only last message subscribes)
	const streamListenersRef = useRef(new Map<string, Set<() => void>>());
	const notifyTimersRef = useRef(new Map<string, number | null>());

	const subscribeToStream = useCallback((tabId: string, cb: () => void) => {
		let set = streamListenersRef.current.get(tabId);
		if (!set) {
			set = new Set();
			streamListenersRef.current.set(tabId, set);
		}
		set.add(cb);

		return () => {
			const s = streamListenersRef.current.get(tabId);
			s?.delete(cb);
			if (s && s.size === 0) streamListenersRef.current.delete(tabId);
		};
	}, []);

	const notifyStreamNow = useCallback(
		(tabId: string) => {
			flushStreamForTab(tabId);
			bumpStreamVersion(tabId);

			const set = streamListenersRef.current.get(tabId);
			if (!set) return;
			for (const cb of set) cb();
		},
		[flushStreamForTab, bumpStreamVersion]
	);

	// Match MessageContentCard debounce (~128ms). Notifying faster just causes wasted work.
	const notifyStreamSoon = useCallback(
		(tabId: string) => {
			if (selectedTabIdRef.current !== tabId) return;

			const existing = notifyTimersRef.current.get(tabId) ?? null;
			if (existing !== null) return;

			const timer = window.setTimeout(() => {
				notifyTimersRef.current.set(tabId, null);
				notifyStreamNow(tabId);
			}, 140);

			notifyTimersRef.current.set(tabId, timer);
		},
		[notifyStreamNow]
	);

	// Input refs per tab (per-tab composer instance)
	const inputRefs = useRef(new Map<string, InputBoxHandle | null>());
	// If a drop arrives before the input handle is available (rare on first mount),
	// keep it and retry shortly.
	type PendingDrop = { tabId: string; payload: AttachmentsDroppedPayload };
	const pendingDropsRef = useRef<PendingDrop[]>([]);

	const tryApplyDropToTab = useCallback((tabId: string, payload: AttachmentsDroppedPayload): boolean => {
		const input = inputRefs.current.get(tabId);
		if (!input) return false;

		// Requires InputBoxHandle.applyAttachmentsDrop to exist (see section 4 below).
		input.applyAttachmentsDrop(payload);
		return true;
	}, []);

	const flushPendingDrops = useCallback(() => {
		const pending = pendingDropsRef.current;
		if (!pending || pending.length === 0) return;

		const remaining: PendingDrop[] = [];
		for (const item of pending) {
			// If tab is gone, drop it (don’t attach to the wrong conversation).
			if (!tabExists(item.tabId)) continue;
			if (!tryApplyDropToTab(item.tabId, item.payload)) {
				remaining.push(item);
			}
		}
		pendingDropsRef.current = remaining;
	}, [tabExists, tryApplyDropToTab]);

	const setInputRef = useCallback(
		(tabId: string) => {
			return (inst: InputBoxHandle | null) => {
				inputRefs.current.set(tabId, inst);
				// If this tab just became available, retry any pending drops.
				if (inst) {
					// Defer a tick so editor mount/focus is stable.
					window.setTimeout(() => {
						flushPendingDrops();
					}, 0);
				}
			};
		},
		[flushPendingDrops]
	);

	// Register exactly one active drop target for the chats page.
	// Always attach to the currently selected tab (open conversation).
	useEffect(() => {
		const unregister = attachmentsDropAPI.registerDropTarget((payload: AttachmentsDroppedPayload) => {
			const tabId = selectedTabIdRef.current;
			if (!tabId || !tabExists(tabId)) {
				// No valid active tab: keep it pending and let it retry after tab restore.
				pendingDropsRef.current.push({ tabId, payload });
				window.setTimeout(() => {
					flushPendingDrops();
				}, 50);
				return;
			}

			const ok = tryApplyDropToTab(tabId, payload);
			if (!ok) {
				pendingDropsRef.current.push({ tabId, payload });
				window.setTimeout(() => {
					flushPendingDrops();
				}, 0);
			}
		});

		return unregister;
	}, [flushPendingDrops, tabExists, tryApplyDropToTab]);

	// Scroll position restore per tab
	const scrollTopByTab = useRef(new Map<string, number>());

	// Seed scroll positions from persisted state (once)
	const seededScrollFromStorageRef = useRef(false);
	if (!seededScrollFromStorageRef.current) {
		seededScrollFromStorageRef.current = true;
		if (initialScrollTopByTab) {
			for (const [id, top] of Object.entries(initialScrollTopByTab)) {
				if (typeof top === 'number') scrollTopByTab.current.set(id, top);
			}
		}
	}

	const disposeTabRuntime = useCallback(
		(tabId: string) => {
			const a = getAbortRef(tabId);
			a.current?.abort();
			a.current = null;

			abortRefs.current.delete(tabId);
			requestIdByTab.current.delete(tabId);
			tokensReceivedByTab.current.delete(tabId);

			streamBuffersRef.current.delete(tabId);
			streamVersionRef.current.delete(tabId);

			inputRefs.current.delete(tabId);
			scrollTopByTab.current.delete(tabId);

			const timer = notifyTimersRef.current.get(tabId);
			if (timer) window.clearTimeout(timer);
			notifyTimersRef.current.delete(tabId);
			streamListenersRef.current.delete(tabId);
		},
		[getAbortRef]
	);

	const clearStreamForTab = useCallback(
		(tabId: string) => {
			clearStreamBuffer(tabId);
			notifyStreamNow(tabId);
		},
		[clearStreamBuffer, notifyStreamNow]
	);

	const syncComposerFromConversation = useCallback((tabId: string, conv: Conversation) => {
		const input = inputRefs.current.get(tabId);
		if (!input) return;

		const tools = deriveConversationToolsFromMessages(conv.messages);
		const web = deriveWebSearchChoiceFromMessages(conv.messages);
		input.setConversationToolsFromChoices(tools);
		input.setWebSearchFromChoices(web);
	}, []);

	const focusInput = useCallback((tabId: string) => inputRefs.current.get(tabId)?.focus(), []);
	const openTemplateMenu = useCallback((tabId: string) => inputRefs.current.get(tabId)?.openTemplateMenu(), []);
	const openToolMenu = useCallback((tabId: string) => inputRefs.current.get(tabId)?.openToolMenu(), []);
	const openAttachmentMenu = useCallback((tabId: string) => inputRefs.current.get(tabId)?.openAttachmentMenu(), []);

	const setScrollTopForTab = useCallback((tabId: string, top: number) => {
		scrollTopByTab.current.set(tabId, top);
	}, []);

	// Abort in-flight streams on unmount (matches original page behavior)
	useEffect(() => {
		return () => {
			try {
				for (const refObj of abortRefs.current.values()) {
					refObj.current?.abort();
				}
			} catch {
				// ignore
			}
			for (const timer of notifyTimersRef.current.values()) {
				if (timer) window.clearTimeout(timer);
			}
		};
	}, []);

	// ---------------- UI refs ----------------
	const chatContainerRef = useRef<HTMLDivElement>(null);
	const { isAtBottom, isAtTop, isScrollable } = useAtTopBottom(chatContainerRef);

	const scrollToBottom = useCallback((tabId: string) => {
		// There is only one visible scroll container; it corresponds to the active tab.
		if (selectedTabIdRef.current !== tabId) return;
		const el = chatContainerRef.current;
		if (!el) return;
		el.scrollTo({ top: el.scrollHeight, behavior: 'smooth' });
	}, []);

	const scrollToBottomSoon = useCallback(
		(tabId: string) => {
			window.setTimeout(() => {
				scrollToBottom(tabId);
			}, 80);
		},
		[scrollToBottom]
	);

	// Save scroll position for active tab on scroll
	const onScrollActive = useCallback(() => {
		const tabId = selectedTabIdRef.current;
		const el = chatContainerRef.current;
		if (!el) return;
		scrollTopByTab.current.set(tabId, el.scrollTop);
	}, []);

	// Restore scroll position when changing tabs (scroll restore requirement)
	useLayoutEffect(() => {
		const el = chatContainerRef.current;
		if (!el) return;

		const top = scrollTopByTab.current.get(selectedTabId) ?? 0;

		// Keep map consistent even if user never scrolls.
		scrollTopByTab.current.set(selectedTabId, top);

		// Use rAF to ensure DOM layout is committed for the new tab content
		requestAnimationFrame(() => {
			const el2 = chatContainerRef.current;
			if (!el2) return;
			el2.scrollTop = top;
		});
	}, [selectedTabId]);

	const resetScrollToTop = useCallback((tabId: string) => {
		scrollTopByTab.current.set(tabId, 0);
		if (selectedTabIdRef.current !== tabId) return;
		requestAnimationFrame(() => {
			const el = chatContainerRef.current;
			if (el) el.scrollTop = 0;
		});
	}, []);

	const getScrollTopByTabSnapshot = useCallback(() => {
		const obj: Record<string, number> = {};
		for (const [k, v] of scrollTopByTab.current.entries()) obj[k] = v;
		return obj;
	}, []);

	// ---------------- Streaming completion (per tab) ----------------
	const updateStreamingMessage = useCallback(
		async (tabId: string, updatedChatWithUserMessage: Conversation, options: UIChatOption) => {
			if (!tabExists(tabId)) return;

			const abortRef = getAbortRef(tabId);

			abortRef.current?.abort();
			tokensReceivedByTab.current.set(tabId, false);

			// mark busy (coarse UI only)
			updateTab(tabId, t => ({ ...t, isBusy: true }));
			let reqId: string;
			try {
				reqId = getUUIDv7();
			} catch {
				reqId = ensureMakeID();
			}

			requestIdByTab.current.set(tabId, reqId);

			const controller = new AbortController();
			abortRef.current = controller;

			let allMessages = updatedChatWithUserMessage.messages;
			if (options.disablePreviousMessages) {
				const last = updatedChatWithUserMessage.messages.at(-1);
				allMessages = last ? [last] : [];
			}
			if (allMessages.length === 0) {
				updateTab(tabId, t => ({ ...t, isBusy: false }));
				return;
			}

			const currentUserMsg = allMessages[allMessages.length - 1];
			const history = allMessages.slice(0, allMessages.length - 1);
			// IMPORTANT: sanitize attachments before sending to completion.
			const effectiveCurrentUserMsg = {
				...currentUserMsg,
				attachments: dedupeAttachmentsByRef(currentUserMsg.attachments),
			};

			// reset stream buffer (ref only)
			clearStreamBuffer(tabId);
			notifyStreamNow(tabId);

			// assistant placeholder for streaming
			const assistantPlaceholder = initConversationMessage(RoleEnum.Assistant);
			const chatWithPlaceholder: Conversation = {
				...updatedChatWithUserMessage,
				messages: [...updatedChatWithUserMessage.messages, assistantPlaceholder],
				modifiedAt: new Date(),
			};

			// Show placeholder immediately (single state update)
			updateTab(tabId, t => ({
				...t,
				conversation: { ...chatWithPlaceholder, messages: [...chatWithPlaceholder.messages] },
			}));

			if (selectedTabIdRef.current === tabId) scrollToBottomSoon(tabId);

			const onStreamTextData = (textData: string) => {
				if (!textData) return;
				if (requestIdByTab.current.get(tabId) !== reqId) return; // stale stream
				tokensReceivedByTab.current.set(tabId, true);

				getStreamBuffer(tabId).text.chunks.push(textData);

				// Only active tab notifies, and throttled.
				notifyStreamSoon(tabId);
			};

			const onStreamThinkingData = (thinkingData: string) => {
				if (!thinkingData) return;
				if (requestIdByTab.current.get(tabId) !== reqId) return; // stale stream
				tokensReceivedByTab.current.set(tabId, true);

				// Keep raw thinking text; render separately in ThinkingFence.
				getStreamBuffer(tabId).thinking.chunks.push(thinkingData);
				notifyStreamSoon(tabId);
			};

			try {
				const inputParams: ModelParam = {
					name: options.name,
					temperature: options.temperature,
					stream: options.stream,
					maxPromptLength: options.maxPromptLength,
					maxOutputLength: options.maxOutputLength,
					reasoning: options.reasoning,
					systemPrompt: options.systemPrompt,
					timeout: options.timeout,
					outputParam: options.outputParam,
					stopSequences: options.stopSequences,
					additionalParametersRawJSON: options.additionalParametersRawJSON,
				};

				let toolStoreChoices: ToolStoreChoice[] | undefined;
				const latestUser = updatedChatWithUserMessage.messages
					.slice()
					.reverse()
					.find(m => m.role === RoleEnum.User);
				if (latestUser?.toolStoreChoices && latestUser.toolStoreChoices.length > 0) {
					toolStoreChoices = latestUser.toolStoreChoices;
				}

				const { responseMessage, rawResponse } = await HandleCompletion(
					options.providerName,
					inputParams,
					effectiveCurrentUserMsg,
					history,
					toolStoreChoices,
					assistantPlaceholder,
					reqId,
					controller.signal,
					onStreamTextData,
					onStreamThinkingData
				);

				if (!tabExists(tabId)) return;
				if (requestIdByTab.current.get(tabId) !== reqId) return; // stale completion

				if (responseMessage) {
					let finalChat: Conversation = {
						...chatWithPlaceholder,
						messages: [...chatWithPlaceholder.messages.slice(0, -1), responseMessage],
						modifiedAt: new Date(),
					};

					if (rawResponse?.hydratedCurrentInputs && currentUserMsg.id) {
						const hydrated = rawResponse.hydratedCurrentInputs;
						finalChat = {
							...finalChat,
							messages: finalChat.messages.map(m =>
								m.id === currentUserMsg.id
									? {
											...m,
											inputs: hydrated,
										}
									: m
							),
						};
					}

					saveUpdatedConversation(tabId, finalChat);

					// Tool calls -> load into THIS tab's composer (even if hidden)
					if (responseMessage.uiToolCalls && responseMessage.uiToolCalls.length > 0) {
						const runnableCalls = responseMessage.uiToolCalls.filter(
							c => c.type === ToolStoreChoiceType.Function || c.type === ToolStoreChoiceType.Custom
						);
						if (runnableCalls.length > 0) {
							inputRefs.current.get(tabId)?.loadToolCalls(runnableCalls);
						}
					}
				}
			} catch (e) {
				if (!tabExists(tabId)) return;
				if (requestIdByTab.current.get(tabId) !== reqId) return; // stale completion path

				if ((e as DOMException).name === 'AbortError') {
					const tokensReceived = tokensReceivedByTab.current.get(tabId);

					if (!tokensReceived) {
						// remove placeholder
						updateTab(tabId, t => {
							const idx = t.conversation.messages.findIndex(m => m.id === assistantPlaceholder.id);
							if (idx === -1) return t;
							const msgs = t.conversation.messages.filter((_, i) => i !== idx);
							return {
								...t,
								conversation: { ...t.conversation, messages: msgs, modifiedAt: new Date() },
							};
						});
					} else {
						const partialText = getFullStreamTextForTab(tabId) + '\n\n>API Aborted after partial response...';

						const partialOutputs: OutputUnion[] = [
							{
								kind: OutputKind.OutputMessage,
								outputMessage: {
									id: assistantPlaceholder.id,
									role: RoleEnum.Assistant,
									status: Status.Completed,
									contents: [{ kind: ContentItemKind.Text, textItem: { text: partialText } }],
								},
							},
						];

						const partialMsg: ConversationMessage = {
							...assistantPlaceholder,
							status: Status.Completed,
							outputs: partialOutputs,
							uiContent: partialText,
						};

						const finalChat: Conversation = {
							...chatWithPlaceholder,
							messages: [...chatWithPlaceholder.messages.slice(0, -1), partialMsg],
							modifiedAt: new Date(),
						};

						saveUpdatedConversation(tabId, finalChat);
					}
				} else {
					console.error(e);
				}
			} finally {
				if (tabExists(tabId)) {
					if (requestIdByTab.current.get(tabId) === reqId) {
						// don't clobber a newer request

						// Clear the buffer, but don't notify: the isBusy=false state update
						// will re-render the row and stop using streamed text anyway.
						clearStreamBuffer(tabId);

						updateTab(tabId, t => ({ ...t, isBusy: false }));

						if (selectedTabIdRef.current === tabId) scrollToBottomSoon(tabId);
					}
				}
			}
		},
		[
			clearStreamBuffer,
			getAbortRef,
			getFullStreamTextForTab,
			getStreamBuffer,
			notifyStreamNow,
			notifyStreamSoon,
			saveUpdatedConversation,
			scrollToBottomSoon,
			tabExists,
			updateTab,
		]
	);

	// ---------------- Per-tab send/edit ----------------
	const sendMessageForTab = useCallback(
		async (tabId: string, payload: EditorSubmitPayload, options: UIChatOption) => {
			const tab = tabsRef.current.find(t => t.tabId === tabId);
			if (!tab) return;
			if (tab.isBusy) return;

			const trimmed = payload.text.trim();
			const hasNonEmptyText = trimmed.length > 0;
			const hasToolOutputs = payload.toolOutputs.length > 0;
			const hasAttachments = payload.attachments.length > 0;
			if (!hasNonEmptyText && !hasToolOutputs && !hasAttachments) return;

			const editingId = tab.editingMessageId ?? undefined;
			const userMsg = buildUserConversationMessageFromEditor(payload, editingId);

			if (tab.editingMessageId) {
				const idx = tab.conversation.messages.findIndex(m => m.id === tab.editingMessageId);
				if (idx !== -1) {
					const oldMsgs = tab.conversation.messages.slice(0, idx);
					const msgs = [...oldMsgs, userMsg];

					const updatedChat: Conversation = {
						...tab.conversation,
						messages: msgs,
						modifiedAt: new Date(),
					};

					updateTab(tabId, t => ({ ...t, editingMessageId: null }));
					saveUpdatedConversation(tabId, updatedChat);

					if (selectedTabIdRef.current === tabId) scrollToBottomSoon(tabId);

					void updateStreamingMessage(tabId, updatedChat, options).catch(console.error);
					return;
				}

				// message vanished -> clear edit state and append normally
				updateTab(tabId, t => ({ ...t, editingMessageId: null }));
			}

			const updated: Conversation = {
				...tab.conversation,
				messages: [...tab.conversation.messages, userMsg],
				modifiedAt: new Date(),
			};

			saveUpdatedConversation(tabId, updated);
			if (selectedTabIdRef.current === tabId) scrollToBottomSoon(tabId);

			void updateStreamingMessage(tabId, updated, options).catch(console.error);
		},
		[saveUpdatedConversation, scrollToBottomSoon, updateStreamingMessage, updateTab]
	);

	const beginEditMessageForTab = useCallback(
		(tabId: string, id: string) => {
			const tab = tabsRef.current.find(t => t.tabId === tabId);
			if (!tab) return;
			if (tab.isBusy) return;

			const msg = tab.conversation.messages.find(m => m.id === id);
			if (!msg) return;
			if (msg.role !== RoleEnum.User) return;

			const external: EditorExternalMessage = {
				text: msg.uiContent ?? '',
				attachments: msg.attachments,
				toolChoices: msg.toolStoreChoices,
				toolOutputs: msg.uiToolOutputs,
			};

			const input = inputRefs.current.get(tabId);
			input?.loadExternalMessage(external);
			input?.focus();

			updateTab(tabId, t => ({ ...t, editingMessageId: id }));
		},
		[updateTab]
	);

	const cancelEditingForTab = useCallback(
		(tabId: string) => {
			updateTab(tabId, t => ({ ...t, editingMessageId: null }));
		},
		[updateTab]
	);
	// ---------------- Expose imperative API to ChatsPage ----------------
	useImperativeHandle(
		ref,
		() => ({
			disposeTabRuntime,
			clearStreamForTab,
			syncComposerFromConversation,
			focusInput,
			openTemplateMenu,
			openToolMenu,
			openAttachmentMenu,
			setScrollTopForTab,
			resetScrollToTop,
			getScrollTopByTabSnapshot,
		}),
		[
			clearStreamForTab,
			disposeTabRuntime,
			focusInput,
			getScrollTopByTabSnapshot,
			openAttachmentMenu,
			openTemplateMenu,
			openToolMenu,
			resetScrollToTop,
			setScrollTopForTab,
			syncComposerFromConversation,
		]
	);

	// ---------------- Render helpers ----------------
	const renderedMessagesExceptLast = useMemo(() => {
		if (!activeTab) return null;
		const msgs = activeTab.conversation.messages;
		if (msgs.length <= 1) return null;

		// IMPORTANT: previous rows are not "busy" just because the tab is generating.
		// This avoids re-mounting heavy Markdown for all prior messages.
		return msgs.slice(0, -1).map(msg => (
			<ChatMessage
				key={msg.id}
				message={msg}
				streamedText={''}
				streamedThinking={''}
				isBusy={false}
				isEditing={activeTab.editingMessageId === msg.id}
				onEdit={() => {
					beginEditMessageForTab(activeTab.tabId, msg.id);
				}}
			/>
		));
	}, [activeTab, beginEditMessageForTab]);

	const renderedLastMessage = useMemo(() => {
		if (!activeTab) return null;
		const msgs = activeTab.conversation.messages;
		if (msgs.length === 0) return null;

		const msg = msgs[msgs.length - 1];
		const isAssistant = msg.role === RoleEnum.Assistant;

		const rowIsBusy = activeTab.isBusy && isAssistant;

		return (
			<StreamingLastMessage
				key={msg.id}
				message={msg}
				rowIsBusy={rowIsBusy}
				isEditing={activeTab.editingMessageId === msg.id}
				onEdit={() => {
					beginEditMessageForTab(activeTab.tabId, msg.id);
				}}
				subscribe={cb => subscribeToStream(activeTab.tabId, cb)}
				getSnapshot={() => getStreamVersionSnapshot(activeTab.tabId)}
				getStreamText={() => getFullStreamTextForTab(activeTab.tabId)}
				getStreamThinking={() => getFullStreamThinkingForTab(activeTab.tabId)}
			/>
		);
	}, [
		activeTab,
		beginEditMessageForTab,
		getFullStreamTextForTab,
		getFullStreamThinkingForTab,
		getStreamVersionSnapshot,
		subscribeToStream,
	]);

	return (
		<>
			{/* Row 2: MESSAGES (single scroll container; scroll position restored per tab) */}
			<div className="relative row-start-2 row-end-3 mt-2 min-h-0">
				<div
					ref={chatContainerRef}
					onScroll={onScrollActive}
					className="relative h-full w-full overflow-y-auto overscroll-contain py-1"
					style={{ scrollbarGutter: 'stable both-edges' }}
				>
					<div className="mx-auto w-11/12 xl:w-5/6">
						{renderedMessagesExceptLast}
						{renderedLastMessage}
					</div>
				</div>

				{/* Overlay scroll buttons (active tab only, since container is shared) */}
				<div className="pointer-events-none absolute right-4 bottom-16 z-10 xl:right-24">
					<div className="pointer-events-auto">
						{isScrollable && !isAtTop ? (
							<ButtonScrollToTop
								scrollContainerRef={chatContainerRef}
								iconSize={32}
								show={isScrollable && !isAtTop}
								className="btn btn-md border-none bg-transparent shadow-none"
							/>
						) : null}
					</div>
				</div>
				<div className="pointer-events-none absolute right-4 bottom-4 z-10 xl:right-24">
					<div className="pointer-events-auto">
						{isScrollable && !isAtBottom ? (
							<ButtonScrollToBottom
								scrollContainerRef={chatContainerRef}
								iconSize={32}
								show={isScrollable && !isAtBottom}
								className="btn btn-md border-none bg-transparent shadow-none"
							/>
						) : null}
					</div>
				</div>
			</div>

			{/* Row 3: INPUT (per tab; all mounted, only active visible) */}
			<div className="row-start-3 row-end-4 flex w-full min-w-0 justify-center">
				<div className="w-11/12 min-w-0 xl:w-5/6">
					{tabs.map(t => (
						<InputPane
							key={t.tabId}
							tabId={t.tabId}
							active={t.tabId === selectedTabId}
							isBusy={t.isBusy}
							editingMessageId={t.editingMessageId}
							setInputRef={setInputRef}
							getAbortRef={getAbortRef}
							shortcutConfig={shortcutConfig}
							sendMessage={sendMessageForTab}
							cancelEditing={cancelEditingForTab}
						/>
					))}
				</div>
			</div>
		</>
	);
});
