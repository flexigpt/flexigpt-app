import { type SubmitEventHandler, useEffect, useMemo, useRef, useState } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle, FiHelpCircle, FiX } from 'react-icons/fi';

import { omitManyKeys } from '@/lib/obj_utils';
import { validateSlug } from '@/lib/text_utils';

interface AddBundleModalProps {
	isOpen: boolean;
	onClose: () => void;
	onSubmit: (slug: string, display: string, description?: string) => void;
	existingSlugs: string[];
}

type ErrorState = {
	slug?: string;
	displayName?: string;
};

export function AddBundleModal({ isOpen, onClose, onSubmit, existingSlugs }: AddBundleModalProps) {
	const [form, setForm] = useState({
		slug: '',
		displayName: '',
		description: '',
	});
	const [errors, setErrors] = useState<ErrorState>({});

	const dialogRef = useRef<HTMLDialogElement | null>(null);

	// Reset local state whenever the modal is opened
	useEffect(() => {
		if (!isOpen) return;
		setForm({ slug: '', displayName: '', description: '' });
		setErrors({});
	}, [isOpen]);

	// Open/close the native <dialog> when isOpen changes
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

	// Sync parent state when the dialog closes (Esc or programmatic close)
	const handleDialogClose = () => {
		onClose();
	};

	const validateField = (field: keyof ErrorState, val: string, currentErrors: ErrorState): ErrorState => {
		const v = val.trim();
		let nextErrors: ErrorState = { ...currentErrors };

		if (!v) {
			nextErrors[field] = 'This field is required.';
		} else if (field === 'slug') {
			const err = validateSlug(v);
			if (err) {
				nextErrors.slug = err;
			} else if (existingSlugs.includes(v)) {
				nextErrors.slug = 'Slug already in use.';
			} else {
				nextErrors = omitManyKeys(nextErrors, ['slug']);
			}
		} else {
			nextErrors = omitManyKeys(nextErrors, [field]);
		}

		return nextErrors;
	};

	const validateForm = (state: typeof form): ErrorState => {
		let nextErrors: ErrorState = {};
		nextErrors = validateField('slug', state.slug, nextErrors);
		nextErrors = validateField('displayName', state.displayName, nextErrors);
		return nextErrors;
	};

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = e => {
		if (e) {
			e.preventDefault();
			e.stopPropagation();
		}

		const trimmed = {
			slug: form.slug.trim(),
			displayName: form.displayName.trim(),
			description: form.description.trim(),
		};

		const nextErrors = validateForm(trimmed);
		setErrors(nextErrors);

		if (Object.keys(nextErrors).length > 0) return;

		onSubmit(trimmed.slug, trimmed.displayName, trimmed.description || undefined);
		dialogRef.current?.close();
	};

	const isFormValid = useMemo(
		() => Boolean(form.slug.trim()) && Boolean(form.displayName.trim()) && Object.keys(errors).length === 0,
		[form.slug, form.displayName, errors]
	);

	if (!isOpen) return null;

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
			<div className="modal-box bg-base-200 max-h-[80vh] max-w-3xl overflow-hidden rounded-2xl p-0">
				<div className="max-h-[80vh] overflow-y-auto p-6">
					{/* header */}
					<div className="mb-4 flex items-center justify-between">
						<h3 className="text-lg font-bold">Add Prompt Bundle</h3>
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
						{/* Slug */}
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
									value={form.slug}
									onChange={e => {
										const value = e.target.value;
										setForm(p => ({ ...p, slug: value }));
										setErrors(prev => validateField('slug', value, prev));
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

						{/* Display Name */}
						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Display Name*</span>
							</label>

							<div className="col-span-9">
								<input
									type="text"
									className={`input input-bordered w-full rounded-xl ${errors.displayName ? 'input-error' : ''}`}
									value={form.displayName}
									onChange={e => {
										const value = e.target.value;
										setForm(p => ({ ...p, displayName: value }));
										setErrors(prev => validateField('displayName', value, prev));
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

						{/* Description */}
						<div className="grid grid-cols-12 items-start gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Description</span>
							</label>

							<div className="col-span-9">
								<textarea
									className="textarea textarea-bordered h-24 w-full rounded-xl"
									value={form.description}
									onChange={e => {
										const value = e.target.value;
										setForm(p => ({ ...p, description: value }));
									}}
									spellCheck="false"
								/>
							</div>
						</div>

						{/* Actions */}
						<div className="modal-action">
							<button type="button" className="btn bg-base-300 rounded-xl" onClick={() => dialogRef.current?.close()}>
								Cancel
							</button>
							<button type="submit" className="btn btn-primary rounded-xl" disabled={!isFormValid}>
								Create
							</button>
						</div>
					</form>
				</div>
			</div>
			{/* NOTE: no modal-backdrop here: backdrop click should NOT close this modal */}
		</dialog>,
		document.body
	);
}
