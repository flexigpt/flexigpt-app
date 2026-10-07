import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import type {
	MCPConversationContext,
	MCPPromptRef,
	MCPPromptSelection,
	MCPResourceRef,
	MCPResourceTemplateRef,
	MCPResourceTemplateSelection,
	MCPRuntimeServerID,
	MCPRuntimeServerView,
	MCPToolCapability,
	MCPToolSelection,
} from '@/spec/mcp';
import { MCPServerStatus, MCPServerType, MCPToolExposure, MCPTransportType } from '@/spec/mcp';

import { getErrorMessage } from '@/lib/error_utils';
import { areComparableValuesEqual, omitManyKeys } from '@/lib/obj_utils';

import { backendAPI, mcpManagementAPI } from '@/apis/baseapi';
import { getMCPRuntimeAuthHealth, isServerOperational } from '@/apis/mcp_management';
import { collectAllPages } from '@/apis/wailsapi/transport';

import type {
	MCPComposerServerOption,
	MCPComposerServerSelection,
	UseComposerMCPResult,
} from '@/chats/composer/mcp/mcp_composer_types';
import {
	countMissingRequiredMCPArguments,
	mcpContextToSelectionMap,
	mcpPromptKey,
	mcpResourceKey,
	mcpResourceTemplateKey,
	mcpSelectionToContext,
	mcpServerKey,
	mcpToolKey,
} from '@/chats/composer/mcp/mcp_composer_types';
import { isMCPToolModelSelectable, isMCPToolVisibleToModel } from '@/mcpservers/lib/mcp_server_utils';

type MCPDiscoveryLoadResult = Pick<MCPComposerServerOption, 'tools' | 'resources' | 'resourceTemplates' | 'prompts'>;

const MCP_CONNECTION_POLL_MS = 500;
const MCP_CONNECTION_TIMEOUT_MS = 11 * 60 * 1000;
const MCP_DISCOVERY_PAGE_SIZE = 100;
const MCP_DISCOVERY_MAX_PAGES = 10_000;

interface NormalizedMCPDiscoveryList<T> {
	items: T[];
	error?: string;
}

function normalizeMCPDiscoveryList<T>(result: PromiseSettledResult<T[]>, label: string): NormalizedMCPDiscoveryList<T> {
	if (result.status === 'rejected') {
		return {
			items: [],
			error: getErrorMessage(result.reason, `Failed to load MCP ${label}.`),
		};
	}

	const value: unknown = result.value;

	if (value === null || value === undefined) {
		return { items: [] };
	}

	if (!Array.isArray(value)) {
		return {
			items: [],
			error: `MCP ${label} discovery returned an invalid response.`,
		};
	}

	return { items: value as T[] };
}

function sleep(ms: number, signal: AbortSignal): Promise<void> {
	return new Promise(resolve => {
		if (signal.aborted) {
			resolve();
			return;
		}
		const finish = () => {
			window.clearTimeout(timer);
			signal.removeEventListener('abort', finish);
			resolve();
		};
		const timer = window.setTimeout(finish, ms);
		signal.addEventListener('abort', finish, { once: true });
	});
}

export function optionKey(option: MCPComposerServerOption): string {
	return mcpServerKey(option.runtimeServerID);
}

function hasOptionPatchChanges(option: MCPComposerServerOption, patch: Partial<MCPComposerServerOption>): boolean {
	for (const [key, value] of Object.entries(patch) as Array<[keyof MCPComposerServerOption, unknown]>) {
		if (!areComparableValuesEqual(option[key] ?? null, value ?? null)) {
			return true;
		}
	}

	return false;
}

function optionFromServer(
	plugin: MCPComposerServerOption['plugin'],
	server: MCPComposerServerOption['server'],
	runtime: MCPComposerServerOption['runtime'],
	authHealth: MCPComposerServerOption['authHealth']
): MCPComposerServerOption | undefined {
	const runtimeServerID = server.runtimeServerID;
	if (!runtimeServerID) {
		return undefined;
	}

	return {
		plugin,
		server,
		runtimeServerID,
		transport:
			server.document?.mcpServer.type === MCPServerType.Stdio
				? MCPTransportType.Stdio
				: MCPTransportType.StreamableHTTP,
		runtime,
		authHealth,
		tools: [],
		resources: [],
		resourceTemplates: [],
		prompts: [],
		discoveryLoaded: false,
		discoveryLoading: false,
	};
}

function toolToSelection(tool: MCPToolCapability): MCPToolSelection {
	return {
		server: tool.server,
		toolName: tool.toolName,
		providerToolName: tool.providerToolName,
		choiceID: tool.choiceID,
		digest: tool.digest,
		approvalRule: tool.approvalRule,
		executionMode: tool.executionMode,
		appResourceUri: tool.app?.resourceUri,
		visibility: tool.app?.visibility,
	};
}

function upsertByKey<T>(items: T[], keyFn: (item: T) => string, item: T): T[] {
	const key = keyFn(item);
	return items.some(existing => keyFn(existing) === key) ? items : [...items, item];
}

function removeByKey<T>(items: T[], keyFn: (item: T) => string, item: T): T[] {
	const key = keyFn(item);
	return items.filter(existing => keyFn(existing) !== key);
}

function withArgumentValue<T extends { argumentValues?: Record<string, string> }>(
	item: T,
	argumentName: string,
	value: string
): T {
	let nextValues = {
		...item.argumentValues,
		[argumentName]: value,
	};

	if (!value.trim()) {
		nextValues = omitManyKeys(nextValues, [argumentName]);
	}

	return {
		...item,
		argumentValues: Object.keys(nextValues).length > 0 ? nextValues : undefined,
	};
}

function modelSelectableTools(tools: MCPToolCapability[]): MCPToolCapability[] {
	return tools.filter(tool => isMCPToolModelSelectable(tool));
}

export function useComposerMCP(catalogRequested: boolean): UseComposerMCPResult {
	const [options, setOptions] = useState<MCPComposerServerOption[]>([]);
	const [selectedByServerKey, setSelectedByServerKey] = useState<Record<string, MCPComposerServerSelection>>({});
	const [loading, setLoading] = useState(false);
	const [error, setError] = useState<string | undefined>();

	const mountedRef = useRef(true);
	const optionsRef = useRef<MCPComposerServerOption[]>([]);
	const selectedByServerKeyRef = useRef<Record<string, MCPComposerServerSelection>>({});
	const discoveryPromisesRef = useRef(new Map<string, Promise<MCPDiscoveryLoadResult | undefined>>());
	const connectionPromisesRef = useRef(new Map<string, Promise<void>>());
	const connectionControllersRef = useRef(new Map<string, AbortController>());
	const observedGenerationsRef = useRef(new Map<string, number>());
	const discoveryVersionsRef = useRef(new Map<string, number>());
	const statusRequestsRef = useRef(new Map<string, number>());
	const lifetimeRef = useRef(0);
	const catalogLoadedRef = useRef(false);
	const catalogPromiseRef = useRef<Promise<void> | undefined>(undefined);

	useEffect(() => {
		mountedRef.current = true;

		return () => {
			mountedRef.current = false;
			lifetimeRef.current += 1;
			catalogPromiseRef.current = undefined;
			// oxlint-disable-next-line react-hooks/exhaustive-deps
			for (const controller of connectionControllersRef.current.values()) {
				controller.abort();
			}
			// oxlint-disable-next-line react-hooks/exhaustive-deps
			connectionControllersRef.current.clear();
			// oxlint-disable-next-line react-hooks/exhaustive-deps
			discoveryPromisesRef.current.clear();
			// oxlint-disable-next-line react-hooks/exhaustive-deps
			connectionPromisesRef.current.clear();
		};
	}, []);

	useEffect(() => {
		selectedByServerKeyRef.current = selectedByServerKey;
	}, [selectedByServerKey]);

	const commitSelectedByServerKey = useCallback(
		(updater: (previous: Record<string, MCPComposerServerSelection>) => Record<string, MCPComposerServerSelection>) => {
			setSelectedByServerKey(previous => {
				const next = updater(previous);
				selectedByServerKeyRef.current = next;
				return next;
			});
		},
		[]
	);

	const patchOption = useCallback((server: MCPRuntimeServerID, patch: Partial<MCPComposerServerOption>) => {
		const key = mcpServerKey(server);
		if (!mountedRef.current) {
			return;
		}
		let changed = false;
		const next = optionsRef.current.map(option => {
			if (optionKey(option) !== key || !hasOptionPatchChanges(option, patch)) {
				return option;
			}
			changed = true;
			return { ...option, ...patch };
		});
		if (changed) {
			optionsRef.current = next;
			setOptions(next);
		}
	}, []);

	const loadDeclarationCatalog = useCallback((force = false): Promise<void> => {
		if (catalogPromiseRef.current) {
			return catalogPromiseRef.current;
		}
		if (!force && catalogLoadedRef.current) {
			return Promise.resolve();
		}
		const lifetime = lifetimeRef.current;
		const promise = (async () => {
			setLoading(true);
			setError(undefined);
			try {
				const declarations = await mcpManagementAPI.listComposerMCPDeclarations(force);
				if (!mountedRef.current || lifetime !== lifetimeRef.current) {
					return;
				}
				const unique = new Map<string, MCPComposerServerOption>();
				for (const { plugin, servers } of declarations) {
					for (const server of servers) {
						const option = optionFromServer(plugin, server, server.runtime, server.authHealth);
						if (!option) {
							continue;
						}
						const existing = unique.get(optionKey(option));
						if (!existing || (!existing.plugin.enabled && option.plugin.enabled)) {
							unique.set(optionKey(option), option);
						}
					}
				}
				for (const key of discoveryVersionsRef.current.keys()) {
					discoveryVersionsRef.current.set(key, (discoveryVersionsRef.current.get(key) ?? 0) + 1);
				}
				discoveryPromisesRef.current.clear();
				optionsRef.current = [...unique.values()];
				catalogLoadedRef.current = true;
				setOptions(optionsRef.current);
			} catch (cause) {
				if (mountedRef.current && lifetime === lifetimeRef.current) {
					setError(getErrorMessage(cause, 'Failed to load MCP servers.'));
				}
			} finally {
				if (mountedRef.current && lifetime === lifetimeRef.current) {
					setLoading(false);
				}
			}
		})().finally(() => {
			if (catalogPromiseRef.current === promise) {
				catalogPromiseRef.current = undefined;
			}
		});
		catalogPromiseRef.current = promise;
		return promise;
	}, []);

	const refreshAll = useCallback(() => loadDeclarationCatalog(true), [loadDeclarationCatalog]);
	const hasSelections = Object.keys(selectedByServerKey).length > 0;

	useEffect(() => {
		if (catalogRequested || hasSelections) {
			void loadDeclarationCatalog();
		}
	}, [catalogRequested, hasSelections, loadDeclarationCatalog]);

	const loadDiscoveryForServer = useCallback(
		async (server: MCPRuntimeServerID, force = false): Promise<MCPDiscoveryLoadResult | undefined> => {
			const key = mcpServerKey(server);
			const current = optionsRef.current.find(option => optionKey(option) === key);

			if (!current || current.runtime?.status !== MCPServerStatus.Ready || !current.runtime.snapshotDigest) {
				return undefined;
			}

			if (!force && current.discoveryLoaded) {
				return {
					tools: current.tools,
					resources: current.resources,
					resourceTemplates: current.resourceTemplates,
					prompts: current.prompts,
				};
			}

			const existing = discoveryPromisesRef.current.get(key);
			if (existing) {
				return existing;
			}

			if (!force && (current.discoveryLoading || current.discoveryError)) {
				return undefined;
			}

			patchOption(server, {
				discoveryLoading: true,
				discoveryError: undefined,
			});

			const lifetime = lifetimeRef.current;
			const version = discoveryVersionsRef.current.get(key) ?? 0;
			discoveryVersionsRef.current.set(key, version);
			const generation = current.runtime.generation;
			const digest = current.runtime.snapshotDigest;
			const isCurrent = () => {
				const latest = optionsRef.current.find(option => optionKey(option) === key);
				return (
					mountedRef.current &&
					lifetime === lifetimeRef.current &&
					version === discoveryVersionsRef.current.get(key) &&
					latest?.runtime?.generation === generation &&
					latest.runtime.snapshotDigest === digest
				);
			};
			const promise = (async (): Promise<MCPDiscoveryLoadResult | undefined> => {
				const [toolsResult, resourcesResult, resourceTemplatesResult, promptsResult] = await Promise.allSettled([
					collectAllPages(
						pageToken => mcpManagementAPI.listMCPServerTools(server, MCP_DISCOVERY_PAGE_SIZE, pageToken),
						MCP_DISCOVERY_MAX_PAGES
					),
					collectAllPages(
						pageToken => mcpManagementAPI.listMCPServerResources(server, MCP_DISCOVERY_PAGE_SIZE, pageToken),
						MCP_DISCOVERY_MAX_PAGES
					),
					collectAllPages(
						pageToken => mcpManagementAPI.listMCPServerResourceTemplates(server, MCP_DISCOVERY_PAGE_SIZE, pageToken),
						MCP_DISCOVERY_MAX_PAGES
					),
					collectAllPages(
						pageToken => mcpManagementAPI.listMCPServerPrompts(server, MCP_DISCOVERY_PAGE_SIZE, pageToken),
						MCP_DISCOVERY_MAX_PAGES
					),
				]);

				const tools = normalizeMCPDiscoveryList<MCPToolCapability>(toolsResult, 'tools');
				const resources = normalizeMCPDiscoveryList<MCPResourceRef>(resourcesResult, 'resources');
				const templates = normalizeMCPDiscoveryList<MCPResourceTemplateRef>(
					resourceTemplatesResult,
					'resource templates'
				);
				const prompts = normalizeMCPDiscoveryList<MCPPromptRef>(promptsResult, 'prompts');

				const errors = [tools.error, resources.error, templates.error, prompts.error].filter(
					(message): message is string => typeof message === 'string' && message.length > 0
				);

				if (!isCurrent()) {
					return undefined;
				}

				patchOption(server, {
					tools: tools.items,
					resources: resources.items,
					resourceTemplates: templates.items,
					prompts: prompts.items,
					discoveryLoaded: errors.length === 0,
					discoveryLoading: false,
					discoveryError: errors[0],
				});

				commitSelectedByServerKey(previous => {
					const selection = previous[key];
					if (errors.length > 0 || !selection || selection.toolExposure !== MCPToolExposure.All) {
						return previous;
					}

					return {
						...previous,
						[key]: {
							...selection,
							selectedTools: modelSelectableTools(tools.items).map(t => {
								return toolToSelection(t);
							}),
						},
					};
				});

				return errors.length === 0
					? {
							tools: tools.items,
							resources: resources.items,
							resourceTemplates: templates.items,
							prompts: prompts.items,
						}
					: undefined;
			})().catch((cause: unknown) => {
				if (!isCurrent()) {
					return undefined;
				}

				patchOption(server, {
					discoveryLoaded: false,
					discoveryLoading: false,
					discoveryError: getErrorMessage(cause, 'Failed to load MCP discovery.'),
				});

				return undefined;
			});

			discoveryPromisesRef.current.set(key, promise);

			try {
				return await promise;
			} finally {
				if (discoveryPromisesRef.current.get(key) === promise) {
					discoveryPromisesRef.current.delete(key);
				}
			}
		},
		[commitSelectedByServerKey, patchOption]
	);

	const ensureDiscoveryLoaded = useCallback(
		async (server: MCPRuntimeServerID) => {
			await loadDiscoveryForServer(server);
		},
		[loadDiscoveryForServer]
	);

	const selectedServerKeys = useMemo(
		() =>
			Object.values(selectedByServerKey)
				.map(selection => mcpServerKey(selection.server))
				.toSorted()
				.join('\n'),
		[selectedByServerKey]
	);

	useEffect(() => {
		for (const selection of Object.values(selectedByServerKeyRef.current)) {
			void ensureDiscoveryLoaded(selection.server);
		}
	}, [ensureDiscoveryLoaded, options, selectedServerKeys]);

	const applyRuntimeView = useCallback(
		(view: MCPRuntimeServerView) => {
			const key = mcpServerKey(view.runtimeServerID);
			const current = optionsRef.current.find(option => optionKey(option) === key);
			if (!current || view.runtime.generation < (current.runtime?.generation ?? 0)) {
				return;
			}
			const changed =
				current.runtime?.generation !== view.runtime.generation ||
				current.runtime?.snapshotDigest !== view.runtime.snapshotDigest;
			if (changed) {
				discoveryVersionsRef.current.set(key, (discoveryVersionsRef.current.get(key) ?? 0) + 1);
				discoveryPromisesRef.current.delete(key);
			}
			patchOption(view.runtimeServerID, {
				runtime: view.runtime,
				authHealth: getMCPRuntimeAuthHealth(current.server.authHealth, view),
				...(changed
					? {
							tools: [],
							resources: [],
							resourceTemplates: [],
							prompts: [],
							discoveryLoaded: false,
							discoveryLoading: false,
							discoveryError: undefined,
						}
					: {}),
			});
		},
		[patchOption]
	);

	const refreshStatuses = useCallback(
		async (servers: MCPRuntimeServerID[]) => {
			const lifetime = lifetimeRef.current;
			const requests = new Map(
				servers.map(server => {
					const next = (statusRequestsRef.current.get(server) ?? 0) + 1;
					statusRequestsRef.current.set(server, next);
					return [server, next] as const;
				})
			);
			const views = await mcpManagementAPI.getMCPServersForRuntimeServers(servers);
			if (!mountedRef.current || lifetime !== lifetimeRef.current) {
				return;
			}
			for (const view of views) {
				if (statusRequestsRef.current.get(view.runtimeServerID) === requests.get(view.runtimeServerID)) {
					applyRuntimeView(view);
				}
			}
		},
		[applyRuntimeView]
	);

	const refreshServerStatus = useCallback(
		async (server: MCPRuntimeServerID) => {
			await refreshStatuses([server]);
			const option = optionsRef.current.find(value => optionKey(value) === mcpServerKey(server));
			return option ? { runtime: option.runtime, authHealth: option.authHealth } : undefined;
		},
		[refreshStatuses]
	);

	const refreshRuntimeStates = useCallback(async () => {
		const wasLoaded = catalogLoadedRef.current;
		await loadDeclarationCatalog();
		if (!mountedRef.current || !catalogLoadedRef.current || !wasLoaded) {
			return;
		}
		await refreshStatuses(optionsRef.current.map(option => option.runtimeServerID)).catch(() => undefined);
	}, [loadDeclarationCatalog, refreshStatuses]);

	const refreshServer = useCallback(
		async (server: MCPRuntimeServerID) => {
			const key = mcpServerKey(server);
			discoveryVersionsRef.current.set(key, (discoveryVersionsRef.current.get(key) ?? 0) + 1);
			discoveryPromisesRef.current.delete(key);
			patchOption(server, {
				discoveryLoaded: false,
				discoveryLoading: true,
				discoveryError: undefined,
			});

			try {
				const runtime = await mcpManagementAPI.refreshMCPServer(server);

				if (mountedRef.current) {
					patchOption(server, { runtime });
				}
			} catch (cause) {
				if (mountedRef.current) {
					patchOption(server, {
						discoveryLoading: false,
						discoveryError: getErrorMessage(cause, 'Failed to refresh MCP discovery.'),
					});
				}

				await refreshServerStatus(server).catch(() => undefined);
				return;
			}

			await refreshServerStatus(server).catch(() => undefined);
			await loadDiscoveryForServer(server, true).catch(() => undefined);
		},
		[loadDiscoveryForServer, patchOption, refreshServerStatus]
	);

	const connectServer = useCallback(
		(server: MCPRuntimeServerID, observeOnly = false): Promise<void> => {
			const key = mcpServerKey(server);
			const inFlight = connectionPromisesRef.current.get(key);

			if (inFlight) {
				return inFlight;
			}

			const controller = new AbortController();
			const { signal } = controller;
			connectionControllersRef.current.set(key, controller);
			const promise = (async () => {
				await refreshServerStatus(server);
				if (signal.aborted || !mountedRef.current) {
					return;
				}
				let runtime = optionsRef.current.find(option => optionKey(option) === key)?.runtime;
				if (!runtime) {
					throw new Error('MCP runtime state is unavailable.');
				}
				if (runtime.status === MCPServerStatus.Ready) {
					await loadDiscoveryForServer(server);
					return;
				}
				if (runtime.status !== MCPServerStatus.Connecting) {
					if (observeOnly) {
						return;
					}
					runtime = await mcpManagementAPI.connectMCPServer(server);
				}
				if (signal.aborted || !mountedRef.current) {
					return;
				}
				const generation = runtime.generation;
				observedGenerationsRef.current.set(key, generation);

				if (mountedRef.current) {
					patchOption(server, {
						runtime,
						discoveryLoaded: false,
						discoveryLoading: false,
						discoveryError: undefined,
					});
				}

				const deadline = Date.now() + MCP_CONNECTION_TIMEOUT_MS;

				while (mountedRef.current && !signal.aborted && runtime.status === MCPServerStatus.Connecting) {
					if (Date.now() >= deadline) {
						throw new Error('Timed out waiting for the MCP server to connect.');
					}

					await sleep(MCP_CONNECTION_POLL_MS, signal);
					if (signal.aborted || !mountedRef.current) {
						return;
					}
					const refreshed = await refreshServerStatus(server).catch(() => undefined);
					runtime = refreshed?.runtime ?? runtime;
					if (runtime.generation !== generation) {
						return;
					}
				}

				if (!mountedRef.current || signal.aborted) {
					return;
				}

				if (runtime.status === MCPServerStatus.Error) {
					throw new Error(runtime.lastError || 'The MCP server connection failed.');
				}

				if (runtime.status !== MCPServerStatus.Ready) {
					throw new Error('The MCP server disconnected before becoming ready.');
				}

				await refreshServerStatus(server).catch(() => undefined);
				await loadDiscoveryForServer(server, true);
			})();

			connectionPromisesRef.current.set(key, promise);

			return promise.finally(() => {
				if (connectionPromisesRef.current.get(key) === promise) {
					connectionPromisesRef.current.delete(key);
				}
				if (connectionControllersRef.current.get(key) === controller) {
					connectionControllersRef.current.delete(key);
				}
			});
		},
		[loadDiscoveryForServer, patchOption, refreshServerStatus]
	);

	useEffect(() => {
		for (const option of options) {
			const runtime = option.runtime;
			const key = optionKey(option);
			if (
				runtime?.status !== MCPServerStatus.Connecting ||
				connectionPromisesRef.current.has(key) ||
				observedGenerationsRef.current.get(key) === runtime.generation
			) {
				continue;
			}
			observedGenerationsRef.current.set(key, runtime.generation);
			void connectServer(option.runtimeServerID, true).catch(console.error);
		}
	}, [connectServer, options]);

	const disconnectServer = useCallback(
		async (server: MCPRuntimeServerID) => {
			connectionControllersRef.current.get(mcpServerKey(server))?.abort();
			statusRequestsRef.current.set(server, (statusRequestsRef.current.get(server) ?? 0) + 1);
			await mcpManagementAPI.disconnectMCPServer(server);
			await refreshServerStatus(server);
		},
		[refreshServerStatus]
	);

	const cancelOAuth = useCallback(
		async (server: MCPRuntimeServerID) => {
			connectionControllersRef.current.get(mcpServerKey(server))?.abort();
			statusRequestsRef.current.set(server, (statusRequestsRef.current.get(server) ?? 0) + 1);
			await mcpManagementAPI.cancelMCPServerAuthorization(server);
			await refreshServerStatus(server);
		},
		[refreshServerStatus]
	);

	const openAuthURL = useCallback((url: string) => {
		if (url) {
			backendAPI.openURL(url);
		}
	}, []);

	const setServerSelected = useCallback(
		(option: MCPComposerServerOption, selected: boolean) => {
			const key = optionKey(option);

			commitSelectedByServerKey(previous => {
				if (!selected) {
					return omitManyKeys(previous, [key]);
				}

				if (previous[key]) {
					return previous;
				}

				return {
					...previous,
					[key]: {
						server: option.runtimeServerID,
						snapshotDigest: option.runtime?.snapshotDigest,
						toolExposure: MCPToolExposure.All,
						selectedTools:
							option.tools.length > 0
								? modelSelectableTools(option.tools).map(t => {
										return toolToSelection(t);
									})
								: [],
						selectedResources: [],
						selectedResourceTemplates: [],
						selectedPrompts: [],
						includeServerInstructions: true,
					},
				};
			});
		},
		[commitSelectedByServerKey]
	);

	const ensureServerSelected = useCallback(
		async (server: MCPRuntimeServerID): Promise<boolean> => {
			await loadDeclarationCatalog();
			if (!mountedRef.current || !catalogLoadedRef.current) {
				return false;
			}
			const key = mcpServerKey(server);

			if (selectedByServerKeyRef.current[key]) {
				return true;
			}

			const option = optionsRef.current.find(item => optionKey(item) === key);

			if (!option || !option.plugin.enabled || !option.server.artifact.enabled || !isServerOperational(option.server)) {
				return false;
			}

			commitSelectedByServerKey(previous => ({
				...previous,
				[key]: {
					server,
					snapshotDigest: option.runtime?.snapshotDigest,
					toolExposure: MCPToolExposure.None,
					selectedTools: [],
					selectedResources: [],
					selectedResourceTemplates: [],
					selectedPrompts: [],
					includeServerInstructions: false,
				},
			}));

			return true;
		},
		[commitSelectedByServerKey, loadDeclarationCatalog]
	);

	const setToolExposure = useCallback(
		(server: MCPRuntimeServerID, exposure: MCPToolExposure) => {
			const key = mcpServerKey(server);

			commitSelectedByServerKey(previous => {
				const current = previous[key];
				if (!current) {
					return previous;
				}

				const option = optionsRef.current.find(item => optionKey(item) === key);

				return {
					...previous,
					[key]: {
						...current,
						toolExposure: exposure,
						selectedTools:
							exposure === MCPToolExposure.All
								? modelSelectableTools(option?.tools ?? []).map(t => {
										return toolToSelection(t);
									})
								: exposure === MCPToolExposure.None
									? []
									: current.selectedTools,
					},
				};
			});
		},
		[commitSelectedByServerKey]
	);

	const setIncludeServerInstructions = useCallback(
		(server: MCPRuntimeServerID, include: boolean) => {
			const key = mcpServerKey(server);

			commitSelectedByServerKey(previous => {
				const current = previous[key];
				if (!current) {
					return previous;
				}

				return {
					...previous,
					[key]: {
						...current,
						includeServerInstructions: include,
					},
				};
			});
		},
		[commitSelectedByServerKey]
	);

	const toggleTool = useCallback(
		(tool: MCPToolCapability, selected: boolean) => {
			if (selected && !isMCPToolVisibleToModel(tool)) {
				return;
			}

			const key = mcpServerKey(tool.server);
			const selection = toolToSelection(tool);

			commitSelectedByServerKey(previous => {
				const current = previous[key];
				if (!current) {
					return previous;
				}

				return {
					...previous,
					[key]: {
						...current,
						selectedTools: selected
							? upsertByKey(current.selectedTools, mcpToolKey, selection)
							: removeByKey(current.selectedTools, mcpToolKey, selection),
					},
				};
			});
		},
		[commitSelectedByServerKey]
	);

	const toggleResource = useCallback(
		(resource: MCPResourceRef, selected: boolean) => {
			const key = mcpServerKey(resource.server);

			commitSelectedByServerKey(previous => {
				const current = previous[key];
				if (!current) {
					return previous;
				}

				return {
					...previous,
					[key]: {
						...current,
						selectedResources: selected
							? upsertByKey(current.selectedResources, mcpResourceKey, resource)
							: removeByKey(current.selectedResources, mcpResourceKey, resource),
					},
				};
			});
		},
		[commitSelectedByServerKey]
	);

	const toggleResourceTemplate = useCallback(
		(template: MCPResourceTemplateRef, selected: boolean) => {
			const key = mcpServerKey(template.server);
			const selection: MCPResourceTemplateSelection = {
				...template,
				argumentValues: {},
			};

			commitSelectedByServerKey(previous => {
				const current = previous[key];
				if (!current) {
					return previous;
				}

				return {
					...previous,
					[key]: {
						...current,
						selectedResourceTemplates: selected
							? upsertByKey(current.selectedResourceTemplates, mcpResourceTemplateKey, selection)
							: removeByKey(current.selectedResourceTemplates, mcpResourceTemplateKey, selection),
					},
				};
			});
		},
		[commitSelectedByServerKey]
	);

	const togglePrompt = useCallback(
		(prompt: MCPPromptRef, selected: boolean) => {
			const key = mcpServerKey(prompt.server);
			const selection: MCPPromptSelection = {
				...prompt,
				argumentValues: {},
			};

			commitSelectedByServerKey(previous => {
				const current = previous[key];
				if (!current) {
					return previous;
				}

				return {
					...previous,
					[key]: {
						...current,
						selectedPrompts: selected
							? upsertByKey(current.selectedPrompts, mcpPromptKey, selection)
							: removeByKey(current.selectedPrompts, mcpPromptKey, selection),
					},
				};
			});
		},
		[commitSelectedByServerKey]
	);

	const setResourceTemplateArgumentValue = useCallback(
		(server: MCPRuntimeServerID, uriTemplate: string, argumentName: string, value: string) => {
			const key = mcpServerKey(server);

			commitSelectedByServerKey(previous => {
				const current = previous[key];
				if (!current) {
					return previous;
				}

				return {
					...previous,
					[key]: {
						...current,
						selectedResourceTemplates: current.selectedResourceTemplates.map(template =>
							template.uriTemplate === uriTemplate ? withArgumentValue(template, argumentName, value) : template
						),
					},
				};
			});
		},
		[commitSelectedByServerKey]
	);

	const setPromptArgumentValue = useCallback(
		(server: MCPRuntimeServerID, promptName: string, argumentName: string, value: string) => {
			const key = mcpServerKey(server);

			commitSelectedByServerKey(previous => {
				const current = previous[key];
				if (!current) {
					return previous;
				}

				return {
					...previous,
					[key]: {
						...current,
						selectedPrompts: current.selectedPrompts.map(prompt =>
							prompt.promptName === promptName ? withArgumentValue(prompt, argumentName, value) : prompt
						),
					},
				};
			});
		},
		[commitSelectedByServerKey]
	);

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
		if (Object.keys(currentSelections).length === 0) {
			return undefined;
		}
		await loadDeclarationCatalog();
		if (!mountedRef.current || !catalogLoadedRef.current) {
			throw new Error('MCP server declarations are unavailable.');
		}
		await refreshStatuses(Object.values(currentSelections).map(selection => selection.server));
		const nextSelections: Record<string, MCPComposerServerSelection> = {};

		for (const selection of Object.values(currentSelections)) {
			const key = mcpServerKey(selection.server);
			const option = optionsRef.current.find(item => optionKey(item) === key);
			if (
				!option ||
				!option.plugin.enabled ||
				!option.server.enabled ||
				option.runtime?.status !== MCPServerStatus.Ready
			) {
				throw new Error(`Connect MCP server ${option?.server.displayName ?? selection.server} before sending.`);
			}

			let selectedTools = selection.selectedTools;

			if (selection.toolExposure === MCPToolExposure.All) {
				const discovery = await loadDiscoveryForServer(selection.server).catch(() => undefined);

				if (!discovery) {
					throw new Error(`Could not load tools from MCP server ${option?.server.displayName ?? selection.server}.`);
				}

				selectedTools = modelSelectableTools(discovery.tools).map(t => {
					return toolToSelection(t);
				});
			}

			nextSelections[key] = {
				...selection,
				snapshotDigest: option?.runtime?.snapshotDigest ?? selection.snapshotDigest,
				selectedTools: selection.toolExposure === MCPToolExposure.None ? [] : selectedTools,
			};
		}

		selectedByServerKeyRef.current = nextSelections;
		commitSelectedByServerKey(() => nextSelections);

		const missing = countMissingRequiredMCPArguments([
			...Object.values(nextSelections).flatMap(selection => selection.selectedResourceTemplates),
			...Object.values(nextSelections).flatMap(selection => selection.selectedPrompts),
		]);

		if (missing > 0) {
			throw new Error(`Fill ${missing} required MCP argument${missing === 1 ? '' : 's'} before sending.`);
		}

		return mcpSelectionToContext(nextSelections);
	}, [commitSelectedByServerKey, loadDeclarationCatalog, loadDiscoveryForServer, refreshStatuses]);

	const mcpContext = useMemo(() => mcpSelectionToContext(selectedByServerKey), [selectedByServerKey]);

	const selectedServerCount = Object.keys(selectedByServerKey).length;

	const selectedToolCount = Object.values(selectedByServerKey).reduce((sum, selection) => {
		if (selection.toolExposure === MCPToolExposure.None) {
			return sum;
		}

		return sum + (selection.selectedTools.length > 0 ? selection.selectedTools.length : 1);
	}, 0);

	const selectedResourceCount = Object.values(selectedByServerKey).reduce(
		(sum, selection) => sum + selection.selectedResources.length + selection.selectedResourceTemplates.length,
		0
	);

	const selectedPromptCount = Object.values(selectedByServerKey).reduce(
		(sum, selection) => sum + selection.selectedPrompts.length,
		0
	);

	const requiredArgumentMissingCount = countMissingRequiredMCPArguments([
		...Object.values(selectedByServerKey).flatMap(selection => selection.selectedResourceTemplates),
		...Object.values(selectedByServerKey).flatMap(selection => selection.selectedPrompts),
	]);

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
		requiredArgumentMissingCount,
		argumentsBlocked: requiredArgumentMissingCount > 0,
		refreshAll,
		refreshRuntimeStates,
		refreshServer,
		ensureDiscoveryLoaded,
		prepareForSubmit,
		connectServer,
		disconnectServer,
		cancelOAuth,
		openAuthURL,
		setServerSelected,
		ensureServerSelected,
		setToolExposure,
		setIncludeServerInstructions,
		toggleTool,
		toggleResource,
		toggleResourceTemplate,
		togglePrompt,
		setResourceTemplateArgumentValue,
		setPromptArgumentValue,
		clear,
		restoreContext,
	};
}
