import { useEffect, useMemo, useRef, useState } from 'react';

import { FiChevronDown, FiChevronUp, FiEdit2, FiEye, FiGitBranch, FiPlus, FiTrash2 } from 'react-icons/fi';

import type { Skill, SkillBundle } from '@/spec/skill';
import { SkillPresenceStatus } from '@/spec/skill';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';

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

function PresenceBadge({ skill }: { skill: Skill }) {
	const p = skill.presence;
	const status = p?.status ?? SkillPresenceStatus.Unknown;

	const { label, cls } = (() => {
		switch (status) {
			case SkillPresenceStatus.Present:
				return { label: 'Present', cls: 'badge badge-success' };
			case SkillPresenceStatus.Missing:
				return { label: 'Missing', cls: 'badge badge-warning' };
			case SkillPresenceStatus.Error:
				return { label: 'Error', cls: 'badge badge-error' };
			default:
				return { label: 'Unknown', cls: 'badge badge-ghost' };
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
		<span className="tooltip tooltip-top" data-tip={tooltip}>
			<span className={`${cls} badge-sm h-auto max-w-full px-2 py-1 text-center wrap-break-word whitespace-normal`}>
				{label}
			</span>
		</span>
	);
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

	const [busyBundleToggle, setBusyBundleToggle] = useState(false);
	const [busySkillIDs, setBusySkillIDs] = useState<Set<string>>(new Set());

	const [isDeleteSkillModalOpen, setIsDeleteSkillModalOpen] = useState(false);
	const [isDeleteSkillPending, setIsDeleteSkillPending] = useState(false);
	const [skillToDelete, setSkillToDelete] = useState<Skill | null>(null);

	const [isSkillModalOpen, setIsSkillModalOpen] = useState(false);
	const [isSkillSubmitPending, setIsSkillSubmitPending] = useState(false);
	const [skillModalMode, setSkillModalMode] = useState<SkillModalMode>('add');
	const [skillToEdit, setSkillToEdit] = useState<Skill | undefined>(undefined);

	const [isBundleDetailsOpen, setIsBundleDetailsOpen] = useState(false);

	const [showAlert, setShowAlert] = useState(false);
	const [isRefreshingSkills, setIsRefreshingSkills] = useState(false);
	const [alertMsg, setAlertMsg] = useState('');

	const isMountedRef = useRef(false);

	useEffect(() => {
		isMountedRef.current = true;

		return () => {
			isMountedRef.current = false;
		};
	}, []);

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

	const toggleBundleEnable = async () => {
		if (busyBundleToggle) {
			return;
		}

		setBusyBundleToggle(true);

		try {
			await onToggleBundleEnable(bundle.id, !bundle.isEnabled);
		} catch (err) {
			console.error('Toggle skill bundle enable failed:', err);

			if (isMountedRef.current) {
				setAlertMsg('Failed to toggle bundle enable state.');
				setShowAlert(true);
			}
		} finally {
			if (isMountedRef.current) {
				setBusyBundleToggle(false);
			}
		}
	};

	const patchSkillEnable = async (skill: Skill) => {
		if (busySkillIDs.has(skill.id)) {
			return;
		}

		setBusySkillIDs(prev => {
			const next = new Set([...prev, skill.id]);
			return next;
		});

		try {
			await onToggleSkillEnable(bundle.id, skill.id, skill.slug, !skill.isEnabled);
		} catch (err) {
			console.error('Toggle skill failed:', err);

			if (isMountedRef.current) {
				setAlertMsg('Failed to toggle skill.');
				setShowAlert(true);
			}
		} finally {
			if (isMountedRef.current) {
				setBusySkillIDs(prev => {
					const next = new Set(prev);
					next.delete(skill.id);
					return next;
				});
			}
		}
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
		if (!skillToDelete || isDeleteSkillPending) {
			return;
		}

		setIsDeleteSkillPending(true);
		let deleted = false;

		try {
			await onDeleteSkill(bundle.id, skillToDelete.id, skillToDelete.slug);
			deleted = true;
		} catch (err) {
			console.error('Delete skill failed:', err);

			if (isMountedRef.current) {
				setAlertMsg(err instanceof Error ? err.message : 'Failed to delete skill.');
				setShowAlert(true);
			}
		} finally {
			if (isMountedRef.current) {
				setIsDeleteSkillPending(false);
				if (deleted) {
					setIsDeleteSkillModalOpen(false);
					setSkillToDelete(null);
				}
			}
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
		if (isRefreshingSkills) {
			return;
		}

		setIsRefreshingSkills(true);
		try {
			await onRefreshSkills();
		} catch (error) {
			setAlertMsg(error instanceof Error ? error.message : 'Failed to reload bundle skills.');
			setShowAlert(true);
		} finally {
			setIsRefreshingSkills(false);
		}
	};

	const handleSubmitSkill = async (partial: SkillUpsertInput) => {
		if (isSkillSubmitPending) {
			return;
		}

		setIsSkillSubmitPending(true);

		try {
			await onSubmitSkill(bundle.id, partial, skillToEdit?.slug);
		} finally {
			if (isMountedRef.current) {
				setIsSkillSubmitPending(false);
			}
		}
	};

	return (
		<section className="bg-base-100 border-base-content/10 mb-6 rounded-2xl border p-4 shadow-sm">
			<div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
				<div className="min-w-0">
					<h3 className="truncate text-sm font-semibold">
						<span className="capitalize">{bundle.displayName || bundle.slug}</span>
						<span className="text-base-content/60 ml-1">({bundle.slug})</span>
					</h3>
					<div className="text-base-content/60 mt-1 text-xs">
						{bundle.isBuiltIn ? 'Built-in bundle' : 'Custom bundle'}
					</div>
				</div>

				<div className="flex flex-wrap items-center justify-end gap-3">
					<button
						type="button"
						className="btn btn-sm btn-ghost rounded-xl"
						title="View bundle details"
						onClick={e => {
							e.stopPropagation();
							setIsBundleDetailsOpen(true);
						}}
					>
						<FiEye size={16} />
						<span>Details</span>
					</button>

					<div className="flex items-center gap-1">
						<label htmlFor={`skill-bundle-${bundle.id}`} className="text-sm">
							Enabled
						</label>
						<input
							id={`skill-bundle-${bundle.id}`}
							type="checkbox"
							className="toggle toggle-accent"
							checked={bundle.isEnabled}
							onChange={toggleBundleEnable}
							disabled={busyBundleToggle}
							aria-label={`Enable ${bundle.displayName || bundle.slug}`}
						/>
					</div>

					<button
						type="button"
						className="btn btn-sm btn-ghost rounded-xl"
						aria-expanded={isExpanded}
						onClick={() => {
							setIsExpanded(prev => !prev);
						}}
					>
						<span className="text-sm whitespace-nowrap">
							Skills: {visibleSkills.length}
							{insertFilter !== 'all' ? <span className="text-base-content/60"> / {skills.length}</span> : null}
						</span>
						{isExpanded ? <FiChevronUp /> : <FiChevronDown />}
					</button>
				</div>
			</div>

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
						disabled={isRefreshingSkills}
					>
						{isRefreshingSkills ? 'Reloading…' : 'Retry'}
					</button>
				</div>
			) : null}

			{isExpanded && (
				<div className="mt-8 space-y-4">
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
								<article
									key={skill.id}
									className="border-base-content/10 hover:border-base-content/20 rounded-2xl border p-4 transition-colors"
								>
									<div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
										<div className="min-w-0">
											<div className="truncate font-medium" title={skill.displayName || skill.name}>
												{skill.displayName || skill.name || skill.slug}
											</div>
											<div className="text-base-content/60 mt-1 text-xs break-all">
												{skill.slug} · {skill.name}
											</div>
											{skill.description ? (
												<p className="text-base-content/70 mt-2 max-h-10 overflow-hidden text-sm">
													{skill.description}
												</p>
											) : null}
										</div>

										<div className="flex shrink-0 flex-wrap items-center gap-2">
											<PresenceBadge skill={skill} />
											<span
												className={`badge h-auto px-2 py-1 text-center whitespace-normal ${
													skill.isEnabled ? 'badge-success' : 'badge-neutral'
												}`}
											>
												{skill.isEnabled ? 'Enabled' : 'Disabled'}
											</span>
										</div>
									</div>

									<div className="mt-3 flex flex-wrap gap-2 text-xs">
										<span
											className="border-base-content/20 rounded-xl border px-2 py-1"
											title={getSkillInsertDescription(insert)}
										>
											{getSkillInsertLabel(skill.insert)}
										</span>
										<span
											className="border-base-content/20 rounded-xl border px-2 py-1"
											title={getSkillArgumentTooltip(skill.arguments)}
										>
											{getSkillArgumentCountLabel(skill.arguments)}
										</span>
										<span
											className="border-base-content/20 rounded-xl border px-2 py-1"
											title={getSkillResourceTooltip(skill.resources)}
										>
											{getSkillResourceCountLabel(skill.resources)}
										</span>
										<span className="border-base-content/20 rounded-xl border px-2 py-1" title={instructionUseReason}>
											{usage}
										</span>
										{skill.isBuiltIn ? (
											<span className="border-base-content/20 rounded-xl border px-2 py-1">Built-in</span>
										) : null}
									</div>

									{(skill.tags ?? []).length > 0 ? (
										<div className="mt-3 flex flex-wrap gap-1">
											{(skill.tags ?? []).map(tag => (
												<span key={tag} className="border-base-content/20 rounded-lg border px-2 py-0.5 text-xs">
													{tag}
												</span>
											))}
										</div>
									) : null}

									{skill.runtimeWarnings?.length ? (
										<div className="text-warning mt-3 text-xs">
											{skill.runtimeWarnings.length} runtime warning
											{skill.runtimeWarnings.length === 1 ? '' : 's'}
										</div>
									) : null}

									<div className="border-base-content/10 mt-4 flex flex-col gap-3 border-t pt-3 sm:flex-row sm:items-center sm:justify-between">
										<div className="flex items-center gap-3">
											<label htmlFor={`skill-${skill.id}`} className="text-sm">
												Enabled
											</label>
											<input
												id={`skill-${skill.id}`}
												type="checkbox"
												className="toggle toggle-accent toggle-sm"
												checked={skill.isEnabled}
												onChange={() => patchSkillEnable(skill)}
												disabled={busySkillIDs.has(skill.id) || busyBundleToggle || !bundle.isEnabled}
												title={!bundle.isEnabled ? 'Enable the bundle first.' : undefined}
												aria-label={`Enable ${skill.displayName || skill.name}`}
											/>
											{busySkillIDs.has(skill.id) ? (
												<span className="loading loading-spinner loading-xs" aria-label="Updating skill" />
											) : null}
										</div>

										<div className="flex flex-wrap justify-end gap-2">
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
												disabled={skill.isBuiltIn || bundle.isBuiltIn}
												title={skill.isBuiltIn || bundle.isBuiltIn ? 'Built-in items cannot be deleted' : 'Delete'}
											>
												<FiTrash2 size={15} />
												<span>Delete</span>
											</button>
										</div>
									</div>
								</article>
							);
						})}

						{skills.length === 0 ? (
							<div className="border-base-content/10 rounded-2xl border py-6 text-center text-sm">
								No skills in this bundle.
							</div>
						) : null}

						{skills.length > 0 && visibleSkills.length === 0 ? (
							<div className="border-base-content/10 rounded-2xl border py-6 text-center text-sm">
								No skills match the current filters.
							</div>
						) : null}
					</div>

					{!bundle.isBuiltIn && (
						<div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
							<button
								type="button"
								className="btn btn-md btn-ghost flex items-center rounded-2xl"
								disabled={skills.length > 0 || Boolean(skillLoadError)}
								title={
									skillLoadError
										? 'Reload bundle skills before deleting the bundle.'
										: skills.length > 0
											? 'Delete all skills from this bundle first.'
											: 'Delete Bundle'
								}
								onClick={() => {
									onRequestBundleDelete(bundle);
								}}
							>
								<FiTrash2 /> <span className="ml-1">Delete Bundle</span>
							</button>

							<button
								type="button"
								className="btn btn-md btn-ghost flex items-center rounded-2xl"
								disabled={!bundle.isEnabled}
								title={
									!bundle.isEnabled
										? 'Enable the bundle first.'
										: 'Create a managed SKILL.md artifact or register an existing filesystem skill folder.'
								}
								onClick={() => {
									openSkillModal('add', undefined);
								}}
							>
								<FiPlus /> <span className="ml-1">Add Skill or Template</span>
							</button>
						</div>
					)}
				</div>
			)}

			<DeleteConfirmationModal
				isOpen={isDeleteSkillModalOpen}
				onClose={() => {
					if (!isDeleteSkillPending) {
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
		</section>
	);
}
