const HTTP_HEADER_NAME_RE = /^[A-Za-z0-9!#$%&'*+.^_`|~-]+$/;
// oxlint-disable-next-line no-control-regex
const HTTP_HEADER_CONTROL_CHAR_RE = /[\r\n\u0000]/;

const SENSITIVE_HEADER_NAMES = new Set([
	'authorization',
	'proxy-authorization',
	'cookie',
	'set-cookie',
	'x-api-key',
	'api-key',
	'x-auth-token',
]);

function parseJSONObject(raw: string, label: string): Record<string, unknown> {
	const value = raw.trim();
	if (!value) {
		return {};
	}

	let parsed: unknown;
	try {
		parsed = JSON.parse(value);
	} catch {
		throw new Error(`${label} must be valid JSON.`);
	}

	if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
		throw new Error(`${label} must be a JSON object.`);
	}

	return parsed as Record<string, unknown>;
}

export function parseStringRecordJSON(raw: string, label: string): Record<string, string> {
	const parsed = parseJSONObject(raw, label);
	const result: Record<string, string> = {};

	for (const [key, value] of Object.entries(parsed)) {
		const normalizedKey = key.trim();
		if (!normalizedKey) {
			throw new Error(`${label} keys must not be empty.`);
		}
		if (typeof value !== 'string') {
			throw new TypeError(`${label} value for "${key}" must be a string.`);
		}

		result[normalizedKey] = value;
	}

	return result;
}

export function parseHTTPHeadersJSON(raw: string, label = 'Headers'): Record<string, string> {
	const headers = parseStringRecordJSON(raw, label);

	for (const [name, value] of Object.entries(headers)) {
		if (!HTTP_HEADER_NAME_RE.test(name)) {
			throw new Error(`${label} contains an invalid header name: "${name}".`);
		}
		if (HTTP_HEADER_CONTROL_CHAR_RE.test(value)) {
			throw new Error(`${label} value for "${name}" must not contain CR, LF, or NUL.`);
		}
	}

	return headers;
}

export function parseHTTPStatusCodes(raw: string): number[] | undefined {
	const value = raw.trim();
	if (!value) {
		return undefined;
	}

	const codes = value
		.split(',')
		.map(part => part.trim())
		.filter(Boolean)
		.map(part => {
			if (!/^\d{3}$/.test(part)) {
				throw new Error('Success codes must be comma-separated HTTP status codes.');
			}

			const code = Number(part);
			if (!Number.isInteger(code) || code < 100 || code > 599) {
				throw new Error('HTTP success codes must be between 100 and 599.');
			}
			return code;
		});

	return [...new Set(codes)];
}

function isSensitiveHTTPHeaderName(name: string): boolean {
	const normalized = name.trim().toLowerCase();
	if (SENSITIVE_HEADER_NAMES.has(normalized)) {
		return true;
	}

	return /(?:^|[-_])(api[-_]?key|auth|authorization|cookie|secret|token)(?:$|[-_])/i.test(normalized);
}

export function redactSensitiveHTTPHeaders(headers?: Record<string, string>): Record<string, string> | undefined {
	if (!headers || Object.keys(headers).length === 0) {
		return undefined;
	}

	return Object.fromEntries(
		Object.entries(headers).map(([name, value]) => [name, isSensitiveHTTPHeaderName(name) ? '[configured]' : value])
	);
}
