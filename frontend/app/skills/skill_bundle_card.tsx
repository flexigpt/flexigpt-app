import { useEffect, useMemo, useRef, useState } from 'react';

import { FiCheck, FiChevronDown, FiChevronUp, FiEdit2, FiEye, FiPlus, FiTrash2, FiX } from 'react-icons/fi';

import type { Skill, SkillBundle } from '@/spec/skill';
import { SkillPresenceStatus } from '@/spec/skill';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';

import { AddEditSkillModal } from '@/skills/skill_add_edit_modal';
import { SkillBundleDetailsModal } from '@/skills/skill_bundle_details_modal';

type SkillModalMode = 'add' | 'edit' | 'view';

interface SkillBundleCardProps {
	bundle: SkillBundle;
	skills: Skill[];
	onToggleBundleEnable: (bundleID: string, nextEnabled: boolean) => Promise<void>;
	onToggleSkillEnable: (bundleID: string, skillID: string, skillSlug: string, nextEnabled: boolean) => Promise<void>;
	onDeleteSkill: (bundleID: string, skillID: string, skillSlug: string) => Promise<void>;
	onSubmitSkill: (bundleID: string, partial: Partial<Skill>, existingSkillSlug?: string) => Promise<void>;
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
	onToggleBundleEnable,
	onToggleSkillEnable,
	onDeleteSkill,
	onSubmitSkill,
	onRequestBundleDelete,
}: SkillBundleCardProps) {
	const [isExpanded, setIsExpanded] = useState(false);

	const [busyBundleToggle, setBusyBundleToggle] = useState(false);
	const [busySkillIDs, setBusySkillIDs] = useState(new Set());

	const [isDeleteSkillModalOpen, setIsDeleteSkillModalOpen] = useState(false);
	const [isDeleteSkillPending, setIsDeleteSkillPending] = useState(false);
	const [skillToDelete, setSkillToDelete] = useState<Skill | null>(null);

	const [isSkillModalOpen, setIsSkillModalOpen] = useState(false);
	const [isSkillSubmitPending, setIsSkillSubmitPending] = useState(false);
	const [skillModalMode, setSkillModalMode] = useState<SkillModalMode>('add');
	const [skillToEdit, setSkillToEdit] = useState<Skill | undefined>(undefined);

	const [isBundleDetailsOpen, setIsBundleDetailsOpen] = useState(false);

	const [showAlert, setShowAlert] = useState(false);
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
			const next = new Set(prev);
			next.add(skill.id);
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

		try {
			await onDeleteSkill(bundle.id, skillToDelete.id, skillToDelete.slug);
		} catch (err) {
			console.error('Delete skill failed:', err);

			if (isMountedRef.current) {
				setAlertMsg('Failed to delete skill.');
				setShowAlert(true);
			}
		} finally {
			if (isMountedRef.current) {
				setIsDeleteSkillPending(false);
				setIsDeleteSkillModalOpen(false);
				setSkillToDelete(null);
			}
		}
	};

	const openSkillModal = (mode: SkillModalMode, skill?: Skill) => {
		if ((mode === 'add' || mode === 'edit') && bundle.isBuiltIn) {
			setAlertMsg('Cannot add or edit skills in a built-in bundle.');
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

	const handleSubmitSkill = async (partial: Partial<Skill>) => {
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
						<label className="text-sm whitespace-nowrap">Skills:&nbsp;{skills.length}</label>
						{isExpanded ? <FiChevronUp /> : <FiChevronDown />}
					</div>
				</div>
			</div>

			{isExpanded && (
				<div className="mt-8 space-y-4">
					<div className="border-base-content/10 overflow-x-auto rounded-2xl border">
						<table className="table-zebra table w-full">
							<thead>
								<tr className="bg-base-300 text-sm font-semibold">
									<th className="w-full">Display Name</th>
									<th className="min-w-32 text-center">Slug</th>
									<th className="min-w-28 text-center">Presence</th>
									<th className="text-center whitespace-nowrap">Enabled</th>
									<th className="text-center whitespace-nowrap">Built-In</th>
									<th className="text-center whitespace-nowrap">Actions</th>
								</tr>
							</thead>

							<tbody>
								{skills.map(skill => (
									<tr key={skill.id} className="hover:bg-base-300">
										<td>{skill.displayName || '-'}</td>
										<td className="text-center">{skill.slug}</td>
										<td className="text-center">
											<div className="flex items-center justify-center">
												<PresenceBadge skill={skill} />
											</div>
										</td>
										<td className="text-center align-middle">
											<input
												type="checkbox"
												className="toggle toggle-accent"
												checked={skill.isEnabled}
												onChange={() => patchSkillEnable(skill)}
												disabled={busySkillIDs.has(skill.id)}
											/>
										</td>
										<td className="text-center">
											{skill.isBuiltIn ? <FiCheck className="mx-auto" /> : <FiX className="mx-auto" />}
										</td>
										<td className="text-center">
											<div className="inline-flex items-center gap-2">
												<button
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
								))}

								{skills.length === 0 && (
									<tr>
										<td colSpan={9} className="py-3 text-center text-sm">
											No skills in this bundle.
										</td>
									</tr>
								)}
							</tbody>
						</table>
					</div>

					{!bundle.isBuiltIn && (
						<div className="flex items-center justify-between">
							<button
								className="btn btn-md btn-ghost flex items-center rounded-2xl"
								onClick={() => {
									onRequestBundleDelete(bundle);
								}}
							>
								<FiTrash2 /> <span className="ml-1">Delete Bundle</span>
							</button>

							<button
								className="btn btn-md btn-ghost flex items-center rounded-2xl"
								onClick={() => {
									openSkillModal('add', undefined);
								}}
							>
								<FiPlus /> <span className="ml-1">Add Skill</span>
							</button>
						</div>
					)}
				</div>
			)}

			<DeleteConfirmationModal
				isOpen={isDeleteSkillModalOpen}
				onClose={() => {
					setIsDeleteSkillModalOpen(false);
					setSkillToDelete(null);
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
			/>

			<SkillBundleDetailsModal
				isOpen={isBundleDetailsOpen}
				onClose={() => {
					setIsBundleDetailsOpen(false);
				}}
				bundle={bundle}
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
