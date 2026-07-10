const SYSTEM_PROMPT_SEPARATOR = '\n---\n';

export enum PromptRoleEnum {
	System = 'system',
	Developer = 'developer',
	User = 'user',
}
type SystemPromptRole = PromptRoleEnum.System | PromptRoleEnum.Developer;

export interface SystemPromptItem {
	identityKey: string;
	templateID?: string;
	bundleID: string;
	templateSlug: string;
	templateVersion: string;
	displayName: string;
	prompt: string;
	role: SystemPromptRole;
	bundleDisplayName: string;
	bundleSlug: string;
	isBuiltIn: boolean;
	createdAt: string;
	modifiedAt: string;
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
		if (!item) {
			continue;
		}

		const prompt = normalizePromptPart(item.prompt);
		if (!prompt) {
			continue;
		}

		parts.push(prompt);
	}

	return concatenateSystemPromptParts(parts);
}
