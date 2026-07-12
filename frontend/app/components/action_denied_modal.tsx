import { useEffect, useRef } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertTriangle } from 'react-icons/fi';

import { ModalBackdrop } from '@/components/modal/modal_backdrop';

interface ActionDeniedAlertModalProps {
	isOpen: boolean;
	onClose: () => void;
	message: string;
	title?: string;
}

export function ActionDeniedAlertModal({
	isOpen,
	onClose,
	message,
	title = 'Action Not Allowed',
}: ActionDeniedAlertModalProps) {
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

	// Keep parent isOpen in sync with native dialog closing
	const handleDialogClose = () => {
		onClose();
	};

	const handleOkClick = () => {
		// Close via native dialog API; this will trigger handleDialogClose -> parent onClose()
		dialogRef.current?.close();
	};

	if (!isOpen) {
		return null;
	}

	return createPortal(
		<dialog ref={dialogRef} className="modal" onClose={handleDialogClose}>
			<div className="modal-box max-h-[80vh] max-w-md overflow-auto rounded-2xl">
				<div className="mb-4 flex items-center">
					<FiAlertTriangle size={24} className="text-warning mr-3" />
					<h3 className="text-lg font-bold">{title}</h3>
				</div>
				<p className="py-2">{message}</p>
				<div className="modal-action">
					<button type="button" className="btn btn-primary rounded-xl" onClick={handleOkClick}>
						OK
					</button>
				</div>
			</div>

			<ModalBackdrop enabled={true} />
		</dialog>,
		document.body
	);
}
