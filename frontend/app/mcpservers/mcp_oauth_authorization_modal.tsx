import { useState } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle, FiExternalLink, FiX } from 'react-icons/fi';

import type { MCPAuthHealth, MCPServerConfig } from '@/spec/mcp';
import { MCPAuthHealthState } from '@/spec/mcp';

import { useDialogController } from '@/hooks/use_dialog_controller';

import { ModalBackdrop } from '@/components/modal/modal_backdrop';

import {
	getEffectiveMCPAuthHealthState,
	getMCPServerAuthHealthBadgeClass,
	getMCPServerAuthHealthLabel,
	isMCPAuthActionable,
} from '@/mcpservers/lib/mcp_server_utils';

interface MCPOAuthAuthorizationModalProps {
	isOpen: boolean;
	server: MCPServerConfig | null;
	authHealth?: MCPAuthHealth;
	onClose: () => void;
	onOpenURL: (url: string) => void;
	onCancel?: () => Promise<void> | void;
}

function MCPOAuthAuthorizationModalContent({
	server,
	authHealth,
	onClose,
	onOpenURL,
	onCancel,
}: MCPOAuthAuthorizationModalProps) {
	const [isCancelling, setIsCancelling] = useState(false);
	const [cancelError, setCancelError] = useState('');

	const { dialogRef, requestClose, handleClose, handleCancel, unmountingRef } = useDialogController({
		onClose,
		blockCancel: true,
		isBusy: isCancelling,
	});

	const authState = getEffectiveMCPAuthHealthState(server ?? undefined, authHealth);
	const authorizationURL = isMCPAuthActionable(authHealth, server ?? undefined)
		? (authHealth?.authorizationURL ?? '')
		: '';
	const isPending = authState === MCPAuthHealthState.MCPAuthHealthStateAuthorizationPending;
	const isAuthorized = authState === MCPAuthHealthState.MCPAuthHealthStateAuthorized;

	const handleAuthorizationCancel = async () => {
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
					error instanceof Error && error.message.trim() ? error.message : 'Failed to cancel authorization.'
				);
			}
		} finally {
			if (!unmountingRef.current) {
				setIsCancelling(false);
			}
		}
	};

	return (
		<dialog ref={dialogRef} className="modal" onClose={handleClose} onCancel={handleCancel}>
			<div className="modal-box bg-base-200 max-h-[calc(100dvh-1rem)] w-[calc(100%-1rem)] max-w-2xl overflow-y-auto rounded-2xl p-0">
				<div className="app-scrollbar-thin p-4 sm:p-6">
					<div className="mb-4 flex items-start justify-between gap-4">
						<div>
							<h3 className="text-lg font-bold">OAuth authorization required</h3>
							<p className="text-base-content/70 mt-1 text-sm">
								{server?.displayName ?? server?.id ?? 'This MCP server'} needs browser authorization before FlexiGPT can
								connect.
							</p>
						</div>

						<button
							type="button"
							className="btn btn-sm btn-circle bg-base-300"
							onClick={() => {
								requestClose();
							}}
							aria-label="Close"
							disabled={isCancelling}
						>
							<FiX size={12} />
						</button>
					</div>

					<div className="mb-4 flex items-center gap-2">
						<span className={`badge rounded-xl ${getMCPServerAuthHealthBadgeClass(server ?? undefined, authHealth)}`}>
							{getMCPServerAuthHealthLabel(server ?? undefined, authHealth)}
						</span>
						{authHealth?.authorizationExpiresAt && (
							<span className="text-base-content/60 text-xs">Expires at {authHealth.authorizationExpiresAt}</span>
						)}
					</div>

					{isAuthorized ? (
						<div className="alert alert-success rounded-2xl text-sm">
							Authorization completed. You can close this dialog.
						</div>
					) : (
						<div className="space-y-4">
							<div className="bg-base-100 rounded-2xl p-4 text-sm">
								<ol className="list-decimal space-y-2 pl-5">
									<li>Click “Open authorization page”.</li>
									<li>Complete login and consent in your browser.</li>
									<li>
										Your browser will return to a local FlexiGPT callback URL. Keep this app open while that happens.
									</li>
									<li>After the callback is received, FlexiGPT will finish connecting automatically.</li>
								</ol>
							</div>

							{authorizationURL ? (
								<div>
									<div className="text-base-content/70 mb-1 text-xs font-semibold uppercase">Authorization URL</div>
									<div className="bg-base-300 max-h-32 overflow-auto rounded-2xl p-3 text-xs break-all">
										{authorizationURL}
									</div>
								</div>
							) : (
								<div className="alert alert-warning rounded-2xl text-sm">
									<div className="flex items-center gap-2">
										<FiAlertCircle size={14} />
										<span>The authorization URL is not available yet. Wait a moment and try again.</span>
									</div>
								</div>
							)}

							{authHealth?.lastError && (
								<div className="alert alert-error rounded-2xl text-sm">
									<div className="flex items-center gap-2">
										<FiAlertCircle size={14} />
										<span>{authHealth.lastError}</span>
									</div>
								</div>
							)}

							{cancelError && (
								<div className="alert alert-error rounded-2xl text-sm">
									<div className="flex items-center gap-2">
										<FiAlertCircle size={14} />
										<span>{cancelError}</span>
									</div>
								</div>
							)}
						</div>
					)}

					<div className="modal-action">
						{isPending && onCancel && (
							<button
								type="button"
								className="btn bg-base-300 rounded-xl"
								disabled={isCancelling}
								onClick={() => {
									void handleAuthorizationCancel();
								}}
							>
								Cancel authorization
							</button>
						)}

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
							onClick={() => {
								requestClose();
							}}
							disabled={isCancelling}
						>
							Close
						</button>
					</div>
				</div>
			</div>
			<ModalBackdrop enabled={!isCancelling} />
		</dialog>
	);
}

export function MCPOAuthAuthorizationModal(props: MCPOAuthAuthorizationModalProps) {
	if (!props.isOpen) {
		return null;
	}
	if (typeof document === 'undefined' || !document.body) {
		return null;
	}

	return createPortal(
		<MCPOAuthAuthorizationModalContent key="mcp-oauth-authorization-modal" {...props} />,
		document.body
	);
}
