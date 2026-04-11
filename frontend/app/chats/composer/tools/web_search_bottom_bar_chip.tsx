import type { KeyboardEvent, MouseEvent } from 'react';
import { useCallback, useEffect, useMemo } from 'react';

import { FiCheck, FiEdit2, FiGlobe, FiX } from 'react-icons/fi';

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
	selectedCount,
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
	selectedCount: number;
	canEdit: boolean;
	argsStatus?: UIToolUserArgsStatus;
	onEnabledChange: (enabled: boolean) => void;
	onSelectTool: (tool: ToolListItem) => void;
	onEditOptions: () => void;
	isInputLocked?: boolean;
}) {
	const hasDropdown = eligibleTools.length > 1;

	// Hooks must not be conditional; safe to always create the store.
	const menu = useMenuStore({ placement: 'top', focusLoop: true });
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

	const enabledCount = selectedCount > 0 ? selectedCount : enabled ? 1 : 0;
	const isConfigured = enabledCount > 0 || enabled;
	const isArgsBad = Boolean(isConfigured && argsStatus?.hasSchema && !argsStatus.isSatisfied);

	const title = useMemo(() => {
		const lines: string[] = [];
		lines.push('Web search');
		if (selectedLabel) lines.push(`Tool: ${selectedLabel}`);
		lines.push(isConfigured ? 'Status: Enabled' : 'Status: Disabled');
		if (hasDropdown) lines.push(`Available tools: ${eligibleTools.length}`);
		if (isArgsBad) lines.push('Options: Missing required fields');
		return lines.join('\n');
	}, [eligibleTools.length, hasDropdown, isArgsBad, isConfigured, selectedLabel]);

	const enable = useCallback(() => {
		onEnabledChange(true);
	}, [onEnabledChange]);

	const disable = useCallback(() => {
		onEnabledChange(false);
	}, [onEnabledChange]);

	if (eligibleTools.length === 0) return null;

	const triggerContent = (
		<ActionTriggerChipContent
			icon={<FiGlobe size={14} />}
			label="Web search"
			count={
				enabledCount > 0 ? (
					<span className="badge badge-success badge-xs bg-success/30">{enabledCount}</span>
				) : undefined
			}
			suffix={isConfigured ? <FiCheck size={14} className="shrink-0" /> : undefined}
			open={open}
			showChevron={hasDropdown}
			labelClassName="max-w-24 truncate text-xs font-normal"
		/>
	);

	return (
		<HoverTip content={title} placement="top" wrapperElement="div" wrapperClassName="inline-flex max-w-full">
			<div
				className={`${actionTriggerChipSurfaceClasses} border ${isConfigured ? 'border-info/50 bg-info/10 hover:bg-info/15' : 'border-transparent'} ${isInputLocked ? 'opacity-60' : ''}`}
				data-bottom-bar-websearch
			>
				{hasDropdown ? (
					<MenuButton
						store={menu}
						className="btn btn-xs text-neutral-custom bg-base-200/70 hover:bg-base-300/80 h-auto min-h-0 flex-1 gap-0 px-0 py-0 text-left font-normal shadow-none"
						aria-label={isConfigured ? 'Choose web search tool' : 'Enable web search'}
						disabled={isInputLocked}
						onClick={(event: MouseEvent) => {
							if (isInputLocked) {
								event.preventDefault();
								return;
							}
							if (!isConfigured) {
								event.preventDefault();
								event.stopPropagation();
								enable();
							}
						}}
						onKeyDown={(event: KeyboardEvent) => {
							if (isInputLocked) {
								event.preventDefault();
								return;
							}
							if (!isConfigured && (event.key === 'Enter' || event.key === ' ')) {
								event.preventDefault();
								event.stopPropagation();
								enable();
							}
						}}
					>
						{triggerContent}
					</MenuButton>
				) : (
					<button
						type="button"
						className="btn btn-xs text-neutral-custom bg-base-200/70 hover:bg-base-300/80 h-auto min-h-0 flex-1 gap-0 px-0 py-0 text-left font-normal shadow-none"
						onClick={() => {
							if (isInputLocked || isConfigured) return;
							enable();
						}}
						aria-label={isConfigured ? 'Web search enabled' : 'Enable web search'}
						disabled={isInputLocked}
					>
						{triggerContent}
					</button>
				)}

				{isArgsBad ? <span className="badge badge-warning badge-xs ml-1">Options</span> : null}

				{isConfigured && canEdit ? (
					<button
						type="button"
						className="btn btn-xs text-neutral-custom bg-base-200/70 hover:bg-base-300/80 h-auto min-h-0 shrink-0 px-1 py-0 shadow-none"
						onClick={event => {
							event.preventDefault();
							event.stopPropagation();
							onEditOptions();
						}}
						aria-label="Edit web search options"
						disabled={isInputLocked}
					>
						<FiEdit2 size={12} />
					</button>
				) : null}

				{isConfigured ? (
					<button
						type="button"
						className="btn btn-xs text-neutral-custom bg-base-200/70 hover:bg-base-300/80 h-auto min-h-0 shrink-0 px-1 py-0 shadow-none"
						onClick={event => {
							event.preventDefault();
							event.stopPropagation();
							disable();
							menu.hide();
						}}
						aria-label="Clear web search"
						disabled={isInputLocked}
					>
						<FiX size={12} />
					</button>
				) : null}

				<Menu
					store={menu}
					gutter={8}
					overflowPadding={8}
					portal
					className="rounded-box bg-base-100 text-base-content border-base-300 z-50 max-h-72 max-w-lg min-w-60 overflow-y-auto border p-2 shadow-xl"
					autoFocusOnShow
				>
					<div className="text-base-content/70 mb-2 text-xs font-semibold">Web search tools</div>

					{eligibleTools.map(tool => {
						const isSelected = !!selected && isSameTool(selected, tool);

						return (
							<MenuItem
								key={toolKey(tool)}
								className="data-active-item:bg-base-200 mb-1 rounded-xl last:mb-0"
								onClick={() => {
									if (isInputLocked) return;
									onSelectTool(tool);
									menu.hide();
								}}
							>
								<div className="flex items-center gap-2 px-2 py-1">
									<FiGlobe size={14} />
									<div className="min-w-0 flex-1">
										<div className="truncate text-xs font-medium">
											{tool.toolDefinition.displayName || tool.toolSlug}
										</div>
										<div className="text-base-content/70 truncate text-xs">
											{tool.bundleSlug ?? tool.bundleID}/{tool.toolSlug}@{tool.toolVersion}
										</div>
									</div>
									{isSelected ? <FiCheck size={14} className="text-success shrink-0" /> : null}
								</div>
							</MenuItem>
						);
					})}
				</Menu>
			</div>
		</HoverTip>
	);
}
