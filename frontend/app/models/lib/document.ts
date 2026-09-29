import type { ArtifactRef } from '@/spec/artifact';
import type {
	ModelAdapterParameters,
	ModelArtifactNameReference,
	ModelCapabilities,
	ModelDefaults,
} from '@/spec/model';

export function modelRefEqual(left: ArtifactRef | undefined, right: ArtifactRef | undefined): boolean {
	return left?.rootID === right?.rootID && left?.artifactID === right?.artifactID;
}

export function formatJSON(value: unknown): string {
	if (value === undefined || value === null) {
		return '';
	}

	return JSON.stringify(value, null, 2);
}

// oxlint-disable-next-line typescript/no-unnecessary-type-parameters
export function parseOptionalJSONObject<T extends object>(value: string, label: string): T | undefined {
	const trimmed = value.trim();
	if (!trimmed) {
		return undefined;
	}

	let parsed: unknown;
	try {
		parsed = JSON.parse(trimmed);
	} catch {
		throw new Error(`${label} must be valid JSON.`);
	}

	if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
		throw new Error(`${label} must be a JSON object.`);
	}

	return parsed as T;
}

export function parseOptionalLabels(value: string): Record<string, string> | undefined {
	const parsed = parseOptionalJSONObject<Record<string, unknown>>(value, 'Labels');
	if (!parsed) {
		return undefined;
	}

	const labels: Record<string, string> = {};
	for (const [key, item] of Object.entries(parsed)) {
		if (typeof item !== 'string') {
			throw new TypeError(`Label "${key}" must have a string value.`);
		}
		labels[key] = item;
	}

	return labels;
}

export function parseOptionalReference(value: string): ModelArtifactNameReference | undefined {
	const parsed = parseOptionalJSONObject<ModelArtifactNameReference>(value, 'Default model reference');

	if (!parsed) {
		return undefined;
	}
	if (!parsed.name?.trim()) {
		throw new Error('Default model reference requires a name.');
	}

	return parsed;
}

export function parseProviderDefaults(value: string): ModelDefaults | undefined {
	return parseOptionalJSONObject<ModelDefaults>(value, 'Provider defaults');
}

export function parseModelDefaults(value: string): ModelDefaults | undefined {
	return parseOptionalJSONObject<ModelDefaults>(value, 'Model defaults');
}

export function parseCapabilities(value: string): ModelCapabilities | undefined {
	return parseOptionalJSONObject<ModelCapabilities>(value, 'Capabilities');
}

export function parseAdapterParameters(value: string): ModelAdapterParameters | undefined {
	return parseOptionalJSONObject<ModelAdapterParameters>(value, 'Adapter parameters');
}
