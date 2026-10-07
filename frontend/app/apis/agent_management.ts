// oxlint-disable typescript/parameter-properties
import type { AgentCapabilityOccurrence, AgentImportDestination, AgentResolution, AgentView } from '@/spec/agent';
import type { ArtifactRef, CapabilityTarget } from '@/spec/artifact';
import type { MCPConversationContext, MCPRuntimeServerID } from '@/spec/mcp';
import type { PluginListItem, PluginView } from '@/spec/plugin';
import type { ArtifactSkillSummary, RuntimeSkillRenderResult, SkillRef } from '@/spec/skill';
import type { TextMaterialization } from '@/spec/text';
import type { ToolStoreChoice } from '@/spec/tool';
import { AgentSkillUseMode } from '@/spec/agent';
import { ArtifactState, CapabilityResolutionStatus, CapabilityTargetForm } from '@/spec/artifact';
import { MCPToolExposure } from '@/spec/mcp';
import { SkillInsert } from '@/spec/skill';
import { TextInsert } from '@/spec/text';
import { ToolImplType } from '@/spec/tool';

import { throwIfAborted } from '@/lib/async_utils';
import { getErrorMessage } from '@/lib/error_utils';
import { createSharedAsyncCatalog } from '@/lib/shared_async_catalog';
import { getUUIDv7 } from '@/lib/uuid_utils';

import type { IAgentStoreAPI, IToolTargetResolver } from '@/apis/interface';
import type { ModelManagementAPI } from '@/apis/model_management';
import { toolStoreChoiceFromSelection } from '@/apis/tool_management';

import { toolIdentityKey } from '@/tools/lib/tool_identity_utils';

type AgentStarterIssueSeverity = 'error' | 'warning';

type AgentPlugin = PluginListItem | PluginView;

interface AgentMCPRuntimeResolver {
	resolveMCPRuntimeServerIDs(artifacts: ArtifactRef[]): Promise<Map<string, MCPRuntimeServerID>>;
}

interface AgentSkillResolver {
	describeArtifactSkill(skill: ArtifactRef): Promise<ArtifactSkillSummary>;
	renderArtifactSkill(skill: ArtifactRef, args?: Record<string, string>): Promise<RuntimeSkillRenderResult>;
}

interface AgentTextResolver {
	materializeText(text: ArtifactRef): Promise<TextMaterialization>;
}

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

interface AgentPreparedToolSelection {
	choice: ToolStoreChoice;
	requiredSDKType?: string;
	occurrencePath: string;
}

export interface AgentPreparedInstructionSource {
	kind: 'skill' | 'text';
	artifact: ArtifactRef;
	displayName: string;
	text: string;
	sourceTags?: string[];
}

interface AgentStarterIssue {
	severity: AgentStarterIssueSeverity;
	code: string;
	message: string;
	path?: string;
}

export interface PreparedAgentStarter {
	agent: AgentView;
	resolution: AgentResolution;

	modelRef?: ArtifactRef;
	includeModelSystemPrompt?: boolean;

	toolSelections: AgentPreparedToolSelection[];
	enabledSkillRefs: SkillRef[];
	activeSkillRefs: SkillRef[];
	instructionSkillRefs: SkillRef[];
	instructionSources: AgentPreparedInstructionSource[];
	instructionText: string;
	startingText: string;

	mcpContext?: MCPConversationContext;

	/**
	 * Text Artifacts materialized while preparing the Agent. Instruction and
	 * user-message Text is already projected into `instructionText` and
	 * `startingText`; warning and information Text remains diagnostic-only.
	 */
	textArtifacts: ArtifactRef[];

	issues: AgentStarterIssue[];
	canApply: boolean;
}

function artifactRefKey(ref: ArtifactRef): string {
	return `${ref.rootID}:${ref.artifactID}`;
}

function capabilityTargetLabel(target: CapabilityTarget): string {
	if (target.form === CapabilityTargetForm.Artifact && target.artifact) {
		return `${target.name} (${target.artifact.rootID}/${target.artifact.artifactID})`;
	}

	return `${target.name} (${target.providerIdentity ?? 'unknown'}/${target.providerLocalID ?? 'unknown'})`;
}

function artifactRefFromTarget(target?: CapabilityTarget): ArtifactRef | undefined {
	if (target?.form !== CapabilityTargetForm.Artifact) {
		return undefined;
	}

	return target.artifact;
}

function pluginRef(plugin: AgentPlugin): ArtifactRef {
	if ('ref' in plugin) {
		return plugin.ref;
	}

	return {
		rootID: plugin.artifact.rootID,
		artifactID: plugin.artifact.id,
	};
}

function pluginBuiltIn(plugin: AgentPlugin): boolean {
	if ('builtIn' in plugin) {
		return plugin.builtIn;
	}

	return !plugin.baseline && !plugin.editable && !plugin.deletable;
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

function issue(
	issues: AgentStarterIssue[],
	severity: AgentStarterIssueSeverity,
	code: string,
	message: string,
	path?: string
): void {
	issues.push({
		severity,
		code,
		message,
		path,
	});
}

function isAvailableOccurrence(occurrence: AgentCapabilityOccurrence): boolean {
	return occurrence.status === CapabilityResolutionStatus.Available;
}

function availabilityIssueForOccurrence(occurrence: AgentCapabilityOccurrence): AgentStarterIssue {
	return {
		severity: occurrence.required ? 'error' : 'warning',
		code: occurrence.code || `agent.recipe.${occurrence.status}`,
		message:
			occurrence.message ||
			`${occurrence.type}${occurrence.name ? ` "${occurrence.name}"` : ''} is ${occurrence.status}.`,
		path: occurrence.path,
	};
}

function skillUseMode(occurrence: AgentCapabilityOccurrence): AgentSkillUseMode {
	return occurrence.skillUseMode ?? AgentSkillUseMode.Available;
}

function toolAutoExecute(occurrence: AgentCapabilityOccurrence, fallback: boolean): boolean {
	return occurrence.autoExecute ?? fallback;
}

export function agentArtifactRefKey(ref: ArtifactRef): string {
	return artifactRefKey(ref);
}

export function agentArtifactRef(agent: AgentView): ArtifactRef {
	return agent.ref;
}

export function agentPluginKey(plugin: AgentPlugin): string {
	return artifactRefKey(pluginRef(plugin));
}

export function agentDisplayName(agent: AgentView): string {
	return agent.displayName || agent.name;
}

export function pluginDisplayName(plugin: Pick<AgentPlugin, 'displayName' | 'name'>): string {
	return plugin.displayName || plugin.name;
}

export function isBuiltInAgentPlugin(plugin: AgentPlugin): boolean {
	return pluginBuiltIn(plugin);
}

export function canEditAgentPluginMetadata(plugin: AgentPlugin): boolean {
	return plugin.editable && !plugin.baseline;
}

export function canDeleteAgentPlugin(plugin: AgentPlugin): boolean {
	return plugin.deletable && !plugin.baseline;
}

export class AgentManagementAPI {
	private readonly pluginAgentLoads = new Map<string, Promise<AgentView[]>>();

	private readonly composerAgentCatalog = createSharedAsyncCatalog<AgentView[]>(async () => {
		return (await this.agents.listAgentsForManagement()) ?? [];
	});

	constructor(
		private readonly agents: IAgentStoreAPI,
		private readonly tools: IToolTargetResolver,
		private readonly models: ModelManagementAPI,
		private readonly mcp: AgentMCPRuntimeResolver,
		private readonly skills: AgentSkillResolver,
		private readonly texts: AgentTextResolver
	) {}

	invalidateAgentCatalog(): void {
		this.pluginAgentLoads.clear();
		this.composerAgentCatalog.invalidate();
	}

	async preloadAgentCatalog(force = false): Promise<void> {
		await this.composerAgentCatalog.load(force);
	}

	listAgentCatalogOptions(force = false): Promise<AgentCatalogOption[]> {
		return this.composerAgentCatalog.load(force).then(agents => this.projectAgentCatalogOptions(agents));
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
						ref: pluginRef(plugin),
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

	/**
	 * Plugin-filtered `ListAgents` cannot include built-ins from another
	 * Root. Resolve the Plugin membership graph and join its ArtifactRefs
	 * against the shared management catalog instead. This retains direct
	 * cross-Root built-in memberships without issuing one `GetAgent` per row.
	 */
	async loadPluginAgents(
		plugin: PluginListItem,
		signal: AbortSignal
	): Promise<Pick<AgentPluginData, 'agents' | 'agentsLoaded' | 'agentLoadError'>> {
		const key = agentPluginKey(plugin);
		let load = this.pluginAgentLoads.get(key);

		if (!load) {
			load = Promise.all([this.agents.listAgentPluginMembers(plugin.ref), this.composerAgentCatalog.load(false)]).then(
				([membership, catalog]) => {
					const selectedRefs = new Set<string>();

					for (const member of membership.members) {
						if (member.entry.type !== 'agent') {
							continue;
						}

						const ref = artifactRefFromTarget(member.target);
						if (ref) {
							selectedRefs.add(artifactRefKey(ref));
						}
						for (const match of member.selectorMatches ?? []) {
							const selected = artifactRefFromTarget(match.target);
							if (selected) {
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

	async prepareAgentStarter(agentRef: ArtifactRef): Promise<PreparedAgentStarter> {
		return this.prepareAgentStarterFromResolution(await this.agents.resolveAgent(agentRef));
	}

	async prepareAgentStarterFromResolution(resolution: AgentResolution): Promise<PreparedAgentStarter> {
		const issues: AgentStarterIssue[] = [];
		const toolSelections = new Map<string, AgentPreparedToolSelection>();
		const enabledSkillRefs = new Map<string, SkillRef>();
		const activeSkillRefs = new Map<string, SkillRef>();
		const instructionSkillRefs = new Map<string, SkillRef>();
		const instructionSources = new Map<string, AgentPreparedInstructionSource>();
		const textArtifacts = new Map<string, ArtifactRef>();
		const mcpServerIDs = new Set<MCPRuntimeServerID>();
		const mcpArtifactRefs: ArtifactRef[] = [];
		const startingTextParts: string[] = [];

		let modelRef: ArtifactRef | undefined;
		let includeModelSystemPrompt: boolean | undefined;

		for (const occurrence of resolution.capabilities.occurrences ?? []) {
			if (!isAvailableOccurrence(occurrence)) {
				issues.push(availabilityIssueForOccurrence(occurrence));
				continue;
			}

			switch (occurrence.type) {
				case 'model': {
					if (!occurrence.target) {
						issue(
							issues,
							'error',
							'agent.recipe.model-target-missing',
							'Resolved Agent Model is missing its capability target.',
							occurrence.path
						);
						continue;
					}

					try {
						const resolvedModel = await this.models.resolveMappedModelTarget(occurrence.target);
						const includeOverride = occurrence.includeSystemPrompt;
						const resolvedRef = resolvedModel.list.ref;

						if (modelRef && artifactRefKey(modelRef) !== artifactRefKey(resolvedRef)) {
							issue(
								issues,
								'error',
								'agent.recipe.multiple-models',
								'An Agent starter recipe must resolve to exactly one model.',
								occurrence.path
							);
							continue;
						}

						if (
							includeModelSystemPrompt !== undefined &&
							includeOverride !== undefined &&
							includeModelSystemPrompt !== includeOverride
						) {
							issue(
								issues,
								'error',
								'agent.recipe.conflicting-model-system-prompt-overrides',
								'Multiple Agent Model relationships provide conflicting includeSystemPrompt overrides.',
								occurrence.path
							);
							continue;
						}

						modelRef = resolvedRef;
						includeModelSystemPrompt = includeOverride ?? includeModelSystemPrompt;
					} catch (error) {
						issue(
							issues,
							'error',
							'agent.recipe.model-mapping-failed',
							`Could not resolve Model target "${capabilityTargetLabel(occurrence.target)}": ${
								error instanceof Error ? error.message : 'unknown error'
							}`,
							occurrence.path
						);
					}
					break;
				}

				case 'tool': {
					if (!occurrence.target) {
						issue(
							issues,
							'error',
							'agent.recipe.tool-target-missing',
							'Resolved Agent Tool is missing its capability target.',
							occurrence.path
						);
						continue;
					}

					try {
						const resolved = await this.tools.resolveToolTarget(occurrence.target);
						const tool = resolved.tool;
						const implementation = tool.implementation;
						const requiredSDKType =
							implementation.kind === ToolImplType.SDK ? implementation.sdkType?.trim() || undefined : undefined;

						if (implementation.kind === ToolImplType.SDK && !requiredSDKType) {
							issue(
								issues,
								'error',
								'agent.recipe.tool-sdk-metadata-missing',
								`Tool "${tool.displayName || tool.name}" is missing SDK compatibility metadata.`,
								occurrence.path
							);
							continue;
						}

						const choice = toolStoreChoiceFromSelection(
							{
								choiceID: getUUIDv7(),
								target: occurrence.target,
								autoExecute: toolAutoExecute(occurrence, tool.autoExecute),
							},
							resolved
						);
						const key = toolIdentityKey(choice.target);
						const existing = toolSelections.get(key);

						if (
							existing &&
							(existing.choice.autoExecute !== choice.autoExecute || existing.requiredSDKType !== requiredSDKType)
						) {
							issue(
								issues,
								'error',
								'agent.recipe.conflicting-tool-overrides',
								`Tool "${tool.displayName || tool.name}" occurs with conflicting Agent relationship behavior.`,
								occurrence.path
							);
							continue;
						}

						toolSelections.set(key, {
							choice,
							requiredSDKType,
							occurrencePath: occurrence.path,
						});
					} catch (error) {
						issue(
							issues,
							'error',
							'agent.recipe.tool-mapping-failed',
							`Could not resolve Tool target "${capabilityTargetLabel(occurrence.target)}": ${
								error instanceof Error ? error.message : 'unknown error'
							}`,
							occurrence.path
						);
					}
					break;
				}

				case 'skill': {
					const skillRef = artifactRefFromTarget(occurrence.target);
					if (!skillRef) {
						issue(
							issues,
							'error',
							'agent.recipe.skill-artifact-missing',
							'Resolved Agent Skill does not have an ArtifactRef.',
							occurrence.path
						);
						continue;
					}

					const key = artifactRefKey(skillRef);
					const useMode = skillUseMode(occurrence);

					try {
						const summary = await this.skills.describeArtifactSkill(skillRef);
						const ins = summary.Insert.toString();

						// User-message Skills are Template UI entries. They must
						// never be rendered into the system prompt or added to a
						// Skill Runtime session merely because an Agent references
						// them.
						if (summary.Insert === SkillInsert.UserMessage) {
							if (useMode !== AgentSkillUseMode.Available) {
								issue(
									issues,
									'warning',
									'agent.recipe.user-message-skill-session-mode-ignored',
									`User-message Skill "${occurrence.name || skillRef.artifactID}" remains a Template UI entry; use.mode "${useMode}" does not create a Skill-session or system-prompt capability.`,
									occurrence.path
								);
							}
							continue;
						}

						if (summary.Insert !== SkillInsert.Instructions) {
							issue(
								issues,
								'error',
								'agent.recipe.skill-insert-unsupported',
								`Skill "${occurrence.name || skillRef.artifactID}" has unsupported insert target "${ins}".`,
								occurrence.path
							);
							continue;
						}

						switch (useMode) {
							case AgentSkillUseMode.Instructions: {
								instructionSkillRefs.set(key, skillRef);

								const rendered = await this.skills.renderArtifactSkill(skillRef);

								if (rendered.insert !== SkillInsert.Instructions) {
									issue(
										issues,
										'error',
										'agent.recipe.skill-instruction-render-mismatch',
										`Skill "${rendered.displayName || rendered.name}" did not render as instructions.`,
										occurrence.path
									);
									continue;
								}

								if (rendered.text.trim()) {
									instructionSources.set(`skill:${key}`, {
										kind: 'skill',
										artifact: skillRef,
										displayName: rendered.displayName || rendered.name || occurrence.name || skillRef.artifactID,
										text: rendered.text.trim(),
										sourceTags: rendered.tags,
									});
								}
								break;
							}

							case AgentSkillUseMode.Active:
								enabledSkillRefs.set(key, skillRef);
								activeSkillRefs.set(key, skillRef);
								break;

							case AgentSkillUseMode.Available:
								enabledSkillRefs.set(key, skillRef);
								break;
						}
					} catch (error) {
						issue(
							issues,
							'error',
							'agent.recipe.skill-inspection-failed',
							`Could not inspect Agent Skill: ${error instanceof Error ? error.message : 'unknown error'}`,
							occurrence.path
						);
					}
					break;
				}

				case 'mcp': {
					const serverRef = artifactRefFromTarget(occurrence.target);
					if (!serverRef) {
						issue(
							issues,
							'error',
							'agent.recipe.mcp-artifact-missing',
							'Resolved Agent MCP does not have an ArtifactRef.',
							occurrence.path
						);
						continue;
					}
					mcpArtifactRefs.push(serverRef);
					break;
				}

				case 'text': {
					const textRef = artifactRefFromTarget(occurrence.target);
					if (!textRef) {
						issue(
							issues,
							'error',
							'agent.recipe.text-artifact-missing',
							'Resolved Agent Text does not have an ArtifactRef.',
							occurrence.path
						);
						continue;
					}

					const key = artifactRefKey(textRef);
					if (textArtifacts.has(key)) {
						continue;
					}

					try {
						const text = await this.texts.materializeText(textRef);
						textArtifacts.set(key, text.artifact);

						switch (text.insert) {
							case TextInsert.Instructions:
								if (text.content.trim()) {
									instructionSources.set(`text:${key}`, {
										kind: 'text',
										artifact: text.artifact,
										displayName: text.name || occurrence.name || 'Agent instructions',
										text: text.content.trim(),
									});
								}
								break;

							case TextInsert.UserMessage:
								if (text.content.trim()) {
									startingTextParts.push(text.content.trim());
								}
								break;
						}
					} catch (error) {
						issue(
							issues,
							'error',
							'agent.recipe.text-materialization-failed',
							`Could not materialize Agent Text: ${error instanceof Error ? error.message : 'unknown error'}`,
							occurrence.path
						);
					}
					break;
				}

				case 'mcp.policy':
				case 'plugin':
				case 'agent':
					break;

				default:
					issue(
						issues,
						'warning',
						'agent.recipe.unsupported-occurrence',
						`Agent declaration type "${occurrence.type}" does not currently contribute to Composer state.`,
						occurrence.path
					);
			}
		}

		if (mcpArtifactRefs.length > 0) {
			try {
				const resolved = await this.mcp.resolveMCPRuntimeServerIDs(mcpArtifactRefs);
				for (const runtimeServerID of resolved.values()) {
					mcpServerIDs.add(runtimeServerID);
				}
			} catch (error) {
				issue(
					issues,
					'error',
					'agent.recipe.mcp-runtime-id-failed',
					`Could not load MCP runtime identities: ${error instanceof Error ? error.message : 'unknown error'}`
				);
			}
		}

		const mcpContext: MCPConversationContext | undefined =
			mcpServerIDs.size === 0
				? undefined
				: {
						servers: [...mcpServerIDs].map(server => ({
							server,
							toolExposure: MCPToolExposure.All,
							includeServerInstructions: true,
						})),
					};

		const preparedInstructionSources = [...instructionSources.values()];

		return {
			agent: resolution.agent,
			resolution,
			modelRef,
			includeModelSystemPrompt,
			toolSelections: [...toolSelections.values()],
			enabledSkillRefs: [...enabledSkillRefs.values()],
			activeSkillRefs: [...activeSkillRefs.values()],
			instructionSkillRefs: [...instructionSkillRefs.values()],
			instructionSources: preparedInstructionSources,
			instructionText: preparedInstructionSources
				.map(source => source.text)
				.join('\n\n')
				.trim(),
			startingText: startingTextParts.join('\n\n').trim(),
			mcpContext,
			textArtifacts: [...textArtifacts.values()],
			issues,
			canApply: !issues.some(value => value.severity === 'error'),
		};
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
