import { useCallback, useEffect, useState } from 'react';

import { FiPlus } from 'react-icons/fi';

import {
	type MCPAuthHealth,
	type MCPBundle,
	MCPSecretKind,
	type MCPServerConfig,
	type MCPServerRuntimeSnapshot,
	type PutMCPServerPayload,
} from '@/spec/mcp';

import { omitManyKeys } from '@/lib/obj_utils';
import { getUUIDv7 } from '@/lib/uuid_utils';

import { backendAPI, mcpAPI } from '@/apis/baseapi';
import { getAllMCPBundles, getAllMCPServers } from '@/apis/list_helper';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';
import { Loader } from '@/components/loader';
import { PageFrame } from '@/components/page_frame';

import type { MCPServerUpsertInput } from '@/mcpservers/lib/mcp_server_utils';
import { AddMCPBundleModal } from '@/mcpservers/mcp_bundle_add_modal';
import { MCPBundleCard } from '@/mcpservers/mcp_bundle_card';

interface BundleData {
	bundle: MCPBundle;
	servers: MCPServerConfig[];
	runtimeByServerID: Record<string, MCPServerRuntimeSnapshot | undefined>;
	authHealthByServerID: Record<string, MCPAuthHealth | undefined>;
}

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
	authHealthByServerID: Record<string, MCPAuthHealth | undefined>
): BundleData {
	return {
		bundle,
		servers,
		runtimeByServerID,
		authHealthByServerID,
	};
}

function sleep(ms: number): Promise<void> {
	return new Promise(resolve => window.setTimeout(resolve, ms));
}

// eslint-disable-next-line no-restricted-exports
export default function MCPServersPage() {
	const [bundles, setBundles] = useState<BundleData[]>([]);
	const [loading, setLoading] = useState(true);

	const [showAlert, setShowAlert] = useState(false);
	const [alertMsg, setAlertMsg] = useState('');

	const [bundleToDeleteID, setBundleToDeleteID] = useState<string | null>(null);
	const [isAddModalOpen, setIsAddModalOpen] = useState(false);

	const bundleToDelete =
		bundleToDeleteID === null
			? null
			: (bundles.find(bundleData => bundleData.bundle.id === bundleToDeleteID)?.bundle ?? null);

	const loadRuntimeAndAuth = useCallback(async (bundleID: string, servers: MCPServerConfig[]) => {
		const entries = await Promise.all(
			servers.map(async server => {
				const [runtime, authHealth] = await Promise.all([
					mcpAPI.getMCPServerStatus(bundleID, server.id).catch(() => undefined),
					mcpAPI.getMCPServerAuthHealth(bundleID, server.id).catch(() => undefined),
				]);

				return {
					serverID: server.id,
					runtime,
					authHealth,
				};
			})
		);

		const runtimeByServerID: Record<string, MCPServerRuntimeSnapshot | undefined> = {};
		const authHealthByServerID: Record<string, MCPAuthHealth | undefined> = {};

		for (const entry of entries) {
			runtimeByServerID[entry.serverID] = entry.runtime;
			authHealthByServerID[entry.serverID] = entry.authHealth;
		}

		return {
			runtimeByServerID,
			authHealthByServerID,
		};
	}, []);

	const loadServersForBundle = useCallback(
		async (bundleID: string) => {
			const servers = await getAllMCPServers(bundleID, undefined, undefined, true);
			const { runtimeByServerID, authHealthByServerID } = await loadRuntimeAndAuth(bundleID, servers);

			return {
				servers,
				runtimeByServerID,
				authHealthByServerID,
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
							}
						: bundleData
				)
			);
		},
		[loadServersForBundle]
	);

	const refreshServerRuntimeAndAuth = useCallback(async (bundleID: string, serverID: string) => {
		const [runtime, authHealth] = await Promise.all([
			mcpAPI.getMCPServerStatus(bundleID, serverID).catch(() => undefined),
			mcpAPI.getMCPServerAuthHealth(bundleID, serverID).catch(() => undefined),
		]);

		setBundles(prev =>
			prev.map(bundleData =>
				bundleData.bundle.id === bundleID
					? {
							...bundleData,
							runtimeByServerID: {
								...bundleData.runtimeByServerID,
								[serverID]: runtime,
							},
							authHealthByServerID: {
								...bundleData.authHealthByServerID,
								[serverID]: authHealth,
							},
						}
					: bundleData
			)
		);
	}, []);

	const fetchAll = useCallback(async () => {
		setLoading(true);

		try {
			const mcpBundles = await getAllMCPBundles(undefined, true);

			const bundleResults: BundleData[] = await Promise.all(
				mcpBundles.map(async bundle => {
					try {
						const { servers, runtimeByServerID, authHealthByServerID } = await loadServersForBundle(bundle.id);
						return mergeBundleData(bundle, servers, runtimeByServerID, authHealthByServerID);
					} catch {
						return mergeBundleData(bundle, [], {}, {});
					}
				})
			);

			setBundles(bundleResults);
		} catch (error) {
			console.error('Failed to load MCP bundles:', error);
			setAlertMsg(getErrorMessage(error, 'Failed to load MCP bundles. Please try again.'));
			setShowAlert(true);
		} finally {
			setLoading(false);
		}
	}, [loadServersForBundle]);

	useEffect(() => {
		// eslint-disable-next-line react-hooks/set-state-in-effect
		void fetchAll();
	}, [fetchAll]);

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

			await mcpAPI.putMCPServer(bundleID, input.serverID, input.initialPayload ?? input.payload);

			const finalPayload = clonePayload(input.payload);
			let requiresFinalPut = Boolean(input.initialPayload);

			if (finalPayload.stdio) {
				let refs: Record<string, string> = { ...(finalPayload.stdio.secretEnvRefs ?? {}) };

				for (const row of input.stdioSecretEnv) {
					const envName = row.envName.trim();
					const slot = row.slot.trim();

					if (!envName || !slot) continue;

					if (row.deleteExisting && row.existingSecretRef) {
						await mcpAPI.deleteMCPServerSecret(bundleID, input.serverID, MCPSecretKind.MCPSecretKindStdioEnv, slot);
						refs = omitManyKeys(refs, [envName]);
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

			if (requiresFinalPut) {
				await mcpAPI.putMCPServer(bundleID, input.serverID, finalPayload);
			}

			await refreshBundleServers(bundleID);
		},
		[bundles, refreshBundleServers]
	);

	const handleConnectServer = useCallback(
		async (bundleID: string, serverID: string) => {
			let settled = false;

			const connectPromise = mcpAPI.connectMCPServer(bundleID, serverID).finally(() => {
				settled = true;
			});

			while (!settled) {
				await Promise.race([connectPromise.catch(() => undefined), sleep(1000)]);

				if (!settled) {
					await refreshServerRuntimeAndAuth(bundleID, serverID).catch(() => undefined);
				}
			}

			const snapshot = await connectPromise;
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
		if (!bundleToDeleteID) {
			return;
		}

		try {
			await mcpAPI.deleteMCPBundle(bundleToDeleteID);

			setBundles(prev => prev.filter(bundleData => bundleData.bundle.id !== bundleToDeleteID));
		} catch (error) {
			console.error('Delete MCP bundle failed:', error);
			setAlertMsg(getErrorMessage(error, 'Failed to delete MCP bundle.'));
			setShowAlert(true);
		} finally {
			setBundleToDeleteID(null);
		}
	}, [bundleToDeleteID]);

	const handleAddBundle = useCallback(
		async (slug: string, display: string, description?: string) => {
			try {
				const id = getUUIDv7();
				await mcpAPI.putMCPBundle(id, slug, display, true, description);
				setIsAddModalOpen(false);
				await fetchAll();
			} catch (error) {
				console.error('Add MCP bundle failed:', error);
				setAlertMsg(getErrorMessage(error, 'Failed to add MCP bundle.'));
				setShowAlert(true);
			}
		},
		[fetchAll]
	);

	if (loading) {
		return <Loader text="Loading MCP servers…" />;
	}

	return (
		<PageFrame>
			<div className="flex h-full w-full flex-col items-center">
				<div className="fixed mt-8 flex w-11/12 items-center px-12 py-2">
					<h1 className="flex grow items-center justify-center text-xl font-semibold">MCP Servers</h1>
					<button
						className="btn btn-ghost flex items-center rounded-2xl"
						onClick={() => {
							setIsAddModalOpen(true);
						}}
					>
						<FiPlus size={20} /> <span className="ml-1">Add Bundle</span>
					</button>
				</div>

				<div
					className="mt-24 flex w-full grow flex-col items-center overflow-y-auto"
					style={{ maxHeight: `calc(100vh - 128px)` }}
				>
					<div className="flex w-11/12 flex-col space-y-4 xl:w-5/6">
						{bundles.length === 0 && <p className="mt-8 text-center text-sm">No MCP bundles configured yet.</p>}

						{bundles.map(bundleData => (
							<MCPBundleCard
								key={bundleData.bundle.id}
								bundle={bundleData.bundle}
								servers={bundleData.servers}
								runtimeByServerID={bundleData.runtimeByServerID}
								authHealthByServerID={bundleData.authHealthByServerID}
								onToggleBundleEnabled={handleToggleBundleEnabled}
								onToggleServerEnabled={handleToggleServerEnabled}
								onSubmitServer={handleSubmitServer}
								onDeleteServer={handleDeleteServer}
								onConnectServer={handleConnectServer}
								onDisconnectServer={handleDisconnectServer}
								onRefreshServer={handleRefreshServer}
								onOpenURL={url => {
									backendAPI.openURL(url);
								}}
								onCancelOAuth={handleCancelOAuth}
								onDeleteBundleRequested={bundleID => {
									setBundleToDeleteID(bundleID);
								}}
							/>
						))}
					</div>
				</div>

				<DeleteConfirmationModal
					isOpen={bundleToDelete !== null}
					onClose={() => {
						setBundleToDeleteID(null);
					}}
					onConfirm={handleBundleDelete}
					title="Delete MCP Bundle"
					message={`Delete empty MCP bundle "${bundleToDelete?.displayName ?? ''}"? Remove all servers first.`}
					confirmButtonText="Delete"
				/>

				<AddMCPBundleModal
					isOpen={isAddModalOpen}
					onClose={() => {
						setIsAddModalOpen(false);
					}}
					onSubmit={handleAddBundle}
					existingSlugs={bundles.map(bundleData => bundleData.bundle.slug)}
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
