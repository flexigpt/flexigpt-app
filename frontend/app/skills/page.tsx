import type { SubmitEventHandler } from 'react';
import { useCallback, useMemo, useRef, useState } from 'react';
import { FiFolderPlus, FiPlus, FiRefreshCw, FiSearch } from 'react-icons/fi';

import type { PluginListItem } from '@/spec/plugin';
import type { StoreSkillListItem } from '@/spec/skill';
import { pluginListItemFromPluginView } from '@/spec/plugin';
import { SkillInsert } from '@/spec/skill';

import { throwIfAborted } from '@/lib/async_utils';
import { getErrorMessage } from '@/lib/error_utils';

import { useAsyncResource } from '@/hooks/use_async_resource';
import { usePendingActions } from '@/hooks/use_pending_actions';

import type { SkillManagementPresentation } from '@/apis/skill_management';
import { backendAPI, skillManagementAPI } from '@/apis/baseapi';

import type { DropdownItem } from '@/components/dropdown';
import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';
import { Dropdown } from '@/components/dropdown';
import { Loader } from '@/components/loader';
import { ManagementPageContent } from '@/components/managementui/management_page_content';
import { ManagementPageHeader } from '@/components/managementui/management_page_header';
import { ManagementPluginCreateModal } from '@/components/managementui/management_plugin_create_modal';
import { ManagementResourceError } from '@/components/managementui/management_resource_error';
import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalField } from '@/components/modal/modal_field';
import { ModalHeader } from '@/components/modal/modal_header';
import { PageFrame } from '@/components/page_frame';

import type { SkillManagementFilter } from '@/skills/lib/skill_management_utils';
import {
	canWriteSkillPlugin,
	skillArtifactKey,
	SkillManagementInsertFilter,
	skillMatchesManagementFilter,
} from '@/skills/lib/skill_management_utils';
import { SkillArtifactDetailsModal } from '@/skills/skill_add_edit_modal';
import { SkillPluginCard } from '@/skills/skill_plugin_card';

interface PluginRow {
	plugin: PluginListItem;
	skills: StoreSkillListItem[];
	skillsLoaded: boolean;
	skillsLoading: boolean;
	skillsError?: string;
}

interface PageData {
	plugins: PluginRow[];
	allSkills: StoreSkillListItem[];
	presentations: Record<string, SkillManagementPresentation>;
}

const skillInsertDropdownKeys = [
	SkillManagementInsertFilter.All,
	SkillInsert.Instructions,
	SkillInsert.UserMessage,
] as const;

const skillInsertDropdownItems: Record<SkillManagementFilter['insert'], DropdownItem> = {
	[SkillManagementInsertFilter.All]: { isEnabled: true },
	[SkillInsert.Instructions]: { isEnabled: true },
	[SkillInsert.UserMessage]: { isEnabled: true },
};

const skillInsertDropdownLabels: Record<SkillManagementFilter['insert'], string> = {
	[SkillManagementInsertFilter.All]: 'All Skills',
	[SkillInsert.Instructions]: 'Instruction Skills',
	[SkillInsert.UserMessage]: 'User-message templates',
};

function pluginKey(plugin: PluginListItem): string {
	return `${plugin.ref.rootID}:${plugin.ref.artifactID}`;
}

async function loadPageData(signal: AbortSignal, loadedPlugins: ReadonlySet<string>): Promise<PageData> {
	const data = await skillManagementAPI.loadManagementPageData();
	throwIfAborted(signal);

	return {
		plugins: await Promise.all(
			data.plugins.map(async plugin => {
				if (!loadedPlugins.has(pluginKey(plugin))) {
					return { plugin, skills: [], skillsLoaded: false, skillsLoading: false };
				}
				try {
					const result = await skillManagementAPI.getSkillPluginSkills(plugin.ref, data.skills);
					throwIfAborted(signal);
					return {
						plugin: pluginListItemFromPluginView(result.plugin, plugin.builtIn),
						skills: result.skills,
						skillsLoaded: true,
						skillsLoading: false,
						skillsError: result.diagnostics.length > 0 ? result.diagnostics.join('\n') : undefined,
					};
				} catch (cause) {
					throwIfAborted(signal);
					return {
						plugin,
						skills: [],
						skillsLoaded: false,
						skillsLoading: false,
						skillsError: getErrorMessage(cause, 'Plugin membership could not be loaded.'),
					};
				}
			})
		),
		allSkills: data.skills,
		presentations: data.presentations,
	};
}

function SkillDirectoryRegistrationModal({
	isOpen,
	roots,
	plugins,
	onClose,
	onRegistered,
	onWarning,
}: {
	isOpen: boolean;
	roots: Array<{ rootID: string; label: string }>;
	plugins: PluginListItem[];
	onClose: () => void;
	onRegistered: () => Promise<void>;
	onWarning: (message: string) => void;
}) {
	const [rootID, setRootID] = useState(roots[0]?.rootID ?? '');
	const [path, setPath] = useState('');
	const [sourceDisplayName, setSourceDisplayName] = useState('');
	const [error, setError] = useState('');
	const [saving, setSaving] = useState(false);
	const [destinationKey, setDestinationKey] = useState('unattached');
	const effectiveRootID = roots.some(root => root.rootID === rootID) ? rootID : (roots[0]?.rootID ?? '');
	const destinations = plugins.filter(plugin => plugin.ref.rootID === effectiveRootID && canWriteSkillPlugin(plugin));
	const destination = destinations.find(plugin => pluginKey(plugin) === destinationKey);
	const destinationValue = destination ? destinationKey : 'unattached';
	const rootDropdownItems = useMemo(
		() => Object.fromEntries(roots.map(root => [root.rootID, { isEnabled: true }])) as Record<string, DropdownItem>,
		[roots]
	);
	const destinationDropdownItems = useMemo(
		() =>
			({
				unattached: { isEnabled: true },
				...Object.fromEntries(destinations.map(plugin => [pluginKey(plugin), { isEnabled: true }])),
			}) as Record<string, DropdownItem>,
		[destinations]
	);
	const destinationLabel = (key: string) => {
		if (key === 'unattached') {
			return 'Register without attaching';
		}

		const plugin = destinations.find(candidate => pluginKey(candidate) === key);
		return plugin?.displayName || plugin?.name || key;
	};

	if (!isOpen) {
		return null;
	}

	const chooseDirectory = async () => {
		try {
			const selected = await backendAPI.pickDirectoryPath();
			if (selected) {
				setPath(selected);
			}
		} catch (cause) {
			setError(getErrorMessage(cause, 'Directory chooser could not be opened.'));
		}
	};

	const submit: SubmitEventHandler<HTMLFormElement> = async event => {
		event.preventDefault();
		event.stopPropagation();

		if (!effectiveRootID || !path.trim() || !sourceDisplayName.trim()) {
			setError('Root, directory, and source display name are required.');
			return;
		}

		setSaving(true);
		setError('');

		try {
			const result = await skillManagementAPI.registerSkillDirectoryAndAttach(
				{
					rootID: effectiveRootID,
					rootPath: path.trim(),
					sourceDisplayName: sourceDisplayName.trim(),
				},
				destination
			);

			onClose();

			const warnings = [...result.warnings];
			try {
				await onRegistered();
			} catch (cause) {
				warnings.push(
					`Directory registered, but the page could not be reloaded: ${getErrorMessage(cause, 'unknown error')}`
				);
			}

			if (warnings.length > 0) {
				onWarning(warnings.join('\n'));
			}
		} catch (cause) {
			setError(cause instanceof Error ? cause.message : 'Skill directory could not be registered.');
		} finally {
			setSaving(false);
		}
	};

	return (
		<ModalDialog isOpen onClose={onClose} blockCancel={saving}>
			<div className="modal-box bg-base-200 w-[calc(100%-1rem)] max-w-xl rounded-2xl p-0">
				<ModalHeader title="Register Skill Directory" onClose={onClose} closeDisabled={saving} />
				<form className="space-y-4 p-4 sm:p-6" onSubmit={submit}>
					{error ? <div className="alert alert-error rounded-2xl text-sm">{error}</div> : null}

					<ModalField label="Artifact Root" required>
						<Dropdown<string>
							dropdownItems={rootDropdownItems}
							orderedKeys={roots.map(root => root.rootID)}
							selectedKey={effectiveRootID}
							onChange={value => {
								setRootID(value);
								setDestinationKey('unattached');
							}}
							filterDisabled={false}
							title="Select Artifact Root"
							placeholderLabel="No Artifact Roots available"
							getDisplayName={value => roots.find(root => root.rootID === value)?.label ?? value}
							disabled={saving || roots.length === 0}
							inlineMenu
							maxMenuHeight={240}
						/>
					</ModalField>

					<ModalField label="Directory" htmlFor="skill-directory-path" required>
						<div className="flex gap-2">
							<input
								id="skill-directory-path"
								className="input min-w-0 flex-1 rounded-xl font-mono text-xs"
								value={path}
								disabled={saving}
								onChange={event => {
									setPath(event.currentTarget.value);
								}}
								placeholder="/path/to/skills"
							/>
							<button type="button" className="btn btn-ghost rounded-xl" disabled={saving} onClick={chooseDirectory}>
								Choose
							</button>
						</div>
					</ModalField>

					<ModalField label="Source display name" htmlFor="skill-directory-source-name" required>
						<input
							id="skill-directory-source-name"
							className="input w-full rounded-xl"
							value={sourceDisplayName}
							disabled={saving}
							onChange={event => {
								setSourceDisplayName(event.currentTarget.value);
							}}
							placeholder="Repository Skills"
						/>
					</ModalField>

					<ModalField label="Destination Plugin">
						<Dropdown<string>
							dropdownItems={destinationDropdownItems}
							orderedKeys={[
								'unattached',
								...destinations.map(d => {
									return pluginKey(d);
								}),
							]}
							selectedKey={destinationValue}
							onChange={value => {
								setDestinationKey(value);
							}}
							filterDisabled={false}
							title="Select Destination Plugin"
							placeholderLabel="Register without attaching"
							getDisplayName={destinationLabel}
							disabled={saving}
							inlineMenu
							maxMenuHeight={240}
						/>
					</ModalField>
					<div className="alert alert-info rounded-2xl text-sm">
						Every discovered Skill from this Source is attached to the selected Plugin. Registration without a
						destination leaves the Skills under Unattached Skills.
					</div>

					<ModalActions>
						<button type="button" className="btn bg-base-300 rounded-xl" disabled={saving} onClick={onClose}>
							Cancel
						</button>
						<button type="submit" className="btn btn-primary rounded-xl" disabled={saving}>
							{saving ? 'Registering...' : 'Register Directory'}
						</button>
					</ModalActions>
				</form>
			</div>
		</ModalDialog>
	);
}

// oxlint-disable-next-line no-restricted-exports
export default function SkillsPage() {
	const loadedPluginsRef = useRef(new Set<string>());
	const loadGenerationRef = useRef(0);
	const pluginRequestsRef = useRef(new Map<string, number>());
	const load = useCallback(async (signal: AbortSignal) => loadPageData(signal, new Set(loadedPluginsRef.current)), []);
	const { data, error, isLoading, isRefreshing, hasResolved, reloadOrThrow, setData } = useAsyncResource(load, {
		initialData: {
			plugins: [],
			allSkills: [],
			presentations: {},
		} satisfies PageData,
	});

	const [isCreatePluginOpen, setIsCreatePluginOpen] = useState(false);
	const [isDirectoryRegistrationOpen, setIsDirectoryRegistrationOpen] = useState(false);
	const [creationRootID, setCreationRootID] = useState('');
	const [alert, setAlert] = useState('');
	const [filter, setFilter] = useState<SkillManagementFilter>({
		query: '',
		insert: SkillManagementInsertFilter.All,
		tags: [],
	});
	const [skillToView, setSkillToView] = useState<StoreSkillListItem | null>(null);
	const [skillToPurge, setSkillToPurge] = useState<StoreSkillListItem | null>(null);
	const [purging, setPurging] = useState(false);
	const { isPending, runAction } = usePendingActions();

	const refresh = useCallback(async () => {
		loadGenerationRef.current += 1;
		skillManagementAPI.invalidateComposerSkillsCatalog();
		await reloadOrThrow();
	}, [reloadOrThrow]);

	const reloadAfterMutation = useCallback(
		async (message: string) => {
			try {
				await refresh();
			} catch (cause) {
				setAlert(
					`${message} succeeded, but the page could not be refreshed: ${getErrorMessage(cause, 'unknown error')}`
				);
			}
		},
		[refresh]
	);

	const refreshSources = useCallback(async () => {
		let sourceRefreshError: unknown;

		try {
			await skillManagementAPI.refreshAllSkillSources();
		} catch (cause) {
			sourceRefreshError = cause;
		}

		try {
			await refresh();
		} catch (cause: unknown) {
			if (sourceRefreshError) {
				throw new Error(
					`${getErrorMessage(sourceRefreshError, 'Skill sources could not be refreshed.')}\n${getErrorMessage(
						cause,
						'Skill management data could not be reloaded.'
					)}`,
					{ cause: cause }
				);
			}
			throw cause;
		}

		if (sourceRefreshError) {
			// oxlint-disable-next-line typescript/only-throw-error
			throw sourceRefreshError;
		}
	}, [refresh]);

	const handleRefreshSources = () => {
		refreshSources().catch((cause: unknown) => {
			setAlert(getErrorMessage(cause, 'Skill management data could not be refreshed.'));
		});
	};

	const handleUnattachedSkillToggle = (skill: StoreSkillListItem) => {
		runAction(`orphan:${skillArtifactKey(skill.ref)}:toggle`, () => toggleSkill(skill, !skill.enabled)).catch(
			(cause: unknown) => {
				setAlert(getErrorMessage(cause, 'Skill enablement could not be changed.'));
			}
		);
	};

	const roots = useMemo(
		() =>
			[
				...new Map(
					data.plugins
						.filter(row => row.plugin.baseline && !row.plugin.builtIn)
						.map(row => [row.plugin.ref.rootID, row.plugin.displayName || row.plugin.name] as const)
				).entries(),
			].map(([rootID, label]) => ({
				rootID,
				label,
			})),
		[data.plugins]
	);

	const effectiveCreationRootID = roots.some(root => root.rootID === creationRootID)
		? creationRootID
		: (roots[0]?.rootID ?? '');
	const creationRootDropdownItems = useMemo(
		() => Object.fromEntries(roots.map(root => [root.rootID, { isEnabled: true }])) as Record<string, DropdownItem>,
		[roots]
	);

	const updatePluginRow = useCallback(
		(plugin: PluginListItem, update: (row: PluginRow) => PluginRow) => {
			setData(previous => ({
				...previous,
				plugins: previous.plugins.map(row => (pluginKey(row.plugin) === pluginKey(plugin) ? update(row) : row)),
			}));
		},
		[setData]
	);

	const loadPluginSkills = useCallback(
		async (plugin: PluginListItem) => {
			const key = pluginKey(plugin);
			loadedPluginsRef.current.add(key);
			const generation = loadGenerationRef.current;
			const request = (pluginRequestsRef.current.get(key) ?? 0) + 1;
			pluginRequestsRef.current.set(key, request);
			updatePluginRow(plugin, row => ({
				...row,
				skillsLoading: true,
				skillsError: undefined,
			}));

			try {
				const result = await skillManagementAPI.getSkillPluginSkills(plugin.ref);
				if (generation !== loadGenerationRef.current || pluginRequestsRef.current.get(key) !== request) {
					return;
				}
				updatePluginRow(plugin, row => ({
					...row,
					plugin: pluginListItemFromPluginView(result.plugin, row.plugin.builtIn),
					skills: result.skills,
					skillsLoaded: true,
					skillsLoading: false,
					skillsError: result.diagnostics.length > 0 ? result.diagnostics.join('\n') : undefined,
				}));
			} catch (cause) {
				if (generation !== loadGenerationRef.current || pluginRequestsRef.current.get(key) !== request) {
					return;
				}
				updatePluginRow(plugin, row => ({
					...row,
					skillsLoading: false,
					skillsError: cause instanceof Error ? cause.message : 'Skill membership could not be resolved.',
				}));
				throw cause;
			}
		},
		[updatePluginRow]
	);

	const refreshPlugin = useCallback(
		async (plugin: PluginListItem) => {
			loadedPluginsRef.current.add(pluginKey(plugin));
			try {
				await skillManagementAPI.refreshSkillPluginSources(plugin.ref);
			} finally {
				await refresh();
			}
		},
		[refresh]
	);

	const togglePlugin = useCallback(
		async (plugin: PluginListItem, enabled: boolean) => {
			await skillManagementAPI.setSkillPluginEnabled(plugin.ref, plugin.revision, enabled);
			await reloadAfterMutation('Plugin enablement change');
		},
		[reloadAfterMutation]
	);

	const toggleSkill = useCallback(
		async (skill: StoreSkillListItem, enabled: boolean) => {
			await skillManagementAPI.setSkillEnabled(skill.ref, skill.revision, enabled);
			await reloadAfterMutation('Skill enablement change');
		},
		[reloadAfterMutation]
	);

	const attachExisting = useCallback(
		async (plugin: PluginListItem, skill: StoreSkillListItem) => {
			await skillManagementAPI.attachSkillArtifactToPlugin({
				plugin: plugin.ref,
				expectedRevision: plugin.revision,
				artifact: skill.ref,
			});
			await reloadAfterMutation('Skill attachment');
		},
		[reloadAfterMutation]
	);

	const attachPath = useCallback(
		async (plugin: PluginListItem, path: string) => {
			const result = await skillManagementAPI.addSkillPathAndAttach(plugin, path);
			await reloadAfterMutation('Skill path registration');
			if (result.warning) {
				setAlert(result.warning);
			}
		},
		[reloadAfterMutation]
	);

	const detachSkill = useCallback(
		async (plugin: PluginListItem, skill: StoreSkillListItem) => {
			await skillManagementAPI.detachSkillFromPlugin(plugin, skill);
			await reloadAfterMutation('Skill detachment');
		},
		[reloadAfterMutation]
	);

	const deleteSkillFromPlugin = useCallback(
		async (plugin: PluginListItem, skill: StoreSkillListItem) => {
			const result = await skillManagementAPI.deleteSkillFromPlugin(plugin, skill);
			await reloadAfterMutation('Skill removal');
			if (result.warning) {
				setAlert(result.warning);
			}
		},
		[reloadAfterMutation]
	);

	const purgeSkill = useCallback(
		async (skill: StoreSkillListItem) => {
			await skillManagementAPI.purgeSkill(skill.ref, skill.revision);
			await reloadAfterMutation('Managed Artifact deletion');
		},
		[reloadAfterMutation]
	);

	const confirmPurgeUnattachedSkill = async () => {
		const skill = skillToPurge;
		if (!skill || purging) {
			return;
		}

		setPurging(true);
		try {
			await purgeSkill(skill);
			setSkillToPurge(null);
		} catch (cause) {
			setAlert(getErrorMessage(cause, 'Managed Artifact could not be deleted.'));
		} finally {
			setPurging(false);
		}
	};

	const deletePlugin = useCallback(
		async (plugin: PluginListItem) => {
			await skillManagementAPI.deleteSkillPlugin({
				plugin: plugin.ref,
				expectedRevision: plugin.revision,
			});
			await reloadAfterMutation('Plugin deletion');
		},
		[reloadAfterMutation]
	);

	const createPlugin = useCallback(
		async (name: string, displayName: string, description?: string) => {
			await skillManagementAPI.createSkillPlugin({
				rootID: effectiveCreationRootID,
				name,
				displayName,
				description,
			});
			await reloadAfterMutation('Plugin creation');
		},
		[effectiveCreationRootID, reloadAfterMutation]
	);

	const matchingSkills = data.allSkills.filter(skill =>
		skillMatchesManagementFilter(skill, data.presentations[skillArtifactKey(skill.ref)], filter)
	);
	const unattachedSkills = matchingSkills.filter(skill => {
		const memberships = data.presentations[skillArtifactKey(skill.ref)]?.memberships;
		return memberships !== undefined && memberships.length === 0;
	});
	const knownTags = [
		...new Set(data.allSkills.flatMap(skill => data.presentations[skillArtifactKey(skill.ref)]?.document?.tags ?? [])),
	].toSorted((left, right) => left.localeCompare(right));
	const instructionCount = data.allSkills.filter(
		skill => data.presentations[skillArtifactKey(skill.ref)]?.summary?.Insert === SkillInsert.Instructions
	).length;
	const templateCount = data.allSkills.filter(
		skill => data.presentations[skillArtifactKey(skill.ref)]?.summary?.Insert === SkillInsert.UserMessage
	).length;

	if (isLoading && !hasResolved) {
		return <Loader text="Loading Skill Plugins..." />;
	}

	return (
		<PageFrame>
			<div className="flex size-full flex-col items-center overflow-hidden">
				<ManagementPageHeader
					title="Skills and Templates"
					description="Manage Artifact-backed Skill Plugins, managed SKILL.md files, filesystem Skill paths, and user-message templates."
					actions={
						<>
							{roots.length > 1 ? (
								<div className="w-56 max-w-72">
									<Dropdown<string>
										dropdownItems={creationRootDropdownItems}
										orderedKeys={roots.map(root => root.rootID)}
										selectedKey={effectiveCreationRootID}
										onChange={setCreationRootID}
										filterDisabled={false}
										title="Skill Plugin Root"
										getDisplayName={rootID => roots.find(root => root.rootID === rootID)?.label ?? rootID}
									/>
								</div>
							) : null}

							<button
								type="button"
								className="btn btn-ghost rounded-xl"
								disabled={isRefreshing}
								onClick={handleRefreshSources}
							>
								<FiRefreshCw size={17} className={isRefreshing ? 'animate-spin' : undefined} />
								Refresh
							</button>

							<button
								type="button"
								className="btn btn-ghost rounded-xl"
								disabled={roots.length === 0}
								onClick={() => {
									setIsDirectoryRegistrationOpen(true);
								}}
							>
								<FiFolderPlus size={17} />
								Register Directory
							</button>

							<button
								type="button"
								className="btn btn-ghost rounded-xl"
								onClick={() => {
									setIsCreatePluginOpen(true);
								}}
							>
								<FiPlus size={17} />
								Add Plugin
							</button>
						</>
					}
				/>

				<ManagementPageContent>
					<div className="grid gap-2 sm:grid-cols-3">
						{[
							{ label: 'Instruction Skills', count: instructionCount, insert: SkillInsert.Instructions },
							{ label: 'Templates', count: templateCount, insert: SkillInsert.UserMessage },
							{ label: 'Known tags', count: knownTags.length, insert: SkillManagementInsertFilter.All },
						].map(card => (
							<button
								key={card.label}
								type="button"
								className="bg-base-100 border-base-300 rounded-2xl border p-3 text-left"
								onClick={() => {
									setFilter(previous => ({ ...previous, insert: card.insert }));
								}}
							>
								<span className="block text-xs">{card.label}</span>
								<strong className="text-lg">{card.count}</strong>
							</button>
						))}
					</div>
					{knownTags.length > 0 ? (
						<div className="flex flex-wrap gap-1" aria-label="Known Skill tags">
							{knownTags.map(tag => (
								<button
									key={tag}
									type="button"
									className="badge badge-ghost cursor-pointer"
									onClick={() => {
										setFilter(previous => ({ ...previous, tags: [tag] }));
									}}
								>
									{tag}
								</button>
							))}
						</div>
					) : null}
					<div className="bg-base-100 border-base-300 grid gap-3 rounded-2xl border p-3 sm:grid-cols-3">
						<label className="input flex items-center gap-2 rounded-xl">
							<FiSearch size={14} />
							<input
								type="search"
								value={filter.query}
								onChange={event => {
									setFilter(previous => ({
										...previous,
										query: event.currentTarget.value,
									}));
								}}
								placeholder="Search names, descriptions, tags, arguments…"
								aria-label="Search Skills"
							/>
						</label>
						<Dropdown<SkillManagementFilter['insert']>
							dropdownItems={skillInsertDropdownItems}
							orderedKeys={skillInsertDropdownKeys}
							selectedKey={filter.insert}
							onChange={insert => {
								setFilter(previous => ({
									...previous,
									insert,
								}));
							}}
							filterDisabled={false}
							title="Skill behavior filter"
							getDisplayName={insert => skillInsertDropdownLabels[insert]}
						/>
						<input
							className="input rounded-xl"
							value={filter.tags.join(', ')}
							onChange={event => {
								const value = event.currentTarget.value;
								setFilter(previous => ({
									...previous,
									tags: value
										.split(',')
										.map(tag => tag.trim())
										.filter(Boolean),
								}));
							}}
							placeholder="Tags, comma separated"
							aria-label="Skill tag filter"
						/>
						<div className="text-base-content/60 text-xs sm:col-span-3">
							{matchingSkills.length} matching Artifact{matchingSkills.length === 1 ? '' : 's'}. Tag and argument
							metadata is searchable for verified Skill documents. Unavailable interpretation is shown in Skill Details.
						</div>
					</div>
					{error ? (
						<ManagementResourceError
							title="Skill management data could not be loaded"
							error={error}
							isRetrying={isRefreshing}
							onRetry={refresh}
						/>
					) : null}

					{data.plugins.length === 0 && !error ? (
						<p className="mt-8 text-center text-sm">No Skill Plugins are currently available.</p>
					) : null}

					<div className="text-base-content/70 border-base-content/10 bg-base-100 rounded-2xl border p-3 text-sm">
						{data.allSkills.length} discovered Skill Artifact{data.allSkills.length === 1 ? '' : 's'} are available
						across all Roots. Use “Attach Existing” to place a discovered Artifact into a Plugin without recreating it.
					</div>

					{data.plugins.map(row => (
						<SkillPluginCard
							key={pluginKey(row.plugin)}
							plugin={row.plugin}
							skills={row.skills}
							allSkills={data.allSkills}
							presentations={data.presentations}
							filter={filter}
							onReload={refresh}
							skillsLoaded={row.skillsLoaded}
							skillsLoading={row.skillsLoading}
							skillsError={row.skillsError}
							onLoadSkills={loadPluginSkills}
							onRefresh={() => refreshPlugin(row.plugin)}
							onTogglePlugin={togglePlugin}
							onToggleSkill={toggleSkill}
							onAttachExisting={attachExisting}
							onAttachPath={attachPath}
							onDetachSkill={detachSkill}
							onDeleteSkill={deleteSkillFromPlugin}
							onDeletePlugin={deletePlugin}
							onError={setAlert}
						/>
					))}

					<section className="space-y-3">
						<h2 className="text-sm font-semibold">Unattached Skills</h2>
						{unattachedSkills.length === 0 ? (
							<p className="text-base-content/60 text-sm">No unattached Skills match these filters.</p>
						) : null}
						{unattachedSkills.map(skill => (
							<div
								key={skillArtifactKey(skill.ref)}
								className="bg-base-100 border-base-content/10 rounded-2xl border p-3"
							>
								<div className="font-medium">{skill.displayName || skill.name}</div>
								<div className="text-base-content/60 mt-1 text-xs">
									{skill.ref.rootID}/{skill.ref.artifactID} · {skill.state}
								</div>
								<div className="mt-3 flex flex-wrap gap-2">
									<button
										type="button"
										className="btn btn-sm rounded-xl"
										onClick={() => {
											setSkillToView(skill);
										}}
									>
										View
									</button>
									<button
										type="button"
										className="btn btn-sm rounded-xl"
										disabled={isPending(`orphan:${skillArtifactKey(skill.ref)}:toggle`)}
										onClick={() => {
											handleUnattachedSkillToggle(skill);
										}}
									>
										{skill.enabled ? 'Disable' : 'Enable'}
									</button>
									{skill.managed && !skill.builtIn ? (
										<button
											type="button"
											className="btn btn-sm rounded-xl"
											disabled={purging}
											onClick={() => {
												setSkillToPurge(skill);
											}}
										>
											Delete Artifact
										</button>
									) : null}
								</div>
								<p className="text-base-content/60 mt-2 text-xs">
									Use Attach Existing on a Plugin to attach this Artifact. External source files are not deleted by
									detachment.
								</p>
							</div>
						))}
					</section>
				</ManagementPageContent>

				<ManagementPluginCreateModal
					isOpen={isCreatePluginOpen}
					title="Add Skill Plugin"
					entityLabel="Skill Plugin"
					onClose={() => {
						setIsCreatePluginOpen(false);
					}}
					onSubmit={createPlugin}
					existingSlugs={data.plugins
						.filter(row => row.plugin.ref.rootID === effectiveCreationRootID)
						.map(row => row.plugin.name)}
					existingDisplayNames={data.plugins
						.filter(row => row.plugin.ref.rootID === effectiveCreationRootID)
						.map(row => row.plugin.displayName || row.plugin.name)}
					failureMessage="Skill Plugin could not be created."
				/>

				<SkillArtifactDetailsModal
					isOpen={skillToView !== null}
					skill={skillToView}
					onClose={() => {
						setSkillToView(null);
					}}
				/>

				<DeleteConfirmationModal
					isOpen={skillToPurge !== null}
					onClose={() => {
						if (!purging) {
							setSkillToPurge(null);
						}
					}}
					onConfirm={confirmPurgeUnattachedSkill}
					title="Delete Unattached Managed Skill"
					message={`Permanently delete "${skillToPurge?.displayName || skillToPurge?.name || ''}"? Memberships are checked again before deletion.`}
					confirmButtonText="Delete"
				/>

				<SkillDirectoryRegistrationModal
					key={`skill-directory:${isDirectoryRegistrationOpen ? 'open' : 'closed'}:${roots
						.map(root => root.rootID)
						.join('|')}`}
					isOpen={isDirectoryRegistrationOpen}
					roots={roots}
					plugins={data.plugins.map(row => row.plugin)}
					onWarning={setAlert}
					onClose={() => {
						setIsDirectoryRegistrationOpen(false);
					}}
					onRegistered={refresh}
				/>

				<ActionDeniedAlertModal
					isOpen={Boolean(alert)}
					onClose={() => {
						setAlert('');
					}}
					message={alert}
				/>
			</div>
		</PageFrame>
	);
}
