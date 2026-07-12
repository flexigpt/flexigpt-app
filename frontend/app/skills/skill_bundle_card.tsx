import { useMemo, useState } from 'react';

import { FiChevronDown, FiChevronUp, FiEdit2, FiEye, FiGitBranch, FiPlus, FiTrash2 } from 'react-icons/fi';

import type { Skill, SkillBundle } from '@/spec/skill';
import { SkillPresenceStatus } from '@/spec/skill';

import { usePendingActions } from '@/hooks/use_pending_actions';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';
import { ActionRow } from '@/components/managementui/action_row';
import { EnabledControl } from '@/components/managementui/enabled_control';
import { ManagementBundleCard } from '@/components/managementui/management_bundle_card';
import { ManagementEmptyState } from '@/components/managementui/management_empty_state';
import { ManagementItemCard } from '@/components/managementui/management_item_card';
import { MetadataPill } from '@/components/managementui/metadata_pill';
import { StatusBadge } from '@/components/managementui/status_badge';

import type { SkillInsertFilter } from '@/skills/lib/skill_artifact_utils';
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
	skillMatchesInsertFilter,
	skillMatchesSearch,
	skillMatchesTags,
} from '@/skills/lib/skill_artifact_utils';
import type { SkillItem, SkillUpsertInput } from '@/skills/skill_add_edit_modal';
import { AddEditSkillModal } from '@/skills/skill_add_edit_modal';
import { SkillBundleDetailsModal } from '@/skills/skill_bundle_details_modal';

type SkillModalMode = 'add' | 'edit' | 'view' | 'fork';

interface SkillBundleCardProps {
	bundle: SkillBundle;
	skills: Skill[];
	skillLoadError?: string;
	prefillSkills: SkillItem[];
	onRefreshSkills: () => Promise<void>;
	insertFilter: SkillInsertFilter;
	searchQuery: string;
	tagFilters: string[];
	onToggleBundleEnable: (bundleID: string, nextEnabled: boolean) => Promise<void>;
	onToggleSkillEnable: (bundleID: string, skillID: string, skillSlug: string, nextEnabled: boolean) => Promise<void>;
	onDeleteSkill: (bundleID: string, skillID: string, skillSlug: string) => Promise<void>;
	onSubmitSkill: (bundleID: string, partial: SkillUpsertInput, existingSkillSlug?: string) => Promise<void>;
	onRequestBundleDelete: (bundle: SkillBundle) => void;
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

function getErrorMessage(error: unknown, fallback: string): string {
	return error instanceof Error && error.message.trim() ? error.message : fallback;
}

export function SkillBundleCard({
	bundle,
	skills,
	skillLoadError,
	prefillSkills,
	onRefreshSkills,
	insertFilter,
	searchQuery,
	tagFilters,
	onToggleBundleEnable,
	onToggleSkillEnable,
	onDeleteSkill,
	onSubmitSkill,
	onRequestBundleDelete,
}: SkillBundleCardProps) {
	const [isExpanded, setIsExpanded] = useState(false);

	const [isDeleteSkillModalOpen, setIsDeleteSkillModalOpen] = useState(false);
	const [skillToDelete, setSkillToDelete] = useState<Skill | null>(null);

	const [isSkillModalOpen, setIsSkillModalOpen] = useState(false);
	const [skillModalMode, setSkillModalMode] = useState<SkillModalMode>('add');
	const [skillToEdit, setSkillToEdit] = useState<Skill | undefined>(undefined);

	const [isBundleDetailsOpen, setIsBundleDetailsOpen] = useState(false);

	const [showAlert, setShowAlert] = useState(false);
	const [alertMsg, setAlertMsg] = useState('');

	const { isPending, runAction } = usePendingActions();

	const existingSkillItems = useMemo(
		() =>
			skills.map(skill => ({
				skill,
				bundleID: bundle.id,
				skillSlug: skill.slug,
			})),
		[skills, bundle.id]
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

	const runActionWithAlert = async (key: string, action: () => Promise<void>, fallback: string) => {
		try {
			await runAction(key, action);
		} catch (err) {
			setAlertMsg(getErrorMessage(err, fallback));
			setShowAlert(true);
			throw err;
		}
	};

	const toggleBundleEnable = (nextEnabled: boolean) => {
		void runActionWithAlert(
			'bundle:toggle',
			() => onToggleBundleEnable(bundle.id, nextEnabled),
			'Failed to toggle bundle enable state.'
		).catch(() => undefined);
	};

	const patchSkillEnable = (skill: Skill, nextEnabled: boolean) => {
		void runActionWithAlert(
			`${skill.id}:toggle`,
			() => onToggleSkillEnable(bundle.id, skill.id, skill.slug, nextEnabled),
			'Failed to toggle skill.'
		).catch(() => undefined);
	};

	const requestDeleteSkill = (skill: Skill) => {
		if (skill.isBuiltIn) {
			setAlertMsg('Cannot delete built-in skill.');
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
				() => onDeleteSkill(bundle.id, skillToDelete.id, skillToDelete.slug),
				'Failed to delete skill.'
			);
			setIsDeleteSkillModalOpen(false);
			setSkillToDelete(null);
		} catch {
			// ok.
		}
	};

	const openSkillModal = (mode: SkillModalMode, skill?: Skill) => {
		if ((mode === 'add' || mode === 'fork') && !bundle.isEnabled) {
			setAlertMsg('Enable the bundle before creating or forking a skill. Enabled skills are indexed by the runtime.');
			setShowAlert(true);
			return;
		}

		if ((mode === 'add' || mode === 'edit') && bundle.isBuiltIn) {
			setAlertMsg('Cannot add or edit skills in a built-in bundle.');
			setShowAlert(true);
			return;
		}

		if (mode === 'fork' && bundle.isBuiltIn) {
			setAlertMsg('Forking into a built-in bundle is not supported. Create or use a custom bundle first.');
			setShowAlert(true);
			return;
		}

		if (mode === 'edit' && skill?.isBuiltIn) {
			setAlertMsg('Built-in skills cannot be edited (only enabled/disabled).');
			setShowAlert(true);
			return;
		}

		setSkillModalMode(mode);
		setSkillToEdit(skill);
		setIsSkillModalOpen(true);
	};

	const refreshSkills = async () => {
		try {
			await runAction('bundle:refresh', onRefreshSkills);
		} catch (error) {
			setAlertMsg(getErrorMessage(error, 'Failed to reload bundle skills.'));
			setShowAlert(true);
		}
	};

	const handleSubmitSkill = async (partial: SkillUpsertInput) => {
		const existingSkillSlug = skillModalMode === 'edit' ? skillToEdit?.slug : undefined;
		await runAction(`${skillToEdit?.id ?? 'new'}:save`, () => onSubmitSkill(bundle.id, partial, existingSkillSlug));
	};

	return (
		<>
			<ManagementBundleCard
				title={bundle.displayName || bundle.slug}
				identity={
					<span className="font-mono">
						{bundle.slug} / {bundle.id}
					</span>
				}
				description={bundle.description}
				status={
					<>
						<StatusBadge tone={bundle.isEnabled ? 'success' : 'neutral'}>
							{bundle.isEnabled ? 'Enabled' : 'Disabled'}
						</StatusBadge>
						<StatusBadge>{bundle.isBuiltIn ? 'Built-in' : 'Custom'}</StatusBadge>
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
							{insertFilter !== 'all' ? ` / ${skills.length}` : ''}
						</span>
						{isExpanded ? <FiChevronUp /> : <FiChevronDown />}
					</button>
				}
				actionLeading={
					<EnabledControl
						id={`skill-bundle-${bundle.id}`}
						checked={bundle.isEnabled}
						onChange={toggleBundleEnable}
						busy={isPending('bundle:toggle')}
						compact={false}
					/>
				}
				actions={
					<>
						<button
							type="button"
							className="btn btn-sm btn-ghost rounded-xl"
							onClick={() => {
								setIsBundleDetailsOpen(true);
							}}
						>
							<FiEye size={16} />
							<span>Details</span>
						</button>
						{!bundle.isBuiltIn ? (
							<>
								<button
									type="button"
									className="btn btn-sm btn-ghost rounded-xl"
									disabled={!bundle.isEnabled || Boolean(skillLoadError)}
									onClick={() => {
										openSkillModal('add');
									}}
								>
									<FiPlus size={16} />
									<span>Add Skill</span>
								</button>
								<button
									type="button"
									className="btn btn-sm btn-ghost rounded-xl"
									disabled={skills.length > 0 || Boolean(skillLoadError)}
									onClick={() => {
										onRequestBundleDelete(bundle);
									}}
								>
									<FiTrash2 size={16} />
									<span>Delete Bundle</span>
								</button>
							</>
						) : null}
					</>
				}
			>
				{skillLoadError ? (
					<div className="alert alert-warning mt-3 rounded-2xl text-sm">
						<div className="grow">
							<div className="font-semibold">Skills could not be loaded for this bundle</div>
							<div>{skillLoadError}</div>
						</div>
						<button
							type="button"
							className="btn btn-sm rounded-xl"
							onClick={() => void refreshSkills()}
							disabled={isPending('bundle:refresh')}
						>
							{isPending('bundle:refresh') ? 'Reloading…' : 'Retry'}
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
								const insert = normalizeSkillInsert(skill.insert).value;
								const instructionUseReason = getSkillInstructionPromptEligibilityReason(skill);
								const usage =
									insert === 'user-message'
										? 'Composer template'
										: instructionUseReason
											? 'Session only'
											: 'System prompt eligible';

								return (
									<ManagementItemCard
										key={skill.id}
										title={skill.displayName || skill.name || skill.slug}
										subtitle={`${skill.slug} / ${skill.name}`}
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
													id={`skill-${bundle.id}-${skill.id}`}
													checked={skill.isEnabled}
													onChange={enabled => {
														patchSkillEnable(skill, enabled);
													}}
													disabled={!bundle.isEnabled}
													busy={isPending(`${skill.id}:toggle`)}
													title={!bundle.isEnabled ? 'Enable the bundle first.' : undefined}
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
												disabled={skill.isBuiltIn || bundle.isBuiltIn}
												title={skill.isBuiltIn || bundle.isBuiltIn ? 'Built-in items cannot be edited' : 'Edit'}
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
												disabled={bundle.isBuiltIn || !bundle.isEnabled}
												title={!bundle.isEnabled ? 'Enable the bundle before forking.' : 'Fork skill'}
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
												disabled={skill.isBuiltIn || bundle.isBuiltIn || isPending(`${skill.id}:delete`)}
												title={skill.isBuiltIn || bundle.isBuiltIn ? 'Built-in items cannot be deleted' : 'Delete'}
											>
												<FiTrash2 size={15} />
												<span>Delete</span>
											</button>
										</ActionRow>
									</ManagementItemCard>
								);
							})}

							{skills.length === 0 ? <ManagementEmptyState>No skills in this bundle.</ManagementEmptyState> : null}

							{skills.length > 0 && visibleSkills.length === 0 ? (
								<ManagementEmptyState>No skills match the current filters.</ManagementEmptyState>
							) : null}
						</div>
					</div>
				)}
			</ManagementBundleCard>

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
				message={`Delete skill "${skillToDelete?.displayName ?? skillToDelete?.name ?? ''}"? This cannot be undone.`}
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
				initialData={skillToEdit ? { skill: skillToEdit, bundleID: bundle.id, skillSlug: skillToEdit.slug } : undefined}
				existingSkills={existingSkillItems}
				prefillSkills={prefillSkills}
			/>

			<SkillBundleDetailsModal
				isOpen={isBundleDetailsOpen}
				onClose={() => {
					setIsBundleDetailsOpen(false);
				}}
				bundle={bundle}
				skills={skills}
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
