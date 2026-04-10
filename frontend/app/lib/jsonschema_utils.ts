/* eslint-disable @typescript-eslint/no-unnecessary-type-assertion */

/**
 * @public
 */
export type JSONPrimitive = string | number | boolean | null;

export interface JSONObject {
	[key: string]: JSONValue | undefined;
}

export type JSONValue = JSONPrimitive | JSONObject | JSONValue[];

export type JSONSchema = JSONValue;
export type JSONRawString = string;

export function isJSONObject(value: unknown): value is JSONObject {
	return typeof value === 'object' && value !== null && !Array.isArray(value);
}

export function getJSONObject(value: unknown): JSONObject | undefined {
	return isJSONObject(value) ? value : undefined;
}

/**
 * @public
 */
export function isStringArray(value: unknown): value is string[] {
	if (!Array.isArray(value)) return false;
	return value.every((item: unknown): item is string => typeof item === 'string');
}

export function getRequiredFromJSONSchema(schema: unknown): string[] | undefined {
	const obj = getJSONObject(schema);
	if (!obj) return undefined;

	const candidate = obj['required'];
	return isStringArray(candidate) ? candidate : undefined;
}

export function getPropertiesFromJSONSchema(schema: unknown): JSONObject | undefined {
	const obj = getJSONObject(schema);
	if (!obj) return undefined;

	const props = obj['properties'];
	return getJSONObject(props);
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
	return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function mergeExamples(a: JSONValue, b: JSONValue): JSONValue {
	if (isPlainObject(a) && isPlainObject(b)) {
		const out: Record<string, JSONValue> = { ...(a as Record<string, JSONValue>) };

		const entries = Object.entries(b as Record<string, JSONValue>) as [string, JSONValue][];
		for (const [key, value] of entries) {
			const existing = out[key];

			out[key] = existing === undefined ? value : mergeExamples(existing, value);
		}
		return out as JSONObject;
	}

	return b !== undefined ? b : a;
}

function pickType(schemaObj: JSONObject): string | undefined {
	const t = schemaObj['type'];

	if (Array.isArray(t)) {
		const arr = t as unknown[];
		const nonNull = arr.find(x => x !== 'null');
		if (typeof nonNull === 'string') return nonNull;
		const first = arr[0];
		return typeof first === 'string' ? first : undefined;
	}

	return typeof t === 'string' ? t : undefined;
}

function exampleForStringFormat(format: string): string {
	switch (format) {
		case 'date-time':
			return '1970-01-01T00:00:00.000Z';
		case 'date':
			return '1970-01-01';
		case 'time':
			return '00:00:00Z';
		case 'email':
			return 'user@example.com';
		case 'hostname':
			return 'example.com';
		case 'uri':
		case 'url':
			return 'https://example.com';
		case 'uuid':
			return '00000000-0000-0000-0000-000000000000';
		case 'ipv4':
			return '192.0.2.1';
		case 'ipv6':
			return '2001:db8::1';
		default:
			return '';
	}
}

function scoreExample(ex: unknown): number {
	if (ex === null || ex === undefined) return 0;
	const t = typeof ex;
	if (t === 'string' || t === 'number' || t === 'boolean') return 1;

	if (Array.isArray(ex)) {
		let score = 2;
		for (const v of ex as unknown[]) {
			score += scoreExample(v);
		}
		return score;
	}

	if (isPlainObject(ex)) {
		const values = Object.values(ex) as unknown[];
		let score = 3 + values.length;
		for (const v of values) {
			score += scoreExample(v);
		}
		return score;
	}

	return 0;
}

function pickBestOf<T>(options: T[], builder: (s: T) => JSONValue): JSONValue {
	let best = builder(options[0]);
	let bestScore = scoreExample(best);

	for (let i = 1; i < options.length; i++) {
		const ex = builder(options[i]);
		const sc = scoreExample(ex);
		if (sc > bestScore) {
			best = ex;
			bestScore = sc;
		}
	}

	return best;
}

function buildNumberExample(schemaObj: JSONObject, integer: boolean): number {
	const minimum = schemaObj['minimum'];
	const exclusiveMinimum = schemaObj['exclusiveMinimum'];
	const maximum = schemaObj['maximum'];
	const exclusiveMaximum = schemaObj['exclusiveMaximum'];
	const multipleOf = schemaObj['multipleOf'];

	let n = 0;

	if (typeof minimum === 'number') {
		n = minimum;
	} else if (typeof exclusiveMinimum === 'number') {
		n = exclusiveMinimum + 1;
	}

	if (typeof multipleOf === 'number' && multipleOf > 0) {
		const m = multipleOf;
		n = Math.ceil(n / m) * m;
	}

	if (typeof maximum === 'number' && n > maximum) n = maximum;
	if (typeof exclusiveMaximum === 'number' && n >= exclusiveMaximum) n = exclusiveMaximum - 1;

	if (integer) n = Math.trunc(n);
	if (!Number.isFinite(n)) n = 0;

	return n;
}

function buildStringExample(schemaObj: JSONObject): string {
	const format = schemaObj['format'];
	if (typeof format === 'string') {
		const formatted = exampleForStringFormat(format);
		if (formatted !== '') return formatted;
	}

	const minLength = schemaObj['minLength'];
	if (typeof minLength === 'number' && minLength > 0) {
		const n = Math.min(Math.max(1, Math.floor(minLength)), 64);
		return 'a'.repeat(n);
	}

	// More helpful than empty string as a generic fallback
	return 'string';
}

/**
 * Build an example instance for a Draft-07 JSON Schema.
 * - Respects: const, default, examples, enum
 * - Supports: object (properties/patternProperties/additionalProperties), array (items/tuple), allOf/anyOf/oneOf
 * - Recurses into nested objects/arrays
 * - Includes ALL keys from `properties` (required + optional)
 * - Tries to pick the "most comprehensive" branch for anyOf/oneOf
 */
export function buildExampleFromDraft7Schema(
	schemaLike: unknown,
	depth = 0,
	path: Set<JSONObject> = new Set<JSONObject>()
): JSONValue {
	// Draft-07 boolean schemas:
	if (schemaLike === true) return {} as JSONObject;
	if (schemaLike === false) return null;

	const schemaObj = getJSONObject(schemaLike);
	if (!schemaObj) return null;

	// Prevent runaway recursion on pathological schemas
	if (depth > 10) return null;

	// Path-based cycle protection (allows re-use in siblings without getting stuck)
	if (path.has(schemaObj)) return null;
	path.add(schemaObj);

	try {
		// Highest priority: const/default/examples/enum
		if (Object.prototype.hasOwnProperty.call(schemaObj, 'const')) {
			return schemaObj['const'] as JSONValue;
		}

		if (Object.prototype.hasOwnProperty.call(schemaObj, 'default')) {
			return schemaObj['default'] as JSONValue;
		}

		const examplesVal = schemaObj['examples'];
		if (Array.isArray(examplesVal) && examplesVal.length > 0) {
			const examples = examplesVal as JSONValue[];
			return examples[0];
		}

		const enumVal = schemaObj['enum'];
		if (Array.isArray(enumVal) && enumVal.length > 0) {
			const enums = enumVal as JSONValue[];
			return enums[0];
		}

		// if/then/else (Draft-07): prefer "then" branch example merged with base when possible
		const hasIf = Object.prototype.hasOwnProperty.call(schemaObj, 'if');
		const hasThen = Object.prototype.hasOwnProperty.call(schemaObj, 'then');
		const hasElse = Object.prototype.hasOwnProperty.call(schemaObj, 'else');

		if (hasIf && (hasThen || hasElse)) {
			const baseSchema: JSONObject = { ...schemaObj };
			delete (baseSchema as Record<string, unknown>)['if'];
			delete (baseSchema as Record<string, unknown>)['then'];
			delete (baseSchema as Record<string, unknown>)['else'];

			const base = buildExampleFromDraft7Schema(baseSchema, depth + 1, path);
			const branchSchema = hasThen ? schemaObj['then'] : schemaObj['else'];
			const branch = buildExampleFromDraft7Schema(branchSchema as unknown, depth + 1, path);
			return mergeExamples(base, branch);
		}

		// Schema composition keywords
		const allOfVal = schemaObj['allOf'];
		if (Array.isArray(allOfVal) && allOfVal.length > 0) {
			const schemas = allOfVal as unknown[];
			let merged: JSONValue | undefined;
			for (const sub of schemas) {
				const ex = buildExampleFromDraft7Schema(sub, depth + 1, path);
				merged = merged === undefined ? ex : mergeExamples(merged, ex);
			}
			return merged === undefined ? null : merged;
		}

		const anyOfVal = schemaObj['anyOf'];
		if (Array.isArray(anyOfVal) && anyOfVal.length > 0) {
			const schemas = anyOfVal as unknown[];
			return pickBestOf(schemas, s => buildExampleFromDraft7Schema(s, depth + 1, path));
		}

		const oneOfVal = schemaObj['oneOf'];
		if (Array.isArray(oneOfVal) && oneOfVal.length > 0) {
			const schemas = oneOfVal as unknown[];
			return pickBestOf(schemas, s => buildExampleFromDraft7Schema(s, depth + 1, path));
		}

		// If we can't resolve $ref here, return a conservative placeholder
		if (typeof schemaObj['$ref'] === 'string') {
			return {} as JSONObject;
		}

		const typeStr = pickType(schemaObj);

		// Heuristics when "type" is omitted
		const hasProps = isPlainObject(schemaObj['properties']) || isPlainObject(schemaObj['patternProperties']);
		const hasItems = schemaObj['items'] !== undefined;

		let effectiveType: string | undefined = typeStr;
		if (!effectiveType && hasProps) {
			effectiveType = 'object';
		} else if (!effectiveType && hasItems) {
			effectiveType = 'array';
		}

		switch (effectiveType) {
			case 'object': {
				const out: JSONObject = {};

				const propsObj = getJSONObject(schemaObj['properties']);
				if (propsObj) {
					const entries = Object.entries(propsObj) as [string, unknown][];
					for (const [key, propSchema] of entries) {
						out[key] = buildExampleFromDraft7Schema(propSchema, depth + 1, path);
					}
				}

				// Ensure required keys exist even if not declared in properties
				const required = getRequiredFromJSONSchema(schemaObj) ?? [];
				for (const key of required) {
					if (out[key] === undefined) {
						out[key] = null;
					}
				}

				// patternProperties: add one example key per pattern (best-effort)
				const patternProps = getJSONObject(schemaObj['patternProperties']);
				if (patternProps) {
					let i = 1;
					const entries = Object.entries(patternProps) as [string, unknown][];
					for (const [, patternSchema] of entries) {
						const key = `patternProp${i++}`;
						if (out[key] === undefined) {
							out[key] = buildExampleFromDraft7Schema(patternSchema, depth + 1, path);
						}
					}
				}

				// additionalProperties: if it's a schema object, add a representative additionalProp
				const additionalProps = schemaObj['additionalProperties'];
				const additionalSchema = getJSONObject(additionalProps);
				if (additionalSchema) {
					const k = 'additionalProp1';
					if (out[k] === undefined) {
						out[k] = buildExampleFromDraft7Schema(additionalSchema, depth + 1, path);
					}
				}

				// minProperties: if caller wants more keys, pad with additionalPropN nulls
				const minProperties = schemaObj['minProperties'];
				if (typeof minProperties === 'number' && minProperties > 0) {
					let idx = 1;
					while (Object.keys(out).length < Math.min(minProperties, 25)) {
						const k = `additionalProp${idx++}`;
						if (out[k] === undefined) {
							out[k] = null;
						}
					}
				}

				return out;
			}

			case 'array': {
				const itemsVal = schemaObj['items'];

				// Tuple form: items is an array of schemas
				if (Array.isArray(itemsVal)) {
					const tupleSchemas = itemsVal as unknown[];
					const tupleExamples: JSONValue[] = [];
					for (const sub of tupleSchemas) {
						tupleExamples.push(buildExampleFromDraft7Schema(sub, depth + 1, path));
					}

					const minItemsVal = schemaObj['minItems'];
					const minItems = typeof minItemsVal === 'number' ? minItemsVal : tupleExamples.length;
					const count = Math.min(Math.max(minItems, tupleExamples.length), 10);

					const out: JSONValue[] = [];
					const maxInitial = Math.min(tupleExamples.length, count);
					for (let i = 0; i < maxInitial; i++) {
						out.push(tupleExamples[i]);
					}

					while (out.length < count) {
						out.push(tupleExamples.length > 0 ? tupleExamples[tupleExamples.length - 1] : null);
					}

					return out;
				}

				// Standard form: items is a schema
				const itemSchema = itemsVal as unknown;
				const itemExample = itemSchema !== undefined ? buildExampleFromDraft7Schema(itemSchema, depth + 1, path) : null;

				const minItemsVal = schemaObj['minItems'];
				const maxItemsVal = schemaObj['maxItems'];

				const minItems = typeof minItemsVal === 'number' ? minItemsVal : 0;
				const maxItems = typeof maxItemsVal === 'number' ? maxItemsVal : undefined;

				let count = Math.min(Math.max(minItems, 1), 3); // cap for readability
				if (typeof maxItems === 'number') {
					count = Math.min(count, Math.max(0, Math.floor(maxItems)));
				}

				// uniqueItems: attempt to make primitive items unique
				const uniqueItems = schemaObj['uniqueItems'] === true;
				if (uniqueItems && (typeof itemExample === 'string' || typeof itemExample === 'number')) {
					const out: JSONValue[] = [];
					for (let i = 0; i < count; i++) {
						if (typeof itemExample === 'string') {
							out.push(`${itemExample}${i + 1}`);
						} else {
							out.push(itemExample + i);
						}
					}
					return out;
				}

				const out: JSONValue[] = [];
				for (let i = 0; i < count; i++) {
					out.push(itemExample);
				}
				return out;
			}

			case 'string':
				return buildStringExample(schemaObj);

			case 'number':
				return buildNumberExample(schemaObj, false);

			case 'integer':
				return buildNumberExample(schemaObj, true);

			case 'boolean':
				return false;

			case 'null':
				return null;

			default:
				// If type is unknown/missing, try to return a sensible placeholder
				return hasProps ? ({} as JSONObject) : null;
		}
	} finally {
		path.delete(schemaObj);
	}
}

function buildTextCodeBlock(text: string): string {
	return ['```text', text, '```'].join('\n');
}

export function buildJSONCodeBlock(value: unknown): string {
	try {
		return ['```json', JSON.stringify(value, null, 2), '```'].join('\n');
	} catch {
		return buildTextCodeBlock(String(value));
	}
}

function parseStructuredJSONString(raw?: string): unknown {
	if (typeof raw !== 'string') return undefined;

	const trimmed = raw.trim();
	if (!trimmed) return undefined;

	try {
		const parsed = JSON.parse(trimmed);
		if (parsed !== null && typeof parsed === 'object') {
			return parsed;
		}
	} catch {
		// ignore
	}

	return undefined;
}

export function buildJSONOrTextCodeBlock(raw?: unknown): string | null {
	if (raw === undefined || raw === null) {
		return null;
	}

	if (typeof raw === 'string') {
		if (raw.trim().length === 0) {
			return null;
		}

		const parsed = parseStructuredJSONString(raw);
		if (parsed !== undefined) {
			return buildJSONCodeBlock(parsed);
		}

		return buildTextCodeBlock(raw);
	}

	return buildJSONCodeBlock(normalizeStructuredJSONStringDeep(raw));
}

function isRecord(value: unknown): value is Record<string, unknown> {
	return value !== null && typeof value === 'object' && !Array.isArray(value);
}

export function normalizeStructuredJSONStringDeep(value: unknown): unknown {
	const parsed = typeof value === 'string' ? parseStructuredJSONString(value) : undefined;
	if (parsed !== undefined) {
		return normalizeStructuredJSONStringDeep(parsed);
	}

	if (value instanceof Date) {
		return value.toISOString();
	}

	if (value instanceof Error) {
		const ownProps = Object.fromEntries(
			Object.entries(value).map(([key, item]) => [key, normalizeStructuredJSONStringDeep(item)])
		);
		return {
			name: value.name,
			message: value.message,
			...(value.stack ? { stack: value.stack } : {}),
			...ownProps,
		};
	}

	if (value instanceof Map) {
		return Object.fromEntries(
			Array.from(value.entries()).map(([key, item]) => [String(key), normalizeStructuredJSONStringDeep(item)])
		);
	}

	if (value instanceof Set) {
		return Array.from(value.values()).map(item => normalizeStructuredJSONStringDeep(item));
	}

	if (Array.isArray(value)) {
		return value.map(item => normalizeStructuredJSONStringDeep(item));
	}
	if (isRecord(value)) {
		return Object.fromEntries(
			Object.entries(value).map(([key, item]) => [key, normalizeStructuredJSONStringDeep(item)])
		);
	}
	return value;
}

// eslint-disable-next-line @typescript-eslint/no-unnecessary-type-parameters
export function normalizeStructuredDisplayObject<T extends object>(value: T): Record<string, unknown> {
	const normalized = normalizeStructuredJSONStringDeep(value);
	return isRecord(normalized) ? normalized : (value as Record<string, unknown>);
}

export function cloneJSONLike<T>(value: T): T {
	if (value === undefined) {
		return value;
	}
	return JSON.parse(JSON.stringify(value)) as T;
}
