import { memo, type RefObject, useEffect, useMemo } from 'react';

import { FiFilePlus } from 'react-icons/fi';

import { Menu, MenuButton, MenuItem, type MenuStore, useStoreState } from '@ariakit/react';

import type { PromptTemplate, PromptTemplateListItem } from '@/spec/prompt';

import { promptStoreAPI } from '@/apis/baseapi';

import {
	actionTriggerChipButtonClasses,
	ActionTriggerChipContent,
	actionTriggerMenuItemClasses,
	actionTriggerMenuWideClasses,
} from '@/components/action_trigger_chip';
import { HoverTip } from '@/components/ariakit_hover_tip';
import { GroupedMenuSection } from '@/components/grouped_menu_sections';

import { usePromptTemplates } from '@/prompts/lib/use_prompt_templates';

export interface PromptTemplateInsertArgs {
	bundleID: string;
	templateSlug: string;
	templateVersion: string;
	template?: PromptTemplate;
}

interface PromptTemplateGroup {
	bundleID: string;
	bundleSlug: string;
	options: PromptTemplateListItem[];
}

interface PromptTemplateDropdownProps {
	store: MenuStore;
	open: boolean;
	loading: boolean;
	items: PromptTemplateListItem[];
	onPick: (item: PromptTemplateListItem) => void;
}

const promptTemplateCollator = new Intl.Collator(undefined, {
	numeric: true,
	sensitivity: 'base',
});

const promptTemplateKey = (item: PromptTemplateListItem) =>
	`${item.bundleID}::${item.bundleSlug}::${item.templateSlug}::${item.templateVersion}`;

function humanizeTemplateSlug(slug: string): string {
	return slug.replaceAll(/[-_]/g, ' ');
}

function comparePromptTemplateListItems(a: PromptTemplateListItem, b: PromptTemplateListItem): number {
	const bundleSlugCompare = promptTemplateCollator.compare(a.bundleSlug, b.bundleSlug);
	if (bundleSlugCompare !== 0) {
		return bundleSlugCompare;
	}

	const bundleIDCompare = promptTemplateCollator.compare(a.bundleID, b.bundleID);
	if (bundleIDCompare !== 0) {
		return bundleIDCompare;
	}

	const templateSlugCompare = promptTemplateCollator.compare(a.templateSlug, b.templateSlug);
	if (templateSlugCompare !== 0) {
		return templateSlugCompare;
	}

	const templateVersionCompare = promptTemplateCollator.compare(a.templateVersion, b.templateVersion);
	if (templateVersionCompare !== 0) {
		return templateVersionCompare;
	}

	return promptTemplateCollator.compare(promptTemplateKey(a), promptTemplateKey(b));
}

function groupPromptTemplates(items: PromptTemplateListItem[]): PromptTemplateGroup[] {
	const groupsByBundle = new Map<string, PromptTemplateGroup>();

	for (const item of [...items].toSorted(comparePromptTemplateListItems)) {
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

function PromptTemplateDropdown({ store, open, loading, items, onPick }: PromptTemplateDropdownProps) {
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

interface PromptTemplateBottomBarChipProps {
	store: MenuStore;
	buttonRef: RefObject<HTMLButtonElement | null>;
	shortcut?: string;
	onInsertTemplate: (args: PromptTemplateInsertArgs) => Promise<void> | void;
	isInputLocked?: boolean;
}

function PromptTemplateBottomBarChipInner({
	store,
	buttonRef,
	shortcut,
	onInsertTemplate,
	isInputLocked = false,
}: PromptTemplateBottomBarChipProps) {
	const open = useStoreState(store, 'open');
	const tooltip = shortcut ? `Insert prompts (${shortcut})` : 'Insert prompts';

	const { data: templates, loading } = usePromptTemplates();

	useEffect(() => {
		if (!isInputLocked) {
			return;
		}
		store.hide();
	}, [isInputLocked, store]);

	const handlePick = async (item: PromptTemplateListItem) => {
		try {
			const tmpl = await promptStoreAPI.getPromptTemplate(item.bundleID, item.templateSlug, item.templateVersion);
			await onInsertTemplate({
				bundleID: item.bundleID,
				templateSlug: item.templateSlug,
				templateVersion: item.templateVersion,
				template: tmpl,
			});
		} catch {
			// Fall back to inserting by ref only; the consumer resolves it lazily.
			await onInsertTemplate({
				bundleID: item.bundleID,
				templateSlug: item.templateSlug,
				templateVersion: item.templateVersion,
			});
		} finally {
			store.hide();
		}
	};

	return (
		<div className="relative shrink-0" data-bottom-bar-prompt-templates>
			<HoverTip content={tooltip} placement="top">
				<MenuButton
					ref={buttonRef}
					store={store}
					disabled={isInputLocked}
					className={`${actionTriggerChipButtonClasses} hover:text-base-content ${isInputLocked ? 'opacity-60' : ''}`}
					aria-label={tooltip}
				>
					<ActionTriggerChipContent icon={<FiFilePlus size={16} />} label="Prompts" open={open} />
				</MenuButton>
			</HoverTip>

			<PromptTemplateDropdown
				store={store}
				open={open}
				loading={loading}
				items={templates}
				onPick={item => {
					void handlePick(item);
				}}
			/>
		</div>
	);
}

/**
 * Isolated wrapper for the prompt-template picker so template loading and
 * menu open/close state changes don't re-render the entire EditorBottomBar.
 */
export const PromptTemplateBottomBarChip = memo(PromptTemplateBottomBarChipInner);
