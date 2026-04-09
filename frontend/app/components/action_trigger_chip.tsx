import type { ReactNode } from 'react';

import { FiChevronDown, FiChevronUp } from 'react-icons/fi';

export const actionTriggerChipButtonClasses =
	'btn btn-xs text-neutral-custom bg-base-200/70 hover:bg-base-300/80 h-7 min-h-0 items-center overflow-hidden rounded-full border-none px-2 text-left normal-case shadow-none';

export const actionTriggerChipSurfaceClasses =
	'text-neutral-custom bg-base-200/70 hover:bg-base-300/80 flex h-7 min-h-0 items-center overflow-hidden rounded-full px-2 shadow-none';

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
			<span className={labelClassName}>{label}</span>
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
