import { FiAlertTriangle } from 'react-icons/fi';

import { ModalConfirmDialog } from '@/components/modal/modal_confirm_dialog';

interface ActionDeniedAlertModalProps {
	isOpen: boolean;
	onClose: () => void;
	message: string;
	title?: string;
}

export function ActionDeniedAlertModal(props: ActionDeniedAlertModalProps) {
	return (
		<ModalConfirmDialog
			isOpen={props.isOpen}
			onClose={props.onClose}
			title={props.title ?? 'Action Not Allowed'}
			icon={<FiAlertTriangle size={16} className="text-warning" />}
			message={<p className="py-2">{props.message}</p>}
			confirmLabel="OK"
			showCancel={false}
		/>
	);
}
