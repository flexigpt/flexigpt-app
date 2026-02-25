import { type Dispatch, type SetStateAction, type SubmitEventHandler, useEffect, useRef, useState } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle, FiHelpCircle, FiX } from 'react-icons/fi';

import type { UIChatOption } from '@/spec/modelpreset';

type AdvancedParamsModalProps = {
	isOpen: boolean;
	onClose: () => void;
	currentModel: UIChatOption;
	onSave: (updatedModel: UIChatOption) => void;
};

export function AdvancedParamsModal({ isOpen, onClose, currentModel, onSave }: AdvancedParamsModalProps) {
	/* local form state (strings for easy blank entry) */
	const [stream, setStream] = useState(false);
	const [maxPromptLength, setMaxPromptLength] = useState('');
	const [maxOutputLength, setMaxOutputLength] = useState('');
	const [timeout, setTimeout] = useState('');

	/* validation errors */
	const [errors, setErrors] = useState<Partial<Record<'maxPromptLength' | 'maxOutputLength' | 'timeout', string>>>({});

	const dialogRef = useRef<HTMLDialogElement | null>(null);

	/* reset form every time the modal opens or the model changes */
	useEffect(() => {
		if (!isOpen) return;

		setStream(currentModel.stream);
		setMaxPromptLength(String(currentModel.maxPromptLength));
		setMaxOutputLength(String(currentModel.maxOutputLength));
		setTimeout(String(currentModel.timeout));

		setErrors({});
	}, [isOpen, currentModel]);

	// Open the dialog natively when isOpen becomes true
	useEffect(() => {
		if (!isOpen) return;

		const dialog = dialogRef.current;
		if (!dialog) return;

		if (!dialog.open) {
			dialog.showModal();
		}

		return () => {
			// If the component unmounts while the dialog is still open, close it.
			if (dialog.open) {
				dialog.close();
			}
		};
	}, [isOpen]);

	// Sync parent state whenever the dialog is closed (Esc or dialog.close()).
	const handleDialogClose = () => {
		onClose();
	};

	const validateNumberField = (field: 'maxPromptLength' | 'maxOutputLength' | 'timeout', value: string) => {
		const num = value.trim() === '' ? undefined : Number(value.trim());

		return num && Number.isFinite(num) && num > 0 ? undefined : `${field} must be a positive number.`;
	};

	const updateField = (
		field: 'maxPromptLength' | 'maxOutputLength' | 'timeout',
		value: string,
		setter: Dispatch<SetStateAction<string>>
	) => {
		setter(value);
		setErrors(prev => ({ ...prev, [field]: validateNumberField(field, value) }));
	};

	const formHasErrors = Object.values(errors).some(Boolean);

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = e => {
		e.preventDefault();

		/* final synchronous validation */
		const maxPromptErr = validateNumberField('maxPromptLength', maxPromptLength);
		const maxOutputErr = validateNumberField('maxOutputLength', maxOutputLength);
		const timeoutErr = validateNumberField('timeout', timeout);

		if (maxPromptErr || maxOutputErr || timeoutErr) {
			setErrors({
				maxPromptLength: maxPromptErr,
				maxOutputLength: maxOutputErr,
				timeout: timeoutErr,
			});
			return;
		}

		const updatedModel: UIChatOption = {
			...currentModel,
			stream,
			maxPromptLength: maxPromptLength.trim() === '' ? currentModel.maxPromptLength : Number(maxPromptLength.trim()),
			maxOutputLength: maxOutputLength.trim() === '' ? currentModel.maxOutputLength : Number(maxOutputLength.trim()),
			timeout: timeout.trim() === '' ? currentModel.timeout : Number(timeout.trim()),
		};

		onSave(updatedModel);

		dialogRef.current?.close();
	};

	if (!isOpen) return null;

	return createPortal(
		<dialog ref={dialogRef} className="modal" onClose={handleDialogClose}>
			<div className="modal-box bg-base-200 max-h-[80vh] max-w-xl overflow-auto rounded-2xl">
				{/* header */}
				<div className="mb-4 flex items-center justify-between">
					<h3 className="text-lg font-bold">Advanced Model Parameters</h3>
					<button
						type="button"
						className="btn btn-sm btn-circle bg-base-300"
						onClick={() => dialogRef.current?.close()}
						aria-label="Close"
					>
						<FiX size={12} />
					</button>
				</div>

				<form onSubmit={handleSubmit} className="space-y-4">
					{/* stream toggle */}
					<div className="grid grid-cols-12 items-center gap-2">
						<label className="label col-span-4 cursor-pointer">
							<span className="label-text text-sm">Streaming</span>
							<span className="label-text-alt tooltip tooltip-right" data-tip="Stream data continuously.">
								<FiHelpCircle size={12} />
							</span>
						</label>
						<div className="col-span-8">
							<input
								type="checkbox"
								checked={stream}
								onChange={e => {
									setStream(e.target.checked);
								}}
								className="toggle toggle-accent"
							/>
						</div>
					</div>

					{/* max prompt length */}
					<div className="grid grid-cols-12 items-center gap-2">
						<label className="label col-span-4">
							<span className="label-text text-sm">Max Prompt Tokens</span>
							<span className="label-text-alt tooltip tooltip-right" data-tip="Maximum tokens for input prompt">
								<FiHelpCircle size={12} />
							</span>
						</label>
						<div className="col-span-8">
							<input
								type="text"
								value={maxPromptLength}
								onChange={e => {
									updateField('maxPromptLength', e.target.value, setMaxPromptLength);
								}}
								className={`input input-bordered w-full rounded-xl ${errors.maxPromptLength ? 'input-error' : ''}`}
								placeholder={`Default: ${currentModel.maxPromptLength}`}
								spellCheck="false"
							/>
							{errors.maxPromptLength && (
								<div className="label">
									<span className="label-text-alt text-error flex items-center gap-1">
										<FiAlertCircle size={12} /> {errors.maxPromptLength}
									</span>
								</div>
							)}
						</div>
					</div>

					{/* max output length */}
					<div className="grid grid-cols-12 items-center gap-2">
						<label className="label col-span-4">
							<span className="label-text text-sm">Max Output Tokens</span>
							<span className="label-text-alt tooltip tooltip-right" data-tip="Maximum tokens for model output">
								<FiHelpCircle size={12} />
							</span>
						</label>
						<div className="col-span-8">
							<input
								type="text"
								value={maxOutputLength}
								onChange={e => {
									updateField('maxOutputLength', e.target.value, setMaxOutputLength);
								}}
								className={`input input-bordered w-full rounded-xl ${errors.maxOutputLength ? 'input-error' : ''}`}
								placeholder={`Default: ${currentModel.maxOutputLength}`}
								spellCheck="false"
							/>
							{errors.maxOutputLength && (
								<div className="label">
									<span className="label-text-alt text-error flex items-center gap-1">
										<FiAlertCircle size={12} /> {errors.maxOutputLength}
									</span>
								</div>
							)}
						</div>
					</div>

					{/* timeout */}
					<div className="grid grid-cols-12 items-center gap-2">
						<label className="label col-span-4">
							<span className="label-text text-sm">Timeout&nbsp;(s)</span>
							<span
								className="label-text-alt tooltip tooltip-right"
								data-tip="Maximum time a request can take (seconds)"
							>
								<FiHelpCircle size={12} />
							</span>
						</label>
						<div className="col-span-8">
							<input
								type="text"
								value={timeout}
								onChange={e => {
									updateField('timeout', e.target.value, setTimeout);
								}}
								className={`input input-bordered w-full rounded-xl ${errors.timeout ? 'input-error' : ''}`}
								placeholder={`Default: ${currentModel.timeout}`}
								spellCheck="false"
							/>
							{errors.timeout && (
								<div className="label">
									<span className="label-text-alt text-error flex items-center gap-1">
										<FiAlertCircle size={12} /> {errors.timeout}
									</span>
								</div>
							)}
						</div>
					</div>

					{/* footer buttons */}
					<div className="modal-action">
						<button type="button" className="btn bg-base-300 rounded-xl" onClick={() => dialogRef.current?.close()}>
							Cancel
						</button>
						<button type="submit" className="btn btn-primary rounded-xl" disabled={formHasErrors}>
							Save
						</button>
					</div>
				</form>
			</div>
		</dialog>,
		document.body
	);
}
