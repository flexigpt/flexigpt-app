import type { ReactNode } from 'react';

interface ActionRowProps {
	children?: ReactNode;
	leading?: ReactNode;
	className?: string;
}

export function ActionRow({ children, leading, className = '' }: ActionRowProps) {
	return (
		<div
			className={`border-base-content/10 mt-4 flex flex-col gap-3 border-t pt-3 sm:flex-row sm:items-center sm:justify-between ${className}`}
		>
			{leading ? <div className="flex min-w-0 flex-wrap items-center gap-2">{leading}</div> : null}

			{children ? (
				<div className={`flex flex-wrap items-center justify-end gap-2 ${leading ? '' : 'sm:ml-auto'}`}>{children}</div>
			) : null}
		</div>
	);
}
