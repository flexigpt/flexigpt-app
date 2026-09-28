import { useMemo, useState } from 'react';
import { FiCheck, FiEye, FiLayers, FiRefreshCcw, FiRefreshCw, FiTrash2 } from 'react-icons/fi';

import { Menu, MenuButton, MenuItem, useMenuStore, useStoreState } from '@ariakit/react';

import type { AgentCatalogOption } from '@/apis/agent_management';

import {
	actionTriggerChipButtonClasses,
	ActionTriggerChipContent,
	actionTriggerMenuItemClasses,
	actionTriggerMenuWideClasses,
} from '@/components/action_trigger_chip';
import { GroupedMenuSection, GroupedMenuSubheading } from '@/components/grouped_menu_sections';
import { HoverTip } from '@/components/hover_tip';
import { searchableMenuEmptyStateClasses, SearchableMenuInput } from '@/components/searchmenu/searchable_menu';
import {
	focusFirstSearchableMenuItem,
	isSearchQueryActive,
	rankSearchableItems,
	useSearchableMenuState,
} from '@/components/searchmenu/searchable_menu_utils';

import type { AgentManagerState } from '@/chats/composer/agents/use_agent_manager';
import { AgentViewModal } from '@/chats/composer/agents/agent_view_modal';

interface AgentDropdownProps {
	manager: AgentManagerState;
}

const agentCollator = new Intl.Collator(undefined, {
	numeric: true,
	sensitivity: 'base',
});

function compareAgentOptions(left: AgentCatalogOption, right: AgentCatalogOption): number {
	if (left.agent.builtIn !== right.agent.builtIn) {
		return left.agent.builtIn ? -1 : 1;
	}

	const displayNameCompare = agentCollator.compare(left.displayName, right.displayName);
	if (displayNameCompare !== 0) {
		return displayNameCompare;
	}

	const logicalNameCompare = agentCollator.compare(left.agent.name, right.agent.name);
	if (logicalNameCompare !== 0) {
		return logicalNameCompare;
	}

	return agentCollator.compare(left.key, right.key);
}

export function AgentDropdown({ manager }: AgentDropdownProps) {
	const menu = useMenuStore({
		placement: 'top',
		focusLoop: true,
	});
	const open = useStoreState(menu, 'open');
	const menuContentElement = useStoreState(menu, 'contentElement');
	const [searchQuery, setSearchQuery] = useSearchableMenuState(open);
	const [viewedAgent, setViewedAgent] = useState<AgentCatalogOption | null>(null);

	const selectedLabel = manager.selectedAgent?.displayName || (manager.loading ? 'Loading Agent...' : 'Agent');
	const triggerTitle = manager.selectedAgent
		? `${manager.selectedAgent.displayName} / ${manager.selectedAgent.agent.name}`
		: 'Apply an Agent starter recipe';

	const sortedAgentOptions = useMemo(
		() => [...manager.agentOptions].toSorted(compareAgentOptions),
		[manager.agentOptions]
	);
	const displayedAgentOptions = useMemo(() => {
		if (!isSearchQueryActive(searchQuery)) {
			return sortedAgentOptions;
		}

		return rankSearchableItems(sortedAgentOptions, {
			query: searchQuery,
			getKey: option => option.key,
			getFields: option => [
				{ value: option.displayName, weight: 6 },
				{ value: option.agent.name, weight: 5 },
				{ value: option.description, weight: 3 },
				{ value: option.ref.rootID, weight: 2 },
				{ value: option.ref.artifactID, weight: 2 },
				{ value: option.availabilityReason, weight: 1 },
			],
			fallbackCompare: compareAgentOptions,
		});
	}, [searchQuery, sortedAgentOptions]);

	const selectableAgents = displayedAgentOptions.filter(option => option.isSelectable);
	const unavailableAgents = displayedAgentOptions.filter(option => !option.isSelectable);
	const firstSelectableAgent = selectableAgents[0] ?? null;
	const resetTargetsBase = manager.baseAgentKey !== null && manager.baseAgentKey === manager.defaultAgentKey;
	const resetLabel = resetTargetsBase ? 'Clear to base' : 'Reset to default';

	const renderAgentOption = (option: AgentCatalogOption) => {
		const selected = option.key === manager.selectedAgentKey;
		const isBase = option.key === manager.baseAgentKey;
		const isDefault = option.key === manager.defaultAgentKey;
		const disabled = manager.loading || manager.isApplying || !option.isSelectable;
		const statusTip = manager.isTrackingOnly
			? 'This Agent is tracked for restored conversation context. Reapply it to apply its starter recipe.'
			: 'This Agent starter recipe was applied. Composer controls remain user-editable.';
		const resetTip = isDefault
			? 'The default Agent is already active.'
			: resetTargetsBase
				? 'Apply the built-in Base Agent starter recipe.'
				: 'The Base Agent is unavailable, so apply the selectable default Agent instead.';

		return (
			<div
				key={option.key}
				className={`border-base-300 flex w-full flex-col rounded-lg border p-2 text-left transition-colors ${
					selected ? 'bg-base-200' : 'hover:bg-base-200'
				}`}
			>
				<div className="flex w-full items-start gap-2">
					<MenuItem
						store={menu}
						data-searchable-menu-item="true"
						disabled={disabled}
						className={`data-active-item:bg-base-200 flex min-w-0 flex-1 items-start gap-2 rounded-lg p-1 text-left outline-none ${
							disabled ? (option.isSelectable ? 'cursor-wait opacity-70' : 'cursor-not-allowed opacity-60') : ''
						}`}
						onClick={() => {
							if (disabled) {
								return;
							}

							void manager.selectAgent(option.ref).then(applied => {
								if (applied) {
									menu.hide();
								}
							});
						}}
					>
						<div className="pt-0.5">{selected ? <FiCheck size={14} /> : <span className="w-3" />}</div>

						<div className="min-w-0 flex-1">
							<div className="truncate text-xs font-medium">{option.displayName}</div>
							<div className="mt-1 text-[10px] opacity-70">
								{option.agent.name}
								{option.agent.builtIn ? ' / Built-in' : ''}
								{option.agent.managed ? ' / Managed' : ''}
							</div>
							{option.description ? (
								<div className="mt-1 line-clamp-2 text-xs opacity-75">{option.description}</div>
							) : null}
							{!option.isSelectable ? (
								<div className="text-warning mt-1 text-xs">
									{option.availabilityReason ?? 'This Agent is not currently available.'}
								</div>
							) : null}
						</div>
					</MenuItem>

					<div className="flex shrink-0 items-start gap-1">
						<HoverTip content="View Agent details" placement="top" wrapperElement="div" wrapperClassName="inline-flex">
							<button
								type="button"
								aria-label={`View details for ${option.displayName}`}
								className="btn btn-ghost btn-xs btn-square rounded-lg"
								onClick={() => {
									setViewedAgent(option);
								}}
							>
								<FiEye size={14} />
							</button>
						</HoverTip>

						{!option.isSelectable ? <span className="badge badge-warning badge-xs shrink-0">Unavailable</span> : null}
					</div>
				</div>

				{selected ? (
					<div className="border-base-300 mt-2 ml-5 flex flex-wrap items-center justify-between gap-1 border-t pt-1">
						<div className="flex items-center gap-1">
							<HoverTip content={statusTip} placement="top" wrapperElement="div" wrapperClassName="inline-flex">
								<span className={`badge badge-xs ${manager.isTrackingOnly ? 'badge-outline' : 'badge-success'}`}>
									{manager.isTrackingOnly ? 'Tracked' : 'Applied'}
								</span>
							</HoverTip>
							{isBase ? <span className="badge badge-ghost badge-xs">Base</span> : null}
							{!isBase && isDefault ? <span className="badge badge-ghost badge-xs">Default</span> : null}
						</div>

						<div className="flex items-center gap-1">
							<HoverTip
								content="Apply this Agent starter recipe again"
								placement="top"
								wrapperElement="div"
								wrapperClassName="inline-flex"
							>
								<button
									type="button"
									className="btn btn-ghost btn-xs rounded-lg"
									disabled={manager.loading || manager.isApplying || !option.isSelectable}
									onClick={() => {
										void manager.reapplySelectedAgent().then(applied => {
											if (applied) {
												menu.hide();
											}
										});
									}}
								>
									<FiRefreshCcw size={14} className="mr-1" />
									Reapply
								</button>
							</HoverTip>

							<HoverTip content={resetTip} placement="top" wrapperElement="div" wrapperClassName="inline-flex">
								<button
									type="button"
									className="btn btn-ghost btn-xs rounded-lg"
									disabled={
										manager.loading || manager.isApplying || !manager.defaultAgentKey || manager.isDefaultAgentSelected
									}
									onClick={() => {
										void manager.resetToDefaultAgent().then(applied => {
											if (applied) {
												menu.hide();
											}
										});
									}}
								>
									<FiTrash2 size={14} className="mr-1" />
									{resetLabel}
								</button>
							</HoverTip>
						</div>
					</div>
				) : null}
			</div>
		);
	};

	return (
		<div className="relative w-full">
			<HoverTip content={triggerTitle} placement="top" wrapperElement="div" wrapperClassName="w-full">
				<MenuButton store={menu} className={`${actionTriggerChipButtonClasses} w-full justify-center`}>
					<ActionTriggerChipContent
						icon={<FiLayers size={14} />}
						label={selectedLabel}
						open={open}
						suffix={manager.selectedAgent ? <FiCheck size={14} /> : undefined}
						className="w-full justify-center"
						labelClassName="min-w-0 truncate text-center text-xs font-normal"
					/>
				</MenuButton>
			</HoverTip>

			{open ? (
				<Menu
					store={menu}
					portal
					gutter={8}
					overflowPadding={8}
					className={actionTriggerMenuWideClasses}
					autoFocusOnShow={false}
				>
					<div className="mb-2 flex items-start justify-between gap-2 px-1">
						<div className="text-xs opacity-70">
							Agents apply a starter Model, Tools, session Skills, MCP selection, system instruction Skills, and, when
							supplied, an Agent request template. Request templates should only prefill an empty Composer draft. After
							application, the Composer remains user-editable.
						</div>

						<HoverTip
							content="Reload available Agents"
							placement="top"
							wrapperElement="div"
							wrapperClassName="inline-flex"
						>
							<button
								type="button"
								className="btn btn-ghost btn-xs shrink-0 rounded-lg"
								disabled={manager.loading || manager.isApplying}
								onClick={() => {
									void manager.refreshAgents();
								}}
							>
								<FiRefreshCw size={13} />
								<span>Refresh</span>
							</button>
						</HoverTip>
					</div>

					{manager.error ? (
						<div className="alert alert-error mb-2 rounded-2xl text-xs">
							<span>{manager.error}</span>
						</div>
					) : null}

					{manager.actionError ? (
						<div className="alert alert-error mb-2 rounded-2xl text-xs whitespace-pre-wrap">
							<span>{manager.actionError}</span>
						</div>
					) : null}

					{manager.agentOptions.length > 0 ? (
						<SearchableMenuInput
							open={open}
							query={searchQuery}
							onQueryChange={setSearchQuery}
							placeholder="Search Agents..."
							resultCount={displayedAgentOptions.length}
							totalCount={manager.agentOptions.length}
							disabled={manager.loading}
							onFocusFirstItem={() => {
								focusFirstSearchableMenuItem(menuContentElement);
							}}
							onEnterFirstResult={() => {
								if (!firstSelectableAgent || manager.loading || manager.isApplying) {
									return;
								}

								void manager.selectAgent(firstSelectableAgent.ref).then(applied => {
									if (applied) {
										menu.hide();
									}
								});
							}}
							onEscape={() => {
								menu.hide();
							}}
						/>
					) : null}

					{manager.loading && manager.agentOptions.length === 0 ? (
						<div className={`${actionTriggerMenuItemClasses} text-base-content/60 cursor-default`}>
							Loading Agents...
						</div>
					) : manager.agentOptions.length === 0 ? (
						<div className={`${actionTriggerMenuItemClasses} text-base-content/60 cursor-default`}>
							No Agents are available.
						</div>
					) : displayedAgentOptions.length === 0 ? (
						<div className={searchableMenuEmptyStateClasses}>No Agents match your search.</div>
					) : (
						<div className="space-y-2">
							{selectableAgents.length > 0 ? (
								<GroupedMenuSection title="Available" ariaLabel="Available Agents">
									{selectableAgents.map(a => {
										return renderAgentOption(a);
									})}
								</GroupedMenuSection>
							) : null}

							{unavailableAgents.length > 0 ? (
								<GroupedMenuSection
									title="Unavailable"
									ariaLabel="Unavailable Agents"
									separatorBefore={selectableAgents.length > 0}
								>
									<GroupedMenuSubheading tone="warning">Unavailable</GroupedMenuSubheading>
									{unavailableAgents.map(a => {
										return renderAgentOption(a);
									})}
								</GroupedMenuSection>
							) : null}
						</div>
					)}
				</Menu>
			) : null}

			<AgentViewModal
				isOpen={viewedAgent !== null}
				agent={viewedAgent}
				onClose={() => {
					setViewedAgent(null);
				}}
			/>
		</div>
	);
}
