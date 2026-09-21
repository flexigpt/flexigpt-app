import type {
	InvokeMCPToolRequestBody,
	MCPApprovalEvaluation,
	MCPApprovalResolution,
	MCPApprovalResolutionResult,
	MCPAuthSettings,
	MCPCompleteArgumentRequestBody,
	MCPCompletionResult,
	MCPDiscoveryPage,
	MCPGetPromptResponseBody,
	MCPGlobalSettings,
	MCPOAuthAuthorization,
	MCPPromptRef,
	MCPProviderToolMapping,
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
	requiredObject,
	requireWailsBoolean,
	requireWailsFiniteNumber,
	wailsObjectArrayOrEmpty,
} from '@/apis/wailsapi/transport';
import {
	CancelPendingMCPOAuthAuthorization,
	CompleteMCPArgument,
	ConnectMCPServer,
	DisconnectMCPServer,
	EvaluateMappedMCPToolCall,
	EvaluateMCPToolCall,
	GetMCPGlobalSettings,
	GetMCPPrompt,
	GetMCPServerStatus,
	InvokeMappedMCPTool,
	InvokeMCPTool,
	ListMCPServerPrompts,
	ListMCPServerPromptsPage,
	ListMCPServerResources,
	ListMCPServerResourcesPage,
	ListMCPServerResourceTemplates,
	ListMCPServerResourceTemplatesPage,
	ListMCPServerTools,
	ListMCPServerToolsPage,
	ListPendingMCPOAuthAuthorizations,
	ReadMCPResource,
	RefreshMCPServer,
	ResolveMCPApproval,
	StartMCPServerConnect,
	UpdateMCPGlobalSettings,
} from '@/apis/wailsjs/go/main/MCPRuntimeWrapper';

export class WailsMCPRuntimeAPI implements IMCPRuntimeAPI {
	async cancelPendingMCPOAuthAuthorization(server: MCPRuntimeServerID): Promise<boolean> {
		return requireWailsBoolean(
			await CancelPendingMCPOAuthAuthorization(server as Parameters<typeof CancelPendingMCPOAuthAuthorization>[0]),
			'CancelPendingMCPOAuthAuthorization'
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

	async evaluateMappedMCPToolCall(
		mapping: MCPProviderToolMapping,
		request: InvokeMCPToolRequestBody
	): Promise<MCPApprovalEvaluation> {
		return requiredObject<MCPApprovalEvaluation>(
			await EvaluateMappedMCPToolCall(
				mapping as Parameters<typeof EvaluateMappedMCPToolCall>[0],
				request as Parameters<typeof EvaluateMappedMCPToolCall>[1]
			),
			'EvaluateMappedMCPToolCall'
		);
	}

	async evaluateMCPToolCall(
		server: MCPRuntimeServerID,
		request: InvokeMCPToolRequestBody
	): Promise<MCPApprovalEvaluation> {
		return requiredObject<MCPApprovalEvaluation>(
			await EvaluateMCPToolCall(
				server as Parameters<typeof EvaluateMCPToolCall>[0],
				request as Parameters<typeof EvaluateMCPToolCall>[1]
			),
			'EvaluateMCPToolCall'
		);
	}

	async getMCPGlobalSettings(): Promise<MCPGlobalSettings> {
		return requiredObject<MCPGlobalSettings>(await GetMCPGlobalSettings(), 'GetMCPGlobalSettings');
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

	async getMCPServerStatus(server: MCPRuntimeServerID): Promise<MCPServerRuntimeSnapshot> {
		return requiredObject<MCPServerRuntimeSnapshot>(
			await GetMCPServerStatus(server as Parameters<typeof GetMCPServerStatus>[0]),
			'GetMCPServerStatus'
		);
	}

	async invokeMappedMCPTool(
		mapping: MCPProviderToolMapping,
		request: InvokeMCPToolRequestBody
	): Promise<MCPRuntimeInvokeToolResponse> {
		return requiredObject<MCPRuntimeInvokeToolResponse>(
			await InvokeMappedMCPTool(
				mapping as Parameters<typeof InvokeMappedMCPTool>[0],
				request as Parameters<typeof InvokeMappedMCPTool>[1]
			),
			'InvokeMappedMCPTool'
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

	async listMCPServerPrompts(server: MCPRuntimeServerID): Promise<MCPPromptRef[]> {
		return wailsObjectArrayOrEmpty<MCPPromptRef>(
			await ListMCPServerPrompts(server as Parameters<typeof ListMCPServerPrompts>[0]),
			'ListMCPServerPrompts'
		);
	}

	async listMCPServerPromptsPage(
		server: MCPRuntimeServerID,
		pageSize: number,
		pageToken = ''
	): Promise<MCPDiscoveryPage<MCPPromptRef>> {
		return requiredObject<MCPDiscoveryPage<MCPPromptRef>>(
			await ListMCPServerPromptsPage(server as Parameters<typeof ListMCPServerPromptsPage>[0], pageSize, pageToken),
			'ListMCPServerPromptsPage'
		);
	}

	async listMCPServerResources(server: MCPRuntimeServerID): Promise<MCPResourceRef[]> {
		return wailsObjectArrayOrEmpty<MCPResourceRef>(
			await ListMCPServerResources(server as Parameters<typeof ListMCPServerResources>[0]),
			'ListMCPServerResources'
		);
	}

	async listMCPServerResourcesPage(
		server: MCPRuntimeServerID,
		pageSize: number,
		pageToken = ''
	): Promise<MCPDiscoveryPage<MCPResourceRef>> {
		return requiredObject<MCPDiscoveryPage<MCPResourceRef>>(
			await ListMCPServerResourcesPage(server as Parameters<typeof ListMCPServerResourcesPage>[0], pageSize, pageToken),
			'ListMCPServerResourcesPage'
		);
	}

	async listMCPServerResourceTemplates(server: MCPRuntimeServerID): Promise<MCPResourceTemplateRef[]> {
		return wailsObjectArrayOrEmpty<MCPResourceTemplateRef>(
			await ListMCPServerResourceTemplates(server as Parameters<typeof ListMCPServerResourceTemplates>[0]),
			'ListMCPServerResourceTemplates'
		);
	}

	async listMCPServerResourceTemplatesPage(
		server: MCPRuntimeServerID,
		pageSize: number,
		pageToken = ''
	): Promise<MCPDiscoveryPage<MCPResourceTemplateRef>> {
		return requiredObject<MCPDiscoveryPage<MCPResourceTemplateRef>>(
			await ListMCPServerResourceTemplatesPage(
				server as Parameters<typeof ListMCPServerResourceTemplatesPage>[0],
				pageSize,
				pageToken
			),
			'ListMCPServerResourceTemplatesPage'
		);
	}

	async listMCPServerTools(server: MCPRuntimeServerID): Promise<MCPToolCapability[]> {
		return wailsObjectArrayOrEmpty<MCPToolCapability>(
			await ListMCPServerTools(server as Parameters<typeof ListMCPServerTools>[0]),
			'ListMCPServerTools'
		);
	}

	async listMCPServerToolsPage(
		server: MCPRuntimeServerID,
		pageSize: number,
		pageToken = ''
	): Promise<MCPDiscoveryPage<MCPToolCapability>> {
		return requiredObject<MCPDiscoveryPage<MCPToolCapability>>(
			await ListMCPServerToolsPage(server as Parameters<typeof ListMCPServerToolsPage>[0], pageSize, pageToken),
			'ListMCPServerToolsPage'
		);
	}

	async listPendingMCPOAuthAuthorizations(): Promise<MCPOAuthAuthorization[]> {
		return wailsObjectArrayOrEmpty<MCPOAuthAuthorization>(
			await ListPendingMCPOAuthAuthorizations(),
			'ListPendingMCPOAuthAuthorizations'
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

	async resolveMCPApproval(
		approvalID: string,
		resolution: MCPApprovalResolution
	): Promise<MCPApprovalResolutionResult> {
		return requiredObject<MCPApprovalResolutionResult>(
			await ResolveMCPApproval(approvalID, resolution as Parameters<typeof ResolveMCPApproval>[1]),
			'ResolveMCPApproval'
		);
	}

	async startMCPServerConnect(server: MCPRuntimeServerID): Promise<MCPServerRuntimeSnapshot> {
		return requiredObject<MCPServerRuntimeSnapshot>(
			await StartMCPServerConnect(server as Parameters<typeof StartMCPServerConnect>[0]),
			'StartMCPServerConnect'
		);
	}

	async updateMCPGlobalSettings(expectedRevision: number, settings: MCPAuthSettings): Promise<number> {
		return requireWailsFiniteNumber(
			await UpdateMCPGlobalSettings(expectedRevision, settings as Parameters<typeof UpdateMCPGlobalSettings>[1]),
			'UpdateMCPGlobalSettings'
		);
	}
}
