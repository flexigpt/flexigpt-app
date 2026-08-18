// oxlint-disable typescript/no-misused-spread
import type { ArtifactCollection, ArtifactCollectionRef, ArtifactRecord, ArtifactRef } from '@/spec/artifact';
import type {
	InvokeMCPToolRequestBody,
	InvokeMCPToolResponseBody,
	MCPApprovalEvaluation,
	MCPApprovalResolution,
	MCPApprovalToken,
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
	MCPSecretKind,
	MCPSecretWriteResult,
	MCPServerData,
	MCPServerInstallation,
	MCPServerResolved,
	MCPServerRuntimeSnapshot,
	MCPToolCapability,
} from '@/spec/mcp_artifact';

import type { IMCPAPI } from '@/apis/interface';
import { rawJSONFromWails, rawJSONObjectToWails } from '@/apis/wailsapi/transport';
import {
	CancelPendingMCPOAuthAuthorization,
	CompleteMCPArgument,
	ConnectMCPServer,
	CreateMCPBundle,
	DeleteMCPServerSecret,
	DisconnectMCPServer,
	EvaluateMappedMCPToolCall,
	EvaluateMCPToolCall,
	GetMCPBundle,
	GetMCPBundleDocument,
	GetMCPBundleInstallation,
	GetMCPGlobalSettings,
	GetMCPPrompt,
	GetMCPServerAuthHealth,
	GetMCPServerInstallation,
	GetMCPServerStatus,
	InspectMCPPolicy,
	InspectMCPServer,
	InvokeMappedMCPTool,
	InvokeMCPTool,
	ListMCPBundlePolicies,
	ListMCPBundles,
	ListMCPBundleServers,
	ListMCPServerPrompts,
	ListMCPServerResources,
	ListMCPServerResourceTemplates,
	ListMCPServerTools,
	ListPendingMCPOAuthAuthorizations,
	PurgeMCPBundle,
	PutMCPServerSecret,
	ReadMCPResource,
	RefreshMCPBundle,
	RefreshMCPServer,
	ReplaceMCPBundleDocument,
	ResolveMCPApproval,
	RetireMCPBundle,
	UpdateMCPBundleEnabled,
	UpdateMCPGlobalSettings,
	UpdateMCPServerInstallation,
	UpdateProtectedMCPBundleInstallation,
	UpdateProtectedMCPServerInstallation,
} from '@/apis/wailsjs/go/main/MCPWrapper';

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

export class WailsMCPArtifactAPI implements IMCPAPI {
	async createMCPBundle(input: MCPCreateBundleInput): Promise<MCPBundle> {
		return (await CreateMCPBundle({
			RootID: input.rootID,
			CollectionID: input.collectionID,
			SourceID: input.sourceID,
			SourceStorageKey: input.sourceStorageKey,
			Document: documentToWails(input.document, 'CreateMCPBundle.document'),
			Registrations: input.registrations.map((registration, index) =>
				registrationToWails(registration, `CreateMCPBundle.registrations[${index}]`)
			),
		} as Parameters<typeof CreateMCPBundle>[0])) as MCPBundle;
	}

	async getMCPBundle(ref: ArtifactCollectionRef): Promise<MCPBundle> {
		return (await GetMCPBundle(ref as Parameters<typeof GetMCPBundle>[0])) as MCPBundle;
	}

	async listMCPBundles(rootID: string): Promise<MCPBundle[]> {
		return (await ListMCPBundles(rootID)) as MCPBundle[];
	}

	async getMCPBundleDocument(bundle: ArtifactCollectionRef): Promise<MCPBundleDocument> {
		return (await GetMCPBundleDocument(bundle as Parameters<typeof GetMCPBundleDocument>[0])) as MCPBundleDocument;
	}

	async listMCPBundleServers(bundle: ArtifactCollectionRef): Promise<ArtifactRecord[]> {
		const servers = await ListMCPBundleServers(bundle as Parameters<typeof ListMCPBundleServers>[0]);
		return servers as ArtifactRecord[];
	}

	async listMCPBundlePolicies(bundle: ArtifactCollectionRef): Promise<ArtifactRecord[]> {
		const policies = await ListMCPBundlePolicies(bundle as Parameters<typeof ListMCPBundlePolicies>[0]);
		return policies as ArtifactRecord[];
	}

	async getMCPBundleInstallation(bundle: ArtifactCollectionRef): Promise<MCPBundleInstallation> {
		return (await GetMCPBundleInstallation(
			bundle as Parameters<typeof GetMCPBundleInstallation>[0]
		)) as MCPBundleInstallation;
	}

	async replaceMCPBundleDocument(input: MCPReplaceBundleDocumentInput): Promise<MCPBundle> {
		return (await ReplaceMCPBundleDocument({
			Bundle: input.bundle,
			ExpectedCollectionRevision: input.expectedCollectionRevision,
			Document: documentToWails(input.document, 'ReplaceMCPBundleDocument.document'),
			Registrations: input.registrations.map((registration, index) =>
				registrationToWails(registration, `ReplaceMCPBundleDocument.registrations[${index}]`)
			),
			AllowProtected: false,
		} as Parameters<typeof ReplaceMCPBundleDocument>[0])) as MCPBundle;
	}

	async refreshMCPBundle(bundle: ArtifactCollectionRef): Promise<MCPBundle> {
		return (await RefreshMCPBundle(bundle as Parameters<typeof RefreshMCPBundle>[0])) as MCPBundle;
	}

	async updateMCPBundleEnabled(
		bundle: ArtifactCollectionRef,
		expectedRevision: number,
		enabled: boolean
	): Promise<MCPBundle> {
		return (await UpdateMCPBundleEnabled(
			bundle as Parameters<typeof UpdateMCPBundleEnabled>[0],
			expectedRevision,
			enabled
		)) as MCPBundle;
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

		return collection;
	}

	async purgeMCPBundle(bundle: ArtifactCollectionRef, expectedRevision: number): Promise<void> {
		await PurgeMCPBundle(bundle as Parameters<typeof PurgeMCPBundle>[0], expectedRevision);
	}

	async getMCPServerInstallation(server: ArtifactRef): Promise<MCPServerInstallation> {
		return (await GetMCPServerInstallation(
			server as Parameters<typeof GetMCPServerInstallation>[0]
		)) as MCPServerInstallation;
	}

	async inspectMCPServer(server: ArtifactRef): Promise<MCPServerResolved> {
		const ret = await InspectMCPServer(server as Parameters<typeof InspectMCPServer>[0]);
		return ret as MCPServerResolved;
	}

	async inspectMCPPolicy(policyRef: ArtifactRef): Promise<MCPPolicyView> {
		const resp = await InspectMCPPolicy(policyRef as Parameters<typeof InspectMCPPolicy>[0]);
		const p = {
			...resp,
			definition: {
				...resp.definition,
				body: rawJSONFromWails(resp.definition.body, 'policy.definition.body'),
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

		return artifact as ArtifactRecord;
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

	async connectMCPServer(server: ArtifactRef): Promise<MCPServerRuntimeSnapshot> {
		return (await ConnectMCPServer(server as Parameters<typeof ConnectMCPServer>[0])) as MCPServerRuntimeSnapshot;
	}

	async disconnectMCPServer(server: ArtifactRef): Promise<void> {
		await DisconnectMCPServer(server as Parameters<typeof DisconnectMCPServer>[0]);
	}

	async refreshMCPServer(server: ArtifactRef): Promise<MCPServerRuntimeSnapshot> {
		return (await RefreshMCPServer(server as Parameters<typeof RefreshMCPServer>[0])) as MCPServerRuntimeSnapshot;
	}

	async getMCPServerStatus(server: ArtifactRef): Promise<MCPServerRuntimeSnapshot> {
		return (await GetMCPServerStatus(server as Parameters<typeof GetMCPServerStatus>[0])) as MCPServerRuntimeSnapshot;
	}

	async listMCPServerTools(server: ArtifactRef): Promise<MCPToolCapability[]> {
		return (await ListMCPServerTools(server as Parameters<typeof ListMCPServerTools>[0])) as MCPToolCapability[];
	}

	async listMCPServerResources(server: ArtifactRef): Promise<MCPResourceRef[]> {
		return (await ListMCPServerResources(server as Parameters<typeof ListMCPServerResources>[0])) as MCPResourceRef[];
	}

	async listMCPServerResourceTemplates(server: ArtifactRef): Promise<MCPResourceTemplateRef[]> {
		return (await ListMCPServerResourceTemplates(
			server as Parameters<typeof ListMCPServerResourceTemplates>[0]
		)) as MCPResourceTemplateRef[];
	}

	async listMCPServerPrompts(server: ArtifactRef): Promise<MCPPromptRef[]> {
		return (await ListMCPServerPrompts(server as Parameters<typeof ListMCPServerPrompts>[0])) as MCPPromptRef[];
	}

	async readMCPResource(server: ArtifactRef, uri: string): Promise<MCPReadResourceResponseBody> {
		return (await ReadMCPResource(server as Parameters<typeof ReadMCPResource>[0], uri)) as MCPReadResourceResponseBody;
	}

	async getMCPPrompt(
		server: ArtifactRef,
		promptName: string,
		promptArguments?: Record<string, string>
	): Promise<MCPGetPromptResponseBody> {
		return (await GetMCPPrompt(
			server as Parameters<typeof GetMCPPrompt>[0],
			promptName,
			promptArguments ?? {}
		)) as MCPGetPromptResponseBody;
	}

	async completeMCPArgument(
		server: ArtifactRef,
		refType: MCPCompletionRefType,
		name: string,
		argumentName: string,
		argumentValue?: string,
		context?: Record<string, string>
	): Promise<MCPCompletionResult> {
		return (await CompleteMCPArgument(
			server as Parameters<typeof CompleteMCPArgument>[0],
			{
				refType,
				name,
				argumentName,
				argumentValue,
				context,
			} as Parameters<typeof CompleteMCPArgument>[1]
		)) as MCPCompletionResult;
	}

	async evaluateMCPToolCall(server: ArtifactRef, request: InvokeMCPToolRequestBody): Promise<MCPApprovalEvaluation> {
		return (await EvaluateMCPToolCall(
			server as Parameters<typeof EvaluateMCPToolCall>[0],
			request as Parameters<typeof EvaluateMCPToolCall>[1]
		)) as MCPApprovalEvaluation;
	}

	async evaluateMappedMCPToolCall(
		mapping: MCPProviderToolMapping,
		request: InvokeMCPToolRequestBody
	): Promise<MCPApprovalEvaluation> {
		return (await EvaluateMappedMCPToolCall(
			mapping as Parameters<typeof EvaluateMappedMCPToolCall>[0],
			request as Parameters<typeof EvaluateMappedMCPToolCall>[1]
		)) as MCPApprovalEvaluation;
	}

	async invokeMCPTool(server: ArtifactRef, request: InvokeMCPToolRequestBody): Promise<InvokeMCPToolResponseBody> {
		return (await InvokeMCPTool(
			server as Parameters<typeof InvokeMCPTool>[0],
			request as Parameters<typeof InvokeMCPTool>[1]
		)) as InvokeMCPToolResponseBody;
	}

	async invokeMappedMCPTool(
		mapping: MCPProviderToolMapping,
		request: InvokeMCPToolRequestBody
	): Promise<InvokeMCPToolResponseBody> {
		return (await InvokeMappedMCPTool(
			mapping as Parameters<typeof InvokeMappedMCPTool>[0],
			request as Parameters<typeof InvokeMappedMCPTool>[1]
		)) as InvokeMCPToolResponseBody;
	}

	async resolveMCPApproval(approvalID: string, resolution: MCPApprovalResolution): Promise<MCPApprovalToken> {
		return (await ResolveMCPApproval(approvalID, resolution)) as MCPApprovalToken;
	}

	async getMCPServerAuthHealth(server: ArtifactRef): Promise<MCPAuthHealth> {
		return (await GetMCPServerAuthHealth(server as Parameters<typeof GetMCPServerAuthHealth>[0])) as MCPAuthHealth;
	}

	async listPendingMCPOAuthAuthorizations(): Promise<MCPOAuthAuthorization[]> {
		return (await ListPendingMCPOAuthAuthorizations()) as MCPOAuthAuthorization[];
	}

	async cancelPendingMCPOAuthAuthorization(server: ArtifactRef): Promise<boolean> {
		return CancelPendingMCPOAuthAuthorization(server as Parameters<typeof CancelPendingMCPOAuthAuthorization>[0]);
	}

	async putMCPServerSecret(
		server: ArtifactRef,
		kind: MCPSecretKind,
		slot: string,
		secret: string
	): Promise<MCPSecretWriteResult> {
		return (await PutMCPServerSecret(
			server as Parameters<typeof PutMCPServerSecret>[0],
			kind,
			slot,
			secret
		)) as MCPSecretWriteResult;
	}

	async deleteMCPServerSecret(server: ArtifactRef, kind: MCPSecretKind, slot: string): Promise<void> {
		await DeleteMCPServerSecret(server as Parameters<typeof DeleteMCPServerSecret>[0], kind, slot);
	}

	async getMCPGlobalSettings(): Promise<MCPGlobalSettings> {
		return (await GetMCPGlobalSettings()) as MCPGlobalSettings;
	}

	async updateMCPGlobalSettings(expectedRevision: number, oauthLoopbackListenAddr?: string): Promise<number> {
		return UpdateMCPGlobalSettings(expectedRevision, {
			oauthLoopbackListenAddr,
		} as Parameters<typeof UpdateMCPGlobalSettings>[1]);
	}
}
