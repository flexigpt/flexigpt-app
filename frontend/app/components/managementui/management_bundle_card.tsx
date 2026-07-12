import type { ReactNode } from 'react';

import { ActionRow } from '@/components/managementui/action_row';

interface ManagementBundleCardProps {
	title: ReactNode;
	identity?: ReactNode;
	subtitle?: ReactNode;
	status?: ReactNode;
	disclosure?: ReactNode;
	description?: ReactNode;
	metadata?: ReactNode;
	actionLeading?: ReactNode;
	actions?: ReactNode;
	headerActions?: ReactNode;
	children?: ReactNode;
	className?: string;
}

/**
 * Header actions are retained for compatibility. New bundle cards should use
 * `status` and `disclosure` in the header, then `actionLeading` and `actions`
 * for mutations in the bottom row.
 *
 */
export function ManagementBundleCard({
	title,
	identity,
	subtitle,
	status,
	disclosure,
	description,
	metadata,
	actionLeading,
	actions,
	headerActions,
	children,
	className = '',
}: ManagementBundleCardProps) {
	const effectiveIdentity = identity ?? subtitle;

	return (
		<section className={`bg-base-100 border-base-content/10 mb-6 rounded-2xl border p-4 shadow-sm ${className}`}>
			<div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
				<div className="min-w-0">
					<div className="truncate text-sm font-semibold">{title}</div>

					{effectiveIdentity ? (
						<div className="text-base-content/60 mt-1 text-xs wrap-break-word">{effectiveIdentity}</div>
					) : null}
				</div>

				{status || disclosure || headerActions ? (
					<div className="flex max-w-full flex-wrap items-center justify-end gap-2">
						{status}
						{disclosure}
						{headerActions}
					</div>
				) : null}
			</div>

			{description ? <div className="text-base-content/70 mt-3 text-sm">{description}</div> : null}
			{metadata ? <div className="mt-3 flex flex-wrap gap-2">{metadata}</div> : null}
			{children}
			{actionLeading || actions ? <ActionRow leading={actionLeading}>{actions}</ActionRow> : null}
		</section>
	);
}
