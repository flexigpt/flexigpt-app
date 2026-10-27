export interface PortableSource<T> {
	data: T;
	text: string;
	fileName: string;
	extension: string;
}

type PortableScalar = string | number | boolean | null;
type PortableValue = PortableScalar | PortableValue[] | { [key: string]: PortableValue };

const OMIT = Symbol('omit');

const PLUGIN_MANAGEMENT_FIELDS = new Set([
	'artifact',
	'artifactid',
	'artifactref',
	'rootid',
	'sourceid',
	'revision',
	'definitiondigest',
	'binding',
	'createdat',
	'modifiedat',
	'target',
	'selectormatches',
	'resolved',
	'diagnostics',
	'state',
	'enabled',
	'id',
]);

function asRecord(value: unknown): Record<string, unknown> | undefined {
	return value !== null && typeof value === 'object' && !Array.isArray(value)
		? (value as Record<string, unknown>)
		: undefined;
}

function normalizePortableValue(value: unknown, ancestors: WeakSet<object>): PortableValue | typeof OMIT {
	if (value === undefined) {
		return OMIT;
	}

	if (value === null || typeof value === 'string' || typeof value === 'boolean') {
		return value;
	}

	if (typeof value === 'number') {
		if (!Number.isFinite(value)) {
			throw new TypeError('Portable source cannot contain a non-finite number.');
		}

		return value;
	}

	if (typeof value === 'bigint' || typeof value === 'function' || typeof value === 'symbol') {
		throw new TypeError(`Portable source cannot contain ${typeof value} values.`);
	}

	if (value instanceof Date) {
		return value.toISOString();
	}

	if (ancestors.has(value)) {
		throw new Error('Portable source cannot contain circular values.');
	}

	ancestors.add(value);

	try {
		if (Array.isArray(value)) {
			const output: PortableValue[] = [];

			for (const item of value) {
				const normalized = normalizePortableValue(item, ancestors);

				if (normalized !== OMIT) {
					output.push(normalized);
				}
			}

			return output;
		}

		const output: Record<string, PortableValue> = {};

		for (const [key, item] of Object.entries(value as Record<string, unknown>)) {
			const normalized = normalizePortableValue(item, ancestors);

			if (normalized !== OMIT) {
				output[key] = normalized;
			}
		}

		return output;
	} finally {
		ancestors.delete(value);
	}
}

function isPortableScalar(value: PortableValue): value is PortableScalar {
	return value === null || typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean';
}

function isInlineValue(value: PortableValue): boolean {
	return (
		isPortableScalar(value) ||
		(Array.isArray(value) && value.length === 0) ||
		(!Array.isArray(value) && Object.keys(value).length === 0)
	);
}

function formatScalar(value: PortableScalar): string {
	if (value === null) {
		return 'null';
	}

	if (typeof value === 'string') {
		return JSON.stringify(value) ?? '""';
	}

	return String(value);
}

function formatKey(value: string): string {
	return /^[A-Za-z_][A-Za-z0-9_-]*$/.test(value) ? value : (JSON.stringify(value) ?? '""');
}

function renderPortableYAML(value: PortableValue, indentation: number): string {
	const prefix = ' '.repeat(indentation);

	if (isPortableScalar(value)) {
		return `${prefix}${formatScalar(value)}`;
	}

	if (Array.isArray(value)) {
		if (value.length === 0) {
			return `${prefix}[]`;
		}

		return value
			.map(item => {
				if (isInlineValue(item)) {
					return `${prefix}- ${renderPortableYAML(item, 0).trimStart()}`;
				}

				return `${prefix}-\n${renderPortableYAML(item, indentation + 2)}`;
			})
			.join('\n');
	}

	const entries = Object.entries(value);

	if (entries.length === 0) {
		return `${prefix}{}`;
	}

	return entries
		.map(([key, item]) => {
			if (isInlineValue(item)) {
				return `${prefix}${formatKey(key)}: ${renderPortableYAML(item, 0).trimStart()}`;
			}

			return `${prefix}${formatKey(key)}:\n${renderPortableYAML(item, indentation + 2)}`;
		})
		.join('\n');
}

/**
 * Produces deterministic, portable YAML from a source document. JSON string
 * escaping is valid YAML syntax and preserves multiline text exactly.
 */
export function formatPortableYAML(value: unknown): string {
	const normalized = normalizePortableValue(value, new WeakSet<object>());

	if (normalized === OMIT) {
		throw new Error('Portable source cannot be empty.');
	}

	return `${renderPortableYAML(normalized, 0)}\n`;
}

function stripPluginManagementFields(value: unknown): unknown {
	if (value === null || typeof value !== 'object') {
		return value;
	}

	if (Array.isArray(value)) {
		return value.map(v => {
			return stripPluginManagementFields(v);
		});
	}

	const output: Record<string, unknown> = {};

	for (const [key, item] of Object.entries(value as Record<string, unknown>)) {
		if (PLUGIN_MANAGEMENT_FIELDS.has(key.toLowerCase())) {
			continue;
		}

		output[key] = stripPluginManagementFields(item);
	}

	return output;
}

function projectPluginMember(value: unknown): unknown {
	const member = asRecord(value);
	const entry = member ? asRecord(member.entry) : undefined;

	return stripPluginManagementFields(entry ?? value);
}

/**
 * Plugin views include runtime resolution and store metadata. A portable
 * Plugin source contains only declaration fields that can be shared.
 */
export function pluginShareableSourceDocument(plugin: {
	name: string;
	displayName?: string;
	description?: string;
	members?: readonly unknown[];
}): Record<string, unknown> {
	const name = plugin.name.trim();

	if (!name) {
		throw new Error('Plugin source requires a name.');
	}

	const displayName = plugin.displayName?.trim();
	const description = plugin.description?.trim();
	const members = (plugin.members ?? []).map(m => {
		return projectPluginMember(m);
	});

	return {
		name,
		...(displayName ? { displayName } : {}),
		...(description ? { description } : {}),
		...(members.length > 0 ? { members } : {}),
	};
}

export function portableSourceFileName(name: string, extension: string, fallback: string): string {
	const safeExtension = extension.replaceAll(/^\.+/g, '') || 'yaml';
	const safeFallback = fallback
		.trim()
		.toLowerCase()
		.replaceAll(/[^a-z0-9._-]+/g, '-')
		.replaceAll(/^-+|-+$/g, '')
		.replace(/\.(?:json|ya?ml)$/i, '');

	const stem =
		name
			.trim()
			.toLowerCase()
			.replace(/\.(?:json|ya?ml)$/i, '')
			.replaceAll(/[^a-z0-9._-]+/g, '-')
			.replaceAll(/^-+|-+$/g, '')
			.slice(0, 96) ||
		safeFallback ||
		'source';

	return `${stem}.${safeExtension}`;
}

export function createPortableYAMLSource<T>(data: T, fileName: string): PortableSource<T> {
	return {
		data,
		text: formatPortableYAML(data),
		fileName,
		extension: 'yaml',
	};
}
