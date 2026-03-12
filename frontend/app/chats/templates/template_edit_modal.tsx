import type { SubmitEventHandler } from 'react';
import { useEffect, useId, useRef, useState } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle, FiHelpCircle, FiX } from 'react-icons/fi';

import type { PlateEditor } from 'platejs/react';

import { type PromptVariable, VarSource, VarType } from '@/spec/prompt';

import { dispatchTemplateVarsUpdated } from '@/chats/events/template_toolbar_vars_updated';
import {
	computeEffectiveTemplate,
	computeRequirements,
	effectiveVarValueLocal,
} from '@/chats/templates/template_processing';
import { type TemplateSelectionElementNode } from '@/chats/templates/template_spec';
import { EnumDropdownInline } from '@/chats/templates/template_variable_enum_dropdown';

export function TemplateEditModal({
	open,
	onClose,
	tsenode,
	editor,
	path,
}: {
	open: boolean;
	onClose: () => void;
	tsenode: TemplateSelectionElementNode;
	editor: PlateEditor;
	path: any;
}) {
	// Compute current effective template for rendering
	const { template, blocks, variablesSchema } = computeEffectiveTemplate(tsenode);

	// Local form state
	const [displayName, setDisplayName] = useState<string>(
		tsenode.overrides?.displayName ?? template?.displayName ?? tsenode.templateSlug
	);
	const [description, setDescription] = useState<string>(tsenode.overrides?.description ?? template?.description ?? '');
	const [tags, setTags] = useState<string>((tsenode.overrides?.tags ?? template?.tags ?? []).join(', '));

	// Editable message blocks (content)
	const [blockEdits, setBlockEdits] = useState(blocks);

	// Variable values
	const [varValues, setVarValues] = useState<Record<string, unknown>>(() => {
		const vals: Record<string, unknown> = { ...tsenode.variables };
		for (const v of variablesSchema) {
			const val = effectiveVarValueLocal(v, tsenode.variables);
			if (val !== undefined) vals[v.name] = val;
		}
		return vals;
	});

	const dialogRef = useRef<HTMLDialogElement | null>(null);

	// Rehydrate form when opening or when the node reference changes
	useEffect(() => {
		if (!open) return;

		// Recompute effective template inside the effect to avoid depending on changing references
		const eff = computeEffectiveTemplate(tsenode);
		const t = eff.template;
		const blks = eff.blocks;
		const varsSchema = eff.variablesSchema;

		setDisplayName(tsenode.overrides?.displayName ?? t?.displayName ?? tsenode.templateSlug);
		setDescription(tsenode.overrides?.description ?? t?.description ?? '');
		setTags((tsenode.overrides?.tags ?? t?.tags ?? []).join(', '));
		setBlockEdits(blks);

		const vals: Record<string, unknown> = { ...tsenode.variables };
		for (const v of varsSchema) {
			const val = effectiveVarValueLocal(v, tsenode.variables);
			if (val !== undefined) vals[v.name] = val;
		}
		setVarValues(vals);
	}, [open, tsenode]);

	// Sync React `open` prop with the native <dialog> lifecycle, like UrlAttachmentModal
	useEffect(() => {
		if (!open) return;

		const dialog = dialogRef.current;
		if (!dialog) return;

		if (!dialog.open) {
			dialog.showModal();
		}

		return () => {
			// If unmounted while still open, ensure we close the dialog
			if (dialog.open) {
				dialog.close();
			}
		};
	}, [open]);

	// Keep parent `open` in sync with native dialog closing (ESC, programmatic close, etc.)
	const handleDialogClose = () => {
		onClose();
	};

	function saveAndClose() {
		const nextOverrides = {
			...tsenode.overrides,
			displayName,
			description,
			tags: tags
				.split(',')
				.map(s => s.trim())
				.filter(Boolean),
			blocks: blockEdits,
		};

		// Persist changes into the editor node
		editor.tf.setNodes(
			{
				overrides: nextOverrides,
				variables: varValues,
			},
			{ at: path }
		);

		if (tsenode.selectionID) {
			dispatchTemplateVarsUpdated(tsenode.selectionID);
		}

		// Close the dialog; this will trigger handleDialogClose -> parent onClose().
		dialogRef.current?.close();
	}

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = e => {
		e.preventDefault();
		e.stopPropagation();
		saveAndClose();
	};

	const req = computeRequirements(variablesSchema, varValues);

	if (!open) return null;

	return createPortal(
		<dialog
			ref={dialogRef}
			className="modal"
			onClose={handleDialogClose}
			onCancel={e => {
				// Form mode: do NOT allow Esc to close.
				e.preventDefault();
			}}
		>
			<div className="modal-box bg-base-200 max-h-[80vh] max-w-[80vw] overflow-hidden rounded-2xl p-0">
				<div className="max-h-[80vh] overflow-y-auto p-6">
					{/* Header */}
					<div className="mb-4 flex items-center justify-between gap-2">
						<div className="flex items-center gap-2">
							<h3 className="text-lg font-bold">Edit Template</h3>
							<span className="badge badge-neutral">{displayName}</span>
						</div>
						<button
							type="button"
							className="btn btn-sm btn-circle bg-base-300"
							onClick={() => dialogRef.current?.close()}
							aria-label="Close"
						>
							<FiX size={12} />
						</button>
					</div>

					<form
						noValidate
						onSubmit={handleSubmit}
						className="space-y-6"
						onKeyDownCapture={e => {
							// Prevent outer editor form shortcuts/submit
							e.stopPropagation();
						}}
						onKeyUpCapture={e => {
							e.stopPropagation();
						}}
					>
						{/* Overview */}
						<section>
							<h4 className="text-base-content/70 mb-3 text-sm font-semibold tracking-wide uppercase">Overview</h4>
							<div className="space-y-3">
								<div className="grid grid-cols-12 items-center gap-3">
									<label className="label col-span-12 md:col-span-4">
										<span className="label-text text-sm">Display Name (local)</span>
										<span
											className="label-text-alt tooltip tooltip-right"
											data-tip="Local override; visible only here."
										>
											<FiHelpCircle size={12} />
										</span>
									</label>
									<div className="col-span-12 md:col-span-8">
										<input
											className="input input-bordered input-sm w-full rounded-xl"
											value={displayName}
											onChange={e => {
												setDisplayName(e.target.value);
											}}
											placeholder={template?.displayName ?? tsenode.templateSlug}
											spellCheck="false"
										/>
									</div>
								</div>

								<div className="grid grid-cols-12 items-center gap-3">
									<label className="label col-span-12 md:col-span-4">
										<span className="label-text text-sm">Tags</span>
										<span
											className="label-text-alt tooltip tooltip-right"
											data-tip="Comma-separated tags used for filtering."
										>
											<FiHelpCircle size={12} />
										</span>
									</label>
									<div className="col-span-12 md:col-span-8">
										<input
											className="input input-bordered input-sm w-full rounded-xl"
											value={tags}
											onChange={e => {
												setTags(e.target.value);
											}}
											placeholder="e.g. brainstorm, draft, review"
											spellCheck="false"
										/>
									</div>
								</div>

								<div className="grid grid-cols-12 items-start gap-3">
									<label className="label col-span-12 md:col-span-4">
										<span className="label-text text-sm">Description (local)</span>
										<span
											className="label-text-alt tooltip tooltip-right"
											data-tip="Local description for your reference."
										>
											<FiHelpCircle size={12} />
										</span>
									</label>
									<div className="col-span-12 md:col-span-8">
										<textarea
											className="textarea textarea-bordered w-full rounded-xl"
											value={description}
											onChange={e => {
												setDescription(e.target.value);
											}}
											placeholder={template?.description ?? 'Describe how this template should be used...'}
										/>
									</div>
								</div>
							</div>
						</section>

						<div className="divider before:bg-base-300 after:bg-base-300 my-0" />

						{/* Variables */}
						<section>
							<div className="mb-3 flex items-center justify-between">
								<h4 className="text-base-content/70 text-sm font-semibold tracking-wide uppercase">Variables</h4>
								{req.requiredCount > 0 ? (
									<div className="text-warning flex items-center gap-2 text-sm">
										<FiAlertCircle size={14} />
										<span>Required remaining: {req.requiredVariables.join(', ')}</span>
									</div>
								) : (
									<div className="badge badge-success badge-outline">All required variables provided</div>
								)}
							</div>

							<div className="space-y-3">
								{variablesSchema.length === 0 && (
									<div className="text-sm opacity-70">No variables defined for this template.</div>
								)}
								{variablesSchema.map(v => (
									<VariableEditorRow
										key={v.name}
										varDef={v}
										value={varValues[v.name]}
										onChange={val => {
											setVarValues(s => ({ ...s, [v.name]: val }));
										}}
									/>
								))}
							</div>
						</section>

						<div className="divider before:bg-base-300 after:bg-base-300 my-0" />

						{/* Blocks */}
						<section>
							<h4 className="text-base-content/70 mb-3 text-sm font-semibold tracking-wide uppercase">
								Blocks (local override)
							</h4>
							<div className="space-y-3">
								{blockEdits.map((b, idx) => (
									<div key={b.id} className="rounded-xl border p-3">
										<div className="mb-2 flex items-center gap-2 text-sm opacity-70">
											<span className="badge badge-outline">{b.role}</span>
											<span className="opacity-60">#{idx + 1}</span>
										</div>
										<textarea
											className="textarea textarea-bordered min-h-32 w-full rounded-xl"
											value={b.content}
											onChange={e => {
												setBlockEdits(arr => {
													const next = [...arr];
													next[idx] = { ...next[idx], content: e.target.value };
													return next;
												});
											}}
										/>
									</div>
								))}
							</div>
						</section>

						{/* Footer */}
						<div className="modal-action">
							<button type="button" className="btn bg-base-300 rounded-xl" onClick={() => dialogRef.current?.close()}>
								Cancel
							</button>
							<button type="submit" className="btn btn-primary rounded-xl">
								Save
							</button>
						</div>
					</form>
				</div>
			</div>

			{/* NOTE: no modal-backdrop here: backdrop click should NOT close this modal; ESC still works via native <dialog>. */}
		</dialog>,
		document.body
	);
}

function VariableEditorRow({
	varDef,
	value,
	onChange,
}: {
	varDef: PromptVariable;
	value: unknown;
	onChange: (val: unknown) => void;
}) {
	const id = useId();
	const label = `${varDef.name}${varDef.required ? ' *' : ''}`;
	const sourceText =
		varDef.source === VarSource.Static
			? `Static${varDef.staticVal ? `: ${varDef.staticVal}` : ''}`
			: `User${varDef.default ? `, default: ${varDef.default}` : ''}`;
	const help = varDef.description ? `${varDef.description} (${sourceText})` : sourceText;

	const isDisabled = varDef.source === VarSource.Static;
	const commonHelp = (
		<div className="label">
			<span className="label-text-alt text-xs opacity-70">
				{help}
				{varDef.type === VarType.Enum && varDef.enumValues?.length ? ` | options: ${varDef.enumValues.join(', ')}` : ''}
			</span>
		</div>
	);

	const labelCol = (
		<label htmlFor={id} className="label col-span-12 md:col-span-4">
			<span className="label-text text-sm">{label}</span>
			<span className="label-text-alt tooltip tooltip-right" data-tip={help}>
				<FiHelpCircle size={12} />
			</span>
		</label>
	);

	switch (varDef.type) {
		case VarType.Boolean:
			return (
				<div className="grid grid-cols-12 items-center gap-3">
					{labelCol}
					<div className="col-span-12 md:col-span-8">
						<input
							id={id}
							type="checkbox"
							className="toggle toggle-accent"
							checked={Boolean(value)}
							disabled={isDisabled}
							onChange={e => {
								onChange(e.target.checked);
							}}
						/>
						{commonHelp}
					</div>
				</div>
			);

		case VarType.Number:
			return (
				<div className="grid grid-cols-12 items-center gap-3">
					{labelCol}
					<div className="col-span-12 md:col-span-8">
						<input
							id={id}
							type="number"
							className="input input-bordered input-sm w-full rounded-xl"
							value={value === undefined || value === null ? '' : (value as number).toString()}
							disabled={isDisabled}
							onChange={e => {
								const v = e.target.value;
								onChange(v === '' ? undefined : Number(v));
							}}
							placeholder={varDef.default !== undefined ? varDef.default : ''}
						/>
						{commonHelp}
					</div>
				</div>
			);

		case VarType.Enum:
			return (
				<div className="grid grid-cols-12 items-center gap-3">
					{labelCol}
					<div className="col-span-12 md:col-span-8">
						<EnumDropdownInline
							options={varDef.enumValues ?? []}
							// eslint-disable-next-line @typescript-eslint/no-base-to-string
							value={value === undefined || value === null ? undefined : String(value)}
							onChange={val => {
								onChange(val);
							}}
							disabled={isDisabled}
							size="sm"
							triggerClassName="btn btn-ghost btn-sm w-full justify-between overflow-hidden"
							placeholder="-- select --"
							clearLabel="Clear"
						/>
						{commonHelp}
					</div>
				</div>
			);

		case VarType.Date:
			return (
				<div className="grid grid-cols-12 items-center gap-3">
					{labelCol}
					<div className="col-span-12 md:col-span-8">
						<input
							id={id}
							type="date"
							className="input input-bordered input-sm w-full rounded-xl"
							value={value ? (value as string) : ''}
							disabled={isDisabled}
							onChange={e => {
								onChange(e.target.value || undefined);
							}}
						/>
						{commonHelp}
					</div>
				</div>
			);

		case VarType.String:
		default:
			return (
				<div className="grid grid-cols-12 items-center gap-3">
					{labelCol}
					<div className="col-span-12 md:col-span-8">
						<input
							id={id}
							className="input input-bordered input-sm w-full rounded-xl"
							value={value === undefined || value === null ? '' : (value as string)}
							disabled={isDisabled}
							onChange={e => {
								onChange(e.target.value || undefined);
							}}
							placeholder={varDef.default !== undefined ? varDef.default : ''}
						/>
						{commonHelp}
					</div>
				</div>
			);
	}
}
