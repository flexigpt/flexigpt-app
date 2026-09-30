import { FiCheck } from 'react-icons/fi';

import { Menu, MenuButton, MenuItem, useMenuStore, useStoreState } from '@ariakit/react';

import type { OutputVerbosity } from '@/spec/inference';
import { ProviderSDKType } from '@/spec/inference';

import {
	actionTriggerChipButtonClasses,
	ActionTriggerChipContent,
	actionTriggerMenuCompactClasses,
	actionTriggerMenuItemClasses,
} from '@/components/action_trigger_chip';
import { HoverTip } from '@/components/hover_tip';

import { getOutputVerbosityDisplayName, OUTPUT_VERBOSITY_VALUES } from '@/models/lib/model_option_labels';

interface OutputVerbosityDropdownProps {
	sdkType: ProviderSDKType;
	verbosity?: OutputVerbosity;
	disabled?: boolean;
	setVerbosity: (v?: OutputVerbosity) => void;
}

const VERBOSITY_OPTIONS: Array<OutputVerbosity | undefined> = [undefined, ...OUTPUT_VERBOSITY_VALUES];

function labelFor(v?: OutputVerbosity) {
	if (v === undefined) {
		return '';
	}
	return `: ${getOutputVerbosityDisplayName(v)}`;
}

export function OutputVerbosityDropdown({
	sdkType,
	verbosity,
	disabled = false,
	setVerbosity,
}: OutputVerbosityDropdownProps) {
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
							label={`${sdkType === ProviderSDKType.ProviderSDKTypeAnthropic ? 'Effort' : 'Verbosity'}${labelFor(verbosity)}`}
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
					{VERBOSITY_OPTIONS.map(option => (
						<MenuItem
							key={option ?? '__default__'}
							className={`${actionTriggerMenuItemClasses} justify-between`}
							onClick={() => {
								setVerbosity(option);
							}}
						>
							<span>{getOutputVerbosityDisplayName(option)}</span>
							{verbosity === option ? <FiCheck /> : null}
						</MenuItem>
					))}
				</Menu>
			</div>
		</div>
	);
}
