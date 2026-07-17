import { FiAlertTriangle } from 'react-icons/fi';

import { ModalConfirmDialog } from '@/components/modal/modal_confirm_dialog';

interface DeleteConfirmationModalProps {
	isOpen: boolean;
	onClose: () => void;
	onConfirm: () => void;
	title: string;
	message: string;
	confirmButtonText: string;
}

export function DeleteConfirmationModal(props: DeleteConfirmationModalProps) {
	return (
		<ModalConfirmDialog
			isOpen={props.isOpen}
			onClose={props.onClose}
			title={props.title}
			icon={<FiAlertTriangle size={16} className="text-warning" />}
			message={<p className="py-2">{props.message}</p>}
			onConfirm={props.onConfirm}
			confirmLabel={props.confirmButtonText}
			confirmTone="error"
		/>
	);
}
