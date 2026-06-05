import {
	type Dispatch,
	type MouseEvent,
	type RefObject,
	type SetStateAction,
	type SyntheticEvent,
	useEffect,
	useMemo,
	useState,
} from 'react';

import { FiAlertTriangle, FiCheck, FiEdit2, FiGlobe, FiTool, FiX } from 'react-icons/fi';

import { Menu, MenuButton, MenuItem, type MenuStore, useStoreState } from '@ariakit/react';

import type { ProviderSDKType } from '@/spec/inference';
import { ToolImplType, type ToolListItem, ToolStoreChoiceType } from '@/spec/tool';

import { useTools } from '@/hooks/use_tool';

import {
	ActionTriggerChipContent,
	actionTriggerChipSurfaceClasses,
	actionTriggerMenuItemClasses,
	actionTriggerMenuWideClasses,
} from '@/components/action_trigger_chip';
import { HoverTip } from '@/components/ariakit_hover_tip';
import { GroupedMenuSection, GroupedMenuSubheading } from '@/components/grouped_menu_sections';

import type { AttachedToolEntry } from '@/chats/composer/platedoc/tool_document_ops';
import { dispatchOpenToolArgs } from '@/chats/composer/toolruntime/use_open_toolargs_event';
import { ToolMenuRow } from '@/chats/composer/tools/tool_menu_row';
import {
	getEligibleWebSearchTools,
	normalizeWebSearchChoiceTemplates,
	type WebSearchChoiceTemplate,
	webSearchIdentityKey,
	webSearchTemplateFromToolListItem,
} from '@/chats/composer/tools/websearch_utils';
import type { ConversationToolStateEntry } from '@/tools/lib/conversation_tool_utils';
import { toolIdentityKey } from '@/tools/lib/tool_identity_utils';
import { computeToolUserArgsStatus } from '@/tools/lib/tool_userargs_utils';

type ToolBundleGroup = {
	bundleID: string;
	bundleSlug: string;
	isBuiltIn: boolean;
	attachedOptions: ToolListItem[];
	availableOptions: ToolListItem[];
};

type ToolsBottomBarChipProps = {
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
	onWebSearchArgsBlockedChange?: (blocked: boolean) => void;
	toolArgsEventTarget?: EventTarget | null;

	isInputLocked?: boolean;
};

const toolMenuCollator = new Intl.Collator(undefined, {
	numeric: true,
	sensitivity: 'base',
});

function stop(e: SyntheticEvent | MouseEvent) {
	e.preventDefault();
	e.stopPropagation();
}

function getToolKey(item: ToolListItem): string {
	return toolIdentityKey(item.bundleID, item.bundleSlug, item.toolSlug, item.toolVersion);
}

function getAttachedToolKey(entry: AttachedToolEntry): string {
	return toolIdentityKey(entry.bundleID, entry.bundleSlug, entry.toolSlug, entry.toolVersion);
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

function getArgsBadgeClass(hasBlockingArgs: boolean) {
	return hasBlockingArgs ? 'badge badge-warning badge-xs animate-pulse' : 'badge badge-success badge-xs bg-success/30';
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
	onDetachToolByKey,
	onSetAttachedToolAutoExecute,
	onRemoveAttachedTool,
	onRemoveAllAttachedTools,
	onEditAttachedToolOptions,
	onOpenAttachedToolDetails,
	onOpenConversationToolDetails,
	webSearchTemplates,
	setWebSearchTemplates,
	onWebSearchArgsBlockedChange,
	toolArgsEventTarget,
	isInputLocked = false,
}: ToolsBottomBarChipProps) {
	const open = useStoreState(store, 'open');
	const { data: toolData, loading: toolsLoading } = useTools();

	const [toolAutoExecOverrides, setToolAutoExecOverrides] = useState<Record<string, boolean>>({});

	useEffect(() => {
		if (isInputLocked) store.hide();
	}, [isInputLocked, store]);

	const attachedToolKeys = useMemo(() => {
		return new Set(attachedToolEntries.map(getAttachedToolKey));
	}, [attachedToolEntries]);

	const visibleAttachedToolEntries = useMemo(
		() => attachedToolEntries.filter(entry => entry.toolType !== ToolStoreChoiceType.WebSearch),
		[attachedToolEntries]
	);

	const attachedAutoExecByKey = useMemo(() => {
		const map: Record<string, boolean> = {};
		for (const entry of visibleAttachedToolEntries) {
			map[getAttachedToolKey(entry)] = entry.autoExecute;
		}
		return map;
	}, [visibleAttachedToolEntries]);

	const getAutoExecForTool = useMemo(() => {
		return (item: ToolListItem): boolean => {
			const key = getToolKey(item);
			if (typeof attachedAutoExecByKey[key] === 'boolean') return attachedAutoExecByKey[key];
			const override = toolAutoExecOverrides[key];
			if (typeof override === 'boolean') return override;
			return item.toolDefinition.autoExecReco ?? false;
		};
	}, [attachedAutoExecByKey, toolAutoExecOverrides]);

	const eligibleWebSearchTools = useMemo(() => {
		if (toolsLoading) return [];
		return getEligibleWebSearchTools(toolData, currentProviderSDKType);
	}, [currentProviderSDKType, toolData, toolsLoading]);

	const eligibleWebSearchKeys = useMemo(() => {
		return new Set(
			eligibleWebSearchTools.map(tool =>
				webSearchIdentityKey({
					bundleID: tool.bundleID,
					toolSlug: tool.toolSlug,
					toolVersion: tool.toolVersion,
				})
			)
		);
	}, [eligibleWebSearchTools]);

	const compatibleWebSearchTemplates = useMemo(
		() =>
			normalizeWebSearchChoiceTemplates(
				webSearchTemplates.filter(template => eligibleWebSearchKeys.has(webSearchIdentityKey(template)))
			),
		[eligibleWebSearchKeys, webSearchTemplates]
	);

	const activeWebSearch = compatibleWebSearchTemplates[0];
	const webSearchEnabled = Boolean(activeWebSearch);

	useEffect(() => {
		if (toolsLoading) return;

		// eslint-disable-next-line react-you-might-not-need-an-effect/no-pass-data-to-parent
		setWebSearchTemplates(prev => {
			if (prev.length === 0) return prev;
			const next = prev.filter(template => eligibleWebSearchKeys.has(webSearchIdentityKey(template)));
			return next.length === prev.length ? prev : next;
		});
	}, [eligibleWebSearchKeys, setWebSearchTemplates, toolsLoading]);

	const activeWebSearchDef = useMemo(() => {
		if (!activeWebSearch) return undefined;
		return eligibleWebSearchTools.find(
			tool =>
				tool.bundleID === activeWebSearch.bundleID &&
				tool.toolSlug === activeWebSearch.toolSlug &&
				tool.toolVersion === activeWebSearch.toolVersion
		);
	}, [activeWebSearch, eligibleWebSearchTools]);

	const activeWebSearchArgsStatus = useMemo(() => {
		const schema = activeWebSearchDef?.toolDefinition.userArgSchema;
		if (!schema || !activeWebSearch) return undefined;
		return computeToolUserArgsStatus(schema, activeWebSearch.userArgSchemaInstance);
	}, [activeWebSearch, activeWebSearchDef]);

	const hasConfiguredWebSearch = webSearchTemplates.length > 0;
	const webSearchSelectionPendingNormalization =
		hasConfiguredWebSearch && compatibleWebSearchTemplates.length !== webSearchTemplates.length;

	const webSearchDefinitionPending =
		hasConfiguredWebSearch &&
		(toolsLoading || webSearchSelectionPendingNormalization || (activeWebSearch && !activeWebSearchDef));
	const webSearchArgsBlocked =
		webSearchDefinitionPending ||
		Boolean(webSearchEnabled && activeWebSearchArgsStatus?.hasSchema && !activeWebSearchArgsStatus.isSatisfied);

	useEffect(() => {
		// eslint-disable-next-line react-you-might-not-need-an-effect/no-pass-data-to-parent
		onWebSearchArgsBlockedChange?.(webSearchArgsBlocked);
	}, [onWebSearchArgsBlockedChange, webSearchArgsBlocked]);

	const availableTools = useMemo<ToolListItem[]>(() => {
		if (toolsLoading) return [];
		const providerSDKType = currentProviderSDKType.toString();

		return toolData.filter(item => {
			if (item.toolDefinition.llmToolType === ToolStoreChoiceType.WebSearch) return false;

			if (item.toolDefinition.type === ToolImplType.SDK && item.toolDefinition.sdkImpl) {
				const sdkType = item.toolDefinition.sdkImpl.sdkType;
				if (!sdkType) return false;
				return sdkType === providerSDKType;
			}

			return true;
		});
	}, [currentProviderSDKType, toolData, toolsLoading]);

	const groupedTools = useMemo(() => groupTools(availableTools, attachedToolKeys), [availableTools, attachedToolKeys]);

	const attachedArgsMissingCount = useMemo(() => {
		let count = 0;
		for (const entry of visibleAttachedToolEntries) {
			const status = computeToolUserArgsStatus(entry.toolSnapshot?.userArgSchema, entry.userArgSchemaInstance);
			if (status.hasSchema && !status.isSatisfied) count += 1;
		}
		return count;
	}, [visibleAttachedToolEntries]);

	const conversationArgsMissingCount = useMemo(() => {
		let count = 0;
		for (const entry of conversationToolsState) {
			if (!entry.enabled) continue;
			const status =
				entry.argStatus ??
				(entry.toolDefinition
					? computeToolUserArgsStatus(entry.toolDefinition.userArgSchema, entry.toolStoreChoice.userArgSchemaInstance)
					: undefined);
			if (status?.hasSchema && !status.isSatisfied) count += 1;
		}
		return count;
	}, [conversationToolsState]);

	const missingArgsCount = attachedArgsMissingCount + conversationArgsMissingCount + (webSearchArgsBlocked ? 1 : 0);

	const enabledConversationToolCount = conversationToolsState.filter(entry => entry.enabled).length;
	const configuredToolCount =
		visibleAttachedToolEntries.length + enabledConversationToolCount + compatibleWebSearchTemplates.length;

	const title = useMemo(() => {
		const lines: string[] = [];
		lines.push(shortcut ? `Tools (${shortcut})` : 'Tools');
		lines.push('Choose per-message tools, conversation tools, and web search.');
		lines.push(configuredToolCount > 0 ? `Configured: ${configuredToolCount}` : 'No tools configured');
		if (visibleAttachedToolEntries.length > 0) lines.push(`Per-message tools: ${visibleAttachedToolEntries.length}`);
		if (enabledConversationToolCount > 0) lines.push(`Conversation tools: ${enabledConversationToolCount}`);
		if (webSearchEnabled) lines.push('Web search: enabled');
		if (missingArgsCount > 0) lines.push(`Missing required options: ${missingArgsCount}`);
		return lines.join('\n');
	}, [
		configuredToolCount,
		enabledConversationToolCount,
		missingArgsCount,
		shortcut,
		visibleAttachedToolEntries.length,
		webSearchEnabled,
	]);

	const chipToneClasses =
		missingArgsCount > 0
			? 'border-warning/70 bg-warning/10 hover:bg-warning/15 animate-pulse'
			: configuredToolCount > 0
				? 'border-primary/50 bg-primary/10 hover:bg-primary/15'
				: open
					? 'border-base-300 bg-base-300/60'
					: 'border-transparent';

	const clearAllConfiguredTools = () => {
		if (visibleAttachedToolEntries.length > 0) {
			onRemoveAllAttachedTools(visibleAttachedToolEntries);
		}
		if (conversationToolsState.length > 0) {
			setConversationToolsState([]);
		}
		if (webSearchTemplates.length > 0) {
			setWebSearchTemplates([]);
		}
		store.hide();
	};

	const handleAttachToolPick = (item: ToolListItem) => {
		onAttachTool(item, getAutoExecForTool(item));
	};

	const handleConversationToolAutoExecute = (key: string, nextAutoExecute: boolean) => {
		setConversationToolsState(prev =>
			prev.map(entry =>
				entry.key === key
					? {
							...entry,
							toolStoreChoice: {
								...entry.toolStoreChoice,
								autoExecute: nextAutoExecute,
							},
						}
					: entry
			)
		);
	};

	const handleConversationToolEnabled = (key: string, nextEnabled: boolean) => {
		setConversationToolsState(prev =>
			prev.map(entry =>
				entry.key === key
					? {
							...entry,
							enabled: nextEnabled,
						}
					: entry
			)
		);
	};

	const handleRemoveConversationTool = (key: string) => {
		setConversationToolsState(prev => prev.filter(entry => entry.key !== key));
	};

	const handleWebSearchEnabled = (enabled: boolean) => {
		if (!enabled) {
			setWebSearchTemplates([]);
			return;
		}

		if (compatibleWebSearchTemplates.length > 0 || eligibleWebSearchTools.length === 0) return;

		const first = eligibleWebSearchTools[0];
		setWebSearchTemplates([webSearchTemplateFromToolListItem(first)]);
	};

	const handleWebSearchToolSelected = (tool: ToolListItem) => {
		setWebSearchTemplates(prev => {
			const nextTemplate = webSearchTemplateFromToolListItem(tool);
			const previousSameTool = prev.find(
				template => webSearchIdentityKey(template) === webSearchIdentityKey(nextTemplate)
			);

			return [
				{
					...nextTemplate,
					userArgSchemaInstance: previousSameTool?.userArgSchemaInstance,
				},
			];
		});
	};

	const orderedWebSearchTools = useMemo(() => {
		if (!activeWebSearch) return eligibleWebSearchTools;
		const activeKey = webSearchIdentityKey(activeWebSearch);
		return [...eligibleWebSearchTools].sort((a, b) => {
			const aActive = webSearchIdentityKey(a) === activeKey;
			const bActive = webSearchIdentityKey(b) === activeKey;
			if (aActive !== bActive) return aActive ? -1 : 1;
			return compareToolListItems(a, b);
		});
	}, [activeWebSearch, eligibleWebSearchTools]);

	const renderWebSearchRow = (tool: ToolListItem) => {
		const key = webSearchIdentityKey(tool);
		const isActive = activeWebSearch ? webSearchIdentityKey(activeWebSearch) === key : false;
		const schema = tool.toolDefinition.userArgSchema;
		const canEdit = isActive && Boolean(schema);
		const status = isActive ? activeWebSearchArgsStatus : undefined;
		const isArgsBad = Boolean(status?.hasSchema && !status.isSatisfied);
		const display = tool.toolDefinition.displayName || tool.toolSlug || 'Web search';
		const slug = `${tool.bundleSlug ?? tool.bundleID}/${tool.toolSlug}@${tool.toolVersion}`;

		return (
			<MenuItem
				key={key}
				hideOnClick={false}
				className={`data-active-item:bg-base-200 mb-1 rounded-xl outline-none last:mb-0 ${isActive ? 'bg-base-200' : ''}`}
				onClick={() => {
					if (isInputLocked) return;
					handleWebSearchToolSelected(tool);
				}}
				title={`Web search tool: ${display} (${slug})`}
			>
				<div className="grid grid-cols-12 items-center gap-x-2 px-2 py-1">
					<div className="col-span-8 flex min-w-0 items-center gap-1">
						<FiGlobe size={14} />
						<div className="min-w-0 flex-1">
							<div className="truncate text-xs font-medium">{display}</div>
							<div className="text-base-content/70 truncate text-xs">{slug}</div>
						</div>
						{isActive ? <FiCheck size={14} className="text-primary shrink-0" /> : null}
					</div>

					<div className="col-span-2 flex justify-center">
						{isActive ? (
							<span className="badge badge-info badge-xs">Active</span>
						) : (
							<span className="text-base-content/40 text-xs">Available</span>
						)}
					</div>

					<div className="col-span-2 flex items-center justify-end gap-1">
						{status?.hasSchema ? (
							<span
								className={isArgsBad ? 'badge badge-warning badge-xs animate-pulse' : 'badge badge-success badge-xs'}
							>
								{isArgsBad ? `Args: ${status.missingRequired.length}` : 'Args: OK'}
							</span>
						) : null}

						{canEdit ? (
							<button
								type="button"
								className="btn btn-ghost btn-xs shrink-0 px-1 py-0 shadow-none"
								onClick={e => {
									stop(e);
									dispatchOpenToolArgs({ kind: 'webSearch' }, toolArgsEventTarget);
								}}
								title="Edit web search options"
								aria-label="Edit web search options"
							>
								<FiEdit2 size={12} />
							</button>
						) : null}

						{isActive ? (
							<button
								type="button"
								className="btn btn-ghost btn-xs text-error shrink-0 px-1 py-0 shadow-none"
								onClick={e => {
									stop(e);
									handleWebSearchEnabled(false);
								}}
								title="Disable web search"
								aria-label="Disable web search"
							>
								<FiX size={12} />
							</button>
						) : null}
					</div>
				</div>
			</MenuItem>
		);
	};

	const renderAttachedToolRow = (entry: AttachedToolEntry) => {
		const rawDisplay = entry.toolSnapshot?.displayName ?? entry.toolSlug;
		const display = rawDisplay && rawDisplay.length > 0 ? rawDisplay : 'Tool';
		const slug = `${entry.bundleSlug ?? entry.bundleID}/${entry.toolSlug}@${entry.toolVersion}`;
		const key = getAttachedToolKey(entry);
		const status = computeToolUserArgsStatus(entry.toolSnapshot?.userArgSchema, entry.userArgSchemaInstance);
		const supportsAutoExecute =
			entry.toolType === ToolStoreChoiceType.Function || entry.toolType === ToolStoreChoiceType.Custom;
		const hasArgs = status.hasSchema;

		return (
			<ToolMenuRow
				key={entry.selectionID}
				store={store}
				menuItemClassName="data-active-item:bg-base-200 mb-1 rounded-xl last:mb-0"
				contentClassName="grid grid-cols-12 items-center gap-x-2 px-2 py-1"
				dataAttachmentChip="bottom-bar-attached-tool"
				dataSelectionId={entry.selectionID}
				title={`Per-message tool: ${display} (${slug})`}
				display={display}
				slug={slug}
				isSelected={true}
				supportsAutoExecute={supportsAutoExecute}
				autoExecute={entry.autoExecute}
				onAutoExecuteChange={next => {
					onSetAttachedToolAutoExecute(key, next);
				}}
				argsStatus={status}
				editIcon={<FiEdit2 size={12} />}
				onEditOptions={
					hasArgs
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
				}}
			/>
		);
	};

	const renderConversationToolRow = (entry: ConversationToolStateEntry) => {
		const { key, toolStoreChoice } = entry;
		const display =
			(toolStoreChoice.displayName && toolStoreChoice.displayName.length > 0
				? toolStoreChoice.displayName
				: toolStoreChoice.toolSlug) || 'Tool';
		const slug = `${toolStoreChoice.bundleID ?? 'bundle'}/${toolStoreChoice.toolSlug}@${toolStoreChoice.toolVersion}`;

		const supportsAutoExecute =
			toolStoreChoice.toolType === ToolStoreChoiceType.Function ||
			toolStoreChoice.toolType === ToolStoreChoiceType.Custom;

		const status =
			entry.argStatus ??
			(entry.toolDefinition
				? computeToolUserArgsStatus(entry.toolDefinition.userArgSchema, entry.toolStoreChoice.userArgSchemaInstance)
				: undefined);
		const hasArgs = status?.hasSchema ?? false;

		return (
			<ToolMenuRow
				key={key}
				store={store}
				menuItemClassName={`data-active-item:bg-base-200 mb-1 rounded-xl last:mb-0 ${
					entry.enabled ? 'bg-primary/10' : ''
				}`}
				contentClassName="grid grid-cols-12 items-center gap-x-2 px-2 py-1"
				dataAttachmentChip="bottom-bar-conversation-tool"
				title={`Conversation tool: ${display} (${slug})`}
				display={display}
				slug={slug}
				isSelected={entry.enabled}
				selectedTitle={entry.enabled ? 'Enabled for next send' : 'Disabled'}
				selectedAriaLabel={entry.enabled ? 'Enabled' : 'Disabled'}
				supportsAutoExecute={supportsAutoExecute}
				autoExecute={entry.toolStoreChoice.autoExecute}
				onAutoExecuteChange={next => {
					handleConversationToolAutoExecute(key, next);
				}}
				argsStatus={status}
				editIcon={<FiEdit2 size={12} />}
				onEditOptions={
					hasArgs
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
					handleConversationToolEnabled(key, !entry.enabled);
				}}
				primaryAction={{
					kind: 'remove',
					onClick: () => {
						handleRemoveConversationTool(key);
					},
					title: 'Remove conversation tool',
				}}
			/>
		);
	};

	const renderAvailableToolRow = (item: ToolListItem) => {
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
					if (isAttached) {
						onSetAttachedToolAutoExecute(key, next);
						return;
					}
					setToolAutoExecOverrides(prev => ({ ...prev, [key]: next }));
				}}
				onRowClick={
					!isAttached
						? () => {
								handleAttachToolPick(item);
							}
						: () => {
								onDetachToolByKey(key);
							}
				}
				primaryAction={
					isAttached
						? {
								kind: 'detach',
								onClick: () => {
									onDetachToolByKey(key);
								},
								title: 'Detach tool',
							}
						: {
								kind: 'attach',
								onClick: () => {
									handleAttachToolPick(item);
								},
								title: 'Attach tool',
								label: 'Attach',
							}
				}
			/>
		);
	};

	return (
		<div className="relative shrink-0" data-bottom-bar-tools>
			<HoverTip content={title} placement="top" wrapperElement="div" wrapperClassName="inline-flex max-w-full">
				<div
					className={`${actionTriggerChipSurfaceClasses} border ${chipToneClasses} ${isInputLocked ? 'opacity-60' : ''}`}
				>
					<MenuButton
						ref={buttonRef}
						store={store}
						disabled={isInputLocked}
						className="btn btn-xs text-neutral-custom h-auto min-h-0 flex-1 gap-0 border-none bg-transparent px-0 py-0 text-left font-normal shadow-none hover:bg-transparent"
						aria-label={shortcut ? `Tools (${shortcut})` : 'Tools'}
					>
						<ActionTriggerChipContent
							icon={<FiTool size={14} />}
							label="Tools"
							count={
								configuredToolCount > 0 ? (
									<span className={getArgsBadgeClass(missingArgsCount > 0)}>{configuredToolCount}</span>
								) : undefined
							}
							suffix={
								missingArgsCount > 0 ? (
									<span className="badge badge-warning badge-xs">Args {missingArgsCount}</span>
								) : configuredToolCount > 0 ? (
									<FiCheck size={14} className="shrink-0" />
								) : undefined
							}
							open={open}
						/>
					</MenuButton>

					{configuredToolCount > 0 || webSearchTemplates.length > 0 || conversationToolsState.length > 0 ? (
						<button
							type="button"
							className="btn btn-ghost btn-xs text-neutral-custom hover:bg-base-300/80 ml-1 h-auto min-h-0 shrink-0 px-1 py-0 shadow-none"
							onClick={event => {
								stop(event);
								clearAllConfiguredTools();
							}}
							aria-label="Clear tools"
							title="Clear tools"
							disabled={isInputLocked}
						>
							<FiX size={12} />
						</button>
					) : null}
				</div>
			</HoverTip>

			<Menu
				store={store}
				gutter={8}
				overflowPadding={8}
				portal
				className={actionTriggerMenuWideClasses}
				data-menu-kind="tools"
				autoFocusOnShow
			>
				{!open ? null : toolsLoading ? (
					<div className={`${actionTriggerMenuItemClasses} text-base-content/60 cursor-default`}>Loading tools…</div>
				) : (
					<div className="space-y-2">
						<GroupedMenuSection
							title="Web search"
							ariaLabel="Web search tools"
							meta={
								<>
									<span className="badge badge-ghost badge-xs">{eligibleWebSearchTools.length}</span>
									{webSearchEnabled ? <span className="badge badge-info badge-xs">enabled</span> : null}
									{webSearchArgsBlocked ? (
										<span className="badge badge-warning badge-xs animate-pulse">args</span>
									) : null}
								</>
							}
						>
							{eligibleWebSearchTools.length === 0 ? (
								<div className={`${actionTriggerMenuItemClasses} text-base-content/60 cursor-default`}>
									No web-search tool is available for this model provider.
								</div>
							) : (
								<>
									<div className="flex items-center justify-between gap-2 px-2 py-1 text-xs">
										<div className="text-base-content/70">
											{webSearchEnabled ? 'The active web-search tool is listed first.' : 'Select a web-search tool.'}
										</div>
										<button
											type="button"
											className="btn btn-ghost btn-xs rounded-lg"
											disabled={isInputLocked || eligibleWebSearchTools.length === 0}
											onClick={e => {
												stop(e);
												handleWebSearchEnabled(!webSearchEnabled);
											}}
										>
											{webSearchEnabled ? 'Disable' : 'Enable'}
										</button>
									</div>

									<div className="space-y-1">{orderedWebSearchTools.map(renderWebSearchRow)}</div>
								</>
							)}
						</GroupedMenuSection>

						{visibleAttachedToolEntries.length > 0 ? (
							<GroupedMenuSection
								title="This message"
								ariaLabel="Per-message tools"
								separatorBefore
								meta={<span className="badge badge-ghost badge-xs">{visibleAttachedToolEntries.length}</span>}
							>
								<div className="space-y-1">{visibleAttachedToolEntries.map(renderAttachedToolRow)}</div>
							</GroupedMenuSection>
						) : null}

						{conversationToolsState.length > 0 ? (
							<GroupedMenuSection
								title="Conversation tools"
								ariaLabel="Conversation tools"
								separatorBefore
								meta={
									<>
										<span className="badge badge-ghost badge-xs">
											{enabledConversationToolCount}/{conversationToolsState.length}
										</span>
										{conversationArgsMissingCount > 0 ? (
											<span className="badge badge-warning badge-xs animate-pulse">
												Args {conversationArgsMissingCount}
											</span>
										) : null}
									</>
								}
							>
								<div className="space-y-1">{conversationToolsState.map(renderConversationToolRow)}</div>
							</GroupedMenuSection>
						) : null}

						<GroupedMenuSection title="Available tools" ariaLabel="Available tools" separatorBefore>
							{availableTools.length === 0 ? (
								<div className={`${actionTriggerMenuItemClasses} text-base-content/60 cursor-default`}>
									No tools available
								</div>
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
														<span className="badge badge-ghost badge-xs">
															{group.isBuiltIn ? 'built-in' : 'custom'}
														</span>
													</>
												}
											>
												{group.attachedOptions.length > 0 ? (
													<>
														{showSubheadings ? <GroupedMenuSubheading>Attached</GroupedMenuSubheading> : null}
														{group.attachedOptions.map(renderAvailableToolRow)}
													</>
												) : null}

												{group.availableOptions.length > 0 ? (
													<>
														{showSubheadings ? (
															<GroupedMenuSubheading separated={group.attachedOptions.length > 0}>
																Available
															</GroupedMenuSubheading>
														) : null}
														{group.availableOptions.map(renderAvailableToolRow)}
													</>
												) : null}
											</GroupedMenuSection>
										);
									})}
								</div>
							)}
						</GroupedMenuSection>

						{missingArgsCount > 0 ? (
							<div className="alert alert-warning mt-2 rounded-xl p-2 text-xs">
								<div className="flex items-center gap-2">
									<FiAlertTriangle size={14} />
									<span>Fill required tool/web-search options before sending.</span>
								</div>
							</div>
						) : null}
					</div>
				)}
			</Menu>
		</div>
	);
}
