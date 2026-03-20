import { type MessageBlock, PromptRoleEnum, PromptTemplateKind, type PromptVariable, VarSource } from '@/spec/prompt';

const PROMPT_PLACEHOLDER_RE = /\{\{([a-zA-Z_][a-zA-Z0-9_-]*)\}\}/g;
const PROMPT_VARIABLE_NAME_RE = /^[a-zA-Z_][a-zA-Z0-9_-]*$/;
export interface PromptTemplateUpsertInput {
	displayName: string;
	slug: string;
	description: string;
	isEnabled: boolean;
	tags: string[];
	version: string;
	blocks: MessageBlock[];
	variables: PromptVariable[];
}

export function cloneVariable(variable: PromptVariable): PromptVariable {
	return {
		...variable,
		enumValues: variable.enumValues ? [...variable.enumValues] : variable.enumValues,
	};
}

export function extractPromptTemplatePlaceholders(blocks: MessageBlock[]): string[] {
	const names = new Set<string>();

	for (const block of blocks) {
		for (const match of block.content.matchAll(PROMPT_PLACEHOLDER_RE)) {
			if (match[1]) {
				names.add(match[1]);
			}
		}
	}

	return [...names];
}

export function validatePromptVariableName(name: string): string | undefined {
	const value = name.trim();

	if (!value) {
		return 'This field is required.';
	}

	if (!PROMPT_VARIABLE_NAME_RE.test(value)) {
		return 'Variable names must match [a-zA-Z_][a-zA-Z0-9_-]*.';
	}

	return undefined;
}

export function derivePromptTemplateKind(blocks: MessageBlock[]): PromptTemplateKind {
	if (
		blocks.length > 0 &&
		blocks.every(block => block.role === PromptRoleEnum.System || block.role === PromptRoleEnum.Developer)
	) {
		return PromptTemplateKind.InstructionsOnly;
	}

	return PromptTemplateKind.Generic;
}

export function derivePromptTemplateResolved(blocks: MessageBlock[], variables?: PromptVariable[]): boolean {
	const placeholders = extractPromptTemplatePlaceholders(blocks);
	const variableMap = new Map((variables ?? []).map(variable => [variable.name.trim(), variable] as const));

	return placeholders.every(name => {
		const variable = variableMap.get(name);

		if (!variable) {
			return false;
		}

		if (variable.source === VarSource.Static) {
			return (variable.staticVal ?? '').trim().length > 0;
		}

		return variable.default !== undefined;
	});
}

export function getPromptTemplateKindLabel(kind: PromptTemplateKind): string {
	return kind === PromptTemplateKind.InstructionsOnly ? 'Instructions Only' : 'Generic';
}

export function getPromptTemplateResolutionLabel(isResolved: boolean): string {
	return isResolved ? 'Resolved' : 'Needs Input';
}
