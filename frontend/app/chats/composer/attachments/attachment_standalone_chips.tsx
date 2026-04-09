import { FiChevronUp, FiPaperclip, FiX } from 'react-icons/fi';

import { Menu, MenuButton, MenuItem, useMenuStore } from '@ariakit/react';

import type { AttachmentContentBlockMode, UIAttachment } from '@/spec/attachment';

import { AttachmentChip } from '@/chats/composer/attachments/attachment_chip';
import { uiAttachmentKey } from '@/chats/composer/attachments/attachment_editor_utils';

interface StandaloneAttachmentsChipProps {
	attachments: UIAttachment[];
	onRemoveAttachment: (att: UIAttachment) => void;
	onChangeAttachmentContentBlockMode: (att: UIAttachment, mode: AttachmentContentBlockMode) => void;
}

/**
 * Aggregated "Attachments" chip for standalone files/links.
 * Opens a dropdown listing each attachment as a full-width chip.
 * Includes a "remove all" cross that clears all standalone attachments.
 */
export function StandaloneAttachmentsChip({
	attachments,
	onRemoveAttachment,
	onChangeAttachmentContentBlockMode,
}: StandaloneAttachmentsChipProps) {
	const count = attachments.length;
	const menu = useMenuStore({ placement: 'bottom-start', focusLoop: true });

	const title = `Attachments\n${count} item${count === 1 ? '' : 's'} attached`;
	if (count === 0) return null;

	return (
		<div
			className="bg-base-200 text-base-content flex shrink-0 items-center gap-1 rounded-2xl px-2 py-0"
			title={title}
			data-attachment-chip="attachments-group"
		>
			<FiPaperclip size={14} />
			<span className="max-w-24 truncate">Attachments</span>
			<span className="text-base-content/60 whitespace-nowrap">{count}</span>

			<MenuButton
				store={menu}
				className="btn btn-ghost btn-xs px-0 py-0 shadow-none"
				aria-label="Show attached files and links"
				title="Show attached files and links"
			>
				<FiChevronUp size={14} />
			</MenuButton>

			{/* Remove all standalone attachments */}
			<button
				type="button"
				className="btn btn-ghost btn-xs text-error shrink-0 px-0 py-0 shadow-none"
				onClick={() => {
					// Call the existing single-remove handler for each attachment.
					for (const att of attachments) {
						onRemoveAttachment(att);
					}
				}}
				title="Remove all attachments"
				aria-label="Remove all attachments"
			>
				<FiX size={14} />
			</button>

			<Menu
				store={menu}
				gutter={8}
				overflowPadding={8}
				className="rounded-box bg-base-100 text-base-content border-base-300 z-50 max-h-72 max-w-lg min-w-60 overflow-y-auto border p-2 shadow-xl focus-visible:outline-none"
				autoFocusOnShow
			>
				<div className="text-base-content/70 mb-1 text-xs font-semibold">Attachments</div>

				{attachments.map(att => (
					<MenuItem
						key={uiAttachmentKey(att)}
						store={menu}
						hideOnClick={false}
						className="data-active-item:bg-base-200 mb-1 rounded-xl last:mb-0"
					>
						<AttachmentChip
							attachment={att}
							onRemoveAttachment={onRemoveAttachment}
							onChangeAttachmentContentBlockMode={onChangeAttachmentContentBlockMode}
							fullWidth
						/>
					</MenuItem>
				))}
			</Menu>
		</div>
	);
}
