import type { ReactNode } from 'react';

import type { ModalDialogProps } from '@/components/modal/modal_dialog';
import { ModalBackdrop } from '@/components/modal/modal_backdrop';
import { ModalDialog } from '@/components/modal/modal_dialog';

type ManagementModalWidth = 'confirmation' | 'compact' | 'standard' | 'form' | 'wide';

const WIDTH_CLASSES: Record<ManagementModalWidth, string> = {
	confirmation: 'max-w-md',
	compact: 'max-w-xl',
	standard: 'max-w-3xl',
	form: 'max-w-4xl',
	wide: 'max-w-6xl',
};

interface ManagementModalPanelProps {
	children: ReactNode;
	width?: ManagementModalWidth;
	className?: string;
}

/**
 * One height, width scale, and scroll contract for management dialogs.
 * Header and footer are direct children; ManagementModalBody scrolls.
 */
export function ManagementModalPanel({ children, width = 'standard', className = '' }: ManagementModalPanelProps) {
	return (
		<div
			className={`modal-box bg-base-200 flex max-h-[calc(100dvh-1rem)] w-[calc(100%-1rem)] ${WIDTH_CLASSES[width]} min-h-0 flex-col overflow-hidden rounded-2xl p-0 ${className}`}
		>
			{children}
		</div>
	);
}

export function ManagementModalBody({ children, className = '' }: { children: ReactNode; className?: string }) {
	return (
		<div
			className={`app-scrollbar-thin min-h-0 flex-1 space-y-4 overflow-x-hidden overflow-y-auto overscroll-contain p-4 sm:p-6 ${className}`}
		>
			{children}
		</div>
	);
}

interface ManagementModalDialogProps extends Omit<ModalDialogProps, 'children'> {
	children: ModalDialogProps['children'];
	allowBackdropClose?: boolean;
}

/**
 * Read-only detail dialogs omit `blockCancel`, so Escape and the backdrop
 * close them. Editors use `blockCancel`, keep a visible header close control
 * and explicit footer actions, and never dismiss from a backdrop click.
 * Busy dialogs also disable backdrop dismissal.
 */
export function ManagementModalDialog({
	children,
	allowBackdropClose = true,
	blockCancel = false,
	isBusy = false,
	...dialogProps
}: ManagementModalDialogProps) {
	const backdropEnabled = allowBackdropClose && !blockCancel && !isBusy;

	if (typeof children === 'function') {
		return (
			<ModalDialog {...dialogProps} blockCancel={blockCancel} isBusy={isBusy}>
				{controller => (
					<>
						{children(controller)}
						<ModalBackdrop enabled={backdropEnabled} />
					</>
				)}
			</ModalDialog>
		);
	}
	return (
		<ModalDialog {...dialogProps} blockCancel={blockCancel} isBusy={isBusy}>
			{children}
			<ModalBackdrop enabled={backdropEnabled} />
		</ModalDialog>
	);
}
