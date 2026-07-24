import { memo, useCallback, useMemo, useState } from 'react';

import { FiUser, FiZap } from 'react-icons/fi';

import type { ConversationMessage } from '@/spec/conversation';
import type { UIToolCall, UIToolOutput } from '@/spec/inference';
import { RoleEnum, Status } from '@/spec/inference';
import type { ToolStoreChoice } from '@/spec/tool';

import type { ToolDetailsState } from '@/chats/composer/tools/tool_details_modal';
import { ToolDetailsModal } from '@/chats/composer/tools/tool_details_modal';
import { buildAppInstanceFromToolOutput } from '@/chats/mcpapps/mcp_app_types';
import { MCPAppView } from '@/chats/mcpapps/mcp_app_view';
import {
	getMCPAppToolResultContent,
	getMCPAppToolResultStructuredContent,
} from '@/chats/messages/mcp_message_context_utils';
import { MessageAttachmentsBar } from '@/chats/messages/message_attachments_bar';
import { MessageCitationsBar } from '@/chats/messages/message_citations_bar';
import type { MessageStreamSource } from '@/chats/messages/message_content_card';
import { MessageContentCard } from '@/chats/messages/message_content_card';
import { MessageFooterArea } from '@/chats/messages/message_footer';
import { MessageThinkingSection } from '@/chats/messages/message_thinking_section';

interface ChatMessageProps {
	message: ConversationMessage;
	streamedText: string;
	streamedThinking: string;
	isBusy: boolean;
	isEditing: boolean;
	deferRichRendering: boolean;
	diffCandidatePaths?: string[];
	onEdit: () => void;
	streamSource?: MessageStreamSource;
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

	if (prev.message.uiDebugDetails !== next.message.uiDebugDetails) {
		//
		// We need to check details as parent is updating details in place for previous message
		return false;
	}
	if (prev.message.debugDetails !== next.message.debugDetails) {
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

	// IMPORTANT: the markdown (and Mermaid) is driven by uiContent.
	// If message objects are mutated in place, message reference may not change,
	// so we must compare the actual content.
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

	// Equivalent freshly-created arrays must not rebuild the markdown and
	// Mermaid subtree.
	if (!stringArraysEqual(prev.diffCandidatePaths, next.diffCandidatePaths)) {
		return false;
	}

	if (prev.streamedText !== next.streamedText) {
		return false;
	}
	if (prev.streamedThinking !== next.streamedThinking) {
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

export const ChatMessage = memo(function ChatMessage({
	message,
	streamedText,
	streamedThinking,
	isBusy,
	isEditing,
	deferRichRendering,
	diffCandidatePaths,
	onEdit,
	streamSource,
}: ChatMessageProps) {
	const isUser = message.role === RoleEnum.User;
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
	const handleToolCallDetails = useCallback((call: UIToolCall) => {
		setToolDetailsState({ kind: 'call', call });
	}, []);
	const handleToolOutputDetails = useCallback((output: UIToolOutput) => {
		setToolDetailsState({ kind: 'output', output });
	}, []);

	const bubbleExtra = [
		isBusy || streamedText || streamedThinking ? '' : 'shadow-lg',
		isEditing ? 'ring-2 ring-primary/70' : '',
	]
		.filter(Boolean)
		.join(' ');

	const baseContent = message.uiContent ?? '';
	const hasAnyContent = /\S/.test(baseContent) || /\S/.test(streamedText);
	const hasError = message.status === Status.Failed || !!message.error;

	const hasAnyReasoning =
		/\S/.test(streamedThinking) ||
		(message.uiReasoningContents?.some(rc => {
			const sum = (rc?.summary ?? []).some(s => /\S/.test(s ?? ''));
			const th = (rc?.thinking ?? []).some(s => /\S/.test(s ?? ''));
			return sum || th;
		}) ??
			false);

	const hasCitations = !isUser && !isBusy && (message.uiCitations?.length ?? 0) > 0;
	const mcpContextItemCount =
		(message.mcpContext?.servers?.length ?? 0) +
		(message.mcpContext?.resources?.length ?? 0) +
		(message.mcpContext?.resourceTemplates?.length ?? 0) +
		(message.mcpContext?.prompts?.length ?? 0);
	const hasMCPContext = mcpContextItemCount > 0;
	const hasSkillContext = (message.enabledSkillRefs?.length ?? 0) > 0 || (message.activeSkillRefs?.length ?? 0) > 0;

	const hasAttachmentsBar =
		(message.attachments?.length ?? 0) > 0 ||
		(message.toolStoreChoices?.length ?? 0) > 0 ||
		(message.mcpAppContextUpdates?.length ?? 0) > 0 ||
		(message.uiToolCalls?.length ?? 0) > 0 ||
		(message.uiToolOutputs?.length ?? 0) > 0 ||
		hasMCPContext ||
		hasSkillContext;

	// Detect MCP Apps in this message's tool outputs.
	const mcpAppViews = useMemo(() => {
		if (isBusy || deferRichRendering) {
			return [];
		}

		const outputs = message.uiToolOutputs ?? [];
		const callsById = new Map((message.uiToolCalls ?? []).map(c => [c.callID || c.id, c] as const));
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
	}, [deferRichRendering, isBusy, message.uiToolCalls, message.uiToolOutputs]);

	const hasMCPAppsView = mcpAppViews.length > 0;

	// Body wrapper should exist only if something inside can render:
	// - content (final/streamed) OR
	// - busy (content card will show loader while busy)
	const showBody = isBusy || hasAnyContent || hasError || hasMCPAppsView;
	return (
		<div className="grid grid-cols-12 p-1" style={{ fontSize: 14 }}>
			{/* Row 1 ── icon + message bubble (only when showCardRow) */}
			{showBody && (
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
								streamedThinking={streamedThinking}
								reasoningContents={message.uiReasoningContents}
								streamSource={streamSource}
							/>
						)}
						<div className="px-4 py-2">
							<MessageContentCard
								messageID={message.id}
								content={baseContent}
								streamedText={streamedText}
								isStreaming={!!streamedText}
								isBusy={isBusy}
								align={align}
								renderAsMarkdown={renderMarkdown && !deferRichRendering}
								diffCandidatePaths={diffCandidatePaths}
								streamSource={streamSource}
							/>
							{/* Fallback for error-only messages with no text content */}
							{!hasAnyContent && hasError && !isBusy && (
								<div className="text-error text-sm opacity-80">
									{message.error?.message || 'An error occurred while processing this request.'}
								</div>
							)}
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
				{!showBody && isUser && (
					<div className="my-0 mr-2 ml-0 flex size-8 items-center justify-center">
						<FiUser size={24} />
					</div>
				)}
			</div>
			<div className="col-span-10 min-w-0 lg:col-span-9">
				<div
					className={`min-w-0 items-center gap-2 px-2 ${hasAttachmentsBar || !showBody ? 'flex' : ''} ${showBody ? 'pt-1' : isUser ? 'justify-start' : 'justify-end'}`}
				>
					{hasAttachmentsBar && (
						<div className="flex min-w-0 items-center justify-start overflow-x-hidden px-1 py-0">
							<MessageAttachmentsBar
								attachments={message.attachments}
								toolChoices={message.toolStoreChoices}
								mcpContext={message.mcpContext}
								mcpAppContextUpdates={message.mcpAppContextUpdates}
								enabledSkillRefs={message.enabledSkillRefs}
								activeSkillRefs={message.activeSkillRefs}
								toolCalls={message.uiToolCalls}
								toolOutputs={message.uiToolOutputs}
								onToolChoiceDetails={handleToolChoiceDetails}
								onToolCallDetails={handleToolCallDetails}
								onToolOutputDetails={handleToolOutputDetails}
							/>
						</div>
					)}

					<div
						className={`min-w-0 items-center px-1 py-0 ${
							+hasAttachmentsBar ? (showBody ? 'flex flex-1' : 'flex') : !showBody ? 'flex' : ''
						} ${!showBody ? (isUser ? 'justify-start' : 'justify-end') : ''}`}
					>
						<MessageFooterArea
							messageID={message.id}
							isUser={isUser}
							cardCopyContent={message.uiContent}
							onEdit={onEdit}
							messageDetails={message.uiDebugDetails ?? ''}
							reasoningContents={message.uiReasoningContents}
							streamedThinking={streamedThinking}
							isStreaming={isBusy || !!streamedText || !!streamedThinking}
							isBusy={isBusy}
							bodyPresent={showBody}
							disableMarkdown={!renderMarkdown}
							onDisableMarkdownChange={handleDisableMarkdownChange}
							usage={message.usage}
							debugDetails={message.debugDetails}
							errorDetails={message.error}
						/>
					</div>
				</div>
			</div>
			<div className={`${rightColSpan} flex justify-start`}>
				{/* If the card row is hidden, keep the assistant icon aligned with the footer row */}
				{!showBody && !isUser && (
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
