import { useEffect, useMemo, useState } from 'react';

import { FiCheck, FiChevronDown, FiChevronUp, FiEdit2, FiEye, FiPlus, FiTrash2, FiX } from 'react-icons/fi';

import type { Skill, SkillBundle } from '@/spec/skill';
import { SkillPresenceStatus } from '@/spec/skill';

import { skillStoreAPI } from '@/apis/baseapi';
import { getAllSkills } from '@/apis/list_helper';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';

import { AddEditSkillModal } from '@/skills/skill_add_edit_modal';
import { SkillBundleDetailsModal } from '@/skills/skill_bundle_details_modal';

type SkillModalMode = 'add' | 'edit' | 'view';

interface SkillBundleCardProps {
	bundle: SkillBundle;
	skills: Skill[];
	onSkillsChange: (bundleID: string, newSkills: Skill[]) => void;
	onBundleEnableChange: (bundleID: string, enabled: boolean) => void;
	onBundleDeleted: (bundle: SkillBundle) => void;
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
	onSkillsChange,
	onBundleEnableChange,
	onBundleDeleted,
}: SkillBundleCardProps) {
	const [isExpanded, setIsExpanded] = useState(false);
	const [localSkills, setLocalSkills] = useState<Skill[]>(skills);
	const [isBundleEnabled, setIsBundleEnabled] = useState(bundle.isEnabled);

	const [busyBundleToggle, setBusyBundleToggle] = useState(false);
	const [busySkillIDs, setBusySkillIDs] = useState<Set<string>>(new Set());

	useEffect(() => {
		setIsBundleEnabled(bundle.isEnabled);
	}, [bundle.isEnabled]);
	useEffect(() => {
		setLocalSkills(skills);
	}, [skills]);

	const [isDeleteSkillModalOpen, setIsDeleteSkillModalOpen] = useState(false);
	const [skillToDelete, setSkillToDelete] = useState<Skill | null>(null);

	const [isSkillModalOpen, setIsSkillModalOpen] = useState(false);
	const [skillModalMode, setSkillModalMode] = useState<SkillModalMode>('add');
	const [skillToEdit, setSkillToEdit] = useState<Skill | undefined>(undefined);

	const [isBundleDetailsOpen, setIsBundleDetailsOpen] = useState(false);

	const [showAlert, setShowAlert] = useState(false);
	const [alertMsg, setAlertMsg] = useState('');

	const existingSkillItems = useMemo(
		() =>
			localSkills.map(s => ({
				skill: s,
				bundleID: bundle.id,
				skillSlug: s.slug,
			})),
		[localSkills, bundle.id]
	);

	const refreshSkills = async () => {
		const listItems = await getAllSkills([bundle.id], undefined, true, true);
		const fresh = listItems.map(li => li.skillDefinition);
		setLocalSkills(fresh);
		onSkillsChange(bundle.id, fresh);
	};

	const toggleBundleEnable = async () => {
		if (busyBundleToggle) return;
		setBusyBundleToggle(true);
		try {
			const newVal = !isBundleEnabled;
			await skillStoreAPI.patchSkillBundle(bundle.id, newVal);
			setIsBundleEnabled(newVal);
			onBundleEnableChange(bundle.id, newVal);
		} catch (err) {
			console.error('Toggle skill bundle enable failed:', err);
			setAlertMsg('Failed to toggle bundle enable state.');
			setShowAlert(true);
		} finally {
			setBusyBundleToggle(false);
		}
	};

	const patchSkillEnable = async (skill: Skill) => {
		if (busySkillIDs.has(skill.id)) return;

		setBusySkillIDs(prev => new Set(prev).add(skill.id));
		try {
			await skillStoreAPI.patchSkill(bundle.id, skill.slug, !skill.isEnabled);
			const updated: Skill = { ...skill, isEnabled: !skill.isEnabled };
			const newArr = localSkills.map(s => (s.id === skill.id ? updated : s));
			setLocalSkills(newArr);
			onSkillsChange(bundle.id, newArr);
		} catch (err) {
			console.error('Toggle skill failed:', err);
			setAlertMsg('Failed to toggle skill.');
			setShowAlert(true);
		} finally {
			setBusySkillIDs(prev => {
				const next = new Set(prev);
				next.delete(skill.id);
				return next;
			});
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
		if (!skillToDelete) return;
		try {
			await skillStoreAPI.deleteSkill(bundle.id, skillToDelete.slug);
			const newArr = localSkills.filter(s => s.id !== skillToDelete.id);
			setLocalSkills(newArr);
			onSkillsChange(bundle.id, newArr);
		} catch (err) {
			console.error('Delete skill failed:', err);
			setAlertMsg('Failed to delete skill.');
			setShowAlert(true);
		} finally {
			setIsDeleteSkillModalOpen(false);
			setSkillToDelete(null);
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
		if (skillToEdit) {
			// Edit existing skill (no versioning)
			await skillStoreAPI.putSkill(
				bundle.id,
				skillToEdit.slug,
				skillToEdit.type,
				partial.location ?? skillToEdit.location,
				partial.name ?? skillToEdit.name,
				partial.isEnabled ?? skillToEdit.isEnabled,
				partial.displayName ?? skillToEdit.displayName,
				partial.description ?? skillToEdit.description,
				partial.tags ?? skillToEdit.tags
			);
		} else {
			// Add new skill
			const slug = (partial.slug ?? '').trim();
			const name = (partial.name ?? '').trim();
			const location = (partial.location ?? '').trim();
			if (!slug) throw new Error('Missing skill slug.');
			if (!name) throw new Error('Missing skill name.');
			if (!partial.type) throw new Error('Missing skill type.');
			if (!location) throw new Error('Missing skill location.');

			await skillStoreAPI.putSkill(
				bundle.id,
				slug,
				partial.type,
				location,
				name,
				partial.isEnabled ?? true,
				partial.displayName,
				partial.description,
				partial.tags
			);
		}

		await refreshSkills();
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
							checked={isBundleEnabled}
							onChange={toggleBundleEnable}
							disabled={busyBundleToggle}
						/>
					</div>

					<div
						className="flex cursor-pointer items-center gap-1"
						onClick={() => {
							setIsExpanded(p => !p);
						}}
					>
						<label className="text-sm whitespace-nowrap">Skills:&nbsp;{localSkills.length}</label>
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
								{localSkills.map(skill => (
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

								{localSkills.length === 0 && (
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
									onBundleDeleted(bundle);
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
