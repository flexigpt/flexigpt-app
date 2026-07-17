import { useState } from 'react';

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

import { usePendingActions } from '@/hooks/use_pending_actions';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';
import { ActionRow } from '@/components/managementui/action_row';
import { EnabledControl } from '@/components/managementui/enabled_control';
import { ManagementBundleCard } from '@/components/managementui/management_bundle_card';
import { ManagementEmptyState } from '@/components/managementui/management_empty_state';
import { ManagementItemCard } from '@/components/managementui/management_item_card';
import { MetadataPill } from '@/components/managementui/metadata_pill';
import { StatusBadge } from '@/components/managementui/status_badge';

import type { MCPServerUpsertInput } from '@/mcpservers/lib/mcp_server_utils';
import {
	getEffectiveMCPAuthHealthState,
	getEffectiveMCPServerStatus,
	getMCPServerAuthHealthBadgeClass,
	getMCPServerAuthHealthLabel,
	getMCPServerSetupStatus,
	getMCPStatusBadgeClass,
	getMCPStatusLabel,
	getMCPTransportLabel,
	getMCPTrustLevelLabel,
	isMCPAuthActionable,
	serverHasSetupInputs,
} from '@/mcpservers/lib/mcp_server_utils';
import { MCPBundleDetailsModal } from '@/mcpservers/mcp_bundle_details_modal';
import { AddEditMCPServerModal } from '@/mcpservers/mcp_server_add_edit_modal';
import { MCPServerDetailsModal } from '@/mcpservers/mcp_server_details_modal';
import { MCPServerSetupModal } from '@/mcpservers/mcp_server_setup_modal';

type ServerModalMode = 'add' | 'edit';

interface MCPServerReadErrors {
	runtime?: string;
	auth?: string;
}

interface MCPBundleCardProps {
	bundle: MCPBundle;
	servers: MCPServerConfig[];
	existingServerIDs: string[];
	prefillServers: MCPServerConfig[];
	runtimeByServerID: Record<string, MCPServerRuntimeSnapshot | undefined>;
	authHealthByServerID: Record<string, MCPAuthHealth | undefined>;
	readErrorsByServerID?: Record<string, MCPServerReadErrors | undefined>;

	serverLoadError?: string;
	onRefreshServers: () => Promise<void>;
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
	onCancelOAuth: (bundleID: string, serverID: string) => Promise<void>;
	onDeleteBundleRequested: (bundleID: string) => void;
	onRequestOAuthAuthorization: (bundleID: string, serverID: string) => void;
}

function getAuthHealthTitle(server: MCPServerConfig, authHealth: MCPAuthHealth | undefined, label: string): string {
	const serverAuthMode = server.streamableHttp?.authMode ?? 'none';

	const parts = [
		authHealth?.lastError || label,
		`serverAuthMode=${serverAuthMode}`,
		`healthAuthMode=${authHealth?.authMode ?? 'unknown'}`,
		`healthState=${authHealth?.state ?? 'unknown'}`,
		`configured=${authHealth?.configured ?? 'unknown'}`,
		authHealth?.bundleID ? `healthBundleID=${authHealth.bundleID}` : undefined,
		authHealth?.serverID ? `healthServerID=${authHealth.serverID}` : undefined,
		authHealth?.resource ? `resource=${authHealth.resource}` : undefined,
	].filter(Boolean);

	return parts.join('\n');
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
	existingServerIDs,
	prefillServers,
	runtimeByServerID,
	authHealthByServerID,
	readErrorsByServerID = {},
	serverLoadError,
	onRefreshServers,
	onToggleBundleEnabled,
	onToggleServerEnabled,
	onSubmitServer,
	onSubmitServerSetup,
	onDeleteServer,
	onConnectServer,
	onDisconnectServer,
	onRefreshServer,
	onCancelOAuth,
	onDeleteBundleRequested,
	onRequestOAuthAuthorization,
}: MCPBundleCardProps) {
	const [isExpanded, setIsExpanded] = useState(false);
	const isReservedBundle = bundle.id === BaseMCPBundleID;

	const [isDeleteServerModalOpen, setIsDeleteServerModalOpen] = useState(false);
	const [serverToDelete, setServerToDelete] = useState<MCPServerConfig | null>(null);

	const [isServerModalOpen, setIsServerModalOpen] = useState(false);
	const [serverModalMode, setServerModalMode] = useState<ServerModalMode>('add');
	const [serverToEdit, setServerToEdit] = useState<MCPServerConfig | undefined>(undefined);

	const [serverDetails, setServerDetails] = useState<MCPServerConfig | null>(null);
	const [isBundleDetailsOpen, setIsBundleDetailsOpen] = useState(false);
	const [setupServer, setSetupServer] = useState<MCPServerConfig | null>(null);

	const [showAlert, setShowAlert] = useState(false);
	const [alertMsg, setAlertMsg] = useState('');

	const { isPending, runAction } = usePendingActions();

	const openAlert = (message: string) => {
		setAlertMsg(message);
		setShowAlert(true);
	};

	const runActionWithAlert = async (key: string, action: () => Promise<void>, fallback: string) => {
		try {
			await runAction(key, action);
		} catch (error) {
			console.error(fallback, error);
			openAlert(getErrorMessage(error, fallback));
			throw error;
		}
	};

	const refreshServers = () => {
		void runActionWithAlert('bundle:refresh', onRefreshServers, 'Failed to reload MCP servers.').catch(() => undefined);
	};

	const handleToggleBundleEnable = async (enabled: boolean) => {
		if (isReservedBundle) {
			openAlert('The base MCP bundle metadata is reserved and cannot be disabled.');
			return;
		}

		await runActionWithAlert(
			'bundle:toggle',
			() => onToggleBundleEnabled(bundle.id, enabled),
			'Failed to toggle MCP bundle enable state.'
		);
	};

	const handleServerEnableToggle = async (server: MCPServerConfig, enabled: boolean) => {
		if (!bundle.isEnabled) {
			openAlert('Enable the MCP bundle before enabling or disabling servers.');
			return;
		}

		await runActionWithAlert(
			`${server.id}:toggle`,
			() => onToggleServerEnabled(bundle.id, server.id, enabled),
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
			await runActionWithAlert(
				`${serverToDelete.id}:delete`,
				() => onDeleteServer(bundle.id, serverToDelete.id),
				'Failed to delete MCP server.'
			);
			setIsDeleteServerModalOpen(false);
			setServerToDelete(null);
		} catch {
			// Keep the confirmation dialog open so the user can retry or cancel.
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
		await runAction(`${serverToEdit?.id ?? 'new'}:save`, () => onSubmitServer(bundle.id, serverToEdit?.id, input));
	};

	return (
		<>
			<ManagementBundleCard
				title={bundle.displayName || bundle.slug}
				identity={
					<span className="font-mono">
						{bundle.slug} / {bundle.id}
					</span>
				}
				description={bundle.description}
				status={
					<>
						<StatusBadge tone={bundle.isEnabled ? 'success' : 'neutral'}>
							{bundle.isEnabled ? 'Enabled' : 'Disabled'}
						</StatusBadge>
						<StatusBadge>{bundle.isBuiltIn ? 'Built-in' : isReservedBundle ? 'Base' : 'Custom'}</StatusBadge>
					</>
				}
				disclosure={
					<button
						type="button"
						className="btn btn-sm btn-ghost rounded-xl"
						aria-expanded={isExpanded}
						onClick={() => {
							setIsExpanded(previous => !previous);
						}}
					>
						<span className="whitespace-nowrap">Servers: {servers.length}</span>
						{isExpanded ? <FiChevronUp /> : <FiChevronDown />}
					</button>
				}
				actionLeading={
					<EnabledControl
						id={`mcp-bundle-${bundle.id}`}
						checked={bundle.isEnabled}
						onChange={enabled => {
							void handleToggleBundleEnable(enabled).catch(() => undefined);
						}}
						disabled={isReservedBundle}
						busy={isPending('bundle:toggle')}
						compact={false}
						title={isReservedBundle ? 'The base MCP bundle is always enabled.' : undefined}
					/>
				}
				actions={
					<>
						<button
							type="button"
							className="btn btn-sm btn-ghost rounded-xl"
							onClick={() => {
								setIsBundleDetailsOpen(true);
							}}
							title="View MCP bundle details"
						>
							<FiEye size={16} />
							<span>Details</span>
						</button>
						{!bundle.isBuiltIn ? (
							<>
								<button
									type="button"
									className="btn btn-sm btn-ghost rounded-xl"
									disabled={!bundle.isEnabled || Boolean(serverLoadError)}
									onClick={() => {
										openServerModal('add');
									}}
								>
									<FiPlus size={16} />
									<span>Add Server</span>
								</button>
								<button
									type="button"
									className="btn btn-sm btn-ghost rounded-xl"
									disabled={isReservedBundle || servers.length > 0 || Boolean(serverLoadError)}
									onClick={() => {
										onDeleteBundleRequested(bundle.id);
									}}
								>
									<FiTrash2 size={16} />
									<span>Delete Bundle</span>
								</button>
							</>
						) : null}
					</>
				}
			>
				{serverLoadError ? (
					<output className="alert alert-warning mt-3 rounded-2xl text-sm">
						<span className="min-w-0 grow">
							<span className="block font-semibold">Servers could not be loaded</span>
							<span className="block wrap-break-word">{serverLoadError}</span>
						</span>
						<button
							type="button"
							className="btn btn-sm rounded-xl"
							onClick={refreshServers}
							disabled={isPending('bundle:refresh')}
						>
							<FiRefreshCw size={14} />
							<span>{isPending('bundle:refresh') ? 'Reloading' : 'Retry'}</span>
						</button>
					</output>
				) : null}

				{isExpanded && (
					<div className="mt-6 space-y-3">
						{servers.length === 0 ? (
							<ManagementEmptyState>
								{serverLoadError ? 'Server contents are unavailable.' : 'No MCP servers in this bundle.'}
							</ManagementEmptyState>
						) : (
							<div className="space-y-3">
								{servers.map(server => {
									const runtime = runtimeByServerID[server.id];
									const authHealth = authHealthByServerID[server.id];
									const readErrors = readErrorsByServerID[server.id];
									const status = getEffectiveMCPServerStatus(server.enabled, bundle.isEnabled, runtime);
									const isReady = status === MCPServerStatus.MCPServerStatusReady;
									const isConnecting = status === MCPServerStatus.MCPServerStatusConnecting;
									const authState = getEffectiveMCPAuthHealthState(server, authHealth);
									const authActionable = isMCPAuthActionable(authHealth, server);
									const setupStatus = getMCPServerSetupStatus(server);
									const setupIncomplete = setupStatus.hasInputs && !setupStatus.complete;
									const authLabel = getMCPServerAuthHealthLabel(server, authHealth);
									const authTitle = getAuthHealthTitle(server, authHealth, authLabel);

									return (
										<ManagementItemCard
											key={server.id}
											title={server.displayName}
											subtitle={server.id}
											status={
												<>
													<StatusBadge className={getMCPStatusBadgeClass(status)}>
														{getMCPStatusLabel(status)}
													</StatusBadge>
													{setupStatus.hasInputs && (
														<StatusBadge
															tone={setupIncomplete ? 'warning' : 'neutral'}
															title={
																setupIncomplete
																	? `Setup required: ${setupStatus.requiredConfigured}/${setupStatus.requiredTotal} configured`
																	: 'Setup complete'
															}
														>
															{setupIncomplete ? 'Setup needed' : 'Setup ✓'}
														</StatusBadge>
													)}
													<StatusBadge
														className={getMCPServerAuthHealthBadgeClass(server, authHealth)}
														title={authTitle}
													>
														{authLabel}
													</StatusBadge>
												</>
											}
											metadata={
												<>
													<MetadataPill label="Transport">{getMCPTransportLabel(server.transport)}</MetadataPill>
													<MetadataPill label="Trust">{getMCPTrustLevelLabel(server.trustLevel)}</MetadataPill>
													<MetadataPill label="Tools">{runtime ? runtime.toolCount : '—'}</MetadataPill>
													<MetadataPill label="Resources">{runtime ? runtime.resourceCount : '—'}</MetadataPill>
													<MetadataPill label="Templates">{runtime ? runtime.resourceTemplateCount : '—'}</MetadataPill>
													<MetadataPill label="Prompts">{runtime ? runtime.promptCount : '—'}</MetadataPill>
												</>
											}
										>
											<div className="mt-3 space-y-1">
												{readErrors?.runtime ? (
													<div className="text-warning text-xs">
														Runtime status could not be read: {readErrors.runtime}
													</div>
												) : null}
												{runtime?.lastError ? (
													<div className="text-error text-xs" title={runtime.lastError}>
														{runtime.lastError}
													</div>
												) : null}
												{readErrors?.auth ? (
													<div className="text-warning text-xs">Auth health could not be read: {readErrors.auth}</div>
												) : null}
												{authHealth?.lastError ? (
													<div className="text-error text-xs" title={authHealth.lastError}>
														{authHealth.lastError}
													</div>
												) : null}
											</div>

											<ActionRow
												leading={
													<EnabledControl
														id={`mcp-server-${bundle.id}-${server.id}`}
														checked={server.enabled}
														onChange={enabled => {
															void handleServerEnableToggle(server, enabled).catch(() => undefined);
														}}
														disabled={!bundle.isEnabled}
														busy={isPending(`${server.id}:toggle`)}
														title={!bundle.isEnabled ? 'Enable the bundle first.' : undefined}
													/>
												}
											>
												<button
													type="button"
													className="btn btn-sm btn-ghost rounded-xl"
													onClick={() => {
														setServerDetails(server);
													}}
												>
													<FiEye size={15} />
													<span>View</span>
												</button>
												{serverHasSetupInputs(server) ? (
													<button
														type="button"
														className="btn btn-sm btn-ghost rounded-xl"
														disabled={!bundle.isEnabled}
														onClick={() => {
															setSetupServer(server);
														}}
													>
														<FiSettings size={15} />
														<span>Setup</span>
													</button>
												) : null}
												<button
													type="button"
													className="btn btn-sm btn-ghost rounded-xl"
													disabled={server.isBuiltIn || bundle.isBuiltIn || !bundle.isEnabled}
													onClick={() => {
														openServerModal('edit', server);
													}}
												>
													<FiEdit2 size={15} />
													<span>Edit</span>
												</button>
												{authActionable ? (
													<button
														type="button"
														className="btn btn-sm btn-ghost rounded-xl"
														onClick={() => {
															onRequestOAuthAuthorization(bundle.id, server.id);
														}}
													>
														<FiExternalLink size={15} />
														<span>Authorize</span>
													</button>
												) : null}
												{authState === MCPAuthHealthState.MCPAuthHealthStateAuthorizationPending ? (
													<button
														type="button"
														className="btn btn-sm btn-ghost rounded-xl"
														disabled={isPending(`${server.id}:cancel-oauth`)}
														onClick={() => {
															void runActionWithAlert(
																`${server.id}:cancel-oauth`,
																() => onCancelOAuth(bundle.id, server.id),
																'Failed to cancel OAuth authorization.'
															).catch(() => undefined);
														}}
													>
														<FiX size={15} />
														<span>Cancel authorization</span>
													</button>
												) : null}
												<button
													type="button"
													className="btn btn-sm btn-ghost rounded-xl"
													disabled={
														!bundle.isEnabled ||
														!server.enabled ||
														isReady ||
														isConnecting ||
														setupIncomplete ||
														isPending(`${server.id}:connect`)
													}
													onClick={() => {
														void runActionWithAlert(
															`${server.id}:connect`,
															() => onConnectServer(bundle.id, server.id),
															'Failed to connect MCP server.'
														).catch(() => undefined);
													}}
												>
													<FiWifi size={15} />
													<span>Connect</span>
												</button>
												<button
													type="button"
													className="btn btn-sm btn-ghost rounded-xl"
													onClick={() => {
														void runActionWithAlert(
															`${server.id}:disconnect`,
															() => onDisconnectServer(bundle.id, server.id),
															'Failed to disconnect MCP server.'
														).catch(() => undefined);
													}}
													disabled={
														!isReady || isPending(`${server.id}:disconnect`) || isPending(`${server.id}:connect`)
													}
												>
													<FiWifiOff size={15} />
													<span>Disconnect</span>
												</button>
												<button
													type="button"
													className="btn btn-sm btn-ghost rounded-xl"
													onClick={() => {
														void runActionWithAlert(
															`${server.id}:refresh`,
															() => onRefreshServer(bundle.id, server.id),
															'Failed to refresh MCP server.'
														).catch(() => undefined);
													}}
													disabled={
														!bundle.isEnabled ||
														!server.enabled ||
														!isReady ||
														isConnecting ||
														isPending(`${server.id}:refresh`)
													}
												>
													<FiRefreshCw size={15} />
													<span>Refresh</span>
												</button>
												<button
													type="button"
													className="btn btn-sm btn-ghost rounded-xl"
													onClick={() => {
														requestDeleteServer(server);
													}}
													disabled={server.isBuiltIn || bundle.isBuiltIn || isPending(`${server.id}:delete`)}
												>
													<FiTrash2 size={15} />
													<span>Delete</span>
												</button>
											</ActionRow>
										</ManagementItemCard>
									);
								})}
							</div>
						)}
					</div>
				)}
			</ManagementBundleCard>

			<DeleteConfirmationModal
				isOpen={isDeleteServerModalOpen}
				onClose={() => {
					if (!serverToDelete || !isPending(`${serverToDelete.id}:delete`)) {
						setIsDeleteServerModalOpen(false);
						setServerToDelete(null);
					}
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

			<MCPBundleDetailsModal
				isOpen={isBundleDetailsOpen}
				onClose={() => {
					setIsBundleDetailsOpen(false);
				}}
				bundle={bundle}
				serverCount={servers.length}
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
					await runAction(`${setupServer.id}:setup`, () =>
						onSubmitServerSetup(bundle.id, setupServer.id, inputValues, reset)
					);
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
		</>
	);
}
