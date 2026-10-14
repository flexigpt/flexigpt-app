import type { MCPServerView } from '@/spec/mcp';
import type { MCPAuthHealth, MCPToolCapability } from '@/spec/session_mcp';
import { MCPApprovalRule, MCPExecutionMode, MCPHTTPAuthMode, MCPTransportType, MCPTrustLevel } from '@/spec/mcp';
import { MCPAppVisibility, MCPAuthHealthState, MCPServerStatus, MCPToolRisk } from '@/spec/session_mcp';

import { getAuthMode, getMCPServerSetupStatus, getServerAuthHealthState } from '@/apis/mcp_management';

export function getMCPTransportLabel(value: MCPTransportType): string {
	switch (value) {
		case MCPTransportType.Stdio:
			return 'Stdio';
		case MCPTransportType.StreamableHTTP:
			return 'Streamable HTTP';
		default:
			return String(value);
	}
}

export function getMCPTrustLevelLabel(value: MCPTrustLevel): string {
	return value === MCPTrustLevel.Trusted ? 'Trusted' : 'Untrusted';
}

export function getMCPHTTPAuthModeLabel(value: MCPHTTPAuthMode): string {
	switch (value) {
		case MCPHTTPAuthMode.None:
			return 'None';
		case MCPHTTPAuthMode.APIKey:
			return 'API Key';
		case MCPHTTPAuthMode.OAuth:
			return 'OAuth';
		case MCPHTTPAuthMode.ClientCredentials:
			return 'Client Credentials';
		default:
			return String(value);
	}
}

export function getMCPApprovalRuleLabel(value: MCPApprovalRule): string {
	switch (value) {
		case MCPApprovalRule.Allow:
			return 'Allow';
		case MCPApprovalRule.Deny:
			return 'Deny';
		default:
			return 'Ask';
	}
}

export function getMCPExecutionModeLabel(value: MCPExecutionMode): string {
	return value === MCPExecutionMode.Auto ? 'Auto' : 'Manual';
}

export function getMCPToolRiskLabel(value: MCPToolRisk): string {
	switch (value) {
		case MCPToolRisk.Read:
			return 'Read';
		case MCPToolRisk.Write:
			return 'Write';
		case MCPToolRisk.Destructive:
			return 'Destructive';
		case MCPToolRisk.OpenWorld:
			return 'Open World';
		default:
			return 'Unknown';
	}
}

export function getEffectiveMCPServerStatus(server: MCPServerView, runtimeStatus?: MCPServerStatus): MCPServerStatus {
	if (!server.enabled) {
		return MCPServerStatus.Disabled;
	}

	return runtimeStatus ?? MCPServerStatus.Disconnected;
}

export function isMCPServerConfiguredForUse(server: MCPServerView, authHealth?: MCPAuthHealth): boolean {
	const setup = getMCPServerSetupStatus(server);

	if (!setup.complete) {
		return false;
	}

	const authMode = getAuthMode(server);

	if (authMode === MCPHTTPAuthMode.None) {
		return true;
	}

	if (authMode === MCPHTTPAuthMode.APIKey || authMode === MCPHTTPAuthMode.ClientCredentials) {
		return authHealth?.configured === true;
	}

	return getServerAuthHealthState(server, authHealth) !== MCPAuthHealthState.NotConfigured;
}

export function isMCPServerConnectionPending(
	server: MCPServerView,
	runtimeStatus?: MCPServerStatus,
	authHealth?: MCPAuthHealth
): boolean {
	return (
		runtimeStatus === MCPServerStatus.Connecting ||
		authHealth?.authorizationPending === true ||
		getServerAuthHealthState(server, authHealth) === MCPAuthHealthState.AuthorizationPending
	);
}

export function getMCPStatusLabel(value: MCPServerStatus): string {
	switch (value) {
		case MCPServerStatus.Disabled:
			return 'Disabled';
		case MCPServerStatus.Disconnected:
			return 'Disconnected';
		case MCPServerStatus.Connecting:
			return 'Connecting';
		case MCPServerStatus.Ready:
			return 'Ready';
		case MCPServerStatus.Error:
			return 'Error';
		default:
			return 'Unknown';
	}
}

const STATUS_BADGE_LAYOUT = 'h-auto max-w-full whitespace-normal break-words px-2 py-1 text-center leading-tight';

export function getMCPStatusBadgeClass(value: MCPServerStatus): string {
	switch (value) {
		case MCPServerStatus.Ready:
			return `${STATUS_BADGE_LAYOUT} badge-success`;
		case MCPServerStatus.Connecting:
			return `${STATUS_BADGE_LAYOUT} badge-info`;
		case MCPServerStatus.Error:
			return `${STATUS_BADGE_LAYOUT} badge-error`;
		case MCPServerStatus.Disabled:
			return `${STATUS_BADGE_LAYOUT} badge-neutral`;
		default:
			return `${STATUS_BADGE_LAYOUT} badge-warning`;
	}
}

export function getMCPServerAuthHealthLabel(server: MCPServerView, authHealth?: MCPAuthHealth): string {
	const authMode = getAuthMode(server);
	const state = getServerAuthHealthState(server, authHealth);

	if (authMode === MCPHTTPAuthMode.None) {
		return 'Auth: not required';
	}

	if (
		(authMode === MCPHTTPAuthMode.APIKey || authMode === MCPHTTPAuthMode.ClientCredentials) &&
		authHealth?.configured
	) {
		return state === MCPAuthHealthState.Error ? 'Auth: error' : 'Auth: configured';
	}

	switch (state) {
		case MCPAuthHealthState.NotConfigured:
			return 'Auth: config needed';
		case MCPAuthHealthState.AuthorizationNeeded:
			return 'OAuth: required';
		case MCPAuthHealthState.AuthorizationPending:
			return 'OAuth: pending';
		case MCPAuthHealthState.Authorized:
			return 'OAuth: authorized';
		case MCPAuthHealthState.Expired:
			return 'OAuth: expired';
		case MCPAuthHealthState.InsufficientScope:
			return 'Auth: insufficient scope';
		case MCPAuthHealthState.Error:
			return 'Auth: error';
		default:
			return 'Auth: unknown';
	}
}

export function getMCPServerAuthHealthBadgeClass(server: MCPServerView, authHealth?: MCPAuthHealth): string {
	const state = getServerAuthHealthState(server, authHealth);

	switch (state) {
		case MCPAuthHealthState.NotRequired:
			return `${STATUS_BADGE_LAYOUT} badge-ghost`;
		case MCPAuthHealthState.Authorized:
			return `${STATUS_BADGE_LAYOUT} badge-success`;
		case MCPAuthHealthState.AuthorizationPending:
			return `${STATUS_BADGE_LAYOUT} badge-info`;
		case MCPAuthHealthState.AuthorizationNeeded:
		case MCPAuthHealthState.NotConfigured:
		case MCPAuthHealthState.Expired:
			return `${STATUS_BADGE_LAYOUT} badge-warning`;
		case MCPAuthHealthState.Error:
		case MCPAuthHealthState.InsufficientScope:
			return `${STATUS_BADGE_LAYOUT} badge-error`;
		default:
			return `${STATUS_BADGE_LAYOUT} badge-neutral`;
	}
}

export function isMCPAuthActionable(server: MCPServerView, authHealth?: MCPAuthHealth): boolean {
	if (getAuthMode(server) !== MCPHTTPAuthMode.OAuth) {
		return false;
	}

	if (!authHealth?.authorizationURL?.trim()) {
		return false;
	}

	return authHealth.state !== MCPAuthHealthState.Authorized && authHealth.state !== MCPAuthHealthState.NotRequired;
}

export function isMCPToolVisibleToModel(tool: Pick<MCPToolCapability, 'app'>): boolean {
	const visibility = tool.app?.visibility;

	if (!visibility || visibility.length === 0) {
		return true;
	}

	return visibility.includes(MCPAppVisibility.Model);
}

export function isMCPToolModelSelectable(tool: Pick<MCPToolCapability, 'enabled' | 'app'>): boolean {
	return tool.enabled && isMCPToolVisibleToModel(tool);
}
