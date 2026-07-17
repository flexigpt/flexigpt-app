import { createPortal } from 'react-dom';

import { FiAlertTriangle } from 'react-icons/fi';

import { useDialogController } from '@/hooks/use_dialog_controller';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalBackdrop } from '@/components/modal/modal_backdrop';

interface ActionDeniedAlertModalProps {
	isOpen: boolean;
	onClose: () => void;
	message: string;
	title?: string;
}

function ActionDeniedAlertModalContent({
	onClose,
	message,
	title = 'Action Not Allowed',
}: Omit<ActionDeniedAlertModalProps, 'isOpen'>) {
	const { dialogRef, requestClose, handleClose, handleCancel } = useDialogController({ onClose });

	return (
		<dialog ref={dialogRef} className="modal" onClose={handleClose} onCancel={handleCancel}>
			<div className="modal-box bg-base-200 flex max-h-[80vh] w-[calc(100%-1rem)] max-w-md flex-col overflow-hidden rounded-2xl p-0">
				<div className="mb-4 flex items-center px-4 pt-4 sm:px-6 sm:pt-6">
					<FiAlertTriangle size={24} className="text-warning mr-3" />
					<h3 className="text-lg font-bold">{title}</h3>
				</div>
				<div className="min-h-0 flex-1 overflow-y-auto px-4 pb-4 sm:px-6 sm:pb-6">
					<p className="py-2">{message}</p>
				</div>
				<ModalActions>
					<button
						type="button"
						className="btn btn-primary rounded-xl"
						onClick={() => {
							requestClose();
						}}
					>
						OK
					</button>
				</ModalActions>
			</div>

			<ModalBackdrop enabled={true} />
		</dialog>
	);
}

export function ActionDeniedAlertModal(props: ActionDeniedAlertModalProps) {
	if (!props.isOpen || typeof document === 'undefined' || !document.body) {
		return null;
	}

	return createPortal(
		<ActionDeniedAlertModalContent onClose={props.onClose} message={props.message} title={props.title} />,
		document.body
	);
}
