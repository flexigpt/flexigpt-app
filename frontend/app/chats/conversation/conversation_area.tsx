import type { ComponentProps, ReactNode, Ref } from 'react';
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
} from 'react';
import { FiChevronDown, FiChevronUp, FiEdit, FiMinimize2, FiPause, FiPlay, FiSquare } from 'react-icons/fi';

import type { Attachment } from '@/spec/attachment';
import type { UIToolCall } from '@/spec/inference';
import type { Conversation, ConversationMessage, SessionConfiguration, SessionRun } from '@/spec/session';
import type { SessionOpaqueJSON, SessionWaitingRequest, SessionWaitResolution } from '@/spec/session_base';
import { RoleEnum, UIToolCallStatus } from '@/spec/inference';
import { MCPExecutionMode } from '@/spec/mcp';
import { SessionAutomationHoldScope, SessionOperationState } from '@/spec/session_base';
import { ToolOutputKind } from '@/spec/tool';

import type { ShortcutConfig } from '@/lib/keyboard_shortcuts';

import { sessionManagementAPI } from '@/apis/baseapi';

import { ButtonScrollToBottom, ButtonScrollToTop } from '@/components/button_scroll_top_bottom';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';
import { HoverTip } from '@/components/hover_tip';

import type { ComposerBoxHandle } from '@/chats/composer/composer_box';
import type { ChatWorkflowStarter } from '@/chats/conversation/starter_intent';
import type { MCPAppModelContextUpdateEventDetail, MCPAppUIMessageEventDetail } from '@/chats/mcpapps/mcp_app_events';
import type { AssistantResponseTiming } from '@/chats/messages/message_footer';
import type { ChatTabState } from '@/chats/tabs/tabs_model';
import { TabInputPane } from '@/chats/conversation/conversation_input_pane';
import { useAttachmentsDropTarget } from '@/chats/conversation/use_attachments_drop_target';
import { useInputRegistry } from '@/chats/conversation/use_input_registry';
import { useScrollRestore } from '@/chats/conversation/use_scroll_restore';
import { useSendMessage } from '@/chats/conversation/use_send_message';
import { MCP_APP_MODEL_CONTEXT_UPDATE_EVENT, MCP_APP_UI_MESSAGE_EVENT } from '@/chats/mcpapps/mcp_app_events';
import { ChatMessage } from '@/chats/messages/message';
import {
	requireAcceptedSessionCommand,
	selectedSessionRun,
	sessionCommandMeta,
} from '@/chats/session/session_commands';
import { getSessionRunComposerState } from '@/chats/session/session_composer_state';
import { projectSessionRunToolActivity } from '@/chats/session/session_projection';
import { SessionWaitModal } from '@/chats/session/session_wait_modal';

const EMPTY_MESSAGES: ConversationMessage[] = [];
const EMPTY_DIFF_CANDIDATE_PATHS = new Map<string, string[]>();
const EMPTY_TOOL_CALLS: UIToolCall[] = [];
const MAX_DIFF_CANDIDATE_PATHS = 1024;
const RICH_RENDER_DEFER_MESSAGE_COUNT = 4;
const RICH_RENDER_DEFER_TEXT_LENGTH = 4_000;
const RICH_RENDER_BATCH_SIZE = 1;
const RICH_RENDER_IDLE_TIMEOUT_MS = 250;
const FLOATING_COMPOSER_SCROLL_CLEARANCE = 24;

const DIFF_MARKDOWN_SIGNAL_PATTERN =
	/```(?:diff|patch|udiff)\b|^diff --git\s|^Index:\s+|^@@|^\*\*\*\s+(?:Begin Patch|(?:Add|Update|Delete)\s+File:)\s*$|^---\s+.+\n\+\+\+\s+/im;

function textContainsDiffPayload(text: string | undefined): boolean {
	return !!text && DIFF_MARKDOWN_SIGNAL_PATTERN.test(text);
}

const ABSOLUTE_PATH_PATTERN = /(?:[A-Za-z]:[\\/][^\s"'`<>|]+|\\\\[^\s"'`<>|]+|\/[^\s"'`<>|]+)/g;
const PATH_LIKE_KEY_PATTERN =
	/(?:^|[_-])(path|paths|file|files|dir|dirs|directory|directories|root|workspace)(?:$|[_-])/i;
const EMBEDDED_POSIX_PATH_PREFIX_PATTERN = /[A-Za-z0-9_%+.-]/;

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

function resolveActiveTab(tabs: ChatTabState[], selectedTabId: string): ChatTabState | undefined {
	return tabs.find(tab => tab.tabId === selectedTabId) ?? tabs[0];
}

function runHasGeneratingModelOperation(run: SessionRun | undefined): boolean {
	if (!run?.activeModelOperationID) {
		return false;
	}

	const operation = run.modelOperations?.find(candidate => candidate.id === run.activeModelOperationID);

	return (
		operation?.state === SessionOperationState.Queued ||
		operation?.state === SessionOperationState.Preparing ||
		operation?.state === SessionOperationState.Running
	);
}

function runHasAutomaticToolConfiguration(run: SessionRun | undefined): boolean {
	return (
		run?.modelOperations?.some(operation => {
			const configuration = operation.request.configuration;
			const hasAutomaticTool = configuration.tools?.some(override => override.value?.autoExecute === true) ?? false;
			const hasAutomaticMCPTool =
				configuration.mcp?.some(override => {
					return override.value?.selectedTools?.some(tool => tool.executionMode === MCPExecutionMode.Auto) ?? false;
				}) ?? false;

			return hasAutomaticTool || hasAutomaticMCPTool;
		}) ?? false
	);
}

function sessionActionErrorMessage(error: unknown): string {
	if (error instanceof Error && error.message.trim()) {
		return error.message;
	}

	return 'The Session Runtime could not accept this action.';
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
				(tab.isLoaded ?? true) ? '1' : '0',
				tab.editingMessageId ?? '',
			].join('|')
		);
	}

	return parts.join('||');
}

function areConversationAreaPropsEqual(prev: ConversationAreaProps, next: ConversationAreaProps): boolean {
	if (prev.tabs !== next.tabs || prev.mountedInputTabIds !== next.mountedInputTabIds) {
		return false;
	}
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
	if (prev.openSessionForMCPAppMessage !== next.openSessionForMCPAppMessage) {
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
	if (!cleaned || !isAbsoluteCandidatePath(cleaned)) {
		return cumulative;
	}

	let next = cumulative;
	const push = (path: string) => {
		if (next === cumulative) {
			next = [...cumulative];
		}
		return pushUniqueCandidatePath(next, seen, path);
	};

	const isDirectory = source.isDirectory === true || looksLikeDirectoryPath(cleaned);
	if (isDirectory) {
		push(formatDirectoryCandidate(cleaned));
		return next;
	}

	push(cleaned);

	// The containing directory is grounded by the supplied file path. Do not
	// turn every ancestor, including the filesystem root, into a workspace hint.
	const parent = dirnameCandidatePath(cleaned);
	if (parent && isAbsoluteCandidatePath(parent)) {
		push(formatDirectoryCandidate(parent));
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
	if (!text || textContainsDiffPayload(text)) {
		return [];
	}

	// A patch's file headers are identifiers, not trusted local filesystem
	// evidence. In particular, do not let appendCandidatePathSource derive
	// parent directories from the patch body itself.
	const out: string[] = [];

	for (const match of text.matchAll(ABSOLUTE_PATH_PATTERN)) {
		const raw = match[0] ?? '';
		const index = match.index ?? 0;
		const previous = index > 0 ? text[index - 1] : '';

		// Avoid URLs and absolute-looking suffixes inside tokens such as
		// `10/foo`, `+10/foo`, and `%q/foo`.
		if (
			previous === ':' ||
			previous === '/' ||
			previous === '\\' ||
			(raw.startsWith('/') && EMBEDDED_POSIX_PATH_PREFIX_PATTERN.test(previous))
		) {
			continue;
		}

		const cleaned = cleanCandidatePathToken(raw);
		if (cleaned && isAbsoluteCandidatePath(cleaned)) {
			out.push(cleaned);
		}
	}

	return out;
}

function valueLooksPathLike(value: string): boolean {
	const cleaned = cleanCandidatePathToken(value);
	if (!cleaned || textContainsDiffPayload(value)) {
		return false;
	}

	return isAbsoluteCandidatePath(cleaned);
}

function collectPathLikeStringsFromJSON(value: unknown, out: string[], keyHint = '', depth = 0) {
	if (depth > 8 || value === null || value === undefined) {
		return;
	}

	if (typeof value === 'string') {
		if (textContainsDiffPayload(value)) {
			return;
		}

		if (PATH_LIKE_KEY_PATTERN.test(keyHint) && valueLooksPathLike(value)) {
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

	try {
		const parsed = JSON.parse(text) as unknown;
		const out: string[] = [];
		const jsonStrings: string[] = [];
		collectPathLikeStringsFromJSON(parsed, jsonStrings);

		for (const value of jsonStrings) {
			if (valueLooksPathLike(value)) {
				out.push(cleanCandidatePathToken(value));
			}
			out.push(...extractPathMentionsFromText(value));
		}

		return out;
	} catch {
		// Tool arguments are usually JSON, but may also be plain text.
		return extractPathMentionsFromText(text);
	}
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

	// User-authored absolute paths are explicit evidence. Assistant prose is
	// not filesystem evidence and must not seed later diff target inference.
	if (message.role === RoleEnum.User) {
		for (const path of extractPathMentionsFromText(message.uiContent)) {
			out.push({ path });
		}
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

		for (const item of output.toolOutputs ?? []) {
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
		// Keep an entry for every message, not only messages that already
		// contain a diff. A streamed assistant placeholder can gain a diff
		// fence after this work has completed, and it must receive the
		// cumulative path evidence that existed before that message.
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

function scheduleDeferredUIWork(work: () => void): () => void {
	if (typeof window.requestIdleCallback === 'function') {
		const requestId = window.requestIdleCallback(
			() => {
				work();
			},
			{ timeout: RICH_RENDER_IDLE_TIMEOUT_MS }
		);

		return () => {
			window.cancelIdleCallback(requestId);
		};
	}

	const timer = window.setTimeout(work, 16);
	return () => {
		window.clearTimeout(timer);
	};
}

type MessageItemRenderer = (
	index: number,
	message: ConversationMessage,
	diffCandidatePaths: string[] | undefined,
	diffCandidatePathsReady: boolean,
	deferRichRendering: boolean
) => ReactNode;

function assistantResponseTimingByMessageID(
	snapshot: ChatTabState['sessionSnapshot']
): Map<string, AssistantResponseTiming> {
	const byOperationID = new Map<string, AssistantResponseTiming>();
	const byMessageID = new Map<string, AssistantResponseTiming>();

	for (const run of snapshot?.runs ?? []) {
		for (const operation of run.modelOperations ?? []) {
			if (!operation.startedAt && !operation.finishedAt) {
				continue;
			}

			const timing: AssistantResponseTiming = {
				startedAt: operation.startedAt,
				finishedAt: operation.finishedAt,
			};
			byOperationID.set(operation.id, timing);
			byMessageID.set(`session-stream:${run.id}:${operation.id}`, timing);
		}
	}

	for (const message of snapshot?.session.transcript ?? []) {
		const timing = byOperationID.get(message.operationID ?? '');
		if (timing) {
			byMessageID.set(message.id, timing);
		}
	}

	return byMessageID;
}

interface ConversationMessageRowProps extends Omit<ComponentProps<typeof ChatMessage>, 'onEdit'> {
	tabId: string;
	onEditMessage: (tabId: string, messageID: string) => void;
}

const ConversationMessageRow = memo(function ConversationMessageRow({
	tabId,
	onEditMessage,
	...messageProps
}: ConversationMessageRowProps) {
	const messageID = messageProps.message.id;
	const onEdit = useCallback(() => {
		onEditMessage(tabId, messageID);
	}, [messageID, onEditMessage, tabId]);
	return <ChatMessage {...messageProps} onEdit={onEdit} />;
});

const ConversationMessageList = memo(function ConversationMessageList(props: {
	messages: ConversationMessage[];
	renderItem: MessageItemRenderer;
	isBusy: boolean;
	deferRichWork: boolean;
}) {
	const { messages, renderItem, isBusy, deferRichWork } = props;

	const [richReadyCount, setRichReadyCount] = useState(() => {
		if (deferRichWork || shouldDeferInitialRichRendering(messages)) {
			return 0;
		}
		return Math.max(0, messages.length - (isBusy ? 1 : 0));
	});

	const richRenderTargetCount = deferRichWork ? 0 : Math.max(0, messages.length - (isBusy ? 1 : 0));

	useEffect(() => {
		if (richReadyCount >= richRenderTargetCount) {
			return;
		}

		return scheduleDeferredUIWork(() => {
			startTransition(() => {
				setRichReadyCount(current => Math.min(richRenderTargetCount, current + RICH_RENDER_BATCH_SIZE));
			});
		});
	}, [richReadyCount, richRenderTargetCount]);

	const [diffCandidatePathsState, setDiffCandidatePathsState] = useState<Map<string, string[]> | null>(null);
	const messagesRef = useRef(messages);
	const canBuildDiffCandidatePaths = !deferRichWork;
	const canBuildDiffCandidatePathsRef = useRef(canBuildDiffCandidatePaths);
	const diffCandidatePathBuildCancelRef = useRef<(() => void) | null>(null);
	const diffCandidatePathBuildMountedRef = useRef(true);

	useLayoutEffect(() => {
		messagesRef.current = messages;
		canBuildDiffCandidatePathsRef.current = canBuildDiffCandidatePaths;
	}, [canBuildDiffCandidatePaths, messages]);

	const scheduleDiffCandidatePathBuild = useCallback(() => {
		if (diffCandidatePathBuildCancelRef.current) {
			return;
		}

		diffCandidatePathBuildCancelRef.current = scheduleDeferredUIWork(() => {
			diffCandidatePathBuildCancelRef.current = null;

			if (!diffCandidatePathBuildMountedRef.current || !canBuildDiffCandidatePathsRef.current) {
				return;
			}

			// Do not capture a streamed snapshot in the scheduled callback.
			// The next idle slot must process the newest available messages.
			const latestMessages = messagesRef.current;
			const nextPaths = conversationMayContainApplicableDiff(latestMessages)
				? buildDiffCandidatePathsByMessageID(latestMessages)
				: null;

			startTransition(() => {
				if (diffCandidatePathBuildMountedRef.current && canBuildDiffCandidatePathsRef.current) {
					setDiffCandidatePathsState(nextPaths);
				}
			});
		});
	}, []);

	useEffect(() => {
		diffCandidatePathBuildMountedRef.current = true;

		return () => {
			diffCandidatePathBuildMountedRef.current = false;
			diffCandidatePathBuildCancelRef.current?.();
			diffCandidatePathBuildCancelRef.current = null;
		};
	}, []);

	useEffect(() => {
		if (!canBuildDiffCandidatePaths) {
			diffCandidatePathBuildCancelRef.current?.();
			diffCandidatePathBuildCancelRef.current = null;
			// oxlint-disable-next-line react/set-state-in-effect react-you-might-not-need-an-effect/no-adjust-state-on-prop-change
			setDiffCandidatePathsState(null);
			return;
		}

		if (!conversationMayContainApplicableDiff(messages)) {
			diffCandidatePathBuildCancelRef.current?.();
			diffCandidatePathBuildCancelRef.current = null;
			// oxlint-disable-next-line react-you-might-not-need-an-effect/no-adjust-state-on-prop-change
			setDiffCandidatePathsState(null);
			return;
		}

		// A stream can replace messages many times before an idle callback
		// executes. Keep one scheduled build alive and let it read messagesRef.
		// oxlint-disable-next-line react-you-might-not-need-an-effect/no-derived-state
		scheduleDiffCandidatePathBuild();
	}, [canBuildDiffCandidatePaths, messages, scheduleDiffCandidatePathBuild]);

	const diffCandidatePathsByMessageID = diffCandidatePathsState ?? EMPTY_DIFF_CANDIDATE_PATHS;

	return messages.map((message, index) => (
		<div key={message.id} data-chat-message-index={index}>
			{renderItem(
				index,
				message,
				diffCandidatePathsByMessageID.get(message.id),
				diffCandidatePathsState?.has(message.id) ?? false,
				deferRichWork || index >= richReadyCount || (isBusy && index === messages.length - 1)
			)}
		</div>
	));
});

export interface ConversationAreaHandle {
	disposeTabRuntime: (tabId: string) => void;
	syncComposerFromConversation: (tabId: string, conv: Conversation) => void;
	resetComposerForNewConversation: (tabId: string) => Promise<void>;
	hideComposer: (tabId: string) => void;
	restoreComposerFromSessionConfiguration: (tabId: string, configuration: SessionConfiguration) => Promise<void>;

	focusInput: (tabId: string) => void;
	openTemplateMenu: (tabId: string) => void;
	openToolMenu: (tabId: string) => void;
	openAttachmentMenu: (tabId: string) => void;
	openSystemPromptMenu: (tabId: string) => void;
	openSkillsMenu: (tabId: string) => void;
	openMCPMenu: (tabId: string) => void;
	openWorkspaceMenu: (tabId: string) => void;

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
	openSessionForMCPAppMessage: (sessionID: string) => Promise<string | null>;
}

interface WaitResolutionState {
	waitID: string;
	isResolving: boolean;
	error?: string;
}

interface SessionActionErrorState {
	tabId: string;
	message: string;
}

interface RunCancellationConfirmation {
	tabId: string;
	runID: string;
}

function isCurrentRunCancellationConfirmation(
	confirmation: RunCancellationConfirmation | null,
	tabs: ChatTabState[]
): boolean {
	if (!confirmation) {
		return false;
	}

	const tab = tabs.find(candidate => candidate.tabId === confirmation.tabId);
	const snapshot = tab?.sessionSnapshot;
	const run = snapshot ? selectedSessionRun(snapshot, tab?.activeRunID) : undefined;

	return run?.id === confirmation.runID && getSessionRunComposerState(run).isActive;
}

function ConversationAreaInner(
	{
		tabs,
		selectedTabId,
		mountedInputTabIds,
		shortcutConfig,
		initialScrollTopByTab,
		updateTab,
		openSessionForMCPAppMessage,
	}: ConversationAreaProps,
	ref: Ref<ConversationAreaHandle>
) {
	const tabsRef = useRef(tabs);
	const selectedTabIdRef = useRef(selectedTabId);
	const tabIDSetRef = useRef(new Set(tabs.map(tab => tab.tabId)));

	const activeTab = useMemo(() => tabs.find(tab => tab.tabId === selectedTabId) ?? tabs[0], [tabs, selectedTabId]);
	const activeTabId = activeTab?.tabId ?? '';
	const activeTabIsHydrating = (activeTab?.isHydrating ?? false) || !(activeTab?.isLoaded ?? true);
	const activeEditingMessageId = activeTab?.editingMessageId ?? null;
	const activeRun = activeTab?.sessionSnapshot
		? selectedSessionRun(activeTab.sessionSnapshot, activeTab.activeRunID)
		: undefined;
	const pendingRunAcknowledgement =
		activeTab?.pendingRunID !== undefined &&
		!activeTab.sessionSnapshot?.runs.some(run => run.id === activeTab.pendingRunID);
	const activeTabIsGenerating = runHasGeneratingModelOperation(activeRun);
	const [waitResolutionState, setWaitResolutionState] = useState<WaitResolutionState | null>(null);
	const [sessionActionErrorState, setSessionActionErrorState] = useState<SessionActionErrorState | null>(null);
	const sessionActionsInFlightRef = useRef(new Set<string>());
	const sessionActionError = sessionActionErrorState?.tabId === activeTabId ? sessionActionErrorState.message : null;
	const messages = activeTab?.conversation?.messages ?? EMPTY_MESSAGES;
	const messageCount = messages.length;

	const responseTimingByMessageID = useMemo(
		() => assistantResponseTimingByMessageID(activeTab?.sessionSnapshot),
		[activeTab?.sessionSnapshot]
	);

	const activeSessionToolActivity = useMemo(
		() =>
			activeTab?.sessionSnapshot ? projectSessionRunToolActivity(activeTab?.sessionSnapshot, activeRun) : undefined,
		[activeRun, activeTab?.sessionSnapshot]
	);

	const activeRunComposerState = getSessionRunComposerState(
		activeRun,
		(activeSessionToolActivity?.outputs.length ?? 0) > 0
	);
	const { pendingWait, pendingExecutableToolCallIDs, canRunPendingToolCalls } = activeRunComposerState;
	const hasAutomaticToolCalls = activeRun?.toolCalls?.some(call => call.automatic === true) ?? false;
	const hasAutomaticToolConfiguration = runHasAutomaticToolConfiguration(activeRun);
	const automationHoldEnabled =
		activeRun?.controls.automaticTools.enabled === true || activeRun?.controls.automaticContinuation.enabled === true;
	const activeConversationContentIsHydrating = activeTabIsHydrating && messageCount === 0;
	const activeSessionToolCalls = activeSessionToolActivity?.calls ?? EMPTY_TOOL_CALLS;
	const [composerVisibilityByTab, setComposerVisibilityByTab] = useState<Record<string, boolean>>({});
	const [composerPanelElement, setComposerPanelElement] = useState<HTMLDivElement | null>(null);
	const [composerPanelHeight, setComposerPanelHeight] = useState(0);
	const [runCancellationConfirmation, setRunCancellationConfirmation] = useState<RunCancellationConfirmation | null>(
		null
	);
	const [dismissedManualComposerRequirementKeyByTab, setDismissedManualComposerRequirementKeyByTab] = useState<
		Record<string, string>
	>({});
	const composerToggleButtonRef = useRef<HTMLButtonElement | null>(null);
	const runCancellationConfirmationOpen = isCurrentRunCancellationConfirmation(runCancellationConfirmation, tabs);

	const activeTabStartsInWritingMode =
		activeTab !== undefined && !activeTab.sessionID && !activeTab.isHydrating && messageCount === 0;
	const composerVisibilityOverride = activeTabId ? composerVisibilityByTab[activeTabId] : undefined;
	const defaultComposerVisible = activeTabStartsInWritingMode;

	useLayoutEffect(() => {
		tabsRef.current = tabs;
		tabIDSetRef.current = new Set(tabs.map(tab => tab.tabId));
	}, [tabs]);

	useLayoutEffect(() => {
		selectedTabIdRef.current = selectedTabId;
	}, [selectedTabId]);

	useLayoutEffect(() => {
		if (!composerPanelElement || typeof ResizeObserver === 'undefined') {
			return;
		}

		const observer = new ResizeObserver(() => {
			const nextHeight = Math.ceil(composerPanelElement.getBoundingClientRect().height);
			setComposerPanelHeight(current => (current === nextHeight ? current : nextHeight));
		});

		observer.observe(composerPanelElement);

		return () => {
			observer.disconnect();
		};
	}, [composerPanelElement]);

	const manualToolActionCalls = useMemo(() => {
		if (activeTabIsHydrating || activeEditingMessageId || !activeRunComposerState.isActive) {
			return EMPTY_TOOL_CALLS;
		}

		return activeSessionToolCalls.filter(call => {
			if (call.status === UIToolCallStatus.Failed) {
				return !activeTabIsGenerating;
			}

			return (
				activeRunComposerState.canRunPendingToolCalls &&
				call.status === UIToolCallStatus.Pending &&
				(!call.automatic || automationHoldEnabled)
			);
		});
	}, [
		activeEditingMessageId,
		activeRunComposerState.canRunPendingToolCalls,
		activeRunComposerState.isActive,
		activeSessionToolCalls,
		activeTabIsGenerating,
		activeTabIsHydrating,
		automationHoldEnabled,
	]);

	const hasContinuationAction =
		!activeTabIsHydrating &&
		!activeEditingMessageId &&
		!activeTabIsGenerating &&
		activeRunComposerState.canContinueWithoutDraft;

	const activeManualComposerRequirementKey = useMemo(() => {
		const requirements = manualToolActionCalls.map(call => `tool:${call.id}:${call.status}`);

		if (hasContinuationAction) {
			requirements.push(`continue:${activeRun?.continuation.revision ?? ''}`);
		}

		return requirements.length > 0 ? `${activeRun?.id ?? ''}:${requirements.join('|')}` : undefined;
	}, [activeRun, hasContinuationAction, manualToolActionCalls]);

	const dismissedManualComposerRequirementKey = activeTabId
		? dismissedManualComposerRequirementKeyByTab[activeTabId]
		: undefined;
	const shouldAutoRevealComposer =
		activeManualComposerRequirementKey !== undefined &&
		dismissedManualComposerRequirementKey !== activeManualComposerRequirementKey;

	const composerVisible =
		activeTab !== undefined &&
		(composerVisibilityOverride === true ||
			(composerVisibilityOverride === undefined && defaultComposerVisible) ||
			shouldAutoRevealComposer);

	const setComposerVisibleForTab = useCallback(
		(tabId: string, visible: boolean) => {
			if (!tabId) {
				return;
			}

			if (!visible && tabId === activeTabId && activeManualComposerRequirementKey) {
				setDismissedManualComposerRequirementKeyByTab(previous => {
					if (previous[tabId] === activeManualComposerRequirementKey) {
						return previous;
					}

					return {
						...previous,
						[tabId]: activeManualComposerRequirementKey,
					};
				});
			}

			setComposerVisibilityByTab(previous => {
				if (previous[tabId] === visible) {
					return previous;
				}

				return {
					...previous,
					[tabId]: visible,
				};
			});
		},
		[activeManualComposerRequirementKey, activeTabId]
	);

	const hideComposerAfterSubmission = useCallback(
		(tabId: string) => {
			setComposerVisibleForTab(tabId, false);
		},
		[setComposerVisibleForTab]
	);

	const activeToolActivityCount = useMemo(
		() =>
			activeSessionToolCalls.filter(
				call =>
					call.status === UIToolCallStatus.Running ||
					(call.status === UIToolCallStatus.Pending && call.automatic === true)
			).length,
		[activeSessionToolCalls]
	);

	const showStopRunControl = !activeTabIsHydrating && !activeEditingMessageId && activeRunComposerState.isActive;
	const showAutomationRunControl =
		showStopRunControl && (hasAutomaticToolCalls || hasAutomaticToolConfiguration || automationHoldEnabled);
	const sideRunStatus =
		!composerVisible && activeManualComposerRequirementKey
			? manualToolActionCalls.length > 0
				? 'Tools need attention'
				: hasContinuationAction
					? 'Ready to continue'
					: undefined
			: !composerVisible && activeToolActivityCount > 0
				? `Running ${activeToolActivityCount} tool${activeToolActivityCount === 1 ? '' : 's'}`
				: undefined;

	const tabExists = useCallback((tabId: string) => tabIDSetRef.current.has(tabId), []);

	const {
		inputRefs,
		setInputRef,
		tryApplyDropToTab,
		queuePendingDrop,
		flushPendingDrops,
		syncComposerFromConversation,
		loadExternalMessageForTab,
		resetComposerForNewConversation,
		restoreSessionConfiguration,
		disposeInputRuntime,
		applyWorkflowStarterToComposer,
	} = useInputRegistry({ tabExists });

	const revealComposerAndInvoke = useCallback(
		(tabId: string, action: (input: ComposerBoxHandle) => void) => {
			if (!tabId) {
				return;
			}

			setComposerVisibleForTab(tabId, true);
			const invokeWhenMounted = (remainingFrames: number): void => {
				if (selectedTabIdRef.current !== tabId) {
					return;
				}

				const input = inputRefs.current.get(tabId);
				if (input) {
					action(input);
					return;
				}

				if (remainingFrames > 0) {
					window.requestAnimationFrame(() => {
						invokeWhenMounted(remainingFrames - 1);
					});
				}
			};

			window.requestAnimationFrame(() => {
				invokeWhenMounted(1);
			});
		},
		[inputRefs, selectedTabIdRef, setComposerVisibleForTab]
	);

	const focusComposerForTab = useCallback(
		(tabId: string) => {
			revealComposerAndInvoke(tabId, input => {
				input.focus();
			});
		},
		[revealComposerAndInvoke]
	);

	const tryApplyDropAndRevealComposer = useCallback(
		(tabId: string, payload: Parameters<typeof tryApplyDropToTab>[1]) => {
			const applied = tryApplyDropToTab(tabId, payload);
			if (applied) {
				setComposerVisibleForTab(tabId, true);
			}
			return applied;
		},
		[setComposerVisibleForTab, tryApplyDropToTab]
	);

	useAttachmentsDropTarget({
		selectedTabIdRef,
		tabExists,
		tryApplyDropToTab: tryApplyDropAndRevealComposer,
		queuePendingDrop,
		flushPendingDrops,
	});

	useEffect(() => {
		const resolveMCPAppTargetTab = async (sessionID: string): Promise<string | null> => {
			const existing = tabsRef.current.find(tab => tab.sessionID === sessionID);
			if (existing) {
				return existing.tabId;
			}

			return openSessionForMCPAppMessage(sessionID);
		};

		const handleUIMessage = (event: Event) => {
			const detail = (event as CustomEvent<MCPAppUIMessageEventDetail>).detail;
			const text = detail?.message?.text?.trim();
			const sessionID = detail?.instance?.session?.sessionID;

			if (!text || !sessionID) {
				return;
			}

			void resolveMCPAppTargetTab(sessionID)
				.then(tabId => {
					if (!tabId || !tabExists(tabId)) {
						return;
					}

					loadExternalMessageForTab(tabId, { text });

					if (selectedTabIdRef.current === tabId) {
						focusComposerForTab(tabId);
					}
				})
				.catch((error: unknown) => {
					console.error('Failed to route MCP App draft message:', error);
				});
		};

		const handleModelContextUpdate = (event: Event) => {
			const detail = (event as CustomEvent<MCPAppModelContextUpdateEventDetail>).detail;
			const sessionID = detail?.instance?.session?.sessionID;

			if (!detail?.update || !sessionID) {
				return;
			}

			void resolveMCPAppTargetTab(sessionID)
				.then(tabId => {
					if (!tabId || !tabExists(tabId)) {
						return;
					}

					inputRefs.current.get(tabId)?.appendMCPAppContextUpdate(detail.update);

					if (selectedTabIdRef.current === tabId) {
						focusComposerForTab(tabId);
					}
				})
				.catch((error: unknown) => {
					console.error('Failed to route MCP App context update:', error);
				});
		};

		window.addEventListener(MCP_APP_UI_MESSAGE_EVENT, handleUIMessage);
		window.addEventListener(MCP_APP_MODEL_CONTEXT_UPDATE_EVENT, handleModelContextUpdate);

		return () => {
			window.removeEventListener(MCP_APP_UI_MESSAGE_EVENT, handleUIMessage);
			window.removeEventListener(MCP_APP_MODEL_CONTEXT_UPDATE_EVENT, handleModelContextUpdate);
		};
	}, [
		focusComposerForTab,
		inputRefs,
		loadExternalMessageForTab,
		openSessionForMCPAppMessage,
		selectedTabIdRef,
		tabExists,
	]);

	const {
		setScrollContainerRef,
		setScrollContentRef,
		setScrollFooterRef,
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
		activeTabIsHydrating: activeConversationContentIsHydrating,
		messageCount,
		initialScrollTopByTab,
	});

	const { sendMessageForTab, stageContinuationForTab, beginEditMessageForTab, cancelEditingForTab } = useSendMessage({
		tabsRef,
		updateTab,
		scrollTabToBottomSoon,
		tabExists,
		inputRefs,
	});

	const beginEditMessageAndShowComposer = useCallback(
		(tabId: string, messageID: string) => {
			if (!beginEditMessageForTab(tabId, messageID)) {
				return;
			}

			focusComposerForTab(tabId);
		},
		[beginEditMessageForTab, focusComposerForTab]
	);

	const runCommand = useCallback(
		async (
			action: (sessionID: string, sessionRevision: number, runID: string, runRevision: number) => Promise<void>
		) => {
			const tab = tabsRef.current.find(candidate => candidate.tabId === activeTabId);
			const snapshot = tab?.sessionSnapshot;
			const run = snapshot ? selectedSessionRun(snapshot, tab?.activeRunID) : undefined;

			if (!snapshot || !run || !tab?.sessionID) {
				throw new Error('The active Run snapshot is unavailable.');
			}
			if (tab.editingMessageId) {
				throw new Error('Finish or cancel editing before changing the active Run.');
			}
			if (!getSessionRunComposerState(run).isActive) {
				throw new Error('This Run is no longer active.');
			}

			await action(tab.sessionID, snapshot.session.revision, run.id, run.revision);
		},
		[activeTabId]
	);

	const runCommandForToolCall = useCallback(
		async (
			call: UIToolCall,
			action: (
				sessionID: string,
				sessionRevision: number,
				runID: string,
				runRevision: number,
				callID: string
			) => Promise<void>
		) => {
			const tab = tabsRef.current.find(candidate => candidate.tabId === activeTabId);
			const snapshot = tab?.sessionSnapshot;
			if (!snapshot || !tab?.sessionID) {
				throw new Error('The active Session snapshot is unavailable.');
			}
			if (tab.editingMessageId) {
				throw new Error('Finish or cancel editing before changing Tool calls.');
			}

			const owner = snapshot.runs.find(run =>
				run.toolCalls?.some(
					sessionCall =>
						sessionCall.id === call.id ||
						(Boolean(sessionCall.providerCallID) && sessionCall.providerCallID === call.callID)
				)
			);
			const sessionCall = owner?.toolCalls?.find(
				candidate =>
					candidate.id === call.id || (Boolean(candidate.providerCallID) && candidate.providerCallID === call.callID)
			);

			if (!owner || !sessionCall) {
				throw new Error('The owning Run for this Tool call is unavailable.');
			}
			const selectedRun = selectedSessionRun(snapshot, tab.activeRunID);
			if (owner.id !== selectedRun?.id || !getSessionRunComposerState(owner).isActive) {
				throw new Error('This Tool call no longer belongs to the active Run.');
			}

			await action(tab.sessionID, snapshot.session.revision, owner.id, owner.revision, sessionCall.id);
		},
		[activeTabId]
	);

	const executeSessionToolCall = useCallback(
		(call: UIToolCall, continueRun = false) =>
			runCommandForToolCall(call, async (sessionID, sessionRevision, runID, runRevision, callID) => {
				requireAcceptedSessionCommand(
					await sessionManagementAPI.executeCalls({
						meta: sessionCommandMeta(sessionRevision, runRevision),
						sessionID,
						runID,
						callIDs: [callID],
						continue: continueRun,
					}),
					'Execute Tool call'
				);
			}),
		[runCommandForToolCall]
	);

	const retrySessionToolCall = useCallback(
		(call: UIToolCall) =>
			runCommandForToolCall(call, async (sessionID, sessionRevision, runID, runRevision, callID) => {
				requireAcceptedSessionCommand(
					await sessionManagementAPI.retryCall({
						meta: sessionCommandMeta(sessionRevision, runRevision),
						sessionID,
						runID,
						callID,
					}),
					'Retry Tool call'
				);
			}),
		[runCommandForToolCall]
	);

	const skipSessionToolCall = useCallback(
		(call: UIToolCall) =>
			runCommandForToolCall(call, async (sessionID, sessionRevision, runID, runRevision, callID) => {
				if (call.status === UIToolCallStatus.Failed) {
					requireAcceptedSessionCommand(
						await sessionManagementAPI.completeCallAsError({
							meta: sessionCommandMeta(sessionRevision, runRevision),
							sessionID,
							runID,
							callID,
							error: call.errorMessage?.trim() || 'The Tool call failed before producing a usable result.',
						}),
						'Submit Tool error'
					);
					return;
				}

				requireAcceptedSessionCommand(
					await sessionManagementAPI.skipCall({
						meta: sessionCommandMeta(sessionRevision, runRevision),
						sessionID,
						runID,
						callID,
					}),
					'Skip Tool call'
				);
			}),
		[runCommandForToolCall]
	);

	const stagedAppContextUpdateCount =
		!activeEditingMessageId && activeRunComposerState.hasStagedContinuation
			? (activeRun?.continuation.mcpAppContext?.length ?? 0)
			: 0;

	const executePendingSessionToolCalls = useCallback(
		(continueRun: boolean) =>
			runCommand(async (sessionID, sessionRevision, runID, runRevision) => {
				if (pendingExecutableToolCallIDs.length === 0) {
					throw new Error('There are no pending Tool calls ready to run.');
				}

				requireAcceptedSessionCommand(
					await sessionManagementAPI.executeCalls({
						meta: sessionCommandMeta(sessionRevision, runRevision),
						sessionID,
						runID,
						callIDs: pendingExecutableToolCallIDs,
						continue: continueRun,
					}),
					continueRun ? 'Run Tools and continue' : 'Run pending Tools'
				);
			}),
		[pendingExecutableToolCallIDs, runCommand]
	);

	const clearStagedAppContext = useCallback(() => {
		const continuation = activeRun?.continuation;
		if (!continuation) {
			return Promise.reject(new Error('The staged App context is unavailable.'));
		}

		return runCommand(async (sessionID, sessionRevision, runID, runRevision) => {
			requireAcceptedSessionCommand(
				await sessionManagementAPI.updateContinuationInput({
					meta: sessionCommandMeta(sessionRevision, runRevision),
					sessionID,
					runID,
					expectedInputRevision: continuation.revision,
					input: {
						...continuation,
						mcpAppContext: [],
					},
				}),
				'Discard staged App context'
			);
		});
	}, [activeRun?.continuation, runCommand]);

	const toggleAutomationHold = useCallback(
		() =>
			runCommand(async (sessionID, sessionRevision, runID, runRevision) => {
				const enabled = !automationHoldEnabled;

				requireAcceptedSessionCommand(
					await sessionManagementAPI.setAutomationHold({
						meta: sessionCommandMeta(sessionRevision, runRevision),
						sessionID,
						runID,
						tools: true,
						continuation: true,
						enabled,
						...(enabled
							? {
									scope: SessionAutomationHoldScope.OneBatch,
								}
							: {}),
					}),
					enabled ? 'Hold automatic Tools' : 'Resume automatic Tools'
				);
			}),
		[automationHoldEnabled, runCommand]
	);

	const runSessionAction = useCallback(
		(action: () => Promise<void>) => {
			const tabId = activeTabId;
			// A double click, or two inline controls clicked before the next
			// snapshot, must not issue competing revision-sensitive commands.
			if (sessionActionsInFlightRef.current.has(tabId)) {
				return;
			}
			sessionActionsInFlightRef.current.add(tabId);
			setSessionActionErrorState(null);
			void Promise.resolve()
				.then(action)
				.catch((error: unknown) => {
					setSessionActionErrorState({
						tabId,
						message: sessionActionErrorMessage(error),
					});
				})
				.finally(() => {
					sessionActionsInFlightRef.current.delete(tabId);
				});
		},
		[activeTabId, setSessionActionErrorState]
	);

	const cancelRunForTab = useCallback(async (tabId: string, expectedRunID?: string) => {
		const tab = tabsRef.current.find(candidate => candidate.tabId === tabId);
		const snapshot = tab?.sessionSnapshot;
		const run = snapshot ? selectedSessionRun(snapshot, tab?.activeRunID) : undefined;

		if (
			!tab?.sessionID ||
			!snapshot ||
			!run ||
			!getSessionRunComposerState(run).isActive ||
			(expectedRunID !== undefined && run.id !== expectedRunID)
		) {
			return;
		}

		// Cancellation is intentionally best-effort. Streaming changes Session
		// and Run revisions continuously, so a revision captured by the UI is
		// expected to be stale by the time the user presses Stop.
		const receipt = await sessionManagementAPI.cancelRun({
			meta: sessionCommandMeta(),
			sessionID: tab.sessionID,
			runID: run.id,
		});

		// This is harmless when a completion won the race with Stop. The next
		// authoritative snapshot will show the terminal response.
		if (!receipt.accepted && receipt.message?.includes('already terminal')) {
			return;
		}

		requireAcceptedSessionCommand(receipt, 'Cancel Run');
	}, []);

	const requestStopResponseForTab = useCallback((tabId: string) => {
		const tab = tabsRef.current.find(candidate => candidate.tabId === tabId);
		const snapshot = tab?.sessionSnapshot;
		const run = snapshot ? selectedSessionRun(snapshot, tab?.activeRunID) : undefined;

		if (!run || !getSessionRunComposerState(run).isActive) {
			return;
		}

		setRunCancellationConfirmation({
			tabId,
			runID: run.id,
		});
	}, []);

	const confirmCancelRun = useCallback(() => {
		const confirmation = runCancellationConfirmation;
		setRunCancellationConfirmation(null);

		if (!confirmation) {
			return;
		}

		void cancelRunForTab(confirmation.tabId, confirmation.runID).catch((error: unknown) => {
			setSessionActionErrorState({
				tabId: confirmation.tabId,
				message: sessionActionErrorMessage(error),
			});
		});
	}, [cancelRunForTab, runCancellationConfirmation]);

	const resolveWait = useCallback(
		async (wait: SessionWaitingRequest, resolution: SessionWaitResolution, input?: SessionOpaqueJSON) => {
			setWaitResolutionState({
				waitID: wait.id,
				isResolving: true,
			});

			try {
				await runCommand(async (sessionID, sessionRevision, runID, runRevision) => {
					requireAcceptedSessionCommand(
						await sessionManagementAPI.resolveWait({
							meta: sessionCommandMeta(sessionRevision, runRevision),
							sessionID,
							runID,
							waitID: wait.id,
							resolution,
							...(input === undefined ? {} : { input }),
						}),
						'Resolve Run wait'
					);
				});
				setWaitResolutionState({
					waitID: wait.id,
					isResolving: false,
				});
			} catch (error) {
				setWaitResolutionState({
					waitID: wait.id,
					isResolving: false,
					error: error instanceof Error ? error.message : 'The wait could not be resolved.',
				});
			}
		},
		[runCommand, setWaitResolutionState]
	);

	const disposeTabRuntime = useCallback(
		(tabId: string) => {
			disposeInputRuntime(tabId);
			disposeScrollRuntime(tabId);
		},
		[disposeInputRuntime, disposeScrollRuntime]
	);

	const openTemplateMenuForTab = useCallback(
		(tabId: string) => {
			revealComposerAndInvoke(tabId, input => {
				input.openTemplateMenu();
			});
		},
		[revealComposerAndInvoke]
	);
	const openToolMenuForTab = useCallback(
		(tabId: string) => {
			revealComposerAndInvoke(tabId, input => {
				input.openToolMenu();
			});
		},
		[revealComposerAndInvoke]
	);
	const openAttachmentMenuForTab = useCallback(
		(tabId: string) => {
			revealComposerAndInvoke(tabId, input => {
				input.openAttachmentMenu();
			});
		},
		[revealComposerAndInvoke]
	);
	const openSystemPromptMenuForTab = useCallback(
		(tabId: string) => {
			revealComposerAndInvoke(tabId, input => {
				input.openSystemPromptMenu();
			});
		},
		[revealComposerAndInvoke]
	);
	const openSkillsMenuForTab = useCallback(
		(tabId: string) => {
			revealComposerAndInvoke(tabId, input => {
				input.openSkillsMenu();
			});
		},
		[revealComposerAndInvoke]
	);
	const openMCPMenuForTab = useCallback(
		(tabId: string) => {
			revealComposerAndInvoke(tabId, input => {
				input.openMCPMenu();
			});
		},
		[revealComposerAndInvoke]
	);
	const openWorkspaceMenuForTab = useCallback(
		(tabId: string) => {
			revealComposerAndInvoke(tabId, input => {
				input.openWorkspaceMenu();
			});
		},
		[revealComposerAndInvoke]
	);

	useImperativeHandle(
		ref,
		() => ({
			disposeTabRuntime,
			syncComposerFromConversation,
			resetComposerForNewConversation,
			hideComposer: tabId => {
				setComposerVisibleForTab(tabId, false);
			},
			restoreComposerFromSessionConfiguration: restoreSessionConfiguration,
			applyWorkflowStarter: applyWorkflowStarterToComposer,
			focusInput: focusComposerForTab,
			openTemplateMenu: openTemplateMenuForTab,
			openToolMenu: openToolMenuForTab,
			openAttachmentMenu: openAttachmentMenuForTab,
			openSystemPromptMenu: openSystemPromptMenuForTab,
			openSkillsMenu: openSkillsMenuForTab,
			openMCPMenu: openMCPMenuForTab,
			openWorkspaceMenu: openWorkspaceMenuForTab,
			requestStopResponse: requestStopResponseForTab,
			setScrollTopForTab,
			resetScrollToTop,
			getScrollTopByTabSnapshot,
		}),
		[
			applyWorkflowStarterToComposer,
			disposeTabRuntime,
			focusComposerForTab,
			getScrollTopByTabSnapshot,
			openAttachmentMenuForTab,
			openSystemPromptMenuForTab,
			openSkillsMenuForTab,
			openMCPMenuForTab,
			openWorkspaceMenuForTab,
			openTemplateMenuForTab,
			openToolMenuForTab,
			requestStopResponseForTab,
			restoreSessionConfiguration,
			resetComposerForNewConversation,
			resetScrollToTop,
			setComposerVisibleForTab,
			setScrollTopForTab,
			syncComposerFromConversation,
		]
	);

	const itemContent = useCallback(
		(
			index: number,
			message: ConversationMessage,
			diffCandidatePaths: string[] | undefined,
			diffCandidatePathsReady: boolean,
			deferRichRendering: boolean
		) => {
			const isLast = index === messageCount - 1;
			const isAssistant = message.role === RoleEnum.Assistant;
			const rowIsBusy = isLast && activeTabIsGenerating && isAssistant;
			const responseTiming = isAssistant ? responseTimingByMessageID.get(message.id) : undefined;

			if (rowIsBusy) {
				return (
					<ConversationMessageRow
						message={message}
						isBusy={true}
						isEditing={activeEditingMessageId === message.id}
						deferRichRendering={deferRichRendering}
						tabId={activeTabId}
						onEditMessage={beginEditMessageAndShowComposer}
						diffCandidatePaths={diffCandidatePaths}
						diffCandidatePathsReady={diffCandidatePathsReady}
						responseTiming={responseTiming}
					/>
				);
			}

			return (
				<ConversationMessageRow
					message={message}
					isBusy={false}
					isEditing={activeEditingMessageId === message.id}
					deferRichRendering={deferRichRendering}
					tabId={activeTabId}
					onEditMessage={beginEditMessageAndShowComposer}
					diffCandidatePaths={diffCandidatePaths}
					diffCandidatePathsReady={diffCandidatePathsReady}
					responseTiming={responseTiming}
				/>
			);
		},
		[
			activeEditingMessageId,
			activeTabId,
			activeTabIsGenerating,
			beginEditMessageAndShowComposer,
			messageCount,
			responseTimingByMessageID,
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
					<div ref={setScrollContentRef} className="mx-auto w-11/12 pb-4 xl:w-5/6">
						{activeTab?.runtimeError ? (
							<div className="alert alert-error mb-2 rounded-2xl text-sm" role="alert">
								<span>{activeTab.runtimeError}</span>
							</div>
						) : null}
						{sessionActionError ? (
							<div className="alert alert-error mb-2 rounded-2xl text-sm" role="alert">
								<span>{sessionActionError}</span>
							</div>
						) : null}
						{pendingRunAcknowledgement ? (
							<output className="border-info/30 bg-info/10 mb-2 flex items-center gap-2 rounded-2xl border px-3 py-2 text-sm">
								<span className="loading loading-dots loading-sm" aria-hidden="true" />
								<span>Run accepted. Preparing the authoritative operation…</span>
							</output>
						) : null}
						{activeTabIsHydrating && messageCount === 0 ? (
							<div className="flex min-h-128 items-center justify-center py-8">
								<span className="loading loading-dots loading-md" aria-label="Loading conversation" />
							</div>
						) : (
							<ConversationMessageList
								key={`${activeTabId}:${activeTab?.conversation.id ?? ''}`}
								messages={messages}
								renderItem={itemContent}
								isBusy={activeTabIsGenerating}
								deferRichWork={activeTabIsHydrating}
							/>
						)}
						{composerVisible && composerPanelHeight > 0 ? (
							<div aria-hidden="true" style={{ height: composerPanelHeight + FLOATING_COMPOSER_SCROLL_CLEARANCE }} />
						) : null}
						<div ref={setScrollFooterRef} className="h-px w-full" data-chat-scroll-footer aria-hidden="true" />
					</div>
				</div>

				<div className="pointer-events-none absolute top-8 right-8 z-30 xl:right-24">
					<nav className="pointer-events-auto flex flex-col items-center gap-1" aria-label="Conversation navigation">
						<HoverTip content="Scroll to top" placement="left">
							<ButtonScrollToTop
								onScrollToTop={scrollActiveToTop}
								iconSize={24}
								show
								disabled={isAtTop}
								className="btn btn-sm btn-circle btn-ghost border-none bg-transparent shadow-none disabled:opacity-40"
							/>
						</HoverTip>
						<HoverTip content="Previous message" placement="left">
							<button
								type="button"
								disabled={!canJumpToPreviousMessage}
								className="btn btn-sm btn-circle btn-ghost border-none bg-transparent shadow-none disabled:opacity-40"
								onClick={scrollActiveToPreviousMessage}
								aria-label="Jump to previous message"
							>
								<FiChevronUp size={24} />
							</button>
						</HoverTip>
						<HoverTip content="Next message" placement="left">
							<button
								type="button"
								disabled={!canJumpToNextMessage}
								className="btn btn-sm btn-circle btn-ghost border-none bg-transparent shadow-none disabled:opacity-40"
								onClick={scrollActiveToNextMessage}
								aria-label="Jump to next message"
							>
								<FiChevronDown size={24} />
							</button>
						</HoverTip>
						<HoverTip content="Scroll to bottom" placement="left">
							<ButtonScrollToBottom
								onScrollToBottom={scrollActiveToBottom}
								iconSize={24}
								show
								disabled={isAtBottom}
								className="btn btn-sm btn-circle btn-ghost border-none bg-transparent shadow-none disabled:opacity-40"
							/>
						</HoverTip>
					</nav>
				</div>

				<div
					className="pointer-events-none absolute right-8 bottom-16 z-30 xl:right-24"
					style={
						composerVisible ? { bottom: `${composerPanelHeight + FLOATING_COMPOSER_SCROLL_CLEARANCE}px` } : undefined
					}
				>
					<section className="pointer-events-auto flex flex-col items-center gap-2" aria-label="Conversation actions">
						{sideRunStatus ? (
							<output className="text-base-content/70 max-w-32 text-center text-xs">{sideRunStatus}</output>
						) : null}

						{showStopRunControl ? (
							<HoverTip content={activeTabIsGenerating ? 'Stop response' : 'Cancel active run'} placement="left">
								<button
									type="button"
									className="btn btn-md btn-circle btn-neutral p-2"
									onClick={() => {
										requestStopResponseForTab(activeTabId);
									}}
									aria-label={activeTabIsGenerating ? 'Stop response' : 'Cancel active run'}
								>
									<FiSquare size={24} />
								</button>
							</HoverTip>
						) : null}

						{showAutomationRunControl ? (
							<HoverTip
								content={
									automationHoldEnabled
										? 'Resume automatic tools and continuation'
										: 'Hold the next automatic tool and continuation batch'
								}
								placement="left"
							>
								<button
									type="button"
									className={`btn btn-circle btn-neutral btn-md p-2 ${
										automationHoldEnabled ? 'text-warning hover:bg-warning/10' : ''
									}`}
									onClick={() => {
										runSessionAction(toggleAutomationHold);
									}}
									aria-label={automationHoldEnabled ? 'Resume automation' : 'Hold automation'}
									aria-pressed={automationHoldEnabled}
								>
									{automationHoldEnabled ? <FiPlay size={24} /> : <FiPause size={24} />}
								</button>
							</HoverTip>
						) : null}

						<HoverTip content={composerVisible ? 'Disable composer' : 'Enable composer'} placement="left">
							<button
								ref={composerToggleButtonRef}
								type="button"
								className="btn btn-md btn-circle btn-neutral p-2"
								onClick={() => {
									if (!activeTabId) {
										return;
									}

									if (composerVisible) {
										setComposerVisibleForTab(activeTabId, false);
										window.requestAnimationFrame(() => {
											composerToggleButtonRef.current?.focus({ preventScroll: true });
										});
										return;
									}

									focusComposerForTab(activeTabId);
								}}
								aria-label={composerVisible ? 'Disable composer' : 'Enable composer'}
								aria-pressed={composerVisible}
							>
								{composerVisible ? <FiMinimize2 size={24} /> : <FiEdit size={24} />}
							</button>
						</HoverTip>
					</section>
				</div>

				<div className="pointer-events-none absolute inset-x-0 inset-y-3 z-20 flex items-end justify-center">
					<div
						ref={setComposerPanelElement}
						role={composerVisible ? 'region' : undefined}
						aria-label={composerVisible ? 'Message composer' : undefined}
						aria-hidden={composerVisible ? undefined : true}
						className={[
							'pointer-events-auto max-h-full w-11/12 min-w-0 xl:w-5/6',
							composerVisible
								? 'app-scrollbar-thin border-base-content/20 bg-base-100/95 ring-base-100/30 overflow-y-auto rounded-2xl border shadow-none ring-1 backdrop-blur'
								: 'overflow-hidden',
						].join(' ')}
					>
						{mountedInputTabs.map(tab =>
							(() => {
								const tabRun = tab.sessionSnapshot
									? selectedSessionRun(tab.sessionSnapshot, tab.activeRunID)
									: undefined;
								const suppressRuntimeActivity =
									Boolean(tab.editingMessageId) || tab.isHydrating || !(tab.isLoaded ?? true);
								const activity = suppressRuntimeActivity
									? undefined
									: tab.tabId === activeTabId
										? activeSessionToolActivity
										: tab.sessionSnapshot
											? projectSessionRunToolActivity(tab.sessionSnapshot, tabRun)
											: undefined;
								const composerState = getSessionRunComposerState(tabRun, (activity?.outputs.length ?? 0) > 0);
								const allowContinuationInput = !suppressRuntimeActivity && composerState.canStageContinuationInput;
								const continuationCanContinue = !suppressRuntimeActivity && composerState.canComposeContinuationInput;
								const showRuntimeControls =
									tab.tabId === activeTabId && !suppressRuntimeActivity && composerState.isActive;

								return (
									<TabInputPane
										key={tab.tabId}
										tabId={tab.tabId}
										active={tab.tabId === selectedTabId}
										visible={tab.tabId === selectedTabId && composerVisible}
										isBusy={tab.isBusy}
										isGenerating={runHasGeneratingModelOperation(tabRun)}
										isHydrating={tab.isHydrating || !(tab.isLoaded ?? true)}
										allowContinuationInput={allowContinuationInput}
										continuationCanContinue={continuationCanContinue}
										editingMessageId={tab.editingMessageId}
										runtimeStateUpdatedAt={tabRun?.updatedAt}
										setInputRef={setInputRef}
										shortcutConfig={shortcutConfig}
										sendMessage={sendMessageForTab}
										sendContinuation={stageContinuationForTab}
										cancelEditing={cancelEditingForTab}
										onSubmissionAccepted={hideComposerAfterSubmission}
										showPendingToolControls={showRuntimeControls && canRunPendingToolCalls}
										canContinueWithoutDraft={continuationCanContinue && composerState.canContinueWithoutDraft}
										onRunPendingTools={continueRun => {
											runSessionAction(() => executePendingSessionToolCalls(continueRun));
										}}
										sessionToolCalls={activity?.calls ?? []}
										sessionToolOutputs={activity?.outputs ?? []}
										onRunSessionToolCall={call => {
											if (tab.tabId !== activeTabId) {
												return;
											}
											runSessionAction(() => executeSessionToolCall(call));
										}}
										onRetrySessionToolCall={call => {
											if (tab.tabId !== activeTabId) {
												return;
											}
											runSessionAction(() => retrySessionToolCall(call));
										}}
										onSkipSessionToolCall={call => {
											if (tab.tabId !== activeTabId) {
												return;
											}
											runSessionAction(() => skipSessionToolCall(call));
										}}
										stagedAppContextUpdateCount={tab.tabId === activeTabId ? stagedAppContextUpdateCount : 0}
										onDiscardStagedAppContext={() => {
											runSessionAction(clearStagedAppContext);
										}}
									/>
								);
							})()
						)}
					</div>
				</div>
			</div>

			<DeleteConfirmationModal
				isOpen={runCancellationConfirmationOpen}
				onClose={() => {
					setRunCancellationConfirmation(null);
				}}
				onConfirm={confirmCancelRun}
				title="Cancel active run?"
				message="Partial output already received remains in the conversation. Stop the request?"
				confirmButtonText="Cancel run"
			/>

			{(() => {
				const currentWaitState =
					pendingWait && waitResolutionState?.waitID === pendingWait.id ? waitResolutionState : undefined;

				return (
					<SessionWaitModal
						wait={pendingWait}
						isResolving={currentWaitState?.isResolving ?? false}
						error={currentWaitState?.error}
						onResolve={(resolution, input) => {
							const wait = pendingWait;
							if (!wait) {
								return Promise.resolve();
							}
							return resolveWait(wait, resolution, input);
						}}
					/>
				);
			})()}
		</>
	);
}

export const ConversationArea = memo(
	forwardRef<ConversationAreaHandle, ConversationAreaProps>(ConversationAreaInner),
	areConversationAreaPropsEqual
);
