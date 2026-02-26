// FILE: output_verbosity_dropdown.tsx
import type { Dispatch, SetStateAction } from 'react';

import { FiCheck, FiChevronDown, FiChevronUp } from 'react-icons/fi';

import { Select, SelectItem, SelectPopover, useSelectStore, useStoreState } from '@ariakit/react';

import { OutputVerbosity } from '@/spec/inference';

type OutputVerbosityDropdownProps = {
	verbosity?: OutputVerbosity;
	setVerbosity: (v?: OutputVerbosity) => void;
	isOpen: boolean;
	setIsOpen: Dispatch<SetStateAction<boolean>>;
};

const VERBOSITY_OPTIONS: Array<{ label: string; value?: OutputVerbosity }> = [
	{ label: 'Default', value: undefined },
	{ label: 'Low', value: OutputVerbosity.Low },
	{ label: 'Medium', value: OutputVerbosity.Medium },
	{ label: 'High', value: OutputVerbosity.High },
	{ label: 'Max', value: OutputVerbosity.Max },
];

function labelFor(v?: OutputVerbosity) {
	const l = VERBOSITY_OPTIONS.find(o => o.value === v)?.label ?? '';
	if (l === 'Default') {
		return '';
	}
	return ': ' + l;
}

export function OutputVerbosityDropdown({ verbosity, setVerbosity, isOpen, setIsOpen }: OutputVerbosityDropdownProps) {
	const select = useSelectStore({
		value: verbosity ?? '',
		setValue: value => {
			if (typeof value !== 'string') return;
			setVerbosity(value ? (value as OutputVerbosity) : undefined);
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
					title="Set effort/verbosity for model output"
				>
					<span className="min-w-0 truncate text-center text-xs font-normal">Effort{labelFor(verbosity)}</span>
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
					{VERBOSITY_OPTIONS.map(opt => (
						<SelectItem
							key={opt.value ?? '__default__'}
							value={opt.value ?? ''}
							className="hover:bg-base-200 data-active-item:bg-base-300 m-0 flex cursor-pointer items-center justify-between rounded-md px-2 py-1 transition-colors outline-none"
						>
							<span>{opt.label}</span>
							{(verbosity ?? undefined) === (opt.value ?? undefined) && <FiCheck />}
						</SelectItem>
					))}
				</SelectPopover>
			</div>
		</div>
	);
}
