import type { Dispatch, SetStateAction } from 'react';

import { FiCheck, FiChevronDown, FiChevronUp, FiX } from 'react-icons/fi';

import { Popover, PopoverDisclosure, usePopoverStore, useStoreState } from '@ariakit/react';

import type { AssistantPresetOptionItem } from '@/chats/inputarea/assitantcontexts/assistant_preset_runtime';

type AssistantPresetDropdownProps = {
	presetOptions: AssistantPresetOptionItem[];
	selectedPresetKey: string | null;
	selectedPreset: AssistantPresetOptionItem | null;
	loading: boolean;
	error: string | null;
	actionError: string | null;
	isApplying: boolean;
	isOpen: boolean;
	setIsOpen: Dispatch<SetStateAction<boolean>>;
	onSelectPreset: (presetKey: string) => Promise<boolean>;
	onClearPreset: () => void;
};

export function AssistantPresetDropdown({
	presetOptions,
	selectedPresetKey,
	selectedPreset,
	loading,
	error,
	actionError,
	isApplying,
	isOpen,
	setIsOpen,
	onSelectPreset,
	onClearPreset,
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
				<PopoverDisclosure
					store={popover}
					className="btn btn-xs text-neutral-custom w-full flex-1 items-center overflow-hidden border-none p-0 text-center text-nowrap shadow-none"
					title={triggerTitle}
				>
					<span className="min-w-0 truncate text-center text-xs font-normal">{triggerLabel}</span>
					{selectedPreset ? <FiCheck size={14} className="ml-1 shrink-0" /> : null}
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
					className="border-base-300 bg-base-100 z-50 mt-1 max-h-80 max-w-xl min-w-96 overflow-y-auto rounded-xl border p-2 text-xs shadow-lg outline-none"
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
								const isSelected = option.key === selectedPresetKey;
								const isDisabled = isApplying || !option.isSelectable;
								return (
									<button
										key={option.key}
										type="button"
										disabled={isDisabled}
										className={`hover:bg-base-200 border-base-300 flex w-full items-start gap-2 rounded-lg border p-2 text-left transition-colors ${
											isSelected ? 'bg-base-200' : ''
										} ${
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

										{!option.isSelectable ? (
											<span className="badge badge-warning badge-xs shrink-0">Unavailable</span>
										) : null}
									</button>
								);
							})}
						</div>
					)}

					{selectedPresetKey ? (
						<div className="border-neutral/20 mt-2 border-t pt-2">
							<button
								type="button"
								className="btn btn-ghost btn-xs rounded-lg"
								disabled={isApplying}
								onClick={() => {
									onClearPreset();
									setIsOpen(false);
								}}
								title="Clear assistant preset selection without changing current values"
							>
								<FiX size={14} className="mr-1" />
								Clear selection
							</button>
						</div>
					) : null}
				</Popover>
			</div>
		</div>
	);
}
