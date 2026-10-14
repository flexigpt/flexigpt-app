import type { ArtifactRef } from '@/spec/artifact';
import type {
	ManagedMCPCreateRequest,
	ManagedMCPCreateResult,
	ManagedMCPPolicyUpsertRequest,
	ManagedMCPPolicyUpsertResult,
	ManagedMCPReplaceRequest,
	ManagedMCPReplaceResult,
	MCPServerData,
	MCPServerDetails,
	MCPServerInstallationDataView,
	MCPServerInstallationView,
} from '@/spec/mcp';
import type {
	MCPAuthHealth,
	MCPAuthStatus,
	MCPOAuthAuthorization,
	MCPRuntimeServerID,
	MCPServerRuntimeDetails,
	MCPServerRuntimeSnapshot,
} from '@/spec/session_mcp';
import { MCPHTTPAuthMode } from '@/spec/mcp';
import { MCPAuthHealthState, MCPAuthState, MCPServerStatus } from '@/spec/session_mcp';

import type { IMCPAggregateAPI } from '@/apis/interface';
import { pluginResultFromWails } from '@/apis/wailsapi/list_item_projection';
import {
	enumFromWails,
	optionalWailsString,
	requiredObject,
	requireWailsBoolean,
	wailsArrayOrEmpty,
	wailsObjectArrayOrEmpty,
	wailsRecordOrEmpty,
} from '@/apis/wailsapi/transport';
import {
	ClearMCPServerSecret,
	CreateMCPServer,
	DeleteMCPPolicy,
	DeleteMCPServer,
	GetMCPServer,
	GetMCPServersForRuntimeServers,
	ListMCPPluginServers,
	SaveMCPPolicy,
	SaveMCPServerSettings,
	SetMCPServerSecret,
	UpdateMCPServer,
} from '@/apis/wailsjs/go/main/MCPAggregateWrapper';

export class WailsMCPAggregateAPI implements IMCPAggregateAPI {
	async getMCPServer(server: ArtifactRef): Promise<MCPServerDetails> {
		return serverDetailsFromWails(await GetMCPServer(server as Parameters<typeof GetMCPServer>[0]), 'GetMCPServer');
	}

	async listMCPPluginServers(plugin: ArtifactRef): Promise<MCPServerDetails[]> {
		return wailsObjectArrayOrEmpty(
			await ListMCPPluginServers(plugin as Parameters<typeof ListMCPPluginServers>[0]),
			'ListMCPPluginServers'
		).map((value, index) => serverDetailsFromWails(value, `ListMCPPluginServers[${index}]`));
	}

	async getMCPServersForRuntimeServers(servers: MCPRuntimeServerID[]): Promise<MCPServerRuntimeDetails[]> {
		return wailsObjectArrayOrEmpty(
			await GetMCPServersForRuntimeServers(servers as Parameters<typeof GetMCPServersForRuntimeServers>[0]),
			'GetMCPServersForRuntimeServers'
		).map((value, index) => mcpServerRuntimeDetailsFromWails(value, `GetMCPServersForRuntimeServers[${index}]`));
	}

	async createMCPServer(request: ManagedMCPCreateRequest): Promise<ManagedMCPCreateResult> {
		return pluginResultFromWails<ManagedMCPCreateResult>(
			await CreateMCPServer(request as Parameters<typeof CreateMCPServer>[0]),
			'CreateMCPServer'
		);
	}

	async updateMCPServer(request: ManagedMCPReplaceRequest): Promise<ManagedMCPReplaceResult> {
		return pluginResultFromWails<ManagedMCPReplaceResult>(
			await UpdateMCPServer(request as Parameters<typeof UpdateMCPServer>[0]),
			'UpdateMCPServer'
		);
	}

	async deleteMCPServer(server: ArtifactRef, expectedRevision: number): Promise<void> {
		await DeleteMCPServer(server as Parameters<typeof DeleteMCPServer>[0], expectedRevision);
	}

	async saveMCPPolicy(request: ManagedMCPPolicyUpsertRequest): Promise<ManagedMCPPolicyUpsertResult> {
		return pluginResultFromWails<ManagedMCPPolicyUpsertResult>(
			await SaveMCPPolicy(request as Parameters<typeof SaveMCPPolicy>[0]),
			'SaveMCPPolicy'
		);
	}

	async deleteMCPPolicy(policy: ArtifactRef, expectedRevision: number): Promise<void> {
		await DeleteMCPPolicy(policy as Parameters<typeof DeleteMCPPolicy>[0], expectedRevision);
	}

	async saveMCPServerSettings(
		server: ArtifactRef,
		expectedSettingsRevision: number,
		data: MCPServerData
	): Promise<MCPServerDetails> {
		return serverDetailsFromWails(
			await SaveMCPServerSettings(
				server as Parameters<typeof SaveMCPServerSettings>[0],
				expectedSettingsRevision,
				data as Parameters<typeof SaveMCPServerSettings>[2]
			),
			'SaveMCPServerSettings'
		);
	}

	async setMCPServerSecret(server: ArtifactRef, input: string, secret: string): Promise<MCPServerDetails> {
		return serverDetailsFromWails(
			await SetMCPServerSecret(server as Parameters<typeof SetMCPServerSecret>[0], input, secret),
			'SetMCPServerSecret'
		);
	}

	async clearMCPServerSecret(server: ArtifactRef, input: string): Promise<MCPServerDetails> {
		return serverDetailsFromWails(
			await ClearMCPServerSecret(server as Parameters<typeof ClearMCPServerSecret>[0], input),
			'ClearMCPServerSecret'
		);
	}
}

function mcpServerRuntimeSnapshotFromWails(value: unknown, operation: string): MCPServerRuntimeSnapshot {
	const snapshot = requiredObject<MCPServerRuntimeSnapshot>(value, operation);

	return {
		...snapshot,
		status: enumFromWails(snapshot.status, MCPServerStatus, `${operation}.status`),
	};
}

function mcpAuthStatusFromWails(value: unknown, operation: string): MCPAuthStatus {
	const status = requiredObject<MCPAuthStatus>(value, operation);

	return {
		...status,
		authMode: enumFromWails(status.authMode, MCPHTTPAuthMode, `${operation}.authMode`),
		state: enumFromWails(status.state, MCPAuthState, `${operation}.state`),
		scopes: wailsArrayOrEmpty<string>(status.scopes, `${operation}.scopes`),
	};
}

function mcpOAuthAuthorizationFromWails(value: unknown, operation: string): MCPOAuthAuthorization {
	return requiredObject<MCPOAuthAuthorization>(value, operation);
}

function mcpAuthHealthFromWails(value: unknown, operation: string): MCPAuthHealth {
	const health = requiredObject<MCPAuthHealth>(value, operation);

	return {
		...health,
		authMode: enumFromWails(health.authMode, MCPHTTPAuthMode, `${operation}.authMode`),
		state: enumFromWails(health.state, MCPAuthHealthState, `${operation}.state`),
		scopes: wailsArrayOrEmpty<string>(health.scopes, `${operation}.scopes`),
	};
}

function mcpServerRuntimeDetailsFromWails(value: unknown, operation: string): MCPServerRuntimeDetails {
	const details = requiredObject<MCPServerRuntimeDetails>(value, operation);

	return {
		...details,
		connection: mcpServerRuntimeSnapshotFromWails(details.connection, `${operation}.connection`),
		authorization:
			details.authorization === undefined
				? undefined
				: mcpAuthStatusFromWails(details.authorization, `${operation}.authorization`),
		pendingAuthorization:
			details.pendingAuthorization === undefined
				? undefined
				: mcpOAuthAuthorizationFromWails(details.pendingAuthorization, `${operation}.pendingAuthorization`),
	};
}

function serverDetailsFromWails(value: unknown, operation: string): MCPServerDetails {
	const details = requiredObject<MCPServerDetails>(value, operation);

	return {
		...details,
		settings: installationViewFromWails(details.settings, `${operation}.settings`),
		authorization: mcpAuthHealthFromWails(details.authorization, `${operation}.authorization`),
		connection: mcpServerRuntimeSnapshotFromWails(details.connection, `${operation}.connection`),
	};
}

function installationViewFromWails(value: unknown, operation: string): MCPServerInstallationView {
	const view = requiredObject<MCPServerInstallationView>(value, operation);
	const rawInstallation = requiredObject<Record<string, unknown>>(view.installation, `${operation}.installation`);
	const rawInputs = wailsRecordOrEmpty(rawInstallation.inputs, `${operation}.installation.inputs`);
	const inputs = Object.fromEntries(
		Object.entries(rawInputs).map(([name, rawInput]) => {
			const input = requiredObject<Record<string, unknown>>(rawInput, `${operation}.installation.inputs.${name}`);
			return [
				name,
				{
					value: optionalWailsString(input.value, `${operation}.installation.inputs.${name}.value`),
					secretConfigured: requireWailsBoolean(
						input.secretConfigured,
						`${operation}.installation.inputs.${name}.secretConfigured`
					),
				},
			] as const;
		})
	);

	return {
		...view,
		installation: {
			selectedConnectionProfile: optionalWailsString(
				rawInstallation.selectedConnectionProfile,
				`${operation}.installation.selectedConnectionProfile`
			),
			inputs,
			additionalPolicies: wailsObjectArrayOrEmpty(
				rawInstallation.additionalPolicies,
				`${operation}.installation.additionalPolicies`
			),
		} satisfies MCPServerInstallationDataView,
	};
}
