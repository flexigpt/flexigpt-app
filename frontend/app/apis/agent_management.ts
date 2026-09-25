// oxlint-disable typescript/parameter-properties
import type { AgentImportDestination, AgentResolution, AgentView } from '@/spec/agent';
import type { ArtifactRef, CapabilityOccurrence, MappedTarget } from '@/spec/artifact';
import type { CollectionView } from '@/spec/collection';
import type { MCPConversationContext, MCPRuntimeServerID } from '@/spec/mcp';
import type { ModelPresetRef } from '@/spec/modelpreset';
import type { RuntimeSkillRenderResult, SkillRef } from '@/spec/skill';
import type { ToolStoreChoice } from '@/spec/tool';
import { AgentTextInsert } from '@/spec/agent';
import { ArtifactState } from '@/spec/artifact';
import { MCPToolExposure } from '@/spec/mcp';
import { SkillInsert } from '@/spec/skill';
import { ToolImplType } from '@/spec/tool';

import { throwIfAborted } from '@/lib/async_utils';
import { getErrorMessage } from '@/lib/error_utils';
import { createSharedAsyncCatalog } from '@/lib/shared_async_catalog';
import { getUUIDv7 } from '@/lib/uuid_utils';

import type { IAgentStoreAPI, IModelPresetStoreAPI, IToolTargetResolver } from '@/apis/interface';
import { toolStoreChoiceFromSelection } from '@/apis/tool_management';

import { toolIdentityKey } from '@/tools/lib/tool_identity_utils';

type AgentStarterIssueSeverity = 'error' | 'warning';

export interface AgentCollectionData {
	collection: CollectionView;
	agents: AgentView[];
	agentsLoaded: boolean;
	isLoadingAgents: boolean;
	importDestination?: AgentImportDestination;
	agentLoadError?: string;
}

export interface AgentManagementPageData {
	collections: AgentCollectionData[];
	importDestinations: AgentImportDestination[];
}

export const EMPTY_AGENT_MANAGEMENT_PAGE_DATA: AgentManagementPageData = {
	collections: [],
	importDestinations: [],
};

export function agentArtifactRef(agent: AgentView): ArtifactRef {
	return {
		rootID: agent.artifact.rootID,
		artifactID: agent.artifact.id,
	};
}

export function agentCollectionRef(collection: CollectionView): ArtifactRef {
	return {
		rootID: collection.artifact.rootID,
		artifactID: collection.artifact.id,
	};
}

export function agentCollectionKey(collection: CollectionView): string {
	return artifactRefKey(agentCollectionRef(collection));
}

export function agentDisplayName(agent: AgentView): string {
	return agent.displayName || agent.name;
}

export function collectionDisplayName(collection: CollectionView): string {
	return collection.displayName || collection.name;
}

export function isBuiltInAgentCollection(collection: CollectionView): boolean {
	// User baselines are non-deletable and can be metadata-read-only, but they
	// are not protected built-in package Collections.
	return !collection.baseline && !collection.editable && !collection.deletable;
}

export function canEditAgentCollectionMetadata(collection: CollectionView): boolean {
	return collection.editable && !collection.baseline;
}

export function canDeleteAgentCollection(collection: CollectionView): boolean {
	return collection.deletable && !collection.baseline;
}

interface AgentStarterIssue {
	severity: AgentStarterIssueSeverity;
	code: string;
	message: string;
	path?: string;
}

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

export interface PreparedAgentStarter {
	agent: AgentView;
	resolution: AgentResolution;

	modelPresetRef?: ModelPresetRef;
	includeModelSystemPrompt?: boolean;

	toolSelections: AgentPreparedToolSelection[];
	enabledSkillRefs: SkillRef[];
	activeSkillRefs: SkillRef[];
	instructionSkillRefs: SkillRef[];
	instructionText: string;
	startingText: string;

	mcpContext?: MCPConversationContext;

	/**
	 * Materialized Text Artifacts used by the recipe. Kept for diagnostics and
	 * future Composer provenance, while the content is projected into
	 * instructionText or startingText.
	 */
	textArtifacts: ArtifactRef[];

	issues: AgentStarterIssue[];
	canApply: boolean;
}

interface AgentMCPRuntimeResolver {
	runtimeServerIDForArtifact(artifact: ArtifactRef): Promise<MCPRuntimeServerID>;
}

interface AgentSkillRenderer {
	renderArtifactSkill(skill: ArtifactRef, args?: Record<string, string>): Promise<RuntimeSkillRenderResult>;
}

function artifactRefKey(ref: ArtifactRef): string {
	return `${ref.rootID}:${ref.artifactID}`;
}

function compareCollections(left: CollectionView, right: CollectionView): number {
	const leftBuiltIn = isBuiltInAgentCollection(left);
	const rightBuiltIn = isBuiltInAgentCollection(right);

	if (leftBuiltIn !== rightBuiltIn) {
		return leftBuiltIn ? -1 : 1;
	}

	const byName = collectionDisplayName(left).localeCompare(collectionDisplayName(right), undefined, {
		sensitivity: 'base',
	});

	if (byName !== 0) {
		return byName;
	}

	return agentCollectionKey(left).localeCompare(agentCollectionKey(right));
}

function modelPresetRefKey(ref: ModelPresetRef): string {
	return `${ref.providerName}/${ref.modelPresetID}`;
}

function mappedTargetLabel(target: MappedTarget): string {
	return `${target.provider}/${target.identifier}`;
}

function issue(
	issues: AgentStarterIssue[],
	severity: AgentStarterIssueSeverity,
	code: string,
	message: string,
	path?: string
) {
	issues.push({
		severity,
		code,
		message,
		path,
	});
}

function decodeOccurrenceJSONMap(value?: Record<string, number[]>): Record<string, unknown> {
	if (!value) {
		return {};
	}

	const decoder = new TextDecoder();
	const output: Record<string, unknown> = {};

	for (const [key, bytes] of Object.entries(value)) {
		try {
			const parsed = JSON.parse(decoder.decode(Uint8Array.from(bytes))) as unknown;
			output[key] = parsed;
		} catch {
			// Resolution data is backend-owned. A malformed relationship payload
			// is surfaced as an unusable occurrence by the caller rather than
			// guessed or coerced here.
		}
	}

	return output;
}

function occurrenceUseMode(occurrence: CapabilityOccurrence): 'available' | 'active' | 'instructions' {
	const use = decodeOccurrenceJSONMap(occurrence.use);
	const mode = use.mode;

	switch (mode) {
		case 'active':
		case 'instructions':
		case 'available':
			return mode;
		default:
			return 'available';
	}
}

function occurrenceAutoExecute(occurrence: CapabilityOccurrence, fallback: boolean): boolean {
	const overrides = decodeOccurrenceJSONMap(occurrence.overrides);
	return typeof overrides.autoExecute === 'boolean' ? overrides.autoExecute : fallback;
}

function occurrenceIncludeModelSystemPrompt(occurrence: CapabilityOccurrence): boolean | undefined {
	const overrides = decodeOccurrenceJSONMap(occurrence.overrides);
	return typeof overrides.includeSystemPrompt === 'boolean' ? overrides.includeSystemPrompt : undefined;
}

function isAvailableOccurrence(occurrence: CapabilityOccurrence): boolean {
	return occurrence.status === 'available';
}

function availabilityIssueForOccurrence(occurrence: CapabilityOccurrence): AgentStarterIssue {
	return {
		severity: occurrence.required ? 'error' : 'warning',
		code: occurrence.code || `agent.recipe.${occurrence.status}`,
		message:
			occurrence.message ||
			`${occurrence.type}${occurrence.name ? ` "${occurrence.name}"` : ''} is ${occurrence.status}.`,
		path: occurrence.path,
	};
}

export class AgentManagementAPI {
	private readonly collectionAgentLoads = new Map<string, Promise<AgentView[]>>();

	private readonly composerAgentCatalog = createSharedAsyncCatalog<AgentView[]>(
		async () => (await this.agents.listAgentsForManagement()) ?? []
	);

	constructor(
		private readonly agents: IAgentStoreAPI,
		private readonly tools: IToolTargetResolver,
		private readonly models: IModelPresetStoreAPI,
		private readonly mcp: AgentMCPRuntimeResolver,
		private readonly skills: AgentSkillRenderer
	) {}

	invalidateAgentCatalog(): void {
		this.composerAgentCatalog.invalidate();
	}

	async preloadAgentCatalog(force = false): Promise<void> {
		await this.composerAgentCatalog.load(force);
	}

	/**
	 * Static Agent declaration catalog only.
	 * Agent recipe preparation remains an uncached runtime operation.
	 */
	listAgentCatalogOptions(force = false): Promise<AgentCatalogOption[]> {
		return this.composerAgentCatalog.load(force).then(agents => this.projectAgentCatalogOptions(agents));
	}

	async loadManagementPageData(signal: AbortSignal): Promise<AgentManagementPageData> {
		const [collectionResult, destinationResult] = await Promise.all([
			this.agents.listAgentCollectionsForManagement(),
			this.agents.listAgentImportDestinations(),
		]);
		throwIfAborted(signal);

		const collectionValues = collectionResult ?? [];
		const importDestinations = destinationResult ?? [];
		const collectionsByKey = new Map<string, CollectionView>();

		for (const collection of collectionValues) {
			collectionsByKey.set(agentCollectionKey(collection), collection);
		}
		for (const destination of importDestinations) {
			const key = agentCollectionKey(destination.collection);
			if (!collectionsByKey.has(key)) {
				collectionsByKey.set(key, destination.collection);
			}
		}

		const destinationByCollectionKey = new Map(
			importDestinations.map(destination => [agentCollectionKey(destination.collection), destination] as const)
		);

		return {
			collections: [...collectionsByKey.values()]
				.map(collection => ({
					collection,
					agents: [],
					agentsLoaded: false,
					isLoadingAgents: false,
					importDestination: destinationByCollectionKey.get(agentCollectionKey(collection)),
				}))
				.toSorted((left, right) => compareCollections(left.collection, right.collection)),
			importDestinations: [...importDestinations].toSorted((left, right) => {
				const rootCompare = (left.rootDisplayName || left.rootID).localeCompare(
					right.rootDisplayName || right.rootID,
					undefined,
					{ sensitivity: 'base' }
				);

				if (rootCompare !== 0) {
					return rootCompare;
				}

				return collectionDisplayName(left.collection).localeCompare(
					collectionDisplayName(right.collection),
					undefined,
					{
						sensitivity: 'base',
					}
				);
			}),
		};
	}

	async loadCollectionAgents(
		collection: CollectionView,
		signal: AbortSignal
	): Promise<Pick<AgentCollectionData, 'agents' | 'agentsLoaded' | 'agentLoadError'>> {
		const key = agentCollectionKey(collection);
		let load = this.collectionAgentLoads.get(key);
		if (!load) {
			load = this.agents.listAgents({
				rootID: collection.artifact.rootID,
				collection: agentCollectionRef(collection),
			});
			this.collectionAgentLoads.set(key, load);

			const clear = () => {
				if (this.collectionAgentLoads.get(key) === load) {
					this.collectionAgentLoads.delete(key);
				}
			};
			void load.then(clear, clear);
		}

		try {
			const agents = (await load) ?? [];
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
				agentLoadError: getErrorMessage(error, 'Agents could not be loaded for this Collection.'),
			};
		}
	}

	private projectAgentCatalogOptions(agents: AgentView[]): AgentCatalogOption[] {
		return agents
			.map(agent => {
				const ref = agentArtifactRef(agent);
				const displayName = agentDisplayName(agent);
				const stateAvailable = agent.artifact.state === ArtifactState.Available;
				const isSelectable = stateAvailable && agent.artifact.enabled;

				let availabilityReason: string | undefined;

				if (!stateAvailable) {
					availabilityReason = `Agent Artifact is ${agent.artifact.state}.`;
				} else if (!agent.artifact.enabled) {
					availabilityReason = 'Agent is disabled.';
				}

				return {
					key: artifactRefKey(ref),
					ref,
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

				const byName = left.displayName.localeCompare(right.displayName, undefined, {
					sensitivity: 'base',
				});

				if (byName !== 0) {
					return byName;
				}

				return left.key.localeCompare(right.key);
			});
	}

	async prepareAgentStarter(agentRef: ArtifactRef): Promise<PreparedAgentStarter> {
		const resolution = await this.agents.resolveAgent(agentRef);
		return this.prepareAgentStarterFromResolution(resolution);
	}

	async prepareAgentStarterFromResolution(resolution: AgentResolution): Promise<PreparedAgentStarter> {
		const issues: AgentStarterIssue[] = [];

		const toolSelections = new Map<string, AgentPreparedToolSelection>();
		const enabledSkillRefs = new Map<string, SkillRef>();
		const activeSkillRefs = new Map<string, SkillRef>();
		const instructionSkillRefs = new Map<string, SkillRef>();
		const textArtifacts = new Map<string, ArtifactRef>();
		const mcpServerIDs = new Set<MCPRuntimeServerID>();
		const startingTextParts: string[] = [];
		const instructionParts: string[] = [];

		let modelPresetRef: ModelPresetRef | undefined;
		let includeModelSystemPrompt: boolean | undefined;

		for (const occurrence of resolution.capabilities?.occurrences ?? []) {
			if (!isAvailableOccurrence(occurrence)) {
				issues.push(availabilityIssueForOccurrence(occurrence));
				continue;
			}

			switch (occurrence.type) {
				case 'model': {
					if (!occurrence.mapped) {
						issue(
							issues,
							'error',
							'agent.recipe.model-artifact-target-unsupported',
							'This Agent resolved a Model as an Artifact. The current frontend can map Model fallback targets, but it has no ArtifactRef-to-ModelPresetRef API.',
							occurrence.path
						);
						continue;
					}

					try {
						const resolvedModel = await this.models.resolveMappedModelTarget(occurrence.mapped);
						const override = occurrenceIncludeModelSystemPrompt(occurrence);

						if (modelPresetRef && modelPresetRefKey(modelPresetRef) !== modelPresetRefKey(resolvedModel)) {
							issue(
								issues,
								'error',
								'agent.recipe.multiple-models',
								'An Agent starter recipe must resolve to one Model preset.',
								occurrence.path
							);
							continue;
						}

						if (
							includeModelSystemPrompt !== undefined &&
							override !== undefined &&
							includeModelSystemPrompt !== override
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

						modelPresetRef = resolvedModel;
						includeModelSystemPrompt = override ?? includeModelSystemPrompt;
					} catch (error) {
						issue(
							issues,
							'error',
							'agent.recipe.model-mapping-failed',
							`Could not map Model target "${mappedTargetLabel(occurrence.mapped)}": ${
								error instanceof Error ? error.message : 'unknown error'
							}`,
							occurrence.path
						);
					}
					break;
				}

				case 'tool': {
					if (!occurrence.mapped) {
						issue(
							issues,
							'error',
							'agent.recipe.tool-mapped-target-missing',
							'This Agent Tool occurrence is missing its aggregate-mapped target.',
							occurrence.path
						);
						continue;
					}

					try {
						const resolved = await this.tools.resolveMappedTool(occurrence.mapped);
						const tool = resolved.tool;
						const implementation = tool.implementation;
						const requiredSDKType =
							implementation.kind === ToolImplType.SDK ? implementation.sdkType.trim() || undefined : undefined;

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
								target: occurrence.mapped,
								autoExecute: occurrenceAutoExecute(occurrence, tool.autoExecute),
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
							`Could not resolve Tool target "${mappedTargetLabel(occurrence.mapped)}": ${
								error instanceof Error ? error.message : 'unknown error'
							}`,
							occurrence.path
						);
					}
					break;
				}

				case 'skill': {
					if (!occurrence.artifact) {
						issue(
							issues,
							'error',
							'agent.recipe.skill-artifact-missing',
							'Resolved Agent Skill does not have an ArtifactRef.',
							occurrence.path
						);
						continue;
					}

					const skillRef = occurrence.artifact;
					const key = artifactRefKey(skillRef);
					const mode = occurrenceUseMode(occurrence);

					if (mode === 'instructions') {
						instructionSkillRefs.set(key, skillRef);

						try {
							const rendered = await this.skills.renderArtifactSkill(skillRef);

							if (rendered.insert !== SkillInsert.Instructions) {
								issue(
									issues,
									'error',
									'agent.recipe.skill-instruction-incompatible',
									`Skill "${rendered.displayName || rendered.name}" does not render instruction text.`,
									occurrence.path
								);
								continue;
							}

							if (rendered.text.trim()) {
								instructionParts.push(rendered.text.trim());
							}
						} catch (error) {
							issue(
								issues,
								'error',
								'agent.recipe.skill-instruction-render-failed',
								`Could not render Agent instruction Skill: ${error instanceof Error ? error.message : 'unknown error'}`,
								occurrence.path
							);
						}

						continue;
					}

					enabledSkillRefs.set(key, skillRef);

					if (mode === 'active') {
						activeSkillRefs.set(key, skillRef);
					}

					break;
				}

				case 'mcp': {
					if (!occurrence.artifact) {
						issue(
							issues,
							'error',
							'agent.recipe.mcp-artifact-missing',
							'Resolved Agent MCP does not have an ArtifactRef.',
							occurrence.path
						);
						continue;
					}

					try {
						mcpServerIDs.add(await this.mcp.runtimeServerIDForArtifact(occurrence.artifact));
					} catch (error) {
						issue(
							issues,
							'error',
							'agent.recipe.mcp-runtime-id-failed',
							`Could not load MCP runtime identity: ${error instanceof Error ? error.message : 'unknown error'}`,
							occurrence.path
						);
					}

					break;
				}

				case 'text': {
					if (!occurrence.artifact) {
						issue(
							issues,
							'error',
							'agent.recipe.text-artifact-missing',
							'Resolved Agent Text does not have an ArtifactRef.',
							occurrence.path
						);
						continue;
					}

					try {
						const text = await this.agents.materializeAgentText(occurrence.artifact);
						textArtifacts.set(artifactRefKey(text.artifact), text.artifact);

						if (text.insert === AgentTextInsert.Instructions) {
							if (text.content.trim()) {
								instructionParts.push(text.content.trim());
							}
							continue;
						}

						if (text.insert === AgentTextInsert.UserMessage && text.content.trim()) {
							startingTextParts.push(text.content.trim());
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
						`Agent declaration type "${occurrence.type}" does not currently contribute to a Composer starter recipe.`,
						occurrence.path
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

		return {
			agent: resolution.agent,
			resolution,
			modelPresetRef,
			includeModelSystemPrompt,
			toolSelections: [...toolSelections.values()],
			enabledSkillRefs: [...enabledSkillRefs.values()],
			activeSkillRefs: [...activeSkillRefs.values()],
			instructionSkillRefs: [...instructionSkillRefs.values()],
			instructionText: instructionParts.join('\n\n').trim(),
			startingText: startingTextParts.join('\n\n').trim(),
			mcpContext,
			textArtifacts: [...textArtifacts.values()],
			issues,
			canApply: !issues.some(value => value.severity === 'error'),
		};
	}
}
