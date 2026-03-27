import type { Dispatch, SetStateAction } from 'react';
import { useCallback, useEffect, useMemo, useState } from 'react';

import { FiCheck, FiChevronDown, FiChevronUp, FiGitBranch, FiPlus, FiX } from 'react-icons/fi';

import { Popover, PopoverDisclosure, Tooltip, usePopoverStore, useStoreState, useTooltipStore } from '@ariakit/react';

import { PREVIOUS_CONVO_SYSTEM_PROMPT_BUNDLEID } from '@/spec/modelpreset';
import { type PromptBundle, PromptRoleEnum } from '@/spec/prompt';

import { DEFAULT_SEMVER } from '@/lib/version_utils';

import { SystemPromptAddModal } from '@/chats/inputarea/systemprompts/system_prompt_add_modal';
import { countEnabledSystemPromptSources } from '@/prompts/lib/system_prompt_utils';
import type { SystemPromptDraft, SystemPromptItem } from '@/prompts/lib/use_system_prompts';

type SystemPromptDropdownProps = {
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
	isOpen: boolean;
	setIsOpen: Dispatch<SetStateAction<boolean>>;
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

export function SystemPromptDropdown({
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
	isOpen,
	setIsOpen,
}: SystemPromptDropdownProps) {
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

	const popover = usePopoverStore({
		open: isOpen,
		setOpen: setIsOpen,
		placement: 'top-start',
	});

	const open = useStoreState(popover, 'open');

	const selectedKeySet = useMemo(() => new Set(selectedPromptKeys), [selectedPromptKeys]);
	const promptTooltip = useTooltipStore({ placement: 'left-end' });
	const tooltipAnchorEl = useStoreState(promptTooltip, 'anchorElement');
	const currentPromptText = tooltipAnchorEl?.dataset.prompt ?? '';

	const showPromptTooltip = useCallback(
		(element: HTMLElement) => {
			const prompt = element.dataset.prompt ?? '';
			if (!prompt.trim()) return;

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
		if (!isOpen) return;
		void onRefreshPrompts().catch((error: unknown) => {
			console.error('Failed to refresh system prompts:', error);
		});
	}, [isOpen, onRefreshPrompts]);

	const openAddModal = useCallback(() => {
		setModalMode('add');
		setComposerInitialDraft({
			bundleID: pickInitialBundleID(bundles, preferredBundleID),
			displayName: 'System Prompt',
			slug: 'system-prompt',
			version: DEFAULT_SEMVER,
			role: PromptRoleEnum.System,
			content: '',
		});
		setIsOpen(false);
		requestAnimationFrame(() => {
			setIsComposerOpen(true);
		});
	}, [bundles, preferredBundleID, setIsOpen]);

	const openForkModal = useCallback(
		(item: SystemPromptItem) => {
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
			setIsOpen(false);
			requestAnimationFrame(() => {
				setIsComposerOpen(true);
			});
		},
		[bundles, preferredBundleID, setIsOpen]
	);

	return (
		<div className="flex w-full justify-center">
			<div className="relative w-full">
				<PopoverDisclosure
					store={popover}
					className="btn btn-xs text-neutral-custom w-full flex-1 items-center overflow-hidden border-none p-0 text-center text-nowrap shadow-none"
					title={
						activeSourceCount > 0 ? `System prompt sources enabled: ${activeSourceCount}` : 'System prompt disabled'
					}
				>
					<span className="min-w-0 truncate text-center text-xs font-normal">System Prompt</span>

					{activeSourceCount > 0 ? (
						<>
							<span className="bg-success/15 text-success rounded-full px-1.5 py-0.5 text-[10px]">
								{activeSourceCount}
							</span>
							<FiCheck size={16} className="m-0 shrink-0 p-0" />
						</>
					) : (
						<FiX size={16} className="m-0 shrink-0 p-0" />
					)}

					{open ? (
						<FiChevronDown size={16} className="ml-1 shrink-0 xl:ml-2" />
					) : (
						<FiChevronUp size={16} className="ml-1 shrink-0 xl:ml-2" />
					)}
				</PopoverDisclosure>

				<Popover
					store={popover}
					gutter={4}
					portal={false}
					className="border-base-300 bg-base-100 z-50 mt-1 max-h-80 max-w-2xl min-w-lg overflow-y-auto rounded-xl border p-2 text-xs shadow-lg outline-none"
				>
					<div className="mb-2 px-1 text-xs opacity-70">
						Add/Fork creates a prompt template. <br />
						{hasModelDefaultPrompt
							? 'Active sources are concatenated in this order: model default, then selected saved prompts.'
							: ''}
						Bundles cannot be created here; use the Prompt Bundles page. <br />
						<br />
					</div>

					{hasModelDefaultPrompt ? (
						<div
							className="border-base-300 mb-2 rounded-lg border p-2"
							data-prompt={modelDefaultPrompt}
							onFocus={e => {
								showPromptTooltip(e.currentTarget);
							}}
							onBlur={hidePromptTooltip}
							onMouseEnter={e => {
								showPromptTooltip(e.currentTarget);
							}}
							onMouseLeave={hidePromptTooltip}
						>
							<div className="mb-1 flex items-start gap-2">
								<input
									type="checkbox"
									className="checkbox checkbox-xs mt-0.5 rounded"
									checked={includeModelDefault}
									onChange={e => {
										onToggleModelDefault(e.target.checked);
									}}
								/>
								<div className="min-w-0 flex-1">
									<div className="flex items-center gap-2">
										<span className="font-medium">Model default</span>
										<span className="badge badge-ghost badge-xs">per selected model</span>
									</div>
								</div>
							</div>
						</div>
					) : null}

					<div className="mb-1 px-1 text-xs font-medium opacity-70">Saved prompts</div>
					{loading ? (
						<div className="m-0 flex cursor-default items-center justify-between rounded-md px-2 py-2 text-xs opacity-70">
							<span>Loading system prompts…</span>
						</div>
					) : error ? (
						<div className="text-error m-0 flex cursor-default items-center justify-between rounded-md px-2 py-2 text-xs">
							<span>{error}</span>
						</div>
					) : prompts.length > 0 ? (
						<div className="space-y-1">
							{prompts.map(item => {
								const isSelected = selectedKeySet.has(item.identityKey);

								return (
									<div
										key={item.identityKey}
										data-prompt={item.prompt}
										className="hover:bg-base-200 border-base-300 flex items-start gap-2 rounded-lg border p-2 transition-colors"
										onFocus={e => {
											showPromptTooltip(e.currentTarget);
										}}
										onBlur={hidePromptTooltip}
										onMouseEnter={e => {
											showPromptTooltip(e.currentTarget);
										}}
										onMouseLeave={hidePromptTooltip}
									>
										<input
											type="checkbox"
											className="checkbox checkbox-xs mt-0.5 rounded"
											checked={isSelected}
											onChange={() => {
												onTogglePrompt(item.identityKey);
											}}
										/>

										<button
											type="button"
											className="min-w-0 flex-1 text-left"
											onClick={() => {
												onTogglePrompt(item.identityKey);
											}}
										>
											<div className="truncate text-xs font-medium">{item.displayName}</div>
											<div className="mt-1 flex items-center gap-2 text-[10px] opacity-70">
												<span>{item.bundleDisplayName}</span>
												<span>•</span>
												<span>{item.templateSlug}</span>
												<span>•</span>
												<span>{item.templateVersion}</span>
												<span>•</span>
												<span>{item.role === PromptRoleEnum.Developer ? 'developer' : 'system'}</span>
											</div>
										</button>

										<div className="ml-2 flex shrink-0 gap-1">
											<button
												type="button"
												className="btn btn-ghost btn-xs"
												title="Fork"
												disabled={!hasWritableCustomBundle}
												onClick={() => {
													hidePromptTooltip();

													openForkModal(item);
												}}
											>
												<FiGitBranch size={12} />
											</button>
										</div>
									</div>
								);
							})}
						</div>
					) : (
						<div className="m-0 flex cursor-default items-center justify-between rounded-md px-2 py-2 text-xs opacity-70">
							<span>No saved prompts</span>
						</div>
					)}

					<div className="border-neutral/20 mt-2 border-t pt-2 text-xs">
						<div className="flex items-center justify-between gap-2 p-1">
							<button
								type="button"
								className="btn btn-ghost btn-xs rounded-lg"
								disabled={!hasWritableCustomBundle}
								onClick={() => {
									openAddModal();
								}}
							>
								<FiPlus size={14} className="mr-1" /> Add
							</button>

							<button
								type="button"
								className="btn btn-ghost btn-xs rounded-lg"
								onClick={() => {
									onClearSelected();
									setIsOpen(false);
								}}
								title="Clear all selected prompt sources"
								disabled={!includeModelDefault && selectedPromptKeys.length === 0}
							>
								<FiX size={14} className="mr-1" /> Clear selected
							</button>
						</div>
						{!hasWritableCustomBundle ? (
							<div className="text-warning px-1 pt-1 text-xs">
								No enabled custom bundle is available for Add/Fork. Create or enable one in Prompt Bundles.
							</div>
						) : null}
					</div>
				</Popover>

				<Tooltip
					store={promptTooltip}
					portal
					className="rounded-box bg-base-100 text-base-content border-base-300 max-w-xl border p-2 text-xs whitespace-pre-wrap shadow-xl"
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
		</div>
	);
}
