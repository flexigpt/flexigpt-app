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
import { SkillInsert } from '@/spec/skill';

import { backendAPI } from '@/apis/baseapi';

import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';
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

import type { ManagedSkillEditorMode, ManagedSkillEditorSeed } from '@/skills/skill_add_edit_modal';
import { ManagedSkillEditorModal, SkillArtifactDetailsModal } from '@/skills/skill_add_edit_modal';
import { SkillPluginDetailsModal } from '@/skills/skill_plugin_details_modal';
import { SkillPluginEditModal } from '@/skills/skill_plugin_edit_modal';

interface SkillPluginCardProps {
	plugin: PluginListItem;
	skills: StoreSkillListItem[];
	allSkills: StoreSkillListItem[];
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
	onPurgeSkill: (skill: StoreSkillListItem) => Promise<void>;
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

function buildForkSeed(skill: StoreSkillListItem): ManagedSkillEditorSeed {
	const baseName = skill.name
		.toLowerCase()
		.replaceAll(/[^a-z0-9-]+/g, '-')
		.replaceAll(/-+/g, '-')
		.replaceAll(/^-|-$/g, '')
		.slice(0, 58);

	return {
		document: {
			name: baseName ? `${baseName}-copy` : 'skill-copy',
			displayName: `${skill.displayName || skill.name} Copy`,
			description: skill.description || `Forked from ${skill.displayName || skill.name}.`,
			insert: SkillInsert.Instructions,
			arguments: [],
			tags: ['forked'],
			markdownBody: `# ${skill.displayName || skill.name} Copy

Replace this placeholder with the managed Skill content you want to retain.

The original Skill Artifact remains unchanged.`,
		},
		enabled: true,
	};
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
									void backendAPI.pickDirectoryPath().then(selected => {
										if (selected) {
											setPath(selected);
										}
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

	const selected = candidates.find(candidate => refKey(candidate) === selectedKey);

	return (
		<ModalDialog isOpen onClose={onClose} blockCancel={submitting}>
			<div className="modal-box bg-base-200 w-[calc(100%-1rem)] max-w-xl rounded-2xl p-0">
				<ModalHeader title="Attach Existing Skill Artifact" onClose={onClose} closeDisabled={submitting} />
				<div className="space-y-4 p-4 sm:p-6">
					{error ? <div className="alert alert-error rounded-2xl text-sm">{error}</div> : null}

					<ModalField label="Skill Artifact" htmlFor="existing-skill-artifact" required>
						<select
							id="existing-skill-artifact"
							className="select w-full rounded-xl"
							value={selectedKey}
							disabled={submitting}
							onChange={event => {
								setSelectedKey(event.currentTarget.value);
							}}
						>
							<option value="">Select a Skill Artifact</option>
							{candidates.map(skill => (
								<option key={refKey(skill)} value={refKey(skill)}>
									{skill.displayName || skill.name} · {skill.ref.rootID}
								</option>
							))}
						</select>
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
	onPurgeSkill,
	onDeletePlugin,
	onError,
}: SkillPluginCardProps) {
	const [expanded, setExpanded] = useState(false);
	const [editorState, setEditorState] = useState<EditorState | null>(null);
	const [skillToView, setSkillToView] = useState<StoreSkillListItem | null>(null);
	const [skillToPurge, setSkillToPurge] = useState<StoreSkillListItem | null>(null);
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
	const attachCandidates = allSkills.filter(skill => !attachedKeys.has(refKey(skill)));

	const refreshPluginSkills = async () => {
		await onRefresh();
		await onLoadSkills(plugin);
	};

	const load = () => {
		void onLoadSkills(plugin).catch((cause: unknown) => {
			onError(cause instanceof Error ? cause.message : 'Skills could not be loaded for this Plugin.');
		});
	};

	const pluginAvailable = plugin.state === ArtifactState.Available;
	const canWrite = pluginAvailable && plugin.enabled && plugin.editable;

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
						disabled={!pluginAvailable}
						title={!pluginAvailable ? 'This Skill Plugin Artifact is unavailable.' : undefined}
						onChange={enabled => {
							void onTogglePlugin(plugin, enabled).catch((cause: unknown) => {
								onError(cause instanceof Error ? cause.message : 'Plugin enablement could not be changed.');
							});
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

						{plugin.editable && !plugin.baseline ? (
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
							disabled={skillsLoading}
							onClick={() => {
								void refreshPluginSkills().catch((cause: unknown) => {
									onError(cause instanceof Error ? cause.message : 'Skill Plugin could not be refreshed.');
								});
							}}
						>
							<FiRefreshCw size={15} />
							Refresh
						</button>

						{plugin.editable ? (
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
								disabled={!skillsLoaded || skills.length > 0 || Boolean(skillsError)}
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

						{skills.map(skill => (
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
										{skill.definitionDigest ? <MetadataPill label="Definition">Pinned</MetadataPill> : null}
									</>
								}
							>
								<ActionRow
									leading={
										<EnabledControl
											id={`skill-${skill.ref.rootID}-${skill.ref.artifactID}`}
											checked={skill.enabled}
											disabled={!plugin.enabled || skill.state !== ArtifactState.Available}
											title={!plugin.enabled ? 'Enable the Skill Plugin first.' : undefined}
											onChange={enabled => {
												void onToggleSkill(skill, enabled).catch((cause: unknown) => {
													onError(cause instanceof Error ? cause.message : 'Skill enablement could not be changed.');
												});
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

									{plugin.editable && skill.managed && !skill.builtIn ? (
										<button
											type="button"
											className="btn btn-sm btn-ghost rounded-xl"
											disabled={!canWrite}
											onClick={() => {
												setEditorState({ mode: 'edit', skill });
											}}
										>
											<FiEdit2 size={15} />
											Edit
										</button>
									) : null}

									{plugin.editable ? (
										<button
											type="button"
											className="btn btn-sm btn-ghost rounded-xl"
											disabled={!canWrite}
											onClick={() => {
												setEditorState({ mode: 'fork', seed: buildForkSeed(skill) });
											}}
										>
											<FiGitBranch size={15} />
											Fork
										</button>
									) : null}

									<button
										type="button"
										className="btn btn-sm btn-ghost rounded-xl"
										onClick={() => {
											void onDetachSkill(plugin, skill).catch((cause: unknown) => {
												onError(
													cause instanceof Error ? cause.message : 'Skill could not be detached from this Plugin.'
												);
											});
										}}
									>
										<FiX size={15} />
										Detach
									</button>

									{skill.managed && !skill.builtIn ? (
										<button
											type="button"
											className="btn btn-sm btn-ghost rounded-xl"
											onClick={() => {
												setSkillToPurge(skill);
											}}
										>
											<FiTrash2 size={15} />
											Delete Artifact
										</button>
									) : null}
								</ActionRow>
							</ManagementItemCard>
						))}

						{skillsLoaded && skills.length === 0 ? (
							<ManagementEmptyState>No Skill Artifact currently resolves through this Plugin.</ManagementEmptyState>
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
				onClose={() => {
					setEditorState(null);
				}}
				onSaved={async () => {
					setEditorState(null);
					await refreshPluginSkills();
				}}
			/>

			<SkillPathAttachModal
				isOpen={isPathOpen}
				onClose={() => {
					setIsPathOpen(false);
				}}
				onSubmit={async path => {
					await onAttachPath(plugin, path);
					await refreshPluginSkills();
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
					await refreshPluginSkills();
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
				isOpen={isDetailsOpen}
				plugin={plugin}
				skills={skills}
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
				onSaved={onRefresh}
			/>

			<DeleteConfirmationModal
				isOpen={skillToPurge !== null}
				onClose={() => {
					setSkillToPurge(null);
				}}
				onConfirm={() => {
					if (!skillToPurge) {
						return Promise.resolve();
					}

					return onPurgeSkill(skillToPurge)
						.then(async () => {
							setSkillToPurge(null);
							await refreshPluginSkills();
						})
						.catch((cause: unknown) => {
							onError(cause instanceof Error ? cause.message : 'Managed Skill Artifact could not be deleted.');
						});
				}}
				title="Delete Managed Skill Artifact"
				message={`Delete "${skillToPurge?.displayName || skillToPurge?.name || ''}" permanently?`}
				confirmButtonText="Delete"
			/>

			<DeleteConfirmationModal
				isOpen={isDeletePluginOpen}
				onClose={() => {
					setIsDeletePluginOpen(false);
				}}
				onConfirm={() => {
					return onDeletePlugin(plugin)
						.then(() => {
							setIsDeletePluginOpen(false);
						})
						.catch((cause: unknown) => {
							onError(cause instanceof Error ? cause.message : 'Skill Plugin could not be deleted.');
						});
				}}
				title="Delete Skill Plugin"
				message={`Delete empty Skill Plugin "${plugin.displayName || plugin.name}"?`}
				confirmButtonText="Delete"
			/>
		</>
	);
}
