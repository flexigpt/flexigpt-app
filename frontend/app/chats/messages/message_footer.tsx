import { useState } from 'react';

import { FiArrowDown, FiArrowUp, FiCode, FiEdit2 } from 'react-icons/fi';

import type { InferenceUsage } from '@/spec/inference';

import { stripCustomMDFences } from '@/lib/text_utils';

import { HoverTip } from '@/components/ariakit_hover_tip';
import { CopyButton } from '@/components/copy_button';

import { MessageDetailsModal } from '@/chats/messages/message_details_modal';

interface MessageFooterAreaProps {
	messageID: string;
	isUser: boolean;
	cardCopyContent: string;
	onEdit: () => void;
	messageDetails: string;
	isStreaming: boolean;
	isBusy: boolean;
	bodyPresent: boolean;
	disableMarkdown: boolean;
	onDisableMarkdownChange: (checked: boolean) => void;
	usage?: InferenceUsage;
}

export function MessageFooterArea({
	messageID,
	isUser,
	cardCopyContent,
	onEdit,
	messageDetails,
	isStreaming,
	isBusy,
	bodyPresent,
	disableMarkdown,
	onDisableMarkdownChange,
	usage,
}: MessageFooterAreaProps) {
	const [isDetailsOpen, setIsDetailsOpen] = useState(false);

	const hasDetails = !!messageDetails;
	const hasContent = !!cardCopyContent;
	const toggleDetailsModal = () => {
		if (!hasDetails || isBusy) return;
		setIsDetailsOpen(prev => !prev);
	};

	const usageTooltip = usage
		? (() => {
				const parts: string[] = [];

				if (usage.inputTokensTotal > 0) parts.push(`Total input tokens: ${usage.inputTokensTotal}`);
				if (usage.inputTokensCached > 0) {
					parts.push(`Input cached tokens: ${usage.inputTokensCached}`);
					if (usage.inputTokensUncached > 0) {
						parts.push(`Input uncached tokens: ${usage.inputTokensUncached}`);
					}
				}
				if (usage.outputTokens > 0) parts.push(`Total output tokens: ${usage.outputTokens}`);
				if (usage.reasoningTokens > 0) parts.push(`Reasoning tokens: ${usage.reasoningTokens}`);

				return parts.join('\n');
			})()
		: '';

	return (
		<div className={bodyPresent ? 'grow' : ''}>
			<div className={`flex items-center space-x-6 ${bodyPresent ? 'justify-between' : ''}`}>
				{usage && !isStreaming && (
					<HoverTip content={usageTooltip} placement="top" wrapperElement="div">
						<div className="flex items-center bg-transparent p-0 text-xs">
							<div className="flex items-center">
								<FiArrowUp size={14} />
								<span className="shrink-0">{usage.inputTokensTotal}&nbsp;&nbsp;-&nbsp;&nbsp;</span>
								<FiArrowDown size={14} /> <span>{usage.outputTokens}</span>
							</div>
						</div>
					</HoverTip>
				)}

				<div className={`flex items-center justify-end space-x-6 ${!isStreaming && !usage ? 'w-full' : ''}`}>
					{hasContent && !isBusy && (
						<label className={`ml-1 flex h-full items-center space-x-2 truncate p-1`} title="Disable Markdown">
							<input
								type="checkbox"
								checked={disableMarkdown}
								onChange={e => {
									onDisableMarkdownChange(e.target.checked);
								}}
								className="checkbox checkbox-xs ml-1 rounded-full"
								spellCheck="false"
							/>
							<span className="text-base-content text-xs text-nowrap">Disable Markdown</span>
						</label>
					)}

					{hasDetails && !isBusy && (
						<HoverTip content="Show message details" placement="top">
							<button
								className="btn btn-sm flex items-center border-none bg-transparent! p-0 shadow-none"
								onClick={toggleDetailsModal}
								aria-label="Show message details"
							>
								<FiCode size={16} />
							</button>
						</HoverTip>
					)}
					{isUser && !isBusy && (
						<HoverTip content="Edit and resend from this message" placement="top">
							<button
								className="btn btn-sm flex items-center border-none bg-transparent! p-0 shadow-none"
								onClick={onEdit}
								aria-label="Edit and resend from this message"
							>
								<FiEdit2 size={16} />
							</button>
						</HoverTip>
					)}

					{cardCopyContent !== '' && !isBusy && (
						<HoverTip content="Copy message text" placement="top">
							<CopyButton
								value={stripCustomMDFences(cardCopyContent)}
								className="btn btn-sm flex items-center border-none bg-transparent! p-0 shadow-none"
								size={16}
							/>
						</HoverTip>
					)}
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
		</div>
	);
}
