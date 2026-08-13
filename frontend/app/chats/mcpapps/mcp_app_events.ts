import type { MCPAppModelContextUpdate } from '@/spec/mcp_artifact';

import type { MCPAppInstance } from '@/chats/mcpapps/mcp_app_types';

export const MCP_APP_UI_MESSAGE_EVENT = 'flexigpt:mcp-app-ui-message';
export const MCP_APP_MODEL_CONTEXT_UPDATE_EVENT = 'flexigpt:mcp-app-model-context-update';

export interface MCPAppUIMessage {
	role: 'user';
	text: string;
}

export interface MCPAppUIMessageEventDetail {
	instance: MCPAppInstance;
	message: MCPAppUIMessage;
}

export interface MCPAppModelContextUpdateEventDetail {
	instance: MCPAppInstance;
	update: MCPAppModelContextUpdate;
}

export type MCPAppModelContextUpdatePayload = Omit<
	MCPAppModelContextUpdate,
	'instanceID' | 'resourceUri' | 'server' | 'updatedAt'
>;

export function dispatchMCPAppUIMessage(instance: MCPAppInstance, message: MCPAppUIMessage) {
	window.dispatchEvent(
		new CustomEvent<MCPAppUIMessageEventDetail>(MCP_APP_UI_MESSAGE_EVENT, {
			detail: {
				instance,
				message,
			},
		})
	);
}

export function dispatchMCPAppModelContextUpdate(instance: MCPAppInstance, update: MCPAppModelContextUpdatePayload) {
	window.dispatchEvent(
		new CustomEvent<MCPAppModelContextUpdateEventDetail>(MCP_APP_MODEL_CONTEXT_UPDATE_EVENT, {
			detail: {
				instance,
				update: {
					...update,
					instanceID: instance.instanceID,
					server: instance.server,
					resourceUri: instance.resourceUri,
					updatedAt: new Date().toISOString(),
				},
			},
		})
	);
}
