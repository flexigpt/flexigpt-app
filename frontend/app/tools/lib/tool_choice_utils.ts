import type { ToolSelection, ToolStoreChoice, UIToolStoreChoice } from '@/spec/tool';

import { toolIdentityKey } from '@/tools/lib/tool_identity_utils';

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
		collectionRef: att.collectionRef,
		collectionName: att.collectionName,
	};
}

/** Exact backend and persistence projection. */
export function toolSelectionFromChoice(choice: ToolSelection): ToolSelection {
	return {
		choiceID: choice.choiceID,
		target: choice.target,
		autoExecute: choice.autoExecute,
		userArgSchemaInstance: choice.userArgSchemaInstance,
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
