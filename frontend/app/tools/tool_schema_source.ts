import type { ToolView } from '@/spec/tool';

export type ToolSchemaKind = 'input' | 'output';

function asRecord(value: unknown): Record<string, unknown> | undefined {
	return value !== null && typeof value === 'object' && !Array.isArray(value)
		? (value as Record<string, unknown>)
		: undefined;
}

function readPath(source: unknown, path: readonly string[]): unknown {
	let current = source;

	for (const segment of path) {
		const record = asRecord(current);

		if (!record) {
			return undefined;
		}

		current = record[segment];
	}

	return current;
}

// oxlint-disable-next-line typescript/no-redundant-type-constituents
function normalizeSchema(value: unknown): unknown | undefined {
	if (typeof value === 'string') {
		try {
			return JSON.parse(value) as unknown;
		} catch {
			return undefined;
		}
	}

	if (typeof value === 'boolean' || (value !== null && typeof value === 'object')) {
		return value;
	}

	return undefined;
}

/**
 * Tool source declarations have used more than one schema field shape during
 * migration. This reads only source-schema fields and never serializes the
 * Tool view itself.
 */
// oxlint-disable-next-line typescript/no-redundant-type-constituents
export function getToolSchema(tool: ToolView, kind: ToolSchemaKind): unknown | undefined {
	const paths =
		kind === 'input'
			? [
					['inputSchema'],
					['input_schema'],
					['parameters'],
					['schema', 'input'],
					['schema', 'inputSchema'],
					['definition', 'inputSchema'],
					['toolDefinition', 'inputSchema'],
				]
			: [
					['outputSchema'],
					['output_schema'],
					['resultSchema'],
					['schema', 'output'],
					['schema', 'outputSchema'],
					['definition', 'outputSchema'],
					['toolDefinition', 'outputSchema'],
				];

	for (const path of paths) {
		const schema = normalizeSchema(readPath(tool, path));

		if (schema !== undefined) {
			return schema;
		}
	}

	return undefined;
}

export function toolSchemaFileName(tool: Pick<ToolView, 'name'>, kind: ToolSchemaKind): string {
	const stem =
		tool.name
			.trim()
			.toLowerCase()
			.replaceAll(/[^a-z0-9._-]+/g, '-')
			.replaceAll(/^-+|-+$/g, '')
			.slice(0, 96) || 'tool';

	return `${stem}.${kind}.schema.json`;
}
