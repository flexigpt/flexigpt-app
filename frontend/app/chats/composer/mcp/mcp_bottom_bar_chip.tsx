import { type MouseEvent, useEffect, useMemo, useState } from 'react';

import { FiCheck, FiExternalLink, FiRefreshCw, FiServer, FiWifi, FiWifiOff, FiX } from 'react-icons/fi';

import { Menu, MenuButton, useMenuStore, useStoreState } from '@ariakit/react';

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
import { Dropdown, type DropdownItem } from '@/components/dropdown';

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
} from '@/mcpservers/lib/mcp_server_utils';

function stop(e: MouseEvent) {
	e.preventDefault();
	e.stopPropagation();
}

function SectionTitle({ children }: { children: React.ReactNode }) {
	return <div className="text-base-content/70 mt-2 mb-1 text-xs font-semibold">{children}</div>;
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
	label: React.ReactNode;
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

const TOOL_EXPOSURE_DROPDOWN_ITEMS: Record<MCPToolExposure, DropdownItem> = {
	[MCPToolExposure.MCPToolExposureNone]: { isEnabled: true },
	[MCPToolExposure.MCPToolExposureAll]: { isEnabled: true },
	[MCPToolExposure.MCPToolExposureSelected]: { isEnabled: true },
};

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
	const discoveryText = option.discoveryLoading
		? 'Loading discovery…'
		: option.discoveryError
			? option.discoveryError
			: !option.discoveryLoaded
				? 'Discovery not loaded yet.'
				: '';

	return (
		<div className="bg-base-100 mt-2 rounded-xl p-2">
			<div className="grid grid-cols-2 gap-2">
				<label className="text-xs">
					<div className="text-base-content/70 mb-1">Tool exposure</div>
					<Dropdown
						dropdownItems={TOOL_EXPOSURE_DROPDOWN_ITEMS}
						selectedKey={selection.toolExposure}
						onChange={next => {
							state.setToolExposure(option.bundle.id, option.server.id, next);
						}}
						getDisplayName={exposure => {
							switch (exposure) {
								case MCPToolExposure.MCPToolExposureNone:
									return 'No tools';
								case MCPToolExposure.MCPToolExposureAll:
									return 'All tools';
								case MCPToolExposure.MCPToolExposureSelected:
									return 'Selected tools';
								default:
									return exposure;
							}
						}}
						title="Tool exposure"
						inlineMenu={true}
					/>
				</label>

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

			{selection.toolExposure === MCPToolExposure.MCPToolExposureSelected && (
				<>
					<SectionTitle>Tools</SectionTitle>
					{option.tools.length === 0 ? (
						<div className="text-base-content/60 px-2 text-xs">No tools discovered.</div>
					) : (
						<div className="max-h-40 overflow-y-auto">
							{option.tools.map((tool: MCPToolCapability) => (
								<CheckboxRow
									key={mcpToolKey(tool)}
									checked={selectedToolKeys.has(mcpToolKey(tool))}
									disabled={isInputLocked || !tool.enabled}
									title={tool.description}
									label={
										<div className="min-w-0">
											<div className="truncate">{tool.displayName || tool.toolName}</div>
											<div className="text-base-content/60 truncate">{tool.toolName}</div>
										</div>
									}
									onChange={next => {
										state.toggleTool(tool, next);
									}}
								/>
							))}
						</div>
					)}
				</>
			)}

			<SectionTitle>Resources</SectionTitle>
			{option.resources.length === 0 && option.resourceTemplates.length === 0 ? (
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
											<div className="truncate">{template.displayName || template.name || template.uriTemplate}</div>
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

			<SectionTitle>Prompts</SectionTitle>
			{option.prompts.length === 0 ? (
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
	const authPending =
		option.authHealth?.state === MCPAuthHealthState.MCPAuthHealthStateAuthorizationPending &&
		Boolean(option.authHealth.authorizationURL);

	return (
		<div className="border-base-300 mb-2 rounded-xl border p-2">
			<div className="flex items-start gap-2">
				<input
					type="checkbox"
					className="checkbox checkbox-xs mt-1 rounded-sm"
					checked={selected}
					disabled={isInputLocked}
					onChange={e => {
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
						<span className={`badge badge-xs rounded-lg ${getMCPAuthHealthBadgeClass(option.authHealth?.state)}`}>
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
					{authPending ? (
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
								void state.disconnectServer(option.bundle.id, option.server.id);
							} else {
								void state.connectServer(option.bundle.id, option.server.id);
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
						onClick={e => {
							stop(e);
							void state.refreshServer(option.bundle.id, option.server.id);
							void state.ensureDiscoveryLoaded(option.bundle.id, option.server.id);
						}}
						disabled={isInputLocked}
					>
						<FiRefreshCw size={12} />
					</button>
				</div>
			</div>

			{selected ? <ServerDiscoverySection option={option} state={state} isInputLocked={isInputLocked} /> : null}
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

	// eslint-disable-next-line react-hooks/exhaustive-deps
	const values = item.argumentValues ?? {};

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
		<div className="bg-base-200/70 mx-2 mb-1 rounded-lg px-2 py-2">
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

export function MCPBottomBarChip({
	state,
	isInputLocked = false,
}: {
	state: UseComposerMCPResult;
	isInputLocked?: boolean;
}) {
	const menu = useMenuStore({ placement: 'top', focusLoop: true });
	const open = useStoreState(menu, 'open');

	const enabledCount = state.selectedServerCount;
	const title = useMemo(() => {
		const lines = ['MCP'];
		lines.push(
			enabledCount > 0 ? `Status: Enabled (${enabledCount} server${enabledCount === 1 ? '' : 's'})` : 'Status: Disabled'
		);
		if (state.selectedToolCount > 0) lines.push(`Tools: ${state.selectedToolCount}`);
		if (state.selectedResourceCount > 0) lines.push(`Resources: ${state.selectedResourceCount}`);
		if (state.selectedPromptCount > 0) lines.push(`Prompts: ${state.selectedPromptCount}`);
		if (state.requiredArgumentMissingCount > 0) {
			lines.push(`Missing required args: ${state.requiredArgumentMissingCount}`);
		}
		return lines.join('\n');
	}, [
		enabledCount,
		state.requiredArgumentMissingCount,
		state.selectedPromptCount,
		state.selectedResourceCount,
		state.selectedToolCount,
	]);

	return (
		<HoverTip content={title} placement="top" wrapperElement="div" wrapperClassName="inline-flex max-w-full">
			<div
				className={`${actionTriggerChipSurfaceClasses} border ${
					enabledCount > 0 ? 'border-secondary/50 bg-secondary/10 hover:bg-secondary/15' : 'border-transparent'
				} ${isInputLocked ? 'opacity-60' : ''}`}
				data-bottom-bar-mcp
			>
				<MenuButton
					store={menu}
					className="btn btn-xs text-neutral-custom bg-base-200/70 hover:bg-base-300/80 h-auto min-h-0 flex-1 gap-0 px-0 py-0 text-left font-normal shadow-none"
					aria-label="Choose MCP servers"
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
							state.argumentsBlocked ? (
								<span className="badge badge-warning badge-xs">Args</span>
							) : enabledCount > 0 ? (
								<FiCheck size={14} className="shrink-0" />
							) : undefined
						}
						open={open}
					/>
				</MenuButton>

				{enabledCount > 0 ? (
					<button
						type="button"
						className="btn btn-xs text-neutral-custom bg-base-200/70 hover:bg-base-300/80 h-auto min-h-0 shrink-0 px-1 py-0 shadow-none"
						onClick={event => {
							event.preventDefault();
							event.stopPropagation();
							state.clear();
							menu.hide();
						}}
						aria-label="Clear MCP context"
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

					{state.loading ? (
						<div className="text-base-content/60 rounded-xl px-2 py-1 text-xs">Loading MCP servers…</div>
					) : state.error ? (
						<div className="text-error rounded-xl px-2 py-1 text-xs">{state.error}</div>
					) : state.options.length === 0 ? (
						<div className="text-base-content/60 rounded-xl px-2 py-1 text-xs">
							No MCP servers configured. Add servers from the MCP Servers management page.
						</div>
					) : (
						state.options.map(option => (
							<ServerRow key={optionKey(option)} option={option} state={state} isInputLocked={isInputLocked} />
						))
					)}
				</Menu>
			</div>
		</HoverTip>
	);
}
