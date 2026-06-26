import type {
	InvokeMCPToolRequestBody,
	InvokeMCPToolResponseBody,
	MCPAppModelContextUpdate,
	MCPContent,
} from '@/spec/mcp';
import { MCPApprovalDecision, MCPApprovalResolution, MCPInvocationSource } from '@/spec/mcp';

import { isJSONObject } from '@/lib/jsonschema_utils';

import { mcpAPI } from '@/apis/baseapi';

import type { MCPApprovalRequest } from '@/chats/composer/mcp/use_mcp_approval';
import type { MCPAppUIMessage } from '@/chats/mcpapps/mcp_app_events';
import type { JSONRPCRequest, JSONRPCResponse, MCPAppInstance } from '@/chats/mcpapps/mcp_app_types';
import {
	JSONRPC_ERR_BLOCKED_BY_POLICY,
	JSONRPC_ERR_INVALID_PARAMS,
	JSONRPC_ERR_METHOD_NOT_FOUND,
} from '@/chats/mcpapps/mcp_app_types';

function errorResp(id: JSONRPCRequest['id'], code: number, message: string): JSONRPCResponse {
	return { jsonrpc: '2.0', id, error: { code, message } };
}

function normalizeToolCallResultForApp(resp: InvokeMCPToolResponseBody | undefined): {
	content: MCPContent[];
	structuredContent?: Record<string, unknown>;
	isError?: boolean;
} {
	const result: {
		content: MCPContent[];
		structuredContent?: Record<string, unknown>;
		isError?: boolean;
	} = {
		content: Array.isArray(resp?.content) ? resp.content : [],
	};

	if (isJSONObject(resp?.structuredContent)) {
		result.structuredContent = resp.structuredContent;
	}

	if (typeof resp?.isError === 'boolean') {
		result.isError = resp.isError;
	}

	return result;
}

export interface MCPAppRouterDeps {
	instance: MCPAppInstance;
	/** Returns true if the user approves opening this URL. */
	requestOpenLinkApproval: (url: string) => Promise<boolean>;
	requestMCPApproval?: (request: MCPApprovalRequest) => Promise<MCPApprovalResolution>;
	requestUIMessageApproval?: (message: MCPAppUIMessage) => Promise<boolean>;
	onUIMessage?: (message: MCPAppUIMessage) => void;
	requestModelContextUpdateApproval?: (
		update: Omit<MCPAppModelContextUpdate, 'instanceID' | 'bundleID' | 'serverID' | 'resourceUri' | 'updatedAt'>
	) => Promise<boolean>;
	onModelContextUpdate?: (
		update: Omit<MCPAppModelContextUpdate, 'instanceID' | 'bundleID' | 'serverID' | 'resourceUri' | 'updatedAt'>
	) => void;
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
				return this.handleUIMessage(req);
			case 'ui/update-model-context':
				return this.handleUpdateModelContext(req);

			default:
				return {
					jsonrpc: '2.0',
					id: req.id,
					error: { code: JSONRPC_ERR_METHOD_NOT_FOUND, message: `Method ${req.method} is not supported` },
				};
		}
	}

	private async handleToolCall(req: JSONRPCRequest): Promise<JSONRPCResponse> {
		const params = isJSONObject(req.params) ? req.params : {};

		const name = typeof params.name === 'string' ? params.name : '';

		if (!name) {
			return errorResp(req.id, JSONRPC_ERR_INVALID_PARAMS, 'tools/call requires a name');
		}
		let args: Record<string, unknown> | undefined;
		if (params.arguments !== undefined) {
			if (!isJSONObject(params.arguments)) {
				return errorResp(req.id, JSONRPC_ERR_INVALID_PARAMS, 'tools/call arguments must be an object');
			}
			args = params.arguments;
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
			if (!evaluation.approvalID || !evaluation.summary || !this.deps.requestMCPApproval) {
				return errorResp(req.id, JSONRPC_ERR_BLOCKED_BY_POLICY, evaluation.reason || 'Approval required');
			}

			const resolution = await this.deps.requestMCPApproval({
				approvalID: evaluation.approvalID,
				summary: evaluation.summary,
				reason: evaluation.reason,
			});

			const token = await mcpAPI.resolveMCPApproval(evaluation.approvalID, resolution);

			if (
				resolution !== MCPApprovalResolution.MCPApprovalResolutionAllowOnce &&
				resolution !== MCPApprovalResolution.MCPApprovalResolutionAllowAlways
			) {
				return errorResp(req.id, JSONRPC_ERR_BLOCKED_BY_POLICY, 'User denied this tool call');
			}

			if (!token?.token) {
				return errorResp(req.id, JSONRPC_ERR_BLOCKED_BY_POLICY, 'Approval did not return a usable token');
			}

			callReq.approvalID = token.approvalID;
			callReq.approvalToken = token.token;
		}
		// Allowed.
		try {
			const resp = await mcpAPI.invokeMCPTool(bundleID, callReq);
			return { jsonrpc: '2.0', id: req.id, result: normalizeToolCallResultForApp(resp) };
		} catch (err) {
			return errorResp(
				req.id,
				JSONRPC_ERR_BLOCKED_BY_POLICY,
				err instanceof Error ? err.message : 'Tool invocation failed'
			);
		}
	}

	private async handleResourceRead(req: JSONRPCRequest): Promise<JSONRPCResponse> {
		const params = isJSONObject(req.params) ? req.params : {};

		const uri = typeof params.uri === 'string' ? params.uri : '';
		if (!uri) {
			return errorResp(req.id, JSONRPC_ERR_INVALID_PARAMS, 'resources/read requires uri');
		}

		const { bundleID, serverID } = this.deps.instance;
		try {
			const resp = await mcpAPI.readMCPResource(bundleID, serverID, uri);
			return { jsonrpc: '2.0', id: req.id, result: resp };
		} catch (err) {
			return errorResp(req.id, JSONRPC_ERR_BLOCKED_BY_POLICY, err instanceof Error ? err.message : 'Read failed');
		}
	}

	private handleDisplayMode(req: JSONRPCRequest): JSONRPCResponse {
		const params = isJSONObject(req.params) ? req.params : {};

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

	private async handleUIMessage(req: JSONRPCRequest): Promise<JSONRPCResponse> {
		const params = isJSONObject(req.params) ? req.params : {};

		const role = params.role === 'user' ? 'user' : '';
		const content = isJSONObject(params.content) ? params.content : undefined;

		const text = content?.type === 'text' && typeof content.text === 'string' ? content.text.trim() : '';

		if (role !== 'user' || !text) {
			return errorResp(req.id, JSONRPC_ERR_INVALID_PARAMS, 'ui/message requires role=user and text content');
		}

		const message: MCPAppUIMessage = { role: 'user', text };
		const approved = this.deps.requestUIMessageApproval ? await this.deps.requestUIMessageApproval(message) : false;

		if (!approved) {
			return errorResp(req.id, JSONRPC_ERR_BLOCKED_BY_POLICY, 'User denied adding this message');
		}

		this.deps.onUIMessage?.(message);
		return { jsonrpc: '2.0', id: req.id, result: {} };
	}

	private async handleUpdateModelContext(req: JSONRPCRequest): Promise<JSONRPCResponse> {
		const params = isJSONObject(req.params) ? req.params : {};

		let content: any;
		if (params.content !== undefined) {
			if (!Array.isArray(params.content)) {
				return errorResp(req.id, JSONRPC_ERR_INVALID_PARAMS, 'ui/update-model-context content must be an array');
			}
			content = params.content;
		}

		let structuredContent: Record<string, unknown> | undefined;
		if (params.structuredContent !== undefined) {
			if (!isJSONObject(params.structuredContent)) {
				return errorResp(
					req.id,
					JSONRPC_ERR_INVALID_PARAMS,
					'ui/update-model-context structuredContent must be an object'
				);
			}
			structuredContent = params.structuredContent;
		}
		const update = {
			content,
			...(structuredContent !== undefined ? { structuredContent } : {}),
		} satisfies Omit<MCPAppModelContextUpdate, 'instanceID' | 'bundleID' | 'serverID' | 'resourceUri' | 'updatedAt'>;

		if (!content && structuredContent === undefined) {
			return errorResp(
				req.id,
				JSONRPC_ERR_INVALID_PARAMS,
				'ui/update-model-context requires content or structuredContent'
			);
		}

		const approved = this.deps.requestModelContextUpdateApproval
			? await this.deps.requestModelContextUpdateApproval(update)
			: false;

		if (!approved) {
			return errorResp(req.id, JSONRPC_ERR_BLOCKED_BY_POLICY, 'User denied the model context update');
		}

		this.deps.onModelContextUpdate?.(update);
		return { jsonrpc: '2.0', id: req.id, result: {} };
	}

	private async handleOpenLink(req: JSONRPCRequest): Promise<JSONRPCResponse> {
		const params = isJSONObject(req.params) ? req.params : {};

		const url = typeof params.url === 'string' ? params.url : '';
		if (!url) {
			return errorResp(req.id, JSONRPC_ERR_INVALID_PARAMS, 'ui/open-link requires url');
		}

		try {
			const ok = await this.deps.requestOpenLinkApproval(url);
			if (!ok) {
				return errorResp(req.id, JSONRPC_ERR_BLOCKED_BY_POLICY, 'User denied opening the link');
			}
			return { jsonrpc: '2.0', id: req.id, result: { opened: true } };
		} catch (err) {
			return errorResp(req.id, JSONRPC_ERR_BLOCKED_BY_POLICY, err instanceof Error ? err.message : 'Open denied');
		}
	}
}
