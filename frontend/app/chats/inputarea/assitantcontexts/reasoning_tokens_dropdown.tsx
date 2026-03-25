import type { Dispatch, SetStateAction } from 'react';

import { FiCheck, FiChevronDown, FiChevronUp } from 'react-icons/fi';

import { Select, SelectItem, SelectPopover, useSelectStore, useStoreState } from '@ariakit/react';

const defaultTokenOptions = [1024, 8192, 32000];

type ReasoningTokensDropdownProps = {
	tokens: number;
	setTokens: (tokens: number) => void;
	isOpen: boolean;
	setIsOpen: Dispatch<SetStateAction<boolean>>;
};

export function ReasoningTokensDropdown({ tokens, setTokens, isOpen, setIsOpen }: ReasoningTokensDropdownProps) {
	const select = useSelectStore({
		value: tokens.toString(),
		setValue: value => {
			if (typeof value !== 'string') return;
			let v = parseInt(value, 10);
			if (isNaN(v)) return;
			if (v < 1024) v = 1024;
			setTokens(v);
		},
		open: isOpen,
		setOpen: setIsOpen,
		placement: 'top-start',
		focusLoop: true,
	});

	const open = useStoreState(select, 'open');

	function clampTokens(rawValue: string) {
		let val = parseInt(rawValue, 10);
		if (isNaN(val)) {
			val = 1024;
		}
		if (val < 1024) {
			val = 1024;
		}
		setTokens(val);
		setIsOpen(false);
	}

	return (
		<div className="flex w-full justify-center">
			<div className="relative w-full">
				<Select
					store={select}
					className="btn btn-xs text-neutral-custom w-full flex-1 items-center overflow-hidden border-none p-0 text-center text-nowrap shadow-none"
					title="Set Tokens"
				>
					<span className="mr-2 text-xs font-normal sm:hidden">Tokens:</span>
					<span className="mr-2 hidden text-xs font-normal sm:inline">Effort Tokens:</span>
					<span className="text-xs font-normal">{tokens}</span>
					{open ? (
						<FiChevronDown size={16} className="ml-1 shrink-0 xl:ml-2" />
					) : (
						<FiChevronUp size={16} className="ml-1 shrink-0 xl:ml-2" />
					)}
				</Select>

				<SelectPopover
					store={select}
					portal={false}
					gutter={4}
					autoFocusOnShow
					sameWidth
					className="border-base-300 bg-base-100 z-50 mt-1 max-h-72 w-full overflow-y-auto rounded-xl border p-4 text-xs shadow-lg outline-none"
				>
					{/* Preset token options */}
					{defaultTokenOptions.map(tk => (
						<SelectItem
							key={tk}
							value={tk.toString()}
							className="hover:bg-base-200 data-active-item:bg-base-300 m-0 flex cursor-pointer items-center justify-between rounded-md px-2 py-1 text-xs transition-colors outline-none"
						>
							<span>{tk}</span>
							{tokens === tk && <FiCheck />}
						</SelectItem>
					))}

					{/* Custom token input */}
					<div className="border-neutral/20 mt-2 border-t pt-2 text-xs">
						<label className="tooltip tooltip-top w-full border-none outline-none">
							<div className="tooltip-content">
								<div className="text-xs">Custom tokens (≥ 1024)</div>
							</div>
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
						</label>
					</div>
				</SelectPopover>
			</div>
		</div>
	);
}
