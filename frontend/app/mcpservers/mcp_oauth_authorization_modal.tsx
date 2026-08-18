import { useState } from 'react';

import { FiAlertCircle, FiExternalLink, FiRefreshCw } from 'react-icons/fi';

import type { MCPAuthHealth } from '@/spec/mcp_artifact';
import { MCPAuthHealthState } from '@/spec/mcp_artifact';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalBackdrop } from '@/components/modal/modal_backdrop';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalHeader } from '@/components/modal/modal_header';

import type { MCPServerView } from '@/mcpservers/lib/mcp_management';
import { getMCPServerAuthHealthBadgeClass, getMCPServerAuthHealthLabel } from '@/mcpservers/lib/mcp_server_utils';

interface MCPOAuthAuthorizationModalProps {
	isOpen: boolean;
	server: MCPServerView | null;
	authHealth?: MCPAuthHealth;
	isConnecting?: boolean;
	isReady?: boolean;
	runtimeError?: string;
	onClose: () => void;
	onOpenURL: (url: string) => void;
	onCancel?: () => Promise<void> | void;
}

function MCPOAuthAuthorizationModalContent({
	server,
	authHealth,
	onOpenURL,
	onCancel,
	isConnecting = false,
	isReady = false,
	runtimeError,
	isCancelling,
	setIsCancelling,
}: MCPOAuthAuthorizationModalProps & {
	server: MCPServerView;
	isCancelling: boolean;
	setIsCancelling: (value: boolean) => void;
}) {
	const [cancelError, setCancelError] = useState('');
	const { requestClose, unmountingRef } = useModalDialogController();

	const authorizationURL =
		authHealth?.state === MCPAuthHealthState.Authorized ? '' : (authHealth?.authorizationURL?.trim() ?? '');
	const isPending = authHealth?.state === MCPAuthHealthState.AuthorizationPending;
	const isAuthorized = authHealth?.state === MCPAuthHealthState.Authorized;
	const lastError = authHealth?.lastError?.trim() || runtimeError?.trim() || '';
	const loopbackUnavailable = authHealth?.oauthLoopbackReady === false;
	const isPreparing = isConnecting && !authorizationURL && !lastError && !loopbackUnavailable;

	const handleCancel = async () => {
		if (!onCancel || isCancelling) {
			return;
		}

		setCancelError('');
		setIsCancelling(true);

		try {
			await onCancel();
			if (!unmountingRef.current) {
				requestClose(true);
			}
		} catch (error) {
			if (!unmountingRef.current) {
				setCancelError(
					error instanceof Error && error.message.trim() ? error.message : 'Failed to cancel OAuth authorization.'
				);
			}
		} finally {
			if (!unmountingRef.current) {
				setIsCancelling(false);
			}
		}
	};

	return (
		<>
			<div className="modal-box bg-base-200 max-h-[calc(100dvh-1rem)] w-[calc(100%-1rem)] max-w-2xl overflow-y-auto rounded-2xl p-0">
				<div className="app-scrollbar-thin p-4 sm:p-6">
					<ModalHeader
						title={isAuthorized ? 'OAuth authorization completed' : 'Connect with OAuth'}
						description={
							isAuthorized
								? `${server.displayName} is completing MCP discovery.`
								: `${server.displayName} needs browser authorization before FlexiGPT can connect.`
						}
						onClose={() => {
							requestClose();
						}}
						closeDisabled={isCancelling}
					/>

					<div className="mb-4 flex flex-wrap items-center gap-2">
						<span className={`badge rounded-xl ${getMCPServerAuthHealthBadgeClass(server, authHealth)}`}>
							{getMCPServerAuthHealthLabel(server, authHealth)}
						</span>
						{authHealth?.authorizationExpiresAt ? (
							<span className="text-base-content/60 text-xs">Expires at {authHealth.authorizationExpiresAt}</span>
						) : null}
					</div>

					{isAuthorized ? (
						<div className="alert alert-success rounded-2xl text-sm">
							{isReady
								? 'Authorization and MCP connection completed.'
								: isConnecting
									? 'Authorization completed. FlexiGPT is loading MCP capabilities.'
									: 'Authorization completed.'}
						</div>
					) : (
						<div className="space-y-4">
							<div className="bg-base-100 rounded-2xl p-4 text-sm">
								<ol className="list-decimal space-y-2 pl-5">
									<li>Open the authorization page.</li>
									<li>Complete login and consent in your browser.</li>
									<li>Keep FlexiGPT open while the browser returns to its loopback callback.</li>
									<li>FlexiGPT finishes the authorization flow automatically.</li>
								</ol>
							</div>

							{authorizationURL ? (
								<div>
									<div className="text-base-content/70 mb-1 text-xs font-semibold uppercase">Authorization URL</div>
									<div className="bg-base-300 max-h-32 overflow-auto rounded-2xl p-3 text-xs break-all">
										{authorizationURL}
									</div>
								</div>
							) : isPreparing ? (
								<div className="alert alert-info rounded-2xl text-sm">
									<FiRefreshCw className="animate-spin" size={14} />
									<span>
										Preparing a secure authorization request. The browser will open automatically when it is ready.
									</span>
								</div>
							) : loopbackUnavailable ? (
								<div className="alert alert-error rounded-2xl text-sm">
									<FiAlertCircle size={14} />
									<span>
										{authHealth?.oauthLoopbackError ||
											'The local OAuth callback listener is unavailable. Check OAuth settings and restart FlexiGPT.'}
									</span>
								</div>
							) : (
								<div className="alert alert-warning rounded-2xl text-sm">
									<FiAlertCircle size={14} />
									<span>No OAuth authorization request is active. Close this dialog and retry Connect.</span>
								</div>
							)}

							{lastError && !loopbackUnavailable ? (
								<div className="alert alert-error rounded-2xl text-sm">
									<FiAlertCircle size={14} />
									<span>{lastError}</span>
								</div>
							) : null}

							{cancelError ? (
								<div className="alert alert-error rounded-2xl text-sm">
									<FiAlertCircle size={14} />
									<span>{cancelError}</span>
								</div>
							) : null}
						</div>
					)}

					<ModalActions className="-mx-4 mt-6 -mb-4 sm:-mx-6 sm:-mb-6">
						{(isPending || isConnecting) && onCancel ? (
							<button
								type="button"
								className="btn bg-base-300 rounded-xl"
								disabled={isCancelling}
								onClick={() => {
									void handleCancel();
								}}
							>
								{isPending ? 'Cancel authorization' : 'Cancel connection'}
							</button>
						) : null}

						<button
							type="button"
							className="btn btn-primary rounded-xl"
							disabled={!authorizationURL || isAuthorized}
							onClick={() => {
								if (authorizationURL) {
									onOpenURL(authorizationURL);
								}
							}}
						>
							<FiExternalLink size={14} />
							<span className="ml-1">Open authorization page</span>
						</button>

						<button
							type="button"
							className="btn bg-base-300 rounded-xl"
							disabled={isCancelling}
							onClick={() => {
								requestClose();
							}}
						>
							Close
						</button>
					</ModalActions>
				</div>
			</div>
			<ModalBackdrop enabled={!isCancelling} />
		</>
	);
}

function MCPOAuthAuthorizationModalSession(props: MCPOAuthAuthorizationModalProps & { server: MCPServerView }) {
	const [isCancelling, setIsCancelling] = useState(false);

	return (
		<ModalDialog isOpen={props.isOpen} onClose={props.onClose} isBusy={isCancelling}>
			<MCPOAuthAuthorizationModalContent {...props} isCancelling={isCancelling} setIsCancelling={setIsCancelling} />
		</ModalDialog>
	);
}

export function MCPOAuthAuthorizationModal(props: MCPOAuthAuthorizationModalProps) {
	if (!props.isOpen || !props.server) {
		return null;
	}

	return <MCPOAuthAuthorizationModalSession {...props} server={props.server} />;
}
