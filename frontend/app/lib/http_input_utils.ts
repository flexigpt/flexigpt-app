const HTTP_HEADER_NAME_RE = /^[A-Za-z0-9!#$%&'*+.^_`|~-]+$/;
const HTTP_HEADER_CONTROL_CHAR_RE = /[\r\n\u0000]/;

export const REDACTED_HTTP_VALUE = '[configured]';

function isIPv4LoopbackHost(host: string): boolean {
	const parts = host.split('.').map(Number);

	return (
		parts.length === 4 && parts.every(part => Number.isInteger(part) && part >= 0 && part <= 255) && parts[0] === 127
	);
}

function isLoopbackHTTPHost(host: string): boolean {
	const normalized = host.trim().toLowerCase().replace(/^\[/, '').replace(/\]$/, '');

	if (!normalized) {
		return false;
	}

	return (
		normalized === 'localhost' ||
		normalized === '::1' ||
		normalized === '0:0:0:0:0:0:0:1' ||
		isIPv4LoopbackHost(normalized)
	);
}

/**
 * Applies the transport security policy used by remotely invoked HTTP tools
 * and MCP Streamable HTTP servers.
 *
 * Plain HTTP is intentionally limited to loopback development endpoints.
 * URL userinfo is rejected because it is easy to leak in logs, exports, and
 * view-only configuration surfaces.
 */
export function validateHTTPURLSecurity(raw: string, fieldLabel = 'URL'): string | undefined {
	try {
		const url = new URL(raw);

		if (url.protocol !== 'http:' && url.protocol !== 'https:') {
			return `${fieldLabel} must use http or https.`;
		}

		if (!url.hostname) {
			return `${fieldLabel} must include a host.`;
		}

		if (url.username || url.password) {
			return `${fieldLabel} must not include embedded credentials. Store credentials in the configured secret fields instead.`;
		}

		if (url.protocol === 'http:' && !isLoopbackHTTPHost(url.hostname)) {
			return `${fieldLabel} must use https for remote hosts. Plain HTTP is allowed only for localhost or loopback addresses.`;
		}

		if (url.hash) {
			return `${fieldLabel} must not include a fragment.`;
		}

		return undefined;
	} catch {
		return `${fieldLabel} must be valid.`;
	}
}

/**
 * Applies URL security checks to an HTTP tool URL template while preventing
 * model-controlled substitutions in the scheme or network authority.
 */
export function validateHTTPURLTemplateSecurity(raw: string, fieldLabel = 'URL'): string | undefined {
	const schemeSeparator = raw.indexOf('://');
	if (schemeSeparator >= 0) {
		const authorityStart = schemeSeparator + 3;
		const relativeAuthorityEnd = raw.slice(authorityStart).search(/[/?#]/);
		const authorityEnd = relativeAuthorityEnd < 0 ? raw.length : authorityStart + relativeAuthorityEnd;
		const authority = new Set(raw.slice(authorityStart, authorityEnd));

		if (authority.has('${') || authority.has('{{')) {
			return `${fieldLabel} must not contain substitutions in the host, port, or user-info section.`;
		}
	}

	return validateHTTPURLSecurity(raw, fieldLabel);
}

export function validateHTTPHeaderName(name: string, fieldLabel = 'Header name'): string | undefined {
	const normalized = name.trim();
	if (!normalized) {
		return `${fieldLabel} is required.`;
	}
	return HTTP_HEADER_NAME_RE.test(normalized) ? undefined : `${fieldLabel} contains invalid characters.`;
}

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
	const seenHeaderNames = new Set<string>();

	for (const [name, value] of Object.entries(headers)) {
		const headerNameError = validateHTTPHeaderName(name, `${label} header name "${name}"`);
		if (headerNameError) {
			throw new Error(headerNameError);
		}

		const normalizedName = name.toLowerCase();
		if (seenHeaderNames.has(normalizedName)) {
			throw new Error(`${label} contains duplicate header name: "${name}".`);
		}
		seenHeaderNames.add(normalizedName);

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
		Object.entries(headers).map(([name, value]) => [
			name,
			isSensitiveHTTPHeaderName(name) ? REDACTED_HTTP_VALUE : value,
		])
	);
}

export function omitSensitiveHTTPHeaders(headers?: Record<string, string>): Record<string, string> | undefined {
	if (!headers || Object.keys(headers).length === 0) {
		return undefined;
	}

	const safeHeaders = Object.fromEntries(Object.entries(headers).filter(([name]) => !isSensitiveHTTPHeaderName(name)));

	return Object.keys(safeHeaders).length > 0 ? safeHeaders : undefined;
}

/**
 * Restores write-only values that were redacted for editing.
 *
 * A redaction marker without a matching original header is rejected so the
 * marker can never accidentally be persisted as a real credential.
 */
export function restoreRedactedHTTPHeaders(
	headers: Record<string, string>,
	originalHeaders?: Record<string, string>
): Record<string, string> {
	const originals = new Map(
		Object.entries(originalHeaders ?? {}).map(([name, value]) => [name.trim().toLowerCase(), value] as const)
	);

	return Object.fromEntries(
		Object.entries(headers).map(([name, value]) => {
			if (!isSensitiveHTTPHeaderName(name) || value !== REDACTED_HTTP_VALUE) {
				return [name, value];
			}

			const original = originals.get(name.trim().toLowerCase());
			if (original === undefined) {
				throw new Error(`Enter a value for sensitive header "${name}" instead of using the redaction marker.`);
			}

			return [name, original];
		})
	);
}
