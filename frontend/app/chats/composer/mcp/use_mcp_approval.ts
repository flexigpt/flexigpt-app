import { useCallback, useEffect, useRef, useState } from 'react';

import type { MCPApprovalSummary } from '@/spec/mcp';
import { MCPApprovalResolution } from '@/spec/mcp';

export interface MCPApprovalRequest {
	approvalID: string;
	summary: MCPApprovalSummary;
	reason?: string;
}

export type RequestMCPApproval = (request: MCPApprovalRequest) => Promise<MCPApprovalResolution>;

export function useMCPApproval() {
	const [approvalRequest, setApprovalRequest] = useState<MCPApprovalRequest | null>(null);
	const resolverRef = useRef<((resolution: MCPApprovalResolution) => void) | null>(null);

	const resolveMCPApproval = useCallback((resolution: MCPApprovalResolution) => {
		const resolver = resolverRef.current;
		resolverRef.current = null;
		setApprovalRequest(null);
		resolver?.(resolution);
	}, []);

	const requestMCPApproval = useCallback((request: MCPApprovalRequest) => {
		return new Promise<MCPApprovalResolution>(resolve => {
			if (resolverRef.current) {
				resolverRef.current(MCPApprovalResolution.MCPApprovalResolutionDenyOnce);
			}

			resolverRef.current = resolve;
			setApprovalRequest(request);
		});
	}, []);

	useEffect(() => {
		return () => {
			const resolver = resolverRef.current;
			resolverRef.current = null;
			resolver?.(MCPApprovalResolution.MCPApprovalResolutionDenyOnce);
		};
	}, []);

	return {
		approvalRequest,
		requestMCPApproval,
		resolveMCPApproval,
		clearMCPApproval: () => {
			resolveMCPApproval(MCPApprovalResolution.MCPApprovalResolutionDenyOnce);
		},
	};
}
