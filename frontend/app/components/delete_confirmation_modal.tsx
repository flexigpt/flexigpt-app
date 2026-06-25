import { useEffect, useRef } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertTriangle, FiX } from 'react-icons/fi';

import { ModalBackdrop } from '@/components/modal_backdrop';

interface DeleteConfirmationModalProps {
	isOpen: boolean;
	onClose: () => void;
	onConfirm: () => void;
	title: string;
	message: string;
	confirmButtonText: string;
}

export function DeleteConfirmationModal({
	isOpen,
	onClose,
	onConfirm,
	title,
	message,
	confirmButtonText,
}: DeleteConfirmationModalProps) {
	const dialogRef = useRef<HTMLDialogElement | null>(null);

	useEffect(() => {
		if (!isOpen) {
			return;
		}

		const dialog = dialogRef.current;
		if (!dialog) {
			return;
		}

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

	// Sync parent state whenever the dialog is closed (Esc, backdrop, or dialog.close()).
	const handleDialogClose = () => {
		onClose();
	};

	const handleCancelClick = () => {
		// Close via native dialog API; this will trigger handleDialogClose -> parent onClose()
		dialogRef.current?.close();
	};

	const handleConfirmClick = () => {
		onConfirm();
		// Close via native dialog API; this will trigger handleDialogClose -> parent onClose()
		dialogRef.current?.close();
	};

	if (!isOpen) {
		return null;
	}

	return createPortal(
		<dialog ref={dialogRef} className="modal" onClose={handleDialogClose}>
			<div className="modal-box bg-base-200 max-h-[80vh] max-w-md overflow-auto rounded-2xl">
				{/* header (matches UrlAttachment / MessageDetails pattern) */}
				<div className="mb-4 flex items-center justify-between">
					<h3 className="flex items-center gap-2 text-lg font-bold">
						<FiAlertTriangle size={16} className="text-warning" />
						<span>{title}</span>
					</h3>
					<button
						type="button"
						className="btn btn-sm btn-circle bg-base-300"
						onClick={handleCancelClick}
						aria-label="Close"
					>
						<FiX size={12} />
					</button>
				</div>

				<p className="py-2">{message}</p>

				<div className="modal-action">
					<button type="button" className="btn bg-base-300 rounded-xl" onClick={handleCancelClick}>
						Cancel
					</button>
					<button type="button" className="btn btn-error rounded-xl" onClick={handleConfirmClick}>
						{confirmButtonText}
					</button>
				</div>
			</div>

			<ModalBackdrop enabled={true} />
		</dialog>,
		document.body
	);
}
