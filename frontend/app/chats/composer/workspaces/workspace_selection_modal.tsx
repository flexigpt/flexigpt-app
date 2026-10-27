import { useMemo, useState } from 'react';
import { FiAlertCircle, FiCheck, FiRefreshCw, FiSettings, FiX } from 'react-icons/fi';

import type { ArtifactRef } from '@/spec/artifact';
import type { SkillRef } from '@/spec/skill';
import type {
	WorkspaceConversationResourceSelectionRef,
	WorkspaceMCPServer,
	WorkspacePromptContribution,
	WorkspaceRuntimePlan,
	WorkspaceSkill,
} from '@/spec/workspace';
import { artifactRefKey } from '@/spec/artifact';
import { WorkspaceInsertTarget } from '@/spec/workspace';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalHeader } from '@/components/modal/modal_header';
import { ModalSection } from '@/components/modal/modal_section';

import type { ComposerWorkspaceController } from '@/chats/composer/workspaces/use_composer_workspace';
import type { WorkspaceContributionSourceSubject } from '@/workspaces/workspace_contribution_details_modal';
import { WorkspaceContents, workspacePathParts } from '@/workspaces/workspace_contents';
import { WorkspaceContributionDetailsModal } from '@/workspaces/workspace_contribution_details_modal';
import { WorkspaceMCPSetupModal } from '@/workspaces/workspace_mcp_setup_modal';
import { workspaceDisplayName } from '@/workspaces/workspace_presentation';
import { WorkspaceSource } from '@/workspaces/workspace_source';

interface WorkspaceSelectionModalProps {
	isOpen: boolean;
	onClose: () => void;
	state: ComposerWorkspaceController;
	activeSkillRefs: SkillRef[];
	isInputLocked?: boolean;
}

interface WorkspaceMCPSetupTarget {
	artifact: ArtifactRef;
	displayName: string;
	status: string;
	runtimeLoaded: boolean;
}

interface MissingSelection {
	key: string;
	type: string;
	ref: WorkspaceConversationResourceSelectionRef;
	remove: () => void | Promise<void>;
}

function mcpSetupTargets(plan: WorkspaceRuntimePlan | undefined): WorkspaceMCPSetupTarget[] {
	if (!plan) {
		return [];
	}

	const loadedByArtifact = new Map<string, WorkspaceMCPServer>(
		plan.mcpServers.servers.map(server => [artifactRefKey(server.artifact), server])
	);
	const targets = new Map<string, WorkspaceMCPSetupTarget>();

	for (const occurrence of plan.capabilities.occurrences) {
		if (occurrence.type !== 'mcp' || !occurrence.target?.artifact) {
			continue;
		}

		const key = artifactRefKey(occurrence.target.artifact);
		if (targets.has(key)) {
			continue;
		}

		const loaded = loadedByArtifact.get(key);
		targets.set(key, {
			artifact: occurrence.target.artifact,
			displayName: loaded?.displayName || loaded?.name || occurrence.name || 'MCP server',
			status: occurrence.status,
			runtimeLoaded: loaded !== undefined,
		});
	}

	return [...targets.values()].toSorted((left, right) =>
		left.displayName.localeCompare(right.displayName, undefined, {
			sensitivity: 'base',
		})
	);
}

function capabilityTypeLabel(type: string): string {
	switch (type) {
		case 'text':
			return 'Repository file';
		case 'skill':
			return 'Skill';
		case 'mcp':
			return 'MCP server';
		case 'agent':
			return 'Agent';
		case 'model':
			return 'Model';
		case 'tool':
			return 'Tool';
		default:
			return type;
	}
}

function missingSelectionLabel(ref: WorkspaceConversationResourceSelectionRef): string {
	if (ref.name) {
		return ref.name;
	}

	const path = workspacePathParts(ref.locator);
	return path.name || 'Previously selected item';
}

function WorkspaceDirectorySelectionModalContent({
	state,
	activeSkillRefs,
	isInputLocked = false,
}: Omit<WorkspaceSelectionModalProps, 'isOpen' | 'onClose'>) {
	const { requestClose } = useModalDialogController();
	const [setupTarget, setSetupTarget] = useState<WorkspaceMCPSetupTarget | null>(null);
	const [actionError, setActionError] = useState('');
	const [viewedContribution, setViewedContribution] = useState<WorkspaceContributionSourceSubject | null>(null);

	const activeSkillIDs = useMemo(() => new Set(activeSkillRefs.map(value => artifactRefKey(value))), [activeSkillRefs]);
	const workspaceName =
		(state.workspace ? workspaceDisplayName(state.workspace.directory, state.workspace.workspace) : undefined) ||
		state.selection?.displayName ||
		'Unavailable Workspace';
	const mcpTargets = useMemo(() => mcpSetupTargets(state.plan), [state.plan]);
	const mcpTargetsByArtifact = useMemo(
		() => new Map(mcpTargets.map(target => [artifactRefKey(target.artifact), target])),
		[mcpTargets]
	);
	const unloadedMCPTargets = mcpTargets.filter(target => !target.runtimeLoaded);

	const missingSelections: MissingSelection[] = [
		...state.missingContextRefs.map(ref => ({
			key: `context:${artifactRefKey(ref.artifact)}`,
			type: 'Repository file',
			ref,
			remove: () => {
				state.removeContextRef(ref.artifact);
			},
		})),
		...state.missingSkillRefs.map(ref => ({
			key: `skill:${artifactRefKey(ref.artifact)}`,
			type: 'Skill',
			ref,
			remove: () => state.removeSkillRef(ref.artifact),
		})),
		...state.missingMCPRefs.map(ref => ({
			key: `mcp:${artifactRefKey(ref.artifact)}`,
			type: 'MCP server',
			ref,
			remove: () => {
				state.removeMCPRef(ref.artifact);
			},
		})),
	];

	const runAsyncAction = (action: () => Promise<void>, fallback: string) => {
		setActionError('');
		void action().catch((error: unknown) => {
			setActionError(error instanceof Error ? error.message : fallback);
		});
	};

	return (
		<>
			<div className="modal-box bg-base-200 flex max-h-[88vh] w-[calc(100%-1rem)] max-w-5xl flex-col overflow-hidden rounded-2xl p-0">
				<ModalHeader
					title="Workspace contents"
					description={`Choose what ${workspaceName} adds to this conversation. Repository-relative paths are shown below.`}
					onClose={requestClose}
				/>

				<div className="app-scrollbar-thin min-h-0 flex-1 space-y-5 overflow-y-auto p-4 sm:p-6">
					{actionError ? (
						<div className="alert alert-error rounded-2xl text-sm">
							<FiAlertCircle size={14} />
							<span>{actionError}</span>
						</div>
					) : null}

					{state.selectionError ? (
						<div className="alert alert-warning rounded-2xl text-sm">
							<FiAlertCircle size={14} />
							<span>{state.selectionError}</span>
						</div>
					) : null}

					{state.changedCount > 0 ? (
						<div className="alert alert-info rounded-2xl text-sm">
							<FiAlertCircle size={14} />
							<span>
								{state.changedCount} saved item{state.changedCount === 1 ? ' has' : 's have'} changed since this
								conversation selected them. Use Current Contents to accept the latest versions.
							</span>
						</div>
					) : null}

					{state.workspace ? (
						<WorkspaceSource
							directory={state.workspace.directory.ref}
							workspace={{
								rootID: state.workspace.workspace.workspace.artifact.rootID,
								artifactID: state.workspace.workspace.workspace.artifact.id,
							}}
						/>
					) : null}

					{state.plan ? (
						<WorkspaceContents
							plan={state.plan}
							selection={{
								selectedContextIDs: state.selectedContextIDs,
								selectedSkillIDs: state.selectedSkillIDs,
								selectedMCPIDs: state.selectedMCPIDs,
								activeSkillIDs,
							}}
							disabled={isInputLocked}
							onToggleContext={(context, selected) => {
								state.toggleContext(context, selected);
							}}
							onToggleSkill={(skill, selected) => {
								runAsyncAction(() => state.toggleSkill(skill, selected), 'The Skill selection could not be changed.');
							}}
							onSetSkillActive={(skill, active) => {
								runAsyncAction(
									() => state.setSkillActive(skill, active),
									'The active Skill state could not be changed.'
								);
							}}
							onToggleMCP={(server, selected) => {
								state.toggleMCP(server, selected);
							}}
							onViewContext={(context: WorkspacePromptContribution) => {
								setViewedContribution({
									kind: 'text',
									artifact: context.artifact,
									title:
										context.insert === WorkspaceInsertTarget.Instructions
											? 'Repository instructions'
											: 'Repository reference files',
								});
							}}
							onViewSkill={(skill: WorkspaceSkill) => {
								setViewedContribution({
									kind: 'skill',
									artifact: skill.artifact,
									title: skill.displayName || skill.name,
								});
							}}
							onViewMCP={server => {
								setViewedContribution({
									kind: 'mcp',
									artifact: server.artifact,
									title: server.displayName || server.name,
								});
							}}
							renderMCPAction={server => {
								const target = mcpTargetsByArtifact.get(artifactRefKey(server.artifact));
								if (!target) {
									return null;
								}

								return (
									<button
										type="button"
										className="btn btn-sm btn-ghost rounded-xl"
										disabled={isInputLocked || target.status !== 'available'}
										onClick={() => {
											setSetupTarget(target);
										}}
									>
										<FiSettings size={14} />
										Configure
									</button>
								);
							}}
						/>
					) : (
						<div className="text-base-content/60 rounded-2xl border border-dashed p-4 text-sm">
							Workspace contents are not currently available.
						</div>
					)}

					{unloadedMCPTargets.length > 0 ? (
						<ModalSection
							title="MCP servers needing setup"
							description="These integrations are referenced by the Workspace but are not currently loaded."
						>
							<div className="space-y-2">
								{unloadedMCPTargets.map(target => (
									<div
										key={artifactRefKey(target.artifact)}
										className="border-base-content/10 bg-base-100 flex flex-col gap-3 rounded-2xl border p-3 sm:flex-row sm:items-center"
									>
										<div className="min-w-0 grow">
											<div className="truncate text-sm font-medium">{target.displayName}</div>
											<div className="text-base-content/60 mt-1 text-xs">
												Configure this MCP server to make it available to the Workspace.
											</div>
										</div>

										<div className="flex items-center gap-2">
											<span
												className={`badge badge-xs ${target.status === 'available' ? 'badge-warning' : 'badge-error'}`}
											>
												{target.status}
											</span>
											<button
												type="button"
												className="btn btn-sm btn-ghost rounded-xl"
												disabled={isInputLocked || target.status !== 'available'}
												onClick={() => {
													setSetupTarget(target);
												}}
											>
												<FiSettings size={14} />
												Configure
											</button>
										</div>
									</div>
								))}
							</div>
						</ModalSection>
					) : null}

					{missingSelections.length > 0 ? (
						<ModalSection
							title="Saved items no longer available"
							description="Remove unavailable saved items or refresh the Workspace before sending."
						>
							<div className="space-y-2">
								{missingSelections.map(item => {
									const path = workspacePathParts(item.ref.locator);

									return (
										<div
											key={item.key}
											className="border-warning/40 bg-warning/10 flex items-center gap-3 rounded-2xl border p-3"
										>
											<div className="min-w-0 grow">
												<div className="flex flex-wrap items-center gap-2">
													<div className="truncate text-sm font-medium">{missingSelectionLabel(item.ref)}</div>
													<span className="badge badge-warning badge-xs">{item.type}</span>
												</div>
												{path.path ? (
													<div className="text-base-content/60 mt-1 truncate font-mono text-xs">{path.path}</div>
												) : null}
											</div>

											<button
												type="button"
												className="btn btn-ghost btn-sm text-error rounded-xl"
												disabled={isInputLocked}
												onClick={() => {
													setActionError('');
													void (async () => {
														try {
															await item.remove();
														} catch (error) {
															setActionError(
																error instanceof Error ? error.message : 'The unavailable item could not be removed.'
															);
														}
													})();
												}}
											>
												<FiX size={14} />
												Remove
											</button>
										</div>
									);
								})}
							</div>
						</ModalSection>
					) : null}

					{state.capabilityIssues.length > 0 ? (
						<ModalSection
							title="Items that need attention"
							description="Other available Workspace content remains usable."
						>
							<div className="space-y-2">
								{state.capabilityIssues.map((issue, index) => (
									<div
										key={`${issue.path}:${issue.type}:${issue.name || ''}:${index}`}
										className="border-warning/40 bg-warning/10 rounded-2xl border p-3 text-sm"
									>
										<div className="flex flex-wrap items-center gap-2">
											<span className="badge badge-warning badge-xs">{capabilityTypeLabel(issue.type)}</span>
											<div className="font-medium">{issue.name || `Required ${capabilityTypeLabel(issue.type)}`}</div>
										</div>
										<div className="text-base-content/70 mt-1 text-xs">
											{issue.message || `This item is currently ${issue.status}.`}
										</div>
									</div>
								))}
							</div>
						</ModalSection>
					) : null}
				</div>

				<ModalActions>
					<button
						type="button"
						className="btn btn-ghost rounded-xl"
						disabled={isInputLocked || state.selectionLoading}
						onClick={() => {
							runAsyncAction(() => state.refreshSelectedWorkspace(), 'The Workspace could not be refreshed.');
						}}
					>
						<FiRefreshCw size={14} />
						Refresh Workspace
					</button>
					<button
						type="button"
						className="btn btn-ghost rounded-xl"
						disabled={isInputLocked || state.selectionLoading || !state.plan}
						onClick={() => {
							runAsyncAction(
								() => state.updateSelectionFromCurrentContents(),
								'The current Workspace contents could not be selected.'
							);
						}}
					>
						<FiCheck size={14} />
						Use Current Contents
					</button>
					<button
						type="button"
						className="btn btn-primary rounded-xl"
						onClick={() => {
							requestClose();
						}}
					>
						Done
					</button>
				</ModalActions>
			</div>

			<WorkspaceContributionDetailsModal
				isOpen={viewedContribution !== null}
				subject={viewedContribution}
				onClose={() => {
					setViewedContribution(null);
				}}
			/>

			<WorkspaceMCPSetupModal
				isOpen={setupTarget !== null}
				artifact={setupTarget?.artifact ?? null}
				displayName={setupTarget?.displayName ?? 'MCP Server'}
				onClose={() => {
					setSetupTarget(null);
				}}
				onSaved={() => {
					runAsyncAction(
						() => state.refreshSelectedWorkspace(),
						'The Workspace could not be refreshed after MCP setup.'
					);
				}}
			/>
		</>
	);
}

export function WorkspaceDirectorySelectionModal(props: WorkspaceSelectionModalProps) {
	if (!props.isOpen) {
		return null;
	}

	return (
		<ModalDialog isOpen onClose={props.onClose} blockCancel>
			<WorkspaceDirectorySelectionModalContent {...props} />
		</ModalDialog>
	);
}
