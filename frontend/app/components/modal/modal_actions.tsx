import type { ReactNode } from 'react';

interface ModalActionsProps {
	children: ReactNode;
	leading?: ReactNode;
	className?: string;
}

export function ModalActions({ children, leading, className = '' }: ModalActionsProps) {
	return (
		<footer
			className={`border-base-content/10 flex shrink-0 flex-col gap-3 border-t p-4 sm:flex-row sm:items-center sm:justify-between sm:px-6 ${className}`}
		>
			{leading ? <div className="flex flex-wrap items-center gap-2">{leading}</div> : <span />}
			<div className="flex flex-wrap items-center justify-end gap-2">{children}</div>
		</footer>
	);
}
