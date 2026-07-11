import { useEffect, useMemo, useRef, useState } from 'react';

import {
	FiCheck,
	FiChevronDown,
	FiChevronUp,
	FiEdit2,
	FiEye,
	FiGitBranch,
	FiPlus,
	FiTrash2,
	FiX,
} from 'react-icons/fi';

import type { Skill, SkillBundle } from '@/spec/skill';
import { SkillPresenceStatus } from '@/spec/skill';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';

import type { SkillInsertFilter } from '@/skills/lib/skill_artifact_utils';
import {
	getSkillArgumentCountLabel,
	getSkillArgumentTooltip,
	getSkillInsertBadgeClass,
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
			<span className={`${cls} badge-sm whitespace-nowrap`}>{label}</span>
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
		<div className="bg-base-100 mb-8 rounded-2xl p-4 shadow-lg">
			<div className="flex items-center justify-between">
				<div className="flex items-center">
					<h3 className="gap-2 text-sm font-semibold">
						<span className="capitalize">{bundle.displayName || bundle.slug}</span>
						<span className="text-base-content/60 ml-1">({bundle.slug})</span>
					</h3>
				</div>

				<div className="flex items-center justify-end gap-4">
					<button
						type="button"
						className="btn btn-sm btn-ghost p-0"
						title="View bundle details"
						onClick={e => {
							e.stopPropagation();
							setIsBundleDetailsOpen(true);
						}}
					>
						<FiEye size={16} />
					</button>

					<span className="text-base-content/60 text-xs tracking-wide uppercase">
						{bundle.isBuiltIn ? 'Built-in' : 'Custom'}
					</span>

					<div className="flex items-center gap-1">
						<label className="text-sm">Enabled</label>
						<input
							type="checkbox"
							className="toggle toggle-accent"
							checked={bundle.isEnabled}
							onChange={toggleBundleEnable}
							disabled={busyBundleToggle}
						/>
					</div>

					<div
						className="flex cursor-pointer items-center gap-1"
						onClick={() => {
							setIsExpanded(prev => !prev);
						}}
					>
						<label className="text-sm whitespace-nowrap">
							Skills:&nbsp;{visibleSkills.length}
							{insertFilter !== 'all' ? <span className="text-base-content/60"> / {skills.length}</span> : null}
						</label>
						{isExpanded ? <FiChevronUp /> : <FiChevronDown />}
					</div>
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

					<div className="border-base-content/10 overflow-x-auto rounded-2xl border">
						<table className="table-zebra table w-full">
							<thead>
								<tr className="bg-base-300 text-sm font-semibold">
									<th className="w-full">Display Name</th>
									<th className="min-w-32 text-center">Slug</th>
									<th className="min-w-32 text-center">Insert</th>
									<th className="min-w-24 text-center">Args</th>
									<th className="min-w-28 text-center">Resources</th>
									<th className="min-w-36 text-center">Instruction use</th>
									<th className="min-w-28 text-center">Presence</th>
									<th className="min-w-24 text-center">Digest</th>
									<th className="text-center whitespace-nowrap">Enabled</th>
									<th className="text-center whitespace-nowrap">Built-In</th>
									<th className="text-center whitespace-nowrap">Actions</th>
								</tr>
							</thead>

							<tbody>
								{visibleSkills.map(skill => {
									const insert = normalizeSkillInsert(skill.insert).value;
									const instructionUseReason = getSkillInstructionPromptEligibilityReason(skill);

									return (
										<tr key={skill.id} className="hover:bg-base-300">
											<td>
												<div className="flex flex-col">
													<span>{skill.displayName || skill.name || '-'}</span>
													<span className="text-base-content/60 text-xs">artifact name: {skill.name}</span>
													{(skill.tags ?? []).length > 0 ? (
														<div className="mt-1 flex flex-wrap gap-1">
															{(skill.tags ?? []).map(tag => (
																<span key={tag} className="badge badge-outline badge-xs rounded-xl" title={tag}>
																	{tag}
																</span>
															))}
														</div>
													) : (
														<div className="text-base-content/50 mt-1 text-xs">No tags declared.</div>
													)}
													{skill.runtimeWarnings?.length ? (
														<span className="bg-warning/20 text-warning-content mt-1 inline-flex w-fit rounded-full px-2 py-0.5 text-xs">
															{skill.runtimeWarnings.length} runtime warning
															{skill.runtimeWarnings.length === 1 ? '' : 's'}
														</span>
													) : null}
												</div>
											</td>
											<td className="text-center">{skill.slug}</td>
											<td className="text-center">
												<span
													className={`badge rounded-xl ${getSkillInsertBadgeClass(insert)}`}
													title={getSkillInsertDescription(insert)}
												>
													{getSkillInsertLabel(skill.insert)}
												</span>
											</td>
											<td className="text-center">
												{skill.arguments?.length ? (
													<span className="tooltip tooltip-top" data-tip={getSkillArgumentTooltip(skill.arguments)}>
														<span className="badge badge-outline rounded-xl">
															{getSkillArgumentCountLabel(skill.arguments)}
														</span>
													</span>
												) : (
													<span className="text-base-content/60">-</span>
												)}
											</td>
											<td className="text-center">
												{skillHasResources(skill) ? (
													<span className="tooltip tooltip-top" data-tip={getSkillResourceTooltip(skill.resources)}>
														<span
															className={`badge rounded-xl ${
																insert === 'user-message' ? 'badge-warning' : 'badge-outline'
															}`}
														>
															{getSkillResourceCountLabel(skill.resources)}
														</span>
													</span>
												) : (
													<span className="text-base-content/60">-</span>
												)}
											</td>
											<td className="text-center">
												{insert === 'user-message' ? (
													<span
														className="badge badge-secondary rounded-xl"
														title="Use from the composer Templates menu."
													>
														Composer template
													</span>
												) : instructionUseReason ? (
													<span className="badge badge-warning rounded-xl" title={instructionUseReason}>
														Session only
													</span>
												) : (
													<span
														className="badge badge-success rounded-xl"
														title="Can be rendered and inserted into the system instruction prompt."
													>
														System prompt eligible
													</span>
												)}
											</td>
											<td className="text-center">
												<div className="flex items-center justify-center">
													<PresenceBadge skill={skill} />
												</div>
											</td>
											<td className="text-center">
												{skill.digest ? (
													<span className="font-mono text-xs" title={skill.digest}>
														{skill.digest.slice(0, 10)}
													</span>
												) : (
													<span className="text-base-content/60">-</span>
												)}
											</td>
											<td className="text-center align-middle">
												<input
													type="checkbox"
													className="toggle toggle-accent"
													checked={skill.isEnabled}
													onChange={() => patchSkillEnable(skill)}
													disabled={busySkillIDs.has(skill.id) || busyBundleToggle || !bundle.isEnabled}
													title={!bundle.isEnabled ? 'Enable the bundle first.' : undefined}
												/>
											</td>
											<td className="text-center">
												{skill.isBuiltIn ? <FiCheck className="mx-auto" /> : <FiX className="mx-auto" />}
											</td>
											<td className="text-center">
												<div className="inline-flex items-center gap-2">
													<button
														type="button"
														className="btn btn-sm btn-ghost rounded-2xl"
														onClick={() => {
															openSkillModal('view', skill);
														}}
														title="View"
														aria-label="View"
													>
														<FiEye size={16} />
													</button>

													<button
														type="button"
														className="btn btn-sm btn-ghost rounded-2xl"
														onClick={() => {
															openSkillModal('edit', skill);
														}}
														disabled={skill.isBuiltIn || bundle.isBuiltIn}
														title={skill.isBuiltIn || bundle.isBuiltIn ? 'Built-in items cannot be edited' : 'Edit'}
														aria-label="Edit"
													>
														<FiEdit2 size={16} />
													</button>

													<button
														type="button"
														className="btn btn-sm btn-ghost rounded-2xl"
														onClick={() => {
															openSkillModal('fork', skill);
														}}
														disabled={bundle.isBuiltIn || !bundle.isEnabled}
														title={
															bundle.isBuiltIn
																? 'To copy this built-in skill, use Add Skill or Template in a custom bundle and choose Copy Existing Skill.'
																: !bundle.isEnabled
																	? 'Enable the bundle before forking.'
																	: 'Fork into a new managed SKILL.md artifact'
														}
														aria-label="Fork"
													>
														<FiGitBranch size={16} />
													</button>

													<button
														type="button"
														className="btn btn-sm btn-ghost rounded-2xl"
														onClick={() => {
															requestDeleteSkill(skill);
														}}
														disabled={skill.isBuiltIn || bundle.isBuiltIn}
														title={
															skill.isBuiltIn || bundle.isBuiltIn ? 'Deleting disabled for built-in items' : 'Delete'
														}
														aria-label="Delete"
													>
														<FiTrash2 size={16} />
													</button>
												</div>
											</td>
										</tr>
									);
								})}

								{skills.length === 0 && (
									<tr>
										<td colSpan={11} className="py-3 text-center text-sm">
											No skills in this bundle.
										</td>
									</tr>
								)}

								{skills.length > 0 && visibleSkills.length === 0 && (
									<tr>
										<td colSpan={11} className="py-3 text-center text-sm">
											No skills match the current filters.
										</td>
									</tr>
								)}
							</tbody>
						</table>
					</div>

					{!bundle.isBuiltIn && (
						<div className="flex items-center justify-between">
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
		</div>
	);
}
