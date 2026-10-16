import { Menu, MenuButton, MenuItem, useMenuStore, useStoreState } from '@ariakit/react';

import { ReasoningLevel } from '@/spec/inference';
import { getReasoningLevelDisplayName } from '@/spec/inference_labels';

import {
	actionTriggerChipButtonClasses,
	ActionTriggerChipContent,
	actionTriggerMenuCompactClasses,
	actionTriggerMenuItemClasses,
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
	const menu = useMenuStore({ placement: 'top', focusLoop: true });

	const open = useStoreState(menu, 'open');

	return (
		<div className="flex w-full justify-center">
			<div className="relative w-full">
				<HoverTip content="Set reasoning level" placement="top" wrapperElement="div" wrapperClassName="w-full">
					<MenuButton store={menu} className={`${actionTriggerChipButtonClasses} w-full flex-1 justify-center`}>
						<ActionTriggerChipContent
							label={`Reasoning: ${getReasoningLevelDisplayName(reasoningLevel)}`}
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

					{options
						.filter(level => level !== reasoningLevel)
						.map(level => (
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

					<ComposerCurrentSettingPanel
						label="Current reasoning setting"
						value={getReasoningLevelDisplayName(reasoningLevel)}
						isDefault={reasoningLevel === undefined}
						onUseDefault={() => {
							setReasoningLevel(undefined);
						}}
					/>
				</Menu>
			</div>
		</div>
	);
}
