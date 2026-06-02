import {
	MCPApprovalRule,
	type MCPAppsPolicy,
	type MCPAuthHealth,
	MCPAuthHealthState,
	MCPExecutionMode,
	MCPHTTPAuthMode,
	MCPServerAvailability,
	type MCPServerPolicy,
	type MCPServerRuntimeSnapshot,
	MCPServerStatus,
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

export function getMCPAvailabilityLabel(availability: MCPServerAvailability): string {
	switch (availability) {
		case MCPServerAvailability.MCPServerAvailabilityAutoAttach:
			return 'Auto Attach';
		case MCPServerAvailability.MCPServerAvailabilityManual:
			return 'Manual';
		default:
			return String(availability);
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
			return 'Not Required';
		case MCPAuthHealthState.MCPAuthHealthStateNotConfigured:
			return 'Not Configured';
		case MCPAuthHealthState.MCPAuthHealthStateAuthorizationNeeded:
			return 'Authorization Needed';
		case MCPAuthHealthState.MCPAuthHealthStateAuthorizationPending:
			return 'Authorization Pending';
		case MCPAuthHealthState.MCPAuthHealthStateAuthorized:
			return 'Authorized';
		case MCPAuthHealthState.MCPAuthHealthStateExpired:
			return 'Expired';
		case MCPAuthHealthState.MCPAuthHealthStateInsufficientScope:
			return 'Insufficient Scope';
		case MCPAuthHealthState.MCPAuthHealthStateError:
			return 'Error';
		default:
			return 'Unknown';
	}
}

export function getMCPAuthHealthBadgeClass(state?: MCPAuthHealthState): string {
	switch (state) {
		case MCPAuthHealthState.MCPAuthHealthStateNotRequired:
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

// eslint-disable-next-line @typescript-eslint/no-unnecessary-type-parameters
export function parseMCPObjectJSON<T extends Record<string, unknown>>(raw: string, fieldLabel: string): T | undefined {
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

	return parsed as T;
}

export function stringifyMCPJSON(value: unknown): string {
	if (!value || (typeof value === 'object' && !Array.isArray(value) && Object.keys(value).length === 0)) {
		return '';
	}

	return JSON.stringify(value, null, 2);
}
