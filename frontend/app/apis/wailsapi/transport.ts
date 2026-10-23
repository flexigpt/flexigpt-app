// oxlint-disable typescript/no-unnecessary-type-parameters
import type { JSONObject, JSONRawString } from '@/lib/jsonschema_utils';

const DEFAULT_MAX_PAGES = 1_000;

function isByte(value: unknown): value is number {
	return typeof value === 'number' && Number.isInteger(value) && value >= 0 && value <= 255;
}

const WAILS_ENUM_ZERO_VALUE = '';
const WAILS_ENUM_ZERO_MEMBER = '$zero';

function wailsEnumZeroError(field: string): Error {
	return new Error(
		`Invalid ${field}: Go enum zero value is not allowed. ` +
			`Wails generates this value as ${JSON.stringify(WAILS_ENUM_ZERO_MEMBER)}.`
	);
}

/**
 * Wails v3 models a Go []byte as a base64 string. Frontend contracts retain
 * byte arrays so application code remains independent of Wails transport
 * details.
 */
export function goBytesToWails(value: readonly number[], field: string): string {
	const bytes = new Uint8Array(value.length);

	for (let index = 0; index < value.length; index += 1) {
		const byte = value[index];

		if (!isByte(byte)) {
			throw new TypeError(`${field}[${index}] must be a byte.`);
		}

		bytes[index] = byte;
	}

	if (typeof globalThis.btoa !== 'function') {
		throw new TypeError(`Base64 encoding is unavailable while serializing ${field}.`);
	}

	let binary = '';
	const chunkSize = 0x8000;

	for (let offset = 0; offset < bytes.length; offset += chunkSize) {
		binary += String.fromCharCode(...bytes.subarray(offset, offset + chunkSize));
	}

	return globalThis.btoa(binary);
}

/**
 * Accepts the generated v3 base64 representation of Go []byte values.
 * Numeric arrays are accepted as a compatibility fallback for persisted or
 * mocked legacy transport fixtures.
 */
export function goBytesFromWails(value: unknown, field: string): number[] | undefined {
	if (value === null || value === undefined) {
		return undefined;
	}

	if (value instanceof Uint8Array) {
		return [...value];
	}

	if (Array.isArray(value)) {
		return value.map((item, index) => {
			if (!isByte(item)) {
				throw new TypeError(`${field}[${index}] must be a byte.`);
			}

			return item;
		});
	}

	if (typeof value !== 'string' || typeof globalThis.atob !== 'function') {
		throw new TypeError(`${field} returned an invalid Go []byte value.`);
	}

	try {
		return Array.from(globalThis.atob(value), character => character.charCodeAt(0));
	} catch {
		throw new TypeError(`${field} returned invalid base64 data.`);
	}
}

/**
 * Wails transports Go string aliases as plain JavaScript strings. Validate
 * values at the bridge boundary before exposing frontend enum types.
 */
export function enumFromWails<T extends string>(
	value: unknown,
	enumValues: Readonly<Record<string, T>>,
	field: string
): T {
	if (value === WAILS_ENUM_ZERO_VALUE || value === WAILS_ENUM_ZERO_MEMBER) {
		throw wailsEnumZeroError(field);
	}

	const allowed = Object.values(enumValues).filter(candidate => candidate !== WAILS_ENUM_ZERO_VALUE) as T[];

	if (typeof value !== 'string' || !allowed.includes(value as T)) {
		throw new Error(`Invalid ${field}: ${String(value)}`);
	}

	return value as T;
}

function checkedWailsString<TWails extends string>(value: string): TWails {
	return value as TWails;
}

/**
 * Validates a frontend enum before it is sent through a Wails binding.
 *
 * `TWails` exists only because independently declared TypeScript enums are
 * nominally incompatible even when their serialized string values match.
 * The cast is safe because `enumFromWails` verifies the exact wire value
 * before it is exposed as the generated binding enum type.
 */
export function enumToWails<TFrontend extends string, TWails extends string = TFrontend>(
	value: unknown,
	enumValues: Readonly<Record<string, TFrontend>>,
	field: string
): TWails {
	const parsed = enumFromWails(value, enumValues, field);

	return checkedWailsString<TWails>(parsed);
}

export function optionalEnumFromWails<T extends string>(
	value: unknown,
	enumValues: Readonly<Record<string, T>>,
	field: string
): T | undefined {
	if (value === null || value === undefined) {
		return undefined;
	}

	return enumFromWails(value, enumValues, field);
}

export function stringArrayFromWails(value: unknown, field: string): string[] {
	return wailsArrayOrEmpty(value, field).map((item, index) => requireWailsString(item, `${field}[${index}]`));
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

export function nonBlankStringToWails<TWails extends string>(value: unknown, field: string): TWails {
	return checkedWailsString<TWails>(requireNonBlankString(value, field));
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

export function requiredWailsResponseBody<T extends object>(response: unknown, operation: string): T {
	const envelope = requiredObject<{ Body?: unknown }>(response, `${operation} response`);

	return requiredObject<T>(envelope.Body, operation);
}

export function requiredObject<T extends object>(value: unknown, operation: string): T {
	return requireWailsBody(value as T | null | undefined, operation);
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

/**
 * Supports both Wails representations seen for `json.RawMessage`:
 *
 * - Raw JSON object
 * - Generated `number[]` byte representation
 */
function rawJSONFromWails(value: unknown, field: string): JSONRawString {
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

export function jsonSchemaFromWails(value: unknown, field: string): JSONObject | boolean {
	const parsed: unknown = typeof value === 'boolean' ? value : JSON.parse(rawJSONFromWails(value, field));

	if (typeof parsed === 'boolean') {
		return parsed;
	}
	if (parsed === null || Array.isArray(parsed) || typeof parsed !== 'object') {
		throw new Error(`${field} must be a JSON object or boolean schema.`);
	}
	return parsed as JSONObject;
}
