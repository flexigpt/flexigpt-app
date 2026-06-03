import type { UIToolOutput } from '@/spec/inference';
import type { MCPToolSelection } from '@/spec/mcp';

/**
 * @public
 *
 * Runtime-only handle for one MCP App view. Apps are never persisted; this
 * exists for the lifetime of the rendered iframe.
 */
export interface MCPAppInstance {
	instanceID: string;
	bundleID: string;
	serverID: string;
	resourceUri: string;
	mimeType?: string;
	toolName: string;
	toolUseID: string;
	callID: string;
	displayName?: string;
}

/** JSON-RPC 2.0 envelope. */
export interface JSONRPCRequest {
	jsonrpc: '2.0';
	id: number | string;
	method: string;
	params?: unknown;
}

export interface JSONRPCNotification {
	jsonrpc: '2.0';
	method: string;
	params?: unknown;
}

export interface JSONRPCResponse {
	jsonrpc: '2.0';
	id: number | string;
	result?: unknown;
	error?: { code: number; message: string; data?: unknown };
}

export type JSONRPCMessage = JSONRPCRequest | JSONRPCNotification | JSONRPCResponse;

export const JSONRPC_ERR_METHOD_NOT_FOUND = -32601;
export const JSONRPC_ERR_INVALID_PARAMS = -32602;
export const JSONRPC_ERR_BLOCKED_BY_POLICY = -32001;

export function isJSONRPCRequest(value: unknown): value is JSONRPCRequest {
	if (!value || typeof value !== 'object') return false;
	const m = value as Partial<JSONRPCRequest>;
	return m.jsonrpc === '2.0' && (typeof m.id === 'number' || typeof m.id === 'string') && typeof m.method === 'string';
}

export function isJSONRPCNotification(value: unknown): value is JSONRPCNotification {
	if (!value || typeof value !== 'object') return false;
	const m = value as Partial<JSONRPCNotification>;
	return m.jsonrpc === '2.0' && typeof m.method === 'string' && !('id' in m);
}

/** Build an instance from a tool output that has app render info. */
export function buildAppInstanceFromToolOutput(output: UIToolOutput): MCPAppInstance | undefined {
	const app = output.mcpApp;
	const sel: MCPToolSelection | undefined = output.mcpToolSelection;
	if (!app?.resourceUri || !sel?.bundleID || !sel.serverID) return undefined;

	const resourceUri = app.resourceUri;
	const instanceID = `mcpapp-${sel.serverID}-${output.callID || output.id}`;
	return {
		instanceID,
		bundleID: sel.bundleID,
		serverID: sel.serverID,
		resourceUri,
		mimeType: app.mimeType,
		toolName: sel.toolName || output.name,
		toolUseID: output.callID || output.id,
		callID: output.callID || output.id,
		displayName: output.name,
	};
}
