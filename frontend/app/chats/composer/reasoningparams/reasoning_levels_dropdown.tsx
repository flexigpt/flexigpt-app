import { FiCheck } from 'react-icons/fi';

import { Menu, MenuButton, MenuItem, useMenuStore, useStoreState } from '@ariakit/react';

import { ReasoningLevel } from '@/spec/inference';

import {
	actionTriggerChipButtonClasses,
	ActionTriggerChipContent,
	actionTriggerMenuCompactClasses,
	actionTriggerMenuItemClasses,
} from '@/components/action_trigger_chip';
import { HoverTip } from '@/components/ariakit_hover_tip';

type SingleReasoningDropdownProps = {
	reasoningLevel: ReasoningLevel;
	levelOptions?: ReasoningLevel[];
	setReasoningLevel: (level: ReasoningLevel) => void;
};

const levelDisplayNames: Record<ReasoningLevel, string> = {
	[ReasoningLevel.None]: 'None',
	[ReasoningLevel.Minimal]: 'Minimal',
	[ReasoningLevel.Low]: 'Low',
	[ReasoningLevel.Medium]: 'Medium',
	[ReasoningLevel.High]: 'High',
	[ReasoningLevel.XHigh]: 'XHigh',
	[ReasoningLevel.Max]: 'Max',
};

const DEFAULT_LEVEL_OPTIONS: ReasoningLevel[] = [
	ReasoningLevel.None,
	ReasoningLevel.Minimal,
	ReasoningLevel.Low,
	ReasoningLevel.Medium,
	ReasoningLevel.High,
];

export function SingleReasoningDropdown({
	reasoningLevel,
	levelOptions,
	setReasoningLevel,
}: SingleReasoningDropdownProps) {
	const options = (levelOptions && levelOptions.length > 0 ? levelOptions : DEFAULT_LEVEL_OPTIONS).filter(Boolean);
	const menu = useMenuStore({ placement: 'top', focusLoop: true });

	const open = useStoreState(menu, 'open');

	return (
		<div className="flex w-full justify-center">
			<div className="relative w-full">
				<HoverTip content="Set reasoning level" placement="top" wrapperElement="div" wrapperClassName="w-full">
					<MenuButton store={menu} className={`${actionTriggerChipButtonClasses} w-full flex-1 justify-center`}>
						<ActionTriggerChipContent
							label={`Reasoning: ${levelDisplayNames[reasoningLevel]}`}
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
					{options.map(level => (
						<MenuItem
							key={level}
							className={`${actionTriggerMenuItemClasses} justify-between`}
							onClick={() => {
								setReasoningLevel(level);
							}}
						>
							<span>{levelDisplayNames[level]}</span>
							{reasoningLevel === level ? <FiCheck /> : null}
						</MenuItem>
					))}
				</Menu>
			</div>
		</div>
	);
}
