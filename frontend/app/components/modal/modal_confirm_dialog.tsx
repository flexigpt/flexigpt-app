import type { DialogHTMLAttributes, ReactNode } from 'react';
import { useRef, useState } from 'react';

import { FiAlertCircle } from 'react-icons/fi';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalBackdrop } from '@/components/modal/modal_backdrop';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalHeader } from '@/components/modal/modal_header';

type ConfirmButtonTone = 'primary' | 'error' | 'warning';
type NativeDialogProps = Omit<
	DialogHTMLAttributes<HTMLDialogElement>,
	'children' | 'className' | 'onCancel' | 'onClose' | 'open'
>;

interface ModalConfirmDialogProps {
	isOpen: boolean;
	onClose: () => void;
	title: ReactNode;
	description?: ReactNode;
	message?: ReactNode;
	children?: ReactNode;
	icon?: ReactNode;
	onConfirm?: () => void | Promise<void>;
	confirmLabel?: ReactNode;
	cancelLabel?: ReactNode;
	busyLabel?: ReactNode;
	showCancel?: boolean;
	closeOnConfirm?: boolean;
	confirmTone?: ConfirmButtonTone;
	confirmDisabled?: boolean;
	cancelDisabled?: boolean;
	isBusy?: boolean;
	blockCancel?: boolean;
	allowBackdropClose?: boolean;
	error?: ReactNode;
	modalBoxClassName?: string;
	bodyClassName?: string;
	actionsClassName?: string;
	confirmButtonClassName?: string;
	cancelButtonClassName?: string;
	onCancel?: DialogHTMLAttributes<HTMLDialogElement>['onCancel'];
	dialogProps?: NativeDialogProps;
}

function getErrorMessage(error: unknown): string {
	return error instanceof Error && error.message.trim()
		? error.message
		: 'The requested action could not be completed.';
}

function isPromiseLike(value: unknown): value is PromiseLike<void> {
	return (
		typeof value === 'object' &&
		value !== null &&
		'then' in value &&
		typeof (value as { then?: unknown }).then === 'function'
	);
}

function getConfirmButtonClassName(tone: ConfirmButtonTone): string {
	switch (tone) {
		case 'error':
			return 'btn btn-error rounded-xl';
		case 'warning':
			return 'btn btn-warning rounded-xl';
		default:
			return 'btn btn-primary rounded-xl';
	}
}

function ModalConfirmDialogContent({
	isOpen,
	onClose,
	title,
	description,
	message,
	children,
	icon,
	onConfirm,
	confirmLabel,
	cancelLabel = 'Cancel',
	busyLabel = 'Saving...',
	showCancel = true,
	closeOnConfirm = true,
	confirmTone = 'primary',
	confirmDisabled = false,
	cancelDisabled = false,
	isBusy = false,
	blockCancel = false,
	allowBackdropClose = true,
	error,
	modalBoxClassName = '',
	bodyClassName = '',
	actionsClassName = '',
	confirmButtonClassName = '',
	cancelButtonClassName = '',
	onCancel,
	dialogProps,
}: ModalConfirmDialogProps) {
	const [isConfirming, setIsConfirming] = useState(false);
	const [confirmError, setConfirmError] = useState<string | null>(null);
	const confirmingRef = useRef(false);
	const effectiveBusy = isBusy || isConfirming;

	const resolvedConfirmLabel = confirmLabel ?? (onConfirm ? 'Confirm' : 'OK');
	const headerTitle = icon ? (
		<span className="flex items-center gap-2">
			{icon}
			<span>{title}</span>
		</span>
	) : (
		title
	);
	const displayError = error ?? confirmError;

	return (
		<ModalDialog
			{...dialogProps}
			isOpen={isOpen}
			onClose={onClose}
			blockCancel={blockCancel || effectiveBusy}
			isBusy={effectiveBusy}
			onCancel={onCancel}
		>
			{({ requestClose, unmountingRef }) => {
				const handleConfirm = () => {
					if (effectiveBusy || confirmingRef.current || confirmDisabled) {
						return;
					}

					if (!onConfirm) {
						requestClose();
						return;
					}

					confirmingRef.current = true;
					setConfirmError(null);
					setIsConfirming(true);

					const complete = () => {
						confirmingRef.current = false;
						if (unmountingRef.current) {
							return;
						}

						setIsConfirming(false);
						if (closeOnConfirm) {
							requestClose(true);
						}
					};

					const fail = (failure: unknown) => {
						confirmingRef.current = false;
						if (!unmountingRef.current) {
							setIsConfirming(false);
							setConfirmError(getErrorMessage(failure));
						}
					};

					try {
						const result = onConfirm();
						if (!isPromiseLike(result)) {
							complete();
							return;
						}

						void Promise.resolve(result).then(complete, fail);
					} catch (failure) {
						fail(failure);
					}
				};

				return (
					<>
						<div
							className={`modal-box bg-base-200 flex max-h-[80vh] w-[calc(100%-1rem)] max-w-md flex-col overflow-hidden rounded-2xl p-0 ${modalBoxClassName}`}
						>
							<ModalHeader
								title={headerTitle}
								description={description}
								onClose={requestClose}
								closeDisabled={effectiveBusy}
							/>

							<div className={`min-h-0 flex-1 overflow-y-auto p-4 sm:px-6 ${bodyClassName}`}>
								{displayError ? (
									<div className="alert alert-error rounded-2xl text-sm" role="alert">
										<FiAlertCircle className="shrink-0" size={14} />
										<span className="wrap-break-word">{displayError}</span>
									</div>
								) : null}
								{message ? <div className={displayError ? 'mt-4' : ''}>{message}</div> : null}
								{children ? <div className={displayError || message ? 'mt-4' : ''}>{children}</div> : null}
							</div>

							<ModalActions className={actionsClassName}>
								{showCancel ? (
									<button
										type="button"
										className={`btn bg-base-300 rounded-xl ${cancelButtonClassName}`}
										onClick={() => {
											requestClose();
										}}
										disabled={effectiveBusy || cancelDisabled}
									>
										{cancelLabel}
									</button>
								) : null}
								<button
									type="button"
									className={`${getConfirmButtonClassName(confirmTone)} ${confirmButtonClassName}`}
									onClick={handleConfirm}
									disabled={effectiveBusy || confirmDisabled}
								>
									{effectiveBusy ? <span className="loading loading-spinner loading-xs" /> : null}
									{effectiveBusy ? busyLabel : resolvedConfirmLabel}
								</button>
							</ModalActions>
						</div>

						<ModalBackdrop enabled={allowBackdropClose && !blockCancel && !effectiveBusy} />
					</>
				);
			}}
		</ModalDialog>
	);
}

/**
 * Reusable confirmation and acknowledgement dialog with synchronous and async
 * confirm support. It remounts for each open cycle so transient busy and error
 * state never leaks into a later confirmation.
 */
export function ModalConfirmDialog(props: ModalConfirmDialogProps) {
	if (!props.isOpen) {
		return null;
	}

	return <ModalConfirmDialogContent {...props} />;
}
