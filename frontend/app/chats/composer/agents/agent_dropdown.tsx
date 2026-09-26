import { FiCheck, FiLayers, FiRefreshCw, FiX } from 'react-icons/fi';

import { Menu, MenuButton, MenuItem, useMenuStore, useStoreState } from '@ariakit/react';

import {
	actionTriggerChipButtonClasses,
	ActionTriggerChipContent,
	actionTriggerMenuItemClasses,
	actionTriggerMenuWideClasses,
} from '@/components/action_trigger_chip';
import { HoverTip } from '@/components/hover_tip';

import type { AgentManagerState } from '@/chats/composer/agents/use_agent_manager';

interface AgentDropdownProps {
	manager: AgentManagerState;
}

export function AgentDropdown({ manager }: AgentDropdownProps) {
	const menu = useMenuStore({
		placement: 'top',
		focusLoop: true,
	});
	const open = useStoreState(menu, 'open');

	const selectedLabel = manager.selectedAgent?.displayName || 'Agent';

	return (
		<div className="relative w-full">
			<HoverTip content="Apply an Agent starter recipe" placement="top" wrapperElement="div" wrapperClassName="w-full">
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
					autoFocusOnShow
				>
					<div className="mb-2 px-1 text-xs opacity-70">
						Agents apply a starter Model, Tools, session Skills, MCP selection, system instruction Skills, and, when
						supplied, an Agent request template. Request templates should only prefill an empty Composer draft. After
						application, the Composer remains user-editable.
					</div>

					{manager.error ? (
						<div className="alert alert-error mb-2 rounded-2xl text-xs">
							<span>{manager.error}</span>
						</div>
					) : null}

					{manager.actionError ? (
						<div className="alert alert-warning mb-2 rounded-2xl text-xs whitespace-pre-wrap">
							<span>{manager.actionError}</span>
						</div>
					) : null}

					<div className="mb-2 flex justify-end">
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
					</div>

					{manager.loading ? (
						<div className={`${actionTriggerMenuItemClasses} text-base-content/60 cursor-default`}>
							Loading Agents...
						</div>
					) : manager.agentOptions.length === 0 ? (
						<div className={`${actionTriggerMenuItemClasses} text-base-content/60 cursor-default`}>
							No Agents are available.
						</div>
					) : (
						<div className="space-y-2">
							{manager.agentOptions.map(option => {
								const selected = option.key === manager.selectedAgentKey;
								const disabled = manager.isApplying || !option.isSelectable;

								return (
									<div
										key={option.key}
										className={`border-base-300 flex items-start gap-2 rounded-xl border p-2 ${
											selected ? 'bg-base-200' : ''
										}`}
									>
										<MenuItem
											store={menu}
											disabled={disabled}
											className="flex min-w-0 flex-1 items-start gap-2 rounded-lg p-1 text-left outline-none"
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
													{option.agent.builtIn ? ' · Built-in' : ''}
													{option.agent.managed ? ' · Managed' : ''}
												</div>
												{option.description ? (
													<div className="mt-1 line-clamp-2 text-xs opacity-75">{option.description}</div>
												) : null}
												{!option.isSelectable ? (
													<div className="text-warning mt-1 text-xs">{option.availabilityReason ?? 'Unavailable'}</div>
												) : null}
											</div>
										</MenuItem>
									</div>
								);
							})}
						</div>
					)}

					{manager.selectedAgent ? (
						<div className="border-base-300 mt-3 flex justify-end border-t pt-2">
							<button
								type="button"
								className="btn btn-ghost btn-xs rounded-lg"
								disabled={manager.isApplying}
								onClick={() => {
									manager.clearAgentTracking();
									menu.hide();
								}}
							>
								<FiX size={13} />
								<span>Stop tracking Agent</span>
							</button>
						</div>
					) : null}
				</Menu>
			) : null}
		</div>
	);
}
