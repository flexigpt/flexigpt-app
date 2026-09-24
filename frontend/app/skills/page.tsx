import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { FiPlus, FiSearch, FiTag, FiX } from 'react-icons/fi';

import type { SkillBundle } from '@/spec/skill';
import { SkillInsert } from '@/spec/skill';

import { throwIfAborted } from '@/lib/async_utils';
import { getErrorMessage } from '@/lib/error_utils';
import { getUUIDv7 } from '@/lib/uuid_utils';

import { useAsyncResource } from '@/hooks/use_async_resource';

import { skillManagementAPI } from '@/apis/baseapi';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';
import { Loader } from '@/components/loader';
import { ManagementBundleCreateModal } from '@/components/managementui/management_bundle_create_modal';
import { ManagementPageContent } from '@/components/managementui/management_page_content';
import { ManagementPageHeader } from '@/components/managementui/management_page_header';
import { ManagementResourceError } from '@/components/managementui/management_resource_error';
import { PageFrame } from '@/components/page_frame';

import type { SkillInsertFilter } from '@/skills/lib/skill_artifact_utils';
import type { BundleData } from '@/skills/lib/skill_bundle_utils';
import type { SkillItem, SkillUpsertInput } from '@/skills/skill_add_edit_modal';
import {
	getAllSkillTags,
	getSkillInsertCounts,
	getSkillInsertDescription,
	skillMatchesInsertFilter,
	skillMatchesSearch,
	skillMatchesTags,
} from '@/skills/lib/skill_artifact_utils';
import { sortBundleData } from '@/skills/lib/skill_bundle_utils';
import { SkillBundleCard } from '@/skills/skill_bundle_card';

const SKILL_BUNDLE_DATA_CACHE_TTL_MS = 5 * 60 * 1000;

interface SkillBundleDataCache {
	data: BundleData[];
	loadedAt: number;
}

interface SkillBundleDataLoad {
	generation: number;
	promise: Promise<BundleData[]>;
}

interface SkillRuntimeMetadataLoad {
	generation: number;
	promise: Promise<BundleData[]>;
}

let skillBundleDataCache: SkillBundleDataCache | undefined;
let skillBundleDataLoad: SkillBundleDataLoad | undefined;
let skillBundleDataCacheGeneration = 0;
let skillRuntimeMetadataLoad: SkillRuntimeMetadataLoad | undefined;
let skillRuntimeMetadataGeneration = 0;

function invalidateSkillBundleDataCache() {
	skillBundleDataCacheGeneration += 1;
	skillRuntimeMetadataGeneration += 1;
	skillBundleDataCache = undefined;
	skillRuntimeMetadataLoad = undefined;
}

function rememberSkillBundleData(data: BundleData[]) {
	// Invalidate any older request which is still completing after a local
	// mutation changed the page state.
	skillBundleDataCacheGeneration += 1;
	skillBundleDataCache = {
		data,
		loadedAt: Date.now(),
	};
}

function buildSkillBundleData(
	skillBundles: SkillBundle[],
	skillListItems: Awaited<ReturnType<typeof skillManagementAPI.listSkills>>,
	runtimeMetadataLoaded: boolean
): BundleData[] {
	const skillsByBundleID = new Map<string, BundleData['skills']>();

	for (const bundle of skillBundles) {
		skillsByBundleID.set(bundle.id, []);
	}
	for (const item of skillListItems) {
		skillsByBundleID.get(item.bundleID)?.push(item.skillDefinition);
	}

	return sortBundleData(
		skillBundles.map(bundle => ({
			bundle,
			skills: skillsByBundleID.get(bundle.id) ?? [],
			runtimeMetadataLoaded,
		}))
	);
}

/**
 * This route intentionally uses only installed Skill Store APIs. Workspace
 * Skills are Workspace Artifacts and are shown only in Workspace management
 * and in the conversation Workspace selector.
 */
async function fetchSkillBundleData(): Promise<BundleData[]> {
	// Load durable collection and Artifact state first. Runtime materialization
	// is enriched per bundle after the page has rendered.
	const { skillBundles, skillListItems } = await skillManagementAPI.loadManagementPageData(true, false);
	if (skillBundles.length === 0) {
		return [];
	}

	return buildSkillBundleData(skillBundles, skillListItems, false);
}

function loadSkillRuntimeMetadata(): SkillRuntimeMetadataLoad {
	const generation = skillRuntimeMetadataGeneration;
	let load = skillRuntimeMetadataLoad;

	if (!load || load.generation !== generation) {
		const promise = skillManagementAPI
			.loadManagementPageData(true, true)
			.then(({ skillBundles, skillListItems }) => buildSkillBundleData(skillBundles, skillListItems, true));

		load = {
			generation,
			promise,
		};
		skillRuntimeMetadataLoad = load;

		const clear = () => {
			if (skillRuntimeMetadataLoad?.promise === promise) {
				skillRuntimeMetadataLoad = undefined;
			}
		};
		void promise.then(clear, clear);
	}

	return load;
}

async function loadSkillBundleData(signal: AbortSignal): Promise<BundleData[]> {
	throwIfAborted(signal);

	const cached = skillBundleDataCache;
	if (cached && Date.now() - cached.loadedAt <= SKILL_BUNDLE_DATA_CACHE_TTL_MS) {
		return cached.data;
	}

	const generation = skillBundleDataCacheGeneration;
	let load = skillBundleDataLoad;
	if (!load || load.generation !== generation) {
		const promise = fetchSkillBundleData().then(data => {
			if (skillBundleDataCacheGeneration === generation) {
				skillBundleDataCache = {
					data,
					loadedAt: Date.now(),
				};
			}
			return data;
		});

		load = { generation, promise };
		skillBundleDataLoad = load;

		const clearLoad = () => {
			if (skillBundleDataLoad?.promise === promise) {
				skillBundleDataLoad = undefined;
			}
		};
		void promise.then(clearLoad, clearLoad);
	}

	// Wails calls currently cannot be cancelled because their wrappers use
	// context.Background(). Keep the shared operation alive so a remounted
	// page can reuse it, but do not update an already-aborted consumer.
	const data = await load.promise;
	throwIfAborted(signal);
	return data;
}

// oxlint-disable-next-line no-restricted-exports
export default function SkillsPage() {
	const loadPageData = useCallback((signal: AbortSignal) => loadSkillBundleData(signal), []);
	const {
		data: bundles,
		error: pageLoadError,
		isLoading,
		isRefreshing,
		hasResolved,
		reloadOrThrow,
		setData: setBundles,
	} = useAsyncResource(loadPageData, {
		initialData: skillBundleDataCache?.data ?? ([] as BundleData[]),
	});

	const reloadPageData = useCallback(async () => {
		invalidateSkillBundleDataCache();
		await reloadOrThrow();
	}, [reloadOrThrow]);

	const [insertFilter, setInsertFilter] = useState<SkillInsertFilter>('all');
	const [searchQuery, setSearchQuery] = useState('');
	const [tagFilterInput, setTagFilterInput] = useState('');

	const [showAlert, setShowAlert] = useState(false);
	const [alertMsg, setAlertMsg] = useState('');

	const [bundleToDelete, setBundleToDelete] = useState<SkillBundle | null>(null);
	const [isDeletingBundle, setIsDeletingBundle] = useState(false);
	const [isAddModalOpen, setIsAddModalOpen] = useState(false);
	const [creationRootID, setCreationRootID] = useState('');

	const isMountedRef = useRef(false);
	const bundleRefreshRequestIdRef = useRef<Record<string, number>>({});
	const runtimeMetadataRequestIDRef = useRef(0);

	const creationRoots = useMemo(
		() =>
			[
				...new Map(
					bundles
						.filter(bundleData => bundleData.bundle.isBaseline && !bundleData.bundle.isBuiltIn)
						.map(
							bundleData =>
								[
									bundleData.bundle.rootID,
									{
										rootID: bundleData.bundle.rootID,
										label: bundleData.bundle.displayName || bundleData.bundle.slug,
									},
								] as const
						)
				).values(),
			].toSorted((left, right) => left.label.localeCompare(right.label)),
		[bundles]
	);
	const effectiveCreationRootID = creationRoots.some(value => value.rootID === creationRootID)
		? creationRootID
		: (creationRoots[0]?.rootID ?? '');

	const existingBundleSlugs = useMemo(
		() =>
			bundles
				.filter(bundleData => bundleData.bundle.rootID === effectiveCreationRootID)
				.map(bundleData => bundleData.bundle.slug),
		[bundles, effectiveCreationRootID]
	);
	const existingBundleNames = useMemo(
		() =>
			bundles
				.filter(bundleData => bundleData.bundle.rootID === effectiveCreationRootID)
				.map(bundleData => (bundleData.bundle.displayName ?? bundleData.bundle.slug).trim()),
		[bundles, effectiveCreationRootID]
	);
	const allSkills = useMemo(() => bundles.flatMap(bundleData => bundleData.skills), [bundles]);
	const allSkillItems = useMemo<SkillItem[]>(
		() =>
			bundles.flatMap(bundleData =>
				bundleData.skills.map(skill => ({
					skill,
					bundleID: bundleData.bundle.id,
					skillSlug: skill.slug,
				}))
			),
		[bundles]
	);
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
				description: 'Show every skill in every Collection.',
			},
			{
				value: SkillInsert.Instructions,
				label: 'Instruction skills',
				count: insertCounts.instructions,
				description: getSkillInsertDescription('instructions'),
			},
			{
				value: SkillInsert.UserMessage,
				label: 'User-message templates',
				count: insertCounts['user-message'],
				description: getSkillInsertDescription('user-message'),
			},
		],
		[allSkills.length, insertCounts]
	);

	const refreshBundleSkills = useCallback(
		async (bundleID: string, refreshSource = true) => {
			const requestId = (bundleRefreshRequestIdRef.current[bundleID] ?? 0) + 1;
			bundleRefreshRequestIdRef.current[bundleID] = requestId;

			try {
				if (refreshSource) {
					await skillManagementAPI.refreshSkillBundle(bundleID);
				}
				const skillListItems = await skillManagementAPI.listSkills([bundleID], true, true);
				const freshSkills = skillListItems.map(item => item.skillDefinition);

				if (!isMountedRef.current || bundleRefreshRequestIdRef.current[bundleID] !== requestId) {
					return;
				}

				setBundles(prev =>
					prev.map(bundleData =>
						bundleData.bundle.id === bundleID
							? {
									...bundleData,
									skills: freshSkills,
									runtimeMetadataLoaded: true,
									skillLoadError: undefined,
								}
							: bundleData
					)
				);
			} catch (err) {
				console.error('Refresh bundle skills failed:', err);
				const message = getErrorMessage(err, 'Failed to load this Collection’s skills.');

				if (isMountedRef.current && bundleRefreshRequestIdRef.current[bundleID] === requestId) {
					setBundles(previous =>
						previous.map(bundleData =>
							bundleData.bundle.id === bundleID ? { ...bundleData, skillLoadError: message } : bundleData
						)
					);
				}

				throw err;
			}
		},
		[setBundles]
	);

	useEffect(() => {
		isMountedRef.current = true;
		return () => {
			isMountedRef.current = false;
			runtimeMetadataRequestIDRef.current += 1;
		};
	}, []);

	useEffect(() => {
		if (hasResolved && !pageLoadError && !isLoading && !isRefreshing) {
			rememberSkillBundleData(bundles);
		}
	}, [bundles, hasResolved, isLoading, isRefreshing, pageLoadError]);

	useEffect(() => {
		if (!hasResolved || pageLoadError || bundles.every(bundle => bundle.runtimeMetadataLoaded)) {
			return;
		}

		const requestID = runtimeMetadataRequestIDRef.current + 1;
		runtimeMetadataRequestIDRef.current = requestID;
		const load = loadSkillRuntimeMetadata();

		void load.promise
			.then(hydrated => {
				if (
					!isMountedRef.current ||
					runtimeMetadataRequestIDRef.current !== requestID ||
					skillRuntimeMetadataGeneration !== load.generation
				) {
					return;
				}

				const byBundleID = new Map(hydrated.map(bundle => [bundle.bundle.id, bundle] as const));

				setBundles(current =>
					current.map(bundle => {
						const next = byBundleID.get(bundle.bundle.id);
						if (!next) {
							return bundle;
						}

						const nextSkillsByID = new Map(next.skills.map(skill => [skill.id, skill] as const));
						const skills = bundle.skills.map(skill => {
							const hydratedSkill = nextSkillsByID.get(skill.id);
							if (!hydratedSkill || hydratedSkill.revision < skill.revision) {
								return skill;
							}
							return hydratedSkill;
						});

						return {
							...bundle,
							bundle: next.bundle.revision >= bundle.bundle.revision ? next.bundle : bundle.bundle,
							skills,
							runtimeMetadataLoaded: skills.length === next.skills.length,
						};
					})
				);
			})
			.catch((error: unknown) => {
				if (!isMountedRef.current) {
					return;
				}
				setAlertMsg(getErrorMessage(error, 'Skill runtime details could not be loaded.'));
				setShowAlert(true);
			});
	}, [bundles, hasResolved, pageLoadError, setBundles]);

	const handleBundleEnableChange = useCallback(
		async (bundleID: string, nextEnabled: boolean) => {
			try {
				await skillManagementAPI.patchSkillBundle(bundleID, nextEnabled);

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
		},
		[setBundles]
	);

	const handleSkillEnableChange = useCallback(
		async (bundleID: string, skillID: string, skillSlug: string, nextEnabled: boolean) => {
			const bundleData = bundles.find(item => item.bundle.id === bundleID);
			if (!bundleData) {
				throw new Error('Skill bundle not found.');
			}
			if (!bundleData.bundle.isEnabled) {
				throw new Error('Enable the skill bundle before changing a skill.');
			}
			if (!bundleData.skills.some(skill => skill.id === skillID && skill.slug === skillSlug)) {
				throw new Error('Skill not found.');
			}

			try {
				await skillManagementAPI.patchSkill(bundleID, skillID, nextEnabled);

				if (!isMountedRef.current) {
					return;
				}

				setBundles(prev =>
					prev.map(b =>
						b.bundle.id === bundleID
							? {
									...b,
									skills: b.skills.map(existingSkill =>
										existingSkill.id === skillID ? { ...existingSkill, isEnabled: nextEnabled } : existingSkill
									),
								}
							: b
					)
				);
			} catch (err) {
				console.error('Toggle skill failed:', err);
				throw err;
			}
		},
		[bundles, setBundles]
	);

	const handleDeleteSkill = useCallback(
		async (bundleID: string, skillID: string, skillSlug: string) => {
			const bundleData = bundles.find(item => item.bundle.id === bundleID);
			if (!bundleData) {
				throw new Error('Skill bundle not found.');
			}
			if (!bundleData.bundle.isEditable) {
				throw new Error('This Skill Bundle only supports enable and disable actions.');
			}
			const skill = bundleData.skills.find(item => item.id === skillID && item.slug === skillSlug);
			if (!skill) {
				throw new Error('Skill not found.');
			}
			if (skill.isBuiltIn) {
				throw new Error('Built-in skills cannot be deleted.');
			}

			try {
				await skillManagementAPI.deleteSkill(bundleID, skillID);

				if (!isMountedRef.current) {
					return;
				}

				setBundles(prev =>
					prev.map(b =>
						b.bundle.id === bundleID
							? {
									...b,
									skills: b.skills.filter(existingSkill => existingSkill.id !== skillID),
								}
							: b
					)
				);
			} catch (err) {
				console.error('Delete skill failed:', err);
				throw err;
			}
		},
		[bundles, setBundles]
	);

	const handleSubmitSkill = useCallback(
		async (bundleID: string, partial: SkillUpsertInput, existingSkillID?: string) => {
			const bundleData = bundles.find(item => item.bundle.id === bundleID);
			if (!bundleData) {
				throw new Error('Skill bundle not found.');
			}
			if (bundleData.bundle.isBuiltIn) {
				throw new Error('Cannot add or edit skills in a built-in bundle.');
			}
			if (!bundleData.bundle.isEnabled) {
				throw new Error('Enable the skill bundle before adding or editing skills.');
			}

			if (existingSkillID) {
				if (!bundleData.bundle.isEditable) {
					throw new Error('This Skill Bundle only supports enable and disable actions.');
				}

				const existingSkill = bundleData.skills.find(skill => skill.id === existingSkillID);
				if (!existingSkill) {
					throw new Error('Skill not found.');
				}
				if (!existingSkill.isManaged) {
					throw new Error('Only managed Skills can be edited. Fork this Skill to create a managed copy.');
				}
				if (!partial.artifactCreate) {
					throw new Error('The managed Skill document was not loaded. Reload the Skill before editing it.');
				}
			}

			try {
				if (existingSkillID && partial.artifactCreate) {
					await skillManagementAPI.replaceManagedSkill(bundleID, existingSkillID, partial.artifactCreate);
				} else if (partial.artifactCreate) {
					const slug = (partial.name ?? partial.slug ?? '').trim();
					const create = partial.artifactCreate;

					if (!slug) {
						throw new Error('Missing skill slug.');
					}

					await skillManagementAPI.putSkillArtifact(bundleID, partial.artifactID ?? getUUIDv7(), {
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
					const location = (partial.location ?? '').trim();
					const sourceDisplayName = (partial.displayName ?? '').trim();

					if (!location) {
						throw new Error('Missing skill location.');
					}
					if (!sourceDisplayName) {
						throw new Error('Missing source display name.');
					}

					await skillManagementAPI.registerFilesystemSkills(bundleID, location, sourceDisplayName);
				}
			} catch (err) {
				console.error(existingSkillID ? 'Edit skill failed:' : 'Add skill failed:', err);
				throw err;
			}

			try {
				await refreshBundleSkills(bundleID, false);
			} catch (err) {
				console.error('Skill was saved but bundle refresh failed:', err);
				if (isMountedRef.current) {
					setAlertMsg(
						'The skill was saved, but the bundle could not be refreshed. Use Retry on the bundle before making further changes.'
					);
					setShowAlert(true);
				}
			}
		},
		[bundles, refreshBundleSkills]
	);

	const handleBundleDelete = useCallback(async () => {
		const deletingBundle = bundleToDelete;

		if (!deletingBundle || isDeletingBundle) {
			return;
		}

		const bundleData = bundles.find(item => item.bundle.id === deletingBundle.id);
		if (!bundleData?.bundle.isDeletable) {
			setBundleToDelete(null);
			setAlertMsg('This Skill Bundle cannot be deleted.');
			setShowAlert(true);
			return;
		}

		if (!bundleData || bundleData.skillLoadError || bundleData.skills.length > 0) {
			setBundleToDelete(null);
			setAlertMsg(
				bundleData?.skillLoadError
					? 'Reload this bundle’s skills before deleting it.'
					: 'Remove all skills from this bundle before deleting it.'
			);
			setShowAlert(true);
			return;
		}

		setIsDeletingBundle(true);

		try {
			await skillManagementAPI.deleteSkillBundle(deletingBundle.id);

			if (!isMountedRef.current) {
				return;
			}

			setBundles(prev => prev.filter(b => b.bundle.id !== deletingBundle.id));
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
	}, [bundleToDelete, bundles, isDeletingBundle, setBundles]);

	const handleAddBundle = useCallback(
		async (slug: string, display: string, description?: string) => {
			try {
				const id = getUUIDv7();
				await skillManagementAPI.putSkillBundle(id, effectiveCreationRootID, slug, display, true, description);
				try {
					await reloadPageData();
				} catch (refreshError) {
					console.error('Skill bundle was created but refresh failed:', refreshError);
					if (isMountedRef.current) {
						setAlertMsg(
							'Skill Collection was created, but the page could not be refreshed. Reload before making destructive changes.'
						);
						setShowAlert(true);
					}
				}
			} catch (err) {
				console.error('Add skill bundle failed:', err);
				throw err;
			}
		},
		[effectiveCreationRootID, reloadPageData]
	);

	const handleEditBundle = useCallback(
		async (bundleID: string, displayName: string, description?: string) => {
			await skillManagementAPI.updateSkillBundleMetadata(bundleID, displayName, description);

			if (!isMountedRef.current) {
				return;
			}

			setBundles(previous =>
				previous.map(item =>
					item.bundle.id === bundleID
						? {
								...item,
								bundle: { ...item.bundle, displayName, description },
							}
						: item
				)
			);
		},
		[setBundles]
	);

	if (isLoading && !hasResolved && bundles.length === 0) {
		return <Loader text="Loading Skill Collections..." />;
	}

	return (
		<PageFrame>
			<div className="flex size-full flex-col items-center overflow-hidden">
				<ManagementPageHeader
					title="Skills and Templates"
					description="Manage instruction skills, user-message templates, arguments, resources, and runtime presence."
					actions={
						<>
							{creationRoots.length > 1 ? (
								<select
									className="select select-sm max-w-72 rounded-xl"
									aria-label="Skill Collection group"
									value={effectiveCreationRootID}
									onChange={event => {
										setCreationRootID(event.currentTarget.value);
									}}
								>
									{creationRoots.map(value => (
										<option key={value.rootID} value={value.rootID}>
											{value.label}
										</option>
									))}
								</select>
							) : null}

							<button
								type="button"
								className="btn btn-ghost rounded-xl"
								onClick={() => {
									setIsAddModalOpen(true);
								}}
							>
								<FiPlus size={18} />
								<span>Add Collection</span>
							</button>
						</>
					}
				/>

				<ManagementPageContent>
					{pageLoadError ? (
						<ManagementResourceError
							title="Skill bundles could not be loaded"
							error={pageLoadError}
							isRetrying={isRefreshing}
							onRetry={async () => {
								await reloadPageData();
							}}
						/>
					) : null}

					<div className="border-base-content/10 bg-base-100 rounded-2xl border p-4 text-sm">
						<div className="font-semibold">Skill basics</div>
						<ul className="text-base-content/70 mt-2 list-disc space-y-1 pl-5 text-xs">
							<li>Instruction skills provide reusable conversation context.</li>
							<li>User-message skills insert reusable text into the composer.</li>
							<li>Managed skills create `SKILL.md`; extra files can be added to the saved folder.</li>
						</ul>
					</div>

					<div className="grid grid-cols-1 gap-3 md:grid-cols-3">
						<div className="bg-base-100 border-base-300 rounded-2xl border p-3">
							<div className="text-sm font-semibold">Instruction skills</div>
							<div className="text-base-content/70 mt-1 text-xs">{insertCounts.instructions} configured</div>
						</div>
						<div className="bg-base-100 border-base-300 rounded-2xl border p-3">
							<div className="text-sm font-semibold">User-message templates</div>
							<div className="text-base-content/70 mt-1 text-xs">{insertCounts['user-message']} configured</div>
						</div>
						<div className="bg-base-100 border-base-300 rounded-2xl border p-3">
							<div className="text-sm font-semibold">Tags</div>
							<div className="text-base-content/70 mt-1 text-xs">{allTags.length} known tag groups</div>
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
								placeholder="Search names, descriptions, tags, and arguments..."
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
							{visibleSkillCount} matching skill{visibleSkillCount === 1 ? '' : 's'} across {bundles.length} Collection
							{bundles.length === 1 ? '' : 's'}
						</div>

						<div className="flex flex-wrap items-center gap-2 lg:col-span-12">
							{skillFilterOptions.map(option => {
								const isActive = insertFilter === option.value;

								return (
									<button
										key={option.value}
										type="button"
										className={`btn btn-sm rounded-xl border ${
											isActive ? 'border-base-content/30 bg-base-200' : 'border-transparent bg-transparent'
										}`}
										onClick={() => {
											setInsertFilter(option.value);
										}}
										title={option.description}
										aria-pressed={isActive}
									>
										<span>{option.label}</span>
										<span className="border-base-content/20 rounded-lg border px-1.5 py-0.5 text-xs">
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
										className="border-base-content/20 hover:bg-base-200 rounded-xl border px-2 py-1 text-xs"
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
						{bundles.length === 0 && <p className="mt-8 text-center text-sm">No Skill Collections configured yet.</p>}

						{bundles.map(bundleData => (
							<SkillBundleCard
								key={bundleData.bundle.id}
								bundle={bundleData.bundle}
								skills={bundleData.skills}
								skillLoadError={bundleData.skillLoadError}
								runtimeMetadataLoaded={Boolean(bundleData.runtimeMetadataLoaded)}
								prefillSkills={allSkillItems}
								onRefreshSkills={() => {
									return refreshBundleSkills(bundleData.bundle.id, !bundleData.bundle.isBuiltIn);
								}}
								insertFilter={insertFilter}
								searchQuery={searchQuery}
								tagFilters={activeTagFilters}
								onToggleBundleEnable={handleBundleEnableChange}
								onToggleSkillEnable={handleSkillEnableChange}
								onDeleteSkill={handleDeleteSkill}
								onSubmitSkill={handleSubmitSkill}
								onEditBundle={handleEditBundle}
								onRequestBundleDelete={bundle => {
									setBundleToDelete(bundle);
								}}
							/>
						))}
					</div>
				</ManagementPageContent>

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
							? `Delete empty Collection "${bundleToDelete.displayName || bundleToDelete.slug}"? Remove all skills from the Collection first.`
							: 'Delete this empty Skill Collection?'
					}
					confirmButtonText="Delete"
				/>

				<ManagementBundleCreateModal
					isOpen={isAddModalOpen}
					title="Add Skill Collection"
					entityLabel="Skill Collection"
					onClose={() => {
						setIsAddModalOpen(false);
					}}
					onSubmit={handleAddBundle}
					existingSlugs={existingBundleSlugs}
					existingDisplayNames={existingBundleNames}
					failureMessage="Failed to create Skill Collection."
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
