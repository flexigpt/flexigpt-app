import { FiCheck } from 'react-icons/fi';

import { Menu, MenuButton, MenuItem, useMenuStore, useStoreState } from '@ariakit/react';

import {
	actionTriggerChipButtonClasses,
	ActionTriggerChipContent,
	actionTriggerMenuCompactClasses,
	actionTriggerMenuItemClasses,
} from '@/components/action_trigger_chip';
import { HoverTip } from '@/components/hover_tip';

import { DEFAULT_REASONING_TOKENS, parseOptionalPositiveInteger } from '@/models/lib/model_runtime_defaults';

const defaultTokenOptions: Array<number | undefined> = [undefined, DEFAULT_REASONING_TOKENS, 8192, 32000];

interface ReasoningTokensDropdownProps {
	tokens?: number;
	setTokens: (tokens?: number) => void;
}

export function ReasoningTokensDropdown({ tokens, setTokens }: ReasoningTokensDropdownProps) {
	const menu = useMenuStore({ placement: 'top', focusLoop: true });

	const open = useStoreState(menu, 'open');

	function clampTokens(rawValue: string) {
		const parsed = parseOptionalPositiveInteger(rawValue);
		if (parsed === undefined) {
			setTokens(undefined);
			return;
		}
		if (Number.isNaN(parsed) || parsed < DEFAULT_REASONING_TOKENS) {
			return;
		}

		setTokens(parsed);
	}

	return (
		<div className="flex w-full justify-center">
			<div className="relative w-full">
				<HoverTip content="Set effort tokens" placement="top" wrapperElement="div" wrapperClassName="w-full">
					<MenuButton store={menu} className={`${actionTriggerChipButtonClasses} w-full flex-1 justify-center`}>
						<ActionTriggerChipContent
							label={tokens === undefined ? 'Effort tokens: Default' : `Effort tokens: ${tokens}`}
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
							key={tk ?? '__default__'}
							className={`${actionTriggerMenuItemClasses} justify-between`}
							onClick={() => {
								setTokens(tk);
							}}
						>
							<span>{tk === undefined ? 'Default' : tk}</span>
							{tokens === tk ? <FiCheck /> : null}
						</MenuItem>
					))}

					<div className="border-neutral/20 mt-2 border-t pt-2 text-xs">
						<div className="text-base-content/70 mb-1 text-xs">Custom (≥ {DEFAULT_REASONING_TOKENS})</div>
						<input
							key={tokens}
							data-disable-chat-shortcuts="true"
							type="text"
							className="input input-xs w-full"
							placeholder={`Default or custom integer ≥ ${DEFAULT_REASONING_TOKENS}`}
							defaultValue={tokens?.toString() ?? ''}
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
