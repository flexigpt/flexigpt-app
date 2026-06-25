import type { ReactElement } from 'react';

interface ModalBackdropProps {
	enabled: boolean;
	label?: string;
}
/**
 * DaisyUI dialog backdrop closer.
 * Must be rendered as a direct child of <dialog className="modal">.
 */
export function ModalBackdrop({ enabled, label = 'Close modal' }: ModalBackdropProps): ReactElement | null {
	if (!enabled) {
		return null;
	}
	return (
		<form method="dialog" className="modal-backdrop">
			<button type="submit" aria-label={label} />
		</form>
	);
}
