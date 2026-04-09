import type { Dispatch, SetStateAction } from 'react';

import { FiCheck, FiEye, FiRefreshCcw, FiTrash2 } from 'react-icons/fi';

import { Popover, PopoverDisclosure, usePopoverStore, useStoreState } from '@ariakit/react';

import { ActionTriggerChipContent } from '@/components/action_trigger_chip';
import { HoverTip } from '@/components/ariakit_hover_tip';

import type { AssistantPresetOptionItem } from '@/chats/composer/assistantpresets/assistant_preset_runtime';

type AssistantPresetDropdownProps = {
	presetOptions: AssistantPresetOptionItem[];
	selectedPresetKey: string | null;
	selectedPreset: AssistantPresetOptionItem | null;
	loading: boolean;
	error: string | null;
	actionError: string | null;
	isApplying: boolean;
	basePresetKey: string | null;
	selectedPresetModifiedLabels: string[];
	canResetToBasePreset: boolean;
	isOpen: boolean;
	setIsOpen: Dispatch<SetStateAction<boolean>>;
	onViewPreset: (preset: AssistantPresetOptionItem) => void;
	onReapplySelectedPreset: () => Promise<boolean>;
	onResetToBasePreset: () => Promise<boolean>;
	onSelectPreset: (presetKey: string) => Promise<boolean>;
};

export function AssistantPresetDropdown({
	presetOptions,
	selectedPresetKey,
	selectedPreset,
	loading,
	error,
	actionError,
	isApplying,
	basePresetKey,
	selectedPresetModifiedLabels,
	canResetToBasePreset,
	isOpen,
	setIsOpen,
	onViewPreset,
	onReapplySelectedPreset,
	onResetToBasePreset,
	onSelectPreset,
}: AssistantPresetDropdownProps) {
	const popover = usePopoverStore({
		open: isOpen,
		setOpen: setIsOpen,
		placement: 'top-start',
	});

	const open = useStoreState(popover, 'open');

	const triggerLabel = selectedPreset ? selectedPreset.displayName : 'Assistant';
	const triggerTitle = selectedPreset
		? `${selectedPreset.displayName} — ${selectedPreset.bundleDisplayName}`
		: 'Apply assistant preset';

	return (
		<div className="flex w-full justify-center">
			<div className="relative w-full">
				<HoverTip content={triggerTitle} placement="top" wrapperElement="div" wrapperClassName="w-full">
					<PopoverDisclosure
						store={popover}
						className="btn btn-xs text-neutral-custom w-full flex-1 items-center overflow-hidden border-none p-0 text-center text-nowrap shadow-none"
					>
						<ActionTriggerChipContent
							label={triggerLabel}
							open={open}
							suffix={selectedPreset ? <FiCheck size={14} className="shrink-0" /> : undefined}
							labelClassName="min-w-0 truncate text-center text-xs font-normal"
							className="w-full justify-center"
						/>
					</PopoverDisclosure>
				</HoverTip>

				<Popover
					store={popover}
					gutter={4}
					portal={false}
					className="border-base-300 bg-base-100 z-50 mt-1 max-h-80 max-w-lg min-w-60 overflow-y-auto rounded-xl border p-2 text-xs shadow-lg outline-none"
				>
					<div className="mb-2 px-1 text-xs opacity-70">
						Assistant presets seeds model, instructions, tools, and skills.
					</div>

					{error ? (
						<div className="alert alert-error mb-2 rounded-2xl text-xs">
							<span>{error}</span>
						</div>
					) : null}

					{actionError ? (
						<div className="alert alert-error mb-2 rounded-2xl text-xs">
							<span>{actionError}</span>
						</div>
					) : null}

					{loading ? (
						<div className="m-0 rounded-md px-2 py-2 text-xs opacity-70">Loading assistant presets…</div>
					) : presetOptions.length === 0 ? (
						<div className="m-0 rounded-md px-2 py-2 text-xs opacity-70">No enabled assistant presets available.</div>
					) : (
						<div className="space-y-1">
							{presetOptions.map(option => {
								const isBasePreset = option.key === basePresetKey;
								const isSelected = option.key === selectedPresetKey;
								const isDisabled = isApplying || !option.isSelectable;
								return (
									<div
										key={option.key}
										className={`hover:bg-base-200 border-base-300 flex w-full flex-col rounded-lg border p-2 text-left transition-colors ${
											isSelected ? 'bg-base-200' : ''
										}`}
									>
										<div className="flex w-full items-start gap-2">
											<button
												type="button"
												disabled={isDisabled}
												className={`flex min-w-0 flex-1 items-start gap-2 text-left ${
													isDisabled
														? option.isSelectable
															? 'cursor-wait opacity-70'
															: 'cursor-not-allowed opacity-60'
														: ''
												}`}
												onClick={() => {
													if (isDisabled) return;
													void (async () => {
														const ok = await onSelectPreset(option.key);
														if (ok) {
															setIsOpen(false);
														}
													})();
												}}
												title={
													option.isSelectable
														? option.label
														: `${option.label} — ${option.availabilityReason ?? 'Unavailable'}`
												}
											>
												<div className="pt-0.5">{isSelected ? <FiCheck size={14} /> : <span className="w-3" />}</div>

												<div className="min-w-0 flex-1">
													<div className="truncate text-xs font-medium">{option.displayName}</div>
													<div className="mt-1 flex items-center gap-2 text-[10px] opacity-70">
														<span>{option.bundleDisplayName}</span>
														<span>•</span>
														<span>
															{option.preset.slug}@{option.preset.version}
														</span>
													</div>
													{option.description ? (
														<div className="mt-1 line-clamp-2 text-xs opacity-75">{option.description}</div>
													) : null}
													{!option.isSelectable ? (
														<div className="text-warning mt-1 text-xs">
															{option.availabilityReason ?? 'This preset is not currently available.'}
														</div>
													) : null}
												</div>
											</button>

											<div className="flex shrink-0 items-start gap-1">
												<button
													type="button"
													className="btn btn-ghost btn-xs btn-square rounded-lg"
													onClick={() => {
														onViewPreset(option);
													}}
													title={isSelected ? 'View active assistant preset details' : 'View assistant preset details'}
												>
													<FiEye size={14} />
												</button>

												{!option.isSelectable ? (
													<span className="badge badge-warning badge-xs shrink-0">Unavailable</span>
												) : null}
											</div>
										</div>

										{isSelected ? (
											<div className="border-base-300 mt-2 ml-5 flex flex-wrap items-center justify-between gap-1 border-t p-0 pt-1">
												<div className="flex">
													<span
														className={`badge badge-xs ${selectedPresetModifiedLabels.length > 0 ? 'badge-warning' : 'badge-success'}`}
														title={
															selectedPresetModifiedLabels.length > 0
																? `Modified sections: ${selectedPresetModifiedLabels.join(', ')}`
																: 'Preset-managed sections are currently in sync'
														}
													>
														{selectedPresetModifiedLabels.length > 0 ? 'Modified' : 'In sync'}
													</span>

													{isBasePreset ? <span className="badge badge-ghost badge-xs">Base</span> : null}
												</div>
												<div className="flex">
													<button
														type="button"
														className="btn btn-ghost btn-xs rounded-lg"
														disabled={isApplying}
														onClick={() => {
															void onReapplySelectedPreset();
														}}
														title={
															selectedPresetModifiedLabels.length > 0
																? `Reset preset-managed sections: ${selectedPresetModifiedLabels.join(', ')}`
																: 'Reapply current assistant preset'
														}
													>
														<FiRefreshCcw size={14} className="mr-1" />
														{selectedPresetModifiedLabels.length > 0 ? 'Reset' : 'Reapply'}
													</button>

													<button
														type="button"
														className="btn btn-ghost btn-xs rounded-lg"
														disabled={isApplying || isBasePreset || !canResetToBasePreset}
														onClick={() => {
															void onResetToBasePreset();
														}}
														title={
															isBasePreset
																? 'Base preset is already active'
																: 'Switch back to the base assistant preset'
														}
													>
														<FiTrash2 size={14} className="mr-1" />
														Clear to base
													</button>
												</div>
											</div>
										) : null}
									</div>
								);
							})}
						</div>
					)}
				</Popover>
			</div>
		</div>
	);
}
