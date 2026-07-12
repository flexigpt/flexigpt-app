import type { ReactNode } from 'react';

interface MetadataPillProps {
	label?: ReactNode;
	children: ReactNode;
	title?: string;
	className?: string;
}

export function MetadataPill({ label, children, title, className = '' }: MetadataPillProps) {
	return (
		<span
			className={`border-base-content/20 inline-flex min-w-0 items-center gap-1 rounded-xl border px-2 py-1 text-xs ${className}`}
			title={title}
		>
			{label ? <span className="text-base-content/60 shrink-0">{label}</span> : null}
			<span className="min-w-0 truncate">{children}</span>
		</span>
	);
}
