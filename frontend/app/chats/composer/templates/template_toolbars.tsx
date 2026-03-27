import { useState } from 'react';

import { NodeApi, type Path, type TElement, type TNode } from 'platejs';
import { type PlateEditor, useEditorRef } from 'platejs/react';

import { type PromptVariable } from '@/spec/prompt';

import { comparePathDeepestFirst } from '@/lib/path_utils';

import {
	KEY_TEMPLATE_SELECTION,
	KEY_TEMPLATE_VARIABLE,
	type TemplateSelectionElementNode,
} from '@/chats/composer/platedoc/nodes';
import {
	computeEffectiveTemplate,
	getTemplateNodesWithPath,
	getTemplateSelections,
} from '@/chats/composer/platedoc/template_document_ops';
import { TemplateEditModal } from '@/chats/composer/templates/template_edit_modal';
import { TemplateFixedToolbar } from '@/chats/composer/templates/template_toolbar_fixed';
import { useTemplateFlashEvent } from '@/chats/composer/templates/use_template_flash_event';
import { dispatchTemplateVarsUpdated } from '@/chats/composer/templates/use_template_toolbar_vars_updated_event';
import { effectiveVarValueLocal } from '@/prompts/lib/prompt_template_var_utils';

type TplKey = string; // path-based unique key

function pathKey(path: Path | undefined): TplKey {
	return Array.isArray(path) ? path.join('.') : (path ?? '');
}

function replaceVariablesForSelectionWithText(
	editor: PlateEditor,
	bundleID: string,
	templateSlug: string,
	templateVersion: string,
	pathOfSelection?: Path
) {
	// Replace all variable chips belonging to this selection with their current value (or {{name}} if empty)
	// Then remove the selection chip itself.
	// Note: we do not flatten other templates.
	const varEntries: Array<[TElement, Path]> = [];
	// Find the matching selection node to read variable values and to remove it at the end
	let tsNodeWithPath: [TemplateSelectionElementNode, Path] | undefined;

	if (pathOfSelection) {
		const got = NodeApi.get(editor, pathOfSelection);
		if (got && got.type === KEY_TEMPLATE_SELECTION) {
			tsNodeWithPath = [got as unknown as TemplateSelectionElementNode, pathOfSelection];
		}
	}
	if (!tsNodeWithPath) {
		for (const [el, p] of NodeApi.elements(editor)) {
			if (el.type === KEY_TEMPLATE_SELECTION) {
				const n = el as unknown as TemplateSelectionElementNode;
				if (n.bundleID === bundleID && n.templateSlug === templateSlug && n.templateVersion === templateVersion) {
					tsNodeWithPath = [n, p];
					break;
				}
			}
		}
	}

	const tsNode = tsNodeWithPath?.[0];
	const EMPTY_VARS: Record<string, unknown> = Object.freeze({}) as Record<string, unknown>;
	const vars: Record<string, unknown> = tsNode?.variables ?? EMPTY_VARS;

	// Precompute variable definitions so collapse‑to‑text uses the same
	// precedence as sending (local override → static → default).
	let varDefByName: Map<string, PromptVariable> | undefined;
	if (tsNode) {
		const { variablesSchema } = computeEffectiveTemplate(tsNode);
		varDefByName = new Map(variablesSchema.map(v => [v.name, v] as const));
	}
	const selectionID: string | undefined = tsNode?.selectionID;

	// Collect variable chips for this specific selection instance (prefer selectionId)
	for (const [el, p] of NodeApi.elements(editor)) {
		if (el.type === KEY_TEMPLATE_VARIABLE) {
			const sameInstance = selectionID
				? el.selectionID === selectionID
				: el.bundleID === bundleID && el.templateSlug === templateSlug && el.templateVersion === templateVersion;
			if (sameInstance) {
				varEntries.push([el, p]);
			}
		}
	}

	// Replace vars from deepest path to shallow to keep paths valid while mutating
	varEntries
		.sort((a, b) => comparePathDeepestFirst(a[1] as number[], b[1] as number[]))
		.forEach(([, path]) => {
			const nentry = NodeApi.get(editor, path);
			if (!nentry) return;

			const name: string = (nentry as TElement).name as string;

			let value: unknown = vars[name];
			const def = varDefByName?.get(name);
			// Reuse the same "effective value" logic used when sending so the
			// collapsed text matches model input.
			if (def) {
				value = effectiveVarValueLocal(def, vars);
			}

			// eslint-disable-next-line @typescript-eslint/no-base-to-string
			const text = value !== undefined && value !== null && value !== '' ? String(value) : `{{${name}}}`;
			editor.tf.removeNodes({ at: path });
			editor.tf.insertNodes({ text }, { at: path });
		});

	// IMPORTANT: Do NOT remove plain text nodes; they are already plain text.
	// Only remove the selection chip itself so the toolbar disappears.
	if (tsNodeWithPath) {
		editor.tf.removeNodes({ at: tsNodeWithPath[1] });
	}
}

function removeSelection(
	editor: PlateEditor,
	bundleID: string,
	templateSlug: string,
	templateVersion: string,
	pathOfSelection?: Path
) {
	editor.tf.withoutNormalizing(() => {
		// Identify target selection (get selectionId for precise removal)
		let targetSelection: [TemplateSelectionElementNode, Path] | undefined = undefined;
		if (pathOfSelection) {
			const got = NodeApi.get(editor, pathOfSelection);
			if (got && got.type === KEY_TEMPLATE_SELECTION) {
				targetSelection = [got as unknown as TemplateSelectionElementNode, pathOfSelection];
			}
		}
		if (!targetSelection) {
			for (const [el, p] of NodeApi.elements(editor)) {
				if (
					el.type === KEY_TEMPLATE_SELECTION &&
					el.bundleID === bundleID &&
					el.templateSlug === templateSlug &&
					el.templateVersion === templateVersion
				) {
					targetSelection = [el as unknown as TemplateSelectionElementNode, p];
					break;
				}
			}
		}
		const selectionID: string | undefined = targetSelection?.[0]?.selectionID;

		// Remove variable chips for this selection
		const entries: Array<[TElement, Path]> = [];
		const it = NodeApi.elements(editor);
		for (const [el, p] of it) {
			if (
				el.type === KEY_TEMPLATE_VARIABLE &&
				(selectionID ? el.selectionID === selectionID : true) &&
				el.bundleID === bundleID &&
				el.templateSlug === templateSlug &&
				el.templateVersion === templateVersion
			) {
				entries.push([el, p]);
			}
		}
		const textEntries: Array<[TNode, Path]> = [];
		for (const [nnode, p] of NodeApi.nodes(editor)) {
			if (typeof nnode === 'object' && 'text' in nnode) {
				const ownerId = nnode.ownerSelectionID as string | undefined;
				if (ownerId && selectionID && ownerId === selectionID) {
					textEntries.push([nnode, p]);
				}
			}
		}

		// Remove deepest first
		[...entries, ...textEntries]
			.sort((a, b) => comparePathDeepestFirst(a[1] as number[], b[1] as number[]))
			.forEach(([, p]) => {
				editor.tf.removeNodes({ at: p });
			});

		// Remove the selection chip
		if (targetSelection) {
			editor.tf.removeNodes({ at: targetSelection[1] });
		}
	});
}

export function TemplateToolbars() {
	const editor = useEditorRef() as PlateEditor;
	const flashAll = useTemplateFlashEvent();

	// Build a stable mapping selection <-> its node+path by document order.
	const selections = getTemplateSelections(editor);
	const nodesWithPath = getTemplateNodesWithPath(editor);

	// There might be a mismatch if something unusual happened; pair by nearest match
	const used = new Set<string>();
	const items = selections.map(sel => {
		const entry = nodesWithPath.find(([n, p]) => {
			const ok =
				(sel.selectionID
					? n.selectionID === sel.selectionID
					: n.bundleID === sel.bundleID &&
						n.templateSlug === sel.templateSlug &&
						n.templateVersion === sel.templateVersion) && !used.has(pathKey(p));
			if (ok) used.add(pathKey(p));
			return ok;
		});
		const id = entry ? pathKey(entry[1]) : `${sel.bundleID}:${sel.templateSlug}:${sel.templateVersion}`;
		return { id, sel, nodeWithPath: entry };
	});

	// Single-open modal keyed by selection path
	const [openId, setOpenId] = useState<string | null>(null);

	if (items.length === 0) return null;

	return (
		<div className="border-base-300 sticky top-0 left-0 w-full min-w-0 border-b">
			{items.map(({ id, sel, nodeWithPath }) => {
				const [tsNode, tsPath] = nodeWithPath ?? [];
				const flashing = flashAll;

				return (
					<div key={id} className="w-full">
						<TemplateFixedToolbar
							selection={sel}
							flashing={flashing}
							onOpenModal={() => {
								setOpenId(id);
							}}
							onRemove={() => {
								removeSelection(editor, sel.bundleID, sel.templateSlug, sel.templateVersion, tsPath);
							}}
							onFlatten={() => {
								editor.tf.withoutNormalizing(() => {
									replaceVariablesForSelectionWithText(
										editor,
										sel.bundleID,
										sel.templateSlug,
										sel.templateVersion,
										tsPath
									);
								});

								// Ensure any badges re-render if they were visible briefly.
								if (nodeWithPath?.[0]?.selectionID) {
									dispatchTemplateVarsUpdated(nodeWithPath[0].selectionID);
								}
							}}
						/>

						{tsNode && tsPath ? (
							<TemplateEditModal
								open={openId === id}
								onClose={() => {
									setOpenId(null);
								}}
								tsenode={tsNode}
								editor={editor}
								path={tsPath}
							/>
						) : null}
					</div>
				);
			})}
		</div>
	);
}
