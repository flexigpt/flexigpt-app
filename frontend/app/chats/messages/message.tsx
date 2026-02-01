import { memo, useState } from 'react';

import { FiUser, FiZap } from 'react-icons/fi';

import type { ConversationMessage } from '@/spec/conversation';
import { RoleEnum } from '@/spec/inference';

import { CustomMDLanguage } from '@/lib/text_utils';

import { MessageAttachmentsBar } from '@/chats/messages/message_attachments_bar';
import { MessageCitationsBar } from '@/chats/messages/message_citations_bar';
import { MessageContentCard } from '@/chats/messages/message_content_card';
import { MessageFooterArea } from '@/chats/messages/message_footer';
import { ToolDetailsModal, type ToolDetailsState } from '@/chats/tools/tool_details_modal';

// Builds final content string: reasoning blocks (summary + thinking) + original content.
// - Uses `reasoningContents` if present.
// - Ignores redactedThinking.
function buildEffectiveContentWithReasoning(message: ConversationMessage): string {
	const baseContent = message.uiContent;
	const reasoningContents = message.uiReasoningContents ?? [];

	if (reasoningContents.length === 0) {
		return baseContent;
	}

	const summaryParts: string[] = [];
	const thinkingParts: string[] = [];

	for (const rc of reasoningContents) {
		if (Array.isArray(rc.summary) && rc.summary.length > 0) {
			// rc.summary: string[]
			summaryParts.push(rc.summary.join('\n'));
		}

		if (Array.isArray(rc.thinking) && rc.thinking.length > 0) {
			// rc.thinking: string[]
			thinkingParts.push(rc.thinking.join('\n'));
		}

		// We intentionally ignore rc.redactedThinking and rc.encryptedContent for display.
	}

	// If we still have nothing, just return the base content.
	if (!summaryParts.length && !thinkingParts.length) {
		return baseContent;
	}

	let reasoningText = '';

	if (summaryParts.length) {
		const summaryText = summaryParts.join('\n\n');
		reasoningText += `\n~~~${CustomMDLanguage.ThinkingSummary}\n${summaryText}\n~~~\n`;
	}

	if (thinkingParts.length) {
		const thinkingText = thinkingParts.join('\n\n');
		reasoningText += `\n~~~${CustomMDLanguage.Thinking}\n${thinkingText}\n~~~\n`;
	}

	// If the message has no visible content, just return the reasoning blocks.
	if (!baseContent.trim()) {
		return reasoningText.trimStart();
	}

	// Otherwise: reasoning (summary + thinking) followed by the normal content.
	return `${reasoningText}\n${baseContent}\n`;
}

interface ChatMessageProps {
	message: ConversationMessage;
	onEdit: () => void;
	onResend: () => void;
	streamedMessage: string;
	isPending: boolean;
	isBusy: boolean;
	isEditing: boolean;
}

function propsAreEqual(prev: ChatMessageProps, next: ChatMessageProps) {
	if (prev.isPending !== next.isPending) return false;
	if (prev.isBusy !== next.isBusy) return false;
	if (prev.isEditing !== next.isEditing) return false;

	if (prev.message.uiDebugDetails !== next.message.uiDebugDetails) {
		//
		// We need to check details as parent is updating details in place for previous message
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
	if (prev.message.status !== next.message.status) return false;
	if (prev.message.outputs !== next.message.outputs) return false;
	if (prev.message.uiCitations !== next.message.uiCitations) return false;

	// We only care if THIS row’s streamed text changed.
	if (prev.streamedMessage !== next.streamedMessage) return false;

	// If the *object reference* for the ConversationMessage changes
	// react must re-render (content edited, message appended).
	if (prev.message !== next.message) return false;

	// Everything else is the same: skip.
	return true;
}

export const ChatMessage = memo(function ChatMessage({
	message,
	onEdit,
	onResend,
	streamedMessage,
	isPending,
	isBusy,
	isEditing,
}: ChatMessageProps) {
	const isUser = message.role === RoleEnum.User;
	const align = !isUser ? 'items-end text-left' : 'items-start text-left';
	const leftColSpan = !isUser ? 'col-span-1 lg:col-span-2' : 'col-span-1';
	const rightColSpan = !isUser ? 'col-span-1' : 'col-span-1 lg:col-span-2';

	const [renderMarkdown, setRenderMarkdown] = useState(!isUser);
	const [toolDetailsState, setToolDetailsState] = useState<ToolDetailsState>(null);

	const bubbleExtra = [streamedMessage ? '' : 'shadow-lg', isEditing ? 'ring-2 ring-primary/70' : '']
		.filter(Boolean)
		.join(' ');

	const effectiveContent = buildEffectiveContentWithReasoning(message);

	return (
		<div className="mb-2 grid grid-cols-12 grid-rows-[auto_auto]" style={{ fontSize: 14 }}>
			{/* Row 1 ── icon + message bubble */}
			<div className={`${leftColSpan} row-start-1 row-end-1 flex justify-end`}>
				{isUser && (
					<div className="my-0 mr-2 ml-0 flex h-8 w-8 items-center justify-center self-end">
						<FiUser size={24} />
					</div>
				)}
			</div>

			<div
				className={`bg-base-100 col-span-10 row-start-1 row-end-1 overflow-x-auto rounded-2xl p-0 lg:col-span-9 ${bubbleExtra}`}
			>
				<div className="px-4 py-2">
					<MessageContentCard
						messageID={message.id}
						content={effectiveContent}
						streamedText={streamedMessage}
						isStreaming={!!streamedMessage}
						isBusy={isBusy}
						isPending={isPending}
						align={align}
						renderAsMarkdown={renderMarkdown}
					/>
				</div>
				{!isUser && message.uiCitations && (
					<div className="border-base-300 border-t p-1">
						<MessageCitationsBar citations={message.uiCitations} />{' '}
					</div>
				)}

				<div
					className={`flex w-full min-w-0 items-center overflow-x-hidden px-1 py-0 ${effectiveContent !== '' ? 'border-base-300 border-t' : ''}`}
				>
					<MessageAttachmentsBar
						attachments={message.attachments}
						toolChoices={message.toolStoreChoices}
						toolCalls={message.uiToolCalls}
						toolOutputs={message.uiToolOutputs}
						onToolChoiceDetails={choice => {
							setToolDetailsState({ kind: 'choice', choice });
						}}
						onToolCallDetails={call => {
							setToolDetailsState({ kind: 'call', call });
						}}
						onToolOutputDetails={output => {
							setToolDetailsState({ kind: 'output', output });
						}}
					/>
				</div>
			</div>

			<div className={`${rightColSpan} row-start-1 row-end-1 flex justify-start`}>
				{!isUser && (
					<div className="my-0 mr-0 ml-2 flex h-8 w-8 items-center justify-center self-end">
						<FiZap size={24} />
					</div>
				)}
			</div>

			{/* Row 2 ── footer bar */}
			<div className={`${leftColSpan} row-start-2 row-end-2`} />
			<div className="col-span-10 row-start-2 row-end-2 lg:col-span-9">
				<MessageFooterArea
					messageID={message.id}
					isUser={isUser}
					cardCopyContent={message.uiContent}
					onEdit={onEdit}
					onResend={onResend}
					messageDetails={message.uiDebugDetails ?? ''}
					isStreaming={!!streamedMessage}
					isBusy={isBusy}
					disableMarkdown={!renderMarkdown}
					onDisableMarkdownChange={checked => {
						setRenderMarkdown(!checked);
					}}
					usage={message.usage}
				/>
			</div>
			<div className={`${rightColSpan} row-start-2 row-end-2`} />

			{/* Tool choice/call/output details (JSON) */}
			<ToolDetailsModal
				state={toolDetailsState}
				onClose={() => {
					setToolDetailsState(null);
				}}
			/>
		</div>
	);
}, propsAreEqual);
