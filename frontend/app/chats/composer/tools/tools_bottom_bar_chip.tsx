import type { Dispatch, MouseEvent, RefObject, SetStateAction, SyntheticEvent } from 'react';
import { useEffect, useMemo, useState } from 'react';
import { FiAlertTriangle, FiCheck, FiEdit2, FiGlobe, FiRefreshCw, FiTool, FiX } from 'react-icons/fi';

import type { MenuStore } from '@ariakit/react';
import { MenuButton, MenuItem, useStoreState } from '@ariakit/react';

import type { ProviderSDKType } from '@/spec/provider_sdk';
import type { ToolListItem } from '@/spec/tool';
import { ToolImplType, ToolStoreChoiceType } from '@/spec/tool';

import type { ToolCatalogState } from '@/hooks/use_tool';

import { toolArtifactKey, toolDisplayName, toolIdentityKey } from '@/apis/tool_management';

import {
	actionTriggerChipButtonClasses,
	ActionTriggerChipContent,
	ActionTriggerMenu,
	ActionTriggerMenuBody,
	ActionTriggerMenuHeader,
	actionTriggerMenuHeaderRowClasses,
	actionTriggerMenuItemClasses,
	actionTriggerMenuMetaClasses,
} from '@/components/action_trigger_chip';
import { ComposerDropdownSelectedPanel } from '@/components/composer_dropdown_selected_panel';
import { GroupedMenuSection } from '@/components/grouped_menu_sections';
import { HoverTip } from '@/components/hover_tip';
import { SearchableMenuInput } from '@/components/searchmenu/searchable_menu';
import {
	focusFirstSearchableMenuItem,
	isSearchQueryActive,
	rankSearchableItems,
	useSearchableMenuState,
} from '@/components/searchmenu/searchable_menu_utils';

import type { AttachedToolEntry } from '@/chats/composer/platedoc/tool_document_ops';
import type { ConversationToolStateEntry } from '@/chats/composer/tools/lib/conversation_tool_utils';
import type { WebSearchChoiceTemplate } from '@/chats/composer/tools/websearch_utils';
import { dispatchOpenToolArgs } from '@/chats/composer/toolruntime/use_open_toolargs_event';
import { computeToolUserArgsStatus } from '@/chats/composer/tools/lib/tool_userargs_utils';
import { ToolMenuRow } from '@/chats/composer/tools/tool_menu_row';
import {
	getWebSearchConfiguration,
	webSearchIdentityKey,
	webSearchTemplateFromToolListItem,
} from '@/chats/composer/tools/websearch_utils';

interface ToolsBottomBarChipProps {
	store: MenuStore;
	buttonRef: RefObject<HTMLButtonElement | null>;
	shortcut?: string;
	currentProviderSDKType: ProviderSDKType;
	attachedToolEntries: AttachedToolEntry[];
	conversationToolsState: ConversationToolStateEntry[];
	setConversationToolsState: Dispatch<SetStateAction<ConversationToolStateEntry[]>>;
	onAttachTool: (item: ToolListItem, autoExecute: boolean) => void;
	onDetachToolByKey: (key: string) => void;
	onSetAttachedToolAutoExecute: (key: string, autoExecute: boolean) => void;
	onRemoveAttachedTool: (entry: AttachedToolEntry) => void;
	onRemoveAllAttachedTools: (entries: AttachedToolEntry[]) => void;
	onEditAttachedToolOptions: (entry: AttachedToolEntry) => void;
	onOpenAttachedToolDetails?: (entry: AttachedToolEntry) => void;
	onOpenConversationToolDetails?: (entry: ConversationToolStateEntry) => void;
	webSearchTemplates: WebSearchChoiceTemplate[];
	setWebSearchTemplates: Dispatch<SetStateAction<WebSearchChoiceTemplate[]>>;
	toolCatalog: ToolCatalogState;
	toolArgsEventTarget?: EventTarget | null;
	isInputLocked?: boolean;
}

interface ToolPluginGroup {
	key: string;
	name: string;
	available: ToolListItem[];
}

const collator = new Intl.Collator(undefined, { numeric: true, sensitivity: 'base' });

function stop(event: SyntheticEvent | MouseEvent) {
	event.preventDefault();
	event.stopPropagation();
}

function itemKey(item: ToolListItem): string {
	return toolIdentityKey(item.target);
}

function itemLabel(item: ToolListItem): string {
	return `${item.pluginName}/${item.toolDefinition.name}@${item.toolDefinition.version}`;
}

function compareItems(left: ToolListItem, right: ToolListItem): number {
	return collator.compare(itemLabel(left), itemLabel(right)) || collator.compare(itemKey(left), itemKey(right));
}

function itemSearchFields(item: ToolListItem) {
	return [
		{ value: item.toolDefinition.displayName, weight: 7 },
		{ value: item.toolDefinition.name, weight: 6 },
		{ value: item.toolDefinition.version, weight: 4 },
		{ value: item.pluginName, weight: 3 },
		{ value: item.toolDefinition.description, weight: 2 },
		{ value: item.toolDefinition.tags, weight: 1 },
	];
}

export function ToolsBottomBarChip({
	store,
	buttonRef,
	shortcut,
	currentProviderSDKType,
	attachedToolEntries,
	conversationToolsState,
	setConversationToolsState,
	onAttachTool,

	onSetAttachedToolAutoExecute,
	onRemoveAttachedTool,
	onRemoveAllAttachedTools,
	onEditAttachedToolOptions,
	onOpenAttachedToolDetails,
	onOpenConversationToolDetails,
	webSearchTemplates,
	setWebSearchTemplates,
	toolCatalog,
	toolArgsEventTarget,
	isInputLocked = false,
}: ToolsBottomBarChipProps) {
	const open = useStoreState(store, 'open');
	const contentElement = useStoreState(store, 'contentElement');
	const [query, setQuery] = useSearchableMenuState(open);
	const [autoOverrides, setAutoOverrides] = useState<Record<string, boolean>>({});

	const {
		data: toolData,
		error: catalogError,
		loading: isLoading,
		isRefreshing,
		hasResolved,
		refresh: reloadOrThrow,
		ready: catalogReady,
	} = toolCatalog;

	useEffect(() => {
		if (isInputLocked) {
			store.hide();
		}
	}, [isInputLocked, store]);

	const attached = useMemo(
		() => attachedToolEntries.filter(entry => entry.toolType !== ToolStoreChoiceType.WebSearch),
		[attachedToolEntries]
	);
	const attachedByKey = useMemo(
		() => new Map(attached.map(entry => [toolIdentityKey(entry.target), entry])),
		[attached]
	);

	const {
		eligible: eligibleWebSearch,
		active: activeWebSearch,
		item: activeWebSearchItem,
		status: webSearchStatus,
		blocked: webSearchBlocked,
	} = useMemo(
		() => getWebSearchConfiguration(webSearchTemplates, toolData, catalogReady, currentProviderSDKType),
		[webSearchTemplates, toolData, catalogReady, currentProviderSDKType]
	);

	const available = useMemo(
		() =>
			catalogReady
				? toolData.filter(item => {
						const implementation = item.toolDefinition.implementation;
						return (
							implementation.kind === ToolImplType.Go ||
							(implementation.sdkToolType !== ToolStoreChoiceType.WebSearch &&
								implementation.sdkType === currentProviderSDKType.toString())
						);
					})
				: [],
		[toolData, currentProviderSDKType, catalogReady]
	);

	const visibleAvailable = useMemo(
		() =>
			isSearchQueryActive(query)
				? rankSearchableItems(available, {
						query,
						getKey: itemKey,
						getFields: itemSearchFields,
						fallbackCompare: compareItems,
					})
				: available.toSorted(compareItems),
		[available, query]
	);
	const visibleWebSearch = useMemo(
		() =>
			isSearchQueryActive(query)
				? rankSearchableItems(eligibleWebSearch, {
						query,
						getKey: itemKey,
						getFields: itemSearchFields,
						fallbackCompare: compareItems,
					})
				: eligibleWebSearch.toSorted((left, right) => {
						const activeKey = activeWebSearch ? webSearchIdentityKey(activeWebSearch) : undefined;
						const leftActive = itemKey(left) === activeKey;
						const rightActive = itemKey(right) === activeKey;
						return leftActive !== rightActive ? (leftActive ? -1 : 1) : compareItems(left, right);
					}),
		[eligibleWebSearch, activeWebSearch, query]
	);
	const configuredToolKeys = useMemo(
		() =>
			new Set([
				...attached.map(entry => toolIdentityKey(entry.target)),
				...conversationToolsState.map(entry => toolIdentityKey(entry.toolStoreChoice.target)),
			]),
		[attached, conversationToolsState]
	);
	const visibleAvailableToAdd = useMemo(
		() => visibleAvailable.filter(item => !configuredToolKeys.has(itemKey(item))),
		[configuredToolKeys, visibleAvailable]
	);
	const visibleWebSearchToAdd = useMemo(
		() => visibleWebSearch.filter(item => !activeWebSearch || itemKey(item) !== webSearchIdentityKey(activeWebSearch)),
		[activeWebSearch, visibleWebSearch]
	);

	const groups = useMemo(() => {
		const result = new Map<string, ToolPluginGroup>();
		const ensure = (key: string, name: string) => {
			let group = result.get(key);
			if (!group) {
				group = { key, name, available: [] };
				result.set(key, group);
			}
			return group;
		};

		for (const item of visibleAvailableToAdd) {
			ensure(toolArtifactKey(item.pluginRef), item.pluginName).available.push(item);
		}
		return [...result.values()].toSorted((left, right) => collator.compare(left.name, right.name));
	}, [visibleAvailableToAdd]);

	const missingCount =
		attached.filter(entry => {
			if (!entry.toolSnapshot) {
				return true;
			}
			const status = computeToolUserArgsStatus(entry.toolSnapshot.userArgSchema, entry.userArgSchemaInstance);
			return status.hasSchema && !status.isSatisfied;
		}).length +
		conversationToolsState.filter(entry => {
			if (!entry.enabled) {
				return false;
			}
			if (!entry.toolDefinition || entry.toolLoadError) {
				return true;
			}
			const status =
				entry.argStatus ??
				computeToolUserArgsStatus(entry.toolDefinition.userArgSchema, entry.toolStoreChoice.userArgSchemaInstance);
			return status.hasSchema && !status.isSatisfied;
		}).length +
		(webSearchBlocked ? 1 : 0);

	const configuredCount = attached.length + conversationToolsState.length + webSearchTemplates.length;
	const resultCount = visibleAvailableToAdd.length + visibleWebSearchToAdd.length;
	const toolsHeaderMeta = isSearchQueryActive(query)
		? `${resultCount} match${resultCount === 1 ? '' : 'es'}`
		: `${resultCount} available · ${configuredCount} added`;

	const autoExecuteFor = (item: ToolListItem) => {
		if (item.toolDefinition.implementation.kind !== ToolImplType.Go) {
			return false;
		}
		const key = itemKey(item);
		return attachedByKey.get(key)?.autoExecute ?? autoOverrides[key] ?? item.toolDefinition.autoExecute;
	};

	const selectWebSearch = (item: ToolListItem) => {
		setWebSearchTemplates(previous => {
			const template = webSearchTemplateFromToolListItem(item);
			const existing = previous.find(value => webSearchIdentityKey(value) === webSearchIdentityKey(template));
			return [{ ...template, userArgSchemaInstance: existing?.userArgSchemaInstance }];
		});
	};

	const updateConversation = (
		key: string,
		update: (entry: ConversationToolStateEntry) => ConversationToolStateEntry
	) => {
		setConversationToolsState(previous => previous.map(entry => (entry.key === key ? update(entry) : entry)));
	};

	const refresh = () => {
		void reloadOrThrow().catch((error: unknown) => {
			console.error('Failed to refresh the Tool catalog:', error);
		});
	};

	const renderAvailable = (item: ToolListItem) => {
		const key = itemKey(item);
		const toggle = () => {
			onAttachTool(item, autoExecuteFor(item));
		};

		return (
			<ToolMenuRow
				key={key}
				store={store}
				disabled={isInputLocked}
				display={toolDisplayName(item.toolDefinition)}
				slug={item.toolDefinition.description?.trim() || 'Available tool'}
				supportsAutoExecute={item.toolDefinition.implementation.kind === ToolImplType.Go}
				autoExecute={autoExecuteFor(item)}
				onAutoExecuteChange={value => {
					setAutoOverrides(previous => ({ ...previous, [key]: value }));
				}}
				onRowClick={toggle}
				primaryAction={{
					kind: 'attach',
					onClick: toggle,
					disabled: isInputLocked,
				}}
			/>
		);
	};

	const renderAttached = (entry: AttachedToolEntry) => {
		const status = computeToolUserArgsStatus(entry.toolSnapshot?.userArgSchema, entry.userArgSchemaInstance);
		return (
			<ToolMenuRow
				key={entry.selectionID}
				store={store}
				disabled={isInputLocked}
				dataAttachmentChip="bottom-bar-attached-tool"
				dataSelectionId={entry.selectionID}
				display={entry.displayName || entry.target.name}
				slug={entry.description?.trim() || 'Available for this message'}
				sourceBadge="Message"
				isSelected
				supportsAutoExecute={entry.implementationKind === ToolImplType.Go}
				autoExecute={entry.autoExecute}
				onAutoExecuteChange={value => {
					onSetAttachedToolAutoExecute(toolIdentityKey(entry.target), value);
				}}
				argsStatus={status}
				onEditOptions={
					status.hasSchema
						? () => {
								onEditAttachedToolOptions(entry);
							}
						: undefined
				}
				onShowDetails={
					onOpenAttachedToolDetails
						? () => {
								onOpenAttachedToolDetails(entry);
							}
						: undefined
				}
				primaryAction={{
					kind: 'remove',
					onClick: () => {
						onRemoveAttachedTool(entry);
					},
					title: 'Remove per-message tool',
					disabled: isInputLocked,
				}}
			/>
		);
	};

	const renderConversation = (entry: ConversationToolStateEntry) => {
		const choice = entry.toolStoreChoice;
		const status =
			entry.argStatus ??
			(entry.toolDefinition
				? computeToolUserArgsStatus(entry.toolDefinition.userArgSchema, choice.userArgSchemaInstance)
				: undefined);

		return (
			<ToolMenuRow
				key={entry.key}
				store={store}
				disabled={isInputLocked}
				dataAttachmentChip="bottom-bar-conversation-tool"
				title={entry.toolLoadError || (!entry.toolDefinition ? 'Tool definition is being resolved.' : undefined)}
				display={choice.displayName || choice.target.name}
				slug={choice.description?.trim() || 'Available for this conversation'}
				sourceBadge="Conversation"
				isSelected={entry.enabled}
				selectedTitle={entry.enabled ? 'Enabled for next send' : 'Disabled'}
				supportsAutoExecute={choice.implementationKind === ToolImplType.Go}
				autoExecute={choice.autoExecute}
				onAutoExecuteChange={value => {
					updateConversation(entry.key, previous => ({
						...previous,
						toolStoreChoice: { ...previous.toolStoreChoice, autoExecute: value },
					}));
				}}
				argsStatus={status}
				onEditOptions={
					status?.hasSchema
						? () => {
								dispatchOpenToolArgs({ kind: 'conversation', key: entry.key }, toolArgsEventTarget);
							}
						: undefined
				}
				onShowDetails={
					onOpenConversationToolDetails
						? () => {
								onOpenConversationToolDetails(entry);
							}
						: undefined
				}
				onRowClick={() => {
					updateConversation(entry.key, previous => ({ ...previous, enabled: !previous.enabled }));
				}}
				primaryAction={{
					kind: 'remove',
					onClick: () => {
						setConversationToolsState(previous => previous.filter(value => value.key !== entry.key));
					},
					title: 'Remove conversation tool',
					disabled: isInputLocked,
				}}
			/>
		);
	};

	const renderWebSearch = (item: ToolListItem) => {
		const selected = activeWebSearch && itemKey(item) === webSearchIdentityKey(activeWebSearch);
		const hasSchema = item.toolDefinition.userArgSchema !== undefined;

		return (
			<MenuItem
				key={itemKey(item)}
				store={store}
				hideOnClick={false}
				disabled={isInputLocked}
				data-searchable-menu-item="true"
				className="data-active-item:bg-base-200 mb-1 rounded-xl"
				onClick={() => {
					selectWebSearch(item);
				}}
			>
				<div className="flex items-center gap-2 px-2 py-1">
					<FiGlobe size={14} />
					<div className="min-w-0 flex-1">
						<div className="truncate text-xs font-medium">{toolDisplayName(item.toolDefinition)}</div>
						<div className="text-base-content/70 truncate text-xs">
							{item.toolDefinition.description?.trim() || 'Provider web search'}
						</div>
					</div>
					{selected ? <FiCheck size={14} className="text-primary" /> : null}
					{selected && webSearchStatus?.hasSchema ? (
						<span className={`badge badge-xs ${webSearchStatus.isSatisfied ? 'badge-success' : 'badge-warning'}`}>
							{webSearchStatus.isSatisfied ? 'Args: OK' : 'Args: Invalid'}
						</span>
					) : null}
					{selected && hasSchema ? (
						<button
							type="button"
							className="btn btn-ghost btn-xs"
							title="Edit web search options"
							aria-label="Edit web search options"
							onClick={event => {
								stop(event);
								dispatchOpenToolArgs({ kind: 'webSearch' }, toolArgsEventTarget);
							}}
						>
							<FiEdit2 size={12} />
						</button>
					) : null}
					{selected ? (
						<button
							type="button"
							className="btn btn-ghost btn-xs text-error"
							title="Disable web search"
							aria-label="Disable web search"
							onClick={event => {
								stop(event);
								setWebSearchTemplates([]);
							}}
						>
							<FiX size={12} />
						</button>
					) : null}
				</div>
			</MenuItem>
		);
	};

	const hoverContent = shortcut ? `Attach tools (${shortcut})` : 'Attach tools';

	const selectedToolsPanel =
		configuredCount > 0 ? (
			<>
				{activeWebSearchItem ? renderWebSearch(activeWebSearchItem) : null}
				{activeWebSearch && !activeWebSearchItem ? (
					<div className="alert alert-warning rounded-xl p-2 text-xs">
						<div className="min-w-0 flex-1">
							<div>{activeWebSearch.displayName || activeWebSearch.target.name}</div>
							<div>This web-search selection is unavailable for the current provider.</div>
						</div>
						<button
							type="button"
							className="btn btn-ghost btn-xs text-error"
							disabled={isInputLocked}
							onClick={() => {
								setWebSearchTemplates([]);
							}}
						>
							Remove
						</button>
					</div>
				) : null}

				{attached.map(a => {
					return renderAttached(a);
				})}
				{conversationToolsState.map(a => {
					return renderConversation(a);
				})}
			</>
		) : undefined;

	return (
		<div className="relative shrink-0" data-bottom-bar-tools>
			<HoverTip content={hoverContent} placement="top" wrapperElement="div" wrapperClassName="inline-flex max-w-full">
				<MenuButton
					ref={buttonRef}
					store={store}
					disabled={isInputLocked}
					className={`${actionTriggerChipButtonClasses} ${
						missingCount
							? 'border-warning/70 bg-warning/10 hover:bg-warning/15'
							: configuredCount
								? 'border-primary/50 bg-primary/10'
								: 'border-transparent'
					} ${isInputLocked ? 'opacity-60' : ''}`}
					aria-label={shortcut ? `Attach tools (${shortcut})` : 'Attach tools'}
				>
					<ActionTriggerChipContent
						icon={<FiTool size={14} />}
						label="Tools"
						suffix={
							missingCount ? (
								<span className="badge badge-warning badge-xs">Check</span>
							) : configuredCount > 0 ? (
								<FiCheck size={14} aria-hidden="true" />
							) : undefined
						}
						open={open}
					/>
				</MenuButton>
			</HoverTip>

			<ActionTriggerMenu store={store} data-menu-kind="tools">
				{open ? (
					<>
						<ActionTriggerMenuHeader>
							<SearchableMenuInput
								open={open}
								query={query}
								onQueryChange={setQuery}
								placeholder="Search tools..."
								disabled={isLoading && !hasResolved}
								onFocusFirstItem={() => focusFirstSearchableMenuItem(contentElement)}
								onEnterFirstResult={() => {
									if (isInputLocked) {
										return;
									}
									if (visibleWebSearch[0]) {
										selectWebSearch(visibleWebSearch[0]);
										return;
									}
									const first = visibleAvailable.find(item => !attachedByKey.has(itemKey(item)));
									if (first) {
										onAttachTool(first, autoExecuteFor(first));
									}
								}}
								onEscape={() => {
									store.hide();
								}}
							/>

							<div className={actionTriggerMenuHeaderRowClasses}>
								<span className="min-w-0 flex-1 truncate text-xs font-semibold">Tools</span>
								<div className="flex shrink-0 items-center gap-2">
									<span className={actionTriggerMenuMetaClasses}>
										{isLoading || isRefreshing ? 'Loading…' : toolsHeaderMeta}
									</span>
									<button
										type="button"
										className="btn btn-ghost btn-xs rounded-lg"
										disabled={isLoading || isRefreshing || isInputLocked}
										onClick={refresh}
									>
										<FiRefreshCw size={12} />
										Refresh
									</button>
								</div>
							</div>

							{catalogError ? (
								<div className="alert alert-warning rounded-xl p-2 text-xs">
									The Tool catalog could not be loaded. Refresh to retry.
								</div>
							) : null}
						</ActionTriggerMenuHeader>

						<ActionTriggerMenuBody className="space-y-2">
							<GroupedMenuSection title="Web search" ariaLabel="Web search tools">
								{visibleWebSearchToAdd.length > 0 ? (
									<>
										{!activeWebSearch ? (
											<button
												type="button"
												className="btn btn-ghost btn-xs"
												disabled={isInputLocked}
												onClick={() => {
													selectWebSearch(visibleWebSearchToAdd[0]);
												}}
											>
												Enable web search
											</button>
										) : null}
										{visibleWebSearchToAdd.map(v => {
											return renderWebSearch(v);
										})}
									</>
								) : (
									<div className={actionTriggerMenuItemClasses}>
										{activeWebSearch
											? 'No additional web-search tool is available for this provider.'
											: 'No matching web-search tool is available for this provider.'}
									</div>
								)}
							</GroupedMenuSection>

							<GroupedMenuSection title="Available tools" ariaLabel="Available tools" separatorBefore>
								{groups.length > 0 ? (
									groups.map((group, index) => (
										<GroupedMenuSection
											key={group.key}
											title={group.name}
											ariaLabel={`${group.name} tools`}
											separatorBefore={index > 0}
											meta={
												<span className="text-base-content/60 text-[11px]">{group.available.length} available</span>
											}
										>
											{group.available.map(renderAvailable)}
										</GroupedMenuSection>
									))
								) : (
									<div className={actionTriggerMenuItemClasses}>No matching tools are available.</div>
								)}
							</GroupedMenuSection>

							{missingCount ? (
								<div className="alert alert-warning mt-2 rounded-xl p-2 text-xs">
									<FiAlertTriangle size={14} />
									<span>Resolve unavailable tools and invalid tool options before sending.</span>
								</div>
							) : null}
						</ActionTriggerMenuBody>

						<ComposerDropdownSelectedPanel
							label="Added tools"
							ariaLabel="Added tools"
							isEmpty={configuredCount === 0}
							emptyLabel="Choose a tool or enable web search above to add it to this message."
							help="Added tools are available for the next message."
							summary={
								configuredCount > 0 ? (
									<>
										<span>{configuredCount} added</span>
										{missingCount > 0 ? <span className="text-warning">{missingCount} need attention</span> : null}
									</>
								) : undefined
							}
							actions={
								configuredCount > 0 ? (
									<button
										type="button"
										className="btn btn-ghost btn-xs text-error rounded-lg"
										disabled={isInputLocked}
										onClick={() => {
											onRemoveAllAttachedTools(attached);
											setConversationToolsState([]);
											setWebSearchTemplates([]);
										}}
									>
										Clear all
									</button>
								) : undefined
							}
						>
							{selectedToolsPanel}
						</ComposerDropdownSelectedPanel>
					</>
				) : null}
			</ActionTriggerMenu>
		</div>
	);
}
