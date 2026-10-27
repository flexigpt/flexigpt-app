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
	AddPluginArtifactMemberRequest,
	AddPluginMemberRequest,
	CreatePluginRequest,
	PluginCapabilityPlan,
	PluginDirectMembership,
	PluginListItem,
	PluginView,
	RemovePluginMemberRequest,
	UpdatePluginRequest,
} from '@/spec/plugin';
import { AgentSkillUseMode } from '@/spec/agent';
import { CapabilityResolutionStatus } from '@/spec/artifact';

import type { IAgentStoreAPI } from '@/apis/interface';
import {
	agentViewFromWails,
	capabilityTargetFromWails,
	pluginCapabilityPlanFromWails,
	pluginDirectMembershipFromWails,
	pluginListItemFromWails,
	pluginResultFromWails,
	pluginViewFromWails,
} from '@/apis/wailsapi/list_item_projection';
import {
	enumFromWails,
	optionalWailsString,
	requiredObject,
	requireWailsBoolean,
	requireWailsString,
	wailsObjectArrayOrEmpty,
} from '@/apis/wailsapi/transport';
import {
	AddAgentPluginArtifactMember,
	AddAgentPluginMember,
	ClearDefaultAgent,
	CommitAgentImport,
	CreateAgentPlugin,
	DeleteAgentPlugin,
	DeleteManagedAgent,
	ExportAgent,
	GetAgent,
	GetAgentPlugin,
	GetDefaultAgent,
	ListAgentImportDestinations,
	ListAgentPluginMembers,
	ListAgentPlugins,
	ListAgentPluginsForManagement,
	ListAgents,
	ListAgentsForManagement,
	PreviewAgentImport,
	RemoveAgentPluginMember,
	ResolveAgent,
	ResolveAgentCapabilities,
	ResolveAgentPluginCapabilities,
	SetAgentEnabled,
	SetAgentPluginEnabled,
	SetDefaultAgent,
	UpdateAgentPlugin,
} from '@/apis/wailsbinding/github.com/flexigpt/flexigpt-app/cmd/agentgo/agentstorewrapper';

function agentCapabilityPlanFromWails(value: unknown, operation: string): AgentCapabilityPlan {
	const plan = requiredObject<Record<string, unknown>>(value, operation);

	return {
		occurrences: wailsObjectArrayOrEmpty(plan.occurrences, `${operation}.occurrences`).map((rawOccurrence, index) => {
			const occurrence = requiredObject<Record<string, unknown>>(rawOccurrence, `${operation}.occurrences[${index}]`);

			return {
				path: requireWailsString(occurrence.path, `${operation}.occurrences[${index}].path`),
				type: requireWailsString(occurrence.type, `${operation}.occurrences[${index}].type`),
				name: optionalWailsString(occurrence.name, `${operation}.occurrences[${index}].name`),
				status: enumFromWails(
					occurrence.status,
					{
						Available: CapabilityResolutionStatus.Available,
						Unavailable: CapabilityResolutionStatus.Unavailable,
						Ambiguous: CapabilityResolutionStatus.Ambiguous,
					} satisfies Record<string, CapabilityResolutionStatus>,
					`${operation}.occurrences[${index}].status`
				),
				required: requireWailsBoolean(occurrence.required, `${operation}.occurrences[${index}].required`),
				target:
					occurrence.target === null || occurrence.target === undefined
						? undefined
						: capabilityTargetFromWails(occurrence.target, `${operation}.occurrences[${index}].target`),
				autoExecute:
					occurrence.autoExecute === null || occurrence.autoExecute === undefined
						? undefined
						: requireWailsBoolean(occurrence.autoExecute, `${operation}.occurrences[${index}].autoExecute`),
				includeSystemPrompt:
					occurrence.includeSystemPrompt === null || occurrence.includeSystemPrompt === undefined
						? undefined
						: requireWailsBoolean(
								occurrence.includeSystemPrompt,
								`${operation}.occurrences[${index}].includeSystemPrompt`
							),
				skillUseMode:
					occurrence.skillUseMode === null || occurrence.skillUseMode === undefined
						? undefined
						: enumFromWails(
								occurrence.skillUseMode,
								AgentSkillUseMode,
								`${operation}.occurrences[${index}].skillUseMode`
							),
				code: optionalWailsString(occurrence.code, `${operation}.occurrences[${index}].code`),
				message: optionalWailsString(occurrence.message, `${operation}.occurrences[${index}].message`),
			};
		}),
		complete: requireWailsBoolean(plan.complete, `${operation}.complete`),
	};
}

function agentResolutionFromWails(value: unknown, operation: string): AgentResolution {
	const resolution = requiredObject<Record<string, unknown>>(value, operation);

	return {
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

	async addAgentPluginMember(request: AddPluginMemberRequest): Promise<PluginView> {
		return pluginViewFromWails(
			await AddAgentPluginMember(request as Parameters<typeof AddAgentPluginMember>[0]),
			'AddAgentPluginMember'
		);
	}

	async addAgentPluginArtifactMember(request: AddPluginArtifactMemberRequest): Promise<PluginView> {
		return pluginViewFromWails(
			await AddAgentPluginArtifactMember(request as Parameters<typeof AddAgentPluginArtifactMember>[0]),
			'AddAgentPluginArtifactMember'
		);
	}

	async removeAgentPluginMember(request: RemovePluginMemberRequest): Promise<PluginView> {
		return pluginViewFromWails(
			await RemoveAgentPluginMember(request as Parameters<typeof RemoveAgentPluginMember>[0]),
			'RemoveAgentPluginMember'
		);
	}

	async resolveAgentPluginCapabilities(plugin: ArtifactRef): Promise<PluginCapabilityPlan> {
		return pluginCapabilityPlanFromWails(
			await ResolveAgentPluginCapabilities(plugin as Parameters<typeof ResolveAgentPluginCapabilities>[0]),
			'ResolveAgentPluginCapabilities'
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

	async clearDefaultAgent(): Promise<void> {
		await ClearDefaultAgent();
	}

	async getDefaultAgent(): Promise<ArtifactRef | null> {
		return requiredObject<ArtifactRef>(await GetDefaultAgent(), 'GetDefaultAgent');
	}

	async setDefaultAgent(ref: ArtifactRef): Promise<void> {
		await SetDefaultAgent(ref as Parameters<typeof SetDefaultAgent>[0]);
	}
}
