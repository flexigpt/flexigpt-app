import { useEffect, useMemo, useState } from 'react';

import { FiPlus } from 'react-icons/fi';

import type { Skill, SkillBundle } from '@/spec/skill';

import { getUUIDv7 } from '@/lib/uuid_utils';

import { skillStoreAPI } from '@/apis/baseapi';
import { getAllSkillBundles, getAllSkills } from '@/apis/list_helper';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';
import { Loader } from '@/components/loader';
import { PageFrame } from '@/components/page_frame';

import { AddSkillBundleModal } from '@/skills/skill_bundle_add_modal';
import { SkillBundleCard } from '@/skills/skill_bundle_card';

interface BundleData {
	bundle: SkillBundle;
	skills: Skill[];
}

// eslint-disable-next-line no-restricted-exports
export default function SkillsPage() {
	const [bundles, setBundles] = useState<BundleData[]>([]);
	const [loading, setLoading] = useState(true);

	const [showAlert, setShowAlert] = useState(false);
	const [alertMsg, setAlertMsg] = useState('');

	const [bundleToDelete, setBundleToDelete] = useState<SkillBundle | null>(null);
	const [isAddModalOpen, setIsAddModalOpen] = useState(false);

	const existingBundleSlugs = useMemo(() => bundles.map(b => b.bundle.slug), [bundles]);
	const existingBundleNames = useMemo(
		() => bundles.map(b => (b.bundle.displayName ?? b.bundle.slug).trim()),
		[bundles]
	);

	const fetchAll = async () => {
		setLoading(true);
		let cancelled = false;

		try {
			const skillBundles = await getAllSkillBundles(undefined, true);
			const bundleResults: BundleData[] = await Promise.all(
				skillBundles.map(async b => {
					try {
						const skillListItems = await getAllSkills([b.id], undefined, true, true);
						const skills = skillListItems.map(itm => itm.skillDefinition);
						return { bundle: b, skills };
					} catch {
						return { bundle: b, skills: [] };
					}
				})
			);

			// sort built-in first then by display name
			bundleResults.sort((a, b) => {
				if (a.bundle.isBuiltIn !== b.bundle.isBuiltIn) return a.bundle.isBuiltIn ? -1 : 1;
				const an = (a.bundle.displayName ?? a.bundle.slug).toLowerCase();
				const bn = (b.bundle.displayName ?? b.bundle.slug).toLowerCase();
				return an.localeCompare(bn);
			});

			if (!cancelled) setBundles(bundleResults);
		} catch (err) {
			console.error('Load skill bundles failed:', err);
			setAlertMsg('Failed to load skill bundles. Please try again.');
			setShowAlert(true);
		} finally {
			if (!cancelled) setLoading(false);
		}

		return () => {
			cancelled = true;
		};
	};

	useEffect(() => {
		void fetchAll();
	}, []);

	const onSkillsChange = (bundleID: string, newSkills: Skill[]) => {
		setBundles(prev => prev.map(bd => (bd.bundle.id === bundleID ? { ...bd, skills: newSkills } : bd)));
	};

	const onBundleEnableChange = (bundleID: string, enabled: boolean) => {
		setBundles(prev =>
			prev.map(bd => (bd.bundle.id === bundleID ? { ...bd, bundle: { ...bd.bundle, isEnabled: enabled } } : bd))
		);
	};

	const handleBundleDelete = async () => {
		if (!bundleToDelete) return;
		try {
			await skillStoreAPI.deleteSkillBundle(bundleToDelete.id);
			setBundles(prev => prev.filter(bd => bd.bundle.id !== bundleToDelete.id));
		} catch (err) {
			console.error('Delete skill bundle failed:', err);
			setAlertMsg('Failed to delete skill bundle.');
			setShowAlert(true);
		} finally {
			setBundleToDelete(null);
		}
	};

	const handleAddBundle = async (slug: string, display: string, description?: string) => {
		try {
			const id = getUUIDv7();
			await skillStoreAPI.putSkillBundle(id, slug, display, true, description);
			setIsAddModalOpen(false);
			await fetchAll();
		} catch (err) {
			console.error('Add skill bundle failed:', err);
			setAlertMsg('Failed to add skill bundle.');
			setShowAlert(true);
		}
	};

	if (loading) return <Loader text="Loading skill bundles…" />;

	return (
		<PageFrame>
			<div className="flex h-full w-full flex-col items-center">
				<div className="fixed mt-8 flex w-10/12 items-center p-2 lg:w-2/3">
					<h1 className="flex grow items-center justify-center text-xl font-semibold">Skill Bundles</h1>
					<button
						className="btn btn-ghost flex items-center rounded-2xl"
						onClick={() => {
							setIsAddModalOpen(true);
						}}
					>
						<FiPlus size={20} /> <span className="ml-1">Add Bundle</span>
					</button>
				</div>

				<div
					className="mt-24 flex w-full grow flex-col items-center overflow-y-auto"
					style={{ maxHeight: `calc(100vh - 128px)` }}
				>
					<div className="flex w-5/6 flex-col space-y-4 xl:w-2/3">
						{bundles.length === 0 && <p className="mt-8 text-center text-sm">No skill bundles configured yet.</p>}

						{bundles.map(bd => (
							<SkillBundleCard
								key={bd.bundle.id}
								bundle={bd.bundle}
								skills={bd.skills}
								onSkillsChange={onSkillsChange}
								onBundleEnableChange={onBundleEnableChange}
								onBundleDeleted={b => {
									setBundleToDelete(b);
								}}
							/>
						))}
					</div>
				</div>

				<DeleteConfirmationModal
					isOpen={bundleToDelete !== null}
					onClose={() => {
						setBundleToDelete(null);
					}}
					onConfirm={handleBundleDelete}
					title="Delete Skill Bundle"
					message={`Delete bundle "${bundleToDelete?.displayName ?? ''}" and all its skills?`}
					confirmButtonText="Delete"
				/>

				<AddSkillBundleModal
					isOpen={isAddModalOpen}
					onClose={() => {
						setIsAddModalOpen(false);
					}}
					onSubmit={handleAddBundle}
					existingSlugs={existingBundleSlugs}
					existingNames={existingBundleNames}
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
		</PageFrame>
	);
}
