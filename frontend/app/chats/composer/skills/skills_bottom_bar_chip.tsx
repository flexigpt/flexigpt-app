// oxlint-disable unicorn/no-immediate-mutation
import type { Dispatch, SetStateAction } from 'react';
import { useEffect, useMemo, useState } from 'react';
import { FiAlertCircle, FiCheck, FiFilePlus, FiGitBranch, FiPlus, FiRefreshCw, FiX, FiZap } from 'react-icons/fi';

import type { MenuStore } from '@ariakit/react';
import { Menu, MenuButton, MenuItem, useStoreState } from '@ariakit/react';

import type { PluginListItem } from '@/spec/plugin';
import type { SkillDocumentInput, SkillRef } from '@/spec/skill';
import { ArtifactState } from '@/spec/artifact';
import { SkillInsert } from '@/spec/skill';

import type { ComposerSkillCatalogItem } from '@/apis/skill_management';
import { skillManagementAPI } from '@/apis/baseapi';

import {
	actionTriggerChipClearButtonClasses,
	ActionTriggerChipContent,
	actionTriggerChipSurfaceClasses,
	actionTriggerMenuItemClasses,
	actionTriggerMenuWideClasses,
} from '@/components/action_trigger_chip';
import { HoverTip, HoverTipContent } from '@/components/hover_tip';
import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalField } from '@/components/modal/modal_field';
import { ModalHeader } from '@/components/modal/modal_header';
import { searchableMenuEmptyStateClasses, SearchableMenuInput } from '@/components/searchmenu/searchable_menu';
import {
	focusFirstSearchableMenuItem,
	isSearchQueryActive,
	rankSearchableItems,
	useSearchableMenuState,
} from '@/components/searchmenu/searchable_menu_utils';

import type { AgentSystemPromptController } from '@/chats/composer/skills/use_agent_system_prompt';
import type { ManagedSkillEditorSeed } from '@/skills/skill_add_edit_modal';
import { skillRefKey } from '@/skills/lib/skill_identity_utils';
import { ManagedSkillEditorModal } from '@/skills/skill_add_edit_modal';

interface SkillsBottomBarChipProps {
	store: MenuStore;
	shortcut: string;
	allSkills: ComposerSkillCatalogItem[];
	loading: boolean;
	loadError?: string | null;
	enabledSkillRefs: SkillRef[];
	activeSkillRefs: SkillRef[];
	setEnabledSkillRefs: Dispatch<SetStateAction<SkillRef[]>>;
	setActiveSkillRefs: Dispatch<SetStateAction<SkillRef[]>>;
	onEnableAll: () => void;
	onDisableAll: () => void;
	onRefreshSkills: () => Promise<void>;
	systemPrompt: AgentSystemPromptController;
	isInputLocked?: boolean;
}

interface PendingManagedSkillAction {
	mode: 'create' | 'fork';
	seed?: ManagedSkillEditorSeed;
}

function skillRef(item: ComposerSkillCatalogItem): SkillRef {
	return {
		rootID: item.ref.rootID,
		artifactID: item.ref.artifactID,
	};
}

function skillKey(item: ComposerSkillCatalogItem): string {
	return skillRefKey(skillRef(item));
}

function skillLabel(item: ComposerSkillCatalogItem): string {
	return item.displayName || item.name;
}

function compareSkills(left: ComposerSkillCatalogItem, right: ComposerSkillCatalogItem): number {
	return (
		skillLabel(left).localeCompare(skillLabel(right), undefined, {
			numeric: true,
			sensitivity: 'base',
		}) || skillKey(left).localeCompare(skillKey(right))
	);
}

function canActivate(item: ComposerSkillCatalogItem): boolean {
	return !item.hasArguments;
}

function uniqueForkName(seed: string, items: ComposerSkillCatalogItem[]): string {
	const base =
		seed
			.trim()
			.toLowerCase()
			.replaceAll(/[^a-z0-9-]+/g, '-')
			.replaceAll(/-+/g, '-')
			.replaceAll(/^-|-$/g, '')
			.slice(0, 58) || 'skill';

	const existing = new Set(items.map(item => item.name));
	let candidate = `${base}-copy`;
	let index = 2;

	while (existing.has(candidate)) {
		const suffix = `-copy-${index}`;
		candidate = `${base.slice(0, Math.max(1, 64 - suffix.length))}${suffix}`;
		index += 1;
	}

	return candidate;
}

function fallbackForkSeed(
	item: ComposerSkillCatalogItem,
	allSkills: ComposerSkillCatalogItem[]
): ManagedSkillEditorSeed {
	const name = uniqueForkName(item.name, allSkills);

	return {
		document: {
			name,
			displayName: `${skillLabel(item)} Copy`,
			description: item.description || `Forked from ${skillLabel(item)}.`,
			insert: SkillInsert.Instructions,
			arguments: [],
			tags: ['composer', 'forked'],
			markdownBody: `# ${skillLabel(item)} Copy

Replace this placeholder with the managed instruction Skill content you want to retain.

The original Skill Artifact remains unchanged.`,
		},
		enabled: true,
	};
}

function ComposerSkillDestinationModal({
	isOpen,
	plugins,
	action,
	onClose,
	onChoose,
}: {
	isOpen: boolean;
	plugins: PluginListItem[];
	action: PendingManagedSkillAction | null;
	onClose: () => void;
	onChoose: (plugin: PluginListItem, action: PendingManagedSkillAction) => void;
}) {
	const [selectedKey, setSelectedKey] = useState('');

	if (!isOpen || !action) {
		return null;
	}

	const writablePlugins = plugins.filter(
		plugin => plugin.editable && plugin.enabled && plugin.state === ArtifactState.Available
	);
	const selected = writablePlugins.find(plugin => `${plugin.ref.rootID}:${plugin.ref.artifactID}` === selectedKey);

	return (
		<ModalDialog isOpen onClose={onClose}>
			<div className="modal-box bg-base-200 w-[calc(100%-1rem)] max-w-xl rounded-2xl p-0">
				<ModalHeader
					title={action.mode === 'fork' ? 'Choose Fork Destination' : 'Choose Skill Destination'}
					onClose={onClose}
				/>

				<div className="space-y-4 p-4 sm:p-6">
					<div className="text-base-content/70 text-sm">
						Managed Skills are created inside an editable, enabled Skill Plugin.
					</div>

					<ModalField label="Skill Plugin" htmlFor="composer-skill-plugin" required>
						<select
							id="composer-skill-plugin"
							className="select w-full rounded-xl"
							value={selectedKey}
							onChange={event => {
								setSelectedKey(event.currentTarget.value);
							}}
						>
							<option value="">Select a Skill Plugin</option>
							{writablePlugins.map(plugin => (
								<option
									key={`${plugin.ref.rootID}:${plugin.ref.artifactID}`}
									value={`${plugin.ref.rootID}:${plugin.ref.artifactID}`}
								>
									{plugin.displayName || plugin.name} · {plugin.ref.rootID}
								</option>
							))}
						</select>
					</ModalField>

					{writablePlugins.length === 0 ? (
						<div className="alert alert-warning rounded-2xl text-sm">
							Create or enable an editable Skill Plugin in Skill management before creating a managed Skill.
						</div>
					) : null}

					<ModalActions>
						<button type="button" className="btn bg-base-300 rounded-xl" onClick={onClose}>
							Cancel
						</button>
						<button
							type="button"
							className="btn btn-primary rounded-xl"
							disabled={!selected}
							onClick={() => {
								if (selected) {
									onChoose(selected, action);
								}
							}}
						>
							Continue
						</button>
					</ModalActions>
				</div>
			</div>
		</ModalDialog>
	);
}

export function SkillsBottomBarChip({
	store,
	shortcut,
	allSkills,
	loading,
	loadError,
	enabledSkillRefs,
	activeSkillRefs,
	setEnabledSkillRefs,
	setActiveSkillRefs,
	onEnableAll,
	onDisableAll,
	onRefreshSkills,
	systemPrompt,
	isInputLocked = false,
}: SkillsBottomBarChipProps) {
	const open = useStoreState(store, 'open');
	const menuContentElement = useStoreState(store, 'contentElement');
	const [searchQuery, setSearchQuery] = useSearchableMenuState(open);
	const [actionError, setActionError] = useState('');
	const [plugins, setPlugins] = useState<PluginListItem[]>([]);
	const [pluginsLoading, setPluginsLoading] = useState(false);
	const [pendingAction, setPendingAction] = useState<PendingManagedSkillAction | null>(null);
	const [editorAction, setEditorAction] = useState<{
		plugin: PluginListItem;
		action: PendingManagedSkillAction;
	} | null>(null);

	useEffect(() => {
		if (isInputLocked) {
			store.hide();
		}
	}, [isInputLocked, store]);

	const instructionSkills = useMemo(
		() =>
			allSkills
				.filter(
					skill =>
						skill.runtimeResolved &&
						skill.insert === SkillInsert.Instructions &&
						skill.enabled &&
						skill.state === ArtifactState.Available
				)
				.toSorted(compareSkills),
		[allSkills]
	);

	const visibleSkills = useMemo(() => {
		if (!isSearchQueryActive(searchQuery)) {
			return instructionSkills;
		}

		return rankSearchableItems(instructionSkills, {
			query: searchQuery,
			getKey: skillKey,
			getFields: skill => [
				{ value: skillLabel(skill), weight: 7 },
				{ value: skill.name, weight: 6 },
				{ value: skill.description, weight: 3 },
				{ value: skill.ref.rootID, weight: 2 },
				{ value: skill.ref.artifactID, weight: 2 },
				{ value: skill.runtimeError, weight: 1 },
			],
			fallbackCompare: compareSkills,
		});
	}, [instructionSkills, searchQuery]);

	const enabledKeys = useMemo(
		() =>
			new Set(
				enabledSkillRefs.map(r => {
					return skillRefKey(r);
				})
			),
		[enabledSkillRefs]
	);
	const activeKeys = useMemo(
		() =>
			new Set(
				activeSkillRefs.map(r => {
					return skillRefKey(r);
				})
			),
		[activeSkillRefs]
	);

	const enabledCount = enabledKeys.size;
	const activeCount = activeKeys.size;
	const selectedSystemSourceCount =
		(systemPrompt.includeModelDefault ? 1 : 0) + systemPrompt.selectedInstructionSourceKeys.length;
	const configuredCount = enabledCount + activeCount + selectedSystemSourceCount;
	const firstVisibleSkill = visibleSkills[0] ?? null;

	const updateEnabled = (item: ComposerSkillCatalogItem, enabled: boolean) => {
		const ref = skillRef(item);
		const key = skillRefKey(ref);

		setEnabledSkillRefs(previous => {
			const next = new Map(previous.map(value => [skillRefKey(value), value] as const));

			if (enabled) {
				next.set(key, ref);
			} else {
				next.delete(key);
			}

			return [...next.values()];
		});

		if (!enabled) {
			setActiveSkillRefs(previous => previous.filter(value => skillRefKey(value) !== key));
		}
	};

	const updateActive = (item: ComposerSkillCatalogItem, active: boolean) => {
		const ref = skillRef(item);
		const key = skillRefKey(ref);

		if (active) {
			setEnabledSkillRefs(previous => {
				const next = new Map(previous.map(value => [skillRefKey(value), value] as const));
				next.set(key, ref);
				return [...next.values()];
			});
			setActiveSkillRefs(previous => {
				const next = new Map(previous.map(value => [skillRefKey(value), value] as const));
				next.set(key, ref);
				return [...next.values()];
			});
			return;
		}

		setActiveSkillRefs(previous => previous.filter(value => skillRefKey(value) !== key));
	};

	const loadWritablePlugins = async () => {
		if (pluginsLoading) {
			return;
		}

		setPluginsLoading(true);
		setActionError('');

		try {
			setPlugins(await skillManagementAPI.listSkillPluginsForManagement());
		} catch (cause) {
			setActionError(cause instanceof Error ? cause.message : 'Skill Plugins could not be loaded.');
		} finally {
			setPluginsLoading(false);
		}
	};

	const openManagedAction = async (action: PendingManagedSkillAction) => {
		await loadWritablePlugins();
		setPendingAction(action);
		store.hide();
	};

	const openFork = async (item: ComposerSkillCatalogItem) => {
		setActionError('');

		let seed = fallbackForkSeed(item, allSkills);

		if (item.managed) {
			try {
				const document = await skillManagementAPI.getManagedSkillDocument(item.ref);
				const sourceDocument: SkillDocumentInput = document.document;
				seed = {
					document: {
						...sourceDocument,
						name: uniqueForkName(sourceDocument.name, allSkills),
						displayName: `${sourceDocument.displayName || sourceDocument.name} Copy`,
					},
					enabled: true,
				};
			} catch (cause) {
				setActionError(
					cause instanceof Error
						? `${cause.message} A blank managed fork draft was prepared instead.`
						: 'The source document could not be loaded. A blank managed fork draft was prepared instead.'
				);
			}
		}

		await openManagedAction({
			mode: 'fork',
			seed,
		});
	};

	const chipTone =
		configuredCount > 0
			? 'border-secondary/50 bg-secondary/10 hover:bg-secondary/15'
			: open
				? 'border-base-300 bg-base-300/60'
				: 'border-transparent';

	const hoverTip = (
		<HoverTipContent
			title={shortcut ? `Instruction Skills (${shortcut})` : 'Instruction Skills'}
			description="Session Runtime owns effective Skill resolution, Skill sessions, lifecycle tools, and active Skill state."
			sections={[
				{
					id: 'current-state',
					title: 'Current state',
					items: [
						`Enabled Skills: ${enabledCount}`,
						`Active Skills: ${activeCount}`,
						`System instruction sources: ${selectedSystemSourceCount}`,
						`Available instruction Skills: ${instructionSkills.length}`,
					],
				},
				{
					id: 'template-behavior',
					title: 'Template behavior',
					items: [
						'User-message Skills are rendered through the Templates menu and are not active Skill-session state.',
					],
				},
			]}
		/>
	);

	return (
		<>
			<div className="relative shrink-0" data-bottom-bar-skills>
				<HoverTip content={hoverTip} placement="top" wrapperElement="div" wrapperClassName="inline-flex max-w-full">
					<div className={`${actionTriggerChipSurfaceClasses} border ${chipTone} ${isInputLocked ? 'opacity-60' : ''}`}>
						<MenuButton
							store={store}
							className="btn btn-xs app-text-neutral h-auto min-h-0 flex-1 gap-0 border-none bg-transparent p-0 text-left font-normal shadow-none hover:bg-transparent"
							aria-label={shortcut ? `Manage instruction Skills (${shortcut})` : 'Manage instruction Skills'}
							disabled={isInputLocked}
						>
							<ActionTriggerChipContent
								icon={<FiFilePlus size={14} />}
								label="Skills"
								count={
									enabledCount > 0 ? (
										<span className="badge badge-success badge-xs">Enabled {enabledCount}</span>
									) : undefined
								}
								suffix={
									activeCount > 0 || selectedSystemSourceCount > 0 ? (
										<span className="flex shrink-0 items-center gap-1">
											{activeCount > 0 ? <span className="badge badge-info badge-xs">Active {activeCount}</span> : null}
											{selectedSystemSourceCount > 0 ? (
												<span className="badge badge-secondary badge-xs">System {selectedSystemSourceCount}</span>
											) : null}
										</span>
									) : undefined
								}
								open={open}
							/>
						</MenuButton>

						{configuredCount > 0 ? (
							<button
								type="button"
								className={actionTriggerChipClearButtonClasses}
								onClick={event => {
									event.preventDefault();
									event.stopPropagation();
									onDisableAll();
									systemPrompt.clearAllSystemPromptState();
									store.hide();
								}}
								disabled={isInputLocked}
								aria-label="Clear Skills and system instruction sources"
								title="Clear Skills and system instruction sources"
							>
								<FiX size={12} />
							</button>
						) : null}
					</div>
				</HoverTip>

				<Menu
					store={store}
					gutter={8}
					overflowPadding={8}
					className={actionTriggerMenuWideClasses}
					autoFocusOnShow={false}
					portal
				>
					<div className="mb-2 flex items-center justify-between gap-2 px-1">
						<div className="text-base-content/70 text-xs font-semibold">Instruction Skills</div>
						<div className="flex items-center gap-1">
							<button
								type="button"
								className="btn btn-ghost btn-xs rounded-lg"
								disabled={isInputLocked || pluginsLoading}
								onClick={() => {
									void openManagedAction({ mode: 'create' });
								}}
								title="Create managed instruction Skill"
							>
								<FiPlus size={12} />
								Create
							</button>
							<button
								type="button"
								className="btn btn-ghost btn-xs rounded-lg"
								disabled={isInputLocked || loading}
								onClick={() => {
									void onRefreshSkills();
								}}
								title="Refresh Skill catalog"
							>
								<FiRefreshCw size={12} />
								Refresh
							</button>
						</div>
					</div>

					{loadError ? (
						<div className="alert alert-warning mb-2 rounded-xl text-xs">
							<FiAlertCircle size={14} />
							<span>{loadError}</span>
						</div>
					) : null}

					{actionError ? (
						<div className="alert alert-warning mb-2 rounded-xl text-xs">
							<FiAlertCircle size={14} />
							<span>{actionError}</span>
						</div>
					) : null}

					<SearchableMenuInput
						open={open}
						query={searchQuery}
						onQueryChange={setSearchQuery}
						placeholder="Search instruction Skills…"
						resultCount={visibleSkills.length}
						totalCount={instructionSkills.length}
						disabled={loading || instructionSkills.length === 0}
						onFocusFirstItem={() => {
							focusFirstSearchableMenuItem(menuContentElement);
						}}
						onEnterFirstResult={() => {
							if (firstVisibleSkill && !isInputLocked) {
								updateEnabled(firstVisibleSkill, !enabledKeys.has(skillKey(firstVisibleSkill)));
							}
						}}
						onEscape={() => {
							store.hide();
						}}
					/>

					<div className="border-base-300 bg-base-100 mb-2 rounded-xl border p-2">
						<div className="mb-2 flex items-center justify-between gap-2">
							<div className="text-xs font-semibold">System instruction sources</div>
							<button
								type="button"
								className="btn btn-ghost btn-xs rounded-lg"
								disabled={isInputLocked || selectedSystemSourceCount === 0}
								onClick={() => {
									systemPrompt.clearInstructionSources();
								}}
							>
								Clear selected
							</button>
						</div>

						{systemPrompt.modelDefaultPrompt.trim() ? (
							<label className="hover:bg-base-200 flex cursor-pointer items-center justify-between gap-2 rounded-lg px-2 py-1">
								<span className="text-xs">Model default instructions</span>
								<input
									type="checkbox"
									className="checkbox checkbox-xs"
									checked={systemPrompt.includeModelDefault}
									disabled={isInputLocked}
									onChange={event => {
										systemPrompt.setIncludeModelDefault(event.currentTarget.checked);
									}}
								/>
							</label>
						) : null}

						{systemPrompt.instructionSources.map(source => (
							<label
								key={source.identityKey}
								className="hover:bg-base-200 flex cursor-pointer items-center justify-between gap-2 rounded-lg px-2 py-1"
							>
								<span className="min-w-0 truncate text-xs">{source.displayName}</span>
								<input
									type="checkbox"
									className="checkbox checkbox-xs"
									checked={systemPrompt.selectedInstructionSourceKeys.includes(source.identityKey)}
									disabled={isInputLocked}
									onChange={() => {
										systemPrompt.toggleInstructionSource(source.identityKey);
									}}
								/>
							</label>
						))}

						{!systemPrompt.modelDefaultPrompt.trim() && systemPrompt.instructionSources.length === 0 ? (
							<div className="text-base-content/60 px-2 py-1 text-xs">
								No flattened Session instruction source is selected.
							</div>
						) : null}
					</div>

					<div className="mb-2 flex flex-wrap items-center justify-between gap-2 px-1">
						<div className="text-base-content/60 text-xs">
							{instructionSkills.length} available · {enabledCount} enabled · {activeCount} active
						</div>
						<div className="flex gap-1">
							<button
								type="button"
								className="btn btn-ghost btn-xs rounded-lg"
								disabled={isInputLocked || instructionSkills.length === 0}
								onClick={onEnableAll}
							>
								Enable all
							</button>
							<button
								type="button"
								className="btn btn-ghost btn-xs text-error rounded-lg"
								disabled={isInputLocked || enabledCount === 0}
								onClick={onDisableAll}
							>
								Disable all
							</button>
						</div>
					</div>

					{loading ? (
						<div className={`${actionTriggerMenuItemClasses} text-base-content/60 cursor-default`}>Loading Skills…</div>
					) : instructionSkills.length === 0 ? (
						<div className={`${actionTriggerMenuItemClasses} text-base-content/60 cursor-default`}>
							No runtime-resolved instruction Skills are available.
						</div>
					) : visibleSkills.length === 0 ? (
						<div className={searchableMenuEmptyStateClasses}>No Skills match your search.</div>
					) : (
						<div className="space-y-1">
							{visibleSkills.map(skill => {
								const key = skillKey(skill);
								const enabled = enabledKeys.has(key);
								const active = activeKeys.has(key);
								const activationAllowed = canActivate(skill);

								return (
									<MenuItem
										key={key}
										store={store}
										hideOnClick={false}
										disabled={isInputLocked}
										data-searchable-menu-item="true"
										className={`data-active-item:bg-base-200 flex flex-col items-start rounded-xl border p-2 ${
											enabled ? 'border-secondary/50 bg-secondary/10' : 'border-base-300'
										}`}
										onClick={() => {
											updateEnabled(skill, !enabled);
										}}
									>
										<div className="flex w-full items-start gap-2">
											<FiZap size={14} className="mt-0.5 shrink-0" />
											<div className="min-w-0 flex-1">
												<div className="truncate text-xs font-medium">{skillLabel(skill)}</div>
												<div className="text-base-content/60 truncate text-[10px]">
													{skill.name} · {skill.ref.rootID}
												</div>
												{skill.description ? (
													<div className="text-base-content/70 mt-1 line-clamp-2 text-xs">{skill.description}</div>
												) : null}
											</div>
											{enabled ? <FiCheck size={14} className="text-secondary shrink-0" /> : null}
										</div>

										<div className="mt-2 ml-6 flex flex-wrap items-center gap-1">
											{skill.hasArguments ? <span className="badge badge-warning badge-xs">Arguments</span> : null}
											{skill.hasResources ? <span className="badge badge-info badge-xs">Resources</span> : null}
											{skill.builtIn ? <span className="badge badge-ghost badge-xs">Built-in</span> : null}
											{active ? <span className="badge badge-success badge-xs">Active</span> : null}
										</div>

										<div className="mt-2 ml-6 flex w-[calc(100%-1.5rem)] justify-end gap-1">
											{enabled ? (
												<>
													<button
														type="button"
														className="btn btn-ghost btn-xs rounded-lg"
														disabled={isInputLocked || !activationAllowed}
														title={
															activationAllowed
																? active
																	? 'Deactivate this Skill for the next Run.'
																	: 'Activate this Skill for the next Run.'
																: 'Argument-backed Skills can be enabled but cannot be preloaded as active.'
														}
														onClick={() => {
															updateActive(skill, !active);
														}}
													>
														{active ? 'Deactivate' : 'Activate'}
													</button>
													<button
														type="button"
														className="btn btn-ghost btn-xs text-error rounded-lg"
														disabled={isInputLocked}
														onClick={() => {
															updateEnabled(skill, false);
														}}
													>
														Disable
													</button>
												</>
											) : (
												<>
													<button
														type="button"
														className="btn btn-ghost btn-xs rounded-lg"
														disabled={isInputLocked}
														onClick={() => {
															updateEnabled(skill, true);
														}}
													>
														Enable
													</button>
													<button
														type="button"
														className="btn btn-ghost btn-xs rounded-lg"
														disabled={isInputLocked}
														onClick={() => {
															void openFork(skill);
														}}
													>
														<FiGitBranch size={12} />
														Fork
													</button>
												</>
											)}
										</div>
									</MenuItem>
								);
							})}
						</div>
					)}
				</Menu>
			</div>

			<ComposerSkillDestinationModal
				isOpen={pendingAction !== null}
				plugins={plugins}
				action={pendingAction}
				onClose={() => {
					setPendingAction(null);
				}}
				onChoose={(plugin, action) => {
					setPendingAction(null);
					setEditorAction({ plugin, action });
				}}
			/>

			<ManagedSkillEditorModal
				isOpen={editorAction !== null}
				plugin={
					editorAction?.plugin ?? {
						ref: { rootID: '', artifactID: '' },
						sourceID: '',
						name: '',
						displayName: '',
						state: ArtifactState.Available,
						enabled: false,
						revision: 0,
						memberCount: 0,
						builtIn: false,
						editable: false,
						deletable: false,
						baseline: false,
					}
				}
				mode={editorAction?.action.mode}
				seed={editorAction?.action.seed}
				onClose={() => {
					setEditorAction(null);
				}}
				onSaved={async artifact => {
					setEditorAction(null);
					await onRefreshSkills();
					setEnabledSkillRefs(previous => {
						const next = new Map(previous.map(value => [skillRefKey(value), value] as const));
						next.set(skillRefKey(artifact), artifact);
						return [...next.values()];
					});
				}}
			/>
		</>
	);
}
