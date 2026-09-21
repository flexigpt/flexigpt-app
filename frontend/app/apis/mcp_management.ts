// oxlint-disable typescript/parameter-properties
import type { ArtifactRef, ArtifactRootID, MappedTarget, StoreArtifact } from '@/spec/artifact';
import type { CollectionView } from '@/spec/collection';
import type {
	InvokeMCPToolRequestBody,
	ManagedMCPPolicyUpsertRequest,
	ManagedMCPPolicyUpsertResult,
	MCPApprovalEvaluation,
	MCPApprovalResolution,
	MCPApprovalResolutionResult,
	MCPAuthenticationDeclaration,
	MCPAuthHealth,
	MCPAuthHealthState,
	MCPAuthSettings,
	MCPBundleView,
	MCPCollectionManagementView,
	MCPCompleteArgumentRequestBody,
	MCPCompletionResult,
	MCPDiscoveryPage,
	MCPEffectivePolicy,
	MCPGetPromptResponseBody,
	MCPGlobalSettings,
	MCPHTTPAuthMode,
	MCPHTTPSecretDraft,
	MCPInputBinding,
	MCPOAuthAuthorization,
	MCPPolicy,
	MCPPolicyManagementView,
	MCPPromptRef,
	MCPProviderToolMapping,
	MCPReadResourceResponseBody,
	MCPResourceRef,
	MCPResourceTemplateRef,
	MCPRuntimeCatalogID,
	MCPRuntimeInvokeToolResponse,
	MCPRuntimeServerID,
	MCPSecretKind,
	MCPSecretWriteResult,
	MCPServerData,
	MCPServerDocument,
	MCPServerDraft,
	MCPServerManagementView,
	MCPServerRuntimeSnapshot,
	MCPServerView,
	MCPSetupInputView,
	MCPSetupSecretTarget,
	MCPSetupSubmissionValue,
	MCPStdioSecretDraft,
	MCPStorePolicyView,
	MCPStoreServerInstallationView,
	MCPToolCapability,
} from '@/spec/mcp';
import type { ModelPresetRef } from '@/spec/modelpreset';
import type { ToolRef } from '@/spec/tool';
import { ArtifactState } from '@/spec/artifact';
import {
	MCP_SCHEMA_VERSION,
	MCPApprovalRule,
	MCPAuthHealthState as MCPAuthHealthStateValue,
	MCPExecutionMode,
	MCPHTTPAuthMode as MCPHTTPAuthModeValue,
	MCPInputKind as MCPInputKindValue,
	MCPSecretKind as MCPSecretKindValue,
	MCPServerType as MCPServerTypeValue,
	MCPTransportType as MCPTransportTypeValue,
	MCPTrustLevel as MCPTrustLevelValue,
} from '@/spec/mcp';

import { omitManyKeys } from '@/lib/obj_utils';

import type {
	IMCPAggregateAPI,
	IMCPRuntimeAPI,
	IMCPStoreAPI,
	IModelPresetStoreAPI,
	IToolStoreAPI,
} from '@/apis/interface';

const MANAGEMENT_PAGE_SIZE = 100;
const MAX_MANAGEMENT_PAGE_HOPS = 10_000;
const PLACEHOLDER_PATTERN = /\$\{([A-Za-z_][A-Za-z0-9_]*)\}/g;

interface MCPSecretTarget {
	kind: MCPSecretKind;
	slot: string;
}

interface PlannedSecretWrite {
	inputName: string;
	kind: MCPSecretKind;
	slot: string;
	secret: string;
}

interface ServerDocumentBuild {
	document: MCPServerDocument;
	preReplaceData: MCPServerData;
	secretWrites: PlannedSecretWrite[];
}

function cloneJSON<T>(value: T): T {
	return JSON.parse(JSON.stringify(value)) as T;
}

function artifactRefKey(value: ArtifactRef): string {
	return `${value.rootID}:${value.artifactID}`;
}

function getErrorMessage(error: unknown, fallback: string): string {
	if (error instanceof Error && error.message.trim()) {
		return error.message;
	}
	return fallback;
}

function sameJSON(left: unknown, right: unknown): boolean {
	return JSON.stringify(left) === JSON.stringify(right);
}

function normalizeInputName(value: string): string {
	const normalized = value
		.trim()
		.replaceAll(/[^A-Za-z0-9_]/g, '_')
		.replaceAll(/_+/g, '_')
		.replaceAll(/^_+|_+$/g, '');

	return normalized ? `mcp_${normalized}`.slice(0, 96) : 'mcp_input';
}

function uniqueInputName(base: string, reserved: Set<string>): string {
	let candidate = base;
	let index = 2;

	while (reserved.has(candidate)) {
		candidate = `${base}_${index}`;
		index += 1;
	}

	reserved.add(candidate);
	return candidate;
}

function placeholderNames(value: string): string[] {
	const names = new Set<string>();

	for (const match of value.matchAll(PLACEHOLDER_PATTERN)) {
		if (match[1]) {
			names.add(match[1]);
		}
	}

	return [...names];
}

function sameSecretSlot(left: string, right: string): boolean {
	return left.trim().toLocaleLowerCase() === right.trim().toLocaleLowerCase();
}

function defaultPolicy(): MCPPolicy {
	return {
		trustLevel: MCPTrustLevelValue.Untrusted,
		defaultPolicy: {
			defaultApprovalRule: MCPApprovalRule.Ask,
			defaultExecutionMode: MCPExecutionMode.Manual,
			requireApprovalForUnknownRisk: true,
			requireApprovalForWrite: true,
			requireApprovalForDestructive: true,
		},
		toolPolicies: {},
		appsPolicy: {
			enabled: false,
			allowAppInitiatedToolCalls: false,
			requireApprovalForOpenLink: true,
			requireApprovalForContextUpdates: true,
		},
	};
}

function defaultServerData(): MCPServerData {
	return {
		schemaVersion: MCP_SCHEMA_VERSION,
		inputs: {},
	};
}

function findSecretTargets(document: MCPServerDocument): Map<string, MCPSecretTarget> {
	const targets = new Map<string, MCPSecretTarget>();
	const declarations = document.configuration.install.inputs ?? {};

	const register = (inputName: string, kind: MCPSecretKind, slot: string) => {
		const declaration = declarations[inputName];

		if (!declaration || declaration.kind !== MCPInputKindValue.Secret) {
			return;
		}

		const existing = targets.get(inputName);
		if (existing && (existing.kind !== kind || !sameSecretSlot(existing.slot, slot))) {
			throw new Error(`Secret input "${inputName}" has more than one materialization target.`);
		}

		targets.set(inputName, {
			kind,
			slot,
		});
	};

	for (const [name, value] of Object.entries(document.mcpServer.env ?? {})) {
		for (const inputName of placeholderNames(value)) {
			register(inputName, MCPSecretKindValue.StdioEnv, name);
		}
	}

	for (const [name, value] of Object.entries(document.mcpServer.headers ?? {})) {
		for (const inputName of placeholderNames(value)) {
			register(inputName, MCPSecretKindValue.HTTPHeader, name);
		}
	}

	return targets;
}

function inputBindingFor(data: MCPServerData | undefined, inputName: string): MCPInputBinding | undefined {
	return data?.inputs?.[inputName];
}

function extractHeaderAffixes(value: string, inputName: string): { prefix: string; suffix: string } {
	const placeholder = `\${${inputName}}`;
	const index = value.indexOf(placeholder);

	if (index < 0) {
		return {
			prefix: '',
			suffix: '',
		};
	}

	return {
		prefix: value.slice(0, index),
		suffix: value.slice(index + placeholder.length),
	};
}

function buildServerDocument(
	existing: MCPServerView | undefined,
	draft: MCPServerDraft,
	policyName: string
): ServerDocumentBuild {
	const previous = existing?.document;
	const previousData = cloneJSON(existing?.installation ?? defaultServerData());
	const previousTargets = previous ? findSecretTargets(previous) : new Map<string, MCPSecretTarget>();
	const previousOAuthInput = previous?.configuration.auth.clientCredentialsInput;
	const previousInputs = cloneJSON(previous?.configuration.install.inputs ?? {});
	const nextData = cloneJSON(previousData);
	let nextInputs = previousInputs;
	let nextBindings = cloneJSON(nextData.inputs ?? {});

	nextData.schemaVersion = MCP_SCHEMA_VERSION;

	const controlledInputNames = new Set<string>([
		...previousTargets.keys(),
		...(previousOAuthInput ? [previousOAuthInput] : []),
	]);

	for (const inputName of controlledInputNames) {
		nextInputs = omitManyKeys(nextInputs, [inputName]);
		nextBindings = omitManyKeys(nextBindings, [inputName]);
	}

	const reservedNames = new Set(Object.keys(nextInputs));
	const secretWrites: PlannedSecretWrite[] = [];

	const env = cloneJSON(draft.stdioEnv);
	const headers = cloneJSON(draft.httpHeaders);

	const mcpServer: MCPServerDocument['mcpServer'] =
		draft.transport === MCPTransportTypeValue.Stdio
			? {
					type: MCPServerTypeValue.Stdio,
					command: draft.stdioCommand.trim(),
					args: draft.stdioArgs.filter(Boolean),
					env,
				}
			: {
					type: MCPServerTypeValue.HTTP,
					url: draft.httpURL.trim(),
					headers,
				};

	if (draft.transport === MCPTransportTypeValue.Stdio) {
		for (const row of draft.stdioSecrets) {
			const envName = row.envName.trim();
			if (!envName) {
				continue;
			}

			const inputName =
				row.inputName && !reservedNames.has(row.inputName)
					? row.inputName
					: uniqueInputName(normalizeInputName(`stdio_${envName}`), reservedNames);

			reservedNames.add(inputName);
			nextInputs[inputName] = {
				kind: MCPInputKindValue.Secret,
				label: envName,
				required: true,
			};
			env[envName] = `\${${inputName}}`;

			const oldTarget = row.inputName ? previousTargets.get(row.inputName) : undefined;
			const oldBinding = row.inputName ? previousData.inputs?.[row.inputName] : undefined;

			if (
				oldBinding?.secretRef &&
				oldTarget?.kind === MCPSecretKindValue.StdioEnv &&
				sameSecretSlot(oldTarget.slot, envName) &&
				!row.secretValue &&
				!row.deleteExisting
			) {
				nextBindings[inputName] = oldBinding;
			}

			if (row.secretValue) {
				secretWrites.push({
					inputName,
					kind: MCPSecretKindValue.StdioEnv,
					slot: envName,
					secret: row.secretValue,
				});
			}
		}
	}

	if (draft.transport === MCPTransportTypeValue.StreamableHTTP && draft.httpAuthMode === MCPHTTPAuthModeValue.APIKey) {
		const apiKey = draft.httpAPIKey;
		if (apiKey) {
			const headerName = apiKey.headerName.trim();
			const inputName =
				apiKey.inputName && !reservedNames.has(apiKey.inputName)
					? apiKey.inputName
					: uniqueInputName(normalizeInputName(`http_${headerName}`), reservedNames);

			reservedNames.add(inputName);
			nextInputs[inputName] = {
				kind: MCPInputKindValue.Secret,
				label: headerName,
				required: true,
			};
			headers[headerName] = `${apiKey.valuePrefix}\${${inputName}}${apiKey.valueSuffix}`;

			const oldTarget = apiKey.inputName ? previousTargets.get(apiKey.inputName) : undefined;
			const oldBinding = apiKey.inputName ? previousData.inputs?.[apiKey.inputName] : undefined;

			if (
				oldBinding?.secretRef &&
				oldTarget?.kind === MCPSecretKindValue.HTTPHeader &&
				sameSecretSlot(oldTarget.slot, headerName) &&
				!apiKey.secretValue &&
				!apiKey.deleteExisting
			) {
				nextBindings[inputName] = oldBinding;
			}

			if (apiKey.secretValue) {
				secretWrites.push({
					inputName,
					kind: MCPSecretKindValue.HTTPHeader,
					slot: headerName,
					secret: apiKey.secretValue,
				});
			}
		}
	}

	const useOAuthCredentials =
		draft.httpAuthMode === MCPHTTPAuthModeValue.ClientCredentials ||
		(draft.httpAuthMode === MCPHTTPAuthModeValue.OAuth && draft.httpOAuthClientCredentials.useClientCredentials);

	const auth: MCPAuthenticationDeclaration = {
		mode: draft.transport === MCPTransportTypeValue.Stdio ? MCPHTTPAuthModeValue.None : draft.httpAuthMode,
		clientIDMetadataDocumentURL:
			draft.httpAuthMode === MCPHTTPAuthModeValue.OAuth
				? draft.httpClientIDMetadataDocumentURL.trim() || undefined
				: undefined,
	};

	if (useOAuthCredentials) {
		const oauth = draft.httpOAuthClientCredentials;
		const inputName =
			oauth.inputName && !reservedNames.has(oauth.inputName)
				? oauth.inputName
				: uniqueInputName('mcp_oauth_client_credentials', reservedNames);

		reservedNames.add(inputName);
		nextInputs[inputName] = {
			kind: MCPInputKindValue.OAuthClientCredentials,
			label: 'OAuth client credentials',
			required: draft.httpAuthMode === MCPHTTPAuthModeValue.ClientCredentials,
			clientSecretRequired: draft.httpAuthMode === MCPHTTPAuthModeValue.ClientCredentials,
		};
		auth.clientCredentialsInput = inputName;

		const oldBinding = oauth.inputName ? previousData.inputs?.[oauth.inputName] : undefined;
		if (oldBinding?.secretRef && !oauth.secretJSON.trim() && !oauth.deleteExisting) {
			nextBindings[inputName] = oldBinding;
		}

		if (oauth.secretJSON.trim()) {
			secretWrites.push({
				inputName,
				kind: MCPSecretKindValue.OAuthClientCredentials,
				slot: 'clientCredentials',
				secret: oauth.secretJSON.trim(),
			});
		}
	}

	nextData.inputs = nextBindings;

	return {
		document: {
			logicalName: previous?.logicalName ?? draft.logicalName.trim(),
			logicalVersion: previous?.logicalVersion,
			displayName: draft.displayName.trim(),
			description: previous?.description,
			labels: cloneJSON(previous?.labels ?? {}),
			mcpServer,
			include: previous?.include ? cloneJSON(previous.include) : undefined,
			configuration: {
				timeoutMS: draft.transport === MCPTransportTypeValue.Stdio ? draft.stdioStartupTimeoutMS : draft.httpTimeoutMS,
				auth,
				install: {
					note: previous?.configuration.install.note,
					inputs: nextInputs,
					allowEnvironment: cloneJSON(previous?.configuration.install.allowEnvironment ?? []),
				},
				connectionProfiles: cloneJSON(previous?.configuration.connectionProfiles ?? {}),
				policy: {
					name: policyName,
					required: true,
				},
			},
		},
		preReplaceData: nextData,
		secretWrites,
	};
}

export function serverDisplayName(server: MCPServerView): string {
	return server.document?.displayName || server.artifact.displayName || server.logicalName;
}

export function getAuthMode(server: MCPServerView): MCPHTTPAuthMode {
	if (server.document?.mcpServer.type === MCPServerTypeValue.Stdio) {
		return MCPHTTPAuthModeValue.None;
	}
	return server.document?.configuration.auth.mode ?? MCPHTTPAuthModeValue.None;
}

export function getServerAuthHealthState(
	server: MCPServerView,
	health?: MCPAuthHealth
): MCPAuthHealthState | undefined {
	if (getAuthMode(server) === MCPHTTPAuthModeValue.None) {
		return MCPAuthHealthStateValue.NotRequired;
	}
	return health?.state;
}

export function isServerOperational(server: MCPServerView): boolean {
	return Boolean(server.document && server.installation && server.artifact.state === ArtifactState.Available);
}

export function requireMCPRuntimeServerID(server: MCPServerView): MCPRuntimeServerID {
	if (!server.runtimeServerID) {
		throw new Error(`MCP runtime identity is unavailable for "${serverDisplayName(server)}".`);
	}
	return server.runtimeServerID;
}

export function serverRefLabel(server: MCPServerView): string {
	return `${server.ref.rootID}/${server.ref.artifactID}`;
}

export function serverSetupInputs(server: MCPServerView): MCPSetupInputView[] {
	if (!server.document) {
		return [];
	}

	const targets = findSecretTargets(server.document);
	const inputs = server.document.configuration.install.inputs ?? {};

	return Object.entries(inputs)
		.map(([name, declaration]) => {
			const target: MCPSetupSecretTarget | undefined =
				declaration.kind === MCPInputKindValue.OAuthClientCredentials
					? {
							kind: MCPSecretKindValue.OAuthClientCredentials,
							slot: 'clientCredentials',
						}
					: targets.get(name);

			const binding = inputBindingFor(server.installation, name);

			return {
				name,
				declaration,
				target,
				boundValue: binding?.value,
				boundSecretRef: binding?.secretRef,
			};
		})
		.toSorted((left, right) => left.name.localeCompare(right.name));
}

export function getMCPServerSetupStatus(server: MCPServerView): {
	hasInputs: boolean;
	requiredTotal: number;
	requiredConfigured: number;
	complete: boolean;
} {
	const inputs = serverSetupInputs(server);
	const required = inputs.filter(input => input.declaration.required);
	const configured = required.filter(input => {
		if (input.declaration.kind === MCPInputKindValue.Text || input.declaration.kind === MCPInputKindValue.Path) {
			return Boolean(input.boundValue?.trim() || input.declaration.default?.trim());
		}
		return Boolean(input.boundSecretRef?.trim());
	});

	return {
		hasInputs: inputs.length > 0,
		requiredTotal: required.length,
		requiredConfigured: configured.length,
		complete: configured.length === required.length,
	};
}

export function serverDraftFromView(server?: MCPServerView): MCPServerDraft {
	const document = server?.document;
	const policy = server?.policy?.body ?? defaultPolicy();
	const targets = document ? findSecretTargets(document) : new Map<string, MCPSecretTarget>();
	let stdioEnv = cloneJSON(document?.mcpServer.env ?? {});
	let httpHeaders = cloneJSON(document?.mcpServer.headers ?? {});
	const stdioSecrets: MCPStdioSecretDraft[] = [];
	let httpAPIKey: MCPHTTPSecretDraft | undefined;

	for (const [inputName, target] of targets) {
		const binding = inputBindingFor(server?.installation, inputName);

		if (target.kind === MCPSecretKindValue.StdioEnv) {
			stdioEnv = omitManyKeys(stdioEnv, [target.slot]);
			stdioSecrets.push({
				inputName,
				envName: target.slot,
				existingSecretRef: binding?.secretRef,
				secretValue: '',
				deleteExisting: false,
			});
			continue;
		}

		const headerValue = httpHeaders[target.slot] ?? '';
		const affixes = extractHeaderAffixes(headerValue, inputName);
		httpHeaders = omitManyKeys(httpHeaders, [target.slot]);
		httpAPIKey = {
			inputName,
			headerName: target.slot,
			valuePrefix: affixes.prefix,
			valueSuffix: affixes.suffix,
			existingSecretRef: binding?.secretRef,
			secretValue: '',
			deleteExisting: false,
		};
	}

	const oauthInputName = document?.configuration.auth.clientCredentialsInput;
	const oauthBinding = oauthInputName ? inputBindingFor(server?.installation, oauthInputName) : undefined;

	return {
		logicalName: document?.logicalName ?? '',
		displayName: document?.displayName ?? '',
		enabled: server?.enabled ?? true,
		transport:
			document?.mcpServer.type === MCPServerTypeValue.Stdio
				? MCPTransportTypeValue.Stdio
				: MCPTransportTypeValue.StreamableHTTP,
		trustLevel: policy.trustLevel,

		stdioCommand: document?.mcpServer.command ?? '',
		stdioArgs: [...(document?.mcpServer.args ?? [])],
		stdioEnv,
		stdioStartupTimeoutMS: document?.configuration.timeoutMS,
		stdioSecrets,

		httpURL: document?.mcpServer.url ?? '',
		httpHeaders,
		httpTimeoutMS: document?.configuration.timeoutMS,
		httpAuthMode: document?.configuration.auth.mode ?? MCPHTTPAuthModeValue.None,
		httpAPIKey,
		httpOAuthClientCredentials: {
			inputName: oauthInputName,
			existingSecretRef: oauthBinding?.secretRef,
			secretJSON: '',
			deleteExisting: false,
			useClientCredentials: Boolean(oauthInputName),
		},
		httpClientIDMetadataDocumentURL: document?.configuration.auth.clientIDMetadataDocumentURL ?? '',

		defaultPolicy: cloneJSON(policy.defaultPolicy),
		toolPolicies: cloneJSON(policy.toolPolicies ?? {}),
		appsPolicy: cloneJSON(policy.appsPolicy),
	};
}

export class MCPManagementAPI {
	constructor(
		private readonly store: IMCPStoreAPI,
		private readonly aggregate: IMCPAggregateAPI,
		private readonly runtime: IMCPRuntimeAPI,
		private readonly toolStore: IToolStoreAPI,
		private readonly modelPresetStore: IModelPresetStoreAPI
	) {}

	async listMCPCollectionsForManagement(): Promise<CollectionView[]> {
		return this.collectPages(pageToken => this.store.listMCPCollectionsPage(MANAGEMENT_PAGE_SIZE, pageToken));
	}

	async listMCPServersForManagement(): Promise<StoreArtifact[]> {
		return this.collectPages(pageToken => this.store.listMCPServersPage(MANAGEMENT_PAGE_SIZE, pageToken));
	}

	async listMCPBundles(): Promise<MCPBundleView[]> {
		return (await this.listMCPCollectionsForManagement())
			.map(value => this.toBundleView(value))
			.toSorted((left, right) => {
				if (left.builtIn !== right.builtIn) {
					return left.builtIn ? -1 : 1;
				}
				return left.displayName.localeCompare(right.displayName, undefined, {
					sensitivity: 'base',
				});
			});
	}

	async getMCPBundle(collection: ArtifactRef): Promise<MCPBundleView> {
		return this.toBundleView(await this.store.getMCPCollection(collection));
	}

	async listMCPServers(bundle: MCPBundleView): Promise<MCPServerView[]> {
		const collection = await this.getMCPCollectionManagementView(bundle.ref);
		const serverRefs = new Map<string, ArtifactRef>();
		const policyRefsByName = new Map<string, ArtifactRef>();

		for (const occurrence of collection.capabilities.occurrences) {
			if (occurrence.status !== 'available' || !occurrence.artifact) {
				continue;
			}
			if (occurrence.type === 'mcp') {
				serverRefs.set(artifactRefKey(occurrence.artifact), occurrence.artifact);
			}
			if (occurrence.type === 'mcp.policy' && occurrence.name) {
				policyRefsByName.set(occurrence.name, occurrence.artifact);
			}
		}

		const values = await Promise.all(
			[...serverRefs.values()].map(async ref => {
				const management = await this.getMCPServerManagementView(ref);
				const policyName = management.installation.document.configuration.policy?.name;

				return this.toServerView(management, bundle.ref, policyName ? policyRefsByName.get(policyName) : undefined);
			})
		);

		return values.toSorted((left, right) =>
			serverDisplayName(left).localeCompare(serverDisplayName(right), undefined, {
				sensitivity: 'base',
			})
		);
	}

	async createMCPBundle(logicalName: string, displayName: string, description?: string): Promise<MCPBundleView> {
		const collections = await this.listMCPCollectionsForManagement();
		const roots = new Set(
			collections
				.filter(value => value.baseline && value.editable && !value.deletable)
				.map(value => value.artifact.rootID)
		);

		if (roots.size !== 1) {
			throw new Error('Expected exactly one editable MCP baseline Collection.');
		}

		const collection = await this.store.createMCPCollection({
			rootID: [...roots][0],
			name: logicalName,
			displayName,
			description,
		});
		return this.toBundleView(collection);
	}

	async saveMCPServer(
		bundle: MCPBundleView,
		existing: MCPServerView | undefined,
		draft: MCPServerDraft
	): Promise<MCPServerView> {
		if (bundle.builtIn || !bundle.editable || existing?.builtIn) {
			throw new Error('Built-in MCP servers cannot be edited.');
		}

		const policyName = existing?.document?.configuration.policy?.name ?? draft.logicalName.trim();
		const built = buildServerDocument(existing, draft, policyName);
		let artifactRevision = existing?.artifact.revision;

		if (existing && !sameJSON(existing.installation ?? defaultServerData(), built.preReplaceData)) {
			const updated = await this.aggregate.updateMCPServerInstallation(
				existing.ref,
				existing.artifact.revision,
				built.preReplaceData
			);
			artifactRevision = updated.revision;
		}

		const policy = await this.aggregate.upsertManagedMCPPolicy({
			collection: bundle.ref,
			expectedCollectionRevision: bundle.collection.artifact.revision,
			name: policyName,
			description: `${draft.displayName.trim()} policy`,
			policy: {
				trustLevel: draft.trustLevel,
				defaultPolicy: cloneJSON(draft.defaultPolicy),
				toolPolicies: cloneJSON(draft.toolPolicies),
				appsPolicy: cloneJSON(draft.appsPolicy),
			},
			enabled: true,
		});

		const result = existing
			? await this.aggregate.replaceManagedMCP({
					collection: bundle.ref,
					expectedCollectionRevision: policy.collection.artifact.revision,
					artifact: existing.ref,
					expectedArtifactRevision: artifactRevision ?? existing.artifact.revision,
					document: built.document,
					enabled: draft.enabled,
				})
			: await this.aggregate.createManagedMCP({
					collection: bundle.ref,
					expectedCollectionRevision: policy.collection.artifact.revision,
					document: built.document,
					enabled: draft.enabled,
				});

		const serverRef: ArtifactRef = {
			rootID: result.artifact.rootID,
			artifactID: result.artifact.id,
		};
		const installation = await this.store.getMCPServerInstallation(serverRef);
		const nextData = cloneJSON(installation.installation);
		nextData.inputs = cloneJSON(nextData.inputs ?? {});

		for (const write of built.secretWrites) {
			const value = await this.aggregate.putMCPServerSecret(serverRef, write.kind, write.slot, write.secret);
			nextData.inputs[write.inputName] = {
				secretRef: value.secretRef,
			};
		}

		if (!sameJSON(installation.installation, nextData)) {
			await this.aggregate.updateMCPServerInstallation(serverRef, installation.artifact.revision, nextData);
		}

		const refreshedBundle = await this.getMCPBundle(bundle.ref);
		const refreshed = (await this.listMCPServers(refreshedBundle)).find(
			value => artifactRefKey(value.ref) === artifactRefKey(serverRef)
		);

		if (!refreshed) {
			throw new Error('MCP server was saved but could not be loaded afterward.');
		}
		return refreshed;
	}

	async setMCPBundleEnabled(bundle: MCPBundleView, enabled: boolean): Promise<void> {
		await this.store.setMCPCollectionEnabled(bundle.ref, bundle.collection.artifact.revision, enabled);
	}

	async setMCPServerEnabled(server: MCPServerView, enabled: boolean): Promise<void> {
		await this.store.setMCPServerEnabled(server.ref, server.artifact.revision, enabled);
	}

	async deleteMCPServer(bundle: MCPBundleView, server: MCPServerView): Promise<void> {
		if (bundle.builtIn || !bundle.editable || server.builtIn) {
			throw new Error('Built-in MCP servers cannot be deleted.');
		}

		const memberships = await this.store.listMCPCollectionMemberships(server.ref);
		const membership = memberships.find(value => artifactRefKey(value.collection) === artifactRefKey(bundle.ref));

		if (!membership) {
			throw new Error('MCP server is not a direct member of this Collection.');
		}

		await this.store.removeMCPCollectionMember({
			collection: membership.collection,
			expectedRevision: membership.collectionRevision,
			index: membership.memberIndex,
		});

		const remaining = await this.store.listMCPCollectionMemberships(server.ref);
		if (remaining.length === 0) {
			await this.aggregate.purgeManagedMCP(server.ref, server.artifact.revision);
		}
	}

	async deleteMCPBundle(bundle: MCPBundleView): Promise<void> {
		if (bundle.builtIn || !bundle.deletable) {
			throw new Error('This MCP Collection cannot be deleted.');
		}

		const servers = await this.listMCPServers(bundle);
		if (servers.length > 0) {
			throw new Error('Remove all MCP servers before deleting this Collection.');
		}

		const management = await this.getMCPCollectionManagementView(bundle.ref);
		const policyRefs = new Map<string, ArtifactRef>();

		for (const occurrence of management.capabilities.occurrences) {
			if (occurrence.type === 'mcp.policy' && occurrence.artifact) {
				policyRefs.set(artifactRefKey(occurrence.artifact), occurrence.artifact);
			}
		}

		for (const policyRef of policyRefs.values()) {
			const memberships = await this.store.listMCPCollectionMemberships(policyRef);
			const membership = memberships.find(value => artifactRefKey(value.collection) === artifactRefKey(bundle.ref));

			if (!membership) {
				continue;
			}
			await this.store.removeMCPCollectionMember({
				collection: membership.collection,
				expectedRevision: membership.collectionRevision,
				index: membership.memberIndex,
			});
		}

		const current = await this.store.getMCPCollection(bundle.ref);
		await this.store.deleteMCPCollection({
			collection: bundle.ref,
			expectedRevision: current.artifact.revision,
		});
	}

	async applyMCPServerSetup(
		server: MCPServerView,
		values: Record<string, MCPSetupSubmissionValue>,
		reset: boolean
	): Promise<void> {
		const latest = await this.store.getMCPServerInstallation(server.ref);
		const nextData = cloneJSON(latest.installation);
		nextData.inputs = cloneJSON(nextData.inputs ?? {});
		const inputs = latest.document.configuration.install.inputs ?? {};
		const targets = findSecretTargets(latest.document);

		for (const [inputName, declaration] of Object.entries(inputs)) {
			const submitted = values[inputName];
			const existing = nextData.inputs[inputName];

			if (declaration.kind === MCPInputKindValue.Text || declaration.kind === MCPInputKindValue.Path) {
				if (submitted?.value?.trim()) {
					nextData.inputs[inputName] = {
						value: submitted.value,
					};
				} else if (reset) {
					nextData.inputs = omitManyKeys(nextData.inputs, [inputName]);
				}
				continue;
			}

			if (declaration.kind === MCPInputKindValue.OAuthClientCredentials) {
				const hasCredentials = Boolean(submitted?.clientID?.trim() || submitted?.clientSecret);

				if (hasCredentials) {
					if (!submitted?.clientID?.trim()) {
						throw new Error(`OAuth input "${inputName}" requires a client ID.`);
					}

					const secret = JSON.stringify({
						clientID: submitted.clientID.trim(),
						...(submitted.clientSecret
							? {
									clientSecret: submitted.clientSecret,
								}
							: {}),
					});

					const result = await this.aggregate.putMCPServerSecret(
						server.ref,
						MCPSecretKindValue.OAuthClientCredentials,
						'clientCredentials',
						secret
					);
					nextData.inputs[inputName] = {
						secretRef: result.secretRef,
					};
				} else if (reset && existing?.secretRef) {
					await this.aggregate.deleteMCPServerSecret(
						server.ref,
						MCPSecretKindValue.OAuthClientCredentials,
						'clientCredentials'
					);
					nextData.inputs = omitManyKeys(nextData.inputs, [inputName]);
				}
				continue;
			}

			if (declaration.kind !== MCPInputKindValue.Secret) {
				continue;
			}

			const target = targets.get(inputName);
			if (!target) {
				throw new Error(`Secret input "${inputName}" has no supported target.`);
			}

			if (submitted?.value) {
				const result = await this.aggregate.putMCPServerSecret(server.ref, target.kind, target.slot, submitted.value);
				nextData.inputs[inputName] = {
					secretRef: result.secretRef,
				};
			} else if (reset && existing?.secretRef) {
				await this.aggregate.deleteMCPServerSecret(server.ref, target.kind, target.slot);
				nextData.inputs = omitManyKeys(nextData.inputs, [inputName]);
			}
		}

		if (latest.builtIn) {
			await this.aggregate.updateProtectedMCPServerInstallation(server.ref, latest.installationRevision, nextData);
			return;
		}

		await this.aggregate.updateMCPServerInstallation(server.ref, latest.artifact.revision, nextData);
	}

	async getMCPCollectionManagementView(collection: ArtifactRef): Promise<MCPCollectionManagementView> {
		const [view, capabilities] = await Promise.all([
			this.store.getMCPCollection(collection),
			this.store.resolveMCPCollection(collection),
		]);

		return {
			collection: view,
			capabilities,
		};
	}

	async getMCPServerManagementView(server: ArtifactRef): Promise<MCPServerManagementView> {
		const [installation, capabilities, runtimeServerID, policy] = await Promise.all([
			this.store.getMCPServerInstallation(server),
			this.store.resolveMCPArtifactCapabilities(server),
			this.aggregate.runtimeServerIDForArtifact(server),
			this.aggregate.getMCPEffectivePolicy(server),
		]);

		const [authHealthResult, runtimeResult] = await Promise.allSettled([
			this.aggregate.getMCPServerAuthHealth(server),
			this.runtime.getMCPServerStatus(runtimeServerID),
		]);

		return {
			installation,
			capabilities,
			runtimeServerID,
			policy,
			authHealth: authHealthResult.status === 'fulfilled' ? authHealthResult.value : undefined,
			runtime: runtimeResult.status === 'fulfilled' ? runtimeResult.value : undefined,
			authHealthError:
				authHealthResult.status === 'rejected'
					? getErrorMessage(authHealthResult.reason, 'MCP authorization health is unavailable.')
					: undefined,
			runtimeError:
				runtimeResult.status === 'rejected'
					? getErrorMessage(runtimeResult.reason, 'MCP runtime status is unavailable.')
					: undefined,
		};
	}

	inspectMCPServer(server: ArtifactRef): Promise<MCPServerManagementView> {
		return this.getMCPServerManagementView(server);
	}

	async getMCPPolicyManagementView(policy: ArtifactRef): Promise<MCPPolicyManagementView> {
		const [view, capabilities] = await Promise.all([
			this.store.getMCPPolicy(policy),
			this.store.resolveMCPArtifactCapabilities(policy),
		]);

		return {
			policy: view,
			capabilities,
		};
	}

	artifactRefForRuntimeServerID(server: MCPRuntimeServerID): Promise<ArtifactRef> {
		return this.aggregate.artifactRefForRuntimeServerID(server);
	}

	runtimeServerIDForArtifact(artifact: ArtifactRef): Promise<MCPRuntimeServerID> {
		return this.aggregate.runtimeServerIDForArtifact(artifact);
	}

	rootIDForRuntimeCatalogID(catalogID: MCPRuntimeCatalogID): Promise<ArtifactRootID> {
		return this.aggregate.rootIDForRuntimeCatalogID(catalogID);
	}

	getMCPEffectivePolicy(server: ArtifactRef): Promise<MCPEffectivePolicy> {
		return this.aggregate.getMCPEffectivePolicy(server);
	}

	getMCPServerAuthHealth(server: ArtifactRef): Promise<MCPAuthHealth> {
		return this.aggregate.getMCPServerAuthHealth(server);
	}

	getMCPServerInstallation(server: ArtifactRef): Promise<MCPStoreServerInstallationView> {
		return this.store.getMCPServerInstallation(server);
	}

	getMCPPolicy(policy: ArtifactRef): Promise<MCPStorePolicyView> {
		return this.store.getMCPPolicy(policy);
	}

	setMCPPolicyEnabled(policy: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<StoreArtifact> {
		return this.store.setMCPPolicyEnabled(policy, expectedRevision, enabled);
	}

	upsertManagedMCPPolicy(request: ManagedMCPPolicyUpsertRequest): Promise<ManagedMCPPolicyUpsertResult> {
		return this.aggregate.upsertManagedMCPPolicy(request);
	}

	purgeManagedMCPPolicy(policy: ArtifactRef, expectedRevision: number): Promise<void> {
		return this.aggregate.purgeManagedMCPPolicy(policy, expectedRevision);
	}

	putMCPServerSecret(
		server: ArtifactRef,
		kind: MCPSecretKind,
		slot: string,
		secret: string
	): Promise<MCPSecretWriteResult> {
		return this.aggregate.putMCPServerSecret(server, kind, slot, secret);
	}

	deleteMCPServerSecret(server: ArtifactRef, kind: MCPSecretKind, slot: string): Promise<void> {
		return this.aggregate.deleteMCPServerSecret(server, kind, slot);
	}

	connectMCPServer(server: MCPRuntimeServerID): Promise<MCPServerRuntimeSnapshot> {
		return this.runtime.connectMCPServer(server);
	}

	disconnectMCPServer(server: MCPRuntimeServerID): Promise<void> {
		return this.runtime.disconnectMCPServer(server);
	}

	refreshMCPServer(server: MCPRuntimeServerID): Promise<MCPServerRuntimeSnapshot> {
		return this.runtime.refreshMCPServer(server);
	}

	getMCPServerStatus(server: MCPRuntimeServerID): Promise<MCPServerRuntimeSnapshot> {
		return this.runtime.getMCPServerStatus(server);
	}

	listMCPServerTools(server: MCPRuntimeServerID): Promise<MCPToolCapability[]> {
		return this.runtime.listMCPServerTools(server);
	}

	listMCPServerToolsPage(
		server: MCPRuntimeServerID,
		pageSize: number,
		pageToken?: string
	): Promise<MCPDiscoveryPage<MCPToolCapability>> {
		return this.runtime.listMCPServerToolsPage(server, pageSize, pageToken);
	}

	listMCPServerResources(server: MCPRuntimeServerID): Promise<MCPResourceRef[]> {
		return this.runtime.listMCPServerResources(server);
	}

	listMCPServerResourcesPage(
		server: MCPRuntimeServerID,
		pageSize: number,
		pageToken?: string
	): Promise<MCPDiscoveryPage<MCPResourceRef>> {
		return this.runtime.listMCPServerResourcesPage(server, pageSize, pageToken);
	}

	listMCPServerResourceTemplates(server: MCPRuntimeServerID): Promise<MCPResourceTemplateRef[]> {
		return this.runtime.listMCPServerResourceTemplates(server);
	}

	listMCPServerResourceTemplatesPage(
		server: MCPRuntimeServerID,
		pageSize: number,
		pageToken?: string
	): Promise<MCPDiscoveryPage<MCPResourceTemplateRef>> {
		return this.runtime.listMCPServerResourceTemplatesPage(server, pageSize, pageToken);
	}

	listMCPServerPrompts(server: MCPRuntimeServerID): Promise<MCPPromptRef[]> {
		return this.runtime.listMCPServerPrompts(server);
	}

	listMCPServerPromptsPage(
		server: MCPRuntimeServerID,
		pageSize: number,
		pageToken?: string
	): Promise<MCPDiscoveryPage<MCPPromptRef>> {
		return this.runtime.listMCPServerPromptsPage(server, pageSize, pageToken);
	}

	readMCPResource(server: MCPRuntimeServerID, uri: string): Promise<MCPReadResourceResponseBody> {
		return this.runtime.readMCPResource(server, uri);
	}

	getMCPPrompt(
		server: MCPRuntimeServerID,
		promptName: string,
		promptArguments: Record<string, string>
	): Promise<MCPGetPromptResponseBody> {
		return this.runtime.getMCPPrompt(server, promptName, promptArguments);
	}

	completeMCPArgument(
		server: MCPRuntimeServerID,
		request: MCPCompleteArgumentRequestBody
	): Promise<MCPCompletionResult> {
		return this.runtime.completeMCPArgument(server, request);
	}

	evaluateMCPToolCall(server: MCPRuntimeServerID, request: InvokeMCPToolRequestBody): Promise<MCPApprovalEvaluation> {
		return this.runtime.evaluateMCPToolCall(server, request);
	}

	evaluateMappedMCPToolCall(
		mapping: MCPProviderToolMapping,
		request: InvokeMCPToolRequestBody
	): Promise<MCPApprovalEvaluation> {
		return this.runtime.evaluateMappedMCPToolCall(mapping, request);
	}

	invokeMCPTool(server: MCPRuntimeServerID, request: InvokeMCPToolRequestBody): Promise<MCPRuntimeInvokeToolResponse> {
		return this.runtime.invokeMCPTool(server, request);
	}

	invokeMappedMCPTool(
		mapping: MCPProviderToolMapping,
		request: InvokeMCPToolRequestBody
	): Promise<MCPRuntimeInvokeToolResponse> {
		return this.runtime.invokeMappedMCPTool(mapping, request);
	}

	resolveMCPApproval(approvalID: string, resolution: MCPApprovalResolution): Promise<MCPApprovalResolutionResult> {
		return this.runtime.resolveMCPApproval(approvalID, resolution);
	}

	listPendingMCPOAuthAuthorizations(): Promise<MCPOAuthAuthorization[]> {
		return this.runtime.listPendingMCPOAuthAuthorizations();
	}

	cancelPendingMCPOAuthAuthorization(server: MCPRuntimeServerID): Promise<boolean> {
		return this.runtime.cancelPendingMCPOAuthAuthorization(server);
	}

	getMCPGlobalSettings(): Promise<MCPGlobalSettings> {
		return this.runtime.getMCPGlobalSettings();
	}

	updateMCPGlobalSettings(expectedRevision: number, settings: MCPAuthSettings): Promise<number> {
		return this.runtime.updateMCPGlobalSettings(expectedRevision, settings);
	}

	resolveMappedToolTarget(target: MappedTarget): Promise<ToolRef> {
		return this.toolStore.resolveMappedToolTarget(target);
	}

	resolveMappedModelTarget(target: MappedTarget): Promise<ModelPresetRef> {
		return this.modelPresetStore.resolveMappedModelTarget(target);
	}

	private toBundleView(collection: CollectionView): MCPBundleView {
		const builtIn = !collection.editable && !collection.deletable;

		return {
			collection,
			ref: {
				rootID: collection.artifact.rootID,
				artifactID: collection.artifact.id,
			},
			displayName: collection.displayName || collection.name,
			logicalName: collection.name,
			description: collection.description,
			enabled: collection.artifact.enabled,
			builtIn,
			editable: collection.editable,
			deletable: collection.deletable,
			baseline: collection.baseline,
		};
	}

	private toServerView(
		management: MCPServerManagementView,
		bundle: ArtifactRef,
		policyRef?: ArtifactRef
	): MCPServerView {
		const installation = management.installation;

		return {
			ref: {
				rootID: installation.artifact.rootID,
				artifactID: installation.artifact.id,
			},
			runtimeServerID: management.runtimeServerID,
			artifact: installation.artifact,
			bundle,
			logicalName: installation.document.logicalName,
			displayName: installation.document.displayName || installation.artifact.displayName,
			document: installation.document,
			installation: installation.installation,
			installationRevision: installation.installationRevision,
			enabled: installation.artifact.enabled,
			builtIn: installation.builtIn,
			policy: management.policy,
			policyRef,
			loadError: management.runtimeError ?? management.authHealthError,
		};
	}

	private async collectPages<T>(
		load: (pageToken: string | undefined) => Promise<{
			items: T[];
			nextPageToken?: string;
		}>
	): Promise<T[]> {
		const output: T[] = [];
		const seenTokens = new Set<string>();
		let pageToken: string | undefined;

		for (let hop = 0; hop < MAX_MANAGEMENT_PAGE_HOPS; hop += 1) {
			const page = await load(pageToken);
			output.push(...page.items);

			if (!page.nextPageToken) {
				return output;
			}
			if (seenTokens.has(page.nextPageToken)) {
				throw new Error('MCP management pagination returned a repeated page token.');
			}

			seenTokens.add(page.nextPageToken);
			pageToken = page.nextPageToken;
		}

		throw new Error(`MCP management pagination exceeded ${MAX_MANAGEMENT_PAGE_HOPS} pages.`);
	}
}
