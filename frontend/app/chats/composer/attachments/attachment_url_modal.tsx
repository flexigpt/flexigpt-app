import type { SubmitEventHandler } from 'react';
import { useCallback, useEffect, useRef, useState } from 'react';

import { FiAlertCircle, FiLink } from 'react-icons/fi';

import type { FieldErrorState } from '@/lib/url_utils';
import { createUrlFieldChangeHandler, MessageEnterValidURL, validateUrlForInput } from '@/lib/url_utils';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalHeader } from '@/components/modal/modal_header';

interface UrlAttachmentModalProps {
	isOpen: boolean;
	onClose: () => void;
	onAttachURL: (url: string) => Promise<void> | void;
}

interface FormState {
	url: string;
}

type UrlAttachmentModalContentProps = Omit<UrlAttachmentModalProps, 'isOpen'>;

const INITIAL_FORM_STATE: FormState = { url: '' };

function UrlAttachmentModalContent({ onAttachURL }: UrlAttachmentModalContentProps) {
	const [formData, setFormData] = useState(INITIAL_FORM_STATE);
	const [errors, setErrors] = useState<FieldErrorState<FormState>>({});
	const [submitting, setSubmitting] = useState(false);

	const { requestClose, unmountingRef } = useModalDialogController();
	const inputRef = useRef<HTMLInputElement | null>(null);

	const focusInputAtEnd = useCallback(() => {
		const input = inputRef.current;
		if (!input) {
			return;
		}
		input.focus({ preventScroll: true });
		const end = input.value.length;
		try {
			input.setSelectionRange(end, end);
		} catch {
			// ok.
		}
	}, []);

	useEffect(() => {
		let raf1 = 0;
		let raf2 = 0;
		raf1 = window.requestAnimationFrame(() => {
			raf2 = window.requestAnimationFrame(() => {
				focusInputAtEnd();
			});
		});
		return () => {
			window.cancelAnimationFrame(raf1);
			window.cancelAnimationFrame(raf2);
		};
	}, [focusInputAtEnd]);

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
			if (!unmountingRef.current) {
				requestClose(true);
			}
		} catch (err) {
			if (!unmountingRef.current) {
				setErrors(prev => ({
					...prev,
					url: (err as Error).message || 'Something went wrong while attaching the URL.',
				}));
				focusInputAtEnd();
			}
		} finally {
			if (!unmountingRef.current) {
				setSubmitting(false);
			}
		}
	};

	const urlError = errors.url ?? null;

	return (
		<>
			<div className="modal-box bg-base-200 max-h-[80vh] max-w-xl overflow-auto rounded-2xl p-0">
				<ModalHeader
					title={
						<span className="flex items-center gap-2">
							<FiLink size={16} />
							<span>Attach Link</span>
						</span>
					}
					onClose={() => {
						requestClose();
					}}
					closeDisabled={submitting}
				/>

				{/* NOTE: noValidate disables the browser's popup UI, but we still read input.validity/checkValidity() in JS. */}
				<form noValidate onSubmit={handleSubmit} className="space-y-4 p-6">
					{/* URL input */}
					<div>
						<label htmlFor="attachment-url" className="label p-1">
							<span className="text-sm">URL</span>
						</label>
						<input
							id="attachment-url"
							ref={inputRef}
							type="url"
							value={formData.url}
							onChange={handleUrlChange}
							className={`input w-full rounded-xl ${urlError ? 'input-error' : ''}`}
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
					<ModalActions className="-mx-6 mt-6 -mb-6">
						<button
							type="button"
							className="btn bg-base-300 rounded-xl"
							onClick={() => {
								requestClose();
							}}
							disabled={submitting}
						>
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
					</ModalActions>
				</form>
			</div>
			{/* NOTE: no modal-backdrop here: backdrop click should NOT close this modal */}
		</>
	);
}

export function UrlAttachmentModal({ isOpen, onClose, onAttachURL }: UrlAttachmentModalProps) {
	if (!isOpen) {
		return null;
	}

	return (
		<ModalDialog isOpen={isOpen} onClose={onClose} blockCancel>
			<UrlAttachmentModalContent onClose={onClose} onAttachURL={onAttachURL} />
		</ModalDialog>
	);
}
