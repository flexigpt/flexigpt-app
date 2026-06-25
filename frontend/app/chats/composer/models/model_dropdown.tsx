import { type Dispatch, type SetStateAction, useMemo } from 'react';

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
import { GroupedMenuSection } from '@/components/grouped_menu_sections';

interface ProviderModelGroup {
	providerName: string;
	providerDisplayName: string;
	options: UIChatOption[];
}

const modelKey = (m: UIChatOption) => `${m.providerName}::${m.modelPresetID}`;

const modelDropdownCollator = new Intl.Collator(undefined, {
	numeric: true,
	sensitivity: 'base',
});

const compareProviderGroups = (a: ProviderModelGroup, b: ProviderModelGroup) =>
	modelDropdownCollator.compare(a.providerDisplayName, b.providerDisplayName) ||
	modelDropdownCollator.compare(a.providerName, b.providerName);

const compareModelOptions = (a: UIChatOption, b: UIChatOption) =>
	modelDropdownCollator.compare(a.modelDisplayName, b.modelDisplayName) ||
	modelDropdownCollator.compare(a.name, b.name) ||
	modelDropdownCollator.compare(a.modelPresetID, b.modelPresetID);

const groupModelOptionsByProvider = (options: UIChatOption[]): ProviderModelGroup[] => {
	const groupsByProvider = new Map<string, ProviderModelGroup>();

	for (const option of options) {
		const group = groupsByProvider.get(option.providerName);
		if (group) {
			group.options.push(option);
			continue;
		}

		groupsByProvider.set(option.providerName, {
			providerName: option.providerName,
			providerDisplayName: option.providerDisplayName || option.providerName,
			options: [option],
		});
	}

	return [...groupsByProvider.values()]
		.map(group => ({ ...group, options: [...group.options].toSorted(compareModelOptions) }))
		.toSorted(compareProviderGroups);
};

interface ModelDropdownProps {
	selectedModel: UIChatOption;
	setSelectedModel: Dispatch<SetStateAction<UIChatOption>>;
	allOptions: UIChatOption[];
}

export function ModelDropdown({ selectedModel, setSelectedModel, allOptions }: ModelDropdownProps) {
	const currentKey = modelKey(selectedModel);
	const menu = useMenuStore({ placement: 'top', focusLoop: true });
	const providerGroups = useMemo(() => groupModelOptionsByProvider(allOptions), [allOptions]);

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
					className={actionTriggerMenuWideClasses}
				>
					{providerGroups.map((group, groupIndex) => (
						<GroupedMenuSection
							key={group.providerName}
							title={group.providerDisplayName}
							ariaLabel={`${group.providerDisplayName} models`}
							separatorBefore={groupIndex > 0}
						>
							{group.options.map(model => (
								<MenuItem
									key={modelKey(model)}
									className={`${actionTriggerMenuItemClasses} justify-between`}
									onClick={() => {
										setSelectedModel(model);
									}}
								>
									<span className="min-w-0 truncate">{model.modelDisplayName}</span>
									{isCurrent(model) ? <FiCheck className="ml-2 shrink-0" /> : null}
								</MenuItem>
							))}
						</GroupedMenuSection>
					))}
				</Menu>
			</div>
		</div>
	);
}
