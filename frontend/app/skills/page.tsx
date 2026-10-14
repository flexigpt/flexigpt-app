import type { SubmitEventHandler } from 'react';
import { useCallback, useMemo, useState } from 'react';
import { FiFolderPlus, FiPlus, FiRefreshCw } from 'react-icons/fi';

import type { PluginListItem } from '@/spec/plugin';
import type { StoreSkillListItem } from '@/spec/skill';
import { CapabilityTargetForm } from '@/spec/artifact';

import { mapWithConcurrency } from '@/lib/async_utils';
import { getErrorMessage } from '@/lib/error_utils';

import { useAsyncResource } from '@/hooks/use_async_resource';

import { backendAPI, skillManagementAPI } from '@/apis/baseapi';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
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
}

function pluginKey(plugin: PluginListItem): string {
	return `${plugin.ref.rootID}:${plugin.ref.artifactID}`;
}

function skillKey(skill: StoreSkillListItem): string {
	return `${skill.ref.rootID}:${skill.ref.artifactID}`;
}

function sameRef(left: { rootID: string; artifactID: string }, right: { rootID: string; artifactID: string }): boolean {
	return left.rootID === right.rootID && left.artifactID === right.artifactID;
}

async function loadPageData(): Promise<PageData> {
	const data = await skillManagementAPI.loadManagementPageData();

	return {
		plugins: data.plugins.map(plugin => ({
			plugin,
			skills: [],
			skillsLoaded: false,
			skillsLoading: false,
		})),
		allSkills: data.skills,
	};
}

async function resolvePluginSkills(
	plugin: PluginListItem,
	allSkills: StoreSkillListItem[]
): Promise<StoreSkillListItem[]> {
	const resolvedSkillKeys = new Set<string>();

	const plan = await skillManagementAPI.resolveSkillPlugin(plugin.ref).catch(() => undefined);
	for (const occurrence of plan?.occurrences ?? []) {
		const target = occurrence.target;
		if (occurrence.type === 'skill' && target?.form === CapabilityTargetForm.Artifact && target.artifact) {
			resolvedSkillKeys.add(`${target.artifact.rootID}:${target.artifact.artifactID}`);
		}
	}

	const membershipResults = await mapWithConcurrency(allSkills, 8, async skill => {
		const memberships = await skillManagementAPI.listSkillPluginMemberships(skill.ref).catch(() => []);
		return {
			skill,
			memberOfPlugin: memberships.some(membership => sameRef(membership.plugin, plugin.ref)),
		};
	});

	for (const membershipResult of membershipResults) {
		if (membershipResult.memberOfPlugin) {
			resolvedSkillKeys.add(skillKey(membershipResult.skill));
		}
	}

	return allSkills
		.filter(skill => resolvedSkillKeys.has(skillKey(skill)))
		.toSorted((left, right) =>
			(left.displayName || left.name).localeCompare(right.displayName || right.name, undefined, {
				numeric: true,
				sensitivity: 'base',
			})
		);
}

function SkillDirectoryRegistrationModal({
	isOpen,
	roots,
	onClose,
	onRegistered,
}: {
	isOpen: boolean;
	roots: Array<{ rootID: string; label: string }>;
	onClose: () => void;
	onRegistered: () => Promise<void>;
}) {
	const [rootID, setRootID] = useState(roots[0]?.rootID ?? '');
	const [path, setPath] = useState('');
	const [sourceDisplayName, setSourceDisplayName] = useState('');
	const [error, setError] = useState('');
	const [saving, setSaving] = useState(false);

	if (!isOpen) {
		return null;
	}

	const submit: SubmitEventHandler<HTMLFormElement> = event => {
		event.preventDefault();

		if (!rootID || !path.trim() || !sourceDisplayName.trim()) {
			setError('Root, directory, and source display name are required.');
			return;
		}

		setSaving(true);
		setError('');

		void skillManagementAPI
			.registerSkillDirectory({
				rootID,
				rootPath: path.trim(),
				sourceDisplayName: sourceDisplayName.trim(),
			})
			.then(async () => {
				await onRegistered();
				onClose();
			})
			.catch((cause: unknown) => {
				setError(cause instanceof Error ? cause.message : 'Skill directory could not be registered.');
			})
			.finally(() => {
				setSaving(false);
			});
	};

	return (
		<ModalDialog isOpen onClose={onClose} blockCancel={saving}>
			<div className="modal-box bg-base-200 w-[calc(100%-1rem)] max-w-xl rounded-2xl p-0">
				<ModalHeader title="Register Skill Directory" onClose={onClose} closeDisabled={saving} />
				<form className="space-y-4 p-4 sm:p-6" onSubmit={submit}>
					{error ? <div className="alert alert-error rounded-2xl text-sm">{error}</div> : null}

					<ModalField label="Artifact Root" htmlFor="skill-directory-root" required>
						<select
							id="skill-directory-root"
							className="select w-full rounded-xl"
							value={rootID}
							disabled={saving}
							onChange={event => {
								setRootID(event.currentTarget.value);
							}}
						>
							{roots.map(root => (
								<option key={root.rootID} value={root.rootID}>
									{root.label}
								</option>
							))}
						</select>
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
							<button
								type="button"
								className="btn btn-ghost rounded-xl"
								disabled={saving}
								onClick={() => {
									void backendAPI.pickDirectoryPath().then(selected => {
										if (selected) {
											setPath(selected);
										}
									});
								}}
							>
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

					<div className="alert alert-info rounded-2xl text-sm">
						Registered Skill Artifacts appear after refresh. Attach them to a Skill Plugin with “Attach Existing”.
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
	const load = useCallback(async (_signal: AbortSignal) => loadPageData(), []);
	const { data, error, isLoading, isRefreshing, hasResolved, reloadOrThrow, setData } = useAsyncResource(load, {
		initialData: {
			plugins: [],
			allSkills: [],
		} satisfies PageData,
	});

	const [isCreatePluginOpen, setIsCreatePluginOpen] = useState(false);
	const [isDirectoryRegistrationOpen, setIsDirectoryRegistrationOpen] = useState(false);
	const [creationRootID, setCreationRootID] = useState('');
	const [alert, setAlert] = useState('');

	const refresh = useCallback(async () => {
		skillManagementAPI.invalidateComposerSkillsCatalog();
		await reloadOrThrow();
	}, [reloadOrThrow]);

	const roots = useMemo(
		() =>
			[...new Map(data.plugins.map(row => [row.plugin.ref.rootID, row.plugin.ref.rootID] as const)).entries()].map(
				([rootID, label]) => ({
					rootID,
					label,
				})
			),
		[data.plugins]
	);

	const effectiveCreationRootID = roots.some(root => root.rootID === creationRootID)
		? creationRootID
		: (roots[0]?.rootID ?? '');

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
			updatePluginRow(plugin, row => ({
				...row,
				skillsLoading: true,
				skillsError: undefined,
			}));

			try {
				const skills = await resolvePluginSkills(plugin, data.allSkills);
				updatePluginRow(plugin, row => ({
					...row,
					skills,
					skillsLoaded: true,
					skillsLoading: false,
					skillsError: undefined,
				}));
			} catch (cause) {
				updatePluginRow(plugin, row => ({
					...row,
					skillsLoading: false,
					skillsError: cause instanceof Error ? cause.message : 'Skill membership could not be resolved.',
				}));
				throw cause;
			}
		},
		[data.allSkills, updatePluginRow]
	);

	const refreshPlugin = useCallback(
		async (plugin: PluginListItem) => {
			await refresh();
			await loadPluginSkills(plugin);
		},
		[loadPluginSkills, refresh]
	);

	const togglePlugin = useCallback(
		async (plugin: PluginListItem, enabled: boolean) => {
			await skillManagementAPI.setSkillPluginEnabled(plugin.ref, plugin.revision, enabled);
			await refresh();
		},
		[refresh]
	);

	const toggleSkill = useCallback(
		async (skill: StoreSkillListItem, enabled: boolean) => {
			await skillManagementAPI.setSkillEnabled(skill.ref, skill.revision, enabled);
			await refresh();
		},
		[refresh]
	);

	const attachExisting = useCallback(
		async (plugin: PluginListItem, skill: StoreSkillListItem) => {
			await skillManagementAPI.attachSkillArtifactToPlugin({
				plugin: plugin.ref,
				expectedRevision: plugin.revision,
				artifact: skill.ref,
			});
			await refresh();
		},
		[refresh]
	);

	const attachPath = useCallback(
		async (plugin: PluginListItem, path: string) => {
			const result = await skillManagementAPI.addSkillPath({
				rootID: plugin.ref.rootID,
				path,
				sourceDisplayName: `${plugin.displayName || plugin.name} external Skill`,
				enabled: true,
			});

			await skillManagementAPI.attachSkillArtifactToPlugin({
				plugin: plugin.ref,
				expectedRevision: plugin.revision,
				artifact: {
					rootID: result.artifact.rootID,
					artifactID: result.artifact.id,
				},
			});
			await refresh();
		},
		[refresh]
	);

	const detachSkill = useCallback(
		async (plugin: PluginListItem, skill: StoreSkillListItem) => {
			const memberships = await skillManagementAPI.listSkillPluginMemberships(skill.ref);
			const membership = memberships.find(value => sameRef(value.plugin, plugin.ref));

			if (!membership) {
				throw new Error('This Skill is not a direct member of the selected Plugin.');
			}

			await skillManagementAPI.removeSkillPluginMember({
				plugin: membership.plugin,
				expectedRevision: membership.pluginRevision,
				index: membership.memberIndex,
			});
			await refresh();
		},
		[refresh]
	);

	const purgeSkill = useCallback(
		async (skill: StoreSkillListItem) => {
			await skillManagementAPI.purgeSkill(skill.ref, skill.revision);
			await refresh();
		},
		[refresh]
	);

	const deletePlugin = useCallback(
		async (plugin: PluginListItem) => {
			await skillManagementAPI.deleteSkillPlugin({
				plugin: plugin.ref,
				expectedRevision: plugin.revision,
			});
			await refresh();
		},
		[refresh]
	);

	const createPlugin = useCallback(
		async (name: string, displayName: string, description?: string) => {
			await skillManagementAPI.createSkillPlugin({
				rootID: effectiveCreationRootID,
				name,
				displayName,
				description,
			});
			await refresh();
		},
		[effectiveCreationRootID, refresh]
	);

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
								<select
									className="select select-sm max-w-72 rounded-xl"
									aria-label="Skill Plugin Root"
									value={effectiveCreationRootID}
									onChange={event => {
										setCreationRootID(event.currentTarget.value);
									}}
								>
									{roots.map(root => (
										<option key={root.rootID} value={root.rootID}>
											{root.label}
										</option>
									))}
								</select>
							) : null}

							<button
								type="button"
								className="btn btn-ghost rounded-xl"
								disabled={isRefreshing}
								onClick={() => {
									void refresh().catch((cause: unknown) => {
										setAlert(getErrorMessage(cause, 'Skill management data could not be refreshed.'));
									});
								}}
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
							onPurgeSkill={purgeSkill}
							onDeletePlugin={deletePlugin}
							onError={setAlert}
						/>
					))}
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

				<SkillDirectoryRegistrationModal
					isOpen={isDirectoryRegistrationOpen}
					roots={roots}
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
