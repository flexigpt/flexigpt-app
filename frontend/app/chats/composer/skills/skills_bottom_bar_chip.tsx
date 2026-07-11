import type { Dispatch, SetStateAction, SubmitEventHandler, SyntheticEvent } from 'react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle, FiCheck, FiFilePlus, FiGitBranch, FiPlus, FiX } from 'react-icons/fi';

import type { MenuStore } from '@ariakit/react';
import { Menu, MenuButton, MenuItem, useMenuStore, useStoreState } from '@ariakit/react';

import type { SkillBundle, SkillListItem, SkillRef } from '@/spec/skill';

import { skillStoreAPI } from '@/apis/baseapi';
import { getAllSkillBundles } from '@/apis/list_helper';

import {
	ActionTriggerChipContent,
	actionTriggerChipSurfaceClasses,
	actionTriggerMenuItemClasses,
	actionTriggerMenuWideClasses,
} from '@/components/action_trigger_chip';
import { HoverTip } from '@/components/ariakit_hover_tip';
import { Dropdown } from '@/components/dropdown';
import { searchableMenuEmptyStateClasses, SearchableMenuInput } from '@/components/searchmenu/searchable_menu';
import {
	focusFirstSearchableMenuItem,
	isSearchQueryActive,
	rankSearchableItems,
	useSearchableMenuState,
} from '@/components/searchmenu/searchable_menu_utils';

import { buildSkillRefKey } from '@/assistantpresets/lib/assistant_preset_utils';
import type { ComposerSystemPromptController } from '@/chats/composer/skills/use_composer_system_prompt';
import {
	getSkillInstructionPromptEligibilityReason,
	isInstructionInsertSkill,
	skillCanBePreloadedAsActive,
	skillCanBeRenderedAsInstructionPrompt,
} from '@/skills/lib/skill_artifact_utils';
import { dedupeSkillRefs, skillRefFromListItem, skillRefKey } from '@/skills/lib/skill_identity_utils';

const SIMPLE_SKILL_NAME_RE = /^[a-z0-9][a-z0-9-]{0,63}$/;

interface InstructionSkillDraft {
	bundleID?: string;
	displayName: string;
	name: string;
	body: string;
	description?: string;
}

const DEFAULT_INSTRUCTION_SKILL_DRAFT: InstructionSkillDraft = {
	displayName: 'Instruction Skill',
	name: 'instruction-skill',
	body: '',
};

const skillDropdownCollator = new Intl.Collator(undefined, {
	numeric: true,
	sensitivity: 'base',
});

function stop(e: SyntheticEvent) {
	e.preventDefault();
	e.stopPropagation();
}

function skillListItemKey(item: SkillListItem): string {
	return skillRefKey(skillRefFromListItem(item));
}

function getSkillDisplayLabel(item: SkillListItem): string {
	return item.skillDefinition.displayName?.trim() || item.skillDefinition.name?.trim() || item.skillSlug;
}

function getInstructionSourceIdentityKey(item: SkillListItem): string {
	return `skill-instructions:${buildSkillRefKey(skillRefFromListItem(item))}`;
}

function compareSkillListItems(a: SkillListItem, b: SkillListItem): number {
	const bundleSlugCompare = skillDropdownCollator.compare(a.bundleSlug, b.bundleSlug);
	if (bundleSlugCompare !== 0) {
		return bundleSlugCompare;
	}

	const bundleIDCompare = skillDropdownCollator.compare(a.bundleID, b.bundleID);
	if (bundleIDCompare !== 0) {
		return bundleIDCompare;
	}

	const skillSlugCompare = skillDropdownCollator.compare(a.skillSlug, b.skillSlug);
	if (skillSlugCompare !== 0) {
		return skillSlugCompare;
	}

	const skillNameCompare = skillDropdownCollator.compare(a.skillDefinition.name, b.skillDefinition.name);
	if (skillNameCompare !== 0) {
		return skillNameCompare;
	}

	return skillDropdownCollator.compare(skillListItemKey(a), skillListItemKey(b));
}

function getSkillSearchFields(item: SkillListItem) {
	return [
		{ value: getSkillDisplayLabel(item), weight: 6 },
		{ value: item.skillSlug, weight: 5 },
		{ value: item.skillDefinition.name, weight: 4 },
		{ value: item.bundleSlug, weight: 3 },
		{ value: item.bundleID, weight: 2 },
		{ value: item.skillDefinition.description, weight: 2 },
		{ value: item.skillDefinition.type, weight: 1 },
		{ value: item.skillDefinition.location, weight: 1 },
		{ value: item.skillDefinition.insert, weight: 2 },
		...(item.skillDefinition.tags ?? []).map(tag => ({ value: tag, weight: 2 })),
		...(item.skillDefinition.arguments ?? []).map(arg => ({ value: arg.name, weight: 1 })),
	];
}

function isInstructionSkill(item: SkillListItem): boolean {
	return isInstructionInsertSkill(item.skillDefinition);
}

function compareSkillRows(a: SkillListItem, b: SkillListItem): number {
	const ai = isInstructionSkill(a);
	const bi = isInstructionSkill(b);
	if (ai !== bi) {
		return ai ? -1 : 1;
	}
	return compareSkillListItems(a, b);
}

function resourceCount(item: SkillListItem): number {
	const resources = item.skillDefinition.resources;
	return resources?.hasResources ? resources.totalCount : 0;
}

function getDefaultCustomBundle(bundles: SkillBundle[]): string {
	return bundles.find(bundle => !bundle.isBuiltIn && bundle.isEnabled)?.id ?? '';
}

function makeUniqueSimpleSkillName(seed: string, allSkills: SkillListItem[]): string {
	const base = slugifySkillName(seed) || 'instruction-skill';
	const existing = new Set(allSkills.map(item => item.skillDefinition.name).filter(Boolean));

	if (!existing.has(base)) {
		return base;
	}

	for (let i = 2; i < 1000; i += 1) {
		const suffix = `-${i}`;
		const candidate = `${base.slice(0, Math.max(1, 64 - suffix.length))}${suffix}`;
		if (!existing.has(candidate)) {
			return candidate;
		}
	}

	return `${base.slice(0, 54)}-${Date.now().toString(36)}`;
}

function slugifySkillName(value: string): string {
	return value
		.trim()
		.toLowerCase()
		.replaceAll(/[^a-z0-9-]+/g, '-')
		.replaceAll(/-+/g, '-')
		.replaceAll(/^-|-$/g, '')
		.slice(0, 64);
}

function AddInstructionSkillModal({
	isOpen,
	allSkills,
	mode,
	initialDraft,
	onClose,
	onCreated,
}: {
	isOpen: boolean;
	allSkills: SkillListItem[];
	mode: 'add' | 'fork';
	initialDraft: InstructionSkillDraft | null;
	onClose: () => void;
	onCreated: (item: SkillListItem) => Promise<void> | void;
}) {
	const initial = initialDraft ?? DEFAULT_INSTRUCTION_SKILL_DRAFT;
	const [bundles, setBundles] = useState<SkillBundle[]>([]);
	const [bundleID, setBundleID] = useState(initial.bundleID ?? '');
	const [displayName, setDisplayName] = useState(initial.displayName);
	const [name, setName] = useState(initial.name);
	const [body, setBody] = useState(initial.body);
	const [submitError, setSubmitError] = useState('');
	const [isSubmitting, setIsSubmitting] = useState(false);
	const dialogRef = useRef<HTMLDialogElement | null>(null);
	const isUnmountingRef = useRef(false);

	useEffect(() => {
		if (!isOpen) {
			return;
		}
		let cancelled = false;
		void getAllSkillBundles(undefined, true)
			.then(nextBundles => {
				if (cancelled) {
					return;
				}
				const custom = nextBundles.filter(bundle => !bundle.isBuiltIn);
				setBundles(custom);
				const preferredBundleID =
					initialDraft?.bundleID && custom.some(bundle => bundle.id === initialDraft.bundleID && bundle.isEnabled)
						? initialDraft.bundleID
						: getDefaultCustomBundle(custom);
				setBundleID(preferredBundleID);
			})
			.catch((error: unknown) => {
				console.error('Failed to load skill bundles:', error);
				if (!cancelled) {
					setBundles([]);
				}
			});
		return () => {
			cancelled = true;
		};
	}, [initialDraft, isOpen]);

	useEffect(() => {
		if (!isOpen) {
			return;
		}
		const dialog = dialogRef.current;
		if (!dialog) {
			return;
		}
		if (!dialog.open) {
			try {
				dialog.showModal();
			} catch {
				// Keep safe.
			}
		}
		return () => {
			isUnmountingRef.current = true;
			if (dialog.open) {
				dialog.close();
			}
		};
	}, [isOpen]);

	const requestClose = useCallback(() => {
		const dialog = dialogRef.current;
		if (dialog?.open) {
			dialog.close();
			return;
		}
		onClose();
	}, [onClose]);

	const handleDialogClose = useCallback(() => {
		if (isUnmountingRef.current) {
			return;
		}
		onClose();
	}, [onClose]);

	const existingSlugsForBundle = useMemo(
		() => new Set(allSkills.filter(item => item.bundleID === bundleID).map(item => item.skillSlug)),
		[allSkills, bundleID]
	);
	const normalizedName = slugifySkillName(name);
	const nameError = !normalizedName
		? 'Name is required.'
		: !SIMPLE_SKILL_NAME_RE.test(normalizedName)
			? 'Use lowercase letters, numbers, and hyphens. Maximum 64 characters.'
			: existingSlugsForBundle.has(normalizedName)
				? 'A skill with this slug already exists in the selected bundle.'
				: '';
	const bodyError = body.trim() ? '' : 'Instruction body is required.';
	const selectedBundle = bundles.find(bundle => bundle.id === bundleID);
	const bundleError = !bundleID
		? 'Select a custom enabled skill bundle.'
		: selectedBundle?.isEnabled
			? ''
			: 'The selected custom skill bundle is disabled.';
	const canSubmit = !nameError && !bodyError && !bundleError && !isSubmitting;

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = event => {
		event.preventDefault();
		event.stopPropagation();

		if (!canSubmit) {
			return;
		}

		setIsSubmitting(true);
		setSubmitError('');

		void skillStoreAPI
			.putSkillArtifact(bundleID, normalizedName, {
				name: normalizedName,
				displayName: displayName.trim() || normalizedName,
				description:
					initialDraft?.description ||
					`Instruction skill created from the composer. Use when these instructions should guide the assistant.`,
				insert: 'instructions',
				arguments: [],
				tags: ['composer'],
				markdownBody: body.trim(),
				isEnabled: true,
			})
			.then(async resp => {
				const bundle = bundles.find(item => item.id === bundleID);
				await onCreated({
					bundleID,
					bundleSlug: bundle?.slug ?? bundleID,
					skillSlug: normalizedName,
					isBuiltIn: false,
					skillDefinition: resp,
				});
				requestClose();
			})
			.catch((error: unknown) => {
				setSubmitError(error instanceof Error ? error.message : 'Failed to create instruction skill.');
			})
			.finally(() => {
				setIsSubmitting(false);
			});
	};

	if (!isOpen || typeof document === 'undefined' || !document.body) {
		return null;
	}

	const bundleDropdownItems = Object.fromEntries(
		bundles.map(bundle => [bundle.id, { isEnabled: bundle.isEnabled }] as const)
	);

	return createPortal(
		<dialog
			ref={dialogRef}
			className="modal"
			onClose={handleDialogClose}
			onCancel={event => {
				event.preventDefault();
			}}
		>
			<div className="modal-box bg-base-200 max-h-[80vh] max-w-xl overflow-auto rounded-2xl">
				<div className="mb-4 flex items-center justify-between">
					<h3 className="text-lg font-bold">{mode === 'fork' ? 'Fork Instruction Skill' : 'Add Instruction Skill'}</h3>
					<button type="button" className="btn btn-sm btn-circle bg-base-300" onClick={requestClose} aria-label="Close">
						<FiX size={12} />
					</button>
				</div>

				<form className="space-y-4" onSubmit={handleSubmit}>
					{submitError ? (
						<div className="alert alert-error rounded-2xl text-sm">
							<FiAlertCircle size={14} />
							<span>{submitError}</span>
						</div>
					) : null}

					<div className="alert alert-info rounded-2xl text-sm">
						{mode === 'fork' ? (
							<span>This creates a new managed filesystem instruction skill from the rendered source skill text.</span>
						) : (
							<span>
								This creates a managed filesystem skill with <span className="font-mono">insert: instructions</span>.
								For arguments, scripts, resources, or user-message templates, use the Skill Bundles page.
							</span>
						)}
					</div>

					<div>
						<label className="label py-1">
							<span className="text-sm">Bundle</span>
						</label>
						<Dropdown<string>
							dropdownItems={bundleDropdownItems}
							orderedKeys={bundles.map(bundle => bundle.id)}
							selectedKey={bundleID}
							onChange={setBundleID}
							filterDisabled={true}
							title="Select skill bundle"
							getDisplayName={key => {
								const bundle = bundles.find(item => item.id === key);
								return bundle ? `${bundle.displayName || bundle.slug} (${bundle.slug})` : key;
							}}
						/>
						{bundleError ? <div className="text-error mt-1 text-xs">{bundleError}</div> : null}
					</div>

					<div>
						<label className="label py-1">
							<span className="text-sm">Display name</span>
						</label>
						<input
							className="input w-full rounded-xl"
							value={displayName}
							onChange={event => {
								setDisplayName(event.target.value);
							}}
							spellCheck="false"
						/>
					</div>

					<div>
						<label className="label py-1">
							<span className="text-sm">Skill name and slug</span>
						</label>
						<input
							className={`input w-full rounded-xl ${nameError ? 'input-error' : ''}`}
							value={name}
							onChange={event => {
								const next = event.target.value;
								setName(next);
								if (displayName === 'Instruction Skill') {
									setDisplayName(next.replaceAll('-', ' ').replaceAll(/\b\w/g, c => c.toUpperCase()));
								}
							}}
							spellCheck="false"
						/>
						<div className="text-base-content/70 mt-1 text-xs">Saved as: {normalizedName || '-'}</div>
						{nameError ? <div className="text-error mt-1 text-xs">{nameError}</div> : null}
					</div>

					<div>
						<label className="label py-1">
							<span className="text-sm">Instructions</span>
						</label>
						<textarea
							className={`textarea h-36 w-full rounded-xl ${bodyError ? 'textarea-error' : ''}`}
							value={body}
							onChange={event => {
								setBody(event.target.value);
							}}
							placeholder="Write standing assistant instructions here..."
							spellCheck="false"
						/>
						{bodyError ? <div className="text-error mt-1 text-xs">{bodyError}</div> : null}
					</div>

					<div className="modal-action">
						<button type="button" className="btn bg-base-300 rounded-xl" onClick={requestClose}>
							Cancel
						</button>
						<button type="submit" className="btn btn-primary rounded-xl" disabled={!canSubmit}>
							{isSubmitting ? 'Creating…' : mode === 'fork' ? 'Fork and add to instructions' : 'Create and add'}
						</button>
					</div>
				</form>
			</div>
		</dialog>,
		document.body
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
}: {
	store: MenuStore;
	shortcut: string;
	allSkills: SkillListItem[];
	loading: boolean;
	loadError?: string | null;
	enabledSkillRefs: SkillRef[];
	activeSkillRefs: SkillRef[];
	setEnabledSkillRefs: Dispatch<SetStateAction<SkillRef[]>>;
	setActiveSkillRefs: Dispatch<SetStateAction<SkillRef[]>>;
	onEnableAll: () => void;
	onDisableAll: () => void;
	onRefreshSkills: () => Promise<void>;
	systemPrompt: ComposerSystemPromptController;
	isInputLocked?: boolean;
}) {
	const internalMenu = useMenuStore({ placement: 'top', focusLoop: true });
	const menu = store ?? internalMenu;
	const open = useStoreState(menu, 'open');
	const menuContentElement = useStoreState(menu, 'contentElement');
	const [searchQuery, setSearchQuery] = useSearchableMenuState(open);
	const [isAddInstructionOpen, setIsAddInstructionOpen] = useState(false);
	const [instructionModalMode, setInstructionModalMode] = useState<'add' | 'fork'>('add');
	const [instructionDraft, setInstructionDraft] = useState<InstructionSkillDraft | null>(null);

	useEffect(() => {
		if (isInputLocked) {
			menu.hide();
		}
	}, [isInputLocked, menu]);

	const enabledKeySet = useMemo(() => new Set(enabledSkillRefs.map(k => skillRefKey(k))), [enabledSkillRefs]);
	const activeKeySet = useMemo(() => new Set(activeSkillRefs.map(k => skillRefKey(k))), [activeSkillRefs]);

	const availableSkillKeySet = useMemo(
		() =>
			new Set(
				(allSkills ?? [])
					.filter(s => {
						return isInstructionSkill(s);
					})
					.map(item => skillRefKey(skillRefFromListItem(item)))
			),
		[allSkills]
	);

	const enabledCount = useMemo(() => {
		if (loading) {
			return enabledKeySet.size;
		}

		let count = 0;
		for (const key of enabledKeySet) {
			if (availableSkillKeySet.has(key)) {
				count += 1;
			}
		}
		return count;
	}, [availableSkillKeySet, enabledKeySet, loading]);

	const activeCount = useMemo(() => {
		if (loading) {
			return activeKeySet.size;
		}

		let count = 0;
		for (const key of activeKeySet) {
			if (availableSkillKeySet.has(key)) {
				count += 1;
			}
		}
		return count;
	}, [activeKeySet, availableSkillKeySet, loading]);

	const instructionCount = allSkills.filter(s => {
		return isInstructionSkill(s);
	}).length;
	const totalCount = instructionCount;
	const isEnabled = enabledCount > 0;

	const sortedSkills = useMemo(
		() => [...(allSkills ?? [])].filter(item => isInstructionSkill(item)).toSorted(compareSkillRows),
		[allSkills]
	);

	const displayedGroups = useMemo(() => {
		if (!isSearchQueryActive(searchQuery)) {
			return [{ bundleID: 'flat', bundleSlug: 'skills', skills: sortedSkills }];
		}

		const ranked = rankSearchableItems(sortedSkills, {
			query: searchQuery,
			getKey: skillListItemKey,
			getFields: getSkillSearchFields,
			fallbackCompare: compareSkillRows,
		});
		return [{ bundleID: 'flat', bundleSlug: 'skills', skills: ranked }];
	}, [searchQuery, sortedSkills]);

	const displayedSkillCount = displayedGroups.reduce((sum, group) => sum + group.skills.length, 0);
	const firstVisibleSkill = displayedGroups[0]?.skills[0] ?? null;

	const setSkillEnabled = useCallback(
		(ref: SkillRef, enabled: boolean) => {
			const k = skillRefKey(ref);

			setEnabledSkillRefs(prev => {
				const byKey = new Map<string, SkillRef>();

				for (const r of prev ?? []) {
					byKey.set(skillRefKey(r), r);
				}

				if (enabled) {
					byKey.set(k, ref);
				} else {
					byKey.delete(k);
					setActiveSkillRefs(prevActive => prevActive.filter(activeRef => skillRefKey(activeRef) !== k));
				}

				return [...byKey.values()];
			});
		},
		[setActiveSkillRefs, setEnabledSkillRefs]
	);

	const toggleSkillItem = useCallback(
		(item: SkillListItem) => {
			const ref = skillRefFromListItem(item);
			const k = skillRefKey(ref);
			setSkillEnabled(ref, !enabledKeySet.has(k));
		},
		[enabledKeySet, setSkillEnabled]
	);

	const enableAndActivateSkill = useCallback(
		(item: SkillListItem) => {
			const ref = skillRefFromListItem(item);
			setEnabledSkillRefs(prev => dedupeSkillRefs([...prev, ref]));
			setActiveSkillRefs(prev => dedupeSkillRefs([...prev, ref]));
		},
		[setActiveSkillRefs, setEnabledSkillRefs]
	);

	const addSkillAsSystemInstructions = useCallback(
		async (item: SkillListItem, values: Record<string, string>) => {
			const rendered = await skillStoreAPI.renderSkill(skillRefFromListItem(item), values);
			if (rendered.insert !== 'instructions') {
				throw new Error(`Expected instruction skill, but renderer returned insert=${rendered.insert}.`);
			}

			systemPrompt.addAndSelectInstructionSkillSource({
				identityKey: getInstructionSourceIdentityKey(item),
				displayName: getSkillDisplayLabel(item),
				prompt: rendered.text,
				skillRef: skillRefFromListItem(item),
			});
		},
		[systemPrompt]
	);

	const selectedInstructionSourceKeySet = useMemo(
		() => new Set(systemPrompt.selectedInstructionSourceKeys),
		[systemPrompt.selectedInstructionSourceKeys]
	);
	const selectedInstructionSourceCount =
		systemPrompt.selectedInstructionSourceKeys.length + (systemPrompt.includeModelDefault ? 1 : 0);

	const title = useMemo(() => {
		const lines: string[] = [
			shortcut ? `Instruction skills (${shortcut})` : 'Instruction skills',
			'Instruction skills can be enabled for this chat or activated as standing session context.',
			'Simple argumentless resource-free skills can also be independently selected as flattened system instruction sources.',
			'User-message templates are shown separately in the Templates menu.',
			'Resource-backed skills can still be enabled or activated, but this menu does not fork or inline-copy their resource context.',
			isEnabled ? `Status: Enabled (${enabledCount})` : 'Status: Disabled',
			`Active now: ${activeCount}`,
			`Flattened instruction sources: ${selectedInstructionSourceCount}`,
		];

		if (totalCount > 0) {
			lines.push(`Available: ${totalCount}`);
		}
		if (loading && totalCount === 0) {
			lines.push('Loading available skills…');
		}
		return lines.join('\n');
	}, [activeCount, enabledCount, isEnabled, loading, selectedInstructionSourceCount, shortcut, totalCount]);

	const chipToneClasses =
		enabledCount > 0 || selectedInstructionSourceCount > 0
			? 'border-secondary/50 bg-secondary/10 hover:bg-secondary/15'
			: open
				? 'border-base-300 bg-base-300/60'
				: 'border-transparent';

	const openAddInstructionModal = useCallback(() => {
		setInstructionModalMode('add');
		setInstructionDraft(null);
		setIsAddInstructionOpen(true);
		menu.hide();
	}, [menu]);

	const openForkInstructionModal = useCallback(
		async (item: SkillListItem) => {
			try {
				const rendered = await skillStoreAPI.renderSkill(skillRefFromListItem(item), {});
				const sourceLabel = getSkillDisplayLabel(item);
				const nextName = makeUniqueSimpleSkillName(`${item.skillDefinition.name || item.skillSlug}-fork`, allSkills);

				setInstructionModalMode('fork');
				setInstructionDraft({
					bundleID: item.isBuiltIn ? undefined : item.bundleID,
					displayName: `${sourceLabel} Copy`,
					name: nextName,
					body: rendered.text || `Forked from "${sourceLabel}". Replace this placeholder with instructions.`,
					description:
						item.skillDefinition.description ||
						`Forked from ${sourceLabel}. Use when these instructions should guide the assistant.`,
				});
				setIsAddInstructionOpen(true);
				menu.hide();
			} catch (error) {
				console.error('Failed to render skill for fork:', error);
			}
		},
		[allSkills, menu]
	);

	const renderSkillItem = (item: SkillListItem) => {
		const ref = skillRefFromListItem(item);
		const k = skillRefKey(ref);
		const checked = enabledKeySet.has(k);
		const isActive = activeKeySet.has(k);
		const isInstruction = isInstructionSkill(item);
		const label = getSkillDisplayLabel(item);
		const args = item.skillDefinition.arguments ?? [];
		const resources = resourceCount(item);
		const canUseAsSystemPrompt = isInstruction && skillCanBeRenderedAsInstructionPrompt(item.skillDefinition);
		const instructionPromptReason = getSkillInstructionPromptEligibilityReason(item.skillDefinition);
		const canPreloadActive = isInstruction && skillCanBePreloadedAsActive(item.skillDefinition);
		const instructionSourceKey = getInstructionSourceIdentityKey(item);
		const instructionSourceKnown = systemPrompt.instructionSources.some(
			source => source.identityKey === instructionSourceKey
		);
		const instructionSourceSelected = selectedInstructionSourceKeySet.has(instructionSourceKey);

		return (
			<MenuItem
				key={k}
				data-searchable-menu-item="true"
				hideOnClick={false}
				className={`data-active-item:bg-base-200 flex w-full flex-col items-start gap-2 rounded-xl border p-2 outline-none ${
					!isInstruction ? 'opacity-75' : ''
				}`}
				title={
					isInstruction
						? `${item.bundleSlug}/${item.skillSlug}\nEnable makes it available. Enable + active loads its instructions now.`
						: `${item.bundleSlug}/${item.skillSlug}\nThis is a user-message template. Use the Templates menu.`
				}
				onClick={() => {
					if (isInputLocked || !isInstruction) {
						return;
					}
					toggleSkillItem(item);
				}}
			>
				<div className="flex w-full flex-col space-y-1">
					<div className="flex items-center gap-2">
						<div className="truncate text-xs font-medium">{label}</div>
						<span className={`badge badge-xs ${isInstruction ? 'badge-info' : 'badge-secondary'}`}>
							{isInstruction ? 'instructions' : 'template'}
						</span>
					</div>
					<div className="text-base-content/60 truncate text-xs">
						{item.bundleSlug}/{item.skillSlug} • {item.skillDefinition.name}
					</div>

					<div className="mt-1 flex flex-wrap items-center justify-end gap-1">
						{checked ? <span className="badge badge-success badge-xs">Enabled</span> : null}
						{isInstruction && isActive ? <span className="badge badge-info badge-xs">Active</span> : null}
						{resources > 0 ? (
							<span className="badge badge-ghost badge-xs">
								{resources} resource{resources === 1 ? '' : 's'}
							</span>
						) : null}
						{isInstruction && args.length === 0 && resources === 0 ? (
							<span className="badge badge-ghost badge-xs" title="Can be copied or forked as plain instructions.">
								simple
							</span>
						) : null}
						{instructionSourceSelected ? (
							<span className="badge badge-secondary badge-xs">System instructions</span>
						) : null}
						<span className="badge badge-ghost badge-xs">{item.isBuiltIn ? 'built-in' : 'custom'}</span>
						{!item.skillDefinition.isEnabled ? <span className="badge badge-warning badge-xs">Disabled</span> : null}
					</div>
				</div>

				<div className="ml-2 flex w-full items-center justify-end gap-1" onClick={stop} onPointerDown={stop}>
					{isInstruction ? (
						checked ? (
							<>
								{isActive ? (
									<button
										type="button"
										className="btn btn-xs rounded-lg"
										disabled={isInputLocked}
										onClick={() => {
											setActiveSkillRefs(prev => prev.filter(activeRef => skillRefKey(activeRef) !== k));
										}}
									>
										Deactivate
									</button>
								) : (
									<button
										type="button"
										className="btn btn-xs rounded-lg"
										disabled={isInputLocked || !canPreloadActive}
										title={
											!canPreloadActive
												? 'This skill cannot be preloaded as active because it has arguments.'
												: undefined
										}
										onClick={() => {
											enableAndActivateSkill(item);
										}}
									>
										Activate
									</button>
								)}
								<button
									type="button"
									className="btn btn-xs rounded-lg"
									disabled={isInputLocked}
									onClick={() => {
										setSkillEnabled(ref, false);
									}}
								>
									Disable
								</button>
							</>
						) : (
							<>
								<button
									type="button"
									className="btn btn-xs rounded-lg"
									disabled={isInputLocked}
									onClick={() => {
										setSkillEnabled(ref, true);
									}}
								>
									Enable
								</button>
								<button
									type="button"
									className="btn btn-xs rounded-lg"
									disabled={isInputLocked || !canPreloadActive}
									title={
										!canPreloadActive ? 'This skill cannot be preloaded as active because it has arguments.' : undefined
									}
									onClick={() => {
										enableAndActivateSkill(item);
									}}
								>
									Enable + activate
								</button>
							</>
						)
					) : (
						<span className="text-base-content/60 text-xs">Use Templates</span>
					)}

					{isInstruction ? (
						<button
							type="button"
							className="btn btn-xs rounded-lg"
							disabled={isInputLocked || !canUseAsSystemPrompt}
							title={
								canUseAsSystemPrompt
									? instructionSourceSelected
										? 'Remove this skill from the flattened system instructions.'
										: 'Select this simple skill as a flattened system instruction source. This does not enable or activate the skill session.'
									: instructionPromptReason
							}
							onClick={() => {
								if (!canUseAsSystemPrompt) {
									return;
								}
								if (instructionSourceKnown) {
									systemPrompt.toggleInstructionSource(instructionSourceKey);
									return;
								}
								void addSkillAsSystemInstructions(item, {});
							}}
						>
							{instructionSourceSelected
								? 'Remove from instructions'
								: instructionSourceKnown
									? 'Select as instructions'
									: 'Add as instructions'}
						</button>
					) : null}

					{isInstruction && args.length === 0 && resources === 0 ? (
						<button
							type="button"
							className="btn btn-xs rounded-lg"
							disabled={isInputLocked}
							title="Create a new managed instruction skill from this rendered skill text."
							onClick={() => {
								void openForkInstructionModal(item);
							}}
						>
							<FiGitBranch size={12} />
							<span className="ml-1">Fork</span>
						</button>
					) : null}
				</div>
			</MenuItem>
		);
	};

	return (
		<div className="relative shrink-0" data-bottom-bar-skills>
			<HoverTip content={title} placement="top" wrapperElement="div" wrapperClassName="inline-flex max-w-full">
				<div
					className={`${actionTriggerChipSurfaceClasses} border ${chipToneClasses} ${isInputLocked ? 'opacity-60' : ''}`}
				>
					<MenuButton
						store={menu}
						className="btn btn-xs app-text-neutral h-auto min-h-0 flex-1 gap-0 border-none bg-transparent p-0 text-left font-normal shadow-none hover:bg-transparent"
						aria-label={shortcut ? `Attach skills (${shortcut})` : 'Attach skills'}
						disabled={isInputLocked}
					>
						<ActionTriggerChipContent
							icon={<FiFilePlus size={14} />}
							label="Skills"
							count={
								isEnabled ? (
									<span className="badge badge-success badge-xs bg-success/30">{enabledCount}</span>
								) : undefined
							}
							suffix={
								activeCount > 0 ? (
									<span className="badge badge-info badge-xs bg-info/30">Active {activeCount}</span>
								) : isEnabled ? (
									<FiCheck size={14} className="shrink-0" />
								) : undefined
							}
							open={open}
						/>
					</MenuButton>

					{enabledCount > 0 ? (
						<button
							type="button"
							className="btn btn-xs app-text-neutral hover:bg-base-300/80 ml-1 h-auto min-h-0 shrink-0 px-1 py-0 shadow-none"
							onClick={event => {
								stop(event);
								onDisableAll();
								menu.hide();
							}}
							aria-label="Clear all skills"
							title="Clear all skills"
							disabled={isInputLocked}
						>
							<FiX size={12} />
						</button>
					) : null}
				</div>
			</HoverTip>

			<Menu
				store={menu}
				gutter={8}
				overflowPadding={8}
				className={actionTriggerMenuWideClasses}
				autoFocusOnShow={false}
				portal
			>
				<div className="mb-2 flex items-center justify-between gap-2 px-1">
					<div className="text-base-content/70 text-xs font-semibold">Instruction Skills</div>
					<div className="text-base-content/60 flex items-center gap-2 text-xs">
						<span>Enabled: {enabledCount}</span>
						<span>•</span>
						<span>Active: {activeCount}</span>
						<span>•</span>
						<span>{totalCount} available</span>
					</div>
				</div>

				{loadError ? (
					<div className="alert alert-warning mb-2 rounded-xl text-xs">
						<div className="grow">
							<div className="font-semibold">Skills could not be refreshed</div>
							<div>{loadError}</div>
						</div>
						<button type="button" className="btn btn-xs rounded-lg" onClick={() => void onRefreshSkills()}>
							Retry
						</button>
					</div>
				) : null}

				<div className="border-base-300 bg-base-100 mb-2 rounded-xl border p-2">
					<div className="mb-2 flex items-center justify-between gap-2">
						<div className="text-xs font-semibold">System instruction sources</div>
						<button
							type="button"
							className="btn btn-ghost btn-xs rounded-lg"
							disabled={isInputLocked || selectedInstructionSourceCount === 0}
							onClick={() => {
								systemPrompt.clearInstructionSources();
							}}
						>
							Clear selected
						</button>
					</div>

					<div className="space-y-1">
						{systemPrompt.modelDefaultPrompt.trim() ? (
							<label className="flex cursor-pointer items-center justify-between gap-2 rounded-lg px-2 py-1">
								<span className="truncate text-xs">Model default instructions</span>
								<input
									type="checkbox"
									className="checkbox checkbox-xs"
									checked={systemPrompt.includeModelDefault}
									disabled={isInputLocked}
									onChange={event => {
										systemPrompt.setIncludeModelDefault(event.target.checked);
									}}
								/>
							</label>
						) : null}

						{systemPrompt.instructionSources.map(source => (
							<label
								key={source.identityKey}
								className="flex cursor-pointer items-center justify-between gap-2 rounded-lg px-2 py-1"
								title={source.text}
							>
								<span className="min-w-0 truncate text-xs">{source.displayName}</span>
								<input
									type="checkbox"
									className="checkbox checkbox-xs"
									checked={selectedInstructionSourceKeySet.has(source.identityKey)}
									disabled={isInputLocked}
									onChange={() => {
										systemPrompt.toggleInstructionSource(source.identityKey);
									}}
								/>
							</label>
						))}

						{!systemPrompt.modelDefaultPrompt.trim() && systemPrompt.instructionSources.length === 0 ? (
							<div className="text-base-content/60 px-2 py-1 text-xs">No instruction sources selected.</div>
						) : null}
					</div>
				</div>

				<SearchableMenuInput
					open={open}
					query={searchQuery}
					onQueryChange={setSearchQuery}
					placeholder="Search skills…"
					resultCount={displayedSkillCount}
					totalCount={totalCount}
					disabled={loading || totalCount === 0}
					onFocusFirstItem={() => {
						focusFirstSearchableMenuItem(menuContentElement);
					}}
					onEnterFirstResult={() => {
						if (firstVisibleSkill && !isInputLocked) {
							toggleSkillItem(firstVisibleSkill);
						}
					}}
					onEscape={() => {
						menu.hide();
					}}
				/>

				{!loading ? (
					<div className="border-base-300 mb-2 flex flex-wrap items-center justify-between gap-2 border-b px-1 pb-2">
						<button
							type="button"
							className="btn btn-xs rounded-lg"
							disabled={isInputLocked}
							onClick={openAddInstructionModal}
							title="Create a simple managed instruction skill and select it as flattened system instructions."
						>
							<FiPlus size={12} />
							<span className="ml-1">Add new instruction skill</span>
						</button>

						{totalCount > 0 ? (
							<div className="flex gap-2">
								<button
									type="button"
									className="btn btn-xs rounded-lg"
									disabled={isInputLocked || totalCount === 0 || enabledCount === totalCount}
									onClick={e => {
										stop(e);
										onEnableAll();
									}}
									title="Enable all instruction skills"
								>
									Enable all
								</button>

								<button
									type="button"
									className="btn btn-xs rounded-lg"
									disabled={isInputLocked || enabledCount === 0}
									onClick={e => {
										stop(e);
										onDisableAll();
										menu.hide();
									}}
									title="Disable all selected instruction skills and remove active skill session instructions."
								>
									<FiX size={12} />
									<span className="ml-1">Clear all</span>
								</button>
							</div>
						) : null}
					</div>
				) : null}

				{loading ? (
					<div className={`${actionTriggerMenuItemClasses} text-base-content/60 cursor-default`}>Loading skills…</div>
				) : totalCount === 0 ? (
					<div className={`${actionTriggerMenuItemClasses} text-base-content/60 cursor-default`}>
						No skills available
					</div>
				) : displayedSkillCount === 0 ? (
					<div className={searchableMenuEmptyStateClasses}>No skills match your search.</div>
				) : (
					<div className="space-y-1">
						{displayedGroups[0]?.skills.map(item => (
							<div className="w-full" key={skillListItemKey(item)}>
								{renderSkillItem(item)}
							</div>
						))}
					</div>
				)}
			</Menu>

			{isAddInstructionOpen ? (
				<AddInstructionSkillModal
					key={`${instructionModalMode}:${instructionDraft?.name ?? 'new'}`}
					isOpen={true}
					allSkills={allSkills}
					mode={instructionModalMode}
					initialDraft={instructionDraft}
					onClose={() => {
						setIsAddInstructionOpen(false);
						setInstructionDraft(null);
					}}
					onCreated={async item => {
						await addSkillAsSystemInstructions(item, {});
						await onRefreshSkills();
						setIsAddInstructionOpen(false);
						setInstructionDraft(null);
						menu.hide();
					}}
				/>
			) : null}
		</div>
	);
}
