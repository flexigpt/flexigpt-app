import { type Tool, type ToolStoreChoice, ToolStoreChoiceType, type UIToolUserArgsStatus } from '@/spec/tool';

import { toolIdentityKey } from '@/tools/lib/tool_identity_utils';

export interface ConversationToolStateEntry {
	key: string;
	toolStoreChoice: ToolStoreChoice;
	enabled: boolean;
	/** Optional full tool definition, used for arg schema etc. */
	toolDefinition?: Tool;
	/** Cached status of userArgSchemaInstance vs schema. */
	argStatus?: UIToolUserArgsStatus;
}

/**
 * Initialize UI state from an array of ToolStoreChoice coming from history
 * (e.g. last user message's toolChoices).
 */
export function initConversationToolsStateFromChoices(choices: ToolStoreChoice[]): ConversationToolStateEntry[] {
	const out: ConversationToolStateEntry[] = [];
	const seen = new Set<string>();

	for (const t of choices ?? []) {
		if (t.toolType !== ToolStoreChoiceType.WebSearch) {
			const key = toolIdentityKey(t.bundleID, undefined, t.toolSlug, t.toolVersion);
			if (seen.has(key)) continue;
			seen.add(key);
			out.push({ key, toolStoreChoice: t, enabled: true });
		}
	}

	return out;
}

/**
 * Extract only the ENABLED tools, deduped by identity, for attachment to a message.
 */
export function conversationToolsToChoices(entries: ConversationToolStateEntry[]): ToolStoreChoice[] {
	if (!entries || entries.length === 0) return [];
	const out: ToolStoreChoice[] = [];
	const seen = new Set<string>();

	for (const e of entries) {
		if (!e.enabled) continue;
		const t = e.toolStoreChoice;
		if (t.toolType !== ToolStoreChoiceType.WebSearch) {
			const key = toolIdentityKey(t.bundleID, undefined, t.toolSlug, t.toolVersion);
			if (seen.has(key)) continue;
			seen.add(key);
			out.push(t);
		}
	}

	return out;
}

/**
 * After a send, merge any newly used tools into the UI state.
 * - Preserves existing enabled/disabled flags.
 * - Adds brand-new tools as enabled=true.
 */
export function mergeConversationToolsWithNewChoices(
	prev: ConversationToolStateEntry[],
	newTools: ToolStoreChoice[]
): ConversationToolStateEntry[] {
	if (!newTools || newTools.length === 0) return prev;

	const next = [...prev];
	const indexByKey = new Map<string, number>();
	for (let i = 0; i < next.length; i += 1) {
		indexByKey.set(next[i].key, i);
	}

	for (const t of newTools) {
		if (t.toolType !== ToolStoreChoiceType.WebSearch) {
			const key = toolIdentityKey(t.bundleID, undefined, t.toolSlug, t.toolVersion);
			const existingIdx = indexByKey.get(key);
			if (existingIdx != null) {
				// Refresh metadata but keep enabled flag.
				next[existingIdx] = {
					...next[existingIdx],
					toolStoreChoice: { ...next[existingIdx].toolStoreChoice, ...t },
				};
			} else {
				next.push({ key, toolStoreChoice: t, enabled: true });
			}
		}
	}

	return next;
}
