import type { ReactNode, Ref } from 'react';
import {
	forwardRef,
	memo,
	startTransition,
	useCallback,
	useEffect,
	useImperativeHandle,
	useLayoutEffect,
	useMemo,
	useRef,
	useState,
	useSyncExternalStore,
} from 'react';

import { FiChevronDown, FiChevronUp } from 'react-icons/fi';

import type { Attachment } from '@/spec/attachment';
import type { Conversation, ConversationMessage } from '@/spec/conversation';
import { RoleEnum } from '@/spec/inference';
import { ToolOutputKind } from '@/spec/tool';

import type { ShortcutConfig } from '@/lib/keyboard_shortcuts';

import { ButtonScrollToBottom, ButtonScrollToTop } from '@/components/button_scroll_top_bottom';

import { TabInputPane } from '@/chats/conversation/conversation_input_pane';
import type { ChatWorkflowStarter } from '@/chats/conversation/starter_intent';
import { useAttachmentsDropTarget } from '@/chats/conversation/use_attachments_drop_target';
import { useInputRegistry } from '@/chats/conversation/use_input_registry';
import { useScrollRestore } from '@/chats/conversation/use_scroll_restore';
import { useSendMessage } from '@/chats/conversation/use_send_message';
import { useStreamingRuntime } from '@/chats/conversation/use_streaming_runtime';
import type { MCPAppModelContextUpdateEventDetail, MCPAppUIMessageEventDetail } from '@/chats/mcpapps/mcp_app_events';
import { MCP_APP_MODEL_CONTEXT_UPDATE_EVENT, MCP_APP_UI_MESSAGE_EVENT } from '@/chats/mcpapps/mcp_app_events';
import { ChatMessage } from '@/chats/messages/message';
import type { ChatTabState } from '@/chats/tabs/tabs_model';

const EMPTY_MESSAGES: ConversationMessage[] = [];
const EMPTY_DIFF_CANDIDATE_PATHS = new Map<string, string[]>();
const MAX_DIFF_CANDIDATE_PATHS = 1024;
const RICH_RENDER_DEFER_MESSAGE_COUNT = 8;
const RICH_RENDER_DEFER_TEXT_LENGTH = 12_000;
const DIFF_MARKDOWN_SIGNAL_PATTERN =
	/```(?:diff|patch|udiff)\b|^diff --git\s|^\*\*\*\s+Begin\s+Patch\s*$|^---\s+.+\n\+\+\+\s+/m;

const ABSOLUTE_PATH_PATTERN = /(?:[A-Za-z]:[\\/][^\s"'`<>|]+|\\\\[^\s"'`<>|]+|\/[^\s"'`<>|]+)/g;
const RELATIVE_PATH_PATTERN = /(?:\.{1,2}[\\/])?(?:[A-Za-z0-9_.@+-]+[\\/]){1,}[A-Za-z0-9_.@+-]+/g;
const PATH_LIKE_KEY_PATTERN =
	/(?:^|[_-])(path|paths|file|files|dir|dirs|directory|directories|root|workspace)(?:$|[_-])/i;

interface CandidatePathSource {
	path: string;
	isDirectory?: boolean;
}

function cleanCandidatePathToken(value: string): string {
	return value
		.trim()
		.replace(/^[`"'<({[]+/, '')
		.replaceAll(/[`"'>)}\].,;:!?]+$/g, '');
}

function normalizeCandidatePathKey(path: string): string {
	return path.trim().replaceAll('\\', '/').replaceAll(/\/+/g, '/');
}

function isAbsoluteCandidatePath(path: string): boolean {
	const normalized = path.trim();
	return normalized.startsWith('/') || /^[A-Za-z]:[\\/]/.test(normalized) || normalized.startsWith('\\\\');
}

function looksLikeDirectoryPath(path: string): boolean {
	const trimmed = path.trim();
	return trimmed.endsWith('/') || trimmed.endsWith('\\');
}

function pathBasename(path: string): string {
	const normalized = path.trim().replaceAll('\\', '/').replaceAll(/\/+$/g, '');
	const index = normalized.lastIndexOf('/');
	return index >= 0 ? normalized.slice(index + 1) : normalized;
}

function pathBasenameLooksLikeFile(path: string): boolean {
	const basename = pathBasename(path);
	return /\.[^./]+$/.test(basename);
}

function resolveActiveTab(tabs: ChatTabState[], selectedTabId: string): ChatTabState | undefined {
	return tabs.find(tab => tab.tabId === selectedTabId) ?? tabs[0];
}

function getMountedInputPaneSignature(
	tabs: ChatTabState[],
	selectedTabId: string,
	mountedInputTabIds: ReadonlySet<string>
): string {
	const parts: string[] = [];

	for (const tab of tabs) {
		const shouldMount = tab.tabId === selectedTabId || tab.isBusy || mountedInputTabIds.has(tab.tabId);
		if (!shouldMount) {
			continue;
		}

		parts.push(
			[
				tab.tabId,
				tab.tabId === selectedTabId ? '1' : '0',
				tab.isBusy ? '1' : '0',
				tab.isHydrating ? '1' : '0',
				tab.editingMessageId ?? '',
			].join('|')
		);
	}

	return parts.join('||');
}

function areConversationAreaPropsEqual(prev: ConversationAreaProps, next: ConversationAreaProps): boolean {
	if (prev.selectedTabId !== next.selectedTabId) {
		return false;
	}
	if (prev.shortcutConfig !== next.shortcutConfig) {
		return false;
	}
	if (prev.initialScrollTopByTab !== next.initialScrollTopByTab) {
		return false;
	}
	if (prev.updateTab !== next.updateTab) {
		return false;
	}
	if (prev.saveUpdatedConversation !== next.saveUpdatedConversation) {
		return false;
	}

	if (resolveActiveTab(prev.tabs, prev.selectedTabId) !== resolveActiveTab(next.tabs, next.selectedTabId)) {
		return false;
	}

	return (
		getMountedInputPaneSignature(prev.tabs, prev.selectedTabId, prev.mountedInputTabIds) ===
		getMountedInputPaneSignature(next.tabs, next.selectedTabId, next.mountedInputTabIds)
	);
}

function stripTrailingCandidateSlashes(path: string): string {
	const normalized = path.trim().replaceAll('\\', '/').replaceAll(/\/+/g, '/');
	if (normalized === '/') {
		return normalized;
	}
	return normalized.replaceAll(/\/+$/g, '');
}

function dirnameCandidatePath(path: string): string {
	const normalized = stripTrailingCandidateSlashes(path);
	const index = normalized.lastIndexOf('/');

	if (index < 0) {
		return '';
	}
	if (index === 0) {
		return '/';
	}
	return normalized.slice(0, index);
}

function formatDirectoryCandidate(path: string): string {
	const normalized = stripTrailingCandidateSlashes(path);
	if (!normalized || normalized.endsWith('/')) {
		return normalized;
	}
	return `${normalized}/`;
}

function pushUniqueCandidatePath(out: string[], seen: Set<string>, path: string): boolean {
	if (out.length >= MAX_DIFF_CANDIDATE_PATHS) {
		return false;
	}

	const cleaned = cleanCandidatePathToken(path);
	if (!cleaned) {
		return false;
	}

	const key = normalizeCandidatePathKey(cleaned);
	if (!key || seen.has(key)) {
		return false;
	}

	seen.add(key);
	out.push(cleaned);
	return true;
}

function appendCandidatePathSource(cumulative: string[], seen: Set<string>, source: CandidatePathSource): string[] {
	const cleaned = cleanCandidatePathToken(source.path);
	if (!cleaned) {
		return cumulative;
	}

	let next = cumulative;
	const push = (path: string) => {
		if (next === cumulative) {
			next = [...cumulative];
		}
		return pushUniqueCandidatePath(next, seen, path);
	};

	push(cleaned);

	const isAbsolute = isAbsoluteCandidatePath(cleaned);
	const directoryHint =
		source.isDirectory || looksLikeDirectoryPath(cleaned) || (isAbsolute && !pathBasenameLooksLikeFile(cleaned));

	if (directoryHint) {
		push(formatDirectoryCandidate(cleaned));
	}

	if (!isAbsolute) {
		return next;
	}

	let dir =
		source.isDirectory || looksLikeDirectoryPath(cleaned)
			? stripTrailingCandidateSlashes(cleaned)
			: dirnameCandidatePath(cleaned);
	let depth = 0;

	while (dir && isAbsoluteCandidatePath(dir) && depth < 12 && next.length < MAX_DIFF_CANDIDATE_PATHS) {
		push(formatDirectoryCandidate(dir));

		const parent = dirnameCandidatePath(dir);
		if (!parent || parent === dir) {
			break;
		}

		dir = parent;
		depth += 1;
	}

	return next;
}

function getAttachmentCandidatePathSources(attachment: Attachment): CandidatePathSource[] {
	const out: CandidatePathSource[] = [];

	if (attachment.fileRef) {
		const path = attachment.fileRef.origPath || attachment.fileRef.path;
		if (path) {
			out.push({ path, isDirectory: attachment.fileRef.isDir });
		}
	}

	if (attachment.imageRef) {
		const path = attachment.imageRef.origPath || attachment.imageRef.path;
		if (path) {
			out.push({ path, isDirectory: attachment.imageRef.isDir });
		}
	}

	if (attachment.contentBlock?.filePath) {
		out.push({ path: attachment.contentBlock.filePath });
	}

	return out;
}

function extractPathMentionsFromText(text: string | undefined): string[] {
	if (!text) {
		return [];
	}

	const out: string[] = [];

	for (const match of text.matchAll(ABSOLUTE_PATH_PATTERN)) {
		const raw = match[0] ?? '';
		const index = match.index ?? 0;
		const previous = index > 0 ? text[index - 1] : '';

		// Avoid extracting the path part of URLs such as https://host/path.
		if (previous === ':' || previous === '/') {
			continue;
		}

		const cleaned = cleanCandidatePathToken(raw);
		if (cleaned) {
			out.push(cleaned);
		}
	}

	for (const match of text.matchAll(RELATIVE_PATH_PATTERN)) {
		const raw = match[0] ?? '';
		const index = match.index ?? 0;
		const previous = index > 0 ? text[index - 1] : '';

		if (previous === '/' || previous === '\\' || previous === ':' || /[A-Za-z0-9_.@+-]/.test(previous)) {
			continue;
		}
		if (/^[a-z][a-z0-9+.-]*:\/\//i.test(raw) || raw.startsWith('www.')) {
			continue;
		}

		const cleaned = cleanCandidatePathToken(raw);
		if (cleaned) {
			out.push(cleaned);
		}
	}

	return out;
}

function valueLooksPathLike(value: string): boolean {
	const cleaned = cleanCandidatePathToken(value);
	return (
		!!cleaned &&
		(isAbsoluteCandidatePath(cleaned) ||
			looksLikeDirectoryPath(cleaned) ||
			cleaned.includes('/') ||
			cleaned.includes('\\') ||
			cleaned.startsWith('./') ||
			cleaned.startsWith('../'))
	);
}

function collectPathLikeStringsFromJSON(value: unknown, out: string[], keyHint = '', depth = 0) {
	if (depth > 8 || value === null || value === undefined) {
		return;
	}

	if (typeof value === 'string') {
		if (PATH_LIKE_KEY_PATTERN.test(keyHint) || valueLooksPathLike(value)) {
			out.push(value);
		}
		return;
	}

	if (Array.isArray(value)) {
		for (const item of value) {
			collectPathLikeStringsFromJSON(item, out, keyHint, depth + 1);
		}
		return;
	}

	if (typeof value === 'object') {
		for (const [key, item] of Object.entries(value)) {
			collectPathLikeStringsFromJSON(item, out, key, depth + 1);
		}
	}
}

function extractPathMentionsFromStructuredText(text: string | undefined): string[] {
	if (!text) {
		return [];
	}

	const out = [...extractPathMentionsFromText(text)];

	try {
		const parsed = JSON.parse(text) as unknown;
		const jsonStrings: string[] = [];
		collectPathLikeStringsFromJSON(parsed, jsonStrings);

		for (const value of jsonStrings) {
			if (valueLooksPathLike(value)) {
				out.push(cleanCandidatePathToken(value));
			}
			out.push(...extractPathMentionsFromText(value));
		}
	} catch {
		// Tool arguments are usually JSON, but may also be plain text.
	}

	return out;
}

interface CachedMessageCandidatePathSources {
	uiContent: ConversationMessage['uiContent'];
	attachments: ConversationMessage['attachments'];
	uiToolCalls: ConversationMessage['uiToolCalls'];
	uiToolOutputs: ConversationMessage['uiToolOutputs'];
	sources: CandidatePathSource[];
}

const MESSAGE_CANDIDATE_PATH_SOURCE_CACHE = new WeakMap<ConversationMessage, CachedMessageCandidatePathSources>();

function getMessageCandidatePathSources(message: ConversationMessage): CandidatePathSource[] {
	const cached = MESSAGE_CANDIDATE_PATH_SOURCE_CACHE.get(message);
	if (
		cached &&
		cached.uiContent === message.uiContent &&
		cached.attachments === message.attachments &&
		cached.uiToolCalls === message.uiToolCalls &&
		cached.uiToolOutputs === message.uiToolOutputs
	) {
		return cached.sources;
	}

	const out: CandidatePathSource[] = [];

	for (const attachment of message.attachments ?? []) {
		out.push(...getAttachmentCandidatePathSources(attachment));
	}

	for (const path of extractPathMentionsFromText(message.uiContent)) {
		out.push({ path });
	}

	for (const call of message.uiToolCalls ?? []) {
		for (const path of extractPathMentionsFromStructuredText(call.arguments)) {
			out.push({ path });
		}
	}

	for (const output of message.uiToolOutputs ?? []) {
		for (const path of extractPathMentionsFromStructuredText(output.arguments)) {
			out.push({ path });
		}
		for (const path of extractPathMentionsFromText(output.summary)) {
			out.push({ path });
		}
		for (const path of extractPathMentionsFromText(output.errorMessage)) {
			out.push({ path });
		}

		for (const item of output.toolOutputs ?? []) {
			if (item.kind === ToolOutputKind.Text && item.textItem?.text) {
				for (const path of extractPathMentionsFromText(item.textItem.text)) {
					out.push({ path });
				}
			}
			if (item.kind === ToolOutputKind.File && item.fileItem?.fileName) {
				out.push({ path: item.fileItem.fileName });
			}
		}
	}

	MESSAGE_CANDIDATE_PATH_SOURCE_CACHE.set(message, {
		uiContent: message.uiContent,
		attachments: message.attachments,
		uiToolCalls: message.uiToolCalls,
		uiToolOutputs: message.uiToolOutputs,
		sources: out,
	});
	return out;
}

function buildDiffCandidatePathsByMessageID(messages: ConversationMessage[]): Map<string, string[]> {
	const byID = new Map<string, string[]>();
	const seen = new Set<string>();
	let cumulative: string[] = [];

	for (let messageIndex = 0; messageIndex < messages.length; messageIndex += 1) {
		const message = messages[messageIndex];
		let next = cumulative;

		for (const source of getMessageCandidatePathSources(message)) {
			if (next.length >= MAX_DIFF_CANDIDATE_PATHS) {
				break;
			}

			next = appendCandidatePathSource(next, seen, source);
		}

		cumulative = next;
		byID.set(message.id, cumulative);
	}

	return byID;
}

function shouldDeferInitialRichRendering(messages: ConversationMessage[]): boolean {
	if (messages.length >= RICH_RENDER_DEFER_MESSAGE_COUNT) {
		return true;
	}

	let textLength = 0;
	for (const message of messages) {
		textLength += message.uiContent?.length ?? 0;
		if (textLength >= RICH_RENDER_DEFER_TEXT_LENGTH) {
			return true;
		}
	}

	return false;
}

function conversationMayContainApplicableDiff(messages: ConversationMessage[]): boolean {
	return messages.some(message => DIFF_MARKDOWN_SIGNAL_PATTERN.test(message.uiContent ?? ''));
}

type MessageItemRenderer = (
	index: number,
	message: ConversationMessage,
	diffCandidatePaths: string[] | undefined,
	deferRichRendering: boolean
) => ReactNode;

const ConversationMessageList = memo(function ConversationMessageList(props: {
	messages: ConversationMessage[];
	renderItem: MessageItemRenderer;
}) {
	const { messages, renderItem } = props;
	const [richRenderingReady, setRichRenderingReady] = useState(() => !shouldDeferInitialRichRendering(messages));

	useEffect(() => {
		if (richRenderingReady) {
			return;
		}

		const frame = window.requestAnimationFrame(() => {
			startTransition(() => {
				setRichRenderingReady(true);
			});
		});

		return () => {
			window.cancelAnimationFrame(frame);
		};
	}, [richRenderingReady]);

	const diffCandidatePathsByMessageID = useMemo(() => {
		if (!richRenderingReady || !conversationMayContainApplicableDiff(messages)) {
			return EMPTY_DIFF_CANDIDATE_PATHS;
		}
		return buildDiffCandidatePathsByMessageID(messages);
	}, [messages, richRenderingReady]);

	return messages.map((message, index) => (
		<div key={message.id} data-chat-message-index={index}>
			{renderItem(index, message, diffCandidatePathsByMessageID.get(message.id), !richRenderingReady)}
		</div>
	));
});

function StreamingLastMessage(props: {
	message: ConversationMessage;
	rowIsBusy: boolean;
	isEditing: boolean;
	deferRichRendering: boolean;
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
			deferRichRendering={props.deferRichRendering}
			onEdit={props.onEdit}
			diffCandidatePaths={props.diffCandidatePaths}
		/>
	);
}

export interface ConversationAreaHandle {
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
}

interface ConversationAreaProps {
	tabs: ChatTabState[];
	selectedTabId: string;
	mountedInputTabIds: ReadonlySet<string>;
	shortcutConfig: ShortcutConfig;
	initialScrollTopByTab?: Record<string, number>;
	updateTab: (tabId: string, updater: (tab: ChatTabState) => ChatTabState) => void;
	saveUpdatedConversation: (tabId: string, updatedConv: Conversation, titleWasExternallyChanged?: boolean) => void;
}

function ConversationAreaInner(
	{
		tabs,
		selectedTabId,
		mountedInputTabIds,
		shortcutConfig,
		initialScrollTopByTab,
		updateTab,
		saveUpdatedConversation,
	}: ConversationAreaProps,
	ref: Ref<ConversationAreaHandle>
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

	useLayoutEffect(() => {
		tabsRef.current = tabs;
	}, [tabs]);

	useLayoutEffect(() => {
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
		loadAssistantTurnForTab,
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
			if (!text) {
				return;
			}

			const tabId = selectedTabIdRef.current;
			if (!tabExists(tabId)) {
				return;
			}

			inputRefs.current.get(tabId)?.loadExternalMessage({ text });
			inputRefs.current.get(tabId)?.focus();
		};

		const handleModelContextUpdate = (event: Event) => {
			const detail = (event as CustomEvent<MCPAppModelContextUpdateEventDetail>).detail;
			if (!detail?.update) {
				return;
			}

			const tabId = selectedTabIdRef.current;
			if (!tabExists(tabId)) {
				return;
			}

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
		loadAssistantTurnForTab,
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
		(
			index: number,
			message: ConversationMessage,
			diffCandidatePaths: string[] | undefined,
			deferRichRendering: boolean
		) => {
			const isLast = index === messageCount - 1;
			const isAssistant = message.role === RoleEnum.Assistant;
			const rowIsBusy = isLast && activeTabIsBusy && isAssistant;

			if (rowIsBusy) {
				return (
					<StreamingLastMessage
						message={message}
						rowIsBusy={true}
						isEditing={activeEditingMessageId === message.id}
						deferRichRendering={deferRichRendering}
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
					deferRichRendering={deferRichRendering}
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
			getActiveStreamSnapshot,
			getActiveStreamText,
			getActiveStreamThinking,
			messageCount,
			subscribeToActiveStream,
		]
	);

	const mountedInputTabs = useMemo(
		() => tabs.filter(tab => tab.tabId === selectedTabId || tab.isBusy || mountedInputTabIds.has(tab.tabId)),
		[mountedInputTabIds, selectedTabId, tabs]
	);

	useEffect(() => {
		const isEditableTarget = (target: EventTarget | null): boolean => {
			if (!(target instanceof HTMLElement)) {
				return false;
			}
			if (target.closest('[data-disable-chat-shortcuts="true"]')) {
				return true;
			}
			if (target.isContentEditable) {
				return true;
			}
			const tag = target.tagName.toLowerCase();
			return tag === 'input' || tag === 'textarea' || tag === 'select';
		};

		const hasOpenModal = () => Boolean(document.querySelector('dialog[open], [role="dialog"][aria-modal="true"]'));

		const onKeyDown = (event: KeyboardEvent) => {
			if (event.defaultPrevented) {
				return;
			}
			if (event.altKey || event.ctrlKey || event.metaKey) {
				return;
			}
			if (event.key !== 'PageUp' && event.key !== 'PageDown') {
				return;
			}
			if (hasOpenModal()) {
				return;
			}
			if (isEditableTarget(event.target)) {
				return;
			}

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
							<ConversationMessageList
								key={`${activeTabId}:${activeTab?.conversation.id ?? ''}`}
								messages={messages}
								renderItem={itemContent}
							/>
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

			<div className="row-start-3 row-end-4 flex min-h-0 w-full min-w-0 items-end justify-center overflow-hidden">
				<div className="max-h-full w-11/12 min-w-0 xl:w-5/6">
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
}

export const ConversationArea = memo(
	forwardRef<ConversationAreaHandle, ConversationAreaProps>(ConversationAreaInner),
	areConversationAreaPropsEqual
);
