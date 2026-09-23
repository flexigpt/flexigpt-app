// oxlint-disable typescript/parameter-properties
import type { AgentResolution, AgentView } from '@/spec/agent';
import type { ArtifactRef, CapabilityOccurrence, MappedTarget } from '@/spec/artifact';
import type { MCPConversationContext, MCPRuntimeServerID } from '@/spec/mcp';
import type { ModelPresetRef } from '@/spec/modelpreset';
import type { RuntimeSkillRenderResult, SkillRef } from '@/spec/skill';
import type { ToolStoreChoice } from '@/spec/tool';
import { AgentTextInsert } from '@/spec/agent';
import { ArtifactState } from '@/spec/artifact';
import { MCPToolExposure } from '@/spec/mcp';
import { SkillInsert } from '@/spec/skill';
import { ToolImplType } from '@/spec/tool';

import { getUUIDv7 } from '@/lib/uuid_utils';

import type { IAgentStoreAPI, IModelPresetStoreAPI, IToolStoreAPI } from '@/apis/interface';

type AgentStarterIssueSeverity = 'error' | 'warning';

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
	constructor(
		private readonly agents: IAgentStoreAPI,
		private readonly tools: IToolStoreAPI,
		private readonly models: IModelPresetStoreAPI,
		private readonly mcp: AgentMCPRuntimeResolver,
		private readonly skills: AgentSkillRenderer
	) {}

	async listAgentCatalogOptions(): Promise<AgentCatalogOption[]> {
		const agents = await this.agents.listAgentsForManagement();

		return agents
			.map(agent => {
				const ref: ArtifactRef = {
					rootID: agent.artifact.rootID,
					artifactID: agent.artifact.id,
				};
				const displayName = agent.displayName || agent.name;
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
					label: `${displayName} (${agent.name})`,
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

		for (const occurrence of resolution.capabilities.occurrences) {
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
							'agent.recipe.tool-artifact-target-unsupported',
							'This Agent resolved a Tool as an Artifact. The current frontend can map Tool fallback targets, but it has no ArtifactRef-to-ToolRef API.',
							occurrence.path
						);
						continue;
					}

					try {
						const toolRef = await this.tools.resolveMappedToolTarget(occurrence.mapped);
						const tool = await this.tools.getTool(toolRef.bundleID, toolRef.toolSlug, toolRef.toolVersion);

						if (!tool) {
							issue(
								issues,
								'error',
								'agent.recipe.tool-not-found',
								`Mapped Tool "${mappedTargetLabel(occurrence.mapped)}" could not be loaded from the Tool Store.`,
								occurrence.path
							);
							continue;
						}

						const requiredSDKType =
							tool.type === ToolImplType.SDK ? tool.sdkImpl?.sdkType?.trim() || undefined : undefined;

						if (tool.type === ToolImplType.SDK && !requiredSDKType) {
							issue(
								issues,
								'error',
								'agent.recipe.tool-sdk-metadata-missing',
								`Tool "${tool.displayName || tool.slug}" is missing SDK compatibility metadata.`,
								occurrence.path
							);
							continue;
						}

						const choice: ToolStoreChoice = {
							choiceID: getUUIDv7(),
							bundleID: toolRef.bundleID,
							toolSlug: toolRef.toolSlug,
							toolVersion: toolRef.toolVersion,
							toolID: tool.id,
							toolType: tool.llmToolType,
							displayName: tool.displayName,
							description: tool.description,
							autoExecute: occurrenceAutoExecute(occurrence, tool.autoExecute),
						};

						const key = `${toolRef.bundleID}/${toolRef.toolSlug}@${toolRef.toolVersion}`;
						const existing = toolSelections.get(key);

						if (
							existing &&
							(existing.choice.autoExecute !== choice.autoExecute || existing.requiredSDKType !== requiredSDKType)
						) {
							issue(
								issues,
								'error',
								'agent.recipe.conflicting-tool-overrides',
								`Tool "${tool.displayName || tool.slug}" occurs with conflicting Agent relationship behavior.`,
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
							`Could not map Tool target "${mappedTargetLabel(occurrence.mapped)}": ${
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
