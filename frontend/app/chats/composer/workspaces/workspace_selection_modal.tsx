import type { Dispatch, SetStateAction } from 'react';
import { useMemo, useState } from 'react';
import { FiAlertCircle, FiCheck, FiFileText, FiRefreshCw, FiSettings, FiZap } from 'react-icons/fi';

import type { ArtifactRef } from '@/spec/artifact';
import type { SkillRef } from '@/spec/skill';
import type { WorkspaceMCPServer, WorkspaceRuntimePlan } from '@/spec/workspace';
import { WorkspaceInsertTarget } from '@/spec/workspace';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalHeader } from '@/components/modal/modal_header';
import { ModalSection } from '@/components/modal/modal_section';

import type { ComposerWorkspaceController } from '@/chats/composer/workspaces/use_composer_workspace';
import { skillRefKey } from '@/skills/lib/skill_identity_utils';
import { WorkspaceMCPSetupModal } from '@/workspaces/workspace_mcp_setup_modal';

interface WorkspaceSelectionModalProps {
	isOpen: boolean;
	onClose: () => void;
	state: ComposerWorkspaceController;
	activeSkillRefs: SkillRef[];
	setActiveSkillRefs: Dispatch<SetStateAction<SkillRef[]>>;
	isInputLocked?: boolean;
}

interface WorkspaceMCPSetupTarget {
	artifact: ArtifactRef;
	displayName: string;
	status: string;
	runtimeLoaded: boolean;
}

function artifactRefKey(ref: ArtifactRef): string {
	return `${ref.rootID}:${ref.artifactID}`;
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
		if (occurrence.type !== 'mcp' || !occurrence.artifact) {
			continue;
		}

		const key = artifactRefKey(occurrence.artifact);
		if (targets.has(key)) {
			continue;
		}

		const loaded = loadedByArtifact.get(key);
		targets.set(key, {
			artifact: occurrence.artifact,
			displayName: loaded?.displayName || loaded?.name || occurrence.name || occurrence.artifact.artifactID,
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

function WorkspaceDirectorySelectionModalContent({
	state,
	activeSkillRefs,
	setActiveSkillRefs,
	isInputLocked = false,
}: Omit<WorkspaceSelectionModalProps, 'isOpen' | 'onClose'>) {
	const { requestClose } = useModalDialogController();
	const [setupTarget, setSetupTarget] = useState<WorkspaceMCPSetupTarget | null>(null);

	const activeSkillKeys = new Set(activeSkillRefs.map(value => skillRefKey(value)));
	const workspaceName =
		state.workspace?.workspace.workspace.artifact.displayName ??
		state.selection?.displayName ??
		'Unavailable Workspace';
	const mcpTargets = useMemo(() => mcpSetupTargets(state.plan), [state.plan]);

	return (
		<>
			<div className="modal-box bg-base-200 flex max-h-[85vh] w-[calc(100%-1rem)] max-w-4xl flex-col overflow-hidden rounded-2xl p-0">
				<ModalHeader
					title="Workspace Conversation Selection"
					description={`Choose Context, instruction Skills, and MCP setup for ${workspaceName}. These choices affect future turns only.`}
					onClose={requestClose}
				/>

				<div className="app-scrollbar-thin min-h-0 flex-1 space-y-4 overflow-y-auto p-4 sm:p-6">
					{state.selectionError ? (
						<div className="alert alert-warning rounded-2xl text-sm">
							<FiAlertCircle size={14} />
							<span>{state.selectionError}</span>
						</div>
					) : null}

					{state.capabilityIssues.length > 0 ? (
						<div className="alert alert-warning rounded-2xl text-sm">
							<FiAlertCircle size={14} />
							<span>
								Some Workspace declarations are unavailable or ambiguous. Available Context, Skills, and MCP setup
								remain usable.
							</span>
						</div>
					) : null}

					<ModalSection
						title="Context"
						description="Selected Text contributions are composed by the backend on every send."
					>
						<div className="space-y-2">
							{state.contexts.map(context => {
								const selected = state.selectedContextIDs.has(artifactRefKey(context.artifact));

								return (
									<label
										key={artifactRefKey(context.artifact)}
										className={`border-base-content/10 bg-base-100 flex cursor-pointer items-start gap-3 rounded-2xl border p-3 ${
											selected ? 'border-secondary/50 bg-secondary/10' : ''
										}`}
									>
										<input
											type="checkbox"
											className="checkbox checkbox-sm mt-0.5"
											checked={selected}
											disabled={isInputLocked}
											onChange={event => {
												state.toggleContext(context, event.currentTarget.checked);
											}}
										/>
										<FiFileText size={15} className="mt-0.5 shrink-0" />
										<span className="min-w-0 flex-1">
											<span className="block truncate text-sm font-medium">{context.name}</span>
											<span className="text-base-content/60 block truncate font-mono text-xs">
												{context.locator || '(inline Text)'}
											</span>
										</span>
										<span className="badge badge-ghost badge-xs">{context.insert}</span>
									</label>
								);
							})}

							{state.contexts.length === 0 ? (
								<div className="text-base-content/60 rounded-2xl border border-dashed p-4 text-sm">
									No materializable Workspace Context contributions were found.
								</div>
							) : null}
						</div>
					</ModalSection>

					<ModalSection
						title="Workspace Skills"
						description="Instruction Skills can enter the normal conversation Skill session. User-message Skills remain ordinary runtime capabilities."
					>
						<div className="space-y-2">
							{state.skills.map(skill => {
								const instruction = skill.insert === WorkspaceInsertTarget.Instructions;
								const selected = state.selectedSkillIDs.has(artifactRefKey(skill.artifact));
								const active = activeSkillKeys.has(skillRefKey(skill.artifact as SkillRef));

								return (
									<div
										key={artifactRefKey(skill.artifact)}
										className={`border-base-content/10 bg-base-100 rounded-2xl border p-3 ${
											selected ? 'border-secondary/50 bg-secondary/10' : ''
										}`}
									>
										<div className="flex items-start gap-3">
											{instruction ? (
												<input
													type="checkbox"
													className="checkbox checkbox-sm mt-0.5"
													checked={selected}
													disabled={isInputLocked}
													onChange={event => {
														void state.toggleSkill(skill, event.currentTarget.checked);
													}}
												/>
											) : (
												<FiZap size={15} className="mt-0.5 shrink-0" />
											)}

											<span className="min-w-0 flex-1">
												<span className="block truncate text-sm font-medium">{skill.displayName || skill.name}</span>
												<span className="text-base-content/60 block truncate font-mono text-xs">
													{skill.locator || skill.artifact.artifactID}
												</span>
											</span>

											<span className="badge badge-ghost badge-xs">{skill.insert || 'unspecified'}</span>
										</div>

										{instruction && selected ? (
											<label className="mt-3 flex justify-end gap-2 text-xs">
												<span>Active</span>
												<input
													type="checkbox"
													className="toggle toggle-accent toggle-sm"
													checked={active}
													disabled={isInputLocked}
													onChange={event => {
														setActiveSkillRefs(previous => {
															const next = new Map(previous.map(value => [skillRefKey(value), value]));
															const key = skillRefKey(skill.artifact as SkillRef);

															if (event.currentTarget.checked) {
																next.set(key, skill.artifact as SkillRef);
															} else {
																next.delete(key);
															}

															return [...next.values()];
														});
													}}
												/>
											</label>
										) : null}
									</div>
								);
							})}

							{state.skills.length === 0 ? (
								<div className="text-base-content/60 rounded-2xl border border-dashed p-4 text-sm">
									No Workspace Skills were loaded.
								</div>
							) : null}
						</div>
					</ModalSection>

					<ModalSection
						title="Discovered MCP Servers"
						description="MCP declarations are resolved by the Workspace. Setup values and secrets remain in MCP management storage and are never persisted in Workspace selection data."
					>
						<div className="space-y-2">
							{mcpTargets.map(target => (
								<div
									key={artifactRefKey(target.artifact)}
									className="border-base-content/10 bg-base-100 flex flex-col gap-3 rounded-2xl border p-3 sm:flex-row sm:items-center"
								>
									<div className="min-w-0 grow">
										<div className="truncate text-sm font-medium">{target.displayName}</div>
										<div className="text-base-content/60 truncate font-mono text-xs">
											{target.artifact.rootID}/{target.artifact.artifactID}
										</div>
									</div>

									<div className="flex flex-wrap items-center gap-2">
										<span
											className={`badge badge-xs ${
												target.runtimeLoaded
													? 'badge-success'
													: target.status === 'available'
														? 'badge-warning'
														: 'badge-error'
											}`}
										>
											{target.runtimeLoaded ? 'Runtime loaded' : target.status}
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
											Setup
										</button>
									</div>
								</div>
							))}

							{mcpTargets.length === 0 ? (
								<div className="text-base-content/60 rounded-2xl border border-dashed p-4 text-sm">
									No MCP declarations are reachable from this Workspace.
								</div>
							) : null}
						</div>
					</ModalSection>

					{state.capabilityIssues.length > 0 ? (
						<ModalSection title="Capability Diagnostics">
							<div className="space-y-2">
								{state.capabilityIssues.map(issue => (
									<div
										key={`${issue.path}:${issue.type}:${issue.name || ''}`}
										className="border-warning/40 bg-warning/10 rounded-2xl border p-3 text-sm"
									>
										<div className="font-medium">
											{issue.type}
											{issue.name ? `/${issue.name}` : ''}
										</div>
										<div className="text-base-content/70 mt-1">
											{issue.status}
											{issue.message ? `: ${issue.message}` : ''}
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
							void state.refreshSelectedWorkspace();
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
							void state.updateSelectionFromCurrentContents();
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

			<WorkspaceMCPSetupModal
				isOpen={setupTarget !== null}
				artifact={setupTarget?.artifact ?? null}
				displayName={setupTarget?.displayName ?? 'MCP Server'}
				onClose={() => {
					setSetupTarget(null);
				}}
				onSaved={() => {
					void state.refreshSelectedWorkspace();
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
