import type { ReactNode } from 'react';

interface ComposerDropdownSelectedPanelProps {
	label: ReactNode;
	ariaLabel?: string;
	summary?: ReactNode;
	actions?: ReactNode;
	help?: ReactNode;
	children?: ReactNode;
	emptyLabel?: string;
	/**
	 * Use this for collection-based panels. React fragments and empty arrays
	 * still count as children, so the component cannot reliably infer whether
	 * a domain has selected content from `children` alone.
	 */
	isEmpty?: boolean;
	className?: string;
}

/**
 * Shared "already added" footer for composer menus.
 *
 * Place this after ActionTriggerMenuBody as a direct child of the Menu.
 * Only the selected items inside this panel scroll when they exceed.
 */
export function ComposerDropdownSelectedPanel({
	label,
	ariaLabel,
	summary,
	actions,
	help,
	children,
	emptyLabel = 'Nothing has been added yet.',
	isEmpty,
	className = '',
}: ComposerDropdownSelectedPanelProps) {
	const hasContent = children !== undefined && children !== null;
	const showEmpty = isEmpty ?? !hasContent;
	const resolvedAriaLabel = ariaLabel ?? (typeof label === 'string' ? label : undefined);

	return (
		<section
			className={`border-base-300 bg-base-100 z-20 mt-2 shrink-0 border-t pt-2 ${className}`}
			data-composer-dropdown-selected
			data-empty={showEmpty ? 'true' : undefined}
			aria-label={resolvedAriaLabel}
		>
			<div className="flex min-w-0 items-center justify-between gap-2 px-1">
				<div className="flex min-w-0 items-center gap-2">
					<div className="text-base-content/70 min-w-0 truncate text-xs font-semibold">{label}</div>
					{summary ? (
						<div className="text-base-content/60 flex shrink-0 items-center gap-1 text-[11px]">{summary}</div>
					) : null}
				</div>
				{actions ? <div className="flex shrink-0 items-center gap-1">{actions}</div> : null}
			</div>

			{showEmpty ? (
				<div className="border-base-300/70 bg-base-200/40 text-base-content/60 mt-2 rounded-xl border border-dashed p-2 text-xs">
					<div>{emptyLabel}</div>
					{help ? <div className="text-base-content/60 mt-1 leading-4">{help}</div> : null}
				</div>
			) : (
				<>
					{help ? <div className="text-base-content/60 mt-1 px-1 text-xs/4">{help}</div> : null}
					<div className="app-scrollbar-thin mt-2 max-h-96 space-y-1 overflow-y-auto overscroll-contain pr-1">
						{children}
					</div>
				</>
			)}
		</section>
	);
}

interface ComposerCurrentSettingPanelProps {
	label: string;
	value: ReactNode;
	help?: ReactNode;
	isDefault?: boolean;
	onUseDefault?: () => void;
	useDefaultLabel?: string;
}

/**
 * Current-value footer for scalar composer controls. Collection pickers
 * use ComposerDropdownSelectedPanel directly; this keeps simple controls thin
 * without pretending they contain a collection of attached objects.
 */
export function ComposerCurrentSettingPanel({
	label,
	value,
	help,
	isDefault = false,
	onUseDefault,
	useDefaultLabel = 'Use default',
}: ComposerCurrentSettingPanelProps) {
	return (
		<ComposerDropdownSelectedPanel label={label} ariaLabel={label} help={help} isEmpty={false}>
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
