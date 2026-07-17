import type { SyntheticEvent } from 'react';
import { useCallback, useLayoutEffect, useRef } from 'react';

interface UseDialogControllerOptions {
	onClose: () => void;
	blockCancel?: boolean;
	isBusy?: boolean;
	isOpen?: boolean;
}

/**
 * Centralizes native dialog lifecycle behavior.
 *
 * It prevents close callbacks caused by unmount cleanup, blocks accidental
 * cancellation while saving, and keeps programmatic and native close paths
 * consistent. Callers may conditionally mount dialog content or provide
 * `isOpen` when a component remains mounted while the dialog is hidden.
 */
export function useDialogController({
	onClose,
	blockCancel = false,
	isBusy = false,
	isOpen = true,
}: UseDialogControllerOptions) {
	const dialogRef = useRef<HTMLDialogElement | null>(null);
	const unmountingRef = useRef(false);

	/*
	 * React Strict Mode can replay an effect without removing the dialog
	 * element. Calling close() in the replay cleanup emits `close`, which can
	 * invoke the parent's onClose handler and unmount a just-opened modal.
	 * Let conditional rendering remove the dialog on a real unmount instead.
	 */
	useLayoutEffect(() => {
		if (!isOpen) {
			unmountingRef.current = true;
			return;
		}

		unmountingRef.current = false;

		const dialog = dialogRef.current;
		if (!dialog) {
			return;
		}

		if (!dialog.open) {
			try {
				dialog.showModal();
			} catch (modalError) {
				console.warn('Failed to open native modal dialog. Falling back to a non-modal dialog.', modalError);

				try {
					dialog.show();
				} catch (showError) {
					console.error('Failed to open native dialog:', showError);
					dialog.setAttribute('open', '');
				}
			}
		}

		return () => {
			unmountingRef.current = true;
		};
	}, [isOpen]);

	const requestClose = useCallback(
		(force = false) => {
			if (isBusy && !force) {
				return;
			}

			const dialog = dialogRef.current;
			if (dialog?.open) {
				dialog.close();
				return;
			}

			onClose();
		},
		[isBusy, onClose]
	);

	const handleClose = useCallback(() => {
		if (!unmountingRef.current) {
			onClose();
		}
	}, [onClose]);

	const handleCancel = useCallback(
		(event: SyntheticEvent<HTMLDialogElement>) => {
			if (blockCancel || isBusy) {
				event.preventDefault();
			}
		},
		[blockCancel, isBusy]
	);

	return {
		dialogRef,
		requestClose,
		handleClose,
		handleCancel,
		unmountingRef,
	};
}
