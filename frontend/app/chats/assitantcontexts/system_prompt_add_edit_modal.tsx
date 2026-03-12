import { type SubmitEventHandler, useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { createPortal } from 'react-dom';

import { FiCopy, FiX } from 'react-icons/fi';

import { Dropdown } from '@/components/dropdown';

type SystemPromptItem = {
	id: string;
	title: string;
	prompt: string;
	locked?: boolean;
};

type SystemPromptAddEditModalProps = {
	isOpen: boolean;
	mode: 'add' | 'edit';
	initialValue?: string;
	promptsForCopy?: SystemPromptItem[];
	onClose: () => void;
	onSave: (value: string) => void;
};

type SystemPromptAddEditModalInnerProps = Omit<SystemPromptAddEditModalProps, 'isOpen'>;

function closeDialogSafely(dialog: HTMLDialogElement | null): boolean {
	if (!dialog?.open) return false;

	try {
		dialog.close();
		return true;
	} catch {
		return false;
	}
}

export function SystemPromptAddEditModal({
	isOpen,
	mode,
	initialValue = '',
	promptsForCopy = [],
	onClose,
	onSave,
}: SystemPromptAddEditModalProps) {
	if (!isOpen || typeof document === 'undefined') return null;

	return createPortal(
		<SystemPromptAddEditModalInner
			key={`${mode}::${initialValue}`}
			mode={mode}
			initialValue={initialValue}
			promptsForCopy={promptsForCopy}
			onClose={onClose}
			onSave={onSave}
		/>,
		document.body
	);
}

function SystemPromptAddEditModalInner({
	mode,
	initialValue = '',
	promptsForCopy = [],
	onClose,
	onSave,
}: SystemPromptAddEditModalInnerProps) {
	const [value, setValue] = useState<string>(() => initialValue);
	const [copyFromId, setCopyFromId] = useState<string>('');

	const dialogRef = useRef<HTMLDialogElement | null>(null);
	const isUnmountingRef = useRef(false);

	useEffect(() => {
		isUnmountingRef.current = false;

		const dialog = dialogRef.current;
		if (!dialog) return;

		try {
			if (!dialog.open) {
				dialog.showModal();
			}
		} catch {
			// Ignore showModal errors if the dialog is already open or not ready.
		}

		return () => {
			isUnmountingRef.current = true;
			closeDialogSafely(dialog);
		};
	}, []);

	const handleDialogClose = useCallback(() => {
		if (!isUnmountingRef.current) {
			onClose();
		}
	}, [onClose]);

	const requestClose = useCallback(() => {
		if (!closeDialogSafely(dialogRef.current)) {
			onClose();
		}
	}, [onClose]);

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = e => {
		e.preventDefault();

		const v = value.trim();
		if (!v) return;

		onSave(v);
		requestClose();
	};

	const handleCopyFrom = useCallback(
		(id: string) => {
			setCopyFromId(id);

			const found = promptsForCopy.find(p => p.id === id);
			if (found) {
				setValue(found.prompt);
			}
		},
		[promptsForCopy]
	);

	const copyDropdownItems = useMemo(() => {
		const map: Record<string, { isEnabled: boolean }> = {};
		for (const p of promptsForCopy) {
			map[p.id] = { isEnabled: true };
		}
		return map;
	}, [promptsForCopy]);

	const getCopyDisplayName = useCallback(
		(id: string) => {
			const found = promptsForCopy.find(p => p.id === id);
			return found?.title ?? id;
		},
		[promptsForCopy]
	);

	return (
		<dialog ref={dialogRef} className="modal" onClose={handleDialogClose}>
			<div className="modal-box bg-base-200 max-h-[80vh] max-w-3xl overflow-auto rounded-2xl">
				<div className="mb-4 flex items-center justify-between">
					<h3 className="text-lg font-bold">{mode === 'add' ? 'Add System Prompt' : 'Edit System Prompt'}</h3>
					<button type="button" className="btn btn-sm btn-circle bg-base-300" onClick={requestClose} aria-label="Close">
						<FiX size={12} />
					</button>
				</div>

				<form onSubmit={handleSubmit} className="space-y-4">
					{mode === 'add' && (
						<div className="grid grid-cols-12 items-center gap-1">
							<label className="col-span-2 text-sm opacity-70">Copy Existing:</label>
							<div className="col-span-9">
								<Dropdown<string>
									dropdownItems={copyDropdownItems}
									selectedKey={copyFromId}
									onChange={handleCopyFrom}
									filterDisabled={false}
									title="Select a saved prompt to copy"
									getDisplayName={getCopyDisplayName}
									maxMenuHeight={260}
								/>
							</div>
							<button
								type="button"
								className="btn btn-ghost btn-xs col-span-1 p-4"
								title="Copy again"
								onClick={() => {
									if (copyFromId) {
										handleCopyFrom(copyFromId);
									}
								}}
								disabled={!copyFromId}
							>
								<FiCopy size={14} />
							</button>
						</div>
					)}

					<div>
						<textarea
							className="textarea textarea-bordered h-40 w-full rounded-xl"
							value={value}
							onChange={e => {
								setValue(e.target.value);
							}}
							placeholder="Enter system prompt instructions here..."
							spellCheck="false"
						/>
					</div>

					<div className="modal-action">
						<button type="button" className="btn bg-base-300 rounded-xl" onClick={requestClose}>
							Cancel
						</button>
						<button type="submit" className="btn btn-primary rounded-xl" disabled={!value.trim()}>
							Save
						</button>
					</div>
				</form>
			</div>
		</dialog>
	);
}
