import { useCallback, useEffect, useRef, useState } from 'react';

import type { MCPApprovalSummary } from '@/spec/mcp';
import { MCPApprovalResolution } from '@/spec/mcp';

export interface MCPApprovalRequest {
	approvalID: string;
	summary: MCPApprovalSummary;
	reason?: string;
}

export type RequestMCPApproval = (request: MCPApprovalRequest) => Promise<MCPApprovalResolution>;

interface PendingMCPApprovalRequest {
	request: MCPApprovalRequest;
	resolve: (resolution: MCPApprovalResolution) => void;
}

export function useMCPApproval() {
	const [approvalRequest, setApprovalRequest] = useState<MCPApprovalRequest | null>(null);
	const activeApprovalRef = useRef<PendingMCPApprovalRequest | null>(null);
	const queuedApprovalsRef = useRef<PendingMCPApprovalRequest[]>([]);
	const advanceTimerRef = useRef<number | null>(null);
	const advancingApprovalRef = useRef(false);

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
		(resolution: MCPApprovalResolution) => {
			const active = activeApprovalRef.current;
			if (!active) {
				return;
			}

			activeApprovalRef.current = null;
			setApprovalRequest(null);
			active.resolve(resolution);
			scheduleNextApproval();
		},
		[scheduleNextApproval]
	);

	const requestMCPApproval = useCallback(
		(request: MCPApprovalRequest) => {
			return new Promise<MCPApprovalResolution>(resolve => {
				queuedApprovalsRef.current.push({
					request,
					resolve,
				});

				if (!advancingApprovalRef.current) {
					showNextApproval();
				}
			});
		},
		[showNextApproval]
	);

	useEffect(() => {
		const queuedCurrent = queuedApprovalsRef.current;
		const active = activeApprovalRef.current;
		return () => {
			clearAdvanceTimer();

			activeApprovalRef.current = null;
			active?.resolve(MCPApprovalResolution.MCPApprovalResolutionDenyOnce);

			const queued = queuedCurrent.splice(0);
			for (const item of queued) {
				item.resolve(MCPApprovalResolution.MCPApprovalResolutionDenyOnce);
			}
		};
	}, [clearAdvanceTimer]);

	return {
		approvalRequest,
		requestMCPApproval,
		resolveMCPApproval,
		clearMCPApproval: () => {
			resolveMCPApproval(MCPApprovalResolution.MCPApprovalResolutionDenyOnce);
		},
	};
}
