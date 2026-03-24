import type { Dispatch, SetStateAction } from 'react';

import { FiCheck, FiChevronDown, FiChevronUp } from 'react-icons/fi';

import { Select, SelectItem, SelectPopover, useSelectStore, useStoreState } from '@ariakit/react';

import { ReasoningLevel } from '@/spec/inference';

type SingleReasoningDropdownProps = {
	reasoningLevel: ReasoningLevel;
	levelOptions?: ReasoningLevel[];
	setReasoningLevel: (level: ReasoningLevel) => void;
	isOpen: boolean;
	setIsOpen: Dispatch<SetStateAction<boolean>>;
};

const levelDisplayNames: Record<ReasoningLevel, string> = {
	[ReasoningLevel.None]: 'None',
	[ReasoningLevel.Minimal]: 'Minimal',
	[ReasoningLevel.Low]: 'Low',
	[ReasoningLevel.Medium]: 'Medium',
	[ReasoningLevel.High]: 'High',
	[ReasoningLevel.XHigh]: 'XHigh',
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
	isOpen,
	setIsOpen,
}: SingleReasoningDropdownProps) {
	const options = (levelOptions && levelOptions.length > 0 ? levelOptions : DEFAULT_LEVEL_OPTIONS).filter(Boolean);

	const select = useSelectStore({
		value: reasoningLevel,
		setValue: value => {
			if (typeof value === 'string' || typeof value === 'number') {
				setReasoningLevel(value as ReasoningLevel);
			}
		},
		open: isOpen,
		setOpen: setIsOpen,
		placement: 'top-start',
		focusLoop: true,
	});

	const open = useStoreState(select, 'open');

	return (
		<div className="flex w-full justify-center">
			<div className="relative w-full">
				<Select
					store={select}
					className="btn btn-xs text-neutral-custom w-full flex-1 items-center overflow-hidden border-none text-center text-nowrap shadow-none"
					title="Set Reasoning Level"
				>
					<span className="min-w-0 truncate text-center text-xs font-normal">
						Reasoning: {levelDisplayNames[reasoningLevel]}
					</span>
					{open ? (
						<FiChevronDown size={16} className="ml-1 shrink-0 md:ml-2" />
					) : (
						<FiChevronUp size={16} className="ml-1 shrink-0 md:ml-2" />
					)}
				</Select>

				<SelectPopover
					store={select}
					portal={false}
					gutter={4}
					autoFocusOnShow
					sameWidth
					className="border-base-300 bg-base-100 z-50 mt-1 max-h-72 w-full overflow-y-auto rounded-xl border p-1 text-xs shadow-lg outline-none"
				>
					{options.map(level => (
						<SelectItem
							key={level}
							value={level}
							className="hover:bg-base-200 data-active-item:bg-base-300 m-0 flex cursor-pointer items-center justify-between rounded-md px-2 py-1 text-xs transition-colors outline-none"
						>
							<span>{levelDisplayNames[level]}</span>
							{reasoningLevel === level && <FiCheck />}
						</SelectItem>
					))}
				</SelectPopover>
			</div>
		</div>
	);
}
