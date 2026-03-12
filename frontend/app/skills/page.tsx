import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

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

function sortBundleData(bundleData: BundleData[]): BundleData[] {
	return [...bundleData].sort((a, b) => {
		if (a.bundle.isBuiltIn !== b.bundle.isBuiltIn) {
			return a.bundle.isBuiltIn ? -1 : 1;
		}

		const aName = (a.bundle.displayName ?? a.bundle.slug).toLowerCase();
		const bName = (b.bundle.displayName ?? b.bundle.slug).toLowerCase();

		return aName.localeCompare(bName);
	});
}

// eslint-disable-next-line no-restricted-exports
export default function SkillsPage() {
	const [bundles, setBundles] = useState<BundleData[]>([]);
	const [loading, setLoading] = useState(true);

	const [showAlert, setShowAlert] = useState(false);
	const [alertMsg, setAlertMsg] = useState('');

	const [bundleToDelete, setBundleToDelete] = useState<SkillBundle | null>(null);
	const [isDeletingBundle, setIsDeletingBundle] = useState(false);
	const [isAddModalOpen, setIsAddModalOpen] = useState(false);

	const isMountedRef = useRef(false);
	const fetchRequestIdRef = useRef(0);
	const bundleRefreshRequestIdRef = useRef<Record<string, number>>({});

	const existingBundleSlugs = useMemo(() => bundles.map(bundleData => bundleData.bundle.slug), [bundles]);
	const existingBundleNames = useMemo(
		() => bundles.map(bundleData => (bundleData.bundle.displayName ?? bundleData.bundle.slug).trim()),
		[bundles]
	);

	const fetchAll = useCallback(async () => {
		const requestId = ++fetchRequestIdRef.current;

		if (isMountedRef.current) {
			setLoading(true);
		}

		try {
			const skillBundles = await getAllSkillBundles(undefined, true);

			const bundleResults = await Promise.all(
				skillBundles.map(async bundle => {
					try {
						const skillListItems = await getAllSkills([bundle.id], undefined, true, true);
						const bundleSkills = skillListItems.map(item => item.skillDefinition);

						return { bundle, skills: bundleSkills };
					} catch {
						return { bundle, skills: [] };
					}
				})
			);

			if (!isMountedRef.current || fetchRequestIdRef.current !== requestId) {
				return;
			}

			setBundles(sortBundleData(bundleResults));
		} catch (err) {
			if (!isMountedRef.current || fetchRequestIdRef.current !== requestId) {
				return;
			}

			console.error('Load skill bundles failed:', err);
			setAlertMsg('Failed to load skill bundles. Please try again.');
			setShowAlert(true);
		} finally {
			if (isMountedRef.current && fetchRequestIdRef.current === requestId) {
				setLoading(false);
			}
		}
	}, []);

	const refreshBundleSkills = useCallback(async (bundleID: string) => {
		const requestId = (bundleRefreshRequestIdRef.current[bundleID] ?? 0) + 1;
		bundleRefreshRequestIdRef.current[bundleID] = requestId;

		try {
			const skillListItems = await getAllSkills([bundleID], undefined, true, true);
			const freshSkills = skillListItems.map(item => item.skillDefinition);

			if (!isMountedRef.current || bundleRefreshRequestIdRef.current[bundleID] !== requestId) {
				return;
			}

			setBundles(prev =>
				prev.map(bundleData =>
					bundleData.bundle.id === bundleID ? { ...bundleData, skills: freshSkills } : bundleData
				)
			);
		} catch (err) {
			console.error('Refresh bundle skills failed:', err);
			throw err;
		}
	}, []);

	useEffect(() => {
		isMountedRef.current = true;

		return () => {
			isMountedRef.current = false;
		};
	}, []);

	useEffect(() => {
		void fetchAll();
	}, [fetchAll]);

	const handleBundleEnableChange = useCallback(async (bundleID: string, nextEnabled: boolean) => {
		try {
			await skillStoreAPI.patchSkillBundle(bundleID, nextEnabled);

			if (!isMountedRef.current) {
				return;
			}

			setBundles(prev =>
				prev.map(bundleData =>
					bundleData.bundle.id === bundleID
						? {
								...bundleData,
								bundle: { ...bundleData.bundle, isEnabled: nextEnabled },
							}
						: bundleData
				)
			);
		} catch (err) {
			console.error('Toggle skill bundle enable failed:', err);
			throw err;
		}
	}, []);

	const handleSkillEnableChange = useCallback(
		async (bundleID: string, skillID: string, skillSlug: string, nextEnabled: boolean) => {
			try {
				await skillStoreAPI.patchSkill(bundleID, skillSlug, nextEnabled);

				if (!isMountedRef.current) {
					return;
				}

				setBundles(prev =>
					prev.map(bundleData =>
						bundleData.bundle.id === bundleID
							? {
									...bundleData,
									skills: bundleData.skills.map(existingSkill =>
										existingSkill.id === skillID ? { ...existingSkill, isEnabled: nextEnabled } : existingSkill
									),
								}
							: bundleData
					)
				);
			} catch (err) {
				console.error('Toggle skill failed:', err);
				throw err;
			}
		},
		[]
	);

	const handleDeleteSkill = useCallback(async (bundleID: string, skillID: string, skillSlug: string) => {
		try {
			await skillStoreAPI.deleteSkill(bundleID, skillSlug);

			if (!isMountedRef.current) {
				return;
			}

			setBundles(prev =>
				prev.map(bundleData =>
					bundleData.bundle.id === bundleID
						? {
								...bundleData,
								skills: bundleData.skills.filter(existingSkill => existingSkill.id !== skillID),
							}
						: bundleData
				)
			);
		} catch (err) {
			console.error('Delete skill failed:', err);
			throw err;
		}
	}, []);

	const handleSubmitSkill = useCallback(
		async (bundleID: string, partial: Partial<Skill>, existingSkillSlug?: string) => {
			try {
				if (existingSkillSlug) {
					await skillStoreAPI.patchSkill(
						bundleID,
						existingSkillSlug,
						partial.isEnabled,
						partial.location,
						partial.displayName,
						partial.description,
						partial.tags
					);
				} else {
					const slug = (partial.slug ?? '').trim();
					const name = (partial.name ?? '').trim();
					const location = (partial.location ?? '').trim();

					if (!slug) {
						throw new Error('Missing skill slug.');
					}

					if (!name) {
						throw new Error('Missing skill name.');
					}

					if (partial.type === undefined) {
						throw new Error('Missing skill type.');
					}

					if (!location) {
						throw new Error('Missing skill location.');
					}

					await skillStoreAPI.putSkill(
						bundleID,
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

				await refreshBundleSkills(bundleID);
			} catch (err) {
				console.error(existingSkillSlug ? 'Edit skill failed:' : 'Add skill failed:', err);
				throw err;
			}
		},
		[refreshBundleSkills]
	);

	const handleBundleDelete = useCallback(async () => {
		const deletingBundle = bundleToDelete;

		if (!deletingBundle || isDeletingBundle) {
			return;
		}

		setIsDeletingBundle(true);

		try {
			await skillStoreAPI.deleteSkillBundle(deletingBundle.id);

			if (!isMountedRef.current) {
				return;
			}

			setBundles(prev => prev.filter(bundleData => bundleData.bundle.id !== deletingBundle.id));
		} catch (err) {
			console.error('Delete skill bundle failed:', err);

			if (isMountedRef.current) {
				setAlertMsg('Failed to delete skill bundle.');
				setShowAlert(true);
			}
		} finally {
			if (isMountedRef.current) {
				setIsDeletingBundle(false);
				setBundleToDelete(null);
			}
		}
	}, [bundleToDelete, isDeletingBundle]);

	const handleAddBundle = useCallback(
		async (slug: string, display: string, description?: string) => {
			try {
				const id = getUUIDv7();
				await skillStoreAPI.putSkillBundle(id, slug, display, true, description);

				if (isMountedRef.current) {
					setIsAddModalOpen(false);
				}

				await fetchAll();
			} catch (err) {
				console.error('Add skill bundle failed:', err);

				if (isMountedRef.current) {
					setAlertMsg('Failed to add skill bundle.');
					setShowAlert(true);
				}
			}
		},
		[fetchAll]
	);

	if (loading) {
		return <Loader text="Loading skill bundles…" />;
	}

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

						{bundles.map(bundleData => (
							<SkillBundleCard
								key={bundleData.bundle.id}
								bundle={bundleData.bundle}
								skills={bundleData.skills}
								onToggleBundleEnable={handleBundleEnableChange}
								onToggleSkillEnable={handleSkillEnableChange}
								onDeleteSkill={handleDeleteSkill}
								onSubmitSkill={handleSubmitSkill}
								onRequestBundleDelete={bundle => {
									setBundleToDelete(bundle);
								}}
							/>
						))}
					</div>
				</div>

				<DeleteConfirmationModal
					isOpen={bundleToDelete !== null}
					onClose={() => {
						if (!isDeletingBundle) {
							setBundleToDelete(null);
						}
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
