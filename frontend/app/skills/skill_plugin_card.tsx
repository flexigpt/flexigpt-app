import type { SubmitEventHandler } from 'react';
import { useState } from 'react';
import {
	FiChevronDown,
	FiChevronUp,
	FiEdit2,
	FiEye,
	FiFolder,
	FiGitBranch,
	FiLink,
	FiPlus,
	FiRefreshCw,
	FiTrash2,
	FiX,
} from 'react-icons/fi';

import type { PluginListItem } from '@/spec/plugin';
import type { StoreSkillListItem } from '@/spec/skill';
import { ArtifactState } from '@/spec/artifact';

import { usePendingActions } from '@/hooks/use_pending_actions';

import type { SkillManagementPresentation } from '@/apis/skill_management';
import { backendAPI, skillManagementAPI } from '@/apis/baseapi';

import type { DropdownItem } from '@/components/dropdown';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';
import { Dropdown } from '@/components/dropdown';
import { ActionRow } from '@/components/managementui/action_row';
import { EnabledControl } from '@/components/managementui/enabled_control';
import { ManagementEmptyState } from '@/components/managementui/management_empty_state';
import { ManagementItemCard } from '@/components/managementui/management_item_card';
import { ManagementPluginCard } from '@/components/managementui/management_plugin_card';
import { MetadataPill } from '@/components/managementui/metadata_pill';
import { StatusBadge } from '@/components/managementui/status_badge';
import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalField } from '@/components/modal/modal_field';
import { ModalHeader } from '@/components/modal/modal_header';

import type { SkillManagementFilter } from '@/skills/lib/skill_management_utils';
import type { ManagedSkillEditorMode, ManagedSkillEditorSeed } from '@/skills/skill_add_edit_modal';
import {
	canManageSkillPlugin,
	canWriteSkillPlugin,
	sameSkillArtifact,
	skillArtifactKey,
	skillMatchesManagementFilter,
} from '@/skills/lib/skill_management_utils';
import { ManagedSkillEditorModal, SkillArtifactDetailsModal } from '@/skills/skill_add_edit_modal';
import { SkillPluginDetailsModal } from '@/skills/skill_plugin_details_modal';
import { SkillPluginEditModal } from '@/skills/skill_plugin_edit_modal';

interface SkillPluginCardProps {
	plugin: PluginListItem;
	skills: StoreSkillListItem[];
	allSkills: StoreSkillListItem[];
	presentations: Record<string, SkillManagementPresentation>;
	filter: SkillManagementFilter;
	onReload: () => Promise<void>;
	skillsLoaded: boolean;
	skillsLoading: boolean;
	skillsError?: string;
	onLoadSkills: (plugin: PluginListItem) => Promise<void>;
	onRefresh: () => Promise<void>;
	onTogglePlugin: (plugin: PluginListItem, enabled: boolean) => Promise<void>;
	onToggleSkill: (skill: StoreSkillListItem, enabled: boolean) => Promise<void>;
	onAttachExisting: (plugin: PluginListItem, skill: StoreSkillListItem) => Promise<void>;
	onAttachPath: (plugin: PluginListItem, path: string) => Promise<void>;
	onDetachSkill: (plugin: PluginListItem, skill: StoreSkillListItem) => Promise<void>;
	onDeleteSkill: (plugin: PluginListItem, skill: StoreSkillListItem) => Promise<void>;
	onDeletePlugin: (plugin: PluginListItem) => Promise<void>;
	onError: (message: string) => void;
}

interface EditorState {
	mode: ManagedSkillEditorMode;
	skill?: StoreSkillListItem;
	seed?: ManagedSkillEditorSeed;
}

function refKey(value: { ref: { rootID: string; artifactID: string } }): string {
	return `${value.ref.rootID}:${value.ref.artifactID}`;
}

function pendingSkillActionKey(skill: StoreSkillListItem | null, action: string): string {
	return skill ? `${refKey(skill)}:${action}` : '';
}

function SkillPathAttachModal({
	isOpen,
	onClose,
	onSubmit,
}: {
	isOpen: boolean;
	onClose: () => void;
	onSubmit: (path: string) => Promise<void>;
}) {
	const [path, setPath] = useState('');
	const [error, setError] = useState('');
	const [submitting, setSubmitting] = useState(false);

	if (!isOpen) {
		return null;
	}

	const submit: SubmitEventHandler<HTMLFormElement> = event => {
		event.preventDefault();
		event.stopPropagation();

		if (!path.trim()) {
			setError('Choose an existing Skill directory.');
			return;
		}

		setSubmitting(true);
		setError('');

		void onSubmit(path.trim())
			.then(onClose)
			.catch((cause: unknown) => {
				setError(cause instanceof Error ? cause.message : 'Skill path could not be attached.');
			})
			.finally(() => {
				setSubmitting(false);
			});
	};

	return (
		<ModalDialog isOpen onClose={onClose} blockCancel={submitting}>
			<div className="modal-box bg-base-200 w-[calc(100%-1rem)] max-w-xl rounded-2xl p-0">
				<ModalHeader title="Attach Existing Skill Path" onClose={onClose} closeDisabled={submitting} />
				<form className="space-y-4 p-4 sm:p-6" onSubmit={submit}>
					{error ? <div className="alert alert-error rounded-2xl text-sm">{error}</div> : null}

					<ModalField label="Skill directory" htmlFor="skill-path" required>
						<div className="flex gap-2">
							<input
								id="skill-path"
								className="input min-w-0 flex-1 rounded-xl font-mono text-xs"
								value={path}
								onChange={event => {
									setPath(event.currentTarget.value);
								}}
								placeholder="/path/to/skill"
								disabled={submitting}
							/>
							<button
								type="button"
								className="btn btn-ghost rounded-xl"
								disabled={submitting}
								onClick={() => {
									void backendAPI
										.pickDirectoryPath()
										.then(selected => {
											if (selected) {
												setPath(selected);
											}
										})
										.catch((cause: unknown) => {
											setError(cause instanceof Error ? cause.message : 'Directory chooser failed.');
										});
								}}
							>
								<FiFolder size={15} />
								Choose
							</button>
						</div>
					</ModalField>

					<ModalActions>
						<button type="button" className="btn bg-base-300 rounded-xl" disabled={submitting} onClick={onClose}>
							Cancel
						</button>
						<button type="submit" className="btn btn-primary rounded-xl" disabled={submitting}>
							{submitting ? 'Attaching...' : 'Attach Skill'}
						</button>
					</ModalActions>
				</form>
			</div>
		</ModalDialog>
	);
}

function ExistingSkillAttachModal({
	isOpen,
	candidates,
	onClose,
	onAttach,
}: {
	isOpen: boolean;
	candidates: StoreSkillListItem[];
	onClose: () => void;
	onAttach: (skill: StoreSkillListItem) => Promise<void>;
}) {
	const [selectedKey, setSelectedKey] = useState('');
	const [error, setError] = useState('');
	const [submitting, setSubmitting] = useState(false);

	if (!isOpen) {
		return null;
	}

	const candidateDropdownItems = Object.fromEntries(
		candidates.map(candidate => [refKey(candidate), { isEnabled: true }])
	) as Record<string, DropdownItem>;
	const selected = candidates.find(candidate => refKey(candidate) === selectedKey);

	return (
		<ModalDialog isOpen onClose={onClose} blockCancel={submitting}>
			<div className="modal-box bg-base-200 w-[calc(100%-1rem)] max-w-xl rounded-2xl p-0">
				<ModalHeader title="Attach Existing Skill Artifact" onClose={onClose} closeDisabled={submitting} />
				<div className="space-y-4 p-4 sm:p-6">
					{error ? <div className="alert alert-error rounded-2xl text-sm">{error}</div> : null}

					<ModalField label="Skill Artifact" required>
						<Dropdown<string>
							dropdownItems={candidateDropdownItems}
							orderedKeys={candidates.map(c => {
								return refKey(c);
							})}
							selectedKey={selected ? selectedKey : ''}
							onChange={setSelectedKey}
							filterDisabled={false}
							title="Select a Skill Artifact"
							placeholderLabel="Select a Skill Artifact"
							getDisplayName={key => {
								const skill = candidates.find(candidate => refKey(candidate) === key);
								return skill ? `${skill.displayName || skill.name} · ${skill.ref.rootID}` : key;
							}}
							disabled={submitting || candidates.length === 0}
							inlineMenu
							maxMenuHeight={240}
						/>
					</ModalField>

					<ModalActions>
						<button type="button" className="btn bg-base-300 rounded-xl" disabled={submitting} onClick={onClose}>
							Cancel
						</button>
						<button
							type="button"
							className="btn btn-primary rounded-xl"
							disabled={!selected || submitting}
							onClick={() => {
								if (!selected) {
									return;
								}

								setSubmitting(true);
								setError('');

								void onAttach(selected)
									.then(onClose)
									.catch((cause: unknown) => {
										setError(cause instanceof Error ? cause.message : 'Skill Artifact could not be attached.');
									})
									.finally(() => {
										setSubmitting(false);
									});
							}}
						>
							<FiLink size={14} />
							<span>{submitting ? 'Attaching...' : 'Attach'}</span>
						</button>
					</ModalActions>
				</div>
			</div>
		</ModalDialog>
	);
}

export function SkillPluginCard({
	plugin,
	skills,
	allSkills,
	presentations,
	filter,
	onReload,
	skillsLoaded,
	skillsLoading,
	skillsError,
	onLoadSkills,
	onRefresh,
	onTogglePlugin,
	onToggleSkill,
	onAttachExisting,
	onAttachPath,
	onDetachSkill,
	onDeleteSkill,
	onDeletePlugin,
	onError,
}: SkillPluginCardProps) {
	const [expanded, setExpanded] = useState(false);
	const [editorState, setEditorState] = useState<EditorState | null>(null);
	const [skillToView, setSkillToView] = useState<StoreSkillListItem | null>(null);
	const [skillToPurge, setSkillToPurge] = useState<StoreSkillListItem | null>(null);
	const [skillToDetach, setSkillToDetach] = useState<StoreSkillListItem | null>(null);
	const { isPending, runAction } = usePendingActions();
	const [isPathOpen, setIsPathOpen] = useState(false);
	const [isAttachExistingOpen, setIsAttachExistingOpen] = useState(false);
	const [isDetailsOpen, setIsDetailsOpen] = useState(false);
	const [isEditPluginOpen, setIsEditPluginOpen] = useState(false);
	const [isDeletePluginOpen, setIsDeletePluginOpen] = useState(false);

	const attachedKeys = new Set(
		skills.map(r => {
			return refKey(r);
		})
	);

	for (const candidate of allSkills) {
		const memberships = presentations[skillArtifactKey(candidate.ref)]?.memberships;
		if (memberships?.some(membership => sameSkillArtifact(membership.plugin, plugin.ref))) {
			attachedKeys.add(refKey(candidate));
		}
	}

	const attachCandidates = allSkills.filter(skill => !attachedKeys.has(refKey(skill)));

	const refreshPluginSkills = async () => {
		await onRefresh();
	};

	const run = (key: string, operation: () => Promise<void>) => {
		runAction(key, operation).catch((cause: unknown) => {
			onError(cause instanceof Error ? cause.message : 'Skill action failed.');
		});
	};

	const visibleSkills = skills.filter(skill =>
		skillMatchesManagementFilter(skill, presentations[skillArtifactKey(skill.ref)], filter)
	);

	const load = () => {
		onLoadSkills(plugin).catch((cause: unknown) => {
			onError(cause instanceof Error ? cause.message : 'Skills could not be loaded for this Plugin.');
		});
	};

	const confirmDetachSkill = async () => {
		const skill = skillToDetach;
		if (!skill) {
			return;
		}

		try {
			await runAction(`${refKey(skill)}:detach`, () => onDetachSkill(plugin, skill));
			setSkillToDetach(null);
		} catch (cause) {
			onError(cause instanceof Error ? cause.message : 'Skill could not be detached from this Plugin.');
		}
	};

	const confirmDeleteSkill = async () => {
		const skill = skillToPurge;
		if (!skill) {
			return;
		}

		try {
			await runAction(`${refKey(skill)}:delete`, () => onDeleteSkill(plugin, skill));
			setSkillToPurge(null);
		} catch (cause) {
			onError(cause instanceof Error ? cause.message : 'Skill could not be removed from this Plugin.');
		}
	};

	const confirmDeletePlugin = async () => {
		try {
			await runAction('plugin:delete', () => onDeletePlugin(plugin));
			setIsDeletePluginOpen(false);
		} catch (cause) {
			onError(cause instanceof Error ? cause.message : 'Skill Plugin could not be deleted.');
		}
	};

	const pluginAvailable = plugin.state === ArtifactState.Available;
	const mutable = canManageSkillPlugin(plugin);
	const canWrite = canWriteSkillPlugin(plugin);

	return (
		<>
			<ManagementPluginCard
				title={plugin.displayName || plugin.name}
				identity={plugin.displayName !== plugin.name ? <span className="font-mono">{plugin.name}</span> : null}
				description={plugin.description}
				status={
					<>
						<StatusBadge tone={plugin.enabled ? 'success' : 'neutral'}>
							{plugin.enabled ? 'Enabled' : 'Disabled'}
						</StatusBadge>
						{plugin.state !== ArtifactState.Available ? <StatusBadge tone="warning">{plugin.state}</StatusBadge> : null}
						{plugin.builtIn ? <StatusBadge>Built-in</StatusBadge> : null}
						{plugin.baseline ? <StatusBadge>Baseline</StatusBadge> : null}
					</>
				}
				disclosure={
					<button
						type="button"
						className="btn btn-sm btn-ghost rounded-xl"
						aria-expanded={expanded}
						onClick={() => {
							setExpanded(value => !value);
							if (!skillsLoaded && !skillsLoading) {
								load();
							}
						}}
					>
						<span>
							{skillsLoading
								? 'Loading...'
								: skillsLoaded
									? `Skills: ${skills.length}`
									: `Members: ${plugin.memberCount}`}
						</span>
						{expanded ? <FiChevronUp /> : <FiChevronDown />}
					</button>
				}
				actionLeading={
					<EnabledControl
						id={`skill-plugin-${plugin.ref.rootID}-${plugin.ref.artifactID}`}
						checked={plugin.enabled}
						disabled={!pluginAvailable || isPending('plugin:toggle')}
						busy={isPending('plugin:toggle')}
						title={!pluginAvailable ? 'This Skill Plugin Artifact is unavailable.' : undefined}
						onChange={enabled => {
							run('plugin:toggle', () => onTogglePlugin(plugin, enabled));
						}}
					/>
				}
				actions={
					<>
						<button
							type="button"
							className="btn btn-sm btn-ghost rounded-xl"
							onClick={() => {
								setIsDetailsOpen(true);
							}}
						>
							<FiEye size={15} />
							Details
						</button>

						{mutable && !plugin.baseline ? (
							<button
								type="button"
								className="btn btn-sm btn-ghost rounded-xl"
								onClick={() => {
									setIsEditPluginOpen(true);
								}}
							>
								<FiEdit2 size={15} />
								Edit Plugin
							</button>
						) : null}

						<button
							type="button"
							className="btn btn-sm btn-ghost rounded-xl"
							disabled={skillsLoading || isPending('plugin:refresh')}
							onClick={() => {
								run('plugin:refresh', refreshPluginSkills);
							}}
						>
							<FiRefreshCw size={15} />
							Refresh
						</button>

						{mutable ? (
							<>
								<button
									type="button"
									className="btn btn-sm btn-ghost rounded-xl"
									disabled={!canWrite}
									onClick={() => {
										setEditorState({ mode: 'create' });
									}}
								>
									<FiPlus size={15} />
									Create Skill
								</button>

								<button
									type="button"
									className="btn btn-sm btn-ghost rounded-xl"
									disabled={!canWrite}
									onClick={() => {
										setIsPathOpen(true);
									}}
								>
									<FiFolder size={15} />
									Attach Path
								</button>

								<button
									type="button"
									className="btn btn-sm btn-ghost rounded-xl"
									disabled={!canWrite || attachCandidates.length === 0}
									onClick={() => {
										setIsAttachExistingOpen(true);
									}}
								>
									<FiLink size={15} />
									Attach Existing
								</button>
							</>
						) : null}

						{plugin.deletable ? (
							<button
								type="button"
								className="btn btn-sm btn-ghost rounded-xl"
								disabled={
									!skillsLoaded ||
									plugin.memberCount > 0 ||
									skills.length > 0 ||
									Boolean(skillsError) ||
									isPending('plugin:delete')
								}
								title={
									!skillsLoaded
										? 'Load Plugin Skills before deletion.'
										: skillsError
											? 'Reload Plugin Skills before deletion.'
											: skills.length > 0
												? 'Detach or delete all Skill Artifacts before deleting this Plugin.'
												: undefined
								}
								onClick={() => {
									setIsDeletePluginOpen(true);
								}}
							>
								<FiTrash2 size={15} />
								Delete Plugin
							</button>
						) : null}
					</>
				}
			>
				{skillsError ? (
					<div className="alert alert-warning mt-3 rounded-2xl text-sm">
						<div className="grow">{skillsError}</div>
						<button type="button" className="btn btn-xs rounded-lg" onClick={load}>
							Retry
						</button>
					</div>
				) : null}

				{expanded ? (
					<div className="mt-5 space-y-3">
						{!skillsLoaded ? (
							<ManagementEmptyState>
								{skillsLoading ? 'Loading Skills...' : 'Skill contents have not been loaded.'}
							</ManagementEmptyState>
						) : null}

						{visibleSkills.map(skill => {
							const presentation = presentations[skillArtifactKey(skill.ref)];
							const summary = presentation?.summary;
							const skillAvailable = skill.state === ArtifactState.Available;
							const editDisabled = !canWrite || !summary || summary.HasResources;

							return (
								<ManagementItemCard
									key={refKey(skill)}
									title={skill.displayName || skill.name}
									subtitle={skill.displayName !== skill.name ? skill.name : undefined}
									description={skill.description}
									status={
										<>
											<StatusBadge tone={skill.enabled ? 'success' : 'neutral'}>
												{skill.enabled ? 'Enabled' : 'Disabled'}
											</StatusBadge>
											{skill.state !== ArtifactState.Available ? (
												<StatusBadge tone="warning">{skill.state}</StatusBadge>
											) : null}
											{skill.managed ? <StatusBadge>Managed</StatusBadge> : null}
											{skill.builtIn ? <StatusBadge>Built-in</StatusBadge> : null}
										</>
									}
									metadata={
										<>
											<MetadataPill label="Revision">{skill.revision}</MetadataPill>
											{presentations[skillArtifactKey(skill.ref)]?.summary ? (
												<>
													<MetadataPill label="Insert">
														{presentations[skillArtifactKey(skill.ref)].summary?.Insert}
													</MetadataPill>
													<MetadataPill label="Arguments">
														{presentation?.document?.arguments?.length ??
															(presentations[skillArtifactKey(skill.ref)].summary?.HasArguments ? 'Declared' : 'None')}
													</MetadataPill>
													<MetadataPill label="Resources">
														{presentation?.resources?.totalCount ??
															(presentation?.summary?.HasResources ? 'Present' : 'None')}
													</MetadataPill>
												</>
											) : (
												<MetadataPill>Interpretation unavailable</MetadataPill>
											)}
											{presentation?.document?.tags?.map(tag => (
												<MetadataPill key={tag} label="Tag">
													{tag}
												</MetadataPill>
											))}
											{skill.definitionDigest ? <MetadataPill label="Definition">Pinned</MetadataPill> : null}
										</>
									}
								>
									<ActionRow
										leading={
											<EnabledControl
												id={`skill-${skill.ref.rootID}-${skill.ref.artifactID}`}
												checked={skill.enabled}
												disabled={!plugin.enabled || !skillAvailable || isPending(`${refKey(skill)}:toggle`)}
												busy={isPending(`${refKey(skill)}:toggle`)}
												title={
													!plugin.enabled
														? 'Enable the Skill Plugin first.'
														: !skillAvailable
															? 'This Skill Artifact is unavailable.'
															: undefined
												}
												onChange={enabled => {
													run(`${refKey(skill)}:toggle`, () => onToggleSkill(skill, enabled));
												}}
											/>
										}
									>
										<button
											type="button"
											className="btn btn-sm btn-ghost rounded-xl"
											onClick={() => {
												setSkillToView(skill);
											}}
										>
											<FiEye size={15} />
											View
										</button>

										{mutable && skill.managed && !skill.builtIn ? (
											<button
												type="button"
												className="btn btn-sm btn-ghost rounded-xl"
												disabled={editDisabled}
												title={
													!canWrite
														? 'Enable this editable Skill Plugin first.'
														: !summary
															? 'Skill interpretation is unavailable, so document-only replacement is blocked.'
															: summary.HasResources
																? 'This Skill contains package resources. Fork its document instead.'
																: undefined
												}
												onClick={() => {
													setEditorState({ mode: 'edit', skill });
												}}
											>
												<FiEdit2 size={15} />
												Edit
											</button>
										) : null}

										{mutable ? (
											<button
												type="button"
												className="btn btn-sm btn-ghost rounded-xl"
												disabled={!canWrite || !presentation?.document || isPending(`${refKey(skill)}:fork`)}
												title="Copy the verified document. Package resource files are not copied."
												onClick={() => {
													run(`${refKey(skill)}:fork`, async () => {
														const document = await skillManagementAPI.copySkillDocument(skill);
														setEditorState({
															mode: 'fork',
															seed: { document, enabled: true },
														});
													});
												}}
											>
												<FiGitBranch size={15} />
												Fork
											</button>
										) : null}

										<button
											type="button"
											className="btn btn-sm btn-ghost rounded-xl"
											disabled={!mutable || isPending(`${refKey(skill)}:detach`)}
											onClick={() => {
												setSkillToDetach(skill);
											}}
										>
											<FiX size={15} />
											Detach
										</button>

										{mutable && !skill.builtIn ? (
											<button
												type="button"
												className="btn btn-sm btn-ghost rounded-xl"
												onClick={() => {
													setSkillToPurge(skill);
												}}
											>
												<FiTrash2 size={15} />
												Delete from Plugin
											</button>
										) : null}
									</ActionRow>
								</ManagementItemCard>
							);
						})}

						{skillsLoaded && skills.length === 0 ? (
							<ManagementEmptyState>No Skill Artifact currently resolves through this Plugin.</ManagementEmptyState>
						) : null}
						{skillsLoaded && skills.length > 0 && visibleSkills.length === 0 ? (
							<ManagementEmptyState>No Skills match the current filters.</ManagementEmptyState>
						) : null}
					</div>
				) : null}
			</ManagementPluginCard>

			<ManagedSkillEditorModal
				isOpen={editorState !== null}
				plugin={plugin}
				skill={editorState?.skill}
				mode={editorState?.mode}
				seed={editorState?.seed}
				existingNames={allSkills.filter(skill => skill.ref.rootID === plugin.ref.rootID).map(skill => skill.name)}
				prefillDocuments={allSkills.flatMap(skill => {
					const document = presentations[skillArtifactKey(skill.ref)]?.document;
					return document
						? [{ key: skillArtifactKey(skill.ref), label: skill.displayName || skill.name, document }]
						: [];
				})}
				onClose={() => {
					setEditorState(null);
				}}
				onSaved={async () => {
					setEditorState(null);
					try {
						await onReload();
					} catch (cause) {
						onError(
							`Skill saved, but the page could not be reloaded: ${
								cause instanceof Error ? cause.message : 'unknown error'
							}`
						);
					}
				}}
			/>

			<SkillPathAttachModal
				isOpen={isPathOpen}
				onClose={() => {
					setIsPathOpen(false);
				}}
				onSubmit={async path => {
					await onAttachPath(plugin, path);
				}}
			/>

			<ExistingSkillAttachModal
				isOpen={isAttachExistingOpen}
				candidates={attachCandidates}
				onClose={() => {
					setIsAttachExistingOpen(false);
				}}
				onAttach={async skill => {
					await onAttachExisting(plugin, skill);
				}}
			/>

			<SkillArtifactDetailsModal
				isOpen={skillToView !== null}
				skill={skillToView}
				onClose={() => {
					setSkillToView(null);
				}}
			/>

			<SkillPluginDetailsModal
				key={`skill-plugin-details:${plugin.ref.rootID}:${plugin.ref.artifactID}:${plugin.revision}:${
					isDetailsOpen ? 'open' : 'closed'
				}`}
				isOpen={isDetailsOpen}
				plugin={plugin}
				onChanged={onReload}
				onClose={() => {
					setIsDetailsOpen(false);
				}}
			/>

			<SkillPluginEditModal
				isOpen={isEditPluginOpen}
				plugin={plugin}
				onClose={() => {
					setIsEditPluginOpen(false);
				}}
				onSaved={async () => {
					try {
						await onReload();
					} catch (cause) {
						onError(`Plugin updated, but refresh failed: ${cause instanceof Error ? cause.message : 'unknown error'}`);
					}
				}}
			/>

			<DeleteConfirmationModal
				isOpen={skillToDetach !== null}
				onClose={() => {
					if (!isPending(pendingSkillActionKey(skillToDetach, 'detach'))) {
						setSkillToDetach(null);
					}
				}}
				onConfirm={confirmDetachSkill}
				title="Detach Skill"
				message={`Remove "${skillToDetach?.displayName || skillToDetach?.name || ''}" from this Plugin? The Artifact and source files are retained.`}
				confirmButtonText="Detach"
			/>

			<DeleteConfirmationModal
				isOpen={skillToPurge !== null}
				onClose={() => {
					if (!isPending(pendingSkillActionKey(skillToPurge, 'delete'))) {
						setSkillToPurge(null);
					}
				}}
				onConfirm={confirmDeleteSkill}
				title="Delete Skill from Plugin"
				message={`Remove "${skillToPurge?.displayName || skillToPurge?.name || ''}" from this Plugin? An unshared managed Artifact is also deleted. Shared Artifacts and external source files are retained.`}
				confirmButtonText="Delete"
			/>

			<DeleteConfirmationModal
				isOpen={isDeletePluginOpen}
				onClose={() => {
					setIsDeletePluginOpen(false);
				}}
				onConfirm={confirmDeletePlugin}
				title="Delete Skill Plugin"
				message={`Delete empty Skill Plugin "${plugin.displayName || plugin.name}"?`}
				confirmButtonText="Delete"
			/>
		</>
	);
}
