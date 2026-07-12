import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { FiPlus, FiSettings } from 'react-icons/fi';

import type {
	MCPAuthHealth,
	MCPBundle,
	MCPOAuthAuthorization,
	MCPServerConfig,
	MCPServerRuntimeSnapshot,
	MCPServerSetupInputValue,
	MCPSettingsView,
	PutMCPServerPayload,
} from '@/spec/mcp';
import { BaseMCPBundleID, MCPAuthHealthState, MCPHTTPAuthMode, MCPSecretKind, MCPTransportType } from '@/spec/mcp';

import { mapWithConcurrency, withTimeout } from '@/lib/async_utils';
import { omitManyKeys } from '@/lib/obj_utils';
import { getUUIDv7 } from '@/lib/uuid_utils';

import { backendAPI, mcpAPI } from '@/apis/baseapi';
import { getAllMCPBundles, getAllMCPServers } from '@/apis/list_helper';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';
import { Loader } from '@/components/loader';
import { ManagementBundleCreateModal } from '@/components/managementui/management_bundle_create_modal';
import { ManagementPageContent } from '@/components/managementui/management_page_content';
import { ManagementPageHeader } from '@/components/managementui/management_page_header';
import { ManagementResourceError } from '@/components/managementui/management_resource_error';
import { PageFrame } from '@/components/page_frame';

import type { MCPServerUpsertInput } from '@/mcpservers/lib/mcp_server_utils';
import { getEffectiveMCPAuthHealthState, isMCPAuthActionable } from '@/mcpservers/lib/mcp_server_utils';
import { MCPBundleCard } from '@/mcpservers/mcp_bundle_card';
import { MCPOAuthAuthorizationModal } from '@/mcpservers/mcp_oauth_authorization_modal';
import { MCPSettingsModal } from '@/mcpservers/mcp_settings_modal';

interface BundleData {
	bundle: MCPBundle;
	servers: MCPServerConfig[];
	runtimeByServerID: Record<string, MCPServerRuntimeSnapshot | undefined>;
	authHealthByServerID: Record<string, MCPAuthHealth | undefined>;
	readErrorsByServerID: Record<string, { runtime?: string; auth?: string } | undefined>;
	serverLoadError?: string;
}

interface MCPPageResource {
	bundles: BundleData[];
	settingsView?: MCPSettingsView;
	warnings: string[];
}

interface OAuthAuthorizationTarget {
	bundleID: string;
	serverID: string;
}

const MCP_STATUS_READ_CONCURRENCY = 6;
const MCP_BUNDLE_LOAD_CONCURRENCY = 4;
const MCP_CONNECT_TIMEOUT_MS = 60_000;

function getErrorMessage(error: unknown, fallback: string): string {
	if (error instanceof Error && error.message.trim().length > 0) {
		return error.message;
	}
	return fallback;
}

function clonePayload(payload: PutMCPServerPayload): PutMCPServerPayload {
	return JSON.parse(JSON.stringify(payload)) as PutMCPServerPayload;
}

function mergeBundleData(
	bundle: MCPBundle,
	servers: MCPServerConfig[],
	runtimeByServerID: Record<string, MCPServerRuntimeSnapshot | undefined>,
	authHealthByServerID: Record<string, MCPAuthHealth | undefined>,
	readErrorsByServerID: Record<string, { runtime?: string; auth?: string } | undefined>,
	serverLoadError?: string
): BundleData {
	return {
		bundle,
		servers,
		runtimeByServerID,
		authHealthByServerID,
		readErrorsByServerID,
		serverLoadError,
	};
}

function sleep(ms: number): Promise<void> {
	return new Promise(resolve => {
		window.setTimeout(() => {
			resolve();
		}, ms);
	});
}

function isRecord(value: unknown): value is Record<string, unknown> {
	return typeof value === 'object' && value !== null;
}

function isMCPServerRuntimeSnapshotValue(value: unknown): value is MCPServerRuntimeSnapshot {
	return isRecord(value) && typeof value.serverID === 'string' && typeof value.status === 'string';
}

function getMatchingMCPServerRuntimeSnapshot(
	bundleID: string,
	serverID: string,
	value: unknown
): MCPServerRuntimeSnapshot | undefined {
	if (!isMCPServerRuntimeSnapshotValue(value)) {
		return undefined;
	}

	if (value.serverID !== serverID || (value.bundleID && value.bundleID !== bundleID)) {
		console.warn('Ignoring MCP runtime snapshot for a different server.', {
			requestedBundleID: bundleID,
			requestedServerID: serverID,
			responseBundleID: value.bundleID,
			responseServerID: value.serverID,
		});
		return undefined;
	}

	return value;
}

function getExpectedMCPServerAuthMode(server: MCPServerConfig): MCPHTTPAuthMode {
	if (server.transport === MCPTransportType.MCPTransportTypeStdio) {
		return MCPHTTPAuthMode.MCPHTTPAuthNone;
	}

	return server.streamableHttp?.authMode ?? MCPHTTPAuthMode.MCPHTTPAuthNone;
}

function getCoercedMCPAuthHealthStateForMode(authMode: MCPHTTPAuthMode, configured: boolean): MCPAuthHealthState {
	switch (authMode) {
		case MCPHTTPAuthMode.MCPHTTPAuthNone:
			return MCPAuthHealthState.MCPAuthHealthStateNotRequired;
		case MCPHTTPAuthMode.MCPHTTPAuthOAuth:
			return configured
				? MCPAuthHealthState.MCPAuthHealthStateAuthorizationNeeded
				: MCPAuthHealthState.MCPAuthHealthStateNotConfigured;
		case MCPHTTPAuthMode.MCPHTTPAuthAPIKey:
		case MCPHTTPAuthMode.MCPHTTPAuthClientCredentials:
			return configured
				? MCPAuthHealthState.MCPAuthHealthStateAuthorized
				: MCPAuthHealthState.MCPAuthHealthStateNotConfigured;
		default:
			return MCPAuthHealthState.MCPAuthHealthStateNotConfigured;
	}
}

function getMatchingMCPAuthHealth(
	bundleID: string,
	server: MCPServerConfig,
	value: MCPAuthHealth | undefined
): MCPAuthHealth | undefined {
	if (!value) {
		return undefined;
	}

	const authHealth = value;

	if (authHealth.serverID !== server.id || (authHealth.bundleID && authHealth.bundleID !== bundleID)) {
		console.warn('Ignoring MCP auth health for a different server.', {
			requestedBundleID: bundleID,
			requestedServerID: server.id,
			responseBundleID: authHealth.bundleID,
			responseServerID: authHealth.serverID,
		});
		return undefined;
	}

	const expectedAuthMode = getExpectedMCPServerAuthMode(server);

	if (authHealth.authMode !== expectedAuthMode) {
		console.warn('Coercing mismatched MCP auth health mode to the server config mode.', {
			bundleID,
			serverID: server.id,
			serverAuthMode: expectedAuthMode,
			healthAuthMode: authHealth.authMode,
			healthState: authHealth.state,
		});

		return {
			...authHealth,
			authMode: expectedAuthMode,
			state: getCoercedMCPAuthHealthStateForMode(expectedAuthMode, authHealth.configured),
		};
	}

	if (
		expectedAuthMode === MCPHTTPAuthMode.MCPHTTPAuthOAuth &&
		authHealth.state === MCPAuthHealthState.MCPAuthHealthStateAuthorized &&
		!authHealth.configured
	) {
		return {
			...authHealth,
			state: MCPAuthHealthState.MCPAuthHealthStateAuthorizationNeeded,
			authorizationPending: false,
		};
	}

	return authHealth;
}

function getPendingOAuthAuthHealth(
	bundleID: string,
	serverID: string,
	authorization: MCPOAuthAuthorization,
	previous?: MCPAuthHealth
): MCPAuthHealth {
	return {
		...previous,
		bundleID: authorization.bundleID || bundleID,
		serverID: authorization.serverID || serverID,
		authMode: MCPHTTPAuthMode.MCPHTTPAuthOAuth,
		state: MCPAuthHealthState.MCPAuthHealthStateAuthorizationPending,
		configured: previous?.configured ?? true,
		authorizationPending: true,
		authorizationURL: authorization.authorizationURL,
		authorizationExpiresAt: authorization.expiresAt,
		lastError: undefined,
	};
}

function isOAuthAuthorizationRelevant(server: MCPServerConfig, authHealth?: MCPAuthHealth): boolean {
	return (
		isMCPAuthActionable(authHealth, server) ||
		getEffectiveMCPAuthHealthState(server, authHealth) === MCPAuthHealthState.MCPAuthHealthStateAuthorizationPending
	);
}

// oxlint-disable-next-line no-restricted-exports
export default function MCPServersPage() {
	const [bundles, setBundles] = useState<BundleData[]>([]);
	const [isInitialLoading, setIsInitialLoading] = useState(true);
	const [isRefreshing, setIsRefreshing] = useState(false);
	const [pageLoadError, setPageLoadError] = useState<unknown>(undefined);
	const [resourceWarnings, setResourceWarnings] = useState<string[]>([]);

	const [showAlert, setShowAlert] = useState(false);
	const [alertMsg, setAlertMsg] = useState('');

	const [bundleToDeleteID, setBundleToDeleteID] = useState<string | null>(null);
	const [isDeletingBundle, setIsDeletingBundle] = useState(false);
	const [isAddModalOpen, setIsAddModalOpen] = useState(false);
	const [isSettingsOpen, setIsSettingsOpen] = useState(false);
	const [settingsView, setSettingsView] = useState<MCPSettingsView | undefined>(undefined);
	const [oauthAuthorizationTarget, setOAuthAuthorizationTarget] = useState<OAuthAuthorizationTarget | null>(null);

	const bundlesRef = useRef<BundleData[]>(bundles);
	const mountedRef = useRef(false);
	const pageLoadRequestIDRef = useRef(0);
	const hasLoadedPageRef = useRef(false);

	const bundleToDelete =
		bundleToDeleteID === null
			? null
			: (bundles.find(bundleData => bundleData.bundle.id === bundleToDeleteID)?.bundle ?? null);

	useEffect(() => {
		bundlesRef.current = bundles;
	}, [bundles]);

	useEffect(() => {
		mountedRef.current = true;

		return () => {
			mountedRef.current = false;
			pageLoadRequestIDRef.current += 1;
		};
	}, []);

	const loadRuntimeAndAuth = useCallback(
		async (bundleID: string, servers: MCPServerConfig[], knownPendingAuthorizations?: MCPOAuthAuthorization[]) => {
			let pendingAuthorizations = knownPendingAuthorizations ?? [];
			let pendingAuthorizationReadError: string | undefined;

			if (knownPendingAuthorizations === undefined) {
				const [pendingResult] = await Promise.allSettled([mcpAPI.listPendingMCPOAuthAuthorizations()]);

				if (pendingResult.status === 'fulfilled') {
					pendingAuthorizations = pendingResult.value;
				} else {
					pendingAuthorizationReadError = getErrorMessage(
						pendingResult.reason,
						'Pending OAuth authorizations could not be loaded.'
					);
				}
			}

			const entries = await mapWithConcurrency(servers, MCP_STATUS_READ_CONCURRENCY, async server => {
				const [runtimeResult, authHealthResult] = await Promise.allSettled([
					mcpAPI.getMCPServerStatus(bundleID, server.id),
					mcpAPI.getMCPServerAuthHealth(bundleID, server.id),
				]);

				const readErrors: { runtime?: string; auth?: string } = {};
				if (runtimeResult.status === 'rejected') {
					readErrors.runtime = getErrorMessage(runtimeResult.reason, 'Runtime status request failed.');
				}
				if (authHealthResult.status === 'rejected') {
					readErrors.auth = getErrorMessage(authHealthResult.reason, 'Auth health request failed.');
				}

				const runtime =
					runtimeResult.status === 'fulfilled'
						? getMatchingMCPServerRuntimeSnapshot(bundleID, server.id, runtimeResult.value)
						: undefined;
				let authHealth =
					authHealthResult.status === 'fulfilled'
						? getMatchingMCPAuthHealth(bundleID, server, authHealthResult.value)
						: undefined;

				const pending = pendingAuthorizations.find(
					authorization =>
						authorization.bundleID === bundleID &&
						authorization.serverID === server.id &&
						authorization.authorizationURL
				);
				if (pending) {
					authHealth = getPendingOAuthAuthHealth(bundleID, server.id, pending, authHealth);
				}

				return {
					serverID: server.id,
					runtime,
					authHealth,
					readErrors: Object.keys(readErrors).length > 0 ? readErrors : undefined,
				};
			});

			const runtimeByServerID: Record<string, MCPServerRuntimeSnapshot | undefined> = {};
			const authHealthByServerID: Record<string, MCPAuthHealth | undefined> = {};
			const readErrorsByServerID: Record<string, { runtime?: string; auth?: string } | undefined> = {};

			for (const entry of entries) {
				runtimeByServerID[entry.serverID] = entry.runtime;
				authHealthByServerID[entry.serverID] = entry.authHealth;
				readErrorsByServerID[entry.serverID] = entry.readErrors;
			}

			return {
				runtimeByServerID,
				authHealthByServerID,
				readErrorsByServerID,
				pendingAuthorizationReadError,
			};
		},
		[]
	);

	const loadServersForBundle = useCallback(
		async (bundleID: string, pendingAuthorizations?: MCPOAuthAuthorization[]) => {
			const servers = await getAllMCPServers(bundleID, undefined, undefined, true);
			const { runtimeByServerID, authHealthByServerID, readErrorsByServerID } = await loadRuntimeAndAuth(
				bundleID,
				servers,
				pendingAuthorizations
			);

			return {
				servers,
				runtimeByServerID,
				authHealthByServerID,
				readErrorsByServerID,
			};
		},
		[loadRuntimeAndAuth]
	);

	const refreshBundleServers = useCallback(
		async (bundleID: string) => {
			const fresh = await loadServersForBundle(bundleID);

			setBundles(prev =>
				prev.map(bundleData =>
					bundleData.bundle.id === bundleID
						? {
								...bundleData,
								servers: fresh.servers,
								runtimeByServerID: fresh.runtimeByServerID,
								authHealthByServerID: fresh.authHealthByServerID,
								readErrorsByServerID: fresh.readErrorsByServerID,
								serverLoadError: undefined,
							}
						: bundleData
				)
			);
		},
		[loadServersForBundle]
	);

	const loadMCPPageResource = useCallback(async (): Promise<MCPPageResource> => {
		const [bundlesResult, pendingResult, settingsResult] = await Promise.allSettled([
			getAllMCPBundles(undefined, true),
			mcpAPI.listPendingMCPOAuthAuthorizations(),
			mcpAPI.getMCPSettings(),
		]);

		if (bundlesResult.status === 'rejected') {
			throw bundlesResult.reason;
		}

		const pendingAuthorizations = pendingResult.status === 'fulfilled' ? pendingResult.value : [];
		const warnings = [
			pendingResult.status === 'rejected'
				? getErrorMessage(pendingResult.reason, 'Pending OAuth authorizations could not be loaded.')
				: undefined,
			settingsResult.status === 'rejected'
				? getErrorMessage(settingsResult.reason, 'MCP OAuth settings could not be loaded.')
				: undefined,
		].filter((warning): warning is string => Boolean(warning));

		const bundleData = await mapWithConcurrency(bundlesResult.value, MCP_BUNDLE_LOAD_CONCURRENCY, async bundle => {
			try {
				const loaded = await loadServersForBundle(bundle.id, pendingAuthorizations);
				return mergeBundleData(
					bundle,
					loaded.servers,
					loaded.runtimeByServerID,
					loaded.authHealthByServerID,
					loaded.readErrorsByServerID
				);
			} catch (error) {
				return mergeBundleData(
					bundle,
					[],
					{},
					{},
					{},
					getErrorMessage(error, 'Failed to load servers for this bundle.')
				);
			}
		});

		return {
			bundles: bundleData,
			settingsView: settingsResult.status === 'fulfilled' ? settingsResult.value : undefined,
			warnings,
		};
	}, [loadServersForBundle]);

	const refreshServerRuntimeAndAuth = useCallback(
		async (bundleID: string, serverID: string, knownPendingAuthorizations?: MCPOAuthAuthorization[]) => {
			const bundleData = bundlesRef.current.find(item => item.bundle.id === bundleID);
			const server = bundleData?.servers.find(candidate => candidate.id === serverID);

			if (!bundleData || !server) {
				throw new Error('MCP server not found.');
			}

			const refreshed = await loadRuntimeAndAuth(bundleID, [server], knownPendingAuthorizations);

			setBundles(previous =>
				previous.map(current =>
					current.bundle.id === bundleID
						? {
								...current,
								runtimeByServerID: {
									...current.runtimeByServerID,
									[serverID]: refreshed.runtimeByServerID[serverID],
								},
								authHealthByServerID: {
									...current.authHealthByServerID,
									[serverID]: refreshed.authHealthByServerID[serverID],
								},
								readErrorsByServerID: {
									...current.readErrorsByServerID,
									[serverID]: refreshed.readErrorsByServerID[serverID],
								},
							}
						: current
				)
			);

			if (refreshed.pendingAuthorizationReadError) {
				setResourceWarnings(previous =>
					[...new Set([...previous, refreshed.pendingAuthorizationReadError ?? ''])].filter(Boolean)
				);
			}
		},
		[loadRuntimeAndAuth]
	);

	const fetchAll = useCallback(async () => {
		const requestID = pageLoadRequestIDRef.current + 1;
		pageLoadRequestIDRef.current = requestID;
		const isInitialRequest = !hasLoadedPageRef.current;

		if (isInitialRequest) {
			setIsInitialLoading(true);
		} else {
			setIsRefreshing(true);
		}

		setPageLoadError(undefined);

		try {
			const resource = await loadMCPPageResource();
			if (!mountedRef.current || pageLoadRequestIDRef.current !== requestID) {
				return;
			}

			setBundles(resource.bundles);
			setSettingsView(resource.settingsView);
			setResourceWarnings(resource.warnings);
			hasLoadedPageRef.current = true;
		} catch (error) {
			if (!mountedRef.current || pageLoadRequestIDRef.current !== requestID) {
				return;
			}

			console.error('Failed to load MCP bundles:', error);
			setPageLoadError(error);
			setAlertMsg(getErrorMessage(error, 'Failed to load MCP bundles. Please try again.'));
			setShowAlert(true);
			throw error;
		} finally {
			if (mountedRef.current && pageLoadRequestIDRef.current === requestID) {
				if (isInitialRequest) {
					setIsInitialLoading(false);
				}

				setIsRefreshing(false);
			}
		}
	}, [loadMCPPageResource]);

	useEffect(() => {
		// oxlint-disable-next-line jsreact-hooks/set-state-in-effect
		void fetchAll().catch(() => undefined);
	}, [fetchAll]);

	const oauthServerPollKey = useMemo(
		() =>
			JSON.stringify(
				bundles.flatMap(bundleData =>
					bundleData.servers
						.filter(
							server =>
								bundleData.bundle.isEnabled &&
								server.enabled &&
								server.transport === MCPTransportType.MCPTransportTypeStreamableHTTP &&
								server.streamableHttp?.authMode === MCPHTTPAuthMode.MCPHTTPAuthOAuth
						)
						.map(server => [bundleData.bundle.id, server.id] as const)
				)
			),
		[bundles]
	);

	useEffect(() => {
		const oauthServers = (JSON.parse(oauthServerPollKey) as Array<[string, string]>).map(([bundleID, serverID]) => ({
			bundleID,
			serverID,
		}));

		if (oauthServers.length === 0) {
			return;
		}

		let polling = false;
		let cancelled = false;

		const pollRuntime = async () => {
			if (polling || cancelled) {
				return;
			}

			polling = true;
			try {
				const [pendingResult] = await Promise.allSettled([mcpAPI.listPendingMCPOAuthAuthorizations()]);
				const pendingAuthorizations = pendingResult.status === 'fulfilled' ? pendingResult.value : [];

				await mapWithConcurrency(oauthServers, MCP_STATUS_READ_CONCURRENCY, async ({ bundleID, serverID }) => {
					if (!cancelled) {
						await refreshServerRuntimeAndAuth(bundleID, serverID, pendingAuthorizations);
					}
				});
			} catch (error) {
				console.error('MCP runtime polling failed:', error);
			} finally {
				polling = false;
			}
		};

		void pollRuntime();
		const timer = window.setInterval(() => void pollRuntime(), 4000);

		return () => {
			cancelled = true;
			window.clearInterval(timer);
		};
	}, [oauthServerPollKey, refreshServerRuntimeAndAuth]);

	const handleToggleBundleEnabled = useCallback(
		async (bundleID: string, enabled: boolean) => {
			const bundleData = bundles.find(item => item.bundle.id === bundleID);

			if (!bundleData) {
				throw new Error('MCP bundle not found.');
			}

			await mcpAPI.patchMCPBundle(bundleID, enabled);

			setBundles(prev =>
				prev.map(item =>
					item.bundle.id === bundleID
						? {
								...item,
								bundle: {
									...item.bundle,
									isEnabled: enabled,
								},
							}
						: item
				)
			);

			await refreshBundleServers(bundleID);
		},
		[bundles, refreshBundleServers]
	);

	const handleToggleServerEnabled = useCallback(
		async (bundleID: string, serverID: string, enabled: boolean) => {
			const bundleData = bundles.find(item => item.bundle.id === bundleID);

			if (!bundleData) {
				throw new Error('MCP bundle not found.');
			}

			if (!bundleData.bundle.isEnabled) {
				throw new Error('Enable the MCP bundle before enabling or disabling servers.');
			}

			const server = bundleData.servers.find(item => item.id === serverID);

			if (!server) {
				throw new Error('MCP server not found.');
			}

			await mcpAPI.patchMCPServerEnabled(bundleID, serverID, enabled);

			setBundles(prev =>
				prev.map(item =>
					item.bundle.id === bundleID
						? {
								...item,
								servers: item.servers.map(existingServer =>
									existingServer.id === serverID
										? {
												...existingServer,
												enabled,
											}
										: existingServer
								),
							}
						: item
				)
			);

			await refreshServerRuntimeAndAuth(bundleID, serverID);
		},
		[bundles, refreshServerRuntimeAndAuth]
	);

	const handleDeleteServer = useCallback(
		async (bundleID: string, serverID: string) => {
			const bundleData = bundles.find(item => item.bundle.id === bundleID);

			if (!bundleData) {
				throw new Error('MCP bundle not found.');
			}

			if (bundleData.bundle.isBuiltIn) {
				throw new Error('Cannot delete servers from a built-in MCP bundle.');
			}

			const server = bundleData.servers.find(item => item.id === serverID);

			if (!server) {
				throw new Error('MCP server not found.');
			}

			if (server.isBuiltIn) {
				throw new Error('Cannot delete built-in MCP server.');
			}

			await mcpAPI.deleteMCPServer(bundleID, serverID);

			setBundles(prev =>
				prev.map(item =>
					item.bundle.id === bundleID
						? {
								...item,
								servers: item.servers.filter(existingServer => existingServer.id !== serverID),
							}
						: item
				)
			);
		},
		[bundles]
	);

	const handleSubmitServer = useCallback(
		async (bundleID: string, serverToEditID: string | undefined, input: MCPServerUpsertInput) => {
			const bundleData = bundles.find(item => item.bundle.id === bundleID);

			if (!bundleData) {
				throw new Error('MCP bundle not found.');
			}

			if (bundleData.bundle.isBuiltIn) {
				throw new Error('Cannot add or edit servers in a built-in MCP bundle.');
			}

			if (!bundleData.bundle.isEnabled) {
				throw new Error('Enable the MCP bundle before adding or editing servers.');
			}

			const serverToEdit =
				serverToEditID === undefined ? undefined : bundleData.servers.find(item => item.id === serverToEditID);

			if (serverToEditID !== undefined && !serverToEdit) {
				throw new Error('MCP server not found.');
			}

			if (serverToEdit?.isBuiltIn) {
				throw new Error('Built-in MCP servers cannot be edited.');
			}

			if (!serverToEdit && bundleData.servers.some(server => server.id === input.serverID)) {
				throw new Error(`MCP server "${input.serverID}" already exists in this bundle.`);
			}

			if (serverToEdit && input.serverID !== serverToEdit.id) {
				throw new Error('MCP server ID cannot be changed.');
			}

			const isCreatingServer = serverToEditID === undefined;
			if (isCreatingServer || !input.initialPayload) {
				await mcpAPI.putMCPServer(bundleID, input.serverID, input.initialPayload ?? input.payload);
			}

			const finalPayload = clonePayload(input.payload);
			let requiresFinalPut = Boolean(input.initialPayload);

			if (finalPayload.stdio) {
				let refs: Record<string, string> = { ...finalPayload.stdio.secretEnvRefs };

				for (const row of input.stdioSecretEnv) {
					const envName = row.envName.trim();
					const slot = row.slot.trim();
					const deleteSlot = (row.deleteSlot ?? slot).trim();

					if (!envName || !slot) {
						continue;
					}

					if (row.deleteExisting && row.existingSecretRef && deleteSlot) {
						await mcpAPI.deleteMCPServerSecret(
							bundleID,
							input.serverID,
							MCPSecretKind.MCPSecretKindStdioEnv,
							deleteSlot
						);
						refs = omitManyKeys(refs, [envName, deleteSlot]);
						requiresFinalPut = true;
					}

					if (row.secretValue && row.secretValue.length > 0) {
						const resp = await mcpAPI.putMCPServerSecret(
							bundleID,
							input.serverID,
							MCPSecretKind.MCPSecretKindStdioEnv,
							slot,
							row.secretValue
						);

						if (!resp?.secretRef) {
							throw new Error(`Secret for ${envName} was saved but no secret reference was returned.`);
						}

						refs[envName] = resp.secretRef;
						requiresFinalPut = true;
					}
				}

				finalPayload.stdio.secretEnvRefs = Object.keys(refs).length > 0 ? refs : undefined;
			}

			if (finalPayload.streamableHttp && input.oauthClientCredentials) {
				const plan = input.oauthClientCredentials;

				if (plan.deleteExisting && plan.existingSecretRef) {
					await mcpAPI.deleteMCPServerSecret(
						bundleID,
						input.serverID,
						MCPSecretKind.MCPSecretKindOAuthClientCredentials,
						plan.slot
					);
					finalPayload.streamableHttp.clientCredentialRef = undefined;
					requiresFinalPut = true;
				}

				if (plan.secretValue && plan.secretValue.length > 0) {
					const resp = await mcpAPI.putMCPServerSecret(
						bundleID,
						input.serverID,
						MCPSecretKind.MCPSecretKindOAuthClientCredentials,
						plan.slot,
						plan.secretValue
					);

					if (!resp?.secretRef) {
						throw new Error('OAuth client credentials were saved but no secret reference was returned.');
					}

					finalPayload.streamableHttp.clientCredentialRef = resp.secretRef;
					requiresFinalPut = true;
				}
			}
			if (finalPayload.streamableHttp && input.httpHeaderSecret) {
				const plan = input.httpHeaderSecret;
				let refs: Record<string, string> = { ...finalPayload.streamableHttp.secretHeaderRefs };

				if (plan.deleteExisting && plan.existingSecretRef) {
					const deleteSlot = (plan.deleteSlot ?? plan.slot).trim();

					await mcpAPI.deleteMCPServerSecret(
						bundleID,
						input.serverID,
						MCPSecretKind.MCPSecretKindHTTPHeader,
						deleteSlot
					);
					refs = omitManyKeys(refs, [plan.headerName, deleteSlot]);
					requiresFinalPut = true;
				}

				if (plan.secretValue && plan.secretValue.length > 0) {
					const resp = await mcpAPI.putMCPServerSecret(
						bundleID,
						input.serverID,
						MCPSecretKind.MCPSecretKindHTTPHeader,
						plan.slot,
						plan.secretValue
					);

					if (!resp?.secretRef) {
						throw new Error('API key was saved but no secret reference was returned.');
					}

					refs[plan.headerName] = resp.secretRef;
					requiresFinalPut = true;
				}

				finalPayload.streamableHttp.secretHeaderRefs = Object.keys(refs).length > 0 ? refs : undefined;
			}
			if (requiresFinalPut) {
				await mcpAPI.putMCPServer(bundleID, input.serverID, finalPayload);
			}

			await refreshBundleServers(bundleID);
		},
		[bundles, refreshBundleServers]
	);
	const handleSubmitServerSetup = useCallback(
		async (
			bundleID: string,
			serverID: string,
			inputValues: Record<string, MCPServerSetupInputValue>,
			reset: boolean
		) => {
			await mcpAPI.patchMCPServerSetup(bundleID, serverID, inputValues, reset);
			await refreshBundleServers(bundleID);
		},
		[refreshBundleServers]
	);
	const handleConnectServer = useCallback(
		async (bundleID: string, serverID: string) => {
			const connectResult = withTimeout(
				mcpAPI.connectMCPServer(bundleID, serverID),
				MCP_CONNECT_TIMEOUT_MS,
				`Connecting MCP server "${serverID}" timed out after ${MCP_CONNECT_TIMEOUT_MS / 1000} seconds.`
			).then(
				snapshot => ({ snapshot }),
				(error: unknown) => ({ error })
			);

			try {
				while (true) {
					const result = await Promise.race([connectResult, sleep(1000).then(() => undefined)]);

					if (result) {
						if ('error' in result) {
							throw result.error;
						}

						if (result.snapshot) {
							setBundles(previous =>
								previous.map(bundleData =>
									bundleData.bundle.id === bundleID
										? {
												...bundleData,
												runtimeByServerID: {
													...bundleData.runtimeByServerID,
													[serverID]: result.snapshot,
												},
											}
										: bundleData
								)
							);
						}
						break;
					}

					await refreshServerRuntimeAndAuth(bundleID, serverID);
				}
			} finally {
				await refreshServerRuntimeAndAuth(bundleID, serverID);
			}
		},
		[refreshServerRuntimeAndAuth]
	);

	const handleDisconnectServer = useCallback(
		async (bundleID: string, serverID: string) => {
			await mcpAPI.disconnectMCPServer(bundleID, serverID);
			await refreshServerRuntimeAndAuth(bundleID, serverID);
		},
		[refreshServerRuntimeAndAuth]
	);

	const handleRefreshServer = useCallback(
		async (bundleID: string, serverID: string) => {
			const snapshot = await mcpAPI.refreshMCPServer(bundleID, serverID);

			if (snapshot) {
				setBundles(prev =>
					prev.map(bundleData =>
						bundleData.bundle.id === bundleID
							? {
									...bundleData,
									runtimeByServerID: {
										...bundleData.runtimeByServerID,
										[serverID]: snapshot,
									},
								}
							: bundleData
					)
				);
			}

			await refreshServerRuntimeAndAuth(bundleID, serverID);
		},
		[refreshServerRuntimeAndAuth]
	);

	const handleCancelOAuth = useCallback(
		async (bundleID: string, serverID: string) => {
			await mcpAPI.cancelPendingMCPOAuthAuthorization(bundleID, serverID);
			await refreshServerRuntimeAndAuth(bundleID, serverID);
		},
		[refreshServerRuntimeAndAuth]
	);

	const handleBundleDelete = useCallback(async () => {
		if (!bundleToDeleteID || isDeletingBundle) {
			return;
		}

		if (bundleToDeleteID === BaseMCPBundleID) {
			setAlertMsg('The base MCP bundle cannot be deleted.');
			setShowAlert(true);
			setBundleToDeleteID(null);
			return;
		}

		const bundleData = bundles.find(item => item.bundle.id === bundleToDeleteID);
		if (!bundleData || bundleData.serverLoadError || bundleData.servers.length > 0) {
			setAlertMsg(
				bundleData?.serverLoadError
					? 'Reload this bundle before deleting it.'
					: 'Remove all MCP servers before deleting this bundle.'
			);
			setShowAlert(true);
			setBundleToDeleteID(null);
			return;
		}

		setIsDeletingBundle(true);
		try {
			await mcpAPI.deleteMCPBundle(bundleToDeleteID);

			setBundles(prev => prev.filter(b => b.bundle.id !== bundleToDeleteID));
		} catch (error) {
			console.error('Delete MCP bundle failed:', error);
			setAlertMsg(getErrorMessage(error, 'Failed to delete MCP bundle.'));
			setShowAlert(true);
		} finally {
			setIsDeletingBundle(false);
			setBundleToDeleteID(null);
		}
	}, [bundleToDeleteID, bundles, isDeletingBundle]);

	const handleAddBundle = useCallback(
		async (slug: string, display: string, description?: string) => {
			const id = getUUIDv7();
			await mcpAPI.putMCPBundle(id, slug, display, true, description);
			try {
				await fetchAll();
			} catch (error) {
				console.error('MCP bundle was created but refresh failed:', error);
				setAlertMsg(
					'MCP bundle was created, but the page could not be refreshed. Reload the page before making destructive changes.'
				);
				setShowAlert(true);
			}
		},
		[fetchAll]
	);

	const allServerConfigs = useMemo(() => bundles.flatMap(bundleData => bundleData.servers), [bundles]);
	const allServerIDs = bundles.flatMap(bundleData => bundleData.servers.map(server => server.id));

	const selectedOAuthAuthorization = useMemo(() => {
		if (!oauthAuthorizationTarget) {
			return null;
		}

		const bundleData = bundles.find(item => item.bundle.id === oauthAuthorizationTarget.bundleID);
		const server = bundleData?.servers.find(candidate => candidate.id === oauthAuthorizationTarget.serverID);

		if (!bundleData || !server) {
			return null;
		}

		return {
			bundle: bundleData.bundle,
			server,
			authHealth: bundleData.authHealthByServerID[server.id],
		};
	}, [bundles, oauthAuthorizationTarget]);

	const requestOAuthAuthorization = useCallback((bundleID: string, serverID: string) => {
		const bundleData = bundlesRef.current.find(item => item.bundle.id === bundleID);
		const server = bundleData?.servers.find(candidate => candidate.id === serverID);
		const authHealth = server ? bundleData?.authHealthByServerID[server.id] : undefined;

		if (!server || !isOAuthAuthorizationRelevant(server, authHealth)) {
			setAlertMsg(
				'OAuth authorization is no longer available for this MCP server. Refresh the server status and try again.'
			);
			setShowAlert(true);
			return;
		}

		setOAuthAuthorizationTarget({ bundleID, serverID });
	}, []);

	const handleSaveSettings = useCallback(async (oauthLoopbackListenAddr: string) => {
		const view = await mcpAPI.patchMCPSettings(oauthLoopbackListenAddr);
		if (view) {
			setSettingsView(view);
		}
		if (view?.oauthRestartRequired) {
			setAlertMsg('The OAuth loopback address was saved and will take effect after restarting FlexiGPT.');
			setShowAlert(true);
		}
	}, []);
	if (isInitialLoading) {
		return <Loader text="Loading MCP servers…" />;
	}

	return (
		<PageFrame>
			<div className="flex size-full flex-col items-center overflow-hidden">
				<ManagementPageHeader
					title="MCP Servers"
					description="Configure MCP transports, authorization, discovery, policy, and setup."
					width="wide"
					leadingActions={
						<button
							type="button"
							className="btn btn-ghost rounded-xl"
							onClick={() => {
								setIsSettingsOpen(true);
							}}
							title="MCP OAuth settings"
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
								setIsAddModalOpen(true);
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

					{resourceWarnings.map(warning => (
						<div key={warning} className="alert alert-warning rounded-2xl text-sm">
							<span>{warning}</span>
						</div>
					))}

					{bundles.length === 0 && <p className="mt-8 text-center text-sm">No MCP bundles configured yet.</p>}

					{bundles.map(bundleData => (
						<MCPBundleCard
							key={bundleData.bundle.id}
							bundle={bundleData.bundle}
							servers={bundleData.servers}
							existingServerIDs={allServerIDs}
							prefillServers={allServerConfigs}
							runtimeByServerID={bundleData.runtimeByServerID}
							authHealthByServerID={bundleData.authHealthByServerID}
							readErrorsByServerID={bundleData.readErrorsByServerID}
							serverLoadError={bundleData.serverLoadError}
							onRefreshServers={() => {
								return refreshBundleServers(bundleData.bundle.id);
							}}
							onToggleBundleEnabled={handleToggleBundleEnabled}
							onToggleServerEnabled={handleToggleServerEnabled}
							onSubmitServer={handleSubmitServer}
							onSubmitServerSetup={handleSubmitServerSetup}
							onDeleteServer={handleDeleteServer}
							onConnectServer={handleConnectServer}
							onDisconnectServer={handleDisconnectServer}
							onRefreshServer={handleRefreshServer}
							onCancelOAuth={handleCancelOAuth}
							onRequestOAuthAuthorization={requestOAuthAuthorization}
							onDeleteBundleRequested={bundleID => {
								if (bundleID === BaseMCPBundleID) {
									setAlertMsg('The base MCP bundle cannot be deleted.');
									setShowAlert(true);
									return;
								}
								setBundleToDeleteID(bundleID);
							}}
						/>
					))}
				</ManagementPageContent>

				<DeleteConfirmationModal
					isOpen={bundleToDelete !== null}
					onClose={() => {
						if (!isDeletingBundle) {
							setBundleToDeleteID(null);
						}
					}}
					onConfirm={handleBundleDelete}
					title="Delete MCP Bundle"
					message={`Delete empty MCP bundle "${bundleToDelete?.displayName ?? ''}"? Remove all servers first.`}
					confirmButtonText={isDeletingBundle ? 'Deleting...' : 'Delete'}
				/>

				<ManagementBundleCreateModal
					isOpen={isAddModalOpen}
					title="Add MCP Bundle"
					entityLabel="MCP bundle"
					onClose={() => {
						setIsAddModalOpen(false);
					}}
					onSubmit={handleAddBundle}
					existingSlugs={bundles.map(bundleData => bundleData.bundle.slug)}
					failureMessage="Failed to create MCP bundle."
				/>
				<MCPSettingsModal
					isOpen={isSettingsOpen}
					initialListenAddr={settingsView?.settings.oauthLoopbackListenAddr}
					activeListenAddr={settingsView?.oauthLoopbackListenAddr}
					oauthRedirectURL={settingsView?.oauthRedirectURL}
					onClose={() => {
						setIsSettingsOpen(false);
					}}
					onSubmit={handleSaveSettings}
				/>
				<MCPOAuthAuthorizationModal
					isOpen={selectedOAuthAuthorization !== null}
					server={selectedOAuthAuthorization?.server ?? null}
					authHealth={selectedOAuthAuthorization?.authHealth}
					onClose={() => {
						setOAuthAuthorizationTarget(null);
					}}
					onOpenURL={url => {
						backendAPI.openURL(url);
					}}
					onCancel={async () => {
						if (!selectedOAuthAuthorization) {
							return;
						}

						try {
							await handleCancelOAuth(selectedOAuthAuthorization.bundle.id, selectedOAuthAuthorization.server.id);
							setOAuthAuthorizationTarget(null);
						} catch (error) {
							setAlertMsg(getErrorMessage(error, 'Failed to cancel OAuth authorization.'));
							setShowAlert(true);
						}
					}}
				/>
				<ActionDeniedAlertModal
					isOpen={showAlert}
					onClose={() => {
						setShowAlert(false);
						setAlertMsg('');
					}}
					message={alertMsg}
				/>
			</div>
		</PageFrame>
	);
}
