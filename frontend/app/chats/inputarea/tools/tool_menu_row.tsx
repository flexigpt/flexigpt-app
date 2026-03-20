import type { MouseEvent, ReactNode } from 'react';

import { FiCheck, FiCode, FiEdit2, FiPlus, FiTool, FiX } from 'react-icons/fi';

import { MenuItem, type MenuStore } from '@ariakit/react';

import type { UIToolUserArgsStatus } from '@/spec/tool';

type PrimaryActionKind = 'attach' | 'detach' | 'remove';

interface ToolMenuRowProps {
	store?: MenuStore;
	disabled?: boolean;
	dataAttachmentChip?: string;
	dataSelectionId?: string;
	title?: string;
	display: string;
	slug: string;
	leftIcon?: ReactNode;
	isSelected?: boolean;

	supportsAutoExecute?: boolean;
	autoExecute?: boolean;
	onAutoExecuteChange?: (next: boolean) => void;
	argsStatus?: UIToolUserArgsStatus;
	onEditOptions?: () => void;
	editIcon?: ReactNode;

	/** Optional JSON/details. */
	onShowDetails?: () => void;
	detailsIcon?: ReactNode;

	/**
	 * Primary action button on the far right.
	 * - attach: adds tool, row remains in list
	 * - detach: removes tool, row remains in list
	 * - remove: removes row from list (caller does it)
	 */
	primaryAction?: {
		kind: PrimaryActionKind;
		onClick: () => void;
		title?: string;
		ariaLabel?: string;
		/** For attach actions, you may want an explicit label. */
		label?: string;
		disabled?: boolean;
	};

	/**
	 * Optional row click handler (e.g. in picker: click row to attach, but
	 * do not allow row-click detach).
	 */
	onRowClick?: () => void;

	hideOnClick?: boolean;
	menuItemClassName?: string;
	contentClassName?: string;
	selectedAriaLabel?: string;
	selectedTitle?: string;
	selectedIconClassName?: string;
	autoExecuteTitle?: string;
}

function stop(e: MouseEvent) {
	e.preventDefault();
	e.stopPropagation();
}

function truncate(s: string, max: number) {
	if (!s) return '';
	return s.length > max ? `${s.slice(0, max - 1)}…` : s;
}

function getArgsBadge(status?: UIToolUserArgsStatus): { label: string; className: string } | null {
	if (!status?.hasSchema) return null;
	// schema exists, but no required fields
	if (status.requiredKeys.length === 0) {
		return { label: 'Args: Optional', className: 'badge badge-ghost badge-xs text-xs p-0' };
	}
	if (status.isSatisfied) {
		return { label: 'Args: OK', className: 'badge badge-success badge-xs text-xs p-0' };
	}
	return {
		label: `Args: ${status.missingRequired.length} Missing`,
		className: 'badge badge-warning badge-xs text-xs p-0',
	};
}

export function ToolMenuRow({
	store,
	disabled,
	dataAttachmentChip,
	dataSelectionId,
	title,
	display,
	slug,
	leftIcon,
	isSelected,
	supportsAutoExecute,
	autoExecute,
	onAutoExecuteChange,
	argsStatus,
	onEditOptions,
	editIcon,
	onShowDetails,
	detailsIcon,
	primaryAction,
	onRowClick,

	hideOnClick = false,
	menuItemClassName = 'data-active-item:bg-base-200 mb-1 rounded-xl last:mb-0',
	contentClassName = 'grid grid-cols-12 items-center gap-x-2 px-2 py-1',
	selectedAriaLabel = 'Selected',
	selectedTitle = 'Selected',
	selectedIconClassName = 'ml-2 text-primary',
	autoExecuteTitle = 'Automatically run tool calls for this tool',
}: ToolMenuRowProps) {
	const truncatedDisplay = truncate(display || 'Tool', 40);
	const badge = getArgsBadge(argsStatus);

	const editNode = editIcon ?? <FiEdit2 size={12} />;
	const detailsNode = detailsIcon ?? <FiCode size={12} />;
	const toolNode = leftIcon ?? <FiTool className="justify-start" size={14} />;

	return (
		<MenuItem
			store={store}
			hideOnClick={hideOnClick}
			disabled={disabled}
			onClick={() => {
				if (disabled) return;
				onRowClick?.();
			}}
			className={menuItemClassName}
			data-attachment-chip={dataAttachmentChip}
			data-selection-id={dataSelectionId}
		>
			<div className={contentClassName} title={title}>
				{/* name + slug + check */}
				<div className="col-span-8 flex items-center gap-1">
					{toolNode}
					<div className="flex-1 justify-start truncate">
						<div className="truncate text-xs font-medium">{truncatedDisplay}</div>
						<div className="text-base-content/70 truncate text-[11px]">{slug}</div>
					</div>
					<div
						className="justify-end"
						title={isSelected ? selectedTitle : ''}
						aria-label={isSelected ? selectedAriaLabel : ''}
					>
						{isSelected ? <FiCheck size={14} className={selectedIconClassName} /> : null}
					</div>
				</div>

				{/* auto-exec column */}
				<div className="col-span-2 shrink-0 justify-self-center whitespace-nowrap">
					{supportsAutoExecute ? (
						<label
							className="flex items-center gap-1 text-[11px]"
							title={autoExecuteTitle}
							onPointerDown={e => {
								// don’t activate MenuItem
								e.stopPropagation?.();
							}}
							onClick={e => {
								// don’t activate MenuItem
								e.stopPropagation?.();
							}}
						>
							<span className="text-base-content/60">Auto</span>
							<input
								type="checkbox"
								className="toggle toggle-xs"
								tabIndex={-1}
								checked={!!autoExecute}
								onChange={e => {
									const next = e.currentTarget.checked;
									onAutoExecuteChange?.(next);
								}}
								aria-label={`Auto-execute ${slug}`}
							/>
						</label>
					) : (
						<span className="text-base-content/40 text-[11px]" title="Auto-exec not applicable">
							—
						</span>
					)}
				</div>

				{/* right actions */}
				<div className="col-span-2 flex items-center justify-end gap-1">
					{badge ? <span className={badge.className}>{badge.label}</span> : null}

					{onEditOptions ? (
						<button
							type="button"
							className="btn btn-ghost btn-xs shrink-0 px-1 py-0 shadow-none"
							onClick={e => {
								stop(e);
								onEditOptions();
							}}
							title="Edit tool options"
							aria-label="Edit tool options"
						>
							{editNode}
						</button>
					) : null}

					{onShowDetails ? (
						<button
							type="button"
							className="btn btn-ghost btn-xs text-base-content/60 shrink-0 px-1 py-0 shadow-none"
							onClick={e => {
								stop(e);
								onShowDetails();
							}}
							title="Show tool details"
							aria-label="Show tool details"
						>
							{detailsNode}
						</button>
					) : null}

					{primaryAction ? (
						primaryAction.kind === 'attach' ? (
							<button
								type="button"
								className="btn btn-ghost btn-xs shrink-0 px-1 py-0 shadow-none"
								disabled={primaryAction.disabled}
								onClick={e => {
									stop(e);
									primaryAction.onClick();
								}}
								title={primaryAction.title ?? 'Attach tool'}
								aria-label={primaryAction.ariaLabel ?? 'Attach tool'}
							>
								<FiPlus size={12} />
							</button>
						) : (
							<button
								type="button"
								className="btn btn-ghost btn-xs text-error shrink-0 px-1 py-0 shadow-none"
								disabled={primaryAction.disabled}
								onClick={e => {
									stop(e);
									primaryAction.onClick();
								}}
								title={primaryAction.title ?? (primaryAction.kind === 'detach' ? 'Detach tool' : 'Remove tool')}
								aria-label={
									primaryAction.ariaLabel ?? (primaryAction.kind === 'detach' ? 'Detach tool' : 'Remove tool')
								}
							>
								<FiX size={12} />
							</button>
						)
					) : null}
				</div>
			</div>
		</MenuItem>
	);
}
