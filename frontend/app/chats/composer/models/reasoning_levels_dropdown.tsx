import { MenuButton, MenuItem, useMenuStore, useStoreState } from '@ariakit/react';

import { ReasoningLevel } from '@/spec/inference';
import { getReasoningLevelDisplayName } from '@/spec/inference_labels';

import {
	actionTriggerChipButtonClasses,
	ActionTriggerChipContent,
	ActionTriggerMenu,
	ActionTriggerMenuBody,
	ActionTriggerMenuHeader,
	actionTriggerMenuHeaderRowClasses,
	actionTriggerMenuItemClasses,
	actionTriggerMenuMetaClasses,
} from '@/components/action_trigger_chip';
import { ComposerCurrentSettingPanel } from '@/components/composer_dropdown_selected_panel';
import { HoverTip } from '@/components/hover_tip';

const COMPOSER_DEFAULT_REASONING_LEVELS: readonly ReasoningLevel[] = [
	ReasoningLevel.None,
	ReasoningLevel.Minimal,
	ReasoningLevel.Low,
	ReasoningLevel.Medium,
	ReasoningLevel.High,
];

interface SingleReasoningDropdownProps {
	reasoningLevel?: ReasoningLevel;
	levelOptions?: ReasoningLevel[];
	setReasoningLevel: (level?: ReasoningLevel) => void;
}

export function SingleReasoningDropdown({
	reasoningLevel,
	levelOptions,
	setReasoningLevel,
}: SingleReasoningDropdownProps) {
	const options = levelOptions && levelOptions.length > 0 ? levelOptions : COMPOSER_DEFAULT_REASONING_LEVELS;
	const menu = useMenuStore({ placement: 'top-start', focusLoop: true });

	const open = useStoreState(menu, 'open');
	const availableLevels = options.filter(level => level !== reasoningLevel);
	const availableOptionCount = availableLevels.length + (reasoningLevel === undefined ? 0 : 1);
	const triggerTone = open
		? 'border-base-300 bg-base-300/60'
		: reasoningLevel !== undefined
			? 'border-secondary/50 bg-secondary/10 hover:bg-secondary/15'
			: '';

	return (
		<div className="flex w-full justify-center">
			<div className="relative w-full">
				<HoverTip content="Choose reasoning effort." placement="top" wrapperElement="div" wrapperClassName="w-full">
					<MenuButton
						store={menu}
						className={`${actionTriggerChipButtonClasses} w-full flex-1 justify-center ${triggerTone}`}
					>
						<ActionTriggerChipContent
							label={`Reasoning: ${getReasoningLevelDisplayName(reasoningLevel)}`}
							open={open}
							labelClassName="min-w-0 truncate text-center text-xs font-normal"
							className="w-full justify-center"
						/>
					</MenuButton>
				</HoverTip>

				<ActionTriggerMenu store={menu}>
					<ActionTriggerMenuHeader>
						<div className={actionTriggerMenuHeaderRowClasses}>
							<div className="min-w-0 flex-1 truncate text-xs font-semibold">Reasoning level</div>
							<span className={actionTriggerMenuMetaClasses}>
								{availableOptionCount} option{availableOptionCount === 1 ? '' : 's'}
							</span>
						</div>
					</ActionTriggerMenuHeader>

					<ActionTriggerMenuBody className="space-y-1">
						{reasoningLevel !== undefined ? (
							<MenuItem
								className={`${actionTriggerMenuItemClasses} justify-between`}
								onClick={() => {
									setReasoningLevel(undefined);
								}}
							>
								<span>Default</span>
							</MenuItem>
						) : null}

						{availableLevels.map(level => (
							<MenuItem
								key={level}
								className={`${actionTriggerMenuItemClasses} justify-between`}
								onClick={() => {
									setReasoningLevel(level);
								}}
							>
								<span>{getReasoningLevelDisplayName(level)}</span>
							</MenuItem>
						))}
					</ActionTriggerMenuBody>

					<ComposerCurrentSettingPanel
						label="Current reasoning"
						value={getReasoningLevelDisplayName(reasoningLevel)}
						help="Use Default to defer to the model or provider."
						isDefault={reasoningLevel === undefined}
						onUseDefault={() => {
							setReasoningLevel(undefined);
						}}
					/>
				</ActionTriggerMenu>
			</div>
		</div>
	);
}
