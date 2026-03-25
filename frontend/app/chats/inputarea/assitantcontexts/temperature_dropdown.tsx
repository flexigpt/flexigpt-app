import type { Dispatch, SetStateAction } from 'react';

import { FiCheck, FiChevronDown, FiChevronUp } from 'react-icons/fi';

import { Select, SelectItem, SelectPopover, useSelectStore, useStoreState } from '@ariakit/react';

const defaultTemperatureOptions = [0.0, 0.1, 0.5, 1.0];

type TemperatureDropdownProps = {
	temperature: number;
	setTemperature: (t: number) => void;
	isOpen: boolean;
	setIsOpen: Dispatch<SetStateAction<boolean>>;
};

export function TemperatureDropdown({ temperature, setTemperature, isOpen, setIsOpen }: TemperatureDropdownProps) {
	const select = useSelectStore({
		value: temperature.toString(),
		setValue: value => {
			if (typeof value !== 'string') return;
			let v = parseFloat(value);
			if (isNaN(v)) return;
			v = Math.max(0, Math.min(1, v));
			setTemperature(v);
		},
		open: isOpen,
		setOpen: setIsOpen,
		placement: 'top-start',
		focusLoop: true,
	});

	const open = useStoreState(select, 'open');

	function clampTemperature(rawValue: string) {
		let val = parseFloat(rawValue);
		if (isNaN(val)) {
			val = 0.1;
		}
		val = Math.max(0, Math.min(1, val));
		setTemperature(val);
		setIsOpen(false);
	}

	return (
		<div className="flex w-full justify-center">
			<div className="relative w-full">
				<Select
					store={select}
					className="btn btn-xs text-neutral-custom w-full flex-1 items-center overflow-hidden border-none p-0 text-left text-nowrap shadow-none"
					title="Set Temperature"
				>
					<span className="mr-2 text-xs font-normal sm:hidden">Temp:</span>
					<span className="mr-2 hidden text-xs font-normal sm:inline">Temperature:</span>
					<span className="text-xs font-normal">{temperature.toFixed(2)}</span>
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
					{/* Preset options */}
					{defaultTemperatureOptions.map(tempVal => (
						<SelectItem
							key={tempVal}
							value={tempVal.toString()}
							className="hover:bg-base-200 data-active-item:bg-base-300 m-0 flex cursor-pointer items-center justify-between rounded-md px-2 py-1 text-xs transition-colors outline-none"
						>
							<span>{tempVal.toFixed(1)}</span>
							{temperature.toFixed(1) === tempVal.toFixed(1) && <FiCheck />}
						</SelectItem>
					))}

					{/* Custom input */}
					<div className="border-neutral/20 mt-2 border-t pt-2 text-xs">
						<label className="tooltip tooltip-top w-full border-none outline-none">
							<div className="tooltip-content">
								<div className="text-xs">Custom value (0.0 - 1.0)</div>
							</div>
							<input
								key={temperature}
								data-disable-chat-shortcuts="true"
								type="text"
								name="temperature"
								className="input input-xs w-full"
								placeholder="Custom value (0.0 - 1.0)"
								defaultValue={temperature.toString()}
								onBlur={e => {
									clampTemperature(e.currentTarget.value);
								}}
								onKeyDown={e => {
									if (e.key === 'Enter') {
										e.preventDefault();
										clampTemperature(e.currentTarget.value);
									}
								}}
								spellCheck="false"
							/>
						</label>
					</div>
				</SelectPopover>
			</div>
		</div>
	);
}
