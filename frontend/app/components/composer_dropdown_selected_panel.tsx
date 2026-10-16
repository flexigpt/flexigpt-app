import type { ReactNode } from 'react';

interface ComposerDropdownSelectedPanelProps {
	label: string;
	count?: number;
	children?: ReactNode;
	emptyLabel?: string;
	className?: string;
}

/**
 * Shared sticky "already added" area for composer menus.
 *
 * The containing Ariakit Menu remains the scroll container. Available items
 * stay above this panel; all post-add actions such as remove, disable, edit,
 * activate, and detach belong here.
 */
export function ComposerDropdownSelectedPanel({
	label,
	count,
	children,
	emptyLabel = 'Nothing has been added yet.',
	className = '',
}: ComposerDropdownSelectedPanelProps) {
	const hasContent = children !== undefined && children !== null;

	return (
		<section
			className={`border-base-300 bg-base-100/95 sticky bottom-0 z-20 mt-2 border-t pt-2 backdrop-blur-sm ${className}`}
			data-composer-dropdown-selected
			aria-label={label}
		>
			<div className="mb-1 flex min-w-0 items-center justify-between gap-2 px-1">
				<div className="min-w-0 truncate text-[11px] font-semibold tracking-wide uppercase opacity-70">{label}</div>
				{count && count > 0 ? <span className="badge badge-ghost badge-xs shrink-0">{count}</span> : null}
			</div>

			<div className="app-scrollbar-thin max-h-48 space-y-1 overflow-y-auto overscroll-contain pr-1">
				{hasContent ? children : <div className="text-base-content/60 rounded-xl p-2 text-xs">{emptyLabel}</div>}
			</div>
		</section>
	);
}

interface ComposerCurrentSettingPanelProps {
	label: string;
	value: ReactNode;
	isDefault?: boolean;
	onUseDefault?: () => void;
	useDefaultLabel?: string;
}

/**
 * Sticky current-value panel for scalar composer controls. Collection pickers
 * use ComposerDropdownSelectedPanel directly; this keeps simple controls thin
 * without pretending they contain a collection of attached objects.
 */
export function ComposerCurrentSettingPanel({
	label,
	value,
	isDefault = false,
	onUseDefault,
	useDefaultLabel = 'Use default',
}: ComposerCurrentSettingPanelProps) {
	return (
		<ComposerDropdownSelectedPanel label={label}>
			<div className="bg-base-200 flex min-w-0 items-center gap-2 rounded-xl px-2 py-1 text-xs">
				<span className="min-w-0 flex-1 truncate">{value}</span>
				{!isDefault && onUseDefault ? (
					<button type="button" className="btn btn-ghost btn-xs text-error shrink-0 rounded-lg" onClick={onUseDefault}>
						{useDefaultLabel}
					</button>
				) : (
					<span className="badge badge-ghost badge-xs shrink-0">Current</span>
				)}
			</div>
		</ComposerDropdownSelectedPanel>
	);
}
