import { FiCheck } from 'react-icons/fi';

import { Menu, MenuButton, MenuItem, useMenuStore, useStoreState } from '@ariakit/react';

import {
	actionTriggerChipButtonClasses,
	ActionTriggerChipContent,
	actionTriggerMenuCompactClasses,
	actionTriggerMenuItemClasses,
} from '@/components/action_trigger_chip';
import { HoverTip } from '@/components/ariakit_hover_tip';

export function HybridReasoningCheckbox({
	isReasoningEnabled,
	setIsReasoningEnabled,
}: {
	isReasoningEnabled: boolean;
	setIsReasoningEnabled: (enabled: boolean) => void;
}) {
	const menu = useMenuStore({ placement: 'top', focusLoop: true });
	const open = useStoreState(menu, 'open');

	return (
		<div className="flex w-full justify-center">
			<div className="relative w-full">
				<HoverTip
					content="Toggle the model's hybrid reasoning mode (uses effort tokens instead of temperature)."
					placement="top"
					wrapperElement="div"
					wrapperClassName="w-full"
				>
					<MenuButton store={menu} className={`${actionTriggerChipButtonClasses} w-full flex-1 justify-center`}>
						<ActionTriggerChipContent
							label="Hybrid"
							secondaryLabel={isReasoningEnabled ? 'On' : 'Off'}
							suffix={isReasoningEnabled ? <FiCheck size={14} className="shrink-0" /> : undefined}
							open={open}
							className="w-full justify-center"
							labelClassName="truncate text-xs font-normal"
							secondaryLabelClassName="truncate text-xs opacity-70"
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
					<MenuItem
						className={`${actionTriggerMenuItemClasses} justify-between`}
						onClick={() => {
							setIsReasoningEnabled(true);
						}}
					>
						<span>Enabled</span>
						{isReasoningEnabled ? <FiCheck /> : null}
					</MenuItem>
					<MenuItem
						className={`${actionTriggerMenuItemClasses} justify-between`}
						onClick={() => {
							setIsReasoningEnabled(false);
						}}
					>
						<span>Disabled</span>
						{!isReasoningEnabled ? <FiCheck /> : null}
					</MenuItem>
				</Menu>
			</div>
		</div>
	);
}
