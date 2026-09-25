import type { ToolStoreChoice, ToolView, UIToolUserArgsStatus } from '@/spec/tool';
import { ToolStoreChoiceType } from '@/spec/tool';

import { toolIdentityKey } from '@/tools/lib/tool_identity_utils';

export interface ConversationToolStateEntry {
	key: string;
	toolStoreChoice: ToolStoreChoice;
	enabled: boolean;
	toolDefinition?: ToolView;
	toolLoadError?: string;
	argStatus?: UIToolUserArgsStatus;
}

/**
 * Input is frontend-enriched choices. Stored ToolSelections must first be
 * hydrated through toolManagementAPI.
 */
export function toolStoreChoicesToConversationTools(choices: ToolStoreChoice[]): ConversationToolStateEntry[] {
	const output: ConversationToolStateEntry[] = [];
	const seen = new Set<string>();

	for (const choice of choices ?? []) {
		if (choice.toolType === ToolStoreChoiceType.WebSearch) {
			continue;
		}
		const key = toolIdentityKey(choice.target);
		if (!seen.has(key)) {
			seen.add(key);
			output.push({ key, toolStoreChoice: choice, enabled: true });
		}
	}
	return output;
}

export function conversationToolsToChoices(entries: ConversationToolStateEntry[]): ToolStoreChoice[] {
	const output: ToolStoreChoice[] = [];
	const seen = new Set<string>();

	for (const entry of entries ?? []) {
		const choice = entry.toolStoreChoice;
		if (!entry.enabled || choice.toolType === ToolStoreChoiceType.WebSearch) {
			continue;
		}
		const key = toolIdentityKey(choice.target);
		if (!seen.has(key)) {
			seen.add(key);
			output.push(choice);
		}
	}
	return output;
}

export function mergeConversationToolsWithNewChoices(
	previous: ConversationToolStateEntry[],
	choices: ToolStoreChoice[]
): ConversationToolStateEntry[] {
	if (!choices?.length) {
		return previous;
	}

	const next = [...previous];
	const indexes = new Map(next.map((entry, index) => [entry.key, index]));

	for (const choice of choices) {
		if (choice.toolType === ToolStoreChoiceType.WebSearch) {
			continue;
		}
		const key = toolIdentityKey(choice.target);
		const index = indexes.get(key);

		if (index !== undefined) {
			next[index] = {
				...next[index],
				toolStoreChoice: choice,
			};
		} else {
			indexes.set(key, next.length);
			next.push({ key, toolStoreChoice: choice, enabled: true });
		}
	}
	return next;
}
