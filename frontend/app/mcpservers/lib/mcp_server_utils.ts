import type {
	MCPAuthHealth,
	MCPHTTPAuthMode,
	MCPServerStatus,
	MCPToolCapability,
	MCPToolRisk,
	MCPTransportType,
} from '@/spec/mcp_artifact';
import {
	MCPApprovalRule,
	MCPAppVisibility,
	MCPAuthHealthState as MCPAuthHealthStateEnum,
	MCPExecutionMode,
	MCPHTTPAuthMode as MCPHTTPAuthModeEnum,
	MCPServerStatus as MCPServerStatusEnum,
	MCPToolRisk as MCPToolRiskEnum,
	MCPTransportType as MCPTransportTypeEnum,
	MCPTrustLevel,
} from '@/spec/mcp_artifact';

import type { MCPServerView } from '@/mcpservers/lib/mcp_management';
import { getAuthMode, getServerAuthHealthState } from '@/mcpservers/lib/mcp_management';

export function getMCPTransportLabel(value: MCPTransportType): string {
	switch (value) {
		case MCPTransportTypeEnum.Stdio:
			return 'Stdio';
		case MCPTransportTypeEnum.StreamableHTTP:
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
		case MCPHTTPAuthModeEnum.None:
			return 'None';
		case MCPHTTPAuthModeEnum.APIKey:
			return 'API Key';
		case MCPHTTPAuthModeEnum.OAuth:
			return 'OAuth';
		case MCPHTTPAuthModeEnum.ClientCredentials:
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
		case MCPToolRiskEnum.Read:
			return 'Read';
		case MCPToolRiskEnum.Write:
			return 'Write';
		case MCPToolRiskEnum.Destructive:
			return 'Destructive';
		case MCPToolRiskEnum.OpenWorld:
			return 'Open World';
		default:
			return 'Unknown';
	}
}

export function getEffectiveMCPServerStatus(server: MCPServerView, runtimeStatus?: MCPServerStatus): MCPServerStatus {
	if (!server.runtimeEnabled) {
		return MCPServerStatusEnum.Disabled;
	}

	return runtimeStatus ?? MCPServerStatusEnum.Disconnected;
}

export function getMCPStatusLabel(value: MCPServerStatus): string {
	switch (value) {
		case MCPServerStatusEnum.Disabled:
			return 'Disabled';
		case MCPServerStatusEnum.Disconnected:
			return 'Disconnected';
		case MCPServerStatusEnum.Connecting:
			return 'Connecting';
		case MCPServerStatusEnum.Ready:
			return 'Ready';
		case MCPServerStatusEnum.Error:
			return 'Error';
		default:
			return 'Unknown';
	}
}

const STATUS_BADGE_LAYOUT = 'h-auto max-w-full whitespace-normal break-words px-2 py-1 text-center leading-tight';

export function getMCPStatusBadgeClass(value: MCPServerStatus): string {
	switch (value) {
		case MCPServerStatusEnum.Ready:
			return `${STATUS_BADGE_LAYOUT} badge-success`;
		case MCPServerStatusEnum.Connecting:
			return `${STATUS_BADGE_LAYOUT} badge-info`;
		case MCPServerStatusEnum.Error:
			return `${STATUS_BADGE_LAYOUT} badge-error`;
		case MCPServerStatusEnum.Disabled:
			return `${STATUS_BADGE_LAYOUT} badge-neutral`;
		default:
			return `${STATUS_BADGE_LAYOUT} badge-warning`;
	}
}

export function getMCPServerAuthHealthLabel(server: MCPServerView, authHealth?: MCPAuthHealth): string {
	const authMode = getAuthMode(server);
	const state = getServerAuthHealthState(server, authHealth);

	if (authMode === MCPHTTPAuthModeEnum.None) {
		return 'Auth: not required';
	}

	if (
		(authMode === MCPHTTPAuthModeEnum.APIKey || authMode === MCPHTTPAuthModeEnum.ClientCredentials) &&
		authHealth?.configured
	) {
		return state === MCPAuthHealthStateEnum.Error ? 'Auth: error' : 'Auth: configured';
	}

	switch (state) {
		case MCPAuthHealthStateEnum.NotConfigured:
			return 'Auth: config needed';
		case MCPAuthHealthStateEnum.AuthorizationNeeded:
			return 'OAuth: required';
		case MCPAuthHealthStateEnum.AuthorizationPending:
			return 'OAuth: pending';
		case MCPAuthHealthStateEnum.Authorized:
			return 'OAuth: authorized';
		case MCPAuthHealthStateEnum.Expired:
			return 'OAuth: expired';
		case MCPAuthHealthStateEnum.InsufficientScope:
			return 'Auth: insufficient scope';
		case MCPAuthHealthStateEnum.Error:
			return 'Auth: error';
		default:
			return 'Auth: unknown';
	}
}

export function getMCPServerAuthHealthBadgeClass(server: MCPServerView, authHealth?: MCPAuthHealth): string {
	const state = getServerAuthHealthState(server, authHealth);

	switch (state) {
		case MCPAuthHealthStateEnum.NotRequired:
			return `${STATUS_BADGE_LAYOUT} badge-ghost`;
		case MCPAuthHealthStateEnum.Authorized:
			return `${STATUS_BADGE_LAYOUT} badge-success`;
		case MCPAuthHealthStateEnum.AuthorizationPending:
			return `${STATUS_BADGE_LAYOUT} badge-info`;
		case MCPAuthHealthStateEnum.AuthorizationNeeded:
		case MCPAuthHealthStateEnum.NotConfigured:
		case MCPAuthHealthStateEnum.Expired:
			return `${STATUS_BADGE_LAYOUT} badge-warning`;
		case MCPAuthHealthStateEnum.Error:
		case MCPAuthHealthStateEnum.InsufficientScope:
			return `${STATUS_BADGE_LAYOUT} badge-error`;
		default:
			return `${STATUS_BADGE_LAYOUT} badge-neutral`;
	}
}

export function isMCPAuthActionable(server: MCPServerView, authHealth?: MCPAuthHealth): boolean {
	if (getAuthMode(server) !== MCPHTTPAuthModeEnum.OAuth) {
		return false;
	}

	if (!authHealth?.authorizationURL?.trim()) {
		return false;
	}

	return (
		authHealth.state !== MCPAuthHealthStateEnum.Authorized && authHealth.state !== MCPAuthHealthStateEnum.NotRequired
	);
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
