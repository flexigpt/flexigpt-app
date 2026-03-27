import { ElementApi, NodeApi, type Path } from 'platejs';
import type { PlateEditor } from 'platejs/react';

import { type Tool, ToolStoreChoiceType, type UIToolStoreChoice } from '@/spec/tool';

import { getUUIDv7 } from '@/lib/uuid_utils';

import { KEY_TOOL_SELECTION, type ToolSelectionElementNode } from '@/chats/composer/platedoc/nodes';
import { toolIdentityKey } from '@/tools/lib/tool_identity_utils';

export interface AttachedToolEntry {
	choiceID: string;
	bundleID: string;
	bundleSlug?: string;
	toolSlug: string;
	toolVersion: string;
	selectionID: string;
	toolType: ToolStoreChoiceType;
	autoExecute: boolean;
	userArgSchemaInstance?: string;
	toolSnapshot?: Tool;
	overrides?: ToolSelectionElementNode['overrides'];
}

function toolIdentityKeyFromNode(
	n: Pick<ToolSelectionElementNode, 'bundleID' | 'bundleSlug' | 'toolSlug' | 'toolVersion'>
): string {
	return toolIdentityKey(n.bundleID, n.bundleSlug, n.toolSlug, n.toolVersion);
}

function getAttachedToolKeySet(editor: PlateEditor): Set<string> {
	const keys = new Set<string>();
	for (const [el] of NodeApi.elements(editor)) {
		if (ElementApi.isElementType(el, KEY_TOOL_SELECTION)) {
			keys.add(toolIdentityKeyFromNode(el as unknown as ToolSelectionElementNode));
		}
	}
	return keys;
}

function toAttachedToolEntry(node: ToolSelectionElementNode): AttachedToolEntry {
	return {
		choiceID: node.choiceID,
		bundleID: node.bundleID,
		bundleSlug: node.bundleSlug,
		toolSlug: node.toolSlug,
		toolVersion: node.toolVersion,
		selectionID: node.selectionID,
		toolType: node.toolType,
		autoExecute: node.autoExecute,
		userArgSchemaInstance: node.userArgSchemaInstance,
		toolSnapshot: node.toolSnapshot,
		overrides: node.overrides,
	};
}

// Insert a hidden tool selection chip (inline+void) to drive bottom bar UI.
export function insertToolSelectionNode(
	editor: PlateEditor,
	item: {
		bundleID: string;
		bundleSlug?: string;
		toolSlug: string;
		toolVersion: string;
	},
	toolSnapshot?: Tool,
	opts?: {
		toolType?: ToolStoreChoiceType;
		choiceID?: string;
		autoExecute: boolean;
		userArgSchemaInstance?: string;
	}
) {
	const identity = toolIdentityKey(item.bundleID, item.bundleSlug, item.toolSlug, item.toolVersion);
	if (getAttachedToolKeySet(editor).has(identity)) {
		return;
	}

	const selectionID = `tool:${item.bundleID}/${item.toolSlug}@${item.toolVersion}:${Date.now().toString(36)}${Math.random()
		.toString(36)
		.slice(2, 8)}`;

	const node: ToolSelectionElementNode = {
		type: KEY_TOOL_SELECTION,
		choiceID: opts?.choiceID ?? getUUIDv7(),

		bundleID: item.bundleID,
		bundleSlug: item.bundleSlug,
		toolSlug: item.toolSlug,
		toolVersion: item.toolVersion,

		toolType: opts?.toolType ?? toolSnapshot?.llmToolType ?? ToolStoreChoiceType.Function,
		autoExecute: opts?.autoExecute ?? toolSnapshot?.autoExecReco ?? false,
		userArgSchemaInstance: opts?.userArgSchemaInstance,

		selectionID,
		toolSnapshot,
		overrides: {},
		children: [{ text: '' }],
	};

	editor.tf.withoutNormalizing(() => {
		// Insert the tool chip (invisible inline) and an empty text leaf after it
		// so the caret has a cheap place to land without forcing block normalization.
		editor.tf.insertNodes([node, { text: '' }], { select: true });
	});
}

// By default, returns only the first occurrence for each unique tool identity.
export function getAttachedToolEntries(editor: PlateEditor, uniqueByIdentity?: boolean): AttachedToolEntry[] {
	const out: AttachedToolEntry[] = [];
	const unique = uniqueByIdentity ?? true;
	const seen = unique ? new Set<string>() : undefined;

	for (const [el] of NodeApi.elements(editor)) {
		if (ElementApi.isElementType(el, KEY_TOOL_SELECTION)) {
			const node = el as unknown as ToolSelectionElementNode;
			if (unique) {
				const key = toolIdentityKeyFromNode(node);
				if ((seen as Set<string>).has(key)) continue;
				(seen as Set<string>).add(key);
			}
			out.push(toAttachedToolEntry(node));
		}
	}

	return out;
}

export function setAttachedToolUserArgSchemaInstanceBySelectionID(
	editor: PlateEditor,
	selectionID: string,
	newInstance: string
): boolean {
	for (const [el, path] of NodeApi.elements(editor)) {
		if (ElementApi.isElementType(el, KEY_TOOL_SELECTION)) {
			const node = el as unknown as ToolSelectionElementNode;
			if (node.selectionID !== selectionID) continue;

			editor.tf.setNodes<ToolSelectionElementNode>({ userArgSchemaInstance: newInstance }, { at: path });
			return true;
		}
	}

	return false;
}

// Remove all instances of a tool by identity key (bundle+slug+version).
export function removeToolByKey(editor: PlateEditor, identityKey: string) {
	const paths: Path[] = [];
	for (const [el, path] of NodeApi.elements(editor)) {
		if (ElementApi.isElementType(el, KEY_TOOL_SELECTION)) {
			const n = el as unknown as ToolSelectionElementNode;
			if (toolIdentityKeyFromNode(n) === identityKey) {
				paths.push(path);
			}
		}
	}
	// Remove from last to first to avoid path shift issues.
	for (const p of paths.reverse()) {
		try {
			editor.tf.removeNodes({ at: p });
		} catch {
			// swallow
		}
	}
}

// Build a serializable list of attached tools for submission
export function getAttachedTools(editor: PlateEditor): UIToolStoreChoice[] {
	const items: UIToolStoreChoice[] = [];
	const seen = new Set<string>();

	for (const [el] of NodeApi.elements(editor)) {
		if (ElementApi.isElementType(el, KEY_TOOL_SELECTION)) {
			const n = el as unknown as ToolSelectionElementNode;
			const key = toolIdentityKeyFromNode(n);
			if (seen.has(key)) continue;
			seen.add(key);
			items.push({
				choiceID: n.choiceID,
				selectionID: n.selectionID,
				bundleID: n.bundleID,
				toolSlug: n.toolSlug,
				toolVersion: n.toolVersion,
				displayName: n.overrides?.displayName
					? n.overrides.displayName
					: n.toolSnapshot?.displayName
						? n.toolSnapshot.displayName
						: n.toolSlug,
				description: n.overrides?.description
					? n.overrides.description
					: n.toolSnapshot?.description
						? n.toolSnapshot.description
						: n.toolSlug,
				toolID: n.toolSnapshot?.id,
				toolType: n.toolType,
				autoExecute: n.autoExecute,
				userArgSchemaInstance: n.userArgSchemaInstance,
			});
		}
	}
	return items;
}

// Update autoExecute for all instances of a tool by identity key (bundle+slug+version).
export function setToolAutoExecuteByKey(editor: PlateEditor, identityKey: string, autoExecute: boolean) {
	const paths: Path[] = [];
	for (const [el, path] of NodeApi.elements(editor)) {
		if (ElementApi.isElementType(el, KEY_TOOL_SELECTION)) {
			const n = el as unknown as ToolSelectionElementNode;
			if (toolIdentityKeyFromNode(n) === identityKey) {
				paths.push(path);
			}
		}
	}

	if (paths.length === 0) return;

	editor.tf.withoutNormalizing(() => {
		for (const p of paths) {
			try {
				// Update the hidden carrier node; chips read from this.
				editor.tf.setNodes<ToolSelectionElementNode>({ autoExecute }, { at: p });
			} catch {
				// swallow
			}
		}
	});
}
