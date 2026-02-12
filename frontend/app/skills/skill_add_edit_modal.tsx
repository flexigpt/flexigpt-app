import { type ChangeEvent, type FormEvent, useEffect, useMemo, useRef, useState } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle, FiHelpCircle, FiX } from 'react-icons/fi';

import type { Skill } from '@/spec/skill';
import { SkillType } from '@/spec/skill';

import { omitManyKeys } from '@/lib/obj_utils';
import { validateSlug, validateTags } from '@/lib/text_utils';

import { Dropdown } from '@/components/dropdown';
import { ModalBackdrop } from '@/components/modal_backdrop';
import { ReadOnlyValue } from '@/components/read_only_value';

interface SkillItem {
	skill: Skill;
	bundleID: string;
	skillSlug: string;
}

type ModalMode = 'add' | 'edit' | 'view';

interface AddEditSkillModalProps {
	isOpen: boolean;
	onClose: () => void;
	onSubmit: (skillData: Partial<Skill>) => Promise<void>;
	initialData?: SkillItem; // editing/viewing
	existingSkills: SkillItem[];
	mode?: ModalMode;
}

type ErrorState = {
	displayName?: string;
	name?: string;
	slug?: string;
	type?: string;
	location?: string;
	tags?: string;
};

const skillTypeDropdownItems: Record<SkillType, { isEnabled: boolean; displayName: string }> = {
	[SkillType.FS]: { isEnabled: true, displayName: 'Filesystem (fs)' },
	// UI restriction: EmbeddedFS skills are built-in; not allowed to be created/edited.
	[SkillType.EmbeddedFS]: { isEnabled: false, displayName: 'EmbeddedFS (embeddedfs)' },
};

function normalizeForUniq(s: string) {
	return s.trim().toLowerCase();
}

export function AddEditSkillModal({
	isOpen,
	onClose,
	onSubmit,
	initialData,
	existingSkills,
	mode,
}: AddEditSkillModalProps) {
	const requestedMode: ModalMode = mode ?? (initialData ? 'edit' : 'add');
	// Match the Tool modal pattern: unsupported impls can exist (viewable),
	// but cannot be created/edited in the UI.
	const isLockedSkill = Boolean(initialData?.skill?.isBuiltIn) || initialData?.skill?.type === SkillType.EmbeddedFS;
	const effectiveMode: ModalMode = isLockedSkill ? 'view' : requestedMode;
	const isViewMode = effectiveMode === 'view';
	const isEditMode = effectiveMode === 'edit';

	const [formData, setFormData] = useState({
		displayName: '',
		name: '',
		slug: '',
		type: SkillType.FS as SkillType,
		location: '',
		description: '',
		tags: '',
		isEnabled: true,
	});

	const [errors, setErrors] = useState<ErrorState>({});
	const [submitError, setSubmitError] = useState<string>('');

	const dialogRef = useRef<HTMLDialogElement | null>(null);
	const nameInputRef = useRef<HTMLInputElement | null>(null);

	useEffect(() => {
		if (!isOpen) return;

		if (initialData) {
			const s = initialData.skill;
			setFormData({
				displayName: s.displayName ?? '',
				name: s.name ?? '',
				slug: s.slug ?? '',
				type: s.type,
				location: s.location ?? '',
				description: s.description ?? '',
				tags: (s.tags ?? []).join(', '),
				isEnabled: s.isEnabled,
			});
		} else {
			setFormData({
				displayName: '',
				name: '',
				slug: '',
				type: SkillType.FS,
				location: '',
				description: '',
				tags: '',
				isEnabled: true,
			});
		}

		setErrors({});
		setSubmitError('');
	}, [isOpen, initialData]);

	useEffect(() => {
		if (!isOpen) return;
		const dialog = dialogRef.current;
		if (!dialog) return;

		if (!dialog.open) dialog.showModal();

		window.setTimeout(() => {
			if (!isViewMode) nameInputRef.current?.focus();
		}, 0);

		return () => {
			if (dialog.open) dialog.close();
		};
	}, [isOpen, isViewMode]);

	const handleDialogClose = () => {
		onClose();
	};

	const validateField = (field: keyof ErrorState, val: string, currentErrors: ErrorState): ErrorState => {
		let nextErrors: ErrorState = { ...currentErrors };
		const v = val.trim();

		const requiredFields: Array<keyof ErrorState> = ['name', 'slug', 'type', 'location'];

		if (!v && requiredFields.includes(field)) {
			nextErrors[field] = 'This field is required.';
			return nextErrors;
		}

		if (field === 'slug') {
			const err = validateSlug(v);
			if (err) {
				nextErrors.slug = err;
			} else {
				const clash = existingSkills.some(x => x.skill.slug === v && x.skill.id !== initialData?.skill.id);
				if (clash) nextErrors.slug = 'Slug already in use in this bundle.';
				else nextErrors = omitManyKeys(nextErrors, ['slug']);
			}
		} else if (field === 'name') {
			// Constraint: within a bundle, skill.name cannot be duplicated
			const norm = normalizeForUniq(v);
			const clash = existingSkills.some(
				x => normalizeForUniq(x.skill.name) === norm && x.skill.id !== initialData?.skill.id
			);
			if (clash) nextErrors.name = 'Skill name must be unique within the bundle.';
			else nextErrors = omitManyKeys(nextErrors, ['name']);
		} else if (field === 'tags') {
			if (v === '') {
				nextErrors = omitManyKeys(nextErrors, ['tags']);
			} else {
				const err = validateTags(val);
				if (err) nextErrors.tags = err;
				else nextErrors = omitManyKeys(nextErrors, ['tags']);
			}
		} else {
			nextErrors = omitManyKeys(nextErrors, [field]);
		}

		return nextErrors;
	};

	const validateForm = (state: typeof formData): ErrorState => {
		let next: ErrorState = {};
		next = validateField('name', state.name, next);
		next = validateField('slug', state.slug, next);
		next = validateField('type', state.type, next);
		next = validateField('location', state.location, next);
		if (state.tags.trim() !== '') next = validateField('tags', state.tags, next);
		return next;
	};

	const onSkillTypeChange = (key: SkillType) => {
		// UI restriction: only FS skills can be created/edited
		if (key !== SkillType.FS) return;
		setFormData(prev => ({ ...prev, type: key }));
		setErrors(prev => validateField('type', key, prev));
	};

	const handleInput = (e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
		const { name, value, type, checked } = e.target as HTMLInputElement;
		const newVal = type === 'checkbox' ? checked : value;

		setFormData(prev => ({ ...prev, [name]: newVal }));

		if (['displayName', 'name', 'slug', 'type', 'location', 'tags'].includes(name)) {
			setErrors(prev => validateField(name as keyof ErrorState, String(newVal), prev));
		}
	};

	const isAllValid = useMemo(() => {
		if (isViewMode) return true;
		const hasErrs = Object.values(errors).some(Boolean);
		const required = formData.name.trim() && formData.slug.trim() && formData.location.trim() && formData.type;
		return Boolean(required) && !hasErrs;
	}, [errors, formData, isViewMode]);

	const handleSubmit = (e: FormEvent) => {
		e.preventDefault();
		e.stopPropagation();
		if (isViewMode) return;

		setSubmitError('');

		const nextErrors = validateForm(formData);
		setErrors(nextErrors);
		if (Object.values(nextErrors).some(Boolean)) return;

		const tagsArr = formData.tags
			.split(',')
			.map(t => t.trim())
			.filter(Boolean);

		onSubmit({
			displayName: formData.displayName.trim() || undefined,
			name: formData.name.trim(),
			slug: formData.slug.trim(),
			type: formData.type,
			location: formData.location.trim(),
			description: formData.description.trim() || undefined,
			tags: tagsArr.length ? tagsArr : undefined,
			isEnabled: formData.isEnabled,
		})
			.then(() => dialogRef.current?.close())
			.catch((err: unknown) => {
				const msg = err instanceof Error ? err.message : 'Failed to save skill.';
				setSubmitError(msg);
			});
	};

	const headerTitle = effectiveMode === 'view' ? 'View Skill' : effectiveMode === 'edit' ? 'Edit Skill' : 'Add Skill';

	if (!isOpen) return null;

	return createPortal(
		<dialog
			ref={dialogRef}
			className="modal"
			onClose={handleDialogClose}
			onCancel={e => {
				// Form mode: block Esc close. View mode: allow.
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

						{/* Name */}
						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Name*</span>
								<span className="label-text-alt tooltip tooltip-right" data-tip="Must be unique within the bundle.">
									<FiHelpCircle size={12} />
								</span>
							</label>
							<div className="col-span-9">
								<input
									ref={nameInputRef}
									type="text"
									name="name"
									value={formData.name}
									onChange={handleInput}
									readOnly={isViewMode}
									className={`input input-bordered w-full rounded-xl ${errors.name ? 'input-error' : ''}`}
									spellCheck="false"
									autoComplete="off"
									aria-invalid={Boolean(errors.name)}
								/>
								{errors.name && (
									<div className="label">
										<span className="label-text-alt text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.name}
										</span>
									</div>
								)}
							</div>
						</div>

						{/* Slug */}
						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Slug*</span>
								<span className="label-text-alt tooltip tooltip-right" data-tip="Lower-case, URL-friendly.">
									<FiHelpCircle size={12} />
								</span>
							</label>
							<div className="col-span-9">
								<input
									type="text"
									name="slug"
									value={formData.slug}
									onChange={handleInput}
									readOnly={isViewMode || isEditMode}
									className={`input input-bordered w-full rounded-xl ${errors.slug ? 'input-error' : ''}`}
									spellCheck="false"
									autoComplete="off"
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

						{/* Type */}
						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Type*</span>
							</label>
							<div className="col-span-9">
								{isEditMode || isViewMode ? (
									<ReadOnlyValue value={skillTypeDropdownItems[formData.type].displayName} />
								) : (
									<Dropdown<SkillType>
										dropdownItems={skillTypeDropdownItems}
										selectedKey={formData.type}
										onChange={onSkillTypeChange}
										filterDisabled={true}
										title="Select skill type"
										getDisplayName={k => skillTypeDropdownItems[k].displayName}
									/>
								)}
								{errors.type && (
									<div className="label">
										<span className="label-text-alt text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.type}
										</span>
									</div>
								)}
							</div>
						</div>

						{/* Location */}
						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Location*</span>
							</label>
							<div className="col-span-9">
								<input
									type="text"
									name="location"
									value={formData.location}
									onChange={handleInput}
									readOnly={isViewMode}
									className={`input input-bordered w-full rounded-xl ${errors.location ? 'input-error' : ''}`}
									spellCheck="false"
									autoComplete="off"
									aria-invalid={Boolean(errors.location)}
								/>
								{errors.location && (
									<div className="label">
										<span className="label-text-alt text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.location}
										</span>
									</div>
								)}
							</div>
						</div>

						{/* Display Name */}
						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Display Name</span>
							</label>
							<div className="col-span-9">
								<input
									type="text"
									name="displayName"
									value={formData.displayName}
									onChange={handleInput}
									readOnly={isViewMode}
									className="input input-bordered w-full rounded-xl"
									spellCheck="false"
									autoComplete="off"
								/>
							</div>
						</div>

						{/* Enabled */}
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
									className="toggle toggle-accent disabled:opacity-80"
									disabled={isViewMode}
								/>
							</div>
						</div>

						{/* Description */}
						<div className="grid grid-cols-12 items-center gap-2">
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

						{/* Tags */}
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

						{isViewMode && initialData?.skill && (
							<>
								<div className="divider">Metadata</div>
								<div className="grid grid-cols-12 gap-2 text-sm">
									<div className="col-span-3 font-semibold">ID</div>
									<div className="col-span-9">{initialData.skill.id}</div>
									<div className="col-span-3 font-semibold">Schema</div>
									<div className="col-span-9">{initialData.skill.schemaVersion}</div>
									<div className="col-span-3 font-semibold">Built-in</div>
									<div className="col-span-9">{initialData.skill.isBuiltIn ? 'Yes' : 'No'}</div>
									<div className="col-span-3 font-semibold">Created</div>
									<div className="col-span-9">{String(initialData.skill.createdAt)}</div>
									<div className="col-span-3 font-semibold">Modified</div>
									<div className="col-span-9">{String(initialData.skill.modifiedAt)}</div>
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
