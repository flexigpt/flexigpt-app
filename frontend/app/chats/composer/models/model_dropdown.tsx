import type { Dispatch, SetStateAction } from 'react';
import { useMemo } from 'react';
import { FiCheck } from 'react-icons/fi';

import { Menu, MenuButton, MenuItem, useMenuStore, useStoreState } from '@ariakit/react';

import type { ArtifactRef } from '@/spec/artifact';
import type { UIModelOption } from '@/spec/model';

import {
	actionTriggerChipButtonClasses,
	ActionTriggerChipContent,
	actionTriggerMenuItemClasses,
	actionTriggerMenuWideClasses,
} from '@/components/action_trigger_chip';
import { GroupedMenuSection } from '@/components/grouped_menu_sections';
import { HoverTip } from '@/components/hover_tip';
import { searchableMenuEmptyStateClasses, SearchableMenuInput } from '@/components/searchmenu/searchable_menu';
import {
	focusFirstSearchableMenuItem,
	isSearchQueryActive,
	rankSearchableItems,
	useSearchableMenuState,
} from '@/components/searchmenu/searchable_menu_utils';

interface ProviderModelGroup {
	providerName: string;
	providerDisplayName: string;
	options: UIModelOption[];
}

const modelKey = (model: UIModelOption) => {
	if (model.model) {
		return `${model.model.rootID}::${model.model.artifactID}`;
	}
	return `${model.providerName}::${model.logicalName}`;
};

const modelDropdownCollator = new Intl.Collator(undefined, {
	numeric: true,
	sensitivity: 'base',
});

const compareProviderGroups = (a: ProviderModelGroup, b: ProviderModelGroup) =>
	modelDropdownCollator.compare(a.providerDisplayName, b.providerDisplayName) ||
	modelDropdownCollator.compare(a.providerName, b.providerName);

const compareModelOptions = (a: UIModelOption, b: UIModelOption) =>
	modelDropdownCollator.compare(a.modelDisplayName, b.modelDisplayName) ||
	modelDropdownCollator.compare(a.name, b.name) ||
	modelDropdownCollator.compare(a.logicalName, b.logicalName);

const groupModelOptionsByProvider = (options: UIModelOption[]): ProviderModelGroup[] => {
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

	const providerGroups = [...groupsByProvider.values()];

	for (const group of providerGroups) {
		group.options = group.options.toSorted(compareModelOptions);
	}

	return providerGroups.toSorted(compareProviderGroups);
};

interface ModelDropdownProps {
	selectedModel: UIModelOption;
	setSelectedModel: Dispatch<SetStateAction<UIModelOption>>;
	allOptions: UIModelOption[];
	modelSelectionSource?: 'session' | 'agent';
	agentModelRef?: ArtifactRef;
}

export function ModelDropdown({
	selectedModel,
	setSelectedModel,
	allOptions,
	modelSelectionSource = 'session',
	agentModelRef,
}: ModelDropdownProps) {
	const currentKey = modelKey(selectedModel);
	const menu = useMenuStore({ placement: 'top', focusLoop: true });
	const providerGroups = useMemo(() => groupModelOptionsByProvider(allOptions), [allOptions]);
	const agentOwnsModel = modelSelectionSource === 'agent';
	const agentPreviewMatchesSelected =
		agentOwnsModel &&
		agentModelRef?.rootID === selectedModel.model?.rootID &&
		agentModelRef?.artifactID === selectedModel.model?.artifactID;
	const triggerLabel = agentOwnsModel
		? agentPreviewMatchesSelected
			? `Agent model: ${selectedModel.modelDisplayName}`
			: agentModelRef
				? `Agent model: ${agentModelRef.artifactID}`
				: 'Agent chooses model'
		: selectedModel.modelDisplayName;

	const open = useStoreState(menu, 'open');
	const menuContentElement = useStoreState(menu, 'contentElement');
	const [searchQuery, setSearchQuery] = useSearchableMenuState(open);
	const isCurrent = (m: UIModelOption) => modelKey(m) === currentKey;

	const displayedProviderGroups = useMemo(() => {
		if (!isSearchQueryActive(searchQuery)) {
			return providerGroups;
		}

		const rankedModels = rankSearchableItems(allOptions, {
			query: searchQuery,
			getKey: modelKey,
			getFields: model => [
				{ value: model.modelDisplayName, weight: 5 },
				{ value: model.name, weight: 4 },
				{ value: model.logicalName, weight: 3 },
				{ value: model.providerDisplayName, weight: 2 },
				{ value: model.providerName, weight: 2 },
			],
			fallbackCompare: compareModelOptions,
		});
		const rankByKey = new Map(rankedModels.map((model, index) => [modelKey(model), index] as const));

		return providerGroups
			.map(group => ({
				...group,
				options: group.options
					.filter(model => rankByKey.has(modelKey(model)))
					.toSorted((a, b) => (rankByKey.get(modelKey(a)) ?? 0) - (rankByKey.get(modelKey(b)) ?? 0)),
			}))
			.filter(group => group.options.length > 0);
	}, [allOptions, providerGroups, searchQuery]);

	const displayedModelCount = displayedProviderGroups.reduce((sum, group) => sum + group.options.length, 0);
	const firstVisibleModel = displayedProviderGroups[0]?.options[0] ?? null;

	const selectModel = (model: UIModelOption) => {
		setSelectedModel(model);
		menu.hide();
	};

	return (
		<div className="flex w-full justify-center">
			<div className="relative w-full">
				<HoverTip
					content={
						agentOwnsModel
							? 'The Agent owns the effective Model selection. Choosing a Model here creates an explicit Session override.'
							: 'Select model'
					}
					placement="top"
					wrapperElement="div"
					wrapperClassName="w-full"
				>
					<MenuButton store={menu} className={`${actionTriggerChipButtonClasses} w-full flex-1 justify-center`}>
						<ActionTriggerChipContent
							label={triggerLabel}
							open={open}
							suffix={agentOwnsModel ? <span className="badge badge-secondary badge-xs">Agent</span> : undefined}
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
					autoFocusOnShow={false}
					className={actionTriggerMenuWideClasses}
				>
					<SearchableMenuInput
						open={open}
						query={searchQuery}
						onQueryChange={setSearchQuery}
						placeholder="Search models…"
						resultCount={displayedModelCount}
						totalCount={allOptions.length}
						onFocusFirstItem={() => {
							focusFirstSearchableMenuItem(menuContentElement);
						}}
						onEnterFirstResult={() => {
							if (firstVisibleModel) {
								selectModel(firstVisibleModel);
							}
						}}
						onEscape={() => {
							menu.hide();
						}}
					/>

					{agentOwnsModel ? (
						<div className="text-base-content/60 mb-2 px-1 text-xs">
							Choosing a Model below replaces Agent Model ownership with an explicit Session Model override.
						</div>
					) : null}

					{displayedProviderGroups.length === 0 ? (
						<div className={searchableMenuEmptyStateClasses}>No models match your search.</div>
					) : null}

					{displayedProviderGroups.map((group, groupIndex) => (
						<GroupedMenuSection
							key={group.providerName}
							title={group.providerDisplayName}
							ariaLabel={`${group.providerDisplayName} models`}
							separatorBefore={groupIndex > 0}
						>
							{group.options.map(model => (
								<MenuItem
									key={modelKey(model)}
									data-searchable-menu-item="true"
									className={`${actionTriggerMenuItemClasses} justify-between`}
									onClick={() => {
										selectModel(model);
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
