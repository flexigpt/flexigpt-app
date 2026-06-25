import { useEffect, useRef } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle, FiExternalLink, FiX } from 'react-icons/fi';

import { type MCPAuthHealth, MCPAuthHealthState, type MCPServerConfig } from '@/spec/mcp';

import { ModalBackdrop } from '@/components/modal_backdrop';

import { getMCPAuthHealthBadgeClass, getMCPAuthHealthLabel } from '@/mcpservers/lib/mcp_server_utils';

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
	const dialogRef = useRef<HTMLDialogElement | null>(null);
	const isUnmountingRef = useRef(false);

	const authorizationURL = authHealth?.authorizationURL ?? '';
	const isPending = authHealth?.state === MCPAuthHealthState.MCPAuthHealthStateAuthorizationPending;
	const isAuthorized = authHealth?.state === MCPAuthHealthState.MCPAuthHealthStateAuthorized;

	useEffect(() => {
		const dialog = dialogRef.current;
		if (!dialog) {
			return;
		}

		if (!dialog.open) {
			try {
				dialog.showModal();
			} catch {
				// Another modal may already be open. Keep rendering safely.
			}
		}

		return () => {
			isUnmountingRef.current = true;

			if (dialog.open) {
				dialog.close();
			}
		};
	}, []);

	const requestClose = () => {
		const dialog = dialogRef.current;

		if (dialog?.open) {
			dialog.close();
			return;
		}

		onClose();
	};

	const handleDialogClose = () => {
		if (isUnmountingRef.current) {
			return;
		}
		onClose();
	};

	return (
		<dialog
			ref={dialogRef}
			className="modal"
			onClose={handleDialogClose}
			onCancel={e => {
				e.preventDefault();
			}}
		>
			<div className="modal-box bg-base-200 max-w-2xl rounded-2xl p-0">
				<div className="p-6">
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
							onClick={requestClose}
							aria-label="Close"
						>
							<FiX size={12} />
						</button>
					</div>

					<div className="mb-4 flex items-center gap-2">
						<span className={`badge rounded-xl ${getMCPAuthHealthBadgeClass(authHealth?.state)}`}>
							{getMCPAuthHealthLabel(authHealth?.state)}
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
						</div>
					)}

					<div className="modal-action">
						{isPending && onCancel && (
							<button type="button" className="btn bg-base-300 rounded-xl" onClick={() => void onCancel()}>
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

						<button type="button" className="btn bg-base-300 rounded-xl" onClick={requestClose}>
							Close
						</button>
					</div>
				</div>
			</div>
			<ModalBackdrop enabled={true} />
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
