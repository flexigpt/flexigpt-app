import type { Dispatch, SetStateAction } from 'react';

import { FiCheck, FiChevronDown, FiChevronUp } from 'react-icons/fi';

import { Select, SelectItem, SelectPopover, useSelectStore, useStoreState } from '@ariakit/react';

import type { IncludePreviousMessages } from '@/spec/modelpreset';

type PreviousMessagesDropdownProps = {
	value: IncludePreviousMessages;
	setValue: Dispatch<SetStateAction<IncludePreviousMessages>>;
	isOpen: boolean;
	setIsOpen: Dispatch<SetStateAction<boolean>>;
};

const OPTIONS: IncludePreviousMessages[] = ['all', 0, 1, 2, 3, 5, 10];

function valueToKey(value: IncludePreviousMessages): string {
	return value === 'all' ? 'all' : String(value);
}

function keyToValue(key: string): IncludePreviousMessages {
	return key === 'all' ? 'all' : Math.max(0, Number.parseInt(key, 10) || 0);
}

function displayCount(value: IncludePreviousMessages): string {
	if (value === 'all') return 'All';
	if (value === 0) return 'No';
	return String(value);
}

function displayLabel(value: IncludePreviousMessages): string {
	return `${displayCount(value)} msgs`;
}

function maybeParseCustomValue(rawValue: string): IncludePreviousMessages | undefined {
	const trimmed = rawValue.trim();
	if (!trimmed) {
		return undefined;
	}
	const parsed = Number.parseInt(trimmed, 10);
	if (Number.isNaN(parsed) || parsed < 0) {
		return 0;
	}

	return parsed;
}

export function PreviousMessagesDropdown({ value, setValue, isOpen, setIsOpen }: PreviousMessagesDropdownProps) {
	const select = useSelectStore({
		value: valueToKey(value),
		setValue: nextValue => {
			if (typeof nextValue !== 'string') return;
			setValue(keyToValue(nextValue));
		},
		open: isOpen,
		setOpen: setIsOpen,
		placement: 'top-start',
		focusLoop: true,
	});

	const open = useStoreState(select, 'open');
	const commitCustomValue = (rawValue: string) => {
		const nextValue = maybeParseCustomValue(rawValue);
		if (nextValue === undefined) {
			setIsOpen(false);
			return;
		}

		setValue(nextValue);
		setIsOpen(false);
	};
	return (
		<div className="flex w-full justify-center">
			<div className="relative w-full">
				<Select
					store={select}
					className="btn btn-xs text-neutral-custom w-full flex-1 items-center overflow-hidden border-none text-center text-nowrap shadow-none"
					title="How many previous messages to include in addition to the new user message"
				>
					<span className="min-w-0 truncate text-center text-xs font-normal">Include Prev: {displayLabel(value)}</span>
					{open ? (
						<FiChevronDown size={16} className="ml-2 shrink-0" />
					) : (
						<FiChevronUp size={16} className="ml-2 shrink-0" />
					)}
				</Select>

				<SelectPopover
					store={select}
					portal={false}
					gutter={4}
					autoFocusOnShow
					sameWidth
					className="border-base-300 bg-base-100 z-50 mt-1 max-h-72 w-full overflow-y-auto rounded-xl border p-1 text-xs shadow-lg outline-none"
				>
					{OPTIONS.map(option => {
						const key = valueToKey(option);
						return (
							<SelectItem
								key={key}
								value={key}
								className="hover:bg-base-200 data-active-item:bg-base-300 m-0 flex cursor-pointer items-center justify-between rounded-md px-2 py-1 text-xs transition-colors outline-none"
							>
								<span>{displayLabel(option)}</span>
								{value === option && <FiCheck />}
							</SelectItem>
						);
					})}

					<div className="border-neutral/20 mt-2 border-t pt-2 text-xs">
						<label className="tooltip tooltip-top w-full border-none outline-none">
							<div className="tooltip-content">
								<div className="text-xs">Custom number of previous messages to include (0 or greater)</div>
							</div>
							<input
								key={valueToKey(value)}
								data-disable-chat-shortcuts="true"
								type="text"
								name="include-previous-messages"
								className="input input-xs w-full"
								placeholder="Custom previous message count"
								defaultValue={value === 'all' ? '' : String(value)}
								spellCheck="false"
								onBlur={e => {
									commitCustomValue(e.currentTarget.value);
								}}
								onKeyDown={e => {
									if (e.key === 'Enter') {
										e.preventDefault();
										commitCustomValue(e.currentTarget.value);
									}
								}}
							/>
						</label>
					</div>
				</SelectPopover>
			</div>
		</div>
	);
}
