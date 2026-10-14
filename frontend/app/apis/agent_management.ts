// oxlint-disable typescript/parameter-properties
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
import { ArtifactState, CapabilityTargetForm } from '@/spec/artifact';

import { throwIfAborted } from '@/lib/async_utils';
import { getErrorMessage } from '@/lib/error_utils';
import { createSharedAsyncCatalog } from '@/lib/shared_async_catalog';

import type { IAgentStoreAPI } from '@/apis/interface';

export interface AgentPluginData {
	plugin: PluginListItem;
	agents: AgentView[];
	agentsLoaded: boolean;
	isLoadingAgents: boolean;
	importDestination?: AgentImportDestination;
	agentLoadError?: string;
}

export interface AgentManagementPageData {
	plugins: AgentPluginData[];
	importDestinations: AgentImportDestination[];
}

export const EMPTY_AGENT_MANAGEMENT_PAGE_DATA: AgentManagementPageData = {
	plugins: [],
	importDestinations: [],
};

export interface AgentCatalogOption {
	key: string;
	ref: ArtifactRef;
	agent: AgentView;
	displayName: string;
	description?: string;
	label: string;
	isSelectable: boolean;
	availabilityReason?: string;
}

function artifactRefKey(ref: ArtifactRef): string {
	return `${ref.rootID}:${ref.artifactID}`;
}

function pluginRef(plugin: PluginListItem): ArtifactRef {
	return plugin.ref;
}

function comparePlugins(left: PluginListItem, right: PluginListItem): number {
	if (left.builtIn !== right.builtIn) {
		return left.builtIn ? -1 : 1;
	}

	const displayNameCompare = pluginDisplayName(left).localeCompare(pluginDisplayName(right), undefined, {
		sensitivity: 'base',
	});

	if (displayNameCompare !== 0) {
		return displayNameCompare;
	}

	return agentPluginKey(left).localeCompare(agentPluginKey(right));
}

function capabilityTargetArtifactRef(
	target: PluginDirectMembership['members'][number]['target']
): ArtifactRef | undefined {
	if (target?.form !== CapabilityTargetForm.Artifact) {
		return undefined;
	}

	return target.artifact;
}

export function agentArtifactRefKey(ref: ArtifactRef): string {
	return artifactRefKey(ref);
}

export function agentArtifactRef(agent: AgentView): ArtifactRef {
	return agent.ref;
}

export function agentPluginKey(plugin: PluginListItem): string {
	return artifactRefKey(pluginRef(plugin));
}

export function agentDisplayName(agent: AgentView): string {
	return agent.displayName || agent.name;
}

export function pluginDisplayName(plugin: Pick<PluginListItem, 'displayName' | 'name'>): string {
	return plugin.displayName || plugin.name;
}

export function isBuiltInAgentPlugin(plugin: PluginListItem): boolean {
	return plugin.builtIn;
}

export function canEditAgentPluginMetadata(plugin: PluginListItem): boolean {
	return plugin.editable && !plugin.baseline;
}

export function canDeleteAgentPlugin(plugin: PluginListItem): boolean {
	return plugin.deletable && !plugin.baseline;
}

/**
 * Frontend management façade for Agent artifacts and Agent Plugins.
 *
 * Session execution does not use this façade as authority. Session runtime
 * setup independently resolves Agent selections from backend-owned artifact
 * state when preparing a model operation.
 */
export class AgentManagementAPI {
	private readonly pluginAgentLoads = new Map<string, Promise<AgentView[]>>();

	private readonly agentCatalog = createSharedAsyncCatalog<AgentView[]>(async () => {
		return (await this.agents.listAgentsForManagement()) ?? [];
	});

	constructor(private readonly agents: IAgentStoreAPI) {}

	private async mutate<T>(operation: () => Promise<T>): Promise<T> {
		const result = await operation();
		this.invalidateAgentCatalog();
		return result;
	}

	invalidateAgentCatalog(): void {
		this.pluginAgentLoads.clear();
		this.agentCatalog.invalidate();
	}

	async preloadAgentCatalog(force = false): Promise<void> {
		await this.agentCatalog.load(force);
	}

	listAgentCatalogOptions(force = false): Promise<AgentCatalogOption[]> {
		return this.agentCatalog.load(force).then(agents => this.projectAgentCatalogOptions(agents));
	}

	async loadManagementPageData(signal: AbortSignal): Promise<AgentManagementPageData> {
		const [pluginResult, destinationResult] = await Promise.all([
			this.agents.listAgentPluginsForManagement(),
			this.agents.listAgentImportDestinations(),
		]);
		throwIfAborted(signal);

		const pluginsByKey = new Map<string, PluginListItem>();

		for (const plugin of pluginResult ?? []) {
			pluginsByKey.set(agentPluginKey(plugin), plugin);
		}

		const missingDestinationPlugins = (destinationResult ?? []).filter(
			destination => !pluginsByKey.has(artifactRefKey(destination.plugin))
		);

		if (missingDestinationPlugins.length > 0) {
			const hydrated = await Promise.all(
				missingDestinationPlugins.map(async destination => {
					const plugin = await this.agents.getAgentPlugin(destination.plugin);

					return {
						ref: {
							rootID: plugin.artifact.rootID,
							artifactID: plugin.artifact.id,
						},
						sourceID: plugin.artifact.binding.sourceID,
						name: plugin.name,
						displayName: plugin.displayName,
						description: plugin.description,
						state: plugin.artifact.state,
						enabled: plugin.artifact.enabled,
						revision: plugin.artifact.revision,
						memberCount: plugin.members.length,
						builtIn: false,
						editable: plugin.editable,
						deletable: plugin.deletable,
						baseline: plugin.baseline,
					} satisfies PluginListItem;
				})
			);
			throwIfAborted(signal);

			for (const plugin of hydrated) {
				pluginsByKey.set(agentPluginKey(plugin), plugin);
			}
		}

		const importDestinations = destinationResult ?? [];
		const destinationByPluginKey = new Map(
			importDestinations.map(destination => [artifactRefKey(destination.plugin), destination] as const)
		);

		return {
			plugins: [...pluginsByKey.values()]
				.map(plugin => ({
					plugin,
					agents: [],
					agentsLoaded: false,
					isLoadingAgents: false,
					importDestination: destinationByPluginKey.get(artifactRefKey(plugin.ref)),
				}))
				.toSorted((left, right) => comparePlugins(left.plugin, right.plugin)),
			importDestinations: [...importDestinations].toSorted((left, right) => {
				const rootCompare = (left.rootDisplayName || left.rootID).localeCompare(
					right.rootDisplayName || right.rootID,
					undefined,
					{
						sensitivity: 'base',
					}
				);

				if (rootCompare !== 0) {
					return rootCompare;
				}

				return left.pluginDisplayName.localeCompare(right.pluginDisplayName, undefined, {
					sensitivity: 'base',
				});
			}),
		};
	}

	async loadPluginAgents(
		plugin: PluginListItem,
		signal: AbortSignal
	): Promise<Pick<AgentPluginData, 'agents' | 'agentsLoaded' | 'agentLoadError'>> {
		const key = agentPluginKey(plugin);
		let load = this.pluginAgentLoads.get(key);

		if (!load) {
			load = Promise.all([this.agents.listAgentPluginMembers(plugin.ref), this.agentCatalog.load(false)]).then(
				([membership, catalog]) => {
					const selectedRefs = new Set<string>();

					for (const member of membership.members) {
						const direct = capabilityTargetArtifactRef(member.target);
						if (member.entry.type === 'agent' && direct) {
							selectedRefs.add(artifactRefKey(direct));
						}

						for (const selectorMatch of member.selectorMatches ?? []) {
							const selected = capabilityTargetArtifactRef(selectorMatch.target);
							if (member.entry.type === 'agent' && selected) {
								selectedRefs.add(artifactRefKey(selected));
							}
						}
					}

					return catalog
						.filter(agent => selectedRefs.has(artifactRefKey(agent.ref)))
						.toSorted((left, right) => {
							const displayCompare = agentDisplayName(left).localeCompare(agentDisplayName(right), undefined, {
								sensitivity: 'base',
							});

							if (displayCompare !== 0) {
								return displayCompare;
							}

							return artifactRefKey(left.ref).localeCompare(artifactRefKey(right.ref));
						});
				}
			);
			this.pluginAgentLoads.set(key, load);

			const clear = () => {
				if (this.pluginAgentLoads.get(key) === load) {
					this.pluginAgentLoads.delete(key);
				}
			};
			void load.then(clear, clear);
		}

		try {
			const agents = await load;
			throwIfAborted(signal);

			return {
				agents,
				agentsLoaded: true,
			};
		} catch (error) {
			throwIfAborted(signal);

			return {
				agents: [],
				agentsLoaded: false,
				agentLoadError: getErrorMessage(error, 'Agents could not be loaded for this Plugin.'),
			};
		}
	}

	listAgents(request: ListAgentsRequest): Promise<AgentView[]> {
		return this.agents.listAgents(request);
	}

	getAgent(agent: ArtifactRef): Promise<AgentView> {
		return this.agents.getAgent(agent);
	}

	resolveAgent(agent: ArtifactRef): Promise<AgentResolution> {
		return this.agents.resolveAgent(agent);
	}

	resolveAgentCapabilities(agent: ArtifactRef): Promise<AgentCapabilityPlan> {
		return this.agents.resolveAgentCapabilities(agent);
	}

	setAgentEnabled(agent: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<AgentView> {
		return this.mutate(() => this.agents.setAgentEnabled(agent, expectedRevision, enabled));
	}

	addAgentPluginMember(request: AddPluginMemberRequest): Promise<PluginView> {
		return this.mutate(() => this.agents.addAgentPluginMember(request));
	}

	addAgentPluginArtifactMember(request: AddPluginArtifactMemberRequest): Promise<PluginView> {
		return this.mutate(() => this.agents.addAgentPluginArtifactMember(request));
	}

	removeAgentPluginMember(request: RemovePluginMemberRequest): Promise<PluginView> {
		return this.mutate(() => this.agents.removeAgentPluginMember(request));
	}

	resolveAgentPluginCapabilities(plugin: ArtifactRef): Promise<PluginCapabilityPlan> {
		return this.agents.resolveAgentPluginCapabilities(plugin);
	}

	listAgentPluginMembers(plugin: ArtifactRef): Promise<PluginDirectMembership> {
		return this.agents.listAgentPluginMembers(plugin);
	}

	createAgentPlugin(request: CreatePluginRequest): Promise<PluginView> {
		return this.mutate(() => this.agents.createAgentPlugin(request));
	}

	getAgentPlugin(plugin: ArtifactRef): Promise<PluginView> {
		return this.agents.getAgentPlugin(plugin);
	}

	listAgentPlugins(rootID: ArtifactRootID): Promise<PluginListItem[]> {
		return this.agents.listAgentPlugins(rootID);
	}

	listAgentPluginsForManagement(): Promise<PluginListItem[]> {
		return this.agents.listAgentPluginsForManagement();
	}

	updateAgentPlugin(request: UpdatePluginRequest): Promise<PluginView> {
		return this.mutate(() => this.agents.updateAgentPlugin(request));
	}

	setAgentPluginEnabled(plugin: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<PluginView> {
		return this.mutate(() => this.agents.setAgentPluginEnabled(plugin, expectedRevision, enabled));
	}

	deleteAgentPlugin(plugin: ArtifactRef, expectedRevision: number): Promise<void> {
		return this.mutate(() => this.agents.deleteAgentPlugin(plugin, expectedRevision));
	}

	listAgentImportDestinations(): Promise<AgentImportDestination[]> {
		return this.agents.listAgentImportDestinations();
	}

	previewAgentImport(request: AgentImportPreviewRequest): Promise<AgentImportPreview> {
		return this.agents.previewAgentImport(request);
	}

	commitAgentImport(request: AgentImportCommitRequest): Promise<AgentImportCommitResult> {
		return this.mutate(() => this.agents.commitAgentImport(request));
	}

	exportAgent(agent: ArtifactRef): Promise<AgentExportResult> {
		return this.agents.exportAgent(agent);
	}

	deleteManagedAgent(agent: ArtifactRef, expectedRevision: number): Promise<void> {
		return this.mutate(() => this.agents.deleteManagedAgent(agent, expectedRevision));
	}

	private projectAgentCatalogOptions(agents: AgentView[]): AgentCatalogOption[] {
		return agents
			.map(agent => {
				const stateAvailable = agent.state === ArtifactState.Available;
				const isSelectable = stateAvailable && agent.enabled;
				let availabilityReason: string | undefined;

				if (!stateAvailable) {
					availabilityReason = `Agent Artifact is ${agent.state}.`;
				} else if (!agent.enabled) {
					availabilityReason = 'Agent is disabled.';
				}

				const displayName = agentDisplayName(agent);

				return {
					key: artifactRefKey(agent.ref),
					ref: agent.ref,
					agent,
					displayName,
					description: agent.description,
					label: displayName === agent.name ? displayName : `${displayName} (${agent.name})`,
					isSelectable,
					availabilityReason,
				};
			})
			.toSorted((left, right) => {
				if (left.agent.builtIn !== right.agent.builtIn) {
					return left.agent.builtIn ? -1 : 1;
				}

				const displayCompare = left.displayName.localeCompare(right.displayName, undefined, {
					sensitivity: 'base',
				});

				if (displayCompare !== 0) {
					return displayCompare;
				}

				return left.key.localeCompare(right.key);
			});
	}
}
