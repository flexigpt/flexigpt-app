import type { ArtifactCollectionRef, ArtifactKind, ArtifactRecord, ArtifactRef, ArtifactRootID } from '@/spec/artifact';
import { ArtifactState } from '@/spec/artifact';
import { newArtifactStorageKey } from '@/spec/artifact';
import type {
	MCPAppsPolicy,
	MCPArtifactRegistration,
	MCPAuthenticationDeclaration,
	MCPAuthHealth,
	MCPBundle,
	MCPBundleDocument,
	MCPBundleInstallation,
	MCPInputBinding,
	MCPInputDeclaration,
	MCPPolicy,
	MCPPolicyDocument,
	MCPSecretKind,
	MCPServerData,
	MCPServerDocument,
	MCPToolPolicyOverride,
} from '@/spec/mcp_artifact';
import {
	MCP_BUILTIN_ROOT_ID,
	MCP_SCHEMA_VERSION,
	MCP_USER_ROOT_ID,
	MCPApprovalRule,
	MCPAuthHealthState,
	MCPExecutionMode,
	MCPHTTPAuthMode,
	MCPInputKind as MCPInputKindEnum,
	MCPSecretKind as MCPSecretKindEnum,
	MCPServerType,
	MCPTransportType,
	MCPTrustLevel,
} from '@/spec/mcp_artifact';

import { omitManyKeys } from '@/lib/obj_utils';
import { getUUIDv7 } from '@/lib/uuid_utils';

import { mcpAPI } from '@/apis/baseapi';

const PLACEHOLDER_PATTERN = /\$\{([A-Za-z_][A-Za-z0-9_]*)\}/g;

const SERVER_SUBRESOURCE_PREFIX = 'mcpServers/';
const POLICY_SUBRESOURCE_PREFIX = 'policies/';

export interface MCPBundleView {
	bundle: MCPBundle;
	installation: MCPBundleInstallation;
	ref: ArtifactCollectionRef;
	displayName: string;
	logicalName: string;
	description?: string;
	enabled: boolean;
	builtIn: boolean;
}

export interface MCPServerView {
	ref: ArtifactRef;
	artifact: ArtifactRecord;
	bundle: ArtifactCollectionRef;
	logicalName: string;
	displayName: string;
	document?: MCPServerDocument;
	installation?: MCPServerData;
	installationRevision?: number;
	runtimeEnabled: boolean;
	builtIn: boolean;
	policy?: MCPPolicy;
	policyDocument?: MCPPolicyDocument;
	policyArtifact?: ArtifactRecord;
	loadError?: string;
}

interface MCPSecretTarget {
	kind: MCPSecretKind;
	slot: string;
}

export interface MCPSetupInputView {
	name: string;
	declaration: MCPInputDeclaration;
	target?: MCPSecretTarget;
	boundValue?: string;
	boundSecretRef?: string;
}

export interface MCPSetupSubmissionValue {
	value?: string;
	clientID?: string;
	clientSecret?: string;
}

export interface MCPStdioSecretDraft {
	inputName?: string;
	envName: string;
	existingEnvName?: string;
	existingSecretRef?: string;
	secretValue: string;
	deleteExisting: boolean;
}

interface MCPHTTPSecretDraft {
	inputName?: string;
	headerName: string;
	valuePrefix: string;
	valueSuffix: string;
	existingHeaderName?: string;
	existingSecretRef?: string;
	secretValue: string;
	deleteExisting: boolean;
}

interface MCPOAuthClientCredentialsDraft {
	inputName?: string;
	existingSecretRef?: string;
	secretJSON: string;
	deleteExisting: boolean;
	useClientCredentials: boolean;
}

export interface MCPServerDraft {
	logicalName: string;
	displayName: string;
	enabled: boolean;
	transport: MCPTransportType;
	trustLevel: MCPTrustLevel;

	stdioCommand: string;
	stdioArgs: string[];
	stdioEnv: Record<string, string>;
	stdioStartupTimeoutMS?: number;
	stdioSecrets: MCPStdioSecretDraft[];

	httpURL: string;
	httpHeaders: Record<string, string>;
	httpTimeoutMS?: number;
	httpAuthMode: MCPHTTPAuthMode;
	httpAPIKey?: MCPHTTPSecretDraft;
	httpOAuthClientCredentials: MCPOAuthClientCredentialsDraft;
	httpClientIDMetadataDocumentURL: string;

	defaultPolicy: MCPPolicy['defaultPolicy'];
	toolPolicies: Record<string, MCPToolPolicyOverride>;
	appsPolicy: MCPAppsPolicy;
}

interface MCPServerIdentity {
	serverKind: ArtifactKind;
	policyKind: ArtifactKind;
	policyTemplate: Pick<MCPPolicyDocument, 'kind' | 'schemaID' | 'schemaVersion'>;
}

interface PlannedSecretWrite {
	inputName: string;
	kind: MCPSecretKind;
	slot: string;
	secret: string;
}

interface PlannedSecretDelete {
	kind: MCPSecretKind;
	slot: string;
}

interface ServerDocumentBuild {
	document: MCPBundleDocument;
	serverData: MCPServerData;
	serverArtifactID: string;
	policyArtifactID: string;
	secretWrites: PlannedSecretWrite[];
	secretDeletes: PlannedSecretDelete[];
}

let serverIdentityPromise: Promise<MCPServerIdentity> | undefined;

function cloneJSON<T>(value: T): T {
	return JSON.parse(JSON.stringify(value)) as T;
}

function artifactRefKey(value: ArtifactRef): string {
	return `${value.rootID}:${value.artifactID}`;
}

function serverSubresource(logicalName: string): string {
	return `${SERVER_SUBRESOURCE_PREFIX}${logicalName}`;
}

function policySubresource(logicalName: string): string {
	return `${POLICY_SUBRESOURCE_PREFIX}${logicalName}`;
}

function getErrorMessage(error: unknown, fallback: string): string {
	if (error instanceof Error && error.message.trim()) {
		return error.message;
	}

	return fallback;
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

function sameHeaderName(left: string, right: string): boolean {
	return left.trim().toLocaleLowerCase() === right.trim().toLocaleLowerCase();
}

function defaultPolicy(): MCPPolicy {
	return {
		trustLevel: MCPTrustLevel.Untrusted,
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
	const declarations = document.extension.install.inputs ?? {};

	const register = (inputName: string, kind: MCPSecretKind, slot: string) => {
		const declaration = declarations[inputName];

		if (!declaration || declaration.kind !== MCPInputKindEnum.Secret) {
			return;
		}

		const current = targets.get(inputName);
		if (current && (current.kind !== kind || current.slot.toLocaleLowerCase() !== slot.toLocaleLowerCase())) {
			throw new Error(`Secret installation input "${inputName}" is used by more than one materialization target.`);
		}

		targets.set(inputName, { kind, slot });
	};

	for (const [envName, value] of Object.entries(document.mcpServer.env ?? {})) {
		for (const inputName of placeholderNames(value)) {
			register(inputName, MCPSecretKindEnum.StdioEnv, envName);
		}
	}

	for (const [headerName, value] of Object.entries(document.mcpServer.headers ?? {})) {
		for (const inputName of placeholderNames(value)) {
			register(inputName, MCPSecretKindEnum.HTTPHeader, headerName);
		}
	}

	return targets;
}

function getPolicyForServer(
	document: MCPBundleDocument,
	serverDocument: MCPServerDocument
): MCPPolicyDocument | undefined {
	const reference = serverDocument.extension.policy?.ref;

	if (!reference) {
		return undefined;
	}

	return document.bundleExtension.policies?.[reference];
}

function inputBindingFor(data: MCPServerData | undefined, inputName: string): MCPInputBinding | undefined {
	return data?.inputs?.[inputName];
}

function extractHeaderPrefixAndSuffix(
	value: string,
	inputName: string
): {
	prefix: string;
	suffix: string;
} {
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

function toTransportType(document: MCPServerDocument): MCPTransportType {
	return document.mcpServer.type === MCPServerType.Stdio ? MCPTransportType.Stdio : MCPTransportType.StreamableHTTP;
}

export function serverDisplayName(server: MCPServerView): string {
	return server.document?.displayName || server.artifact.name || server.logicalName;
}

export function serverSetupInputs(server: MCPServerView): MCPSetupInputView[] {
	if (!server.document) {
		return [];
	}

	const secretTargets = findSecretTargets(server.document);
	const inputs = server.document.extension.install.inputs ?? {};

	return Object.entries(inputs)
		.map(([name, declaration]) => {
			const target =
				declaration.kind === MCPInputKindEnum.OAuthClientCredentials
					? {
							kind: MCPSecretKindEnum.OAuthClientCredentials,
							slot: 'clientCredentials',
						}
					: secretTargets.get(name);

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
		if (input.declaration.kind === MCPInputKindEnum.Text || input.declaration.kind === MCPInputKindEnum.Path) {
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
	const policyDocument = server?.policyDocument;
	const policy = policyDocument?.body ?? defaultPolicy();
	const secretTargets = document ? findSecretTargets(document) : new Map<string, MCPSecretTarget>();
	let plainEnv = cloneJSON(document?.mcpServer.env ?? {});
	let plainHeaders = cloneJSON(document?.mcpServer.headers ?? {});

	const stdioSecrets: MCPStdioSecretDraft[] = [];
	let apiKey: MCPHTTPSecretDraft | undefined;

	for (const [inputName, target] of secretTargets) {
		const binding = inputBindingFor(server?.installation, inputName);

		if (target.kind === MCPSecretKindEnum.StdioEnv) {
			plainEnv = omitManyKeys(plainEnv, [target.slot]);
			stdioSecrets.push({
				inputName,
				envName: target.slot,
				existingEnvName: target.slot,
				existingSecretRef: binding?.secretRef,
				secretValue: '',
				deleteExisting: false,
			});
			continue;
		}

		if (target.kind === MCPSecretKindEnum.HTTPHeader) {
			const headerValue = plainHeaders[target.slot] ?? '';
			const parts = extractHeaderPrefixAndSuffix(headerValue, inputName);
			plainHeaders = omitManyKeys(plainHeaders, [target.slot]);
			apiKey = {
				inputName,
				headerName: target.slot,
				existingHeaderName: target.slot,
				valuePrefix: parts.prefix,
				valueSuffix: parts.suffix,
				existingSecretRef: binding?.secretRef,
				secretValue: '',
				deleteExisting: false,
			};
		}
	}

	const oauthInputName = document?.extension.auth.clientCredentialsInput;
	const oauthBinding = oauthInputName ? inputBindingFor(server?.installation, oauthInputName) : undefined;

	return {
		logicalName: document?.logicalName ?? '',
		displayName: document?.displayName ?? '',
		enabled: server?.artifact.enabled ?? true,
		transport: document ? toTransportType(document) : MCPTransportType.StreamableHTTP,
		trustLevel: policy.trustLevel,

		stdioCommand: document?.mcpServer.command ?? '',
		stdioArgs: [...(document?.mcpServer.args ?? [])],
		stdioEnv: plainEnv,
		stdioStartupTimeoutMS: document?.extension.timeoutMS,
		stdioSecrets,

		httpURL: document?.mcpServer.url ?? '',
		httpHeaders: plainHeaders,
		httpTimeoutMS: document?.extension.timeoutMS,
		httpAuthMode: document?.extension.auth.mode ?? MCPHTTPAuthMode.None,
		httpAPIKey: apiKey,
		httpOAuthClientCredentials: {
			inputName: oauthInputName,
			existingSecretRef: oauthBinding?.secretRef,
			secretJSON: '',
			deleteExisting: false,
			useClientCredentials: Boolean(oauthInputName),
		},
		httpClientIDMetadataDocumentURL: document?.extension.auth.clientIDMetadataDocumentURL ?? '',

		defaultPolicy: cloneJSON(policy.defaultPolicy),
		toolPolicies: cloneJSON(policy.toolPolicies ?? {}),
		appsPolicy: cloneJSON(policy.appsPolicy),
	};
}

export async function loadMCPBundleViews(): Promise<MCPBundleView[]> {
	const roots: ArtifactRootID[] = [MCP_USER_ROOT_ID, MCP_BUILTIN_ROOT_ID];
	const responses = await Promise.allSettled(roots.map(rootID => mcpAPI.listMCPBundles(rootID)));

	const bundles: MCPBundle[] = [];

	for (const response of responses) {
		if (response.status === 'fulfilled') {
			bundles.push(...response.value);
		}
	}

	const loaded = await Promise.all(
		bundles.map(async bundle => {
			const ref: ArtifactCollectionRef = {
				rootID: bundle.collection.rootID,
				collectionID: bundle.collection.id,
			};

			const installation = await mcpAPI.getMCPBundleInstallation(ref);

			return {
				bundle,
				installation,
				ref,
				displayName: bundle.collection.displayName || bundle.data.logicalName,
				logicalName: bundle.data.logicalName,
				description: bundle.collection.description,
				enabled: installation.runtimeEnabled,
				builtIn: installation.builtIn,
			} satisfies MCPBundleView;
		})
	);

	return loaded.toSorted((left, right) => {
		if (left.builtIn !== right.builtIn) {
			return left.builtIn ? -1 : 1;
		}

		return left.displayName.localeCompare(right.displayName, undefined, {
			sensitivity: 'base',
		});
	});
}

export async function loadMCPServerViews(bundle: MCPBundleView): Promise<MCPServerView[]> {
	const [document, artifacts] = await Promise.all([
		mcpAPI.getMCPBundleDocument(bundle.ref),
		mcpAPI.listMCPBundleServers(bundle.ref),
	]);

	const policies = await mcpAPI.listMCPBundlePolicies(bundle.ref);
	const policiesBySubresource = new Map(
		policies.map(policy => [policy.binding.subresourceLocator ?? '', policy] as const)
	);

	const views = await Promise.all(
		artifacts.map(async artifact => {
			try {
				const installation = await mcpAPI.getMCPServerInstallation({
					rootID: artifact.rootID,
					artifactID: artifact.id,
				});
				const policyDocument = getPolicyForServer(document, installation.document);
				const policyReference = installation.document.extension.policy?.ref;
				const policyArtifact = policyReference
					? policiesBySubresource.get(policySubresource(policyReference))
					: undefined;

				return {
					ref: {
						rootID: artifact.rootID,
						artifactID: artifact.id,
					},
					artifact,
					bundle: bundle.ref,
					logicalName: installation.document.logicalName,
					displayName: installation.document.displayName || artifact.name,
					document: installation.document,
					installation: installation.installation,
					installationRevision: installation.installationRevision,
					runtimeEnabled: installation.runtimeEnabled,
					builtIn: installation.builtIn,
					policy: policyDocument?.body,
					policyDocument,
					policyArtifact,
				} satisfies MCPServerView;
			} catch (error) {
				return {
					ref: {
						rootID: artifact.rootID,
						artifactID: artifact.id,
					},
					artifact,
					bundle: bundle.ref,
					logicalName: artifact.name,
					displayName: artifact.name,
					runtimeEnabled: false,
					builtIn: bundle.builtIn,
					loadError: getErrorMessage(error, 'Unable to load MCP server installation.'),
				} satisfies MCPServerView;
			}
		})
	);

	return views.toSorted((left, right) =>
		serverDisplayName(left).localeCompare(serverDisplayName(right), undefined, {
			sensitivity: 'base',
		})
	);
}

async function discoverServerIdentity(): Promise<MCPServerIdentity> {
	const bundles = await loadMCPBundleViews();

	for (const bundle of bundles) {
		const [document, servers, policies] = await Promise.all([
			mcpAPI.getMCPBundleDocument(bundle.ref),
			mcpAPI.listMCPBundleServers(bundle.ref),
			mcpAPI.listMCPBundlePolicies(bundle.ref),
		]);

		const policyTemplate = Object.values(document.bundleExtension.policies ?? {})[0];
		const serverKind = servers[0]?.kind;
		const policyKind = policies[0]?.kind;

		if (policyTemplate && serverKind && policyKind) {
			return {
				serverKind,
				policyKind,
				policyTemplate: {
					kind: policyTemplate.kind,
					schemaID: policyTemplate.schemaID,
					schemaVersion: policyTemplate.schemaVersion,
				},
			};
		}
	}

	throw new Error(
		'MCP schema identity could not be discovered from installed bundles. Install the built-in MCP topology before creating a server.'
	);
}

async function getServerIdentity(): Promise<MCPServerIdentity> {
	if (!serverIdentityPromise) {
		serverIdentityPromise = discoverServerIdentity();
	}

	return serverIdentityPromise;
}

function currentServerRecordBySubresource(records: ArtifactRecord[]): Map<string, ArtifactRecord> {
	return new Map(
		records
			.filter(record => record.binding.subresourceLocator)
			.map(record => [record.binding.subresourceLocator as string, record] as const)
	);
}

function preserveSecretBinding(
	existing: MCPServerData,
	previousInputName: string | undefined,
	nextInputName: string,
	replaced: boolean,
	deleted: boolean
): MCPInputBinding | undefined {
	if (!previousInputName || replaced || deleted || previousInputName !== nextInputName) {
		return undefined;
	}

	return existing.inputs?.[previousInputName];
}

function nextPolicyName(serverArtifactID: string): string {
	return `policy-${serverArtifactID}`;
}

function buildPolicyDocument(
	identity: MCPServerIdentity,
	policyName: string,
	displayName: string,
	draft: MCPServerDraft,
	existing?: MCPPolicyDocument
): MCPPolicyDocument {
	return {
		kind: existing?.kind ?? identity.policyTemplate.kind,
		schemaID: existing?.schemaID ?? identity.policyTemplate.schemaID,
		schemaVersion: existing?.schemaVersion ?? identity.policyTemplate.schemaVersion,
		logicalName: policyName,
		logicalVersion: existing?.logicalVersion,
		displayName: `${displayName} Policy`,
		description: existing?.description,
		labels: cloneJSON(existing?.labels ?? {}),
		body: {
			trustLevel: draft.trustLevel,
			defaultPolicy: cloneJSON(draft.defaultPolicy),
			toolPolicies: cloneJSON(draft.toolPolicies),
			appsPolicy: cloneJSON(draft.appsPolicy),
		},
	};
}

function replaceControlledSecretTargets(
	document: MCPServerDocument,
	existingInstallation: MCPServerData | undefined,
	draft: MCPServerDraft,
	_serverArtifactID: string
): {
	document: MCPServerDocument;
	serverData: MCPServerData;
	secretWrites: PlannedSecretWrite[];
	secretDeletes: PlannedSecretDelete[];
} {
	const nextDocument = cloneJSON(document);
	const existingData = cloneJSON(existingInstallation ?? defaultServerData());
	const existingTargets = findSecretTargets(document);
	let existingInputs = nextDocument.extension.install.inputs ?? {};
	const reservedNames = new Set(Object.keys(existingInputs));
	const controlledInputNames = new Set<string>();

	for (const inputName of existingTargets.keys()) {
		controlledInputNames.add(inputName);
	}
	if (document.extension.auth.clientCredentialsInput) {
		controlledInputNames.add(document.extension.auth.clientCredentialsInput);
	}

	for (const inputName of controlledInputNames) {
		existingInputs = omitManyKeys(existingInputs, [inputName]);
	}

	nextDocument.extension.install.inputs = existingInputs;

	const nextData = cloneJSON(existingData);
	nextData.inputs = {};

	const secretWrites: PlannedSecretWrite[] = [];
	const secretDeletes: PlannedSecretDelete[] = [];

	for (const [inputName, target] of existingTargets) {
		const oldBinding = existingData.inputs?.[inputName];

		if (!oldBinding?.secretRef) {
			continue;
		}

		const retainedByStdioRow = draft.stdioSecrets.some(
			row => row.inputName === inputName && row.envName === target.slot && !row.deleteExisting && !row.secretValue
		);

		const retainedByAPIKey =
			draft.httpAPIKey?.inputName === inputName &&
			sameHeaderName(draft.httpAPIKey.headerName, target.slot) &&
			!draft.httpAPIKey.deleteExisting &&
			!draft.httpAPIKey.secretValue;

		if (!retainedByStdioRow && !retainedByAPIKey) {
			secretDeletes.push(target);
		}
	}

	const previousOAuthInput = document.extension.auth.clientCredentialsInput;
	const previousOAuthBinding = previousOAuthInput ? existingData.inputs?.[previousOAuthInput] : undefined;

	if (
		previousOAuthInput &&
		previousOAuthBinding?.secretRef &&
		(!draft.httpOAuthClientCredentials.useClientCredentials ||
			draft.httpOAuthClientCredentials.deleteExisting ||
			Boolean(draft.httpOAuthClientCredentials.secretJSON))
	) {
		secretDeletes.push({
			kind: MCPSecretKindEnum.OAuthClientCredentials,
			slot: 'clientCredentials',
		});
	}

	const baseCore = nextDocument.mcpServer;
	baseCore.type = draft.transport === MCPTransportType.Stdio ? MCPServerType.Stdio : MCPServerType.HTTP;

	if (draft.transport === MCPTransportType.Stdio) {
		baseCore.command = draft.stdioCommand.trim();
		baseCore.args = draft.stdioArgs.filter(Boolean);
		baseCore.env = cloneJSON(draft.stdioEnv);
		delete baseCore.url;
		delete baseCore.headers;

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
			existingInputs[inputName] = {
				kind: MCPInputKindEnum.Secret,
				label: envName,
				required: true,
			};
			baseCore.env[envName] = `\${${inputName}}`;

			const oldBinding = preserveSecretBinding(
				existingData,
				row.inputName,
				inputName,
				Boolean(row.secretValue),
				row.deleteExisting
			);
			if (oldBinding) {
				nextData.inputs[inputName] = oldBinding;
			}

			if (row.secretValue) {
				secretWrites.push({
					inputName,
					kind: MCPSecretKindEnum.StdioEnv,
					slot: envName,
					secret: row.secretValue,
				});
			}
		}

		nextDocument.extension.auth = {
			mode: MCPHTTPAuthMode.None,
		};
	} else {
		baseCore.url = draft.httpURL.trim();
		baseCore.headers = cloneJSON(draft.httpHeaders);
		delete baseCore.command;
		delete baseCore.args;
		delete baseCore.env;

		const auth: MCPAuthenticationDeclaration = {
			mode: draft.httpAuthMode,
			clientIDMetadataDocumentURL:
				draft.httpAuthMode === MCPHTTPAuthMode.OAuth
					? draft.httpClientIDMetadataDocumentURL.trim() || undefined
					: undefined,
		};

		if (draft.httpAuthMode === MCPHTTPAuthMode.APIKey && draft.httpAPIKey) {
			const headerName = draft.httpAPIKey.headerName.trim();
			const inputName =
				draft.httpAPIKey.inputName && !reservedNames.has(draft.httpAPIKey.inputName)
					? draft.httpAPIKey.inputName
					: uniqueInputName(normalizeInputName(`http_${headerName}`), reservedNames);

			reservedNames.add(inputName);
			existingInputs[inputName] = {
				kind: MCPInputKindEnum.Secret,
				label: headerName,
				required: true,
			};
			baseCore.headers[headerName] = `${draft.httpAPIKey.valuePrefix}\${${inputName}}${draft.httpAPIKey.valueSuffix}`;

			const oldBinding = preserveSecretBinding(
				existingData,
				draft.httpAPIKey.inputName,
				inputName,
				Boolean(draft.httpAPIKey.secretValue),
				draft.httpAPIKey.deleteExisting
			);
			if (oldBinding) {
				nextData.inputs[inputName] = oldBinding;
			}

			if (draft.httpAPIKey.secretValue) {
				secretWrites.push({
					inputName,
					kind: MCPSecretKindEnum.HTTPHeader,
					slot: headerName,
					secret: draft.httpAPIKey.secretValue,
				});
			}
		}

		const oauthDraft = draft.httpOAuthClientCredentials;
		const clientCredentialsRequired = draft.httpAuthMode === MCPHTTPAuthMode.ClientCredentials;
		const useOAuthCredentials = clientCredentialsRequired || oauthDraft.useClientCredentials;

		if (useOAuthCredentials) {
			const inputName =
				oauthDraft.inputName && !reservedNames.has(oauthDraft.inputName)
					? oauthDraft.inputName
					: uniqueInputName('mcp_oauth_client_credentials', reservedNames);

			reservedNames.add(inputName);
			existingInputs[inputName] = {
				kind: MCPInputKindEnum.OAuthClientCredentials,
				label: 'OAuth client credentials',
				required: clientCredentialsRequired,
				clientSecretRequired: clientCredentialsRequired,
			};

			auth.clientCredentialsInput = inputName;

			const oldBinding = preserveSecretBinding(
				existingData,
				oauthDraft.inputName,
				inputName,
				Boolean(oauthDraft.secretJSON),
				oauthDraft.deleteExisting
			);
			if (oldBinding) {
				nextData.inputs[inputName] = oldBinding;
			}

			if (oauthDraft.secretJSON.trim()) {
				secretWrites.push({
					inputName,
					kind: MCPSecretKindEnum.OAuthClientCredentials,
					slot: 'clientCredentials',
					secret: oauthDraft.secretJSON.trim(),
				});
			}
		}

		nextDocument.extension.auth = auth;
	}

	nextDocument.extension.timeoutMS =
		draft.transport === MCPTransportType.Stdio ? draft.stdioStartupTimeoutMS : draft.httpTimeoutMS;

	nextDocument.extension.install.inputs = existingInputs;

	return {
		document: nextDocument,
		serverData: nextData,
		secretWrites,
		secretDeletes,
	};
}

function buildServerDocument(
	bundleDocument: MCPBundleDocument,
	existing: MCPServerView | undefined,
	draft: MCPServerDraft,
	serverArtifactID: string,
	policyArtifactID: string,
	identity: MCPServerIdentity
): ServerDocumentBuild {
	const existingServerDocument = existing?.document;
	const baseDocument: MCPServerDocument = existingServerDocument
		? cloneJSON(existingServerDocument)
		: {
				kind: identity.serverKind,
				schemaID: '',
				schemaVersion: MCP_SCHEMA_VERSION,
				logicalName: draft.logicalName,
				displayName: draft.displayName,
				mcpServer: {
					type: draft.transport === MCPTransportType.Stdio ? MCPServerType.Stdio : MCPServerType.HTTP,
				},
				extension: {
					displayName: draft.displayName,
					auth: {
						mode: draft.httpAuthMode,
					},
					install: {
						inputs: {},
					},
				},
			};

	baseDocument.logicalName = existingServerDocument?.logicalName ?? draft.logicalName.trim();
	baseDocument.displayName = draft.displayName.trim();
	baseDocument.extension.displayName = draft.displayName.trim();
	baseDocument.extension.connectionProfiles = cloneJSON(existingServerDocument?.extension.connectionProfiles ?? {});
	baseDocument.extension.labels = cloneJSON(existingServerDocument?.extension.labels ?? {});

	const controlled = replaceControlledSecretTargets(baseDocument, existing?.installation, draft, serverArtifactID);
	const policyName = existingServerDocument?.extension.policy?.ref ?? nextPolicyName(serverArtifactID);
	const existingPolicy = bundleDocument.bundleExtension.policies?.[policyName];

	controlled.document.extension.policy = {
		ref: policyName,
		required: true,
	};

	const nextBundleDocument = cloneJSON(bundleDocument);
	nextBundleDocument.digest = undefined;
	nextBundleDocument.mcpServers = {
		...nextBundleDocument.mcpServers,
		[controlled.document.logicalName]: controlled.document.mcpServer,
	};
	nextBundleDocument.bundleExtension = {
		...nextBundleDocument.bundleExtension,
		servers: {
			...nextBundleDocument.bundleExtension.servers,
			[controlled.document.logicalName]: controlled.document.extension,
		},
		policies: {
			...nextBundleDocument.bundleExtension.policies,
			[policyName]: buildPolicyDocument(identity, policyName, draft.displayName.trim(), draft, existingPolicy),
		},
	};

	return {
		document: nextBundleDocument,
		serverData: controlled.serverData,
		serverArtifactID,
		policyArtifactID,
		secretWrites: controlled.secretWrites,
		secretDeletes: controlled.secretDeletes,
	};
}

function registrationsForDocument(
	document: MCPBundleDocument,
	serverRecords: ArtifactRecord[],
	policyRecords: ArtifactRecord[],
	targetServerArtifactID: string,
	targetLogicalName: string,
	targetServerEnabled: boolean,
	targetServerData: MCPServerData,
	targetPolicyArtifactID: string
): MCPArtifactRegistration[] {
	const serverBySubresource = currentServerRecordBySubresource(serverRecords);
	const policyBySubresource = currentServerRecordBySubresource(policyRecords);
	const registrations: MCPArtifactRegistration[] = [];

	for (const logicalName of Object.keys(document.mcpServers).toSorted()) {
		const subresource = serverSubresource(logicalName);
		const existing = serverBySubresource.get(subresource);

		if (logicalName === targetLogicalName) {
			registrations.push({
				artifactID: existing?.id ?? targetServerArtifactID,
				subresource,
				kind: existing?.kind ?? '',
				enabled: targetServerEnabled,
				data: JSON.stringify(targetServerData),
			});
			continue;
		}

		if (!existing) {
			throw new Error(`MCP Bundle document contains server "${logicalName}" without a registered Artifact.`);
		}

		registrations.push({
			artifactID: existing.id,
			subresource,
			kind: existing.kind,
			enabled: existing.enabled,
		});
	}

	for (const logicalName of Object.keys(document.bundleExtension.policies ?? {}).toSorted()) {
		const subresource = policySubresource(logicalName);
		const existing = policyBySubresource.get(subresource);

		if (logicalName === nextPolicyName(targetServerArtifactID) || !existing) {
			registrations.push({
				artifactID: existing?.id ?? targetPolicyArtifactID,
				subresource,
				kind: existing?.kind ?? '',
				enabled: true,
			});
			continue;
		}

		registrations.push({
			artifactID: existing.id,
			subresource,
			kind: existing.kind,
			enabled: existing.enabled,
		});
	}

	return registrations;
}

function fillMissingKinds(
	registrations: MCPArtifactRegistration[],
	identity: MCPServerIdentity,
	targetServerArtifactID: string,
	targetPolicyArtifactID: string
): MCPArtifactRegistration[] {
	return registrations.map(registration => {
		if (registration.kind) {
			return registration;
		}

		if (registration.artifactID === targetServerArtifactID) {
			return {
				...registration,
				kind: identity.serverKind,
			};
		}

		if (registration.artifactID === targetPolicyArtifactID) {
			return {
				...registration,
				kind: identity.policyKind,
			};
		}

		throw new Error(`MCP Artifact registration "${registration.subresource}" has no Artifact kind.`);
	});
}

function dedupeSecretDeletes(values: PlannedSecretDelete[]): PlannedSecretDelete[] {
	const seen = new Set<string>();

	return values.filter(value => {
		const key = `${value.kind}:${value.slot.toLocaleLowerCase()}`;

		if (seen.has(key)) {
			return false;
		}

		seen.add(key);
		return true;
	});
}

export async function createMCPBundle(
	logicalName: string,
	displayName: string,
	description?: string
): Promise<MCPBundleView> {
	const templateRef: ArtifactCollectionRef = {
		rootID: MCP_USER_ROOT_ID,
		collectionID: '0198f097-0d5b-7000-8000-000000000020',
	};
	const template = await mcpAPI.getMCPBundleDocument(templateRef);

	const collectionID = getUUIDv7();
	const sourceID = getUUIDv7();
	const sourceStorageKey = newArtifactStorageKey();
	const document: MCPBundleDocument = {
		...cloneJSON(template),
		digest: undefined,
		logicalName,
		displayName,
		description,
		labels: {},
		mcpServers: {},
		bundleExtension: {
			servers: {},
			policies: {},
		},
	};

	const bundle = await mcpAPI.createMCPBundle({
		rootID: MCP_USER_ROOT_ID,
		collectionID,
		sourceID,
		sourceStorageKey,
		document,
		registrations: [],
	});

	const installation = await mcpAPI.getMCPBundleInstallation({
		rootID: bundle.collection.rootID,
		collectionID: bundle.collection.id,
	});

	return {
		bundle,
		installation,
		ref: {
			rootID: bundle.collection.rootID,
			collectionID: bundle.collection.id,
		},
		displayName: bundle.collection.displayName || bundle.data.logicalName,
		logicalName: bundle.data.logicalName,
		description: bundle.collection.description,
		enabled: installation.runtimeEnabled,
		builtIn: installation.builtIn,
	};
}

export async function saveMCPServer(
	bundle: MCPBundleView,
	existing: MCPServerView | undefined,
	draft: MCPServerDraft
): Promise<MCPServerView> {
	if (bundle.builtIn || existing?.builtIn) {
		throw new Error('Built-in MCP definitions cannot be edited.');
	}

	if (!draft.logicalName.trim()) {
		throw new Error('MCP server logical name is required.');
	}

	if (!draft.displayName.trim()) {
		throw new Error('MCP server display name is required.');
	}

	const [currentBundle, currentDocument, serverRecords, policyRecords, identity] = await Promise.all([
		mcpAPI.getMCPBundle(bundle.ref),
		mcpAPI.getMCPBundleDocument(bundle.ref),
		mcpAPI.listMCPBundleServers(bundle.ref),
		mcpAPI.listMCPBundlePolicies(bundle.ref),
		getServerIdentity(),
	]);

	const serverArtifactID = existing?.artifact.id ?? getUUIDv7();
	const policyArtifactID = existing?.policyArtifact?.id ?? getUUIDv7();

	if (!existing && currentDocument.mcpServers[draft.logicalName.trim()]) {
		throw new Error(`An MCP server named "${draft.logicalName.trim()}" already exists in this bundle.`);
	}

	const built = buildServerDocument(currentDocument, existing, draft, serverArtifactID, policyArtifactID, identity);

	const registrations = fillMissingKinds(
		registrationsForDocument(
			built.document,
			serverRecords,
			policyRecords,
			serverArtifactID,
			existing?.logicalName ?? draft.logicalName.trim(),
			draft.enabled,
			built.serverData,
			policyArtifactID
		),
		identity,
		serverArtifactID,
		policyArtifactID
	);

	await mcpAPI.replaceMCPBundleDocument({
		bundle: bundle.ref,
		expectedCollectionRevision: currentBundle.collection.revision,
		document: built.document,
		registrations,
	});

	const serverRef: ArtifactRef = {
		rootID: bundle.ref.rootID,
		artifactID: serverArtifactID,
	};

	const freshInstallation = await mcpAPI.getMCPServerInstallation(serverRef);
	const nextData = cloneJSON(freshInstallation.installation);
	nextData.inputs = cloneJSON(nextData.inputs ?? {});

	for (const deletion of dedupeSecretDeletes(built.secretDeletes)) {
		await mcpAPI.deleteMCPServerSecret(serverRef, deletion.kind, deletion.slot);
	}

	for (const write of built.secretWrites) {
		const result = await mcpAPI.putMCPServerSecret(serverRef, write.kind, write.slot, write.secret);
		nextData.inputs[write.inputName] = {
			secretRef: result.secretRef,
		};
	}

	if (JSON.stringify(nextData) !== JSON.stringify(freshInstallation.installation)) {
		await mcpAPI.updateMCPServerInstallation(serverRef, freshInstallation.artifact.revision, nextData);
	}

	const refreshedBundle: MCPBundleView = {
		bundle: await mcpAPI.getMCPBundle(bundle.ref),
		installation: await mcpAPI.getMCPBundleInstallation(bundle.ref),
		ref: bundle.ref,
		displayName: bundle.displayName,
		logicalName: bundle.logicalName,
		description: bundle.description,
		enabled: bundle.enabled,
		builtIn: false,
	};
	const refreshedServers = await loadMCPServerViews(refreshedBundle);
	const refreshed = refreshedServers.find(server => artifactRefKey(server.ref) === artifactRefKey(serverRef));

	if (!refreshed) {
		throw new Error('MCP server was saved, but could not be loaded afterward.');
	}

	return refreshed;
}

export async function setMCPServerRuntimeEnabled(
	bundle: MCPBundleView,
	server: MCPServerView,
	enabled: boolean
): Promise<void> {
	if (!server.document || !server.installation) {
		throw new Error('The MCP server installation is unavailable.');
	}

	if (server.builtIn) {
		await mcpAPI.updateProtectedMCPServerInstallation(
			server.ref,
			server.installationRevision ?? 0,
			enabled,
			server.installation
		);
		return;
	}

	const [currentBundle, document, serverRecords, policyRecords] = await Promise.all([
		mcpAPI.getMCPBundle(bundle.ref),
		mcpAPI.getMCPBundleDocument(bundle.ref),
		mcpAPI.listMCPBundleServers(bundle.ref),
		mcpAPI.listMCPBundlePolicies(bundle.ref),
	]);

	const registrations = [
		// oxlint-disable-next-line oxc/no-map-spread
		...serverRecords.map(record => ({
			artifactID: record.id,
			subresource: record.binding.subresourceLocator ?? '',
			kind: record.kind,
			enabled: record.id === server.artifact.id ? enabled : record.enabled,
			...(record.id === server.artifact.id ? { data: JSON.stringify(server.installation) } : {}),
		})),
		...policyRecords.map(record => ({
			artifactID: record.id,
			subresource: record.binding.subresourceLocator ?? '',
			kind: record.kind,
			enabled: record.enabled,
		})),
	].filter(registration => registration.subresource);

	await mcpAPI.replaceMCPBundleDocument({
		bundle: bundle.ref,
		expectedCollectionRevision: currentBundle.collection.revision,
		document,
		registrations,
	});
}

export async function setMCPBundleRuntimeEnabled(bundle: MCPBundleView, enabled: boolean): Promise<void> {
	if (bundle.builtIn) {
		await mcpAPI.updateProtectedMCPBundleInstallation(bundle.ref, bundle.installation.overlayRevision, enabled);
		return;
	}

	await mcpAPI.updateMCPBundleEnabled(bundle.ref, bundle.bundle.collection.revision, enabled);
}

export async function deleteMCPServer(bundle: MCPBundleView, server: MCPServerView): Promise<void> {
	if (bundle.builtIn || server.builtIn) {
		throw new Error('Built-in MCP servers cannot be deleted.');
	}

	if (!server.document) {
		throw new Error('The MCP server document is unavailable.');
	}

	const [currentBundle, document, serverRecords, policyRecords] = await Promise.all([
		mcpAPI.getMCPBundle(bundle.ref),
		mcpAPI.getMCPBundleDocument(bundle.ref),
		mcpAPI.listMCPBundleServers(bundle.ref),
		mcpAPI.listMCPBundlePolicies(bundle.ref),
	]);

	const nextDocument = cloneJSON(document);
	nextDocument.mcpServers = omitManyKeys(nextDocument.mcpServers, [server.document.logicalName]);
	if (nextDocument.bundleExtension.servers) {
		nextDocument.bundleExtension.servers = omitManyKeys(nextDocument.bundleExtension.servers, [
			server.document.logicalName,
		]);
	}
	const policyName = server.document.extension.policy?.ref;
	const policyStillReferenced = policyName
		? Object.values(nextDocument.bundleExtension.servers ?? {}).some(extension => extension.policy?.ref === policyName)
		: false;

	if (policyName && !policyStillReferenced) {
		delete nextDocument.bundleExtension.policies?.[policyName];
	}

	const registrations: MCPArtifactRegistration[] = [];

	for (const record of serverRecords) {
		if (record.id === server.artifact.id) {
			continue;
		}

		registrations.push({
			artifactID: record.id,
			subresource: record.binding.subresourceLocator ?? '',
			kind: record.kind,
			enabled: record.enabled,
		});
	}

	for (const record of policyRecords) {
		if (policyName && record.binding.subresourceLocator === policySubresource(policyName) && !policyStillReferenced) {
			continue;
		}

		registrations.push({
			artifactID: record.id,
			subresource: record.binding.subresourceLocator ?? '',
			kind: record.kind,
			enabled: record.enabled,
		});
	}

	await mcpAPI.replaceMCPBundleDocument({
		bundle: bundle.ref,
		expectedCollectionRevision: currentBundle.collection.revision,
		document: nextDocument,
		registrations,
	});
}

export async function deleteMCPBundle(bundle: MCPBundleView): Promise<void> {
	if (bundle.builtIn) {
		throw new Error('Built-in MCP bundles cannot be deleted.');
	}

	const servers = await mcpAPI.listMCPBundleServers(bundle.ref);
	const policies = await mcpAPI.listMCPBundlePolicies(bundle.ref);

	if (servers.length > 0 || policies.length > 0) {
		throw new Error('Remove all MCP server and policy Artifacts before deleting this bundle.');
	}

	const retired = await mcpAPI.retireMCPBundle(bundle.ref, bundle.bundle.collection.revision);

	await mcpAPI.purgeMCPBundle(bundle.ref, retired.revision);
}

export async function applyMCPServerSetup(
	server: MCPServerView,
	values: Record<string, MCPSetupSubmissionValue>,
	reset: boolean
): Promise<void> {
	if (!server.document || !server.installation) {
		throw new Error('The MCP server installation is unavailable.');
	}

	const latest = await mcpAPI.getMCPServerInstallation(server.ref);
	const nextData = cloneJSON(latest.installation);
	nextData.inputs = cloneJSON(nextData.inputs ?? {});
	const inputs = latest.document.extension.install.inputs ?? {};
	const targets = findSecretTargets(latest.document);

	for (const [inputName, declaration] of Object.entries(inputs)) {
		const submitted = values[inputName];
		const existing = nextData.inputs[inputName];

		if (declaration.kind === MCPInputKindEnum.Text || declaration.kind === MCPInputKindEnum.Path) {
			if (submitted?.value?.trim()) {
				nextData.inputs[inputName] = {
					value: submitted.value,
				};
			} else if (reset) {
				nextData.inputs = omitManyKeys(nextData.inputs, [inputName]);
			}
			continue;
		}

		if (declaration.kind === MCPInputKindEnum.OAuthClientCredentials) {
			const hasCredentials = Boolean(submitted?.clientID?.trim() || submitted?.clientSecret);

			if (hasCredentials) {
				if (!submitted?.clientID?.trim()) {
					throw new Error(`OAuth input "${inputName}" requires a client ID.`);
				}

				const secret = JSON.stringify({
					clientID: submitted.clientID.trim(),
					...(submitted.clientSecret ? { clientSecret: submitted.clientSecret } : {}),
				});

				const result = await mcpAPI.putMCPServerSecret(
					server.ref,
					MCPSecretKindEnum.OAuthClientCredentials,
					'clientCredentials',
					secret
				);

				nextData.inputs[inputName] = {
					secretRef: result.secretRef,
				};
			} else if (reset && existing?.secretRef) {
				await mcpAPI.deleteMCPServerSecret(server.ref, MCPSecretKindEnum.OAuthClientCredentials, 'clientCredentials');
				nextData.inputs = omitManyKeys(nextData.inputs, [inputName]);
			}
			continue;
		}

		if (declaration.kind === MCPInputKindEnum.Secret) {
			const target = targets.get(inputName);

			if (!target) {
				throw new Error(`Secret installation input "${inputName}" has no supported environment or HTTP-header target.`);
			}

			if (submitted?.value) {
				const result = await mcpAPI.putMCPServerSecret(server.ref, target.kind, target.slot, submitted.value);

				nextData.inputs[inputName] = {
					secretRef: result.secretRef,
				};
			} else if (reset && existing?.secretRef) {
				await mcpAPI.deleteMCPServerSecret(server.ref, target.kind, target.slot);
				nextData.inputs = omitManyKeys(nextData.inputs, [inputName]);
			}
		}
	}

	if (server.builtIn) {
		await mcpAPI.updateProtectedMCPServerInstallation(
			server.ref,
			latest.installationRevision,
			latest.runtimeEnabled,
			nextData
		);
		return;
	}

	await mcpAPI.updateMCPServerInstallation(server.ref, latest.artifact.revision, nextData);
}

export function isServerOperational(server: MCPServerView): boolean {
	return Boolean(server.document && server.installation && server.artifact.state === ArtifactState.Available);
}

export function serverRefLabel(server: MCPServerView): string {
	return `${server.ref.rootID}/${server.ref.artifactID}`;
}

export function getAuthMode(server: MCPServerView): MCPHTTPAuthMode {
	if (server.document?.mcpServer.type === MCPServerType.Stdio) {
		return MCPHTTPAuthMode.None;
	}

	return server.document?.extension.auth.mode ?? MCPHTTPAuthMode.None;
}

export function getServerAuthHealthState(
	server: MCPServerView,
	health?: MCPAuthHealth
): MCPAuthHealthState | undefined {
	if (getAuthMode(server) === MCPHTTPAuthMode.None) {
		return MCPAuthHealthState.NotRequired;
	}
	return health?.state;
}
