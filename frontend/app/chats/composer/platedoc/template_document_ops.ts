import { ElementApi, NodeApi, type Path } from 'platejs';
import type { PlateEditor } from 'platejs/react';

import {
	type MessageBlock,
	PromptRoleEnum,
	type PromptTemplate,
	PromptTemplateKind,
	type PromptVariable,
} from '@/spec/prompt';

import {
	KEY_TEMPLATE_SELECTION,
	KEY_TEMPLATE_VARIABLE,
	type TemplateSelectionElementNode,
	type TemplateVariableElementNode,
} from '@/chats/composer/platedoc/nodes';
import { computeTemplateVarRequirements, effectiveVarValueLocal } from '@/prompts/lib/prompt_template_var_utils';

const TEMPLATE_PLACEHOLDER_RE = /\{\{([a-zA-Z_][a-zA-Z0-9_-]*)\}\}/g;
type TemplateNodeWithPath = [TemplateSelectionElementNode, Path];

/**
 * Execution-ready derived representation of a selected template.
 */
export interface SelectedTemplateForRun {
	type: typeof KEY_TEMPLATE_SELECTION;
	bundleID: string;
	templateSlug: string;
	templateVersion: string;
	selectionID: string;

	// Final structures after applying local overrides
	template: PromptTemplate;
	blocks: MessageBlock[];
	variablesSchema: PromptVariable[];

	// Effective variable values for execution
	variableValues: Record<string, unknown>;

	// Requirements state
	requiredVariables: string[];
	requiredCount: number;

	// Convenience
	isReady: boolean;
}

function renderTemplateTextWithVariableValues(text: string, variableValues: Record<string, unknown>): string {
	return text.replace(TEMPLATE_PLACEHOLDER_RE, (_match, name: string) => {
		const value = variableValues[name];
		return value === undefined || value === null ? '' : (value as string);
	});
}

function getInstructionPromptPartFromSelection(selection: SelectedTemplateForRun): string {
	return selection.blocks
		.filter(block => block.role === PromptRoleEnum.System || block.role === PromptRoleEnum.Developer)
		.map(block => renderTemplateTextWithVariableValues(block.content, selection.variableValues).trim())
		.filter(Boolean)
		.join('\n\n');
}

export function getInstructionPromptPartsFromSelections(selections: SelectedTemplateForRun[]): string[] {
	return selections.map(getInstructionPromptPartFromSelection).filter(Boolean);
}

/**
 * Merge templateSnapshot with local overrides to produce effective template structures.
 */
export function computeEffectiveTemplate(el: TemplateSelectionElementNode): {
	template: PromptTemplate | undefined;
	blocks: MessageBlock[];
	variablesSchema: PromptVariable[];
} {
	const base = el.templateSnapshot;
	const blocks = el.overrides?.blocks ?? base?.blocks ?? [];
	const variablesSchema = el.overrides?.variables ?? base?.variables ?? [];

	return { template: base, blocks, variablesSchema };
}

function makeSelectedTemplateForRun(tsenode: TemplateSelectionElementNode): SelectedTemplateForRun {
	const { template, blocks, variablesSchema } = computeEffectiveTemplate(tsenode);

	const effTemplate: PromptTemplate =
		template ??
		({
			kind: PromptTemplateKind.Generic,
			id: '',
			displayName: tsenode.templateSlug,
			slug: tsenode.templateSlug,
			isEnabled: true,
			description: '',
			tags: [],
			blocks,
			variables: variablesSchema,
			isResolved: true,
			version: tsenode.templateVersion,
			createdAt: new Date().toISOString(),
			modifiedAt: new Date().toISOString(),
			isBuiltIn: false,
		} as PromptTemplate);

	const req = computeTemplateVarRequirements(variablesSchema, tsenode.variables);

	return {
		type: KEY_TEMPLATE_SELECTION,
		bundleID: tsenode.bundleID,
		templateSlug: tsenode.templateSlug,
		templateVersion: tsenode.templateVersion,
		selectionID: tsenode.selectionID,
		template: effTemplate,
		blocks,
		variablesSchema,

		variableValues: req.variableValues,
		requiredVariables: req.requiredVariables,
		requiredCount: req.requiredCount,

		isReady: req.requiredCount === 0,
	};
}

// Returns all user-facing blocks concatenated in template order.
// System/developer blocks are sent via systemPrompt, so only user blocks belong
// in the editor/message body.
export function getUserBlocksContent(el: TemplateSelectionElementNode): string {
	const { blocks } = computeEffectiveTemplate(el);
	return blocks
		.filter(block => block.role === PromptRoleEnum.User)
		.map(block => block.content)
		.filter(content => content.trim().length > 0)
		.join('\n\n');
}

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
