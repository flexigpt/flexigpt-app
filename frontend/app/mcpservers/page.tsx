import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { FiPlus, FiSettings } from 'react-icons/fi';

import type { ArtifactRef } from '@/spec/artifact';
import type {
	MCPAuthHealth,
	MCPPluginView,
	MCPServerDraft,
	MCPServerRuntimeSnapshot,
	MCPServerView,
	MCPSettings,
	MCPSetupSubmissionValue,
} from '@/spec/mcp';
import { MCPAuthHealthState, MCPHTTPAuthMode, MCPServerStatus } from '@/spec/mcp';

import { mapWithConcurrency } from '@/lib/async_utils';
import { getErrorMessage } from '@/lib/error_utils';

import { backendAPI, mcpManagementAPI } from '@/apis/baseapi';
import { getAuthMode, isServerOperational, requireMCPRuntimeServerID } from '@/apis/mcp_management';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';
import { Loader } from '@/components/loader';
import { ManagementPageContent } from '@/components/managementui/management_page_content';
import { ManagementPageHeader } from '@/components/managementui/management_page_header';
import { ManagementPluginCreateModal } from '@/components/managementui/management_plugin_create_modal';
import { ManagementResourceError } from '@/components/managementui/management_resource_error';
import { PageFrame } from '@/components/page_frame';

import { MCPOAuthAuthorizationModal } from '@/mcpservers/mcp_oauth_authorization_modal';
import { MCPPluginCard } from '@/mcpservers/mcp_plugin_card';
import { MCPSettingsModal } from '@/mcpservers/mcp_settings_modal';

interface PluginData {
	plugin: MCPPluginView;
	servers: MCPServerView[];
	runtimeByArtifactID: Record<string, MCPServerRuntimeSnapshot | undefined>;
	authHealthByArtifactID: Record<string, MCPAuthHealth | undefined>;
	readErrorsByArtifactID: Record<string, { runtime?: string; auth?: string } | undefined>;
	serverLoadError?: string;
	isLoadingServers: boolean;
	serversLoaded: boolean;
}

interface OAuthTarget {
	server: ArtifactRef;
	openAutomatically: boolean;
}

const STATUS_READ_CONCURRENCY = 4;
const PLUGIN_PREFETCH_CONCURRENCY = 3;

function artifactKey(ref: ArtifactRef): string {
	return `${ref.rootID}:${ref.artifactID}`;
}

function getMatchingAuthHealth(server: MCPServerView, value: MCPAuthHealth | undefined): MCPAuthHealth | undefined {
	if (!value) {
		return undefined;
	}

	if (!server.runtimeServerID || value.server !== server.runtimeServerID) {
		return undefined;
	}

	return value;
}

function getMatchingRuntimeSnapshot(
	server: MCPServerView,
	value: MCPServerRuntimeSnapshot | undefined
): MCPServerRuntimeSnapshot | undefined {
	if (!value) {
		return undefined;
	}

	if (!server.runtimeServerID || value.server !== server.runtimeServerID) {
		return undefined;
	}

	return value;
}

// oxlint-disable-next-line no-restricted-exports
export default function MCPServersPage() {
	const [plugins, setPlugins] = useState<PluginData[]>([]);
	const [settings, setSettings] = useState<MCPSettings>();
	const [isInitialLoading, setIsInitialLoading] = useState(true);
	const [isRefreshing, setIsRefreshing] = useState(false);
	const [pageLoadError, setPageLoadError] = useState<unknown>();
	const [warnings, setWarnings] = useState<string[]>([]);

	const [isAddPluginOpen, setIsAddPluginOpen] = useState(false);
	const [isSettingsOpen, setIsSettingsOpen] = useState(false);
	const [pluginToDelete, setPluginToDelete] = useState<MCPPluginView | null>(null);
	const [isDeletingPlugin, setIsDeletingPlugin] = useState(false);
	const [oauthTarget, setOAuthTarget] = useState<OAuthTarget | null>(null);
	const [alertMessage, setAlertMessage] = useState('');

	const mountedRef = useRef(false);
	const loadIDRef = useRef(0);
	const loadedOnceRef = useRef(false);
	const pluginsRef = useRef<PluginData[]>([]);
	const openedAuthorizationURLsRef = useRef(new Set<string>());
	const pluginLoadsRef = useRef(new Map<string, Promise<void>>());

	useEffect(() => {
		pluginsRef.current = plugins;
	}, [plugins]);

	useEffect(() => {
		mountedRef.current = true;

		return () => {
			mountedRef.current = false;
			loadIDRef.current += 1;
			// oxlint-disable-next-line react-hooks/exhaustive-deps
			pluginLoadsRef.current.clear();
		};
	}, []);

	const readServerStatus = useCallback(
		async (
			servers: MCPServerView[]
		): Promise<{
			runtimeByArtifactID: Record<string, MCPServerRuntimeSnapshot | undefined>;
			authHealthByArtifactID: Record<string, MCPAuthHealth | undefined>;
			readErrorsByArtifactID: Record<string, { runtime?: string; auth?: string } | undefined>;
		}> => {
			const entries = await mapWithConcurrency(servers, STATUS_READ_CONCURRENCY, async server => {
				const key = server.ref.artifactID;

				if (!server.enabled || !isServerOperational(server)) {
					return {
						key,
						runtime: undefined,
						auth: undefined,
						errors: undefined,
					};
				}

				try {
					const refreshed = await mcpManagementAPI.getMCPServer(server.ref, server.plugin);

					return {
						key,
						runtime: getMatchingRuntimeSnapshot(server, refreshed.runtime),
						auth: getMatchingAuthHealth(server, refreshed.authHealth),
						errors: undefined,
					};
				} catch (error) {
					const message = getErrorMessage(error, 'MCP server state could not be loaded.');

					return {
						key,
						runtime: undefined,
						auth: undefined,
						errors: {
							runtime: message,
							auth: message,
						},
					};
				}
			});

			return {
				runtimeByArtifactID: Object.fromEntries(entries.map(entry => [entry.key, entry.runtime])),
				authHealthByArtifactID: Object.fromEntries(entries.map(entry => [entry.key, entry.auth])),
				readErrorsByArtifactID: Object.fromEntries(entries.map(entry => [entry.key, entry.errors])),
			};
		},
		[]
	);

	const applyStatus = useCallback((pluginRef: ArtifactRef, statuses: Awaited<ReturnType<typeof readServerStatus>>) => {
		setPlugins(previous =>
			previous.map(item => {
				if (item.plugin.ref.rootID !== pluginRef.rootID || item.plugin.ref.artifactID !== pluginRef.artifactID) {
					return item;
				}

				return {
					...item,
					runtimeByArtifactID: {
						...item.runtimeByArtifactID,
						...statuses.runtimeByArtifactID,
					},
					authHealthByArtifactID: {
						...item.authHealthByArtifactID,
						...statuses.authHealthByArtifactID,
					},
					readErrorsByArtifactID: {
						...item.readErrorsByArtifactID,
						...statuses.readErrorsByArtifactID,
					},
				};
			})
		);
	}, []);

	const loadPluginServerList = useCallback(async (plugin: MCPPluginView): Promise<PluginData> => {
		const servers = await mcpManagementAPI.listMCPServers(plugin);

		return {
			plugin,
			servers,
			runtimeByArtifactID: Object.fromEntries(servers.map(server => [server.ref.artifactID, server.runtime])),
			authHealthByArtifactID: Object.fromEntries(servers.map(server => [server.ref.artifactID, server.authHealth])),
			readErrorsByArtifactID: {},
			serversLoaded: true,
			isLoadingServers: false,
		};
	}, []);

	const fetchAll = useCallback(async () => {
		const requestID = loadIDRef.current + 1;
		loadIDRef.current = requestID;

		if (loadedOnceRef.current) {
			setIsRefreshing(true);
		} else {
			setIsInitialLoading(true);
		}

		setPageLoadError(undefined);

		try {
			const [pluginResult, settingsResult] = await Promise.allSettled([
				mcpManagementAPI.listMCPPlugins(),
				mcpManagementAPI.getMCPSettings(),
			]);

			if (pluginResult.status === 'rejected') {
				throw pluginResult.reason;
			}

			if (!mountedRef.current || loadIDRef.current !== requestID) {
				return;
			}

			const incomingPlugins = pluginResult.value ?? [];
			const nextPlugins = incomingPlugins.map(plugin => {
				const existing = pluginsRef.current.find(
					item => item.plugin.ref.rootID === plugin.ref.rootID && item.plugin.ref.artifactID === plugin.ref.artifactID
				);

				const sameRevision = existing?.plugin.plugin.revision === plugin.plugin.revision;

				if (existing && sameRevision && existing.serversLoaded) {
					return {
						...existing,
						plugin,
						isLoadingServers: false,
					};
				}

				return {
					plugin,
					servers: [],
					runtimeByArtifactID: {},
					authHealthByArtifactID: {},
					readErrorsByArtifactID: {},
					serversLoaded: false,
					isLoadingServers: false,
				} satisfies PluginData;
			});

			pluginsRef.current = nextPlugins;
			setPlugins(nextPlugins);

			setWarnings(
				settingsResult.status === 'rejected'
					? [getErrorMessage(settingsResult.reason, 'MCP OAuth settings could not be loaded.')]
					: []
			);

			if (settingsResult.status === 'fulfilled') {
				setSettings(settingsResult.value);
			}

			loadedOnceRef.current = true;
		} catch (error) {
			if (!mountedRef.current || loadIDRef.current !== requestID) {
				return;
			}

			setPageLoadError(error);
			setAlertMessage(getErrorMessage(error, 'Failed to load MCP Plugins.'));
		} finally {
			if (mountedRef.current && loadIDRef.current === requestID) {
				setIsInitialLoading(false);
				setIsRefreshing(false);
			}
		}
	}, []);

	useEffect(() => {
		void fetchAll();
	}, [fetchAll]);

	const loadPluginServers = useCallback(
		(pluginRef: ArtifactRef, refreshStatus = true): Promise<void> => {
			const key = artifactKey(pluginRef);
			const existingLoad = pluginLoadsRef.current.get(key);

			if (existingLoad) {
				return existingLoad;
			}

			const load = (async () => {
				const current = pluginsRef.current.find(
					item => item.plugin.ref.rootID === pluginRef.rootID && item.plugin.ref.artifactID === pluginRef.artifactID
				);

				if (!current) {
					throw new Error('MCP Plugin is no longer available.');
				}

				setPlugins(previous =>
					previous.map(item =>
						item.plugin.ref.rootID === pluginRef.rootID && item.plugin.ref.artifactID === pluginRef.artifactID
							? {
									...item,
									isLoadingServers: true,
									serverLoadError: undefined,
								}
							: item
					)
				);

				try {
					const plugin = await mcpManagementAPI.getMCPPlugin(pluginRef);
					const loaded = await loadPluginServerList(plugin);

					if (!mountedRef.current) {
						return;
					}

					setPlugins(previous =>
						previous.map(item =>
							item.plugin.ref.rootID === pluginRef.rootID && item.plugin.ref.artifactID === pluginRef.artifactID
								? loaded
								: item
						)
					);

					if (!refreshStatus || loaded.servers.length === 0) {
						return;
					}

					const statuses = await readServerStatus(loaded.servers);

					if (mountedRef.current) {
						applyStatus(pluginRef, statuses);
					}
				} catch (error) {
					if (!mountedRef.current) {
						return;
					}

					setPlugins(previous =>
						previous.map(item =>
							item.plugin.ref.rootID === pluginRef.rootID && item.plugin.ref.artifactID === pluginRef.artifactID
								? {
										...item,
										servers: [],
										runtimeByArtifactID: {},
										authHealthByArtifactID: {},
										readErrorsByArtifactID: {},
										serversLoaded: false,
										isLoadingServers: false,
										serverLoadError: getErrorMessage(error, 'Failed to load MCP servers for this Plugin.'),
									}
								: item
						)
					);

					throw error;
				}
			})();

			pluginLoadsRef.current.set(key, load);

			void load.finally(() => {
				if (pluginLoadsRef.current.get(key) === load) {
					pluginLoadsRef.current.delete(key);
				}
			});

			return load;
		},
		[applyStatus, loadPluginServerList, readServerStatus]
	);

	useEffect(() => {
		if (isInitialLoading || isRefreshing || pageLoadError) {
			return;
		}

		const pending = plugins
			.filter(plugin => !plugin.serversLoaded && !plugin.isLoadingServers && !plugin.serverLoadError)
			.map(plugin => plugin.plugin.ref);

		if (pending.length === 0) {
			return;
		}

		const timer = window.setTimeout(() => {
			void mapWithConcurrency(pending, PLUGIN_PREFETCH_CONCURRENCY, ref => loadPluginServers(ref, false)).catch(
				() => undefined
			);
		}, 0);

		return () => {
			window.clearTimeout(timer);
		};
	}, [plugins, isInitialLoading, isRefreshing, loadPluginServers, pageLoadError]);

	const refreshPlugin = useCallback(
		async (pluginRef: ArtifactRef) => {
			await loadPluginServers(pluginRef, true);
		},
		[loadPluginServers]
	);

	const refreshSingleServer = useCallback(
		async (server: MCPServerView) => {
			const statuses = await readServerStatus([server]);
			applyStatus(server.plugin, statuses);
		},
		[applyStatus, readServerStatus]
	);

	const markServerConnecting = useCallback((server: MCPServerView) => {
		const runtimeServerID = server.runtimeServerID;
		if (!runtimeServerID) {
			return;
		}

		setPlugins(previous =>
			previous.map(item => {
				if (
					item.plugin.ref.rootID !== server.plugin.rootID ||
					item.plugin.ref.artifactID !== server.plugin.artifactID
				) {
					return item;
				}

				const current = item.runtimeByArtifactID[server.ref.artifactID];

				return {
					...item,
					runtimeByArtifactID: {
						...item.runtimeByArtifactID,
						[server.ref.artifactID]: {
							...current,
							server: runtimeServerID,
							catalog: current?.catalog ?? '',
							status: MCPServerStatus.Connecting,
							lastError: undefined,
							toolCount: current?.toolCount ?? 0,
							resourceCount: current?.resourceCount ?? 0,
							resourceTemplateCount: current?.resourceTemplateCount ?? 0,
							promptCount: current?.promptCount ?? 0,
							generation: current?.generation ?? 0,
						},
					},
				};
			})
		);
	}, []);

	const applyRuntimeSnapshot = useCallback((server: MCPServerView, snapshot: MCPServerRuntimeSnapshot) => {
		const matching = getMatchingRuntimeSnapshot(server, snapshot);
		if (!matching) {
			throw new Error('The MCP backend returned a runtime snapshot for another server.');
		}

		setPlugins(previous =>
			previous.map(item =>
				item.plugin.ref.rootID === server.plugin.rootID && item.plugin.ref.artifactID === server.plugin.artifactID
					? {
							...item,
							runtimeByArtifactID: {
								...item.runtimeByArtifactID,
								[server.ref.artifactID]: matching,
							},
						}
					: item
			)
		);
	}, []);

	const handleConnectServer = useCallback(
		async (server: MCPServerView) => {
			if (getAuthMode(server) === MCPHTTPAuthMode.OAuth) {
				setOAuthTarget({
					server: server.ref,
					openAutomatically: true,
				});
			}

			const runtimeServerID = requireMCPRuntimeServerID(server);
			markServerConnecting(server);

			try {
				const snapshot = await mcpManagementAPI.connectMCPServer(runtimeServerID);

				applyRuntimeSnapshot(server, snapshot);

				if (snapshot.status === MCPServerStatus.Error) {
					throw new Error(snapshot.lastError || 'The MCP server connection failed.');
				}
			} catch (error) {
				await refreshSingleServer(server).catch(() => undefined);
				throw error;
			}
		},
		[applyRuntimeSnapshot, markServerConnecting, refreshSingleServer]
	);

	const connectionPollKey = useMemo(
		() =>
			JSON.stringify(
				plugins.flatMap(plugin =>
					plugin.servers
						.filter(server => {
							const runtime = plugin.runtimeByArtifactID[server.ref.artifactID];
							const auth = plugin.authHealthByArtifactID[server.ref.artifactID];

							return (
								runtime?.status === MCPServerStatus.Connecting ||
								auth?.state === MCPAuthHealthState.AuthorizationPending
							);
						})
						.map(server => server.ref)
				)
			),
		[plugins]
	);

	useEffect(() => {
		const refs = JSON.parse(connectionPollKey) as ArtifactRef[];

		if (refs.length === 0) {
			return;
		}

		let cancelled = false;
		let timer: number | undefined;

		const poll = async () => {
			try {
				await mapWithConcurrency(refs, STATUS_READ_CONCURRENCY, async ref => {
					const plugin = pluginsRef.current.find(item =>
						item.servers.some(server => artifactKey(server.ref) === artifactKey(ref))
					);
					const server = plugin?.servers.find(item => artifactKey(item.ref) === artifactKey(ref));

					if (!cancelled && server) {
						await refreshSingleServer(server);
					}
				});
			} catch {
				// Status refresh failure is reflected by the next aggregate read.
			} finally {
				if (!cancelled) {
					timer = window.setTimeout(() => void poll(), document.hidden ? 4000 : 1000);
				}
			}
		};

		void poll();

		return () => {
			cancelled = true;
			if (timer !== undefined) {
				window.clearTimeout(timer);
			}
		};
	}, [connectionPollKey, refreshSingleServer]);

	const handleSavePlugin = useCallback(
		async (logicalName: string, displayName: string, description?: string) => {
			await mcpManagementAPI.createMCPPlugin(logicalName, displayName, description);
			await fetchAll();
		},
		[fetchAll]
	);

	const handleSaveServer = useCallback(
		async (plugin: MCPPluginView, server: MCPServerView | undefined, draft: MCPServerDraft) => {
			await mcpManagementAPI.saveMCPServer(plugin, server, draft);
			await refreshPlugin(plugin.ref);
		},
		[refreshPlugin]
	);

	const handleSaveSetup = useCallback(
		async (server: MCPServerView, values: Record<string, MCPSetupSubmissionValue>, reset: boolean) => {
			await mcpManagementAPI.applyMCPServerSetup(server, values, reset);
			await refreshPlugin(server.plugin);
		},
		[refreshPlugin]
	);

	const patchPluginEnabled = useCallback((pluginRef: ArtifactRef, enabled: boolean) => {
		setPlugins(previous =>
			previous.map(item => {
				if (item.plugin.ref.rootID !== pluginRef.rootID || item.plugin.ref.artifactID !== pluginRef.artifactID) {
					return item;
				}

				return {
					...item,
					plugin: {
						...item.plugin,
						enabled,
						plugin: {
							...item.plugin.plugin,
							enabled,
						},
					},
				};
			})
		);
	}, []);

	const openAuthorizationURL = useCallback((server: ArtifactRef, url: string, once: boolean) => {
		const key = `${artifactKey(server)}:${url}`;

		if (once && openedAuthorizationURLsRef.current.has(key)) {
			return;
		}

		openedAuthorizationURLsRef.current.add(key);

		try {
			backendAPI.openURL(url);
		} catch {
			openedAuthorizationURLsRef.current.delete(key);
			if (mountedRef.current) {
				setAlertMessage('The OAuth authorization page could not be opened.');
			}
		}
	}, []);

	const selectedOAuth = useMemo(() => {
		if (!oauthTarget) {
			return undefined;
		}

		for (const plugin of plugins) {
			const server = plugin.servers.find(item => artifactKey(item.ref) === artifactKey(oauthTarget.server));

			if (server) {
				return {
					server,
					authHealth: plugin.authHealthByArtifactID[server.ref.artifactID],
					runtime: plugin.runtimeByArtifactID[server.ref.artifactID],
				};
			}
		}

		return undefined;
	}, [plugins, oauthTarget]);

	useEffect(() => {
		if (!oauthTarget?.openAutomatically || !selectedOAuth) {
			return;
		}

		const url = selectedOAuth.authHealth?.authorizationURL?.trim();
		if (!url) {
			return;
		}

		openAuthorizationURL(selectedOAuth.server.ref, url, true);
	}, [oauthTarget, openAuthorizationURL, selectedOAuth]);

	useEffect(() => {
		if (!oauthTarget || !selectedOAuth || selectedOAuth.runtime?.status !== MCPServerStatus.Ready) {
			return;
		}

		const targetKey = artifactKey(oauthTarget.server);
		const timer = window.setTimeout(() => {
			setOAuthTarget(current => (current && artifactKey(current.server) === targetKey ? null : current));
		}, 700);

		return () => {
			window.clearTimeout(timer);
		};
	}, [oauthTarget, selectedOAuth]);

	return (
		<PageFrame>
			<div className="flex size-full flex-col items-center overflow-hidden">
				<ManagementPageHeader
					title="MCP Servers"
					description="Organize MCP servers in Plugins and configure their connections, credentials, runtime state, and tool policies."
					width="wide"
					leadingActions={
						<button
							type="button"
							className="btn btn-ghost rounded-xl"
							onClick={() => {
								setIsSettingsOpen(true);
							}}
						>
							<FiSettings size={18} />
							<span className="hidden sm:inline">OAuth Settings</span>
						</button>
					}
					actions={
						<button
							type="button"
							className="btn btn-ghost rounded-xl"
							onClick={() => {
								setIsAddPluginOpen(true);
							}}
						>
							<FiPlus size={18} />
							<span>Add Plugin</span>
						</button>
					}
				/>

				<ManagementPageContent width="wide">
					{isInitialLoading ? <Loader text="Loading MCP Plugins..." /> : null}

					{pageLoadError ? (
						<ManagementResourceError
							title="MCP servers could not be loaded"
							error={pageLoadError}
							isRetrying={isRefreshing}
							onRetry={fetchAll}
						/>
					) : null}

					{warnings.map(warning => (
						<div key={warning} className="alert alert-warning rounded-2xl text-sm">
							<span>{warning}</span>
						</div>
					))}

					{!isInitialLoading && plugins.length === 0 ? (
						<p className="mt-8 text-center text-sm">No MCP Plugins configured yet.</p>
					) : null}

					{plugins.map(pluginData => (
						<MCPPluginCard
							key={`${pluginData.plugin.ref.rootID}:${pluginData.plugin.ref.artifactID}`}
							plugin={pluginData.plugin}
							servers={pluginData.servers}
							existingLogicalNames={pluginData.servers.map(server => server.logicalName)}
							runtimeByArtifactID={pluginData.runtimeByArtifactID}
							authHealthByArtifactID={pluginData.authHealthByArtifactID}
							readErrorsByArtifactID={pluginData.readErrorsByArtifactID}
							serverLoadError={pluginData.serverLoadError}
							isLoadingServers={pluginData.isLoadingServers}
							serversLoaded={pluginData.serversLoaded}
							onLoadServers={() => loadPluginServers(pluginData.plugin.ref)}
							onRefreshServers={() => refreshPlugin(pluginData.plugin.ref)}
							onTogglePluginEnabled={async (plugin, enabled) => {
								const previous = plugin.enabled;
								patchPluginEnabled(plugin.ref, enabled);

								try {
									await mcpManagementAPI.setMCPPluginEnabled(plugin, enabled);
								} catch (error) {
									patchPluginEnabled(plugin.ref, previous);
									throw error;
								}

								if (pluginData.serversLoaded) {
									await refreshPlugin(plugin.ref);
								}
							}}
							onSaveServer={handleSaveServer}
							onSaveSetup={handleSaveSetup}
							onDeleteServer={async (plugin, server) => {
								await mcpManagementAPI.deleteMCPServer(plugin, server);
								await refreshPlugin(plugin.ref);
							}}
							onConnectServer={handleConnectServer}
							onDisconnectServer={async server => {
								await mcpManagementAPI.disconnectMCPServer(requireMCPRuntimeServerID(server));
								await refreshSingleServer(server);
							}}
							onRefreshServer={async server => {
								await mcpManagementAPI.refreshMCPServer(requireMCPRuntimeServerID(server));
								await refreshSingleServer(server);
							}}
							onCancelOAuth={async server => {
								await mcpManagementAPI.cancelMCPServerAuthorization(requireMCPRuntimeServerID(server));
								await refreshSingleServer(server);
							}}
							onRequestOAuthAuthorization={server => {
								setOAuthTarget({
									server: server.ref,
									openAutomatically: false,
								});
							}}
							onDeletePluginRequested={plugin => {
								setPluginToDelete(plugin);
							}}
						/>
					))}
				</ManagementPageContent>

				<DeleteConfirmationModal
					isOpen={pluginToDelete !== null}
					onClose={() => {
						if (!isDeletingPlugin) {
							setPluginToDelete(null);
						}
					}}
					onConfirm={async () => {
						if (!pluginToDelete || isDeletingPlugin) {
							return;
						}

						const current = pluginsRef.current.find(
							item =>
								item.plugin.ref.rootID === pluginToDelete.ref.rootID &&
								item.plugin.ref.artifactID === pluginToDelete.ref.artifactID
						);

						if (!current?.serversLoaded || current.serverLoadError || current.servers.length > 0) {
							setAlertMessage('Load the Plugin and remove all currently available MCP servers before deleting it.');
							return;
						}

						setIsDeletingPlugin(true);

						try {
							await mcpManagementAPI.deleteMCPPlugin(pluginToDelete);
							setPluginToDelete(null);
							await fetchAll();
						} catch (error) {
							setAlertMessage(getErrorMessage(error, 'Failed to delete MCP Plugin.'));
						} finally {
							setIsDeletingPlugin(false);
						}
					}}
					title="Delete MCP Plugin"
					message={`Delete empty MCP Plugin "${pluginToDelete?.displayName ?? ''}"? Remove all servers first.`}
					confirmButtonText={isDeletingPlugin ? 'Deleting...' : 'Delete'}
				/>

				<ManagementPluginCreateModal
					isOpen={isAddPluginOpen}
					title="Add MCP Plugin"
					entityLabel="MCP Plugin"
					onClose={() => {
						setIsAddPluginOpen(false);
					}}
					onSubmit={handleSavePlugin}
					existingSlugs={plugins.map(plugin => plugin.plugin.logicalName)}
					failureMessage="Failed to create MCP Plugin."
				/>

				<MCPSettingsModal
					isOpen={isSettingsOpen}
					initialListenAddr={settings?.oauthLoopbackListenAddr}
					onClose={() => {
						setIsSettingsOpen(false);
					}}
					onSubmit={async oauthLoopbackListenAddr => {
						const current = settings ?? (await mcpManagementAPI.getMCPSettings());

						const next = await mcpManagementAPI.saveMCPSettings(current.revision, {
							oauthLoopbackListenAddr: oauthLoopbackListenAddr || undefined,
						});

						setSettings(next);
					}}
				/>

				<MCPOAuthAuthorizationModal
					isOpen={Boolean(selectedOAuth)}
					server={selectedOAuth?.server ?? null}
					authHealth={selectedOAuth?.authHealth}
					isConnecting={selectedOAuth?.runtime?.status === MCPServerStatus.Connecting}
					isReady={selectedOAuth?.runtime?.status === MCPServerStatus.Ready}
					runtimeError={selectedOAuth?.runtime?.lastError}
					onClose={() => {
						setOAuthTarget(null);
					}}
					onOpenURL={url => {
						if (selectedOAuth) {
							openAuthorizationURL(selectedOAuth.server.ref, url, false);
						}
					}}
					onCancel={async () => {
						if (!selectedOAuth) {
							return;
						}

						await mcpManagementAPI.cancelMCPServerAuthorization(requireMCPRuntimeServerID(selectedOAuth.server));
						await refreshSingleServer(selectedOAuth.server);
						setOAuthTarget(null);
					}}
				/>

				<ActionDeniedAlertModal
					isOpen={Boolean(alertMessage)}
					message={alertMessage}
					onClose={() => {
						setAlertMessage('');
					}}
				/>
			</div>
		</PageFrame>
	);
}
