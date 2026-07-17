import { FiCode } from 'react-icons/fi';

import type { ReasoningContent } from '@/spec/inference';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalBackdrop } from '@/components/modal/modal_backdrop';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalHeader } from '@/components/modal/modal_header';

import { MessageContentCard } from '@/chats/messages/message_content_card';

interface MessageDetailsModalProps {
	isOpen: boolean;
	onClose: () => void;
	messageID: string;
	title: string;
	content: string;
	isBusy: boolean;
	reasoningContents?: ReasoningContent[];
	streamedThinking?: string;
	showReasoningAtTop?: boolean;
}

function joinReasoningParts(reasoning: ReasoningContent[] | undefined, key: 'summary' | 'thinking'): string {
	const items = reasoning ?? [];
	const parts: string[] = [];

	for (const rc of items) {
		const arr = rc?.[key];
		if (Array.isArray(arr) && arr.length > 0) {
			parts.push(arr.join('\n'));
		}
	}

	return parts.join('\n\n').trim();
}

function MessageDetailsModalContent({
	messageID,
	title,
	content,
	isBusy,
	reasoningContents,
	streamedThinking = '',
	showReasoningAtTop = false,
}: MessageDetailsModalProps) {
	const { requestClose } = useModalDialogController();

	const summaryText = joinReasoningParts(reasoningContents, 'summary');
	const finalThinkingText = joinReasoningParts(reasoningContents, 'thinking');
	const thinkingText = (isBusy ? streamedThinking : finalThinkingText).trimEnd();

	const hasSummary = summaryText.trim().length > 0;
	const hasThinking = thinkingText.trim().length > 0;
	const hasReasoningSection = showReasoningAtTop && (hasSummary || hasThinking);
	const hasDebugContent = content.trim().length > 0;

	return (
		<>
			<div className="modal-box bg-base-200 flex max-h-[80vh] max-w-[80vw] flex-col overflow-hidden rounded-2xl p-0">
				<ModalHeader
					title={
						<span className="flex items-center gap-2">
							<FiCode size={16} />
							<span>{title}</span>
						</span>
					}
					onClose={() => {
						requestClose();
					}}
				/>
				<div className="min-h-0 flex-1 overflow-y-auto p-6">
					<div className="mt-2">
						{hasReasoningSection && (
							<div className="border-base-300 bg-base-100 mb-4 rounded-2xl border p-4">
								<div className="mb-3 text-sm font-semibold">Thinking content</div>

								{hasSummary && (
									<div className={hasThinking ? 'mb-4' : ''}>
										<MessageContentCard
											messageID={`${messageID}:details:reasoning-summary`}
											content={summaryText}
											streamedText=""
											isStreaming={false}
											isBusy={false}
											align="items-start text-left"
											renderAsMarkdown={false}
										/>
									</div>
								)}

								{hasThinking && (
									<div>
										<MessageContentCard
											messageID={`${messageID}:details:reasoning-thinking`}
											content={thinkingText}
											streamedText=""
											isStreaming={false}
											isBusy={false}
											align="items-start text-left"
											renderAsMarkdown={false}
										/>
									</div>
								)}
							</div>
						)}

						{hasDebugContent ? (
							<MessageContentCard
								messageID={messageID}
								content={content}
								streamedText=""
								isStreaming={false}
								isBusy={isBusy}
								align="items-start text-left"
								renderAsMarkdown={true}
								defaultCodeBlockExpanded={false}
							/>
						) : !hasReasoningSection && !isBusy ? (
							<div className="text-base-content/70 text-sm">No additional debug details available.</div>
						) : null}
					</div>
				</div>
				<ModalActions>
					<button
						type="button"
						className="btn bg-base-300 rounded-xl"
						onClick={() => {
							requestClose();
						}}
					>
						Close
					</button>
				</ModalActions>
			</div>

			<ModalBackdrop enabled={true} />
		</>
	);
}

export function MessageDetailsModal(props: MessageDetailsModalProps) {
	if (!props.isOpen) {
		return null;
	}

	return (
		<ModalDialog isOpen={props.isOpen} onClose={props.onClose}>
			<MessageDetailsModalContent {...props} />
		</ModalDialog>
	);
}
