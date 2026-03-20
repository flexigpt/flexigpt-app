import type { UIToolUserArgsStatus } from '@/spec/tool';

import { getRequiredFromJSONSchema, isJSONObject, type JSONSchema } from '@/lib/jsonschema_utils';

/**
 * Inspect a tool's userArgSchema and a JSON-encoded instance string and
 * compute whether all required keys are populated.
 *
 * We intentionally treat:
 *  - no schema            => satisfied (no args required)
 *  - schema with no "required" keys => satisfied
 *  - invalid / non-object instance  => all required missing
 */
export function computeToolUserArgsStatus(
	schema: JSONSchema | undefined,
	rawInstance?: string | null
): UIToolUserArgsStatus {
	const base: UIToolUserArgsStatus = {
		hasSchema: false,
		requiredKeys: [],
		missingRequired: [],
		isInstancePresent: false,
		isInstanceJSONValid: false,
		isSatisfied: true,
	};

	if (!schema || !isJSONObject(schema)) {
		// No schema at all -> nothing to validate.
		return base;
	}

	const required = getRequiredFromJSONSchema(schema) ?? [];

	const status: UIToolUserArgsStatus = {
		...base,
		hasSchema: true,
		requiredKeys: required,
		isSatisfied: true,
	};

	if (required.length === 0) {
		// Schema exists but does not require anything -> always satisfied.
		return status;
	}

	// From here on, there ARE required keys.
	if (!rawInstance || rawInstance.trim() === '') {
		return {
			...status,
			isInstancePresent: false,
			isInstanceJSONValid: false,
			missingRequired: required,
			isSatisfied: false,
		};
	}

	status.isInstancePresent = true;

	let parsed: unknown;
	try {
		parsed = JSON.parse(rawInstance);
	} catch {
		return {
			...status,
			isInstanceJSONValid: false,
			missingRequired: required,
			isSatisfied: false,
		};
	}

	if (!parsed || typeof parsed !== 'object') {
		return {
			...status,
			isInstanceJSONValid: false,
			missingRequired: required,
			isSatisfied: false,
		};
	}

	status.isInstanceJSONValid = true;

	const obj = parsed as Record<string, unknown>;
	const missing: string[] = [];

	for (const key of required) {
		const v = obj[key];
		if (v === undefined || v === null) {
			missing.push(key);
			continue;
		}
		if (typeof v === 'string' && v.trim() === '') {
			missing.push(key);
			continue;
		}
	}

	status.missingRequired = missing;
	status.isSatisfied = missing.length === 0;
	return status;
}
