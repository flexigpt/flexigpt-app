import type { ReactNode } from 'react';
import { useMemo, useState } from 'react';

import {
	FiChevronDown,
	FiChevronUp,
	FiEdit2,
	FiExternalLink,
	FiEye,
	FiPlus,
	FiRefreshCw,
	FiSettings,
	FiTrash2,
	FiWifi,
	FiWifiOff,
	FiX,
} from 'react-icons/fi';

import type {
	MCPAuthHealth,
	MCPBundle,
	MCPServerConfig,
	MCPServerRuntimeSnapshot,
	MCPServerSetupInputValue,
} from '@/spec/mcp';
import { BaseMCPBundleID, MCPAuthHealthState, MCPServerStatus } from '@/spec/mcp';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';

import type { MCPServerUpsertInput } from '@/mcpservers/lib/mcp_server_utils';
import {
	getEffectiveMCPServerStatus,
	getMCPAuthHealthBadgeClass,
	getMCPAuthHealthLabel,
	getMCPServerSetupStatus,
	getMCPStatusBadgeClass,
	getMCPStatusLabel,
	getMCPTransportLabel,
	getMCPTrustLevelLabel,
	isMCPAuthActionable,
	serverHasSetupInputs,
} from '@/mcpservers/lib/mcp_server_utils';
import { MCPOAuthAuthorizationModal } from '@/mcpservers/mcp_oauth_authorization_modal';
import { AddEditMCPServerModal } from '@/mcpservers/mcp_server_add_edit_modal';
import { MCPServerDetailsModal } from '@/mcpservers/mcp_server_details_modal';
import { MCPServerSetupModal } from '@/mcpservers/mcp_server_setup_modal';

type ServerModalMode = 'add' | 'edit';

interface MCPBundleCardProps {
	bundle: MCPBundle;
	servers: MCPServerConfig[];
	existingServerIDs: string[];
	prefillServers: MCPServerConfig[];
	runtimeByServerID: Record<string, MCPServerRuntimeSnapshot | undefined>;
	authHealthByServerID: Record<string, MCPAuthHealth | undefined>;

	onToggleBundleEnabled: (bundleID: string, enabled: boolean) => Promise<void>;
	onToggleServerEnabled: (bundleID: string, serverID: string, enabled: boolean) => Promise<void>;
	onSubmitServer: (bundleID: string, serverToEditID: string | undefined, input: MCPServerUpsertInput) => Promise<void>;
	onSubmitServerSetup: (
		bundleID: string,
		serverID: string,
		inputValues: Record<string, MCPServerSetupInputValue>,
		reset: boolean
	) => Promise<void>;
	onDeleteServer: (bundleID: string, serverID: string) => Promise<void>;
	onConnectServer: (bundleID: string, serverID: string) => Promise<void>;
	onDisconnectServer: (bundleID: string, serverID: string) => Promise<void>;
	onRefreshServer: (bundleID: string, serverID: string) => Promise<void>;
	onOpenURL: (url: string) => void;
	onCancelOAuth: (bundleID: string, serverID: string) => Promise<void>;
	onDeleteBundleRequested: (bundleID: string) => void;
}

function isOAuthModalRelevant(authHealth?: MCPAuthHealth): boolean {
	return (
		isMCPAuthActionable(authHealth) || authHealth?.state === MCPAuthHealthState.MCPAuthHealthStateAuthorizationPending
	);
}

function oauthDismissKey(bundleID: string, serverID: string, authHealth?: MCPAuthHealth): string {
	return `${bundleID}:${serverID}:${authHealth?.authorizationURL ?? authHealth?.state ?? ''}`;
}

function getErrorMessage(error: unknown, fallback: string): string {
	if (error instanceof Error && error.message.trim().length > 0) {
		return error.message;
	}
	return fallback;
}

function InfoPill({ label, children, className = '' }: { label: string; children: ReactNode; className?: string }) {
	return (
		<div className={`bg-base-200 flex min-w-0 rounded-xl p-2 ${className}`}>
			<div className="grid size-full grid-cols-3 items-center gap-4">
				<div className="text-base-content/60 col-span-1 text-[10px] uppercase">{label}</div>
				<div className="col-span-2 text-xs">{children}</div>
			</div>
		</div>
	);
}

function TruncatedValue({ value }: { value: ReactNode }) {
	return (
		<div className="min-w-0 truncate" title={typeof value === 'string' ? value : undefined}>
			{value}
		</div>
	);
}

export function MCPBundleCard({
	bundle,
	servers,
	existingServerIDs,
	prefillServers,
	runtimeByServerID,
	authHealthByServerID,
	onToggleBundleEnabled,
	onToggleServerEnabled,
	onSubmitServer,
	onSubmitServerSetup,
	onDeleteServer,
	onConnectServer,
	onDisconnectServer,
	onRefreshServer,
	onOpenURL,
	onCancelOAuth,
	onDeleteBundleRequested,
}: MCPBundleCardProps) {
	const [isExpanded, setIsExpanded] = useState(false);
	const isReservedBundle = bundle.id === BaseMCPBundleID;

	const [isDeleteServerModalOpen, setIsDeleteServerModalOpen] = useState(false);
	const [serverToDelete, setServerToDelete] = useState<MCPServerConfig | null>(null);

	const [isServerModalOpen, setIsServerModalOpen] = useState(false);
	const [serverModalMode, setServerModalMode] = useState<ServerModalMode>('add');
	const [serverToEdit, setServerToEdit] = useState<MCPServerConfig | undefined>(undefined);

	const [serverDetails, setServerDetails] = useState<MCPServerConfig | null>(null);
	const [setupServer, setSetupServer] = useState<MCPServerConfig | null>(null);
	const [showAlert, setShowAlert] = useState(false);
	const [alertMsg, setAlertMsg] = useState('');

	const [isBundleTogglePending, setIsBundleTogglePending] = useState(false);
	const [pendingActionKeys, setPendingActionKeys] = useState<Set<string>>(() => new Set());
	const [manualOAuthModalServerID, setManualOAuthModalServerID] = useState<string | null>(null);
	const [dismissedOAuthKeys, setDismissedOAuthKeys] = useState<Set<string>>(() => new Set());

	const openAlert = (message: string) => {
		setAlertMsg(message);
		setShowAlert(true);
	};

	const setActionPending = (key: string, pending: boolean) => {
		setPendingActionKeys(prev => {
			const next = new Set(prev);

			if (pending) {
				next.add(key);
			} else {
				next.delete(key);
			}

			return next;
		});
	};

	const runServerAction = async (key: string, action: () => Promise<void>, fallback: string) => {
		try {
			setActionPending(key, true);
			await action();
		} catch (error) {
			console.error(fallback, error);
			openAlert(getErrorMessage(error, fallback));
		} finally {
			setActionPending(key, false);
		}
	};

	const handleToggleBundleEnable = async () => {
		if (isReservedBundle) {
			openAlert('The base MCP bundle metadata is reserved and cannot be disabled.');
			return;
		}
		try {
			setIsBundleTogglePending(true);
			await onToggleBundleEnabled(bundle.id, !bundle.isEnabled);
		} catch (error) {
			console.error('Failed to toggle MCP bundle:', error);
			openAlert(getErrorMessage(error, 'Failed to toggle MCP bundle enable state.'));
		} finally {
			setIsBundleTogglePending(false);
		}
	};

	const handleServerEnableToggle = async (server: MCPServerConfig) => {
		if (!bundle.isEnabled) {
			openAlert('Enable the MCP bundle before enabling or disabling servers.');
			return;
		}

		await runServerAction(
			`toggle:${server.id}`,
			() => onToggleServerEnabled(bundle.id, server.id, !server.enabled),
			'Failed to toggle MCP server.'
		);
	};

	const requestDeleteServer = (server: MCPServerConfig) => {
		if (bundle.isBuiltIn) {
			openAlert('Cannot delete servers from a built-in MCP bundle.');
			return;
		}

		if (server.isBuiltIn) {
			openAlert('Cannot delete built-in MCP server.');
			return;
		}

		setServerToDelete(server);
		setIsDeleteServerModalOpen(true);
	};

	const confirmDeleteServer = async () => {
		if (!serverToDelete) {
			return;
		}

		try {
			await onDeleteServer(bundle.id, serverToDelete.id);
		} catch (error) {
			console.error('Delete MCP server failed:', error);
			openAlert(getErrorMessage(error, 'Failed to delete MCP server.'));
		} finally {
			setIsDeleteServerModalOpen(false);
			setServerToDelete(null);
		}
	};

	const openServerModal = (mode: ServerModalMode, server?: MCPServerConfig) => {
		if (bundle.isBuiltIn) {
			openAlert('Cannot add or edit servers in a built-in MCP bundle.');
			return;
		}

		if (!bundle.isEnabled) {
			openAlert('Enable the MCP bundle before adding or editing servers.');
			return;
		}

		if (mode === 'edit' && server?.isBuiltIn) {
			openAlert('Built-in MCP servers cannot be edited.');
			return;
		}

		setServerModalMode(mode);
		setServerToEdit(server);
		setIsServerModalOpen(true);
	};

	const handleModifySubmit = async (input: MCPServerUpsertInput) => {
		await onSubmitServer(bundle.id, serverToEdit?.id, input);
	};
	const manualOAuthModalServer = useMemo(() => {
		if (!manualOAuthModalServerID) {
			return null;
		}

		const server = servers.find(candidate => candidate.id === manualOAuthModalServerID);
		if (!server) {
			return null;
		}

		const authHealth = authHealthByServerID[server.id];
		return isOAuthModalRelevant(authHealth) ? server : null;
	}, [authHealthByServerID, manualOAuthModalServerID, servers]);

	const autoOAuthModalServer = useMemo(() => {
		if (manualOAuthModalServer) {
			return null;
		}

		return (
			servers.find(server => {
				const authHealth = authHealthByServerID[server.id];
				if (!isOAuthModalRelevant(authHealth)) {
					return false;
				}

				const key = oauthDismissKey(bundle.id, server.id, authHealth);
				return !dismissedOAuthKeys.has(key);
			}) ?? null
		);
	}, [authHealthByServerID, bundle.id, dismissedOAuthKeys, manualOAuthModalServer, servers]);

	const oauthModalServer = manualOAuthModalServer ?? autoOAuthModalServer;
	const oauthModalAuthHealth = oauthModalServer ? authHealthByServerID[oauthModalServer.id] : undefined;

	const dismissOAuthModal = () => {
		if (oauthModalServer) {
			const authHealth = authHealthByServerID[oauthModalServer.id];
			const key = oauthDismissKey(bundle.id, oauthModalServer.id, authHealth);
			setDismissedOAuthKeys(prev => new Set(prev).add(key));
		}

		setManualOAuthModalServerID(null);
	};

	const cancelOAuthModal = async () => {
		if (!oauthModalServer) {
			return;
		}

		await runServerAction(
			`cancel-oauth:${oauthModalServer.id}`,
			() => onCancelOAuth(bundle.id, oauthModalServer.id),
			'Failed to cancel OAuth authorization.'
		);
		dismissOAuthModal();
	};
	return (
		<div className="bg-base-100 mb-8 rounded-2xl p-4 shadow-lg">
			<div className="flex items-center justify-between">
				<div className="flex items-center">
					<h3 className="gap-2 text-sm font-semibold">
						<span className="capitalize">{bundle.displayName || bundle.slug}</span>
						<span className="text-base-content/60 ml-1">({bundle.slug})</span>
					</h3>
				</div>

				<div className="flex items-center justify-end gap-4">
					<span className="text-base-content/60 text-xs tracking-wide uppercase">
						{bundle.isBuiltIn ? 'Built-in' : isReservedBundle ? 'Base' : 'Custom'}
					</span>

					<div className="flex items-center gap-1">
						<label className="text-sm">Enabled</label>
						<input
							type="checkbox"
							className="toggle toggle-accent"
							checked={bundle.isEnabled}
							disabled={isBundleTogglePending || isReservedBundle}
							title={isReservedBundle ? 'The base MCP bundle is always enabled.' : undefined}
							onChange={() => {
								void handleToggleBundleEnable();
							}}
						/>
					</div>

					<div
						className="flex cursor-pointer items-center gap-1"
						onClick={() => {
							setIsExpanded(prev => !prev);
						}}
					>
						<label className="text-sm whitespace-nowrap">Servers:&nbsp;{servers.length}</label>
						{isExpanded ? <FiChevronUp /> : <FiChevronDown />}
					</div>
				</div>
			</div>

			{isExpanded && (
				<div className="mt-8 space-y-4">
					{servers.length === 0 ? (
						<div className="border-base-content/10 rounded-2xl border py-6 text-center text-sm">
							No MCP servers in this bundle.
						</div>
					) : (
						<div className="grid grid-cols-1 gap-4 xl:grid-cols-3">
							{servers.map(server => {
								const runtime = runtimeByServerID[server.id];
								const authHealth = authHealthByServerID[server.id];
								const status = getEffectiveMCPServerStatus(server.enabled, bundle.isEnabled, runtime);
								const isReady = status === MCPServerStatus.MCPServerStatusReady;
								const isConnecting = status === MCPServerStatus.MCPServerStatusConnecting;
								const authActionable = isMCPAuthActionable(authHealth);
								const setupStatus = getMCPServerSetupStatus(server);
								const setupIncomplete = setupStatus.hasInputs && !setupStatus.complete;
								return (
									<article
										key={server.id}
										className="border-base-content/10 bg-base-100 min-w-0 rounded-2xl border p-4 shadow-sm"
									>
										<div className="mb-2 flex min-w-0 items-start justify-between gap-3">
											<div className="flex min-w-0 items-center gap-2">
												<div className="truncate text-sm font-semibold" title={server.displayName}>
													{server.displayName}
												</div>
												<div className="text-base-content/60 text-xs break-all">{server.id}</div>
											</div>

											<div className="flex shrink-0 items-end gap-2">
												<span className={`badge badge-xs rounded-xl ${getMCPStatusBadgeClass(status)}`}>
													{getMCPStatusLabel(status)}
												</span>
												{setupStatus.hasInputs && (
													<span
														className={`badge badge-xs rounded-xl ${setupIncomplete ? 'badge-warning' : 'badge-ghost'}`}
														title={
															setupIncomplete
																? `Setup required: ${setupStatus.requiredConfigured}/${setupStatus.requiredTotal} configured`
																: 'Setup complete'
														}
													>
														{setupIncomplete ? 'Setup needed' : 'Setup ✓'}
													</span>
												)}
												<span
													className={`badge badge-xs rounded-xl ${getMCPAuthHealthBadgeClass(authHealth?.state)}`}
													title={authHealth?.lastError || getMCPAuthHealthLabel(authHealth?.state)}
												>
													{getMCPAuthHealthLabel(authHealth?.state)}
												</span>
											</div>
										</div>

										<div className="grid grid-cols-2 gap-2">
											<InfoPill label="Enabled">
												<input
													type="checkbox"
													className="toggle toggle-accent toggle-sm"
													checked={server.enabled}
													disabled={pendingActionKeys.has(`toggle:${server.id}`) || !bundle.isEnabled}
													title={!bundle.isEnabled ? 'Enable the bundle first.' : undefined}
													onChange={() => {
														void handleServerEnableToggle(server);
													}}
												/>
											</InfoPill>

											<InfoPill label="Transport">
												<TruncatedValue value={getMCPTransportLabel(server.transport)} />
											</InfoPill>

											<InfoPill label="Trust">
												<TruncatedValue value={getMCPTrustLevelLabel(server.trustLevel)} />
											</InfoPill>

											<InfoPill label="Kind">{server.isBuiltIn ? 'Built-in' : 'Custom'}</InfoPill>

											<InfoPill label="Discovery" className="col-span-2">
												{runtime
													? `${runtime.toolCount} tools / ${runtime.resourceCount} resources / ${runtime.resourceTemplateCount} templates / ${runtime.promptCount} prompts`
													: '-'}
											</InfoPill>
										</div>

										<div className="mt-4 flex flex-col flex-wrap items-center">
											<div className="flex w-full flex-col gap-2">
												{runtime?.lastError && (
													<div className="text-error truncate text-xs" title={runtime.lastError}>
														{runtime.lastError}
													</div>
												)}

												{(authActionable ||
													authHealth?.state === MCPAuthHealthState.MCPAuthHealthStateAuthorizationPending) && (
													<div className="border-info/20 bg-info/10 flex flex-wrap items-center justify-between gap-2 rounded-2xl border p-2">
														<div className="min-w-0 text-xs">
															<div className="font-semibold">OAuth authorization required</div>
															<div className="text-base-content/70">
																Open the browser authorization page, complete login, then return here.
															</div>
														</div>

														<div className="flex shrink-0 flex-wrap gap-1">
															{authActionable && (
																<button
																	className="btn btn-xs btn-primary rounded-xl"
																	onClick={() => {
																		setManualOAuthModalServerID(server.id);
																	}}
																	title="Authorize MCP server"
																>
																	<FiExternalLink size={12} />
																	<span className="ml-1">Authorize</span>
																</button>
															)}

															{authHealth?.state === MCPAuthHealthState.MCPAuthHealthStateAuthorizationPending && (
																<button
																	className="btn btn-xs bg-base-300 rounded-xl"
																	onClick={() => {
																		void runServerAction(
																			`cancel-oauth:${server.id}`,
																			() => onCancelOAuth(bundle.id, server.id),
																			'Failed to cancel OAuth authorization.'
																		);
																	}}
																	title="Cancel authorization"
																>
																	<FiX size={12} />
																	<span className="ml-1">Cancel</span>
																</button>
															)}
														</div>
													</div>
												)}

												{authHealth?.lastError && (
													<div className="text-error truncate text-xs" title={authHealth.lastError}>
														{authHealth.lastError}
													</div>
												)}
											</div>
											<div className="flex w-full items-center justify-end gap-2">
												<button
													className="btn btn-sm btn-ghost rounded-2xl"
													onClick={() => {
														setServerDetails(server);
													}}
													title="View"
													aria-label="View"
												>
													<FiEye size={16} />
												</button>
												{serverHasSetupInputs(server) && (
													<button
														className="btn btn-sm btn-ghost rounded-2xl"
														onClick={() => {
															if (!bundle.isEnabled) {
																openAlert('Enable the MCP bundle before configuring servers.');
																return;
															}
															setSetupServer(server);
														}}
														disabled={!bundle.isEnabled}
														title="Configure setup"
														aria-label="Configure setup"
													>
														<FiSettings size={16} />
													</button>
												)}
												<button
													className="btn btn-sm btn-ghost rounded-2xl"
													onClick={() => {
														openServerModal('edit', server);
													}}
													disabled={server.isBuiltIn || bundle.isBuiltIn || !bundle.isEnabled}
													title={
														server.isBuiltIn || bundle.isBuiltIn
															? 'Built-in items cannot be edited'
															: !bundle.isEnabled
																? 'Enable the bundle first.'
																: 'Edit'
													}
													aria-label="Edit"
												>
													<FiEdit2 size={16} />
												</button>

												<button
													className="btn btn-sm btn-ghost rounded-2xl"
													onClick={() => {
														void runServerAction(
															`connect:${server.id}`,
															() => onConnectServer(bundle.id, server.id),
															'Failed to connect MCP server.'
														);
													}}
													disabled={
														!bundle.isEnabled ||
														!server.enabled ||
														isReady ||
														isConnecting ||
														setupIncomplete ||
														pendingActionKeys.has(`connect:${server.id}`)
													}
													title={
														setupIncomplete
															? 'Complete required setup before connecting.'
															: authActionable
																? 'Authorization pending. Open auth URL first if needed.'
																: 'Connect'
													}
													aria-label="Connect"
												>
													<FiWifi size={16} />
												</button>

												<button
													className="btn btn-sm btn-ghost rounded-2xl"
													onClick={() => {
														void runServerAction(
															`disconnect:${server.id}`,
															() => onDisconnectServer(bundle.id, server.id),
															'Failed to disconnect MCP server.'
														);
													}}
													disabled={
														!isReady ||
														pendingActionKeys.has(`disconnect:${server.id}`) ||
														pendingActionKeys.has(`connect:${server.id}`)
													}
													title="Disconnect"
													aria-label="Disconnect"
												>
													<FiWifiOff size={16} />
												</button>

												<button
													className="btn btn-sm btn-ghost rounded-2xl"
													onClick={() => {
														void runServerAction(
															`refresh:${server.id}`,
															() => onRefreshServer(bundle.id, server.id),
															'Failed to refresh MCP server.'
														);
													}}
													disabled={
														!bundle.isEnabled ||
														!server.enabled ||
														!isReady ||
														isConnecting ||
														pendingActionKeys.has(`refresh:${server.id}`)
													}
													title="Refresh discovery"
													aria-label="Refresh discovery"
												>
													<FiRefreshCw size={16} />
												</button>

												<button
													className="btn btn-sm btn-ghost rounded-2xl"
													onClick={() => {
														requestDeleteServer(server);
													}}
													disabled={server.isBuiltIn || bundle.isBuiltIn}
													title={
														server.isBuiltIn || bundle.isBuiltIn ? 'Deleting disabled for built-in items' : 'Delete'
													}
													aria-label="Delete"
												>
													<FiTrash2 size={16} />
												</button>
											</div>
										</div>
									</article>
								);
							})}
						</div>
					)}

					{!bundle.isBuiltIn && (
						<div className="flex items-center justify-between">
							<button
								className="btn btn-md btn-ghost flex items-center rounded-2xl"
								disabled={isReservedBundle || servers.length > 0}
								title={
									isReservedBundle
										? 'The base MCP bundle cannot be deleted.'
										: servers.length > 0
											? 'Delete all servers from this bundle first.'
											: 'Delete Bundle'
								}
								onClick={() => {
									if (isReservedBundle) {
										openAlert('The base MCP bundle cannot be deleted.');
										return;
									}
									onDeleteBundleRequested(bundle.id);
								}}
							>
								<FiTrash2 /> <span className="ml-1">Delete Bundle</span>
							</button>

							<button
								className="btn btn-md btn-ghost flex items-center rounded-2xl"
								disabled={!bundle.isEnabled}
								title={!bundle.isEnabled ? 'Enable the bundle first.' : 'Add MCP Server'}
								onClick={() => {
									openServerModal('add');
								}}
							>
								<FiPlus /> <span className="ml-1">Add Server</span>
							</button>
						</div>
					)}
				</div>
			)}

			<DeleteConfirmationModal
				isOpen={isDeleteServerModalOpen}
				onClose={() => {
					setIsDeleteServerModalOpen(false);
					setServerToDelete(null);
				}}
				onConfirm={confirmDeleteServer}
				title="Delete MCP Server"
				message={`Delete MCP server "${serverToDelete?.displayName ?? ''}"? This cannot be undone.`}
				confirmButtonText="Delete"
			/>

			<AddEditMCPServerModal
				isOpen={isServerModalOpen}
				onClose={() => {
					setIsServerModalOpen(false);
					setServerToEdit(undefined);
				}}
				onSubmit={handleModifySubmit}
				mode={serverModalMode}
				initialData={serverToEdit}
				existingServerIDs={existingServerIDs}
				prefillServers={prefillServers}
			/>

			<MCPServerDetailsModal
				isOpen={serverDetails !== null}
				onClose={() => {
					setServerDetails(null);
				}}
				bundle={bundle}
				server={serverDetails}
				runtime={serverDetails ? runtimeByServerID[serverDetails.id] : undefined}
				authHealth={serverDetails ? authHealthByServerID[serverDetails.id] : undefined}
			/>
			<MCPServerSetupModal
				isOpen={setupServer !== null}
				server={setupServer}
				onClose={() => {
					setSetupServer(null);
				}}
				onSubmit={async (inputValues, reset) => {
					if (!setupServer) {
						return;
					}
					await onSubmitServerSetup(bundle.id, setupServer.id, inputValues, reset);
				}}
			/>
			<MCPOAuthAuthorizationModal
				isOpen={oauthModalServer !== null}
				onClose={dismissOAuthModal}
				server={oauthModalServer}
				authHealth={oauthModalAuthHealth}
				onOpenURL={onOpenURL}
				onCancel={cancelOAuthModal}
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
	);
}
