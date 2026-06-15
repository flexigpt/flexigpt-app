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
	type MCPBundle,
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
	type MCPServerSetupInputValue,
	type MCPSettingsView,
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
	DeleteMCPBundle,
	DeleteMCPServer,
	DeleteMCPServerSecret,
	DisconnectMCPServer,
	EvaluateMCPToolCall,
	GetMCPPrompt,
	GetMCPServer,
	GetMCPServerAuthHealth,
	GetMCPServerAuthStatus,
	GetMCPServerStatus,
	GetMCPSettings,
	InvokeMCPTool,
	ListMCPBundles,
	ListMCPServerPrompts,
	ListMCPServerResources,
	ListMCPServerResourceTemplates,
	ListMCPServers,
	ListMCPServerTools,
	ListPendingMCPOAuthAuthorizations,
	PatchMCPBundle,
	PatchMCPServerEnabled,
	PatchMCPServerPolicy,
	PatchMCPServerSetup,
	PatchMCPSettings,
	PutMCPBundle,
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
	async listMCPBundles(
		bundleIDs?: string[],
		includeDisabled?: boolean,
		pageSize?: number,
		pageToken?: string
	): Promise<{ bundles: MCPBundle[]; nextPageToken?: string }> {
		const resp = await ListMCPBundles({
			BundleIDs: bundleIDs ?? [],
			IncludeDisabled: includeDisabled ?? false,
			PageSize: normalizeMCPPageSize(pageSize),
			PageToken: pageToken,
		} as wailsSpec.ListMCPBundlesRequest);

		return {
			bundles: (resp.Body?.bundles ?? []) as MCPBundle[],
			nextPageToken: resp.Body?.nextPageToken ?? undefined,
		};
	}

	async putMCPBundle(
		bundleID: string,
		slug: string,
		displayName: string,
		isEnabled: boolean,
		description?: string
	): Promise<void> {
		const req = {
			BundleID: bundleID,
			Body: {
				slug,
				displayName,
				isEnabled,
				description,
			} as wailsSpec.PutMCPBundleRequestBody,
		} as wailsSpec.PutMCPBundleRequest;

		await PutMCPBundle(req);
	}

	async patchMCPBundle(bundleID: string, isEnabled: boolean): Promise<void> {
		const req = {
			BundleID: bundleID,
			Body: {
				isEnabled,
			} as wailsSpec.PatchMCPBundleRequestBody,
		} as wailsSpec.PatchMCPBundleRequest;

		await PatchMCPBundle(req);
	}

	async deleteMCPBundle(bundleID: string): Promise<void> {
		await DeleteMCPBundle({
			BundleID: bundleID,
		} as wailsSpec.DeleteMCPBundleRequest);
	}

	async listMCPServers(
		bundleID: string,
		serverIDs?: MCPServerID[],
		enabled?: boolean,
		includeDisabled?: boolean,
		pageSize?: number,
		pageToken?: string
	): Promise<{ servers: MCPServerConfig[]; nextPageToken?: string }> {
		const resp = await ListMCPServers({
			BundleID: bundleID,
			ServerIDs: serverIDs ?? [],
			Enabled: enabled,
			IncludeDisabled: includeDisabled ?? false,
			PageSize: normalizeMCPPageSize(pageSize),
			PageToken: pageToken,
		} as wailsSpec.ListMCPServersRequest);

		return {
			servers: (resp.Body?.servers ?? []) as MCPServerConfig[],
			nextPageToken: resp.Body?.nextPageToken ?? undefined,
		};
	}

	async putMCPServer(bundleID: string, serverID: MCPServerID, payload: PutMCPServerPayload): Promise<void> {
		const req = {
			BundleID: bundleID,
			ServerID: serverID,
			Body: payload as unknown as wailsSpec.PutMCPServerPayload,
		} as wailsSpec.PutMCPServerRequest;

		await PutMCPServer(req);
	}

	async getMCPServer(bundleID: string, serverID: MCPServerID): Promise<MCPServerConfig | undefined> {
		const resp = await GetMCPServer({
			BundleID: bundleID,
			ServerID: serverID,
		} as wailsSpec.GetMCPServerRequest);

		return resp?.Body as MCPServerConfig | undefined;
	}

	async patchMCPServerEnabled(bundleID: string, serverID: MCPServerID, enabled: boolean): Promise<void> {
		const req = {
			BundleID: bundleID,
			ServerID: serverID,
			Body: {
				enabled,
			} as wailsSpec.PatchMCPServerEnabledRequestBody,
		} as wailsSpec.PatchMCPServerEnabledRequest;

		await PatchMCPServerEnabled(req);
	}

	async patchMCPServerPolicy(
		bundleID: string,
		serverID: MCPServerID,
		payload: PatchMCPServerPolicyPayload
	): Promise<void> {
		const req = {
			BundleID: bundleID,
			ServerID: serverID,
			Body: payload as unknown as wailsSpec.PatchMCPServerPolicyPayload,
		} as wailsSpec.PatchMCPServerPolicyRequest;

		await PatchMCPServerPolicy(req);
	}
	async patchMCPServerSetup(
		bundleID: string,
		serverID: MCPServerID,
		inputValues: Record<string, MCPServerSetupInputValue>,
		reset?: boolean
	): Promise<MCPServerConfig | undefined> {
		const resp = await PatchMCPServerSetup({
			BundleID: bundleID,
			ServerID: serverID,
			Body: {
				reset: reset ?? false,
				inputValues,
			} as wailsSpec.PatchMCPServerSetupRequestBody,
		} as wailsSpec.PatchMCPServerSetupRequest);

		return resp?.Body as MCPServerConfig | undefined;
	}

	async patchMCPSettings(oauthLoopbackListenAddr?: string): Promise<MCPSettingsView | undefined> {
		const resp = await PatchMCPSettings({
			Body: {
				oauthLoopbackListenAddr,
			} as wailsSpec.PatchMCPSettingsRequestBody,
		} as wailsSpec.PatchMCPSettingsRequest);

		return resp?.Body as MCPSettingsView | undefined;
	}

	async getMCPSettings(): Promise<MCPSettingsView | undefined> {
		const resp = await GetMCPSettings({} as wailsSpec.GetMCPSettingsRequest);
		return resp?.Body as MCPSettingsView | undefined;
	}

	async deleteMCPServer(bundleID: string, serverID: MCPServerID): Promise<void> {
		await DeleteMCPServer({
			BundleID: bundleID,
			ServerID: serverID,
		} as wailsSpec.DeleteMCPServerRequest);
	}

	async connectMCPServer(bundleID: string, serverID: MCPServerID): Promise<MCPServerRuntimeSnapshot | undefined> {
		const resp = await ConnectMCPServer({
			BundleID: bundleID,
			ServerID: serverID,
		} as wailsSpec.ConnectMCPServerRequest);

		return resp?.Body as MCPServerRuntimeSnapshot | undefined;
	}

	async disconnectMCPServer(bundleID: string, serverID: MCPServerID): Promise<void> {
		await DisconnectMCPServer({
			BundleID: bundleID,
			ServerID: serverID,
		} as wailsSpec.DisconnectMCPServerRequest);
	}

	async refreshMCPServer(bundleID: string, serverID: MCPServerID): Promise<MCPServerRuntimeSnapshot | undefined> {
		const resp = await RefreshMCPServer({
			BundleID: bundleID,
			ServerID: serverID,
		} as wailsSpec.RefreshMCPServerRequest);

		return resp?.Body as MCPServerRuntimeSnapshot | undefined;
	}

	async getMCPServerStatus(bundleID: string, serverID: MCPServerID): Promise<MCPServerRuntimeSnapshot | undefined> {
		const resp = await GetMCPServerStatus({
			BundleID: bundleID,
			ServerID: serverID,
		} as wailsSpec.GetMCPServerStatusRequest);

		return resp?.Body as MCPServerRuntimeSnapshot | undefined;
	}

	async listMCPServerTools(
		bundleID: string,
		serverID: MCPServerID,
		pageSize?: number,
		pageToken?: string
	): Promise<{ tools: MCPToolCapability[]; nextPageToken?: string }> {
		const resp = await ListMCPServerTools({
			BundleID: bundleID,
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
		bundleID: string,
		serverID: MCPServerID,
		pageSize?: number,
		pageToken?: string
	): Promise<{ resources: MCPResourceRef[]; nextPageToken?: string }> {
		const resp = await ListMCPServerResources({
			BundleID: bundleID,
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
		bundleID: string,
		serverID: MCPServerID,
		pageSize?: number,
		pageToken?: string
	): Promise<{ resourceTemplates: MCPResourceTemplateRef[]; nextPageToken?: string }> {
		const resp = await ListMCPServerResourceTemplates({
			BundleID: bundleID,
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
		bundleID: string,
		serverID: MCPServerID,
		pageSize?: number,
		pageToken?: string
	): Promise<{ prompts: MCPPromptRef[]; nextPageToken?: string }> {
		const resp = await ListMCPServerPrompts({
			BundleID: bundleID,
			ServerID: serverID,
			PageSize: normalizeMCPPageSize(pageSize),
			PageToken: pageToken,
		} as wailsSpec.ListMCPServerPromptsRequest);

		return {
			prompts: (resp.Body?.prompts ?? []) as MCPPromptRef[],
			nextPageToken: resp.Body?.nextPageToken ?? undefined,
		};
	}

	async readMCPResource(
		bundleID: string,
		serverID: MCPServerID,
		uri: string
	): Promise<MCPReadResourceResponseBody | undefined> {
		const resp = await ReadMCPResource({
			BundleID: bundleID,
			ServerID: serverID,
			Body: {
				serverID,
				uri,
			} as wailsSpec.MCPReadResourceRequestBody,
		} as wailsSpec.MCPReadResourceRequest);

		return resp?.Body as MCPReadResourceResponseBody | undefined;
	}

	async getMCPPrompt(
		bundleID: string,
		serverID: MCPServerID,
		promptName: string,
		promptArguments?: Record<string, string>
	): Promise<MCPGetPromptResponseBody | undefined> {
		const resp = await GetMCPPrompt({
			BundleID: bundleID,
			ServerID: serverID,
			Body: {
				serverID,
				promptName,
				arguments: promptArguments,
			} as wailsSpec.MCPGetPromptRequestBody,
		} as wailsSpec.MCPGetPromptRequest);

		return resp?.Body as MCPGetPromptResponseBody | undefined;
	}

	async completeMCPArgument(
		bundleID: string,
		serverID: MCPServerID,
		refType: MCPRefType,
		name: string,
		argumentName: string,
		argumentValue?: string,
		context?: Record<string, string>
	): Promise<MCPCompletionResult> {
		const resp = await CompleteMCPArgument({
			BundleID: bundleID,
			ServerID: serverID,
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

	async evaluateMCPToolCall(
		bundleID: string,
		request: InvokeMCPToolRequestBody
	): Promise<MCPApprovalEvaluation | undefined> {
		const resp = await EvaluateMCPToolCall({
			BundleID: bundleID,
			ServerID: request.serverID,
			Body: request as unknown as wailsSpec.InvokeMCPToolRequestBody,
		} as wailsSpec.EvaluateMCPToolCallRequest);

		return resp?.Body as MCPApprovalEvaluation | undefined;
	}

	async invokeMCPTool(
		bundleID: string,
		request: InvokeMCPToolRequestBody
	): Promise<InvokeMCPToolResponseBody | undefined> {
		const resp = await InvokeMCPTool({
			BundleID: bundleID,
			ServerID: request.serverID,
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

	async cancelPendingMCPOAuthAuthorization(bundleID: string, serverID: MCPServerID): Promise<void> {
		await CancelPendingMCPOAuthAuthorization({
			BundleID: bundleID,
			ServerID: serverID,
		} as wailsSpec.CancelPendingMCPOAuthAuthorizationRequest);
	}

	async getMCPServerAuthStatus(bundleID: string, serverID: MCPServerID): Promise<MCPAuthStatus | undefined> {
		const resp = await GetMCPServerAuthStatus({
			BundleID: bundleID,
			ServerID: serverID,
		} as wailsSpec.GetMCPServerAuthStatusRequest);

		return resp?.Body as MCPAuthStatus | undefined;
	}

	async getMCPServerAuthHealth(bundleID: string, serverID: MCPServerID): Promise<MCPAuthHealth | undefined> {
		const resp = await GetMCPServerAuthHealth({
			BundleID: bundleID,
			ServerID: serverID,
		} as wailsSpec.GetMCPServerAuthHealthRequest);

		return resp?.Body as MCPAuthHealth | undefined;
	}

	async putMCPServerSecret(
		bundleID: string,
		serverID: MCPServerID,
		kind: MCPSecretKind,
		slot: string,
		secret: string
	): Promise<PutMCPServerSecretResponseBody | undefined> {
		const resp = await PutMCPServerSecret({
			BundleID: bundleID,
			ServerID: serverID,
			Body: {
				kind,
				slot,
				secret,
			} as wailsSpec.PutMCPServerSecretRequestBody,
		} as wailsSpec.PutMCPServerSecretRequest);

		return resp?.Body as PutMCPServerSecretResponseBody | undefined;
	}

	async deleteMCPServerSecret(
		bundleID: string,
		serverID: MCPServerID,
		kind: MCPSecretKind,
		slot: string
	): Promise<void> {
		await DeleteMCPServerSecret({
			BundleID: bundleID,
			ServerID: serverID,
			Kind: kind,
			Slot: slot,
		} as wailsSpec.DeleteMCPServerSecretRequest);
	}
}
