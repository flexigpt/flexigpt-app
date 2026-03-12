import { type SubmitEventHandler, useEffect, useMemo, useRef, useState } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle, FiHelpCircle, FiX } from 'react-icons/fi';

import { omitManyKeys } from '@/lib/obj_utils';
import { validateSlug } from '@/lib/text_utils';

interface AddSkillBundleModalProps {
	isOpen: boolean;
	onClose: () => void;
	onSubmit: (slug: string, display: string, description?: string) => void;
	existingSlugs: string[];
	existingNames: string[]; // bundle name uniqueness constraint
}

type ErrorState = {
	slug?: string;
	displayName?: string;
};

type BundleFormData = {
	slug: string;
	displayName: string;
	description: string;
};

function normalizeForUniq(s: string) {
	return s.trim().toLowerCase();
}

function getInitialFormData(): BundleFormData {
	return {
		slug: '',
		displayName: '',
		description: '',
	};
}

function AddSkillBundleModalContent({ onClose, onSubmit, existingSlugs, existingNames }: AddSkillBundleModalProps) {
	const [formData, setFormData] = useState<BundleFormData>(() => getInitialFormData());
	const [errors, setErrors] = useState<ErrorState>({});

	const dialogRef = useRef<HTMLDialogElement | null>(null);
	const isUnmountingRef = useRef(false);

	useEffect(() => {
		const dialog = dialogRef.current;
		if (!dialog) return;

		if (!dialog.open) {
			try {
				dialog.showModal();
			} catch {
				// Ignore if the dialog cannot be shown; keep rendering safely.
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

	const validateField = (field: keyof ErrorState, val: string, currentErrors: ErrorState): ErrorState => {
		const v = val.trim();
		let nextErrors: ErrorState = { ...currentErrors };

		if (!v) {
			nextErrors[field] = 'This field is required.';
			return nextErrors;
		}

		if (field === 'slug') {
			const err = validateSlug(v);
			if (err) {
				nextErrors.slug = err;
			} else if (existingSlugs.includes(v)) {
				nextErrors.slug = 'Slug already in use.';
			} else {
				nextErrors = omitManyKeys(nextErrors, ['slug']);
			}
			return nextErrors;
		}

		if (field === 'displayName') {
			const norm = normalizeForUniq(v);
			const clash = existingNames.some(x => normalizeForUniq(x) === norm);
			if (clash) nextErrors.displayName = 'Bundle name must be unique.';
			else nextErrors = omitManyKeys(nextErrors, ['displayName']);
			return nextErrors;
		}

		return omitManyKeys(nextErrors, [field]);
	};

	const validateForm = (state: BundleFormData): ErrorState => {
		let next: ErrorState = {};
		next = validateField('slug', state.slug, next);
		next = validateField('displayName', state.displayName, next);
		return next;
	};

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = e => {
		e.preventDefault();
		e.stopPropagation();

		const trimmed: BundleFormData = {
			slug: formData.slug.trim(),
			displayName: formData.displayName.trim(),
			description: formData.description.trim(),
		};

		const nextErrors = validateForm(trimmed);
		setErrors(nextErrors);
		if (Object.keys(nextErrors).length > 0) return;

		onSubmit(trimmed.slug, trimmed.displayName, trimmed.description || undefined);
		requestClose();
	};

	const isFormValid = useMemo(
		() => Boolean(formData.slug.trim()) && Boolean(formData.displayName.trim()) && Object.keys(errors).length === 0,
		[formData.slug, formData.displayName, errors]
	);

	return (
		<dialog
			ref={dialogRef}
			className="modal"
			onClose={handleDialogClose}
			onCancel={e => {
				// Form mode: do NOT allow Esc to close.
				e.preventDefault();
			}}
		>
			<div className="modal-box bg-base-200 max-h-[80vh] max-w-3xl overflow-auto rounded-2xl">
				<div className="mb-4 flex items-center justify-between">
					<h3 className="text-lg font-bold">Add Skill Bundle</h3>
					<button type="button" className="btn btn-sm btn-circle bg-base-300" onClick={requestClose} aria-label="Close">
						<FiX size={12} />
					</button>
				</div>

				<form noValidate onSubmit={handleSubmit} className="space-y-4">
					<div className="grid grid-cols-12 items-center gap-2">
						<label className="label col-span-3">
							<span className="label-text text-sm">Bundle Slug*</span>
							<span className="label-text-alt tooltip tooltip-right" data-tip="Lower-case, URL-friendly.">
								<FiHelpCircle size={12} />
							</span>
						</label>
						<div className="col-span-9">
							<input
								type="text"
								className={`input input-bordered w-full rounded-xl ${errors.slug ? 'input-error' : ''}`}
								value={formData.slug}
								onChange={e => {
									const value = e.target.value;
									setFormData(prev => ({ ...prev, slug: value }));
									setErrors(prevErrors => validateField('slug', value, prevErrors));
								}}
								spellCheck="false"
								autoComplete="off"
								autoFocus
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

					<div className="grid grid-cols-12 items-center gap-2">
						<label className="label col-span-3">
							<span className="label-text text-sm">Display Name*</span>
						</label>
						<div className="col-span-9">
							<input
								type="text"
								className={`input input-bordered w-full rounded-xl ${errors.displayName ? 'input-error' : ''}`}
								value={formData.displayName}
								onChange={e => {
									const value = e.target.value;
									setFormData(prev => ({ ...prev, displayName: value }));
									setErrors(prevErrors => validateField('displayName', value, prevErrors));
								}}
								spellCheck="false"
								autoComplete="off"
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

					<div className="grid grid-cols-12 items-start gap-2">
						<label className="label col-span-3">
							<span className="label-text text-sm">Description</span>
						</label>
						<div className="col-span-9">
							<textarea
								className="textarea textarea-bordered h-24 w-full rounded-xl"
								value={formData.description}
								onChange={e => {
									setFormData(prev => ({ ...prev, description: e.target.value }));
								}}
								spellCheck="false"
							/>
						</div>
					</div>

					<div className="modal-action">
						<button type="button" className="btn bg-base-300 rounded-xl" onClick={requestClose}>
							Cancel
						</button>
						<button type="submit" className="btn btn-primary rounded-xl" disabled={!isFormValid}>
							Create
						</button>
					</div>
				</form>
			</div>

			{/* No modal-backdrop: backdrop click should NOT close */}
		</dialog>
	);
}

export function AddSkillBundleModal(props: AddSkillBundleModalProps) {
	if (!props.isOpen) return null;
	if (typeof document === 'undefined' || !document.body) return null;

	return createPortal(<AddSkillBundleModalContent key="add-skill-bundle-modal" {...props} />, document.body);
}
