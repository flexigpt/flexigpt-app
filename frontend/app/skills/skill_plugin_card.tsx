import { useMemo, useState } from 'react';
import { FiChevronDown, FiChevronUp, FiEdit2, FiEye, FiGitBranch, FiPlus, FiRefreshCw, FiTrash2 } from 'react-icons/fi';

import type { Skill, SkillPlugin } from '@/spec/skill';
import { SkillInsert, SkillPresenceStatus } from '@/spec/skill';

import { getErrorMessage } from '@/lib/error_utils';

import { usePendingActions } from '@/hooks/use_pending_actions';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';
import { ActionRow } from '@/components/managementui/action_row';
import { EnabledControl } from '@/components/managementui/enabled_control';
import { ManagementEmptyState } from '@/components/managementui/management_empty_state';
import { ManagementItemCard } from '@/components/managementui/management_item_card';
import { ManagementPluginCard } from '@/components/managementui/management_plugin_card';
import { MetadataPill } from '@/components/managementui/metadata_pill';
import { StatusBadge } from '@/components/managementui/status_badge';

import type { SkillInsertFilter } from '@/skills/lib/skill_artifact_utils';
import type { SkillItem, SkillUpsertInput } from '@/skills/skill_add_edit_modal';
import {
	getSkillArgumentCountLabel,
	getSkillArgumentTooltip,
	getSkillInsertDescription,
	getSkillInsertLabel,
	getSkillInsertShortLabel,
	getSkillInstructionPromptEligibilityReason,
	getSkillResourceCountLabel,
	getSkillResourceTooltip,
	normalizeSkillInsert,
	skillHasResources,
	skillMatchesInsertFilter,
	skillMatchesSearch,
	skillMatchesTags,
} from '@/skills/lib/skill_artifact_utils';
import { AddEditSkillModal } from '@/skills/skill_add_edit_modal';
import { SkillPluginDetailsModal } from '@/skills/skill_plugin_details_modal';
import { SkillPluginEditModal } from '@/skills/skill_plugin_edit_modal';

type SkillModalMode = 'add' | 'edit' | 'view' | 'fork';

interface SkillPluginCardProps {
	plugin: SkillPlugin;
	skills: Skill[];
	skillLoadError?: string;
	runtimeMetadataLoaded: boolean;
	prefillSkills: SkillItem[];
	onRefreshSkills: () => Promise<void>;
	insertFilter: SkillInsertFilter;
	searchQuery: string;
	tagFilters: string[];
	onTogglePluginEnable: (pluginID: string, nextEnabled: boolean) => Promise<void>;
	onToggleSkillEnable: (pluginID: string, skillID: string, skillSlug: string, nextEnabled: boolean) => Promise<void>;
	onDeleteSkill: (pluginID: string, skillID: string, skillSlug: string) => Promise<void>;
	onSubmitSkill: (pluginID: string, partial: SkillUpsertInput, existingSkillID?: string) => Promise<void>;
	onRequestPluginDelete: (plugin: SkillPlugin) => void;
	onEditPlugin: (pluginID: string, displayName: string, description?: string) => Promise<void>;
}

function PresenceStatusBadge({ skill }: { skill: Skill }) {
	const p = skill.presence;
	const status = p?.status ?? SkillPresenceStatus.Unknown;

	const { label, tone } = (() => {
		switch (status) {
			case SkillPresenceStatus.Present:
				return { label: 'Present', tone: 'success' as const };
			case SkillPresenceStatus.Missing:
				return { label: 'Missing', tone: 'warning' as const };
			case SkillPresenceStatus.Error:
				return { label: 'Error', tone: 'error' as const };
			default:
				return { label: 'Unknown', tone: 'neutral' as const };
		}
	})();

	const tooltip = [
		`Status: ${status}`,
		p?.lastCheckedAt ? `Last checked: ${p.lastCheckedAt}` : null,
		p?.lastSeenAt ? `Last seen: ${p.lastSeenAt}` : null,
		p?.missingSince ? `Missing since: ${p.missingSince}` : null,
		p?.lastCheckError ? `Error: ${p.lastCheckError}` : null,
	]
		.filter(Boolean)
		.join('\n');

	return (
		<StatusBadge tone={tone} title={tooltip}>
			{label}
		</StatusBadge>
	);
}

export function SkillPluginCard({
	plugin,
	skills,
	skillLoadError,
	runtimeMetadataLoaded,
	prefillSkills,
	onRefreshSkills,
	insertFilter,
	searchQuery,
	tagFilters,
	onTogglePluginEnable,
	onToggleSkillEnable,
	onDeleteSkill,
	onSubmitSkill,
	onRequestPluginDelete,
	onEditPlugin,
}: SkillPluginCardProps) {
	const [isExpanded, setIsExpanded] = useState(false);

	const [isDeleteSkillModalOpen, setIsDeleteSkillModalOpen] = useState(false);
	const [skillToDelete, setSkillToDelete] = useState<Skill | null>(null);

	const [isSkillModalOpen, setIsSkillModalOpen] = useState(false);
	const [skillModalMode, setSkillModalMode] = useState<SkillModalMode>('add');
	const [skillToEdit, setSkillToEdit] = useState<Skill | undefined>(undefined);

	const [isPluginDetailsOpen, setIsPluginDetailsOpen] = useState(false);
	const [isPluginEditOpen, setIsPluginEditOpen] = useState(false);

	const [showAlert, setShowAlert] = useState(false);
	const [alertMsg, setAlertMsg] = useState('');

	const { isPending, runAction } = usePendingActions();

	const existingSkillItems = useMemo(
		() =>
			skills.map(skill => ({
				skill,
				pluginID: plugin.id,
				skillSlug: skill.slug,
			})),
		[skills, plugin.id]
	);

	const visibleSkills = useMemo(
		() =>
			skills.filter(
				skill =>
					skillMatchesInsertFilter(skill.insert, insertFilter) &&
					skillMatchesSearch(skill, searchQuery) &&
					skillMatchesTags(skill, tagFilters)
			),
		[insertFilter, searchQuery, skills, tagFilters]
	);

	const hasActiveFilters = insertFilter !== 'all' || searchQuery.trim().length > 0 || tagFilters.length > 0;

	const runActionWithAlert = async (key: string, action: () => Promise<void>, fallback: string) => {
		try {
			await runAction(key, action);
		} catch (err) {
			setAlertMsg(getErrorMessage(err, fallback));
			setShowAlert(true);
			throw err;
		}
	};

	const togglePluginEnable = (nextEnabled: boolean) => {
		void runActionWithAlert(
			'plugin:toggle',
			() => onTogglePluginEnable(plugin.id, nextEnabled),
			'Failed to toggle plugin enable state.'
		).catch(() => undefined);
	};

	const patchSkillEnable = (skill: Skill, nextEnabled: boolean) => {
		void runActionWithAlert(
			`${skill.id}:toggle`,
			() => onToggleSkillEnable(plugin.id, skill.id, skill.slug, nextEnabled),
			'Failed to toggle skill.'
		).catch(() => undefined);
	};

	const requestDeleteSkill = (skill: Skill) => {
		if (!plugin.isEditable) {
			setAlertMsg('This Skill Plugin only supports enable and disable actions.');
			setShowAlert(true);
			return;
		}

		setSkillToDelete(skill);
		setIsDeleteSkillModalOpen(true);
	};

	const confirmDeleteSkill = async () => {
		if (!skillToDelete) {
			return;
		}

		try {
			await runActionWithAlert(
				`${skillToDelete.id}:delete`,
				() => onDeleteSkill(plugin.id, skillToDelete.id, skillToDelete.slug),
				'Failed to delete skill.'
			);
			setIsDeleteSkillModalOpen(false);
			setSkillToDelete(null);
		} catch {
			// ok.
		}
	};

	const openSkillModal = (mode: SkillModalMode, skill?: Skill) => {
		if ((mode === 'add' || mode === 'edit' || mode === 'fork') && !plugin.isEnabled) {
			setAlertMsg('Enable the Plugin before creating, editing, or forking a skill.');
			setShowAlert(true);
			return;
		}

		if ((mode === 'add' || mode === 'edit' || mode === 'fork') && !plugin.isEditable) {
			setAlertMsg('This Skill Plugin only supports enable and disable actions.');
			setShowAlert(true);
			return;
		}

		if (mode === 'edit' && (!skill?.isManaged || skillHasResources(skill))) {
			setAlertMsg('Fork this Skill to edit it. Managed Skills with package resources cannot be replaced safely.');
			setShowAlert(true);
			return;
		}

		setSkillModalMode(mode);
		setSkillToEdit(skill);
		setIsSkillModalOpen(true);
	};

	const refreshSkills = async () => {
		try {
			await runAction('plugin:refresh', onRefreshSkills);
		} catch (error) {
			setAlertMsg(getErrorMessage(error, 'Failed to reload Plugin skills.'));
			setShowAlert(true);
		}
	};

	const handleSubmitSkill = async (partial: SkillUpsertInput) => {
		const existingSkillID = skillModalMode === 'edit' ? skillToEdit?.id : undefined;
		await runAction(`${skillToEdit?.id ?? 'new'}:save`, () => onSubmitSkill(plugin.id, partial, existingSkillID));
	};

	return (
		<>
			<ManagementPluginCard
				title={plugin.displayName || plugin.slug}
				identity={
					plugin.displayName && plugin.displayName !== plugin.slug ? (
						<span className="font-mono">{plugin.slug}</span>
					) : null
				}
				description={plugin.description}
				status={
					<>
						<StatusBadge tone={plugin.isEnabled ? 'success' : 'neutral'}>
							{plugin.isEnabled ? 'Enabled' : 'Disabled'}
						</StatusBadge>
						<StatusBadge>{plugin.isBuiltIn ? 'Built-in' : 'Custom'}</StatusBadge>
						{!runtimeMetadataLoaded && !skillLoadError ? (
							<StatusBadge tone="neutral">Loading details</StatusBadge>
						) : null}
					</>
				}
				disclosure={
					<button
						type="button"
						className="btn btn-sm btn-ghost rounded-xl"
						aria-expanded={isExpanded}
						onClick={() => {
							setIsExpanded(previous => !previous);
						}}
					>
						<span className="whitespace-nowrap">
							Skills: {visibleSkills.length}
							{hasActiveFilters ? ` / ${skills.length}` : ''}
						</span>
						{isExpanded ? <FiChevronUp /> : <FiChevronDown />}
					</button>
				}
				actionLeading={
					<EnabledControl
						id={`skill-plugin-${plugin.id}`}
						checked={plugin.isEnabled}
						onChange={togglePluginEnable}
						busy={isPending('plugin:toggle')}
						compact={false}
					/>
				}
				actions={
					<>
						<button
							type="button"
							className="btn btn-sm btn-ghost rounded-xl"
							onClick={() => {
								setIsPluginDetailsOpen(true);
							}}
						>
							<FiEye size={16} />
							<span>Details</span>
						</button>
						{plugin.isEditable ? (
							<>
								{!plugin.isBaseline ? (
									<button
										type="button"
										className="btn btn-sm btn-ghost rounded-xl"
										onClick={() => {
											setIsPluginEditOpen(true);
										}}
									>
										<FiEdit2 size={16} />
										<span>Edit Plugin</span>
									</button>
								) : null}
								<button
									type="button"
									className="btn btn-sm btn-ghost rounded-xl"
									disabled={isPending('plugin:refresh') || !plugin.isEnabled}
									onClick={() => void refreshSkills()}
								>
									<FiRefreshCw size={16} />
									<span>{isPending('plugin:refresh') ? 'Refreshing...' : 'Refresh'}</span>
								</button>
								<button
									type="button"
									className="btn btn-sm btn-ghost rounded-xl"
									disabled={!plugin.isEnabled || Boolean(skillLoadError)}
									onClick={() => {
										openSkillModal('add');
									}}
								>
									<FiPlus size={16} />
									<span>Add Skill</span>
								</button>
							</>
						) : null}

						{plugin.isDeletable ? (
							<button
								type="button"
								className="btn btn-sm btn-ghost rounded-xl"
								disabled={skills.length > 0 || Boolean(skillLoadError)}
								onClick={() => {
									onRequestPluginDelete(plugin);
								}}
							>
								<FiTrash2 size={16} />
								<span>Delete Plugin</span>
							</button>
						) : null}
					</>
				}
			>
				{skillLoadError ? (
					<div className="alert alert-warning mt-3 rounded-2xl text-sm">
						<div className="grow">
							<div className="font-semibold">Skills could not be loaded for this Plugin</div>
							<div>{skillLoadError}</div>
						</div>
						<button
							type="button"
							className="btn btn-sm rounded-xl"
							onClick={() => void refreshSkills()}
							disabled={isPending('plugin:refresh')}
						>
							{isPending('plugin:refresh') ? 'Reloading…' : 'Retry'}
						</button>
					</div>
				) : null}

				{isExpanded && (
					<div className="mt-6 space-y-4">
						{insertFilter !== 'all' && (
							<div className="alert alert-info rounded-2xl py-3 text-sm">
								<div>
									Showing only <span className="font-semibold">{getSkillInsertShortLabel(insertFilter)}</span> skills.{' '}
									{getSkillInsertDescription(insertFilter)}
								</div>
							</div>
						)}
						{(searchQuery.trim() || tagFilters.length > 0) && (
							<div className="text-base-content/70 rounded-2xl px-1 text-xs">
								Additional filters active: {searchQuery.trim() ? `search "${searchQuery.trim()}"` : ''}
								{searchQuery.trim() && tagFilters.length > 0 ? ' · ' : ''}
								{tagFilters.length > 0 ? `tags ${tagFilters.join(', ')}` : ''}
							</div>
						)}

						<div className="space-y-3">
							{visibleSkills.map(skill => {
								const title = skill.displayName || skill.name || skill.slug;
								const insert = normalizeSkillInsert(skill.insert).value;
								const instructionUseReason = getSkillInstructionPromptEligibilityReason(skill);

								const usage =
									insert === SkillInsert.UserMessage
										? 'Composer template'
										: instructionUseReason
											? 'Session only'
											: 'System prompt eligible';

								return (
									<ManagementItemCard
										key={skill.id}
										title={title}
										subtitle={
											[skill.name, skill.slug]
												.filter((value, index, values) => value && value !== title && values.indexOf(value) === index)
												.join(' · ') || undefined
										}
										description={skill.description}
										status={
											<>
												<PresenceStatusBadge skill={skill} />
												<StatusBadge tone={skill.isEnabled ? 'success' : 'neutral'}>
													{skill.isEnabled ? 'Enabled' : 'Disabled'}
												</StatusBadge>
											</>
										}
										metadata={
											<>
												<MetadataPill label="Insert" title={getSkillInsertDescription(insert)}>
													{getSkillInsertLabel(skill.insert)}
												</MetadataPill>
												<MetadataPill label="Arguments" title={getSkillArgumentTooltip(skill.arguments)}>
													{getSkillArgumentCountLabel(skill.arguments)}
												</MetadataPill>
												<MetadataPill label="Resources" title={getSkillResourceTooltip(skill.resources)}>
													{getSkillResourceCountLabel(skill.resources)}
												</MetadataPill>
												<MetadataPill label="Usage" title={instructionUseReason}>
													{usage}
												</MetadataPill>
												{(skill.tags ?? []).map(tag => (
													<MetadataPill key={tag} label="Tag">
														{tag}
													</MetadataPill>
												))}
												{skill.isBuiltIn ? <MetadataPill>Built-in</MetadataPill> : null}
											</>
										}
									>
										{skill.runtimeWarnings?.length ? (
											<div className="text-warning mt-3 text-xs">
												{skill.runtimeWarnings.length} runtime warning
												{skill.runtimeWarnings.length === 1 ? '' : 's'}
											</div>
										) : null}

										<ActionRow
											leading={
												<EnabledControl
													id={`skill-${plugin.id}-${skill.id}`}
													checked={skill.isEnabled}
													onChange={enabled => {
														patchSkillEnable(skill, enabled);
													}}
													disabled={!plugin.isEnabled}
													busy={isPending(`${skill.id}:toggle`)}
													title={!plugin.isEnabled ? 'Enable the Plugin first.' : undefined}
												/>
											}
										>
											<button
												type="button"
												className="btn btn-sm btn-ghost rounded-xl"
												onClick={() => {
													openSkillModal('view', skill);
												}}
												title="View skill"
											>
												<FiEye size={15} />
												<span>View</span>
											</button>
											<button
												type="button"
												className="btn btn-sm btn-ghost rounded-xl"
												onClick={() => {
													openSkillModal('edit', skill);
												}}
												disabled={!plugin.isEditable || !skill.isManaged || skillHasResources(skill)}
												title={
													!plugin.isEditable
														? 'This Skill Plugin only supports enable and disable actions.'
														: !skill.isManaged
															? 'Only managed Skills can be edited. Fork this Skill to create a managed copy.'
															: skillHasResources(skill)
																? 'Fork this Skill to preserve package resources.'
																: 'Edit Skill'
												}
											>
												<FiEdit2 size={15} />
												<span>Edit</span>
											</button>
											<button
												type="button"
												className="btn btn-sm btn-ghost rounded-xl"
												onClick={() => {
													openSkillModal('fork', skill);
												}}
												disabled={!plugin.isEditable || !plugin.isEnabled}
												title={!plugin.isEnabled ? 'Enable the plugin before forking.' : 'Fork skill'}
											>
												<FiGitBranch size={15} />
												<span>Fork</span>
											</button>
											<button
												type="button"
												className="btn btn-sm btn-ghost rounded-xl"
												onClick={() => {
													requestDeleteSkill(skill);
												}}
												disabled={!plugin.isEditable || isPending(`${skill.id}:delete`)}
												title={!plugin.isEditable ? 'This Skill Plugin is read-only' : 'Delete'}
											>
												<FiTrash2 size={15} />
												<span>Delete</span>
											</button>
										</ActionRow>
									</ManagementItemCard>
								);
							})}

							{skills.length === 0 ? <ManagementEmptyState>No skills in this Plugin.</ManagementEmptyState> : null}

							{skills.length > 0 && visibleSkills.length === 0 ? (
								<ManagementEmptyState>No skills match the current filters.</ManagementEmptyState>
							) : null}
						</div>
					</div>
				)}
			</ManagementPluginCard>

			<DeleteConfirmationModal
				isOpen={isDeleteSkillModalOpen}
				onClose={() => {
					if (!skillToDelete || !isPending(`${skillToDelete.id}:delete`)) {
						setIsDeleteSkillModalOpen(false);
						setSkillToDelete(null);
					}
				}}
				onConfirm={confirmDeleteSkill}
				title="Delete Skill"
				message={`Delete skill "${skillToDelete?.displayName || skillToDelete?.name || ''}"? This cannot be undone.`}
				confirmButtonText="Delete"
			/>

			<AddEditSkillModal
				isOpen={isSkillModalOpen}
				onClose={() => {
					setIsSkillModalOpen(false);
					setSkillToEdit(undefined);
				}}
				onSubmit={handleSubmitSkill}
				mode={skillModalMode}
				initialData={skillToEdit ? { skill: skillToEdit, pluginID: plugin.id, skillSlug: skillToEdit.slug } : undefined}
				existingSkills={existingSkillItems}
				prefillSkills={prefillSkills}
			/>

			<SkillPluginDetailsModal
				isOpen={isPluginDetailsOpen}
				onClose={() => {
					setIsPluginDetailsOpen(false);
				}}
				plugin={plugin}
				skills={skills}
			/>

			<SkillPluginEditModal
				isOpen={isPluginEditOpen}
				onClose={() => {
					setIsPluginEditOpen(false);
				}}
				plugin={plugin}
				onSubmit={onEditPlugin}
			/>

			<ActionDeniedAlertModal
				isOpen={showAlert}
				onClose={() => {
					setShowAlert(false);
					setAlertMsg('');
				}}
				message={alertMsg}
			/>
		</>
	);
}
