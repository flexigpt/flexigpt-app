import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import {
	type MCPConversationContext,
	type MCPPromptRef,
	type MCPResourceRef,
	type MCPResourceTemplateRef,
	type MCPToolCapability,
	MCPToolExposure,
	type MCPToolSelection,
} from '@/spec/mcp';

import { omitManyKeys } from '@/lib/obj_utils';

import { backendAPI, mcpAPI } from '@/apis/baseapi';
import {
	getAllMCPBundles,
	getAllMCPServerPrompts,
	getAllMCPServerResources,
	getAllMCPServerResourceTemplates,
	getAllMCPServers,
	getAllMCPServerTools,
} from '@/apis/list_helper';

import {
	type MCPComposerServerOption,
	type MCPComposerServerSelection,
	mcpContextToSelectionMap,
	mcpPromptKey,
	mcpResourceKey,
	mcpResourceTemplateKey,
	mcpSelectionToContext,
	mcpServerKey,
	mcpToolKey,
	type UseComposerMCPResult,
} from '@/chats/composer/mcp/mcp_composer_types';

type MCPDiscoveryLoadResult = Pick<MCPComposerServerOption, 'tools' | 'resources' | 'resourceTemplates' | 'prompts'>;

function getErrorMessage(error: unknown, fallback: string): string {
	if (error instanceof Error && error.message.trim().length > 0) return error.message;
	return fallback;
}

function sleep(ms: number): Promise<void> {
	return new Promise(resolve => window.setTimeout(resolve, ms));
}

export function optionKey(option: MCPComposerServerOption): string {
	return mcpServerKey(option.bundle.id, option.server.id);
}

function toolToSelection(tool: MCPToolCapability): MCPToolSelection {
	return {
		bundleID: tool.bundleID,
		serverID: tool.serverID,
		toolName: tool.toolName,
		providerToolName: tool.providerToolName,
		choiceID: tool.choiceID,
		digest: tool.digest,
		approvalRule: tool.approvalRule,
		executionMode: tool.executionMode,
	};
}

function upsertByKey<T>(items: T[], keyFn: (item: T) => string, item: T): T[] {
	const key = keyFn(item);
	const exists = items.some(existing => keyFn(existing) === key);
	return exists ? items : [...items, item];
}

function removeByKey<T>(items: T[], keyFn: (item: T) => string, item: T): T[] {
	const key = keyFn(item);
	return items.filter(existing => keyFn(existing) !== key);
}

export function useComposerMCP(): UseComposerMCPResult {
	const [options, setOptions] = useState<MCPComposerServerOption[]>([]);
	const [selectedByServerKey, setSelectedByServerKey] = useState<Record<string, MCPComposerServerSelection>>({});
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | undefined>(undefined);

	const mountedRef = useRef(true);
	const optionsRef = useRef<MCPComposerServerOption[]>([]);
	const selectedByServerKeyRef = useRef<Record<string, MCPComposerServerSelection>>({});
	const discoveryPromisesRef = useRef(new Map<string, Promise<MCPDiscoveryLoadResult | undefined>>());

	useEffect(() => {
		return () => {
			mountedRef.current = false;
		};
	}, []);
	useEffect(() => {
		optionsRef.current = options;
	}, [options]);

	useEffect(() => {
		selectedByServerKeyRef.current = selectedByServerKey;
	}, [selectedByServerKey]);

	const patchOption = useCallback((bundleID: string, serverID: string, patch: Partial<MCPComposerServerOption>) => {
		setOptions(prev => {
			const next = prev.map(option =>
				option.bundle.id === bundleID && option.server.id === serverID
					? {
							...option,
							...patch,
						}
					: option
			);
			optionsRef.current = next;
			return next;
		});
	}, []);

	const refreshServer = useCallback(
		async (bundleID: string, serverID: string) => {
			const [runtime, authHealth] = await Promise.all([
				mcpAPI.getMCPServerStatus(bundleID, serverID).catch(() => undefined),
				mcpAPI.getMCPServerAuthHealth(bundleID, serverID).catch(() => undefined),
			]);

			if (!mountedRef.current) return;

			patchOption(bundleID, serverID, { runtime, authHealth });
		},
		[patchOption]
	);

	const refreshAll = useCallback(async () => {
		setLoading(true);
		setError(undefined);

		try {
			const bundles = await getAllMCPBundles(undefined, true);
			const nextOptions: MCPComposerServerOption[] = [];

			for (const bundle of bundles) {
				const servers = await getAllMCPServers(bundle.id, undefined, undefined, true).catch(() => []);

				for (const server of servers) {
					const [runtime, authHealth] = await Promise.all([
						mcpAPI.getMCPServerStatus(bundle.id, server.id).catch(() => undefined),
						mcpAPI.getMCPServerAuthHealth(bundle.id, server.id).catch(() => undefined),
					]);

					nextOptions.push({
						bundle,
						server,
						runtime,
						authHealth,
						tools: [],
						resources: [],
						resourceTemplates: [],
						prompts: [],
						discoveryLoaded: false,
						discoveryLoading: false,
					});
				}
			}

			if (!mountedRef.current) return;
			optionsRef.current = nextOptions;
			setOptions(nextOptions);
		} catch (err) {
			if (!mountedRef.current) return;
			setError(getErrorMessage(err, 'Failed to load MCP servers.'));
		} finally {
			if (mountedRef.current) setLoading(false);
		}
	}, []);

	useEffect(() => {
		// eslint-disable-next-line react-hooks/set-state-in-effect
		void refreshAll();
	}, [refreshAll]);

	const loadDiscoveryForServer = useCallback(
		async (bundleID: string, serverID: string, force = false): Promise<MCPDiscoveryLoadResult | undefined> => {
			const key = mcpServerKey(bundleID, serverID);
			const current = optionsRef.current.find(option => option.bundle.id === bundleID && option.server.id === serverID);
			if (!current) return undefined;

			if (!force && current.discoveryLoaded) {
				return {
					tools: current.tools,
					resources: current.resources,
					resourceTemplates: current.resourceTemplates,
					prompts: current.prompts,
				};
			}

			const existing = discoveryPromisesRef.current.get(key);
			if (existing) return existing;
			patchOption(bundleID, serverID, {
				discoveryLoading: true,
				discoveryError: undefined,
			});

			const promise = (async (): Promise<MCPDiscoveryLoadResult | undefined> => {
				const [tools, resources, resourceTemplates, prompts] = await Promise.all([
					getAllMCPServerTools(bundleID, serverID).catch(() => []),
					getAllMCPServerResources(bundleID, serverID).catch(() => []),
					getAllMCPServerResourceTemplates(bundleID, serverID).catch(() => []),
					getAllMCPServerPrompts(bundleID, serverID).catch(() => []),
				]);

				if (!mountedRef.current) return undefined;

				patchOption(bundleID, serverID, {
					tools,
					resources,
					resourceTemplates,
					prompts,
					discoveryLoaded: true,
					discoveryLoading: false,
				});
				setSelectedByServerKey(prev => {
					const currentSelection = prev[key];
					if (!currentSelection || currentSelection.toolExposure !== MCPToolExposure.MCPToolExposureAll) {
						return prev;
					}

					const next = {
						...prev,
						[key]: {
							...currentSelection,
							selectedTools: tools.filter(tool => tool.enabled).map(toolToSelection),
						},
					};
					selectedByServerKeyRef.current = next;
					return next;
				});

				return {
					tools,
					resources,
					resourceTemplates,
					prompts,
				};
			})().catch((err: unknown) => {
				if (!mountedRef.current) return undefined;

				patchOption(bundleID, serverID, {
					discoveryLoading: false,
					discoveryError: getErrorMessage(err, 'Failed to load MCP discovery.'),
				});
				throw err;
			});

			discoveryPromisesRef.current.set(key, promise);

			try {
				return await promise;
			} finally {
				discoveryPromisesRef.current.delete(key);
			}
		},
		[patchOption]
	);

	const ensureDiscoveryLoaded = useCallback(
		async (bundleID: string, serverID: string) => {
			await loadDiscoveryForServer(bundleID, serverID);
		},
		[loadDiscoveryForServer]
	);

	useEffect(() => {
		for (const selection of Object.values(selectedByServerKey)) {
			void ensureDiscoveryLoaded(selection.bundleID, selection.serverID);
		}
	}, [ensureDiscoveryLoaded, selectedByServerKey]);

	const connectServer = useCallback(
		async (bundleID: string, serverID: string) => {
			let settled = false;

			const connectPromise = mcpAPI.connectMCPServer(bundleID, serverID).finally(() => {
				settled = true;
			});

			while (!settled) {
				await Promise.race([connectPromise.catch(() => undefined), sleep(1000)]);
				await refreshServer(bundleID, serverID).catch(() => undefined);
			}

			const snapshot = await connectPromise;
			if (snapshot && mountedRef.current) {
				patchOption(bundleID, serverID, {
					runtime: snapshot,
					discoveryLoaded: false,
				});
			}

			await refreshServer(bundleID, serverID).catch(() => undefined);
			await ensureDiscoveryLoaded(bundleID, serverID).catch(() => undefined);
		},
		[ensureDiscoveryLoaded, patchOption, refreshServer]
	);

	const disconnectServer = useCallback(
		async (bundleID: string, serverID: string) => {
			await mcpAPI.disconnectMCPServer(bundleID, serverID);
			await refreshServer(bundleID, serverID);
		},
		[refreshServer]
	);

	const cancelOAuth = useCallback(
		async (bundleID: string, serverID: string) => {
			await mcpAPI.cancelPendingMCPOAuthAuthorization(bundleID, serverID);
			await refreshServer(bundleID, serverID);
		},
		[refreshServer]
	);

	const openAuthURL = useCallback((url: string) => {
		if (!url) return;
		backendAPI.openURL(url);
	}, []);

	const setServerSelected = useCallback((option: MCPComposerServerOption, selected: boolean) => {
		const key = optionKey(option);

		setSelectedByServerKey(prev => {
			if (!selected) {
				let next = { ...prev };
				next = omitManyKeys(next, [key]);
				selectedByServerKeyRef.current = next;
				return next;
			}

			if (prev[key]) return prev;

			const next = {
				...prev,
				[key]: {
					bundleID: option.bundle.id,
					serverID: option.server.id,
					snapshotDigest: option.runtime?.snapshotDigest,
					toolExposure: MCPToolExposure.MCPToolExposureAll,
					selectedTools: option.discoveryLoaded ? option.tools.filter(tool => tool.enabled).map(toolToSelection) : [],

					selectedResources: [],
					selectedResourceTemplates: [],
					selectedPrompts: [],
					includeServerInstructions: false,
				},
			};
			selectedByServerKeyRef.current = next;
			return next;
		});
	}, []);

	const setToolExposure = useCallback((bundleID: string, serverID: string, exposure: MCPToolExposure) => {
		const key = mcpServerKey(bundleID, serverID);
		setSelectedByServerKey(prev => {
			const current = prev[key];
			if (!current) return prev;
			const option = optionsRef.current.find(item => item.bundle.id === bundleID && item.server.id === serverID);
			const next = {
				...prev,
				[key]: {
					...current,
					toolExposure: exposure,
					selectedTools:
						exposure === MCPToolExposure.MCPToolExposureAll
							? (option?.tools ?? []).filter(tool => tool.enabled).map(toolToSelection)
							: exposure === MCPToolExposure.MCPToolExposureNone
								? []
								: current.selectedTools,
				},
			};
			selectedByServerKeyRef.current = next;
			return next;
		});
	}, []);

	const setIncludeServerInstructions = useCallback((bundleID: string, serverID: string, include: boolean) => {
		const key = mcpServerKey(bundleID, serverID);
		setSelectedByServerKey(prev => {
			const current = prev[key];
			if (!current) return prev;
			return {
				...prev,
				[key]: {
					...current,
					includeServerInstructions: include,
				},
			};
		});
	}, []);

	const toggleTool = useCallback((tool: MCPToolCapability, selected: boolean) => {
		const key = mcpServerKey(tool.bundleID, tool.serverID);
		const selection = toolToSelection(tool);

		setSelectedByServerKey(prev => {
			const current = prev[key];
			if (!current) return prev;

			return {
				...prev,
				[key]: {
					...current,
					selectedTools: selected
						? upsertByKey(current.selectedTools, mcpToolKey, selection)
						: removeByKey(current.selectedTools, mcpToolKey, selection),
				},
			};
		});
	}, []);

	const toggleResource = useCallback((resource: MCPResourceRef, selected: boolean) => {
		const key = mcpServerKey(resource.bundleID, resource.serverID);

		setSelectedByServerKey(prev => {
			const current = prev[key];
			if (!current) return prev;

			return {
				...prev,
				[key]: {
					...current,
					selectedResources: selected
						? upsertByKey(current.selectedResources, mcpResourceKey, resource)
						: removeByKey(current.selectedResources, mcpResourceKey, resource),
				},
			};
		});
	}, []);

	const toggleResourceTemplate = useCallback((template: MCPResourceTemplateRef, selected: boolean) => {
		const key = mcpServerKey(template.bundleID, template.serverID);

		setSelectedByServerKey(prev => {
			const current = prev[key];
			if (!current) return prev;

			return {
				...prev,
				[key]: {
					...current,
					selectedResourceTemplates: selected
						? upsertByKey(current.selectedResourceTemplates, mcpResourceTemplateKey, template)
						: removeByKey(current.selectedResourceTemplates, mcpResourceTemplateKey, template),
				},
			};
		});
	}, []);

	const togglePrompt = useCallback((prompt: MCPPromptRef, selected: boolean) => {
		const key = mcpServerKey(prompt.bundleID, prompt.serverID);

		setSelectedByServerKey(prev => {
			const current = prev[key];
			if (!current) return prev;

			return {
				...prev,
				[key]: {
					...current,
					selectedPrompts: selected
						? upsertByKey(current.selectedPrompts, mcpPromptKey, prompt)
						: removeByKey(current.selectedPrompts, mcpPromptKey, prompt),
				},
			};
		});
	}, []);

	const clear = useCallback(() => {
		selectedByServerKeyRef.current = {};

		setSelectedByServerKey({});
	}, []);

	const restoreContext = useCallback((context?: MCPConversationContext) => {
		const next = mcpContextToSelectionMap(context);
		selectedByServerKeyRef.current = next;
		setSelectedByServerKey(next);
	}, []);

	const prepareForSubmit = useCallback(async (): Promise<MCPConversationContext | undefined> => {
		const currentSelections = selectedByServerKeyRef.current;
		const nextSelections: Record<string, MCPComposerServerSelection> = {};

		for (const selection of Object.values(currentSelections)) {
			const key = mcpServerKey(selection.bundleID, selection.serverID);
			const option = optionsRef.current.find(
				item => item.bundle.id === selection.bundleID && item.server.id === selection.serverID
			);

			let selectedTools = selection.selectedTools;

			if (selection.toolExposure === MCPToolExposure.MCPToolExposureAll) {
				const discovery = await loadDiscoveryForServer(selection.bundleID, selection.serverID).catch(() => undefined);
				const tools = discovery?.tools ?? option?.tools ?? [];
				selectedTools = tools.filter(tool => tool.enabled).map(toolToSelection);
			}

			nextSelections[key] = {
				...selection,
				snapshotDigest: option?.runtime?.snapshotDigest ?? selection.snapshotDigest,
				selectedTools: selection.toolExposure === MCPToolExposure.MCPToolExposureNone ? [] : selectedTools,
			};
		}

		selectedByServerKeyRef.current = nextSelections;
		setSelectedByServerKey(nextSelections);

		return mcpSelectionToContext(nextSelections);
	}, [loadDiscoveryForServer]);

	const mcpContext = useMemo(() => mcpSelectionToContext(selectedByServerKey), [selectedByServerKey]);
	const selectedServerCount = Object.keys(selectedByServerKey).length;

	const selectedToolCount = Object.values(selectedByServerKey).reduce((sum, selection) => {
		if (selection.toolExposure === MCPToolExposure.MCPToolExposureNone) return sum;
		if (selection.selectedTools.length > 0) return sum + selection.selectedTools.length;
		return sum + 1;
	}, 0);

	const selectedResourceCount = Object.values(selectedByServerKey).reduce(
		(sum, selection) => sum + selection.selectedResources.length + selection.selectedResourceTemplates.length,
		0
	);

	const selectedPromptCount = Object.values(selectedByServerKey).reduce(
		(sum, selection) => sum + selection.selectedPrompts.length,
		0
	);

	return {
		options,
		loading,
		error,
		selectedByServerKey,
		mcpContext,
		selectedServerCount,
		selectedToolCount,
		selectedResourceCount,
		selectedPromptCount,
		refreshAll,
		refreshServer,
		ensureDiscoveryLoaded,
		prepareForSubmit,
		connectServer,
		disconnectServer,
		cancelOAuth,
		openAuthURL,
		setServerSelected,
		setToolExposure,
		setIncludeServerInstructions,
		toggleTool,
		toggleResource,
		toggleResourceTemplate,
		togglePrompt,
		clear,
		restoreContext,
	};
}
