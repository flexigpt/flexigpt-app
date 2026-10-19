import type { Dispatch, SetStateAction } from 'react';
import { useState } from 'react';

import { Menu, MenuButton, MenuItem, useMenuStore, useStoreState } from '@ariakit/react';

import type { IncludePreviousMessages } from '@/spec/model';

import {
	actionTriggerChipButtonClasses,
	ActionTriggerChipContent,
	ActionTriggerMenuBody,
	actionTriggerMenuCompactClasses,
	actionTriggerMenuFieldCardClasses,
	actionTriggerMenuFieldInputClasses,
	ActionTriggerMenuHeader,
	actionTriggerMenuHeaderRowClasses,
	actionTriggerMenuItemClasses,
	actionTriggerMenuMetaClasses,
} from '@/components/action_trigger_chip';
import { ComposerCurrentSettingPanel } from '@/components/composer_dropdown_selected_panel';
import { HoverTip } from '@/components/hover_tip';

interface PreviousMessagesDropdownProps {
	value: IncludePreviousMessages;
	setValue: Dispatch<SetStateAction<IncludePreviousMessages>>;
}

const OPTIONS: IncludePreviousMessages[] = [1, 2, 3, 0, 'all'];

function valueToKey(value: IncludePreviousMessages): string {
	return value === 'all' ? 'all' : String(value);
}

function displayButtonValue(value: IncludePreviousMessages): string {
	if (value === 'all') {
		return 'All';
	}
	if (value === 0) {
		return 'Current';
	}
	return `${value} turn${value === 1 ? '' : 's'}`;
}

function displayOptionLabel(value: IncludePreviousMessages): string {
	if (value === 'all') {
		return 'All previous user turns';
	}
	if (value === 0) {
		return 'Current user turn only';
	}
	return `${value} previous user turn${value === 1 ? '' : 's'}`;
}

export function PreviousMessagesDropdown({ value, setValue }: PreviousMessagesDropdownProps) {
	const menu = useMenuStore({ placement: 'top', focusLoop: true });

	const open = useStoreState(menu, 'open');
	const availableOptions = OPTIONS.filter(option => option !== value);
	const [customError, setCustomError] = useState<string | null>(null);
	const isOverride = value !== 0;
	const triggerTone = open
		? 'border-base-300 bg-base-300/60'
		: isOverride
			? 'border-secondary/50 bg-secondary/10 hover:bg-secondary/15'
			: '';

	const commitCustomValue = (rawValue: string) => {
		const trimmed = rawValue.trim();
		if (!trimmed) {
			setCustomError(null);
			return false;
		}

		if (!/^\d+$/.test(trimmed)) {
			setCustomError('Enter a whole number of previous user turns.');
			return false;
		}

		const nextValue = Number(trimmed);
		if (!Number.isSafeInteger(nextValue)) {
			setCustomError('Enter a safe whole number of previous user turns.');
			return false;
		}

		setValue(nextValue);
		setCustomError(null);
		return true;
	};

	return (
		<div className="flex w-full justify-center">
			<div className="relative w-full">
				<HoverTip
					placement="top"
					wrapperElement="div"
					wrapperClassName="w-full"
					content="Choose prior user-turn history."
				>
					<MenuButton
						store={menu}
						className={`${actionTriggerChipButtonClasses} w-full flex-1 justify-center ${triggerTone}`}
					>
						<ActionTriggerChipContent
							label={`History: ${displayButtonValue(value)}`}
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
					className={actionTriggerMenuCompactClasses}
				>
					<ActionTriggerMenuHeader>
						<div className={actionTriggerMenuHeaderRowClasses}>
							<div className="min-w-0 flex-1 truncate text-xs font-semibold">Previous user turns</div>
							<span className={actionTriggerMenuMetaClasses}>{availableOptions.length} presets</span>
						</div>
					</ActionTriggerMenuHeader>

					<ActionTriggerMenuBody className="space-y-2">
						<div className="space-y-1">
							{availableOptions.map(option => {
								const key = valueToKey(option);
								return (
									<MenuItem
										key={key}
										className={`${actionTriggerMenuItemClasses} justify-between`}
										onClick={() => {
											setCustomError(null);
											setValue(option);
										}}
									>
										<span>{displayOptionLabel(option)}</span>
									</MenuItem>
								);
							})}
						</div>

						<div className={actionTriggerMenuFieldCardClasses}>
							<label htmlFor="include-previous-messages" className="block text-xs font-semibold">
								Custom number of turns
							</label>
							<div className="text-base-content/60 mt-1 mb-2 text-[11px]">
								Enter 0 or a whole number of previous pure user turns.
							</div>
							<input
								id="include-previous-messages"
								key={valueToKey(value)}
								data-disable-chat-shortcuts="true"
								type="text"
								name="include-previous-messages"
								inputMode="numeric"
								autoComplete="off"
								className={`${actionTriggerMenuFieldInputClasses} ${customError ? 'input-error' : ''}`}
								placeholder="For example: 4"
								defaultValue={value === 'all' ? '' : String(value)}
								spellCheck={false}
								aria-invalid={customError ? 'true' : undefined}
								aria-describedby={customError ? 'previous-messages-error' : undefined}
								onChange={() => {
									setCustomError(null);
								}}
								onClick={event => {
									event.stopPropagation();
								}}
								onBlur={event => {
									commitCustomValue(event.currentTarget.value);
								}}
								onKeyDown={event => {
									if (event.key !== 'Escape') {
										event.stopPropagation();
									}
									if (event.key === 'Enter') {
										event.preventDefault();
										if (commitCustomValue(event.currentTarget.value)) {
											menu.hide();
										}
									}
								}}
							/>
							{customError ? (
								<div id="previous-messages-error" role="alert" className="text-error mt-1 text-[11px]">
									{customError}
								</div>
							) : null}
						</div>
					</ActionTriggerMenuBody>

					<ComposerCurrentSettingPanel
						label="Current history"
						value={displayOptionLabel(value)}
						help="This controls how many earlier pure user turns are sent with the current message. Turns containing tool output are not included."
						isDefault={value === 0}
						onUseDefault={() => {
							setCustomError(null);
							setValue(0);
						}}
						useDefaultLabel="Current turn only"
					/>
				</Menu>
			</div>
		</div>
	);
}
