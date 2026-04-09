import { FiCheck } from 'react-icons/fi';

import { Menu, MenuButton, MenuItem, useMenuStore, useStoreState } from '@ariakit/react';

import {
	actionTriggerChipButtonClasses,
	ActionTriggerChipContent,
	actionTriggerMenuCompactClasses,
	actionTriggerMenuItemClasses,
} from '@/components/action_trigger_chip';
import { HoverTip } from '@/components/ariakit_hover_tip';

const defaultTemperatureOptions = [0.0, 0.1, 0.5, 1.0];

type TemperatureDropdownProps = {
	temperature: number;
	setTemperature: (t: number) => void;
};

export function TemperatureDropdown({ temperature, setTemperature }: TemperatureDropdownProps) {
	const menu = useMenuStore({ placement: 'top', focusLoop: true });
	const open = useStoreState(menu, 'open');

	function clampTemperature(rawValue: string) {
		let val = parseFloat(rawValue);
		if (isNaN(val)) {
			val = 0.1;
		}
		val = Math.max(0, Math.min(1, val));
		setTemperature(val);
	}

	return (
		<div className="flex w-full justify-center">
			<div className="relative w-full">
				<HoverTip content="Set temperature" placement="top" wrapperElement="div" wrapperClassName="w-full">
					<MenuButton store={menu} className={`${actionTriggerChipButtonClasses} w-full flex-1 justify-center`}>
						<ActionTriggerChipContent
							label={`Temperature: ${temperature.toFixed(2)}`}
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
					className={`${actionTriggerMenuCompactClasses} p-4 text-xs`}
				>
					{defaultTemperatureOptions.map(tempVal => (
						<MenuItem
							key={tempVal}
							className={`${actionTriggerMenuItemClasses} justify-between`}
							onClick={() => {
								setTemperature(tempVal);
							}}
						>
							<span>{tempVal.toFixed(1)}</span>
							{temperature.toFixed(1) === tempVal.toFixed(1) ? <FiCheck /> : null}
						</MenuItem>
					))}

					<div className="border-neutral/20 mt-2 border-t pt-2 text-xs">
						<div className="text-base-content/70 mb-1 text-xs">Custom (0.0 - 1.0)</div>
						<input
							key={temperature}
							data-disable-chat-shortcuts="true"
							type="text"
							name="temperature"
							className="input input-xs w-full"
							placeholder="Custom value (0.0 - 1.0)"
							defaultValue={temperature.toString()}
							onBlur={e => {
								clampTemperature(e.currentTarget.value);
							}}
							onKeyDown={e => {
								if (e.key === 'Enter') {
									e.preventDefault();
									clampTemperature(e.currentTarget.value);
								}
							}}
							spellCheck="false"
						/>
					</div>
				</Menu>
			</div>
		</div>
	);
}
