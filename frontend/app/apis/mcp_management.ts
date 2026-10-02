// oxlint-disable typescript/parameter-properties
import type { ArtifactRef, MappedTarget } from '@/spec/artifact';
import type { CollectionListItem } from '@/spec/collection';
import type {
	InvokeMCPToolRequestBody,
	MCPApprovalEvaluation,
	MCPApprovalResolution,
	MCPApprovalResolutionResult,
	MCPAuthenticationDeclaration,
	MCPAuthHealth,
	MCPAuthHealthState,
	MCPAuthSettings,
	MCPBundleView,
	MCPCompleteArgumentRequestBody,
	MCPCompletionResult,
	MCPDiscoveryPage,
	MCPGetPromptResponseBody,
	MCPHTTPAuthMode,
	MCPHTTPSecretDraft,
	MCPPolicy,
	MCPPromptRef,
	MCPReadResourceResponseBody,
	MCPResourceRef,
	MCPResourceTemplateRef,
	MCPRuntimeInvokeToolResponse,
	MCPRuntimeServerID,
	MCPRuntimeServerView,
	MCPServerAggregateDetails,
	MCPServerData,
	MCPServerDocument,
	MCPServerDraft,
	MCPServerInstallationDataView,
	MCPServerListItem,
	MCPServerRuntimeDetails,
	MCPServerRuntimeSnapshot,
	MCPServerSecretsView,
	MCPServerSetupView,
	MCPServerView,
	MCPSettings,
	MCPSetupInputView,
	MCPSetupSecretTarget,
	MCPSetupSubmissionValue,
	MCPStdioSecretDraft,
	MCPToolCapability,
} from '@/spec/mcp';
import type { ResolvedToolView } from '@/spec/tool';
import { ArtifactState } from '@/spec/artifact';
import { collectionListItemFromCollectionView } from '@/spec/collection';
import {
	MCP_SCHEMA_VERSION,
	MCPApprovalRule,
	MCPAuthHealthState as MCPAuthHealthStateValue,
	MCPAuthState,
	MCPExecutionMode,
	MCPHTTPAuthMode as MCPHTTPAuthModeValue,
	MCPInputKind as MCPInputKindValue,
	MCPSecretKind,
	MCPServerType as MCPServerTypeValue,
	MCPTransportType as MCPTransportTypeValue,
	MCPTrustLevel as MCPTrustLevelValue,
} from '@/spec/mcp';

import { mapWithConcurrency } from '@/lib/async_utils';
import { omitManyKeys } from '@/lib/obj_utils';
import { createSharedAsyncCatalog } from '@/lib/shared_async_catalog';

import type { IMCPAggregateAPI, IMCPRuntimeAPI, IMCPStoreAPI, IToolTargetResolver } from '@/apis/interface';
import type { ModelManagementAPI, ModelManagementItem } from '@/apis/model_management';

const MANAGEMENT_PAGE_SIZE = 100;
const MAX_MANAGEMENT_PAGE_HOPS = 10_000;
const PLACEHOLDER_PATTERN = /\$\{([A-Za-z_][A-Za-z0-9_]*)\}/g;
const MCP_SERVER_MANAGEMENT_CONCURRENCY = 4;
const MCP_RUNTIME_READ_FRESH_MS = 500;
const MCP_RUNTIME_READ_BATCH_SIZE = 256;

interface RuntimeRead {
	server: MCPRuntimeServerID;
	expiresAt: number;
	promise: Promise<MCPRuntimeServerView>;
	resolve: (value: MCPRuntimeServerView) => void;
	reject: (error: unknown) => void;
}

export interface MCPComposerDeclarations {
	bundle: MCPBundleView;
	servers: MCPServerView[];
}

interface MCPSecretTarget {
	kind: MCPSecretKind;
	slot: string;
}

interface PlannedSecretWrite {
	inputName: string;
	secret: string;
}

interface ServerDocumentBuild {
	document: MCPServerDocument;
	secretWrites: PlannedSecretWrite[];
	secretDeletes: string[];
}

function cloneJSON<T>(value: T): T {
	return JSON.parse(JSON.stringify(value)) as T;
}

function artifactRefKey(value: ArtifactRef): string {
	return `${value.rootID}:${value.artifactID}`;
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

function secretTargetChanged(left: MCPSecretTarget | undefined, right: MCPSecretTarget): boolean {
	return !left || left.kind !== right.kind || !sameSecretSlot(left.slot, right.slot);
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

/**
 * The aggregate retains existing secret bindings during a normal settings
 * update. This payload therefore contains only non-secret values.
 */
function writableServerData(installation?: MCPServerInstallationDataView): MCPServerData {
	const inputs = Object.fromEntries(
		Object.entries(installation?.inputs ?? {}).flatMap(([name, input]) =>
			input.value === undefined ? [] : [[name, { value: input.value }] as const]
		)
	);

	return {
		schemaVersion: MCP_SCHEMA_VERSION,
		selectedConnectionProfile: installation?.selectedConnectionProfile,
		...(Object.keys(inputs).length > 0 ? { inputs } : {}),
		...(installation?.additionalPolicies?.length ? { additionalPolicies: [...installation.additionalPolicies] } : {}),
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

		targets.set(inputName, { kind, slot });
	};

	for (const [name, value] of Object.entries(document.mcpServer.env ?? {})) {
		for (const inputName of placeholderNames(value)) {
			register(inputName, MCPSecretKind.StdioEnv, name);
		}
	}

	for (const [name, value] of Object.entries(document.mcpServer.headers ?? {})) {
		for (const inputName of placeholderNames(value)) {
			register(inputName, MCPSecretKind.HTTPHeader, name);
		}
	}

	return targets;
}

function inputBindingFor(data: MCPServerInstallationDataView | undefined, inputName: string) {
	return data?.inputs?.[inputName];
}

function extractHeaderAffixes(value: string, inputName: string): { prefix: string; suffix: string } {
	const placeholder = `\${${inputName}}`;
	const index = value.indexOf(placeholder);

	if (index < 0) {
		return { prefix: '', suffix: '' };
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
	const previousInstallation = existing?.installation;
	const previousTargets = previous ? findSecretTargets(previous) : new Map<string, MCPSecretTarget>();
	const previousOAuthInput = previous?.configuration.auth.clientCredentialsInput;
	const previousInputs = cloneJSON(previous?.configuration.install.inputs ?? {});
	let nextInputs = previousInputs;

	const controlledInputNames = new Set<string>([
		...previousTargets.keys(),
		...(previousOAuthInput ? [previousOAuthInput] : []),
	]);
	const retainedSecretInputs = new Set<string>();
	const secretDeletes = new Set<string>();

	for (const inputName of controlledInputNames) {
		nextInputs = omitManyKeys(nextInputs, [inputName]);
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
			if (row.inputName) {
				retainedSecretInputs.add(row.inputName);
			}

			nextInputs[inputName] = {
				kind: MCPInputKindValue.Secret,
				label: envName,
				required: true,
			};
			env[envName] = `\${${inputName}}`;

			const oldTarget = row.inputName ? previousTargets.get(row.inputName) : undefined;
			const oldBinding = row.inputName ? previousInstallation?.inputs?.[row.inputName] : undefined;
			const nextTarget: MCPSecretTarget = {
				kind: MCPSecretKind.StdioEnv,
				slot: envName,
			};

			if (
				row.inputName &&
				oldBinding?.secretConfigured &&
				(row.deleteExisting || secretTargetChanged(oldTarget, nextTarget))
			) {
				secretDeletes.add(row.inputName);
			}

			if (row.secretValue) {
				secretWrites.push({
					inputName,
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
			if (apiKey.inputName) {
				retainedSecretInputs.add(apiKey.inputName);
			}

			nextInputs[inputName] = {
				kind: MCPInputKindValue.Secret,
				label: headerName,
				required: true,
			};
			headers[headerName] = `${apiKey.valuePrefix}\${${inputName}}${apiKey.valueSuffix}`;

			const oldTarget = apiKey.inputName ? previousTargets.get(apiKey.inputName) : undefined;
			const oldBinding = apiKey.inputName ? previousInstallation?.inputs?.[apiKey.inputName] : undefined;
			const nextTarget: MCPSecretTarget = {
				kind: MCPSecretKind.HTTPHeader,
				slot: headerName,
			};

			if (
				apiKey.inputName &&
				oldBinding?.secretConfigured &&
				(apiKey.deleteExisting || secretTargetChanged(oldTarget, nextTarget))
			) {
				secretDeletes.add(apiKey.inputName);
			}

			if (apiKey.secretValue) {
				secretWrites.push({
					inputName,
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
		if (oauth.inputName) {
			retainedSecretInputs.add(oauth.inputName);
		}

		nextInputs[inputName] = {
			kind: MCPInputKindValue.OAuthClientCredentials,
			label: 'OAuth client credentials',
			required: draft.httpAuthMode === MCPHTTPAuthModeValue.ClientCredentials,
			clientSecretRequired: draft.httpAuthMode === MCPHTTPAuthModeValue.ClientCredentials,
		};
		auth.clientCredentialsInput = inputName;

		const oldBinding = oauth.inputName ? previousInstallation?.inputs?.[oauth.inputName] : undefined;

		if (oauth.inputName && oauth.deleteExisting && oldBinding?.secretConfigured) {
			secretDeletes.add(oauth.inputName);
		}

		if (oauth.secretJSON.trim()) {
			secretWrites.push({
				inputName,
				secret: oauth.secretJSON.trim(),
			});
		}
	}

	for (const inputName of controlledInputNames) {
		if (!retainedSecretInputs.has(inputName)) {
			secretDeletes.add(inputName);
		}
	}

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
		secretWrites,
		secretDeletes: [...secretDeletes],
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

function setupInputsFor(document: MCPServerDocument, installation: MCPServerInstallationDataView): MCPSetupInputView[] {
	const targets = findSecretTargets(document);
	const inputs = document.configuration.install.inputs ?? {};

	return Object.entries(inputs)
		.map(([name, declaration]) => {
			const target: MCPSetupSecretTarget | undefined =
				declaration.kind === MCPInputKindValue.OAuthClientCredentials
					? {
							kind: MCPSecretKind.OAuthClientCredentials,
							slot: 'clientCredentials',
						}
					: targets.get(name);
			const binding = inputBindingFor(installation, name);

			return {
				name,
				declaration,
				target,
				boundValue: binding?.value,
				secretConfigured: binding?.secretConfigured ?? false,
			};
		})
		.toSorted((left, right) => left.name.localeCompare(right.name));
}

export function serverSetupInputs(server: MCPServerView): MCPSetupInputView[] {
	if (!server.document) {
		return [];
	}

	return setupInputsFor(server.document, server.installation ?? {});
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
		return input.secretConfigured;
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

		if (target.kind === MCPSecretKind.StdioEnv) {
			stdioEnv = omitManyKeys(stdioEnv, [target.slot]);
			stdioSecrets.push({
				inputName,
				envName: target.slot,
				existingSecretConfigured: binding?.secretConfigured,
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
			existingSecretConfigured: binding?.secretConfigured,
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
			existingSecretConfigured: oauthBinding?.secretConfigured,
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

const buildByArtifact = (declarations: MCPComposerDeclarations[]): Map<string, MCPServerView> => {
	const byArtifact = new Map<string, MCPServerView>();
	for (const { servers } of declarations) {
		for (const server of servers) {
			const key = artifactRefKey(server.ref);
			if (!byArtifact.has(key) && server.runtimeServerID) {
				byArtifact.set(key, server);
			}
		}
	}
	return byArtifact;
};

export class MCPManagementAPI {
	constructor(
		private readonly store: IMCPStoreAPI,
		private readonly aggregate: IMCPAggregateAPI,
		private readonly runtime: IMCPRuntimeAPI,
		private readonly tools: IToolTargetResolver,
		private readonly models: ModelManagementAPI
	) {}

	private readonly runtimeReads = new Map<MCPRuntimeServerID, RuntimeRead>();
	private queuedRuntimeReads: RuntimeRead[] = [];
	private runtimeFlushScheduled = false;
	private readonly connectionRequests = new Map<MCPRuntimeServerID, Promise<MCPServerRuntimeSnapshot>>();

	private readonly composerMCPDeclarationsCatalog = createSharedAsyncCatalog<MCPComposerDeclarations[]>(async () => {
		const bundles = await this.listMCPBundles();

		return mapWithConcurrency(bundles, MCP_SERVER_MANAGEMENT_CONCURRENCY, async bundle => ({
			bundle,
			servers: await this.listMCPServers(bundle),
		}));
	});

	listComposerMCPDeclarations(force = false): Promise<MCPComposerDeclarations[]> {
		return this.composerMCPDeclarationsCatalog.load(force);
	}

	invalidateComposerMCPDeclarations(): void {
		this.composerMCPDeclarationsCatalog.invalidate();
		this.runtimeReads.clear();
	}

	async listMCPCollectionsForManagement(): Promise<CollectionListItem[]> {
		return this.collectPages(pageToken => this.store.listMCPCollectionsPage(MANAGEMENT_PAGE_SIZE, pageToken));
	}

	async listMCPServersForManagement(): Promise<MCPServerListItem[]> {
		return this.collectPages(pageToken => this.store.listMCPServersPage(MANAGEMENT_PAGE_SIZE, pageToken));
	}

	async listMCPBundles(): Promise<MCPBundleView[]> {
		return (await this.listMCPCollectionsForManagement())
			.map(value => this.toBundleView(value))
			.toSorted((left, right) => {
				if (left.builtIn !== right.builtIn) {
					return left.builtIn ? -1 : 1;
				}
				return left.displayName.localeCompare(right.displayName, undefined, { sensitivity: 'base' });
			});
	}

	async getMCPBundle(collection: ArtifactRef): Promise<MCPBundleView> {
		return this.toBundleView(collectionListItemFromCollectionView(await this.store.getMCPCollection(collection)));
	}

	async getMCPServer(server: ArtifactRef, bundle: ArtifactRef): Promise<MCPServerView> {
		return this.toServerView(await this.aggregate.getMCPServer(server), bundle);
	}

	async getMCPServerForRuntimeServer(server: MCPRuntimeServerID): Promise<MCPRuntimeServerView> {
		const servers = await this.getMCPServersForRuntimeServers([server]);
		if (servers !== null && servers !== undefined && servers.length > 0) {
			return servers[0];
		}
		throw new Error('could not get servers');
	}

	getMCPServersForRuntimeServers(servers: MCPRuntimeServerID[]): Promise<MCPRuntimeServerView[]> {
		const reads = [...new Set(servers)].map(server => {
			const existing = this.runtimeReads.get(server);
			if (existing && existing.expiresAt > Date.now()) {
				return existing.promise;
			}
			let resolve!: RuntimeRead['resolve'];
			let reject!: RuntimeRead['reject'];
			// oxlint-disable-next-line promise/param-names
			const promise = new Promise<MCPRuntimeServerView>((accept, decline) => {
				resolve = accept;
				reject = decline;
			});
			const read: RuntimeRead = {
				server,
				expiresAt: Number.POSITIVE_INFINITY,
				promise,
				resolve,
				reject,
			};
			this.runtimeReads.set(server, read);
			this.queuedRuntimeReads.push(read);
			return promise;
		});
		if (this.queuedRuntimeReads.length > 0 && !this.runtimeFlushScheduled) {
			this.runtimeFlushScheduled = true;
			queueMicrotask(() => {
				void this.flushRuntimeReads();
			});
		}
		return Promise.all(reads);
	}

	private async flushRuntimeReads(): Promise<void> {
		const reads = this.queuedRuntimeReads;
		this.queuedRuntimeReads = [];
		this.runtimeFlushScheduled = false;
		for (let start = 0; start < reads.length; start += MCP_RUNTIME_READ_BATCH_SIZE) {
			const batch = reads.slice(start, start + MCP_RUNTIME_READ_BATCH_SIZE);
			try {
				const details = await this.aggregate.getMCPServersForRuntimeServers(batch.map(read => read.server));
				const values = new Map(details.map(value => [value.connection.server, this.toRuntimeServerView(value)]));
				for (const read of batch) {
					const value = values.get(read.server);
					if (!value) {
						if (this.runtimeReads.get(read.server) === read) {
							this.runtimeReads.delete(read.server);
						}
						read.reject(new Error('MCP runtime response omitted a requested server.'));
						continue;
					}
					read.expiresAt = Date.now() + MCP_RUNTIME_READ_FRESH_MS;
					read.resolve(value);
				}
			} catch (error) {
				for (const read of batch) {
					if (this.runtimeReads.get(read.server) === read) {
						this.runtimeReads.delete(read.server);
					}
					read.reject(error);
				}
			}
		}
	}

	getMCPServerSecrets(server: ArtifactRef): Promise<MCPServerSecretsView> {
		return this.store.getMCPServerSecrets(server);
	}

	async getMCPServerSetup(server: ArtifactRef): Promise<MCPServerSetupView> {
		const details = await this.aggregate.getMCPServer(server);
		const settings = details.settings;

		return {
			server: {
				rootID: settings.artifact.rootID,
				artifactID: settings.artifact.id,
			},
			displayName: settings.document.displayName || settings.artifact.displayName || settings.document.logicalName,
			builtIn: settings.builtIn,
			settingsRevision: settings.installationRevision,
			note: settings.document.configuration.install.note,
			inputs: setupInputsFor(settings.document, settings.installation),
		};
	}

	getMCPSettings(): Promise<MCPSettings> {
		return this.store.getMCPSettings();
	}

	saveMCPSettings(expectedRevision: number, settings: MCPAuthSettings): Promise<MCPSettings> {
		return this.store.saveMCPSettings(expectedRevision, settings);
	}

	async listMCPServers(bundle: MCPBundleView): Promise<MCPServerView[]> {
		const records = await this.aggregate.listMCPCollectionServers(bundle.ref);
		const values = records.map(record => this.toServerView(record, bundle.ref));

		return values.toSorted((left, right) =>
			serverDisplayName(left).localeCompare(serverDisplayName(right), undefined, { sensitivity: 'base' })
		);
	}

	async resolveMCPRuntimeServerIDs(artifacts: ArtifactRef[]): Promise<Map<string, MCPRuntimeServerID>> {
		if (artifacts.length === 0) {
			return new Map();
		}

		let declarations = await this.listComposerMCPDeclarations(false);
		let byArtifact = buildByArtifact(declarations);
		const hasMissing = artifacts.some(ref => !byArtifact.get(artifactRefKey(ref))?.runtimeServerID);
		if (hasMissing) {
			declarations = await this.listComposerMCPDeclarations(true);
			byArtifact = buildByArtifact(declarations);
		}

		const output = new Map<string, MCPRuntimeServerID>();
		const stillMissing: string[] = [];
		for (const ref of artifacts) {
			const key = artifactRefKey(ref);
			if (output.has(key)) {
				continue;
			}
			const view = byArtifact.get(key);
			if (view?.runtimeServerID) {
				output.set(key, view.runtimeServerID);
			} else if (!stillMissing.includes(key)) {
				stillMissing.push(key);
			}
		}

		if (stillMissing.length > 0) {
			throw new Error(`MCP servers not found for artifact(s): ${stillMissing.join(', ')}.`);
		}
		return output;
	}

	async findComposerServerByRuntimeID(server: MCPRuntimeServerID): Promise<MCPServerView | undefined> {
		const find = (declarations: MCPComposerDeclarations[]): MCPServerView | undefined => {
			for (const { servers } of declarations) {
				const found = servers.find(candidate => candidate.runtimeServerID === server);
				if (found) {
					return found;
				}
			}
			return undefined;
		};

		const cached = await this.listComposerMCPDeclarations(false);
		const hit = find(cached);
		if (hit) {
			return hit;
		}

		const refreshed = await this.listComposerMCPDeclarations(true);
		return find(refreshed);
	}

	async createMCPBundle(logicalName: string, displayName: string, description?: string): Promise<MCPBundleView> {
		const collection = await this.store.createMCPCollection({
			rootID: '',
			name: logicalName,
			displayName,
			description,
		});

		this.invalidateComposerMCPDeclarations();

		return this.toBundleView(collectionListItemFromCollectionView(collection));
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

		const policy = await this.aggregate.saveMCPPolicy({
			collection: bundle.ref,
			expectedCollectionRevision: bundle.collection.revision,
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

		if (existing) {
			for (const input of built.secretDeletes) {
				await this.aggregate.clearMCPServerSecret(existing.ref, input);
			}
		}

		const result = existing
			? await this.aggregate.updateMCPServer({
					collection: bundle.ref,
					expectedCollectionRevision: policy.collection.artifact.revision,
					artifact: existing.ref,
					expectedArtifactRevision: existing.artifact.revision,
					document: built.document,
					enabled: draft.enabled,
				})
			: await this.aggregate.createMCPServer({
					collection: bundle.ref,
					expectedCollectionRevision: policy.collection.artifact.revision,
					document: built.document,
					enabled: draft.enabled,
				});

		const serverRef: ArtifactRef = {
			rootID: result.artifact.rootID,
			artifactID: result.artifact.id,
		};

		for (const write of built.secretWrites) {
			await this.aggregate.setMCPServerSecret(serverRef, write.inputName, write.secret);
		}

		this.invalidateComposerMCPDeclarations();

		return this.getMCPServer(serverRef, bundle.ref);
	}

	async setMCPBundleEnabled(bundle: MCPBundleView, enabled: boolean): Promise<void> {
		await this.store.setMCPCollectionEnabled(bundle.ref, bundle.collection.revision, enabled);
		this.invalidateComposerMCPDeclarations();
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
			await this.aggregate.deleteMCPServer(server.ref, server.artifact.revision);
		}

		this.invalidateComposerMCPDeclarations();
	}

	async deleteMCPBundle(bundle: MCPBundleView): Promise<void> {
		if (bundle.builtIn || !bundle.deletable) {
			throw new Error('This MCP Collection cannot be deleted.');
		}

		const servers = await this.listMCPServers(bundle);
		if (servers.length > 0) {
			throw new Error('Remove all MCP servers before deleting this Collection.');
		}

		const policies = await this.store.listMCPPolicies(bundle.ref.rootID);

		for (const policy of policies) {
			const memberships = await this.store.listMCPCollectionMemberships(policy.ref);
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

		this.invalidateComposerMCPDeclarations();
	}

	async applyMCPServerSetup(
		server: MCPServerView | ArtifactRef,
		values: Record<string, MCPSetupSubmissionValue>,
		reset: boolean
	): Promise<void> {
		const serverRef = 'ref' in server ? server.ref : server;
		let latest = await this.aggregate.getMCPServer(serverRef);
		const nextData = writableServerData(latest.settings.installation);
		nextData.inputs = cloneJSON(nextData.inputs ?? {});

		const declarations = latest.settings.document.configuration.install.inputs ?? {};
		let settingsChanged = false;

		for (const [inputName, declaration] of Object.entries(declarations)) {
			if (declaration.kind !== MCPInputKindValue.Text && declaration.kind !== MCPInputKindValue.Path) {
				continue;
			}

			const submitted = values[inputName];
			const existing = nextData.inputs[inputName];

			if (submitted?.value?.trim()) {
				nextData.inputs[inputName] = {
					value: submitted.value,
				};
				settingsChanged = true;
			} else if (reset && existing?.value !== undefined) {
				nextData.inputs = omitManyKeys(nextData.inputs, [inputName]);
				settingsChanged = true;
			}
		}

		if (settingsChanged) {
			latest = await this.aggregate.saveMCPServerSettings(serverRef, latest.settings.installationRevision, nextData);
		}

		const secrets = await this.store.getMCPServerSecrets(serverRef);
		const secretByInput = new Map(secrets.inputs.map(input => [input.name, input] as const));

		for (const [inputName, declaration] of Object.entries(declarations)) {
			const submitted = values[inputName];

			if (declaration.kind === MCPInputKindValue.OAuthClientCredentials) {
				const hasClientID = Boolean(submitted?.clientID?.trim());
				const hasClientSecret = Boolean(submitted?.clientSecret);
				const hasCredentials = hasClientID || hasClientSecret;

				if (hasCredentials) {
					if (!hasClientID) {
						throw new Error(`OAuth input "${inputName}" requires a client ID.`);
					}
					if (declaration.clientSecretRequired && !hasClientSecret) {
						throw new Error(`OAuth input "${inputName}" requires a client secret when replacing its credentials.`);
					}

					const secret = JSON.stringify({
						clientID: submitted.clientID?.trim(),
						...(submitted.clientSecret
							? {
									clientSecret: submitted.clientSecret,
								}
							: {}),
					});

					latest = await this.aggregate.setMCPServerSecret(serverRef, inputName, secret);
				} else if (reset && secretByInput.get(inputName)?.configured) {
					latest = await this.aggregate.clearMCPServerSecret(serverRef, inputName);
				}

				continue;
			}

			if (declaration.kind !== MCPInputKindValue.Secret) {
				continue;
			}

			if (submitted?.value) {
				latest = await this.aggregate.setMCPServerSecret(serverRef, inputName, submitted.value);
			} else if (reset && secretByInput.get(inputName)?.configured) {
				latest = await this.aggregate.clearMCPServerSecret(serverRef, inputName);
			}
		}

		void latest;
		this.invalidateComposerMCPDeclarations();
	}

	connectMCPServer(server: MCPRuntimeServerID): Promise<MCPServerRuntimeSnapshot> {
		const existing = this.connectionRequests.get(server);
		if (existing) {
			return existing;
		}
		const request = this.runtimeMutation(server, () => this.runtime.connectMCPServer(server)).finally(() => {
			if (this.connectionRequests.get(server) === request) {
				this.connectionRequests.delete(server);
			}
		});
		this.connectionRequests.set(server, request);
		return request;
	}

	disconnectMCPServer(server: MCPRuntimeServerID): Promise<void> {
		return this.runtimeMutation(server, () => this.runtime.disconnectMCPServer(server));
	}

	refreshMCPServer(server: MCPRuntimeServerID): Promise<MCPServerRuntimeSnapshot> {
		return this.runtimeMutation(server, () => this.runtime.refreshMCPServer(server));
	}

	cancelMCPServerAuthorization(server: MCPRuntimeServerID): Promise<boolean> {
		return this.runtimeMutation(server, () => this.runtime.cancelMCPServerAuthorization(server));
	}

	private async runtimeMutation<T>(server: MCPRuntimeServerID, mutate: () => Promise<T>): Promise<T> {
		this.runtimeReads.delete(server);
		try {
			return await mutate();
		} finally {
			this.runtimeReads.delete(server);
		}
	}

	listMCPServerTools(
		server: MCPRuntimeServerID,
		pageSize: number,
		pageToken?: string
	): Promise<MCPDiscoveryPage<MCPToolCapability>> {
		return this.runtime.listMCPServerTools(server, pageSize, pageToken);
	}

	listMCPServerResources(
		server: MCPRuntimeServerID,
		pageSize: number,
		pageToken?: string
	): Promise<MCPDiscoveryPage<MCPResourceRef>> {
		return this.runtime.listMCPServerResources(server, pageSize, pageToken);
	}

	listMCPServerResourceTemplates(
		server: MCPRuntimeServerID,
		pageSize: number,
		pageToken?: string
	): Promise<MCPDiscoveryPage<MCPResourceTemplateRef>> {
		return this.runtime.listMCPServerResourceTemplates(server, pageSize, pageToken);
	}

	listMCPServerPrompts(
		server: MCPRuntimeServerID,
		pageSize: number,
		pageToken?: string
	): Promise<MCPDiscoveryPage<MCPPromptRef>> {
		return this.runtime.listMCPServerPrompts(server, pageSize, pageToken);
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

	checkMCPToolCall(server: MCPRuntimeServerID, request: InvokeMCPToolRequestBody): Promise<MCPApprovalEvaluation> {
		return this.runtime.checkMCPToolCall(server, request);
	}

	resolveMCPToolApproval(approvalID: string, resolution: MCPApprovalResolution): Promise<MCPApprovalResolutionResult> {
		return this.runtime.resolveMCPToolApproval(approvalID, resolution);
	}

	invokeMCPTool(server: MCPRuntimeServerID, request: InvokeMCPToolRequestBody): Promise<MCPRuntimeInvokeToolResponse> {
		return this.runtime.invokeMCPTool(server, request);
	}

	resolveMappedTool(target: MappedTarget): Promise<ResolvedToolView> {
		return this.tools.resolveMappedTool(target);
	}

	resolveMappedModelTarget(target: MappedTarget): Promise<ModelManagementItem> {
		return this.models.resolveMappedModelTarget(target);
	}

	private toBundleView(collection: CollectionListItem): MCPBundleView {
		const builtIn = collection.builtIn;

		return {
			collection,
			ref: {
				rootID: collection.ref.rootID,
				artifactID: collection.ref.artifactID,
			},
			displayName: collection.displayName || collection.name,
			logicalName: collection.name,
			description: collection.description,
			enabled: collection.enabled,
			builtIn,
			editable: collection.baseline || collection.editable,
			deletable: !collection.baseline && collection.deletable,
			baseline: collection.baseline,
		};
	}

	private toRuntimeServerView(details: MCPServerRuntimeDetails): MCPRuntimeServerView {
		return {
			ref: details.ref,
			runtimeServerID: details.connection.server,
			authorization: details.authorization,
			pendingAuthorization: details.pendingAuthorization,
			runtime: details.connection,
		};
	}

	private toServerView(details: MCPServerAggregateDetails, bundle: ArtifactRef): MCPServerView {
		const settings = details.settings;

		return {
			ref: {
				rootID: settings.artifact.rootID,
				artifactID: settings.artifact.id,
			},
			runtimeServerID: details.connection.server,
			artifact: settings.artifact,
			bundle,
			logicalName: settings.document.logicalName,
			displayName: settings.document.displayName || settings.artifact.displayName,
			document: settings.document,
			installation: settings.installation,
			installationRevision: settings.installationRevision,
			enabled: settings.artifact.enabled,
			builtIn: settings.builtIn,
			policy: details.policy,
			authHealth: details.authorization,
			runtime: details.connection,
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
			output.push(...(page.items ?? []));

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

// Configuration health is established by a verified detail read. Runtime
// observations replace only its live authorization fields.
export function getMCPRuntimeAuthHealth(
	base: MCPAuthHealth | undefined,
	view: MCPRuntimeServerView
): MCPAuthHealth | undefined {
	if (!base) {
		return undefined;
	}
	const status = view.authorization;
	const states: Record<MCPAuthState, MCPAuthHealthState> = {
		[MCPAuthState.NotRequired]: MCPAuthHealthStateValue.NotRequired,
		[MCPAuthState.Required]: MCPAuthHealthStateValue.AuthorizationNeeded,
		[MCPAuthState.Authorized]: MCPAuthHealthStateValue.Authorized,
		[MCPAuthState.Expired]: MCPAuthHealthStateValue.Expired,
		[MCPAuthState.InsufficientScope]: MCPAuthHealthStateValue.InsufficientScope,
		[MCPAuthState.Error]: MCPAuthHealthStateValue.Error,
	};
	let state = status
		? states[status.state]
		: base.state === MCPAuthHealthStateValue.AuthorizationPending
			? MCPAuthHealthStateValue.AuthorizationNeeded
			: base.state;
	const expiresAt = status ? status.expiresAt : base.expiresAt;
	if (state === MCPAuthHealthStateValue.Authorized && expiresAt && Date.parse(expiresAt) <= Date.now()) {
		state = MCPAuthHealthStateValue.Expired;
	}
	const pending = view.pendingAuthorization;
	return {
		...base,
		...(status
			? {
					authMode: status.authMode,
					resource: status.resource,
					scopes: status.scopes,
					lastError: status.lastError,
				}
			: {}),
		expiresAt,
		state: pending ? MCPAuthHealthStateValue.AuthorizationPending : state,
		authorizationPending: Boolean(pending),
		authorizationURL: pending?.authorizationURL,
		authorizationExpiresAt: pending?.expiresAt,
	};
}
