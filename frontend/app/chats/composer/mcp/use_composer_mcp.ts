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

	useEffect(() => {
		return () => {
			mountedRef.current = false;
		};
	}, []);

	const patchOption = useCallback((bundleID: string, serverID: string, patch: Partial<MCPComposerServerOption>) => {
		setOptions(prev =>
			prev.map(option =>
				option.bundle.id === bundleID && option.server.id === serverID
					? {
							...option,
							...patch,
						}
					: option
			)
		);
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

	const ensureDiscoveryLoaded = useCallback(
		async (bundleID: string, serverID: string) => {
			const current = options.find(option => option.bundle.id === bundleID && option.server.id === serverID);
			if (!current || current.discoveryLoaded || current.discoveryLoading) return;

			patchOption(bundleID, serverID, {
				discoveryLoading: true,
				discoveryError: undefined,
			});

			try {
				const [tools, resources, resourceTemplates, prompts] = await Promise.all([
					getAllMCPServerTools(bundleID, serverID).catch(() => []),
					getAllMCPServerResources(bundleID, serverID).catch(() => []),
					getAllMCPServerResourceTemplates(bundleID, serverID).catch(() => []),
					getAllMCPServerPrompts(bundleID, serverID).catch(() => []),
				]);

				if (!mountedRef.current) return;

				patchOption(bundleID, serverID, {
					tools,
					resources,
					resourceTemplates,
					prompts,
					discoveryLoaded: true,
					discoveryLoading: false,
				});
			} catch (err) {
				if (!mountedRef.current) return;

				patchOption(bundleID, serverID, {
					discoveryLoading: false,
					discoveryError: getErrorMessage(err, 'Failed to load MCP discovery.'),
				});
			}
		},
		[options, patchOption]
	);

	useEffect(() => {
		for (const selection of Object.values(selectedByServerKey)) {
			// eslint-disable-next-line react-hooks/set-state-in-effect
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
				return next;
			}

			if (prev[key]) return prev;

			return {
				...prev,
				[key]: {
					bundleID: option.bundle.id,
					serverID: option.server.id,
					snapshotDigest: option.runtime?.snapshotDigest,
					toolExposure: MCPToolExposure.MCPToolExposureAll,
					selectedTools: [],
					selectedResources: [],
					selectedResourceTemplates: [],
					selectedPrompts: [],
					includeServerInstructions: false,
				},
			};
		});
	}, []);

	const setToolExposure = useCallback((bundleID: string, serverID: string, exposure: MCPToolExposure) => {
		const key = mcpServerKey(bundleID, serverID);
		setSelectedByServerKey(prev => {
			const current = prev[key];
			if (!current) return prev;
			return {
				...prev,
				[key]: {
					...current,
					toolExposure: exposure,
				},
			};
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
		setSelectedByServerKey({});
	}, []);

	const restoreContext = useCallback((context?: MCPConversationContext) => {
		setSelectedByServerKey(mcpContextToSelectionMap(context));
	}, []);

	const mcpContext = useMemo(() => mcpSelectionToContext(selectedByServerKey), [selectedByServerKey]);

	const selectedServerCount = Object.keys(selectedByServerKey).length;
	const selectedToolCount = Object.values(selectedByServerKey).reduce(
		(sum, selection) =>
			sum +
			(selection.toolExposure === MCPToolExposure.MCPToolExposureAll
				? 1
				: selection.toolExposure === MCPToolExposure.MCPToolExposureSelected
					? selection.selectedTools.length
					: 0),
		0
	);
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
