import type { ReactNode } from 'react';

interface ModalActionsProps {
	children: ReactNode;
	leading?: ReactNode;
	className?: string;
}

export function ModalActions({ children, leading, className = '' }: ModalActionsProps) {
	return (
		<footer
			className={`bg-base-200/95 border-base-content/10 sticky bottom-0 z-20 flex shrink-0 flex-col gap-3 border-t p-4 backdrop-blur-sm sm:flex-row sm:items-center sm:justify-between sm:px-6 ${className}`}
		>
			{leading ? (
				<div className="flex flex-wrap items-center gap-2">{leading}</div>
			) : (
				<span className="hidden sm:block" />
			)}
			<div className="flex w-full flex-col-reverse gap-2 sm:w-auto sm:flex-row sm:flex-wrap sm:items-center sm:justify-end [&>button]:w-full sm:[&>button]:w-auto">
				{children}
			</div>
		</footer>
	);
}
