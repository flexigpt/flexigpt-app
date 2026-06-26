import { ATTACHMENT_MODE_DESC, ATTACHMENT_MODE_LABELS, AttachmentContentBlockMode } from '@/spec/attachment';

export function getAttachmentContentBlockModeLabel(mode: AttachmentContentBlockMode): string {
	return ATTACHMENT_MODE_LABELS[mode] ?? mode;
}

export function getAttachmentContentBlockModeTooltip(mode: AttachmentContentBlockMode): string {
	if (mode === AttachmentContentBlockMode.notReadable) {
		return 'This attachment could not be read (unsupported type, too large, or inaccessible).';
	}

	const desc = ATTACHMENT_MODE_DESC[mode];
	if (desc) {
		// Tooltip focuses on *extra* explanation; the pill text already shows the label.
		return desc;
	}
	return getAttachmentContentBlockModeLabel(mode);
}

export function getAttachmentContentBlockModePillClasses(
	mode: AttachmentContentBlockMode,
	interactive: boolean
): string {
	const base = 'inline-flex items-center gap-1 rounded-full border px-2 py-[1px] text-xs/tight transition-colors';

	const interactiveClasses = interactive
		? 'cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-base-300'
		: '';

	const isError = mode === AttachmentContentBlockMode.notReadable;

	if (isError) {
		return [base, interactive ? 'hover:bg-error/20' : '', 'border-error/40 bg-error/10 text-error', interactiveClasses]
			.filter(Boolean)
			.join(' ');
	}

	return [
		base,
		interactive ? 'hover:bg-base-200' : '',
		'border-base-300 bg-base-100 text-base-content/80',
		interactiveClasses,
	]
		.filter(Boolean)
		.join(' ');
}
