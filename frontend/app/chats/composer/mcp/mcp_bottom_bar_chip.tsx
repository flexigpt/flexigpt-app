import { type MouseEvent, type ReactNode, useEffect, useMemo, useState } from 'react';

import {
	FiCheck,
	FiChevronDown,
	FiChevronRight,
	FiExternalLink,
	FiRefreshCw,
	FiServer,
	FiWifi,
	FiWifiOff,
	FiX,
} from 'react-icons/fi';

import { Menu, MenuButton, type MenuStore, useMenuStore, useStoreState } from '@ariakit/react';
import { Link } from 'react-router';

import {
	MCPAuthHealthState,
	type MCPPromptRef,
	type MCPPromptSelection,
	MCPRefType,
	type MCPResourceRef,
	type MCPResourceTemplateRef,
	type MCPResourceTemplateSelection,
	MCPServerStatus,
	type MCPToolCapability,
	MCPToolExposure,
} from '@/spec/mcp';

import { mcpAPI } from '@/apis/baseapi';

import { ActionTriggerChipContent, actionTriggerChipSurfaceClasses } from '@/components/action_trigger_chip';
import { HoverTip } from '@/components/ariakit_hover_tip';

import {
	type MCPComposerServerOption,
	mcpPromptKey,
	mcpResourceKey,
	mcpResourceTemplateKey,
	mcpServerKey,
	mcpToolKey,
	normalizeMCPArgumentDefinitions,
	type UseComposerMCPResult,
} from '@/chats/composer/mcp/mcp_composer_types';
import { optionKey } from '@/chats/composer/mcp/use_composer_mcp';
import {
	getEffectiveMCPServerStatus,
	getMCPAuthHealthBadgeClass,
	getMCPAuthHealthLabel,
	getMCPStatusBadgeClass,
	getMCPStatusLabel,
	getMCPTransportLabel,
	isMCPToolVisibleToModel,
} from '@/mcpservers/lib/mcp_server_utils';

function stop(e: MouseEvent) {
	e.preventDefault();
	e.stopPropagation();
}

function isEnabledMCPOption(option: MCPComposerServerOption) {
	return option.bundle.isEnabled && option.server.enabled;
}

function CheckboxRow({
	checked,
	disabled,
	label,
	title,
	onChange,
}: {
	checked: boolean;
	disabled?: boolean;
	label: ReactNode;
	title?: string;
	onChange: (next: boolean) => void;
}) {
	return (
		<label
			className={`hover:bg-base-200 flex cursor-pointer items-center gap-2 rounded-lg px-2 py-1 text-xs ${
				disabled ? 'opacity-60' : ''
			}`}
			title={title}
			onClick={e => {
				e.stopPropagation();
			}}
		>
			<input
				type="checkbox"
				className="checkbox checkbox-xs rounded-sm"
				checked={checked}
				disabled={disabled}
				onChange={e => {
					onChange(e.currentTarget.checked);
				}}
			/>
			<div className="min-w-0 flex-1">{label}</div>
		</label>
	);
}

const EMPTY_MCP_ARGUMENT_VALUES: Record<string, string> = {};

function BulkSelectionChips({
	onSelectAll,
	onUnselectAll,
	selectAllDisabled,
	unselectAllDisabled,
}: {
	onSelectAll: () => void;
	onUnselectAll: () => void;
	selectAllDisabled?: boolean;
	unselectAllDisabled?: boolean;
}) {
	return (
		<div className="flex shrink-0 items-center gap-1">
			<button
				type="button"
				className="btn btn-ghost btn-xs h-5 min-h-0 rounded-full px-2 py-0 text-[11px]"
				disabled={selectAllDisabled}
				onClick={e => {
					stop(e);
					onSelectAll();
				}}
			>
				Select all
			</button>
			<button
				type="button"
				className="btn btn-ghost btn-xs h-5 min-h-0 rounded-full px-2 py-0 text-[11px]"
				disabled={unselectAllDisabled}
				onClick={e => {
					stop(e);
					onUnselectAll();
				}}
			>
				Unselect all
			</button>
		</div>
	);
}

function DiscoverySection({
	title,
	presentCount,
	selectedCount,
	showBulkActions = true,
	selectAllDisabled,
	unselectAllDisabled,
	onSelectAll,
	onUnselectAll,
	children,
}: {
	title: string;
	presentCount: number;
	selectedCount: number;
	showBulkActions?: boolean;
	selectAllDisabled?: boolean;
	unselectAllDisabled?: boolean;
	onSelectAll: () => void;
	onUnselectAll: () => void;
	children: ReactNode;
}) {
	const [open, setOpen] = useState(false);

	return (
		<div className="border-base-300/70 rounded-lg border">
			<div className="flex w-full items-center gap-2 px-2 py-1.5 text-xs">
				<button
					type="button"
					className="hover:bg-base-200 -mx-1 flex min-w-0 flex-1 items-center gap-2 rounded-lg px-1 py-0.5 text-left"
					onClick={e => {
						e.stopPropagation();
						setOpen(value => !value);
					}}
				>
					{open ? <FiChevronDown size={13} /> : <FiChevronRight size={13} />}
					<span className="min-w-0 flex-1 font-semibold">{title}</span>
					<span className="badge badge-ghost badge-xs rounded-lg">{presentCount} present</span>
					<span className="badge badge-info badge-xs rounded-lg">{selectedCount} selected</span>
				</button>
				{showBulkActions ? (
					<BulkSelectionChips
						selectAllDisabled={selectAllDisabled}
						unselectAllDisabled={unselectAllDisabled}
						onSelectAll={onSelectAll}
						onUnselectAll={onUnselectAll}
					/>
				) : null}
			</div>

			{open ? <div className="border-base-300/70 border-t p-2">{children}</div> : null}
		</div>
	);
}

function MCPArgumentFields({
	bundleID,
	serverID,
	refType,
	name,
	item,
	disabled,
	onValueChange,
}: {
	bundleID: string;
	serverID: string;
	refType: MCPRefType;
	name: string;
	item: MCPPromptSelection | MCPResourceTemplateSelection;
	disabled?: boolean;
	onValueChange: (argumentName: string, value: string) => void;
}) {
	const args = normalizeMCPArgumentDefinitions(item.arguments);
	const [focusedArg, setFocusedArg] = useState<string | null>(null);
	const [completionsByArg, setCompletionsByArg] = useState<Record<string, string[]>>({});

	const values = item.argumentValues ?? EMPTY_MCP_ARGUMENT_VALUES;
	const focusedValue = focusedArg ? (values[focusedArg] ?? '') : '';

	useEffect(() => {
		if (!focusedArg) return;

		let cancelled = false;
		const timer = window.setTimeout(() => {
			void mcpAPI
				.completeMCPArgument(bundleID, serverID, refType, name, focusedArg, focusedValue, values)
				.then(result => {
					if (cancelled) return;
					setCompletionsByArg(prev => ({
						...prev,
						[focusedArg]: result.values ?? [],
					}));
				})
				.catch(() => {
					if (cancelled) return;
					setCompletionsByArg(prev => ({
						...prev,
						[focusedArg]: [],
					}));
				});
		}, 200);

		return () => {
			cancelled = true;
			window.clearTimeout(timer);
		};
	}, [bundleID, focusedArg, focusedValue, name, refType, serverID, values]);

	if (args.length === 0) return null;

	return (
		<div className="bg-base-200/70 mx-2 mb-1 rounded-lg p-2">
			<div className="text-base-content/70 mb-1 text-[10px] font-semibold uppercase">Arguments</div>
			<div className="space-y-2">
				{args.map(arg => {
					const value = values[arg.name] ?? '';
					const listID = `mcp-arg-${bundleID}-${serverID}-${refType}-${name}-${arg.name}`.replace(
						/[^A-Za-z0-9_-]/g,
						'_'
					);
					const missing = arg.required && value.trim().length === 0;

					return (
						<label key={arg.name} className="block text-xs">
							<div className="mb-1 flex items-center gap-2">
								<span className={missing ? 'text-warning' : ''}>
									{arg.name}
									{arg.required ? '*' : ''}
								</span>
								{arg.description ? (
									<span className="text-base-content/50 truncate" title={arg.description}>
										{arg.description}
									</span>
								) : null}
							</div>
							<input
								className={`input input-bordered input-xs w-full rounded-lg ${missing ? 'input-warning' : ''}`}
								value={value}
								list={listID}
								disabled={disabled}
								onFocus={() => {
									setFocusedArg(arg.name);
								}}
								onBlur={() => {
									setFocusedArg(null);
								}}
								onChange={event => {
									onValueChange(arg.name, event.currentTarget.value);
								}}
							/>
							<datalist id={listID}>
								{(completionsByArg[arg.name] ?? []).map(option => (
									<option key={option} value={option} />
								))}
							</datalist>
						</label>
					);
				})}
			</div>
		</div>
	);
}

function ServerDiscoverySection({
	option,
	state,
	isInputLocked,
}: {
	option: MCPComposerServerOption;
	state: UseComposerMCPResult;
	isInputLocked: boolean;
}) {
	const key = mcpServerKey(option.bundle.id, option.server.id);
	const selection = state.selectedByServerKey[key];
	if (!selection) return null;

	const selectedToolKeys = new Set(selection.selectedTools.map(mcpToolKey));
	const selectedResourceKeys = new Set(selection.selectedResources.map(mcpResourceKey));
	const selectedTemplateKeys = new Set(selection.selectedResourceTemplates.map(mcpResourceTemplateKey));
	const selectedPromptKeys = new Set(selection.selectedPrompts.map(mcpPromptKey));
	const selectedTemplateByKey = new Map(
		selection.selectedResourceTemplates.map(template => [mcpResourceTemplateKey(template), template] as const)
	);
	const selectedPromptByKey = new Map(selection.selectedPrompts.map(prompt => [mcpPromptKey(prompt), prompt] as const));
	const selectableToolCount = option.tools.filter(tool => tool.enabled && isMCPToolVisibleToModel(tool)).length;
	const selectedToolCount =
		selection.toolExposure === MCPToolExposure.MCPToolExposureNone
			? 0
			: selection.toolExposure === MCPToolExposure.MCPToolExposureAll
				? selectableToolCount
				: selectedToolKeys.size;
	const resourcePresentCount = option.resources.length + option.resourceTemplates.length;
	const selectedResourceCount = selectedResourceKeys.size + selectedTemplateKeys.size;
	const promptPresentCount = option.prompts.length;
	const selectedPromptCount = selectedPromptKeys.size;
	const discoveryText = option.discoveryLoading
		? 'Loading discovery…'
		: option.discoveryError
			? option.discoveryError
			: !option.discoveryLoaded
				? 'Discovery not loaded yet.'
				: '';

	return (
		<div className="bg-base-100 rounded-xl p-2">
			<div className="mb-2 flex flex-wrap items-center gap-2">
				<label className="flex items-end gap-2 text-xs">
					<input
						type="checkbox"
						className="checkbox checkbox-xs rounded-sm"
						checked={Boolean(selection.includeServerInstructions)}
						disabled={isInputLocked}
						onChange={e => {
							state.setIncludeServerInstructions(option.bundle.id, option.server.id, e.currentTarget.checked);
						}}
					/>
					<span>Include instructions</span>
				</label>
			</div>

			{discoveryText ? <div className="text-base-content/70 mt-2 text-xs">{discoveryText}</div> : null}

			<div className="mt-2 space-y-2">
				<DiscoverySection
					title="Tools"
					presentCount={option.tools.length}
					selectedCount={selectedToolCount}
					showBulkActions={selection.toolExposure === MCPToolExposure.MCPToolExposureSelected}
					selectAllDisabled={isInputLocked || selectableToolCount === 0}
					unselectAllDisabled={isInputLocked || selectedToolKeys.size === 0}
					onSelectAll={() => {
						option.tools.forEach(tool => {
							if (tool.enabled && isMCPToolVisibleToModel(tool)) {
								state.toggleTool(tool, true);
							}
						});
					}}
					onUnselectAll={() => {
						option.tools.forEach(tool => {
							state.toggleTool(tool, false);
						});
					}}
				>
					{selection.toolExposure === MCPToolExposure.MCPToolExposureSelected ? (
						option.tools.length === 0 ? (
							<div className="text-base-content/60 px-2 text-xs">No tools discovered.</div>
						) : (
							<div className="max-h-40 overflow-y-auto">
								{option.tools.map((tool: MCPToolCapability) => {
									const visibleToModel = isMCPToolVisibleToModel(tool);

									return (
										<CheckboxRow
											key={mcpToolKey(tool)}
											checked={selectedToolKeys.has(mcpToolKey(tool))}
											disabled={isInputLocked || !tool.enabled || !visibleToModel}
											title={
												!visibleToModel ? 'This tool is app-only and is not exposed to the model.' : tool.description
											}
											label={
												<div className="min-w-0">
													<div className="flex min-w-0 items-center gap-1">
														<span className="truncate">{tool.displayName || tool.toolName}</span>
														{!visibleToModel ? <span className="badge badge-ghost badge-xs">App only</span> : null}
													</div>
													<div className="text-base-content/60 truncate">{tool.toolName}</div>
												</div>
											}
											onChange={next => {
												state.toggleTool(tool, next);
											}}
										/>
									);
								})}
							</div>
						)
					) : (
						<div className="text-base-content/60 px-2 text-xs">
							{selection.toolExposure === MCPToolExposure.MCPToolExposureAll
								? 'All enabled model-visible tools will be exposed.'
								: 'No tools will be exposed.'}
						</div>
					)}
				</DiscoverySection>

				<DiscoverySection
					title="Resources"
					presentCount={resourcePresentCount}
					selectedCount={selectedResourceCount}
					selectAllDisabled={isInputLocked || resourcePresentCount === 0}
					unselectAllDisabled={isInputLocked || selectedResourceCount === 0}
					onSelectAll={() => {
						option.resources.forEach(resource => {
							state.toggleResource(resource, true);
						});
						option.resourceTemplates.forEach(template => {
							state.toggleResourceTemplate(template, true);
						});
					}}
					onUnselectAll={() => {
						option.resources.forEach(resource => {
							state.toggleResource(resource, false);
						});
						option.resourceTemplates.forEach(template => {
							state.toggleResourceTemplate(template, false);
						});
					}}
				>
					{resourcePresentCount === 0 ? (
						<div className="text-base-content/60 px-2 text-xs">No resources discovered.</div>
					) : (
						<div className="max-h-40 overflow-y-auto">
							{option.resources.map((resource: MCPResourceRef) => (
								<CheckboxRow
									key={mcpResourceKey(resource)}
									checked={selectedResourceKeys.has(mcpResourceKey(resource))}
									disabled={isInputLocked}
									title={resource.uri}
									label={
										<div className="min-w-0">
											<div className="truncate">{resource.displayName || resource.name || resource.uri}</div>
											<div className="text-base-content/60 truncate">{resource.uri}</div>
										</div>
									}
									onChange={next => {
										state.toggleResource(resource, next);
									}}
								/>
							))}

							{option.resourceTemplates.map((template: MCPResourceTemplateRef) => {
								const templateKey = mcpResourceTemplateKey(template);
								const selectedTemplate = selectedTemplateByKey.get(templateKey);

								return (
									<div key={templateKey}>
										<CheckboxRow
											checked={selectedTemplateKeys.has(templateKey)}
											disabled={isInputLocked}
											title={template.uriTemplate}
											label={
												<div className="min-w-0">
													<div className="truncate">
														{template.displayName || template.name || template.uriTemplate}
													</div>
													<div className="text-base-content/60 truncate">{template.uriTemplate}</div>
												</div>
											}
											onChange={next => {
												state.toggleResourceTemplate(template, next);
											}}
										/>
										{selectedTemplate ? (
											<MCPArgumentFields
												bundleID={template.bundleID}
												serverID={template.serverID}
												refType={MCPRefType.MCPRefTypeResource}
												name={template.uriTemplate}
												item={selectedTemplate}
												disabled={isInputLocked}
												onValueChange={(argumentName, value) => {
													state.setResourceTemplateArgumentValue(
														template.bundleID,
														template.serverID,
														template.uriTemplate,
														argumentName,
														value
													);
												}}
											/>
										) : null}
									</div>
								);
							})}
						</div>
					)}
				</DiscoverySection>

				<DiscoverySection
					title="Prompts"
					presentCount={promptPresentCount}
					selectedCount={selectedPromptCount}
					selectAllDisabled={isInputLocked || promptPresentCount === 0}
					unselectAllDisabled={isInputLocked || selectedPromptCount === 0}
					onSelectAll={() => {
						option.prompts.forEach(prompt => {
							state.togglePrompt(prompt, true);
						});
					}}
					onUnselectAll={() => {
						option.prompts.forEach(prompt => {
							state.togglePrompt(prompt, false);
						});
					}}
				>
					{promptPresentCount === 0 ? (
						<div className="text-base-content/60 px-2 text-xs">No prompts discovered.</div>
					) : (
						<div className="max-h-40 overflow-y-auto">
							{option.prompts.map((prompt: MCPPromptRef) => {
								const promptKey = mcpPromptKey(prompt);
								const selectedPrompt = selectedPromptByKey.get(promptKey);

								return (
									<div key={promptKey}>
										<CheckboxRow
											checked={selectedPromptKeys.has(promptKey)}
											disabled={isInputLocked}
											title={prompt.description}
											label={
												<div className="min-w-0">
													<div className="truncate">{prompt.displayName || prompt.promptName}</div>
													<div className="text-base-content/60 truncate">{prompt.promptName}</div>
												</div>
											}
											onChange={next => {
												state.togglePrompt(prompt, next);
											}}
										/>
										{selectedPrompt ? (
											<MCPArgumentFields
												bundleID={prompt.bundleID}
												serverID={prompt.serverID}
												refType={MCPRefType.MCPRefTypePrompt}
												name={prompt.promptName}
												item={selectedPrompt}
												disabled={isInputLocked}
												onValueChange={(argumentName, value) => {
													state.setPromptArgumentValue(
														prompt.bundleID,
														prompt.serverID,
														prompt.promptName,
														argumentName,
														value
													);
												}}
											/>
										) : null}
									</div>
								);
							})}
						</div>
					)}
				</DiscoverySection>
			</div>
		</div>
	);
}

function ServerRow({
	option,
	state,
	isInputLocked,
}: {
	option: MCPComposerServerOption;
	state: UseComposerMCPResult;
	isInputLocked: boolean;
}) {
	const key = mcpServerKey(option.bundle.id, option.server.id);
	const selected = Boolean(state.selectedByServerKey[key]);
	const status = getEffectiveMCPServerStatus(option.server.enabled, option.bundle.isEnabled, option.runtime);
	const isReady = status === MCPServerStatus.MCPServerStatusReady;
	const selectable = isEnabledMCPOption(option);

	const authActionable = Boolean(option.authHealth?.authorizationURL);
	const authPending = option.authHealth?.state === MCPAuthHealthState.MCPAuthHealthStateAuthorizationPending;

	return (
		<div className="border-base-300 mb-2 rounded-xl border p-2">
			<div className="flex items-start gap-2">
				<input
					type="checkbox"
					className="checkbox checkbox-xs mt-1 rounded-sm"
					checked={selected}
					disabled={isInputLocked || (!selected && !selectable)}
					title={!selectable ? 'Enable the MCP bundle and server before selecting it.' : undefined}
					onChange={e => {
						if (e.currentTarget.checked && !selectable) return;

						state.setServerSelected(option, e.currentTarget.checked);
						if (e.currentTarget.checked) {
							void state.ensureDiscoveryLoaded(option.bundle.id, option.server.id);
						}
					}}
					onClick={e => {
						e.stopPropagation();
					}}
				/>

				<div className="min-w-0 flex-1">
					<div className="flex min-w-0 items-center gap-2">
						<div className="truncate text-xs font-semibold">{option.server.displayName}</div>
						<span className={`badge badge-xs rounded-lg ${getMCPStatusBadgeClass(status)}`}>
							{getMCPStatusLabel(status)}
						</span>
						<span
							className={`badge badge-xs rounded-lg ${getMCPAuthHealthBadgeClass(option.authHealth?.state)}`}
							title={option.authHealth?.lastError || getMCPAuthHealthLabel(option.authHealth?.state)}
						>
							{getMCPAuthHealthLabel(option.authHealth?.state)}
						</span>
					</div>

					<div className="text-base-content/60 truncate text-xs">
						{option.bundle.slug}/{option.server.id} • {getMCPTransportLabel(option.server.transport)}
					</div>

					{option.runtime?.lastError ? <div className="text-error mt-1 text-xs">{option.runtime.lastError}</div> : null}
					{option.authHealth?.lastError ? (
						<div className="text-error mt-1 text-xs">{option.authHealth.lastError}</div>
					) : null}
				</div>

				<div className="flex shrink-0 items-center gap-1">
					{authActionable ? (
						<button
							type="button"
							className="btn btn-ghost btn-xs px-1"
							title="Open authorization URL"
							onClick={e => {
								stop(e);
								state.openAuthURL(option.authHealth?.authorizationURL ?? '');
							}}
							disabled={isInputLocked}
						>
							<FiExternalLink size={12} />
						</button>
					) : null}

					{authPending ? (
						<button
							type="button"
							className="btn btn-ghost btn-xs px-1"
							title="Cancel authorization"
							onClick={e => {
								stop(e);
								void state.cancelOAuth(option.bundle.id, option.server.id);
							}}
							disabled={isInputLocked}
						>
							<FiX size={12} />
						</button>
					) : null}

					<button
						type="button"
						className="btn btn-ghost btn-xs px-1"
						title={isReady ? 'Disconnect' : 'Connect'}
						onClick={e => {
							stop(e);
							if (isReady) {
								void state.disconnectServer(option.bundle.id, option.server.id).catch(console.error);
							} else {
								void state.connectServer(option.bundle.id, option.server.id).catch(console.error);
							}
						}}
						disabled={isInputLocked || !option.bundle.isEnabled || !option.server.enabled}
					>
						{isReady ? <FiWifiOff size={12} /> : <FiWifi size={12} />}
					</button>

					<button
						type="button"
						className="btn btn-ghost btn-xs px-1"
						title="Refresh"
						disabled={isInputLocked || !isReady}
						onClick={e => {
							stop(e);
							void state
								.refreshServer(option.bundle.id, option.server.id)
								.then(() => state.ensureDiscoveryLoaded(option.bundle.id, option.server.id))
								.catch(console.error);
						}}
					>
						<FiRefreshCw size={12} />
					</button>
				</div>
			</div>

			{selected ? <ServerDiscoverySection option={option} state={state} isInputLocked={isInputLocked} /> : null}
		</div>
	);
}

export function MCPBottomBarChip({
	store,
	shortcut,
	state,
	isInputLocked = false,
	appContextUpdateCount = 0,
	onClearAppContextUpdates,
}: {
	store: MenuStore;
	shortcut: string;
	state: UseComposerMCPResult;
	isInputLocked?: boolean;
	appContextUpdateCount?: number;
	onClearAppContextUpdates?: () => void;
}) {
	const internalMenu = useMenuStore({ placement: 'top', focusLoop: true });
	const menu = store ?? internalMenu;
	const open = useStoreState(menu, 'open');

	const enabledCount = state.selectedServerCount;
	const hasAppContextUpdates = appContextUpdateCount > 0;
	const hasBlockingArgs = state.argumentsBlocked;
	const visibleOptions = useMemo(() => state.options.filter(isEnabledMCPOption), [state.options]);
	const hiddenDisabledServerCount = state.options.length - visibleOptions.length;
	const hiddenDisabledServerLabel = hiddenDisabledServerCount === 1 ? 'server' : 'servers';
	const hiddenDisabledServerVerb = hiddenDisabledServerCount === 1 ? 'is' : 'are';

	const title = useMemo(() => {
		const lines = [shortcut ? `Attach MCP (${shortcut})` : 'Attach MCP'];
		lines.push('Choose MCP servers, tools, resources, and prompts for the next message.');
		lines.push(
			enabledCount > 0 ? `Status: Enabled (${enabledCount} server${enabledCount === 1 ? '' : 's'})` : 'Status: Disabled'
		);
		if (state.selectedToolCount > 0) lines.push(`Tools: ${state.selectedToolCount}`);
		if (state.selectedResourceCount > 0) lines.push(`Resources: ${state.selectedResourceCount}`);
		if (state.selectedPromptCount > 0) lines.push(`Prompts: ${state.selectedPromptCount}`);
		if (state.requiredArgumentMissingCount > 0) {
			lines.push(`Missing required args: ${state.requiredArgumentMissingCount}`);
		}
		if (hasAppContextUpdates) {
			lines.push(`Queued app context updates: ${appContextUpdateCount}`);
		}
		return lines.join('\n');
	}, [
		appContextUpdateCount,
		enabledCount,
		hasAppContextUpdates,
		shortcut,
		state.requiredArgumentMissingCount,
		state.selectedPromptCount,
		state.selectedResourceCount,
		state.selectedToolCount,
	]);

	useEffect(() => {
		if (isInputLocked) menu.hide();
	}, [isInputLocked, menu]);

	const chipToneClasses = hasBlockingArgs
		? 'border-warning/70 bg-warning/10 hover:bg-warning/15 animate-pulse'
		: enabledCount > 0 || hasAppContextUpdates
			? 'border-secondary/50 bg-secondary/10 hover:bg-secondary/15'
			: open
				? 'border-base-300 bg-base-300/60'
				: 'border-transparent';

	return (
		<div className="relative shrink-0" data-bottom-bar-mcp>
			<HoverTip content={title} placement="top" wrapperElement="div" wrapperClassName="inline-flex max-w-full">
				<div
					className={`${actionTriggerChipSurfaceClasses} border ${chipToneClasses} ${isInputLocked ? 'opacity-60' : ''}`}
				>
					<MenuButton
						store={menu}
						className="btn btn-xs text-neutral-custom h-auto min-h-0 flex-1 gap-0 border-none bg-transparent p-0 text-left font-normal shadow-none hover:bg-transparent"
						aria-label={shortcut ? `Attach MCP (${shortcut})` : 'Attach MCP'}
						disabled={isInputLocked}
					>
						<ActionTriggerChipContent
							icon={<FiServer size={14} />}
							label="MCP"
							count={
								enabledCount > 0 ? (
									<span className="badge badge-success badge-xs bg-success/30">{enabledCount}</span>
								) : undefined
							}
							suffix={
								hasBlockingArgs ? (
									<span className="badge badge-warning badge-xs">Args</span>
								) : hasAppContextUpdates ? (
									<span className="badge badge-info badge-xs">App {appContextUpdateCount}</span>
								) : enabledCount > 0 ? (
									<FiCheck size={14} className="shrink-0" />
								) : undefined
							}
							open={open}
							labelClassName="max-w-20 truncate text-xs font-normal"
						/>
					</MenuButton>

					{enabledCount > 0 || hasAppContextUpdates ? (
						<button
							type="button"
							className="btn btn-ghost btn-xs text-neutral-custom hover:bg-base-300/80 ml-1 h-auto min-h-0 shrink-0 px-1 py-0 shadow-none"
							onClick={event => {
								event.preventDefault();
								event.stopPropagation();
								state.clear();
								onClearAppContextUpdates?.();
								menu.hide();
							}}
							aria-label="Clear MCP context"
							title="Clear MCP context"
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
						className="rounded-box bg-base-100 text-base-content border-base-300 z-50 max-h-[75vh] w-2xl max-w-[90vw] overflow-y-auto border p-2 shadow-xl"
						autoFocusOnShow
					>
						<div className="mb-2 flex items-center justify-between gap-2">
							<div className="text-base-content/70 text-xs font-semibold">MCP servers</div>
							<button
								type="button"
								className="btn btn-ghost btn-xs rounded-lg"
								onClick={e => {
									stop(e);
									void state.refreshAll();
								}}
								disabled={isInputLocked || state.loading}
							>
								<FiRefreshCw size={12} />
								<span className="ml-1">Refresh</span>
							</button>
						</div>

						{hasAppContextUpdates ? (
							<div className="border-info/30 bg-info/10 mb-2 rounded-xl border p-2 text-xs">
								<div className="flex items-center justify-between gap-2">
									<div>
										MCP app model context queued for next send: {appContextUpdateCount} update
										{appContextUpdateCount === 1 ? '' : 's'}.
									</div>
									<button
										type="button"
										className="btn btn-ghost btn-xs rounded-lg"
										onClick={e => {
											stop(e);
											onClearAppContextUpdates?.();
										}}
										disabled={isInputLocked}
									>
										Clear
									</button>
								</div>
							</div>
						) : null}

						{state.loading ? (
							<div className="text-base-content/60 rounded-xl px-2 py-1 text-xs">Loading MCP servers…</div>
						) : state.error ? (
							<div className="text-error rounded-xl px-2 py-1 text-xs">{state.error}</div>
						) : state.options.length === 0 ? (
							<div className="text-base-content/60 rounded-xl px-2 py-1 text-xs">
								No MCP servers configured. Add servers from the MCP Servers management page.
							</div>
						) : (
							<>
								{visibleOptions.map(option => (
									<ServerRow key={optionKey(option)} option={option} state={state} isInputLocked={isInputLocked} />
								))}

								{hiddenDisabledServerCount > 0 ? (
									<div className="border-base-300 text-base-content/70 mt-2 border-t pt-2 text-xs">
										{hiddenDisabledServerCount} disabled MCP {hiddenDisabledServerLabel} {hiddenDisabledServerVerb}{' '}
										hidden.{' '}
										<Link
											to="/mcpservers"
											className="link link-info"
											onClick={event => {
												event.stopPropagation();
												menu.hide();
											}}
										>
											Go to MCP Servers
										</Link>{' '}
										to enable {hiddenDisabledServerCount === 1 ? 'it' : 'them'}.
									</div>
								) : null}
							</>
						)}
					</Menu>
				</div>
			</HoverTip>
		</div>
	);
}
