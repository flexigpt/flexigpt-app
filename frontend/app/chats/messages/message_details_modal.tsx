import { useEffect, useRef } from 'react';

import { createPortal } from 'react-dom';

import { FiCode, FiX } from 'react-icons/fi';

import type { ReasoningContent } from '@/spec/inference';

import { ModalBackdrop } from '@/components/modal_backdrop';

import { MessageContentCard } from '@/chats/messages/message_content_card';

type MessageDetailsModalProps = {
	isOpen: boolean;
	onClose: () => void;
	messageID: string;
	title: string;
	content: string;
	isBusy: boolean;
	reasoningContents?: ReasoningContent[];
	streamedThinking?: string;
	showReasoningAtTop?: boolean;
};

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

export function MessageDetailsModal({
	isOpen,
	onClose,
	messageID,
	title,
	content,
	isBusy,
	reasoningContents,
	streamedThinking = '',
	showReasoningAtTop = false,
}: MessageDetailsModalProps) {
	const dialogRef = useRef<HTMLDialogElement | null>(null);

	const summaryText = joinReasoningParts(reasoningContents, 'summary');
	const finalThinkingText = joinReasoningParts(reasoningContents, 'thinking');
	const thinkingText = (isBusy ? streamedThinking : finalThinkingText).trimEnd();

	const hasSummary = summaryText.trim().length > 0;
	const hasThinking = thinkingText.trim().length > 0;
	const hasReasoningSection = showReasoningAtTop && (hasSummary || hasThinking);
	const hasDebugContent = content.trim().length > 0;

	// Open the dialog natively when isOpen becomes true
	useEffect(() => {
		if (!isOpen) return;

		const dialog = dialogRef.current;
		if (!dialog) return;

		if (!dialog.open) {
			dialog.showModal();
		}

		return () => {
			// If the component unmounts while the dialog is still open, close it.
			if (dialog.open) {
				dialog.close();
			}
		};
	}, [isOpen]);

	// Sync parent state whenever the dialog is closed (Esc, backdrop, or dialog.close()).
	const handleDialogClose = () => {
		onClose();
	};

	if (!isOpen) return null;

	return createPortal(
		<dialog ref={dialogRef} className="modal" onClose={handleDialogClose}>
			<div className="modal-box bg-base-200 max-h-[80vh] max-w-[80vw] overflow-hidden rounded-2xl p-0">
				<div className="max-h-[80vh] overflow-y-auto p-6">
					{/* header */}
					<div className="mb-4 flex items-center justify-between">
						<h3 className="flex items-center gap-2 text-lg font-bold">
							<FiCode size={16} />
							<span>{title}</span>
						</h3>
						<button
							type="button"
							className="btn btn-sm btn-circle bg-base-300"
							onClick={() => dialogRef.current?.close()}
							aria-label="Close"
						>
							<FiX size={12} />
						</button>
					</div>
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
							/>
						) : !hasReasoningSection && !isBusy ? (
							<div className="text-base-content/70 text-sm">No additional debug details available.</div>
						) : null}
					</div>
				</div>
			</div>

			<ModalBackdrop enabled={true} />
		</dialog>,
		document.body
	);
}
