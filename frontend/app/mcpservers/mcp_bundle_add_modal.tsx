import type { SubmitEventHandler } from 'react';
import { useEffect, useMemo, useRef, useState } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle, FiHelpCircle, FiX } from 'react-icons/fi';

import { omitManyKeys } from '@/lib/obj_utils';
import { validateSlug } from '@/lib/text_utils';

interface AddMCPBundleModalProps {
	isOpen: boolean;
	onClose: () => void;
	onSubmit: (slug: string, display: string, description?: string) => Promise<void>;
	existingSlugs: string[];
}

interface ErrorState {
	slug?: string;
	displayName?: string;
}

interface BundleFormData {
	slug: string;
	displayName: string;
	description: string;
}

function getInitialFormData(): BundleFormData {
	return {
		slug: '',
		displayName: '',
		description: '',
	};
}

function AddMCPBundleModalContent({ onClose, onSubmit, existingSlugs }: AddMCPBundleModalProps) {
	const [formData, setFormData] = useState<BundleFormData>(() => getInitialFormData());
	const [errors, setErrors] = useState<ErrorState>({});
	const [submitError, setSubmitError] = useState('');
	const [isSubmitting, setIsSubmitting] = useState(false);

	const dialogRef = useRef<HTMLDialogElement | null>(null);
	const isUnmountingRef = useRef(false);

	useEffect(() => {
		const dialog = dialogRef.current;
		if (!dialog) {
			return;
		}

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
		if (isSubmitting) {
			return;
		}

		const dialog = dialogRef.current;

		if (dialog?.open) {
			dialog.close();
			return;
		}

		onClose();
	};

	const handleDialogClose = () => {
		if (isUnmountingRef.current) {
			return;
		}
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

	const validateForm = (state: BundleFormData): ErrorState => {
		let nextErrors: ErrorState = {};
		nextErrors = validateField('slug', state.slug, nextErrors);
		nextErrors = validateField('displayName', state.displayName, nextErrors);
		return nextErrors;
	};

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = async e => {
		e.preventDefault();
		e.stopPropagation();

		if (isSubmitting) {
			return;
		}

		const trimmed: BundleFormData = {
			slug: formData.slug.trim(),
			displayName: formData.displayName.trim(),
			description: formData.description.trim(),
		};

		const nextErrors = validateForm(trimmed);
		setErrors(nextErrors);

		if (Object.keys(nextErrors).length > 0) {
			return;
		}

		setSubmitError('');
		setIsSubmitting(true);
		try {
			await onSubmit(trimmed.slug, trimmed.displayName, trimmed.description || undefined);
			if (!isUnmountingRef.current) {
				const dialog = dialogRef.current;
				if (dialog?.open) {
					dialog.close();
				}
			}
		} catch (error) {
			if (!isUnmountingRef.current) {
				setSubmitError(error instanceof Error ? error.message : 'Failed to create MCP bundle.');
			}
		} finally {
			if (!isUnmountingRef.current) {
				setIsSubmitting(false);
			}
		}
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
				e.preventDefault();
			}}
		>
			<div className="modal-box bg-base-200 max-h-[80vh] max-w-3xl overflow-hidden rounded-2xl p-0">
				<div className="max-h-[80vh] overflow-y-auto p-6">
					<div className="mb-4 flex items-center justify-between">
						<h3 className="text-lg font-bold">Add MCP Bundle</h3>
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
						{submitError ? (
							<div className="alert alert-error rounded-2xl text-sm" role="alert">
								<FiAlertCircle size={14} />
								<span>{submitError}</span>
							</div>
						) : null}

						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-12 sm:col-span-3">
								<span className="text-sm">Bundle Slug*</span>
								<span className="tooltip tooltip-right" data-tip="Lower-case, URL-friendly.">
									<FiHelpCircle size={12} />
								</span>
							</label>

							<div className="col-span-12 sm:col-span-9">
								<input
									type="text"
									className={`input w-full rounded-xl ${errors.slug ? 'input-error' : ''}`}
									value={formData.slug}
									onChange={e => {
										const value = e.target.value;
										setFormData(prev => ({ ...prev, slug: value }));
										setErrors(prev => validateField('slug', value, prev));
									}}
									spellCheck="false"
									autoComplete="off"
									autoFocus
									aria-invalid={Boolean(errors.slug)}
								/>
								{errors.slug && (
									<div className="label">
										<span className="text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.slug}
										</span>
									</div>
								)}
							</div>
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="text-sm">Display Name*</span>
							</label>

							<div className="col-span-9">
								<input
									type="text"
									className={`input w-full rounded-xl ${errors.displayName ? 'input-error' : ''}`}
									value={formData.displayName}
									onChange={e => {
										const value = e.target.value;
										setFormData(prev => ({ ...prev, displayName: value }));
										setErrors(prev => validateField('displayName', value, prev));
									}}
									spellCheck="false"
									autoComplete="off"
									aria-invalid={Boolean(errors.displayName)}
								/>
								{errors.displayName && (
									<div className="label">
										<span className="text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.displayName}
										</span>
									</div>
								)}
							</div>
						</div>

						<div className="grid grid-cols-12 items-start gap-2">
							<label className="label col-span-3">
								<span className="text-sm">Description</span>
							</label>

							<div className="col-span-9">
								<textarea
									className="textarea h-24 w-full rounded-xl"
									value={formData.description}
									onChange={e => {
										const value = e.target.value;
										setFormData(prev => ({ ...prev, description: value }));
									}}
									spellCheck="false"
								/>
							</div>
						</div>

						<div className="modal-action">
							<button
								type="button"
								className="btn bg-base-300 rounded-xl"
								onClick={requestClose}
								disabled={isSubmitting}
							>
								Cancel
							</button>
							<button type="submit" className="btn btn-primary rounded-xl" disabled={!isFormValid || isSubmitting}>
								{isSubmitting ? 'Creating...' : 'Create'}
							</button>
						</div>
					</form>
				</div>
			</div>
		</dialog>
	);
}

export function AddMCPBundleModal(props: AddMCPBundleModalProps) {
	if (!props.isOpen) {
		return null;
	}
	if (typeof document === 'undefined' || !document.body) {
		return null;
	}

	return createPortal(<AddMCPBundleModalContent key="add-mcp-bundle-modal" {...props} />, document.body);
}
