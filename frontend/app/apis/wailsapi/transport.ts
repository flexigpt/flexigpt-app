import type { Attachment, DirectoryAttachmentsResult, PathAttachmentsResult } from '@/spec/attachment';
import { AttachmentContentBlockKind, AttachmentContentBlockMode, AttachmentKind } from '@/spec/attachment';

import type { JSONRawString, JSONSchema } from '@/lib/jsonschema_utils';

export type WailsObject = Record<string, unknown>;

const DEFAULT_MAX_PAGES = 1_000;

function isByte(value: unknown): value is number {
	return typeof value === 'number' && Number.isInteger(value) && value >= 0 && value <= 255;
}

/**
 * Wails transports Go string aliases as plain JavaScript strings. Validate
 * values at the bridge boundary before exposing frontend enum types.
 */
export function enumFromWails<T extends string>(value: unknown, enumValues: Record<string, T>, field: string): T {
	const allowed = Object.values(enumValues) as T[];

	if (typeof value !== 'string' || !allowed.includes(value as T)) {
		throw new Error(`Invalid ${field}: ${String(value)}`);
	}

	return value as T;
}

export function requireWailsObject(value: unknown, field: string): WailsObject {
	if (value === null || typeof value !== 'object' || Array.isArray(value)) {
		throw new Error(`${field} returned an invalid object.`);
	}

	return value as WailsObject;
}

export function requireWailsArray<T = unknown>(value: unknown, field: string): T[] {
	if (!Array.isArray(value)) {
		throw new TypeError(`${field} returned an invalid array.`);
	}

	return value as T[];
}

export function optionalWailsArray(value: unknown, field: string): unknown[] | undefined {
	if (value === undefined || value === null) {
		return undefined;
	}

	return requireWailsArray(value, field);
}

export function requireWailsString(value: unknown, field: string): string {
	if (typeof value !== 'string') {
		throw new TypeError(`${field} returned an invalid string.`);
	}

	return value;
}

export function optionalWailsString(value: unknown, field: string): string | undefined {
	if (value === undefined || value === null) {
		return undefined;
	}

	return requireWailsString(value, field);
}

export function requireWailsBoolean(value: unknown, field: string): boolean {
	if (typeof value !== 'boolean') {
		throw new TypeError(`${field} returned an invalid boolean.`);
	}

	return value;
}

export function requireWailsNumber(value: unknown, field: string): number {
	if (typeof value !== 'number' || !Number.isFinite(value)) {
		throw new TypeError(`${field} returned an invalid number.`);
	}

	return value;
}

export function requireNonBlankString(value: string, field: string): string {
	if (value.trim().length === 0) {
		throw new Error(`${field} must not be empty.`);
	}

	return value;
}

export function omitUndefined<T extends Record<string, unknown>>(value: T): Partial<T> {
	return Object.fromEntries(Object.entries(value).filter(([, item]) => item !== undefined)) as Partial<T>;
}

export async function collectAllPages<T>(
	fetchPage: (pageToken: string | undefined) => Promise<{ items: T[]; nextPageToken?: string }>,
	maxPages = DEFAULT_MAX_PAGES
): Promise<T[]> {
	const items: T[] = [];
	const seenTokens = new Set<string>();
	let pageToken: string | undefined;

	for (let page = 0; page < maxPages; page++) {
		const result = await fetchPage(pageToken);
		items.push(...result.items);

		if (!result.nextPageToken) {
			return items;
		}

		if (seenTokens.has(result.nextPageToken)) {
			throw new Error('Pagination response repeated a page token.');
		}

		seenTokens.add(result.nextPageToken);
		pageToken = result.nextPageToken;
	}

	throw new Error(`Pagination exceeded the ${maxPages}-page safety limit.`);
}

export function requireWailsBody<T>(body: T | null | undefined, operation: string): T {
	if (body === null || body === undefined) {
		throw new Error(`${operation} returned an empty response body.`);
	}
	return body;
}

export function optionalWailsBody<T>(body: T | null | undefined): T | undefined {
	return body === null || body === undefined ? undefined : body;
}

function createAbortError(): Error {
	if (typeof DOMException !== 'undefined') {
		return new DOMException('Aborted', 'AbortError');
	}

	const error = new Error('Aborted');
	error.name = 'AbortError';
	return error;
}

export function throwIfAborted(signal?: AbortSignal): void {
	if (signal?.aborted) {
		throw createAbortError();
	}
}

export function toFrontendTimestamp(value: unknown, field: string): string {
	return toFrontendDate(value, field).toISOString();
}

export function toFrontendDate(value: unknown, field: string): Date {
	if (value instanceof Date) {
		return value;
	}

	if (typeof value === 'string' || typeof value === 'number') {
		const date = new Date(value);
		if (!Number.isNaN(date.getTime())) {
			return date;
		}
	}

	throw new Error(`Invalid date received for ${field}.`);
}

export function optionalFrontendDate(value: unknown, field: string): Date | undefined {
	if (value === undefined || value === null || value === '') {
		return undefined;
	}

	return toFrontendDate(value, field);
}

/**
 * `json.RawMessage` is generated by Wails as `number[]`, but it is transported
 * as raw JSON. Send the parsed JSON object, not an UTF-8 byte array.
 */
export function rawJSONObjectToWails(value: JSONRawString, field: string): unknown {
	if (typeof value !== 'string') {
		throw new TypeError(`${field} must be valid JSON.`);
	}

	let parsed: unknown;

	try {
		parsed = JSON.parse(value);
	} catch {
		throw new Error(`${field} must be valid JSON.`);
	}

	if (parsed === null || Array.isArray(parsed) || typeof parsed !== 'object') {
		throw new Error(`${field} must be a JSON object.`);
	}

	return parsed;
}

/**
 * Supports both Wails representations seen for `json.RawMessage`:
 *
 * - Raw JSON object
 * - Generated `number[]` byte representation
 */
export function rawJSONFromWails(value: unknown, field: string): JSONRawString {
	if (Array.isArray(value) && value.every(item => typeof item === 'number')) {
		if (
			!value.every(b => {
				return isByte(b);
			})
		) {
			throw new Error(`${field} contains an invalid byte array.`);
		}
		const text = new TextDecoder('utf-8', { fatal: true }).decode(Uint8Array.from(value));
		JSON.parse(text);
		return text;
	}

	if (typeof value === 'string') {
		JSON.parse(value);
		return value;
	}

	if (value === null || typeof value !== 'object') {
		throw new Error(`${field} is not valid JSON.`);
	}

	const serialized = JSON.stringify(value);
	if (serialized === undefined) {
		throw new Error(`${field} is not valid JSON.`);
	}

	return serialized;
}

export function rawJSONToWails(value: JSONRawString, field: string): string {
	if (typeof value !== 'string') {
		throw new TypeError(`${field} must be valid JSON.`);
	}

	try {
		JSON.parse(value);
	} catch {
		throw new Error(`${field} must be valid JSON.`);
	}

	return value;
}

export function jsonObjectFromWails(value: unknown, field: string): JSONSchema {
	const parsed = JSON.parse(rawJSONFromWails(value, field));

	if (parsed === null || Array.isArray(parsed) || typeof parsed !== 'object') {
		throw new Error(`${field} must be a JSON object.`);
	}

	return parsed as JSONSchema;
}

export function jsonObjectToWails(value: JSONSchema, field: string): string {
	if (value === null || Array.isArray(value) || typeof value !== 'object') {
		throw new Error(`${field} must be a JSON object.`);
	}

	const serialized = JSON.stringify(value);
	if (serialized === undefined) {
		throw new Error(`${field} must be a JSON object.`);
	}

	return serialized;
}

export function byteArrayToWails(value: Uint8Array): number[] {
	if (!(value instanceof Uint8Array)) {
		throw new Error('Expected a Uint8Array.');
	}

	return [...value];
}

function attachmentFileRefFromWails(value: unknown, field: string): NonNullable<Attachment['fileRef']> {
	const ref = requireWailsObject(value, field);

	return {
		path: requireWailsString(ref.path, `${field}.path`),
		name: requireWailsString(ref.name, `${field}.name`),
		exists: requireWailsBoolean(ref.exists, `${field}.exists`),
		isDir: requireWailsBoolean(ref.isDir, `${field}.isDir`),
		size: ref.size === undefined ? undefined : requireWailsNumber(ref.size, `${field}.size`),
		modTime: optionalFrontendDate(ref.modTime, `${field}.modTime`),
		origPath: requireWailsString(ref.origPath, `${field}.origPath`),
		origSize: requireWailsNumber(ref.origSize, `${field}.origSize`),
		origModTime: toFrontendDate(ref.origModTime, `${field}.origModTime`),
	};
}

function attachmentImageRefFromWails(value: unknown, field: string): NonNullable<Attachment['imageRef']> {
	const ref = requireWailsObject(value, field);

	return {
		...attachmentFileRefFromWails(ref, field),
		width: ref.width === undefined ? undefined : requireWailsNumber(ref.width, `${field}.width`),
		height: ref.height === undefined ? undefined : requireWailsNumber(ref.height, `${field}.height`),
		format: optionalWailsString(ref.format, `${field}.format`),
		mimeType: optionalWailsString(ref.mimeType, `${field}.mimeType`),
	};
}

export function attachmentFromWails(value: unknown, field: string): Attachment {
	const attachment = requireWailsObject(value, field);
	const contentBlock =
		attachment.contentBlock === undefined
			? undefined
			: (() => {
					const block = requireWailsObject(attachment.contentBlock, `${field}.contentBlock`);
					return {
						...block,
						kind: enumFromWails(block.kind, AttachmentContentBlockKind, `${field}.contentBlock.kind`),
					};
				})();

	return {
		...attachment,
		kind: enumFromWails(attachment.kind, AttachmentKind, `${field}.kind`),
		label: requireWailsString(attachment.label, `${field}.label`),
		mode:
			attachment.mode === undefined
				? undefined
				: enumFromWails(attachment.mode, AttachmentContentBlockMode, `${field}.mode`),
		availableContentBlockModes: optionalWailsArray(
			attachment.availableContentBlockModes,
			`${field}.availableContentBlockModes`
		)?.map((mode, index) =>
			enumFromWails(mode, AttachmentContentBlockMode, `${field}.availableContentBlockModes[${index}]`)
		),
		fileRef:
			attachment.fileRef === undefined ? undefined : attachmentFileRefFromWails(attachment.fileRef, `${field}.fileRef`),
		imageRef:
			attachment.imageRef === undefined
				? undefined
				: attachmentImageRefFromWails(attachment.imageRef, `${field}.imageRef`),
		contentBlock,
	} as Attachment;
}

export function directoryAttachmentsResultFromWails(value: unknown, field: string): DirectoryAttachmentsResult {
	const result = requireWailsObject(value, field);

	return {
		dirPath: requireWailsString(result.dirPath, `${field}.dirPath`),
		attachments: requireWailsArray(result.attachments, `${field}.attachments`).map((attachment, index) =>
			attachmentFromWails(attachment, `${field}.attachments[${index}]`)
		),
		overflowDirs: requireWailsArray(result.overflowDirs, `${field}.overflowDirs`).map((overflow, index) => {
			const item = requireWailsObject(overflow, `${field}.overflowDirs[${index}]`);
			return {
				dirPath: requireWailsString(item.dirPath, `${field}.overflowDirs[${index}].dirPath`),
				relativePath: requireWailsString(item.relativePath, `${field}.overflowDirs[${index}].relativePath`),
				fileCount: requireWailsNumber(item.fileCount, `${field}.overflowDirs[${index}].fileCount`),
				partial: requireWailsBoolean(item.partial, `${field}.overflowDirs[${index}].partial`),
			};
		}),
		maxFiles: requireWailsNumber(result.maxFiles, `${field}.maxFiles`),
		totalSize: requireWailsNumber(result.totalSize, `${field}.totalSize`),
		hasMore: requireWailsBoolean(result.hasMore, `${field}.hasMore`),
	};
}

export function pathAttachmentsResultFromWails(value: unknown, field: string): PathAttachmentsResult {
	const result = requireWailsObject(value, field);

	return {
		fileAttachments: requireWailsArray(result.fileAttachments, `${field}.fileAttachments`).map((attachment, index) =>
			attachmentFromWails(attachment, `${field}.fileAttachments[${index}]`)
		),
		dirAttachments: requireWailsArray(result.dirAttachments, `${field}.dirAttachments`).map((directory, index) =>
			directoryAttachmentsResultFromWails(directory, `${field}.dirAttachments[${index}]`)
		),
		errors: optionalWailsArray(result.errors, `${field}.errors`)?.map((error, index) =>
			requireWailsString(error, `${field}.errors[${index}]`)
		),
	};
}
