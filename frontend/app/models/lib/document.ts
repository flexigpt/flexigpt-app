import type { ArtifactRef } from '@/spec/artifact';
import type { ModelAuthentication, ModelConnection } from '@/spec/model';
import { ModelAuthenticationMode } from '@/spec/model';

import {
	containsHTTPControlCharacters,
	normalizeHTTPOrigin,
	parseHTTPHeaderPatch,
	validateHTTPHeaderName,
} from '@/lib/http_input_utils';
import { parseOptionalJSONObject } from '@/lib/jsonschema_utils';

export function modelRefEqual(left: ArtifactRef | undefined, right: ArtifactRef | undefined): boolean {
	return left?.rootID === right?.rootID && left?.artifactID === right?.artifactID;
}

export function parseProviderConnection(value: string): ModelConnection {
	const connection = parseOptionalJSONObject<ModelConnection>(value, 'Connection');
	if (!connection || typeof connection.origin !== 'string') {
		throw new Error('Connection must include an origin.');
	}

	const { normalized, error } = normalizeHTTPOrigin(connection.origin, 'Connection origin');
	if (!normalized || error) {
		throw new Error(error ?? 'Connection origin must be valid.');
	}

	const rawPath = connection.path;
	if (rawPath !== undefined && typeof rawPath !== 'string') {
		throw new Error('Connection path must be a string.');
	}

	const path = rawPath?.trim();
	if (path && (!path.startsWith('/') || containsHTTPControlCharacters(path))) {
		throw new Error('Connection path must start with "/" and must not contain control characters.');
	}

	const headers =
		connection.headers === undefined
			? undefined
			: parseHTTPHeaderPatch(connection.headers, 'Connection headers', { rejectSensitive: true });

	return {
		...connection,
		origin: normalized,
		...(path ? { path } : {}),
		...(headers ? { headers } : {}),
	};
}

export function parseProviderAuthentication(value: string): ModelAuthentication {
	const authentication = parseOptionalJSONObject<ModelAuthentication>(value, 'Authentication');
	if (!authentication || typeof authentication.mode !== 'string') {
		throw new Error('Authentication must include a mode.');
	}

	if (!Object.values(ModelAuthenticationMode).includes(authentication.mode)) {
		throw new Error('Authentication mode is invalid.');
	}

	const headerName = authentication.headerName?.trim();
	if (headerName) {
		const headerError = validateHTTPHeaderName(headerName, 'Authentication header name');
		if (headerError) {
			throw new Error(headerError);
		}
	}

	if (authentication.mode === ModelAuthenticationMode.APIKeyHeader && !headerName) {
		throw new Error('API-key header authentication requires a headerName.');
	}

	if (authentication.prefix !== undefined && containsHTTPControlCharacters(authentication.prefix)) {
		throw new Error('Authentication prefix must not contain CR, LF, or NUL.');
	}

	return {
		...authentication,
		...(headerName ? { headerName } : {}),
	};
}
