import type { ToolJSONSchema, UIToolUserArgsStatus } from '@/spec/tool';

import { getRequiredFromJSONSchema, isJSONObject } from '@/lib/jsonschema_utils';

export function computeToolUserArgsStatus(
	schema: ToolJSONSchema | undefined,
	rawInstance?: string | null
): UIToolUserArgsStatus {
	const base: UIToolUserArgsStatus = {
		hasSchema: schema !== undefined,
		requiredKeys: [],
		missingRequired: [],
		isInstancePresent: Boolean(rawInstance?.trim()),
		isInstanceJSONValid: false,
		isSatisfied: true,
	};

	if (schema === undefined) {
		return { ...base, hasSchema: false };
	}

	let parsed: unknown;
	if (base.isInstancePresent) {
		try {
			parsed = JSON.parse(rawInstance ?? '');
			base.isInstanceJSONValid = isJSONObject(parsed);
		} catch {
			base.isInstanceJSONValid = false;
		}
	}

	if (schema === false) {
		return { ...base, isSatisfied: false };
	}
	if (schema === true) {
		return {
			...base,
			isSatisfied: !base.isInstancePresent || base.isInstanceJSONValid,
		};
	}

	const required = getRequiredFromJSONSchema(schema) ?? [];
	if (!base.isInstancePresent) {
		return {
			...base,
			requiredKeys: required,
			missingRequired: required,
			isSatisfied: required.length === 0,
		};
	}
	if (!base.isInstanceJSONValid || !isJSONObject(parsed)) {
		return {
			...base,
			requiredKeys: required,
			missingRequired: required,
			isSatisfied: false,
		};
	}

	const missing = required.filter(key => {
		const value = parsed[key];
		return value === undefined || value === null || (typeof value === 'string' && value.trim() === '');
	});

	return {
		...base,
		requiredKeys: required,
		missingRequired: missing,
		isSatisfied: missing.length === 0,
	};
}
