import { memo, type SyntheticEvent, useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { FiCheck, FiFileText, FiGitBranch, FiPlus, FiX } from 'react-icons/fi';

import {
	Menu,
	MenuButton,
	MenuItem,
	type MenuStore,
	Tooltip,
	useMenuStore,
	useStoreState,
	useTooltipStore,
} from '@ariakit/react';

import { PREVIOUS_CONVO_SYSTEM_PROMPT_BUNDLEID } from '@/spec/modelpreset';
import { type PromptBundle, PromptRoleEnum } from '@/spec/prompt';

import { DEFAULT_SEMVER } from '@/lib/version_utils';

import {
	ActionTriggerChipContent,
	actionTriggerChipSurfaceClasses,
	actionTriggerMenuWideClasses,
} from '@/components/action_trigger_chip';
import { HoverTip } from '@/components/ariakit_hover_tip';
import { GroupedMenuSection } from '@/components/grouped_menu_sections';

import { SystemPromptAddModal } from '@/chats/composer/systemprompts/system_prompt_add_modal';
import type { ComposerSystemPromptController } from '@/chats/composer/systemprompts/use_composer_system_prompt';
import { countEnabledSystemPromptSources } from '@/prompts/lib/system_prompt_utils';
import type { SystemPromptDraft, SystemPromptItem } from '@/prompts/lib/use_system_prompts';

const REFRESH_STALE_MS = 60_000;

const PROMPT_SORT_COLLATOR = new Intl.Collator(undefined, {
	numeric: true,
	sensitivity: 'base',
});

type SystemPromptBundleGroup = {
	bundleID: string;
	bundleSlug: string;
	bundleDisplayName: string;
	isKnownBundle: boolean;
	isBundleEnabled: boolean;
	isBuiltIn: boolean;
	sortKey: string;
	prompts: SystemPromptItem[];
};

type SystemPromptBottomBarChipProps = {
	store: MenuStore;
	shortcut: string;
	prompts: SystemPromptItem[];
	bundles: PromptBundle[];
	selectedPromptKeys: string[];
	preferredBundleID: string | null;
	loading: boolean;
	error: string | null;
	modelDefaultPrompt: string;
	includeModelDefault: boolean;
	onTogglePrompt: (identityKey: string) => void;
	onToggleModelDefault: (next: boolean) => void;
	onAddPrompt: (draft: SystemPromptDraft) => Promise<void>;
	onClearSelected: () => void;
	onRefreshPrompts: () => Promise<void>;
	getExistingVersions: (bundleID: string, slug: string) => string[];
	isInputLocked?: boolean;
};

function pickInitialBundleID(
	bundles: PromptBundle[],
	preferredBundleID: string | null,
	sourceBundleID?: string
): string {
	const writable = bundles.filter(bundle => !bundle.isBuiltIn && bundle.isEnabled);
	const custom = bundles.filter(bundle => !bundle.isBuiltIn);

	if (sourceBundleID && writable.some(bundle => bundle.id === sourceBundleID)) {
		return sourceBundleID;
	}
	if (preferredBundleID && writable.some(bundle => bundle.id === preferredBundleID)) {
		return preferredBundleID;
	}
	return writable[0]?.id ?? custom[0]?.id ?? '';
}

function stopMenuToggleEvent(event: SyntheticEvent) {
	event.preventDefault();
	event.stopPropagation();
}

function stopMenuBubbleEvent(event: SyntheticEvent) {
	event.stopPropagation();
}

function SystemPromptBottomBarChipInner({
	store,
	shortcut,
	prompts,
	bundles,
	selectedPromptKeys,
	preferredBundleID,
	loading,
	error,
	modelDefaultPrompt,
	includeModelDefault,
	onTogglePrompt,
	onToggleModelDefault,
	onAddPrompt,
	onClearSelected,
	onRefreshPrompts,
	getExistingVersions,
	isInputLocked = false,
}: SystemPromptBottomBarChipProps) {
	const [modalMode, setModalMode] = useState<'add' | 'fork'>('add');
	const [isComposerOpen, setIsComposerOpen] = useState(false);
	const [composerInitialDraft, setComposerInitialDraft] = useState<SystemPromptDraft | null>(null);

	const hasModelDefaultPrompt = modelDefaultPrompt.trim().length > 0;
	const hasWritableCustomBundle = useMemo(
		() => bundles.some(bundle => !bundle.isBuiltIn && bundle.isEnabled),
		[bundles]
	);

	const promptsByKey = useMemo(() => new Map(prompts.map(item => [item.identityKey, item])), [prompts]);

	const activeSourceCount = useMemo(
		() =>
			countEnabledSystemPromptSources({
				modelDefaultPrompt,
				includeModelDefault,
				selectedPromptKeys,
				promptsByKey,
			}),
		[includeModelDefault, modelDefaultPrompt, promptsByKey, selectedPromptKeys]
	);

	const internalMenu = useMenuStore({ placement: 'top', focusLoop: true });
	const menu = store ?? internalMenu;
	const promptTooltip = useTooltipStore({ placement: 'right-end' });

	const lastRefreshTsRef = useRef(0);
	const open = useStoreState(menu, 'open');
	const selectedKeySet = useMemo(() => new Set(selectedPromptKeys), [selectedPromptKeys]);

	const groupedPrompts = useMemo(() => {
		const bundlesByID = new Map(bundles.map(bundle => [bundle.id, bundle]));
		const groupsByBundleID = new Map<string, SystemPromptBundleGroup>();

		for (const item of prompts) {
			const bundle = bundlesByID.get(item.bundleID);
			const bundleSlug = bundle?.slug?.trim() || item.bundleID;
			const bundleDisplayName = bundle?.displayName?.trim() || item.bundleDisplayName?.trim() || bundleSlug;

			let group = groupsByBundleID.get(item.bundleID);
			if (!group) {
				group = {
					bundleID: item.bundleID,
					bundleSlug,
					bundleDisplayName,
					isKnownBundle: Boolean(bundle),
					isBundleEnabled: bundle?.isEnabled ?? true,
					isBuiltIn: bundle?.isBuiltIn ?? item.isBuiltIn,
					sortKey: bundleSlug,
					prompts: [],
				};
				groupsByBundleID.set(item.bundleID, group);
			}

			group.prompts.push(item);
		}

		return Array.from(groupsByBundleID.values())
			.map(group => ({
				...group,
				prompts: [...group.prompts].sort((left, right) => {
					const slugCompare = PROMPT_SORT_COLLATOR.compare(left.templateSlug, right.templateSlug);
					if (slugCompare !== 0) return slugCompare;

					const versionCompare = PROMPT_SORT_COLLATOR.compare(left.templateVersion, right.templateVersion);
					if (versionCompare !== 0) return versionCompare;

					const nameCompare = PROMPT_SORT_COLLATOR.compare(left.displayName, right.displayName);
					if (nameCompare !== 0) return nameCompare;

					return PROMPT_SORT_COLLATOR.compare(left.identityKey, right.identityKey);
				}),
			}))
			.sort((left, right) => {
				const bundleCompare = PROMPT_SORT_COLLATOR.compare(left.sortKey, right.sortKey);
				if (bundleCompare !== 0) return bundleCompare;
				return PROMPT_SORT_COLLATOR.compare(left.bundleID, right.bundleID);
			});
	}, [bundles, prompts]);

	const tooltipAnchorEl = useStoreState(promptTooltip, 'anchorElement');
	const currentPromptText = tooltipAnchorEl?.dataset.prompt ?? '';

	const triggerTooltip = [
		shortcut ? `Insert system prompt (${shortcut})` : 'Insert system prompt',
		activeSourceCount > 0 ? `System prompt sources enabled: ${activeSourceCount}` : 'System prompt disabled',
	].join('\n');

	const showPromptTooltip = useCallback(
		(element: HTMLElement) => {
			const prompt = element.dataset.prompt ?? '';
			if (!prompt.trim()) {
				promptTooltip.hide();
				promptTooltip.setAnchorElement(null);
				return;
			}

			promptTooltip.setAnchorElement(element);
			promptTooltip.show();
		},
		[promptTooltip]
	);

	const hidePromptTooltip = useCallback(() => {
		promptTooltip.hide();
		promptTooltip.setAnchorElement(null);
	}, [promptTooltip]);

	useEffect(() => {
		if (!isInputLocked) return;
		hidePromptTooltip();
		menu.hide();
	}, [hidePromptTooltip, isInputLocked, menu]);

	const handleMenuOpen = useCallback(() => {
		if (isInputLocked) return;

		const now = Date.now();
		if (now - lastRefreshTsRef.current < REFRESH_STALE_MS) return;

		lastRefreshTsRef.current = now;
		void onRefreshPrompts().catch((refreshError: unknown) => {
			console.error('Failed to refresh system prompts:', refreshError);
		});
	}, [isInputLocked, onRefreshPrompts]);

	const openAddModal = useCallback(() => {
		if (isInputLocked) return;

		setModalMode('add');
		setComposerInitialDraft({
			bundleID: pickInitialBundleID(bundles, preferredBundleID),
			displayName: 'System Prompt',
			slug: 'system-prompt',
			version: DEFAULT_SEMVER,
			role: PromptRoleEnum.System,
			content: '',
		});

		requestAnimationFrame(() => {
			setIsComposerOpen(true);
		});
	}, [bundles, isInputLocked, preferredBundleID]);

	const openForkModal = useCallback(
		(item: SystemPromptItem) => {
			if (isInputLocked) return;

			const isSyntheticPreviousConversationPrompt = item.bundleID === PREVIOUS_CONVO_SYSTEM_PROMPT_BUNDLEID;
			setModalMode(isSyntheticPreviousConversationPrompt ? 'add' : 'fork');

			setComposerInitialDraft({
				bundleID: pickInitialBundleID(
					bundles,
					preferredBundleID,
					isSyntheticPreviousConversationPrompt ? undefined : item.bundleID
				),
				displayName: item.displayName,
				slug: item.templateSlug,
				version: isSyntheticPreviousConversationPrompt ? DEFAULT_SEMVER : item.templateVersion,
				role: item.role,
				content: item.prompt,
			});

			requestAnimationFrame(() => {
				setIsComposerOpen(true);
			});
		},
		[bundles, isInputLocked, preferredBundleID]
	);

	const chipToneClasses =
		activeSourceCount > 0
			? 'border-secondary/50 bg-secondary/10 hover:bg-secondary/15'
			: open
				? 'border-base-300 bg-base-300/60'
				: 'border-transparent';

	return (
		<div className="relative shrink-0" data-bottom-bar-system-prompt>
			<HoverTip content={triggerTooltip} placement="top" wrapperElement="div" wrapperClassName="inline-flex max-w-full">
				<div
					className={`${actionTriggerChipSurfaceClasses} border ${chipToneClasses} ${isInputLocked ? 'opacity-60' : ''}`}
				>
					<MenuButton
						store={menu}
						className="btn btn-xs text-neutral-custom h-auto min-h-0 flex-1 gap-0 border-none bg-transparent p-0 text-left font-normal shadow-none hover:bg-transparent"
						onClick={handleMenuOpen}
						disabled={isInputLocked}
						aria-label={shortcut ? `Insert system prompt (${shortcut})` : 'Insert system prompt'}
					>
						<ActionTriggerChipContent
							icon={<FiFileText size={14} />}
							label="System prompt"
							count={
								activeSourceCount > 0 ? (
									<span className="badge badge-success badge-xs bg-success/30">{activeSourceCount}</span>
								) : undefined
							}
							suffix={activeSourceCount > 0 ? <FiCheck size={14} className="shrink-0" /> : undefined}
							open={open}
							labelClassName="max-w-28 truncate text-xs font-normal"
						/>
					</MenuButton>

					{activeSourceCount > 0 ? (
						<button
							type="button"
							className="btn btn-ghost btn-xs text-neutral-custom hover:bg-base-300/80 ml-1 h-auto min-h-0 shrink-0 px-1 py-0 shadow-none"
							onClick={event => {
								stopMenuToggleEvent(event);
								hidePromptTooltip();
								onClearSelected();
								menu.hide();
							}}
							aria-label="Clear system prompt sources"
							title="Clear system prompt sources"
							disabled={isInputLocked}
						>
							<FiX size={12} />
						</button>
					) : null}
				</div>
			</HoverTip>

			<Menu
				store={menu}
				portal
				gutter={8}
				overflowPadding={8}
				autoFocusOnShow
				className={actionTriggerMenuWideClasses}
				onKeyDownCapture={event => {
					if (event.key === 'Escape') {
						hidePromptTooltip();
						menu.hide();
					}
				}}
			>
				<div className="mb-2 px-1 text-xs opacity-70">
					Add/Fork creates a prompt template. <br />
					{hasModelDefaultPrompt
						? 'Active sources are concatenated in this order: model default, then selected saved prompts.'
						: ''}
					Bundles cannot be created here; use the Prompt Bundles page.
				</div>

				{hasModelDefaultPrompt ? (
					<MenuItem
						hideOnClick={false}
						className={`data-active-item:bg-base-200 border-base-300 mb-2 rounded-lg border p-2 outline-none ${
							isInputLocked ? 'opacity-60' : ''
						}`}
						data-prompt={modelDefaultPrompt}
						onFocus={event => {
							showPromptTooltip(event.currentTarget);
						}}
						onBlur={hidePromptTooltip}
						onMouseEnter={event => {
							showPromptTooltip(event.currentTarget);
						}}
						onMouseLeave={hidePromptTooltip}
						onClick={() => {
							if (isInputLocked) return;
							onToggleModelDefault(!includeModelDefault);
						}}
					>
						<div className="flex w-full items-start gap-2">
							<input
								type="checkbox"
								className="checkbox checkbox-xs mt-0.5 rounded"
								checked={includeModelDefault}
								disabled={isInputLocked}
								onChange={event => {
									stopMenuBubbleEvent(event);
									if (isInputLocked) return;
									onToggleModelDefault(event.currentTarget.checked);
								}}
								onPointerDown={stopMenuToggleEvent}
								onClick={stopMenuToggleEvent}
							/>
							<div className="min-w-0 flex-1">
								<div className="flex items-center gap-2">
									<span className="font-medium">Model default</span>
									<span className="badge badge-ghost badge-xs">per selected model</span>
								</div>
							</div>
							{includeModelDefault ? <FiCheck size={14} className="text-success mt-0.5 shrink-0" /> : null}
						</div>
					</MenuItem>
				) : null}

				<div className="mb-1 px-1 text-xs font-medium opacity-70">Saved prompts</div>

				{loading ? (
					<div className="m-0 flex cursor-default items-center justify-between rounded-md p-2 text-xs opacity-70">
						<span>Loading system prompts…</span>
					</div>
				) : error ? (
					<div className="text-error m-0 flex cursor-default items-center justify-between rounded-md p-2 text-xs">
						<span>{error}</span>
					</div>
				) : prompts.length > 0 ? (
					<div className="space-y-2">
						{groupedPrompts.map((group, groupIndex) => (
							<GroupedMenuSection
								key={group.bundleID}
								title={group.bundleDisplayName}
								ariaLabel={`${group.bundleDisplayName} saved prompts`}
								separatorBefore={groupIndex > 0}
								meta={
									<>
										<span className="badge badge-ghost badge-xs">{group.prompts.length}</span>
										<span className="badge badge-ghost badge-xs">{group.isBuiltIn ? 'built-in' : 'custom'}</span>
										{!group.isKnownBundle ? (
											<span className="badge badge-warning badge-xs">missing</span>
										) : !group.isBundleEnabled ? (
											<span className="badge badge-warning badge-xs">disabled</span>
										) : null}
									</>
								}
							>
								<div className="space-y-1">
									{group.prompts.map(item => {
										const isSelected = selectedKeySet.has(item.identityKey);

										return (
											<MenuItem
												key={item.identityKey}
												hideOnClick={false}
												data-prompt={item.prompt}
												className={`data-active-item:bg-base-200 border-base-300 flex items-start gap-2 rounded-lg border p-2 outline-none ${
													isInputLocked ? 'opacity-60' : ''
												}`}
												onFocus={event => {
													showPromptTooltip(event.currentTarget);
												}}
												onBlur={hidePromptTooltip}
												onMouseEnter={event => {
													showPromptTooltip(event.currentTarget);
												}}
												onMouseLeave={hidePromptTooltip}
												onClick={() => {
													if (isInputLocked) return;
													onTogglePrompt(item.identityKey);
												}}
											>
												<input
													type="checkbox"
													className="checkbox checkbox-xs mt-0.5 rounded"
													checked={isSelected}
													disabled={isInputLocked}
													onChange={event => {
														stopMenuBubbleEvent(event);
														if (isInputLocked) return;
														onTogglePrompt(item.identityKey);
													}}
													onPointerDown={stopMenuToggleEvent}
													onClick={stopMenuToggleEvent}
												/>

												<div className="min-w-0 flex-1">
													<div className="truncate text-xs font-medium">{item.displayName}</div>
													<div className="mt-1 flex items-center gap-2 text-[10px] opacity-70">
														<span>{item.templateSlug}</span>
														<span>•</span>
														<span>{item.templateVersion}</span>
														<span>•</span>
														<span>{item.role === PromptRoleEnum.Developer ? 'developer' : 'system'}</span>
													</div>
												</div>

												<div className="ml-2 flex shrink-0 items-start gap-1">
													{isSelected ? <FiCheck size={14} className="text-success mt-0.5 shrink-0" /> : null}
													<button
														type="button"
														className="btn btn-ghost btn-xs btn-square rounded-lg"
														title="Fork"
														disabled={isInputLocked || !hasWritableCustomBundle}
														onClick={event => {
															stopMenuToggleEvent(event);
															if (isInputLocked) return;
															hidePromptTooltip();
															openForkModal(item);
														}}
													>
														<FiGitBranch size={12} />
													</button>
												</div>
											</MenuItem>
										);
									})}
								</div>
							</GroupedMenuSection>
						))}
					</div>
				) : (
					<div className="m-0 flex cursor-default items-center justify-between rounded-md p-2 text-xs opacity-70">
						<span>No saved prompts</span>
					</div>
				)}

				<div className="border-neutral/20 mt-2 border-t pt-2 text-xs">
					<div className="flex items-center justify-between gap-2 p-1">
						<button
							type="button"
							className="btn btn-ghost btn-xs rounded-lg"
							disabled={isInputLocked || !hasWritableCustomBundle}
							onClick={() => {
								if (isInputLocked) return;
								hidePromptTooltip();
								openAddModal();
							}}
						>
							<FiPlus size={14} className="mr-1" /> Add
						</button>

						<button
							type="button"
							className="btn btn-ghost btn-xs rounded-lg"
							onClick={() => {
								if (isInputLocked) return;
								hidePromptTooltip();
								onClearSelected();
								menu.hide();
							}}
							title="Clear all selected prompt sources"
							disabled={isInputLocked || (!includeModelDefault && selectedPromptKeys.length === 0)}
						>
							<FiX size={14} className="mr-1" /> Clear all
						</button>
					</div>

					{!hasWritableCustomBundle ? (
						<div className="text-warning px-1 pt-1 text-xs">
							No enabled custom bundle is available for Add/Fork. Create or enable one in Prompt Bundles.
						</div>
					) : null}
				</div>
			</Menu>

			<Tooltip
				store={promptTooltip}
				portal
				gutter={8}
				overflowPadding={8}
				className="rounded-box bg-base-100 text-base-content border-base-300 z-1000 max-w-sm border p-2 text-xs whitespace-pre-wrap shadow-xl"
			>
				{currentPromptText}
			</Tooltip>

			<SystemPromptAddModal
				isOpen={isComposerOpen}
				mode={modalMode}
				initialDraft={composerInitialDraft}
				bundles={bundles}
				getExistingVersions={getExistingVersions}
				onClose={() => {
					setIsComposerOpen(false);
					setComposerInitialDraft(null);
				}}
				onSave={async draft => {
					await onAddPrompt(draft);
					setIsComposerOpen(false);
					setComposerInitialDraft(null);
				}}
			/>
		</div>
	);
}

/**
 * Isolated wrapper for systemPromptBottomBarChip so its open/close state changes
 * don't re-render the entire EditorBottomBar.
 */
export const SystemPromptBottomBarChip = memo(function SystemPromptBottomBarChip({
	store,
	shortcut,
	systemPrompt,
	isInputLocked,
}: {
	store: MenuStore;
	shortcut: string;
	systemPrompt: ComposerSystemPromptController;
	isInputLocked: boolean;
}) {
	return (
		<SystemPromptBottomBarChipInner
			store={store}
			shortcut={shortcut}
			prompts={systemPrompt.prompts}
			bundles={systemPrompt.systemPromptBundles}
			selectedPromptKeys={systemPrompt.selectedPromptKeys}
			preferredBundleID={systemPrompt.preferredSystemPromptBundleID}
			loading={systemPrompt.systemPromptsLoading}
			error={systemPrompt.systemPromptError}
			modelDefaultPrompt={systemPrompt.modelDefaultPrompt}
			includeModelDefault={systemPrompt.includeModelDefault}
			onTogglePrompt={systemPrompt.togglePromptSelection}
			onToggleModelDefault={systemPrompt.setIncludeModelDefault}
			onAddPrompt={systemPrompt.addAndSelectPrompt}
			onClearSelected={systemPrompt.clearSelectedPromptSources}
			onRefreshPrompts={systemPrompt.refreshSystemPrompts}
			getExistingVersions={systemPrompt.getExistingSystemPromptVersions}
			isInputLocked={isInputLocked}
		/>
	);
});
