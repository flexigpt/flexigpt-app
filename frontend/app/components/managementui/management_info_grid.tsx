import type { ReactNode } from 'react';

interface ManagementInfoGridProps {
	children: ReactNode;
	className?: string;
}

export function ManagementInfoGrid({ children, className = '' }: ManagementInfoGridProps) {
	return <dl className={`space-y-3 ${className}`}>{children}</dl>;
}
