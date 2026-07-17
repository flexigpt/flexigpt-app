import { createPortal } from 'react-dom';

import { FiAlertTriangle } from 'react-icons/fi';

import { useDialogController } from '@/hooks/use_dialog_controller';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalBackdrop } from '@/components/modal/modal_backdrop';
import { ModalHeader } from '@/components/modal/modal_header';

interface DeleteConfirmationModalProps {
	isOpen: boolean;
	onClose: () => void;
	onConfirm: () => void;
	title: string;
	message: string;
	confirmButtonText: string;
}

function DeleteConfirmationModalContent({
	onClose,
	onConfirm,
	title,
	message,
	confirmButtonText,
}: Omit<DeleteConfirmationModalProps, 'isOpen'>) {
	const { dialogRef, requestClose, handleClose, handleCancel } = useDialogController({ onClose });

	return (
		<dialog ref={dialogRef} className="modal" onClose={handleClose} onCancel={handleCancel}>
			<div className="modal-box bg-base-200 flex max-h-[80vh] w-[calc(100%-1rem)] max-w-md flex-col overflow-hidden rounded-2xl p-0">
				<ModalHeader
					title={
						<span className="flex items-center gap-2">
							<FiAlertTriangle size={16} className="text-warning" />
							<span>{title}</span>
						</span>
					}
					onClose={() => {
						requestClose();
					}}
				/>

				<div className="min-h-0 flex-1 overflow-y-auto p-4 sm:px-6">
					<p className="py-2">{message}</p>
				</div>

				<ModalActions>
					<button
						type="button"
						className="btn bg-base-300 rounded-xl"
						onClick={() => {
							requestClose();
						}}
					>
						Cancel
					</button>
					<button
						type="button"
						className="btn btn-error rounded-xl"
						onClick={() => {
							onConfirm();
							requestClose();
						}}
					>
						{confirmButtonText}
					</button>
				</ModalActions>
			</div>

			<ModalBackdrop enabled={true} />
		</dialog>
	);
}

export function DeleteConfirmationModal(props: DeleteConfirmationModalProps) {
	if (!props.isOpen || typeof document === 'undefined' || !document.body) {
		return null;
	}

	return createPortal(
		<DeleteConfirmationModalContent
			onClose={props.onClose}
			onConfirm={props.onConfirm}
			title={props.title}
			message={props.message}
			confirmButtonText={props.confirmButtonText}
		/>,
		document.body
	);
}
