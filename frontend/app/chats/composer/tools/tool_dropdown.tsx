import { useMemo } from 'react';

import { Menu, type MenuStore } from '@ariakit/react';

import { type ToolListItem, ToolStoreChoiceType } from '@/spec/tool';

import { actionTriggerMenuItemClasses, actionTriggerMenuWideClasses } from '@/components/action_trigger_chip';
import { GroupedMenuSection, GroupedMenuSubheading } from '@/components/grouped_menu_sections';

import { ToolMenuRow } from '@/chats/composer/tools/tool_menu_row';
import { toolIdentityKey } from '@/tools/lib/tool_identity_utils';

type ToolBundleGroup = {
	bundleID: string;
	bundleSlug: string;
	isBuiltIn: boolean;
	attachedOptions: ToolListItem[];
	availableOptions: ToolListItem[];
};

type ToolDropdownProps = {
	store: MenuStore;
	open: boolean;
	loading: boolean;
	tools: ToolListItem[];
	attachedToolKeys: Set<string>;
	getAutoExecForTool: (item: ToolListItem) => boolean;
	onAutoExecuteChange: (item: ToolListItem, key: string, isAttached: boolean, next: boolean) => void;
	onAttachTool: (item: ToolListItem) => void;
	onDetachTool: (key: string) => void;
};

const toolMenuCollator = new Intl.Collator(undefined, {
	numeric: true,
	sensitivity: 'base',
});

function getToolKey(item: ToolListItem): string {
	return toolIdentityKey(item.bundleID, item.bundleSlug, item.toolSlug, item.toolVersion);
}

function compareToolListItems(a: ToolListItem, b: ToolListItem): number {
	const bundleSlugCompare = toolMenuCollator.compare(a.bundleSlug, b.bundleSlug);
	if (bundleSlugCompare !== 0) return bundleSlugCompare;

	const bundleIDCompare = toolMenuCollator.compare(a.bundleID, b.bundleID);
	if (bundleIDCompare !== 0) return bundleIDCompare;

	const toolSlugCompare = toolMenuCollator.compare(a.toolSlug, b.toolSlug);
	if (toolSlugCompare !== 0) return toolSlugCompare;

	const toolVersionCompare = toolMenuCollator.compare(a.toolVersion, b.toolVersion);
	if (toolVersionCompare !== 0) return toolVersionCompare;

	return toolMenuCollator.compare(getToolKey(a), getToolKey(b));
}

function groupTools(tools: ToolListItem[], attachedToolKeys: Set<string>): ToolBundleGroup[] {
	const groupsByBundle = new Map<string, ToolBundleGroup>();

	for (const item of [...tools].sort(compareToolListItems)) {
		const groupKey = item.bundleID || item.bundleSlug;
		let group = groupsByBundle.get(groupKey);

		if (!group) {
			group = {
				bundleID: item.bundleID,
				bundleSlug: item.bundleSlug || item.bundleID,
				isBuiltIn: item.isBuiltIn,
				attachedOptions: [],
				availableOptions: [],
			};
			groupsByBundle.set(groupKey, group);
		}

		if (attachedToolKeys.has(getToolKey(item))) {
			group.attachedOptions.push(item);
		} else {
			group.availableOptions.push(item);
		}
	}

	return Array.from(groupsByBundle.values());
}

export function ToolDropdown({
	store,
	open,
	loading,
	tools,
	attachedToolKeys,
	getAutoExecForTool,
	onAutoExecuteChange,
	onAttachTool,
	onDetachTool,
}: ToolDropdownProps) {
	const groupedTools = useMemo(() => groupTools(tools, attachedToolKeys), [attachedToolKeys, tools]);

	const renderToolRow = (item: ToolListItem) => {
		const display = item.toolDefinition.displayName || item.toolSlug || 'Tool';
		const slug = `${item.bundleSlug ?? item.bundleID}/${item.toolSlug}@${item.toolVersion}`;
		const key = getToolKey(item);
		const isAttached = attachedToolKeys.has(key);
		const supportsAutoExecute =
			item.toolDefinition.llmToolType === ToolStoreChoiceType.Function ||
			item.toolDefinition.llmToolType === ToolStoreChoiceType.Custom;

		return (
			<ToolMenuRow
				key={key}
				store={store}
				menuItemClassName={`rounded-xl px-0 py-0 text-sm outline-none transition-colors hover:bg-base-200 data-[active-item]:bg-base-300 overflow-hidden ${
					isAttached ? 'bg-base-200' : ''
				}`}
				title={`Tool: ${display} (${slug})`}
				display={display}
				slug={slug}
				isSelected={isAttached}
				supportsAutoExecute={supportsAutoExecute}
				autoExecute={getAutoExecForTool(item)}
				onAutoExecuteChange={next => {
					onAutoExecuteChange(item, key, isAttached, next);
				}}
				onRowClick={
					!isAttached
						? () => {
								onAttachTool(item);
							}
						: () => {
								onDetachTool(key);
							}
				}
				primaryAction={
					isAttached
						? {
								kind: 'detach',
								onClick: () => {
									onDetachTool(key);
								},
								title: 'Detach tool',
							}
						: {
								kind: 'attach',
								onClick: () => {
									onAttachTool(item);
								},
								title: 'Attach tool',
								label: 'Attach',
							}
				}
			/>
		);
	};

	return (
		<Menu
			store={store}
			gutter={8}
			overflowPadding={8}
			portal
			className={actionTriggerMenuWideClasses}
			data-menu-kind="tools"
			autoFocusOnShow
		>
			{!open ? null : loading ? (
				<div className={`${actionTriggerMenuItemClasses} text-base-content/60 cursor-default`}>Loading tools…</div>
			) : tools.length === 0 ? (
				<div className={`${actionTriggerMenuItemClasses} text-base-content/60 cursor-default`}>No tools available</div>
			) : (
				<div className="space-y-2">
					{groupedTools.map((group, groupIndex) => {
						const totalCount = group.attachedOptions.length + group.availableOptions.length;
						const showSubheadings = group.attachedOptions.length > 0 && group.availableOptions.length > 0;

						return (
							<GroupedMenuSection
								key={group.bundleID || group.bundleSlug}
								title={group.bundleSlug}
								ariaLabel={`${group.bundleSlug} tools`}
								separatorBefore={groupIndex > 0}
								meta={
									<>
										<span className="badge badge-ghost badge-xs">{totalCount}</span>
										<span className="badge badge-ghost badge-xs">{group.isBuiltIn ? 'built-in' : 'custom'}</span>
									</>
								}
							>
								{group.attachedOptions.length > 0 ? (
									<>
										{showSubheadings ? <GroupedMenuSubheading>Attached</GroupedMenuSubheading> : null}
										{group.attachedOptions.map(renderToolRow)}
									</>
								) : null}

								{group.availableOptions.length > 0 ? (
									<>
										{showSubheadings ? (
											<GroupedMenuSubheading separated={group.attachedOptions.length > 0}>
												Available
											</GroupedMenuSubheading>
										) : null}
										{group.availableOptions.map(renderToolRow)}
									</>
								) : null}
							</GroupedMenuSection>
						);
					})}
				</div>
			)}
		</Menu>
	);
}
