import type { JSONRawString, JSONSchema } from '@/lib/jsonschema_utils';

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

export function wailsArrayOrEmpty<T = unknown>(value: unknown, field: string): T[] {
	if (value === null || value === undefined) {
		return [];
	}
	if (!Array.isArray(value)) {
		throw new TypeError(`${field} returned an invalid array.`);
	}

	return value as T[];
}

export function requireWailsString(value: unknown, field: string): string {
	if (typeof value !== 'string') {
		throw new TypeError(`${field} returned an invalid string.`);
	}

	return value;
}

export function optionalWailsString(value: unknown, field: string): string | undefined {
	if (value === null || value === undefined) {
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

export function requireWailsFiniteNumber(value: unknown, field: string): number {
	if (typeof value !== 'number' || !Number.isFinite(value)) {
		throw new TypeError(`${field} returned an invalid number.`);
	}

	return value;
}

export function requireNonBlankString(value: unknown, field: string): string {
	if (typeof value !== 'string') {
		throw new TypeError(`${field} must be a string.`);
	}

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
	if (!Number.isInteger(maxPages) || maxPages <= 0) {
		throw new RangeError('maxPages must be a positive integer.');
	}

	const items: T[] = [];
	const seenTokens = new Set<string>();
	let pageToken: string | undefined;

	for (let page = 0; page < maxPages; page++) {
		const result = requireWailsBody(await fetchPage(pageToken), `Pagination page ${page + 1}`);
		const pageItems = wailsArrayOrEmpty<T>(result.items, `Pagination page ${page + 1}.items`);
		const nextPageToken = optionalWailsString(result.nextPageToken, `Pagination page ${page + 1}.nextPageToken`);

		items.push(...pageItems);

		if (!nextPageToken) {
			return items;
		}

		if (seenTokens.has(nextPageToken)) {
			throw new Error('Pagination response repeated a page token.');
		}

		seenTokens.add(nextPageToken);
		pageToken = nextPageToken;
	}

	throw new Error(`Pagination exceeded the ${maxPages}-page safety limit.`);
}

export function requireWailsBody<T>(body: T | null | undefined, operation: string): T {
	if (body === null || body === undefined) {
		throw new Error(`${operation} returned an empty response body.`);
	}

	if (typeof body !== 'object' || Array.isArray(body)) {
		throw new TypeError(`${operation} returned an invalid response body.`);
	}

	return body;
}

export function optionalWailsBody<T>(body: T | null | undefined, operation = 'Wails operation'): T | undefined {
	if (body === null || body === undefined) {
		return undefined;
	}

	if (typeof body !== 'object' || Array.isArray(body)) {
		throw new TypeError(`${operation} returned an invalid response body.`);
	}

	return body;
}

export function wailsObjectArrayOrEmpty<T extends object = Record<string, unknown>>(
	value: unknown,
	field: string
): T[] {
	return wailsArrayOrEmpty(value, field).map((item, index) => {
		const object = requireWailsBody(item as Record<string, unknown> | null | undefined, `${field}[${index}]`);
		return object as T;
	});
}

export function wailsRecordOrEmpty<T = unknown>(value: unknown, field: string): Record<string, T> {
	if (value === null || value === undefined) {
		return {};
	}

	return requireWailsBody(value as Record<string, T> | null | undefined, field);
}

export function createAbortError(): Error {
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
