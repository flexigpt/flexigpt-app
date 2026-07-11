const SYSTEM_PROMPT_SEPARATOR = '\n---\n';

export interface SystemInstructionSource {
	identityKey: string;
	sourceKind: 'restored-conversation' | 'skill';
	bundleID: string;
	sourceSlug: string;
	displayName: string;
	text: string;
	bundleDisplayName: string;
	bundleSlug?: string;
	isBuiltIn: boolean;
}

function normalizePromptPart(value: string): string {
	return (value || '').trim();
}

function concatenateSystemPromptParts(parts: string[]): string {
	return parts
		.map(p => normalizePromptPart(p))
		.filter(Boolean)
		.join(SYSTEM_PROMPT_SEPARATOR);
}

export function buildEffectiveSystemPrompt(params: {
	modelDefaultPrompt: string;
	includeModelDefault: boolean;
	selectedInstructionSourceKeys: string[];
	instructionSourcesByKey: Map<string, SystemInstructionSource>;
}): string {
	const parts: string[] = [];

	if (params.includeModelDefault) {
		const modelDefault = normalizePromptPart(params.modelDefaultPrompt);
		if (modelDefault) {
			parts.push(modelDefault);
		}
	}

	for (const key of params.selectedInstructionSourceKeys) {
		const item = params.instructionSourcesByKey.get(key);
		if (!item) {
			continue;
		}

		const prompt = normalizePromptPart(item.text);
		if (!prompt) {
			continue;
		}

		parts.push(prompt);
	}

	return concatenateSystemPromptParts(parts);
}
