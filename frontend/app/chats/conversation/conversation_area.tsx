import { forwardRef, useCallback, useEffect, useImperativeHandle, useMemo, useRef, useSyncExternalStore } from 'react';

import type { Conversation, ConversationMessage } from '@/spec/conversation';
import { RoleEnum } from '@/spec/inference';

import type { ShortcutConfig } from '@/lib/keyboard_shortcuts';

import { ButtonScrollToBottom, ButtonScrollToTop } from '@/components/button_scroll_top_bottom';

import { TabInputPane } from '@/chats/conversation/conversation_input_pane';
import { useAttachmentsDropTarget } from '@/chats/conversation/use_attachments_drop_target';
import { useInputRegistry } from '@/chats/conversation/use_input_registry';
import { useScrollRestore } from '@/chats/conversation/use_scroll_restore';
import { useSendMessage } from '@/chats/conversation/use_send_message';
import { useStreamingRuntime } from '@/chats/conversation/use_streaming_runtime';
import { ChatMessage } from '@/chats/messages/message';
import type { ChatTabState } from '@/chats/tabs/tabs_model';

const EMPTY_MESSAGES: ConversationMessage[] = [];

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
	useSyncExternalStore(props.subscribe, props.getSnapshot, () => 0);

	const streamedText = props.rowIsBusy ? props.getStreamText() : '';
	const streamedThinking = props.rowIsBusy ? props.getStreamThinking() : '';

	return (
		<ChatMessage
			message={props.message}
			onEdit={props.onEdit}
			streamedText={streamedText}
			streamedThinking={streamedThinking}
			isBusy={props.rowIsBusy}
			isEditing={props.isEditing}
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

	setScrollTopForTab: (tabId: string, top: number) => void;
	resetScrollToTop: (tabId: string) => void;
	getScrollTopByTabSnapshot: () => Record<string, number>;
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
		disposeInputRuntime,
	} = useInputRegistry({ tabExists });

	useAttachmentsDropTarget({
		selectedTabIdRef,
		tabExists,
		tryApplyDropToTab,
		queuePendingDrop,
		flushPendingDrops,
	});

	const {
		setScrollContainerRef,
		setScrollContentRef,
		handleScroll,
		isAtBottom,
		isAtTop,
		scrollActiveToTop,
		scrollActiveToBottom,
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
				/>
			);
		},
		[
			activeEditingMessageId,
			activeTabId,
			activeTabIsBusy,
			beginEditMessageForTab,
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

	return (
		<>
			<div className="relative row-start-2 row-end-3 mt-2 min-h-0">
				<div
					ref={setScrollContainerRef}
					onScroll={handleScroll}
					className="h-full w-full overscroll-contain py-1"
					style={{ scrollbarGutter: 'stable both-edges', overflowAnchor: 'none', overflowY: 'auto' }}
				>
					<div ref={setScrollContentRef} className="mx-auto w-11/12 xl:w-5/6">
						{activeTabIsHydrating && messageCount === 0 ? (
							<div className="flex min-h-128 items-center justify-center py-8">
								<span className="loading loading-dots loading-md" aria-label="Loading conversation" />
							</div>
						) : (
							messages.map((message, index) => <div key={message.id}>{itemContent(index, message)}</div>)
						)}
					</div>
				</div>

				<div className="pointer-events-none absolute right-4 bottom-16 z-10 xl:right-24">
					<div className="pointer-events-auto">
						{!isAtTop ? (
							<ButtonScrollToTop
								onScrollToTop={scrollActiveToTop}
								iconSize={32}
								show={!isAtTop}
								className="btn btn-md border-none bg-transparent shadow-none"
							/>
						) : null}
					</div>
				</div>

				<div className="pointer-events-none absolute right-4 bottom-4 z-10 xl:right-24">
					<div className="pointer-events-auto">
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
