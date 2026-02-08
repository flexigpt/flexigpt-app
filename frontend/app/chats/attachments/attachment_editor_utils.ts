import type { Attachment, DirectoryOverflowInfo, UIAttachment } from '@/spec/attachment';
import { AttachmentContentBlockMode, AttachmentErrorReason, AttachmentKind } from '@/spec/attachment';

/**
 * @public
 */
export const MAX_SINGLE_ATTACHMENT_BYTES = 16 * 1024 * 1024; // 16 MiB
export const MAX_FILES_PER_DIRECTORY = 128;

// Directory grouping is UI-only.
export interface DirectoryAttachmentGroup {
	id: string;
	dirPath: string;
	label: string;
	/**
	 * All attachment keys (uiAttachmentKey) that belong to this folder.
	 * Includes both "owned" and "referenced" attachments.
	 */
	attachmentKeys: string[];
	/**
	 * Subset of attachmentKeys created specifically for this folder selection.
	 * Used to decide which attachments to remove when the folder is removed.
	 */
	ownedAttachmentKeys: string[];
	/**
	 * Subdirectories that were not fully walked due to maxFiles limits, etc.
	 */
	overflowDirs: DirectoryOverflowInfo[];
}

// Convert editor attachment to API attachment payload.
export function uiAttachmentToConversation(att: UIAttachment): Attachment {
	return {
		kind: att.kind,
		label: att.label,
		mode: att.mode,
		availableContentBlockModes: att.availableContentBlockModes,
		fileRef: att.fileRef,
		imageRef: att.imageRef,
		urlRef: att.urlRef,
		genericRef: att.genericRef,
	};
}

// Identity key used for de-duplication in the composer.
export function uiAttachmentKey(att: UIAttachment): string {
	let label = att.label;
	if (att.kind === AttachmentKind.file && att.fileRef) {
		label = att.fileRef.path;
	}
	if (att.kind === AttachmentKind.image && att.imageRef) {
		label = att.imageRef.path;
	}
	if (att.kind === AttachmentKind.url && att.urlRef) {
		label = att.urlRef.url;
	}
	return `${att.kind}:${label}`;
}

// Used only for tooltip/debug display in the chips bar.
export function getUIAttachmentPath(att: UIAttachment): string {
	if (att.kind === AttachmentKind.file && att.fileRef) {
		return att.fileRef.path;
	}
	if (att.kind === AttachmentKind.image && att.imageRef) {
		return att.imageRef.path;
	}
	if (att.kind === AttachmentKind.url && att.urlRef) {
		return att.urlRef.url;
	}
	return '';
}

function buildErrorAttachmentForLocalPath(att: Attachment, reason: AttachmentErrorReason): UIAttachment {
	return {
		kind: AttachmentKind.file,
		label: att.label,
		mode: AttachmentContentBlockMode.notReadable,
		availableContentBlockModes: [AttachmentContentBlockMode.notReadable],
		fileRef: att.fileRef,
		imageRef: att.imageRef,
		urlRef: att.urlRef,
		genericRef: att.genericRef,
		isError: true,
		errorReason: reason,
	};
}

/**
 * Classify a local file into an UIAttachment with smart default mode
 * and allowed modes based on extension.
 */
export function buildUIAttachmentForLocalPath(att: Attachment): UIAttachment | undefined {
	if (!att.fileRef && !att.imageRef) {
		return undefined;
	}
	let sizeBytes = 0;
	if (att.fileRef && att.fileRef.size && att.fileRef.size > 0) {
		sizeBytes = att.fileRef.size;
	} else if (att.imageRef && att.imageRef.size && att.imageRef.size > 0) {
		sizeBytes = att.imageRef.size;
	}

	if (typeof sizeBytes === 'number' && sizeBytes > MAX_SINGLE_ATTACHMENT_BYTES) {
		return buildErrorAttachmentForLocalPath(att, AttachmentErrorReason.TooLargeSingle);
	}

	if (att.mode && att.mode == AttachmentContentBlockMode.notReadable) {
		return buildErrorAttachmentForLocalPath(att, AttachmentErrorReason.Unreadable);
	}
	return {
		kind: att.kind,
		label: att.label,
		mode: att.mode ?? AttachmentContentBlockMode.notReadable,
		availableContentBlockModes: att.availableContentBlockModes ?? [AttachmentContentBlockMode.notReadable],
		fileRef: att.fileRef,
		imageRef: att.imageRef,
		urlRef: att.urlRef,
		genericRef: att.genericRef,
		isError: false,
	};
}

/**
 * Build a URL-based attachment with smart default modes.
 */
export function buildUIAttachmentForURL(att: Attachment): UIAttachment {
	return {
		kind: att.kind,
		label: att.label,
		mode: att.mode ?? AttachmentContentBlockMode.notReadable,
		availableContentBlockModes: att.availableContentBlockModes ?? [AttachmentContentBlockMode.notReadable],
		fileRef: att.fileRef,
		imageRef: att.imageRef,
		urlRef: att.urlRef,
		genericRef: att.genericRef,
		isError: false,
	};
}

// Human-readable explanation for an error attachment, for tooltips.
export function getAttachmentErrorMessage(att: UIAttachment): string | null {
	if (!att.isError || !att.errorReason) return null;
	const limit = formatBytes(MAX_SINGLE_ATTACHMENT_BYTES);
	let size = 0;
	if (att.fileRef && att.fileRef.size && att.fileRef.size > 0) {
		size = att.fileRef.size;
	} else if (att.imageRef && att.imageRef.size && att.imageRef.size > 0) {
		size = att.imageRef.size;
	}
	if (size <= 0) {
		return null;
	}
	switch (att.errorReason) {
		case AttachmentErrorReason.TooLargeSingle: {
			return `Too large to attach (${formatBytes(size)}; limit is ${limit}).`;
		}
		case AttachmentErrorReason.TooLargeTotal:
			return 'Too many or too large files in this folder were skipped; only a subset was attached.';
		case AttachmentErrorReason.Unreadable:
		default:
			return 'This file type is not supported or could not be read.';
	}
}

function formatBytes(bytes: number): string {
	if (!Number.isFinite(bytes) || bytes < 0) return `${bytes} B`;
	const units = ['B', 'KB', 'MB', 'GB', 'TB'];
	let i = 0;
	let v = bytes;
	while (v >= 1024 && i < units.length - 1) {
		v /= 1024;
		i++;
	}
	return `${v.toFixed(v >= 10 || i === 0 ? 0 : 1)} ${units[i]}`;
}
