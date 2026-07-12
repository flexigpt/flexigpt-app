import type { SubmitEventHandler } from 'react';
import { useId, useMemo, useState } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle } from 'react-icons/fi';

import { validateSlug } from '@/lib/text_utils';

import { useDialogController } from '@/hooks/use_dialog_controller';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalField } from '@/components/modal/modal_field';
import { ModalHeader } from '@/components/modal/modal_header';

interface ManagementBundleCreateModalProps {
	isOpen: boolean;
	title: string;
	entityLabel: string;
	onClose: () => void;
	onSubmit: (slug: string, displayName: string, description?: string) => Promise<void>;
	existingSlugs: readonly string[];
	existingDisplayNames?: readonly string[];
	failureMessage: string;
}

interface BundleFormData {
	slug: string;
	displayName: string;
	description: string;
}

interface BundleValidationErrors {
	slug?: string;
	displayName?: string;
}

const EMPTY_FORM: BundleFormData = {
	slug: '',
	displayName: '',
	description: '',
};

function normalizeIdentity(value: string): string {
	return value.trim().toLowerCase();
}

function getErrorMessage(error: unknown, fallback: string): string {
	if (error instanceof Error && error.message.trim()) {
		return error.message;
	}
	return fallback;
}

function validateBundleForm(
	formData: BundleFormData,
	existingSlugs: readonly string[],
	existingDisplayNames: readonly string[]
): BundleValidationErrors {
	const errors: BundleValidationErrors = {};
	const slug = formData.slug.trim();
	const displayName = formData.displayName.trim();

	if (!slug) {
		errors.slug = 'This field is required.';
	} else {
		const slugError = validateSlug(slug);
		if (slugError) {
			errors.slug = slugError;
		} else if (existingSlugs.some(existing => normalizeIdentity(existing) === normalizeIdentity(slug))) {
			errors.slug = 'Slug already in use.';
		}
	}

	if (!displayName) {
		errors.displayName = 'This field is required.';
	} else if (existingDisplayNames.some(existing => normalizeIdentity(existing) === normalizeIdentity(displayName))) {
		errors.displayName = `${formData.displayName.trim()} is already used by another bundle.`;
	}

	return errors;
}

function ManagementBundleCreateModalContent({
	title,
	entityLabel,
	onClose,
	onSubmit,
	existingSlugs,
	existingDisplayNames = [],
	failureMessage,
}: ManagementBundleCreateModalProps) {
	const [formData, setFormData] = useState<BundleFormData>(EMPTY_FORM);
	const [touched, setTouched] = useState<Partial<Record<keyof BundleFormData, boolean>>>({});
	const [submitted, setSubmitted] = useState(false);
	const [submitError, setSubmitError] = useState('');
	const [isSubmitting, setIsSubmitting] = useState(false);

	const slugID = useId();
	const displayNameID = useId();
	const descriptionID = useId();

	const validationErrors = useMemo(
		() => validateBundleForm(formData, existingSlugs, existingDisplayNames),
		[existingDisplayNames, existingSlugs, formData]
	);

	const { dialogRef, requestClose, handleClose, handleCancel, unmountingRef } = useDialogController({
		onClose,
		blockCancel: true,
		isBusy: isSubmitting,
	});

	const updateField = (field: keyof BundleFormData, value: string) => {
		setFormData(previous => ({ ...previous, [field]: value }));
		if (submitError) {
			setSubmitError('');
		}
	};

	const markTouched = (field: keyof BundleFormData) => {
		setTouched(previous => ({ ...previous, [field]: true }));
	};

	const visibleError = (field: keyof BundleValidationErrors): string | undefined =>
		submitted || touched[field] ? validationErrors[field] : undefined;

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = async event => {
		event.preventDefault();
		event.stopPropagation();

		if (isSubmitting) {
			return;
		}

		setSubmitted(true);
		setSubmitError('');

		if (Object.keys(validationErrors).length > 0) {
			return;
		}

		const slug = formData.slug.trim();
		const displayName = formData.displayName.trim();
		const description = formData.description.trim();

		setIsSubmitting(true);
		try {
			await onSubmit(slug, displayName, description || undefined);
			if (!unmountingRef.current) {
				requestClose(true);
			}
		} catch (error) {
			if (!unmountingRef.current) {
				setSubmitError(getErrorMessage(error, failureMessage));
			}
		} finally {
			if (!unmountingRef.current) {
				setIsSubmitting(false);
			}
		}
	};

	return (
		<dialog
			ref={dialogRef}
			className="modal"
			onClose={handleClose}
			onCancel={handleCancel}
			aria-labelledby={`${displayNameID}-modal-title`}
		>
			<div className="modal-box bg-base-200 flex max-h-[85vh] w-[calc(100%-1rem)] max-w-3xl flex-col overflow-hidden rounded-2xl p-0">
				<form noValidate onSubmit={handleSubmit} className="flex min-h-0 flex-1 flex-col" aria-busy={isSubmitting}>
					<ModalHeader
						title={<span id={`${displayNameID}-modal-title`}>{title}</span>}
						description={`Create a custom ${entityLabel.toLowerCase()} container.`}
						onClose={() => {
							requestClose();
						}}
						closeDisabled={isSubmitting}
					/>

					<div className="app-scrollbar-thin min-h-0 flex-1 space-y-4 overflow-y-auto px-4 py-5 sm:px-6">
						{submitError ? (
							<div className="alert alert-error rounded-2xl text-sm" role="alert">
								<FiAlertCircle className="shrink-0" size={14} />
								<span className="wrap-break-word">{submitError}</span>
							</div>
						) : null}

						<ModalField
							label="Bundle Slug"
							htmlFor={slugID}
							required
							hint="Lower-case, URL-friendly identifier."
							error={visibleError('slug')}
						>
							<input
								id={slugID}
								type="text"
								className={`input w-full rounded-xl ${visibleError('slug') ? 'input-error' : ''}`}
								value={formData.slug}
								onChange={event => {
									updateField('slug', event.currentTarget.value);
								}}
								onBlur={() => {
									markTouched('slug');
								}}
								spellCheck="false"
								autoComplete="off"
								autoFocus
								disabled={isSubmitting}
								aria-invalid={Boolean(visibleError('slug'))}
							/>
						</ModalField>

						<ModalField label="Display Name" htmlFor={displayNameID} required error={visibleError('displayName')}>
							<input
								id={displayNameID}
								type="text"
								className={`input w-full rounded-xl ${visibleError('displayName') ? 'input-error' : ''}`}
								value={formData.displayName}
								onChange={event => {
									updateField('displayName', event.currentTarget.value);
								}}
								onBlur={() => {
									markTouched('displayName');
								}}
								spellCheck="false"
								autoComplete="off"
								disabled={isSubmitting}
								aria-invalid={Boolean(visibleError('displayName'))}
							/>
						</ModalField>

						<ModalField label="Description" htmlFor={descriptionID} align="start">
							<textarea
								id={descriptionID}
								className="textarea min-h-24 w-full rounded-xl"
								value={formData.description}
								onChange={event => {
									updateField('description', event.currentTarget.value);
								}}
								spellCheck="false"
								disabled={isSubmitting}
							/>
						</ModalField>
					</div>

					<ModalActions>
						<button
							type="button"
							className="btn bg-base-300 rounded-xl"
							onClick={() => {
								requestClose();
							}}
							disabled={isSubmitting}
						>
							Cancel
						</button>
						<button
							type="submit"
							className="btn btn-primary rounded-xl"
							disabled={Object.keys(validationErrors).length > 0 || isSubmitting}
						>
							{isSubmitting ? 'Creating...' : 'Create'}
						</button>
					</ModalActions>
				</form>
			</div>
		</dialog>
	);
}

export function ManagementBundleCreateModal(props: ManagementBundleCreateModalProps) {
	if (!props.isOpen || typeof document === 'undefined' || !document.body) {
		return null;
	}

	return createPortal(
		<ManagementBundleCreateModalContent key={`${props.entityLabel}:create-bundle`} {...props} />,
		document.body
	);
}
