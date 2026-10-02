import type {
	InvokeMCPToolRequestBody,
	MCPApprovalEvaluation,
	MCPApprovalResolution,
	MCPApprovalResolutionResult,
	MCPCompleteArgumentRequestBody,
	MCPCompletionResult,
	MCPDiscoveryPage,
	MCPGetPromptResponseBody,
	MCPPromptRef,
	MCPReadResourceResponseBody,
	MCPResourceRef,
	MCPResourceTemplateRef,
	MCPRuntimeInvokeToolResponse,
	MCPRuntimeServerID,
	MCPServerRuntimeSnapshot,
	MCPToolCapability,
} from '@/spec/mcp';

import type { IMCPRuntimeAPI } from '@/apis/interface';
import {
	optionalWailsString,
	requiredObject,
	requireWailsBoolean,
	wailsObjectArrayOrEmpty,
} from '@/apis/wailsapi/transport';
import {
	CancelMCPServerAuthorization,
	CheckMCPToolCall,
	CompleteMCPArgument,
	ConnectMCPServer,
	DisconnectMCPServer,
	GetMCPPrompt,
	InvokeMCPTool,
	ListMCPServerPrompts,
	ListMCPServerResources,
	ListMCPServerResourceTemplates,
	ListMCPServerTools,
	ReadMCPResource,
	RefreshMCPServer,
	ResolveMCPToolApproval,
} from '@/apis/wailsjs/go/main/MCPRuntimeWrapper';

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
		return requiredObject<MCPServerRuntimeSnapshot>(
			await ConnectMCPServer(server as Parameters<typeof ConnectMCPServer>[0]),
			'ConnectMCPServer'
		);
	}

	async disconnectMCPServer(server: MCPRuntimeServerID): Promise<void> {
		await DisconnectMCPServer(server as Parameters<typeof DisconnectMCPServer>[0]);
	}

	async checkMCPToolCall(
		server: MCPRuntimeServerID,
		request: InvokeMCPToolRequestBody
	): Promise<MCPApprovalEvaluation> {
		return requiredObject<MCPApprovalEvaluation>(
			await CheckMCPToolCall(
				server as Parameters<typeof CheckMCPToolCall>[0],
				request as Parameters<typeof CheckMCPToolCall>[1]
			),
			'CheckMCPToolCall'
		);
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

	async invokeMCPTool(
		server: MCPRuntimeServerID,
		request: InvokeMCPToolRequestBody
	): Promise<MCPRuntimeInvokeToolResponse> {
		return requiredObject<MCPRuntimeInvokeToolResponse>(
			await InvokeMCPTool(
				server as Parameters<typeof InvokeMCPTool>[0],
				request as Parameters<typeof InvokeMCPTool>[1]
			),
			'InvokeMCPTool'
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
		return discoveryPageFromWails<MCPToolCapability>(
			await ListMCPServerTools(server as Parameters<typeof ListMCPServerTools>[0], pageSize, pageToken),
			'ListMCPServerTools'
		);
	}

	async readMCPResource(server: MCPRuntimeServerID, uri: string): Promise<MCPReadResourceResponseBody> {
		return requiredObject<MCPReadResourceResponseBody>(
			await ReadMCPResource(server as Parameters<typeof ReadMCPResource>[0], uri),
			'ReadMCPResource'
		);
	}

	async refreshMCPServer(server: MCPRuntimeServerID): Promise<MCPServerRuntimeSnapshot> {
		return requiredObject<MCPServerRuntimeSnapshot>(
			await RefreshMCPServer(server as Parameters<typeof RefreshMCPServer>[0]),
			'RefreshMCPServer'
		);
	}

	async resolveMCPToolApproval(
		approvalID: string,
		resolution: MCPApprovalResolution
	): Promise<MCPApprovalResolutionResult> {
		return requiredObject<MCPApprovalResolutionResult>(
			await ResolveMCPToolApproval(approvalID, resolution as Parameters<typeof ResolveMCPToolApproval>[1]),
			'ResolveMCPToolApproval'
		);
	}
}
