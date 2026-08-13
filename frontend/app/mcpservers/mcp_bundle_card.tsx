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

import type { MCPAuthHealth, MCPServerRuntimeSnapshot } from '@/spec/mcp_artifact';
import { MCPAuthHealthState, MCPServerStatus } from '@/spec/mcp_artifact';

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

import type {
	MCPBundleView,
	MCPServerDraft,
	MCPServerView,
	MCPSetupSubmissionValue,
} from '@/mcpservers/lib/mcp_management';
import { getMCPServerSetupStatus, isServerOperational, serverDisplayName } from '@/mcpservers/lib/mcp_management';
import {
	getEffectiveMCPServerStatus,
	getMCPServerAuthHealthBadgeClass,
	getMCPServerAuthHealthLabel,
	getMCPStatusBadgeClass,
	getMCPStatusLabel,
	isMCPAuthActionable,
} from '@/mcpservers/lib/mcp_server_utils';
import { MCPBundleDetailsModal } from '@/mcpservers/mcp_bundle_details_modal';
import { AddEditMCPServerModal } from '@/mcpservers/mcp_server_add_edit_modal';
import { MCPServerDetailsModal } from '@/mcpservers/mcp_server_details_modal';
import { MCPServerSetupModal } from '@/mcpservers/mcp_server_setup_modal';

interface MCPServerReadErrors {
	runtime?: string;
	auth?: string;
}

interface MCPBundleCardProps {
	bundle: MCPBundleView;
	servers: MCPServerView[];
	existingLogicalNames: string[];
	runtimeByArtifactID: Record<string, MCPServerRuntimeSnapshot | undefined>;
	authHealthByArtifactID: Record<string, MCPAuthHealth | undefined>;
	readErrorsByArtifactID?: Record<string, MCPServerReadErrors | undefined>;
	serverLoadError?: string;

	onRefreshServers: () => Promise<void>;
	onToggleBundleEnabled: (bundle: MCPBundleView, enabled: boolean) => Promise<void>;
	onToggleServerEnabled: (bundle: MCPBundleView, server: MCPServerView, enabled: boolean) => Promise<void>;
	onSaveServer: (bundle: MCPBundleView, server: MCPServerView | undefined, draft: MCPServerDraft) => Promise<void>;
	onSaveSetup: (
		server: MCPServerView,
		values: Record<string, MCPSetupSubmissionValue>,
		reset: boolean
	) => Promise<void>;
	onDeleteServer: (bundle: MCPBundleView, server: MCPServerView) => Promise<void>;
	onConnectServer: (server: MCPServerView) => Promise<void>;
	onDisconnectServer: (server: MCPServerView) => Promise<void>;
	onRefreshServer: (server: MCPServerView) => Promise<void>;
	onCancelOAuth: (server: MCPServerView) => Promise<void>;
	onDeleteBundleRequested: (bundle: MCPBundleView) => void;
	onRequestOAuthAuthorization: (server: MCPServerView) => void;
}

function getErrorMessage(error: unknown, fallback: string): string {
	if (error instanceof Error && error.message.trim()) {
		return error.message;
	}

	return fallback;
}

export function MCPBundleCard({
	bundle,
	servers,
	existingLogicalNames,
	runtimeByArtifactID,
	authHealthByArtifactID,
	readErrorsByArtifactID = {},
	serverLoadError,
	onRefreshServers,
	onToggleBundleEnabled,
	onToggleServerEnabled,
	onSaveServer,
	onSaveSetup,
	onDeleteServer,
	onConnectServer,
	onDisconnectServer,
	onRefreshServer,
	onCancelOAuth,
	onDeleteBundleRequested,
	onRequestOAuthAuthorization,
}: MCPBundleCardProps) {
	const [isExpanded, setIsExpanded] = useState(false);
	const [serverToDelete, setServerToDelete] = useState<MCPServerView | null>(null);
	const [serverToEdit, setServerToEdit] = useState<MCPServerView | undefined>();
	const [serverDetails, setServerDetails] = useState<MCPServerView | null>(null);
	const [setupServer, setSetupServer] = useState<MCPServerView | null>(null);
	const [isBundleDetailsOpen, setIsBundleDetailsOpen] = useState(false);
	const [alertMessage, setAlertMessage] = useState('');

	const { isPending, runAction } = usePendingActions();

	const showAlert = (message: string) => {
		setAlertMessage(message);
	};

	const clearAlert = () => {
		setAlertMessage('');
	};

	const refresh = () => {
		void runAction('bundle:refresh', onRefreshServers).catch((error: unknown) => {
			showAlert(getErrorMessage(error, 'Failed to reload MCP servers.'));
		});
	};

	return (
		<>
			<ManagementBundleCard
				title={bundle.displayName}
				identity={
					<span className="font-mono">
						{bundle.logicalName} / {bundle.ref.collectionID}
					</span>
				}
				description={bundle.description}
				status={
					<>
						<StatusBadge tone={bundle.enabled ? 'success' : 'neutral'}>
							{bundle.enabled ? 'Enabled' : 'Disabled'}
						</StatusBadge>
						<StatusBadge>{bundle.builtIn ? 'Built-in' : 'Custom'}</StatusBadge>
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
						<span>Servers: {servers.length}</span>
						{isExpanded ? <FiChevronUp /> : <FiChevronDown />}
					</button>
				}
				actionLeading={
					<EnabledControl
						id={`mcp-bundle-${bundle.ref.collectionID}`}
						checked={bundle.enabled}
						compact={false}
						busy={isPending('bundle:toggle')}
						onChange={enabled => {
							void runAction('bundle:toggle', () => onToggleBundleEnabled(bundle, enabled)).catch((error: unknown) => {
								showAlert(getErrorMessage(error, 'Failed to change MCP bundle state.'));
							});
						}}
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
						>
							<FiEye size={16} />
							<span>Details</span>
						</button>

						{!bundle.builtIn ? (
							<>
								<button
									type="button"
									className="btn btn-sm btn-ghost rounded-xl"
									disabled={!bundle.enabled || Boolean(serverLoadError)}
									onClick={() => {
										setServerToEdit(undefined);
									}}
								>
									<FiPlus size={16} />
									<span>Add Server</span>
								</button>

								<button
									type="button"
									className="btn btn-sm btn-ghost rounded-xl"
									disabled={servers.length > 0 || Boolean(serverLoadError)}
									onClick={() => {
										onDeleteBundleRequested(bundle);
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
							disabled={isPending('bundle:refresh')}
							onClick={refresh}
						>
							<FiRefreshCw size={14} />
							<span>{isPending('bundle:refresh') ? 'Reloading' : 'Retry'}</span>
						</button>
					</output>
				) : null}

				{isExpanded ? (
					<div className="mt-6 space-y-3">
						{servers.length === 0 ? (
							<ManagementEmptyState>
								{serverLoadError ? 'Server contents are unavailable.' : 'No MCP servers in this bundle.'}
							</ManagementEmptyState>
						) : (
							servers.map(server => {
								const artifactID = server.ref.artifactID;
								const runtime = runtimeByArtifactID[artifactID];
								const authHealth = authHealthByArtifactID[artifactID];
								const readErrors = readErrorsByArtifactID[artifactID];
								const status = getEffectiveMCPServerStatus(server, runtime?.status);
								const setup = getMCPServerSetupStatus(server);
								const ready = status === MCPServerStatus.Ready;
								const connecting = status === MCPServerStatus.Connecting;
								const operational = isServerOperational(server);
								const authPending = authHealth?.state === MCPAuthHealthState.AuthorizationPending;

								return (
									<ManagementItemCard
										key={`${server.ref.rootID}:${artifactID}`}
										title={serverDisplayName(server)}
										subtitle={server.logicalName}
										status={
											<>
												<StatusBadge className={getMCPStatusBadgeClass(status)}>
													{getMCPStatusLabel(status)}
												</StatusBadge>
												{setup.hasInputs ? (
													<StatusBadge tone={setup.complete ? 'neutral' : 'warning'}>
														{setup.complete ? 'Setup ✓' : `Setup ${setup.requiredConfigured}/${setup.requiredTotal}`}
													</StatusBadge>
												) : null}
												<StatusBadge className={getMCPServerAuthHealthBadgeClass(server, authHealth)}>
													{getMCPServerAuthHealthLabel(server, authHealth)}
												</StatusBadge>
											</>
										}
										metadata={
											<>
												<MetadataPill label="Artifact">{artifactID}</MetadataPill>
												<MetadataPill label="Tools">{runtime?.toolCount ?? '—'}</MetadataPill>
												<MetadataPill label="Resources">{runtime?.resourceCount ?? '—'}</MetadataPill>
												<MetadataPill label="Prompts">{runtime?.promptCount ?? '—'}</MetadataPill>
											</>
										}
									>
										<div className="mt-3 space-y-1">
											{server.loadError ? <div className="text-error text-xs">{server.loadError}</div> : null}
											{readErrors?.runtime ? (
												<div className="text-warning text-xs">
													Runtime status could not be read: {readErrors.runtime}
												</div>
											) : null}
											{readErrors?.auth ? (
												<div className="text-warning text-xs">Auth health could not be read: {readErrors.auth}</div>
											) : null}
											{runtime?.lastError ? <div className="text-error text-xs">{runtime.lastError}</div> : null}
											{authHealth?.lastError ? <div className="text-error text-xs">{authHealth.lastError}</div> : null}
										</div>

										<ActionRow
											leading={
												<EnabledControl
													id={`mcp-server-${artifactID}`}
													checked={server.runtimeEnabled}
													disabled={!operational}
													busy={isPending(`${artifactID}:toggle`)}
													title={!operational ? 'The server installation is unavailable.' : undefined}
													onChange={enabled => {
														void runAction(`${artifactID}:toggle`, () =>
															onToggleServerEnabled(bundle, server, enabled)
														).catch((error: unknown) => {
															showAlert(getErrorMessage(error, 'Failed to change MCP server state.'));
														});
													}}
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

											{setup.hasInputs ? (
												<button
													type="button"
													className="btn btn-sm btn-ghost rounded-xl"
													disabled={!operational}
													onClick={() => {
														setSetupServer(server);
													}}
												>
													<FiSettings size={15} />
													<span>Setup</span>
												</button>
											) : null}

											{!server.builtIn ? (
												<button
													type="button"
													className="btn btn-sm btn-ghost rounded-xl"
													disabled={!bundle.enabled || !operational}
													onClick={() => {
														setServerToEdit(server);
													}}
												>
													<FiEdit2 size={15} />
													<span>Edit</span>
												</button>
											) : null}

											{isMCPAuthActionable(server, authHealth) ? (
												<button
													type="button"
													className="btn btn-sm btn-ghost rounded-xl"
													onClick={() => {
														onRequestOAuthAuthorization(server);
													}}
												>
													<FiExternalLink size={15} />
													<span>Authorize</span>
												</button>
											) : null}

											{authPending ? (
												<button
													type="button"
													className="btn btn-sm btn-ghost rounded-xl"
													disabled={isPending(`${artifactID}:cancel-oauth`)}
													onClick={() => {
														void runAction(`${artifactID}:cancel-oauth`, () => onCancelOAuth(server)).catch(
															(error: unknown) => {
																showAlert(getErrorMessage(error, 'Failed to cancel OAuth authorization.'));
															}
														);
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
													!server.runtimeEnabled ||
													!operational ||
													ready ||
													connecting ||
													!setup.complete ||
													isPending(`${artifactID}:connect`)
												}
												onClick={() => {
													void runAction(`${artifactID}:connect`, () => onConnectServer(server)).catch(
														(error: unknown) => {
															showAlert(getErrorMessage(error, 'Failed to connect MCP server.'));
														}
													);
												}}
											>
												<FiWifi size={15} />
												<span>Connect</span>
											</button>

											<button
												type="button"
												className="btn btn-sm btn-ghost rounded-xl"
												disabled={!ready || isPending(`${artifactID}:disconnect`)}
												onClick={() => {
													void runAction(`${artifactID}:disconnect`, () => onDisconnectServer(server)).catch(
														(error: unknown) => {
															showAlert(getErrorMessage(error, 'Failed to disconnect MCP server.'));
														}
													);
												}}
											>
												<FiWifiOff size={15} />
												<span>Disconnect</span>
											</button>

											<button
												type="button"
												className="btn btn-sm btn-ghost rounded-xl"
												disabled={!ready || isPending(`${artifactID}:refresh`)}
												onClick={() => {
													void runAction(`${artifactID}:refresh`, () => onRefreshServer(server)).catch(
														(error: unknown) => {
															showAlert(getErrorMessage(error, 'Failed to refresh MCP server.'));
														}
													);
												}}
											>
												<FiRefreshCw size={15} />
												<span>Refresh</span>
											</button>

											{!server.builtIn ? (
												<button
													type="button"
													className="btn btn-sm btn-ghost rounded-xl"
													disabled={isPending(`${artifactID}:delete`)}
													onClick={() => {
														setServerToDelete(server);
													}}
												>
													<FiTrash2 size={15} />
													<span>Delete</span>
												</button>
											) : null}
										</ActionRow>
									</ManagementItemCard>
								);
							})
						)}
					</div>
				) : null}
			</ManagementBundleCard>

			<DeleteConfirmationModal
				isOpen={serverToDelete !== null}
				onClose={() => {
					setServerToDelete(null);
				}}
				onConfirm={async () => {
					if (!serverToDelete) {
						return;
					}

					await runAction(`${serverToDelete.ref.artifactID}:delete`, () => onDeleteServer(bundle, serverToDelete));
					setServerToDelete(null);
				}}
				title="Delete MCP Server"
				message={`Delete MCP server "${serverToDelete ? serverDisplayName(serverToDelete) : ''}"? This cannot be undone.`}
				confirmButtonText="Delete"
			/>

			<AddEditMCPServerModal
				isOpen={serverToEdit !== undefined}
				bundle={bundle}
				initialServer={serverToEdit}
				existingLogicalNames={existingLogicalNames}
				onClose={() => {
					setServerToEdit(undefined);
				}}
				onSubmit={async draft => {
					await onSaveServer(bundle, serverToEdit, draft);
				}}
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
				runtime={serverDetails ? runtimeByArtifactID[serverDetails.ref.artifactID] : undefined}
				authHealth={serverDetails ? authHealthByArtifactID[serverDetails.ref.artifactID] : undefined}
			/>

			<MCPServerSetupModal
				isOpen={setupServer !== null}
				server={setupServer}
				onClose={() => {
					setSetupServer(null);
				}}
				onSubmit={onSaveSetup}
			/>

			<ActionDeniedAlertModal isOpen={Boolean(alertMessage)} message={alertMessage} onClose={clearAlert} />
		</>
	);
}
