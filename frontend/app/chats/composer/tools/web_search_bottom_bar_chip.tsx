import { useCallback, useEffect, useMemo } from 'react';

import { FiChevronUp, FiEdit2, FiGlobe, FiX } from 'react-icons/fi';

import { Menu, MenuButton, MenuItem, useMenuStore } from '@ariakit/react';

import type { ToolListItem, UIToolUserArgsStatus } from '@/spec/tool';

type SelectedIdentity = { bundleID: string; toolSlug: string; toolVersion: string };

function isSameTool(a: SelectedIdentity, b: ToolListItem) {
	return a.bundleID === b.bundleID && a.toolSlug === b.toolSlug && a.toolVersion === b.toolVersion;
}

function toolKey(t: ToolListItem) {
	return `${t.bundleID}-${t.toolSlug}-${t.toolVersion}`;
}

export function WebSearchBottomBarChip({
	eligibleTools,
	enabled,
	selected,
	canEdit,
	argsStatus,
	onEnabledChange,
	onSelectTool,
	onEditOptions,
	isInputLocked = false,
}: {
	eligibleTools: ToolListItem[];
	enabled: boolean;
	selected?: SelectedIdentity;
	canEdit: boolean;
	argsStatus?: UIToolUserArgsStatus;
	onEnabledChange: (enabled: boolean) => void;
	onSelectTool: (tool: ToolListItem) => void;
	onEditOptions: () => void;
	isInputLocked?: boolean;
}) {
	const hasDropdown = eligibleTools.length > 1;

	// Hooks must not be conditional; safe to always create the store.
	const menu = useMenuStore({ placement: 'top-end', focusLoop: true });
	useEffect(() => {
		if (isInputLocked) menu.hide();
	}, [isInputLocked, menu]);

	const selectedTool = useMemo(() => {
		if (!selected) return undefined;
		return eligibleTools.find(t => isSameTool(selected, t));
	}, [eligibleTools, selected]);

	const selectedLabel = useMemo(() => {
		if (!selectedTool) return '';

		return selectedTool.toolDefinition.displayName ?? selectedTool.toolSlug ?? '';
	}, [selectedTool]);

	const isArgsBad = Boolean(enabled && argsStatus?.hasSchema && !argsStatus.isSatisfied);

	const title = useMemo(() => {
		const lines: string[] = [];
		lines.push('Web search');
		if (selectedLabel) lines.push(`Tool: ${selectedLabel}`);
		lines.push(enabled ? 'Status: Enabled' : 'Status: Disabled');
		if (isArgsBad) lines.push('Options: Missing required fields');
		return lines.join('\n');
	}, [enabled, isArgsBad, selectedLabel]);

	const enable = useCallback(() => {
		onEnabledChange(true);
	}, [onEnabledChange]);
	const disable = useCallback(() => {
		onEnabledChange(false);
	}, [onEnabledChange]);

	const onClickEnableWhenDisabled = useCallback(() => {
		if (isInputLocked) return;
		// Requested behavior: clicking the chip enables only when currently disabled.
		if (!enabled) enable();
	}, [isInputLocked, enabled, enable]);

	const onClickDisableButton = useCallback(
		(e: React.MouseEvent) => {
			if (isInputLocked) return;
			e.preventDefault();
			e.stopPropagation();
			disable();
		},
		[disable, isInputLocked]
	);

	const onClickEditButton = useCallback(
		(e: React.MouseEvent) => {
			if (isInputLocked) return;
			e.preventDefault();
			e.stopPropagation();
			onEditOptions();
		},
		[isInputLocked, onEditOptions]
	);

	if (eligibleTools.length === 0) return null;

	return (
		<div
			className={
				'text-neutral-custom flex items-center gap-1 rounded-2xl border px-2 py-0 text-xs font-normal' +
				(enabled ? 'bg-info/10 border-info/50 hover:bg-info/15' : 'bg-base-200 hover:bg-base-300/80 border-0') +
				(isInputLocked ? ' opacity-60' : '')
			}
			title={title}
			data-bottom-bar-websearch
		>
			{/* Main chip/label area
          - When disabled: it's a button that enables
          - When enabled: it's non-clickable (disabling only via the X button)
      */}
			{enabled ? (
				<div className="flex items-center gap-2">
					<FiGlobe size={14} />
					<span className="max-w-28 truncate">Web search</span>
					{selectedLabel ? <span className="max-w-36 truncate">{selectedLabel}</span> : null}
				</div>
			) : (
				<button
					type="button"
					className="flex items-center gap-2"
					onClick={onClickEnableWhenDisabled}
					aria-label="Enable web search"
					disabled={isInputLocked}
				>
					<FiGlobe size={14} />
					<span className="max-w-28 truncate">Enable web search</span>
					{selectedLabel ? <span className="max-w-36 truncate">{selectedLabel}</span> : null}
				</button>
			)}

			{/* Args warning badge (only meaningful when enabled) */}
			{isArgsBad ? <span className="badge badge-warning badge-xs ml-1">Options</span> : null}

			{/* Edit options (only when enabled) */}
			{enabled && canEdit ? (
				<button
					type="button"
					className="btn btn-ghost btn-xs p-0 shadow-none"
					onClick={onClickEditButton}
					title="Edit web-search options"
					aria-label="Edit web-search options"
					disabled={isInputLocked}
				>
					<FiEdit2 size={12} />
				</button>
			) : null}

			{/* Disable (only when enabled). This is now the only way to disable. */}
			{enabled ? (
				<button
					type="button"
					className="btn btn-ghost btn-xs p-0 shadow-none"
					onClick={onClickDisableButton}
					title="Disable web search"
					aria-label="Disable web search"
					disabled={isInputLocked}
				>
					<FiX size={12} />
				</button>
			) : null}

			{/* Dropdown only when multiple eligible tools */}
			{hasDropdown ? (
				<>
					<MenuButton
						store={menu}
						className="btn btn-ghost btn-xs p-0 shadow-none"
						aria-label="Choose web-search tool"
						title="Choose web-search tool"
						disabled={isInputLocked}
					>
						<FiChevronUp size={14} />
					</MenuButton>

					<Menu
						store={menu}
						gutter={6}
						className="rounded-box bg-base-100 text-base-content border-base-300 z-50 max-h-72 min-w-72 overflow-y-auto border p-2 shadow-xl"
						autoFocusOnShow
						portal
					>
						<div className="text-base-content/70 mb-2 text-xs font-semibold">Web search tools</div>

						{eligibleTools.map(t => {
							const isSelected = !!selected && isSameTool(selected, t);

							return (
								<MenuItem
									key={toolKey(t)}
									className="data-active-item:bg-base-200 mb-1 rounded-xl last:mb-0"
									onClick={() => {
										if (isInputLocked) return;
										onSelectTool(t);
										menu.hide();
									}}
								>
									<div className="flex items-center gap-2 px-2 py-1">
										<FiGlobe size={14} />
										<div className="min-w-0 flex-1">
											<div className="truncate text-xs font-medium">
												{t.toolDefinition.displayName || t.toolSlug}
												{isSelected ? ' (selected)' : ''}
											</div>
											<div className="text-base-content/70 truncate text-xs">
												{t.bundleSlug ?? t.bundleID}/{t.toolSlug}@{t.toolVersion}
											</div>
										</div>
									</div>
								</MenuItem>
							);
						})}
					</Menu>
				</>
			) : null}
		</div>
	);
}
