import type { Dispatch, SetStateAction } from 'react';
import { useCallback, useMemo, useState } from 'react';

import { FiCheck, FiChevronDown, FiChevronUp, FiGitBranch, FiPlus, FiTrash, FiX } from 'react-icons/fi';

import { Popover, PopoverDisclosure, Tooltip, usePopoverStore, useStoreState, useTooltipStore } from '@ariakit/react';

import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';

import { SystemPromptAddModal } from '@/chats/assitantcontexts/system_prompt_add_modal';
import { countEnabledSystemPromptSources } from '@/chats/assitantcontexts/system_prompt_utils';
import type { SystemPromptItem } from '@/chats/assitantcontexts/use_system_prompts';

type SystemPromptDropdownProps = {
	prompts: SystemPromptItem[];
	selectedPromptIds: string[];
	modelDefaultPrompt: string;
	includeModelDefault: boolean;
	onTogglePrompt: (id: string) => void;
	onToggleModelDefault: (next: boolean) => void;
	onAddPrompt: (prompt: string) => void;
	onDeletePrompt: (id: string) => void;
	onClearSelected: () => void;
	isOpen: boolean;
	setIsOpen: Dispatch<SetStateAction<boolean>>;
};

function buildPromptPreview(prompt: string): string {
	const trimmed = prompt.trim();
	if (!trimmed) return '(empty)';
	return trimmed.length > 120 ? `${trimmed.slice(0, 120)}…` : trimmed;
}

export function SystemPromptDropdown({
	prompts,
	selectedPromptIds,
	modelDefaultPrompt,
	includeModelDefault,
	onTogglePrompt,
	onToggleModelDefault,
	onAddPrompt,
	onDeletePrompt,
	onClearSelected,
	isOpen,
	setIsOpen,
}: SystemPromptDropdownProps) {
	const [isComposerOpen, setIsComposerOpen] = useState(false);
	const [composerTitle, setComposerTitle] = useState('Add System Prompt');
	const [composerInitialValue, setComposerInitialValue] = useState('');
	const [itemPendingDelete, setItemPendingDelete] = useState<SystemPromptItem | null>(null);

	const hasModelDefaultPrompt = modelDefaultPrompt.trim().length > 0;

	const promptsById = useMemo(() => new Map(prompts.map(item => [item.id, item])), [prompts]);

	const activeSourceCount = useMemo(
		() =>
			countEnabledSystemPromptSources({
				modelDefaultPrompt,
				includeModelDefault,
				selectedPromptIds,
				promptsById,
			}),
		[includeModelDefault, modelDefaultPrompt, promptsById, selectedPromptIds]
	);

	const popover = usePopoverStore({
		open: isOpen,
		setOpen: setIsOpen,
		placement: 'top-start',
	});

	const open = useStoreState(popover, 'open');

	const selectedIdSet = useMemo(() => new Set(selectedPromptIds), [selectedPromptIds]);
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

	return (
		<div className="flex w-full justify-center">
			<div className="relative w-full">
				<PopoverDisclosure
					store={popover}
					className="btn btn-xs text-neutral-custom w-full flex-1 items-center overflow-hidden border-none text-center text-nowrap shadow-none"
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
						<FiChevronDown size={16} className="ml-2 shrink-0" />
					) : (
						<FiChevronUp size={16} className="ml-2 shrink-0" />
					)}
				</PopoverDisclosure>

				<Popover
					store={popover}
					gutter={4}
					portal={false}
					className="border-base-300 bg-base-100 z-50 mt-1 max-h-80 max-w-xl min-w-md overflow-y-auto rounded-xl border p-2 text-xs shadow-lg outline-none"
				>
					<div className="mb-2 px-1 text-[11px] font-medium opacity-70">
						Active sources are concatenated in this order: model default, then selected saved prompts.
					</div>

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
								checked={hasModelDefaultPrompt && includeModelDefault}
								disabled={!hasModelDefaultPrompt}
								onChange={e => {
									onToggleModelDefault(e.target.checked);
								}}
							/>
							<div className="min-w-0 flex-1">
								<div className="flex items-center gap-2">
									<span className="font-medium">Model default</span>
									<span className="badge badge-ghost badge-xs">per selected model</span>
								</div>
								<div className="mt-1 text-[11px] whitespace-pre-wrap opacity-75">
									{hasModelDefaultPrompt
										? buildPromptPreview(modelDefaultPrompt)
										: 'This model has no default system prompt.'}
								</div>
							</div>
						</div>
					</div>

					<div className="mb-1 px-1 text-[11px] font-medium opacity-70">Saved prompts</div>

					{prompts.length > 0 ? (
						<div className="space-y-1">
							{prompts.map(item => {
								const isSelected = selectedIdSet.has(item.id);

								return (
									<div
										key={item.id}
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
												onTogglePrompt(item.id);
											}}
										/>

										<button
											type="button"
											className="min-w-0 flex-1 text-left"
											onClick={() => {
												onTogglePrompt(item.id);
											}}
										>
											<div className="truncate text-xs font-medium">{item.title}</div>
											<div className="mt-1 text-[11px] whitespace-pre-wrap opacity-75">
												{buildPromptPreview(item.prompt)}
											</div>
										</button>

										<div className="ml-2 flex shrink-0 gap-1">
											<button
												type="button"
												className="btn btn-ghost btn-xs"
												title="Fork"
												onClick={() => {
													hidePromptTooltip();

													setComposerTitle('Fork System Prompt');
													setComposerInitialValue(item.prompt);
													setIsOpen(false);
													requestAnimationFrame(() => {
														setIsComposerOpen(true);
													});
												}}
											>
												<FiGitBranch size={12} />
											</button>

											<button
												type="button"
												className="btn btn-ghost btn-xs"
												title="Delete"
												onClick={() => {
													hidePromptTooltip();
													setItemPendingDelete(item);
													setIsOpen(false);
												}}
											>
												<FiTrash size={12} />
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
								onClick={() => {
									setComposerTitle('Add System Prompt');
									setComposerInitialValue('');
									setIsOpen(false);
									requestAnimationFrame(() => {
										setIsComposerOpen(true);
									});
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
								disabled={!includeModelDefault && selectedPromptIds.length === 0}
							>
								<FiX size={14} className="mr-1" /> Clear selected
							</button>
						</div>
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
					title={composerTitle}
					initialValue={composerInitialValue}
					promptsForCopy={prompts}
					onClose={() => {
						setIsComposerOpen(false);
						setComposerInitialValue('');
					}}
					onSave={value => {
						onAddPrompt(value);
						setIsComposerOpen(false);
						setComposerInitialValue('');
					}}
				/>

				<DeleteConfirmationModal
					isOpen={!!itemPendingDelete}
					onClose={() => {
						setItemPendingDelete(null);
					}}
					onConfirm={() => {
						if (itemPendingDelete) {
							onDeletePrompt(itemPendingDelete.id);
						}
						setItemPendingDelete(null);
					}}
					title="Delete saved prompt?"
					message={
						itemPendingDelete
							? `This will delete "${itemPendingDelete.title}" from the saved system prompt library.`
							: 'This will delete the saved system prompt.'
					}
					confirmButtonText="Delete"
				/>
			</div>
		</div>
	);
}
