import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { FiPlus, FiSearch, FiTag, FiX } from 'react-icons/fi';

import type { SkillPlugin } from '@/spec/skill';
import { SkillInsert } from '@/spec/skill';

import { throwIfAborted } from '@/lib/async_utils';
import { getErrorMessage } from '@/lib/error_utils';
import { getUUIDv7 } from '@/lib/uuid_utils';

import { useAsyncResource } from '@/hooks/use_async_resource';

import { skillManagementAPI } from '@/apis/baseapi';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';
import { Loader } from '@/components/loader';
import { ManagementPageContent } from '@/components/managementui/management_page_content';
import { ManagementPageHeader } from '@/components/managementui/management_page_header';
import { ManagementPluginCreateModal } from '@/components/managementui/management_plugin_create_modal';
import { ManagementResourceError } from '@/components/managementui/management_resource_error';
import { PageFrame } from '@/components/page_frame';

import type { SkillInsertFilter } from '@/skills/lib/skill_artifact_utils';
import type { PluginData } from '@/skills/lib/skill_plugin_utils';
import type { SkillItem, SkillUpsertInput } from '@/skills/skill_add_edit_modal';
import {
	getAllSkillTags,
	getSkillInsertCounts,
	getSkillInsertDescription,
	skillMatchesInsertFilter,
	skillMatchesSearch,
	skillMatchesTags,
} from '@/skills/lib/skill_artifact_utils';
import { sortPluginData } from '@/skills/lib/skill_plugin_utils';
import { SkillPluginCard } from '@/skills/skill_plugin_card';

const SKILL_PLUGIN_DATA_CACHE_TTL_MS = 5 * 60 * 1000;

interface SkillPluginDataCache {
	data: PluginData[];
	loadedAt: number;
}

interface SkillPluginDataLoad {
	generation: number;
	promise: Promise<PluginData[]>;
}

interface SkillRuntimeMetadataLoad {
	generation: number;
	promise: Promise<PluginData[]>;
}

let skillPluginDataCache: SkillPluginDataCache | undefined;
let skillPluginDataLoad: SkillPluginDataLoad | undefined;
let skillPluginDataCacheGeneration = 0;
let skillRuntimeMetadataLoad: SkillRuntimeMetadataLoad | undefined;
let skillRuntimeMetadataGeneration = 0;

function invalidateSkillPluginDataCache() {
	skillPluginDataCacheGeneration += 1;
	skillRuntimeMetadataGeneration += 1;
	skillPluginDataCache = undefined;
	skillRuntimeMetadataLoad = undefined;
}

function rememberSkillPluginData(data: PluginData[]) {
	// Invalidate any older request which is still completing after a local
	// mutation changed the page state.
	skillPluginDataCacheGeneration += 1;
	skillPluginDataCache = {
		data,
		loadedAt: Date.now(),
	};
}

function buildSkillPluginData(
	skillPlugins: SkillPlugin[],
	skillListItems: Awaited<ReturnType<typeof skillManagementAPI.listSkills>>,
	runtimeMetadataLoaded: boolean
): PluginData[] {
	const skillsByPluginID = new Map<string, PluginData['skills']>();

	for (const plugin of skillPlugins) {
		skillsByPluginID.set(plugin.id, []);
	}
	for (const item of skillListItems) {
		skillsByPluginID.get(item.pluginID)?.push(item.skillDefinition);
	}

	return sortPluginData(
		skillPlugins.map(plugin => ({
			plugin,
			skills: skillsByPluginID.get(plugin.id) ?? [],
			runtimeMetadataLoaded,
		}))
	);
}

/**
 * This route intentionally uses only installed Skill Store APIs. Workspace
 * Skills are Workspace Artifacts and are shown only in Workspace management
 * and in the conversation Workspace selector.
 */
async function fetchSkillPluginData(): Promise<PluginData[]> {
	// Load durable plugin and Artifact state first. Runtime materialization
	// is enriched per plugin after the page has rendered.
	const { skillPlugins, skillListItems } = await skillManagementAPI.loadManagementPageData(true, false);
	if (skillPlugins.length === 0) {
		return [];
	}

	return buildSkillPluginData(skillPlugins, skillListItems, false);
}

function loadSkillRuntimeMetadata(): SkillRuntimeMetadataLoad {
	const generation = skillRuntimeMetadataGeneration;
	let load = skillRuntimeMetadataLoad;

	if (!load || load.generation !== generation) {
		const promise = skillManagementAPI
			.loadManagementPageData(true, true)
			.then(({ skillPlugins, skillListItems }) => buildSkillPluginData(skillPlugins, skillListItems, true));

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

async function loadSkillPluginData(signal: AbortSignal): Promise<PluginData[]> {
	throwIfAborted(signal);

	const cached = skillPluginDataCache;
	if (cached && Date.now() - cached.loadedAt <= SKILL_PLUGIN_DATA_CACHE_TTL_MS) {
		return cached.data;
	}

	const generation = skillPluginDataCacheGeneration;
	let load = skillPluginDataLoad;
	if (!load || load.generation !== generation) {
		const promise = fetchSkillPluginData().then(data => {
			if (skillPluginDataCacheGeneration === generation) {
				skillPluginDataCache = {
					data,
					loadedAt: Date.now(),
				};
			}
			return data;
		});

		load = { generation, promise };
		skillPluginDataLoad = load;

		const clearLoad = () => {
			if (skillPluginDataLoad?.promise === promise) {
				skillPluginDataLoad = undefined;
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
	const loadPageData = useCallback((signal: AbortSignal) => loadSkillPluginData(signal), []);
	const {
		data: plugins,
		error: pageLoadError,
		isLoading,
		isRefreshing,
		hasResolved,
		reloadOrThrow,
		setData: setPlugins,
	} = useAsyncResource(loadPageData, {
		initialData: skillPluginDataCache?.data ?? ([] as PluginData[]),
	});

	const reloadPageData = useCallback(async () => {
		invalidateSkillPluginDataCache();
		await reloadOrThrow();
	}, [reloadOrThrow]);

	const [insertFilter, setInsertFilter] = useState<SkillInsertFilter>('all');
	const [searchQuery, setSearchQuery] = useState('');
	const [tagFilterInput, setTagFilterInput] = useState('');

	const [showAlert, setShowAlert] = useState(false);
	const [alertMsg, setAlertMsg] = useState('');

	const [pluginToDelete, setPluginToDelete] = useState<SkillPlugin | null>(null);
	const [isDeletingPlugin, setIsDeletingPlugin] = useState(false);
	const [isAddModalOpen, setIsAddModalOpen] = useState(false);
	const [creationRootID, setCreationRootID] = useState('');

	const isMountedRef = useRef(false);
	const pluginRefreshRequestIdRef = useRef<Record<string, number>>({});
	const runtimeMetadataRequestIDRef = useRef(0);

	const creationRoots = useMemo(
		() =>
			[
				...new Map(
					plugins
						.filter(pluginData => pluginData.plugin.isBaseline && !pluginData.plugin.isBuiltIn)
						.map(
							pluginData =>
								[
									pluginData.plugin.rootID,
									{
										rootID: pluginData.plugin.rootID,
										label: pluginData.plugin.displayName || pluginData.plugin.slug,
									},
								] as const
						)
				).values(),
			].toSorted((left, right) => left.label.localeCompare(right.label)),
		[plugins]
	);
	const effectiveCreationRootID = creationRoots.some(value => value.rootID === creationRootID)
		? creationRootID
		: (creationRoots[0]?.rootID ?? '');

	const existingPluginSlugs = useMemo(
		() =>
			plugins
				.filter(pluginData => pluginData.plugin.rootID === effectiveCreationRootID)
				.map(pluginData => pluginData.plugin.slug),
		[plugins, effectiveCreationRootID]
	);
	const existingPluginNames = useMemo(
		() =>
			plugins
				.filter(pluginData => pluginData.plugin.rootID === effectiveCreationRootID)
				.map(pluginData => (pluginData.plugin.displayName ?? pluginData.plugin.slug).trim()),
		[plugins, effectiveCreationRootID]
	);
	const allSkills = useMemo(() => plugins.flatMap(pluginData => pluginData.skills), [plugins]);
	const allSkillItems = useMemo<SkillItem[]>(
		() =>
			plugins.flatMap(pluginData =>
				pluginData.skills.map(skill => ({
					skill,
					pluginID: pluginData.plugin.id,
					skillSlug: skill.slug,
				}))
			),
		[plugins]
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
				description: 'Show every skill in every Plugin.',
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

	const refreshPluginSkills = useCallback(
		async (pluginID: string, refreshSource = true) => {
			const requestId = (pluginRefreshRequestIdRef.current[pluginID] ?? 0) + 1;
			pluginRefreshRequestIdRef.current[pluginID] = requestId;

			try {
				if (refreshSource) {
					await skillManagementAPI.refreshSkillPlugin(pluginID);
				}
				const skillListItems = await skillManagementAPI.listSkills([pluginID], true, true);
				const freshSkills = skillListItems.map(item => item.skillDefinition);

				if (!isMountedRef.current || pluginRefreshRequestIdRef.current[pluginID] !== requestId) {
					return;
				}

				setPlugins(prev =>
					prev.map(pluginData =>
						pluginData.plugin.id === pluginID
							? {
									...pluginData,
									skills: freshSkills,
									runtimeMetadataLoaded: true,
									skillLoadError: undefined,
								}
							: pluginData
					)
				);
			} catch (err) {
				console.error('Refresh plugin skills failed:', err);
				const message = getErrorMessage(err, 'Failed to load this Plugin’s skills.');

				if (isMountedRef.current && pluginRefreshRequestIdRef.current[pluginID] === requestId) {
					setPlugins(previous =>
						previous.map(pluginData =>
							pluginData.plugin.id === pluginID ? { ...pluginData, skillLoadError: message } : pluginData
						)
					);
				}

				throw err;
			}
		},
		[setPlugins]
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
			rememberSkillPluginData(plugins);
		}
	}, [plugins, hasResolved, isLoading, isRefreshing, pageLoadError]);

	useEffect(() => {
		if (!hasResolved || pageLoadError || plugins.every(plugin => plugin.runtimeMetadataLoaded)) {
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

				const byPluginID = new Map(hydrated.map(plugin => [plugin.plugin.id, plugin] as const));

				setPlugins(current =>
					current.map(plugin => {
						const next = byPluginID.get(plugin.plugin.id);
						if (!next) {
							return plugin;
						}

						const nextSkillsByID = new Map(next.skills.map(skill => [skill.id, skill] as const));
						const skills = plugin.skills.map(skill => {
							const hydratedSkill = nextSkillsByID.get(skill.id);
							if (!hydratedSkill || hydratedSkill.revision < skill.revision) {
								return skill;
							}
							return hydratedSkill;
						});

						return {
							...plugin,
							plugin: next.plugin.revision >= plugin.plugin.revision ? next.plugin : plugin.plugin,
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
	}, [plugins, hasResolved, pageLoadError, setPlugins]);

	const handlePluginEnableChange = useCallback(
		async (pluginID: string, nextEnabled: boolean) => {
			try {
				await skillManagementAPI.patchSkillPlugin(pluginID, nextEnabled);

				if (!isMountedRef.current) {
					return;
				}

				setPlugins(prev =>
					prev.map(pluginData =>
						pluginData.plugin.id === pluginID
							? {
									...pluginData,
									plugin: { ...pluginData.plugin, isEnabled: nextEnabled },
								}
							: pluginData
					)
				);
			} catch (err) {
				console.error('Toggle skill plugin enable failed:', err);
				throw err;
			}
		},
		[setPlugins]
	);

	const handleSkillEnableChange = useCallback(
		async (pluginID: string, skillID: string, skillSlug: string, nextEnabled: boolean) => {
			const pluginData = plugins.find(item => item.plugin.id === pluginID);
			if (!pluginData) {
				throw new Error('Skill plugin not found.');
			}
			if (!pluginData.plugin.isEnabled) {
				throw new Error('Enable the skill plugin before changing a skill.');
			}
			if (!pluginData.skills.some(skill => skill.id === skillID && skill.slug === skillSlug)) {
				throw new Error('Skill not found.');
			}

			try {
				await skillManagementAPI.patchSkill(pluginID, skillID, nextEnabled);

				if (!isMountedRef.current) {
					return;
				}

				setPlugins(prev =>
					prev.map(b =>
						b.plugin.id === pluginID
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
		[plugins, setPlugins]
	);

	const handleDeleteSkill = useCallback(
		async (pluginID: string, skillID: string, skillSlug: string) => {
			const pluginData = plugins.find(item => item.plugin.id === pluginID);
			if (!pluginData) {
				throw new Error('Skill plugin not found.');
			}
			if (!pluginData.plugin.isEditable) {
				throw new Error('This Skill Plugin only supports enable and disable actions.');
			}
			const skill = pluginData.skills.find(item => item.id === skillID && item.slug === skillSlug);
			if (!skill) {
				throw new Error('Skill not found.');
			}
			if (skill.isBuiltIn) {
				throw new Error('Built-in skills cannot be deleted.');
			}

			try {
				await skillManagementAPI.deleteSkill(pluginID, skillID);

				if (!isMountedRef.current) {
					return;
				}

				setPlugins(prev =>
					prev.map(b =>
						b.plugin.id === pluginID
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
		[plugins, setPlugins]
	);

	const handleSubmitSkill = useCallback(
		async (pluginID: string, partial: SkillUpsertInput, existingSkillID?: string) => {
			const pluginData = plugins.find(item => item.plugin.id === pluginID);
			if (!pluginData) {
				throw new Error('Skill plugin not found.');
			}
			if (pluginData.plugin.isBuiltIn) {
				throw new Error('Cannot add or edit skills in a built-in plugin.');
			}
			if (!pluginData.plugin.isEnabled) {
				throw new Error('Enable the skill plugin before adding or editing skills.');
			}

			if (existingSkillID) {
				if (!pluginData.plugin.isEditable) {
					throw new Error('This Skill Plugin only supports enable and disable actions.');
				}

				const existingSkill = pluginData.skills.find(skill => skill.id === existingSkillID);
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
					await skillManagementAPI.replaceManagedSkill(pluginID, existingSkillID, partial.artifactCreate);
				} else if (partial.artifactCreate) {
					const slug = (partial.name ?? partial.slug ?? '').trim();
					const create = partial.artifactCreate;

					if (!slug) {
						throw new Error('Missing skill slug.');
					}

					await skillManagementAPI.putSkillArtifact(pluginID, partial.artifactID ?? getUUIDv7(), {
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

					await skillManagementAPI.registerFilesystemSkills(pluginID, location, sourceDisplayName);
				}
			} catch (err) {
				console.error(existingSkillID ? 'Edit skill failed:' : 'Add skill failed:', err);
				throw err;
			}

			try {
				await refreshPluginSkills(pluginID, false);
			} catch (err) {
				console.error('Skill was saved but plugin refresh failed:', err);
				if (isMountedRef.current) {
					setAlertMsg(
						'The skill was saved, but the plugin could not be refreshed. Use Retry on the plugin before making further changes.'
					);
					setShowAlert(true);
				}
			}
		},
		[plugins, refreshPluginSkills]
	);

	const handlePluginDelete = useCallback(async () => {
		const deletingPlugin = pluginToDelete;

		if (!deletingPlugin || isDeletingPlugin) {
			return;
		}

		const pluginData = plugins.find(item => item.plugin.id === deletingPlugin.id);
		if (!pluginData?.plugin.isDeletable) {
			setPluginToDelete(null);
			setAlertMsg('This Skill Plugin cannot be deleted.');
			setShowAlert(true);
			return;
		}

		if (!pluginData || pluginData.skillLoadError || pluginData.skills.length > 0) {
			setPluginToDelete(null);
			setAlertMsg(
				pluginData?.skillLoadError
					? 'Reload this plugin’s skills before deleting it.'
					: 'Remove all skills from this plugin before deleting it.'
			);
			setShowAlert(true);
			return;
		}

		setIsDeletingPlugin(true);

		try {
			await skillManagementAPI.deleteSkillPlugin(deletingPlugin.id);

			if (!isMountedRef.current) {
				return;
			}

			setPlugins(prev => prev.filter(b => b.plugin.id !== deletingPlugin.id));
		} catch (err) {
			console.error('Delete skill plugin failed:', err);

			if (isMountedRef.current) {
				setAlertMsg(err instanceof Error ? err.message : 'Failed to delete skill plugin.');
				setShowAlert(true);
			}
		} finally {
			if (isMountedRef.current) {
				setIsDeletingPlugin(false);
				setPluginToDelete(null);
			}
		}
	}, [pluginToDelete, plugins, isDeletingPlugin, setPlugins]);

	const handleAddPlugin = useCallback(
		async (slug: string, display: string, description?: string) => {
			try {
				const id = getUUIDv7();
				await skillManagementAPI.putSkillPlugin(id, effectiveCreationRootID, slug, display, true, description);
				try {
					await reloadPageData();
				} catch (refreshError) {
					console.error('Skill plugin was created but refresh failed:', refreshError);
					if (isMountedRef.current) {
						setAlertMsg(
							'Skill Plugin was created, but the page could not be refreshed. Reload before making destructive changes.'
						);
						setShowAlert(true);
					}
				}
			} catch (err) {
				console.error('Add skill plugin failed:', err);
				throw err;
			}
		},
		[effectiveCreationRootID, reloadPageData]
	);

	const handleEditPlugin = useCallback(
		async (pluginID: string, displayName: string, description?: string) => {
			await skillManagementAPI.updateSkillPluginMetadata(pluginID, displayName, description);

			if (!isMountedRef.current) {
				return;
			}

			setPlugins(previous =>
				previous.map(item =>
					item.plugin.id === pluginID
						? {
								...item,
								plugin: { ...item.plugin, displayName, description },
							}
						: item
				)
			);
		},
		[setPlugins]
	);

	if (isLoading && !hasResolved && plugins.length === 0) {
		return <Loader text="Loading Skill Plugins..." />;
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
									aria-label="Skill Plugin group"
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
								<span>Add Plugin</span>
							</button>
						</>
					}
				/>

				<ManagementPageContent>
					{pageLoadError ? (
						<ManagementResourceError
							title="Skill plugins could not be loaded"
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
							{visibleSkillCount} matching skill{visibleSkillCount === 1 ? '' : 's'} across {plugins.length} Plugin
							{plugins.length === 1 ? '' : 's'}
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
						{plugins.length === 0 && <p className="mt-8 text-center text-sm">No Skill Plugins configured yet.</p>}

						{plugins.map(pluginData => (
							<SkillPluginCard
								key={pluginData.plugin.id}
								plugin={pluginData.plugin}
								skills={pluginData.skills}
								skillLoadError={pluginData.skillLoadError}
								runtimeMetadataLoaded={Boolean(pluginData.runtimeMetadataLoaded)}
								prefillSkills={allSkillItems}
								onRefreshSkills={() => {
									return refreshPluginSkills(pluginData.plugin.id, !pluginData.plugin.isBuiltIn);
								}}
								insertFilter={insertFilter}
								searchQuery={searchQuery}
								tagFilters={activeTagFilters}
								onTogglePluginEnable={handlePluginEnableChange}
								onToggleSkillEnable={handleSkillEnableChange}
								onDeleteSkill={handleDeleteSkill}
								onSubmitSkill={handleSubmitSkill}
								onEditPlugin={handleEditPlugin}
								onRequestPluginDelete={plugin => {
									setPluginToDelete(plugin);
								}}
							/>
						))}
					</div>
				</ManagementPageContent>

				<DeleteConfirmationModal
					isOpen={pluginToDelete !== null}
					onClose={() => {
						if (!isDeletingPlugin) {
							setPluginToDelete(null);
						}
					}}
					onConfirm={handlePluginDelete}
					title="Delete Skill Plugin"
					message={
						pluginToDelete
							? `Delete empty Plugin "${pluginToDelete.displayName || pluginToDelete.slug}"? Remove all skills from the Plugin first.`
							: 'Delete this empty Skill Plugin?'
					}
					confirmButtonText="Delete"
				/>

				<ManagementPluginCreateModal
					isOpen={isAddModalOpen}
					title="Add Skill Plugin"
					entityLabel="Skill Plugin"
					onClose={() => {
						setIsAddModalOpen(false);
					}}
					onSubmit={handleAddPlugin}
					existingSlugs={existingPluginSlugs}
					existingDisplayNames={existingPluginNames}
					failureMessage="Failed to create Skill Plugin."
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
