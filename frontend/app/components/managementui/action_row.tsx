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
			{leading ? <div className="flex w-full min-w-0 flex-wrap items-center gap-2 sm:w-auto">{leading}</div> : null}

			{children ? (
				<div
					className={`flex w-full flex-wrap items-center justify-start gap-2 sm:w-auto sm:justify-end ${
						leading ? '' : 'sm:ml-auto'
					}`}
				>
					{children}
				</div>
			) : null}
		</div>
	);
}
