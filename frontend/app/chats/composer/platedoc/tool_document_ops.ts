import type { Path } from 'platejs';
import type { PlateEditor } from 'platejs/react';
import { ElementApi, NodeApi } from 'platejs';

import type { ToolListItem, ToolView, UIToolStoreChoice } from '@/spec/tool';
import { ToolImplType } from '@/spec/tool';

import { getUUIDv7 } from '@/lib/uuid_utils';

import { toolChoiceFromListItem } from '@/apis/tool_management';

import type { ToolSelectionElementNode } from '@/chats/composer/platedoc/nodes';
import { KEY_TOOL_SELECTION } from '@/chats/composer/platedoc/nodes';
import { uiToolChoiceToToolStoreChoice } from '@/tools/lib/tool_choice_utils';
import { toolIdentityKey } from '@/tools/lib/tool_identity_utils';

export interface AttachedToolEntry extends UIToolStoreChoice {
	toolSnapshot?: ToolView;
	overrides?: ToolSelectionElementNode['overrides'];
}

function toAttachedToolEntry(node: ToolSelectionElementNode): AttachedToolEntry {
	return {
		...uiToolChoiceToToolStoreChoice(node),
		selectionID: node.selectionID,
		displayName: node.overrides?.displayName ?? node.displayName ?? node.toolSnapshot?.displayName ?? node.target.name,
		description: node.overrides?.description ?? node.description ?? node.toolSnapshot?.description,
		toolSnapshot: node.toolSnapshot,
		overrides: node.overrides,
	};
}

export function insertToolSelectionNode(editor: PlateEditor, item: ToolListItem, autoExecute: boolean): void {
	const key = toolIdentityKey(item.target);
	if (getAttachedToolEntries(editor).some(entry => toolIdentityKey(entry.target) === key)) {
		return;
	}

	const choice = toolChoiceFromListItem(item, autoExecute);
	const node: ToolSelectionElementNode = {
		...choice,
		target: { ...choice.target },
		type: KEY_TOOL_SELECTION,
		selectionID: `tool:${getUUIDv7()}`,
		toolSnapshot: structuredClone(item.toolDefinition),
		overrides: {},
		children: [{ text: '' }],
	};

	editor.tf.withoutNormalizing(() => {
		editor.tf.insertNodes([node, { text: '' }], { select: true });
	});
}

export function getAttachedToolEntries(editor: PlateEditor, uniqueByIdentity = true): AttachedToolEntry[] {
	const output: AttachedToolEntry[] = [];
	const seen = new Set<string>();

	for (const [element] of NodeApi.elements(editor)) {
		if (!ElementApi.isElementType(element, KEY_TOOL_SELECTION)) {
			continue;
		}
		const node = element as unknown as ToolSelectionElementNode;
		const key = toolIdentityKey(node.target);
		if (uniqueByIdentity && seen.has(key)) {
			continue;
		}
		seen.add(key);
		output.push(toAttachedToolEntry(node));
	}
	return output;
}

export function getAttachedTools(editor: PlateEditor): UIToolStoreChoice[] {
	return getAttachedToolEntries(editor).map(entry => ({
		...uiToolChoiceToToolStoreChoice(entry),
		selectionID: entry.selectionID,
	}));
}

export function setAttachedToolUserArgSchemaInstanceBySelectionID(
	editor: PlateEditor,
	selectionID: string,
	newInstance: string
): boolean {
	for (const [element, path] of NodeApi.elements(editor)) {
		if (!ElementApi.isElementType(element, KEY_TOOL_SELECTION)) {
			continue;
		}
		const node = element as unknown as ToolSelectionElementNode;
		if (node.selectionID === selectionID) {
			editor.tf.setNodes<ToolSelectionElementNode>({ userArgSchemaInstance: newInstance }, { at: path });
			return true;
		}
	}
	return false;
}

export function removeToolByKey(editor: PlateEditor, identityKey: string): void {
	const paths: Path[] = [];
	for (const [element, path] of NodeApi.elements(editor)) {
		if (
			ElementApi.isElementType(element, KEY_TOOL_SELECTION) &&
			toolIdentityKey((element as unknown as ToolSelectionElementNode).target) === identityKey
		) {
			paths.push(path);
		}
	}

	editor.tf.withoutNormalizing(() => {
		for (const path of paths.toReversed()) {
			editor.tf.removeNodes({ at: path });
		}
	});
}

export function setToolAutoExecuteByKey(editor: PlateEditor, identityKey: string, autoExecute: boolean): void {
	const updates: Array<{ path: Path; autoExecute: boolean }> = [];

	for (const [element, path] of NodeApi.elements(editor)) {
		if (!ElementApi.isElementType(element, KEY_TOOL_SELECTION)) {
			continue;
		}
		const node = element as unknown as ToolSelectionElementNode;
		if (toolIdentityKey(node.target) === identityKey) {
			updates.push({
				path,
				autoExecute: node.implementationKind === ToolImplType.Go && autoExecute,
			});
		}
	}

	editor.tf.withoutNormalizing(() => {
		for (const update of updates) {
			editor.tf.setNodes<ToolSelectionElementNode>({ autoExecute: update.autoExecute }, { at: update.path });
		}
	});
}
