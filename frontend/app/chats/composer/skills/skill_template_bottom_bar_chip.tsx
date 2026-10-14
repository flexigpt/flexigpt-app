import type { RefObject, SubmitEventHandler } from 'react';
import { memo, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { FiAlertCircle, FiFile, FiFilePlus } from 'react-icons/fi';

import type { MenuStore } from '@ariakit/react';
import { Menu, MenuButton, MenuItem, useStoreState } from '@ariakit/react';

import { ArtifactState } from '@/spec/artifact';
import { SkillInsert } from '@/spec/skill';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import type { ComposerSkillCatalogItem } from '@/apis/skill_management';
import { sessionManagementAPI } from '@/apis/baseapi';

import {
	ActionTriggerChipContent,
	actionTriggerChipSurfaceClasses,
	actionTriggerMenuItemClasses,
	actionTriggerMenuWideClasses,
} from '@/components/action_trigger_chip';
import { GroupedMenuSection } from '@/components/grouped_menu_sections';
import { HoverTip, HoverTipContent } from '@/components/hover_tip';
import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalHeader } from '@/components/modal/modal_header';
import { searchableMenuEmptyStateClasses, SearchableMenuInput } from '@/components/searchmenu/searchable_menu';
import {
	focusFirstSearchableMenuItem,
	isSearchQueryActive,
	rankSearchableItems,
	useSearchableMenuState,
} from '@/components/searchmenu/searchable_menu_utils';

interface SkillTemplateDropdownProps {
	store: MenuStore;
	open: boolean;
	loading: boolean;
	items: ComposerSkillCatalogItem[];
	loadError?: string | null;
	actionError?: string | null;
	onRetry?: () => void;
	onPick: (item: ComposerSkillCatalogItem) => void;
}

interface SkillTemplateRenderModalProps {
	isOpen: boolean;
	item: ComposerSkillCatalogItem | null;
	onClose: () => void;
	onInsert: (args: Record<string, string>) => Promise<void>;
}

export interface SkillTemplateBottomBarChipProps {
	store: MenuStore;
	buttonRef: RefObject<HTMLButtonElement | null>;
	shortcut?: string;
	items: ComposerSkillCatalogItem[];
	loading: boolean;
	loadError?: string | null;
	onRefresh?: () => Promise<void> | void;
	onInsertTemplateText: (text: string) => Promise<void> | void;
	isInputLocked?: boolean;
}

const skillTemplateCollator = new Intl.Collator(undefined, {
	numeric: true,
	sensitivity: 'base',
});

function getSkillTemplateLabel(item: ComposerSkillCatalogItem): string {
	return item.displayName || item.name;
}

function skillTemplateKey(item: ComposerSkillCatalogItem): string {
	return `${item.ref.rootID}:${item.ref.artifactID}`;
}

function compareSkillTemplateListItems(a: ComposerSkillCatalogItem, b: ComposerSkillCatalogItem): number {
	const labelCompare = skillTemplateCollator.compare(getSkillTemplateLabel(a), getSkillTemplateLabel(b));
	if (labelCompare !== 0) {
		return labelCompare;
	}

	return skillTemplateCollator.compare(skillTemplateKey(a), skillTemplateKey(b));
}

function templateIsSelectable(item: ComposerSkillCatalogItem): boolean {
	return item.runtimeResolved && item.enabled && item.state === ArtifactState.Available;
}

function parseTemplateArguments(raw: string): Record<string, string> {
	const trimmed = raw.trim();
	if (!trimmed) {
		return {};
	}

	let value: unknown;
	try {
		value = JSON.parse(trimmed);
	} catch {
		throw new Error('Template arguments must be a JSON object.');
	}

	if (!value || typeof value !== 'object' || Array.isArray(value)) {
		throw new Error('Template arguments must be a JSON object.');
	}

	const result: Record<string, string> = {};
	for (const [key, argument] of Object.entries(value)) {
		if (!key.trim()) {
			throw new Error('Template argument names must not be empty.');
		}
		if (typeof argument === 'string') {
			result[key] = argument;
			continue;
		}
		if (typeof argument === 'number' || typeof argument === 'boolean') {
			result[key] = String(argument);
			continue;
		}
		if (argument === null) {
			result[key] = '';
			continue;
		}
		throw new Error(`Template argument "${key}" must be a string, number, boolean, or null.`);
	}

	return result;
}

function renderSkillTemplateMenuItem(item: ComposerSkillCatalogItem, onPick: (item: ComposerSkillCatalogItem) => void) {
	const disabled = !templateIsSelectable(item);

	return (
		<MenuItem
			key={skillTemplateKey(item)}
			data-searchable-menu-item="true"
			hideOnClick={false}
			disabled={disabled}
			onClick={() => {
				if (disabled) {
					return;
				}
				onPick(item);
			}}
			className={`${actionTriggerMenuItemClasses} items-start`}
			title={item.description}
		>
			<FiFilePlus size={14} className="text-warning mt-0.5 shrink-0" />

			<div className="min-w-0 flex-1">
				<div className="truncate text-xs font-medium">{getSkillTemplateLabel(item)}</div>
				<div className="mt-1 flex min-w-0 items-center gap-2 text-[10px] opacity-70">
					<span>{item.name}</span>
					<span>•</span>
					<span>{item.ref.rootID}</span>
				</div>
			</div>

			<div className="ml-auto flex shrink-0 items-center gap-1">
				{item.hasArguments ? <span className="badge badge-warning badge-xs">Args</span> : null}
				{item.hasResources ? <span className="badge badge-info badge-xs">Resources</span> : null}
				{item.builtIn ? <span className="badge badge-ghost badge-xs">Built-in</span> : null}
				{!templateIsSelectable(item) ? <span className="badge badge-warning badge-xs">Unavailable</span> : null}
			</div>
		</MenuItem>
	);
}

function SkillTemplateDropdown({
	store,
	open,
	loading,
	items,
	loadError,
	actionError,
	onRetry,
	onPick,
}: SkillTemplateDropdownProps) {
	const menuContentElement = useStoreState(store, 'contentElement');
	const [searchQuery, setSearchQuery] = useSearchableMenuState(open);

	const displayedItems = useMemo(() => {
		if (!isSearchQueryActive(searchQuery)) {
			return items;
		}

		return rankSearchableItems(items, {
			query: searchQuery,
			getKey: skillTemplateKey,
			getFields: item => [
				{ value: getSkillTemplateLabel(item), weight: 7 },
				{ value: item.name, weight: 6 },
				{ value: item.description, weight: 3 },
				{ value: item.ref.rootID, weight: 2 },
				{ value: item.ref.artifactID, weight: 2 },
			],
			fallbackCompare: compareSkillTemplateListItems,
		});
	}, [items, searchQuery]);

	const firstTemplate = displayedItems[0] ?? null;

	return (
		<Menu
			store={store}
			gutter={8}
			overflowPadding={8}
			portal
			className={actionTriggerMenuWideClasses}
			data-menu-kind="templates"
			autoFocusOnShow={false}
		>
			{!open ? null : (
				<>
					<div className="mb-2 flex items-center justify-between gap-2 px-1">
						<div className="text-base-content/70 text-xs font-semibold">Templates</div>
						<span className="badge badge-ghost badge-xs">Available {items.length}</span>
					</div>
					<div className="text-base-content/70 mb-2 px-1 text-xs">
						Templates are skills with <span className="font-mono">insert: user-message</span>. They render into plain
						composer text and do not remain selected after insertion.
					</div>
					{loadError ? (
						<div className="alert alert-warning mb-2 rounded-xl text-xs">
							<FiAlertCircle size={14} className="shrink-0" />
							<div className="grow">
								<div className="font-semibold">Templates could not be refreshed</div>
								<div>{loadError}</div>
							</div>
							{onRetry ? (
								<button type="button" className="btn btn-xs rounded-lg" onClick={onRetry}>
									Retry
								</button>
							) : null}
						</div>
					) : null}
					{actionError ? (
						<div className="alert alert-error mb-2 rounded-xl text-xs">
							<FiAlertCircle size={14} className="shrink-0" />
							<div className="grow">{actionError}</div>
						</div>
					) : null}
					<SearchableMenuInput
						open={open}
						query={searchQuery}
						onQueryChange={setSearchQuery}
						placeholder="Search templates…"
						resultCount={displayedItems.length}
						totalCount={items.length}
						disabled={loading || items.length === 0}
						onFocusFirstItem={() => {
							focusFirstSearchableMenuItem(menuContentElement);
						}}
						onEnterFirstResult={() => {
							if (firstTemplate) {
								onPick(firstTemplate);
							}
						}}
						onEscape={() => {
							store.hide();
						}}
					/>
				</>
			)}

			{!open ? null : loading ? (
				<div className={`${actionTriggerMenuItemClasses} text-base-content/60 cursor-default`}>Loading templates…</div>
			) : items.length === 0 ? (
				<div className={`${actionTriggerMenuItemClasses} text-base-content/60 cursor-default`}>
					No user-message skill templates available
				</div>
			) : displayedItems.length === 0 ? (
				<div className={searchableMenuEmptyStateClasses}>No templates match your search.</div>
			) : (
				<GroupedMenuSection title="User-message templates" ariaLabel="User-message templates">
					{displayedItems.map(item => renderSkillTemplateMenuItem(item, onPick))}
				</GroupedMenuSection>
			)}
		</Menu>
	);
}

function SkillTemplateRenderModalContent({
	item,
	onInsert,
}: Omit<SkillTemplateRenderModalProps, 'isOpen' | 'onClose'>) {
	const [argumentsJSON, setArgumentsJSON] = useState('{}');
	const [submitError, setSubmitError] = useState('');
	const [submitting, setSubmitting] = useState(false);
	const { requestClose, unmountingRef } = useModalDialogController();

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = event => {
		event.preventDefault();
		event.stopPropagation();

		if (!item || submitting) {
			return;
		}

		setSubmitting(true);
		setSubmitError('');

		void Promise.resolve()
			.then(async () => {
				await onInsert(parseTemplateArguments(argumentsJSON));
				if (unmountingRef.current) {
					return;
				}
				requestClose(true);
			})
			.catch((error: unknown) => {
				if (!unmountingRef.current) {
					setSubmitError(error instanceof Error ? error.message : 'Failed to render template.');
				}
			})
			.finally(() => {
				if (!unmountingRef.current) {
					setSubmitting(false);
				}
			});
	};

	if (!item) {
		return null;
	}

	return (
		<div className="modal-box bg-base-200 max-h-[80vh] max-w-2xl overflow-hidden rounded-2xl p-0">
			<div className="max-h-[80vh] overflow-y-auto p-6">
				<ModalHeader
					title="Use Template"
					description={`${getSkillTemplateLabel(item)} is rendered by the Session Runtime before insertion.`}
					onClose={() => {
						requestClose();
					}}
					closeDisabled={submitting}
				/>

				<form className="space-y-4" onSubmit={handleSubmit}>
					{submitError ? (
						<div className="alert alert-error rounded-2xl text-sm">
							<FiAlertCircle size={14} />
							<span>{submitError}</span>
						</div>
					) : null}

					{item.hasArguments ? (
						<div>
							<label htmlFor="user-message-skill-arguments" className="label py-1">
								<span className="text-sm">Arguments JSON</span>
							</label>
							<textarea
								id="user-message-skill-arguments"
								className="textarea w-full rounded-xl font-mono text-xs"
								rows={8}
								value={argumentsJSON}
								onChange={event => {
									setArgumentsJSON(event.currentTarget.value);
								}}
								placeholder={'{\n  "argument": "value"\n}'}
								spellCheck="false"
							/>
						</div>
					) : (
						<div className="text-base-content/70 text-sm">This template has no declared arguments.</div>
					)}

					<ModalActions className="-mx-6 mt-6 -mb-6">
						<button
							type="button"
							className="btn bg-base-300 rounded-xl"
							onClick={() => {
								requestClose();
							}}
							disabled={submitting}
						>
							Cancel
						</button>
						<button type="submit" className="btn btn-primary rounded-xl" disabled={submitting}>
							{submitting ? 'Rendering…' : 'Insert'}
						</button>
					</ModalActions>
				</form>
			</div>
		</div>
	);
}

function SkillTemplateRenderModal(props: SkillTemplateRenderModalProps) {
	const { isOpen, item } = props;

	if (!isOpen || !item || typeof document === 'undefined' || !document.body) {
		return null;
	}

	return (
		<ModalDialog isOpen={isOpen} onClose={props.onClose} blockCancel>
			<SkillTemplateRenderModalContent key={skillTemplateKey(item)} item={item} onInsert={props.onInsert} />
		</ModalDialog>
	);
}

function SkillTemplateBottomBarChipInner({
	store,
	buttonRef,
	shortcut,
	items,
	loading,
	loadError,
	onRefresh,
	onInsertTemplateText,
	isInputLocked = false,
}: SkillTemplateBottomBarChipProps) {
	const open = useStoreState(store, 'open');
	const hoverTipContent = (
		<HoverTipContent
			title={shortcut ? `Insert template (${shortcut})` : 'Insert template'}
			description="Templates are user-message skills that render into plain composer text."
			sections={[
				{
					id: 'current-state',
					title: 'Current state',
					items: ['No template is active. Inserted templates do not remain selected after insertion.'],
				},
				{
					id: 'insertion-behavior',
					title: 'When you insert',
					items: [
						'The rendered template text is added directly to the composer.',
						'Templates with arguments or resources open a configuration dialog first.',
						'There is no active-template count or clear action after insertion.',
					],
				},
			]}
		/>
	);
	const templates = useMemo(
		() => items.filter(item => item.insert === SkillInsert.UserMessage).toSorted(compareSkillTemplateListItems),
		[items]
	);
	const templateCatalogRefreshAttemptedRef = useRef(false);
	const [modalItem, setModalItem] = useState<ComposerSkillCatalogItem | null>(null);
	const [templateActionError, setTemplateActionError] = useState<string | null>(null);

	useEffect(() => {
		const refresh = onRefresh;
		if (!open || loading || templateCatalogRefreshAttemptedRef.current || !refresh) {
			return;
		}

		templateCatalogRefreshAttemptedRef.current = true;
		void (async () => {
			try {
				await refresh();
			} catch (error) {
				console.error('Failed to refresh Composer template catalog:', error);
			}
		})();
	}, [loading, onRefresh, open]);

	useEffect(() => {
		if (!isInputLocked) {
			return;
		}
		store.hide();
		// oxlint-disable-next-line react/set-state-in-effect react-you-might-not-need-an-effect/no-adjust-state-on-prop-change
		setModalItem(null);
	}, [isInputLocked, store]);

	const handleInsertRendered = useCallback(
		async (item: ComposerSkillCatalogItem, args: Record<string, string>) => {
			const rendered = await sessionManagementAPI.prepareUserMessageSkillTemplate({
				skill: {
					rootID: item.ref.rootID,
					artifactID: item.ref.artifactID,
					expectedRevision: item.revision,
					...(item.definitionDigest ? { expectedDigest: item.definitionDigest } : {}),
				},
				arguments: args,
			});
			await onInsertTemplateText(rendered.text);
			store.hide();
		},
		[onInsertTemplateText, store]
	);

	const handlePick = useCallback(
		async (item: ComposerSkillCatalogItem) => {
			setTemplateActionError(null);

			if (item.hasArguments) {
				store.hide();
				setModalItem(item);
				return;
			}

			try {
				await handleInsertRendered(item, {});
			} catch (error) {
				setTemplateActionError(error instanceof Error ? error.message : 'Template rendering was rejected.');
			}
		},
		[handleInsertRendered, store]
	);

	return (
		<div className="relative shrink-0" data-bottom-bar-skill-templates>
			<HoverTip
				content={hoverTipContent}
				placement="top"
				wrapperElement="div"
				wrapperClassName="inline-flex max-w-full"
				tooltipClassName="max-w-sm"
			>
				<div
					className={`${actionTriggerChipSurfaceClasses} border ${
						open ? 'border-base-300 bg-base-300/60' : 'border-transparent'
					} ${isInputLocked ? 'opacity-60' : ''}`}
				>
					<MenuButton
						ref={buttonRef}
						store={store}
						disabled={isInputLocked}
						className="btn btn-xs app-text-neutral h-auto min-h-0 flex-1 gap-0 border-none bg-transparent p-0 text-left font-normal shadow-none hover:bg-transparent"
						aria-label={shortcut ? `Insert template (${shortcut})` : 'Insert template'}
					>
						<ActionTriggerChipContent icon={<FiFile size={14} />} label="Templates" open={open} />
					</MenuButton>
				</div>
			</HoverTip>

			<SkillTemplateDropdown
				store={store}
				open={open}
				loading={loading}
				items={templates}
				loadError={loadError}
				actionError={templateActionError}
				onRetry={
					onRefresh
						? () => {
								void onRefresh();
							}
						: undefined
				}
				onPick={item => {
					void handlePick(item);
				}}
			/>

			<SkillTemplateRenderModal
				isOpen={modalItem !== null}
				item={modalItem}
				onClose={() => {
					setModalItem(null);
				}}
				onInsert={args => {
					if (modalItem) {
						return handleInsertRendered(modalItem, args);
					}
					return Promise.resolve();
				}}
			/>
		</div>
	);
}

export const SkillTemplateBottomBarChip = memo(SkillTemplateBottomBarChipInner);
