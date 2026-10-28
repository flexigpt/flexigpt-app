import { memo, useCallback, useMemo, useState } from 'react';
import { FiAlertTriangle, FiUser, FiZap } from 'react-icons/fi';

import type { UIToolCall, UIToolOutput } from '@/spec/inference';
import type { ConversationMessage } from '@/spec/session';
import type { ToolSelectionIssue, ToolStoreChoice } from '@/spec/tool';
import { RoleEnum, Status } from '@/spec/inference';

import type { ToolDetailsState } from '@/chats/composer/tools/tool_details_modal';
import type { MessageStreamSource } from '@/chats/messages/message_content_card';
import type { AssistantResponseTiming } from '@/chats/messages/message_footer';
import { ToolDetailsModal } from '@/chats/composer/tools/tool_details_modal';
import { buildAppInstanceFromToolOutput } from '@/chats/mcpapps/mcp_app_types';
import { MCPAppView } from '@/chats/mcpapps/mcp_app_view';
import {
	getMCPAppToolResultContent,
	getMCPAppToolResultStructuredContent,
} from '@/chats/messages/mcp_message_context_utils';
import { MessageAttachmentsBar } from '@/chats/messages/message_attachments_bar';
import { MessageCitationsBar } from '@/chats/messages/message_citations_bar';
import { MessageContentCard, useMessageStreamSnapshot } from '@/chats/messages/message_content_card';
import { MessageFooterArea } from '@/chats/messages/message_footer';
import { mediaItemsFromOutputUnions, MessageMediaContent } from '@/chats/messages/message_media_content';
import { MessageThinkingSection } from '@/chats/messages/message_thinking_section';

interface ChatMessageProps {
	message: ConversationMessage;
	isBusy: boolean;
	isEditing: boolean;
	deferRichRendering: boolean;
	diffCandidatePaths?: string[];
	diffCandidatePathsReady?: boolean;
	onEdit: () => void;
	streamSource?: MessageStreamSource;
	responseTiming?: AssistantResponseTiming;
	/**
	 * The following message in this turn, when available. For a user message,
	 * its assistant response can supply recorded selections. For an assistant
	 * message, the following user message can hold its tool results.
	 */
	turnContextMessage?: ConversationMessage;
}

function stringArraysEqual(left?: string[], right?: string[]): boolean {
	if (left === right) {
		return true;
	}
	if (!left || !right || left.length !== right.length) {
		return false;
	}

	return left.every((value, index) => value === right[index]);
}

function propsAreEqual(prev: ChatMessageProps, next: ChatMessageProps) {
	if (prev.onEdit !== next.onEdit) {
		return false;
	}
	if (prev.isBusy !== next.isBusy) {
		return false;
	}
	if (prev.isEditing !== next.isEditing) {
		return false;
	}
	if (prev.deferRichRendering !== next.deferRichRendering) {
		return false;
	}
	if (prev.streamSource !== next.streamSource) {
		return false;
	}
	if (prev.turnContextMessage !== next.turnContextMessage) {
		return false;
	}
	if (
		prev.responseTiming?.startedAt !== next.responseTiming?.startedAt ||
		prev.responseTiming?.finishedAt !== next.responseTiming?.finishedAt
	) {
		return false;
	}

	if (prev.message.debugDetails !== next.message.debugDetails) {
		return false;
	}
	if (prev.message.debugDetailsError !== next.message.debugDetailsError) {
		return false;
	}
	if (prev.message.error !== next.message.error) {
		return false;
	}
	if (prev.message.usage !== next.message.usage) {
		return false;
	}

	if (prev.message.uiReasoningContents !== next.message.uiReasoningContents) {
		return false;
	}

	// Markdown and Mermaid are driven by uiContent.
	if (prev.message.uiContent !== next.message.uiContent) {
		return false;
	}

	// Optional but recommended: these can affect effective rendered markdown too.
	if (prev.message.status !== next.message.status) {
		return false;
	}
	if (prev.message.outputs !== next.message.outputs) {
		return false;
	}
	if (prev.message.uiCitations !== next.message.uiCitations) {
		return false;
	}
	if (prev.message.uiToolCalls !== next.message.uiToolCalls) {
		return false;
	}
	if (prev.message.uiToolOutputs !== next.message.uiToolOutputs) {
		return false;
	}
	if (prev.message.workspaceSelection !== next.message.workspaceSelection) {
		return false;
	}
	if (prev.message.workspaceUsage !== next.message.workspaceUsage) {
		return false;
	}
	if (prev.message.operationResolution !== next.message.operationResolution) {
		return false;
	}
	if (prev.message.appContextUpdates !== next.message.appContextUpdates) {
		return false;
	}

	// Equivalent freshly-created arrays must not rebuild the markdown and
	// Mermaid subtree.
	if (prev.diffCandidatePathsReady !== next.diffCandidatePathsReady) {
		return false;
	}
	if (!stringArraysEqual(prev.diffCandidatePaths, next.diffCandidatePaths)) {
		return false;
	}

	// If the *object reference* for the ConversationMessage changes
	// react must re-render (content edited, message appended).
	if (prev.message !== next.message) {
		return false;
	}

	// Everything else is the same: skip.
	return true;
}

function terminalStateMessage(message: ConversationMessage): string | null {
	switch (message.status) {
		case Status.Cancelled:
			return message.error?.message?.trim()
				? `Generation was cancelled: ${message.error.message.trim()}`
				: 'Generation was cancelled before completion.';

		case Status.Incomplete:
			return message.error?.message?.trim()
				? `Generation ended before completion: ${message.error.message.trim()}`
				: 'Generation ended before completion.';

		case Status.Failed:
			return message.error?.message?.trim()
				? `Generation failed: ${message.error.message.trim()}`
				: 'Generation failed before completion.';

		default:
			return null;
	}
}

export const ChatMessage = memo(function ChatMessage({
	message,
	isBusy,
	isEditing,
	deferRichRendering,
	diffCandidatePaths,
	diffCandidatePathsReady,
	onEdit,
	streamSource,
	responseTiming,
	turnContextMessage,
}: ChatMessageProps) {
	const isUser = message.role === RoleEnum.User;
	const streamSnapshot = useMessageStreamSnapshot(streamSource, isBusy);
	const align = !isUser ? 'items-end text-left' : 'items-start text-left';
	const leftColSpan = !isUser ? 'col-span-1 lg:col-span-2' : 'col-span-1';
	const rightColSpan = !isUser ? 'col-span-1' : 'col-span-1 lg:col-span-2';

	const [renderMarkdown, setRenderMarkdown] = useState(!isUser);
	const [toolDetailsState, setToolDetailsState] = useState<ToolDetailsState>(null);

	const handleDisableMarkdownChange = useCallback((checked: boolean) => {
		setRenderMarkdown(!checked);
	}, []);
	const handleToolChoiceDetails = useCallback((choice: ToolStoreChoice) => {
		setToolDetailsState({ kind: 'choice', choice });
	}, []);
	const handleToolSelectionIssueDetails = useCallback((issue: ToolSelectionIssue) => {
		setToolDetailsState({ kind: 'selection', selection: issue.selection, message: issue.message });
	}, []);
	const handleToolCallDetails = useCallback((call: UIToolCall) => {
		setToolDetailsState({ kind: 'call', call });
	}, []);
	const handleToolOutputDetails = useCallback((output: UIToolOutput) => {
		setToolDetailsState({ kind: 'output', output });
	}, []);

	const bubbleExtra = [isBusy ? '' : 'shadow-lg', isEditing ? 'ring-2 ring-primary/70' : ''].filter(Boolean).join(' ');

	const baseContent = message.uiContent ?? '';
	const hasAnyContent = /\S/.test(baseContent);
	const hasTerminalFailure =
		message.status === Status.Failed || message.status === Status.Cancelled || message.status === Status.Incomplete;
	const hasError = hasTerminalFailure || !!message.error;
	const hasStreamedText = streamSnapshot.text.length > 0;
	const hasStreamedThinking = streamSnapshot.thinking.length > 0;
	const terminalState = !isBusy ? terminalStateMessage(message) : null;

	const outputMedia = useMemo(
		() => (isUser ? [] : mediaItemsFromOutputUnions(message.outputs)),
		[isUser, message.outputs]
	);

	const hasAnyReasoning =
		message.uiReasoningContents?.some(rc => {
			const sum = (rc?.summary ?? []).some(s => /\S/.test(s ?? ''));
			const th = (rc?.thinking ?? []).some(s => /\S/.test(s ?? ''));
			return sum || th;
		}) ?? false;

	const hasCitations = !isUser && !isBusy && (message.uiCitations?.length ?? 0) > 0;

	// Some recorded send-time selections are stored on the response. Display
	// them with the user's request, never with the assistant's activity.
	const selectionSource = isUser && turnContextMessage?.role === RoleEnum.Assistant ? turnContextMessage : undefined;
	const toolChoices = isUser
		? message.uiToolChoices?.length
			? message.uiToolChoices
			: selectionSource?.uiToolChoices
		: undefined;
	const toolSelectionIssues = isUser
		? message.uiToolSelectionIssues?.length
			? message.uiToolSelectionIssues
			: selectionSource?.uiToolSelectionIssues
		: undefined;
	const operationResolution = isUser
		? (message.operationResolution ?? selectionSource?.operationResolution)
		: undefined;
	const appContextUpdates = isUser
		? message.appContextUpdates?.length
			? message.appContextUpdates
			: selectionSource?.appContextUpdates
		: undefined;

	// Tool results can be persisted on the following user message. Only
	// results that match this assistant's calls belong under this response.
	const toolOutputs = useMemo(() => {
		if (isUser) {
			return undefined;
		}
		if (message.uiToolOutputs?.length) {
			return message.uiToolOutputs;
		}
		const followingOutputs = turnContextMessage?.role === RoleEnum.User ? turnContextMessage.uiToolOutputs : undefined;
		if (!followingOutputs?.length) {
			return message.uiToolOutputs;
		}
		const callIDs = new Set<string>();
		for (const call of message.uiToolCalls ?? []) {
			if (call.id) {
				callIDs.add(call.id);
			}
			if (call.callID) {
				callIDs.add(call.callID);
			}
		}
		return followingOutputs.filter(output => callIDs.has(output.callID));
	}, [isUser, message.uiToolCalls, message.uiToolOutputs, turnContextMessage]);

	const hasMCPContext = (operationResolution?.mcp?.length ?? 0) > 0;
	const hasWorkspaceContext = Boolean(operationResolution?.workspace);
	const hasSkillContext = (operationResolution?.skills?.length ?? 0) > 0;
	const hasAppContext = (appContextUpdates?.length ?? 0) > 0;

	const hasAttachmentsBar =
		(message.attachments?.length ?? 0) > 0 ||
		(toolChoices?.length ?? 0) > 0 ||
		(toolSelectionIssues?.length ?? 0) > 0 ||
		hasAppContext ||
		(!isUser && (message.uiToolCalls?.length ?? 0) > 0) ||
		(toolOutputs?.length ?? 0) > 0 ||
		hasMCPContext ||
		hasWorkspaceContext ||
		hasSkillContext;

	// Detect MCP Apps in this message's tool outputs.
	const mcpAppViews = useMemo(() => {
		if (isBusy || deferRichRendering) {
			return [];
		}

		const outputs = toolOutputs ?? [];
		const callsById = new Map<string, UIToolCall>();
		for (const call of message.uiToolCalls ?? []) {
			if (call.id) {
				callsById.set(call.id, call);
			}
			if (call.callID) {
				callsById.set(call.callID, call);
			}
		}
		return outputs
			.map(out => {
				const instance = buildAppInstanceFromToolOutput(out);
				if (!instance) {
					return null;
				}
				const call = callsById.get(out.callID);
				return { instance, call, output: out } as const;
			})
			.filter((v): v is NonNullable<typeof v> => v !== null);
	}, [deferRichRendering, isBusy, message.uiToolCalls, toolOutputs]);

	const hasMCPAppsView = mcpAppViews.length > 0;

	// A busy placeholder is intentionally not enough to render a bubble. Until
	// the provider sends meaningful content, only the compact footer status is
	// shown. This eliminates the empty assistant bubble.
	const showBody =
		hasAnyContent ||
		hasError ||
		hasMCPAppsView ||
		hasAnyReasoning ||
		(isBusy && (hasStreamedText || hasStreamedThinking));
	const showBodyWithMedia = showBody || outputMedia.length > 0;
	return (
		<div className="grid grid-cols-12 p-1" style={{ fontSize: 14 }}>
			{/* Row 1 ── icon + message bubble (only when showCardRow) */}
			{showBodyWithMedia && (
				<>
					<div className={`${leftColSpan} flex justify-end`}>
						{isUser && (
							<div className="my-0 mr-2 ml-0 flex size-8 items-center justify-center self-end">
								<FiUser size={24} />
							</div>
						)}
					</div>

					<div
						className={`bg-base-100 col-span-10 mt-1 min-w-0 overflow-x-hidden rounded-2xl p-0 lg:col-span-9 ${bubbleExtra}`}
					>
						{!isUser && (hasAnyReasoning || (isBusy && streamSource)) && (
							<MessageThinkingSection
								isBusy={isBusy}
								reasoningContents={message.uiReasoningContents}
								streamedThinking={streamSnapshot.thinking}
								openWhenComplete={hasError}
							/>
						)}
						<div className="px-4 py-2">
							<MessageContentCard
								messageID={message.id}
								content={baseContent}
								isBusy={isBusy}
								align={align}
								renderAsMarkdown={renderMarkdown}
								deferRichRendering={deferRichRendering}
								diffCandidatePaths={diffCandidatePaths}
								streamingText={isBusy ? streamSnapshot.text : undefined}
								diffCandidatePathsReady={diffCandidatePathsReady}
							/>
							<MessageMediaContent items={outputMedia} />
							{!hasAnyContent && hasError && !terminalState && !isBusy ? (
								<div className="text-error text-sm opacity-80">
									{message.error?.message || 'An error occurred while processing this request.'}
								</div>
							) : null}
							{!isUser && terminalState ? (
								<div
									className={`mt-2 flex items-start gap-2 rounded-xl px-2 py-1 text-xs ${
										message.status === Status.Incomplete
											? 'bg-warning/10 text-warning-content'
											: 'bg-error/10 text-error'
									}`}
									role={message.status === Status.Incomplete ? 'status' : 'alert'}
								>
									<FiAlertTriangle className="mt-0.5 shrink-0" size={13} aria-hidden="true" />
									<span className="italic">{terminalState}</span>
								</div>
							) : null}
						</div>
						{/* MCP Apps row. One sandboxed iframe per app-capable MCP tool output. */}
						{mcpAppViews.length > 0 && (
							<div className="border-base-300 border-t px-4 py-3">
								<div className="space-y-2">
									{mcpAppViews.map(({ instance, call, output }) => {
										const content = getMCPAppToolResultContent(output);
										const structuredContent = getMCPAppToolResultStructuredContent(output);

										return (
											<MCPAppView
												key={instance.instanceID}
												instance={instance}
												toolInput={call?.arguments}
												toolResult={{
													content,
													structuredContent,
													isError: output.mcpApp?.isError ?? output.isError,
												}}
											/>
										);
									})}
								</div>
							</div>
						)}
						{hasCitations && (
							<div className="border-base-300 border-t p-1">
								<MessageCitationsBar citations={message.uiCitations} />{' '}
							</div>
						)}
					</div>

					<div className={`${rightColSpan} flex justify-start`}>
						{!isUser && (
							<div className="my-0 mr-0 ml-2 flex size-8 items-center justify-center self-end">
								<FiZap size={24} />
							</div>
						)}
					</div>
				</>
			)}

			{/* Row 2 ── footer bar */}
			<div className={`${leftColSpan} flex justify-end`}>
				{/* If the card row is hidden, keep the avatar aligned with the footer row */}
				{!showBodyWithMedia && isUser && (
					<div className="my-0 mr-2 ml-0 flex size-8 items-center justify-center">
						<FiUser size={24} />
					</div>
				)}
			</div>
			<div className="col-span-10 min-w-0 lg:col-span-9">
				<div
					className={`flex min-w-0 items-center gap-2 overflow-x-auto overscroll-contain px-2 ${showBodyWithMedia ? 'pt-1' : isUser ? 'justify-start' : 'justify-end'}`}
				>
					{hasAttachmentsBar && (
						<div className="flex min-w-0 items-center justify-start overflow-x-hidden px-1 py-0">
							<MessageAttachmentsBar
								isUser={isUser}
								attachments={message.attachments}
								toolChoices={toolChoices}
								toolSelectionIssues={toolSelectionIssues}
								onToolSelectionIssueDetails={handleToolSelectionIssueDetails}
								operationResolution={operationResolution}
								appContextUpdates={appContextUpdates}
								toolCalls={message.uiToolCalls}
								toolOutputs={toolOutputs}
								onToolChoiceDetails={handleToolChoiceDetails}
								onToolCallDetails={handleToolCallDetails}
								onToolOutputDetails={handleToolOutputDetails}
							/>
						</div>
					)}

					<div
						className={`flex min-w-0 flex-1 items-center px-1 py-0 ${!showBodyWithMedia ? (isUser ? 'justify-start' : 'justify-end') : ''}`}
					>
						<MessageFooterArea
							messageID={message.id}
							isUser={isUser}
							cardCopyContent={message.uiContent}
							onEdit={onEdit}
							reasoningContents={message.uiReasoningContents}
							isBusy={isBusy}
							bodyPresent={showBodyWithMedia}
							streamText={streamSnapshot.text || baseContent}
							streamThinking={streamSnapshot.thinking || (hasAnyReasoning ? ' ' : '')}
							disableMarkdown={!renderMarkdown}
							onDisableMarkdownChange={handleDisableMarkdownChange}
							usage={message.usage}
							debugDetails={message.debugDetails}
							debugDetailsError={message.debugDetailsError}
							errorDetails={message.error}
							responseTiming={responseTiming}
						/>
					</div>
				</div>
			</div>
			<div className={`${rightColSpan} flex justify-start`}>
				{/* If the card row is hidden, keep the assistant icon aligned with the footer row */}
				{!showBodyWithMedia && !isUser && (
					<div className="my-0 mr-0 ml-2 flex size-8 items-center justify-center">
						<FiZap size={24} />
					</div>
				)}
			</div>
			{/* Tool choice/call/output details (JSON) */}
			{toolDetailsState ? (
				<ToolDetailsModal
					state={toolDetailsState}
					onClose={() => {
						setToolDetailsState(null);
					}}
				/>
			) : null}
		</div>
	);
}, propsAreEqual);
