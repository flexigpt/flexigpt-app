import type { SyntheticEvent } from 'react';
import { useCallback, useEffect, useRef } from 'react';

interface UseDialogControllerOptions {
	onClose: () => void;
	blockCancel?: boolean;
	isBusy?: boolean;
}

/**
 * Centralizes native dialog lifecycle behavior.
 *
 * It prevents close callbacks caused by unmount cleanup, blocks accidental
 * cancellation while saving, and keeps programmatic and native close paths
 * consistent.
 */
export function useDialogController({ onClose, blockCancel = false, isBusy = false }: UseDialogControllerOptions) {
	const dialogRef = useRef<HTMLDialogElement | null>(null);
	const unmountingRef = useRef(false);

	useEffect(() => {
		unmountingRef.current = false;

		const dialog = dialogRef.current;
		if (!dialog) {
			return;
		}

		if (!dialog.open) {
			try {
				dialog.showModal();
			} catch (error) {
				console.error('Failed to open native dialog:', error);
			}
		}

		return () => {
			unmountingRef.current = true;

			if (dialog.open) {
				dialog.close();
			}
		};
	}, []);

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
