import type { SystemPromptItem } from '@/prompts/lib/use_system_prompts';

const SYSTEM_PROMPT_SEPARATOR = '\n---\n';

function normalizePromptPart(value: string): string {
	return (value || '').trim();
}

function concatenateSystemPromptParts(parts: string[]): string {
	return parts.map(normalizePromptPart).filter(Boolean).join(SYSTEM_PROMPT_SEPARATOR);
}

export function buildEffectiveSystemPrompt(params: {
	modelDefaultPrompt: string;
	includeModelDefault: boolean;
	selectedPromptKeys: string[];
	promptsByKey: Map<string, SystemPromptItem>;
}): string {
	const parts: string[] = [];

	if (params.includeModelDefault) {
		const modelDefault = normalizePromptPart(params.modelDefaultPrompt);
		if (modelDefault) {
			parts.push(modelDefault);
		}
	}

	for (const key of params.selectedPromptKeys) {
		const item = params.promptsByKey.get(key);
		if (!item) continue;

		const prompt = normalizePromptPart(item.prompt);
		if (!prompt) continue;

		parts.push(prompt);
	}

	return concatenateSystemPromptParts(parts);
}

export function countEnabledSystemPromptSources(params: {
	modelDefaultPrompt: string;
	includeModelDefault: boolean;
	selectedPromptKeys: string[];
	promptsByKey: Map<string, SystemPromptItem>;
}): number {
	let count = 0;

	if (params.includeModelDefault && normalizePromptPart(params.modelDefaultPrompt)) {
		count += 1;
	}

	for (const key of params.selectedPromptKeys) {
		const item = params.promptsByKey.get(key);
		if (item?.prompt.trim()) {
			count += 1;
		}
	}

	return count;
}
