import {
	MCPApprovalRule,
	type MCPAppsPolicy,
	MCPAppVisibility,
	type MCPAuthHealth,
	MCPAuthHealthState,
	MCPExecutionMode,
	MCPHTTPAuthMode,
	type MCPServerConfig,
	type MCPServerPolicy,
	type MCPServerRuntimeSnapshot,
	type MCPServerSetupInput,
	MCPServerSetupInputKind,
	MCPServerStatus,
	type MCPToolCapability,
	MCPToolRisk,
	MCPTransportType,
	MCPTrustLevel,
	type PutMCPServerPayload,
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
	if (!visibility || visibility.length === 0) return true;

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

export function getMCPStatusBadgeClass(status?: MCPServerStatus): string {
	switch (status) {
		case MCPServerStatus.MCPServerStatusReady:
			return 'badge-success';
		case MCPServerStatus.MCPServerStatusConnecting:
			return 'badge-info';
		case MCPServerStatus.MCPServerStatusError:
			return 'badge-error';
		case MCPServerStatus.MCPServerStatusDisabled:
			return 'badge-neutral';
		case MCPServerStatus.MCPServerStatusDisconnected:
		default:
			return 'badge-warning';
	}
}

export function getMCPAuthHealthLabel(state?: MCPAuthHealthState): string {
	switch (state) {
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

export function getMCPAuthHealthBadgeClass(state?: MCPAuthHealthState): string {
	switch (state) {
		case MCPAuthHealthState.MCPAuthHealthStateNotRequired:
			return 'badge-ghost';
		case MCPAuthHealthState.MCPAuthHealthStateAuthorized:
			return 'badge-success';
		case MCPAuthHealthState.MCPAuthHealthStateAuthorizationPending:
			return 'badge-info';
		case MCPAuthHealthState.MCPAuthHealthStateAuthorizationNeeded:
		case MCPAuthHealthState.MCPAuthHealthStateExpired:
		case MCPAuthHealthState.MCPAuthHealthStateNotConfigured:
			return 'badge-warning';
		case MCPAuthHealthState.MCPAuthHealthStateInsufficientScope:
		case MCPAuthHealthState.MCPAuthHealthStateError:
			return 'badge-error';
		default:
			return 'badge-neutral';
	}
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
		case MCPToolRisk.MCPToolRiskUnknown:
		default:
			return 'Unknown';
	}
}

export function getMCPToolRiskBadgeClass(risk: MCPToolRisk): string {
	switch (risk) {
		case MCPToolRisk.MCPToolRiskRead:
			return 'badge-success';
		case MCPToolRisk.MCPToolRiskWrite:
			return 'badge-warning';
		case MCPToolRisk.MCPToolRiskDestructive:
		case MCPToolRisk.MCPToolRiskOpenWorld:
			return 'badge-error';
		case MCPToolRisk.MCPToolRiskUnknown:
		default:
			return 'badge-neutral';
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

export function isMCPAuthActionable(authHealth?: MCPAuthHealth): boolean {
	return Boolean(authHealth?.authorizationURL);
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
	if (!map) return false;
	const target = key.trim().toLowerCase();
	return Object.keys(map).some(existing => existing.trim().toLowerCase() === target);
}

export function isMCPSetupInputConfigured(server: MCPServerConfig, input: MCPServerSetupInput): boolean {
	switch (input.kind) {
		case MCPServerSetupInputKind.OAuthClientCredentials:
			return Boolean(server.streamableHttp?.clientCredentialRef?.trim());
		case MCPServerSetupInputKind.HTTPHeader: {
			const name = input.httpHeader?.headerName ?? '';
			if (!name) return false;
			return input.httpHeader?.secret
				? hasMapKeyFold(server.streamableHttp?.secretHeaderRefs, name)
				: hasMapKeyFold(server.streamableHttp?.headers, name);
		}
		case MCPServerSetupInputKind.StdioEnv: {
			const env = input.stdioEnv?.envName ?? '';
			if (!env) return false;
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
			throw new Error(`${fieldLabel} values must all be strings.`);
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
