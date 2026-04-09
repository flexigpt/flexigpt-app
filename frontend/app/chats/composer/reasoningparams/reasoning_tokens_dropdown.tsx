import { FiCheck } from 'react-icons/fi';

import { Menu, MenuButton, MenuItem, useMenuStore, useStoreState } from '@ariakit/react';

import {
	actionTriggerChipButtonClasses,
	ActionTriggerChipContent,
	actionTriggerMenuCompactClasses,
	actionTriggerMenuItemClasses,
} from '@/components/action_trigger_chip';
import { HoverTip } from '@/components/ariakit_hover_tip';

const defaultTokenOptions = [1024, 8192, 32000];

type ReasoningTokensDropdownProps = {
	tokens: number;
	setTokens: (tokens: number) => void;
};

export function ReasoningTokensDropdown({ tokens, setTokens }: ReasoningTokensDropdownProps) {
	const menu = useMenuStore({ placement: 'top', focusLoop: true });

	const open = useStoreState(menu, 'open');

	function clampTokens(rawValue: string) {
		let val = parseInt(rawValue, 10);
		if (isNaN(val)) {
			val = 1024;
		}
		if (val < 1024) {
			val = 1024;
		}
		setTokens(val);
	}

	return (
		<div className="flex w-full justify-center">
			<div className="relative w-full">
				<HoverTip content="Set effort tokens" placement="top" wrapperElement="div" wrapperClassName="w-full">
					<MenuButton store={menu} className={`${actionTriggerChipButtonClasses} w-full flex-1 justify-center`}>
						<ActionTriggerChipContent
							label={`Effort tokens: ${tokens}`}
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
					{defaultTokenOptions.map(tk => (
						<MenuItem
							key={tk}
							className={`${actionTriggerMenuItemClasses} justify-between`}
							onClick={() => {
								setTokens(tk);
							}}
						>
							<span>{tk}</span>
							{tokens === tk ? <FiCheck /> : null}
						</MenuItem>
					))}

					<div className="border-neutral/20 mt-2 border-t pt-2 text-xs">
						<div className="text-base-content/70 mb-1 text-xs">Custom (≥ 1024)</div>
						<input
							key={tokens}
							data-disable-chat-shortcuts="true"
							type="text"
							className="input input-xs w-full"
							placeholder="Enter a custom integer ≥ 1024"
							defaultValue={tokens.toString()}
							onBlur={e => {
								clampTokens(e.currentTarget.value);
							}}
							onKeyDown={e => {
								if (e.key === 'Enter') {
									e.preventDefault();
									clampTokens(e.currentTarget.value);
								}
							}}
						/>
					</div>
				</Menu>
			</div>
		</div>
	);
}
