import type {
	MCPPromptRef,
	MCPResourceRef,
	MCPResourceTemplateRef,
	MCPRuntimeServerID,
	MCPServerRuntimeSnapshot,
	MCPToolCapability,
} from '@/spec/mcp';
import type {
	MCPCompleteArgumentRequestBody,
	MCPCompletionResult,
	MCPDiscoveryPage,
	MCPGetPromptResponseBody,
	MCPReadResourceResponseBody,
} from '@/spec/session_mcp';
import {
	MCPApprovalRule,
	MCPAppVisibility,
	MCPExecutionMode,
	MCPServerStatus,
	MCPTaskSupport,
	MCPToolRisk,
} from '@/spec/mcp';

import type { IMCPRuntimeAPI } from '@/apis/interface';
import {
	enumFromWails,
	optionalWailsString,
	requiredObject,
	requireWailsBoolean,
	wailsArrayOrEmpty,
	wailsObjectArrayOrEmpty,
} from '@/apis/wailsapi/transport';
import {
	CancelMCPServerAuthorization,
	CompleteMCPArgument,
	ConnectMCPServer,
	DisconnectMCPServer,
	GetMCPPrompt,
	ListMCPServerPrompts,
	ListMCPServerResources,
	ListMCPServerResourceTemplates,
	ListMCPServerTools,
	ReadMCPResource,
	RefreshMCPServer,
} from '@/apis/wailsjs/go/main/MCPRuntimeWrapper';

function runtimeSnapshotFromWails(value: unknown, operation: string): MCPServerRuntimeSnapshot {
	const snapshot = requiredObject<MCPServerRuntimeSnapshot>(value, operation);

	return {
		...snapshot,
		status: enumFromWails(snapshot.status, MCPServerStatus, `${operation}.status`),
	};
}

function mcpToolCapabilityFromWails(value: unknown, operation: string): MCPToolCapability {
	const tool = requiredObject<MCPToolCapability>(value, operation);

	return {
		...tool,
		inferredRisk: enumFromWails(tool.inferredRisk, MCPToolRisk, `${operation}.inferredRisk`),
		approvalRule: enumFromWails(tool.approvalRule, MCPApprovalRule, `${operation}.approvalRule`),
		executionMode: enumFromWails(tool.executionMode, MCPExecutionMode, `${operation}.executionMode`),
		taskSupport: enumFromWails(tool.taskSupport, MCPTaskSupport, `${operation}.taskSupport`),
		app:
			tool.app === undefined
				? undefined
				: {
						...tool.app,
						visibility: wailsArrayOrEmpty(tool.app.visibility, `${operation}.app.visibility`).map((item, index) =>
							enumFromWails(item, MCPAppVisibility, `${operation}.app.visibility[${index}]`)
						),
					},
	};
}

function discoveryPageFromWails<T extends object>(value: unknown, operation: string): MCPDiscoveryPage<T> {
	const page = requiredObject<MCPDiscoveryPage<T>>(value, operation);

	return {
		...page,
		items: wailsObjectArrayOrEmpty<T>(page.items, `${operation}.items`),
		nextPageToken: optionalWailsString(page.nextPageToken, `${operation}.nextPageToken`) || undefined,
	};
}

export class WailsMCPRuntimeAPI implements IMCPRuntimeAPI {
	async cancelMCPServerAuthorization(server: MCPRuntimeServerID): Promise<boolean> {
		return requireWailsBoolean(
			await CancelMCPServerAuthorization(server as Parameters<typeof CancelMCPServerAuthorization>[0]),
			'CancelMCPServerAuthorization'
		);
	}

	async completeMCPArgument(
		server: MCPRuntimeServerID,
		request: MCPCompleteArgumentRequestBody
	): Promise<MCPCompletionResult> {
		return requiredObject<MCPCompletionResult>(
			await CompleteMCPArgument(
				server as Parameters<typeof CompleteMCPArgument>[0],
				request as Parameters<typeof CompleteMCPArgument>[1]
			),
			'CompleteMCPArgument'
		);
	}

	async connectMCPServer(server: MCPRuntimeServerID): Promise<MCPServerRuntimeSnapshot> {
		return runtimeSnapshotFromWails(
			await ConnectMCPServer(server as Parameters<typeof ConnectMCPServer>[0]),
			'ConnectMCPServer'
		);
	}

	async disconnectMCPServer(server: MCPRuntimeServerID): Promise<void> {
		await DisconnectMCPServer(server as Parameters<typeof DisconnectMCPServer>[0]);
	}

	async getMCPPrompt(
		server: MCPRuntimeServerID,
		promptName: string,
		promptArguments: Record<string, string>
	): Promise<MCPGetPromptResponseBody> {
		return requiredObject<MCPGetPromptResponseBody>(
			await GetMCPPrompt(server as Parameters<typeof GetMCPPrompt>[0], promptName, promptArguments),
			'GetMCPPrompt'
		);
	}

	async listMCPServerPrompts(
		server: MCPRuntimeServerID,
		pageSize: number,
		pageToken = ''
	): Promise<MCPDiscoveryPage<MCPPromptRef>> {
		return discoveryPageFromWails<MCPPromptRef>(
			await ListMCPServerPrompts(server as Parameters<typeof ListMCPServerPrompts>[0], pageSize, pageToken),
			'ListMCPServerPrompts'
		);
	}

	async listMCPServerResources(
		server: MCPRuntimeServerID,
		pageSize: number,
		pageToken = ''
	): Promise<MCPDiscoveryPage<MCPResourceRef>> {
		return discoveryPageFromWails<MCPResourceRef>(
			await ListMCPServerResources(server as Parameters<typeof ListMCPServerResources>[0], pageSize, pageToken),
			'ListMCPServerResources'
		);
	}

	async listMCPServerResourceTemplates(
		server: MCPRuntimeServerID,
		pageSize: number,
		pageToken = ''
	): Promise<MCPDiscoveryPage<MCPResourceTemplateRef>> {
		return discoveryPageFromWails<MCPResourceTemplateRef>(
			await ListMCPServerResourceTemplates(
				server as Parameters<typeof ListMCPServerResourceTemplates>[0],
				pageSize,
				pageToken
			),
			'ListMCPServerResourceTemplates'
		);
	}

	async listMCPServerTools(
		server: MCPRuntimeServerID,
		pageSize: number,
		pageToken = ''
	): Promise<MCPDiscoveryPage<MCPToolCapability>> {
		const page = discoveryPageFromWails<MCPToolCapability>(
			await ListMCPServerTools(server as Parameters<typeof ListMCPServerTools>[0], pageSize, pageToken),
			'ListMCPServerTools'
		);

		return {
			...page,
			items: page.items.map((tool, index) => mcpToolCapabilityFromWails(tool, `ListMCPServerTools.items[${index}]`)),
		};
	}

	async readMCPResource(server: MCPRuntimeServerID, uri: string): Promise<MCPReadResourceResponseBody> {
		return requiredObject<MCPReadResourceResponseBody>(
			await ReadMCPResource(server as Parameters<typeof ReadMCPResource>[0], uri),
			'ReadMCPResource'
		);
	}

	async refreshMCPServer(server: MCPRuntimeServerID): Promise<MCPServerRuntimeSnapshot> {
		return runtimeSnapshotFromWails(
			await RefreshMCPServer(server as Parameters<typeof RefreshMCPServer>[0]),
			'RefreshMCPServer'
		);
	}
}
