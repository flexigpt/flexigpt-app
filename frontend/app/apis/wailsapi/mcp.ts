import type {
	InvokeMCPToolRequestBody,
	InvokeMCPToolResponseBody,
	MCPApprovalEvaluation,
	MCPApprovalResolution,
	MCPApprovalToken,
	MCPAuthHealth,
	MCPAuthStatus,
	MCPBundle,
	MCPCompletionResult,
	MCPGetPromptResponseBody,
	MCPOAuthAuthorization,
	MCPPromptRef,
	MCPReadResourceResponseBody,
	MCPRefType,
	MCPResourceRef,
	MCPResourceTemplateRef,
	MCPSecretKind,
	MCPServerConfig,
	MCPServerID,
	MCPServerRuntimeSnapshot,
	MCPServerSetupInputValue,
	MCPSettingsView,
	MCPToolCapability,
	PatchMCPServerPolicyPayload,
	PutMCPServerPayload,
	PutMCPServerSecretResponseBody,
} from '@/spec/mcp';
import {
	DefaultMCPPageSize,
	MaxMCPServerPageSize,
	MCPAuthHealthState,
	MCPAuthState,
	MCPHTTPAuthMode,
	MCPServerStatus,
	MCPTransportType,
	MCPTrustLevel,
} from '@/spec/mcp';

import type { IMCPAPI } from '@/apis/interface';
import {
	enumFromWails,
	normalizePageSize,
	optionalWailsBody,
	requireWailsBody,
	toFrontendTimestamp,
} from '@/apis/wailsapi/transport';
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
import type { spec as wailsSpec } from '@/apis/wailsjs/go/models';

function mcpServerFromWails(server: MCPServerConfig, field: string): MCPServerConfig {
	return {
		...server,
		transport: enumFromWails(server.transport, MCPTransportType, `${field}.transport`),
		trustLevel: enumFromWails(server.trustLevel, MCPTrustLevel, `${field}.trustLevel`),
		createdAt: toFrontendTimestamp(server.createdAt, `${field}.createdAt`),
		modifiedAt: toFrontendTimestamp(server.modifiedAt, `${field}.modifiedAt`),
		streamableHttp:
			server.streamableHttp === undefined
				? undefined
				: {
						...server.streamableHttp,
						authMode: enumFromWails(
							server.streamableHttp.authMode,
							MCPHTTPAuthMode,
							`${field}.streamableHttp.authMode`
						),
					},
	};
}

function mcpRuntimeSnapshotFromWails(snapshot: MCPServerRuntimeSnapshot, field: string): MCPServerRuntimeSnapshot {
	return {
		...snapshot,
		status: enumFromWails(snapshot.status, MCPServerStatus, `${field}.status`),
	};
}

function mcpAuthStatusFromWails(status: MCPAuthStatus, field: string): MCPAuthStatus {
	return {
		...status,
		authMode: enumFromWails(status.authMode, MCPHTTPAuthMode, `${field}.authMode`),
		state: enumFromWails(status.state, MCPAuthState, `${field}.state`),
		expiresAt: status.expiresAt ? toFrontendTimestamp(status.expiresAt, `${field}.expiresAt`) : undefined,
	};
}

function mcpAuthHealthFromWails(health: MCPAuthHealth, field: string): MCPAuthHealth {
	return {
		...health,
		authMode: enumFromWails(health.authMode, MCPHTTPAuthMode, `${field}.authMode`),
		state: enumFromWails(health.state, MCPAuthHealthState, `${field}.state`),
		expiresAt: health.expiresAt ? toFrontendTimestamp(health.expiresAt, `${field}.expiresAt`) : undefined,
	};
}

/**
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
			PageSize: normalizePageSize(pageSize, DefaultMCPPageSize, MaxMCPServerPageSize),
			PageToken: pageToken ?? '',
		} as wailsSpec.ListMCPBundlesRequest);

		const body = requireWailsBody(resp.Body, 'ListMCPBundles');
		return {
			bundles: (body.bundles ?? []) as MCPBundle[],
			nextPageToken: body.nextPageToken || undefined,
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
			PageSize: normalizePageSize(pageSize, DefaultMCPPageSize, MaxMCPServerPageSize),
			PageToken: pageToken ?? '',
		} as wailsSpec.ListMCPServersRequest);

		const body = requireWailsBody(resp.Body, 'ListMCPServers');
		return {
			servers: (body.servers ?? []).map(server =>
				mcpServerFromWails(server as MCPServerConfig, 'ListMCPServers.servers')
			),
			nextPageToken: body.nextPageToken || undefined,
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

		const body = optionalWailsBody(resp.Body);
		return body === undefined ? undefined : mcpServerFromWails(body as MCPServerConfig, 'GetMCPServer');
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

		const body = optionalWailsBody(resp.Body);
		return body === undefined ? undefined : mcpServerFromWails(body as MCPServerConfig, 'PatchMCPServerSetup');
	}

	async patchMCPSettings(oauthLoopbackListenAddr?: string): Promise<MCPSettingsView | undefined> {
		const resp = await PatchMCPSettings({
			Body: {
				oauthLoopbackListenAddr,
			} as wailsSpec.PatchMCPSettingsRequestBody,
		} as wailsSpec.PatchMCPSettingsRequest);

		return optionalWailsBody(resp.Body) as MCPSettingsView | undefined;
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

		const body = optionalWailsBody(resp.Body);
		return body === undefined
			? undefined
			: mcpRuntimeSnapshotFromWails(body as MCPServerRuntimeSnapshot, 'ConnectMCPServer');
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

		const body = optionalWailsBody(resp.Body);
		return body === undefined
			? undefined
			: mcpRuntimeSnapshotFromWails(body as MCPServerRuntimeSnapshot, 'RefreshMCPServer');
	}

	async getMCPServerStatus(bundleID: string, serverID: MCPServerID): Promise<MCPServerRuntimeSnapshot | undefined> {
		const resp = await GetMCPServerStatus({
			BundleID: bundleID,
			ServerID: serverID,
		} as wailsSpec.GetMCPServerStatusRequest);

		const body = optionalWailsBody(resp.Body);
		return body === undefined
			? undefined
			: mcpRuntimeSnapshotFromWails(body as MCPServerRuntimeSnapshot, 'GetMCPServerStatus');
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
			PageSize: normalizePageSize(pageSize, DefaultMCPPageSize, MaxMCPServerPageSize),
			PageToken: pageToken ?? '',
		} as wailsSpec.ListMCPServerToolsRequest);

		const body = requireWailsBody(resp.Body, 'ListMCPServerTools');
		return {
			tools: (body.tools ?? []) as MCPToolCapability[],
			nextPageToken: body.nextPageToken || undefined,
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
			PageSize: normalizePageSize(pageSize, DefaultMCPPageSize, MaxMCPServerPageSize),
			PageToken: pageToken ?? '',
		} as wailsSpec.ListMCPServerResourcesRequest);

		const body = requireWailsBody(resp.Body, 'ListMCPServerResources');
		return {
			resources: (body.resources ?? []) as MCPResourceRef[],
			nextPageToken: body.nextPageToken || undefined,
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
			PageSize: normalizePageSize(pageSize, DefaultMCPPageSize, MaxMCPServerPageSize),
			PageToken: pageToken ?? '',
		} as wailsSpec.ListMCPServerResourceTemplatesRequest);

		const body = requireWailsBody(resp.Body, 'ListMCPServerResourceTemplates');
		return {
			resourceTemplates: (body.resourceTemplates ?? []) as MCPResourceTemplateRef[],
			nextPageToken: body.nextPageToken || undefined,
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
			PageSize: normalizePageSize(pageSize, DefaultMCPPageSize, MaxMCPServerPageSize),
			PageToken: pageToken ?? '',
		} as wailsSpec.ListMCPServerPromptsRequest);

		const body = requireWailsBody(resp.Body, 'ListMCPServerPrompts');
		return {
			prompts: (body.prompts ?? []) as MCPPromptRef[],
			nextPageToken: body.nextPageToken || undefined,
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
				uri,
			} as wailsSpec.MCPReadResourceRequestBody,
		} as wailsSpec.MCPReadResourceRequest);

		return optionalWailsBody(resp.Body) as MCPReadResourceResponseBody | undefined;
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
				promptName,
				arguments: promptArguments,
			} as wailsSpec.MCPGetPromptRequestBody,
		} as wailsSpec.MCPGetPromptRequest);

		return optionalWailsBody(resp.Body) as MCPGetPromptResponseBody | undefined;
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
			Body: {
				source: request.source,
				toolName: request.toolName,
				providerToolName: request.providerToolName,
				toolDigest: request.toolDigest,
				arguments: request.arguments,
				approvalID: request.approvalID,
				approvalToken: request.approvalToken,
				conversationID: request.conversationID,
				messageID: request.messageID,
				toolUseID: request.toolUseID,
				appInstanceID: request.appInstanceID,
			} as wailsSpec.InvokeMCPToolRequestBody,
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
			Body: {
				source: request.source,
				toolName: request.toolName,
				providerToolName: request.providerToolName,
				toolDigest: request.toolDigest,
				arguments: request.arguments,
				approvalID: request.approvalID,
				approvalToken: request.approvalToken,
				conversationID: request.conversationID,
				messageID: request.messageID,
				toolUseID: request.toolUseID,
				appInstanceID: request.appInstanceID,
			} as wailsSpec.InvokeMCPToolRequestBody,
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

		const body = optionalWailsBody(resp.Body);
		return body === undefined ? undefined : mcpAuthStatusFromWails(body as MCPAuthStatus, 'GetMCPServerAuthStatus');
	}

	async getMCPServerAuthHealth(bundleID: string, serverID: MCPServerID): Promise<MCPAuthHealth | undefined> {
		const resp = await GetMCPServerAuthHealth({
			BundleID: bundleID,
			ServerID: serverID,
		} as wailsSpec.GetMCPServerAuthHealthRequest);

		const body = optionalWailsBody(resp.Body);
		return body === undefined ? undefined : mcpAuthHealthFromWails(body as MCPAuthHealth, 'GetMCPServerAuthHealth');
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
