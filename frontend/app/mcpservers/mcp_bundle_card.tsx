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
	MCPBundleView,
	MCPServerDraft,
	MCPServerRuntimeSnapshot,
	MCPServerView,
	MCPSetupSubmissionValue,
} from '@/spec/mcp';
import { MCPAuthHealthState, MCPServerStatus } from '@/spec/mcp';

import { getErrorMessage } from '@/lib/error_utils';

import { usePendingActions } from '@/hooks/use_pending_actions';

import { getMCPServerSetupStatus, isServerOperational, serverDisplayName } from '@/apis/mcp_management';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';
import { ActionRow } from '@/components/managementui/action_row';
import { EnabledControl } from '@/components/managementui/enabled_control';
import { ManagementBundleCard } from '@/components/managementui/management_bundle_card';
import { ManagementEmptyState } from '@/components/managementui/management_empty_state';
import { ManagementItemCard } from '@/components/managementui/management_item_card';
import { MetadataPill } from '@/components/managementui/metadata_pill';
import { StatusBadge } from '@/components/managementui/status_badge';

import {
	getEffectiveMCPServerStatus,
	getMCPServerAuthHealthBadgeClass,
	getMCPServerAuthHealthLabel,
	getMCPStatusBadgeClass,
	getMCPStatusLabel,
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
	isLoadingServers?: boolean;
	serversLoaded: boolean;

	onLoadServers: () => Promise<void>;
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

interface MCPServerEditorState {
	server?: MCPServerView;
}

interface MCPSetupTarget {
	server: MCPServerView;
	connectAfterSave: boolean;
}

export function MCPBundleCard({
	bundle,
	servers,
	existingLogicalNames,
	runtimeByArtifactID,
	authHealthByArtifactID,
	readErrorsByArtifactID = {},
	serverLoadError,
	isLoadingServers = false,
	serversLoaded,
	onLoadServers,
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
	const [serverEditor, setServerEditor] = useState<MCPServerEditorState | null>(null);
	const [serverDetails, setServerDetails] = useState<MCPServerView | null>(null);
	const [setupTarget, setSetupTarget] = useState<MCPSetupTarget | null>(null);
	const [isBundleDetailsOpen, setIsBundleDetailsOpen] = useState(false);
	const [alertMessage, setAlertMessage] = useState('');

	const { isPending, runAction } = usePendingActions();

	const showAlert = (message: string) => {
		setAlertMessage(message);
	};

	const clearAlert = () => {
		setAlertMessage('');
	};
	const bundleIdentity = bundle.displayName === bundle.logicalName ? undefined : bundle.logicalName;

	const refresh = () => {
		void runAction('bundle:refresh', onRefreshServers).catch((error: unknown) => {
			showAlert(getErrorMessage(error, 'Failed to reload MCP servers.'));
		});
	};

	const loadServers = () => {
		void runAction('bundle:load', onLoadServers).catch((error: unknown) => {
			showAlert(getErrorMessage(error, 'Failed to load MCP servers.'));
		});
	};

	const connectServer = async (server: MCPServerView) => {
		const artifactID = server.ref.artifactID;

		try {
			await runAction(`${artifactID}:connect`, () => onConnectServer(server));
		} catch (error) {
			showAlert(getErrorMessage(error, 'Failed to start the MCP server connection.'));
			throw error;
		}
	};

	return (
		<>
			<ManagementBundleCard
				title={bundle.displayName}
				identity={bundleIdentity ? <span className="font-mono">{bundleIdentity}</span> : null}
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
							const next = !isExpanded;
							setIsExpanded(next);
							if (next && !serversLoaded && !isLoadingServers) {
								loadServers();
							}
						}}
					>
						<span>
							{isLoadingServers
								? 'Loading servers...'
								: serversLoaded
									? `Servers: ${servers.length}`
									: 'Servers: not loaded'}
						</span>
						{isExpanded ? <FiChevronUp /> : <FiChevronDown />}
					</button>
				}
				actionLeading={
					<EnabledControl
						id={`mcp-bundle-${bundle.ref.artifactID}`}
						checked={bundle.enabled}
						compact={false}
						busy={isPending('bundle:toggle')}
						onChange={enabled => {
							void runAction('bundle:toggle', () => onToggleBundleEnabled(bundle, enabled)).catch((error: unknown) => {
								showAlert(getErrorMessage(error, 'Failed to change MCP Collection state.'));
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
									disabled={!bundle.editable || !serversLoaded || Boolean(serverLoadError)}
									onClick={() => {
										setServerEditor({});
									}}
								>
									<FiPlus size={16} />
									<span>Add Server</span>
								</button>

								<button
									type="button"
									className="btn btn-sm btn-ghost rounded-xl"
									disabled={!bundle.deletable || !serversLoaded || servers.length > 0 || Boolean(serverLoadError)}
									onClick={() => {
										onDeleteBundleRequested(bundle);
									}}
								>
									<FiTrash2 size={16} />
									<span>Delete Collection</span>
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
						{!serversLoaded ? (
							<ManagementEmptyState>
								{isLoadingServers
									? 'Loading MCP servers...'
									: serverLoadError
										? 'Server contents are unavailable.'
										: 'Expand this Collection to load its MCP servers.'}
							</ManagementEmptyState>
						) : null}

						{serversLoaded && servers.length === 0 ? (
							<ManagementEmptyState>
								{isLoadingServers
									? 'Loading MCP servers...'
									: serverLoadError
										? 'Server contents are unavailable.'
										: 'No MCP servers in this Collection.'}
							</ManagementEmptyState>
						) : serversLoaded ? (
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
								const title = serverDisplayName(server);
								const subtitle = title === server.logicalName ? undefined : server.logicalName;

								return (
									<ManagementItemCard
										key={`${server.ref.rootID}:${artifactID}`}
										title={title}
										subtitle={subtitle}
										status={
											<>
												<StatusBadge className={getMCPStatusBadgeClass(status)}>
													{getMCPStatusLabel(status)}
												</StatusBadge>
												{setup.hasInputs ? (
													<StatusBadge tone={setup.complete ? 'neutral' : 'warning'}>
														{setup.complete
															? 'Setup complete'
															: `Setup ${setup.requiredConfigured}/${setup.requiredTotal}`}
													</StatusBadge>
												) : null}
												<StatusBadge className={getMCPServerAuthHealthBadgeClass(server, authHealth)}>
													{getMCPServerAuthHealthLabel(server, authHealth)}
												</StatusBadge>
											</>
										}
										metadata={
											<>
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
													checked={server.enabled}
													disabled={!operational || !bundle.enabled}
													busy={isPending(`${artifactID}:toggle`)}
													title={
														!operational
															? 'The server installation is unavailable.'
															: !bundle.enabled
																? 'Enable the Collection before changing its server settings.'
																: undefined
													}
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
														setSetupTarget({
															server,
															connectAfterSave: false,
														});
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
														setServerEditor({
															server,
														});
													}}
												>
													<FiEdit2 size={15} />
													<span>Edit</span>
												</button>
											) : null}

											{authPending && authHealth?.authorizationURL?.trim() ? (
												<button
													type="button"
													className="btn btn-sm btn-ghost rounded-xl"
													onClick={() => {
														onRequestOAuthAuthorization(server);
													}}
												>
													<FiExternalLink size={15} />
													<span>Resume authorization</span>
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
													!server.enabled ||
													!operational ||
													ready ||
													connecting ||
													authPending ||
													isPending(`${artifactID}:connect`)
												}
												onClick={() => {
													if (!setup.complete) {
														setSetupTarget({
															server,
															connectAfterSave: true,
														});
														return;
													}

													void connectServer(server).catch(() => undefined);
												}}
											>
												<FiWifi size={15} />
												<span>{setup.complete ? 'Connect' : 'Set up & connect'}</span>
											</button>

											{!authPending ? (
												<button
													type="button"
													className="btn btn-sm btn-ghost rounded-xl"
													disabled={(!ready && !connecting) || isPending(`${artifactID}:disconnect`)}
													onClick={() => {
														void runAction(`${artifactID}:disconnect`, () => onDisconnectServer(server)).catch(
															(error: unknown) => {
																showAlert(getErrorMessage(error, 'Failed to disconnect MCP server.'));
															}
														);
													}}
												>
													<FiWifiOff size={15} />
													<span>{connecting ? 'Cancel connection' : 'Disconnect'}</span>
												</button>
											) : null}

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
						) : null}
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
				isOpen={serverEditor !== null}
				bundle={bundle}
				initialServer={serverEditor?.server}
				existingLogicalNames={existingLogicalNames}
				onClose={() => {
					setServerEditor(null);
				}}
				onSubmit={async draft => {
					await onSaveServer(bundle, serverEditor?.server, draft);
				}}
			/>

			<MCPBundleDetailsModal
				isOpen={isBundleDetailsOpen}
				onClose={() => {
					setIsBundleDetailsOpen(false);
				}}
				bundle={bundle}
				serverCount={servers.length}
				serversLoaded={serversLoaded}
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
				isOpen={setupTarget !== null}
				server={setupTarget?.server ?? null}
				onClose={() => {
					setSetupTarget(null);
				}}
				onSubmit={async (server, values, reset) => {
					const connectAfterSave = setupTarget?.connectAfterSave ?? false;
					await onSaveSetup(server, values, reset);

					if (connectAfterSave) {
						try {
							await connectServer(server);
						} catch {
							// Setup was saved successfully. The card-level alert
							// reports only the failed connection start.
						}
					}
				}}
			/>

			<ActionDeniedAlertModal isOpen={Boolean(alertMessage)} message={alertMessage} onClose={clearAlert} />
		</>
	);
}
