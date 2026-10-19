import { Menu, MenuButton, MenuItem, useMenuStore, useStoreState } from '@ariakit/react';

import { OutputVerbosity } from '@/spec/inference';
import { getOutputVerbosityDisplayName } from '@/spec/inference_labels';
import { ProviderSDKType } from '@/spec/provider_sdk';

import {
	actionTriggerChipButtonClasses,
	ActionTriggerChipContent,
	ActionTriggerMenuBody,
	actionTriggerMenuCompactClasses,
	ActionTriggerMenuHeader,
	actionTriggerMenuHeaderRowClasses,
	actionTriggerMenuItemClasses,
	actionTriggerMenuMetaClasses,
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

function displayVerbosity(value?: OutputVerbosity) {
	return value === undefined ? 'Default' : getOutputVerbosityDisplayName(value);
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
	const settingName = sdkType === ProviderSDKType.ProviderSDKTypeAnthropic ? 'Effort' : 'Verbosity';
	const isOverride = verbosity !== undefined;
	const triggerTone = open
		? 'border-base-300 bg-base-300/60'
		: isOverride
			? 'border-secondary/50 bg-secondary/10 hover:bg-secondary/15'
			: '';
	const tooltipText = disabled
		? `${settingName} is unavailable for this model/provider.`
		: `Choose ${settingName.toLocaleLowerCase()} for the next response.`;

	return (
		<div className="flex w-full justify-center">
			<div className="relative w-full">
				<HoverTip content={tooltipText} placement="top" wrapperElement="div" wrapperClassName="w-full">
					<MenuButton
						store={menu}
						disabled={disabled}
						className={`${actionTriggerChipButtonClasses} w-full flex-1 justify-center ${triggerTone} ${
							disabled ? 'cursor-not-allowed opacity-50' : ''
						}`}
					>
						<ActionTriggerChipContent
							label={`${settingName}: ${displayVerbosity(verbosity)}`}
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
							<div className="min-w-0 flex-1 truncate text-xs font-semibold">{settingName}</div>
							<span className={actionTriggerMenuMetaClasses}>
								{availableOptions.length} option{availableOptions.length === 1 ? '' : 's'}
							</span>
						</div>
					</ActionTriggerMenuHeader>

					<ActionTriggerMenuBody className="space-y-1">
						{availableOptions.map(option => (
							<MenuItem
								key={option ?? '__default__'}
								className={`${actionTriggerMenuItemClasses} justify-between`}
								onClick={() => {
									setVerbosity(option);
								}}
							>
								<span>{displayVerbosity(option)}</span>
							</MenuItem>
						))}
					</ActionTriggerMenuBody>

					<ComposerCurrentSettingPanel
						label={`Current ${settingName.toLocaleLowerCase()}`}
						value={displayVerbosity(verbosity)}
						help={`Use Default to leave ${settingName.toLocaleLowerCase()} to the selected model or provider. An explicit value applies to the next response.`}
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
