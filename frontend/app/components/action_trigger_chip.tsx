import type { ComponentProps, ReactNode } from 'react';
import { FiChevronDown, FiChevronUp } from 'react-icons/fi';

import { Menu } from '@ariakit/react';

const actionTriggerMenuClasses =
	'rounded-box bg-base-100 text-base-content border-base-300 z-50 flex min-h-0 min-w-60 max-w-xl flex-col overflow-hidden border p-2 text-xs shadow-xl outline-none';

export const actionTriggerMenuItemClasses =
	'flex min-h-7 items-center gap-2 rounded-xl px-2 py-1 text-xs outline-none transition-colors hover:bg-base-200 data-[active-item]:bg-base-300';

export const actionTriggerMenuFieldCardClasses = 'border-base-300 bg-base-200/40 rounded-xl border p-2';

export const actionTriggerMenuFieldInputClasses =
	'input input-xs bg-base-100 border-base-300 h-7 w-full rounded-xl text-xs';

export const actionTriggerChipButtonClasses =
	'btn btn-xs app-text-neutral h-7 min-h-0 items-center overflow-hidden rounded-full border border-transparent bg-base-200/70 px-2 text-left normal-case shadow-none transition-colors hover:bg-base-300/80';

export const actionTriggerChipRemoveButtonClasses =
	'btn btn-ghost btn-xs text-error hover:bg-error/10 size-5 min-h-0 shrink-0 rounded-full p-0 shadow-none';

/**
 * Place inside ActionTriggerMenuHeader, after SearchableMenuInput when the
 * menu has a search field. The header therefore always follows:
 *
 *   search -> title/actions/errors
 */
export const actionTriggerMenuHeaderRowClasses = 'flex min-h-7 min-w-0 items-center justify-between gap-2 px-1';

export const actionTriggerMenuMetaClasses = 'text-base-content/60 shrink-0 whitespace-nowrap text-[11px]';

type AriakitMenuProps = ComponentProps<typeof Menu>;

export interface ActionTriggerMenuProps extends Omit<AriakitMenuProps, 'children' | 'className' | 'store'> {
	store: NonNullable<AriakitMenuProps['store']>;
	children: ReactNode;
	className?: string;
}

export function ActionTriggerMenu({
	store,
	children,
	className = '',
	portal = true,
	gutter = 8,
	overflowPadding = 8,
	fitViewport = true,
	autoFocusOnShow = true,
	...props
}: ActionTriggerMenuProps) {
	return (
		<Menu
			{...props}
			store={store}
			portal={portal}
			gutter={gutter}
			overflowPadding={overflowPadding}
			fitViewport={fitViewport}
			autoFocusOnShow={autoFocusOnShow}
			className={`${actionTriggerMenuClasses} max-h-(--popover-available-height) ${className}`}
		>
			{children}
		</Menu>
	);
}

interface ActionTriggerMenuRegionProps {
	children: ReactNode;
	className?: string;
}

export function ActionTriggerMenuHeader({ children, className = '' }: ActionTriggerMenuRegionProps) {
	return (
		<div className={`bg-base-100 border-base-300 z-10 mb-2 shrink-0 space-y-2 border-b pb-2 ${className}`}>
			{children}
		</div>
	);
}

export function ActionTriggerMenuBody({ children, className = '' }: ActionTriggerMenuRegionProps) {
	return <div className={`min-h-0 flex-1 overflow-y-auto overscroll-contain ${className}`}>{children}</div>;
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
	secondaryLabelClassName = 'max-w-28 truncate text-xs opacity-70 font-normal',
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
