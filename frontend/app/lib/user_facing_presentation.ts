const INTERNAL_PRESENTATION_FIELDS = new Set([
	'id',
	'choiceid',
	'selectionid',
	'callid',
	'providercallid',
	'rootid',
	'artifactid',
	'definitiondigest',
	'digest',
	'artifactrevision',
	'expectedrevision',
	'expecteddigest',
	'provideridentity',
	'providerlocalid',
	'runtimeserverid',
	'appinstanceid',
	'viewid',
	'sessionid',
	'runid',
	'operationid',
	'originoperationid',
	'selectedresultid',
	'attemptid',
	'waitid',
	'consumedbyoperationid',
	'resourceuri',
	'appresourceuri',
	'sessionapp',
	'structuredcontent',
]);

function normalizedLabel(value: string): string {
	return value.replaceAll(/\s+/g, ' ').trim().toLocaleLowerCase();
}

function isRecord(value: unknown): value is Record<string, unknown> {
	return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function isInternalPresentationField(key: string): boolean {
	return INTERNAL_PRESENTATION_FIELDS.has(key.replaceAll(/[_-]/g, '').toLocaleLowerCase());
}

/**
 * Prefer display names, then logical names, and never return an empty label.
 */
export function getPrimaryUserFacingLabel(...candidates: Array<string | null | undefined>): string {
	for (const candidate of candidates) {
		const trimmed = candidate?.trim();
		if (trimmed) {
			return trimmed;
		}
	}

	return 'Unnamed item';
}

/**
 * Return a secondary label only when it adds information instead of repeating
 * the visible display name.
 */
export function getDistinctUserFacingLabel(primary: string, candidate: string | null | undefined): string | undefined {
	const trimmed = candidate?.trim();
	if (!trimmed || normalizedLabel(primary) === normalizedLabel(trimmed)) {
		return undefined;
	}

	return trimmed;
}

function redact(value: unknown, seen: WeakSet<object>, depth: number): unknown {
	if (depth > 16) {
		return '[Nested value omitted]';
	}

	if (typeof value === 'string') {
		try {
			return redact(JSON.parse(value) as unknown, seen, depth + 1);
		} catch {
			return value;
		}
	}

	if (Array.isArray(value)) {
		return value.map(item => redact(item, seen, depth + 1));
	}

	if (!isRecord(value)) {
		return value;
	}

	if (seen.has(value)) {
		return '[Circular]';
	}
	seen.add(value);

	const output: Record<string, unknown> = {};
	for (const [key, item] of Object.entries(value)) {
		if (isInternalPresentationField(key)) {
			continue;
		}
		output[key] = redact(item, seen, depth + 1);
	}

	return output;
}

/**
 * Keeps user-provided arguments and results visible while removing known
 * runtime/session identifiers from generic detail views.
 */
export function redactInternalPresentationFields(value: unknown): unknown {
	return redact(value, new WeakSet<object>(), 0);
}
