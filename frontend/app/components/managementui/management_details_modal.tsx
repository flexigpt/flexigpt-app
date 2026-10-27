import type { ReactNode } from 'react';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import {
	ManagementModalBody,
	ManagementModalDialog,
	ManagementModalPanel,
} from '@/components/managementui/management_modal_layout';
import { ModalActions } from '@/components/modal/modal_actions';
import { ModalHeader } from '@/components/modal/modal_header';

interface ManagementDetailsModalProps {
	isOpen: boolean;
	onClose: () => void;
	title: ReactNode;
	description?: ReactNode;
	children: ReactNode;
	modalKey: string;
	width?: 'standard' | 'wide';
	height?: 'standard' | 'tall';
	closeLabel?: string;
	allowBackdropClose?: boolean;
}

function ManagementDetailsModalContent({
	title,
	description,
	children,
	width = 'standard',
	height = 'standard',
	closeLabel = 'Close',
}: Omit<ManagementDetailsModalProps, 'isOpen' | 'modalKey' | 'onClose' | 'allowBackdropClose'>) {
	const { requestClose } = useModalDialogController();

	return (
		<ManagementModalPanel width={width} className={height === 'tall' ? 'max-h-[calc(100dvh-1rem)]' : ''}>
			<ModalHeader
				title={title}
				description={description}
				onClose={() => {
					requestClose();
				}}
			/>

			<ManagementModalBody>{children}</ManagementModalBody>

			<ModalActions>
				<button
					type="button"
					className="btn bg-base-300 rounded-xl"
					onClick={() => {
						requestClose();
					}}
				>
					{closeLabel}
				</button>
			</ModalActions>
		</ManagementModalPanel>
	);
}

export function ManagementDetailsModal({
	isOpen,
	onClose,
	title,
	description,
	children,
	modalKey,
	width,
	height,
	closeLabel,
	allowBackdropClose,
}: ManagementDetailsModalProps) {
	if (!isOpen) {
		return null;
	}

	return (
		<ManagementModalDialog isOpen={isOpen} onClose={onClose} allowBackdropClose={allowBackdropClose}>
			<ManagementDetailsModalContent
				key={modalKey}
				title={title}
				description={description}
				width={width}
				height={height}
				closeLabel={closeLabel}
			>
				{children}
			</ManagementDetailsModalContent>
		</ManagementModalDialog>
	);
}
