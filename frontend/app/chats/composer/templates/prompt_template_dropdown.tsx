import { useMemo } from 'react';

import { FiFilePlus } from 'react-icons/fi';

import { Menu, MenuItem, type MenuStore } from '@ariakit/react';

import type { PromptTemplateListItem } from '@/spec/prompt';

import { actionTriggerMenuItemClasses, actionTriggerMenuWideClasses } from '@/components/action_trigger_chip';
import { GroupedMenuSection } from '@/components/grouped_menu_sections';

type PromptTemplateGroup = {
	bundleID: string;
	bundleSlug: string;
	options: PromptTemplateListItem[];
};

type PromptTemplateDropdownProps = {
	store: MenuStore;
	open: boolean;
	loading: boolean;
	items: PromptTemplateListItem[];
	onPick: (item: PromptTemplateListItem) => void;
};

const promptTemplateCollator = new Intl.Collator(undefined, {
	numeric: true,
	sensitivity: 'base',
});

const promptTemplateKey = (item: PromptTemplateListItem) =>
	`${item.bundleID}::${item.bundleSlug}::${item.templateSlug}::${item.templateVersion}`;

function humanizeTemplateSlug(slug: string): string {
	return slug.replace(/[-_]/g, ' ');
}

function comparePromptTemplateListItems(a: PromptTemplateListItem, b: PromptTemplateListItem): number {
	const bundleSlugCompare = promptTemplateCollator.compare(a.bundleSlug, b.bundleSlug);
	if (bundleSlugCompare !== 0) return bundleSlugCompare;

	const bundleIDCompare = promptTemplateCollator.compare(a.bundleID, b.bundleID);
	if (bundleIDCompare !== 0) return bundleIDCompare;

	const templateSlugCompare = promptTemplateCollator.compare(a.templateSlug, b.templateSlug);
	if (templateSlugCompare !== 0) return templateSlugCompare;

	const templateVersionCompare = promptTemplateCollator.compare(a.templateVersion, b.templateVersion);
	if (templateVersionCompare !== 0) return templateVersionCompare;

	return promptTemplateCollator.compare(promptTemplateKey(a), promptTemplateKey(b));
}

function groupPromptTemplates(items: PromptTemplateListItem[]): PromptTemplateGroup[] {
	const groupsByBundle = new Map<string, PromptTemplateGroup>();

	for (const item of [...items].sort(comparePromptTemplateListItems)) {
		const groupKey = item.bundleID || item.bundleSlug;
		let group = groupsByBundle.get(groupKey);

		if (!group) {
			group = {
				bundleID: item.bundleID,
				bundleSlug: item.bundleSlug || item.bundleID,
				options: [],
			};
			groupsByBundle.set(groupKey, group);
		}

		group.options.push(item);
	}

	return Array.from(groupsByBundle.values());
}

export function PromptTemplateDropdown({ store, open, loading, items, onPick }: PromptTemplateDropdownProps) {
	const groupedTemplates = useMemo(() => groupPromptTemplates(items), [items]);

	return (
		<Menu
			store={store}
			gutter={8}
			overflowPadding={8}
			portal
			className={actionTriggerMenuWideClasses}
			data-menu-kind="templates"
			autoFocusOnShow
		>
			{!open ? null : loading ? (
				<div className={`${actionTriggerMenuItemClasses} text-base-content/60 cursor-default`}>Loading templates…</div>
			) : items.length === 0 ? (
				<div className={`${actionTriggerMenuItemClasses} text-base-content/60 cursor-default`}>
					No templates available
				</div>
			) : (
				<div className="space-y-2">
					{groupedTemplates.map((group, groupIndex) => (
						<GroupedMenuSection
							key={group.bundleID || group.bundleSlug}
							title={group.bundleSlug}
							ariaLabel={`${group.bundleSlug} prompt templates`}
							separatorBefore={groupIndex > 0}
							meta={<span className="badge badge-ghost badge-xs">{group.options.length}</span>}
						>
							{group.options.map(item => (
								<MenuItem
									key={promptTemplateKey(item)}
									onClick={() => {
										onPick(item);
									}}
									className={`${actionTriggerMenuItemClasses} items-start`}
									title={`${item.bundleSlug}/${item.templateSlug}@${item.templateVersion}`}
								>
									<FiFilePlus size={14} className="text-warning mt-0.5 shrink-0" />

									<div className="min-w-0 flex-1">
										<div className="truncate text-xs font-medium">{humanizeTemplateSlug(item.templateSlug)}</div>
										<div className="mt-1 flex min-w-0 items-center gap-2 text-[10px] opacity-70">
											<span className="truncate">{item.templateSlug}</span>
											<span>•</span>
											<span>{item.templateVersion}</span>
											<span>•</span>
											<span>{item.kind}</span>
										</div>
									</div>

									<div className="ml-auto flex shrink-0 items-center gap-1">
										<span className="badge badge-ghost badge-xs">{item.isBuiltIn ? 'built-in' : 'custom'}</span>
										{!item.isResolved ? <span className="badge badge-warning badge-xs">unresolved</span> : null}
									</div>
								</MenuItem>
							))}
						</GroupedMenuSection>
					))}
				</div>
			)}
		</Menu>
	);
}
