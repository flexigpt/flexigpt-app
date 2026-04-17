import { type SubmitEventHandler, useCallback, useEffect, useRef, useState } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle, FiLink, FiX } from 'react-icons/fi';

import {
	createUrlFieldChangeHandler,
	type FieldErrorState,
	MessageEnterValidURL,
	validateUrlForInput,
} from '@/lib/url_utils';

type UrlAttachmentModalProps = {
	isOpen: boolean;
	onClose: () => void;
	onAttachURL: (url: string) => Promise<void> | void;
};

type FormState = {
	url: string;
};

type UrlAttachmentModalContentProps = Omit<UrlAttachmentModalProps, 'isOpen'>;

const INITIAL_FORM_STATE: FormState = { url: '' };

function UrlAttachmentModalContent({ onClose, onAttachURL }: UrlAttachmentModalContentProps) {
	const [formData, setFormData] = useState(INITIAL_FORM_STATE);
	const [errors, setErrors] = useState<FieldErrorState<FormState>>({});
	const [submitting, setSubmitting] = useState(false);

	const dialogRef = useRef<HTMLDialogElement | null>(null);
	const inputRef = useRef<HTMLInputElement | null>(null);
	const isUnmountingRef = useRef(false);

	const focusInputAtEnd = useCallback(() => {
		const input = inputRef.current;
		if (!input) return;
		input.focus({ preventScroll: true });
		const end = input.value.length;
		try {
			input.setSelectionRange(end, end);
		} catch {
			// ok.
		}
	}, []);

	useEffect(() => {
		const dialog = dialogRef.current;
		if (!dialog) return;

		if (!dialog.open) {
			dialog.showModal();
		}

		let raf1 = 0;
		let raf2 = 0;
		raf1 = window.requestAnimationFrame(() => {
			raf2 = window.requestAnimationFrame(() => {
				focusInputAtEnd();
			});
		});
		return () => {
			isUnmountingRef.current = true;
			window.cancelAnimationFrame(raf1);
			window.cancelAnimationFrame(raf2);
			if (dialog.open) {
				dialog.close();
			}
		};
	}, [focusInputAtEnd]);

	const handleDialogClose = () => {
		if (isUnmountingRef.current) return;
		onClose();
	};

	// URL field change handler (field is required)
	const handleUrlChange = createUrlFieldChangeHandler<FormState>('url', setFormData, setErrors, { required: true });

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = async e => {
		e.preventDefault();
		e.stopPropagation();

		const input = inputRef.current;

		// Use shared validator with required semantics
		const { normalized, error } = validateUrlForInput(formData.url, input, {
			required: true,
		});

		if (!normalized || error) {
			setErrors(prev => ({
				...prev,
				url: error ?? MessageEnterValidURL,
			}));
			focusInputAtEnd();

			return;
		}

		setSubmitting(true);
		try {
			await onAttachURL(normalized);
			dialogRef.current?.close();
		} catch (err) {
			setErrors(prev => ({
				...prev,
				url: (err as Error).message || 'Something went wrong while attaching the URL.',
			}));
			focusInputAtEnd();
		} finally {
			setSubmitting(false);
		}
	};

	const urlError = errors.url ?? null;

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
			<div className="modal-box bg-base-200 max-h-[80vh] max-w-xl overflow-auto rounded-2xl">
				{/* header */}
				<div className="mb-4 flex items-center justify-between">
					<h3 className="flex items-center gap-2 text-lg font-bold">
						<FiLink size={16} />
						<span>Attach Link</span>
					</h3>
					<button
						type="button"
						className="btn btn-sm btn-circle bg-base-300"
						onClick={() => dialogRef.current?.close()}
						aria-label="Close"
					>
						<FiX size={12} />
					</button>
				</div>

				{/* NOTE: noValidate disables the browser's popup UI, but we still read input.validity/checkValidity() in JS. */}
				<form noValidate onSubmit={handleSubmit} className="space-y-4">
					{/* URL input */}
					<div>
						<label className="label p-1">
							<span className="label-text text-sm">URL</span>
						</label>
						<input
							ref={inputRef}
							type="url"
							value={formData.url}
							onChange={handleUrlChange}
							className={`input input-bordered w-full rounded-xl ${urlError ? 'input-error' : ''}`}
							placeholder="https://example.com/resource OR example.com/resource"
							spellCheck="false"
						/>
						<p className="text-base-content/70 p-1 text-xs">Paste a single URL to attach to this message.</p>

						{/* Fixed-height error area to avoid layout shift */}
						<div className="mt-1 h-5 text-xs">
							{urlError && (
								<span className="text-error flex items-center gap-1">
									<FiAlertCircle size={12} /> {urlError}
								</span>
							)}
						</div>
					</div>

					{/* footer buttons */}
					<div className="modal-action">
						<button type="button" className="btn bg-base-300 rounded-xl" onClick={() => dialogRef.current?.close()}>
							Cancel
						</button>
						<button
							type="submit"
							className="btn btn-primary rounded-xl"
							// You can keep this simple: empty stays disabled even before first validation
							disabled={submitting || !formData.url.trim() || !!urlError}
						>
							{submitting ? (
								<>
									<span className="loading loading-spinner loading-xs" />
									Attaching…
								</>
							) : (
								'Attach'
							)}
						</button>
					</div>
				</form>
			</div>
			{/* NOTE: no modal-backdrop here: backdrop click should NOT close this modal */}
		</dialog>
	);
}

export function UrlAttachmentModal({ isOpen, onClose, onAttachURL }: UrlAttachmentModalProps) {
	if (!isOpen) return null;
	if (typeof document === 'undefined' || !document.body) return null;

	return createPortal(<UrlAttachmentModalContent onClose={onClose} onAttachURL={onAttachURL} />, document.body);
}
