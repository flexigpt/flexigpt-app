import { useCallback, useEffect, useMemo } from 'react';

import { FiEdit2, FiGlobe, FiX } from 'react-icons/fi';

import { Menu, MenuButton, MenuItem, useMenuStore, useStoreState } from '@ariakit/react';

import type { ToolListItem, UIToolUserArgsStatus } from '@/spec/tool';

import { ActionTriggerChipContent, actionTriggerChipSurfaceClasses } from '@/components/action_trigger_chip';
import { HoverTip } from '@/components/ariakit_hover_tip';

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
	const open = useStoreState(menu, 'open');
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
	const summarySecondaryLabel = useMemo(() => {
		if (enabled) return 'On';
		if (hasDropdown && selectedLabel) return 'On';
		if (hasDropdown) return 'Choose tool';
		if (selectedLabel) return 'On';
		return 'Off';
	}, [enabled, hasDropdown, selectedLabel]);

	const title = useMemo(() => {
		const lines: string[] = [];
		lines.push('Web search');
		if (selectedLabel) lines.push(`Tool: ${selectedLabel}`);
		lines.push(enabled ? 'Status: Enabled' : 'Status: Disabled');
		if (hasDropdown) lines.push(`Available tools: ${eligibleTools.length}`);
		if (isArgsBad) lines.push('Options: Missing required fields');
		return lines.join('\n');
	}, [eligibleTools.length, enabled, hasDropdown, isArgsBad, selectedLabel]);

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

	const summaryContent = (
		<ActionTriggerChipContent
			icon={<FiGlobe size={14} />}
			label="Web search"
			secondaryLabel={summarySecondaryLabel}
			open={open}
			showChevron={hasDropdown}
			labelClassName="max-w-24 truncate text-xs font-normal"
			secondaryLabelClassName="max-w-28 truncate text-xs opacity-70"
		/>
	);

	if (eligibleTools.length === 0) return null;

	return (
		<HoverTip content={title} placement="top" wrapperElement="div" wrapperClassName="inline-flex max-w-full">
			<div
				className={`${actionTriggerChipSurfaceClasses} ${enabled ? 'border-info/50 bg-info/10 hover:bg-info/15 border' : 'border-none'} ${isInputLocked ? 'opacity-60' : ''}`}
				data-bottom-bar-websearch
			>
				{hasDropdown ? (
					<MenuButton
						store={menu}
						className="btn btn-ghost btn-xs h-auto min-h-0 flex-1 gap-0 px-0 py-0 text-left font-normal shadow-none hover:bg-transparent disabled:bg-transparent"
						aria-label="Choose web search tool"
						disabled={isInputLocked}
					>
						{summaryContent}
					</MenuButton>
				) : enabled ? (
					<div className="flex min-w-0 flex-1 items-center">{summaryContent}</div>
				) : (
					<button
						type="button"
						className="btn btn-ghost btn-xs h-auto min-h-0 flex-1 gap-0 px-0 py-0 text-left font-normal shadow-none hover:bg-transparent disabled:bg-transparent"
						onClick={onClickEnableWhenDisabled}
						aria-label="Enable web search"
						disabled={isInputLocked}
					>
						{summaryContent}
					</button>
				)}

				{isArgsBad ? <span className="badge badge-warning badge-xs ml-1">Options</span> : null}

				{enabled && canEdit ? (
					<button
						type="button"
						className="btn btn-ghost btn-xs h-auto min-h-0 shrink-0 px-1 py-0 shadow-none hover:bg-transparent"
						onClick={e => {
							e.preventDefault();
							e.stopPropagation();
							onEditOptions();
						}}
						aria-label="Edit web search options"
						disabled={isInputLocked}
					>
						<FiEdit2 size={12} />
					</button>
				) : null}

				{enabled ? (
					<button
						type="button"
						className="btn btn-ghost btn-xs h-auto min-h-0 shrink-0 px-1 py-0 shadow-none hover:bg-transparent"
						onClick={e => {
							e.preventDefault();
							e.stopPropagation();
							disable();
						}}
						aria-label="Disable web search"
						disabled={isInputLocked}
					>
						<FiX size={12} />
					</button>
				) : null}

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
			</div>
		</HoverTip>
	);
}
