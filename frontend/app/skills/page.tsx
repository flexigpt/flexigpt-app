import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { FiPlus, FiSearch, FiTag, FiX } from 'react-icons/fi';

import type { SkillBundle } from '@/spec/skill';

import { getUUIDv7 } from '@/lib/uuid_utils';

import { skillStoreAPI } from '@/apis/baseapi';
import { getAllSkillBundles, getAllSkills } from '@/apis/list_helper';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';
import { Loader } from '@/components/loader';
import { PageFrame } from '@/components/page_frame';

import type { SkillInsertFilter } from '@/skills/lib/skill_artifact_utils';
import {
	getAllSkillTags,
	getSkillInsertCounts,
	getSkillInsertDescription,
	skillMatchesInsertFilter,
	skillMatchesSearch,
	skillMatchesTags,
} from '@/skills/lib/skill_artifact_utils';
import type { BundleData } from '@/skills/lib/skill_bundle_utils';
import { sortBundleData } from '@/skills/lib/skill_bundle_utils';
import type { SkillUpsertInput } from '@/skills/skill_add_edit_modal';
import { AddSkillBundleModal } from '@/skills/skill_bundle_add_modal';
import { SkillBundleCard } from '@/skills/skill_bundle_card';

// oxlint-disable-next-line no-restricted-exports
export default function SkillsPage() {
	const [bundles, setBundles] = useState<BundleData[]>([]);
	const [loading, setLoading] = useState(true);
	const [insertFilter, setInsertFilter] = useState<SkillInsertFilter>('all');
	const [searchQuery, setSearchQuery] = useState('');
	const [tagFilterInput, setTagFilterInput] = useState('');

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
	const allSkills = useMemo(() => bundles.flatMap(bundleData => bundleData.skills), [bundles]);
	const insertCounts = useMemo(() => getSkillInsertCounts(allSkills), [allSkills]);
	const allTags = useMemo(() => getAllSkillTags(allSkills), [allSkills]);
	const activeTagFilters = useMemo(
		() =>
			tagFilterInput
				.split(',')
				.map(tag => tag.trim())
				.filter(Boolean),
		[tagFilterInput]
	);
	const visibleSkillCount = useMemo(
		() =>
			allSkills.filter(
				skill =>
					skillMatchesInsertFilter(skill.insert, insertFilter) &&
					skillMatchesSearch(skill, searchQuery) &&
					skillMatchesTags(skill, activeTagFilters)
			).length,
		[activeTagFilters, allSkills, insertFilter, searchQuery]
	);

	const skillFilterOptions = useMemo(
		() => [
			{
				value: 'all' as const,
				label: 'All skills',
				count: allSkills.length,
				description: 'Show every skill record in every bundle.',
			},
			{
				value: 'instructions' as const,
				label: 'Instruction skills',
				count: insertCounts.instructions,
				description: getSkillInsertDescription('instructions'),
			},
			{
				value: 'user-message' as const,
				label: 'User-message templates',
				count: insertCounts['user-message'],
				description: getSkillInsertDescription('user-message'),
			},
		],
		[allSkills.length, insertCounts]
	);

	const fetchAll = useCallback(async () => {
		const requestId = (fetchRequestIdRef.current += 1);

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
			setAlertMsg(err instanceof Error ? err.message : 'Failed to load skill bundles. Please try again.');
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
		// oxlint-disable-next-line jsreact-hooks/set-state-in-effect
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
		async (bundleID: string, partial: SkillUpsertInput, existingSkillSlug?: string) => {
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
				} else if (partial.artifactCreate) {
					const slug = (partial.slug ?? '').trim();
					const create = partial.artifactCreate;

					if (!slug) {
						throw new Error('Missing skill slug.');
					}

					const _ = await skillStoreAPI.putSkillArtifact(bundleID, slug, {
						name: create.name,
						displayName: create.displayName,
						description: create.description,
						insert: create.insert,
						arguments: create.arguments,
						tags: create.tags,
						markdownBody: create.markdownBody,
						isEnabled: create.isEnabled,
					});
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
				setAlertMsg(err instanceof Error ? err.message : 'Failed to delete skill bundle.');
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
					setAlertMsg(err instanceof Error ? err.message : 'Failed to add skill bundle.');
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
			<div className="flex size-full flex-col items-center overflow-hidden">
				<div className="bg-base-200/95 sticky top-0 z-10 mt-4 flex w-11/12 items-center gap-4 px-4 py-2 backdrop-blur-sm xl:w-2/3">
					<div className="min-w-0 grow">
						<h1 className="text-xl font-semibold">Skill and Template Bundles</h1>
						<p className="text-base-content/70 text-xs">
							Manage Agent Skills artifacts. New prompt templates are represented here as skills with{' '}
							<span className="font-mono">insert: user-message</span>.
						</p>
					</div>
					<button
						type="button"
						className="btn btn-ghost flex shrink-0 items-center rounded-2xl"
						onClick={() => {
							setIsAddModalOpen(true);
						}}
					>
						<FiPlus size={20} /> <span className="ml-1">Add Bundle</span>
					</button>
				</div>

				<div className="app-scrollbar-thin flex w-full grow flex-col items-center overflow-y-auto">
					<div className="mt-4 w-11/12 space-y-4 xl:w-2/3">
						<div className="alert alert-info rounded-2xl text-sm">
							<div className="space-y-1">
								<div className="font-semibold">How skills and prompt-like templates work</div>
								<div>
									A skill artifact is a directory with a required <span className="font-mono">SKILL.md</span>. FlexiGPT
									gives special meaning only to <span className="font-mono">name</span>,{' '}
									<span className="font-mono">description</span>, <span className="font-mono">insert</span>, and{' '}
									<span className="font-mono">arguments</span>.
								</div>
								<div>
									<span className="font-semibold">Instruction skills</span> are standing context and can be activated in
									a conversation session. <span className="font-semibold">User-message skills</span> are render-only
									prompt templates for the composer.
								</div>
								<div>
									Managed creation writes only <span className="font-mono">SKILL.md</span>. Add extra files such as{' '}
									<span className="font-mono">references/</span>, <span className="font-mono">assets/</span>, or{' '}
									<span className="font-mono">scripts/</span> inside the generated skill folder after saving. Re-enable
									the skill or restart the app to force runtime metadata to refresh.
								</div>
								<div>
									The old Prompt Bundles page remains available for compatibility. From a management perspective, new
									instruction snippets and user-message templates should be created here.
								</div>
							</div>
						</div>

						<div className="border-base-300 bg-base-100 grid grid-cols-1 gap-3 rounded-2xl border p-3 lg:grid-cols-12">
							<label className="input input-sm flex items-center gap-2 rounded-xl lg:col-span-5">
								<FiSearch size={14} />
								<input
									type="search"
									className="grow"
									value={searchQuery}
									onChange={e => {
										setSearchQuery(e.target.value);
									}}
									placeholder="Search name, slug, description, location, tags, arguments…"
									spellCheck="false"
								/>
								{searchQuery ? (
									<button
										type="button"
										className="btn btn-ghost btn-xs rounded-lg"
										onClick={() => {
											setSearchQuery('');
										}}
										aria-label="Clear search"
									>
										<FiX size={12} />
									</button>
								) : null}
							</label>

							<label className="input input-sm flex items-center gap-2 rounded-xl border lg:col-span-4">
								<FiTag size={14} />
								<input
									type="text"
									className="grow"
									value={tagFilterInput}
									onChange={e => {
										setTagFilterInput(e.target.value);
									}}
									placeholder="Filter tags, comma separated"
									spellCheck="false"
								/>
								{tagFilterInput ? (
									<button
										type="button"
										className="btn btn-ghost btn-xs rounded-lg"
										onClick={() => {
											setTagFilterInput('');
										}}
										aria-label="Clear tag filter"
									>
										<FiX size={12} />
									</button>
								) : null}
							</label>

							<div className="text-base-content/70 flex items-center justify-end text-xs lg:col-span-3">
								{visibleSkillCount} matching skill{visibleSkillCount === 1 ? '' : 's'} across {bundles.length} bundle
								{bundles.length === 1 ? '' : 's'}
							</div>

							<div className="flex flex-wrap items-center gap-2 lg:col-span-12">
								{skillFilterOptions.map(option => {
									const isActive = insertFilter === option.value;

									return (
										<button
											key={option.value}
											type="button"
											className={`btn btn-sm rounded-xl ${isActive ? 'btn-primary' : 'btn-ghost'}`}
											onClick={() => {
												setInsertFilter(option.value);
											}}
											title={option.description}
										>
											<span>{option.label}</span>
											<span className={`badge badge-sm ${isActive ? 'badge-neutral' : 'badge-outline'}`}>
												{option.count}
											</span>
										</button>
									);
								})}
							</div>

							{allTags.length > 0 && (
								<div className="flex flex-wrap items-center gap-1 lg:col-span-12">
									<span className="text-base-content/60 mr-1 text-xs">Known tags:</span>
									{allTags.slice(0, 32).map(tag => (
										<button
											key={tag}
											type="button"
											className="badge badge-outline rounded-xl"
											onClick={() => {
												setTagFilterInput(current => {
													const existing = current
														.split(',')
														.map(item => item.trim())
														.filter(Boolean);
													return existing.includes(tag) ? current : [...existing, tag].join(', ');
												});
											}}
										>
											{tag}
										</button>
									))}
								</div>
							)}
						</div>

						<div className="flex flex-col space-y-4 pb-8">
							{bundles.length === 0 && <p className="mt-8 text-center text-sm">No skill bundles configured yet.</p>}

							{bundles.map(bundleData => (
								<SkillBundleCard
									key={bundleData.bundle.id}
									bundle={bundleData.bundle}
									skills={bundleData.skills}
									insertFilter={insertFilter}
									searchQuery={searchQuery}
									tagFilters={activeTagFilters}
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
					message={
						bundleToDelete
							? `Delete empty bundle "${bundleToDelete.displayName || bundleToDelete.slug}"? Remove all skills from the bundle first.`
							: 'Delete this empty skill bundle?'
					}
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
