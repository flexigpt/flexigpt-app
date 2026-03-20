import { type ToolStoreChoice, type UIToolStoreChoice } from '@/spec/tool';

import { toolIdentityKey } from '@/tools/lib/tool_identity_utils';

// Convert the editor's attached-tool shape into the persisted ToolStoreChoice shape.
export function uiToolChoiceToToolStoreChoice(att: UIToolStoreChoice): ToolStoreChoice {
	return {
		choiceID: att.choiceID,
		bundleID: att.bundleID,
		toolSlug: att.toolSlug,
		toolVersion: att.toolVersion,
		displayName: att.displayName,
		description: att.description,
		toolID: att.toolID,
		toolType: att.toolType,
		autoExecute: att.autoExecute,
		userArgSchemaInstance: att.userArgSchemaInstance,
	};
}

export function dedupeToolChoices(choices: ToolStoreChoice[]): ToolStoreChoice[] {
	const out: ToolStoreChoice[] = [];
	const seen = new Set<string>();

	for (const t of choices ?? []) {
		const key = toolIdentityKey(t.bundleID, undefined, t.toolSlug, t.toolVersion);
		if (seen.has(key)) continue;
		seen.add(key);
		out.push(t);
	}

	return out;
}
