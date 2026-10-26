import { useState } from 'react';

import { MenuButton, MenuItem, useMenuStore, useStoreState } from '@ariakit/react';

import {
	actionTriggerChipButtonClasses,
	ActionTriggerChipContent,
	ActionTriggerMenu,
	ActionTriggerMenuBody,
	actionTriggerMenuFieldCardClasses,
	actionTriggerMenuFieldInputClasses,
	ActionTriggerMenuHeader,
	actionTriggerMenuHeaderRowClasses,
	actionTriggerMenuItemClasses,
	actionTriggerMenuMetaClasses,
} from '@/components/action_trigger_chip';
import { ComposerCurrentSettingPanel } from '@/components/composer_dropdown_selected_panel';
import { HoverTip } from '@/components/hover_tip';

const defaultTemperatureOptions: Array<number | undefined> = [undefined, 0.0, 0.1, 0.5, 1.0];

function displayTemperature(value?: number): string {
	return value === undefined ? 'Default' : value.toFixed(2);
}

interface TemperatureDropdownProps {
	temperature?: number;
	setTemperature: (t?: number) => void;
}

export function TemperatureDropdown({ temperature, setTemperature }: TemperatureDropdownProps) {
	const menu = useMenuStore({ placement: 'top-start', focusLoop: true });
	const open = useStoreState(menu, 'open');
	const availableTemperatureOptions = defaultTemperatureOptions.filter(value => value !== temperature);
	const [customError, setCustomError] = useState<string | null>(null);
	const triggerTone = open
		? 'border-base-300 bg-base-300/60'
		: temperature !== undefined
			? 'border-secondary/50 bg-secondary/10 hover:bg-secondary/15'
			: '';

	function commitTemperature(rawValue: string): boolean {
		const trimmed = rawValue.trim();
		if (!trimmed) {
			setTemperature(undefined);
			setCustomError(null);
			return true;
		}

		const parsed = Number(trimmed);
		if (!Number.isFinite(parsed) || parsed < 0 || parsed > 1) {
			setCustomError('Enter a number from 0.00 to 1.00.');
			return false;
		}

		setTemperature(parsed);
		setCustomError(null);
		return true;
	}

	return (
		<div className="flex w-full justify-center">
			<div className="relative w-full">
				<HoverTip
					content="Control response variability."
					placement="top"
					wrapperElement="div"
					wrapperClassName="w-full"
				>
					<MenuButton
						store={menu}
						className={`${actionTriggerChipButtonClasses} w-full flex-1 justify-center ${triggerTone}`}
					>
						<ActionTriggerChipContent
							label={`Temperature: ${displayTemperature(temperature)}`}
							open={open}
							labelClassName="min-w-0 truncate text-center text-xs font-normal"
							className="w-full justify-center"
						/>
					</MenuButton>
				</HoverTip>

				<ActionTriggerMenu store={menu}>
					<ActionTriggerMenuHeader>
						<div className={actionTriggerMenuHeaderRowClasses}>
							<div className="min-w-0 flex-1 truncate text-xs font-semibold">Temperature</div>
							<span className={actionTriggerMenuMetaClasses}>{availableTemperatureOptions.length} presets</span>
						</div>
					</ActionTriggerMenuHeader>

					<ActionTriggerMenuBody className="space-y-2">
						<div className="space-y-1">
							{availableTemperatureOptions.map(tempVal => (
								<MenuItem
									key={tempVal ?? '__default__'}
									className={`${actionTriggerMenuItemClasses} justify-between`}
									onClick={() => {
										setCustomError(null);
										setTemperature(tempVal);
									}}
								>
									<span>{displayTemperature(tempVal)}</span>
								</MenuItem>
							))}
						</div>

						<div className={actionTriggerMenuFieldCardClasses}>
							<label htmlFor="composer-temperature" className="block text-xs font-semibold">
								Custom temperature
							</label>
							<div className="text-base-content/60 mt-1 mb-2 text-[11px]">Enter a value from 0.00 to 1.00.</div>
							<input
								id="composer-temperature"
								key={temperature ?? '__default__'}
								data-disable-chat-shortcuts="true"
								type="text"
								name="temperature"
								inputMode="decimal"
								autoComplete="off"
								className={`${actionTriggerMenuFieldInputClasses} ${customError ? 'input-error' : ''}`}
								placeholder="For example: 0.50"
								defaultValue={temperature?.toString() ?? ''}
								spellCheck={false}
								aria-invalid={customError ? 'true' : undefined}
								aria-describedby={customError ? 'temperature-error' : undefined}
								onChange={() => {
									setCustomError(null);
								}}
								onClick={event => {
									event.stopPropagation();
								}}
								onBlur={event => {
									commitTemperature(event.currentTarget.value);
								}}
								onKeyDown={event => {
									if (event.key !== 'Escape') {
										event.stopPropagation();
									}
									if (event.key === 'Enter') {
										event.preventDefault();
										if (commitTemperature(event.currentTarget.value)) {
											menu.hide();
										}
									}
								}}
							/>
							{customError ? (
								<div id="temperature-error" role="alert" className="text-error mt-1 text-[11px]">
									{customError}
								</div>
							) : null}
						</div>
					</ActionTriggerMenuBody>

					<ComposerCurrentSettingPanel
						label="Current temperature"
						value={displayTemperature(temperature)}
						help="Lower values are more deterministic. Higher values allow more variation in the response."
						isDefault={temperature === undefined}
						onUseDefault={() => {
							setCustomError(null);
							setTemperature(undefined);
						}}
					/>
				</ActionTriggerMenu>
			</div>
		</div>
	);
}
