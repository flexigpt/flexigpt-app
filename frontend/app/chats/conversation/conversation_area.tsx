import { forwardRef, useCallback, useEffect, useImperativeHandle, useMemo, useRef, useSyncExternalStore } from 'react';

import { FiChevronDown, FiChevronUp } from 'react-icons/fi';

import type { Attachment } from '@/spec/attachment';
import type { Conversation, ConversationMessage } from '@/spec/conversation';
import { RoleEnum } from '@/spec/inference';

import type { ShortcutConfig } from '@/lib/keyboard_shortcuts';

import { ButtonScrollToBottom, ButtonScrollToTop } from '@/components/button_scroll_top_bottom';

import { TabInputPane } from '@/chats/conversation/conversation_input_pane';
import type { ChatWorkflowStarter } from '@/chats/conversation/starter_intent';
import { useAttachmentsDropTarget } from '@/chats/conversation/use_attachments_drop_target';
import { useInputRegistry } from '@/chats/conversation/use_input_registry';
import { useScrollRestore } from '@/chats/conversation/use_scroll_restore';
import { useSendMessage } from '@/chats/conversation/use_send_message';
import { useStreamingRuntime } from '@/chats/conversation/use_streaming_runtime';
import {
	MCP_APP_MODEL_CONTEXT_UPDATE_EVENT,
	MCP_APP_UI_MESSAGE_EVENT,
	type MCPAppModelContextUpdateEventDetail,
	type MCPAppUIMessageEventDetail,
} from '@/chats/mcpapps/mcp_app_events';
import { ChatMessage } from '@/chats/messages/message';
import type { ChatTabState } from '@/chats/tabs/tabs_model';

const EMPTY_MESSAGES: ConversationMessage[] = [];
const MAX_DIFF_CANDIDATE_PATHS = 2048;

function getLocalAttachmentPath(attachment: Attachment): string {
	return (
		attachment.fileRef?.origPath ||
		attachment.fileRef?.path ||
		attachment.imageRef?.origPath ||
		attachment.imageRef?.path ||
		attachment.contentBlock?.filePath ||
		''
	);
}

function normalizeCandidatePathKey(path: string): string {
	return path.trim().replaceAll('\\', '/').replaceAll(/\/+/g, '/');
}

function buildDiffCandidatePathsByMessageID(messages: ConversationMessage[]): Map<string, string[]> {
	const byID = new Map<string, string[]>();
	const seen = new Set<string>();
	let cumulative: string[] = [];

	for (let messageIndex = 0; messageIndex < messages.length; messageIndex += 1) {
		const message = messages[messageIndex];
		let next = cumulative;

		for (const attachment of message.attachments ?? []) {
			if (next.length >= MAX_DIFF_CANDIDATE_PATHS) break;

			const path = getLocalAttachmentPath(attachment).trim();
			if (!path) continue;

			const key = normalizeCandidatePathKey(path);
			if (!key || seen.has(key)) continue;

			seen.add(key);

			if (next === cumulative) {
				next = [...cumulative];
			}

			next.push(path);
		}

		cumulative = next;
		byID.set(message.id, cumulative);
	}

	return byID;
}

function StreamingLastMessage(props: {
	message: ConversationMessage;
	rowIsBusy: boolean;
	isEditing: boolean;
	onEdit: () => void;
	subscribe: (cb: () => void) => () => void;
	getSnapshot: () => number;
	getStreamText: () => string;
	getStreamThinking: () => string;
	diffCandidatePaths?: string[];
}) {
	useSyncExternalStore(props.subscribe, props.getSnapshot, () => 0);

	const streamedText = props.rowIsBusy ? props.getStreamText() : '';
	const streamedThinking = props.rowIsBusy ? props.getStreamThinking() : '';

	return (
		<ChatMessage
			message={props.message}
			streamedText={streamedText}
			streamedThinking={streamedThinking}
			isBusy={props.rowIsBusy}
			isEditing={props.isEditing}
			onEdit={props.onEdit}
			diffCandidatePaths={props.diffCandidatePaths}
		/>
	);
}

export type ConversationAreaHandle = {
	disposeTabRuntime: (tabId: string) => void;
	clearStreamForTab: (tabId: string) => void;
	syncComposerFromConversation: (tabId: string, conv: Conversation) => void;
	resetComposerForNewConversation: (tabId: string) => Promise<void>;

	focusInput: (tabId: string) => void;
	openTemplateMenu: (tabId: string) => void;
	openToolMenu: (tabId: string) => void;
	openAttachmentMenu: (tabId: string) => void;
	openSystemPromptMenu: (tabId: string) => void;
	openSkillsMenu: (tabId: string) => void;
	openMCPMenu: (tabId: string) => void;
	requestStopResponse: (tabId: string) => void;

	setScrollTopForTab: (tabId: string, top: number) => void;
	resetScrollToTop: (tabId: string) => void;
	getScrollTopByTabSnapshot: () => Record<string, number>;
	applyWorkflowStarter: (tabId: string, starter: ChatWorkflowStarter) => Promise<boolean>;
};

type ConversationAreaProps = {
	tabs: ChatTabState[];
	selectedTabId: string;
	mountedInputTabIds: ReadonlySet<string>;
	shortcutConfig: ShortcutConfig;
	initialScrollTopByTab?: Record<string, number>;
	updateTab: (tabId: string, updater: (tab: ChatTabState) => ChatTabState) => void;
	saveUpdatedConversation: (tabId: string, updatedConv: Conversation, titleWasExternallyChanged?: boolean) => void;
};

export const ConversationArea = forwardRef<ConversationAreaHandle, ConversationAreaProps>(function ConversationArea(
	{
		tabs,
		selectedTabId,
		mountedInputTabIds,
		shortcutConfig,
		initialScrollTopByTab,
		updateTab,
		saveUpdatedConversation,
	},
	ref
) {
	const tabsRef = useRef(tabs);
	const selectedTabIdRef = useRef(selectedTabId);

	const activeTab = useMemo(() => tabs.find(tab => tab.tabId === selectedTabId) ?? tabs[0], [tabs, selectedTabId]);
	const activeTabId = activeTab?.tabId ?? '';
	const activeTabIsBusy = activeTab?.isBusy ?? false;
	const activeTabIsHydrating = (activeTab?.isHydrating ?? false) || !(activeTab?.isLoaded ?? true);
	const activeEditingMessageId = activeTab?.editingMessageId ?? null;
	const messages = activeTab?.conversation?.messages ?? EMPTY_MESSAGES;
	const messageCount = messages.length;
	const diffCandidatePathsByMessageID = useMemo(() => buildDiffCandidatePathsByMessageID(messages), [messages]);

	useEffect(() => {
		tabsRef.current = tabs;
	}, [tabs]);

	useEffect(() => {
		selectedTabIdRef.current = selectedTabId;
	}, [selectedTabId]);

	const {
		tabExists,
		getAbortRef,
		getStreamBuffer,
		clearStreamBuffer,
		clearStreamForTab,
		getFullStreamTextForTab,
		getFullStreamThinkingForTab,
		getStreamVersionSnapshot,
		subscribeToStream,
		notifyStreamNow,
		notifyStreamSoon,
		requestIdByTabRef,
		tokensReceivedByTabRef,
		disposeStreamRuntime,
	} = useStreamingRuntime({
		tabs,
		selectedTabIdRef,
	});

	const {
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
	} = useInputRegistry({ tabExists });

	useAttachmentsDropTarget({
		selectedTabIdRef,
		tabExists,
		tryApplyDropToTab,
		queuePendingDrop,
		flushPendingDrops,
	});

	useEffect(() => {
		const handleUIMessage = (event: Event) => {
			const detail = (event as CustomEvent<MCPAppUIMessageEventDetail>).detail;
			const text = detail?.message?.text?.trim();
			if (!text) return;

			const tabId = selectedTabIdRef.current;
			if (!tabExists(tabId)) return;

			inputRefs.current.get(tabId)?.loadExternalMessage({ text });
			inputRefs.current.get(tabId)?.focus();
		};

		const handleModelContextUpdate = (event: Event) => {
			const detail = (event as CustomEvent<MCPAppModelContextUpdateEventDetail>).detail;
			if (!detail?.update) return;

			const tabId = selectedTabIdRef.current;
			if (!tabExists(tabId)) return;

			inputRefs.current.get(tabId)?.appendMCPAppContextUpdate(detail.update);
			inputRefs.current.get(tabId)?.focus();
		};

		window.addEventListener(MCP_APP_UI_MESSAGE_EVENT, handleUIMessage);
		window.addEventListener(MCP_APP_MODEL_CONTEXT_UPDATE_EVENT, handleModelContextUpdate);

		return () => {
			window.removeEventListener(MCP_APP_UI_MESSAGE_EVENT, handleUIMessage);
			window.removeEventListener(MCP_APP_MODEL_CONTEXT_UPDATE_EVENT, handleModelContextUpdate);
		};
	}, [inputRefs, selectedTabIdRef, tabExists]);

	const {
		setScrollContainerRef,
		setScrollContentRef,
		handleScroll,
		isAtBottom,
		isAtTop,
		canJumpToPreviousMessage,
		canJumpToNextMessage,
		scrollActiveToTop,
		scrollActiveToBottom,
		scrollActiveToPreviousMessage,
		scrollActiveToNextMessage,
		scrollActivePageBy,
		scrollTabToBottomSoon,
		setScrollTopForTab,
		resetScrollToTop,
		getScrollTopByTabSnapshot,
		disposeScrollRuntime,
	} = useScrollRestore({
		selectedTabId,
		selectedTabIdRef,
		activeTabIsHydrating,
		messageCount,
		initialScrollTopByTab,
	});

	const { sendMessageForTab, beginEditMessageForTab, cancelEditingForTab } = useSendMessage({
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
	});

	const disposeTabRuntime = useCallback(
		(tabId: string) => {
			disposeStreamRuntime(tabId);
			disposeInputRuntime(tabId);
			disposeScrollRuntime(tabId);
		},
		[disposeInputRuntime, disposeScrollRuntime, disposeStreamRuntime]
	);

	useImperativeHandle(
		ref,
		() => ({
			disposeTabRuntime,
			clearStreamForTab,
			syncComposerFromConversation,
			resetComposerForNewConversation,
			applyWorkflowStarter: applyWorkflowStarterToComposer,
			focusInput,
			openTemplateMenu,
			openToolMenu,
			openAttachmentMenu,
			openSystemPromptMenu,
			openSkillsMenu,
			openMCPMenu,
			requestStopResponse,
			setScrollTopForTab,
			resetScrollToTop,
			getScrollTopByTabSnapshot,
		}),
		[
			applyWorkflowStarterToComposer,
			clearStreamForTab,
			disposeTabRuntime,
			focusInput,
			getScrollTopByTabSnapshot,
			openAttachmentMenu,
			openSystemPromptMenu,
			openSkillsMenu,
			openMCPMenu,
			requestStopResponse,
			openTemplateMenu,
			openToolMenu,
			resetComposerForNewConversation,
			resetScrollToTop,
			setScrollTopForTab,
			syncComposerFromConversation,
		]
	);

	const subscribeToActiveStream = useCallback(
		(cb: () => void) => subscribeToStream(activeTabId, cb),
		[activeTabId, subscribeToStream]
	);

	const getActiveStreamSnapshot = useCallback(
		() => getStreamVersionSnapshot(activeTabId),
		[activeTabId, getStreamVersionSnapshot]
	);

	const getActiveStreamText = useCallback(
		() => getFullStreamTextForTab(activeTabId),
		[activeTabId, getFullStreamTextForTab]
	);

	const getActiveStreamThinking = useCallback(
		() => getFullStreamThinkingForTab(activeTabId),
		[activeTabId, getFullStreamThinkingForTab]
	);

	const itemContent = useCallback(
		(index: number, message: ConversationMessage) => {
			const isLast = index === messageCount - 1;
			const isAssistant = message.role === RoleEnum.Assistant;
			const rowIsBusy = isLast && activeTabIsBusy && isAssistant;
			const diffCandidatePaths = diffCandidatePathsByMessageID.get(message.id);
			if (rowIsBusy) {
				return (
					<StreamingLastMessage
						message={message}
						rowIsBusy={true}
						isEditing={activeEditingMessageId === message.id}
						onEdit={() => {
							beginEditMessageForTab(activeTabId, message.id);
						}}
						subscribe={subscribeToActiveStream}
						getSnapshot={getActiveStreamSnapshot}
						getStreamText={getActiveStreamText}
						getStreamThinking={getActiveStreamThinking}
						diffCandidatePaths={diffCandidatePaths}
					/>
				);
			}

			return (
				<ChatMessage
					message={message}
					streamedText=""
					streamedThinking=""
					isBusy={false}
					isEditing={activeEditingMessageId === message.id}
					onEdit={() => {
						beginEditMessageForTab(activeTabId, message.id);
					}}
					diffCandidatePaths={diffCandidatePaths}
				/>
			);
		},
		[
			activeEditingMessageId,
			activeTabId,
			activeTabIsBusy,
			beginEditMessageForTab,
			diffCandidatePathsByMessageID,
			getActiveStreamSnapshot,
			getActiveStreamText,
			getActiveStreamThinking,
			messageCount,
			subscribeToActiveStream,
		]
	);

	const mountedInputTabs = useMemo(
		() => tabs.filter(tab => tab.tabId === selectedTabId || mountedInputTabIds.has(tab.tabId)),
		[mountedInputTabIds, selectedTabId, tabs]
	);

	useEffect(() => {
		const isEditableTarget = (target: EventTarget | null): boolean => {
			if (!(target instanceof HTMLElement)) return false;
			if (target.closest('[data-disable-chat-shortcuts="true"]')) return true;
			if (target.isContentEditable) return true;
			const tag = target.tagName.toLowerCase();
			return tag === 'input' || tag === 'textarea' || tag === 'select';
		};

		const hasOpenModal = () => Boolean(document.querySelector('dialog[open], [role="dialog"][aria-modal="true"]'));

		const onKeyDown = (event: KeyboardEvent) => {
			if (event.defaultPrevented) return;
			if (event.altKey || event.ctrlKey || event.metaKey) return;
			if (event.key !== 'PageUp' && event.key !== 'PageDown') return;
			if (hasOpenModal()) return;
			if (isEditableTarget(event.target)) return;

			event.preventDefault();
			scrollActivePageBy(event.key === 'PageDown' ? 1 : -1);
		};

		window.addEventListener('keydown', onKeyDown);
		return () => {
			window.removeEventListener('keydown', onKeyDown);
		};
	}, [scrollActivePageBy]);

	return (
		<>
			<div className="relative row-start-2 row-end-3 mt-2 min-h-0">
				<div
					ref={setScrollContainerRef}
					onScroll={handleScroll}
					className="size-full overscroll-contain py-1"
					style={{ scrollbarGutter: 'stable both-edges', overflowAnchor: 'none', overflowY: 'auto' }}
				>
					<div ref={setScrollContentRef} className="mx-auto w-11/12 xl:w-5/6">
						{activeTabIsHydrating && messageCount === 0 ? (
							<div className="flex min-h-128 items-center justify-center py-8">
								<span className="loading loading-dots loading-md" aria-label="Loading conversation" />
							</div>
						) : (
							messages.map((message, index) => (
								<div key={message.id} data-chat-message-index={index}>
									{itemContent(index, message)}
								</div>
							))
						)}
					</div>
				</div>

				<div className="pointer-events-none absolute right-4 bottom-4 z-10 xl:right-24">
					<div className="pointer-events-auto flex flex-col items-center gap-2">
						{!isAtTop ? (
							<ButtonScrollToTop
								onScrollToTop={scrollActiveToTop}
								iconSize={32}
								show={!isAtTop}
								className="btn btn-md border-none bg-transparent shadow-none"
							/>
						) : null}
						{canJumpToPreviousMessage ? (
							<button
								type="button"
								className="btn btn-md border-none bg-transparent shadow-none"
								onClick={scrollActiveToPreviousMessage}
								aria-label="Jump to previous message"
								title="Jump to previous message"
							>
								<FiChevronUp size={32} />
							</button>
						) : null}

						{canJumpToNextMessage ? (
							<button
								type="button"
								className="btn btn-md border-none bg-transparent shadow-none"
								onClick={scrollActiveToNextMessage}
								aria-label="Jump to next message"
								title="Jump to next message"
							>
								<FiChevronDown size={32} />
							</button>
						) : null}
						{!isAtBottom ? (
							<ButtonScrollToBottom
								onScrollToBottom={scrollActiveToBottom}
								iconSize={32}
								show={!isAtBottom}
								className="btn btn-md border-none bg-transparent shadow-none"
							/>
						) : null}
					</div>
				</div>
			</div>

			<div className="row-start-3 row-end-4 flex w-full min-w-0 justify-center">
				<div className="w-11/12 min-w-0 xl:w-5/6">
					{mountedInputTabs.map(tab => (
						<TabInputPane
							key={tab.tabId}
							tabId={tab.tabId}
							active={tab.tabId === selectedTabId}
							isBusy={tab.isBusy}
							isHydrating={tab.isHydrating || !tab.isLoaded}
							editingMessageId={tab.editingMessageId}
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
