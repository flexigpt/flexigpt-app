import { useState } from 'react';

import {
	FiChevronDown,
	FiChevronUp,
	FiEdit2,
	FiExternalLink,
	FiEye,
	FiPlus,
	FiRefreshCw,
	FiTrash2,
	FiWifi,
	FiWifiOff,
	FiX,
} from 'react-icons/fi';

import {
	type MCPAuthHealth,
	MCPAuthHealthState,
	type MCPBundle,
	type MCPServerConfig,
	type MCPServerRuntimeSnapshot,
	MCPServerStatus,
} from '@/spec/mcp';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';

import {
	getEffectiveMCPServerStatus,
	getMCPAuthHealthBadgeClass,
	getMCPAuthHealthLabel,
	getMCPAvailabilityLabel,
	getMCPStatusBadgeClass,
	getMCPStatusLabel,
	getMCPTransportLabel,
	getMCPTrustLevelLabel,
	isMCPAuthActionable,
	type MCPServerUpsertInput,
} from '@/mcpservers/lib/mcp_server_utils';
import { AddEditMCPServerModal } from '@/mcpservers/mcp_server_add_edit_modal';
import { MCPServerDetailsModal } from '@/mcpservers/mcp_server_details_modal';

type ServerModalMode = 'add' | 'edit';

interface MCPBundleCardProps {
	bundle: MCPBundle;
	servers: MCPServerConfig[];
	runtimeByServerID: Record<string, MCPServerRuntimeSnapshot | undefined>;
	authHealthByServerID: Record<string, MCPAuthHealth | undefined>;

	onToggleBundleEnabled: (bundleID: string, enabled: boolean) => Promise<void>;
	onToggleServerEnabled: (bundleID: string, serverID: string, enabled: boolean) => Promise<void>;
	onSubmitServer: (bundleID: string, serverToEditID: string | undefined, input: MCPServerUpsertInput) => Promise<void>;
	onDeleteServer: (bundleID: string, serverID: string) => Promise<void>;
	onConnectServer: (bundleID: string, serverID: string) => Promise<void>;
	onDisconnectServer: (bundleID: string, serverID: string) => Promise<void>;
	onRefreshServer: (bundleID: string, serverID: string) => Promise<void>;
	onOpenURL: (url: string) => void;
	onCancelOAuth: (bundleID: string, serverID: string) => Promise<void>;
	onDeleteBundleRequested: (bundleID: string) => void;
}

function getErrorMessage(error: unknown, fallback: string): string {
	if (error instanceof Error && error.message.trim().length > 0) {
		return error.message;
	}
	return fallback;
}

export function MCPBundleCard({
	bundle,
	servers,
	runtimeByServerID,
	authHealthByServerID,
	onToggleBundleEnabled,
	onToggleServerEnabled,
	onSubmitServer,
	onDeleteServer,
	onConnectServer,
	onDisconnectServer,
	onRefreshServer,
	onOpenURL,
	onCancelOAuth,
	onDeleteBundleRequested,
}: MCPBundleCardProps) {
	const [isExpanded, setIsExpanded] = useState(false);

	const [isDeleteServerModalOpen, setIsDeleteServerModalOpen] = useState(false);
	const [serverToDelete, setServerToDelete] = useState<MCPServerConfig | null>(null);

	const [isServerModalOpen, setIsServerModalOpen] = useState(false);
	const [serverModalMode, setServerModalMode] = useState<ServerModalMode>('add');
	const [serverToEdit, setServerToEdit] = useState<MCPServerConfig | undefined>(undefined);

	const [serverDetails, setServerDetails] = useState<MCPServerConfig | null>(null);

	const [showAlert, setShowAlert] = useState(false);
	const [alertMsg, setAlertMsg] = useState('');

	const [isBundleTogglePending, setIsBundleTogglePending] = useState(false);
	const [pendingActionKeys, setPendingActionKeys] = useState<Set<string>>(() => new Set());

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
						{bundle.isBuiltIn ? 'Built-in' : 'Custom'}
					</span>

					<div className="flex items-center gap-1">
						<label className="text-sm">Enabled</label>
						<input
							type="checkbox"
							className="toggle toggle-accent"
							checked={bundle.isEnabled}
							disabled={isBundleTogglePending}
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
					<div className="border-base-content/10 overflow-x-auto rounded-2xl border">
						<table className="table-zebra table w-full">
							<thead>
								<tr className="bg-base-300 text-sm font-semibold">
									<th className="w-full">Display Name</th>
									<th className="text-center">ID</th>
									<th className="text-center whitespace-nowrap">Enabled</th>
									<th className="text-center whitespace-nowrap">Transport</th>
									<th className="text-center whitespace-nowrap">Status</th>
									<th className="text-center whitespace-nowrap">Auth</th>
									<th className="text-center whitespace-nowrap">Discovery</th>
									<th className="text-center whitespace-nowrap">Availability</th>
									<th className="text-center whitespace-nowrap">Trust</th>
									<th className="text-center whitespace-nowrap">Built-In</th>
									<th className="text-center whitespace-nowrap">Actions</th>
								</tr>
							</thead>
							<tbody>
								{servers.map(server => {
									const runtime = runtimeByServerID[server.id];
									const authHealth = authHealthByServerID[server.id];
									const status = getEffectiveMCPServerStatus(server.enabled, bundle.isEnabled, runtime);
									const isReady = status === MCPServerStatus.MCPServerStatusReady;
									const isConnecting = status === MCPServerStatus.MCPServerStatusConnecting;
									const authActionable = isMCPAuthActionable(authHealth);

									return (
										<tr key={server.id} className="hover:bg-base-300">
											<td>{server.displayName}</td>
											<td className="text-center">{server.id}</td>
											<td className="text-center align-middle">
												<input
													type="checkbox"
													className="toggle toggle-accent"
													checked={server.enabled}
													disabled={pendingActionKeys.has(`toggle:${server.id}`) || !bundle.isEnabled}
													title={!bundle.isEnabled ? 'Enable the bundle first.' : undefined}
													onChange={() => {
														void handleServerEnableToggle(server);
													}}
												/>
											</td>
											<td className="text-center">{getMCPTransportLabel(server.transport)}</td>
											<td className="text-center">
												<span className={`badge rounded-xl ${getMCPStatusBadgeClass(status)}`}>
													{getMCPStatusLabel(status)}
												</span>
												{runtime?.lastError && (
													<div className="text-error mt-1 max-w-xs truncate text-xs" title={runtime.lastError}>
														{runtime.lastError}
													</div>
												)}
											</td>
											<td className="text-center">
												<span className={`badge rounded-xl ${getMCPAuthHealthBadgeClass(authHealth?.state)}`}>
													{getMCPAuthHealthLabel(authHealth?.state)}
												</span>
												{authHealth?.state === MCPAuthHealthState.MCPAuthHealthStateAuthorizationPending &&
													authHealth.authorizationURL && (
														<div className="mt-1 flex justify-center gap-1">
															<button
																className="btn btn-xs btn-ghost rounded-xl"
																onClick={() => {
																	onOpenURL(authHealth.authorizationURL ?? '');
																}}
																title="Open authorization URL"
															>
																<FiExternalLink size={12} />
															</button>
															<button
																className="btn btn-xs btn-ghost rounded-xl"
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
															</button>
														</div>
													)}
												{authHealth?.lastError && (
													<div className="text-error mt-1 max-w-xs truncate text-xs" title={authHealth.lastError}>
														{authHealth.lastError}
													</div>
												)}
											</td>
											<td className="text-center whitespace-nowrap">
												{runtime ? `${runtime.toolCount}T / ${runtime.resourceCount}R / ${runtime.promptCount}P` : '-'}
											</td>
											<td className="text-center">{getMCPAvailabilityLabel(server.availability)}</td>
											<td className="text-center">{getMCPTrustLevelLabel(server.trustLevel)}</td>
											<td className="text-center">{server.isBuiltIn ? 'Yes' : 'No'}</td>
											<td className="justify-end text-center">
												<div className="inline-flex items-center gap-2">
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
															pendingActionKeys.has(`connect:${server.id}`)
														}
														title={authActionable ? 'Authorization pending. Open auth URL first if needed.' : 'Connect'}
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
											</td>
										</tr>
									);
								})}

								{servers.length === 0 && (
									<tr>
										<td colSpan={11} className="py-3 text-center text-sm">
											No MCP servers in this bundle.
										</td>
									</tr>
								)}
							</tbody>
						</table>
					</div>

					{!bundle.isBuiltIn && (
						<div className="flex items-center justify-between">
							<button
								className="btn btn-md btn-ghost flex items-center rounded-2xl"
								disabled={servers.length > 0}
								title={servers.length > 0 ? 'Delete all servers from this bundle first.' : 'Delete Bundle'}
								onClick={() => {
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
				existingServerIDs={servers.map(server => server.id)}
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
