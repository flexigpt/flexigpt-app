import { FiFileText, FiImage, FiLink, FiX } from 'react-icons/fi';

import type { AttachmentContentBlockMode, UIAttachment } from '@/spec/attachment';
import { AttachmentKind } from '@/spec/attachment';

import { getAttachmentErrorMessage, getUIAttachmentPath } from '@/chats/attachments/attachment_editor_utils';
import {
	AttachmentContentBlockModeMenu,
	getAttachmentContentBlockModeLabel,
	getAttachmentContentBlockModePillClasses,
	getAttachmentContentBlockModeTooltip,
} from '@/chats/attachments/attachment_mode_menu';

interface AttachmentChipProps {
	attachment: UIAttachment;
	onRemoveAttachment: (att: UIAttachment) => void;
	onChangeAttachmentContentBlockMode: (att: UIAttachment, mode: AttachmentContentBlockMode) => void;

	/**
	 * When true, the chip stretches to fill its container width and keeps
	 * the mode pill + remove button right-justified (used in folder menu).
	 * In the top-level chips bar this should be left as false.
	 */
	fullWidth?: boolean;
}

/**
 * Single attachment pill (file/image/url), used both in the main bar and inside folder menus.
 */
export function AttachmentChip({
	attachment,
	onRemoveAttachment,
	onChangeAttachmentContentBlockMode,
	fullWidth = false,
}: AttachmentChipProps) {
	const att = attachment;
	const isLabelTruncated = att.label.length > 40;
	const label = isLabelTruncated ? att.label.slice(0, 37) + '...' : att.label;

	const path = getUIAttachmentPath(att);

	const icon =
		att.kind === AttachmentKind.image ? (
			<FiImage size={14} />
		) : att.kind === AttachmentKind.url ? (
			<FiLink size={14} />
		) : (
			<FiFileText size={14} />
		);

	const errorMessage = getAttachmentErrorMessage(att);

	// Build a tooltip that only adds *extra* information:
	// - full label if we truncated visually
	// - underlying path / URL if it differs from the label
	// - error message (if this is an error attachment)
	const tooltipLines: string[] = [];
	if (isLabelTruncated) {
		tooltipLines.push(att.label);
	}
	if (path && path !== att.label) {
		tooltipLines.push(path);
	}
	if (errorMessage) {
		tooltipLines.push(errorMessage);
	}
	const title = tooltipLines.length > 0 ? tooltipLines.join('\n') : undefined;

	return (
		<div
			className={`bg-base-200 hover:bg-base-300/80 text-base-content flex items-center gap-2 rounded-2xl px-2 py-0 ${
				fullWidth ? 'w-full' : 'shrink-0'
			}`}
			title={title}
			data-attachment-chip="attachment"
		>
			{/* Left: icon + label */}
			<span className="shrink-0">{icon}</span>
			<span className={`${fullWidth ? 'min-w-0 flex-1' : 'max-w-64'} truncate`}>{label}</span>

			{/* Right: mode pill / menu + remove button, right-justified within the chip */}
			<div className="ml-auto flex shrink-0 items-center gap-1">
				{att.availableContentBlockModes.length > 1 ? (
					<AttachmentContentBlockModeMenu
						attachment={att}
						onChangeAttachmentContentBlockMode={onChangeAttachmentContentBlockMode}
					/>
				) : att.availableContentBlockModes.length === 1 ? (
					<span
						className={getAttachmentContentBlockModePillClasses(att.mode, false)}
						title={getAttachmentContentBlockModeTooltip(att.mode)}
						data-attachment-mode-pill
					>
						{getAttachmentContentBlockModeLabel(att.mode)}
					</span>
				) : null}

				<button
					type="button"
					className="btn btn-ghost btn-xs text-error shrink-0 px-1 py-0 shadow-none"
					onClick={() => {
						onRemoveAttachment(att);
					}}
					title="Remove attachment"
					aria-label="Remove attachment"
				>
					<FiX size={14} />
				</button>
			</div>
		</div>
	);
}
