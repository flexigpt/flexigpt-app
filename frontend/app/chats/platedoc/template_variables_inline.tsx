import { type KeyboardEvent, type MouseEvent, useMemo, useState } from 'react';

import { FiEdit2 } from 'react-icons/fi';

import { NodeApi, type Path } from 'platejs';
import type { PlateEditor, PlateElementProps } from 'platejs/react';

import { VarType } from '@/spec/prompt';

import { omitManyKeys } from '@/lib/obj_utils';
import { cssEscape } from '@/lib/text_utils';

import {
	dispatchTemplateVarsUpdated,
	useTemplateVarsUpdatedForSelection,
} from '@/chats/events/template_toolbar_vars_updated';
import {
	KEY_TEMPLATE_SELECTION,
	KEY_TEMPLATE_VARIABLE,
	type TemplateSelectionElementNode,
	type TemplateVariableElementNode,
} from '@/chats/platedoc/nodes';
import {
	computeEffectiveTemplate,
	computeRequirements,
	effectiveVarValueLocal,
} from '@/chats/templates/template_processing';
import { EnumDropdownInline } from '@/chats/templates/template_variable_enum_dropdown';

type TemplateVariableDef = ReturnType<typeof computeEffectiveTemplate>['variablesSchema'][number];
type InlineEditorKeyTarget = HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement;

type TemplateInlineTextNode = {
	text: string;
	ownerSelectionID?: string;
};

type TemplateInlineChild = TemplateInlineTextNode | TemplateVariableElementNode;

const EMPTY_REQUIREMENTS: {
	variableValues: Record<string, unknown>;
	requiredVariables: string[];
	requiredCount: number;
} = {
	variableValues: {},
	requiredVariables: [],
	requiredCount: 0,
};

function formatTemplateVarValue(value: unknown): string {
	return value === undefined || value === null ? '' : (value as string);
}

function buildVariablePillSelector(selectionID?: string, varName?: string): string {
	let selector = 'span[data-template-variable]';

	if (selectionID) {
		selector += `[data-selection-id="${cssEscape(selectionID)}"]`;
	}

	if (varName) {
		selector += `[data-var-name="${cssEscape(varName)}"]`;
	}

	return selector;
}

function getVariablePills(selectionID?: string): HTMLElement[] {
	if (!selectionID) return [];
	return Array.from(document.querySelectorAll<HTMLElement>(buildVariablePillSelector(selectionID)));
}

function getVariablePill(selectionID?: string, varName?: string): HTMLElement | null {
	if (!varName) return null;
	return document.querySelector<HTMLElement>(buildVariablePillSelector(selectionID, varName));
}

function findTemplateNode(
	editor: PlateEditor,
	bundleID: string,
	templateSlug: string,
	templateVersion: string,
	selectionID: string
): [TemplateSelectionElementNode, Path] | undefined {
	const els = NodeApi.elements(editor);

	for (const [el, path] of els) {
		if (el.type === KEY_TEMPLATE_SELECTION) {
			const node = el as unknown as TemplateSelectionElementNode;
			const nodeSelectionID = node.selectionID as string | undefined;

			const matchesById = Boolean(selectionID) && nodeSelectionID === selectionID;
			const matchesByTriple =
				node.bundleID === bundleID && node.templateSlug === templateSlug && node.templateVersion === templateVersion;

			if (matchesById || matchesByTriple) {
				return [node, path];
			}
		}
	}

	return undefined;
}

type TemplateVariableInlineEditorProps = {
	el: TemplateVariableElementNode;
	varDef: TemplateVariableDef | undefined;
	currentValue: unknown;
	commitValue: (next: unknown) => void;
	cancelEdit: () => void;
};

function TemplateVariableInlineEditor({
	el,
	varDef,
	currentValue,
	commitValue,
	cancelEdit,
}: TemplateVariableInlineEditorProps) {
	const type = varDef?.type ?? VarType.String;

	const commonProps = {
		onKeyDown: (e: KeyboardEvent) => {
			e.stopPropagation();

			if (e.key === 'Escape') {
				e.preventDefault();
				cancelEdit();
				return;
			}

			if (e.key === 'Enter') {
				e.preventDefault();

				const target = e.target as InlineEditorKeyTarget;

				if (type === VarType.Number) {
					const num = target.value.trim() === '' ? undefined : Number(target.value);
					commitValue(Number.isFinite(num as number) ? num : undefined);
				} else if (type === VarType.Boolean) {
					// no-op; checkbox commits on change / blur
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
					onBlur={e => {
						commitValue(e.currentTarget.checked);
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
				placeholder={varDef?.default !== undefined ? formatTemplateVarValue(varDef.default) : ''}
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

	return (
		<input
			autoFocus
			className="input input-ghost input-xs w-40 min-w-24 rounded-md bg-transparent"
			aria-label={`Set ${el.name}`}
			defaultValue={currentValue === undefined || currentValue === null ? '' : (currentValue as string)}
			maxLength={32}
			placeholder={varDef?.default !== undefined ? formatTemplateVarValue(varDef.default) : ''}
			onBlur={e => {
				commitValue(e.currentTarget.value);
			}}
			{...commonProps}
		/>
	);
}

// Inline variable pill renderer with inline editing
export function TemplateVariableElement(props: PlateElementProps<any>) {
	const { element, attributes, children, editor } = props;
	const el = element as TemplateVariableElementNode;

	const tpl = findTemplateNode(editor, el.bundleID, el.templateSlug, el.templateVersion, el.selectionID);
	const [tsenode, tsPath] = tpl ?? [];

	const eff = tsenode ? computeEffectiveTemplate(tsenode) : undefined;
	const variablesSchema = eff?.variablesSchema ?? [];

	const varDef = variablesSchema.find(v => v.name === el.name);
	const isRequired = Boolean(varDef?.required);

	const req = tsenode ? computeRequirements(variablesSchema, tsenode.variables) : EMPTY_REQUIREMENTS;
	const isMissing = isRequired && req.requiredVariables.includes(el.name);

	const [isEditing, setIsEditing] = useState(false);
	const [refreshTick, setRefreshTick] = useState(0);

	useTemplateVarsUpdatedForSelection(el.selectionID, () => {
		setRefreshTick(t => t + 1);
	});

	const currentValue = useMemo(() => {
		if (!tsenode || !varDef) return undefined;
		return effectiveVarValueLocal(varDef, tsenode.variables ?? {});
	}, [tsenode, varDef]);

	const hasDisplayValue = currentValue !== undefined && currentValue !== null;
	const displayValue = formatTemplateVarValue(currentValue);

	function focusNextVariablePill() {
		const selectionID = el.selectionID as string | undefined;
		if (!selectionID) return;

		requestAnimationFrame(() => {
			try {
				const chips = getVariablePills(selectionID);

				if (!chips.length) {
					return;
				}

				const missingInSelection = chips.filter(chip => chip.dataset.state === 'required');

				if (missingInSelection.length === 0) {
					return;
				}

				const currentIndex = chips.findIndex(chip => chip.dataset.varName === el.name);
				const later = currentIndex >= 0 ? chips.slice(currentIndex + 1) : chips;

				const next = later.find(chip => chip.dataset.state === 'required') ?? missingInSelection[0];

				if (next) {
					next.focus();
				}
			} catch {
				// Ok.
			}
		});
	}

	function commitValue(next: unknown) {
		if (!tsenode || !tsPath) return;

		let nextVars = { ...(tsenode.variables ?? {}) };

		const shouldUnset = next === undefined || next === null;
		if (shouldUnset) {
			nextVars = omitManyKeys(nextVars, [el.name]);
		} else {
			nextVars[el.name] = next;
		}

		editor.tf.setNodes(
			{
				variables: nextVars,
			},
			{ at: tsPath }
		);

		if (tsenode.selectionID) {
			dispatchTemplateVarsUpdated(tsenode.selectionID);
		}

		setIsEditing(false);
		focusNextVariablePill();
	}

	function cancelEdit() {
		setIsEditing(false);

		requestAnimationFrame(() => {
			try {
				const pill = getVariablePill(el.selectionID, el.name);
				if (pill) {
					pill.focus();
				}
			} catch {
				// Ok.
			}
		});
	}

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
				if ((e.key === 'Enter' || e.key === ' ') && !isEditing) {
					e.preventDefault();
					e.stopPropagation();
					setIsEditing(true);
				}
			}}
			onMouseDown={e => {
				// allow focusing pill without bubbling into editor selection changes
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
						{varDef?.type === VarType.Enum ? (
							<EnumDropdownInline
								key={`enum-${el.selectionID}-${el.name}-${refreshTick}`}
								options={varDef?.enumValues ?? []}
								value={hasDisplayValue && displayValue !== '' ? displayValue : undefined}
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
							<TemplateVariableInlineEditor
								el={el}
								varDef={varDef}
								currentValue={currentValue}
								commitValue={commitValue}
								cancelEdit={cancelEdit}
							/>
						)}
					</div>
				) : (
					<span className="flex items-center gap-1 font-mono text-xs">
						<span>{el.name}</span>
						{hasDisplayValue && (
							<span className="ml-1">
								= <span>"{displayValue}"</span>
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
export function buildUserInlineChildrenFromText(
	tsenode: TemplateSelectionElementNode,
	text: string
): TemplateInlineChild[] {
	const { variablesSchema } = computeEffectiveTemplate(tsenode);
	const variablesByName = new Map(variablesSchema.map(v => [v.name, v] as const));
	const selectionID = tsenode.selectionID;

	const result: TemplateInlineChild[] = [];
	const re = /\{\{\s*([a-zA-Z0-9_]+)\s*\}\}/g;

	let idx = 0;
	let m: RegExpExecArray | null;

	while ((m = re.exec(text)) !== null) {
		const pre = text.slice(idx, m.index);

		if (pre) {
			result.push({ text: pre, ownerSelectionID: selectionID });
		}

		const varName = m[1];
		const matchedVar = variablesByName.get(varName);

		if (matchedVar) {
			const node: TemplateVariableElementNode = {
				type: KEY_TEMPLATE_VARIABLE,
				bundleID: tsenode.bundleID,
				templateSlug: tsenode.templateSlug,
				templateVersion: tsenode.templateVersion,
				selectionID,
				name: varName,
				required: matchedVar.required ?? false,
				children: [{ text: '' }],
			};

			result.push(node);
		} else {
			// unknown variable -> keep as literal
			result.push({ text: m[0], ownerSelectionID: selectionID });
		}

		idx = m.index + m[0].length;
	}

	const tail = text.slice(idx);
	if (tail) {
		result.push({ text: tail, ownerSelectionID: selectionID });
	}

	if (result.length === 0) {
		result.push({ text: '', ownerSelectionID: selectionID });
	}

	return result;
}
