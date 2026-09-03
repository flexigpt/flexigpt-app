import type { ArtifactCollection, ArtifactCollectionRef, ArtifactRecord, ArtifactRef } from '@/spec/artifact';
import type {
	InvokeMCPToolRequestBody,
	InvokeMCPToolResponseBody,
	MCPApprovalEvaluation,
	MCPApprovalResolution,
	MCPApprovalResolutionResult,
	MCPArtifactRegistration,
	MCPAuthHealth,
	MCPBundle,
	MCPBundleDocument,
	MCPBundleInstallation,
	MCPCompletionRefType,
	MCPCompletionResult,
	MCPCreateBundleInput,
	MCPGetPromptResponseBody,
	MCPGlobalSettings,
	MCPOAuthAuthorization,
	MCPPolicyView,
	MCPPromptRef,
	MCPProviderToolMapping,
	MCPReadResourceResponseBody,
	MCPReplaceBundleDocumentInput,
	MCPResourceRef,
	MCPResourceTemplateRef,
	MCPRuntimeServerID,
	MCPSecretKind,
	MCPSecretWriteResult,
	MCPServerData,
	MCPServerInstallation,
	MCPServerResolved,
	MCPServerRuntimeSnapshot,
	MCPToolCapability,
} from '@/spec/mcp_artifact';

import type { IMCPAPI } from '@/apis/interface';
import {
	rawJSONFromWails,
	rawJSONObjectToWails,
	requireNonBlankString,
	requireWailsBody,
	requireWailsBoolean,
	requireWailsFiniteNumber,
	wailsObjectArrayOrEmpty,
} from '@/apis/wailsapi/transport';
import {
	ArtifactRefForRuntimeServerID,
	DeleteMCPServerSecret,
	GetMCPServerAuthHealth,
	PurgeMCPBundle,
	PutMCPServerSecret,
	RefreshMCPBundle,
	ReplaceMCPBundleDocument,
	RetireMCPBundle,
	RuntimeServerIDForArtifact,
	UpdateMCPBundleEnabled,
	UpdateMCPServerInstallation,
	UpdateProtectedMCPBundleInstallation,
	UpdateProtectedMCPServerInstallation,
} from '@/apis/wailsjs/go/main/MCPAggregateWrapper';
import {
	CancelPendingMCPOAuthAuthorization,
	CompleteMCPArgument,
	DisconnectMCPServer,
	EvaluateMappedMCPToolCall,
	EvaluateMCPToolCall,
	GetMCPGlobalSettings,
	GetMCPPrompt,
	GetMCPServerStatus,
	InvokeMappedMCPTool,
	InvokeMCPTool,
	ListMCPServerPrompts,
	ListMCPServerResources,
	ListMCPServerResourceTemplates,
	ListMCPServerTools,
	ListPendingMCPOAuthAuthorizations,
	ReadMCPResource,
	RefreshMCPServer,
	ResolveMCPApproval,
	StartMCPServerConnect,
	UpdateMCPGlobalSettings,
} from '@/apis/wailsjs/go/main/MCPRuntimeWrapper';
import {
	CreateMCPBundle,
	GetMCPBundle,
	GetMCPBundleDocument,
	GetMCPBundleInstallation,
	GetMCPServerInstallation,
	InspectMCPPolicy,
	InspectMCPServer,
	ListMCPBundlePolicies,
	ListMCPBundles,
	ListMCPBundleServers,
} from '@/apis/wailsjs/go/main/MCPStoreWrapper';

function registrationToWails(value: MCPArtifactRegistration, field: string): unknown {
	return {
		ArtifactID: value.artifactID,
		Subresource: value.subresource,
		Kind: value.kind,
		Enabled: value.enabled,
		...(value.data === undefined ? {} : { Data: rawJSONObjectToWails(value.data, `${field}.data`) }),
	};
}

function documentToWails(value: MCPBundleDocument, field: string): unknown {
	return rawJSONObjectToWails(JSON.stringify(value), field);
}

// oxlint-disable-next-line typescript/no-unnecessary-type-parameters
function objectFromWails<T extends object>(value: unknown, operation: string): T {
	return requireWailsBody(value as T | null | undefined, operation);
}

export class WailsMCPArtifactAPI implements IMCPAPI {
	async runtimeServerIDForArtifact(artifact: ArtifactRef): Promise<MCPRuntimeServerID> {
		const server = await RuntimeServerIDForArtifact(artifact as Parameters<typeof RuntimeServerIDForArtifact>[0]);

		return requireNonBlankString(server, 'RuntimeServerIDForArtifact');
	}

	async artifactRefForRuntimeServerID(server: MCPRuntimeServerID): Promise<ArtifactRef> {
		const artifact = await ArtifactRefForRuntimeServerID(server as Parameters<typeof ArtifactRefForRuntimeServerID>[0]);

		return objectFromWails<ArtifactRef>(artifact, 'ArtifactRefForRuntimeServerID');
	}

	async createMCPBundle(input: MCPCreateBundleInput): Promise<MCPBundle> {
		const bundle = await CreateMCPBundle({
			RootID: input.rootID,
			CollectionID: input.collectionID,
			SourceID: input.sourceID,
			SourceStorageKey: input.sourceStorageKey,
			Document: documentToWails(input.document, 'CreateMCPBundle.document'),
			Registrations: input.registrations.map((registration, index) =>
				registrationToWails(registration, `CreateMCPBundle.registrations[${index}]`)
			),
		} as Parameters<typeof CreateMCPBundle>[0]);
		return objectFromWails<MCPBundle>(bundle, 'CreateMCPBundle');
	}

	async getMCPBundle(ref: ArtifactCollectionRef): Promise<MCPBundle> {
		const bundle = await GetMCPBundle(ref as Parameters<typeof GetMCPBundle>[0]);
		return objectFromWails<MCPBundle>(bundle, 'GetMCPBundle');
	}

	async listMCPBundles(rootID: string): Promise<MCPBundle[]> {
		const bundles = await ListMCPBundles(rootID);
		return wailsObjectArrayOrEmpty<MCPBundle>(bundles, 'ListMCPBundles');
	}

	async getMCPBundleDocument(bundle: ArtifactCollectionRef): Promise<MCPBundleDocument> {
		const document = await GetMCPBundleDocument(bundle as Parameters<typeof GetMCPBundleDocument>[0]);
		return objectFromWails<MCPBundleDocument>(document, 'GetMCPBundleDocument');
	}

	async listMCPBundleServers(bundle: ArtifactCollectionRef): Promise<ArtifactRecord[]> {
		const servers = await ListMCPBundleServers(bundle as Parameters<typeof ListMCPBundleServers>[0]);
		return wailsObjectArrayOrEmpty<ArtifactRecord>(servers, 'ListMCPBundleServers');
	}

	async listMCPBundlePolicies(bundle: ArtifactCollectionRef): Promise<ArtifactRecord[]> {
		const policies = await ListMCPBundlePolicies(bundle as Parameters<typeof ListMCPBundlePolicies>[0]);
		return wailsObjectArrayOrEmpty<ArtifactRecord>(policies, 'ListMCPBundlePolicies');
	}

	async getMCPBundleInstallation(bundle: ArtifactCollectionRef): Promise<MCPBundleInstallation> {
		const installation = await GetMCPBundleInstallation(bundle as Parameters<typeof GetMCPBundleInstallation>[0]);
		return objectFromWails<MCPBundleInstallation>(installation, 'GetMCPBundleInstallation');
	}

	async replaceMCPBundleDocument(input: MCPReplaceBundleDocumentInput): Promise<MCPBundle> {
		const bundle = await ReplaceMCPBundleDocument({
			Bundle: input.bundle,
			ExpectedCollectionRevision: input.expectedCollectionRevision,
			Document: documentToWails(input.document, 'ReplaceMCPBundleDocument.document'),
			Registrations: input.registrations.map((registration, index) =>
				registrationToWails(registration, `ReplaceMCPBundleDocument.registrations[${index}]`)
			),
			AllowProtected: false,
		} as Parameters<typeof ReplaceMCPBundleDocument>[0]);
		return objectFromWails<MCPBundle>(bundle, 'ReplaceMCPBundleDocument');
	}

	async refreshMCPBundle(bundle: ArtifactCollectionRef): Promise<MCPBundle> {
		const refreshed = await RefreshMCPBundle(bundle as Parameters<typeof RefreshMCPBundle>[0]);
		return objectFromWails<MCPBundle>(refreshed, 'RefreshMCPBundle');
	}

	async updateMCPBundleEnabled(
		bundle: ArtifactCollectionRef,
		expectedRevision: number,
		enabled: boolean
	): Promise<MCPBundle> {
		const updated = await UpdateMCPBundleEnabled(
			bundle as Parameters<typeof UpdateMCPBundleEnabled>[0],
			expectedRevision,
			enabled
		);
		return objectFromWails<MCPBundle>(updated, 'UpdateMCPBundleEnabled');
	}

	async updateProtectedMCPBundleInstallation(
		bundle: ArtifactCollectionRef,
		expectedOverlayRevision: number,
		runtimeEnabled: boolean
	): Promise<void> {
		await UpdateProtectedMCPBundleInstallation(
			bundle as Parameters<typeof UpdateProtectedMCPBundleInstallation>[0],
			expectedOverlayRevision,
			runtimeEnabled
		);
	}

	async retireMCPBundle(bundle: ArtifactCollectionRef, expectedRevision: number): Promise<ArtifactCollection> {
		const collection = await RetireMCPBundle(bundle as Parameters<typeof RetireMCPBundle>[0], expectedRevision);
		return objectFromWails<ArtifactCollection>(collection, 'RetireMCPBundle');
	}

	async purgeMCPBundle(bundle: ArtifactCollectionRef, expectedRevision: number): Promise<void> {
		await PurgeMCPBundle(bundle as Parameters<typeof PurgeMCPBundle>[0], expectedRevision);
	}

	async getMCPServerInstallation(server: ArtifactRef): Promise<MCPServerInstallation> {
		const installation = await GetMCPServerInstallation(server as Parameters<typeof GetMCPServerInstallation>[0]);
		return objectFromWails<MCPServerInstallation>(installation, 'GetMCPServerInstallation');
	}

	async inspectMCPServer(server: ArtifactRef): Promise<MCPServerResolved> {
		const ret = await InspectMCPServer(server as Parameters<typeof InspectMCPServer>[0]);
		return objectFromWails<MCPServerResolved>(ret, 'InspectMCPServer');
	}

	async inspectMCPPolicy(policyRef: ArtifactRef): Promise<MCPPolicyView> {
		const raw = await InspectMCPPolicy(policyRef as Parameters<typeof InspectMCPPolicy>[0]);
		const resp = objectFromWails<MCPPolicyView>(raw, 'InspectMCPPolicy');
		const definition = requireWailsBody(resp.definition, 'InspectMCPPolicy.definition');
		const p = {
			...resp,
			definition: {
				...definition,
				body: rawJSONFromWails(definition.body, 'policy.definition.body'),
			},
		} as MCPPolicyView;
		return p;
	}

	async updateMCPServerInstallation(
		server: ArtifactRef,
		expectedArtifactRevision: number,
		data: MCPServerData
	): Promise<ArtifactRecord> {
		const artifact = await UpdateMCPServerInstallation(
			server as Parameters<typeof UpdateMCPServerInstallation>[0],
			expectedArtifactRevision,
			data as Parameters<typeof UpdateMCPServerInstallation>[2]
		);

		return objectFromWails<ArtifactRecord>(artifact, 'UpdateMCPServerInstallation');
	}

	async updateProtectedMCPServerInstallation(
		server: ArtifactRef,
		expectedOverlayRevision: number,
		runtimeEnabled: boolean,
		data: MCPServerData
	): Promise<void> {
		await UpdateProtectedMCPServerInstallation(
			server as Parameters<typeof UpdateProtectedMCPServerInstallation>[0],
			expectedOverlayRevision,
			runtimeEnabled,
			data as Parameters<typeof UpdateProtectedMCPServerInstallation>[3]
		);
	}

	async connectMCPServer(server: MCPRuntimeServerID): Promise<MCPServerRuntimeSnapshot> {
		// UI callers poll runtime state and must not block the Wails invocation
		// while an interactive OAuth authorization is pending.
		const snapshot = await StartMCPServerConnect(server as Parameters<typeof StartMCPServerConnect>[0]);
		return objectFromWails<MCPServerRuntimeSnapshot>(snapshot, 'StartMCPServerConnect');
	}

	async disconnectMCPServer(server: MCPRuntimeServerID): Promise<void> {
		await DisconnectMCPServer(server as Parameters<typeof DisconnectMCPServer>[0]);
	}

	async refreshMCPServer(server: MCPRuntimeServerID): Promise<MCPServerRuntimeSnapshot> {
		const snapshot = await RefreshMCPServer(server as Parameters<typeof RefreshMCPServer>[0]);
		return objectFromWails<MCPServerRuntimeSnapshot>(snapshot, 'RefreshMCPServer');
	}

	async getMCPServerStatus(server: MCPRuntimeServerID): Promise<MCPServerRuntimeSnapshot> {
		const snapshot = await GetMCPServerStatus(server as Parameters<typeof GetMCPServerStatus>[0]);
		return objectFromWails<MCPServerRuntimeSnapshot>(snapshot, 'GetMCPServerStatus');
	}

	async listMCPServerTools(server: MCPRuntimeServerID): Promise<MCPToolCapability[]> {
		const tools = await ListMCPServerTools(server as Parameters<typeof ListMCPServerTools>[0]);
		return wailsObjectArrayOrEmpty<MCPToolCapability>(tools, 'ListMCPServerTools');
	}

	async listMCPServerResources(server: MCPRuntimeServerID): Promise<MCPResourceRef[]> {
		const resources = await ListMCPServerResources(server as Parameters<typeof ListMCPServerResources>[0]);
		return wailsObjectArrayOrEmpty<MCPResourceRef>(resources, 'ListMCPServerResources');
	}

	async listMCPServerResourceTemplates(server: MCPRuntimeServerID): Promise<MCPResourceTemplateRef[]> {
		const templates = await ListMCPServerResourceTemplates(
			server as Parameters<typeof ListMCPServerResourceTemplates>[0]
		);
		return wailsObjectArrayOrEmpty<MCPResourceTemplateRef>(templates, 'ListMCPServerResourceTemplates');
	}

	async listMCPServerPrompts(server: MCPRuntimeServerID): Promise<MCPPromptRef[]> {
		const prompts = await ListMCPServerPrompts(server as Parameters<typeof ListMCPServerPrompts>[0]);
		return wailsObjectArrayOrEmpty<MCPPromptRef>(prompts, 'ListMCPServerPrompts');
	}

	async readMCPResource(server: MCPRuntimeServerID, uri: string): Promise<MCPReadResourceResponseBody> {
		const response = await ReadMCPResource(server as Parameters<typeof ReadMCPResource>[0], uri);
		return objectFromWails<MCPReadResourceResponseBody>(response, 'ReadMCPResource');
	}

	async getMCPPrompt(
		server: MCPRuntimeServerID,
		promptName: string,
		promptArguments?: Record<string, string>
	): Promise<MCPGetPromptResponseBody> {
		const response = await GetMCPPrompt(
			server as Parameters<typeof GetMCPPrompt>[0],
			promptName,
			promptArguments ?? {}
		);
		return objectFromWails<MCPGetPromptResponseBody>(response, 'GetMCPPrompt');
	}

	async completeMCPArgument(
		server: MCPRuntimeServerID,
		refType: MCPCompletionRefType,
		name: string,
		argumentName: string,
		argumentValue?: string,
		context?: Record<string, string>
	): Promise<MCPCompletionResult> {
		const response = await CompleteMCPArgument(
			server as Parameters<typeof CompleteMCPArgument>[0],
			{
				refType,
				name,
				argumentName,
				argumentValue,
				context,
			} as Parameters<typeof CompleteMCPArgument>[1]
		);
		return objectFromWails<MCPCompletionResult>(response, 'CompleteMCPArgument');
	}

	async evaluateMCPToolCall(
		server: MCPRuntimeServerID,
		request: InvokeMCPToolRequestBody
	): Promise<MCPApprovalEvaluation> {
		const response = await EvaluateMCPToolCall(
			server as Parameters<typeof EvaluateMCPToolCall>[0],
			request as Parameters<typeof EvaluateMCPToolCall>[1]
		);
		return objectFromWails<MCPApprovalEvaluation>(response, 'EvaluateMCPToolCall');
	}

	async evaluateMappedMCPToolCall(
		mapping: MCPProviderToolMapping,
		request: InvokeMCPToolRequestBody
	): Promise<MCPApprovalEvaluation> {
		const response = await EvaluateMappedMCPToolCall(
			mapping as Parameters<typeof EvaluateMappedMCPToolCall>[0],
			request as Parameters<typeof EvaluateMappedMCPToolCall>[1]
		);
		return objectFromWails<MCPApprovalEvaluation>(response, 'EvaluateMappedMCPToolCall');
	}

	async invokeMCPTool(
		server: MCPRuntimeServerID,
		request: InvokeMCPToolRequestBody
	): Promise<InvokeMCPToolResponseBody> {
		const response = await InvokeMCPTool(
			server as Parameters<typeof InvokeMCPTool>[0],
			request as Parameters<typeof InvokeMCPTool>[1]
		);
		return objectFromWails<InvokeMCPToolResponseBody>(response, 'InvokeMCPTool');
	}

	async invokeMappedMCPTool(
		mapping: MCPProviderToolMapping,
		request: InvokeMCPToolRequestBody
	): Promise<InvokeMCPToolResponseBody> {
		const response = await InvokeMappedMCPTool(
			mapping as Parameters<typeof InvokeMappedMCPTool>[0],
			request as Parameters<typeof InvokeMappedMCPTool>[1]
		);
		return objectFromWails<InvokeMCPToolResponseBody>(response, 'InvokeMappedMCPTool');
	}

	async resolveMCPApproval(
		approvalID: string,
		resolution: MCPApprovalResolution
	): Promise<MCPApprovalResolutionResult> {
		const token = await ResolveMCPApproval(approvalID, resolution);
		return objectFromWails<MCPApprovalResolutionResult>(token, 'ResolveMCPApproval');
	}

	async getMCPServerAuthHealth(server: ArtifactRef): Promise<MCPAuthHealth> {
		const health = await GetMCPServerAuthHealth(server as Parameters<typeof GetMCPServerAuthHealth>[0]);
		return objectFromWails<MCPAuthHealth>(health, 'GetMCPServerAuthHealth');
	}

	async listPendingMCPOAuthAuthorizations(): Promise<MCPOAuthAuthorization[]> {
		const authorizations = await ListPendingMCPOAuthAuthorizations();
		return wailsObjectArrayOrEmpty<MCPOAuthAuthorization>(authorizations, 'ListPendingMCPOAuthAuthorizations');
	}

	async cancelPendingMCPOAuthAuthorization(server: MCPRuntimeServerID): Promise<boolean> {
		const cancelled = await CancelPendingMCPOAuthAuthorization(
			server as Parameters<typeof CancelPendingMCPOAuthAuthorization>[0]
		);
		return requireWailsBoolean(cancelled, 'CancelPendingMCPOAuthAuthorization');
	}

	async putMCPServerSecret(
		server: ArtifactRef,
		kind: MCPSecretKind,
		slot: string,
		secret: string
	): Promise<MCPSecretWriteResult> {
		const result = await PutMCPServerSecret(server as Parameters<typeof PutMCPServerSecret>[0], kind, slot, secret);
		return objectFromWails<MCPSecretWriteResult>(result, 'PutMCPServerSecret');
	}

	async deleteMCPServerSecret(server: ArtifactRef, kind: MCPSecretKind, slot: string): Promise<void> {
		await DeleteMCPServerSecret(server as Parameters<typeof DeleteMCPServerSecret>[0], kind, slot);
	}

	async getMCPGlobalSettings(): Promise<MCPGlobalSettings> {
		const settings = await GetMCPGlobalSettings();
		return objectFromWails<MCPGlobalSettings>(settings, 'GetMCPGlobalSettings');
	}

	async updateMCPGlobalSettings(expectedRevision: number, oauthLoopbackListenAddr?: string): Promise<number> {
		const revision = await UpdateMCPGlobalSettings(expectedRevision, {
			oauthLoopbackListenAddr,
		} as Parameters<typeof UpdateMCPGlobalSettings>[1]);
		return requireWailsFiniteNumber(revision, 'UpdateMCPGlobalSettings');
	}
}
