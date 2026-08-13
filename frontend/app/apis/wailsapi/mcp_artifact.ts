// oxlint-disable oxc/no-map-spread
import type {
	ArtifactCollection,
	ArtifactCollectionAttachment,
	ArtifactCollectionRef,
	ArtifactDefinitionView,
	ArtifactDiagnostic,
	ArtifactRecord,
	ArtifactRef,
	ArtifactSourceSummary,
} from '@/spec/artifact';
import { ArtifactAdoptionMode, ArtifactDiagnosticSeverity, ArtifactState } from '@/spec/artifact';
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
	MCPPolicy,
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
	MCPServerDocument,
	MCPServerInstallation,
	MCPServerResolved,
	MCPServerRuntimeSnapshot,
	MCPToolCapability,
} from '@/spec/mcp_artifact';
import {
	MCPApprovalDecision,
	MCPApprovalRule,
	MCPAppVisibility,
	MCPAuthHealthState,
	MCPExecutionMode,
	MCPHTTPAuthMode,
	MCPInputKind,
	MCPPromptRole,
	MCPServerStatus,
	MCPServerType,
	MCPTaskSupport,
	MCPToolRisk,
	MCPTrustLevel,
} from '@/spec/mcp_artifact';

import type { IMCPAPI } from '@/apis/interface';
import {
	enumFromWails,
	omitUndefined,
	optionalFrontendDate,
	optionalWailsArray,
	optionalWailsString,
	rawJSONFromWails,
	rawJSONObjectToWails,
	requireNonBlankString,
	requireWailsArray,
	requireWailsBoolean,
	requireWailsNumber,
	requireWailsObject,
	requireWailsString,
	toFrontendDate,
	toFrontendTimestamp,
} from '@/apis/wailsapi/transport';
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

function artifactRefFromWails(value: unknown, field: string): ArtifactRef {
	const ref = requireWailsObject(value, field);
	return {
		rootID: requireWailsString(ref.rootID, `${field}.rootID`),
		artifactID: requireWailsString(ref.artifactID, `${field}.artifactID`),
	};
}

function artifactRefToWails(ref: ArtifactRef): unknown {
	return {
		rootID: ref.rootID,
		artifactID: ref.artifactID,
	};
}

function collectionRefFromWails(value: unknown, field: string): ArtifactCollectionRef {
	const ref = requireWailsObject(value, field);
	return {
		rootID: requireWailsString(ref.rootID, `${field}.rootID`),
		collectionID: requireWailsString(ref.collectionID, `${field}.collectionID`),
	};
}

function collectionRefToWails(ref: ArtifactCollectionRef): unknown {
	return {
		rootID: ref.rootID,
		collectionID: ref.collectionID,
	};
}

function diagnosticFromWails(value: unknown, field: string): ArtifactDiagnostic {
	const diagnostic = requireWailsObject(value, field);
	const rawLocation = diagnostic.location;
	const location =
		rawLocation === undefined || rawLocation === null
			? undefined
			: (() => {
					const item = requireWailsObject(rawLocation, `${field}.location`);
					return {
						locator: optionalWailsString(item.locator, `${field}.location.locator`),
						subresourceLocator: optionalWailsString(item.subresourceLocator, `${field}.location.subresourceLocator`),
						line: item.line === undefined ? undefined : requireWailsNumber(item.line, `${field}.location.line`),
						column: item.column === undefined ? undefined : requireWailsNumber(item.column, `${field}.location.column`),
					};
				})();

	return {
		severity: enumFromWails(diagnostic.severity, ArtifactDiagnosticSeverity, `${field}.severity`),
		code: requireWailsString(diagnostic.code, `${field}.code`),
		message: requireWailsString(diagnostic.message, `${field}.message`),
		location,
	};
}

function artifactRecordFromWails(value: unknown, field: string): ArtifactRecord {
	const record = requireWailsObject(value, field);
	const binding = requireWailsObject(record.binding, `${field}.binding`);

	return {
		id: requireWailsString(record.id, `${field}.id`),
		rootID: requireWailsString(record.rootID, `${field}.rootID`),
		collectionID: requireWailsString(record.collectionID, `${field}.collectionID`),
		binding: {
			sourceID: requireWailsString(binding.sourceID, `${field}.binding.sourceID`),
			locator: requireWailsString(binding.locator, `${field}.binding.locator`),
			subresourceLocator: optionalWailsString(binding.subresourceLocator, `${field}.binding.subresourceLocator`),
			expectedKind: requireWailsString(binding.expectedKind, `${field}.binding.expectedKind`),
		},
		kind: requireWailsString(record.kind, `${field}.kind`),
		name: requireWailsString(record.name, `${field}.name`),
		enabled: requireWailsBoolean(record.enabled, `${field}.enabled`),
		adoption: enumFromWails(record.adoption, ArtifactAdoptionMode, `${field}.adoption`),
		resolvedDefinition: optionalWailsString(record.resolvedDefinition, `${field}.resolvedDefinition`),
		state: enumFromWails(record.state, ArtifactState, `${field}.state`),
		diagnostics: optionalWailsArray(record.diagnostics, `${field}.diagnostics`)?.map((item, index) =>
			diagnosticFromWails(item, `${field}.diagnostics[${index}]`)
		),
		revision: requireWailsNumber(record.revision, `${field}.revision`),
		createdAt: toFrontendDate(record.createdAt, `${field}.createdAt`),
		modifiedAt: toFrontendDate(record.modifiedAt, `${field}.modifiedAt`),
	};
}

function collectionFromWails(value: unknown, field: string): ArtifactCollection {
	const collection = requireWailsObject(value, field);

	return {
		id: requireWailsString(collection.id, `${field}.id`),
		rootID: requireWailsString(collection.rootID, `${field}.rootID`),
		kind: requireWailsString(collection.kind, `${field}.kind`),
		displayName: requireWailsString(collection.displayName, `${field}.displayName`),
		description: optionalWailsString(collection.description, `${field}.description`),
		enabled: requireWailsBoolean(collection.enabled, `${field}.enabled`),
		revision: requireWailsNumber(collection.revision, `${field}.revision`),
		createdAt: toFrontendDate(collection.createdAt, `${field}.createdAt`),
		modifiedAt: toFrontendDate(collection.modifiedAt, `${field}.modifiedAt`),
		retiredAt: optionalFrontendDate(collection.retiredAt, `${field}.retiredAt`),
	};
}

function attachmentFromWails(value: unknown, field: string): ArtifactCollectionAttachment {
	const attachment = requireWailsObject(value, field);

	return {
		rootID: requireWailsString(attachment.rootID, `${field}.rootID`),
		collectionID: requireWailsString(attachment.collectionID, `${field}.collectionID`),
		sourceID: requireWailsString(attachment.sourceID, `${field}.sourceID`),
		role: requireWailsString(attachment.role, `${field}.role`),
		enabled: requireWailsBoolean(attachment.enabled, `${field}.enabled`),
		revision: requireWailsNumber(attachment.revision, `${field}.revision`),
		createdAt: toFrontendDate(attachment.createdAt, `${field}.createdAt`),
		modifiedAt: toFrontendDate(attachment.modifiedAt, `${field}.modifiedAt`),
	};
}

function sourceFromWails(value: unknown, field: string): ArtifactSourceSummary {
	const source = requireWailsObject(value, field);

	return {
		id: requireWailsString(source.id, `${field}.id`),
		rootID: requireWailsString(source.rootID, `${field}.rootID`),
		rootStorageKey: requireWailsString(source.rootStorageKey, `${field}.rootStorageKey`),
		storageKey: requireWailsString(source.storageKey, `${field}.storageKey`),
		kind: requireWailsString(source.kind, `${field}.kind`),
		displayName: requireWailsString(source.displayName, `${field}.displayName`),
		enabled: requireWailsBoolean(source.enabled, `${field}.enabled`),
		revision: requireWailsNumber(source.revision, `${field}.revision`),
		createdAt: toFrontendDate(source.createdAt, `${field}.createdAt`),
		modifiedAt: toFrontendDate(source.modifiedAt, `${field}.modifiedAt`),
		retiredAt: optionalFrontendDate(source.retiredAt, `${field}.retiredAt`),
	};
}

function policyFromWails(value: unknown, field: string): MCPPolicy {
	const policy = requireWailsObject(value, field);
	const defaultPolicy = requireWailsObject(policy.defaultPolicy, `${field}.defaultPolicy`);
	const appsPolicy = requireWailsObject(policy.appsPolicy, `${field}.appsPolicy`);

	let toolPolicies: Record<string, any> | undefined;
	if (policy.toolPolicies !== undefined) {
		const raw = requireWailsObject(policy.toolPolicies, `${field}.toolPolicies`);
		toolPolicies = Object.fromEntries(
			Object.entries(raw).map(([name, item]) => {
				const override = requireWailsObject(item, `${field}.toolPolicies.${name}`);
				return [
					name,
					{
						toolName: requireWailsString(override.toolName, `${field}.toolPolicies.${name}.toolName`),
						approvalRule:
							override.approvalRule === undefined
								? undefined
								: enumFromWails(override.approvalRule, MCPApprovalRule, `${field}.toolPolicies.${name}.approvalRule`),
						executionMode:
							override.executionMode === undefined
								? undefined
								: enumFromWails(
										override.executionMode,
										MCPExecutionMode,
										`${field}.toolPolicies.${name}.executionMode`
									),
						allowStaleDigest:
							override.allowStaleDigest === undefined
								? undefined
								: requireWailsBoolean(override.allowStaleDigest, `${field}.toolPolicies.${name}.allowStaleDigest`),
						expectedDigest: optionalWailsString(
							override.expectedDigest,
							`${field}.toolPolicies.${name}.expectedDigest`
						),
					},
				];
			})
		);
	}

	return {
		trustLevel: enumFromWails(policy.trustLevel, MCPTrustLevel, `${field}.trustLevel`),
		defaultPolicy: {
			defaultApprovalRule: enumFromWails(
				defaultPolicy.defaultApprovalRule,
				MCPApprovalRule,
				`${field}.defaultPolicy.defaultApprovalRule`
			),
			defaultExecutionMode: enumFromWails(
				defaultPolicy.defaultExecutionMode,
				MCPExecutionMode,
				`${field}.defaultPolicy.defaultExecutionMode`
			),
			requireApprovalForUnknownRisk: requireWailsBoolean(
				defaultPolicy.requireApprovalForUnknownRisk,
				`${field}.defaultPolicy.requireApprovalForUnknownRisk`
			),
			requireApprovalForWrite: requireWailsBoolean(
				defaultPolicy.requireApprovalForWrite,
				`${field}.defaultPolicy.requireApprovalForWrite`
			),
			requireApprovalForDestructive: requireWailsBoolean(
				defaultPolicy.requireApprovalForDestructive,
				`${field}.defaultPolicy.requireApprovalForDestructive`
			),
		},
		toolPolicies,
		appsPolicy: {
			enabled: requireWailsBoolean(appsPolicy.enabled, `${field}.appsPolicy.enabled`),
			allowAppInitiatedToolCalls: requireWailsBoolean(
				appsPolicy.allowAppInitiatedToolCalls,
				`${field}.appsPolicy.allowAppInitiatedToolCalls`
			),
			requireApprovalForOpenLink: requireWailsBoolean(
				appsPolicy.requireApprovalForOpenLink,
				`${field}.appsPolicy.requireApprovalForOpenLink`
			),
			requireApprovalForContextUpdates: requireWailsBoolean(
				appsPolicy.requireApprovalForContextUpdates,
				`${field}.appsPolicy.requireApprovalForContextUpdates`
			),
		},
	};
}

function serverDocumentFromWails(value: unknown, field: string): MCPServerDocument {
	const document = requireWailsObject(value, field);
	const core = requireWailsObject(document.mcpServer, `${field}.mcpServer`);
	const extension = requireWailsObject(document.extension, `${field}.extension`);
	const auth = requireWailsObject(extension.auth, `${field}.extension.auth`);
	const install = requireWailsObject(extension.install, `${field}.extension.install`);

	let inputs: Record<string, any> | undefined;
	if (install.inputs !== undefined) {
		const rawInputs = requireWailsObject(install.inputs, `${field}.extension.install.inputs`);
		inputs = Object.fromEntries(
			Object.entries(rawInputs).map(([name, input]) => {
				const declaration = requireWailsObject(input, `${field}.extension.install.inputs.${name}`);
				return [
					name,
					{
						...declaration,
						kind: enumFromWails(declaration.kind, MCPInputKind, `${field}.extension.install.inputs.${name}.kind`),
					},
				];
			})
		);
	}

	return {
		kind: requireWailsString(document.kind, `${field}.kind`),
		schemaID: requireWailsString(document.schemaID, `${field}.schemaID`),
		schemaVersion: requireWailsString(document.schemaVersion, `${field}.schemaVersion`),
		digest: optionalWailsString(document.digest, `${field}.digest`),
		logicalName: requireWailsString(document.logicalName, `${field}.logicalName`),
		logicalVersion: optionalWailsString(document.logicalVersion, `${field}.logicalVersion`),
		displayName: optionalWailsString(document.displayName, `${field}.displayName`),
		description: optionalWailsString(document.description, `${field}.description`),
		labels: document.labels as Record<string, string> | undefined,
		mcpServer: {
			type: enumFromWails(core.type, MCPServerType, `${field}.mcpServer.type`),
			command: optionalWailsString(core.command, `${field}.mcpServer.command`),
			args: core.args as string[] | undefined,
			env: core.env as Record<string, string> | undefined,
			url: optionalWailsString(core.url, `${field}.mcpServer.url`),
			headers: core.headers as Record<string, string> | undefined,
		},
		extension: {
			logicalVersion: optionalWailsString(extension.logicalVersion, `${field}.extension.logicalVersion`),
			displayName: optionalWailsString(extension.displayName, `${field}.extension.displayName`),
			description: optionalWailsString(extension.description, `${field}.extension.description`),
			timeoutMS:
				extension.timeoutMS === undefined
					? undefined
					: requireWailsNumber(extension.timeoutMS, `${field}.extension.timeoutMS`),
			labels: extension.labels as Record<string, string> | undefined,
			auth: {
				mode: enumFromWails(auth.mode, MCPHTTPAuthMode, `${field}.extension.auth.mode`),
				clientCredentialsInput: optionalWailsString(
					auth.clientCredentialsInput,
					`${field}.extension.auth.clientCredentialsInput`
				),
				clientIDMetadataDocumentURL: optionalWailsString(
					auth.clientIDMetadataDocumentURL,
					`${field}.extension.auth.clientIDMetadataDocumentURL`
				),
			},
			install: {
				note: optionalWailsString(install.note, `${field}.extension.install.note`),
				inputs,
				allowEnvironment: install.allowEnvironment as string[] | undefined,
			},
			connectionProfiles: extension.connectionProfiles as any,
			policy: extension.policy as any,
		},
	};
}

function serverDataFromWails(value: unknown, field: string): MCPServerData {
	const data = requireWailsObject(value, field);
	return {
		schemaVersion: requireWailsString(data.schemaVersion, `${field}.schemaVersion`),
		selectedConnectionProfile: optionalWailsString(
			data.selectedConnectionProfile,
			`${field}.selectedConnectionProfile`
		),
		inputs: data.inputs as MCPServerData['inputs'],
		additionalPolicies: optionalWailsArray(data.additionalPolicies, `${field}.additionalPolicies`)?.map((item, index) =>
			artifactRefFromWails(item, `${field}.additionalPolicies[${index}]`)
		),
	};
}

function bundleDocumentFromWails(value: unknown, field: string): MCPBundleDocument {
	const document = requireWailsObject(value, field);
	return {
		...document,
		kind: requireWailsString(document.kind, `${field}.kind`),
		schemaID: requireWailsString(document.schemaID, `${field}.schemaID`),
		schemaVersion: requireWailsString(document.schemaVersion, `${field}.schemaVersion`),
		digest: optionalWailsString(document.digest, `${field}.digest`),
		logicalName: requireWailsString(document.logicalName, `${field}.logicalName`),
		mcpServers: requireWailsObject(document.mcpServers, `${field}.mcpServers`) as any,
		bundleExtension: requireWailsObject(document.bundleExtension, `${field}.bundleExtension`) as any,
	} as MCPBundleDocument;
}

function bundleFromWails(value: unknown, field: string): MCPBundle {
	const bundle = requireWailsObject(value, field);
	const collection = collectionFromWails(bundle.Collection, `${field}.Collection`);
	const data = requireWailsObject(bundle.Data, `${field}.Data`);
	const attachments = requireWailsArray(bundle.Attachments, `${field}.Attachments`).map((item, index) =>
		attachmentFromWails(item, `${field}.Attachments[${index}]`)
	);
	const sources = requireWailsArray(bundle.Sources, `${field}.Sources`).map((item, index) =>
		sourceFromWails(item, `${field}.Sources[${index}]`)
	);

	return {
		collection,
		data: {
			schemaVersion: requireWailsString(data.schemaVersion, `${field}.Data.schemaVersion`),
			discoveryPolicyRevision: requireWailsString(
				data.discoveryPolicyRevision,
				`${field}.Data.discoveryPolicyRevision`
			),
			logicalName: requireWailsString(data.logicalName, `${field}.Data.logicalName`),
			logicalVersion: optionalWailsString(data.logicalVersion, `${field}.Data.logicalVersion`),
			labels: data.labels as Record<string, string> | undefined,
			managedSourceID: optionalWailsString(data.managedSourceID, `${field}.Data.managedSourceID`),
		},
		attachments,
		sources,
		builtIn: attachments.some(attachment => attachment.role === 'builtin'),
	};
}

function runtimeSnapshotFromWails(value: unknown, field: string): MCPServerRuntimeSnapshot {
	const snapshot = requireWailsObject(value, field);
	return {
		...snapshot,
		server: artifactRefFromWails(snapshot.server, `${field}.server`),
		collection: collectionRefFromWails(snapshot.collection, `${field}.collection`),
		status: enumFromWails(snapshot.status, MCPServerStatus, `${field}.status`),
		lastConnectedAt: optionalWailsString(snapshot.lastConnectedAt, `${field}.lastConnectedAt`),
		lastSyncedAt: optionalWailsString(snapshot.lastSyncedAt, `${field}.lastSyncedAt`),
	} as MCPServerRuntimeSnapshot;
}

function toolFromWails(value: unknown, field: string): MCPToolCapability {
	const tool = requireWailsObject(value, field);
	const app =
		tool.app === undefined
			? undefined
			: (() => {
					const raw = requireWailsObject(tool.app, `${field}.app`);
					return {
						resourceUri: optionalWailsString(raw.resourceUri, `${field}.app.resourceUri`),
						visibility: optionalWailsArray(raw.visibility, `${field}.app.visibility`)?.map((item, index) =>
							enumFromWails(item, MCPAppVisibility, `${field}.app.visibility[${index}]`)
						),
					};
				})();

	return {
		...tool,
		server: artifactRefFromWails(tool.server, `${field}.server`),
		inferredRisk: enumFromWails(tool.inferredRisk, MCPToolRisk, `${field}.inferredRisk`),
		approvalRule: enumFromWails(tool.approvalRule, MCPApprovalRule, `${field}.approvalRule`),
		executionMode: enumFromWails(tool.executionMode, MCPExecutionMode, `${field}.executionMode`),
		taskSupport: enumFromWails(tool.taskSupport, MCPTaskSupport, `${field}.taskSupport`),
		app,
	} as MCPToolCapability;
}

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

function providerMappingToWails(mapping: MCPProviderToolMapping): unknown {
	return {
		server: artifactRefToWails(mapping.server),
		providerToolName: mapping.providerToolName,
		choiceID: mapping.choiceID,
		toolName: mapping.toolName,
		toolDigest: mapping.toolDigest,
		approvalRule: mapping.approvalRule,
		executionMode: mapping.executionMode,
		appResourceUri: mapping.appResourceUri,
		visibility: mapping.visibility,
	};
}

function toolRequestToWails(request: InvokeMCPToolRequestBody): unknown {
	return omitUndefined({
		source: request.source,
		toolName: request.toolName,
		providerToolName: request.providerToolName,
		choiceID: request.choiceID,
		toolDigest: request.toolDigest,
		arguments: request.arguments,
		approvalID: request.approvalID,
		approvalToken: request.approvalToken,
		conversationID: request.conversationID,
		messageID: request.messageID,
		toolUseID: request.toolUseID,
		appInstanceID: request.appInstanceID,
	});
}

export class WailsMCPArtifactAPI implements IMCPAPI {
	async createMCPBundle(input: MCPCreateBundleInput): Promise<MCPBundle> {
		const result = await CreateMCPBundle({
			RootID: input.rootID,
			CollectionID: input.collectionID,
			SourceID: input.sourceID,
			SourceStorageKey: requireNonBlankString(input.sourceStorageKey, 'CreateMCPBundle.sourceStorageKey'),
			Document: documentToWails(input.document, 'CreateMCPBundle.document'),
			Registrations: input.registrations.map((item, index) =>
				registrationToWails(item, `CreateMCPBundle.registrations[${index}]`)
			),
		} as Parameters<typeof CreateMCPBundle>[0]);

		return bundleFromWails(result, 'CreateMCPBundle');
	}

	async getMCPBundle(bundle: ArtifactCollectionRef): Promise<MCPBundle> {
		return bundleFromWails(
			await GetMCPBundle(collectionRefToWails(bundle) as Parameters<typeof GetMCPBundle>[0]),
			'GetMCPBundle'
		);
	}

	async listMCPBundles(rootID: string): Promise<MCPBundle[]> {
		const values = requireWailsArray(await ListMCPBundles(rootID), 'ListMCPBundles');
		return values.map((value, index) => bundleFromWails(value, `ListMCPBundles[${index}]`));
	}

	async getMCPBundleDocument(bundle: ArtifactCollectionRef): Promise<MCPBundleDocument> {
		return bundleDocumentFromWails(
			await GetMCPBundleDocument(collectionRefToWails(bundle) as Parameters<typeof GetMCPBundleDocument>[0]),
			'GetMCPBundleDocument'
		);
	}

	async listMCPBundleServers(bundle: ArtifactCollectionRef): Promise<ArtifactRecord[]> {
		const values = requireWailsArray(
			await ListMCPBundleServers(collectionRefToWails(bundle) as Parameters<typeof ListMCPBundleServers>[0]),
			'ListMCPBundleServers'
		);
		return values.map((value, index) => artifactRecordFromWails(value, `ListMCPBundleServers[${index}]`));
	}

	async listMCPBundlePolicies(bundle: ArtifactCollectionRef): Promise<ArtifactRecord[]> {
		const values = requireWailsArray(
			await ListMCPBundlePolicies(collectionRefToWails(bundle) as Parameters<typeof ListMCPBundlePolicies>[0]),
			'ListMCPBundlePolicies'
		);
		return values.map((value, index) => artifactRecordFromWails(value, `ListMCPBundlePolicies[${index}]`));
	}

	async getMCPBundleInstallation(bundle: ArtifactCollectionRef): Promise<MCPBundleInstallation> {
		const value = requireWailsObject(
			await GetMCPBundleInstallation(collectionRefToWails(bundle) as Parameters<typeof GetMCPBundleInstallation>[0]),
			'GetMCPBundleInstallation'
		);
		return {
			bundle: collectionRefFromWails(value.bundle, 'GetMCPBundleInstallation.bundle'),
			builtIn: requireWailsBoolean(value.builtIn, 'GetMCPBundleInstallation.builtIn'),
			collectionRevision: requireWailsNumber(value.collectionRevision, 'GetMCPBundleInstallation.collectionRevision'),
			overlayRevision: requireWailsNumber(value.overlayRevision, 'GetMCPBundleInstallation.overlayRevision'),
			runtimeEnabled: requireWailsBoolean(value.runtimeEnabled, 'GetMCPBundleInstallation.runtimeEnabled'),
		};
	}

	async replaceMCPBundleDocument(input: MCPReplaceBundleDocumentInput): Promise<MCPBundle> {
		const result = await ReplaceMCPBundleDocument({
			Bundle: collectionRefToWails(input.bundle),
			ExpectedCollectionRevision: input.expectedCollectionRevision,
			Document: documentToWails(input.document, 'ReplaceMCPBundleDocument.document'),
			Registrations: input.registrations.map((item, index) =>
				registrationToWails(item, `ReplaceMCPBundleDocument.registrations[${index}]`)
			),
			AllowProtected: false,
		} as Parameters<typeof ReplaceMCPBundleDocument>[0]);

		return bundleFromWails(result, 'ReplaceMCPBundleDocument');
	}

	async refreshMCPBundle(bundle: ArtifactCollectionRef): Promise<MCPBundle> {
		return bundleFromWails(
			await RefreshMCPBundle(collectionRefToWails(bundle) as Parameters<typeof RefreshMCPBundle>[0]),
			'RefreshMCPBundle'
		);
	}

	async updateMCPBundleEnabled(
		bundle: ArtifactCollectionRef,
		expectedRevision: number,
		enabled: boolean
	): Promise<MCPBundle> {
		return bundleFromWails(
			await UpdateMCPBundleEnabled(
				collectionRefToWails(bundle) as Parameters<typeof UpdateMCPBundleEnabled>[0],
				expectedRevision,
				enabled
			),
			'UpdateMCPBundleEnabled'
		);
	}

	async updateProtectedMCPBundleInstallation(
		bundle: ArtifactCollectionRef,
		expectedOverlayRevision: number,
		runtimeEnabled: boolean
	): Promise<void> {
		await UpdateProtectedMCPBundleInstallation(
			collectionRefToWails(bundle) as Parameters<typeof UpdateProtectedMCPBundleInstallation>[0],
			expectedOverlayRevision,
			runtimeEnabled
		);
	}

	async retireMCPBundle(bundle: ArtifactCollectionRef, expectedRevision: number): Promise<ArtifactCollection> {
		return collectionFromWails(
			await RetireMCPBundle(collectionRefToWails(bundle) as Parameters<typeof RetireMCPBundle>[0], expectedRevision),
			'RetireMCPBundle'
		);
	}

	async purgeMCPBundle(bundle: ArtifactCollectionRef, expectedRevision: number): Promise<void> {
		await PurgeMCPBundle(collectionRefToWails(bundle) as Parameters<typeof PurgeMCPBundle>[0], expectedRevision);
	}

	async getMCPServerInstallation(server: ArtifactRef): Promise<MCPServerInstallation> {
		const value = requireWailsObject(
			await GetMCPServerInstallation(artifactRefToWails(server) as Parameters<typeof GetMCPServerInstallation>[0]),
			'GetMCPServerInstallation'
		);
		return {
			artifact: artifactRecordFromWails(value.artifact, 'GetMCPServerInstallation.artifact'),
			collection: collectionRefFromWails(value.collection, 'GetMCPServerInstallation.collection'),
			catalogRevision: requireWailsNumber(value.catalogRevision, 'GetMCPServerInstallation.catalogRevision'),
			document: serverDocumentFromWails(value.document, 'GetMCPServerInstallation.document'),
			installation: serverDataFromWails(value.installation, 'GetMCPServerInstallation.installation'),
			installationRevision: requireWailsNumber(
				value.installationRevision,
				'GetMCPServerInstallation.installationRevision'
			),
			runtimeEnabled: requireWailsBoolean(value.runtimeEnabled, 'GetMCPServerInstallation.runtimeEnabled'),
			builtIn: requireWailsBoolean(value.builtIn, 'GetMCPServerInstallation.builtIn'),
		};
	}

	async inspectMCPServer(server: ArtifactRef): Promise<MCPServerResolved> {
		const value = requireWailsObject(
			await InspectMCPServer(artifactRefToWails(server) as Parameters<typeof InspectMCPServer>[0]),
			'InspectMCPServer'
		);
		const policy = requireWailsObject(value.Policy, 'InspectMCPServer.Policy');

		return {
			server: artifactRefFromWails(value.Server, 'InspectMCPServer.Server'),
			collection: collectionRefFromWails(value.Collection, 'InspectMCPServer.Collection'),
			artifactRevision: requireWailsNumber(value.ArtifactRevision, 'InspectMCPServer.ArtifactRevision'),
			catalogRevision: requireWailsNumber(value.CatalogRevision, 'InspectMCPServer.CatalogRevision'),
			definitionDigest: requireWailsString(value.DefinitionDigest, 'InspectMCPServer.DefinitionDigest'),
			sourceContentDigest: requireWailsString(value.SourceContentDigest, 'InspectMCPServer.SourceContentDigest'),
			sourceGeneration: requireWailsString(value.SourceGeneration, 'InspectMCPServer.SourceGeneration'),
			document: serverDocumentFromWails(value.Document, 'InspectMCPServer.Document'),
			installation: serverDataFromWails(value.Installation, 'InspectMCPServer.Installation'),
			policy: {
				body: policyFromWails(policy.body, 'InspectMCPServer.Policy.body'),
				conflicts: policy.conflicts as Record<string, string> | undefined,
				digest: requireWailsString(policy.digest, 'InspectMCPServer.Policy.digest'),
			},
			installationRevision: requireWailsNumber(value.InstallationRevision, 'InspectMCPServer.InstallationRevision'),
			runtimeEnabled: requireWailsBoolean(value.RuntimeEnabled, 'InspectMCPServer.RuntimeEnabled'),
			builtIn: requireWailsBoolean(value.BuiltIn, 'InspectMCPServer.BuiltIn'),
			version: requireWailsString(value.Version, 'InspectMCPServer.Version'),
		};
	}

	async inspectMCPPolicy(policyRef: ArtifactRef): Promise<MCPPolicyView> {
		const value = requireWailsObject(
			await InspectMCPPolicy(artifactRefToWails(policyRef) as Parameters<typeof InspectMCPPolicy>[0]),
			'InspectMCPPolicy'
		);
		const definition = requireWailsObject(value.definition, 'InspectMCPPolicy.definition');

		return {
			artifact: artifactRecordFromWails(value.artifact, 'InspectMCPPolicy.artifact'),
			collection: collectionRefFromWails(value.collection, 'InspectMCPPolicy.collection'),
			catalogRevision: requireWailsNumber(value.catalogRevision, 'InspectMCPPolicy.catalogRevision'),
			definition: {
				...definition,
				body: rawJSONFromWails(definition.body, 'InspectMCPPolicy.definition.body'),
			} as ArtifactDefinitionView,
			body: policyFromWails(value.body, 'InspectMCPPolicy.body'),
			effectiveEnabled: requireWailsBoolean(value.effectiveEnabled, 'InspectMCPPolicy.effectiveEnabled'),
			builtIn: requireWailsBoolean(value.builtIn, 'InspectMCPPolicy.builtIn'),
		};
	}

	async updateMCPServerInstallation(
		server: ArtifactRef,
		expectedArtifactRevision: number,
		data: MCPServerData
	): Promise<ArtifactRecord> {
		return artifactRecordFromWails(
			await UpdateMCPServerInstallation(
				artifactRefToWails(server) as Parameters<typeof UpdateMCPServerInstallation>[0],
				expectedArtifactRevision,
				data as Parameters<typeof UpdateMCPServerInstallation>[2]
			),
			'UpdateMCPServerInstallation'
		);
	}

	async updateProtectedMCPServerInstallation(
		server: ArtifactRef,
		expectedOverlayRevision: number,
		runtimeEnabled: boolean,
		data: MCPServerData
	): Promise<void> {
		await UpdateProtectedMCPServerInstallation(
			artifactRefToWails(server) as Parameters<typeof UpdateProtectedMCPServerInstallation>[0],
			expectedOverlayRevision,
			runtimeEnabled,
			data as Parameters<typeof UpdateProtectedMCPServerInstallation>[3]
		);
	}

	async connectMCPServer(server: ArtifactRef): Promise<MCPServerRuntimeSnapshot> {
		return runtimeSnapshotFromWails(
			await ConnectMCPServer(artifactRefToWails(server) as Parameters<typeof ConnectMCPServer>[0]),
			'ConnectMCPServer'
		);
	}

	async disconnectMCPServer(server: ArtifactRef): Promise<void> {
		await DisconnectMCPServer(artifactRefToWails(server) as Parameters<typeof DisconnectMCPServer>[0]);
	}

	async refreshMCPServer(server: ArtifactRef): Promise<MCPServerRuntimeSnapshot> {
		return runtimeSnapshotFromWails(
			await RefreshMCPServer(artifactRefToWails(server) as Parameters<typeof RefreshMCPServer>[0]),
			'RefreshMCPServer'
		);
	}

	async getMCPServerStatus(server: ArtifactRef): Promise<MCPServerRuntimeSnapshot> {
		return runtimeSnapshotFromWails(
			await GetMCPServerStatus(artifactRefToWails(server) as Parameters<typeof GetMCPServerStatus>[0]),
			'GetMCPServerStatus'
		);
	}

	async listMCPServerTools(server: ArtifactRef): Promise<MCPToolCapability[]> {
		const values = requireWailsArray(
			await ListMCPServerTools(artifactRefToWails(server) as Parameters<typeof ListMCPServerTools>[0]),
			'ListMCPServerTools'
		);
		return values.map((value, index) => toolFromWails(value, `ListMCPServerTools[${index}]`));
	}

	async listMCPServerResources(server: ArtifactRef): Promise<MCPResourceRef[]> {
		const values = requireWailsArray(
			await ListMCPServerResources(artifactRefToWails(server) as Parameters<typeof ListMCPServerResources>[0]),
			'ListMCPServerResources'
		);
		return values.map((value, index) => {
			const item = requireWailsObject(value, `ListMCPServerResources[${index}]`);
			return {
				...item,
				server: artifactRefFromWails(item.server, `ListMCPServerResources[${index}].server`),
			} as MCPResourceRef;
		});
	}

	async listMCPServerResourceTemplates(server: ArtifactRef): Promise<MCPResourceTemplateRef[]> {
		const values = requireWailsArray(
			await ListMCPServerResourceTemplates(
				artifactRefToWails(server) as Parameters<typeof ListMCPServerResourceTemplates>[0]
			),
			'ListMCPServerResourceTemplates'
		);
		return values.map((value, index) => {
			const item = requireWailsObject(value, `ListMCPServerResourceTemplates[${index}]`);
			return {
				...item,
				server: artifactRefFromWails(item.server, `ListMCPServerResourceTemplates[${index}].server`),
			} as MCPResourceTemplateRef;
		});
	}

	async listMCPServerPrompts(server: ArtifactRef): Promise<MCPPromptRef[]> {
		const values = requireWailsArray(
			await ListMCPServerPrompts(artifactRefToWails(server) as Parameters<typeof ListMCPServerPrompts>[0]),
			'ListMCPServerPrompts'
		);
		return values.map((value, index) => {
			const item = requireWailsObject(value, `ListMCPServerPrompts[${index}]`);
			return {
				...item,
				server: artifactRefFromWails(item.server, `ListMCPServerPrompts[${index}].server`),
			} as MCPPromptRef;
		});
	}

	async readMCPResource(server: ArtifactRef, uri: string): Promise<MCPReadResourceResponseBody> {
		const value = requireWailsObject(
			await ReadMCPResource(artifactRefToWails(server) as Parameters<typeof ReadMCPResource>[0], uri),
			'ReadMCPResource'
		);
		return {
			...value,
			server: artifactRefFromWails(value.server, 'ReadMCPResource.server'),
		} as MCPReadResourceResponseBody;
	}

	async getMCPPrompt(
		server: ArtifactRef,
		promptName: string,
		promptArguments?: Record<string, string>
	): Promise<MCPGetPromptResponseBody> {
		const value = requireWailsObject(
			await GetMCPPrompt(
				artifactRefToWails(server) as Parameters<typeof GetMCPPrompt>[0],
				promptName,
				promptArguments ?? {}
			),
			'GetMCPPrompt'
		);
		return {
			...value,
			server: artifactRefFromWails(value.server, 'GetMCPPrompt.server'),
			messages: optionalWailsArray(value.messages, 'GetMCPPrompt.messages')?.map((message, index) => {
				const item = requireWailsObject(message, `GetMCPPrompt.messages[${index}]`);
				return {
					...item,
					role: enumFromWails(item.role, MCPPromptRole, `GetMCPPrompt.messages[${index}].role`),
				};
			}),
		} as MCPGetPromptResponseBody;
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
			artifactRefToWails(server) as Parameters<typeof CompleteMCPArgument>[0],
			omitUndefined({
				refType,
				name,
				argumentName,
				argumentValue,
				context,
			}) as Parameters<typeof CompleteMCPArgument>[1]
		)) as MCPCompletionResult;
	}

	async evaluateMCPToolCall(server: ArtifactRef, request: InvokeMCPToolRequestBody): Promise<MCPApprovalEvaluation> {
		const value = requireWailsObject(
			await EvaluateMCPToolCall(
				artifactRefToWails(server) as Parameters<typeof EvaluateMCPToolCall>[0],
				toolRequestToWails(request) as Parameters<typeof EvaluateMCPToolCall>[1]
			),
			'EvaluateMCPToolCall'
		);
		return {
			...value,
			decision: enumFromWails(value.decision, MCPApprovalDecision, 'EvaluateMCPToolCall.decision'),
		} as MCPApprovalEvaluation;
	}

	async evaluateMappedMCPToolCall(
		mapping: MCPProviderToolMapping,
		request: InvokeMCPToolRequestBody
	): Promise<MCPApprovalEvaluation> {
		const value = requireWailsObject(
			await EvaluateMappedMCPToolCall(
				providerMappingToWails(mapping) as Parameters<typeof EvaluateMappedMCPToolCall>[0],
				toolRequestToWails(request) as Parameters<typeof EvaluateMappedMCPToolCall>[1]
			),
			'EvaluateMappedMCPToolCall'
		);
		return {
			...value,
			decision: enumFromWails(value.decision, MCPApprovalDecision, 'EvaluateMappedMCPToolCall.decision'),
		} as MCPApprovalEvaluation;
	}

	async invokeMCPTool(server: ArtifactRef, request: InvokeMCPToolRequestBody): Promise<InvokeMCPToolResponseBody> {
		const value = requireWailsObject(
			await InvokeMCPTool(
				artifactRefToWails(server) as Parameters<typeof InvokeMCPTool>[0],
				toolRequestToWails(request) as Parameters<typeof InvokeMCPTool>[1]
			),
			'InvokeMCPTool'
		);
		return {
			...value,
			server: artifactRefFromWails(value.server, 'InvokeMCPTool.server'),
		} as InvokeMCPToolResponseBody;
	}

	async invokeMappedMCPTool(
		mapping: MCPProviderToolMapping,
		request: InvokeMCPToolRequestBody
	): Promise<InvokeMCPToolResponseBody> {
		const value = requireWailsObject(
			await InvokeMappedMCPTool(
				providerMappingToWails(mapping) as Parameters<typeof InvokeMappedMCPTool>[0],
				toolRequestToWails(request) as Parameters<typeof InvokeMappedMCPTool>[1]
			),
			'InvokeMappedMCPTool'
		);
		return {
			...value,
			server: artifactRefFromWails(value.server, 'InvokeMappedMCPTool.server'),
		} as InvokeMCPToolResponseBody;
	}

	async resolveMCPApproval(approvalID: string, resolution: MCPApprovalResolution): Promise<MCPApprovalToken> {
		return (await ResolveMCPApproval(approvalID, resolution)) as MCPApprovalToken;
	}

	async getMCPServerAuthHealth(server: ArtifactRef): Promise<MCPAuthHealth> {
		const value = requireWailsObject(
			await GetMCPServerAuthHealth(artifactRefToWails(server) as Parameters<typeof GetMCPServerAuthHealth>[0]),
			'GetMCPServerAuthHealth'
		);
		return {
			...value,
			server: artifactRefFromWails(value.server, 'GetMCPServerAuthHealth.server'),
			authMode: enumFromWails(value.authMode, MCPHTTPAuthMode, 'GetMCPServerAuthHealth.authMode'),
			state: enumFromWails(value.state, MCPAuthHealthState, 'GetMCPServerAuthHealth.state'),
			expiresAt:
				value.expiresAt === undefined
					? undefined
					: toFrontendTimestamp(value.expiresAt, 'GetMCPServerAuthHealth.expiresAt'),
		} as MCPAuthHealth;
	}

	async listPendingMCPOAuthAuthorizations(): Promise<MCPOAuthAuthorization[]> {
		const values = requireWailsArray(await ListPendingMCPOAuthAuthorizations(), 'ListPendingMCPOAuthAuthorizations');
		return values.map((value, index) => {
			const item = requireWailsObject(value, `ListPendingMCPOAuthAuthorizations[${index}]`);
			return {
				server: artifactRefFromWails(item.server, `ListPendingMCPOAuthAuthorizations[${index}].server`),
				authorizationURL: requireWailsString(
					item.authorizationURL,
					`ListPendingMCPOAuthAuthorizations[${index}].authorizationURL`
				),
				expiresAt: optionalWailsString(item.expiresAt, `ListPendingMCPOAuthAuthorizations[${index}].expiresAt`),
			};
		});
	}

	async cancelPendingMCPOAuthAuthorization(server: ArtifactRef): Promise<boolean> {
		return CancelPendingMCPOAuthAuthorization(
			artifactRefToWails(server) as Parameters<typeof CancelPendingMCPOAuthAuthorization>[0]
		);
	}

	async putMCPServerSecret(
		server: ArtifactRef,
		kind: MCPSecretKind,
		slot: string,
		secret: string
	): Promise<MCPSecretWriteResult> {
		const value = requireWailsObject(
			await PutMCPServerSecret(
				artifactRefToWails(server) as Parameters<typeof PutMCPServerSecret>[0],
				kind,
				slot,
				secret
			),
			'PutMCPServerSecret'
		);
		return {
			secretRef: requireWailsString(value.secretRef, 'PutMCPServerSecret.secretRef'),
			sha256: optionalWailsString(value.sha256, 'PutMCPServerSecret.sha256'),
			nonEmpty: requireWailsBoolean(value.nonEmpty, 'PutMCPServerSecret.nonEmpty'),
		};
	}

	async deleteMCPServerSecret(server: ArtifactRef, kind: MCPSecretKind, slot: string): Promise<void> {
		await DeleteMCPServerSecret(artifactRefToWails(server) as Parameters<typeof DeleteMCPServerSecret>[0], kind, slot);
	}

	async getMCPGlobalSettings(): Promise<MCPGlobalSettings> {
		const value = requireWailsObject(await GetMCPGlobalSettings(), 'GetMCPGlobalSettings');
		const settings = requireWailsObject(value.settings, 'GetMCPGlobalSettings.settings');

		return {
			settings: {
				oauthLoopbackListenAddr: optionalWailsString(
					settings.oauthLoopbackListenAddr,
					'GetMCPGlobalSettings.settings.oauthLoopbackListenAddr'
				),
			},
			revision: requireWailsNumber(value.revision, 'GetMCPGlobalSettings.revision'),
			oauthRedirectURL: optionalWailsString(value.oauthRedirectURL, 'GetMCPGlobalSettings.oauthRedirectURL'),
			oauthLoopbackListenAddr: optionalWailsString(
				value.oauthLoopbackListenAddr,
				'GetMCPGlobalSettings.oauthLoopbackListenAddr'
			),
			oauthRestartRequired: requireWailsBoolean(
				value.oauthRestartRequired,
				'GetMCPGlobalSettings.oauthRestartRequired'
			),
		};
	}

	async updateMCPGlobalSettings(expectedRevision: number, oauthLoopbackListenAddr?: string): Promise<number> {
		return UpdateMCPGlobalSettings(expectedRevision, {
			oauthLoopbackListenAddr,
		} as Parameters<typeof UpdateMCPGlobalSettings>[1]);
	}
}
