import { useCallback, useEffect, useRef, useState } from 'react';

import type { MCPApprovalResolutionResult, MCPApprovalSummary } from '@/spec/mcp_artifact';
import { MCPApprovalResolution } from '@/spec/mcp_artifact';

import { mcpAPI } from '@/apis/baseapi';

export interface MCPApprovalRequest {
	approvalID: string;
	summary: MCPApprovalSummary;
	reason?: string;
}

export type RequestMCPApproval = (request: MCPApprovalRequest) => Promise<MCPApprovalResolutionResult>;

interface PendingMCPApprovalRequest {
	request: MCPApprovalRequest;
	resolve: (result: MCPApprovalResolutionResult) => void;
	reject: (error: Error) => void;
}

function getErrorMessage(error: unknown): string {
	return error instanceof Error && error.message.trim() ? error.message : 'Failed to resolve MCP approval.';
}

export function useMCPApproval() {
	const [approvalRequest, setApprovalRequest] = useState<MCPApprovalRequest | null>(null);
	const [approvalError, setApprovalError] = useState<string | null>(null);
	const [isResolving, setIsResolving] = useState(false);
	const activeApprovalRef = useRef<PendingMCPApprovalRequest | null>(null);
	const queuedApprovalsRef = useRef<PendingMCPApprovalRequest[]>([]);
	const advanceTimerRef = useRef<number | null>(null);
	const advancingApprovalRef = useRef(false);
	const resolvingApprovalIDRef = useRef<string | null>(null);

	const clearAdvanceTimer = useCallback(() => {
		advancingApprovalRef.current = false;
		if (advanceTimerRef.current === null || typeof window === 'undefined') {
			advanceTimerRef.current = null;
			return;
		}

		window.clearTimeout(advanceTimerRef.current);
		advanceTimerRef.current = null;
	}, []);

	const showNextApproval = useCallback(() => {
		if (activeApprovalRef.current) {
			return;
		}

		advancingApprovalRef.current = false;
		const next = queuedApprovalsRef.current.shift();
		setApprovalError(null);
		setIsResolving(false);
		if (!next) {
			setApprovalRequest(null);
			return;
		}

		activeApprovalRef.current = next;
		setApprovalRequest(next.request);
	}, []);

	const scheduleNextApproval = useCallback(() => {
		clearAdvanceTimer();
		advancingApprovalRef.current = true;

		if (typeof window === 'undefined') {
			showNextApproval();
			return;
		}

		advanceTimerRef.current = window.setTimeout(() => {
			advanceTimerRef.current = null;
			showNextApproval();
		}, 0);
	}, [clearAdvanceTimer, showNextApproval]);

	const resolveMCPApproval = useCallback(
		async (resolution: MCPApprovalResolution): Promise<void> => {
			const active = activeApprovalRef.current;
			if (!active) {
				return;
			}
			if (resolvingApprovalIDRef.current === active.request.approvalID) {
				return;
			}

			resolvingApprovalIDRef.current = active.request.approvalID;
			setApprovalError(null);
			setIsResolving(true);

			try {
				const result = await mcpAPI.resolveMCPApproval(active.request.approvalID, resolution);

				if (activeApprovalRef.current !== active) {
					return;
				}
				if (result.approvalID !== active.request.approvalID) {
					throw new Error('The backend resolved another MCP approval.');
				}

				activeApprovalRef.current = null;
				setApprovalRequest(null);
				active.resolve(result);
				scheduleNextApproval();
			} catch (error) {
				if (activeApprovalRef.current === active) {
					setApprovalError(getErrorMessage(error));
				}
			} finally {
				if (resolvingApprovalIDRef.current === active.request.approvalID) {
					resolvingApprovalIDRef.current = null;
					setIsResolving(false);
				}
			}
		},
		[scheduleNextApproval]
	);

	const requestMCPApproval = useCallback(
		(request: MCPApprovalRequest) => {
			return new Promise<MCPApprovalResolutionResult>((resolve, reject) => {
				queuedApprovalsRef.current.push({
					request,
					resolve,
					reject,
				});

				if (!advancingApprovalRef.current) {
					showNextApproval();
				}
			});
		},
		[showNextApproval]
	);

	useEffect(() => {
		return () => {
			clearAdvanceTimer();

			const active = activeApprovalRef.current;
			activeApprovalRef.current = null;
			resolvingApprovalIDRef.current = null;

			if (active) {
				void mcpAPI
					.resolveMCPApproval(active.request.approvalID, MCPApprovalResolution.DenyOnce)
					.catch(() => undefined);
				active.reject(new Error('MCP approval UI was closed.'));
			}

			// oxlint-disable-next-line react-hooks/exhaustive-deps
			const queued = queuedApprovalsRef.current.splice(0);
			for (const item of queued) {
				void mcpAPI.resolveMCPApproval(item.request.approvalID, MCPApprovalResolution.DenyOnce).catch(() => undefined);
				item.reject(new Error('MCP approval UI was closed.'));
			}
		};
	}, [clearAdvanceTimer]);

	return {
		approvalRequest,
		approvalError,
		isResolving,
		requestMCPApproval,
		resolveMCPApproval,
		clearMCPApproval: () => {
			void resolveMCPApproval(MCPApprovalResolution.DenyOnce);
		},
	};
}
