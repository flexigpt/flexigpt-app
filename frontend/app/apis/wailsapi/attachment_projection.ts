import type { Attachment, DirectoryAttachmentsResult, PathAttachmentsResult } from '@/spec/attachment';
import { AttachmentContentBlockKind, AttachmentContentBlockMode, AttachmentKind } from '@/spec/attachment';

import {
	enumFromWails,
	optionalEnumFromWails,
	requiredObject,
	requireWailsString,
	wailsArrayOrEmpty,
	wailsObjectArrayOrEmpty,
} from '@/apis/wailsapi/transport';

export function attachmentFromWails(value: unknown, operation: string): Attachment {
	const rawAttachment = requiredObject<Record<string, unknown>>(value, operation);
	const attachment = requiredObject<Attachment>(value, operation);

	const contentBlock =
		rawAttachment.contentBlock === null || rawAttachment.contentBlock === undefined
			? undefined
			: (() => {
					const rawContentBlock = requiredObject<Record<string, unknown>>(
						rawAttachment.contentBlock,
						`${operation}.contentBlock`
					);
					const current = requiredObject<NonNullable<Attachment['contentBlock']>>(
						rawAttachment.contentBlock,
						`${operation}.contentBlock`
					);

					return {
						...current,
						kind: enumFromWails(rawContentBlock.kind, AttachmentContentBlockKind, `${operation}.contentBlock.kind`),
					};
				})();

	return {
		...attachment,
		kind: enumFromWails(rawAttachment.kind, AttachmentKind, `${operation}.kind`),
		mode: optionalEnumFromWails(rawAttachment.mode, AttachmentContentBlockMode, `${operation}.mode`),
		availableContentBlockModes:
			rawAttachment.availableContentBlockModes === null || rawAttachment.availableContentBlockModes === undefined
				? undefined
				: wailsArrayOrEmpty(rawAttachment.availableContentBlockModes, `${operation}.availableContentBlockModes`).map(
						(mode, index) =>
							enumFromWails(mode, AttachmentContentBlockMode, `${operation}.availableContentBlockModes[${index}]`)
					),
		contentBlock,
	};
}

export function directoryAttachmentsResultFromWails(value: unknown, operation: string): DirectoryAttachmentsResult {
	const result = requiredObject<DirectoryAttachmentsResult>(value, operation);

	return {
		...result,
		attachments: wailsObjectArrayOrEmpty(result.attachments, `${operation}.attachments`).map((attachment, index) =>
			attachmentFromWails(attachment, `${operation}.attachments[${index}]`)
		),
		overflowDirs: wailsObjectArrayOrEmpty<DirectoryAttachmentsResult['overflowDirs'][number]>(
			result.overflowDirs,
			`${operation}.overflowDirs`
		),
	};
}

export function pathAttachmentsResultFromWails(value: unknown, operation: string): PathAttachmentsResult {
	const result = requiredObject<PathAttachmentsResult>(value, operation);

	return {
		...result,
		fileAttachments: wailsObjectArrayOrEmpty(result.fileAttachments, `${operation}.fileAttachments`).map(
			(attachment, index) => attachmentFromWails(attachment, `${operation}.fileAttachments[${index}]`)
		),
		dirAttachments: wailsObjectArrayOrEmpty<DirectoryAttachmentsResult>(
			result.dirAttachments,
			`${operation}.dirAttachments`
		).map((directory, index) =>
			directoryAttachmentsResultFromWails(directory, `${operation}.dirAttachments[${index}]`)
		),
		errors: wailsArrayOrEmpty<string>(result.errors, `${operation}.errors`).map((error, index) =>
			requireWailsString(error, `${operation}.errors[${index}]`)
		),
	};
}
