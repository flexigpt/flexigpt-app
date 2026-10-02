import type { Attachment, DirectoryAttachmentsResult, PathAttachmentsResult } from '@/spec/attachment';

import {
	requiredObject,
	requireWailsString,
	wailsArrayOrEmpty,
	wailsObjectArrayOrEmpty,
} from '@/apis/wailsapi/transport';

export function directoryAttachmentsResultFromWails(value: unknown, operation: string): DirectoryAttachmentsResult {
	const result = requiredObject<DirectoryAttachmentsResult>(value, operation);

	return {
		...result,
		attachments: wailsObjectArrayOrEmpty<Attachment>(result.attachments, `${operation}.attachments`),
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
		fileAttachments: wailsObjectArrayOrEmpty<Attachment>(result.fileAttachments, `${operation}.fileAttachments`),
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
