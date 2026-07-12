import type {
	MCPAppsPolicy,
	MCPAuthHealth,
	MCPServerConfig,
	MCPServerPolicy,
	MCPServerRuntimeSnapshot,
	MCPServerSetupInput,
	MCPToolCapability,
	PutMCPServerPayload,
} from '@/spec/mcp';
import {
	MCPApprovalRule,
	MCPAppVisibility,
	MCPAuthHealthState,
	MCPExecutionMode,
	MCPHTTPAuthMode,
	MCPServerSetupInputKind,
	MCPServerStatus,
	MCPToolRisk,
	MCPTransportType,
	MCPTrustLevel,
} from '@/spec/mcp';

export const MCP_OAUTH_CLIENT_CREDENTIALS_SLOT = 'clientCredentials';

/**
 * @public
 */
export interface MCPStdioSecretEnvInput {
	envName: string;
	slot: string;
	deleteSlot?: string;
	existingSecretRef?: string;
	secretValue?: string;
	deleteExisting?: boolean;
}

/**
 * @public
 */
export interface MCPOAuthClientCredentialsInput {
	slot: string;
	existingSecretRef?: string;
	secretValue?: string;
	deleteExisting?: boolean;
}

/**
 * @public
 */
export interface MCPHTTPHeaderSecretInput {
	headerName: string;
	slot: string;
	deleteSlot?: string;
	existingSecretRef?: string;
	secretValue?: string;
	deleteExisting?: boolean;
}

export interface MCPServerUpsertInput {
	serverID: string;
	/**
	 * Optional first-pass payload used when the final payload cannot be
	 * validated until secrets are created. Example: new HTTP
	 * clientCredentials servers need a clientCredentialRef, but the secret
	 * endpoint requires the server to already exist.
	 */
	initialPayload?: PutMCPServerPayload;

	payload: PutMCPServerPayload;
	stdioSecretEnv: MCPStdioSecretEnvInput[];
	oauthClientCredentials?: MCPOAuthClientCredentialsInput;
	httpHeaderSecret?: MCPHTTPHeaderSecretInput;
}

export function getDefaultMCPServerPolicy(): MCPServerPolicy {
	return {
		defaultApprovalRule: MCPApprovalRule.MCPApprovalRuleAsk,
		defaultExecutionMode: MCPExecutionMode.MCPExecutionModeManual,
		requireApprovalForUnknownRisk: true,
		requireApprovalForWrite: true,
		requireApprovalForDestructive: true,
	};
}

export function getDefaultMCPAppsPolicy(): MCPAppsPolicy {
	return {
		enabled: false,
		allowAppInitiatedToolCalls: false,
		requireApprovalForOpenLink: true,
		requireApprovalForContextUpdates: true,
	};
}

function isMCPAppVisibilityAllowed(visibility: string[] | undefined, target: 'model' | 'app'): boolean {
	if (!visibility || visibility.length === 0) {
		return true;
	}

	return visibility.some(item => item.trim().toLowerCase() === target);
}

export function isMCPToolVisibleToModel(tool: Pick<MCPToolCapability, 'app'>): boolean {
	return isMCPAppVisibilityAllowed(tool.app?.visibility, MCPAppVisibility.MCPAppVisibilityModel);
}

export function isMCPToolModelSelectable(tool: Pick<MCPToolCapability, 'app' | 'enabled'>): boolean {
	return tool.enabled && isMCPToolVisibleToModel(tool);
}

export function getMCPTransportLabel(transport: MCPTransportType): string {
	switch (transport) {
		case MCPTransportType.MCPTransportTypeStreamableHTTP:
			return 'Streamable HTTP';
		case MCPTransportType.MCPTransportTypeStdio:
			return 'Stdio';
		default:
			return String(transport);
	}
}

export function getMCPTrustLevelLabel(trustLevel: MCPTrustLevel): string {
	switch (trustLevel) {
		case MCPTrustLevel.MCPTrustLevelTrusted:
			return 'Trusted';
		case MCPTrustLevel.MCPTrustLevelUntrusted:
			return 'Untrusted';
		default:
			return String(trustLevel);
	}
}

export function getMCPApprovalRuleLabel(rule: MCPApprovalRule): string {
	switch (rule) {
		case MCPApprovalRule.MCPApprovalRuleAllow:
			return 'Allow';
		case MCPApprovalRule.MCPApprovalRuleAsk:
			return 'Ask';
		case MCPApprovalRule.MCPApprovalRuleDeny:
			return 'Deny';
		default:
			return String(rule);
	}
}

export function getMCPExecutionModeLabel(mode: MCPExecutionMode): string {
	switch (mode) {
		case MCPExecutionMode.MCPExecutionModeAuto:
			return 'Auto';
		case MCPExecutionMode.MCPExecutionModeManual:
			return 'Manual';
		default:
			return String(mode);
	}
}

export function getMCPHTTPAuthModeLabel(mode: MCPHTTPAuthMode): string {
	switch (mode) {
		case MCPHTTPAuthMode.MCPHTTPAuthNone:
			return 'None';
		case MCPHTTPAuthMode.MCPHTTPAuthAPIKey:
			return 'API Key';
		case MCPHTTPAuthMode.MCPHTTPAuthOAuth:
			return 'OAuth';
		case MCPHTTPAuthMode.MCPHTTPAuthClientCredentials:
			return 'Client Credentials';
		default:
			return String(mode);
	}
}

export type MCPAuthDisplayServer = Pick<MCPServerConfig, 'transport' | 'streamableHttp'>;

const MCP_AUTH_HEALTH_STATE_VALUES = new Set<string>(Object.values(MCPAuthHealthState));

function hasRecordEntries(record?: Record<string, string>): boolean {
	return Boolean(record && Object.keys(record).length > 0);
}

function getMCPServerAuthMode(
	server?: MCPAuthDisplayServer,
	authHealth?: Pick<MCPAuthHealth, 'authMode'>
): MCPHTTPAuthMode {
	if (server?.transport === MCPTransportType.MCPTransportTypeStdio) {
		return MCPHTTPAuthMode.MCPHTTPAuthNone;
	}

	return server?.streamableHttp?.authMode ?? authHealth?.authMode ?? MCPHTTPAuthMode.MCPHTTPAuthNone;
}

function isNonInteractiveMCPAuthConfigured(
	server: MCPAuthDisplayServer | undefined,
	authMode: MCPHTTPAuthMode
): boolean {
	if (!server?.streamableHttp) {
		return false;
	}

	switch (authMode) {
		case MCPHTTPAuthMode.MCPHTTPAuthAPIKey:
			// apiKey auth requires an explicit secret header ref. Plain headers
			// may be harmless metadata and must not make a server appear
			// authenticated.
			return hasRecordEntries(server.streamableHttp.secretHeaderRefs);
		case MCPHTTPAuthMode.MCPHTTPAuthClientCredentials:
			return Boolean(server.streamableHttp.clientCredentialRef?.trim());
		default:
			return false;
	}
}

export function getEffectiveMCPAuthHealthState(
	server?: MCPAuthDisplayServer,
	authHealth?: MCPAuthHealth
): MCPAuthHealthState | undefined {
	const authMode = getMCPServerAuthMode(server, authHealth);

	if (server?.transport === MCPTransportType.MCPTransportTypeStdio || authMode === MCPHTTPAuthMode.MCPHTTPAuthNone) {
		return MCPAuthHealthState.MCPAuthHealthStateNotRequired;
	}

	const state = normalizeMCPAuthHealthState(authHealth?.state);

	if (authMode === MCPHTTPAuthMode.MCPHTTPAuthAPIKey || authMode === MCPHTTPAuthMode.MCPHTTPAuthClientCredentials) {
		const configuredByServer = isNonInteractiveMCPAuthConfigured(server, authMode);
		const nonInteractiveConfigured = server ? configuredByServer : authHealth?.configured === true;

		// Config is authoritative for non-interactive auth. Do not let a stale
		// "authorized" health response make an unconfigured server look ready.
		if (!configuredByServer && authHealth?.configured !== true) {
			return MCPAuthHealthState.MCPAuthHealthStateNotConfigured;
		}

		if (
			state === MCPAuthHealthState.MCPAuthHealthStateError ||
			state === MCPAuthHealthState.MCPAuthHealthStateExpired ||
			state === MCPAuthHealthState.MCPAuthHealthStateInsufficientScope
		) {
			return state;
		}

		return nonInteractiveConfigured
			? MCPAuthHealthState.MCPAuthHealthStateAuthorized
			: MCPAuthHealthState.MCPAuthHealthStateNotConfigured;
	}

	if (authMode === MCPHTTPAuthMode.MCPHTTPAuthOAuth && !authHealth) {
		return MCPAuthHealthState.MCPAuthHealthStateAuthorizationNeeded;
	}

	if (
		authMode === MCPHTTPAuthMode.MCPHTTPAuthOAuth &&
		authHealth &&
		!authHealth.configured &&
		state === MCPAuthHealthState.MCPAuthHealthStateAuthorized
	) {
		return MCPAuthHealthState.MCPAuthHealthStateAuthorizationNeeded;
	}

	if (
		authHealth?.authorizationPending &&
		state !== MCPAuthHealthState.MCPAuthHealthStateAuthorized &&
		state !== MCPAuthHealthState.MCPAuthHealthStateNotRequired
	) {
		return MCPAuthHealthState.MCPAuthHealthStateAuthorizationPending;
	}

	if (authHealth && !authHealth.configured && !state) {
		return MCPAuthHealthState.MCPAuthHealthStateNotConfigured;
	}

	return state;
}

function normalizeMCPAuthHealthState(state?: MCPAuthHealthState | string): MCPAuthHealthState | undefined {
	if (!state) {
		return undefined;
	}

	if (MCP_AUTH_HEALTH_STATE_VALUES.has(state)) {
		return state as MCPAuthHealthState;
	}

	switch (state) {
		case 'required':
			return MCPAuthHealthState.MCPAuthHealthStateAuthorizationNeeded;
		default:
			return undefined;
	}
}

export function getMCPStatusLabel(status?: MCPServerStatus): string {
	switch (status) {
		case MCPServerStatus.MCPServerStatusDisabled:
			return 'Disabled';
		case MCPServerStatus.MCPServerStatusDisconnected:
			return 'Disconnected';
		case MCPServerStatus.MCPServerStatusConnecting:
			return 'Connecting';
		case MCPServerStatus.MCPServerStatusReady:
			return 'Ready';
		case MCPServerStatus.MCPServerStatusError:
			return 'Error';
		default:
			return 'Unknown';
	}
}

const STATUS_BADGE_LAYOUT = 'h-auto max-w-full whitespace-normal break-words px-2 py-1 text-center leading-tight';

export function getMCPStatusBadgeClass(status?: MCPServerStatus): string {
	switch (status) {
		case MCPServerStatus.MCPServerStatusReady:
			return `${STATUS_BADGE_LAYOUT} badge-success`;
		case MCPServerStatus.MCPServerStatusConnecting:
			return `${STATUS_BADGE_LAYOUT} badge-info`;
		case MCPServerStatus.MCPServerStatusError:
			return `${STATUS_BADGE_LAYOUT} badge-error`;
		case MCPServerStatus.MCPServerStatusDisabled:
			return `${STATUS_BADGE_LAYOUT} badge-neutral`;
		default:
			return `${STATUS_BADGE_LAYOUT} badge-warning`;
	}
}

function getMCPAuthHealthLabel(state?: MCPAuthHealthState | string): string {
	const normalizedState = normalizeMCPAuthHealthState(state);

	switch (normalizedState) {
		case MCPAuthHealthState.MCPAuthHealthStateNotRequired:
			return 'Auth: not required';
		case MCPAuthHealthState.MCPAuthHealthStateNotConfigured:
			return 'Auth: config needed';
		case MCPAuthHealthState.MCPAuthHealthStateAuthorizationNeeded:
			return 'Auth: required';
		case MCPAuthHealthState.MCPAuthHealthStateAuthorizationPending:
			return 'Auth: pending';
		case MCPAuthHealthState.MCPAuthHealthStateAuthorized:
			return 'Auth: authorized';
		case MCPAuthHealthState.MCPAuthHealthStateExpired:
			return 'Auth: expired';
		case MCPAuthHealthState.MCPAuthHealthStateInsufficientScope:
			return 'Auth: insufficient scope';
		case MCPAuthHealthState.MCPAuthHealthStateError:
			return 'Auth: error';
		default:
			return 'Auth: unknown';
	}
}

function getMCPAuthHealthBadgeClass(state?: MCPAuthHealthState | string): string {
	const normalizedState = normalizeMCPAuthHealthState(state);

	switch (normalizedState) {
		case MCPAuthHealthState.MCPAuthHealthStateNotRequired:
			return `${STATUS_BADGE_LAYOUT} badge-ghost`;
		case MCPAuthHealthState.MCPAuthHealthStateAuthorized:
			return `${STATUS_BADGE_LAYOUT} badge-success`;
		case MCPAuthHealthState.MCPAuthHealthStateAuthorizationPending:
			return `${STATUS_BADGE_LAYOUT} badge-info`;
		case MCPAuthHealthState.MCPAuthHealthStateAuthorizationNeeded:
		case MCPAuthHealthState.MCPAuthHealthStateExpired:
		case MCPAuthHealthState.MCPAuthHealthStateNotConfigured:
			return `${STATUS_BADGE_LAYOUT} badge-warning`;
		case MCPAuthHealthState.MCPAuthHealthStateInsufficientScope:
		case MCPAuthHealthState.MCPAuthHealthStateError:
			return `${STATUS_BADGE_LAYOUT} badge-error`;
		default:
			return `${STATUS_BADGE_LAYOUT} badge-neutral`;
	}
}

export function getMCPServerAuthHealthLabel(server?: MCPAuthDisplayServer, authHealth?: MCPAuthHealth): string {
	const authMode = getMCPServerAuthMode(server, authHealth);
	const state = getEffectiveMCPAuthHealthState(server, authHealth);

	if (
		(authMode === MCPHTTPAuthMode.MCPHTTPAuthAPIKey || authMode === MCPHTTPAuthMode.MCPHTTPAuthClientCredentials) &&
		state === MCPAuthHealthState.MCPAuthHealthStateAuthorized
	) {
		return 'Auth: configured';
	}

	if (authMode === MCPHTTPAuthMode.MCPHTTPAuthOAuth && state === MCPAuthHealthState.MCPAuthHealthStateAuthorized) {
		return 'OAuth: authorized';
	}

	return getMCPAuthHealthLabel(state);
}

export function getMCPServerAuthHealthBadgeClass(server?: MCPAuthDisplayServer, authHealth?: MCPAuthHealth): string {
	const authMode = getMCPServerAuthMode(server, authHealth);
	const state = getEffectiveMCPAuthHealthState(server, authHealth);

	if (
		(authMode === MCPHTTPAuthMode.MCPHTTPAuthAPIKey || authMode === MCPHTTPAuthMode.MCPHTTPAuthClientCredentials) &&
		state === MCPAuthHealthState.MCPAuthHealthStateAuthorized
	) {
		return `${STATUS_BADGE_LAYOUT} badge-ghost`;
	}

	return getMCPAuthHealthBadgeClass(state);
}

export function getMCPToolRiskLabel(risk: MCPToolRisk): string {
	switch (risk) {
		case MCPToolRisk.MCPToolRiskRead:
			return 'Read';
		case MCPToolRisk.MCPToolRiskWrite:
			return 'Write';
		case MCPToolRisk.MCPToolRiskDestructive:
			return 'Destructive';
		case MCPToolRisk.MCPToolRiskOpenWorld:
			return 'Open World';
		default:
			return 'Unknown';
	}
}

export function getEffectiveMCPServerStatus(
	serverEnabled: boolean,
	bundleEnabled: boolean,
	runtime?: MCPServerRuntimeSnapshot
): MCPServerStatus {
	if (!bundleEnabled || !serverEnabled) {
		return MCPServerStatus.MCPServerStatusDisabled;
	}

	return runtime?.status ?? MCPServerStatus.MCPServerStatusDisconnected;
}

export function isMCPAuthActionable(authHealth?: MCPAuthHealth, server?: MCPAuthDisplayServer): boolean {
	if (!authHealth?.authorizationURL) {
		return false;
	}

	if (getMCPServerAuthMode(server, authHealth) !== MCPHTTPAuthMode.MCPHTTPAuthOAuth) {
		return false;
	}

	const state = getEffectiveMCPAuthHealthState(server, authHealth);

	return (
		state !== MCPAuthHealthState.MCPAuthHealthStateNotRequired &&
		state !== MCPAuthHealthState.MCPAuthHealthStateAuthorized
	);
}

export function serverHasSetupInputs(server: Pick<MCPServerConfig, 'setup'>): boolean {
	return (server.setup?.inputs?.length ?? 0) > 0;
}

export function getMCPSetupInputKindLabel(kind: MCPServerSetupInputKind): string {
	switch (kind) {
		case MCPServerSetupInputKind.OAuthClientCredentials:
			return 'OAuth client credentials';
		case MCPServerSetupInputKind.HTTPHeader:
			return 'HTTP header';
		case MCPServerSetupInputKind.StdioEnv:
			return 'Environment variable';
		case MCPServerSetupInputKind.StreamableHTTPURL:
			return 'Server URL';
		case MCPServerSetupInputKind.ClientIDMetadataDocumentURL:
			return 'Client ID metadata URL';
		default:
			return String(kind);
	}
}

function hasMapKeyFold(map: Record<string, string> | undefined, key: string): boolean {
	if (!map) {
		return false;
	}
	const target = key.trim().toLowerCase();
	return Object.keys(map).some(existing => existing.trim().toLowerCase() === target);
}

export function isMCPSetupInputConfigured(server: MCPServerConfig, input: MCPServerSetupInput): boolean {
	switch (input.kind) {
		case MCPServerSetupInputKind.OAuthClientCredentials:
			return Boolean(server.streamableHttp?.clientCredentialRef?.trim());
		case MCPServerSetupInputKind.HTTPHeader: {
			const name = input.httpHeader?.headerName ?? '';
			if (!name) {
				return false;
			}
			return input.httpHeader?.secret
				? hasMapKeyFold(server.streamableHttp?.secretHeaderRefs, name)
				: hasMapKeyFold(server.streamableHttp?.headers, name);
		}
		case MCPServerSetupInputKind.StdioEnv: {
			const env = input.stdioEnv?.envName ?? '';
			if (!env) {
				return false;
			}
			return input.stdioEnv?.secret ? Boolean(server.stdio?.secretEnvRefs?.[env]) : Boolean(server.stdio?.env?.[env]);
		}
		case MCPServerSetupInputKind.StreamableHTTPURL:
			return Boolean(server.streamableHttp?.url?.trim());
		case MCPServerSetupInputKind.ClientIDMetadataDocumentURL:
			return Boolean(server.streamableHttp?.clientIDMetadataDocumentURL?.trim());
		default:
			return false;
	}
}

export interface MCPServerSetupStatus {
	hasInputs: boolean;
	requiredTotal: number;
	requiredConfigured: number;
	firstUnconfiguredRequired?: MCPServerSetupInput;
	complete: boolean;
}

/**
 * Summarizes setup readiness purely from the persisted server config. This is
 * intentionally distinct from auth health: setup covers user-supplied config
 * inputs (URLs, env, headers, OAuth client credentials), while auth health
 * covers the live OAuth/token state.
 */
export function getMCPServerSetupStatus(server: MCPServerConfig): MCPServerSetupStatus {
	const inputs = server.setup?.inputs ?? [];
	const required = inputs.filter(input => Boolean(input.required));
	const firstUnconfiguredRequired = required.find(input => !isMCPSetupInputConfigured(server, input));

	return {
		hasInputs: inputs.length > 0,
		requiredTotal: required.length,
		requiredConfigured: required.filter(input => isMCPSetupInputConfigured(server, input)).length,
		firstUnconfiguredRequired,
		complete: !firstUnconfiguredRequired,
	};
}

export function parseMCPStringRecordJSON(raw: string, fieldLabel: string): Record<string, string> | undefined {
	const value = raw.trim();

	if (!value) {
		return undefined;
	}

	let parsed: unknown;

	try {
		parsed = JSON.parse(value);
	} catch {
		throw new Error(`${fieldLabel} must be valid JSON.`);
	}

	if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
		throw new Error(`${fieldLabel} must be a JSON object.`);
	}

	const result: Record<string, string> = {};

	for (const [key, val] of Object.entries(parsed as Record<string, unknown>)) {
		if (typeof val !== 'string') {
			throw new TypeError(`${fieldLabel} values must all be strings.`);
		}

		result[key] = val;
	}

	return Object.keys(result).length > 0 ? result : undefined;
}

export function parseMCPObjectJSON(raw: string, fieldLabel: string): Record<string, unknown> | undefined {
	const value = raw.trim();

	if (!value) {
		return undefined;
	}

	let parsed: unknown;

	try {
		parsed = JSON.parse(value);
	} catch {
		throw new Error(`${fieldLabel} must be valid JSON.`);
	}

	if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
		throw new Error(`${fieldLabel} must be a JSON object.`);
	}

	return parsed as Record<string, unknown>;
}

export function stringifyMCPJSON(value: unknown): string {
	if (!value || (typeof value === 'object' && !Array.isArray(value) && Object.keys(value).length === 0)) {
		return '';
	}

	return JSON.stringify(value, null, 2);
}
