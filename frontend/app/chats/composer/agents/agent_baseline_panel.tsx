import { useEffect, useMemo, useState } from 'react';
import { FiCheck, FiFileText, FiLayers, FiServer, FiTool, FiX, FiZap } from 'react-icons/fi';

import type { AgentCapabilityOccurrence } from '@/spec/agent';
import type { ArtifactRef, CapabilityTarget } from '@/spec/artifact';
import type { SkillRef } from '@/spec/skill';
import type { ToolStoreChoice } from '@/spec/tool';
import { AgentSkillUseMode, isAgentSkillUseAsInstructions } from '@/spec/agent';
import { CapabilityResolutionStatus, CapabilityTargetForm } from '@/spec/artifact';

import { toolManagementAPI } from '@/apis/baseapi';
import { toolStoreChoiceFromSelection } from '@/apis/tool_management';

import type { AgentManagerState } from '@/chats/composer/agents/use_agent_manager';

interface ProjectedAgentTool {
	key: string;
	occurrence: AgentCapabilityOccurrence;
	choice?: ToolStoreChoice;
	error?: string;
}

interface AgentBaselinePanelProps {
	manager: AgentManagerState;
	modelSelectionSource: 'session' | 'agent';
	onAdoptTool: (choice: ToolStoreChoice) => void;
	onAdoptSkill: (ref: SkillRef, active: boolean) => void;
	onAdoptMCP: (ref: ArtifactRef) => Promise<boolean>;
	onOpenSkills: () => void;
}

function capabilityKey(target: CapabilityTarget): string {
	if (target.form === CapabilityTargetForm.Artifact && target.artifact) {
		return `artifact:${target.artifact.rootID}:${target.artifact.artifactID}`;
	}

	return ['direct', target.providerIdentity ?? '', target.providerLocalID ?? '', target.type, target.name].join(':');
}

function availableTarget(occurrence: AgentCapabilityOccurrence): CapabilityTarget | undefined {
	if (occurrence.status !== CapabilityResolutionStatus.Available) {
		return undefined;
	}

	return occurrence.target ?? undefined;
}

function agentSkillRef(occurrence: AgentCapabilityOccurrence): SkillRef | undefined {
	const target = availableTarget(occurrence);
	if (occurrence.type !== 'skill' || target?.form !== CapabilityTargetForm.Artifact || !target.artifact) {
		return undefined;
	}

	return {
		rootID: target.artifact.rootID,
		artifactID: target.artifact.artifactID,
	};
}

function agentMCPRef(occurrence: AgentCapabilityOccurrence): ArtifactRef | undefined {
	const target = availableTarget(occurrence);
	if (occurrence.type !== 'mcp' || target?.form !== CapabilityTargetForm.Artifact || !target.artifact) {
		return undefined;
	}

	return {
		rootID: target.artifact.rootID,
		artifactID: target.artifact.artifactID,
	};
}

export function AgentBaselinePanel({
	manager,
	modelSelectionSource,
	onAdoptTool,
	onAdoptSkill,
	onAdoptMCP,
	onOpenSkills,
}: AgentBaselinePanelProps) {
	const resolution = manager.selectedAgentResolution;
	const [mcpActionError, setMCPActionError] = useState<string | null>(null);
	const occurrences = useMemo(() => resolution?.capabilities.occurrences ?? [], [resolution?.capabilities.occurrences]);

	const toolCandidates = useMemo(() => {
		const seen = new Set<string>();

		return occurrences.flatMap(occurrence => {
			const target = availableTarget(occurrence);
			if (occurrence.type !== 'tool' || !target) {
				return [];
			}

			const key = capabilityKey(target);
			if (seen.has(key)) {
				return [];
			}
			seen.add(key);

			return [{ key, occurrence, target }];
		});
	}, [occurrences]);

	const [tools, setTools] = useState<ProjectedAgentTool[]>([]);

	useEffect(() => {
		let cancelled = false;

		if (toolCandidates.length === 0) {
			// oxlint-disable-next-line react/set-state-in-effect
			setTools([]);
			return () => {
				cancelled = true;
			};
		}

		void Promise.all(
			toolCandidates.map(async candidate => {
				try {
					const resolved = await toolManagementAPI.resolveToolTarget(candidate.target);

					return {
						key: candidate.key,
						occurrence: candidate.occurrence,
						choice: toolStoreChoiceFromSelection(
							{
								choiceID: `agent-projection:${candidate.key}`,
								target: candidate.target,
								autoExecute: candidate.occurrence.autoExecute === true,
							},
							resolved
						),
					} satisfies ProjectedAgentTool;
				} catch (error) {
					return {
						key: candidate.key,
						occurrence: candidate.occurrence,
						error: error instanceof Error ? error.message : 'Agent Tool could not be projected.',
					} satisfies ProjectedAgentTool;
				}
			})
		).then(values => {
			if (!cancelled) {
				setTools(values);
			}
		});

		return () => {
			cancelled = true;
		};
	}, [toolCandidates]);

	if (!manager.selectedAgent || !resolution) {
		return null;
	}

	const model = occurrences.find(
		occurrence =>
			occurrence.type === 'model' && occurrence.status === CapabilityResolutionStatus.Available && occurrence.target
	);
	const skills = occurrences
		.map(occurrence => ({
			occurrence,
			ref: agentSkillRef(occurrence),
		}))
		.filter((value): value is { occurrence: AgentCapabilityOccurrence; ref: SkillRef } => value.ref !== undefined);
	const mcpServers = occurrences
		.map(occurrence => ({
			occurrence,
			ref: agentMCPRef(occurrence),
		}))
		.filter((value): value is { occurrence: AgentCapabilityOccurrence; ref: ArtifactRef } => value.ref !== undefined);
	const textContributions = occurrences.filter(occurrence => occurrence.type === 'text');

	const isExcluded = (occurrence: AgentCapabilityOccurrence) => Boolean(manager.getCapabilityExclusion(occurrence));

	const toggleExclusion = (occurrence: AgentCapabilityOccurrence) => {
		manager.toggleCapabilityExclusion(occurrence);
	};

	return (
		<details className="border-base-300 bg-base-200 mx-2 mb-1 rounded-xl border px-3 py-2 xl:mx-4" open>
			<summary className="flex cursor-pointer list-none items-center gap-2 text-xs font-semibold">
				<FiLayers size={14} />
				<span className="min-w-0 flex-1 truncate">Agent baseline in this Composer</span>
				<span className="text-base-content/60">
					{tools.length} tools · {skills.length} skills · {mcpServers.length} MCP
				</span>
			</summary>

			<p className="text-base-content/70 mt-2 text-xs">
				These are backend-owned Agent contributions. Override copies one contribution into Session intent. Exclude emits
				a Session remove override. Unchanged contributions remain Agent-owned.
			</p>

			{mcpActionError ? <div className="alert alert-warning mt-2 rounded-xl text-xs">{mcpActionError}</div> : null}

			<div className="mt-3 space-y-2">
				{model ? (
					<div className="bg-base-100 flex flex-wrap items-center gap-2 rounded-lg px-2 py-1 text-xs">
						<span className="font-medium">Model</span>
						<span className="min-w-0 flex-1 truncate">{model.name}</span>
						{modelSelectionSource === 'agent' ? (
							<span className="badge badge-success badge-xs">Agent-owned</span>
						) : (
							<button
								type="button"
								className="btn btn-ghost btn-xs rounded-lg"
								onClick={() => {
									void manager.reapplySelectedAgent();
								}}
							>
								Use Agent model
							</button>
						)}
					</div>
				) : null}

				{tools.map(tool => {
					const excluded = isExcluded(tool.occurrence);

					return (
						<div key={tool.key} className="bg-base-100 flex flex-wrap items-center gap-2 rounded-lg px-2 py-1 text-xs">
							<FiTool size={13} />
							<span className="min-w-0 flex-1 truncate">{tool.choice?.displayName || tool.occurrence.name}</span>
							<span className="badge badge-secondary badge-xs">Agent</span>
							{tool.occurrence.autoExecute ? <span className="badge badge-ghost badge-xs">Auto</span> : null}
							{excluded ? <span className="badge badge-warning badge-xs">Excluded</span> : null}
							{tool.error ? <span className="text-warning">{tool.error}</span> : null}
							{!excluded && tool.choice ? (
								<button
									type="button"
									className="btn btn-ghost btn-xs rounded-lg"
									onClick={() => {
										onAdoptTool(tool.choice as ToolStoreChoice);
									}}
								>
									Override
								</button>
							) : null}
							<button
								type="button"
								className="btn btn-ghost btn-xs text-error rounded-lg"
								onClick={() => {
									toggleExclusion(tool.occurrence);
								}}
							>
								{excluded ? <FiCheck size={12} /> : <FiX size={12} />}
								{excluded ? 'Include' : 'Exclude'}
							</button>
						</div>
					);
				})}

				{skills.map(({ occurrence, ref }) => {
					const excluded = isExcluded(occurrence);
					const instructionOnly = isAgentSkillUseAsInstructions(occurrence.skillUseMode);
					const active = occurrence.skillUseMode === AgentSkillUseMode.Active;

					return (
						<div
							key={capabilityKey(occurrence.target as CapabilityTarget)}
							className="bg-base-100 flex flex-wrap items-center gap-2 rounded-lg px-2 py-1 text-xs"
						>
							<FiZap size={13} />
							<span className="min-w-0 flex-1 truncate">{occurrence.name || ref.artifactID}</span>
							<span className="badge badge-secondary badge-xs">Agent</span>
							{instructionOnly ? <span className="badge badge-info badge-xs">Instructions</span> : null}
							{active ? <span className="badge badge-success badge-xs">Active</span> : null}
							{excluded ? <span className="badge badge-warning badge-xs">Excluded</span> : null}
							{instructionOnly ? (
								<button type="button" className="btn btn-ghost btn-xs rounded-lg" onClick={onOpenSkills}>
									Open instructions
								</button>
							) : !excluded ? (
								<button
									type="button"
									className="btn btn-ghost btn-xs rounded-lg"
									onClick={() => {
										onAdoptSkill(ref, active);
									}}
								>
									Override
								</button>
							) : null}
							<button
								type="button"
								className="btn btn-ghost btn-xs text-error rounded-lg"
								onClick={() => {
									toggleExclusion(occurrence);
								}}
							>
								{excluded ? <FiCheck size={12} /> : <FiX size={12} />}
								{excluded ? 'Include' : 'Exclude'}
							</button>
						</div>
					);
				})}

				{mcpServers.map(({ occurrence, ref }) => {
					const excluded = isExcluded(occurrence);

					return (
						<div
							key={capabilityKey(occurrence.target as CapabilityTarget)}
							className="bg-base-100 flex flex-wrap items-center gap-2 rounded-lg px-2 py-1 text-xs"
						>
							<FiServer size={13} />
							<span className="min-w-0 flex-1 truncate">{occurrence.name || ref.artifactID}</span>
							<span className="badge badge-secondary badge-xs">Agent</span>
							{excluded ? <span className="badge badge-warning badge-xs">Excluded</span> : null}
							{!excluded ? (
								<button
									type="button"
									className="btn btn-ghost btn-xs rounded-lg"
									onClick={() => {
										setMCPActionError(null);
										void onAdoptMCP(ref).then(applied => {
											if (!applied) {
												setMCPActionError('The Agent MCP server is not currently available in the Composer catalog.');
											}
										});
									}}
								>
									Override
								</button>
							) : null}
							<button
								type="button"
								className="btn btn-ghost btn-xs text-error rounded-lg"
								onClick={() => {
									toggleExclusion(occurrence);
								}}
							>
								{excluded ? <FiCheck size={12} /> : <FiX size={12} />}
								{excluded ? 'Include' : 'Exclude'}
							</button>
						</div>
					);
				})}

				{textContributions.map(occurrence => {
					const excluded = isExcluded(occurrence);

					return (
						<div
							key={occurrence.path}
							className="bg-base-100 flex flex-wrap items-center gap-2 rounded-lg px-2 py-1 text-xs"
						>
							<FiFileText size={13} />
							<span className="min-w-0 flex-1 truncate">{occurrence.name || 'Agent text contribution'}</span>
							<span className="badge badge-secondary badge-xs">Agent</span>
							{excluded ? <span className="badge badge-warning badge-xs">Excluded</span> : null}
							<button
								type="button"
								className="btn btn-ghost btn-xs text-error rounded-lg"
								onClick={() => {
									toggleExclusion(occurrence);
								}}
							>
								{excluded ? <FiCheck size={12} /> : <FiX size={12} />}
								{excluded ? 'Include' : 'Exclude'}
							</button>
						</div>
					);
				})}
			</div>
		</details>
	);
}
