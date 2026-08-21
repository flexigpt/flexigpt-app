import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { FiPlus, FiSettings } from 'react-icons/fi';

import type { ArtifactRef } from '@/spec/artifact';
import type {
	MCPAuthHealth,
	MCPGlobalSettings,
	MCPOAuthAuthorization,
	MCPServerRuntimeSnapshot,
} from '@/spec/mcp_artifact';
import { MCPAuthHealthState, MCPHTTPAuthMode, MCPServerStatus } from '@/spec/mcp_artifact';

import { mapWithConcurrency } from '@/lib/async_utils';

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
	loadMCPBundleView,
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
	isLoadingServers: boolean;
}

interface OAuthTarget {
	server: ArtifactRef;
	openAutomatically: boolean;
}

const STATUS_READ_CONCURRENCY = 4;
const BUNDLE_LOAD_CONCURRENCY = 2;
const CONNECTION_POLL_INTERVAL_MS = 500;
const CONNECTION_WAIT_TIMEOUT_MS = 11 * 60 * 1000;

function artifactKey(ref: ArtifactRef): string {
	return `${ref.rootID}:${ref.artifactID}`;
}

function sleep(ms: number): Promise<void> {
	return new Promise(resolve => {
		window.setTimeout(resolve, ms);
	});
}

function getErrorMessage(error: unknown, fallback: string): string {
	if (error instanceof Error && error.message.trim()) {
		return error.message;
	}

	return fallback;
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
	const openedAuthorizationURLsRef = useRef(new Set<string>());

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
				const pendingAuthorization = pending.find(
					item => item.server.rootID === server.ref.rootID && item.server.artifactID === server.ref.artifactID
				);
				const authRequest: Promise<MCPAuthHealth | undefined> = pendingAuthorization
					? Promise.resolve(undefined)
					: mcpAPI.getMCPServerAuthHealth(server.ref);

				const [runtimeResult, authResult] = await Promise.allSettled([
					mcpAPI.getMCPServerStatus(server.ref),
					authRequest,
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
				const baseAuth =
					authResult.status === 'fulfilled' ? getMatchingAuthHealth(server, authResult.value) : undefined;
				const auth = pendingAuthForServer(server, pending, baseAuth);

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
					isLoadingServers: false,
				};
			} catch (error) {
				return {
					bundle,
					servers: [],
					runtimeByArtifactID: {},
					authHealthByArtifactID: {},
					readErrorsByArtifactID: {},
					serverLoadError: getErrorMessage(error, 'Failed to load servers for this MCP Bundle.'),
					isLoadingServers: false,
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
			const settingsPromise = mcpAPI.getMCPGlobalSettings();
			const [bundleResult, pendingResult] = await Promise.allSettled([
				loadMCPBundleViews(),
				mcpAPI.listPendingMCPOAuthAuthorizations(),
			]);

			if (bundleResult.status === 'rejected') {
				throw bundleResult.reason;
			}

			const pending = pendingResult.status === 'fulfilled' ? pendingResult.value : [];

			const warningsNext = [
				pendingResult.status === 'rejected'
					? getErrorMessage(pendingResult.reason, 'Pending OAuth authorizations could not be loaded.')
					: undefined,
			].filter((value): value is string => Boolean(value));

			if (!mountedRef.current || loadIDRef.current !== requestID) {
				return;
			}

			const shells = bundleResult.value.map(bundle => {
				const existing = bundlesRef.current.find(
					item =>
						item.bundle.ref.rootID === bundle.ref.rootID && item.bundle.ref.collectionID === bundle.ref.collectionID
				);

				if (existing) {
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
					isLoadingServers: true,
				} satisfies BundleData;
			});

			setBundles(shells);
			setWarnings(warningsNext);
			loadedOnceRef.current = true;
			setIsInitialLoading(false);

			void settingsPromise
				.then(value => {
					if (mountedRef.current && loadIDRef.current === requestID) {
						setSettings(value);
					}
				})
				.catch((error: unknown) => {
					if (!mountedRef.current || loadIDRef.current !== requestID) {
						return;
					}

					const warning = getErrorMessage(error, 'MCP OAuth settings could not be loaded.');
					setWarnings(previous => (previous.includes(warning) ? previous : [...previous, warning]));
				});

			const loaded = await mapWithConcurrency(bundleResult.value, BUNDLE_LOAD_CONCURRENCY, async bundle => {
				const value = await loadBundleData(bundle, pending);

				if (mountedRef.current && loadIDRef.current === requestID) {
					setBundles(previous =>
						previous.map(item =>
							item.bundle.ref.rootID === bundle.ref.rootID && item.bundle.ref.collectionID === bundle.ref.collectionID
								? value
								: item
						)
					);
				}

				return value;
			});

			if (!mountedRef.current || loadIDRef.current !== requestID) {
				return;
			}

			setBundles(loaded);
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
		// oxlint-disable-next-line react/set-state-in-effect
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

			const refreshedBundle = await loadMCPBundleView(bundleRef);

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

	const markServerConnecting = useCallback((server: MCPServerView) => {
		setBundles(previous =>
			previous.map(item => {
				if (
					item.bundle.ref.rootID !== server.bundle.rootID ||
					item.bundle.ref.collectionID !== server.bundle.collectionID
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
							server: server.ref,
							collection: server.bundle,
							status: MCPServerStatus.Connecting,
							lastError: undefined,
							toolCount: current?.toolCount ?? 0,
							resourceCount: current?.resourceCount ?? 0,
							resourceTemplateCount: current?.resourceTemplateCount ?? 0,
							promptCount: current?.promptCount ?? 0,
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
				item.bundle.ref.rootID === server.bundle.rootID && item.bundle.ref.collectionID === server.bundle.collectionID
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

			markServerConnecting(server);

			try {
				let snapshot = await mcpAPI.connectMCPServer(server.ref);
				applyRuntimeSnapshot(server, snapshot);

				const deadline = Date.now() + CONNECTION_WAIT_TIMEOUT_MS;
				while (mountedRef.current && snapshot.status === MCPServerStatus.Connecting) {
					if (Date.now() >= deadline) {
						throw new Error('Timed out waiting for the MCP server to connect.');
					}

					await sleep(CONNECTION_POLL_INTERVAL_MS);
					snapshot = await mcpAPI.getMCPServerStatus(server.ref);
					applyRuntimeSnapshot(server, snapshot);

					// Keep OAuth health and pending authorization state in sync
					// while the runtime is connecting.
					await refreshSingleServer(server).catch(() => undefined);
				}

				if (snapshot.status === MCPServerStatus.Error) {
					throw new Error(snapshot.lastError || 'The MCP server connection failed.');
				}
				if (snapshot.status !== MCPServerStatus.Ready && snapshot.status !== MCPServerStatus.Disconnected) {
					throw new Error('The MCP server did not reach a usable terminal state.');
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
							const authHealth = bundle.authHealthByArtifactID[server.ref.artifactID];
							return (
								runtime?.status === MCPServerStatus.Connecting ||
								authHealth?.state === MCPAuthHealthState.AuthorizationPending
							);
						})
						.map(server => server.ref)
				)
			),
		[bundles]
	);

	useEffect(() => {
		const servers = JSON.parse(connectionPollKey) as ArtifactRef[];

		if (servers.length === 0) {
			return;
		}

		let cancelled = false;
		let polling = false;
		let pollDelay = 300;
		let timer: number | undefined;

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
				console.error('MCP connection polling failed:', error);
			} finally {
				polling = false;

				if (!cancelled) {
					pollDelay = Math.min(Math.round(pollDelay * 1.5), 2000);
					timer = window.setTimeout(
						() => {
							void poll();
						},
						document.hidden ? 4000 : pollDelay
					);
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
	}, [connectionPollKey, readServerStatus]);

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

	const patchBundleEnabled = useCallback((bundleRef: MCPBundleView['ref'], enabled: boolean) => {
		setBundles(previous =>
			previous.map(item => {
				if (item.bundle.ref.rootID !== bundleRef.rootID || item.bundle.ref.collectionID !== bundleRef.collectionID) {
					return item;
				}

				return {
					...item,
					bundle: {
						...item.bundle,
						enabled,
						installation: {
							...item.bundle.installation,
							runtimeEnabled: enabled,
						},
					},
					servers: item.servers.map(server => ({
						...server,
						runtimeEnabled: enabled && server.installationEnabled,
					})),
					runtimeByArtifactID: Object.fromEntries(
						item.servers.map(server => {
							const current = item.runtimeByArtifactID[server.ref.artifactID];
							return [
								server.ref.artifactID,
								current
									? {
											...current,
											status: MCPServerStatus.Disconnected,
											lastError: undefined,
										}
									: undefined,
							];
						})
					),
				};
			})
		);
	}, []);

	const patchServerEnabled = useCallback((server: MCPServerView, enabled: boolean) => {
		setBundles(previous =>
			previous.map(item => {
				if (
					item.bundle.ref.rootID !== server.bundle.rootID ||
					item.bundle.ref.collectionID !== server.bundle.collectionID
				) {
					return item;
				}

				const current = item.runtimeByArtifactID[server.ref.artifactID];
				return {
					...item,
					servers: item.servers.map(value =>
						artifactKey(value.ref) === artifactKey(server.ref)
							? {
									...value,
									installationEnabled: enabled,
									runtimeEnabled: item.bundle.enabled && enabled,
								}
							: value
					),
					runtimeByArtifactID: {
						...item.runtimeByArtifactID,
						[server.ref.artifactID]: current
							? { ...current, status: MCPServerStatus.Disconnected, lastError: undefined }
							: undefined,
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

		const reportFailure = () => {
			openedAuthorizationURLsRef.current.delete(key);
			if (mountedRef.current) {
				setAlertMessage('The OAuth authorization page could not be opened.');
			}
		};

		try {
			backendAPI.openURL(url);
		} catch {
			reportFailure();
		}
	}, []);

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
					runtime: bundle.runtimeByArtifactID[server.ref.artifactID],
				};
			}
		}
		return undefined;
	})();

	useEffect(() => {
		if (!oauthTarget?.openAutomatically) {
			return;
		}

		const authorizationURL = selectedOAuth?.authHealth?.authorizationURL?.trim();
		if (!authorizationURL || !selectedOAuth) {
			return;
		}

		openAuthorizationURL(selectedOAuth.server.ref, authorizationURL, true);
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
					{isInitialLoading ? <Loader text="Loading MCP bundle index..." /> : null}

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
						<p className="mt-8 text-center text-sm">No MCP Bundles configured yet.</p>
					) : null}

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
							isLoadingServers={bundleData.isLoadingServers}
							onRefreshServers={() => refreshBundle(bundleData.bundle.ref)}
							onToggleBundleEnabled={async (bundle, enabled) => {
								const previous = bundle.enabled;
								patchBundleEnabled(bundle.ref, enabled);

								try {
									await setMCPBundleRuntimeEnabled(bundle, enabled);
								} catch (error) {
									patchBundleEnabled(bundle.ref, previous);
									throw error;
								}

								await refreshBundle(bundle.ref);
							}}
							onToggleServerEnabled={async (bundle, server, enabled) => {
								const previous = server.installationEnabled;
								patchServerEnabled(server, enabled);

								try {
									await setMCPServerRuntimeEnabled(bundle, server, enabled);
								} catch (error) {
									patchServerEnabled(server, previous);
									throw error;
								}

								await refreshBundle(bundle.ref);
							}}
							onSaveServer={handleSaveServer}
							onSaveSetup={handleSaveSetup}
							onDeleteServer={async (bundle, server) => {
								await deleteMCPServer(bundle, server);
								await refreshBundle(bundle.ref);
							}}
							onConnectServer={handleConnectServer}
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
					oauthRestartRequired={settings?.oauthRestartRequired}
					oauthLoopbackReady={settings?.oauthLoopbackReady}
					oauthLoopbackError={settings?.oauthLoopbackError}
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
