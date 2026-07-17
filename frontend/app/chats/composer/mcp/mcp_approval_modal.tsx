import { useEffect, useRef } from 'react';

import { FiAlertCircle } from 'react-icons/fi';

import type { MCPApprovalSummary } from '@/spec/mcp';
import { MCPApprovalResolution } from '@/spec/mcp';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalBackdrop } from '@/components/modal/modal_backdrop';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalHeader } from '@/components/modal/modal_header';

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
	const resolvedApprovalIDRef = useRef<string | null>(null);

	useEffect(() => {
		if (!approvalRequest) {
			return;
		}

		resolvedApprovalIDRef.current = null;
	}, [approvalRequest]);

	const resolveOnce = (resolution: MCPApprovalResolution) => {
		const approvalID = approvalRequest?.approvalID;
		if (!approvalID || resolvedApprovalIDRef.current === approvalID) {
			return;
		}

		resolvedApprovalIDRef.current = approvalID;
		onResolve(resolution);
	};

	const closeAsDenyOnce = () => {
		resolveOnce(MCPApprovalResolution.MCPApprovalResolutionDenyOnce);
	};

	if (!approvalRequest) {
		return null;
	}

	const { summary, reason } = approvalRequest;

	return (
		<ModalDialog
			isOpen={true}
			onClose={closeAsDenyOnce}
			onCancel={e => {
				e.preventDefault();
				closeAsDenyOnce();
			}}
		>
			<div className="modal-box bg-base-200 max-h-[80vh] max-w-3xl overflow-hidden rounded-2xl p-0">
				<div className="max-h-[80vh] overflow-y-auto p-6">
					<ModalHeader title="MCP approval required" onClose={closeAsDenyOnce} />

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

					<ModalActions
						className="-mx-6 mt-6 -mb-6"
						leading={
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
						}
					>
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
					</ModalActions>
				</div>
			</div>

			<ModalBackdrop enabled={true} />
		</ModalDialog>
	);
}
