import type { Dispatch, SetStateAction } from 'react';

import { FiCheck } from 'react-icons/fi';

import { Menu, MenuButton, MenuItem, useMenuStore, useStoreState } from '@ariakit/react';

import type { IncludePreviousMessages } from '@/spec/modelpreset';

import {
	actionTriggerChipButtonClasses,
	ActionTriggerChipContent,
	actionTriggerMenuCompactClasses,
	actionTriggerMenuItemClasses,
} from '@/components/action_trigger_chip';
import { HoverTip } from '@/components/ariakit_hover_tip';

type PreviousMessagesDropdownProps = {
	value: IncludePreviousMessages;
	setValue: Dispatch<SetStateAction<IncludePreviousMessages>>;
};

const OPTIONS: IncludePreviousMessages[] = [1, 2, 3, 0, 'all'];

function valueToKey(value: IncludePreviousMessages): string {
	return value === 'all' ? 'all' : String(value);
}

function displayButtonValue(value: IncludePreviousMessages): string {
	if (value === 'all') return 'All';
	return String(value);
}

function displayOptionLabel(value: IncludePreviousMessages): string {
	if (value === 'all') return 'All previous user turns';
	if (value === 0) return 'Current user turn only';
	return `${value} previous user turn${value === 1 ? '' : 's'}`;
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

export function PreviousMessagesDropdown({ value, setValue }: PreviousMessagesDropdownProps) {
	const menu = useMenuStore({ placement: 'top', focusLoop: true });

	const open = useStoreState(menu, 'open');
	const commitCustomValue = (rawValue: string) => {
		const nextValue = maybeParseCustomValue(rawValue);
		if (nextValue === undefined) {
			return;
		}

		setValue(nextValue);
	};

	return (
		<div className="flex w-full justify-center">
			<div className="relative w-full">
				<HoverTip
					placement="top"
					wrapperElement="div"
					wrapperClassName="w-full"
					content={
						<div className="space-y-1">
							<p>
								- Send "N" previous pure user turns excluding current message. <br />
								- A pure user turn is one without any tool outputs. <br />
							</p>
						</div>
					}
				>
					<MenuButton store={menu} className={`${actionTriggerChipButtonClasses} w-full flex-1 justify-center`}>
						<ActionTriggerChipContent
							label={`Prev user turns: ${displayButtonValue(value)}`}
							open={open}
							labelClassName="min-w-0 truncate text-center text-xs font-normal"
							className="w-full justify-center"
						/>
					</MenuButton>
				</HoverTip>

				<Menu
					store={menu}
					portal
					gutter={8}
					overflowPadding={8}
					autoFocusOnShow
					className={`${actionTriggerMenuCompactClasses} text-xs`}
				>
					{OPTIONS.map(option => {
						const key = valueToKey(option);
						return (
							<MenuItem
								key={key}
								className={`${actionTriggerMenuItemClasses} justify-between`}
								onClick={() => {
									setValue(option);
								}}
							>
								<span>{displayOptionLabel(option)}</span>
								{value === option ? <FiCheck /> : null}
							</MenuItem>
						);
					})}

					<div className="border-neutral/20 mt-2 border-t pt-2 text-xs">
						<input
							key={valueToKey(value)}
							data-disable-chat-shortcuts="true"
							type="text"
							name="include-previous-messages"
							className="input input-xs w-full"
							placeholder="<n> previous user turns"
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
					</div>
				</Menu>
			</div>
		</div>
	);
}
