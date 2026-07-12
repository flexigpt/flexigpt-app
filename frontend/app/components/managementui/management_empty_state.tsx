import type { ReactNode } from 'react';

interface ManagementEmptyStateProps {
	children: ReactNode;
	className?: string;
}

export function ManagementEmptyState({ children, className = '' }: ManagementEmptyStateProps) {
	return (
		<div
			className={`border-base-content/10 text-base-content/70 rounded-2xl border px-4 py-8 text-center text-sm ${className}`}
		>
			{children}
		</div>
	);
}
