import { ElementApi, NodeApi, type Path } from 'platejs';
import type { PlateEditor } from 'platejs/react';

import type { PromptTemplate, PromptVariable } from '@/spec/prompt';

import {
	KEY_TEMPLATE_SELECTION,
	KEY_TEMPLATE_VARIABLE,
	type TemplateSelectionElementNode,
	type TemplateVariableElementNode,
} from '@/chats/inputarea/platedoc/nodes';
import {
	computeEffectiveTemplate,
	effectiveVarValueLocal,
	makeSelectedTemplateForRun,
	type SelectedTemplateForRun,
} from '@/chats/inputarea/platedoc/templates/template_processing';

type TemplateNodeWithPath = [TemplateSelectionElementNode, Path];

export function insertTemplateSelectionNode(
	editor: PlateEditor,
	bundleID: string,
	templateSlug: string,
	templateVersion: string,
	template?: PromptTemplate
) {
	const selectionID = `tpl:${bundleID}/${templateSlug}@${templateVersion}:${Date.now().toString(36)}${Math.random()
		.toString(36)
		.slice(2, 8)}`;
	const nnode: TemplateSelectionElementNode = {
		type: KEY_TEMPLATE_SELECTION,
		bundleID,
		templateSlug,
		templateVersion,
		selectionID,
		variables: {} as Record<string, unknown>,
		// Snapshot full template for downstream sync "get" to have the full context.
		templateSnapshot: template,
		// Local overrides
		overrides: {} as {
			displayName?: string;
			description?: string;
			tags?: string[];
			blocks?: PromptTemplate['blocks'];
			variables?: PromptTemplate['variables'];
		},

		// void elements still need one empty text child in Slate
		children: [{ text: '' }],
	};

	editor.tf.withoutNormalizing(() => {
		// Insert ONLY inline content here. This composer is single-block; inserting
		// a paragraph block from an inline helper creates unstable intermediate
		// structure and forces unnecessary normalization.
		//
		// The trailing empty text leaf gives Slate a caret target immediately after
		// the hidden selection node until the template-population effect inserts the
		// user-visible inline content/variables.
		editor.tf.insertNodes([nnode, { text: '' }], { select: true });
		try {
			editor.tf.collapse({ edge: 'end' });
		} catch {
			// Best-effort: keep inserted nodes even if selection collapse is unavailable.
		}
	});
}

// Utility to get selections for sending
export function getTemplateSelections(editor: PlateEditor): SelectedTemplateForRun[] {
	const elList = NodeApi.elements(editor);
	const selections: SelectedTemplateForRun[] = [];
	for (const [el] of elList) {
		if (ElementApi.isElementType(el, KEY_TEMPLATE_SELECTION)) {
			const node = el as unknown as TemplateSelectionElementNode;
			selections.push(makeSelectedTemplateForRun(node));
		}
	}

	return selections;
}

// Utility to get the first template node with its path
export function getFirstTemplateNodeWithPath(editor: PlateEditor): TemplateNodeWithPath | undefined {
	const elList = NodeApi.elements(editor);
	for (const [el, path] of elList) {
		if (ElementApi.isElementType(el, KEY_TEMPLATE_SELECTION)) {
			return [el as unknown as TemplateSelectionElementNode, path];
		}
	}
	return undefined;
}

// Utility to get all template selection nodes with their paths (document order)
export function getTemplateNodesWithPath(editor: PlateEditor): TemplateNodeWithPath[] {
	const out: TemplateNodeWithPath[] = [];
	const elList = NodeApi.elements(editor);
	for (const [el, path] of elList) {
		if (ElementApi.isElementType(el, KEY_TEMPLATE_SELECTION)) {
			out.push([el as unknown as TemplateSelectionElementNode, path]);
		}
	}
	return out;
}

// Flatten current editor content into plain text (single-block), replacing variable pills of the first template.
// Used when extracting text to submit without mutating content.
export function toPlainTextReplacingVariables(editor: PlateEditor): string {
	// Build per-selection effective context (defs + overrides + tools) so we can resolve values consistently
	const selections = getTemplateNodesWithPath(editor);
	const ctxBySelection = new Map<
		string,
		{
			defsByName: Map<string, PromptVariable>;
			userValues: Record<string, unknown>;
		}
	>();

	for (const [node] of selections) {
		if (!node.selectionID) continue;
		const { variablesSchema } = computeEffectiveTemplate(node);
		ctxBySelection.set(node.selectionID, {
			defsByName: new Map(variablesSchema.map(v => [v.name, v] as const)),
			userValues: node.variables,
		});
	}
	function toStringDeepWithVars(n: any): string {
		if (!n || typeof n !== 'object' || n === null) return '';

		if (isTemplateVarNode(n)) {
			const name = n.name;
			const sid = n.selectionID as string | undefined;
			const placeholder = `{{${name}}}`;

			if (!sid) return placeholder;
			const ctx = ctxBySelection.get(sid);
			if (!ctx) return placeholder;

			const def = ctx.defsByName.get(name);
			if (!def) return placeholder; // unknown var (shouldn't happen)

			// Resolve the effective value like the inline pill does
			const val = effectiveVarValueLocal(def, ctx.userValues);
			if (val !== undefined && val !== null) {
				// eslint-disable-next-line @typescript-eslint/no-base-to-string
				return String(val);
			}
			// If variable is optional and still unresolved, substitute empty string
			if (!def.required) {
				return '';
			}
			// For required or unknown vars, keep the placeholder to signal missing data
			return placeholder;
		}

		const obj = n as Record<PropertyKey, unknown>;

		if ('text' in obj && typeof obj.text === 'string') {
			return obj.text;
		}

		if ('children' in obj && Array.isArray(obj.children)) {
			return obj.children.map(toStringDeepWithVars).join('');
		}

		return '';
	}

	const rootNodes = editor.children ?? [];
	return rootNodes.map(toStringDeepWithVars).join('\n');
}

function isTemplateVarNode(n: unknown): n is TemplateVariableElementNode {
	if (!n || typeof n !== 'object') return false;
	const obj = n as Record<PropertyKey, unknown>;
	return 'type' in obj && obj.type === KEY_TEMPLATE_VARIABLE && 'name' in obj && typeof obj.name === 'string';
}

export function analyzeTemplateSelectionInfo(editor: PlateEditor) {
	// Fast path: if the document contains no template-selection elements at all,
	// short-circuit instead of running the heavier helpers.
	const tplNodeWithPath = getFirstTemplateNodeWithPath(editor);
	if (!tplNodeWithPath) {
		return {
			tplNodeWithPath: undefined,
			hasTemplate: false,
			requiredCount: 0,
			firstPendingVar: undefined,
		};
	}

	const selections = getTemplateSelections(editor);
	const hasTemplate = selections.length > 0;

	let requiredCount = 0;
	let firstPendingVar: { name: string; selectionID?: string } | undefined = undefined;

	for (const s of selections) {
		requiredCount += s.requiredCount;

		if (!firstPendingVar && s.requiredVariables.length > 0) {
			firstPendingVar = {
				name: s.requiredVariables[0],
				selectionID: s.selectionID,
			};
		}
	}

	return {
		tplNodeWithPath,
		hasTemplate,
		requiredCount,
		firstPendingVar,
	};
}
