import { type KeyboardEvent, type MouseEvent, useMemo, useState } from 'react';

import { FiEdit2 } from 'react-icons/fi';

import { NodeApi } from 'platejs';
import type { PlateEditor, PlateElementProps } from 'platejs/react';

import { VarType } from '@/spec/prompt';

import { cssEscape } from '@/lib/text_utils';

import {
	dispatchTemplateVarsUpdated,
	useTemplateVarsUpdatedForSelection,
} from '@/chats/events/template_toolbar_vars_updated';
import {
	computeEffectiveTemplate,
	computeRequirements,
	effectiveVarValueLocal,
} from '@/chats/templates/template_processing';
import {
	KEY_TEMPLATE_SELECTION,
	KEY_TEMPLATE_VARIABLE,
	type TemplateSelectionElementNode,
	type TemplateVariableElementNode,
} from '@/chats/templates/template_spec';
import { EnumDropdownInline } from '@/chats/templates/template_variable_enum_dropdown';

function findTemplateNode(
	editor: PlateEditor,
	bundleID: string,
	templateSlug: string,
	templateVersion: string,
	selectionID: string
): [TemplateSelectionElementNode, any] | undefined {
	const els = NodeApi.elements(editor);
	for (const [el, path] of els) {
		if (el.type === KEY_TEMPLATE_SELECTION) {
			const node = el as unknown as TemplateSelectionElementNode;
			const nodeSelectionID = node.selectionID as string | undefined;
			const matchesById = selectionID && nodeSelectionID === selectionID;
			const matchesByTriple =
				node.bundleID === bundleID && node.templateSlug === templateSlug && node.templateVersion === templateVersion;
			if (matchesById || matchesByTriple) {
				return [node, path];
			}
		}
	}
	return undefined;
}

// Inline variable pill renderer with inline editing
export function TemplateVariableElement(props: PlateElementProps<any>) {
	const { element, attributes, children, editor } = props as any;
	const el = element as TemplateVariableElementNode;
	const plEditor = editor as PlateEditor;

	const tpl = findTemplateNode(plEditor, el.bundleID, el.templateSlug, el.templateVersion, el.selectionID);
	const [tsenode, tsPath] = tpl ?? [];
	const eff = tsenode ? computeEffectiveTemplate(tsenode) : undefined;
	const variablesSchema = eff?.variablesSchema ?? [];

	const varDef = variablesSchema.find(v => v.name === el.name);
	const isRequired = Boolean(varDef?.required);

	// Compute requirements for status color using effective preProcessors (overrides included)
	const req = tsenode
		? computeRequirements(variablesSchema, tsenode.variables)
		: { variableValues: {}, requiredVariables: [] as string[], requiredCount: 0 };

	const isMissing = isRequired && req.requiredVariables.includes(el.name);

	// Local inline edit state
	const [isEditing, setIsEditing] = useState(false);
	const [refreshTick, setRefreshTick] = useState(0);

	// Ensure badges re-render when variables updated from modal or elsewhere
	useTemplateVarsUpdatedForSelection(el.selectionID, () => {
		setRefreshTick(t => t + 1);
	});

	// Current effective value for display and starting edit
	const currentValue = useMemo(() => {
		if (!tsenode || !varDef) return undefined;
		const v = effectiveVarValueLocal(varDef, tsenode.variables ?? {});
		return v;
	}, [tsenode?.variables, varDef?.name, refreshTick]);
	// console.log(varDef, tsenode?.variables);
	// Commit helper
	function commitValue(next: unknown) {
		if (!tsenode || !tsPath) return;

		const nextVars = { ...(tsenode.variables ?? {}) };

		// Normalize empty to undefined (remove)
		const shouldUnset = next === undefined || next === null;
		if (shouldUnset) {
			// eslint-disable-next-line @typescript-eslint/no-dynamic-delete
			delete nextVars[el.name];
		} else {
			nextVars[el.name] = next;
		}

		plEditor.tf.setNodes(
			{
				variables: nextVars,
			},
			{ at: tsPath }
		);

		// Signal badges/toolbar for this selection only
		if (tsenode.selectionID) {
			dispatchTemplateVarsUpdated(tsenode.selectionID);
		}

		setIsEditing(false);
		// After committing, move to the next variable pill instead of jumping
		// the caret to the end of the template.
		focusNextVariablePill();
	}

	function cancelEdit() {
		setIsEditing(false);
		// Return focus to this pill
		requestAnimationFrame(() => {
			try {
				const selId = el.selectionID ? cssEscape(el.selectionID) : '';
				const nameEsc = cssEscape(el.name);
				const pill = document.querySelector<HTMLElement>(
					`span[data-template-variable]` +
						(selId ? `[data-selection-id="${selId}"]` : '') +
						`[data-var-name="${nameEsc}"]`
				);
				if (pill) pill.focus();
				else plEditor.tf.focus();
			} catch {
				plEditor.tf.focus();
			}
		});
	}

	// Move focus to the next variable pill in this selection.
	// If no required vars remain missing in this selection, focus the editor.
	function focusNextVariablePill() {
		const selId = el.selectionID as string | undefined;
		if (!selId) {
			plEditor.tf.focus();
			return;
		}

		requestAnimationFrame(() => {
			try {
				const escapedSel = cssEscape(selId);

				// All variable pills for this selection, in DOM order
				const chips = Array.from(
					document.querySelectorAll<HTMLElement>(`span[data-template-variable][data-selection-id="${escapedSel}"]`)
				);

				if (!chips.length) {
					plEditor.tf.focus();
					return;
				}

				// Are there ANY required vars still missing in THIS selection?
				const missingInSelection = chips.filter(chip => chip.dataset.state === 'required');

				if (missingInSelection.length === 0) {
					// Nothing left to fill here -> move back to editor
					plEditor.tf.focus();
					return;
				}

				// There is at least one missing var in this selection.
				// Prefer the next missing var after the current one; wrap to first missing if needed.
				const currentIndex = chips.findIndex(chip => chip.dataset.varName === el.name);
				const later = currentIndex >= 0 ? chips.slice(currentIndex + 1) : chips;

				const next = later.find(chip => chip.dataset.state === 'required') ?? missingInSelection[0]; // wrap to the first missing one

				if (next) next.focus();
				else plEditor.tf.focus();
			} catch {
				plEditor.tf.focus();
			}
		});
	}
	// Inline input renderer based on var type
	function InlineEditor() {
		const type = varDef?.type ?? VarType.String;

		const commonProps = {
			onKeyDown: (e: KeyboardEvent) => {
				e.stopPropagation();
				if (e.key === 'Escape') {
					e.preventDefault();
					cancelEdit();
				}
				if (e.key === 'Enter') {
					e.preventDefault();
					const target = e.target as HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement;
					if (type === VarType.Number) {
						const num = target.value.trim() === '' ? undefined : Number(target.value);
						commitValue(Number.isFinite(num as number) ? num : undefined);
					} else if (type === VarType.Boolean) {
						// no-op; toggles commit on change
					} else if (type === VarType.Enum) {
						commitValue(target.value || undefined);
					} else if (type === VarType.Date) {
						commitValue(target.value || undefined);
					} else {
						commitValue(target.value ?? '');
					}
				}
			},
			onMouseDown: (e: MouseEvent) => {
				// keep focus inside input
				e.stopPropagation();
			},
		};

		if (type === VarType.Boolean) {
			return (
				<label className="flex items-center gap-1">
					<input
						type="checkbox"
						className="toggle toggle-accent toggle-xs"
						checked={Boolean(currentValue)}
						aria-label={`Set ${el.name}`}
						onChange={e => {
							commitValue(e.target.checked);
						}}
						onBlur={() => {
							commitValue(Boolean(currentValue));
						}}
						{...commonProps}
					/>
				</label>
			);
		}

		if (type === VarType.Number) {
			return (
				<input
					autoFocus
					type="number"
					className="input input-ghost input-xs w-28 min-w-20 rounded-md bg-transparent"
					aria-label={`Set number for ${el.name}`}
					defaultValue={currentValue === undefined || currentValue === null ? '' : String(currentValue as number)}
					placeholder={varDef?.default !== undefined ? varDef.default : ''}
					onBlur={e => {
						const val = e.currentTarget.value.trim();
						commitValue(val === '' ? undefined : Number(val));
					}}
					{...commonProps}
				/>
			);
		}

		if (type === VarType.Date) {
			return (
				<input
					autoFocus
					type="date"
					className="input input-ghost input-xs w-36 min-w-28 bg-transparent"
					aria-label={`Pick date for ${el.name}`}
					defaultValue={currentValue ? (currentValue as string) : ''}
					onBlur={e => {
						commitValue(e.currentTarget.value || undefined);
					}}
					{...commonProps}
				/>
			);
		}

		// String
		return (
			<input
				autoFocus
				className="input input-ghost input-xs w-40 min-w-24 rounded-md bg-transparent"
				aria-label={`Set ${el.name}`}
				// eslint-disable-next-line @typescript-eslint/no-base-to-string
				defaultValue={currentValue === undefined || currentValue === null ? '' : String(currentValue)}
				maxLength={32}
				placeholder={varDef?.default !== undefined ? varDef.default : ''}
				onBlur={e => {
					commitValue(e.currentTarget.value);
				}}
				{...commonProps}
			/>
		);
	}

	// console.log(el.name, currentValue);
	return (
		<span
			{...attributes}
			contentEditable={false}
			tabIndex={0}
			role="button"
			aria-label={
				isEditing
					? `Editing variable ${el.name}`
					: isMissing
						? `Variable ${el.name} is required. Press Enter or Space to edit`
						: `Variable ${el.name}. Press Enter or Space to edit`
			}
			aria-required={isRequired || undefined}
			aria-invalid={isMissing || undefined}
			data-template-variable
			data-var-name={el.name}
			data-selection-id={el.selectionID}
			data-state={isMissing ? 'required' : 'ready'}
			className={`badge badge-sm gap-1 py-0 whitespace-nowrap select-none ${isMissing ? 'badge-warning' : 'badge-success'}`}
			title={isEditing ? `Editing: ${el.name}` : isMissing ? `Required: ${el.name}` : `Variable: ${el.name}`}
			onKeyDown={e => {
				// Enter toggles inline edit; Esc handled inside edit
				if ((e.key === 'Enter' || e.key === ' ') && !isEditing) {
					e.preventDefault();
					e.stopPropagation();
					setIsEditing(true);
				}
			}}
			onMouseDown={e => {
				// allow focusing pill without bubbling into editor selection changes and ensure the pill gets focus
				e.preventDefault();
				(e.currentTarget as HTMLElement).focus();
			}}
			onClick={() => {
				setIsEditing(true);
			}}
		>
			<span>
				{isEditing ? (
					<div className="flex items-center gap-1">
						{/* For enum type only, add a key to force a fresh instance when things change */}
						{varDef?.type === VarType.Enum ? (
							<EnumDropdownInline
								key={`enum-${el.selectionID}-${el.name}-${refreshTick}`}
								options={varDef?.enumValues ?? []}
								value={
									currentValue === undefined || currentValue === null || currentValue === ''
										? undefined
										: // eslint-disable-next-line @typescript-eslint/no-base-to-string
											String(currentValue)
								}
								onChange={val => {
									commitValue(val);
								}}
								withinSlate
								autoOpen
								onCancel={cancelEdit}
								size="xs"
								triggerClassName="btn btn-ghost btn-xs font-normal w-40 min-w-24 justify-between truncate bg-transparent"
								placeholder="-- select --"
								clearLabel="Clear"
							/>
						) : (
							<InlineEditor />
						)}
					</div>
				) : (
					<span className="flex items-center gap-1 font-mono text-xs">
						<span>{el.name}</span>
						{currentValue !== undefined && currentValue !== null && (
							<span className="ml-1">
								= <span>"{currentValue as string}"</span>
							</span>
						)}

						<FiEdit2 size={10} className="opacity-70" />
					</span>
				)}
			</span>
			{children}
		</span>
	);
}

/**
 * Build Slate inline children from a plain text that may include {{varName}} tokens.
 * Unknown variables are left as plain text.
 */
export function buildUserInlineChildrenFromText(tsenode: TemplateSelectionElementNode, text: string): any[] {
	const { variablesSchema } = computeEffectiveTemplate(tsenode);
	const known = new Set(variablesSchema.map(v => v.name));
	const selectionID: string | undefined = tsenode.selectionID;

	const result: any[] = [];
	const re = /\{\{\s*([a-zA-Z0-9_]+)\s*\}\}/g;
	let idx = 0;
	let m: RegExpExecArray | null;
	while ((m = re.exec(text)) !== null) {
		const pre = text.slice(idx, m.index);

		if (pre) result.push({ text: pre, ownerSelectionID: selectionID });

		const varName = m[1];
		if (known.has(varName)) {
			const node: TemplateVariableElementNode = {
				type: KEY_TEMPLATE_VARIABLE,
				bundleID: tsenode.bundleID,
				templateSlug: tsenode.templateSlug,
				templateVersion: tsenode.templateVersion,
				selectionID: selectionID,
				name: varName,
				required: variablesSchema.find(v => v.name === varName)?.required ?? false,
				children: [{ text: '' }],
			};
			result.push(node as any);
		} else {
			// unknown variable -> keep as literal
			result.push({ text: m[0] });
		}

		idx = m.index + m[0].length;
	}
	const tail = text.slice(idx);
	if (tail) result.push({ text: tail, ownerSelectionID: selectionID });

	if (result.length === 0) result.push({ text: '' });
	return result;
}
