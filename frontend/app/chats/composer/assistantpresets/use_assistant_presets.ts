import { useCallback, useEffect, useState } from 'react';

import type { AssistantPreset } from '@/spec/assistantpreset';
import type {
	MCPBundle,
	MCPConversationContext,
	MCPPromptRef,
	MCPResourceRef,
	MCPResourceTemplateRef,
	MCPServerConfig,
	MCPToolCapability,
} from '@/spec/mcp';
import { MCPToolExposure } from '@/spec/mcp';
import type { AssistantModelPresetOption } from '@/spec/modelpreset';
import type { AssistantInstructionTemplateOption } from '@/spec/prompt';
import type { AssistantSkillOption } from '@/spec/skill';
import type { AssistantToolOption } from '@/spec/tool';
import { ToolImplType } from '@/spec/tool';

import { assistantPresetStoreAPI } from '@/apis/baseapi';
import {
	getAllMCPBundles,
	getAllMCPServerPrompts,
	getAllMCPServerResources,
	getAllMCPServerResourceTemplates,
	getAllMCPServers,
	getAllMCPServerTools,
} from '@/apis/list_helper';

import { loadAssistantPresetEditorCatalog } from '@/assistantpresets/lib/assistant_preset_catalog';
import {
	getAllAssistantPresetBundles,
	getAllAssistantPresetListItems,
} from '@/assistantpresets/lib/assistant_preset_store_list_utils';
import {
	buildModelPresetRefKey,
	buildSkillRefKey,
	buildToolRefKey,
} from '@/assistantpresets/lib/assistant_preset_utils';
import type { AssistantPresetOptionItem } from '@/chats/composer/assistantpresets/assistant_preset_runtime';
import { buildAssistantPresetIdentityKey } from '@/chats/composer/assistantpresets/assistant_preset_runtime';
import { isMCPToolModelSelectable } from '@/mcpservers/lib/mcp_server_utils';
import { buildPromptTemplateRefKey } from '@/prompts/lib/prompt_template_ref';

function getErrorMessage(error: unknown, fallback: string): string {
	if (error instanceof Error && error.message.trim().length > 0) {
		return error.message;
	}
	return fallback;
}

function getBundleDisplayName(bundle: { displayName?: string; slug: string }, fallbackID: string): string {
	return bundle.displayName || bundle.slug || fallbackID;
}

interface MCPPresetServerRef {
	bundleID: string;
	serverID: string;
}

interface AssistantPresetMCPAvailabilityLookups {
	bundlesByID: Map<string, MCPBundle>;
	serversByKey: Map<string, MCPServerConfig>;
	toolsByKey: Map<string, MCPToolCapability>;
	resourcesByKey: Map<string, MCPResourceRef>;
	resourceTemplatesByKey: Map<string, MCPResourceTemplateRef>;
	promptsByKey: Map<string, MCPPromptRef>;
	serverErrorsByBundleID: Map<string, string>;
	toolErrorsByServerKey: Map<string, string>;
	resourceErrorsByServerKey: Map<string, string>;
	resourceTemplateErrorsByServerKey: Map<string, string>;
	promptErrorsByServerKey: Map<string, string>;
}

function mcpServerKeyForAvailability(bundleID: string, serverID: string): string {
	return `${bundleID}::${serverID}`;
}

function mcpToolKeyForAvailability(bundleID: string, serverID: string, toolName: string): string {
	return `${bundleID}::${serverID}::${toolName}`;
}

function mcpResourceKeyForAvailability(bundleID: string, serverID: string, uri: string): string {
	return `${bundleID}::${serverID}::${uri}`;
}

function mcpResourceTemplateKeyForAvailability(bundleID: string, serverID: string, uriTemplate: string): string {
	return `${bundleID}::${serverID}::${uriTemplate}`;
}

function mcpPromptKeyForAvailability(bundleID: string, serverID: string, promptName: string): string {
	return `${bundleID}::${serverID}::${promptName}`;
}

function addMCPServerRef(
	refsByKey: Map<string, MCPPresetServerRef>,
	bundleID: string | undefined,
	serverID: string | undefined
) {
	const bid = bundleID?.trim();
	const sid = serverID?.trim();
	if (!bid || !sid) {
		return;
	}

	refsByKey.set(mcpServerKeyForAvailability(bid, sid), {
		bundleID: bid,
		serverID: sid,
	});
}

function collectMCPServerRefsFromContext(context?: MCPConversationContext): MCPPresetServerRef[] {
	const refsByKey = new Map<string, MCPPresetServerRef>();

	for (const server of context?.servers ?? []) {
		addMCPServerRef(refsByKey, server.bundleID, server.serverID);

		for (const tool of server.selectedTools ?? []) {
			addMCPServerRef(refsByKey, tool.bundleID || server.bundleID, tool.serverID || server.serverID);
		}
	}

	for (const resource of context?.resources ?? []) {
		addMCPServerRef(refsByKey, resource.bundleID, resource.serverID);
	}

	for (const template of context?.resourceTemplates ?? []) {
		addMCPServerRef(refsByKey, template.bundleID, template.serverID);
	}

	for (const prompt of context?.prompts ?? []) {
		addMCPServerRef(refsByKey, prompt.bundleID, prompt.serverID);
	}

	return [...refsByKey.values()];
}

function collectMCPServerRefsFromPresets(presets: AssistantPreset[]): MCPPresetServerRef[] {
	const refsByKey = new Map<string, MCPPresetServerRef>();

	for (const preset of presets) {
		for (const ref of collectMCPServerRefsFromContext(preset.startingMCPContext)) {
			refsByKey.set(mcpServerKeyForAvailability(ref.bundleID, ref.serverID), ref);
		}
	}

	return [...refsByKey.values()];
}

async function loadAssistantPresetMCPAvailabilityLookups(
	presets: AssistantPreset[]
): Promise<AssistantPresetMCPAvailabilityLookups | undefined> {
	const serverRefs = collectMCPServerRefsFromPresets(presets);
	if (serverRefs.length === 0) {
		return undefined;
	}

	const bundles = await getAllMCPBundles(undefined, true);
	const bundlesByID = new Map(bundles.map(bundle => [bundle.id, bundle] as const));

	const serversByKey = new Map<string, MCPServerConfig>();
	const toolsByKey = new Map<string, MCPToolCapability>();
	const resourcesByKey = new Map<string, MCPResourceRef>();
	const resourceTemplatesByKey = new Map<string, MCPResourceTemplateRef>();
	const promptsByKey = new Map<string, MCPPromptRef>();
	const serverErrorsByBundleID = new Map<string, string>();
	const toolErrorsByServerKey = new Map<string, string>();
	const resourceErrorsByServerKey = new Map<string, string>();
	const resourceTemplateErrorsByServerKey = new Map<string, string>();
	const promptErrorsByServerKey = new Map<string, string>();

	const referencedBundleIDs = [...new Set(serverRefs.map(ref => ref.bundleID))].filter(bundleID =>
		bundlesByID.has(bundleID)
	);

	await Promise.all(
		referencedBundleIDs.map(async bundleID => {
			try {
				const servers = await getAllMCPServers(bundleID, undefined, undefined, true);
				for (const server of servers) {
					serversByKey.set(mcpServerKeyForAvailability(bundleID, server.id), server);
				}
			} catch (error) {
				serverErrorsByBundleID.set(
					bundleID,
					getErrorMessage(error, `Could not verify MCP servers for bundle "${bundleID}".`)
				);
			}
		})
	);

	await Promise.all(
		serverRefs
			.filter(ref => serversByKey.has(mcpServerKeyForAvailability(ref.bundleID, ref.serverID)))
			.map(async ref => {
				const serverKey = mcpServerKeyForAvailability(ref.bundleID, ref.serverID);
				const [toolsResult, resourcesResult, resourceTemplatesResult, promptsResult] = await Promise.allSettled([
					getAllMCPServerTools(ref.bundleID, ref.serverID),
					getAllMCPServerResources(ref.bundleID, ref.serverID),
					getAllMCPServerResourceTemplates(ref.bundleID, ref.serverID),
					getAllMCPServerPrompts(ref.bundleID, ref.serverID),
				]);

				if (toolsResult.status === 'fulfilled') {
					for (const tool of toolsResult.value) {
						toolsByKey.set(mcpToolKeyForAvailability(tool.bundleID, tool.serverID, tool.toolName), tool);
					}
				} else {
					toolErrorsByServerKey.set(
						serverKey,
						getErrorMessage(toolsResult.reason, `Could not verify MCP tools for "${ref.serverID}".`)
					);
				}

				if (resourcesResult.status === 'fulfilled') {
					for (const resource of resourcesResult.value) {
						resourcesByKey.set(
							mcpResourceKeyForAvailability(resource.bundleID, resource.serverID, resource.uri),
							resource
						);
					}
				} else {
					resourceErrorsByServerKey.set(
						serverKey,
						getErrorMessage(resourcesResult.reason, `Could not verify MCP resources for "${ref.serverID}".`)
					);
				}

				if (resourceTemplatesResult.status === 'fulfilled') {
					for (const template of resourceTemplatesResult.value) {
						resourceTemplatesByKey.set(
							mcpResourceTemplateKeyForAvailability(template.bundleID, template.serverID, template.uriTemplate),
							template
						);
					}
				} else {
					resourceTemplateErrorsByServerKey.set(
						serverKey,
						getErrorMessage(
							resourceTemplatesResult.reason,
							`Could not verify MCP resource templates for "${ref.serverID}".`
						)
					);
				}

				if (promptsResult.status === 'fulfilled') {
					for (const prompt of promptsResult.value) {
						promptsByKey.set(mcpPromptKeyForAvailability(prompt.bundleID, prompt.serverID, prompt.promptName), prompt);
					}
				} else {
					promptErrorsByServerKey.set(
						serverKey,
						getErrorMessage(promptsResult.reason, `Could not verify MCP prompts for "${ref.serverID}".`)
					);
				}
			})
	);

	return {
		bundlesByID,
		serversByKey,
		toolsByKey,
		resourcesByKey,
		resourceTemplatesByKey,
		promptsByKey,
		serverErrorsByBundleID,
		toolErrorsByServerKey,
		resourceErrorsByServerKey,
		resourceTemplateErrorsByServerKey,
		promptErrorsByServerKey,
	};
}

function getAssistantPresetMCPAvailability(
	context: MCPConversationContext | undefined,
	lookups: AssistantPresetMCPAvailabilityLookups | undefined
): Pick<AssistantPresetOptionItem, 'isSelectable' | 'availabilityReason'> {
	const serverRefs = collectMCPServerRefsFromContext(context);
	if (serverRefs.length === 0) {
		return { isSelectable: true };
	}

	if (!lookups) {
		return {
			isSelectable: false,
			availabilityReason: 'This preset references MCP context, but MCP availability could not be verified.',
		};
	}

	for (const ref of serverRefs) {
		const bundle = lookups.bundlesByID.get(ref.bundleID);
		if (!bundle) {
			return {
				isSelectable: false,
				availabilityReason: `MCP bundle "${ref.bundleID}" no longer exists.`,
			};
		}
		if (!bundle.isEnabled) {
			return {
				isSelectable: false,
				availabilityReason: `MCP bundle "${bundle.slug || ref.bundleID}" is disabled.`,
			};
		}

		const serverKey = mcpServerKeyForAvailability(ref.bundleID, ref.serverID);
		const server = lookups.serversByKey.get(serverKey);
		if (!server) {
			return {
				isSelectable: false,
				availabilityReason:
					lookups.serverErrorsByBundleID.get(ref.bundleID) ?? `MCP server "${ref.serverID}" no longer exists.`,
			};
		}
		if (!server.enabled) {
			return {
				isSelectable: false,
				availabilityReason: `MCP server "${server.displayName || ref.serverID}" is disabled.`,
			};
		}
	}

	for (const server of context?.servers ?? []) {
		if (server.toolExposure !== MCPToolExposure.MCPToolExposureSelected) {
			continue;
		}

		for (const selection of server.selectedTools ?? []) {
			const bundleID = selection.bundleID || server.bundleID;
			const serverID = selection.serverID || server.serverID;
			const serverKey = mcpServerKeyForAvailability(bundleID, serverID);
			const key = mcpToolKeyForAvailability(bundleID, serverID, selection.toolName);
			const tool = lookups.toolsByKey.get(key);

			if (!tool) {
				return {
					isSelectable: false,
					availabilityReason:
						lookups.toolErrorsByServerKey.get(serverKey) ??
						`MCP tool "${selection.toolName}" no longer exists on server "${serverID}".`,
				};
			}

			if (!tool.enabled) {
				return {
					isSelectable: false,
					availabilityReason: `MCP tool "${tool.displayName || tool.toolName}" is disabled.`,
				};
			}

			if (!isMCPToolModelSelectable(tool)) {
				return {
					isSelectable: false,
					availabilityReason: `MCP tool "${tool.displayName || tool.toolName}" is not exposed to the model.`,
				};
			}
		}
	}

	for (const resource of context?.resources ?? []) {
		const serverKey = mcpServerKeyForAvailability(resource.bundleID, resource.serverID);
		const key = mcpResourceKeyForAvailability(resource.bundleID, resource.serverID, resource.uri);
		if (!lookups.resourcesByKey.has(key)) {
			return {
				isSelectable: false,
				availabilityReason:
					lookups.resourceErrorsByServerKey.get(serverKey) ??
					`MCP resource "${resource.uri}" no longer exists on server "${resource.serverID}".`,
			};
		}
	}

	for (const template of context?.resourceTemplates ?? []) {
		const serverKey = mcpServerKeyForAvailability(template.bundleID, template.serverID);
		const key = mcpResourceTemplateKeyForAvailability(template.bundleID, template.serverID, template.uriTemplate);
		if (!lookups.resourceTemplatesByKey.has(key)) {
			return {
				isSelectable: false,
				availabilityReason:
					lookups.resourceTemplateErrorsByServerKey.get(serverKey) ??
					`MCP resource template "${template.uriTemplate}" no longer exists on server "${template.serverID}".`,
			};
		}
	}

	for (const prompt of context?.prompts ?? []) {
		const serverKey = mcpServerKeyForAvailability(prompt.bundleID, prompt.serverID);
		const key = mcpPromptKeyForAvailability(prompt.bundleID, prompt.serverID, prompt.promptName);
		if (!lookups.promptsByKey.has(key)) {
			return {
				isSelectable: false,
				availabilityReason:
					lookups.promptErrorsByServerKey.get(serverKey) ??
					`MCP prompt "${prompt.promptName}" no longer exists on server "${prompt.serverID}".`,
			};
		}
	}

	return { isSelectable: true };
}

function getAssistantPresetAvailability(
	preset: AssistantPreset,
	lookups: {
		modelOptionsByKey: Map<string, AssistantModelPresetOption>;
		instructionOptionsByKey: Map<string, AssistantInstructionTemplateOption>;
		toolOptionsByKey: Map<string, AssistantToolOption>;
		skillOptionsByKey: Map<string, AssistantSkillOption>;
		mcpLookups?: AssistantPresetMCPAvailabilityLookups;
	}
): Pick<AssistantPresetOptionItem, 'isSelectable' | 'availabilityReason'> {
	let targetProviderSDKType: string | undefined;

	if (preset.startingModelPresetRef) {
		const key = buildModelPresetRefKey(preset.startingModelPresetRef);
		const option = lookups.modelOptionsByKey.get(key);

		if (!option) {
			return {
				isSelectable: false,
				availabilityReason: `Starting model preset "${key}" no longer exists.`,
			};
		}

		if (!option.isSelectable) {
			return {
				isSelectable: false,
				availabilityReason: option.availabilityReason ?? `Starting model preset "${key}" is not available.`,
			};
		}

		targetProviderSDKType = option.providerPreset.sdkType;
	}

	for (const ref of preset.startingInstructionTemplateRefs ?? []) {
		const key = buildPromptTemplateRefKey(ref);
		const option = lookups.instructionOptionsByKey.get(key);

		if (!option) {
			return {
				isSelectable: false,
				availabilityReason: `Instruction template "${key}" no longer exists.`,
			};
		}

		if (!option.isSelectable) {
			return {
				isSelectable: false,
				availabilityReason: option.availabilityReason ?? `Instruction template "${key}" is not available.`,
			};
		}
	}

	for (const selection of preset.startingToolSelections ?? []) {
		const key = buildToolRefKey(selection.toolRef);
		const option = lookups.toolOptionsByKey.get(key);

		if (!option) {
			return {
				isSelectable: false,
				availabilityReason: `Tool "${key}" no longer exists.`,
			};
		}

		if (!option.isSelectable) {
			return {
				isSelectable: false,
				availabilityReason: option.availabilityReason ?? `Tool "${key}" is not available.`,
			};
		}

		if (targetProviderSDKType && option.toolDefinition.type === ToolImplType.SDK) {
			const toolSDKType = option.toolDefinition.sdkImpl?.sdkType?.trim();
			if (!toolSDKType) {
				return {
					isSelectable: false,
					availabilityReason: `Tool "${option.toolDefinition.displayName || option.toolDefinition.slug}" is missing SDK metadata.`,
				};
			}

			if (toolSDKType !== targetProviderSDKType) {
				return {
					isSelectable: false,
					availabilityReason: `Tool "${option.toolDefinition.displayName || option.toolDefinition.slug}" requires "${toolSDKType}", but this preset’s starting model uses "${targetProviderSDKType}".`,
				};
			}
		}
	}

	for (const sel of preset.startingSkillSelections ?? []) {
		const key = buildSkillRefKey(sel.skillRef);
		const option = lookups.skillOptionsByKey.get(key);

		if (!option) {
			return {
				isSelectable: false,
				availabilityReason: `Skill "${key}" no longer exists.`,
			};
		}

		if (!option.isSelectable) {
			return {
				isSelectable: false,
				availabilityReason: option.availabilityReason ?? `Skill "${key}" is not available.`,
			};
		}
	}

	const mcpAvailability = getAssistantPresetMCPAvailability(preset.startingMCPContext, lookups.mcpLookups);
	if (!mcpAvailability.isSelectable) {
		return mcpAvailability;
	}

	return { isSelectable: true };
}

export function useAssistantPresets() {
	const [presetOptions, setPresetOptions] = useState<AssistantPresetOptionItem[]>([]);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);

	const refreshPresets = useCallback(async () => {
		setLoading(true);
		setError(null);

		try {
			const bundles = await getAllAssistantPresetBundles(undefined, false);
			const catalog = await loadAssistantPresetEditorCatalog();

			if (bundles.length === 0) {
				setPresetOptions([]);
				return;
			}
			const bundleByID = new Map(bundles.map(bundle => [bundle.id, bundle]));
			const modelOptionsByKey = new Map(catalog.modelPresetOptions.map(option => [option.key, option] as const));
			const instructionOptionsByKey = new Map(
				catalog.instructionTemplateOptions.map(option => [option.key, option] as const)
			);
			const toolOptionsByKey = new Map(catalog.toolOptions.map(option => [option.key, option] as const));
			const skillOptionsByKey = new Map(catalog.skillOptions.map(option => [option.key, option] as const));

			const listItems = await getAllAssistantPresetListItems(
				bundles.map(bundle => bundle.id),
				false
			);

			const fullResults = await Promise.allSettled(
				listItems.map(async item => {
					const preset = await assistantPresetStoreAPI.getAssistantPreset(
						item.bundleID,
						item.assistantPresetSlug,
						item.assistantPresetVersion
					);

					return {
						item,
						preset,
					};
				})
			);

			const loadedPresetResults = fullResults.flatMap(result => {
				if (result.status !== 'fulfilled') {
					console.error('Failed to load assistant preset:', result.reason);
					return [];
				}

				const { item, preset } = result.value;
				if (!preset) {
					return [];
				}

				return [{ item, preset }];
			});

			const mcpLookups = await loadAssistantPresetMCPAvailabilityLookups(
				loadedPresetResults.map(result => result.preset)
			);

			const nextOptions: AssistantPresetOptionItem[] = loadedPresetResults.flatMap(({ item, preset }) => {
				const bundle = bundleByID.get(item.bundleID);
				const bundleDisplayName = bundle
					? getBundleDisplayName(bundle, item.bundleID)
					: item.bundleSlug || item.bundleID;

				const displayName = preset.displayName || preset.slug;
				const label = `${displayName} — ${bundleDisplayName} (${preset.slug}@${preset.version})`;
				const availability = getAssistantPresetAvailability(preset, {
					modelOptionsByKey,
					instructionOptionsByKey,
					toolOptionsByKey,
					skillOptionsByKey,
					mcpLookups,
				});

				return [
					{
						key: buildAssistantPresetIdentityKey(item.bundleID, item.assistantPresetSlug, item.assistantPresetVersion),
						bundleID: item.bundleID,
						bundleSlug: item.bundleSlug,
						bundleDisplayName,
						displayName,
						description: preset.description,
						preset,
						label,
						isSelectable: availability.isSelectable,
						availabilityReason: availability.availabilityReason,
					},
				];
			});

			setPresetOptions(nextOptions);
		} catch (refreshError) {
			console.error('Failed to load assistant presets:', refreshError);
			setError(getErrorMessage(refreshError, 'Failed to load assistant presets.'));
			setPresetOptions([]);
		} finally {
			setLoading(false);
		}
	}, []);

	useEffect(() => {
		// oxlint-disable-next-line jsreact-hooks/set-state-in-effect
		void refreshPresets();
	}, [refreshPresets]);

	return {
		presetOptions,
		loading,
		error,
		refreshPresets,
	};
}
