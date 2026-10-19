import type { ReactNode } from 'react';
import { FiChevronDown, FiChevronUp } from 'react-icons/fi';

export const actionTriggerChipButtonClasses =
	'btn btn-xs app-text-neutral bg-base-200/70 hover:bg-base-300/80 h-7 min-h-0 items-center overflow-hidden rounded-full border-none px-2 text-left normal-case shadow-none';

export const actionTriggerChipSurfaceClasses =
	'app-text-neutral bg-base-200/70 hover:bg-base-300/80 flex h-7 min-h-0 items-center overflow-hidden rounded-full px-2 shadow-none';

export const actionTriggerMenuWideClasses =
	'rounded-box bg-base-100 text-base-content border-base-300 z-50 flex max-h-120 max-w-lg min-w-60 flex-col overflow-hidden border p-2 text-xs shadow-xl outline-none';

export const actionTriggerMenuCompactClasses =
	'rounded-box bg-base-100 text-base-content z-50 flex max-h-72 min-w-48 max-w-sm flex-col overflow-hidden border border-base-300 p-1 shadow-xl outline-none';

export const actionTriggerMenuItemClasses =
	'flex items-center gap-2 rounded-xl px-2 py-1 text-xs outline-none transition-colors hover:bg-base-200 data-[active-item]:bg-base-300';

/**
 * Place inside ActionTriggerMenuHeader, after SearchableMenuInput when the
 * menu has a search field. The header therefore always follows:
 *
 *   search -> title/actions/errors
 */
export const actionTriggerMenuHeaderRowClasses = 'flex min-w-0 items-center justify-between gap-2 px-1 py-0';

interface ActionTriggerMenuRegionProps {
	children: ReactNode;
	className?: string;
}

export function ActionTriggerMenuHeader({ children, className = '' }: ActionTriggerMenuRegionProps) {
	return <div className={`bg-base-100 border-base-300 z-10 mb-1 shrink-0 border-b ${className}`}>{children}</div>;
}

export function ActionTriggerMenuBody({ children, className = '' }: ActionTriggerMenuRegionProps) {
	return <div className={`min-h-0 flex-auto overflow-y-auto overscroll-contain ${className}`}>{children}</div>;
}

interface ActionTriggerChipContentProps {
	icon?: ReactNode;
	label: ReactNode;
	secondaryLabel?: ReactNode;
	count?: ReactNode;
	suffix?: ReactNode;
	open?: boolean;
	showChevron?: boolean;
	className?: string;
	labelClassName?: string;
	secondaryLabelClassName?: string;
}

export function ActionTriggerChipContent({
	icon,
	label,
	secondaryLabel,
	count,
	suffix,
	open = false,
	showChevron = true,
	className = '',
	labelClassName = 'min-w-0 max-w-28 truncate text-xs font-normal',
	secondaryLabelClassName = 'max-w-28 truncate text-xs opacity-70',
}: ActionTriggerChipContentProps) {
	return (
		<span className={`flex min-w-0 items-center gap-1 ${className}`}>
			{icon ? <span className="shrink-0">{icon}</span> : null}
			{label ? <span className={labelClassName}>{label}</span> : null}
			{secondaryLabel ? <span className={secondaryLabelClassName}>{secondaryLabel}</span> : null}
			{count ? <span className="shrink-0">{count}</span> : null}
			{suffix ? <span className="shrink-0">{suffix}</span> : null}
			{showChevron ? (
				open ? (
					<FiChevronDown size={14} className="shrink-0" />
				) : (
					<FiChevronUp size={14} className="shrink-0" />
				)
			) : null}
		</span>
	);
}
