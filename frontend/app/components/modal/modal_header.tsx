import type { ReactNode } from 'react';

import { FiX } from 'react-icons/fi';

interface ModalHeaderProps {
	title: ReactNode;
	description?: ReactNode;
	onClose: () => void;
	closeDisabled?: boolean;
}

export function ModalHeader({ title, description, onClose, closeDisabled = false }: ModalHeaderProps) {
	return (
		<header className="bg-base-200/95 border-base-content/10 sticky top-0 z-20 flex shrink-0 items-start justify-between gap-4 border-b p-4 backdrop-blur-sm sm:px-6">
			<div className="min-w-0">
				<h3 className="text-lg font-semibold tracking-tight">{title}</h3>
				{description ? <div className="text-base-content/70 mt-1 text-sm">{description}</div> : null}
			</div>

			<button
				type="button"
				className="btn btn-sm btn-circle bg-base-300 shrink-0"
				onClick={onClose}
				disabled={closeDisabled}
				aria-label="Close"
			>
				<FiX size={14} />
			</button>
		</header>
	);
}
