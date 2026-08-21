import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import type { ArtifactRef } from '@/spec/artifact';
import type {
	MCPAuthHealth,
	MCPConversationContext,
	MCPOAuthAuthorization,
	MCPPromptRef,
	MCPPromptSelection,
	MCPResourceRef,
	MCPResourceTemplateRef,
	MCPResourceTemplateSelection,
	MCPToolCapability,
	MCPToolSelection,
} from '@/spec/mcp_artifact';
import {
	MCPAuthHealthState,
	MCPHTTPAuthMode,
	MCPServerStatus,
	MCPServerType,
	MCPToolExposure,
	MCPTransportType,
} from '@/spec/mcp_artifact';

import { areComparableValuesEqual, omitManyKeys } from '@/lib/obj_utils';

import { backendAPI, mcpAPI } from '@/apis/baseapi';

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
import {
	getAuthMode,
	isServerOperational,
	loadMCPBundleViews,
	loadMCPServerViews,
} from '@/mcpservers/lib/mcp_management';
import { isMCPToolModelSelectable, isMCPToolVisibleToModel } from '@/mcpservers/lib/mcp_server_utils';

type MCPDiscoveryLoadResult = Pick<MCPComposerServerOption, 'tools' | 'resources' | 'resourceTemplates' | 'prompts'>;

const MCP_CONNECTION_POLL_MS = 500;
const MCP_CONNECTION_TIMEOUT_MS = 11 * 60 * 1000;

function getErrorMessage(error: unknown, fallback: string): string {
	if (error instanceof Error && error.message.trim().length > 0) {
		return error.message;
	}
	if (typeof error === 'string' && error.trim().length > 0) {
		return error.trim();
	}
	return fallback;
}

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

function sleep(ms: number): Promise<void> {
	return new Promise(resolve => {
		window.setTimeout(resolve, ms);
	});
}

function overlayPendingOAuthAuthHealth(
	server: ArtifactRef,
	authHealth: MCPAuthHealth | undefined,
	pendingAuthorizations: MCPOAuthAuthorization[]
): MCPAuthHealth | undefined {
	const pending = pendingAuthorizations.find(
		authorization =>
			authorization.server.rootID === server.rootID &&
			authorization.server.artifactID === server.artifactID &&
			authorization.authorizationURL
	);

	if (!pending) {
		return authHealth;
	}

	return {
		...authHealth,
		server,
		authMode: MCPHTTPAuthMode.OAuth,
		state: MCPAuthHealthState.AuthorizationPending,
		configured: authHealth?.configured ?? true,
		authorizationPending: true,
		authorizationURL: pending.authorizationURL,
		authorizationExpiresAt: pending.expiresAt,
		lastError: undefined,
	};
}

function isOAuthServerOption(option: MCPComposerServerOption): boolean {
	return getAuthMode(option.server) === MCPHTTPAuthMode.OAuth;
}

function hasPendingOAuthHealth(option: MCPComposerServerOption): boolean {
	return (
		option.authHealth?.state === MCPAuthHealthState.AuthorizationPending ||
		Boolean(option.authHealth?.authorizationPending)
	);
}

export function optionKey(option: MCPComposerServerOption): string {
	return mcpServerKey(option.server.ref);
}

function areMCPAuthHealthEqual(a: MCPAuthHealth | undefined, b: MCPAuthHealth | undefined): boolean {
	return areComparableValuesEqual(a ?? null, b ?? null);
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
	bundle: MCPComposerServerOption['bundle'],
	server: MCPComposerServerOption['server'],
	runtime: MCPComposerServerOption['runtime'],
	authHealth: MCPComposerServerOption['authHealth']
): MCPComposerServerOption {
	return {
		bundle,
		server,
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

export function useComposerMCP(): UseComposerMCPResult {
	const [options, setOptions] = useState<MCPComposerServerOption[]>([]);
	const [selectedByServerKey, setSelectedByServerKey] = useState<Record<string, MCPComposerServerSelection>>({});
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | undefined>(undefined);

	const mountedRef = useRef(true);
	const optionsRef = useRef<MCPComposerServerOption[]>([]);
	const selectedByServerKeyRef = useRef<Record<string, MCPComposerServerSelection>>({});
	const discoveryPromisesRef = useRef(new Map<string, Promise<MCPDiscoveryLoadResult | undefined>>());
	const connectionPromisesRef = useRef(new Map<string, Promise<void>>());

	useEffect(() => {
		mountedRef.current = true;

		return () => {
			mountedRef.current = false;
			// oxlint-disable-next-line react-hooks/exhaustive-deps
			connectionPromisesRef.current.clear();
		};
	}, []);

	useEffect(() => {
		optionsRef.current = options;
	}, [options]);

	useEffect(() => {
		selectedByServerKeyRef.current = selectedByServerKey;
	}, [selectedByServerKey]);

	const commitSelectedByServerKey = useCallback(
		(updater: (prev: Record<string, MCPComposerServerSelection>) => Record<string, MCPComposerServerSelection>) => {
			setSelectedByServerKey(prev => {
				const next = updater(prev);
				selectedByServerKeyRef.current = next;
				return next;
			});
		},
		[]
	);

	const patchOption = useCallback((server: ArtifactRef, patch: Partial<MCPComposerServerOption>) => {
		const key = mcpServerKey(server);

		setOptions(prev => {
			let changed = false;
			const next = prev.map(option =>
				optionKey(option) === key && hasOptionPatchChanges(option, patch)
					? (() => {
							changed = true;
							return {
								...option,
								...patch,
							};
						})()
					: option
			);

			if (!changed) {
				return prev;
			}

			optionsRef.current = next;
			return next;
		});
	}, []);

	const refreshAll = useCallback(async () => {
		setLoading(true);
		setError(undefined);

		try {
			const [bundles, pendingAuthorizations] = await Promise.all([
				loadMCPBundleViews(),
				mcpAPI.listPendingMCPOAuthAuthorizations().catch(() => []),
			]);

			const nestedOptions = await Promise.all(
				bundles.map(async bundle => {
					const servers = await loadMCPServerViews(bundle);

					return await Promise.all(
						servers.map(async server => {
							const [runtime, authHealth] = await Promise.all([
								mcpAPI.getMCPServerStatus(server.ref).catch(() => undefined),
								mcpAPI.getMCPServerAuthHealth(server.ref).catch(() => undefined),
							]);

							return optionFromServer(
								bundle,
								server,
								runtime,
								overlayPendingOAuthAuthHealth(server.ref, authHealth, pendingAuthorizations)
							);
						})
					);
				})
			);

			const nextOptions = nestedOptions.flat();

			if (!mountedRef.current) {
				return;
			}

			optionsRef.current = nextOptions;
			setOptions(nextOptions);
		} catch (cause) {
			if (!mountedRef.current) {
				return;
			}

			setError(getErrorMessage(cause, 'Failed to load MCP servers.'));
		} finally {
			if (mountedRef.current) {
				setLoading(false);
			}
		}
	}, []);

	useEffect(() => {
		void refreshAll();
	}, [refreshAll]);

	const loadDiscoveryForServer = useCallback(
		async (server: ArtifactRef, force = false): Promise<MCPDiscoveryLoadResult | undefined> => {
			const key = mcpServerKey(server);
			const current = optionsRef.current.find(option => optionKey(option) === key);

			if (!current) {
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

			if (!force && (current.discoveryError || current.discoveryLoading)) {
				return undefined;
			}

			patchOption(server, {
				discoveryLoading: true,
				discoveryError: undefined,
			});

			const promise = (async (): Promise<MCPDiscoveryLoadResult | undefined> => {
				const [toolsResult, resourcesResult, resourceTemplatesResult, promptsResult] = await Promise.allSettled([
					mcpAPI.listMCPServerTools(server),
					mcpAPI.listMCPServerResources(server),
					mcpAPI.listMCPServerResourceTemplates(server),
					mcpAPI.listMCPServerPrompts(server),
				]);

				const toolsDiscovery = normalizeMCPDiscoveryList<MCPToolCapability>(toolsResult, 'tools');
				const resourcesDiscovery = normalizeMCPDiscoveryList<MCPResourceRef>(resourcesResult, 'resources');
				const templatesDiscovery = normalizeMCPDiscoveryList<MCPResourceTemplateRef>(
					resourceTemplatesResult,
					'resource templates'
				);
				const promptsDiscovery = normalizeMCPDiscoveryList<MCPPromptRef>(promptsResult, 'prompts');

				const tools = toolsDiscovery.items;
				const resources = resourcesDiscovery.items;
				const resourceTemplates = templatesDiscovery.items;
				const prompts = promptsDiscovery.items;

				const discoveryErrors = [
					toolsDiscovery.error,
					resourcesDiscovery.error,
					templatesDiscovery.error,
					promptsDiscovery.error,
				].filter((message): message is string => typeof message === 'string' && message.length > 0);

				if (!mountedRef.current) {
					return undefined;
				}

				patchOption(server, {
					tools,
					resources,
					resourceTemplates,
					prompts,
					discoveryLoaded: discoveryErrors.length === 0,
					discoveryLoading: false,
					discoveryError: discoveryErrors[0],
				});

				commitSelectedByServerKey(prev => {
					const currentSelection = prev[key];
					if (!currentSelection || currentSelection.toolExposure !== MCPToolExposure.All) {
						return prev;
					}

					return {
						...prev,
						[key]: {
							...currentSelection,
							selectedTools: modelSelectableTools(tools).map(t => {
								return toolToSelection(t);
							}),
						},
					};
				});

				return discoveryErrors.length === 0
					? {
							tools,
							resources,
							resourceTemplates,
							prompts,
						}
					: undefined;
			})().catch((cause: unknown) => {
				if (!mountedRef.current) {
					return undefined;
				}

				patchOption(server, {
					discoveryLoading: false,
					discoveryLoaded: false,
					discoveryError: getErrorMessage(cause, 'Failed to load MCP discovery.'),
				});

				return undefined;
			});

			discoveryPromisesRef.current.set(key, promise);

			try {
				return await promise;
			} finally {
				discoveryPromisesRef.current.delete(key);
			}
		},
		[commitSelectedByServerKey, patchOption]
	);

	const ensureDiscoveryLoaded = useCallback(
		async (server: ArtifactRef) => {
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

	const refreshServerStatus = useCallback(
		async (server: ArtifactRef) => {
			const previous = optionsRef.current.find(option => optionKey(option) === mcpServerKey(server));
			const [runtime, authHealth, pendingAuthorizations] = await Promise.all([
				mcpAPI.getMCPServerStatus(server).catch(() => undefined),
				mcpAPI.getMCPServerAuthHealth(server).catch(() => undefined),
				mcpAPI.listPendingMCPOAuthAuthorizations().catch(() => []),
			]);

			if (!mountedRef.current) {
				return {
					runtime: runtime ?? previous?.runtime,
					authHealth: authHealth ?? previous?.authHealth,
				};
			}

			const nextRuntime = runtime ?? previous?.runtime;
			const nextAuthHealth = overlayPendingOAuthAuthHealth(
				server,
				authHealth ?? previous?.authHealth,
				pendingAuthorizations
			);
			patchOption(server, {
				runtime: nextRuntime,
				authHealth: nextAuthHealth,
			});
			return { runtime: nextRuntime, authHealth: nextAuthHealth };
		},
		[patchOption]
	);

	const refreshServer = useCallback(
		async (server: ArtifactRef) => {
			patchOption(server, {
				discoveryLoaded: false,
				discoveryLoading: true,
				discoveryError: undefined,
			});

			try {
				const runtime = await mcpAPI.refreshMCPServer(server);

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
		(server: ArtifactRef): Promise<void> => {
			const key = mcpServerKey(server);
			const existing = connectionPromisesRef.current.get(key);
			if (existing) {
				return existing;
			}

			const promise = (async () => {
				const current = optionsRef.current.find(option => optionKey(option) === key);
				let runtime =
					current?.runtime?.status === MCPServerStatus.Connecting
						? current.runtime
						: await mcpAPI.connectMCPServer(server);

				if (mountedRef.current) {
					patchOption(server, {
						runtime,
						discoveryLoaded: false,
						discoveryLoading: false,
						discoveryError: undefined,
					});
				}

				const deadline = Date.now() + MCP_CONNECTION_TIMEOUT_MS;
				while (mountedRef.current && runtime.status === MCPServerStatus.Connecting) {
					if (Date.now() >= deadline) {
						throw new Error('Timed out waiting for the MCP server to connect.');
					}

					await sleep(MCP_CONNECTION_POLL_MS);
					const refreshed = await refreshServerStatus(server).catch(() => undefined);
					runtime = refreshed?.runtime ?? runtime;
				}

				if (!mountedRef.current) {
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
			});
		},
		[loadDiscoveryForServer, patchOption, refreshServerStatus]
	);

	const refreshPendingOAuthAuthorizations = useCallback(async () => {
		const pendingAuthorizations = await mcpAPI.listPendingMCPOAuthAuthorizations().catch(() => []);
		const stalePending = optionsRef.current.filter(option => {
			if (!hasPendingOAuthHealth(option)) {
				return false;
			}

			return !pendingAuthorizations.some(authorization => mcpServerKey(authorization.server) === optionKey(option));
		});

		const freshHealthEntries = await Promise.all(
			stalePending.map(async option => ({
				key: optionKey(option),
				authHealth: await mcpAPI.getMCPServerAuthHealth(option.server.ref).catch(() => undefined),
			}))
		);

		const freshHealthByKey = new Map(freshHealthEntries.map(entry => [entry.key, entry.authHealth] as const));

		if (!mountedRef.current) {
			return;
		}

		setOptions(prev => {
			let changed = false;

			const next = prev.map(option => {
				const refreshed = freshHealthByKey.get(optionKey(option));
				const authHealth = overlayPendingOAuthAuthHealth(
					option.server.ref,
					refreshed ?? option.authHealth,
					pendingAuthorizations
				);

				if (areMCPAuthHealthEqual(authHealth, option.authHealth)) {
					return option;
				}

				changed = true;
				return {
					...option,
					authHealth,
				};
			});

			if (!changed) {
				return prev;
			}

			optionsRef.current = next;
			return next;
		});
	}, []);

	const disconnectServer = useCallback(
		async (server: ArtifactRef) => {
			await mcpAPI.disconnectMCPServer(server);
			await refreshServerStatus(server);
		},
		[refreshServerStatus]
	);

	const cancelOAuth = useCallback(
		async (server: ArtifactRef) => {
			await mcpAPI.cancelPendingMCPOAuthAuthorization(server);
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

			commitSelectedByServerKey(prev => {
				if (!selected) {
					return omitManyKeys(prev, [key]);
				}

				if (prev[key]) {
					return prev;
				}

				return {
					...prev,
					[key]: {
						server: option.server.ref,
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
		(server: ArtifactRef): boolean => {
			const key = mcpServerKey(server);
			if (selectedByServerKeyRef.current[key]) {
				return true;
			}

			const option = optionsRef.current.find(item => optionKey(item) === key);
			if (
				!option ||
				!option.bundle.enabled ||
				!option.server.runtimeEnabled ||
				!option.server.artifact.enabled ||
				!isServerOperational(option.server)
			) {
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
		[commitSelectedByServerKey]
	);

	const setToolExposure = useCallback(
		(server: ArtifactRef, exposure: MCPToolExposure) => {
			const key = mcpServerKey(server);

			commitSelectedByServerKey(prev => {
				const current = prev[key];
				if (!current) {
					return prev;
				}

				const option = optionsRef.current.find(item => optionKey(item) === key);

				return {
					...prev,
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
		(server: ArtifactRef, include: boolean) => {
			const key = mcpServerKey(server);

			commitSelectedByServerKey(prev => {
				const current = prev[key];
				if (!current) {
					return prev;
				}

				return {
					...prev,
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

			commitSelectedByServerKey(prev => {
				const current = prev[key];
				if (!current) {
					return prev;
				}

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
		},
		[commitSelectedByServerKey]
	);

	const toggleResource = useCallback(
		(resource: MCPResourceRef, selected: boolean) => {
			const key = mcpServerKey(resource.server);

			commitSelectedByServerKey(prev => {
				const current = prev[key];
				if (!current) {
					return prev;
				}

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
		},
		[commitSelectedByServerKey]
	);

	const toggleResourceTemplate = useCallback(
		(template: MCPResourceTemplateRef, selected: boolean) => {
			const key = mcpServerKey(template.server);
			const templateSelection: MCPResourceTemplateSelection = {
				...template,
				argumentValues: {},
			};

			commitSelectedByServerKey(prev => {
				const current = prev[key];
				if (!current) {
					return prev;
				}

				return {
					...prev,
					[key]: {
						...current,
						selectedResourceTemplates: selected
							? upsertByKey(current.selectedResourceTemplates, mcpResourceTemplateKey, templateSelection)
							: removeByKey(current.selectedResourceTemplates, mcpResourceTemplateKey, templateSelection),
					},
				};
			});
		},
		[commitSelectedByServerKey]
	);

	const togglePrompt = useCallback(
		(prompt: MCPPromptRef, selected: boolean) => {
			const key = mcpServerKey(prompt.server);
			const promptSelection: MCPPromptSelection = {
				...prompt,
				argumentValues: {},
			};

			commitSelectedByServerKey(prev => {
				const current = prev[key];
				if (!current) {
					return prev;
				}

				return {
					...prev,
					[key]: {
						...current,
						selectedPrompts: selected
							? upsertByKey(current.selectedPrompts, mcpPromptKey, promptSelection)
							: removeByKey(current.selectedPrompts, mcpPromptKey, promptSelection),
					},
				};
			});
		},
		[commitSelectedByServerKey]
	);

	const setResourceTemplateArgumentValue = useCallback(
		(server: ArtifactRef, uriTemplate: string, argumentName: string, value: string) => {
			const key = mcpServerKey(server);

			commitSelectedByServerKey(prev => {
				const current = prev[key];
				if (!current) {
					return prev;
				}

				return {
					...prev,
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
		(server: ArtifactRef, promptName: string, argumentName: string, value: string) => {
			const key = mcpServerKey(server);

			commitSelectedByServerKey(prev => {
				const current = prev[key];
				if (!current) {
					return prev;
				}

				return {
					...prev,
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
		const nextSelections: Record<string, MCPComposerServerSelection> = {};

		for (const selection of Object.values(currentSelections)) {
			const key = mcpServerKey(selection.server);
			let option = optionsRef.current.find(item => optionKey(item) === key);

			if (!option) {
				throw new Error(`Selected MCP server ${selection.server.artifactID} is no longer available.`);
			}
			if (
				!option.bundle.enabled ||
				!option.server.runtimeEnabled ||
				!option.server.artifact.enabled ||
				!isServerOperational(option.server)
			) {
				throw new Error(`Selected MCP server ${option.server.displayName} is disabled or unavailable.`);
			}
			if (option.runtime?.status !== MCPServerStatus.Ready) {
				await connectServer(selection.server);
				option = optionsRef.current.find(item => optionKey(item) === key);
			}
			if (!option || option.runtime?.status !== MCPServerStatus.Ready) {
				throw new Error(`MCP server ${option?.server.displayName ?? selection.server.artifactID} is not ready.`);
			}

			let selectedTools = selection.selectedTools;

			if (selection.toolExposure === MCPToolExposure.All) {
				const discovery = await loadDiscoveryForServer(selection.server).catch(() => undefined);
				if (!discovery) {
					throw new Error(`Could not load tools from MCP server ${option.server.displayName}.`);
				}
				const tools = discovery?.tools ?? option?.tools ?? [];
				selectedTools = modelSelectableTools(tools).map(t => {
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
	}, [commitSelectedByServerKey, connectServer, loadDiscoveryForServer]);

	const shouldPollOAuthAuthorizations = useMemo(
		() => options.some(option => isOAuthServerOption(option) || hasPendingOAuthHealth(option)),
		[options]
	);

	useEffect(() => {
		if (!shouldPollOAuthAuthorizations) {
			return;
		}

		let cancelled = false;
		const poll = () => {
			if (!cancelled) {
				void refreshPendingOAuthAuthorizations();
			}
		};

		poll();
		const timer = window.setInterval(poll, 2000);

		return () => {
			cancelled = true;
			window.clearInterval(timer);
		};
	}, [refreshPendingOAuthAuthorizations, shouldPollOAuthAuthorizations]);

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
