import type { ReactNode } from 'react';

import { ActionRow } from '@/components/managementui/action_row';

interface ManagementItemCardProps {
	title: ReactNode;
	subtitle?: ReactNode;
	status?: ReactNode;
	description?: ReactNode;
	metadata?: ReactNode;
	actions?: ReactNode;
	children?: ReactNode;
	className?: string;
}

export function ManagementItemCard({
	title,
	subtitle,
	status,
	description,
	metadata,
	actions,
	children,
	className = '',
}: ManagementItemCardProps) {
	return (
		<article
			className={`border-base-content/10 hover:border-base-content/20 min-w-0 rounded-2xl border p-4 transition-colors ${className}`}
		>
			<div className="flex min-w-0 flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
				<div className="min-w-0">
					<div className="truncate font-medium">{title}</div>

					{subtitle ? <div className="text-base-content/60 mt-1 text-xs wrap-break-word">{subtitle}</div> : null}

					{description ? (
						<div className="text-base-content/70 mt-2 max-h-16 overflow-hidden text-sm">{description}</div>
					) : null}
				</div>

				{status ? <div className="flex max-w-full shrink-0 flex-wrap items-center gap-2">{status}</div> : null}
			</div>

			{metadata ? <div className="mt-3 flex flex-wrap gap-2">{metadata}</div> : null}
			{children}
			{actions ? <ActionRow>{actions}</ActionRow> : null}
		</article>
	);
}
