import { type ChangeEvent, type SubmitEventHandler, useEffect, useMemo, useRef, useState } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle, FiHelpCircle, FiPlus, FiTrash2, FiX } from 'react-icons/fi';

import {
	type MessageBlock,
	PromptRoleEnum,
	type PromptTemplate,
	type PromptVariable,
	VarSource,
	VarType,
} from '@/spec/prompt';

import { omitManyKeys } from '@/lib/obj_utils';
import { validateSlug, validateTags } from '@/lib/text_utils';
import { getUUIDv7 } from '@/lib/uuid_utils';
import { DEFAULT_SEMVER, isSemverVersion, suggestNextMinorVersion } from '@/lib/version_utils';

import { Dropdown, type DropdownItem } from '@/components/dropdown';
import { ModalBackdrop } from '@/components/modal_backdrop';
import { ReadOnlyValue } from '@/components/read_only_value';

interface TemplateItem {
	template: PromptTemplate;
	bundleID: string;
	templateSlug: string;
}

type ModalMode = 'add' | 'edit' | 'view';

interface AddEditPromptTemplateModalProps {
	isOpen: boolean;
	onClose: () => void;
	onSubmit: (templateData: Partial<PromptTemplate>) => Promise<void>;
	initialData?: TemplateItem; // when editing/viewing
	existingTemplates: TemplateItem[];
	mode?: ModalMode;
}

type ErrorState = {
	displayName?: string;
	slug?: string;
	content?: string;
	tags?: string;
	version?: string;
	blocks?: string;
	variables?: string;
};

export function AddEditPromptTemplateModal({
	isOpen,
	onClose,
	onSubmit,
	initialData,
	existingTemplates,
	mode,
}: AddEditPromptTemplateModalProps) {
	const effectiveMode: ModalMode = mode ?? (initialData ? 'edit' : 'add');
	const isViewMode = effectiveMode === 'view';
	const isEditMode = effectiveMode === 'edit';

	const [formData, setFormData] = useState({
		displayName: '',
		slug: '',
		description: '',
		tags: '',
		isEnabled: true,
		version: DEFAULT_SEMVER,
		blocks: [] as MessageBlock[],
		variables: [] as PromptVariable[],
	});

	const [errors, setErrors] = useState<ErrorState>({});
	const [submitError, setSubmitError] = useState<string>('');

	const dialogRef = useRef<HTMLDialogElement | null>(null);
	const roleDropdownItems = useMemo(() => {
		const obj = {} as Record<PromptRoleEnum, DropdownItem>;
		Object.values(PromptRoleEnum).forEach(r => {
			obj[r] = { isEnabled: true };
		});
		return obj;
	}, []);

	const varTypeDropdownItems = useMemo(() => {
		const obj = {} as Record<VarType, DropdownItem>;
		Object.values(VarType).forEach(t => {
			obj[t] = { isEnabled: true };
		});
		return obj;
	}, []);

	const varSourceDropdownItems = useMemo(() => {
		const obj = {} as Record<VarSource, DropdownItem>;
		Object.values(VarSource).forEach(s => {
			obj[s] = { isEnabled: true };
		});
		return obj;
	}, []);

	useEffect(() => {
		if (!isOpen) return;

		if (initialData) {
			const src = initialData.template;
			const existingVersionsForSlug = existingTemplates
				.filter(t => t.template.slug === src.slug)
				.map(t => t.template.version);

			const nextV = isEditMode ? suggestNextMinorVersion(src.version, existingVersionsForSlug).suggested : src.version;

			setFormData({
				displayName: src.displayName,
				slug: src.slug,
				description: src.description ?? '',
				tags: (src.tags ?? []).join(', '),
				isEnabled: src.isEnabled,
				version: nextV,
				blocks: src.blocks?.length ? src.blocks : [{ id: getUUIDv7(), role: PromptRoleEnum.User, content: '' }],
				variables: src.variables ?? [],
			});
		} else {
			setFormData({
				displayName: '',
				slug: '',
				description: '',
				tags: '',
				isEnabled: true,
				version: DEFAULT_SEMVER,
				blocks: [{ id: getUUIDv7(), role: PromptRoleEnum.User, content: '' }],
				variables: [],
			});
		}
		setErrors({});
		setSubmitError('');
	}, [isOpen, initialData, isEditMode]);

	useEffect(() => {
		if (!isOpen) return;

		const dialog = dialogRef.current;
		if (!dialog) return;

		if (!dialog.open) {
			dialog.showModal();
		}

		return () => {
			if (dialog.open) {
				dialog.close();
			}
		};
	}, [isOpen]);

	const handleDialogClose = () => {
		onClose();
	};

	const validateField = (field: keyof ErrorState, val: string, currentErrors: ErrorState): ErrorState => {
		let newErrs: ErrorState = { ...currentErrors };
		const v = val.trim();

		if (field === 'slug') {
			if (!v) {
				newErrs.slug = 'This field is required.';
				return newErrs;
			}
			const err = validateSlug(v);
			if (err) {
				newErrs.slug = err;
			} else {
				const clash = existingTemplates.some(t => t.template.slug === v && t.template.id !== initialData?.template.id);
				if (clash) newErrs.slug = 'Slug already in use.';
				else newErrs = omitManyKeys(newErrs, ['slug']);
			}
		} else if (field === 'displayName') {
			if (!v) newErrs.displayName = 'This field is required.';
			else newErrs = omitManyKeys(newErrs, ['displayName']);
		} else if (field === 'version') {
			if (!v) {
				newErrs.version = 'Version is required.';
			} else if (isEditMode && initialData?.template && v === initialData.template.version) {
				newErrs.version = 'New version must be different from the current version.';
			} else {
				const slugToCheck = initialData?.template.slug ?? formData.slug.trim();
				const versionClash = existingTemplates.some(t => t.template.slug === slugToCheck && t.template.version === v);
				if (versionClash) newErrs.version = 'That version already exists for this slug.';
				else newErrs = omitManyKeys(newErrs, ['version']);
			}
		} else if (field === 'tags') {
			const err = validateTags(val);
			if (err) newErrs.tags = err;
			else newErrs = omitManyKeys(newErrs, ['tags']);
		} else if (field === 'blocks') {
			if (!formData.blocks.length) {
				newErrs.blocks = 'At least one block is required.';
			} else if (formData.blocks.some(b => !b.content.trim())) {
				newErrs.blocks = 'All blocks must have non-empty content.';
			} else {
				newErrs = omitManyKeys(newErrs, ['blocks']);
			}
		} else if (field === 'variables') {
			const vars = formData.variables ?? [];
			const names = vars.map(x => x.name.trim()).filter(Boolean);
			const unique = new Set(names);
			const hasDupes = unique.size !== names.length;
			const hasMissing = vars.some(vr => !vr.name.trim());
			const badEnum = vars.some(vr => vr.type === VarType.Enum && (vr.enumValues ?? []).length === 0);
			const badStatic = vars.some(vr => vr.source === VarSource.Static && !(vr.staticVal ?? '').trim());

			if (hasMissing) newErrs.variables = 'All variables must have a name.';
			else if (hasDupes) newErrs.variables = 'Variable names must be unique.';
			else if (badEnum) newErrs.variables = 'Enum variables must include at least one enum value.';
			else if (badStatic) newErrs.variables = 'Static variables must include a static value.';
			else newErrs = omitManyKeys(newErrs, ['variables']);
		} else {
			newErrs = omitManyKeys(newErrs, [field]);
		}

		return newErrs;
	};

	const validateForm = (state: typeof formData): ErrorState => {
		let newErrs: ErrorState = {};
		newErrs = validateField('displayName', state.displayName, newErrs);
		if (!isEditMode) newErrs = validateField('slug', state.slug, newErrs);
		newErrs = validateField('version', state.version, newErrs);
		newErrs = validateField('blocks', 'x', newErrs);
		newErrs = validateField('variables', 'x', newErrs);

		// Fix: validate tags ONLY when non-empty
		if (state.tags.trim() !== '') {
			newErrs = validateField('tags', state.tags, newErrs);
		}

		return newErrs;
	};

	const handleInput = (e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
		const { name, value, type, checked } = e.target as HTMLInputElement;
		const newVal = type === 'checkbox' ? checked : value;

		setFormData(prev => {
			const next = { ...prev, [name]: newVal };
			if (!isViewMode) setErrors(validateForm(next));
			return next;
		});

		if (name === 'displayName' || name === 'slug' || name === 'tags' || name === 'version') {
			setErrors(prev => validateField(name as keyof ErrorState, String(newVal), prev));
		}
	};

	const updateBlock = (idx: number, patch: Partial<MessageBlock>) => {
		setFormData(prev => {
			const blocks = prev.blocks.map((b, i) => (i === idx ? { ...b, ...patch } : b));
			const next = { ...prev, blocks };
			if (!isViewMode) setErrors(validateForm(next));
			return next;
		});
	};

	const addBlock = () => {
		setFormData(prev => {
			const next = {
				...prev,
				blocks: [...prev.blocks, { id: getUUIDv7(), role: PromptRoleEnum.User, content: '' }],
			};
			if (!isViewMode) setErrors(validateForm(next));
			return next;
		});
	};

	const removeBlock = (idx: number) => {
		setFormData(prev => {
			const next = { ...prev, blocks: prev.blocks.filter((_, i) => i !== idx) };
			if (!isViewMode) setErrors(validateForm(next));
			return next;
		});
	};

	const updateVariable = (idx: number, patch: Partial<PromptVariable>) => {
		setFormData(prev => {
			const variables = prev.variables.map((v, i) => (i === idx ? { ...v, ...patch } : v));
			const next = { ...prev, variables };
			if (!isViewMode) setErrors(validateForm(next));
			return next;
		});
	};

	const addVariable = () => {
		setFormData(prev => {
			const nextVar: PromptVariable = {
				name: '',
				type: VarType.String,
				required: false,
				source: VarSource.User,
			};
			const next = { ...prev, variables: [...prev.variables, nextVar] };
			if (!isViewMode) setErrors(validateForm(next));
			return next;
		});
	};

	const removeVariable = (idx: number) => {
		setFormData(prev => {
			const next = { ...prev, variables: prev.variables.filter((_, i) => i !== idx) };
			if (!isViewMode) setErrors(validateForm(next));
			return next;
		});
	};

	const isAllValid = useMemo(() => {
		if (isViewMode) return true;
		const v = validateForm(formData);
		return Object.keys(v).length === 0;
	}, [formData, isViewMode, isEditMode]);

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = e => {
		e.preventDefault();
		e.stopPropagation();
		if (isViewMode) return;
		setSubmitError('');

		const newErrs = validateForm(formData);

		setErrors(newErrs);

		if (Object.keys(newErrs).length > 0) return;

		const tagsArr = formData.tags
			.split(',')
			.map(t => t.trim())
			.filter(Boolean);

		onSubmit({
			displayName: formData.displayName.trim(),
			slug: formData.slug.trim(),
			description: formData.description.trim() || undefined,
			isEnabled: formData.isEnabled,
			tags: tagsArr.length ? tagsArr : undefined,
			version: formData.version.trim(),
			blocks: formData.blocks.map(b => ({ ...b, content: b.content })),
			variables: formData.variables.length ? formData.variables : undefined,
		})
			.then(() => {
				dialogRef.current?.close();
			})
			.catch((err: unknown) => {
				const msg = err instanceof Error ? err.message : 'Failed to save prompt template.';
				setSubmitError(msg);
			});
	};

	if (!isOpen) return null;

	const headerTitle =
		effectiveMode === 'view'
			? 'View Prompt Template'
			: effectiveMode === 'edit'
				? 'Create New Prompt Template Version'
				: 'Add Prompt Template';

	return createPortal(
		<dialog
			ref={dialogRef}
			className="modal"
			onClose={handleDialogClose}
			onCancel={e => {
				// Form mode (add/edit): block Esc close. View mode: allow.
				if (!isViewMode) e.preventDefault();
			}}
		>
			<div className="modal-box bg-base-200 max-h-[80vh] max-w-3xl overflow-hidden rounded-2xl p-0">
				<div className="max-h-[80vh] overflow-y-auto p-6">
					<div className="mb-4 flex items-center justify-between">
						<h3 className="text-lg font-bold">{headerTitle}</h3>
						<button
							type="button"
							className="btn btn-sm btn-circle bg-base-300"
							onClick={() => dialogRef.current?.close()}
							aria-label="Close"
						>
							<FiX size={12} />
						</button>
					</div>

					<form noValidate onSubmit={handleSubmit} className="space-y-4">
						{submitError && (
							<div className="alert alert-error rounded-2xl text-sm">
								<div className="flex items-center gap-2">
									<FiAlertCircle size={14} />
									<span>{submitError}</span>
								</div>
							</div>
						)}
						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Display Name*</span>
							</label>
							<div className="col-span-9">
								<input
									type="text"
									name="displayName"
									value={formData.displayName}
									onChange={handleInput}
									readOnly={isViewMode}
									className={`input input-bordered w-full rounded-xl ${errors.displayName ? 'input-error' : ''}`}
									spellCheck="false"
									autoComplete="off"
									autoFocus={!isViewMode}
									aria-invalid={Boolean(errors.displayName)}
								/>
								{errors.displayName && (
									<div className="label">
										<span className="label-text-alt text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.displayName}
										</span>
									</div>
								)}
							</div>
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Slug*</span>
								<span className="label-text-alt tooltip tooltip-right" data-tip="Short user friendly command">
									<FiHelpCircle size={12} />
								</span>
							</label>
							<div className="col-span-9">
								<input
									type="text"
									name="slug"
									value={formData.slug}
									onChange={handleInput}
									className={`input input-bordered w-full rounded-xl ${errors.slug ? 'input-error' : ''}`}
									spellCheck="false"
									autoComplete="off"
									readOnly={isViewMode || isEditMode}
									aria-invalid={Boolean(errors.slug)}
								/>
								{errors.slug && (
									<div className="label">
										<span className="label-text-alt text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.slug}
										</span>
									</div>
								)}
							</div>
						</div>

						{/* Version */}
						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Version*</span>
								<span
									className="label-text-alt tooltip tooltip-right"
									data-tip="Once created, existing versions are not edited. Edit creates a new version."
								>
									<FiHelpCircle size={12} />
								</span>
							</label>
							<div className="col-span-9">
								<input
									type="text"
									name="version"
									value={formData.version}
									onChange={handleInput}
									readOnly={isViewMode}
									className={`input input-bordered w-full rounded-xl ${errors.version ? 'input-error' : ''}`}
									spellCheck="false"
									autoComplete="off"
									aria-invalid={Boolean(errors.version)}
									placeholder={DEFAULT_SEMVER}
								/>
								{isEditMode && initialData?.template && (
									<div className="label">
										<span className="label-text-alt text-base-content/70 text-xs">
											Current: {initialData.template.version} · Suggested next:{' '}
											{
												suggestNextMinorVersion(
													initialData.template.version,
													existingTemplates
														.filter(t => t.template.slug === initialData.template.slug)
														.map(t => t.template.version)
												).suggested
											}
											{!isSemverVersion(initialData.template.version) ? ' (current is not semver)' : ''}
										</span>
									</div>
								)}
								{errors.version && (
									<div className="label">
										<span className="label-text-alt text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.version}
										</span>
									</div>
								)}
							</div>
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3 cursor-pointer">
								<span className="label-text text-sm">Enabled</span>
							</label>
							<div className="col-span-9">
								<input
									type="checkbox"
									name="isEnabled"
									checked={formData.isEnabled}
									onChange={handleInput}
									className="toggle toggle-accent"
									disabled={isViewMode}
								/>
							</div>
						</div>

						<div className="grid grid-cols-12 items-start gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Description</span>
							</label>
							<div className="col-span-9">
								<textarea
									name="description"
									value={formData.description}
									onChange={handleInput}
									readOnly={isViewMode}
									className="textarea textarea-bordered h-20 w-full rounded-xl"
									spellCheck="false"
								/>
							</div>
						</div>
						{/* Blocks */}
						<div className="divider">Blocks</div>
						{errors.blocks && (
							<div className="text-error flex items-center gap-1 text-sm">
								<FiAlertCircle size={12} /> {errors.blocks}
							</div>
						)}

						<div className="space-y-3">
							{formData.blocks.map((b, idx) => (
								<div key={b.id} className="border-base-content/10 rounded-2xl border p-3">
									<div className="mb-2 flex items-center justify-between gap-2">
										<div className="flex items-center gap-2">
											<span className="text-base-content/70 text-xs font-semibold uppercase">Role</span>
											{isViewMode ? (
												<ReadOnlyValue value={b.role} />
											) : (
												<div className="w-44">
													<Dropdown<PromptRoleEnum>
														dropdownItems={roleDropdownItems}
														selectedKey={b.role}
														onChange={role => {
															updateBlock(idx, { role });
														}}
														filterDisabled={false}
														title="Select role"
													/>
												</div>
											)}
										</div>

										{!isViewMode && (
											<button
												type="button"
												className="btn btn-ghost btn-sm rounded-xl"
												onClick={() => {
													removeBlock(idx);
												}}
												disabled={formData.blocks.length <= 1}
												title="Remove block"
											>
												<FiTrash2 size={14} />
											</button>
										)}
									</div>

									<textarea
										className="textarea textarea-bordered bg-base-100 w-full rounded-xl"
										readOnly={isViewMode}
										spellCheck="false"
										value={b.content}
										onChange={e => {
											updateBlock(idx, { content: e.target.value });
										}}
									/>
								</div>
							))}

							{!isViewMode && (
								<button type="button" className="btn btn-ghost rounded-xl" onClick={addBlock}>
									<FiPlus size={14} />
									<span className="ml-1">Add Block</span>
								</button>
							)}
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Tags</span>
							</label>
							<div className="col-span-9">
								<input
									type="text"
									name="tags"
									value={formData.tags}
									onChange={handleInput}
									readOnly={isViewMode}
									className={`input input-bordered w-full rounded-xl ${errors.tags ? 'input-error' : ''}`}
									placeholder="comma, separated, tags"
									spellCheck="false"
									aria-invalid={Boolean(errors.tags)}
								/>
								{errors.tags && (
									<div className="label">
										<span className="label-text-alt text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.tags}
										</span>
									</div>
								)}
							</div>
						</div>

						{/* Variables */}
						<div className="divider">Variables</div>
						{errors.variables && (
							<div className="text-error flex items-center gap-1 text-sm">
								<FiAlertCircle size={12} /> {errors.variables}
							</div>
						)}

						<div className="space-y-3">
							{formData.variables.map((v, idx) => (
								<div key={`${idx}-${v.name}`} className="border-base-content/10 rounded-2xl border p-3">
									<div className="mb-2 flex items-center justify-between">
										<div className="text-base-content/70 text-xs font-semibold uppercase">Variable</div>
										<div>
											<div className="flex gap-1">
												<label className="label">
													<span className="label-text text-sm">Required</span>
												</label>
												<input
													type="checkbox"
													className="toggle toggle-accent disabled:opacity-80"
													checked={v.required}
													disabled={isViewMode}
													onChange={e => {
														updateVariable(idx, { required: e.target.checked });
													}}
												/>
												{!isViewMode && (
													<button
														type="button"
														className="btn btn-ghost btn-sm rounded-xl"
														onClick={() => {
															removeVariable(idx);
														}}
														title="Remove variable"
													>
														<FiTrash2 size={14} />
													</button>
												)}
											</div>
										</div>
									</div>

									<div className="grid grid-cols-12 gap-2">
										<div className="col-span-12 md:col-span-4">
											<label className="label py-1">
												<span className="label-text text-sm">Name</span>
											</label>
											<input
												className="input input-bordered bg-base-100 w-full rounded-xl"
												readOnly={isViewMode}
												value={v.name}
												onChange={e => {
													updateVariable(idx, { name: e.target.value });
												}}
											/>
										</div>

										<div className="col-span-6 md:col-span-4">
											<label className="label py-1">
												<span className="label-text text-sm">Type</span>
											</label>
											{isViewMode ? (
												<ReadOnlyValue value={v.type} />
											) : (
												<Dropdown<VarType>
													dropdownItems={varTypeDropdownItems}
													selectedKey={v.type}
													onChange={type => {
														updateVariable(idx, { type });
													}}
													filterDisabled={false}
													title="Select type"
												/>
											)}
										</div>

										<div className="col-span-6 md:col-span-4">
											<label className="label py-1">
												<span className="label-text text-sm">Source</span>
											</label>
											{isViewMode ? (
												<ReadOnlyValue value={v.source} />
											) : (
												<Dropdown<VarSource>
													dropdownItems={varSourceDropdownItems}
													selectedKey={v.source}
													onChange={source => {
														updateVariable(idx, { source });
													}}
													filterDisabled={false}
													title="Select source"
												/>
											)}
										</div>

										<div className="col-span-12">
											<label className="label py-1">
												<span className="label-text text-sm">Description</span>
											</label>
											<input
												className="input input-bordered bg-base-100 w-full rounded-xl"
												readOnly={isViewMode}
												value={v.description ?? ''}
												onChange={e => {
													updateVariable(idx, { description: e.target.value });
												}}
											/>
										</div>

										<div className="col-span-6 md:col-span-4">
											<label className="label py-1">
												<span className="label-text text-sm">Default</span>
											</label>
											<input
												className="input input-bordered bg-base-100 w-full rounded-xl"
												readOnly={isViewMode}
												value={v.default ?? ''}
												onChange={e => {
													updateVariable(idx, { default: e.target.value });
												}}
											/>
										</div>

										{v.source === VarSource.Static && (
											<div className="col-span-12 md:col-span-6">
												<label className="label py-1">
													<span className="label-text text-sm">Static Value</span>
												</label>
												<input
													className="input input-bordered bg-base-100 w-full rounded-xl"
													readOnly={isViewMode}
													value={v.staticVal ?? ''}
													onChange={e => {
														updateVariable(idx, { staticVal: e.target.value });
													}}
												/>
											</div>
										)}

										{v.type === VarType.Enum && (
											<div className="col-span-12 md:col-span-6">
												<label className="label py-1">
													<span className="label-text text-sm">Enum Values (comma)</span>
												</label>
												<input
													className="input input-bordered bg-base-100 w-full rounded-xl"
													readOnly={isViewMode}
													value={(v.enumValues ?? []).join(', ')}
													onChange={e => {
														updateVariable(idx, {
															enumValues: e.target.value
																.split(',')
																.map(s => s.trim())
																.filter(Boolean),
														});
													}}
												/>
											</div>
										)}
									</div>
								</div>
							))}

							{!isViewMode && (
								<button type="button" className="btn btn-ghost rounded-xl" onClick={addVariable}>
									<FiPlus size={14} />
									<span className="ml-1">Add Variable</span>
								</button>
							)}
						</div>

						{/* View mode: show meta */}
						{isViewMode && initialData?.template && (
							<>
								<div className="divider">Metadata</div>
								<div className="grid grid-cols-12 gap-2 text-sm">
									<div className="col-span-3 font-semibold">Version</div>
									<div className="col-span-9">{initialData.template.version}</div>

									<div className="col-span-3 font-semibold">Built-in</div>
									<div className="col-span-9">{initialData.template.isBuiltIn ? 'Yes' : 'No'}</div>

									<div className="col-span-3 font-semibold">Created</div>
									<div className="col-span-9">{initialData.template.createdAt}</div>

									<div className="col-span-3 font-semibold">Modified</div>
									<div className="col-span-9">{initialData.template.modifiedAt}</div>
								</div>
							</>
						)}

						<div className="modal-action">
							<button type="button" className="btn bg-base-300 rounded-xl" onClick={() => dialogRef.current?.close()}>
								{isViewMode ? 'Close' : 'Cancel'}
							</button>
							{!isViewMode && (
								<button type="submit" className="btn btn-primary rounded-xl" disabled={!isAllValid}>
									Save
								</button>
							)}
						</div>
					</form>
				</div>
			</div>
			<ModalBackdrop enabled={isViewMode} />
		</dialog>,
		document.body
	);
}
