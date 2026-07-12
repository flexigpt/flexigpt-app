import { useEffect, useRef } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle, FiX } from 'react-icons/fi';

import type { MCPApprovalSummary } from '@/spec/mcp';
import { MCPApprovalResolution } from '@/spec/mcp';

import { ModalBackdrop } from '@/components/modal/modal_backdrop';

import type { MCPApprovalRequest } from '@/chats/composer/mcp/use_mcp_approval';
import { getMCPToolRiskLabel } from '@/mcpservers/lib/mcp_server_utils';

interface MCPApprovalModalProps {
	approvalRequest: MCPApprovalRequest | null;
	onResolve: (resolution: MCPApprovalResolution) => void;
}

function formatArguments(summary?: MCPApprovalSummary): string {
	const raw = summary?.arguments?.trim() ?? '';
	if (!raw) {
		return '-';
	}

	try {
		const parsed = JSON.parse(raw) as unknown;
		return JSON.stringify(parsed, null, 2);
	} catch {
		return raw;
	}
}

export function MCPApprovalModal({ approvalRequest, onResolve }: MCPApprovalModalProps) {
	const dialogRef = useRef<HTMLDialogElement | null>(null);
	const resolvedApprovalIDRef = useRef<string | null>(null);

	useEffect(() => {
		if (!approvalRequest) {
			return;
		}

		const dialog = dialogRef.current;
		if (!dialog) {
			return;
		}

		if (!dialog.open) {
			try {
				dialog.showModal();
			} catch {
				// Ignore showModal errors and keep rendering safely.
			}
		}

		return () => {
			if (dialog.open) {
				dialog.close();
			}
		};
	}, [approvalRequest]);

	useEffect(() => {
		if (!approvalRequest) {
			return;
		}

		resolvedApprovalIDRef.current = null;
	}, [approvalRequest]);

	if (!approvalRequest) {
		return null;
	}
	if (typeof document === 'undefined' || !document.body) {
		return null;
	}

	const { approvalID, summary, reason } = approvalRequest;

	const resolveOnce = (resolution: MCPApprovalResolution) => {
		if (resolvedApprovalIDRef.current === approvalID) {
			return;
		}

		resolvedApprovalIDRef.current = approvalID;
		onResolve(resolution);
	};

	const closeAsDenyOnce = () => {
		resolveOnce(MCPApprovalResolution.MCPApprovalResolutionDenyOnce);
	};

	return createPortal(
		<dialog
			ref={dialogRef}
			className="modal"
			onCancel={e => {
				e.preventDefault();
				closeAsDenyOnce();
			}}
			onClose={() => {
				closeAsDenyOnce();
			}}
		>
			<div className="modal-box bg-base-200 max-h-[80vh] max-w-3xl overflow-hidden rounded-2xl p-0">
				<div className="max-h-[80vh] overflow-y-auto p-6">
					<div className="mb-4 flex items-center justify-between gap-3">
						<h3 className="text-lg font-bold">MCP approval required</h3>
						<button
							type="button"
							className="btn btn-sm btn-circle bg-base-300"
							onClick={closeAsDenyOnce}
							aria-label="Close"
						>
							<FiX size={12} />
						</button>
					</div>

					<div className="space-y-4">
						<div className="grid grid-cols-12 gap-2 text-sm">
							<div className="col-span-3 font-semibold">Server</div>
							<div className="col-span-9 break-all">
								{summary.serverDisplayName?.trim() || summary.serverID}
								<div className="text-base-content/60 text-xs">{summary.serverID}</div>
							</div>

							<div className="col-span-3 font-semibold">Tool</div>
							<div className="col-span-9 break-all">
								{summary.toolName}
								{summary.toolDigest ? <div className="text-base-content/60 text-xs">{summary.toolDigest}</div> : null}
							</div>

							<div className="col-span-3 font-semibold">Risk</div>
							<div className="col-span-9">{getMCPToolRiskLabel(summary.risk)}</div>

							{reason ? (
								<>
									<div className="col-span-3 font-semibold">Reason</div>
									<div className="text-base-content/80 col-span-9">{reason}</div>
								</>
							) : null}
						</div>

						<div>
							<div className="mb-2 text-sm font-semibold">Arguments</div>
							<pre className="bg-base-300 max-h-60 overflow-auto rounded-2xl p-3 text-xs whitespace-pre-wrap">
								{formatArguments(summary)}
							</pre>
						</div>

						<div className="alert alert-info rounded-2xl text-sm">
							<div className="flex items-center gap-2">
								<FiAlertCircle size={14} />
								<span>This tool call is waiting for your approval before execution.</span>
							</div>
						</div>
					</div>

					<div className="modal-action flex flex-wrap items-center justify-between gap-2">
						<div className="flex flex-wrap gap-2">
							<button
								type="button"
								className="btn btn-sm bg-base-300 rounded-xl"
								onClick={() => {
									resolveOnce(MCPApprovalResolution.MCPApprovalResolutionDenyOnce);
								}}
							>
								Deny once
							</button>
							<button
								type="button"
								className="btn btn-sm btn-error rounded-xl"
								onClick={() => {
									resolveOnce(MCPApprovalResolution.MCPApprovalResolutionDenyAlways);
								}}
							>
								Deny always
							</button>
						</div>

						<div className="flex flex-wrap gap-2">
							<button
								type="button"
								className="btn btn-sm bg-base-300 rounded-xl"
								onClick={() => {
									resolveOnce(MCPApprovalResolution.MCPApprovalResolutionAllowOnce);
								}}
							>
								Allow once
							</button>
							<button
								type="button"
								className="btn btn-sm btn-primary rounded-xl"
								onClick={() => {
									resolveOnce(MCPApprovalResolution.MCPApprovalResolutionAllowAlways);
								}}
							>
								Allow always
							</button>
						</div>
					</div>
				</div>
			</div>

			<ModalBackdrop enabled={true} />
		</dialog>,
		document.body
	);
}
