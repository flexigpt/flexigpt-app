import type { ToolSelection, ToolStoreChoice, UIToolStoreChoice } from '@/spec/tool';

import { toolIdentityKey } from '@/apis/tool_management';

/** Editor selection to frontend choice. */
export function uiToolChoiceToToolStoreChoice(att: UIToolStoreChoice): ToolStoreChoice {
	return {
		choiceID: att.choiceID,
		target: att.target,
		autoExecute: att.autoExecute,
		userArgSchemaInstance: att.userArgSchemaInstance,
		toolType: att.toolType,
		implementationKind: att.implementationKind,
		sdkType: att.sdkType,
		displayName: att.displayName,
		description: att.description,
		toolVersion: att.toolVersion,
		artifactRevision: att.artifactRevision,
		definitionDigest: att.definitionDigest,
		pluginRef: att.pluginRef,
		pluginName: att.pluginName,
	};
}

export function dedupeToolChoices<T extends ToolSelection>(choices: T[]): T[] {
	const output: T[] = [];
	const seen = new Set<string>();

	for (const choice of choices ?? []) {
		const key = toolIdentityKey(choice.target);
		if (!seen.has(key)) {
			seen.add(key);
			output.push(choice);
		}
	}
	return output;
}
