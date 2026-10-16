import { Menu, MenuButton, MenuItem, useMenuStore, useStoreState } from '@ariakit/react';

import { OutputVerbosity } from '@/spec/inference';
import { getOutputVerbosityDisplayName } from '@/spec/inference_labels';
import { ProviderSDKType } from '@/spec/provider_sdk';

import {
	actionTriggerChipButtonClasses,
	ActionTriggerChipContent,
	actionTriggerMenuCompactClasses,
	actionTriggerMenuItemClasses,
} from '@/components/action_trigger_chip';
import { ComposerCurrentSettingPanel } from '@/components/composer_dropdown_selected_panel';
import { HoverTip } from '@/components/hover_tip';

interface OutputVerbosityDropdownProps {
	sdkType: ProviderSDKType;
	verbosity?: OutputVerbosity;
	disabled?: boolean;
	setVerbosity: (v?: OutputVerbosity) => void;
}

const VERBOSITY_OPTIONS: Array<OutputVerbosity | undefined> = [undefined, ...Object.values(OutputVerbosity)];

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
	const availableOptions = VERBOSITY_OPTIONS.filter(option => option !== verbosity);
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
					{availableOptions.map(option => (
						<MenuItem
							key={option ?? '__default__'}
							className={`${actionTriggerMenuItemClasses} justify-between`}
							onClick={() => {
								setVerbosity(option);
							}}
						>
							<span>{getOutputVerbosityDisplayName(option)}</span>
						</MenuItem>
					))}

					<ComposerCurrentSettingPanel
						label="Current output setting"
						value={getOutputVerbosityDisplayName(verbosity)}
						isDefault={verbosity === undefined}
						onUseDefault={() => {
							setVerbosity(undefined);
						}}
					/>
				</Menu>
			</div>
		</div>
	);
}
