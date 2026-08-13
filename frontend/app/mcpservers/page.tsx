import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { FiPlus, FiSettings } from 'react-icons/fi';

import type { ArtifactRef } from '@/spec/artifact';
import type {
	MCPAuthHealth,
	MCPGlobalSettings,
	MCPOAuthAuthorization,
	MCPServerRuntimeSnapshot,
} from '@/spec/mcp_artifact';
import { MCPAuthHealthState, MCPHTTPAuthMode } from '@/spec/mcp_artifact';

import { mapWithConcurrency, withTimeout } from '@/lib/async_utils';

import { backendAPI, mcpAPI } from '@/apis/baseapi';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';
import { Loader } from '@/components/loader';
import { ManagementBundleCreateModal } from '@/components/managementui/management_bundle_create_modal';
import { ManagementPageContent } from '@/components/managementui/management_page_content';
import { ManagementPageHeader } from '@/components/managementui/management_page_header';
import { ManagementResourceError } from '@/components/managementui/management_resource_error';
import { PageFrame } from '@/components/page_frame';

import type {
	MCPBundleView,
	MCPServerDraft,
	MCPServerView,
	MCPSetupSubmissionValue,
} from '@/mcpservers/lib/mcp_management';
import {
	applyMCPServerSetup,
	createMCPBundle,
	deleteMCPBundle,
	deleteMCPServer,
	getAuthMode,
	loadMCPBundleViews,
	loadMCPServerViews,
	saveMCPServer,
	setMCPBundleRuntimeEnabled,
	setMCPServerRuntimeEnabled,
} from '@/mcpservers/lib/mcp_management';
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
}

interface OAuthTarget {
	server: ArtifactRef;
}

const STATUS_READ_CONCURRENCY = 6;
const BUNDLE_LOAD_CONCURRENCY = 4;
const CONNECT_TIMEOUT_MS = 60_000;

function artifactKey(ref: ArtifactRef): string {
	return `${ref.rootID}:${ref.artifactID}`;
}

function getErrorMessage(error: unknown, fallback: string): string {
	if (error instanceof Error && error.message.trim()) {
		return error.message;
	}

	return fallback;
}

function sleep(milliseconds: number): Promise<void> {
	return new Promise(resolve => {
		window.setTimeout(resolve, milliseconds);
	});
}

function getMatchingAuthHealth(server: MCPServerView, value: MCPAuthHealth | undefined): MCPAuthHealth | undefined {
	if (!value) {
		return undefined;
	}

	if (value.server.rootID !== server.ref.rootID || value.server.artifactID !== server.ref.artifactID) {
		console.warn('Ignoring MCP auth health returned for another Artifact.', {
			expected: server.ref,
			actual: value.server,
		});
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

	if (value.server.rootID !== server.ref.rootID || value.server.artifactID !== server.ref.artifactID) {
		console.warn('Ignoring MCP runtime snapshot returned for another Artifact.', {
			expected: server.ref,
			actual: value.server,
		});
		return undefined;
	}

	return value;
}

function pendingAuthForServer(
	server: MCPServerView,
	pending: MCPOAuthAuthorization[],
	previous?: MCPAuthHealth
): MCPAuthHealth | undefined {
	const authorization = pending.find(
		item => item.server.rootID === server.ref.rootID && item.server.artifactID === server.ref.artifactID
	);

	if (!authorization) {
		return previous;
	}

	return {
		...previous,
		server: server.ref,
		authMode: MCPHTTPAuthMode.OAuth,
		state: MCPAuthHealthState.AuthorizationPending,
		configured: previous?.configured ?? true,
		authorizationPending: true,
		authorizationURL: authorization.authorizationURL,
		authorizationExpiresAt: authorization.expiresAt,
		lastError: undefined,
	};
}

// oxlint-disable-next-line no-restricted-exports
export default function MCPServersPage() {
	const [bundles, setBundles] = useState<BundleData[]>([]);
	const [settings, setSettings] = useState<MCPGlobalSettings>();
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

	useEffect(() => {
		bundlesRef.current = bundles;
	}, [bundles]);

	useEffect(() => {
		mountedRef.current = true;

		return () => {
			mountedRef.current = false;
			loadIDRef.current += 1;
		};
	}, []);

	const readServerStatus = useCallback(
		async (
			servers: MCPServerView[],
			knownPending?: MCPOAuthAuthorization[]
		): Promise<{
			runtimeByArtifactID: Record<string, MCPServerRuntimeSnapshot | undefined>;
			authHealthByArtifactID: Record<string, MCPAuthHealth | undefined>;
			readErrorsByArtifactID: Record<string, { runtime?: string; auth?: string } | undefined>;
		}> => {
			const pending =
				knownPending ?? (await mcpAPI.listPendingMCPOAuthAuthorizations().catch(() => [] as MCPOAuthAuthorization[]));

			const entries = await mapWithConcurrency(servers, STATUS_READ_CONCURRENCY, async server => {
				const key = server.ref.artifactID;

				const [runtimeResult, authResult] = await Promise.allSettled([
					mcpAPI.getMCPServerStatus(server.ref),
					mcpAPI.getMCPServerAuthHealth(server.ref),
				]);

				const errors: { runtime?: string; auth?: string } = {};

				if (runtimeResult.status === 'rejected') {
					errors.runtime = getErrorMessage(runtimeResult.reason, 'Runtime status request failed.');
				}

				if (authResult.status === 'rejected') {
					errors.auth = getErrorMessage(authResult.reason, 'Authorization health request failed.');
				}

				const runtime =
					runtimeResult.status === 'fulfilled' ? getMatchingRuntimeSnapshot(server, runtimeResult.value) : undefined;
				const auth =
					authResult.status === 'fulfilled'
						? pendingAuthForServer(server, pending, getMatchingAuthHealth(server, authResult.value))
						: undefined;

				return {
					key,
					runtime,
					auth,
					errors: Object.keys(errors).length > 0 ? errors : undefined,
				};
			});

			return {
				runtimeByArtifactID: Object.fromEntries(entries.map(entry => [entry.key, entry.runtime])),
				authHealthByArtifactID: Object.fromEntries(entries.map(entry => [entry.key, entry.auth])),
				readErrorsByArtifactID: Object.fromEntries(entries.map(entry => [entry.key, entry.errors])),
			};
		},
		[]
	);

	const loadBundleData = useCallback(
		async (bundle: MCPBundleView, pending?: MCPOAuthAuthorization[]): Promise<BundleData> => {
			try {
				const servers = await loadMCPServerViews(bundle);
				const statuses = await readServerStatus(servers, pending);

				return {
					bundle,
					servers,
					...statuses,
				};
			} catch (error) {
				return {
					bundle,
					servers: [],
					runtimeByArtifactID: {},
					authHealthByArtifactID: {},
					readErrorsByArtifactID: {},
					serverLoadError: getErrorMessage(error, 'Failed to load servers for this MCP Bundle.'),
				};
			}
		},
		[readServerStatus]
	);

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
			const [bundleResult, pendingResult, settingsResult] = await Promise.allSettled([
				loadMCPBundleViews(),
				mcpAPI.listPendingMCPOAuthAuthorizations(),
				mcpAPI.getMCPGlobalSettings(),
			]);

			if (bundleResult.status === 'rejected') {
				throw bundleResult.reason;
			}

			const pending = pendingResult.status === 'fulfilled' ? pendingResult.value : [];

			const warningsNext = [
				pendingResult.status === 'rejected'
					? getErrorMessage(pendingResult.reason, 'Pending OAuth authorizations could not be loaded.')
					: undefined,
				settingsResult.status === 'rejected'
					? getErrorMessage(settingsResult.reason, 'MCP OAuth settings could not be loaded.')
					: undefined,
			].filter((value): value is string => Boolean(value));

			const loaded = await mapWithConcurrency(bundleResult.value, BUNDLE_LOAD_CONCURRENCY, bundle =>
				loadBundleData(bundle, pending)
			);

			if (!mountedRef.current || loadIDRef.current !== requestID) {
				return;
			}

			setBundles(loaded);
			setWarnings(warningsNext);
			setSettings(settingsResult.status === 'fulfilled' ? settingsResult.value : undefined);
			loadedOnceRef.current = true;
		} catch (error) {
			if (!mountedRef.current || loadIDRef.current !== requestID) {
				return;
			}

			setPageLoadError(error);
			setAlertMessage(getErrorMessage(error, 'Failed to load MCP Bundles.'));
		} finally {
			if (mountedRef.current && loadIDRef.current === requestID) {
				setIsInitialLoading(false);
				setIsRefreshing(false);
			}
		}
	}, [loadBundleData]);

	useEffect(() => {
		// oxlint-disable-next-line jsreact-hooks/set-state-in-effect
		void fetchAll().catch(() => undefined);
	}, [fetchAll]);

	const refreshBundle = useCallback(
		async (bundleRef: MCPBundleView['ref']) => {
			const existing = bundlesRef.current.find(
				item => item.bundle.ref.rootID === bundleRef.rootID && item.bundle.ref.collectionID === bundleRef.collectionID
			);

			if (!existing) {
				throw new Error('MCP Bundle is no longer available.');
			}

			const refreshedBundle = (await loadMCPBundleViews()).find(
				item => item.ref.rootID === bundleRef.rootID && item.ref.collectionID === bundleRef.collectionID
			);

			if (!refreshedBundle) {
				throw new Error('MCP Bundle could not be reloaded.');
			}

			const refreshed = await loadBundleData(refreshedBundle);

			setBundles(previous =>
				previous.map(item =>
					item.bundle.ref.rootID === bundleRef.rootID && item.bundle.ref.collectionID === bundleRef.collectionID
						? refreshed
						: item
				)
			);
		},
		[loadBundleData]
	);

	const refreshSingleServer = useCallback(
		async (server: MCPServerView) => {
			const parent = bundlesRef.current.find(
				item =>
					item.bundle.ref.rootID === server.bundle.rootID && item.bundle.ref.collectionID === server.bundle.collectionID
			);

			if (!parent) {
				throw new Error('MCP Bundle is no longer available.');
			}

			const refreshed = await readServerStatus([server]);

			setBundles(previous =>
				previous.map(item => {
					if (
						item.bundle.ref.rootID !== server.bundle.rootID ||
						item.bundle.ref.collectionID !== server.bundle.collectionID
					) {
						return item;
					}

					return {
						...item,
						runtimeByArtifactID: {
							...item.runtimeByArtifactID,
							[server.ref.artifactID]: refreshed.runtimeByArtifactID[server.ref.artifactID],
						},
						authHealthByArtifactID: {
							...item.authHealthByArtifactID,
							[server.ref.artifactID]: refreshed.authHealthByArtifactID[server.ref.artifactID],
						},
						readErrorsByArtifactID: {
							...item.readErrorsByArtifactID,
							[server.ref.artifactID]: refreshed.readErrorsByArtifactID[server.ref.artifactID],
						},
					};
				})
			);
		},
		[readServerStatus]
	);

	const oauthPollKey = useMemo(
		() =>
			JSON.stringify(
				bundles.flatMap(bundle =>
					bundle.servers
						.filter(server => {
							return (
								server.runtimeEnabled &&
								server.document?.mcpServer.type !== undefined &&
								getAuthMode(server) === MCPHTTPAuthMode.OAuth
							);
						})
						.map(server => server.ref)
				)
			),
		[bundles]
	);

	useEffect(() => {
		const servers = JSON.parse(oauthPollKey) as ArtifactRef[];

		if (servers.length === 0) {
			return;
		}

		let cancelled = false;
		let polling = false;

		const poll = async () => {
			if (cancelled || polling) {
				return;
			}

			polling = true;

			try {
				const pending = await mcpAPI.listPendingMCPOAuthAuthorizations();

				await mapWithConcurrency(servers, STATUS_READ_CONCURRENCY, async ref => {
					const bundle = bundlesRef.current.find(item =>
						item.servers.some(server => artifactKey(server.ref) === artifactKey(ref))
					);
					const server = bundle?.servers.find(item => artifactKey(item.ref) === artifactKey(ref));

					if (!cancelled && server) {
						const statuses = await readServerStatus([server], pending);

						setBundles(previous =>
							previous.map(item => {
								if (
									item.bundle.ref.rootID !== server.bundle.rootID ||
									item.bundle.ref.collectionID !== server.bundle.collectionID
								) {
									return item;
								}

								return {
									...item,
									runtimeByArtifactID: {
										...item.runtimeByArtifactID,
										[server.ref.artifactID]: statuses.runtimeByArtifactID[server.ref.artifactID],
									},
									authHealthByArtifactID: {
										...item.authHealthByArtifactID,
										[server.ref.artifactID]: statuses.authHealthByArtifactID[server.ref.artifactID],
									},
									readErrorsByArtifactID: {
										...item.readErrorsByArtifactID,
										[server.ref.artifactID]: statuses.readErrorsByArtifactID[server.ref.artifactID],
									},
								};
							})
						);
					}
				});
			} catch (error) {
				console.error('MCP OAuth polling failed:', error);
			} finally {
				polling = false;
			}
		};

		void poll();
		const timer = window.setInterval(() => {
			void poll();
		}, 4000);

		return () => {
			cancelled = true;
			window.clearInterval(timer);
		};
	}, [oauthPollKey, readServerStatus]);

	const handleSaveBundle = useCallback(
		async (logicalName: string, displayName: string, description?: string) => {
			await createMCPBundle(logicalName, displayName, description);
			await fetchAll();
		},
		[fetchAll]
	);

	const handleSaveServer = useCallback(
		async (bundle: MCPBundleView, server: MCPServerView | undefined, draft: MCPServerDraft) => {
			await saveMCPServer(bundle, server, draft);
			await refreshBundle(bundle.ref);
		},
		[refreshBundle]
	);

	const handleSaveSetup = useCallback(
		async (server: MCPServerView, values: Record<string, MCPSetupSubmissionValue>, reset: boolean) => {
			await applyMCPServerSetup(server, values, reset);
			await refreshBundle(server.bundle);
		},
		[refreshBundle]
	);

	const selectedOAuth = (() => {
		if (!oauthTarget) {
			return undefined;
		}

		for (const bundle of bundles) {
			const server = bundle.servers.find(item => artifactKey(item.ref) === artifactKey(oauthTarget.server));

			if (server) {
				return {
					server,
					authHealth: bundle.authHealthByArtifactID[server.ref.artifactID],
				};
			}
		}
		return undefined;
	})();

	if (isInitialLoading) {
		return <Loader text="Loading MCP servers…" />;
	}

	return (
		<PageFrame>
			<div className="flex size-full flex-col items-center overflow-hidden">
				<ManagementPageHeader
					title="MCP Servers"
					description="Configure artifact-backed MCP Bundles, server Definitions, installation-local secrets, runtime state, and policy."
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
							<span>Add Bundle</span>
						</button>
					}
				/>

				<ManagementPageContent width="wide">
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

					{bundles.length === 0 ? <p className="mt-8 text-center text-sm">No MCP Bundles configured yet.</p> : null}

					{bundles.map(bundleData => (
						<MCPBundleCard
							key={`${bundleData.bundle.ref.rootID}:${bundleData.bundle.ref.collectionID}`}
							bundle={bundleData.bundle}
							servers={bundleData.servers}
							existingLogicalNames={bundleData.servers.map(server => server.logicalName)}
							runtimeByArtifactID={bundleData.runtimeByArtifactID}
							authHealthByArtifactID={bundleData.authHealthByArtifactID}
							readErrorsByArtifactID={bundleData.readErrorsByArtifactID}
							serverLoadError={bundleData.serverLoadError}
							onRefreshServers={() => refreshBundle(bundleData.bundle.ref)}
							onToggleBundleEnabled={async (bundle, enabled) => {
								await setMCPBundleRuntimeEnabled(bundle, enabled);
								await refreshBundle(bundle.ref);
							}}
							onToggleServerEnabled={async (bundle, server, enabled) => {
								await setMCPServerRuntimeEnabled(bundle, server, enabled);
								await refreshBundle(bundle.ref);
							}}
							onSaveServer={handleSaveServer}
							onSaveSetup={handleSaveSetup}
							onDeleteServer={async (bundle, server) => {
								await deleteMCPServer(bundle, server);
								await refreshBundle(bundle.ref);
							}}
							onConnectServer={async server => {
								const promise = withTimeout(
									mcpAPI.connectMCPServer(server.ref),
									CONNECT_TIMEOUT_MS,
									`Connecting "${server.displayName}" timed out after ${CONNECT_TIMEOUT_MS / 1000} seconds.`
								);

								await Promise.race([
									promise,
									sleep(1000).then(async () => {
										await refreshSingleServer(server);
									}),
								]);

								await refreshSingleServer(server);
							}}
							onDisconnectServer={async server => {
								await mcpAPI.disconnectMCPServer(server.ref);
								await refreshSingleServer(server);
							}}
							onRefreshServer={async server => {
								await mcpAPI.refreshMCPServer(server.ref);
								await refreshSingleServer(server);
							}}
							onCancelOAuth={async server => {
								await mcpAPI.cancelPendingMCPOAuthAuthorization(server.ref);
								await refreshSingleServer(server);
							}}
							onRequestOAuthAuthorization={server => {
								setOAuthTarget({
									server: server.ref,
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

						setIsDeletingBundle(true);

						try {
							await deleteMCPBundle(bundleToDelete);
							setBundleToDelete(null);
							await fetchAll();
						} catch (error) {
							setAlertMessage(getErrorMessage(error, 'Failed to delete MCP Bundle.'));
						} finally {
							setIsDeletingBundle(false);
						}
					}}
					title="Delete MCP Bundle"
					message={`Delete empty MCP Bundle "${bundleToDelete?.displayName ?? ''}"? Remove all server and policy Artifacts first.`}
					confirmButtonText={isDeletingBundle ? 'Deleting...' : 'Delete'}
				/>

				<ManagementBundleCreateModal
					isOpen={isAddBundleOpen}
					title="Add MCP Bundle"
					entityLabel="MCP Bundle"
					onClose={() => {
						setIsAddBundleOpen(false);
					}}
					onSubmit={handleSaveBundle}
					existingSlugs={bundles.map(bundle => bundle.bundle.logicalName)}
					failureMessage="Failed to create MCP Bundle."
				/>

				<MCPSettingsModal
					isOpen={isSettingsOpen}
					initialListenAddr={settings?.settings.oauthLoopbackListenAddr}
					activeListenAddr={settings?.oauthLoopbackListenAddr}
					oauthRedirectURL={settings?.oauthRedirectURL}
					onClose={() => {
						setIsSettingsOpen(false);
					}}
					onSubmit={async oauthLoopbackListenAddr => {
						const current = settings ?? (await mcpAPI.getMCPGlobalSettings());
						await mcpAPI.updateMCPGlobalSettings(current.revision, oauthLoopbackListenAddr || undefined);
						setSettings(await mcpAPI.getMCPGlobalSettings());
					}}
				/>

				<MCPOAuthAuthorizationModal
					isOpen={Boolean(selectedOAuth)}
					server={selectedOAuth?.server ?? null}
					authHealth={selectedOAuth?.authHealth}
					onClose={() => {
						setOAuthTarget(null);
					}}
					onOpenURL={url => {
						backendAPI.openURL(url);
					}}
					onCancel={async () => {
						if (!selectedOAuth) {
							return;
						}

						await mcpAPI.cancelPendingMCPOAuthAuthorization(selectedOAuth.server.ref);
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
