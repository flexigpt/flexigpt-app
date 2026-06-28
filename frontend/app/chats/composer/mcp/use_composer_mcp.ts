import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

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
} from '@/spec/mcp';
import { MCPAuthHealthState, MCPHTTPAuthMode, MCPToolExposure } from '@/spec/mcp';

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

function getErrorMessage(error: unknown, fallback: string): string {
	if (error instanceof Error && error.message.trim().length > 0) {
		return error.message;
	}
	return fallback;
}

function sleep(ms: number): Promise<void> {
	return new Promise(resolve => {
		window.setTimeout(() => {
			resolve();
		}, ms);
	});
}

function overlayPendingOAuthAuthHealth(
	bundleID: string,
	serverID: string,
	authHealth: MCPAuthHealth | undefined,
	pendingAuthorizations: MCPOAuthAuthorization[]
): MCPAuthHealth | undefined {
	const pending = pendingAuthorizations.find(
		authorization =>
			authorization.bundleID === bundleID && authorization.serverID === serverID && authorization.authorizationURL
	);

	if (!pending) {
		return authHealth;
	}

	return {
		...authHealth,
		bundleID: pending.bundleID || bundleID,
		serverID: pending.serverID || serverID,
		authMode: MCPHTTPAuthMode.MCPHTTPAuthOAuth,
		state: MCPAuthHealthState.MCPAuthHealthStateAuthorizationPending,
		configured: authHealth?.configured ?? true,
		authorizationPending: true,
		authorizationURL: pending.authorizationURL,
		authorizationExpiresAt: pending.expiresAt,
		lastError: undefined,
	};
}

function isOAuthServerOption(option: MCPComposerServerOption): boolean {
	return option.server.streamableHttp?.authMode === MCPHTTPAuthMode.MCPHTTPAuthOAuth;
}

function hasPendingOAuthHealth(option: MCPComposerServerOption): boolean {
	return (
		option.authHealth?.state === MCPAuthHealthState.MCPAuthHealthStateAuthorizationPending ||
		Boolean(option.authHealth?.authorizationPending)
	);
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
		appResourceUri: tool.app?.resourceUri,
		visibility: tool.app?.visibility,
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
	return tools.filter(t => isMCPToolModelSelectable(t));
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
		mountedRef.current = true;

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

	const refreshAll = useCallback(async () => {
		setLoading(true);
		setError(undefined);

		try {
			const [bundles, pendingAuthorizations] = await Promise.all([
				getAllMCPBundles(undefined, true),
				mcpAPI.listPendingMCPOAuthAuthorizations().catch(() => []),
			]);
			const bundleOptions = await Promise.all(
				bundles.map(async bundle => {
					try {
						const servers = await getAllMCPServers(bundle.id, undefined, undefined, true).catch(() => []);

						const serverOptions = await Promise.all(
							servers.map(async server => {
								const [runtime, authHealth] = await Promise.all([
									mcpAPI.getMCPServerStatus(bundle.id, server.id).catch(() => undefined),
									mcpAPI.getMCPServerAuthHealth(bundle.id, server.id).catch(() => undefined),
								]);
								const authHealthWithPending = overlayPendingOAuthAuthHealth(
									bundle.id,
									server.id,
									authHealth,
									pendingAuthorizations
								);

								return {
									bundle,
									server,
									runtime,
									authHealth: authHealthWithPending,
									tools: [],
									resources: [],
									resourceTemplates: [],
									prompts: [],
									discoveryLoaded: false,
									discoveryLoading: false,
								} satisfies MCPComposerServerOption;
							})
						);

						return serverOptions;
					} catch {
						return [] as MCPComposerServerOption[];
					}
				})
			);

			const nextOptions = bundleOptions.flat();
			if (!mountedRef.current) {
				return;
			}
			optionsRef.current = nextOptions;
			setOptions(nextOptions);
		} catch (err) {
			if (!mountedRef.current) {
				return;
			}
			setError(getErrorMessage(err, 'Failed to load MCP servers.'));
		} finally {
			if (mountedRef.current) {
				setLoading(false);
			}
		}
	}, []);

	useEffect(() => {
		// oxlint-disable-next-line jsreact-hooks/set-state-in-effect
		void refreshAll();
	}, [refreshAll]);

	const loadDiscoveryForServer = useCallback(
		async (bundleID: string, serverID: string, force = false): Promise<MCPDiscoveryLoadResult | undefined> => {
			const key = mcpServerKey(bundleID, serverID);
			const current = optionsRef.current.find(option => option.bundle.id === bundleID && option.server.id === serverID);
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
			patchOption(bundleID, serverID, {
				discoveryLoading: true,
				discoveryError: undefined,
			});

			const promise = (async (): Promise<MCPDiscoveryLoadResult | undefined> => {
				const [toolsResult, resourcesResult, resourceTemplatesResult, promptsResult] = await Promise.allSettled([
					getAllMCPServerTools(bundleID, serverID),
					getAllMCPServerResources(bundleID, serverID),
					getAllMCPServerResourceTemplates(bundleID, serverID),
					getAllMCPServerPrompts(bundleID, serverID),
				]);

				const tools = toolsResult.status === 'fulfilled' ? toolsResult.value : [];
				const resources = resourcesResult.status === 'fulfilled' ? resourcesResult.value : [];
				const resourceTemplates = resourceTemplatesResult.status === 'fulfilled' ? resourceTemplatesResult.value : [];
				const prompts = promptsResult.status === 'fulfilled' ? promptsResult.value : [];

				const discoveryErrors = [
					toolsResult.status === 'rejected' ? getErrorMessage(toolsResult.reason, '') : '',
					resourcesResult.status === 'rejected' ? getErrorMessage(resourcesResult.reason, '') : '',
					resourceTemplatesResult.status === 'rejected' ? getErrorMessage(resourceTemplatesResult.reason, '') : '',
					promptsResult.status === 'rejected' ? getErrorMessage(promptsResult.reason, '') : '',
				].filter((message): message is string => message.trim().length > 0);
				if (!mountedRef.current) {
					return undefined;
				}

				patchOption(bundleID, serverID, {
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
					if (!currentSelection) {
						return prev;
					}
					if (currentSelection.toolExposure !== MCPToolExposure.MCPToolExposureAll) {
						return prev;
					}

					return {
						...prev,
						[key]: {
							...currentSelection,
							selectedTools: modelSelectableTools(tools).map(t => toolToSelection(t)),
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
			})().catch((err: unknown) => {
				if (!mountedRef.current) {
					return undefined;
				}

				patchOption(bundleID, serverID, {
					discoveryLoading: false,
					discoveryLoaded: false,
					discoveryError: getErrorMessage(err, 'Failed to load MCP discovery.'),
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
		async (bundleID: string, serverID: string) => {
			await loadDiscoveryForServer(bundleID, serverID);
		},
		[loadDiscoveryForServer]
	);

	useEffect(() => {
		for (const selection of Object.values(selectedByServerKey)) {
			void ensureDiscoveryLoaded(selection.bundleID, selection.serverID);
		}
	}, [ensureDiscoveryLoaded, options, selectedByServerKey]);

	const refreshServerStatus = useCallback(
		async (bundleID: string, serverID: string) => {
			const [runtime, authHealth, pendingAuthorizations] = await Promise.all([
				mcpAPI.getMCPServerStatus(bundleID, serverID).catch(() => undefined),
				mcpAPI.getMCPServerAuthHealth(bundleID, serverID).catch(() => undefined),
				mcpAPI.listPendingMCPOAuthAuthorizations().catch(() => []),
			]);

			if (!mountedRef.current) {
				return;
			}

			patchOption(bundleID, serverID, {
				runtime,
				authHealth: overlayPendingOAuthAuthHealth(bundleID, serverID, authHealth, pendingAuthorizations),
			});
		},
		[patchOption]
	);

	const refreshServer = useCallback(
		async (bundleID: string, serverID: string) => {
			patchOption(bundleID, serverID, {
				discoveryLoaded: false,
				discoveryLoading: true,
				discoveryError: undefined,
			});

			try {
				const snapshot = await mcpAPI.refreshMCPServer(bundleID, serverID);
				if (snapshot && mountedRef.current) {
					patchOption(bundleID, serverID, { runtime: snapshot });
				}
			} catch (err) {
				if (mountedRef.current) {
					patchOption(bundleID, serverID, {
						discoveryLoading: false,
						discoveryError: getErrorMessage(err, 'Failed to refresh MCP discovery.'),
					});
				}
				await refreshServerStatus(bundleID, serverID).catch(() => undefined);
				return;
			}

			await refreshServerStatus(bundleID, serverID).catch(() => undefined);
			await loadDiscoveryForServer(bundleID, serverID, true).catch(() => undefined);
		},
		[loadDiscoveryForServer, patchOption, refreshServerStatus]
	);

	const connectServer = useCallback(
		async (bundleID: string, serverID: string) => {
			let settled = false;

			const connectPromise = mcpAPI.connectMCPServer(bundleID, serverID).finally(() => {
				settled = true;
			});

			try {
				// oxlint-disable-next-line no-unmodified-loop-condition
				while (!settled) {
					await Promise.race([connectPromise.catch(() => undefined), sleep(1000)]);
					if (!settled) {
						await refreshServerStatus(bundleID, serverID).catch(() => undefined);
					}
				}

				const snapshot = await connectPromise;
				if (snapshot && mountedRef.current) {
					patchOption(bundleID, serverID, {
						runtime: snapshot,
						discoveryLoaded: false,
					});
				}
			} finally {
				await refreshServerStatus(bundleID, serverID).catch(() => undefined);
			}
			await ensureDiscoveryLoaded(bundleID, serverID).catch(() => undefined);
		},
		[ensureDiscoveryLoaded, patchOption, refreshServerStatus]
	);

	const refreshPendingOAuthAuthorizations = useCallback(async () => {
		const pendingAuthorizations = await mcpAPI.listPendingMCPOAuthAuthorizations().catch(() => []);
		const currentOptions = optionsRef.current;
		const stalePending = currentOptions.filter(option => {
			if (!hasPendingOAuthHealth(option)) {
				return false;
			}
			return !pendingAuthorizations.some(
				authorization =>
					authorization.bundleID === option.bundle.id &&
					authorization.serverID === option.server.id &&
					authorization.authorizationURL
			);
		});

		const freshHealthEntries = await Promise.all(
			stalePending.map(async option => {
				const authHealth = await mcpAPI
					.getMCPServerAuthHealth(option.bundle.id, option.server.id)
					.catch(() => undefined);
				return {
					key: optionKey(option),
					authHealth,
				};
			})
		);
		const freshHealthByKey = new Map(freshHealthEntries.map(entry => [entry.key, entry.authHealth] as const));

		if (!mountedRef.current) {
			return;
		}

		setOptions(prev => {
			const next = prev.map(option => {
				const overlaid = overlayPendingOAuthAuthHealth(
					option.bundle.id,
					option.server.id,
					option.authHealth,
					pendingAuthorizations
				);
				const refreshed = freshHealthByKey.get(optionKey(option));
				const authHealth = refreshed ?? overlaid;
				return authHealth === option.authHealth ? option : { ...option, authHealth };
			});
			optionsRef.current = next;
			return next;
		});
	}, []);

	const disconnectServer = useCallback(
		async (bundleID: string, serverID: string) => {
			await mcpAPI.disconnectMCPServer(bundleID, serverID);
			await refreshServerStatus(bundleID, serverID);
		},
		[refreshServerStatus]
	);

	const cancelOAuth = useCallback(
		async (bundleID: string, serverID: string) => {
			await mcpAPI.cancelPendingMCPOAuthAuthorization(bundleID, serverID);
			await refreshServerStatus(bundleID, serverID);
		},
		[refreshServerStatus]
	);

	const openAuthURL = useCallback((url: string) => {
		if (!url) {
			return;
		}
		backendAPI.openURL(url);
	}, []);

	const setServerSelected = useCallback(
		(option: MCPComposerServerOption, selected: boolean) => {
			const key = optionKey(option);

			commitSelectedByServerKey(prev => {
				if (!selected) {
					let next = { ...prev };
					next = omitManyKeys(next, [key]);

					return next;
				}

				if (prev[key]) {
					return prev;
				}

				const next = {
					...prev,
					[key]: {
						bundleID: option.bundle.id,
						serverID: option.server.id,
						snapshotDigest: option.runtime?.snapshotDigest,
						toolExposure: MCPToolExposure.MCPToolExposureAll,
						selectedTools:
							option.tools.length > 0 ? modelSelectableTools(option.tools).map(t => toolToSelection(t)) : [],
						selectedResources: [],
						selectedResourceTemplates: [],
						selectedPrompts: [],
						includeServerInstructions: true,
					},
				};

				return next;
			});
		},
		[commitSelectedByServerKey]
	);

	const setToolExposure = useCallback(
		(bundleID: string, serverID: string, exposure: MCPToolExposure) => {
			const key = mcpServerKey(bundleID, serverID);
			commitSelectedByServerKey(prev => {
				const current = prev[key];
				if (!current) {
					return prev;
				}
				const option = optionsRef.current.find(item => item.bundle.id === bundleID && item.server.id === serverID);
				const next = {
					...prev,
					[key]: {
						...current,
						toolExposure: exposure,
						selectedTools:
							exposure === MCPToolExposure.MCPToolExposureAll
								? modelSelectableTools(option?.tools ?? []).map(t => toolToSelection(t))
								: exposure === MCPToolExposure.MCPToolExposureNone
									? []
									: current.selectedTools,
					},
				};

				return next;
			});
		},
		[commitSelectedByServerKey]
	);

	const setIncludeServerInstructions = useCallback(
		(bundleID: string, serverID: string, include: boolean) => {
			const key = mcpServerKey(bundleID, serverID);
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

			const key = mcpServerKey(tool.bundleID, tool.serverID);
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
			const key = mcpServerKey(resource.bundleID, resource.serverID);

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
			const key = mcpServerKey(template.bundleID, template.serverID);
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
			const key = mcpServerKey(prompt.bundleID, prompt.serverID);
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
		(bundleID: string, serverID: string, uriTemplate: string, argumentName: string, value: string) => {
			const key = mcpServerKey(bundleID, serverID);
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
		(bundleID: string, serverID: string, promptName: string, argumentName: string, value: string) => {
			const key = mcpServerKey(bundleID, serverID);
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
			const key = mcpServerKey(selection.bundleID, selection.serverID);
			const option = optionsRef.current.find(
				item => item.bundle.id === selection.bundleID && item.server.id === selection.serverID
			);

			let selectedTools = selection.selectedTools;

			if (selection.toolExposure === MCPToolExposure.MCPToolExposureAll) {
				const discovery = await loadDiscoveryForServer(selection.bundleID, selection.serverID).catch(() => undefined);
				const tools = discovery?.tools ?? option?.tools ?? [];
				selectedTools = modelSelectableTools(tools).map(t => toolToSelection(t));
			}

			nextSelections[key] = {
				...selection,
				snapshotDigest: option?.runtime?.snapshotDigest ?? selection.snapshotDigest,
				selectedTools: selection.toolExposure === MCPToolExposure.MCPToolExposureNone ? [] : selectedTools,
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
	}, [commitSelectedByServerKey, loadDiscoveryForServer]);

	useEffect(() => {
		const hasOAuthServer = options.some(s => {
			return isOAuthServerOption(s);
		});
		const hasPending = options.some(s => {
			return hasPendingOAuthHealth(s);
		});
		if (!hasOAuthServer && !hasPending) {
			return;
		}

		let cancelled = false;
		const poll = () => {
			if (cancelled) {
				return;
			}
			void refreshPendingOAuthAuthorizations();
		};
		poll();
		const timer = window.setInterval(poll, 2000);
		return () => {
			cancelled = true;
			window.clearInterval(timer);
		};
	}, [options, refreshPendingOAuthAuthorizations]);

	const mcpContext = useMemo(() => mcpSelectionToContext(selectedByServerKey), [selectedByServerKey]);
	const selectedServerCount = Object.keys(selectedByServerKey).length;

	const selectedToolCount = Object.values(selectedByServerKey).reduce((sum, selection) => {
		if (selection.toolExposure === MCPToolExposure.MCPToolExposureNone) {
			return sum;
		}
		if (selection.selectedTools.length > 0) {
			return sum + selection.selectedTools.length;
		}
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
