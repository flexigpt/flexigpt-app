import { memo, useMemo, useState } from 'react';

import { FiPlus, FiTrash2 } from 'react-icons/fi';

import { MCPToolExposure } from '@/spec/mcp';

import { Dropdown } from '@/components/dropdown';

import type { UseComposerMCPResult } from '@/chats/composer/mcp/mcp_composer_types';
import {
	mcpPromptKey,
	mcpResourceKey,
	mcpResourceTemplateKey,
	mcpServerKey,
	mcpToolKey,
	normalizeMCPArgumentDefinitions,
} from '@/chats/composer/mcp/mcp_composer_types';
import { isMCPToolVisibleToModel } from '@/mcpservers/lib/mcp_server_utils';

const TOOL_EXPOSURE_OPTIONS = [
	MCPToolExposure.MCPToolExposureNone,
	MCPToolExposure.MCPToolExposureAll,
	MCPToolExposure.MCPToolExposureSelected,
];

const TOOL_EXPOSURE_DROPDOWN_ITEMS = {
	[MCPToolExposure.MCPToolExposureNone]: { isEnabled: true },
	[MCPToolExposure.MCPToolExposureAll]: { isEnabled: true },
	[MCPToolExposure.MCPToolExposureSelected]: { isEnabled: true },
};

function getToolExposureLabel(exposure: MCPToolExposure): string {
	switch (exposure) {
		case MCPToolExposure.MCPToolExposureNone:
			return 'No tools';
		case MCPToolExposure.MCPToolExposureAll:
			return 'All tools';
		case MCPToolExposure.MCPToolExposureSelected:
			return 'Selected tools';
		default:
			return String(exposure);
	}
}

interface MCPSelectionSectionProps {
	mcpState: UseComposerMCPResult;
}

export const MCPSelectionSection = memo(function MCPSelectionSection({ mcpState }: MCPSelectionSectionProps) {
	const [nextServerKey, setNextServerKey] = useState('');

	const availableServerOptions = useMemo(() => {
		return mcpState.options.filter(
			option => !mcpState.selectedByServerKey[mcpServerKey(option.bundle.id, option.server.id)]
		);
	}, [mcpState.options, mcpState.selectedByServerKey]);

	const dropdownItems = useMemo<Record<string, { isEnabled: boolean }>>(() => {
		const items: Record<string, { isEnabled: boolean }> = {};
		for (const option of availableServerOptions) {
			items[mcpServerKey(option.bundle.id, option.server.id)] = {
				isEnabled: option.bundle.isEnabled && option.server.enabled,
			};
		}
		return items;
	}, [availableServerOptions]);

	const orderedKeys = useMemo(
		() => availableServerOptions.map(option => mcpServerKey(option.bundle.id, option.server.id)),
		[availableServerOptions]
	);

	const effectiveNextServerKey =
		orderedKeys.length > 0 ? (orderedKeys.includes(nextServerKey) ? nextServerKey : orderedKeys[0]) : '';

	const handleAdd = () => {
		const option = availableServerOptions.find(o => mcpServerKey(o.bundle.id, o.server.id) === effectiveNextServerKey);
		if (option) {
			mcpState.setServerSelected(option, true);
			void mcpState.ensureDiscoveryLoaded(option.bundle.id, option.server.id);
			setNextServerKey('');
		}
	};

	const selectedOptions = useMemo(() => {
		return mcpState.options.filter(
			option => mcpState.selectedByServerKey[mcpServerKey(option.bundle.id, option.server.id)]
		);
	}, [mcpState.options, mcpState.selectedByServerKey]);

	return (
		<>
			{mcpState.loading && (
				<div className="alert mb-4 rounded-2xl text-sm">
					<span className="loading loading-spinner loading-sm" />
					<span>Loading MCP servers…</span>
				</div>
			)}
			{mcpState.error && (
				<div className="alert alert-error mb-4 rounded-2xl text-sm">
					<span>{mcpState.error}</span>
				</div>
			)}

			<div className="grid grid-cols-12 items-center gap-2">
				<div className="col-span-10">
					<Dropdown<string>
						dropdownItems={dropdownItems}
						orderedKeys={orderedKeys}
						selectedKey={effectiveNextServerKey}
						onChange={setNextServerKey}
						disabled={availableServerOptions.length === 0}
						placeholderLabel={
							availableServerOptions.length === 0 ? 'No eligible MCP servers available' : 'Select an MCP server'
						}
						title="Select an MCP server to add"
						getDisplayName={key => {
							const option = availableServerOptions.find(o => mcpServerKey(o.bundle.id, o.server.id) === key);
							return option ? `${option.server.displayName} (${option.bundle.slug}/${option.server.id})` : key;
						}}
					/>
				</div>
				<div className="col-span-2">
					<button
						type="button"
						className="btn btn-ghost w-full rounded-xl"
						onClick={handleAdd}
						disabled={!effectiveNextServerKey}
					>
						<FiPlus size={14} />
						<span className="ml-1">Add</span>
					</button>
				</div>
			</div>

			<div className="mt-4 space-y-3">
				{selectedOptions.map(option => {
					const serverKey = mcpServerKey(option.bundle.id, option.server.id);
					const selection = mcpState.selectedByServerKey[serverKey];
					if (!selection) {
						return null;
					}

					const selectedToolKeys = new Set(selection.selectedTools.map(mcpToolKey));
					const selectedResourceKeys = new Set(selection.selectedResources.map(mcpResourceKey));
					const selectedTemplateKeys = new Set(selection.selectedResourceTemplates.map(mcpResourceTemplateKey));
					const selectedPromptKeys = new Set(selection.selectedPrompts.map(mcpPromptKey));

					const discoveryText = option.discoveryLoading
						? 'Loading discovery...'
						: option.discoveryError
							? option.discoveryError
							: !option.discoveryLoaded
								? 'Discovery not loaded yet.'
								: '';

					return (
						<div key={serverKey} className="border-base-content/10 rounded-2xl border p-3">
							<div className="flex items-start justify-between gap-3">
								<div>
									<div className="font-medium">{option.server.displayName}</div>
									<div className="text-base-content/70 mt-1 text-xs">
										{option.bundle.slug}/{option.server.id}
									</div>
								</div>

								<button
									type="button"
									className="btn btn-sm btn-ghost rounded-xl"
									onClick={() => {
										mcpState.setServerSelected(option, false);
									}}
									title="Remove"
								>
									<FiTrash2 size={14} />
								</button>
							</div>

							<div className="mt-3 grid grid-cols-12 gap-4">
								<div className="col-span-12 md:col-span-4">
									<label className="label py-1">
										<span className="text-sm">Include Instructions</span>
									</label>
									<div>
										<input
											type="checkbox"
											className="toggle toggle-accent"
											checked={Boolean(selection.includeServerInstructions)}
											onChange={e => {
												mcpState.setIncludeServerInstructions(option.bundle.id, option.server.id, e.target.checked);
											}}
										/>
									</div>
								</div>
								<div className="col-span-12 md:col-span-4">
									<label className="label py-1">
										<span className="text-sm">Tool Exposure</span>
									</label>
									<Dropdown<MCPToolExposure>
										dropdownItems={TOOL_EXPOSURE_DROPDOWN_ITEMS}
										orderedKeys={TOOL_EXPOSURE_OPTIONS}
										selectedKey={selection.toolExposure}
										onChange={value => {
											mcpState.setToolExposure(option.bundle.id, option.server.id, value);
										}}
										getDisplayName={getToolExposureLabel}
										title="Tool Exposure"
									/>
								</div>
							</div>

							{discoveryText && <div className="text-base-content/70 mt-3 text-sm">{discoveryText}</div>}

							{!discoveryText && (
								<div className="mt-4 space-y-4">
									{selection.toolExposure === MCPToolExposure.MCPToolExposureSelected && option.tools.length > 0 && (
										<div>
											<div className="mb-2 text-sm font-semibold">Selected Tools</div>
											<div className="grid grid-cols-1 gap-2 md:grid-cols-2">
												{option.tools.filter(isMCPToolVisibleToModel).map(tool => (
													<label
														key={mcpToolKey(tool)}
														className="hover:bg-base-200 flex cursor-pointer items-center gap-2 rounded-lg p-1 text-sm"
													>
														<input
															type="checkbox"
															className="checkbox checkbox-sm rounded-sm"
															checked={selectedToolKeys.has(mcpToolKey(tool))}
															onChange={e => {
																mcpState.toggleTool(tool, e.target.checked);
															}}
														/>
														<span className="truncate">{tool.displayName || tool.toolName}</span>
													</label>
												))}
											</div>
										</div>
									)}

									{option.resources.length > 0 && (
										<div>
											<div className="mb-2 text-sm font-semibold">Resources</div>
											<div className="grid grid-cols-1 gap-2">
												{option.resources.map(res => (
													<label
														key={mcpResourceKey(res)}
														className="hover:bg-base-200 flex cursor-pointer items-center gap-2 rounded-lg p-1 text-sm"
													>
														<input
															type="checkbox"
															className="checkbox checkbox-sm rounded-sm"
															checked={selectedResourceKeys.has(mcpResourceKey(res))}
															onChange={e => {
																mcpState.toggleResource(res, e.target.checked);
															}}
														/>
														<span className="truncate">{res.displayName || res.name || res.uri}</span>
													</label>
												))}
											</div>
										</div>
									)}

									{option.resourceTemplates.length > 0 && (
										<div>
											<div className="mb-2 text-sm font-semibold">Resource Templates</div>
											<div className="space-y-2">
												{option.resourceTemplates.map(template => {
													const isSelected = selectedTemplateKeys.has(mcpResourceTemplateKey(template));
													const selTemplate = selection.selectedResourceTemplates.find(
														t => mcpResourceTemplateKey(t) === mcpResourceTemplateKey(template)
													);
													const args = normalizeMCPArgumentDefinitions(template.arguments);

													return (
														<div key={mcpResourceTemplateKey(template)} className="hover:bg-base-200 rounded-lg p-1">
															<label className="mb-1 flex cursor-pointer items-center gap-2 text-sm">
																<input
																	type="checkbox"
																	className="checkbox checkbox-sm rounded-sm"
																	checked={isSelected}
																	onChange={e => {
																		mcpState.toggleResourceTemplate(template, e.target.checked);
																	}}
																/>
																<span className="truncate">
																	{template.displayName || template.name || template.uriTemplate}
																</span>
															</label>
															{isSelected && args.length > 0 && (
																<div className="ml-6 grid grid-cols-1 gap-2">
																	{args.map(arg => {
																		const val = selTemplate?.argumentValues?.[arg.name] ?? '';
																		const missing = arg.required && !val.trim();
																		return (
																			<div key={arg.name}>
																				<label className="text-base-content/70 text-xs">
																					{arg.name} {arg.required ? '*' : ''}
																				</label>
																				<input
																					className={`input input-xs w-full rounded-lg ${
																						missing ? 'input-warning' : ''
																					}`}
																					value={val}
																					onChange={e => {
																						mcpState.setResourceTemplateArgumentValue(
																							template.bundleID,
																							template.serverID,
																							template.uriTemplate,
																							arg.name,
																							e.target.value
																						);
																					}}
																				/>
																			</div>
																		);
																	})}
																</div>
															)}
														</div>
													);
												})}
											</div>
										</div>
									)}

									{option.prompts.length > 0 && (
										<div>
											<div className="mb-2 text-sm font-semibold">Prompts</div>
											<div className="space-y-2">
												{option.prompts.map(prompt => {
													const isSelected = selectedPromptKeys.has(mcpPromptKey(prompt));
													const selPrompt = selection.selectedPrompts.find(
														p => mcpPromptKey(p) === mcpPromptKey(prompt)
													);
													const args = normalizeMCPArgumentDefinitions(prompt.arguments);

													return (
														<div key={mcpPromptKey(prompt)} className="hover:bg-base-200 rounded-lg p-1">
															<label className="mb-1 flex cursor-pointer items-center gap-2 text-sm">
																<input
																	type="checkbox"
																	className="checkbox checkbox-sm rounded-sm"
																	checked={isSelected}
																	onChange={e => {
																		mcpState.togglePrompt(prompt, e.target.checked);
																	}}
																/>
																<span className="truncate">{prompt.displayName || prompt.promptName}</span>
															</label>
															{isSelected && args.length > 0 && (
																<div className="ml-6 grid grid-cols-1 gap-2">
																	{args.map(arg => {
																		const val = selPrompt?.argumentValues?.[arg.name] ?? '';
																		const missing = arg.required && !val.trim();
																		return (
																			<div key={arg.name}>
																				<label className="text-base-content/70 text-xs">
																					{arg.name} {arg.required ? '*' : ''}
																				</label>
																				<input
																					className={`input input-xs w-full rounded-lg ${
																						missing ? 'input-warning' : ''
																					}`}
																					value={val}
																					onChange={e => {
																						mcpState.setPromptArgumentValue(
																							prompt.bundleID,
																							prompt.serverID,
																							prompt.promptName,
																							arg.name,
																							e.target.value
																						);
																					}}
																				/>
																			</div>
																		);
																	})}
																</div>
															)}
														</div>
													);
												})}
											</div>
										</div>
									)}
								</div>
							)}
						</div>
					);
				})}

				{selectedOptions.length === 0 && <div className="text-base-content/70 text-sm">No MCP servers selected.</div>}
			</div>
		</>
	);
});
