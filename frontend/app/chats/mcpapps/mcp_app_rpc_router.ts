import { type InvokeMCPToolRequestBody, MCPApprovalDecision, MCPInvocationSource } from '@/spec/mcp';

import { mcpAPI } from '@/apis/baseapi';

import {
	JSONRPC_ERR_BLOCKED_BY_POLICY,
	JSONRPC_ERR_INVALID_PARAMS,
	JSONRPC_ERR_METHOD_NOT_FOUND,
	type JSONRPCRequest,
	type JSONRPCResponse,
	type MCPAppInstance,
} from '@/chats/mcpapps/mcp_app_types';

export interface MCPAppRouterDeps {
	instance: MCPAppInstance;
	/** Returns true if the user approves opening this URL. */
	requestOpenLinkApproval: (url: string) => Promise<boolean>;
	/** Routes a "log" notification from the app for the diagnostics surface. */
	onAppLog?: (level: string, data: unknown) => void;
}

/**
 * Routes app->host JSON-RPC requests with same-server and policy enforcement.
 *
 * Supported view-to-host requests:
 *   - tools/call        (forwarded to backend with source="app")
 *   - resources/read    (same server only)
 *   - ui/open-link      (with user approval)
 *
 * Everything else returns method-not-found. The backend is the final
 * authority on policy decisions; this router is just a guard rail.
 */
export class MCPAppRPCRouter {
	private readonly deps: MCPAppRouterDeps;

	constructor(deps: MCPAppRouterDeps) {
		this.deps = deps;
	}

	async handle(req: JSONRPCRequest): Promise<JSONRPCResponse> {
		switch (req.method) {
			case 'ping':
				return { jsonrpc: '2.0', id: req.id, result: {} };
			case 'tools/call':
				return this.handleToolCall(req);
			case 'resources/read':
				return this.handleResourceRead(req);
			case 'ui/open-link':
				return this.handleOpenLink(req);
			case 'ui/request-display-mode':
				return this.handleDisplayMode(req);
			case 'ui/message':
				return errorResp(req.id, JSONRPC_ERR_BLOCKED_BY_POLICY, 'ui/message is not enabled in this host surface');
			case 'ui/update-model-context':
				return errorResp(
					req.id,
					JSONRPC_ERR_BLOCKED_BY_POLICY,
					'ui/update-model-context is not enabled in this host surface'
				);
			default:
				return {
					jsonrpc: '2.0',
					id: req.id,
					error: { code: JSONRPC_ERR_METHOD_NOT_FOUND, message: `Method ${req.method} is not supported` },
				};
		}
	}

	private async handleToolCall(req: JSONRPCRequest): Promise<JSONRPCResponse> {
		const params = (req.params as Record<string, unknown>) ?? {};
		const name = typeof params.name === 'string' ? params.name : '';
		const args = (params.arguments as Record<string, unknown>) ?? undefined;
		if (!name) {
			return errorResp(req.id, JSONRPC_ERR_INVALID_PARAMS, 'tools/call requires a name');
		}

		const { bundleID, serverID, instanceID } = this.deps.instance;
		const callReq: InvokeMCPToolRequestBody = {
			source: MCPInvocationSource.MCPInvocationSourceApp,
			serverID,
			toolName: name,
			providerToolName: name,
			arguments: args,
			appInstanceID: instanceID,
		};

		// Backend enforces appsPolicy + cross-server. Frontend only needs to
		// route via the same bundle/server the app belongs to.
		const evaluation = await mcpAPI.evaluateMCPToolCall(bundleID, callReq);
		if (!evaluation) {
			return errorResp(req.id, JSONRPC_ERR_BLOCKED_BY_POLICY, 'MCP could not evaluate this tool call');
		}
		if (evaluation.decision === MCPApprovalDecision.MCPApprovalDecisionDenied) {
			return errorResp(req.id, JSONRPC_ERR_BLOCKED_BY_POLICY, evaluation.reason || 'Denied by policy');
		}
		if (evaluation.decision === MCPApprovalDecision.MCPApprovalDecisionApprovalRequired) {
			// Approval modal is owned by composer; for app-initiated calls we
			// fail closed and surface the reason to the app.
			return errorResp(req.id, JSONRPC_ERR_BLOCKED_BY_POLICY, evaluation.reason || 'Approval required');
		}
		// Allowed.
		try {
			const resp = await mcpAPI.invokeMCPTool(bundleID, callReq);
			return { jsonrpc: '2.0', id: req.id, result: resp };
		} catch (err) {
			return errorResp(
				req.id,
				JSONRPC_ERR_BLOCKED_BY_POLICY,
				err instanceof Error ? err.message : 'Tool invocation failed'
			);
		}
	}

	private async handleResourceRead(req: JSONRPCRequest): Promise<JSONRPCResponse> {
		const params = (req.params as Record<string, unknown>) ?? {};
		const uri = typeof params.uri === 'string' ? params.uri : '';
		if (!uri) return errorResp(req.id, JSONRPC_ERR_INVALID_PARAMS, 'resources/read requires uri');

		const { bundleID, serverID } = this.deps.instance;
		try {
			const resp = await mcpAPI.readMCPResource(bundleID, serverID, uri);
			return { jsonrpc: '2.0', id: req.id, result: resp };
		} catch (err) {
			return errorResp(req.id, JSONRPC_ERR_BLOCKED_BY_POLICY, err instanceof Error ? err.message : 'Read failed');
		}
	}

	private handleDisplayMode(req: JSONRPCRequest): JSONRPCResponse {
		const params = (req.params as Record<string, unknown>) ?? {};
		const requested = typeof params.mode === 'string' ? params.mode : 'inline';

		if (requested !== 'inline') {
			return {
				jsonrpc: '2.0',
				id: req.id,
				result: { mode: 'inline' },
			};
		}
		return { jsonrpc: '2.0', id: req.id, result: { mode: 'inline' } };
	}

	private async handleOpenLink(req: JSONRPCRequest): Promise<JSONRPCResponse> {
		const params = (req.params as Record<string, unknown>) ?? {};
		const url = typeof params.url === 'string' ? params.url : '';
		if (!url) return errorResp(req.id, JSONRPC_ERR_INVALID_PARAMS, 'ui/open-link requires url');

		try {
			const ok = await this.deps.requestOpenLinkApproval(url);
			if (!ok) return errorResp(req.id, JSONRPC_ERR_BLOCKED_BY_POLICY, 'User denied opening the link');
			return { jsonrpc: '2.0', id: req.id, result: { opened: true } };
		} catch (err) {
			return errorResp(req.id, JSONRPC_ERR_BLOCKED_BY_POLICY, err instanceof Error ? err.message : 'Open denied');
		}
	}
}

function errorResp(id: JSONRPCRequest['id'], code: number, message: string): JSONRPCResponse {
	return { jsonrpc: '2.0', id, error: { code, message } };
}
