import type { Dispatch, SetStateAction, SyntheticEvent } from 'react';
import { useEffect, useMemo, useState } from 'react';
import { FiAlertCircle, FiCheck, FiFilePlus, FiGitBranch, FiPlus, FiRefreshCw, FiX, FiZap } from 'react-icons/fi';

import type { MenuStore } from '@ariakit/react';
import { Menu, MenuButton, MenuItem, useStoreState } from '@ariakit/react';

import type { PluginListItem } from '@/spec/plugin';
import type { SkillRef } from '@/spec/skill';
import { ArtifactState } from '@/spec/artifact';
import { SkillInsert } from '@/spec/skill';

import { getDistinctUserFacingLabel, getPrimaryUserFacingLabel } from '@/lib/user_facing_presentation';

import type { ComposerSkillCatalogItem } from '@/apis/skill_management';
import { skillManagementAPI } from '@/apis/baseapi';
import { canWriteSkillPlugin, skillRefKey } from '@/apis/skill_management';

import type { DropdownItem } from '@/components/dropdown';
import {
	ActionTriggerChipContent,
	actionTriggerChipSurfaceClasses,
	actionTriggerMenuItemClasses,
	actionTriggerMenuWideClasses,
} from '@/components/action_trigger_chip';
import { ComposerDropdownSelectedPanel } from '@/components/composer_dropdown_selected_panel';
import { Dropdown } from '@/components/dropdown';
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
import { ManagedSkillEditorModal } from '@/skills/skill_add_edit_modal';
import { makeUniqueSkillName } from '@/skills/skill_management_utils';

function stopSkillRowAction(event: SyntheticEvent) {
	event.stopPropagation();
}

function selectable(item: ComposerSkillCatalogItem): boolean {
	return (
		item.runtimeResolved &&
		item.enabled &&
		item.state === ArtifactState.Available &&
		item.insert === SkillInsert.Instructions
	);
}

function addSkillRef(refs: SkillRef[], ref: SkillRef): SkillRef[] {
	const key = skillRefKey(ref);
	return [...refs.filter(value => skillRefKey(value) !== key), ref];
}

function removeSkillRef(refs: SkillRef[], ref: SkillRef): SkillRef[] {
	const key = skillRefKey(ref);
	return refs.filter(value => skillRefKey(value) !== key);
}

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

function ComposerSkillDestinationModal({
	isOpen,
	plugins,
	action,
	busy,
	onClose,
	onChoose,
}: {
	isOpen: boolean;
	plugins: PluginListItem[];
	action: PendingManagedSkillAction | null;
	busy: boolean;
	onClose: () => void;
	onChoose: (plugin: PluginListItem, action: PendingManagedSkillAction) => void;
}) {
	const [selectedKey, setSelectedKey] = useState('');

	if (!isOpen || !action) {
		return null;
	}

	const writablePlugins = plugins.filter(p => {
		return canWriteSkillPlugin(p);
	});
	const writablePluginDropdownItems = Object.fromEntries(
		writablePlugins.map(plugin => [`${plugin.ref.rootID}:${plugin.ref.artifactID}`, { isEnabled: true }])
	) as Record<string, DropdownItem>;
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

					<ModalField label="Skill Plugin" required>
						<Dropdown<string>
							dropdownItems={writablePluginDropdownItems}
							orderedKeys={writablePlugins.map(plugin => `${plugin.ref.rootID}:${plugin.ref.artifactID}`)}
							selectedKey={selected ? selectedKey : ''}
							onChange={setSelectedKey}
							filterDisabled={false}
							title="Select a Skill Plugin"
							placeholderLabel="Select a Skill Plugin"
							getDisplayName={key => {
								const plugin = writablePlugins.find(
									candidate => `${candidate.ref.rootID}:${candidate.ref.artifactID}` === key
								);
								return plugin ? getPrimaryUserFacingLabel(plugin.displayName, plugin.name) : 'Skill Plugin';
							}}
							disabled={busy || writablePlugins.length === 0}
							inlineMenu
							maxMenuHeight={240}
						/>
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
							disabled={!selected || busy}
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
		existingNames: string[];
	} | null>(null);

	useEffect(() => {
		if (isInputLocked) {
			store.hide();
		}
	}, [isInputLocked, store]);

	const instructionSkills = useMemo(
		() => allSkills.filter(skill => skill.insert !== SkillInsert.UserMessage).toSorted(compareSkills),
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
				{ value: skill.location, weight: 2 },
				...(skill.tags ?? []).map(tag => ({ value: tag, weight: 2 })),
				...(skill.arguments ?? []).map(argument => ({ value: argument.name, weight: 1 })),
				...(skill.runtimeWarnings ?? []).map(warning => ({ value: warning, weight: 1 })),
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

	const selectedSourceKeys = useMemo(
		() => new Set(systemPrompt.selectedInstructionSourceKeys),
		[systemPrompt.selectedInstructionSourceKeys]
	);
	const selectedInstructionSources = useMemo(
		() => systemPrompt.instructionSources.filter(source => selectedSourceKeys.has(source.identityKey)),
		[selectedSourceKeys, systemPrompt.instructionSources]
	);
	const availableInstructionSources = useMemo(
		() => systemPrompt.instructionSources.filter(source => !selectedSourceKeys.has(source.identityKey)),
		[selectedSourceKeys, systemPrompt.instructionSources]
	);
	const selectedInstructionSkillKeys = useMemo(
		() =>
			new Set(selectedInstructionSources.flatMap(source => (source.skillRef ? [skillRefKey(source.skillRef)] : []))),
		[selectedInstructionSources]
	);
	const availableSkills = useMemo(
		() =>
			visibleSkills.filter(
				skill => !enabledKeys.has(skillKey(skill)) && !selectedInstructionSkillKeys.has(skillKey(skill))
			),
		[enabledKeys, selectedInstructionSkillKeys, visibleSkills]
	);
	const instructionSkillByKey = useMemo(
		() => new Map(instructionSkills.map(skill => [skillKey(skill), skill] as const)),
		[instructionSkills]
	);
	const selectedSessionSkills = useMemo(
		() =>
			enabledSkillRefs.map(ref => ({
				ref,
				skill: instructionSkillByKey.get(skillRefKey(ref)),
			})),
		[enabledSkillRefs, instructionSkillByKey]
	);

	const enabledCount = enabledKeys.size;
	const activeCount = activeKeys.size;

	const selectedSystemSourceCount =
		(systemPrompt.includeModelDefault && systemPrompt.modelDefaultPrompt.trim() ? 1 : 0) +
		selectedInstructionSources.filter(source => Boolean(source.text.trim())).length;
	const configuredCount = enabledCount + selectedSystemSourceCount;
	const firstVisibleSkill =
		availableSkills.find(s => {
			return selectable(s);
		}) ?? null;

	const updateEnabled = (item: ComposerSkillCatalogItem, enabled: boolean) => {
		if (enabled && !selectable(item)) {
			return;
		}

		const ref = skillRef(item);

		setEnabledSkillRefs(previous => {
			if (enabled) {
				return addSkillRef(previous, ref);
			}
			return removeSkillRef(previous, ref);
		});

		if (!enabled) {
			setActiveSkillRefs(previous => removeSkillRef(previous, ref));
		}
	};

	const updateActive = (item: ComposerSkillCatalogItem, active: boolean) => {
		if (active && (!selectable(item) || item.hasArguments)) {
			return;
		}

		const ref = skillRef(item);

		if (active) {
			setEnabledSkillRefs(previous => addSkillRef(previous, ref));
			setActiveSkillRefs(previous => addSkillRef(previous, ref));
			return;
		}

		setActiveSkillRefs(previous => removeSkillRef(previous, ref));
	};

	const loadWritablePlugins = async () => {
		if (pluginsLoading) {
			return false;
		}

		setPluginsLoading(true);
		setActionError('');

		try {
			setPlugins(await skillManagementAPI.listSkillPluginsForManagement());
			return true;
		} catch (cause) {
			setActionError(cause instanceof Error ? cause.message : 'Skill Plugins could not be loaded.');
			return false;
		} finally {
			setPluginsLoading(false);
		}
	};

	const openManagedAction = async (action: PendingManagedSkillAction) => {
		if (!(await loadWritablePlugins())) {
			return;
		}

		setPendingAction(action);
		store.hide();
	};

	const openFork = async (item: ComposerSkillCatalogItem) => {
		setActionError('');
		try {
			const document = await skillManagementAPI.copySkillDocument({
				ref: item.ref,
				name: item.name,
				displayName: item.displayName,
				description: item.description,
				state: item.state,
				enabled: item.enabled,
				revision: item.revision,
				definitionDigest: item.definitionDigest,
				builtIn: item.builtIn,
				managed: item.managed,
			});
			await openManagedAction({
				mode: 'fork',
				seed: { document, enabled: true },
			});
		} catch (cause) {
			setActionError(cause instanceof Error ? cause.message : 'Source document could not be loaded.');
			store.show();
		}
	};

	const removeFromSkillSession = (item: ComposerSkillCatalogItem) => {
		const ref = skillRef(item);
		setEnabledSkillRefs(previous => removeSkillRef(previous, ref));
		setActiveSkillRefs(previous => removeSkillRef(previous, ref));
	};

	const addAsInstructions = async (item: ComposerSkillCatalogItem) => {
		const ref = skillRef(item);
		const identity = `skill-instructions:${skillRefKey(ref)}`;
		const existingSource = systemPrompt.instructionSources.find(source => source.identityKey === identity);

		if (existingSource) {
			removeFromSkillSession(item);
			if (!systemPrompt.selectedInstructionSourceKeys.includes(identity)) {
				systemPrompt.toggleInstructionSource(identity);
			}
			return;
		}

		try {
			const source = await skillManagementAPI.prepareInstructionSource({
				ref: item.ref,
				name: item.name,
				displayName: item.displayName,
				description: item.description,
				state: item.state,
				enabled: item.enabled,
				revision: item.revision,
				definitionDigest: item.definitionDigest,
				builtIn: item.builtIn,
				managed: item.managed,
			});

			// This mode is deliberately text-only. Moving a Skill into the
			// system-instruction channel must remove it from Skill-session state.
			removeFromSkillSession(item);

			systemPrompt.addAndSelectInstructionSkillSource({
				displayName: skillLabel(item),
				prompt: source.text,
				skillRef: item.ref,
				sourceTags: source.tags,
			});
		} catch (cause) {
			setActionError(cause instanceof Error ? cause.message : 'Instructions could not be copied.');
		}
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

	const selectedSkillsPanel = (
		<>
			{selectedSystemSourceCount > 0 ? (
				<div className="border-secondary/30 bg-secondary/10 rounded-xl border p-2">
					<div className="mb-2 flex items-center justify-between gap-2">
						<span className="text-xs font-semibold">System instruction sources</span>
						<button
							type="button"
							className="btn btn-ghost btn-xs text-error rounded-lg"
							disabled={isInputLocked}
							onClick={() => {
								systemPrompt.clearInstructionSources();
							}}
						>
							<FiX size={12} />
							Clear
						</button>
					</div>

					{systemPrompt.includeModelDefault && systemPrompt.modelDefaultPrompt.trim() ? (
						<div className="bg-base-100 flex items-center gap-2 rounded-lg px-2 py-1 text-xs">
							<span className="min-w-0 flex-1 truncate">Model default instructions</span>
							<button
								type="button"
								className="btn btn-ghost btn-xs text-error rounded-lg"
								disabled={isInputLocked}
								onClick={() => {
									systemPrompt.setIncludeModelDefault(false);
								}}
							>
								Remove
							</button>
						</div>
					) : null}

					{selectedInstructionSources.map(source => (
						<div
							key={source.identityKey}
							className="bg-base-100 mt-1 flex min-w-0 items-center gap-2 rounded-lg px-2 py-1"
						>
							<span className="min-w-0 flex-1 truncate text-xs">{source.displayName}</span>
							{source.sourceKind === 'agent' ? <span className="badge badge-secondary badge-xs">Agent</span> : null}
							<button
								type="button"
								className="btn btn-ghost btn-xs text-error rounded-lg"
								disabled={isInputLocked}
								onClick={() => {
									systemPrompt.toggleInstructionSource(source.identityKey);
								}}
							>
								Remove
							</button>
						</div>
					))}
				</div>
			) : null}

			{enabledCount > 0 ? (
				<div className="border-base-300 bg-base-100 rounded-xl border p-2">
					<div className="mb-2 flex items-center justify-between gap-2">
						<span className="text-xs font-semibold">Enabled Skill sessions</span>
						<button
							type="button"
							className="btn btn-ghost btn-xs text-error rounded-lg"
							disabled={isInputLocked}
							onClick={onDisableAll}
						>
							<FiX size={12} />
							Disable all
						</button>
					</div>

					{selectedSessionSkills.map(({ ref, skill }) => {
						const key = skillRefKey(ref);
						const active = activeKeys.has(key);
						const label = skill ? skillLabel(skill) : 'Unavailable saved Skill';
						const secondaryName = skill ? getDistinctUserFacingLabel(label, skill.name) : undefined;
						const canMoveToInstructions = skill && selectable(skill) && !skill.hasArguments && !skill.hasResources;

						return (
							<div key={key} className="bg-base-200 mt-1 flex min-w-0 flex-wrap items-center gap-2 rounded-lg p-2">
								<FiZap size={13} className="shrink-0" />
								<div className="min-w-0 flex-1">
									<div className="truncate text-xs font-medium">{label}</div>
									{secondaryName ? (
										<div className="text-base-content/60 truncate text-[10px]">{secondaryName}</div>
									) : null}
								</div>
								{active ? <span className="badge badge-success badge-xs">Active</span> : null}
								{!skill ? <span className="badge badge-warning badge-xs">Unavailable</span> : null}
								{skill ? (
									<button
										type="button"
										className="btn btn-ghost btn-xs rounded-lg"
										disabled={isInputLocked || (!active && !canActivate(skill))}
										onClick={() => {
											updateActive(skill, !active);
										}}
									>
										{active ? 'Deactivate' : 'Activate'}
									</button>
								) : null}
								{canMoveToInstructions && skill ? (
									<button
										type="button"
										className="btn btn-ghost btn-xs rounded-lg"
										disabled={isInputLocked}
										onClick={() => {
											void addAsInstructions(skill);
										}}
									>
										Use only as instructions
									</button>
								) : null}
								<button
									type="button"
									className="btn btn-ghost btn-xs text-error rounded-lg"
									disabled={isInputLocked}
									onClick={() => {
										setEnabledSkillRefs(previous => removeSkillRef(previous, ref));
										setActiveSkillRefs(previous => removeSkillRef(previous, ref));
									}}
								>
									Disable
								</button>
							</div>
						);
					})}
				</div>
			) : null}
		</>
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
									configuredCount > 0 ? (
										<span className="flex shrink-0 items-center gap-1">
											<FiCheck size={14} aria-hidden="true" />
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
									void openManagedAction({
										mode: 'create',
										seed: {
											document: {
												name: 'instruction-skill',
												displayName: 'Instruction Skill',
												description: 'Instruction Skill created from the composer.',
												insert: SkillInsert.Instructions,
												arguments: [],
												tags: ['composer'],
												markdownBody: '',
											},
											enabled: true,
										},
									});
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
						resultCount={availableSkills.length}
						totalCount={instructionSkills.length}
						disabled={loading || instructionSkills.length === 0}
						onFocusFirstItem={() => {
							focusFirstSearchableMenuItem(menuContentElement);
						}}
						onEnterFirstResult={() => {
							if (firstVisibleSkill && !isInputLocked && selectable(firstVisibleSkill)) {
								updateEnabled(firstVisibleSkill, !enabledKeys.has(skillKey(firstVisibleSkill)));
							}
						}}
						onEscape={() => {
							store.hide();
						}}
					/>

					<div className="border-base-300 bg-base-100 mb-2 rounded-xl border p-2">
						<div className="mb-2 flex items-center justify-between gap-2">
							<div className="text-xs font-semibold">Available system instruction sources</div>
						</div>

						{systemPrompt.modelDefaultPrompt.trim() && !systemPrompt.includeModelDefault ? (
							<button
								type="button"
								className="hover:bg-base-200 flex w-full items-center justify-between gap-2 rounded-lg px-2 py-1 text-left"
								disabled={isInputLocked}
								onClick={() => {
									systemPrompt.setIncludeModelDefault(true);
								}}
							>
								<span className="text-xs">Use model default instructions</span>
								<FiPlus size={12} />
							</button>
						) : null}

						{availableInstructionSources.map(source => (
							<button
								key={source.identityKey}
								type="button"
								className="hover:bg-base-200 flex w-full items-center justify-between gap-2 rounded-lg px-2 py-1 text-left"
								disabled={isInputLocked}
								onClick={() => {
									systemPrompt.toggleInstructionSource(source.identityKey);
								}}
							>
								<span className="min-w-0 text-xs">
									<span className="flex min-w-0 items-center gap-1">
										<span className="truncate">{source.displayName}</span>
										{source.sourceKind === 'agent' ? (
											<span className="badge badge-secondary badge-xs">Agent</span>
										) : null}
									</span>
									{source.sourceTags?.length ? (
										<span className="text-base-content/60 block text-[10px]">{source.sourceTags.join(', ')}</span>
									) : null}
								</span>
								<FiPlus size={12} className="shrink-0" />
							</button>
						))}

						{!systemPrompt.modelDefaultPrompt.trim() && availableInstructionSources.length === 0 ? (
							<div className="text-base-content/60 px-2 py-1 text-xs">
								All available instruction sources are already added below.
							</div>
						) : null}
					</div>

					<div className="mb-2 flex flex-wrap items-center justify-between gap-2 px-1">
						<div className="text-base-content/60 text-xs">
							{
								instructionSkills.filter(s => {
									return selectable(s);
								}).length
							}{' '}
							available · {enabledCount} enabled · {activeCount} active
						</div>
						<div className="flex gap-1">
							<button
								type="button"
								className="btn btn-ghost btn-xs rounded-lg"
								disabled={
									isInputLocked ||
									instructionSkills.filter(s => {
										return selectable(s);
									}).length === 0
								}
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
							{availableSkills.map(skill => {
								const key = skillKey(skill);
								const enabled = enabledKeys.has(key);
								const active = activeKeys.has(key);
								const activationAllowed = canActivate(skill);
								const secondaryName = getDistinctUserFacingLabel(skillLabel(skill), skill.name);

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
											if (!selectable(skill) && !enabled) {
												return;
											}
											updateEnabled(skill, !enabled);
										}}
									>
										<div className="flex w-full items-start gap-2">
											<FiZap size={14} className="mt-0.5 shrink-0" />
											<div className="min-w-0 flex-1">
												<div className="truncate text-xs font-medium">{skillLabel(skill)}</div>
												{secondaryName ? (
													<div className="text-base-content/60 truncate text-[10px]">{secondaryName}</div>
												) : null}
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
											{!selectable(skill) ? (
												<span
													className="badge badge-warning badge-xs"
													title={skill.runtimeError || skill.metadataError}
												>
													Unavailable
												</span>
											) : null}
										</div>
										{/* oxlint-disable-next-line jsx-a11y/click-events-have-key-events,jsx-a11y/no-static-element-interactions */}
										<div
											className="mt-2 ml-6 flex w-[calc(100%-1.5rem)] flex-wrap justify-end gap-1"
											onClick={stopSkillRowAction}
											onPointerDown={stopSkillRowAction}
										>
											{enabled ? (
												<>
													<button
														type="button"
														className="btn btn-ghost btn-xs rounded-lg"
														disabled={isInputLocked || (!active && (!activationAllowed || !selectable(skill)))}
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
														disabled={isInputLocked || !selectable(skill)}
														onClick={() => {
															updateEnabled(skill, true);
														}}
													>
														Enable
													</button>
													<button
														type="button"
														className="btn btn-ghost btn-xs rounded-lg"
														disabled={isInputLocked || !selectable(skill) || !activationAllowed}
														onClick={() => {
															updateActive(skill, true);
														}}
													>
														Enable + activate
													</button>
												</>
											)}
											<button
												type="button"
												className="btn btn-ghost btn-xs rounded-lg"
												disabled={isInputLocked || !skill.document}
												title="Copy the verified document; resource files are not copied."
												onClick={() => {
													void openFork(skill);
												}}
											>
												<FiGitBranch size={12} /> Fork
											</button>
											<button
												type="button"
												className="btn btn-ghost btn-xs rounded-lg"
												disabled={isInputLocked || !selectable(skill) || skill.hasArguments || skill.hasResources}
												title="Render this simple Skill as an explicit instruction source. This does not enable its lifecycle."
												onClick={() => {
													void addAsInstructions(skill);
												}}
											>
												{systemPrompt.selectedInstructionSourceKeys.includes(
													`skill-instructions:${skillRefKey(skill.ref)}`
												)
													? 'Remove from instructions'
													: 'Add as instructions'}
											</button>
										</div>
									</MenuItem>
								);
							})}
						</div>
					)}

					<ComposerDropdownSelectedPanel label="Added Skills and instructions" count={configuredCount}>
						{selectedSkillsPanel}
					</ComposerDropdownSelectedPanel>
				</Menu>
			</div>

			<ComposerSkillDestinationModal
				isOpen={pendingAction !== null}
				plugins={plugins}
				action={pendingAction}
				busy={pluginsLoading}
				onClose={() => {
					setPendingAction(null);
				}}
				onChoose={(plugin, action) => {
					if (pluginsLoading) {
						return;
					}
					setPluginsLoading(true);
					void skillManagementAPI
						.listSkills(plugin.ref.rootID)
						.then(skills => {
							const existingNames = skills.map(skill => skill.name);
							const selectedAction =
								action.mode === 'create' && action.seed
									? {
											...action,
											seed: {
												...action.seed,
												document: {
													...action.seed.document,
													name: makeUniqueSkillName(action.seed.document.name, existingNames),
												},
											},
										}
									: action;
							setPendingAction(null);
							setEditorAction({ plugin, action: selectedAction, existingNames });
						})
						.catch((cause: unknown) => {
							setActionError(cause instanceof Error ? cause.message : 'Destination could not be inspected.');
							setPendingAction(null);
							store.show();
						})
						.finally(() => {
							setPluginsLoading(false);
						});
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
				existingNames={editorAction?.existingNames}
				onClose={() => {
					setEditorAction(null);
				}}
				onSaved={async (artifact, document, enabled) => {
					setEditorAction(null);
					if (enabled && document.insert === SkillInsert.Instructions) {
						if ((document.arguments?.length ?? 0) === 0) {
							// This is the user's newly saved document copy, not trusted
							// effective execution content or an adopted Agent baseline.
							systemPrompt.addAndSelectInstructionSkillSource({
								displayName: document.displayName || document.name,
								prompt: document.markdownBody,
								skillRef: artifact,
								sourceTags: document.tags,
							});
						} else {
							setEnabledSkillRefs(previous => addSkillRef(previous, artifact));
						}
					}
					try {
						await onRefreshSkills();
					} catch (cause) {
						setActionError(
							`Skill saved, but catalog refresh failed: ${cause instanceof Error ? cause.message : 'unknown error'}`
						);
						store.show();
					}
				}}
			/>
		</>
	);
}
