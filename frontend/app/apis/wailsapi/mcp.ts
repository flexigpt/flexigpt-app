import {
	DefaultMCPPageSize,
	type InvokeMCPToolRequestBody,
	type InvokeMCPToolResponseBody,
	MaxMCPServerPageSize,
	type MCPApprovalEvaluation,
	type MCPApprovalResolution,
	type MCPApprovalToken,
	type MCPAuthHealth,
	type MCPAuthStatus,
	type MCPCompletionResult,
	type MCPGetPromptResponseBody,
	type MCPOAuthAuthorization,
	type MCPPromptRef,
	type MCPReadResourceResponseBody,
	type MCPRefType,
	type MCPResourceRef,
	type MCPResourceTemplateRef,
	type MCPSecretKind,
	type MCPServerConfig,
	type MCPServerID,
	type MCPServerRuntimeSnapshot,
	type MCPToolCapability,
	type PatchMCPServerPolicyPayload,
	type PutMCPServerPayload,
	type PutMCPServerSecretResponseBody,
} from '@/spec/mcp';

import type { IMCPAPI } from '@/apis/interface';
import {
	CancelPendingMCPOAuthAuthorization,
	CompleteMCPArgument,
	ConnectMCPServer,
	DeleteMCPServer,
	DeleteMCPServerSecret,
	DisconnectMCPServer,
	EvaluateMCPToolCall,
	GetMCPPrompt,
	GetMCPServer,
	GetMCPServerAuthHealth,
	GetMCPServerAuthStatus,
	GetMCPServerStatus,
	InvokeMCPTool,
	ListMCPServerPrompts,
	ListMCPServerResources,
	ListMCPServerResourceTemplates,
	ListMCPServers,
	ListMCPServerTools,
	ListPendingMCPOAuthAuthorizations,
	PatchMCPServerEnabled,
	PatchMCPServerPolicy,
	PutMCPServer,
	PutMCPServerSecret,
	ReadMCPResource,
	RefreshMCPServer,
	ResolveMCPApproval,
} from '@/apis/wailsjs/go/main/MCPWrapper';
import { type spec as wailsSpec } from '@/apis/wailsjs/go/models';

function normalizeMCPPageSize(pageSize?: number): number {
	if (typeof pageSize !== 'number' || !Number.isFinite(pageSize) || pageSize <= 0) {
		return DefaultMCPPageSize;
	}
	return Math.min(pageSize, MaxMCPServerPageSize);
}

/**
 * @public
 *
 * Wails bridge for the frontend-facing MCP API.
 */
export class WailsMCPAPI implements IMCPAPI {
	async listMCPServers(
		serverIDs?: MCPServerID[],
		enabled?: boolean,
		pageSize?: number,
		pageToken?: string
	): Promise<{ servers: MCPServerConfig[]; nextPageToken?: string }> {
		const resp = await ListMCPServers({
			ServerIDs: serverIDs,
			Enabled: enabled,
			PageSize: normalizeMCPPageSize(pageSize),
			PageToken: pageToken,
		} as wailsSpec.ListMCPServersRequest);

		return {
			servers: (resp.Body?.servers ?? []) as MCPServerConfig[],
			nextPageToken: resp.Body?.nextPageToken ?? undefined,
		};
	}

	async putMCPServer(serverID: MCPServerID, payload: PutMCPServerPayload): Promise<void> {
		const req = {
			ServerID: serverID,
			Body: payload as unknown as wailsSpec.PutMCPServerPayload,
		} as wailsSpec.PutMCPServerRequest;

		await PutMCPServer(req);
	}

	async getMCPServer(serverID: MCPServerID, includeDeleted?: boolean): Promise<MCPServerConfig | undefined> {
		const resp = await GetMCPServer({
			ServerID: serverID,
			IncludeDeleted: !!includeDeleted,
		} as wailsSpec.GetMCPServerRequest);

		return resp?.Body as MCPServerConfig | undefined;
	}

	async patchMCPServerEnabled(serverID: MCPServerID, enabled: boolean): Promise<void> {
		const req = {
			ServerID: serverID,
			Body: {
				enabled,
			},
		} as wailsSpec.PatchMCPServerEnabledRequest;

		await PatchMCPServerEnabled(req);
	}

	async patchMCPServerPolicy(serverID: MCPServerID, payload: PatchMCPServerPolicyPayload): Promise<void> {
		const req = {
			ServerID: serverID,
			Body: payload as unknown as wailsSpec.PatchMCPServerPolicyPayload,
		} as wailsSpec.PatchMCPServerPolicyRequest;

		await PatchMCPServerPolicy(req);
	}

	async deleteMCPServer(serverID: MCPServerID): Promise<void> {
		await DeleteMCPServer({
			ServerID: serverID,
		} as wailsSpec.DeleteMCPServerRequest);
	}

	async connectMCPServer(serverID: MCPServerID): Promise<MCPServerRuntimeSnapshot | undefined> {
		const resp = await ConnectMCPServer({
			ServerID: serverID,
		} as wailsSpec.ConnectMCPServerRequest);

		return resp?.Body as MCPServerRuntimeSnapshot | undefined;
	}

	async disconnectMCPServer(serverID: MCPServerID): Promise<void> {
		await DisconnectMCPServer({
			ServerID: serverID,
		} as wailsSpec.DisconnectMCPServerRequest);
	}

	async refreshMCPServer(serverID: MCPServerID): Promise<MCPServerRuntimeSnapshot | undefined> {
		const resp = await RefreshMCPServer({
			ServerID: serverID,
		} as wailsSpec.RefreshMCPServerRequest);

		return resp?.Body as MCPServerRuntimeSnapshot | undefined;
	}

	async getMCPServerStatus(serverID: MCPServerID): Promise<MCPServerRuntimeSnapshot | undefined> {
		const resp = await GetMCPServerStatus({
			ServerID: serverID,
		} as wailsSpec.GetMCPServerStatusRequest);

		return resp?.Body as MCPServerRuntimeSnapshot | undefined;
	}

	async listMCPServerTools(
		serverID: MCPServerID,
		pageSize?: number,
		pageToken?: string
	): Promise<{ tools: MCPToolCapability[]; nextPageToken?: string }> {
		const resp = await ListMCPServerTools({
			ServerID: serverID,
			PageSize: normalizeMCPPageSize(pageSize),
			PageToken: pageToken,
		} as wailsSpec.ListMCPServerToolsRequest);

		return {
			tools: (resp.Body?.tools ?? []) as MCPToolCapability[],
			nextPageToken: resp.Body?.nextPageToken ?? undefined,
		};
	}

	async listMCPServerResources(
		serverID: MCPServerID,
		pageSize?: number,
		pageToken?: string
	): Promise<{ resources: MCPResourceRef[]; nextPageToken?: string }> {
		const resp = await ListMCPServerResources({
			ServerID: serverID,
			PageSize: normalizeMCPPageSize(pageSize),
			PageToken: pageToken,
		} as wailsSpec.ListMCPServerResourcesRequest);

		return {
			resources: (resp.Body?.resources ?? []) as MCPResourceRef[],
			nextPageToken: resp.Body?.nextPageToken ?? undefined,
		};
	}

	async listMCPServerResourceTemplates(
		serverID: MCPServerID,
		pageSize?: number,
		pageToken?: string
	): Promise<{ resourceTemplates: MCPResourceTemplateRef[]; nextPageToken?: string }> {
		const resp = await ListMCPServerResourceTemplates({
			ServerID: serverID,
			PageSize: normalizeMCPPageSize(pageSize),
			PageToken: pageToken,
		} as wailsSpec.ListMCPServerResourceTemplatesRequest);

		return {
			resourceTemplates: (resp.Body?.resourceTemplates ?? []) as MCPResourceTemplateRef[],
			nextPageToken: resp.Body?.nextPageToken ?? undefined,
		};
	}

	async listMCPServerPrompts(
		serverID: MCPServerID,
		pageSize?: number,
		pageToken?: string
	): Promise<{ prompts: MCPPromptRef[]; nextPageToken?: string }> {
		const resp = await ListMCPServerPrompts({
			ServerID: serverID,
			PageSize: normalizeMCPPageSize(pageSize),
			PageToken: pageToken,
		} as wailsSpec.ListMCPServerPromptsRequest);

		return {
			prompts: (resp.Body?.prompts ?? []) as MCPPromptRef[],
			nextPageToken: resp.Body?.nextPageToken ?? undefined,
		};
	}

	async readMCPResource(serverID: MCPServerID, uri: string): Promise<MCPReadResourceResponseBody | undefined> {
		const resp = await ReadMCPResource({
			Body: {
				serverID,
				uri,
			} as wailsSpec.MCPReadResourceRequestBody,
		} as wailsSpec.MCPReadResourceRequest);

		return resp?.Body as MCPReadResourceResponseBody | undefined;
	}

	async getMCPPrompt(
		serverID: MCPServerID,
		promptName: string,
		promptArguments?: Record<string, string>
	): Promise<MCPGetPromptResponseBody | undefined> {
		const resp = await GetMCPPrompt({
			Body: {
				serverID,
				promptName,
				arguments: promptArguments,
			} as wailsSpec.MCPGetPromptRequestBody,
		} as wailsSpec.MCPGetPromptRequest);

		return resp?.Body as MCPGetPromptResponseBody | undefined;
	}

	async completeMCPArgument(
		serverID: MCPServerID,
		refType: MCPRefType,
		name: string,
		argumentName: string,
		argumentValue?: string,
		context?: Record<string, string>
	): Promise<MCPCompletionResult> {
		const resp = await CompleteMCPArgument({
			Body: {
				serverID,
				refType,
				name,
				argumentName,
				argumentValue,
				context,
			} as wailsSpec.MCPCompleteArgumentRequestBody,
		} as wailsSpec.MCPCompleteArgumentRequest);

		return resp as MCPCompletionResult;
	}

	async evaluateMCPToolCall(request: InvokeMCPToolRequestBody): Promise<MCPApprovalEvaluation | undefined> {
		const resp = await EvaluateMCPToolCall({
			Body: request as unknown as wailsSpec.InvokeMCPToolRequestBody,
		} as wailsSpec.EvaluateMCPToolCallRequest);

		return resp?.Body as MCPApprovalEvaluation | undefined;
	}

	async invokeMCPTool(request: InvokeMCPToolRequestBody): Promise<InvokeMCPToolResponseBody | undefined> {
		const resp = await InvokeMCPTool({
			Body: request as unknown as wailsSpec.InvokeMCPToolRequestBody,
		} as wailsSpec.InvokeMCPToolRequest);

		return resp?.Body as InvokeMCPToolResponseBody | undefined;
	}

	async resolveMCPApproval(
		approvalID: string,
		resolution: MCPApprovalResolution
	): Promise<MCPApprovalToken | undefined> {
		const resp = await ResolveMCPApproval({
			Body: {
				approvalID,
				resolution,
			} as wailsSpec.ResolveMCPApprovalRequestBody,
		} as wailsSpec.ResolveMCPApprovalRequest);

		return resp?.Body as MCPApprovalToken | undefined;
	}

	async listPendingMCPOAuthAuthorizations(): Promise<MCPOAuthAuthorization[]> {
		const resp = await ListPendingMCPOAuthAuthorizations({} as wailsSpec.ListPendingMCPOAuthAuthorizationsRequest);

		return (resp?.Body?.authorizations ?? []) as MCPOAuthAuthorization[];
	}

	async cancelPendingMCPOAuthAuthorization(serverID: MCPServerID): Promise<void> {
		await CancelPendingMCPOAuthAuthorization({
			ServerID: serverID,
		} as wailsSpec.CancelPendingMCPOAuthAuthorizationRequest);
	}

	async getMCPServerAuthStatus(serverID: MCPServerID): Promise<MCPAuthStatus | undefined> {
		const resp = await GetMCPServerAuthStatus({
			ServerID: serverID,
		} as wailsSpec.GetMCPServerAuthStatusRequest);

		return resp?.Body as MCPAuthStatus | undefined;
	}

	async getMCPServerAuthHealth(serverID: MCPServerID): Promise<MCPAuthHealth | undefined> {
		const resp = await GetMCPServerAuthHealth({
			ServerID: serverID,
		} as wailsSpec.GetMCPServerAuthHealthRequest);

		return resp?.Body as MCPAuthHealth | undefined;
	}

	async putMCPServerSecret(
		serverID: MCPServerID,
		kind: MCPSecretKind,
		slot: string,
		secret: string
	): Promise<PutMCPServerSecretResponseBody | undefined> {
		const resp = await PutMCPServerSecret({
			ServerID: serverID,
			Body: {
				kind,
				slot,
				secret,
			} as wailsSpec.PutMCPServerSecretRequestBody,
		} as wailsSpec.PutMCPServerSecretRequest);

		return resp?.Body as PutMCPServerSecretResponseBody | undefined;
	}

	async deleteMCPServerSecret(serverID: MCPServerID, kind: MCPSecretKind, slot: string): Promise<void> {
		await DeleteMCPServerSecret({
			ServerID: serverID,
			Kind: kind,
			Slot: slot,
		} as wailsSpec.DeleteMCPServerSecretRequest);
	}
}
