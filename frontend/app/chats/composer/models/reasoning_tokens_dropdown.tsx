import { useState } from 'react';

import { Menu, MenuButton, MenuItem, useMenuStore, useStoreState } from '@ariakit/react';

import { DEFAULT_REASONING_TOKENS } from '@/spec/model';

import { parseOptionalPositiveInteger } from '@/lib/number_utils';

import {
	actionTriggerChipButtonClasses,
	ActionTriggerChipContent,
	ActionTriggerMenuBody,
	actionTriggerMenuCompactClasses,
	actionTriggerMenuFieldCardClasses,
	actionTriggerMenuFieldInputClasses,
	ActionTriggerMenuHeader,
	actionTriggerMenuHeaderRowClasses,
	actionTriggerMenuItemClasses,
	actionTriggerMenuMetaClasses,
} from '@/components/action_trigger_chip';
import { ComposerCurrentSettingPanel } from '@/components/composer_dropdown_selected_panel';
import { HoverTip } from '@/components/hover_tip';

const defaultTokenOptions: Array<number | undefined> = [undefined, DEFAULT_REASONING_TOKENS, 8192, 32000];

function displayTokenValue(value?: number): string {
	return value === undefined ? 'Default' : value.toLocaleString();
}

interface ReasoningTokensDropdownProps {
	tokens?: number;
	setTokens: (tokens?: number) => void;
}

export function ReasoningTokensDropdown({ tokens, setTokens }: ReasoningTokensDropdownProps) {
	const menu = useMenuStore({ placement: 'top', focusLoop: true });

	const open = useStoreState(menu, 'open');
	const availableTokenOptions = defaultTokenOptions.filter(value => value !== tokens);
	const [customError, setCustomError] = useState<string | null>(null);
	const triggerTone = open
		? 'border-base-300 bg-base-300/60'
		: tokens !== undefined
			? 'border-secondary/50 bg-secondary/10 hover:bg-secondary/15'
			: '';

	function commitTokens(rawValue: string): boolean {
		const parsed = parseOptionalPositiveInteger(rawValue);
		if (parsed === undefined) {
			setTokens(undefined);
			setCustomError(null);
			return true;
		}
		if (Number.isNaN(parsed) || parsed < DEFAULT_REASONING_TOKENS) {
			setCustomError(`Enter a whole number of at least ${DEFAULT_REASONING_TOKENS.toLocaleString()}.`);
			return false;
		}

		setTokens(parsed);
		setCustomError(null);
		return true;
	}

	return (
		<div className="flex w-full justify-center">
			<div className="relative w-full">
				<HoverTip
					content="Set a reasoning-token budget."
					placement="top"
					wrapperElement="div"
					wrapperClassName="w-full"
				>
					<MenuButton
						store={menu}
						className={`${actionTriggerChipButtonClasses} w-full flex-1 justify-center ${triggerTone}`}
					>
						<ActionTriggerChipContent
							label={`Effort tokens: ${displayTokenValue(tokens)}`}
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
							<div className="min-w-0 flex-1 truncate text-xs font-semibold">Effort tokens</div>
							<span className={actionTriggerMenuMetaClasses}>{availableTokenOptions.length} presets</span>
						</div>
					</ActionTriggerMenuHeader>

					<ActionTriggerMenuBody className="space-y-2">
						<div className="space-y-1">
							{availableTokenOptions.map(tokenValue => (
								<MenuItem
									key={tokenValue ?? '__default__'}
									className={`${actionTriggerMenuItemClasses} justify-between`}
									onClick={() => {
										setCustomError(null);
										setTokens(tokenValue);
									}}
								>
									<span>{displayTokenValue(tokenValue)}</span>
								</MenuItem>
							))}
						</div>

						<div className={actionTriggerMenuFieldCardClasses}>
							<label htmlFor="reasoning-token-budget" className="block text-xs font-semibold">
								Custom token budget
							</label>
							<div className="text-base-content/60 mt-1 mb-2 text-[11px]">
								Minimum: {DEFAULT_REASONING_TOKENS.toLocaleString()} tokens.
							</div>
							<input
								id="reasoning-token-budget"
								key={tokens ?? '__default__'}
								data-disable-chat-shortcuts="true"
								type="text"
								inputMode="numeric"
								autoComplete="off"
								className={`${actionTriggerMenuFieldInputClasses} ${customError ? 'input-error' : ''}`}
								placeholder={`For example: ${DEFAULT_REASONING_TOKENS.toLocaleString()}`}
								defaultValue={tokens?.toString() ?? ''}
								spellCheck={false}
								aria-invalid={customError ? 'true' : undefined}
								aria-describedby={customError ? 'reasoning-token-error' : undefined}
								onChange={() => {
									setCustomError(null);
								}}
								onClick={event => {
									event.stopPropagation();
								}}
								onBlur={event => {
									commitTokens(event.currentTarget.value);
								}}
								onKeyDown={event => {
									if (event.key !== 'Escape') {
										event.stopPropagation();
									}
									if (event.key === 'Enter') {
										event.preventDefault();
										if (commitTokens(event.currentTarget.value)) {
											menu.hide();
										}
									}
								}}
							/>
							{customError ? (
								<div id="reasoning-token-error" role="alert" className="text-error mt-1 text-[11px]">
									{customError}
								</div>
							) : null}
						</div>
					</ActionTriggerMenuBody>

					<ComposerCurrentSettingPanel
						label="Current effort tokens"
						value={displayTokenValue(tokens)}
						help={`Use Default to defer to the model or provider. Custom budgets must be at least ${DEFAULT_REASONING_TOKENS.toLocaleString()} tokens.`}
						isDefault={tokens === undefined}
						onUseDefault={() => {
							setCustomError(null);
							setTokens(undefined);
						}}
					/>
				</Menu>
			</div>
		</div>
	);
}
