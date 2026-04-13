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

import {
	cloneVariable,
	derivePromptTemplateKind,
	derivePromptTemplateResolved,
	extractPromptTemplatePlaceholders,
	getPromptTemplateKindLabel,
	getPromptTemplateResolutionLabel,
	type PromptTemplateUpsertInput,
	validatePromptVariableName,
} from '@/prompts/lib/prompt_template_utils';

interface TemplateItem {
	template: PromptTemplate;
	bundleID: string;
	templateSlug: string;
}

type ModalMode = 'add' | 'edit' | 'view';

interface AddEditPromptTemplateModalProps {
	isOpen: boolean;
	onClose: () => void;
	onSubmit: (templateData: PromptTemplateUpsertInput) => Promise<void>;
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

type PromptTemplateFormData = {
	displayName: string;
	slug: string;
	description: string;
	tags: string;
	isEnabled: boolean;
	version: string;
	blocks: MessageBlock[];
	variables: PromptVariable[];
};

function getSuggestedNextVersion(initialData: TemplateItem, existingTemplates: TemplateItem[]): string {
	return suggestNextMinorVersion(
		initialData.template.version,
		existingTemplates.filter(t => t.template.slug === initialData.template.slug).map(t => t.template.version)
	).suggested;
}

function getInitialFormData(
	initialData: TemplateItem | undefined,
	existingTemplates: TemplateItem[],
	isEditMode: boolean
): PromptTemplateFormData {
	if (initialData) {
		const src = initialData.template;
		const nextVersion = isEditMode ? getSuggestedNextVersion(initialData, existingTemplates) : src.version;

		return {
			displayName: src.displayName,
			slug: src.slug,
			description: src.description ?? '',
			tags: (src.tags ?? []).join(', '),
			isEnabled: src.isEnabled,
			version: nextVersion,
			blocks: src.blocks?.length
				? src.blocks.map(block => ({ ...block }))
				: [{ id: getUUIDv7(), role: PromptRoleEnum.User, content: '' }],
			variables: src.variables?.length ? src.variables.map(cloneVariable) : [],
		};
	}

	return {
		displayName: '',
		slug: '',
		description: '',
		tags: '',
		isEnabled: true,
		version: DEFAULT_SEMVER,
		blocks: [{ id: getUUIDv7(), role: PromptRoleEnum.User, content: '' }],
		variables: [],
	};
}

function AddEditPromptTemplateModalContent({
	onClose,
	onSubmit,
	initialData,
	existingTemplates,
	mode,
}: AddEditPromptTemplateModalProps) {
	const effectiveMode: ModalMode = mode ?? (initialData ? 'edit' : 'add');
	const isViewMode = effectiveMode === 'view';
	const isEditMode = effectiveMode === 'edit';

	const [formData, setFormData] = useState<PromptTemplateFormData>(() =>
		getInitialFormData(initialData, existingTemplates, isEditMode)
	);
	const [errors, setErrors] = useState<ErrorState>({});
	const [submitError, setSubmitError] = useState('');

	const initialTemplateId = initialData?.template?.id;
	const initialTemplateSlug = initialData?.template?.slug;
	const initialTemplateVersion = initialData?.template?.version;

	const dialogRef = useRef<HTMLDialogElement | null>(null);
	const isUnmountingRef = useRef(false);

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

	const suggestedNextVersion = useMemo(() => {
		if (!initialData) return DEFAULT_SEMVER;
		return getSuggestedNextVersion(initialData, existingTemplates);
	}, [initialData, existingTemplates]);
	const derivedKind = useMemo(() => derivePromptTemplateKind(formData.blocks), [formData.blocks]);
	const derivedIsResolved = useMemo(
		() => derivePromptTemplateResolved(formData.blocks, formData.variables),
		[formData.blocks, formData.variables]
	);
	useEffect(() => {
		const dialog = dialogRef.current;
		if (!dialog) return;

		if (!dialog.open) {
			try {
				dialog.showModal();
			} catch {
				// Ignore showModal errors and keep rendering safely.
			}
		}

		return () => {
			isUnmountingRef.current = true;

			if (dialog.open) {
				dialog.close();
			}
		};
	}, []);

	const requestClose = () => {
		const dialog = dialogRef.current;

		if (dialog?.open) {
			dialog.close();
			return;
		}

		onClose();
	};

	const handleDialogClose = () => {
		if (isUnmountingRef.current) return;
		onClose();
	};

	const validateField = (
		field: keyof ErrorState,
		val: string,
		state: PromptTemplateFormData,
		currentErrors: ErrorState
	): ErrorState => {
		let newErrs: ErrorState = { ...currentErrors };
		const variables = state.variables ?? [];
		const blocks = state.blocks ?? [];
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
				const clash = existingTemplates.some(t => t.template.slug === v && t.template.id !== initialTemplateId);

				if (clash) newErrs.slug = 'Slug already in use.';
				else newErrs = omitManyKeys(newErrs, ['slug']);
			}
		} else if (field === 'displayName') {
			if (!v) newErrs.displayName = 'This field is required.';
			else newErrs = omitManyKeys(newErrs, ['displayName']);
		} else if (field === 'version') {
			if (!v) {
				newErrs.version = 'Version is required.';
			} else if (isEditMode && initialTemplateVersion && v === initialTemplateVersion) {
				newErrs.version = 'New version must be different from the current version.';
			} else {
				const slugToCheck = isEditMode ? (initialTemplateSlug ?? state.slug.trim()) : state.slug.trim();
				const versionClash =
					Boolean(slugToCheck) &&
					existingTemplates.some(t => t.template.slug === slugToCheck && t.template.version === v);

				if (versionClash) newErrs.version = 'That version already exists for this slug.';
				else newErrs = omitManyKeys(newErrs, ['version']);
			}
		} else if (field === 'tags') {
			if (v === '') {
				newErrs = omitManyKeys(newErrs, ['tags']);
			} else {
				const err = validateTags(val);
				if (err) newErrs.tags = err;
				else newErrs = omitManyKeys(newErrs, ['tags']);
			}
		} else if (field === 'blocks') {
			if (!blocks.length) {
				newErrs.blocks = 'At least one block is required.';
			} else if (blocks.some(block => !block.content.trim())) {
				newErrs.blocks = 'All blocks must have non-empty content.';
			} else {
				newErrs = omitManyKeys(newErrs, ['blocks']);
			}
		} else if (field === 'variables') {
			const names = variables.map(variable => variable.name.trim()).filter(Boolean);
			const unique = new Set(names);
			const hasDupes = unique.size !== names.length;
			const hasMissing = variables.some(variable => !variable.name.trim());
			const invalidName = variables.find(variable => validatePromptVariableName(variable.name) !== undefined);
			const badEnum = variables.some(
				variable => variable.type === VarType.Enum && (variable.enumValues ?? []).length === 0
			);
			const badEnumDefault = variables.some(
				variable =>
					variable.type === VarType.Enum &&
					variable.default !== undefined &&
					!(variable.enumValues ?? []).includes(variable.default)
			);
			const badStatic = variables.some(
				variable => variable.source === VarSource.Static && (variable.staticVal ?? '').trim().length === 0
			);
			const badStaticRequired = variables.some(variable => variable.source === VarSource.Static && variable.required);
			const placeholders = new Set(extractPromptTemplatePlaceholders(blocks));
			const missingDeclaredPlaceholder = [...placeholders].some(name => !unique.has(name));
			const unusedUserVariable = variables.some(variable => {
				const name = variable.name.trim();
				return Boolean(name) && variable.source === VarSource.User && !placeholders.has(name);
			});

			if (hasMissing) newErrs.variables = 'All variables must have a name.';
			else if (invalidName) newErrs.variables = validatePromptVariableName(invalidName.name);
			else if (hasDupes) newErrs.variables = 'Variable names must be unique.';
			else if (badEnum) newErrs.variables = 'Enum variables must include at least one enum value.';
			else if (badEnumDefault) newErrs.variables = 'Enum defaults must be one of the enum values.';
			else if (badStatic) {
				newErrs.variables = 'Static variables must include a non-empty static value.';
			} else if (badStaticRequired) {
				newErrs.variables = 'Static variables cannot be required.';
			} else if (missingDeclaredPlaceholder) {
				newErrs.variables = 'All placeholders used in blocks must be declared as variables.';
			} else if (unusedUserVariable) {
				newErrs.variables = 'User variables must be referenced by at least one block.';
			} else newErrs = omitManyKeys(newErrs, ['variables']);
		} else {
			newErrs = omitManyKeys(newErrs, [field]);
		}

		return newErrs;
	};

	const validateForm = (state: PromptTemplateFormData): ErrorState => {
		let newErrs: ErrorState = {};
		newErrs = validateField('displayName', state.displayName, state, newErrs);
		if (!isEditMode) newErrs = validateField('slug', state.slug, state, newErrs);
		newErrs = validateField('version', state.version, state, newErrs);
		newErrs = validateField('blocks', 'x', state, newErrs);
		newErrs = validateField('variables', 'x', state, newErrs);

		if (state.tags.trim() !== '') {
			newErrs = validateField('tags', state.tags, state, newErrs);
		}

		return newErrs;
	};

	const handleInput = (e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
		const target = e.target as HTMLInputElement;
		const { name, value, type, checked } = target;
		const newVal = type === 'checkbox' ? checked : value;
		const next = { ...formData, [name]: newVal } as PromptTemplateFormData;

		setFormData(next);

		if (!isViewMode) {
			setErrors(validateForm(next));
		}
	};

	const updateBlock = (idx: number, patch: Partial<MessageBlock>) => {
		const next: PromptTemplateFormData = {
			...formData,
			blocks: formData.blocks.map((block, i) => (i === idx ? { ...block, ...patch } : block)),
		};

		setFormData(next);

		if (!isViewMode) {
			setErrors(validateForm(next));
		}
	};

	const addBlock = () => {
		const next: PromptTemplateFormData = {
			...formData,
			blocks: [...formData.blocks, { id: getUUIDv7(), role: PromptRoleEnum.User, content: '' }],
		};

		setFormData(next);

		if (!isViewMode) {
			setErrors(validateForm(next));
		}
	};

	const removeBlock = (idx: number) => {
		const next: PromptTemplateFormData = {
			...formData,
			blocks: formData.blocks.filter((_, i) => i !== idx),
		};

		setFormData(next);

		if (!isViewMode) {
			setErrors(validateForm(next));
		}
	};

	const updateVariable = (idx: number, patch: Partial<PromptVariable>) => {
		const next: PromptTemplateFormData = {
			...formData,
			variables: formData.variables.map((variable, i) => (i === idx ? { ...variable, ...patch } : variable)),
		};

		setFormData(next);

		if (!isViewMode) {
			setErrors(validateForm(next));
		}
	};

	const addVariable = () => {
		const nextVar: PromptVariable = {
			name: '',
			type: VarType.String,
			required: false,
			source: VarSource.User,
		};

		const next: PromptTemplateFormData = {
			...formData,
			variables: [...formData.variables, nextVar],
		};

		setFormData(next);

		if (!isViewMode) {
			setErrors(validateForm(next));
		}
	};

	const removeVariable = (idx: number) => {
		const next: PromptTemplateFormData = {
			...formData,
			variables: formData.variables.filter((_, i) => i !== idx),
		};

		setFormData(next);

		if (!isViewMode) {
			setErrors(validateForm(next));
		}
	};

	// Memoize so validateForm only re-runs when formData actually changes,
	// not on every parent-triggered re-render.
	const isAllValid = useMemo(
		() => (isViewMode ? true : Object.keys(validateForm(formData)).length === 0),
		// validateForm captures existingTemplates / initialTemplate* which are
		// stable for the lifetime of a single modal mount — safe to omit.
		// eslint-disable-next-line react-hooks/exhaustive-deps
		[isViewMode, formData]
	);

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
		const normalizedVariables = formData.variables.map(variable => ({
			...variable,
			name: variable.name.trim(),
			required: variable.source === VarSource.Static ? false : variable.required,
			description: variable.description?.trim() || undefined,
			enumValues: variable.enumValues?.map(value => value.trim()).filter(Boolean),
		}));

		void onSubmit({
			displayName: formData.displayName.trim(),
			slug: formData.slug.trim(),
			description: formData.description.trim(),
			isEnabled: formData.isEnabled,
			tags: tagsArr,
			version: formData.version.trim(),
			blocks: formData.blocks.map(block => ({ ...block, content: block.content })),
			variables: normalizedVariables,
		})
			.then(() => {
				requestClose();
			})
			.catch((err: unknown) => {
				const msg = err instanceof Error ? err.message : 'Failed to save prompt template.';
				setSubmitError(msg);
			});
	};

	const headerTitle =
		effectiveMode === 'view'
			? 'View Prompt Template'
			: effectiveMode === 'edit'
				? 'Create New Prompt Template Version'
				: 'Add Prompt Template';

	return (
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
							onClick={requestClose}
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
											Current: {initialData.template.version} · Suggested next: {suggestedNextVersion}
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

						<div className="grid grid-cols-12 items-start gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Derived</span>
								<span
									className="label-text-alt tooltip tooltip-right"
									data-tip="Kind is derived from block roles. Resolved is derived from whether every referenced placeholder already has a static value or default."
								>
									<FiHelpCircle size={12} />
								</span>
							</label>
							<div className="col-span-9 space-y-2">
								<div className="flex flex-wrap gap-2">
									<span className="badge badge-outline rounded-xl">{getPromptTemplateKindLabel(derivedKind)}</span>
									<span className={`badge rounded-xl ${derivedIsResolved ? 'badge-success' : 'badge-warning'}`}>
										{getPromptTemplateResolutionLabel(derivedIsResolved)}
									</span>
								</div>
								<p className="text-base-content/70 text-xs">
									Instructions only = every block is System or Developer. Resolved = every referenced placeholder can be
									satisfied locally by a static value or default.
								</p>
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
							{formData.blocks.map((block, idx) => (
								<div key={block.id} className="border-base-content/10 rounded-2xl border p-3">
									<div className="mb-2 flex items-center justify-between gap-2">
										<div className="flex items-center gap-2">
											<span className="text-base-content/70 text-xs font-semibold uppercase">Role</span>
											{isViewMode ? (
												<ReadOnlyValue value={block.role} />
											) : (
												<div className="w-44">
													<Dropdown<PromptRoleEnum>
														dropdownItems={roleDropdownItems}
														selectedKey={block.role}
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
										value={block.content}
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
							{formData.variables.map((variable, idx) => (
								<div key={`${idx}-${variable.name}`} className="border-base-content/10 rounded-2xl border p-3">
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
													checked={variable.source === VarSource.Static ? false : variable.required}
													disabled={isViewMode || variable.source === VarSource.Static}
													title={
														variable.source === VarSource.Static ? 'Static variables cannot be required.' : undefined
													}
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
												value={variable.name}
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
												<ReadOnlyValue value={variable.type} />
											) : (
												<Dropdown<VarType>
													dropdownItems={varTypeDropdownItems}
													selectedKey={variable.type}
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
												<ReadOnlyValue value={variable.source} />
											) : (
												<Dropdown<VarSource>
													dropdownItems={varSourceDropdownItems}
													selectedKey={variable.source}
													onChange={source => {
														updateVariable(idx, {
															source,
															required: source === VarSource.Static ? false : variable.required,
														});
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
												value={variable.description ?? ''}
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
												value={variable.default ?? ''}
												onChange={e => {
													updateVariable(idx, { default: e.target.value });
												}}
											/>
										</div>

										{variable.source === VarSource.Static && (
											<div className="col-span-12 md:col-span-6">
												<label className="label py-1">
													<span className="label-text text-sm">Static Value</span>
												</label>
												<input
													className="input input-bordered bg-base-100 w-full rounded-xl"
													readOnly={isViewMode}
													value={variable.staticVal ?? ''}
													onChange={e => {
														updateVariable(idx, { staticVal: e.target.value });
													}}
												/>
											</div>
										)}

										{variable.type === VarType.Enum && (
											<div className="col-span-12 md:col-span-6">
												<label className="label py-1">
													<span className="label-text text-sm">Enum Values (comma)</span>
												</label>
												<input
													className="input input-bordered bg-base-100 w-full rounded-xl"
													readOnly={isViewMode}
													value={(variable.enumValues ?? []).join(', ')}
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

									<div className="col-span-3 font-semibold">Kind</div>
									<div className="col-span-9">{getPromptTemplateKindLabel(initialData.template.kind)}</div>

									<div className="col-span-3 font-semibold">Resolved</div>
									<div className="col-span-9">{getPromptTemplateResolutionLabel(initialData.template.isResolved)}</div>

									<div className="col-span-3 font-semibold">Created</div>
									<div className="col-span-9">{initialData.template.createdAt}</div>

									<div className="col-span-3 font-semibold">Modified</div>
									<div className="col-span-9">{initialData.template.modifiedAt}</div>
								</div>
							</>
						)}

						<div className="modal-action">
							<button type="button" className="btn bg-base-300 rounded-xl" onClick={requestClose}>
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
		</dialog>
	);
}

export function AddEditPromptTemplateModal(props: AddEditPromptTemplateModalProps) {
	if (!props.isOpen) return null;
	if (typeof document === 'undefined' || !document.body) return null;

	const remountKey = props.initialData
		? `${props.mode ?? 'auto'}:${props.initialData.bundleID}:${props.initialData.template.id}:${props.initialData.template.version}:${props.initialData.template.modifiedAt}`
		: `${props.mode ?? 'auto'}:new`;

	return createPortal(<AddEditPromptTemplateModalContent key={remountKey} {...props} />, document.body);
}
