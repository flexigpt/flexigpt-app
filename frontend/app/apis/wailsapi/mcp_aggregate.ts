import type { ArtifactRef } from '@/spec/artifact';
import type {
	ManagedMCPCreateRequest,
	ManagedMCPCreateResult,
	ManagedMCPPolicyUpsertRequest,
	ManagedMCPPolicyUpsertResult,
	ManagedMCPReplaceRequest,
	ManagedMCPReplaceResult,
	MCPRuntimeServerID,
	MCPServerData,
	MCPServerDetails,
	MCPServerInstallationDataView,
	MCPServerInstallationView,
	MCPServerRuntimeDetails,
} from '@/spec/mcp';

import type { IMCPAggregateAPI } from '@/apis/interface';
import { pluginResultFromWails } from '@/apis/wailsapi/list_item_projection';
import {
	optionalWailsString,
	requiredObject,
	requireWailsBoolean,
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
		return wailsObjectArrayOrEmpty<MCPServerRuntimeDetails>(
			await GetMCPServersForRuntimeServers(servers as Parameters<typeof GetMCPServersForRuntimeServers>[0]),
			'GetMCPServersForRuntimeServers'
		);
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

function serverDetailsFromWails(value: unknown, operation: string): MCPServerDetails {
	const details = requiredObject<MCPServerDetails>(value, operation);
	return {
		...details,
		settings: installationViewFromWails(details.settings, `${operation}.settings`),
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
