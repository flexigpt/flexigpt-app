import type { Dispatch, SetStateAction } from 'react';

import { FiCheck } from 'react-icons/fi';

import { Menu, MenuButton, MenuItem, useMenuStore, useStoreState } from '@ariakit/react';

import type { UIChatOption } from '@/spec/modelpreset';

import {
	actionTriggerChipButtonClasses,
	ActionTriggerChipContent,
	actionTriggerMenuItemClasses,
	actionTriggerMenuWideClasses,
} from '@/components/action_trigger_chip';
import { HoverTip } from '@/components/ariakit_hover_tip';

const modelKey = (m: UIChatOption) => `${m.providerName}::${m.modelPresetID}`;

type ModelDropdownProps = {
	selectedModel: UIChatOption;
	setSelectedModel: Dispatch<SetStateAction<UIChatOption>>;
	allOptions: UIChatOption[];
};

export function ModelDropdown({ selectedModel, setSelectedModel, allOptions }: ModelDropdownProps) {
	const currentKey = modelKey(selectedModel);
	const menu = useMenuStore({ placement: 'top', focusLoop: true });

	const open = useStoreState(menu, 'open');
	const isCurrent = (m: UIChatOption) => modelKey(m) === currentKey;

	return (
		<div className="flex w-full justify-center">
			<div className="relative w-full">
				<HoverTip content="Select model" placement="top" wrapperElement="div" wrapperClassName="w-full">
					<MenuButton store={menu} className={`${actionTriggerChipButtonClasses} w-full flex-1 justify-center`}>
						<ActionTriggerChipContent
							label={selectedModel.modelDisplayName}
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
					className={`${actionTriggerMenuWideClasses} text-xs`}
				>
					{allOptions.map(model => (
						<MenuItem
							key={modelKey(model)}
							className={`${actionTriggerMenuItemClasses} justify-between`}
							onClick={() => {
								setSelectedModel(model);
							}}
						>
							<span className="truncate">{model.modelDisplayName}</span>
							{isCurrent(model) ? <FiCheck /> : null}
						</MenuItem>
					))}
				</Menu>
			</div>
		</div>
	);
}
