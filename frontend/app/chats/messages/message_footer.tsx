import { useState } from 'react';

import { FiArrowDown, FiArrowUp, FiCode, FiEdit2, FiMoreHorizontal, FiRepeat } from 'react-icons/fi';

import type { InferenceUsage } from '@/spec/inference';

import { stripCustomMDFences } from '@/lib/text_utils';

import { CopyButton } from '@/components/copy_button';

import { MessageDetailsModal } from '@/chats/messages/message_details_modal';

interface MessageFooterAreaProps {
	messageID: string;
	isUser: boolean;
	cardCopyContent: string;
	onEdit: () => void;
	onResend: () => void;
	messageDetails: string;
	isStreaming: boolean;
	isBusy: boolean;
	disableMarkdown: boolean;
	onDisableMarkdownChange: (checked: boolean) => void;
	usage?: InferenceUsage;
}

export function MessageFooterArea({
	messageID,
	isUser,
	cardCopyContent,
	onEdit,
	onResend,
	messageDetails,
	isStreaming,
	isBusy,
	disableMarkdown,
	onDisableMarkdownChange,
	usage,
}: MessageFooterAreaProps) {
	const [isDetailsOpen, setIsDetailsOpen] = useState(false);

	const hasDetails = !!messageDetails;

	const toggleDetailsModal = () => {
		if (!hasDetails || isBusy) return;
		setIsDetailsOpen(prev => !prev);
	};

	return (
		<>
			<div className="p-0">
				<div className="flex h-8 items-center justify-between">
					<div className="flex gap-2">
						<label
							className={`ml-1 flex items-center space-x-2 truncate ${isBusy ? 'cursor-not-allowed opacity-50' : ''}`}
							title="Disable Markdown"
						>
							<input
								type="checkbox"
								checked={disableMarkdown}
								onChange={e => {
									onDisableMarkdownChange(e.target.checked);
								}}
								className="checkbox checkbox-xs ml-1 rounded-full"
								spellCheck="false"
								disabled={isBusy}
							/>
							<span className="text-base-content text-xs text-nowrap">Disable Markdown</span>
						</label>
						<button
							className={`btn btn-sm flex shrink-0 items-center border-none bg-transparent! shadow-none ${
								isBusy || !hasDetails ? 'btn-disabled' : ''
							}`}
							onClick={toggleDetailsModal}
							aria-label="Details"
							title={hasDetails ? 'Show details' : 'No details'}
							disabled={isBusy || !hasDetails}
						>
							<FiCode size={16} />
						</button>
					</div>

					{isStreaming && (
						<div className="text-sm">
							<div className="flex items-center gap-2 bg-transparent px-4 py-2">
								Streaming
								<FiMoreHorizontal size={14} />
							</div>
						</div>
					)}
					{usage && !isStreaming && (
						<div
							className="tooltip tooltip-top before:whitespace-pre-line"
							data-tip={(() => {
								const parts: string[] = [];

								// Total input
								if (usage.inputTokensTotal > 0) {
									parts.push(`Total input tokens: ${usage.inputTokensTotal}`);
								}

								// Cached + Uncached: only show uncached if cached > 0
								if (usage.inputTokensCached > 0) {
									parts.push(`Input cached tokens: ${usage.inputTokensCached}`);

									if (usage.inputTokensUncached > 0) {
										parts.push(`Input uncached tokens: ${usage.inputTokensUncached}`);
									}
								}

								// Output
								if (usage.outputTokens > 0) {
									parts.push(`Total output tokens: ${usage.outputTokens}`);
								}

								// Reasoning
								if (usage.reasoningTokens > 0) {
									parts.push(`Reasoning tokens: ${usage.reasoningTokens}`);
								}

								return parts.join('\n'); // each item on its own line
							})()}
						>
							<div className="flex items-center bg-transparent px-4 py-2 text-xs">
								<FiArrowUp size={14} /> {usage.inputTokensTotal}
								&nbsp;&nbsp;-&nbsp;&nbsp;
								<FiArrowDown size={14} /> {usage.outputTokens}
							</div>
						</div>
					)}

					<div className="flex items-center gap-1">
						{isUser && (
							<button
								className={`btn btn-sm flex items-center border-none bg-transparent! shadow-none ${
									isBusy ? 'btn-disabled' : ''
								}`}
								onClick={onResend}
								aria-label="Resend Message"
								title="Resend Message"
								disabled={isBusy}
							>
								<FiRepeat size={16} />
							</button>
						)}
						{isUser && (
							<button
								className={`btn btn-sm flex items-center border-none bg-transparent! shadow-none ${
									isBusy ? 'btn-disabled' : ''
								}`}
								onClick={onEdit}
								aria-label="Edit Message"
								title="Edit Message"
								disabled={isBusy}
							>
								<FiEdit2 size={16} />
							</button>
						)}

						<CopyButton
							value={stripCustomMDFences(cardCopyContent)}
							className={`btn btn-sm flex items-center border-none bg-transparent! shadow-none ${
								isBusy ? 'btn-disabled' : ''
							}`}
							size={16}
							disabled={isBusy}
						/>
					</div>
				</div>
			</div>

			{/* Details modal (works for both user & assistant messages) */}
			<MessageDetailsModal
				isOpen={isDetailsOpen && hasDetails}
				onClose={() => {
					setIsDetailsOpen(false);
				}}
				messageID={messageID}
				title={isUser ? 'User message details' : 'Assistant message details'}
				content={messageDetails}
				isBusy={isBusy}
			/>
		</>
	);
}
