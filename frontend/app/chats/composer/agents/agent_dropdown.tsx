import { useMemo, useState } from 'react';
import { FiCheck, FiEye, FiLayers, FiRefreshCw } from 'react-icons/fi';

import { MenuButton, MenuItem, useMenuStore, useStoreState } from '@ariakit/react';

import { getDistinctUserFacingLabel, getPrimaryUserFacingLabel } from '@/lib/user_facing_presentation';

import type { AgentCatalogOption } from '@/apis/agent_management';

import {
	actionTriggerChipButtonClasses,
	ActionTriggerChipContent,
	ActionTriggerMenu,
	ActionTriggerMenuBody,
	ActionTriggerMenuHeader,
	actionTriggerMenuHeaderRowClasses,
	actionTriggerMenuItemClasses,
	actionTriggerMenuMetaClasses,
} from '@/components/action_trigger_chip';
import { ComposerDropdownSelectedPanel } from '@/components/composer_dropdown_selected_panel';
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
import { AgentDetailsModal } from '@/agents/agent_details_modal';

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

function AgentOptionSummary({ option }: { option: AgentCatalogOption }) {
	const logicalName = getDistinctUserFacingLabel(option.displayName, option.agent.name);

	return (
		<>
			<div className="truncate text-xs font-medium">{option.displayName}</div>
			<div className="mt-1 text-[10px] opacity-70">
				{[logicalName, option.agent.builtIn ? 'Built-in' : undefined, option.agent.managed ? 'Managed' : undefined]
					.filter(Boolean)
					.join(' · ')}
			</div>

			{option.description ? <div className="mt-1 line-clamp-2 text-xs opacity-75">{option.description}</div> : null}

			{!option.isSelectable ? (
				<div className="text-warning mt-1 text-xs">
					{option.availabilityReason ?? 'This Agent is not currently available.'}
				</div>
			) : null}
		</>
	);
}

function SelectedAgentSummary({ option, onView }: { option: AgentCatalogOption; onView: () => void }) {
	return (
		<div className="border-base-300 bg-base-200 flex w-full items-start gap-2 rounded-lg border p-2 text-left">
			<div className="min-w-0 flex-1">
				<div className="flex flex-wrap items-center gap-2">
					<AgentOptionSummary option={option} />
				</div>
			</div>

			<HoverTip content="View Agent details" placement="top" wrapperElement="div" wrapperClassName="inline-flex">
				<button
					type="button"
					aria-label={`View details for ${option.displayName}`}
					className="btn btn-ghost btn-xs btn-square shrink-0 rounded-lg"
					onClick={onView}
				>
					<FiEye size={14} />
				</button>
			</HoverTip>
		</div>
	);
}

export function AgentDropdown({ manager }: AgentDropdownProps) {
	const menu = useMenuStore({
		placement: 'top-start',
		focusLoop: true,
	});
	const open = useStoreState(menu, 'open');
	const menuContentElement = useStoreState(menu, 'contentElement');
	const [searchQuery, setSearchQuery] = useSearchableMenuState(open);
	const [viewedAgent, setViewedAgent] = useState<AgentCatalogOption | null>(null);

	const selectedLabel = manager.selectedAgent
		? getPrimaryUserFacingLabel(manager.selectedAgent.displayName, manager.selectedAgent.agent.name)
		: manager.loading
			? 'Loading Agent...'
			: 'Agent';
	const triggerTitle = manager.selectedAgent ? 'Change the selected Agent' : 'Apply an Agent starter recipe';

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

	const selectedAgentOption = sortedAgentOptions.find(option => option.key === manager.selectedAgentKey) ?? null;
	const hasUnavailableSelectedAgent = Boolean(manager.selectedAgentKey && !selectedAgentOption);
	const topListAgentOptions = displayedAgentOptions.filter(option => option.key !== manager.selectedAgentKey);
	const selectableAgents = topListAgentOptions.filter(option => option.isSelectable);
	const unavailableAgents = topListAgentOptions.filter(option => !option.isSelectable);
	const firstSelectableAgent = selectableAgents[0] ?? null;
	const agentHeaderMeta = isSearchQueryActive(searchQuery)
		? `${selectableAgents.length} match${selectableAgents.length === 1 ? '' : 'es'}`
		: `${selectableAgents.length} available`;
	const openAgentDetails = (option: AgentCatalogOption) => {
		menu.hide();
		setViewedAgent(option);
	};

	const renderAgentOption = (option: AgentCatalogOption) => {
		const disabled = manager.loading || manager.isApplying || !option.isSelectable;

		return (
			<div
				key={option.key}
				className="border-base-300 hover:bg-base-200 flex w-full flex-col rounded-lg border p-2 text-left transition-colors"
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
						<div className="min-w-0 flex-1">
							<AgentOptionSummary option={option} />
						</div>
					</MenuItem>

					<div className="flex shrink-0 items-start gap-1">
						<HoverTip content="View Agent details" placement="top" wrapperElement="div" wrapperClassName="inline-flex">
							<button
								type="button"
								aria-label={`View details for ${option.displayName}`}
								className="btn btn-ghost btn-xs btn-square rounded-lg"
								onClick={() => {
									openAgentDetails(option);
								}}
							>
								<FiEye size={14} />
							</button>
						</HoverTip>

						{!option.isSelectable ? <span className="badge badge-warning badge-xs shrink-0">Unavailable</span> : null}
					</div>
				</div>
			</div>
		);
	};

	return (
		<div className="relative w-full">
			<HoverTip content={triggerTitle} placement="top" wrapperElement="div" wrapperClassName="w-full">
				<MenuButton
					store={menu}
					className={`${actionTriggerChipButtonClasses} w-full justify-center ${
						manager.selectedAgent
							? 'border-secondary/50 bg-secondary/10 hover:bg-secondary/15'
							: open
								? 'border-base-300 bg-base-300/60'
								: ''
					}`}
				>
					<ActionTriggerChipContent
						icon={<FiLayers size={14} />}
						label={selectedLabel}
						open={open}
						suffix={manager.selectedAgent ? <FiCheck size={14} className="shrink-0" /> : undefined}
						className="w-full justify-center"
						labelClassName="min-w-0 truncate text-center text-xs font-normal"
					/>
				</MenuButton>
			</HoverTip>

			{open ? (
				<ActionTriggerMenu store={menu}>
					<ActionTriggerMenuHeader>
						{manager.agentOptions.length > 0 ? (
							<SearchableMenuInput
								open={open}
								query={searchQuery}
								onQueryChange={setSearchQuery}
								placeholder="Search Agents..."
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

						<div className={actionTriggerMenuHeaderRowClasses}>
							<div className="min-w-0 flex-1 truncate text-xs font-semibold">Agents</div>
							<div className="flex shrink-0 items-center gap-2">
								<span className={actionTriggerMenuMetaClasses}>{agentHeaderMeta}</span>
								<HoverTip
									content="Reload available Agents"
									placement="top"
									wrapperElement="div"
									wrapperClassName="inline-flex"
								>
									<button
										type="button"
										className="btn btn-ghost btn-xs rounded-lg"
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
						</div>

						{manager.error ? (
							<div className="alert alert-error rounded-xl text-xs">
								<span>{manager.error}</span>
							</div>
						) : null}

						{manager.actionError ? (
							<div className="alert alert-error rounded-xl text-xs whitespace-pre-wrap">
								<span>{manager.actionError}</span>
							</div>
						) : null}
					</ActionTriggerMenuHeader>

					<ActionTriggerMenuBody>
						{manager.loading && manager.agentOptions.length === 0 ? (
							<div className={`${actionTriggerMenuItemClasses} text-base-content/60 cursor-default`}>
								Loading Agents...
							</div>
						) : manager.agentOptions.length === 0 ? (
							<div className={`${actionTriggerMenuItemClasses} text-base-content/60 cursor-default`}>
								No Agents are available.
							</div>
						) : topListAgentOptions.length === 0 ? (
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
					</ActionTriggerMenuBody>

					<ComposerDropdownSelectedPanel
						label="Selected Agent"
						ariaLabel="Selected Agent"
						isEmpty={!selectedAgentOption && !hasUnavailableSelectedAgent}
						summary={
							selectedAgentOption ? (
								<span>1 selected</span>
							) : hasUnavailableSelectedAgent ? (
								<span className="text-warning">Saved selection unavailable</span>
							) : undefined
						}
						emptyLabel="Choose an Agent above to apply its starter recipe."
						help="Applying an Agent saves Session intent. The runtime resolves its effective Model, Tools, Skills, MCP, and instructions for the next Run."
					>
						{selectedAgentOption ? (
							<SelectedAgentSummary
								option={selectedAgentOption}
								onView={() => {
									openAgentDetails(selectedAgentOption);
								}}
							/>
						) : hasUnavailableSelectedAgent ? (
							<div className="alert alert-warning rounded-xl p-2 text-xs">
								The saved Agent selection is not currently available. Choose an available Agent above to replace it.
							</div>
						) : null}
					</ComposerDropdownSelectedPanel>
				</ActionTriggerMenu>
			) : null}

			<AgentDetailsModal
				isOpen={viewedAgent !== null}
				agent={viewedAgent?.agent ?? null}
				onClose={() => {
					setViewedAgent(null);
				}}
				capabilityControls={
					viewedAgent?.key === manager.selectedAgentKey
						? {
								isExcluded: occurrence => Boolean(manager.getCapabilityExclusion(occurrence)),
								onToggle: occurrence => {
									manager.toggleCapabilityExclusion(occurrence);
								},
							}
						: undefined
				}
			/>
		</div>
	);
}
