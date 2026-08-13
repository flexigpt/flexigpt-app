import { useCallback, useEffect, useState } from 'react';

import type { ArtifactRef } from '@/spec/artifact';
import type { AssistantPreset } from '@/spec/assistantpreset';
import type {
	MCPConversationContext,
	MCPPromptRef,
	MCPResourceRef,
	MCPResourceTemplateRef,
	MCPToolCapability,
} from '@/spec/mcp_artifact';
import { MCPToolExposure } from '@/spec/mcp_artifact';
import type { AssistantModelPresetOption } from '@/spec/modelpreset';
import type { AssistantSkillOption } from '@/spec/skill';
import type { AssistantToolOption } from '@/spec/tool';
import { ToolImplType } from '@/spec/tool';

import { assistantPresetStoreAPI, mcpAPI } from '@/apis/baseapi';

import type { AssistantPresetCatalogLoadErrors } from '@/assistantpresets/lib/assistant_preset_catalog';
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
import { isServerOperational, loadMCPBundleViews, loadMCPServerViews } from '@/mcpservers/lib/mcp_management';
import { isMCPToolModelSelectable } from '@/mcpservers/lib/mcp_server_utils';
import {
	getSkillInstructionPromptEligibilityReason,
	getSkillPreloadEligibilityReason,
} from '@/skills/lib/skill_artifact_utils';
import { isSkillArtifactRef } from '@/skills/lib/skill_identity_utils';

function getErrorMessage(error: unknown, fallback: string): string {
	if (error instanceof Error && error.message.trim().length > 0) {
		return error.message;
	}
	return fallback;
}

function getBundleDisplayName(bundle: { displayName?: string; slug: string }, fallbackID: string): string {
	return bundle.displayName || bundle.slug || fallbackID;
}

interface AssistantPresetMCPAvailabilityLookups {
	serversByKey: Map<string, Awaited<ReturnType<typeof loadMCPServerViews>>[number]>;
	toolsByKey: Map<string, MCPToolCapability>;
	resourcesByKey: Map<string, MCPResourceRef>;
	resourceTemplatesByKey: Map<string, MCPResourceTemplateRef>;
	promptsByKey: Map<string, MCPPromptRef>;
	toolErrorsByServerKey: Map<string, string>;
	resourceErrorsByServerKey: Map<string, string>;
	resourceTemplateErrorsByServerKey: Map<string, string>;
	promptErrorsByServerKey: Map<string, string>;
}

function mcpServerKeyForAvailability(server: ArtifactRef): string {
	return `${server.rootID}::${server.artifactID}`;
}

function mcpToolKeyForAvailability(server: ArtifactRef, toolName: string): string {
	return `${mcpServerKeyForAvailability(server)}::${toolName}`;
}

function mcpResourceKeyForAvailability(server: ArtifactRef, uri: string): string {
	return `${mcpServerKeyForAvailability(server)}::${uri}`;
}

function mcpResourceTemplateKeyForAvailability(server: ArtifactRef, uriTemplate: string): string {
	return `${mcpServerKeyForAvailability(server)}::${uriTemplate}`;
}

function mcpPromptKeyForAvailability(server: ArtifactRef, promptName: string): string {
	return `${mcpServerKeyForAvailability(server)}::${promptName}`;
}

function addMCPServerRef(refsByKey: Map<string, ArtifactRef>, server: ArtifactRef | undefined) {
	if (!server?.rootID?.trim() || !server.artifactID?.trim()) {
		return;
	}

	refsByKey.set(mcpServerKeyForAvailability(server), server);
}

function collectMCPServerRefsFromContext(context?: MCPConversationContext): ArtifactRef[] {
	const refsByKey = new Map<string, ArtifactRef>();

	for (const server of context?.servers ?? []) {
		addMCPServerRef(refsByKey, server.server);

		for (const tool of server.selectedTools ?? []) {
			addMCPServerRef(refsByKey, tool.server ?? server.server);
		}
	}

	for (const resource of context?.resources ?? []) {
		addMCPServerRef(refsByKey, resource.server);
	}

	for (const template of context?.resourceTemplates ?? []) {
		addMCPServerRef(refsByKey, template.server);
	}

	for (const prompt of context?.prompts ?? []) {
		addMCPServerRef(refsByKey, prompt.server);
	}

	return [...refsByKey.values()];
}

function collectMCPServerRefsFromPresets(presets: AssistantPreset[]): ArtifactRef[] {
	const refsByKey = new Map<string, ArtifactRef>();

	for (const preset of presets) {
		for (const ref of collectMCPServerRefsFromContext(preset.startingMCPContext)) {
			refsByKey.set(mcpServerKeyForAvailability(ref), ref);
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

	const bundleViews = await loadMCPBundleViews();
	const serverGroups = await Promise.all(bundleViews.map(bundle => loadMCPServerViews(bundle)));
	const serversByKey = new Map(
		serverGroups.flat().map(server => [mcpServerKeyForAvailability(server.ref), server] as const)
	);
	const toolsByKey = new Map<string, MCPToolCapability>();
	const resourcesByKey = new Map<string, MCPResourceRef>();
	const resourceTemplatesByKey = new Map<string, MCPResourceTemplateRef>();
	const promptsByKey = new Map<string, MCPPromptRef>();
	const toolErrorsByServerKey = new Map<string, string>();
	const resourceErrorsByServerKey = new Map<string, string>();
	const resourceTemplateErrorsByServerKey = new Map<string, string>();
	const promptErrorsByServerKey = new Map<string, string>();

	await Promise.all(
		serverRefs
			.filter(server => serversByKey.has(mcpServerKeyForAvailability(server)))
			.map(async ref => {
				const serverKey = mcpServerKeyForAvailability(ref);
				const [toolsResult, resourcesResult, resourceTemplatesResult, promptsResult] = await Promise.allSettled([
					mcpAPI.listMCPServerTools(ref),
					mcpAPI.listMCPServerResources(ref),
					mcpAPI.listMCPServerResourceTemplates(ref),
					mcpAPI.listMCPServerPrompts(ref),
				]);

				if (toolsResult.status === 'fulfilled') {
					for (const tool of toolsResult.value) {
						toolsByKey.set(mcpToolKeyForAvailability(tool.server, tool.toolName), tool);
					}
				} else {
					toolErrorsByServerKey.set(
						serverKey,
						getErrorMessage(toolsResult.reason, `Could not verify MCP tools for "${ref.artifactID}".`)
					);
				}

				if (resourcesResult.status === 'fulfilled') {
					for (const resource of resourcesResult.value) {
						resourcesByKey.set(mcpResourceKeyForAvailability(resource.server, resource.uri), resource);
					}
				} else {
					resourceErrorsByServerKey.set(
						serverKey,
						getErrorMessage(resourcesResult.reason, `Could not verify MCP resources for "${ref.artifactID}".`)
					);
				}

				if (resourceTemplatesResult.status === 'fulfilled') {
					for (const template of resourceTemplatesResult.value) {
						resourceTemplatesByKey.set(
							mcpResourceTemplateKeyForAvailability(template.server, template.uriTemplate),
							template
						);
					}
				} else {
					resourceTemplateErrorsByServerKey.set(
						serverKey,
						getErrorMessage(
							resourceTemplatesResult.reason,
							`Could not verify MCP resource templates for "${ref.artifactID}".`
						)
					);
				}

				if (promptsResult.status === 'fulfilled') {
					for (const prompt of promptsResult.value) {
						promptsByKey.set(mcpPromptKeyForAvailability(prompt.server, prompt.promptName), prompt);
					}
				} else {
					promptErrorsByServerKey.set(
						serverKey,
						getErrorMessage(promptsResult.reason, `Could not verify MCP prompts for "${ref.artifactID}".`)
					);
				}
			})
	);

	return {
		serversByKey,
		toolsByKey,
		resourcesByKey,
		resourceTemplatesByKey,
		promptsByKey,
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
		const serverKey = mcpServerKeyForAvailability(ref);
		const server = lookups.serversByKey.get(serverKey);
		if (!server) {
			return {
				isSelectable: false,
				availabilityReason: `MCP server "${ref.artifactID}" no longer exists or is inaccessible.`,
			};
		}
		if (!server.artifact.enabled) {
			return {
				isSelectable: false,
				availabilityReason: `MCP server "${server.displayName}" is disabled.`,
			};
		}
		if (!server.artifact.enabled || !server.runtimeEnabled || !isServerOperational(server)) {
			return {
				isSelectable: false,
				availabilityReason: `MCP server "${server.displayName}" is disabled or unavailable.`,
			};
		}
	}

	for (const server of context?.servers ?? []) {
		if (server.toolExposure !== MCPToolExposure.Selected) {
			continue;
		}

		for (const selection of server.selectedTools ?? []) {
			const toolServer = selection.server ?? server.server;
			const serverKey = mcpServerKeyForAvailability(toolServer);
			const key = mcpToolKeyForAvailability(toolServer, selection.toolName);
			const tool = lookups.toolsByKey.get(key);

			if (!tool) {
				return {
					isSelectable: false,
					availabilityReason:
						lookups.toolErrorsByServerKey.get(serverKey) ??
						`MCP tool "${selection.toolName}" no longer exists on server "${toolServer.artifactID}".`,
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
		const serverKey = mcpServerKeyForAvailability(resource.server);
		const key = mcpResourceKeyForAvailability(resource.server, resource.uri);
		if (!lookups.resourcesByKey.has(key)) {
			return {
				isSelectable: false,
				availabilityReason:
					lookups.resourceErrorsByServerKey.get(serverKey) ??
					`MCP resource "${resource.uri}" no longer exists on server "${resource.server.artifactID}".`,
			};
		}
	}

	for (const template of context?.resourceTemplates ?? []) {
		const serverKey = mcpServerKeyForAvailability(template.server);
		const key = mcpResourceTemplateKeyForAvailability(template.server, template.uriTemplate);
		if (!lookups.resourceTemplatesByKey.has(key)) {
			return {
				isSelectable: false,
				availabilityReason:
					lookups.resourceTemplateErrorsByServerKey.get(serverKey) ??
					`MCP resource template "${template.uriTemplate}" no longer exists on server "${template.server.artifactID}".`,
			};
		}
	}

	for (const prompt of context?.prompts ?? []) {
		const serverKey = mcpServerKeyForAvailability(prompt.server);
		const key = mcpPromptKeyForAvailability(prompt.server, prompt.promptName);
		if (!lookups.promptsByKey.has(key)) {
			return {
				isSelectable: false,
				availabilityReason:
					lookups.promptErrorsByServerKey.get(serverKey) ??
					`MCP prompt "${prompt.promptName}" no longer exists on server "${prompt.server.artifactID}".`,
			};
		}
	}

	return { isSelectable: true };
}

function getAssistantPresetAvailability(
	preset: AssistantPreset,
	lookups: {
		modelOptionsByKey: Map<string, AssistantModelPresetOption>;
		toolOptionsByKey: Map<string, AssistantToolOption>;
		skillOptionsByKey: Map<string, AssistantSkillOption>;
		catalogLoadErrors?: AssistantPresetCatalogLoadErrors;
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
				availabilityReason: lookups.catalogLoadErrors?.models
					? `Could not verify starting model preset "${key}": ${lookups.catalogLoadErrors.models}`
					: `Starting model preset "${key}" no longer exists.`,
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

	for (const selection of preset.startingToolSelections ?? []) {
		const key = buildToolRefKey(selection.toolRef);
		const option = lookups.toolOptionsByKey.get(key);

		if (!option) {
			return {
				isSelectable: false,
				availabilityReason: lookups.catalogLoadErrors?.tools
					? `Could not verify tool "${key}": ${lookups.catalogLoadErrors.tools}`
					: `Tool "${key}" no longer exists.`,
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
		if (!isSkillArtifactRef(sel.artifact)) {
			return {
				isSelectable: false,
				availabilityReason:
					'Assistant preset contains an invalid Skill reference. ArtifactRef (rootID and artifactID) is required.',
			};
		}

		const key = buildSkillRefKey(sel.artifact);
		const option = lookups.skillOptionsByKey.get(key);

		if (!option) {
			return {
				isSelectable: false,
				availabilityReason: lookups.catalogLoadErrors?.skills
					? `Could not verify skill "${key}": ${lookups.catalogLoadErrors.skills}`
					: `Skill "${key}" no longer exists.`,
			};
		}

		if (!option.isSelectable) {
			return {
				isSelectable: false,
				availabilityReason: option.availabilityReason ?? `Skill "${key}" is not available.`,
			};
		}

		if (sel.useAsInstructions) {
			if (sel.preLoadAsActive) {
				return {
					isSelectable: false,
					availabilityReason: `Skill "${key}" cannot be both system instructions and active session preload.`,
				};
			}

			const reason = getSkillInstructionPromptEligibilityReason(option.skillDefinition);
			if (reason) {
				return { isSelectable: false, availabilityReason: reason };
			}
		} else if (sel.preLoadAsActive) {
			const reason = getSkillPreloadEligibilityReason(option.skillDefinition);
			if (reason) {
				return { isSelectable: false, availabilityReason: reason };
			}
		}
	}

	const mcpAvailability = getAssistantPresetMCPAvailability(preset.startingMCPContext, lookups.mcpLookups);
	if (!mcpAvailability.isSelectable) {
		return mcpAvailability;
	}

	return { isSelectable: true };
}

let assistantPresetOptionsCache: AssistantPresetOptionItem[] | undefined;
let assistantPresetOptionsPromise: Promise<AssistantPresetOptionItem[]> | undefined;

async function loadAssistantPresetOptions(force = false): Promise<AssistantPresetOptionItem[]> {
	if (assistantPresetOptionsPromise) {
		return assistantPresetOptionsPromise;
	}
	if (!force && assistantPresetOptionsCache !== undefined) {
		return assistantPresetOptionsCache;
	}

	const request = (async () => {
		const bundles = await getAllAssistantPresetBundles(undefined, false);
		const catalog = await loadAssistantPresetEditorCatalog({ force });

		if (bundles.length === 0) {
			return [];
		}

		const bundleByID = new Map(bundles.map(bundle => [bundle.id, bundle]));
		const modelOptionsByKey = new Map(catalog.modelPresetOptions.map(option => [option.key, option] as const));
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

		let mcpLookups: AssistantPresetMCPAvailabilityLookups | undefined;
		try {
			mcpLookups = await loadAssistantPresetMCPAvailabilityLookups(loadedPresetResults.map(result => result.preset));
		} catch (error: unknown) {
			console.error('Failed to load MCP availability lookups for assistant presets:', error);
		}
		return loadedPresetResults.flatMap(({ item, preset }): AssistantPresetOptionItem[] => {
			const bundle = bundleByID.get(item.bundleID);
			const bundleDisplayName = bundle ? getBundleDisplayName(bundle, item.bundleID) : item.bundleSlug || item.bundleID;

			const displayName = preset.displayName || preset.slug;
			const label = `${displayName} — ${bundleDisplayName} (${preset.slug}@${preset.version})`;
			const availability = getAssistantPresetAvailability(preset, {
				modelOptionsByKey,
				toolOptionsByKey,
				skillOptionsByKey,
				catalogLoadErrors: catalog.loadErrors,
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
	})();

	assistantPresetOptionsPromise = request;

	try {
		const result = await request;
		assistantPresetOptionsCache = result;
		return result;
	} finally {
		if (assistantPresetOptionsPromise === request) {
			assistantPresetOptionsPromise = undefined;
		}
	}
}

export function useAssistantPresets() {
	const [presetOptions, setPresetOptions] = useState<AssistantPresetOptionItem[]>(
		() => assistantPresetOptionsCache ?? []
	);
	const [loading, setLoading] = useState(() => assistantPresetOptionsCache === undefined);
	const [error, setError] = useState<string | null>(null);

	const refreshPresets = useCallback(async () => {
		setLoading(true);
		setError(null);

		try {
			setPresetOptions(await loadAssistantPresetOptions(true));
		} catch (refreshError) {
			console.error('Failed to load assistant presets:', refreshError);
			setError(getErrorMessage(refreshError, 'Failed to load assistant presets.'));
			setPresetOptions([]);
		} finally {
			setLoading(false);
		}
	}, []);

	useEffect(() => {
		let cancelled = false;

		void loadAssistantPresetOptions()
			.then(options => {
				if (!cancelled) {
					setPresetOptions(options);
				}
			})
			.catch((loadError: unknown) => {
				if (cancelled) {
					return;
				}

				console.error('Failed to load assistant presets:', loadError);
				setError(getErrorMessage(loadError, 'Failed to load assistant presets.'));
				setPresetOptions([]);
			})
			.finally(() => {
				if (!cancelled) {
					setLoading(false);
				}
			});

		return () => {
			cancelled = true;
		};
	}, []);

	return {
		presetOptions,
		loading,
		error,
		refreshPresets,
	};
}
