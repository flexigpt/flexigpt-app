import type {
	AgentCapabilityPlan,
	AgentExportResult,
	AgentImportCommitRequest,
	AgentImportCommitResult,
	AgentImportDestination,
	AgentImportPreview,
	AgentImportPreviewRequest,
	AgentResolution,
	AgentView,
	ListAgentsRequest,
} from '@/spec/agent';
import type { ArtifactRef, ArtifactRootID } from '@/spec/artifact';
import type {
	CreatePluginRequest,
	PluginDirectMembership,
	PluginListItem,
	PluginView,
	UpdatePluginRequest,
} from '@/spec/plugin';

import type { IAgentStoreAPI } from '@/apis/interface';
import {
	agentViewFromWails,
	capabilityTargetFromWails,
	pluginDirectMembershipFromWails,
	pluginListItemFromWails,
	pluginResultFromWails,
	pluginViewFromWails,
} from '@/apis/wailsapi/list_item_projection';
import { requiredObject, wailsObjectArrayOrEmpty } from '@/apis/wailsapi/transport';
import {
	CommitAgentImport,
	CreateAgentPlugin,
	DeleteAgentPlugin,
	DeleteManagedAgent,
	ExportAgent,
	GetAgent,
	GetAgentPlugin,
	ListAgentImportDestinations,
	ListAgentPluginMembers,
	ListAgentPlugins,
	ListAgentPluginsForManagement,
	ListAgents,
	ListAgentsForManagement,
	PreviewAgentImport,
	ResolveAgent,
	ResolveAgentCapabilities,
	SetAgentEnabled,
	SetAgentPluginEnabled,
	UpdateAgentPlugin,
} from '@/apis/wailsjs/go/main/AgentStoreWrapper';

function agentCapabilityPlanFromWails(value: unknown, operation: string): AgentCapabilityPlan {
	const plan = requiredObject<AgentCapabilityPlan>(value, operation);

	return {
		...plan,
		occurrences: wailsObjectArrayOrEmpty<AgentCapabilityPlan['occurrences'][number]>(
			plan.occurrences,
			`${operation}.occurrences`
		).map((occurrence, index) => ({
			...occurrence,
			target:
				occurrence.target === null || occurrence.target === undefined
					? undefined
					: capabilityTargetFromWails(occurrence.target, `${operation}.occurrences[${index}].target`),
		})),
	};
}

function agentResolutionFromWails(value: unknown, operation: string): AgentResolution {
	const resolution = requiredObject<AgentResolution>(value, operation);

	return {
		...resolution,
		agent: agentViewFromWails(resolution.agent, `${operation}.agent`),
		capabilities: agentCapabilityPlanFromWails(resolution.capabilities, `${operation}.capabilities`),
	};
}

export class WailsAgentStoreAPI implements IAgentStoreAPI {
	async listAgents(request: ListAgentsRequest): Promise<AgentView[]> {
		return wailsObjectArrayOrEmpty(await ListAgents(request as Parameters<typeof ListAgents>[0]), 'ListAgents').map(
			(value, index) => agentViewFromWails(value, `ListAgents[${index}]`)
		);
	}

	async listAgentsForManagement(): Promise<AgentView[]> {
		return wailsObjectArrayOrEmpty(await ListAgentsForManagement(), 'ListAgentsForManagement').map((value, index) =>
			agentViewFromWails(value, `ListAgentsForManagement[${index}]`)
		);
	}

	async getAgent(agent: ArtifactRef): Promise<AgentView> {
		return agentViewFromWails(await GetAgent(agent as Parameters<typeof GetAgent>[0]), 'GetAgent');
	}

	async resolveAgent(agent: ArtifactRef): Promise<AgentResolution> {
		return agentResolutionFromWails(await ResolveAgent(agent as Parameters<typeof ResolveAgent>[0]), 'ResolveAgent');
	}

	async resolveAgentCapabilities(agent: ArtifactRef): Promise<AgentCapabilityPlan> {
		return agentCapabilityPlanFromWails(
			await ResolveAgentCapabilities(agent as Parameters<typeof ResolveAgentCapabilities>[0]),
			'ResolveAgentCapabilities'
		);
	}

	async setAgentEnabled(agent: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<AgentView> {
		return agentViewFromWails(
			await SetAgentEnabled(agent as Parameters<typeof SetAgentEnabled>[0], expectedRevision, enabled),
			'SetAgentEnabled'
		);
	}

	async listAgentPluginMembers(plugin: ArtifactRef): Promise<PluginDirectMembership> {
		return pluginDirectMembershipFromWails(
			await ListAgentPluginMembers(plugin as Parameters<typeof ListAgentPluginMembers>[0]),
			'ListAgentPluginMembers'
		);
	}

	async createAgentPlugin(request: CreatePluginRequest): Promise<PluginView> {
		return pluginViewFromWails(
			await CreateAgentPlugin(request as Parameters<typeof CreateAgentPlugin>[0]),
			'CreateAgentPlugin'
		);
	}

	async getAgentPlugin(plugin: ArtifactRef): Promise<PluginView> {
		return pluginViewFromWails(await GetAgentPlugin(plugin as Parameters<typeof GetAgentPlugin>[0]), 'GetAgentPlugin');
	}

	async listAgentPlugins(rootID: ArtifactRootID): Promise<PluginListItem[]> {
		return wailsObjectArrayOrEmpty(
			await ListAgentPlugins(rootID as Parameters<typeof ListAgentPlugins>[0]),
			'ListAgentPlugins'
		).map((value, index) => pluginListItemFromWails(value, `ListAgentPlugins[${index}]`));
	}

	async listAgentPluginsForManagement(): Promise<PluginListItem[]> {
		return wailsObjectArrayOrEmpty(await ListAgentPluginsForManagement(), 'ListAgentPluginsForManagement').map(
			(value, index) => pluginListItemFromWails(value, `ListAgentPluginsForManagement[${index}]`)
		);
	}

	async updateAgentPlugin(request: UpdatePluginRequest): Promise<PluginView> {
		return pluginViewFromWails(
			await UpdateAgentPlugin(request as Parameters<typeof UpdateAgentPlugin>[0]),
			'UpdateAgentPlugin'
		);
	}

	async setAgentPluginEnabled(plugin: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<PluginView> {
		return pluginViewFromWails(
			await SetAgentPluginEnabled(plugin as Parameters<typeof SetAgentPluginEnabled>[0], expectedRevision, enabled),
			'SetAgentPluginEnabled'
		);
	}

	async deleteAgentPlugin(plugin: ArtifactRef, expectedRevision: number): Promise<void> {
		await DeleteAgentPlugin(plugin as Parameters<typeof DeleteAgentPlugin>[0], expectedRevision);
	}

	async listAgentImportDestinations(): Promise<AgentImportDestination[]> {
		return wailsObjectArrayOrEmpty<AgentImportDestination>(
			await ListAgentImportDestinations(),
			'ListAgentImportDestinations'
		);
	}

	async previewAgentImport(request: AgentImportPreviewRequest): Promise<AgentImportPreview> {
		return requiredObject<AgentImportPreview>(
			await PreviewAgentImport(request as Parameters<typeof PreviewAgentImport>[0]),
			'PreviewAgentImport'
		);
	}

	async commitAgentImport(request: AgentImportCommitRequest): Promise<AgentImportCommitResult> {
		const result = pluginResultFromWails<AgentImportCommitResult>(
			await CommitAgentImport(request as Parameters<typeof CommitAgentImport>[0]),
			'CommitAgentImport'
		);

		return {
			...result,
			agent: agentViewFromWails(result.agent, 'CommitAgentImport.agent'),
		};
	}

	async exportAgent(agent: ArtifactRef): Promise<AgentExportResult> {
		return requiredObject<AgentExportResult>(
			await ExportAgent({ agent } as Parameters<typeof ExportAgent>[0]),
			'ExportAgent'
		);
	}

	async deleteManagedAgent(agent: ArtifactRef, expectedRevision: number): Promise<void> {
		await DeleteManagedAgent({
			agent,
			expectedRevision,
		} as Parameters<typeof DeleteManagedAgent>[0]);
	}
}
