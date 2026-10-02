import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { FiPlus, FiSettings } from 'react-icons/fi';

import type { ArtifactRef } from '@/spec/artifact';
import type {
	MCPAuthHealth,
	MCPBundleView,
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
import { ManagementBundleCreateModal } from '@/components/managementui/management_bundle_create_modal';
import { ManagementPageContent } from '@/components/managementui/management_page_content';
import { ManagementPageHeader } from '@/components/managementui/management_page_header';
import { ManagementResourceError } from '@/components/managementui/management_resource_error';
import { PageFrame } from '@/components/page_frame';

import { MCPBundleCard } from '@/mcpservers/mcp_bundle_card';
import { MCPOAuthAuthorizationModal } from '@/mcpservers/mcp_oauth_authorization_modal';
import { MCPSettingsModal } from '@/mcpservers/mcp_settings_modal';

interface BundleData {
	bundle: MCPBundleView;
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
const BUNDLE_PREFETCH_CONCURRENCY = 3;

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
	const [bundles, setBundles] = useState<BundleData[]>([]);
	const [settings, setSettings] = useState<MCPSettings>();
	const [isInitialLoading, setIsInitialLoading] = useState(true);
	const [isRefreshing, setIsRefreshing] = useState(false);
	const [pageLoadError, setPageLoadError] = useState<unknown>();
	const [warnings, setWarnings] = useState<string[]>([]);

	const [isAddBundleOpen, setIsAddBundleOpen] = useState(false);
	const [isSettingsOpen, setIsSettingsOpen] = useState(false);
	const [bundleToDelete, setBundleToDelete] = useState<MCPBundleView | null>(null);
	const [isDeletingBundle, setIsDeletingBundle] = useState(false);
	const [oauthTarget, setOAuthTarget] = useState<OAuthTarget | null>(null);
	const [alertMessage, setAlertMessage] = useState('');

	const mountedRef = useRef(false);
	const loadIDRef = useRef(0);
	const loadedOnceRef = useRef(false);
	const bundlesRef = useRef<BundleData[]>([]);
	const openedAuthorizationURLsRef = useRef(new Set<string>());
	const bundleLoadsRef = useRef(new Map<string, Promise<void>>());

	useEffect(() => {
		bundlesRef.current = bundles;
	}, [bundles]);

	useEffect(() => {
		mountedRef.current = true;

		return () => {
			mountedRef.current = false;
			loadIDRef.current += 1;
			// oxlint-disable-next-line react-hooks/exhaustive-deps
			bundleLoadsRef.current.clear();
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
					const refreshed = await mcpManagementAPI.getMCPServer(server.ref, server.bundle);

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

	const applyStatus = useCallback((bundleRef: ArtifactRef, statuses: Awaited<ReturnType<typeof readServerStatus>>) => {
		setBundles(previous =>
			previous.map(item => {
				if (item.bundle.ref.rootID !== bundleRef.rootID || item.bundle.ref.artifactID !== bundleRef.artifactID) {
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

	const loadBundleServerList = useCallback(async (bundle: MCPBundleView): Promise<BundleData> => {
		const servers = await mcpManagementAPI.listMCPServers(bundle);

		return {
			bundle,
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
			const [bundleResult, settingsResult] = await Promise.allSettled([
				mcpManagementAPI.listMCPBundles(),
				mcpManagementAPI.getMCPSettings(),
			]);

			if (bundleResult.status === 'rejected') {
				throw bundleResult.reason;
			}

			if (!mountedRef.current || loadIDRef.current !== requestID) {
				return;
			}

			const incomingBundles = bundleResult.value ?? [];
			const nextBundles = incomingBundles.map(bundle => {
				const existing = bundlesRef.current.find(
					item => item.bundle.ref.rootID === bundle.ref.rootID && item.bundle.ref.artifactID === bundle.ref.artifactID
				);

				const sameRevision = existing?.bundle.collection.revision === bundle.collection.revision;

				if (existing && sameRevision && existing.serversLoaded) {
					return {
						...existing,
						bundle,
						isLoadingServers: false,
					};
				}

				return {
					bundle,
					servers: [],
					runtimeByArtifactID: {},
					authHealthByArtifactID: {},
					readErrorsByArtifactID: {},
					serversLoaded: false,
					isLoadingServers: false,
				} satisfies BundleData;
			});

			bundlesRef.current = nextBundles;
			setBundles(nextBundles);

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
			setAlertMessage(getErrorMessage(error, 'Failed to load MCP Collections.'));
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

	const loadBundleServers = useCallback(
		(bundleRef: ArtifactRef, refreshStatus = true): Promise<void> => {
			const key = artifactKey(bundleRef);
			const existingLoad = bundleLoadsRef.current.get(key);

			if (existingLoad) {
				return existingLoad;
			}

			const load = (async () => {
				const current = bundlesRef.current.find(
					item => item.bundle.ref.rootID === bundleRef.rootID && item.bundle.ref.artifactID === bundleRef.artifactID
				);

				if (!current) {
					throw new Error('MCP Collection is no longer available.');
				}

				setBundles(previous =>
					previous.map(item =>
						item.bundle.ref.rootID === bundleRef.rootID && item.bundle.ref.artifactID === bundleRef.artifactID
							? {
									...item,
									isLoadingServers: true,
									serverLoadError: undefined,
								}
							: item
					)
				);

				try {
					const bundle = await mcpManagementAPI.getMCPBundle(bundleRef);
					const loaded = await loadBundleServerList(bundle);

					if (!mountedRef.current) {
						return;
					}

					setBundles(previous =>
						previous.map(item =>
							item.bundle.ref.rootID === bundleRef.rootID && item.bundle.ref.artifactID === bundleRef.artifactID
								? loaded
								: item
						)
					);

					if (!refreshStatus || loaded.servers.length === 0) {
						return;
					}

					const statuses = await readServerStatus(loaded.servers);

					if (mountedRef.current) {
						applyStatus(bundleRef, statuses);
					}
				} catch (error) {
					if (!mountedRef.current) {
						return;
					}

					setBundles(previous =>
						previous.map(item =>
							item.bundle.ref.rootID === bundleRef.rootID && item.bundle.ref.artifactID === bundleRef.artifactID
								? {
										...item,
										servers: [],
										runtimeByArtifactID: {},
										authHealthByArtifactID: {},
										readErrorsByArtifactID: {},
										serversLoaded: false,
										isLoadingServers: false,
										serverLoadError: getErrorMessage(error, 'Failed to load MCP servers for this Collection.'),
									}
								: item
						)
					);

					throw error;
				}
			})();

			bundleLoadsRef.current.set(key, load);

			void load.finally(() => {
				if (bundleLoadsRef.current.get(key) === load) {
					bundleLoadsRef.current.delete(key);
				}
			});

			return load;
		},
		[applyStatus, loadBundleServerList, readServerStatus]
	);

	useEffect(() => {
		if (isInitialLoading || isRefreshing || pageLoadError) {
			return;
		}

		const pending = bundles
			.filter(bundle => !bundle.serversLoaded && !bundle.isLoadingServers && !bundle.serverLoadError)
			.map(bundle => bundle.bundle.ref);

		if (pending.length === 0) {
			return;
		}

		const timer = window.setTimeout(() => {
			void mapWithConcurrency(pending, BUNDLE_PREFETCH_CONCURRENCY, ref => loadBundleServers(ref, false)).catch(
				() => undefined
			);
		}, 0);

		return () => {
			window.clearTimeout(timer);
		};
	}, [bundles, isInitialLoading, isRefreshing, loadBundleServers, pageLoadError]);

	const refreshBundle = useCallback(
		async (bundleRef: ArtifactRef) => {
			await loadBundleServers(bundleRef, true);
		},
		[loadBundleServers]
	);

	const refreshSingleServer = useCallback(
		async (server: MCPServerView) => {
			const statuses = await readServerStatus([server]);
			applyStatus(server.bundle, statuses);
		},
		[applyStatus, readServerStatus]
	);

	const markServerConnecting = useCallback((server: MCPServerView) => {
		const runtimeServerID = server.runtimeServerID;
		if (!runtimeServerID) {
			return;
		}

		setBundles(previous =>
			previous.map(item => {
				if (
					item.bundle.ref.rootID !== server.bundle.rootID ||
					item.bundle.ref.artifactID !== server.bundle.artifactID
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

		setBundles(previous =>
			previous.map(item =>
				item.bundle.ref.rootID === server.bundle.rootID && item.bundle.ref.artifactID === server.bundle.artifactID
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
				bundles.flatMap(bundle =>
					bundle.servers
						.filter(server => {
							const runtime = bundle.runtimeByArtifactID[server.ref.artifactID];
							const auth = bundle.authHealthByArtifactID[server.ref.artifactID];

							return (
								runtime?.status === MCPServerStatus.Connecting ||
								auth?.state === MCPAuthHealthState.AuthorizationPending
							);
						})
						.map(server => server.ref)
				)
			),
		[bundles]
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
					const bundle = bundlesRef.current.find(item =>
						item.servers.some(server => artifactKey(server.ref) === artifactKey(ref))
					);
					const server = bundle?.servers.find(item => artifactKey(item.ref) === artifactKey(ref));

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

	const handleSaveBundle = useCallback(
		async (logicalName: string, displayName: string, description?: string) => {
			await mcpManagementAPI.createMCPBundle(logicalName, displayName, description);
			await fetchAll();
		},
		[fetchAll]
	);

	const handleSaveServer = useCallback(
		async (bundle: MCPBundleView, server: MCPServerView | undefined, draft: MCPServerDraft) => {
			await mcpManagementAPI.saveMCPServer(bundle, server, draft);
			await refreshBundle(bundle.ref);
		},
		[refreshBundle]
	);

	const handleSaveSetup = useCallback(
		async (server: MCPServerView, values: Record<string, MCPSetupSubmissionValue>, reset: boolean) => {
			await mcpManagementAPI.applyMCPServerSetup(server, values, reset);
			await refreshBundle(server.bundle);
		},
		[refreshBundle]
	);

	const patchBundleEnabled = useCallback((bundleRef: ArtifactRef, enabled: boolean) => {
		setBundles(previous =>
			previous.map(item => {
				if (item.bundle.ref.rootID !== bundleRef.rootID || item.bundle.ref.artifactID !== bundleRef.artifactID) {
					return item;
				}

				return {
					...item,
					bundle: {
						...item.bundle,
						enabled,
						collection: {
							...item.bundle.collection,
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

		for (const bundle of bundles) {
			const server = bundle.servers.find(item => artifactKey(item.ref) === artifactKey(oauthTarget.server));

			if (server) {
				return {
					server,
					authHealth: bundle.authHealthByArtifactID[server.ref.artifactID],
					runtime: bundle.runtimeByArtifactID[server.ref.artifactID],
				};
			}
		}

		return undefined;
	}, [bundles, oauthTarget]);

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
					description="Organize MCP servers in Collections and configure their connections, credentials, runtime state, and tool policies."
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
								setIsAddBundleOpen(true);
							}}
						>
							<FiPlus size={18} />
							<span>Add Collection</span>
						</button>
					}
				/>

				<ManagementPageContent width="wide">
					{isInitialLoading ? <Loader text="Loading MCP Collections..." /> : null}

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

					{!isInitialLoading && bundles.length === 0 ? (
						<p className="mt-8 text-center text-sm">No MCP Collections configured yet.</p>
					) : null}

					{bundles.map(bundleData => (
						<MCPBundleCard
							key={`${bundleData.bundle.ref.rootID}:${bundleData.bundle.ref.artifactID}`}
							bundle={bundleData.bundle}
							servers={bundleData.servers}
							existingLogicalNames={bundleData.servers.map(server => server.logicalName)}
							runtimeByArtifactID={bundleData.runtimeByArtifactID}
							authHealthByArtifactID={bundleData.authHealthByArtifactID}
							readErrorsByArtifactID={bundleData.readErrorsByArtifactID}
							serverLoadError={bundleData.serverLoadError}
							isLoadingServers={bundleData.isLoadingServers}
							serversLoaded={bundleData.serversLoaded}
							onLoadServers={() => loadBundleServers(bundleData.bundle.ref)}
							onRefreshServers={() => refreshBundle(bundleData.bundle.ref)}
							onToggleBundleEnabled={async (bundle, enabled) => {
								const previous = bundle.enabled;
								patchBundleEnabled(bundle.ref, enabled);

								try {
									await mcpManagementAPI.setMCPBundleEnabled(bundle, enabled);
								} catch (error) {
									patchBundleEnabled(bundle.ref, previous);
									throw error;
								}

								if (bundleData.serversLoaded) {
									await refreshBundle(bundle.ref);
								}
							}}
							onSaveServer={handleSaveServer}
							onSaveSetup={handleSaveSetup}
							onDeleteServer={async (bundle, server) => {
								await mcpManagementAPI.deleteMCPServer(bundle, server);
								await refreshBundle(bundle.ref);
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
							onDeleteBundleRequested={bundle => {
								setBundleToDelete(bundle);
							}}
						/>
					))}
				</ManagementPageContent>

				<DeleteConfirmationModal
					isOpen={bundleToDelete !== null}
					onClose={() => {
						if (!isDeletingBundle) {
							setBundleToDelete(null);
						}
					}}
					onConfirm={async () => {
						if (!bundleToDelete || isDeletingBundle) {
							return;
						}

						const current = bundlesRef.current.find(
							item =>
								item.bundle.ref.rootID === bundleToDelete.ref.rootID &&
								item.bundle.ref.artifactID === bundleToDelete.ref.artifactID
						);

						if (!current?.serversLoaded || current.serverLoadError || current.servers.length > 0) {
							setAlertMessage('Load the Collection and remove all currently available MCP servers before deleting it.');
							return;
						}

						setIsDeletingBundle(true);

						try {
							await mcpManagementAPI.deleteMCPBundle(bundleToDelete);
							setBundleToDelete(null);
							await fetchAll();
						} catch (error) {
							setAlertMessage(getErrorMessage(error, 'Failed to delete MCP Collection.'));
						} finally {
							setIsDeletingBundle(false);
						}
					}}
					title="Delete MCP Collection"
					message={`Delete empty MCP Collection "${bundleToDelete?.displayName ?? ''}"? Remove all servers first.`}
					confirmButtonText={isDeletingBundle ? 'Deleting...' : 'Delete'}
				/>

				<ManagementBundleCreateModal
					isOpen={isAddBundleOpen}
					title="Add MCP Collection"
					entityLabel="MCP Collection"
					onClose={() => {
						setIsAddBundleOpen(false);
					}}
					onSubmit={handleSaveBundle}
					existingSlugs={bundles.map(bundle => bundle.bundle.logicalName)}
					failureMessage="Failed to create MCP Collection."
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
