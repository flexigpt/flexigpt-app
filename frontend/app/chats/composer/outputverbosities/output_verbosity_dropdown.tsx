import { FiCheck } from 'react-icons/fi';

import { Menu, MenuButton, MenuItem, useMenuStore, useStoreState } from '@ariakit/react';

import { OutputVerbosity } from '@/spec/inference';

import {
	actionTriggerChipButtonClasses,
	ActionTriggerChipContent,
	actionTriggerMenuCompactClasses,
	actionTriggerMenuItemClasses,
} from '@/components/action_trigger_chip';
import { HoverTip } from '@/components/ariakit_hover_tip';

type OutputVerbosityDropdownProps = {
	verbosity?: OutputVerbosity;
	disabled?: boolean;
	setVerbosity: (v?: OutputVerbosity) => void;
};

const VERBOSITY_OPTIONS: Array<{ label: string; value?: OutputVerbosity }> = [
	{ label: 'Default', value: undefined },
	{ label: 'Low', value: OutputVerbosity.Low },
	{ label: 'Medium', value: OutputVerbosity.Medium },
	{ label: 'High', value: OutputVerbosity.High },
	{ label: 'Max', value: OutputVerbosity.Max },
];

function labelFor(v?: OutputVerbosity) {
	const l = VERBOSITY_OPTIONS.find(o => o.value === v)?.label ?? '';
	if (l === 'Default') {
		return '';
	}
	return ': ' + l;
}

export function OutputVerbosityDropdown({ verbosity, disabled = false, setVerbosity }: OutputVerbosityDropdownProps) {
	const menu = useMenuStore({ placement: 'top', focusLoop: true });

	const open = useStoreState(menu, 'open');
	const tooltipText = disabled
		? 'Effort/Verbosity not supported by this model/provider'
		: 'Set effort/verbosity for model output';

	return (
		<div className="flex w-full justify-center">
			<div className="relative w-full">
				<HoverTip content={tooltipText} placement="top" wrapperElement="div" wrapperClassName="w-full">
					<MenuButton
						store={menu}
						disabled={disabled}
						className={`${actionTriggerChipButtonClasses} w-full flex-1 justify-center ${disabled ? 'cursor-not-allowed opacity-50' : ''}`}
					>
						<ActionTriggerChipContent
							label={`Effort${labelFor(verbosity)}`}
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
					{VERBOSITY_OPTIONS.map(opt => (
						<MenuItem
							key={opt.value ?? '__default__'}
							className={`${actionTriggerMenuItemClasses} justify-between`}
							onClick={() => {
								setVerbosity(opt.value);
							}}
						>
							<span>{opt.label}</span>
							{(verbosity ?? undefined) === (opt.value ?? undefined) ? <FiCheck /> : null}
						</MenuItem>
					))}
				</Menu>
			</div>
		</div>
	);
}
