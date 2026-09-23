import type { UIToolOutput } from '@/spec/inference';
import type { MCPRuntimeServerID, MCPToolSelection } from '@/spec/mcp';

/**
 * Runtime-only handle for one MCP App view. Apps are never persisted; this
 * exists for the lifetime of the rendered iframe.
 */
export interface MCPAppInstance {
	instanceID: string;
	server: MCPRuntimeServerID;
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

function isJSONRPCObject(value: unknown): value is Record<string, unknown> {
	return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function isJSONRPCID(value: unknown): value is number | string {
	return typeof value === 'string' || (typeof value === 'number' && Number.isFinite(value));
}

function isJSONRPCError(value: unknown): value is NonNullable<JSONRPCResponse['error']> {
	return (
		isJSONRPCObject(value) &&
		typeof value.code === 'number' &&
		Number.isInteger(value.code) &&
		typeof value.message === 'string'
	);
}

function hasOwn(value: object, key: PropertyKey): boolean {
	return Object.hasOwn(value, key);
}

export function isJSONRPCResponse(value: unknown): value is JSONRPCResponse {
	if (
		!isJSONRPCObject(value) ||
		value.jsonrpc !== '2.0' ||
		!hasOwn(value, 'id') ||
		!isJSONRPCID(value.id) ||
		hasOwn(value, 'method')
	) {
		return false;
	}

	const hasResult = hasOwn(value, 'result');
	const hasError = hasOwn(value, 'error');
	if (hasResult === hasError) {
		return false;
	}

	return !hasError || isJSONRPCError(value.error);
}

export function isJSONRPCRequest(value: unknown): value is JSONRPCRequest {
	return (
		isJSONRPCObject(value) &&
		value.jsonrpc === '2.0' &&
		hasOwn(value, 'id') &&
		isJSONRPCID(value.id) &&
		typeof value.method === 'string'
	);
}

export function isJSONRPCNotification(value: unknown): value is JSONRPCNotification {
	return isJSONRPCObject(value) && value.jsonrpc === '2.0' && typeof value.method === 'string' && !hasOwn(value, 'id');
}

/** Build an instance from a tool output that has app render info. */
export function buildAppInstanceFromToolOutput(output: UIToolOutput): MCPAppInstance | undefined {
	const app = output.mcpApp;
	const sel: MCPToolSelection | undefined = output.mcpToolSelection;
	if (!app?.resourceUri || !sel?.server) {
		return undefined;
	}

	const resourceUri = app.resourceUri;
	const instanceID = `mcpapp-${sel.server}-${output.callID || output.id}`;
	return {
		instanceID,
		server: sel.server,
		resourceUri,
		mimeType: app.mimeType,
		toolName: sel.toolName || output.name,
		toolUseID: output.callID || output.id,
		callID: output.callID || output.id,
		displayName: output.name,
	};
}
