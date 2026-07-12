import type { ReactNode } from 'react';

interface ManagementInfoRowProps {
	label: ReactNode;
	children: ReactNode;
	mono?: boolean;
	className?: string;
}

export function ManagementInfoRow({ label, children, mono = false, className = '' }: ManagementInfoRowProps) {
	return (
		<div
			className={`border-base-content/10 grid grid-cols-1 gap-1 border-b pb-3 text-sm last:border-b-0 last:pb-0 sm:grid-cols-12 sm:gap-3 ${className}`}
		>
			<dt className="text-base-content/60 font-medium sm:col-span-4">{label}</dt>
			<dd className={`min-w-0 wrap-break-word sm:col-span-8 ${mono ? 'font-mono text-xs' : ''}`}>{children}</dd>
		</div>
	);
}
