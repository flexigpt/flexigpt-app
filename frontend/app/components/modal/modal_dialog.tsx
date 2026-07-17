import type { DialogHTMLAttributes, ReactNode } from 'react';
import { useMemo } from 'react';

import { createPortal } from 'react-dom';

import type { ModalDialogController } from '@/hooks/use_dialog_controller';
import { ModalDialogControllerContext, useDialogController } from '@/hooks/use_dialog_controller';

type ModalDialogRender = (controller: ModalDialogController) => ReactNode;
type ModalDialogChildren = ReactNode | ModalDialogRender;
type NativeCancelHandler = NonNullable<DialogHTMLAttributes<HTMLDialogElement>['onCancel']>;

interface ModalDialogChildrenRendererProps {
	controller: ModalDialogController;
	render: ModalDialogRender;
}

/**
 * Runs render-function children behind a component boundary so refs are passed
 * as props rather than accessed through an arbitrary function call during
 * `ModalDialog`'s render.
 */
function ModalDialogChildrenRenderer({ controller, render }: ModalDialogChildrenRendererProps) {
	return <>{render(controller)}</>;
}

export interface ModalDialogProps extends Omit<
	DialogHTMLAttributes<HTMLDialogElement>,
	'children' | 'className' | 'onCancel' | 'onClose' | 'open'
> {
	/** Whether the native dialog is mounted and opened. */
	isOpen: boolean;
	/** Called after a successful native or programmatic close. */
	onClose: () => void;
	/** Prevent Escape dismissal, while preserving an explicitly supplied `onCancel` handler. */
	blockCancel?: boolean;
	/** Prevent normal close requests while an async action is running. */
	isBusy?: boolean;
	/** Additional classes for the native `<dialog>`. The DaisyUI `modal` class is always applied. */
	className?: string;
	/**
	 * Runs after the standard cancellation guard. Use this only for modal-specific
	 * Escape behavior, such as resolving an approval as denied.
	 */
	onCancel?: DialogHTMLAttributes<HTMLDialogElement>['onCancel'];
	/**
	 * Static content or a render function that receives lifecycle-safe close
	 * controls. `ModalBackdrop` remains a direct child when it is used here.
	 */
	children: ModalDialogChildren;
}

/**
 * Shared native-dialog root for application modals.
 *
 * The component is safe to import during server rendering: it only accesses
 * `document.body` after confirming that a browser DOM is available. In the
 * browser, it owns the body portal and native dialog lifecycle so individual
 * modals do not need repeated environment guards or `createPortal` calls.
 */
export function ModalDialog({
	isOpen,
	onClose,
	blockCancel = false,
	isBusy = false,
	className = '',
	onCancel,
	children,
	...dialogProps
}: ModalDialogProps) {
	const { dialogRef, requestClose, handleClose, handleCancel, unmountingRef } = useDialogController({
		onClose,
		blockCancel,
		isBusy,
		isOpen,
	});
	const controller = useMemo<ModalDialogController>(
		() => ({ dialogRef, requestClose, unmountingRef }),
		[dialogRef, requestClose, unmountingRef]
	);

	const portalTarget = typeof document === 'undefined' ? null : document.body;
	if (!isOpen || !portalTarget) {
		return null;
	}

	const handleNativeCancel: NativeCancelHandler = event => {
		handleCancel(event);
		onCancel?.(event);
	};
	const content =
		typeof children === 'function' ? (
			<ModalDialogChildrenRenderer controller={controller} render={children} />
		) : (
			children
		);
	const dialogClassName = className ? `modal ${className}` : 'modal';

	return createPortal(
		<ModalDialogControllerContext.Provider value={controller}>
			<dialog
				{...dialogProps}
				ref={dialogRef}
				className={dialogClassName}
				onClose={handleClose}
				onCancel={handleNativeCancel}
			>
				{content}
			</dialog>
		</ModalDialogControllerContext.Provider>,
		portalTarget
	);
}
