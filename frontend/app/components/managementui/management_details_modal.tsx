import type { ReactNode } from 'react';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalBackdrop } from '@/components/modal/modal_backdrop';
import { ModalDialog } from '@/components/modal/modal_dialog';
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
	allowBackdropClose = true,
}: Omit<ManagementDetailsModalProps, 'isOpen' | 'modalKey' | 'onClose'>) {
	const { requestClose } = useModalDialogController();

	const widthClassName = width === 'wide' ? 'max-w-6xl' : 'max-w-3xl';
	const heightClassName = height === 'tall' ? 'max-h-[85vh]' : 'max-h-[80vh]';

	return (
		<>
			<div
				className={`modal-box bg-base-200 flex ${heightClassName} w-[calc(100%-1rem)] ${widthClassName} flex-col overflow-hidden rounded-2xl p-0`}
			>
				<ModalHeader
					title={title}
					description={description}
					onClose={() => {
						requestClose();
					}}
				/>

				<div className="app-scrollbar-thin min-h-0 flex-1 space-y-4 overflow-y-auto p-4 sm:p-6">{children}</div>

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
			</div>

			<ModalBackdrop enabled={allowBackdropClose} />
		</>
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
		<ModalDialog isOpen={isOpen} onClose={onClose}>
			<ManagementDetailsModalContent
				key={modalKey}
				title={title}
				description={description}
				width={width}
				height={height}
				closeLabel={closeLabel}
				allowBackdropClose={allowBackdropClose}
			>
				{children}
			</ManagementDetailsModalContent>
		</ModalDialog>
	);
}
