import {
	type Dispatch,
	memo,
	type ReactNode,
	type RefObject,
	type SetStateAction,
	useEffect,
	useMemo,
	useState,
} from 'react';

import { FiFilePlus, FiFolder, FiLink, FiPaperclip, FiTool, FiUpload } from 'react-icons/fi';

import { Menu, MenuButton, MenuItem, type MenuStore, useStoreState } from '@ariakit/react';

import type { ProviderSDKType } from '@/spec/inference';
import type { PromptTemplate, PromptTemplateListItem } from '@/spec/prompt';
import type { SkillListItem, SkillRef } from '@/spec/skill';
import { ToolImplType, type ToolListItem, ToolStoreChoiceType } from '@/spec/tool';

import { formatShortcut, type ShortcutConfig } from '@/lib/keyboard_shortcuts';

import { useTools } from '@/hooks/use_tool';

import { promptStoreAPI } from '@/apis/baseapi';

import { actionTriggerChipButtonClasses, ActionTriggerChipContent } from '@/components/action_trigger_chip';
import { HoverTip } from '@/components/ariakit_hover_tip';

import { UrlAttachmentModal } from '@/chats/composer/attachments/attachment_url_modal';
import { CommandTipsMenu } from '@/chats/composer/inputtips/command_tips_menu';
import type { AttachedToolEntry } from '@/chats/composer/platedoc/tool_document_ops';
import { SkillsBottomBarChip } from '@/chats/composer/skills/skill_bottom_bar_chip';
import { SystemPromptDropdown } from '@/chats/composer/systemprompts/system_prompt_dropdown';
import type { ComposerSystemPromptController } from '@/chats/composer/systemprompts/use_composer_system_prompt';
import { dispatchOpenToolArgs } from '@/chats/composer/toolruntime/use_open_toolargs_event';
import { ToolMenuRow } from '@/chats/composer/tools/tool_menu_row';
import { WebSearchBottomBarChip } from '@/chats/composer/tools/web_search_bottom_bar_chip';
import {
	getEligibleWebSearchTools,
	normalizeWebSearchChoiceTemplates,
	type WebSearchChoiceTemplate,
	webSearchIdentityKey,
	webSearchTemplateFromToolListItem,
} from '@/chats/composer/tools/websearch_utils';
import { usePromptTemplates } from '@/prompts/lib/use_prompt_templates';
import { toolIdentityKey } from '@/tools/lib/tool_identity_utils';
import { computeToolUserArgsStatus } from '@/tools/lib/tool_userargs_utils';

interface EditorBottomBarProps {
	onAttachFiles: () => Promise<void> | void;
	onAttachDirectory: () => Promise<void> | void;
	onAttachURL: (url: string) => Promise<void> | void;
	onInsertTemplate: (args: {
		bundleID: string;
		templateSlug: string;
		templateVersion: string;
		template?: PromptTemplate;
	}) => Promise<void> | void;

	templateMenuState: MenuStore;
	toolMenuState: MenuStore;
	attachmentMenuState: MenuStore;

	templateButtonRef: RefObject<HTMLButtonElement | null>;
	toolButtonRef: RefObject<HTMLButtonElement | null>;
	attachmentButtonRef: RefObject<HTMLButtonElement | null>;
	toolArgsEventTarget?: EventTarget | null;

	shortcutConfig: ShortcutConfig;
	currentProviderSDKType: ProviderSDKType;

	attachedToolEntries: AttachedToolEntry[];
	onAttachTool: (item: ToolListItem, autoExecute: boolean) => void;
	onDetachToolByKey: (key: string) => void;
	onSetAttachedToolAutoExecute: (key: string, autoExecute: boolean) => void;

	// Web-search state comes from EditorArea (separate UX/state)
	webSearchTemplates: WebSearchChoiceTemplate[];
	setWebSearchTemplates: Dispatch<SetStateAction<WebSearchChoiceTemplate[]>>;
	onWebSearchArgsBlockedChange?: (blocked: boolean) => void;

	// Skills state comes from EditorArea (conversation-level)
	allSkills: SkillListItem[];
	skillsLoading?: boolean;
	enabledSkillRefs: SkillRef[];
	activeSkillRefs: SkillRef[];
	setEnabledSkillRefs: Dispatch<SetStateAction<SkillRef[]>>;
	onEnableAllSkills: () => void;
	onDisableAllSkills: () => void;
	isInputLocked?: boolean;
	systemPrompt: ComposerSystemPromptController;
}

interface PickerButtonProps {
	label: string;
	icon: ReactNode;
	buttonRef: RefObject<HTMLButtonElement | null>;
	menuState: MenuStore;
	shortcut?: string;
	disabled?: boolean;
}

function PickerButton({ label, icon, buttonRef, menuState, shortcut, disabled }: PickerButtonProps) {
	const open = useStoreState(menuState, 'open');
	const tooltip = shortcut ? `${label} (${shortcut})` : label;

	return (
		<HoverTip content={tooltip} placement="top">
			<MenuButton
				ref={buttonRef}
				store={menuState}
				disabled={disabled}
				className={`${actionTriggerChipButtonClasses} hover:text-base-content ${disabled ? 'opacity-60' : ''}`}
				aria-label={tooltip}
			>
				<ActionTriggerChipContent icon={icon} label={label} open={open} />
			</MenuButton>
		</HoverTip>
	);
}

const menuClasses =
	'rounded-box bg-base-100 text-base-content z-50 max-h-72 max-w-lg min-w-60 overflow-y-auto border border-base-300 p-1 shadow-xl';

const menuItemClasses =
	'flex items-center gap-2 rounded-xl px-2 py-1 text-sm outline-none transition-colors ' +
	'hover:bg-base-200 data-[active-item]:bg-base-300';

/**
 * Isolated wrapper for SystemPromptDropdown so its open/close state changes
 * don't re-render the entire EditorBottomBar.
 */
const SystemPromptSection = memo(function SystemPromptSection({
	systemPrompt,
}: {
	systemPrompt: ComposerSystemPromptController;
}) {
	return (
		<SystemPromptDropdown
			prompts={systemPrompt.prompts}
			bundles={systemPrompt.systemPromptBundles}
			selectedPromptKeys={systemPrompt.selectedPromptKeys}
			preferredBundleID={systemPrompt.preferredSystemPromptBundleID}
			loading={systemPrompt.systemPromptsLoading}
			error={systemPrompt.systemPromptError}
			modelDefaultPrompt={systemPrompt.modelDefaultPrompt}
			includeModelDefault={systemPrompt.includeModelDefault}
			onTogglePrompt={systemPrompt.togglePromptSelection}
			onToggleModelDefault={systemPrompt.setIncludeModelDefault}
			onAddPrompt={systemPrompt.addAndSelectPrompt}
			onClearSelected={systemPrompt.clearSelectedPromptSources}
			onRefreshPrompts={systemPrompt.refreshSystemPrompts}
			getExistingVersions={systemPrompt.getExistingSystemPromptVersions}
		/>
	);
});

export const EditorBottomBar = memo(function EditorBottomBar({
	onAttachFiles,
	onAttachDirectory,
	onAttachURL,
	onInsertTemplate,
	templateMenuState,
	toolMenuState,
	attachmentMenuState,
	templateButtonRef,
	toolButtonRef,
	attachmentButtonRef,
	toolArgsEventTarget,
	shortcutConfig,
	currentProviderSDKType,
	attachedToolEntries,
	onAttachTool,
	onDetachToolByKey,
	onSetAttachedToolAutoExecute,
	webSearchTemplates,
	setWebSearchTemplates,
	onWebSearchArgsBlockedChange,
	allSkills,
	skillsLoading = false,
	enabledSkillRefs,
	activeSkillRefs,
	setEnabledSkillRefs,
	onEnableAllSkills,
	onDisableAllSkills,
	isInputLocked = false,
	systemPrompt,
}: EditorBottomBarProps) {
	const [isUrlModalOpen, setIsUrlModalOpen] = useState(false);
	const templateMenuOpen = useStoreState(templateMenuState, 'open');
	const toolMenuOpen = useStoreState(toolMenuState, 'open');

	const shortcutLabels = useMemo(
		() => ({
			templates: formatShortcut(shortcutConfig.insertTemplate),
			tools: formatShortcut(shortcutConfig.insertTool),
			attachments: formatShortcut(shortcutConfig.insertAttachment),
		}),
		[shortcutConfig]
	);
	const { data: templateData, loading: templatesLoading } = usePromptTemplates();
	const { data: toolData, loading: toolsLoading } = useTools();
	const toolEntries = attachedToolEntries;

	const attachedAutoExecByKey = useMemo(() => {
		const map: Record<string, boolean> = {};
		for (const node of toolEntries) {
			const key = toolIdentityKey(node.bundleID, node.bundleSlug, node.toolSlug, node.toolVersion);
			map[key] = node.autoExecute;
		}
		return map;
	}, [toolEntries]);

	const [toolAutoExecOverrides, setToolAutoExecOverrides] = useState<Record<string, boolean>>({});
	const getAutoExecForTool = useMemo(() => {
		return (item: ToolListItem): boolean => {
			const key = toolIdentityKey(item.bundleID, item.bundleSlug, item.toolSlug, item.toolVersion);
			if (typeof attachedAutoExecByKey[key] === 'boolean') return attachedAutoExecByKey[key];
			const override = toolAutoExecOverrides[key];
			if (typeof override === 'boolean') return override;
			return item.toolDefinition.autoExecReco ?? false;
		};
	}, [toolAutoExecOverrides, attachedAutoExecByKey]);

	const eligibleWebSearchTools = useMemo(() => {
		if (toolsLoading) return [];
		return getEligibleWebSearchTools(toolData, currentProviderSDKType);
	}, [toolData, toolsLoading, currentProviderSDKType]);

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

	const webSearchEnabled = compatibleWebSearchTemplates.length > 0;
	const hasConfiguredWebSearch = webSearchTemplates.length > 0;
	const webSearchSelectionPendingNormalization =
		hasConfiguredWebSearch && compatibleWebSearchTemplates.length !== webSearchTemplates.length;

	useEffect(() => {
		if (isInputLocked) {
			templateMenuState.hide();
			toolMenuState.hide();
			attachmentMenuState.hide();
			setIsUrlModalOpen(false);
		}
	}, [attachmentMenuState, isInputLocked, templateMenuState, toolMenuState]);

	useEffect(() => {
		if (toolsLoading) return;
		setWebSearchTemplates(prev => {
			if (prev.length === 0) return prev;
			const next = prev.filter(template => eligibleWebSearchKeys.has(webSearchIdentityKey(template)));
			return next.length === prev.length ? prev : next;
		});
	}, [eligibleWebSearchKeys, setWebSearchTemplates, toolsLoading]);

	// "Active" web-search tool for the bottom-bar UX (first one wins).
	const activeWebSearch = compatibleWebSearchTemplates.length > 0 ? compatibleWebSearchTemplates[0] : undefined;

	// Try to find the active tool definition from eligible tools (so we can show args status).
	const activeWebSearchDef = useMemo(() => {
		if (!activeWebSearch) return undefined;
		return eligibleWebSearchTools.find(
			t =>
				t.bundleID === activeWebSearch.bundleID &&
				t.toolSlug === activeWebSearch.toolSlug &&
				t.toolVersion === activeWebSearch.toolVersion
		);
	}, [eligibleWebSearchTools, activeWebSearch]);

	const activeWebSearchArgsStatus = useMemo(() => {
		const schema = activeWebSearchDef?.toolDefinition.userArgSchema;
		if (!schema || !activeWebSearch) return undefined;
		return computeToolUserArgsStatus(schema, activeWebSearch.userArgSchemaInstance);
	}, [activeWebSearchDef, activeWebSearch]);

	const webSearchDefinitionPending = Boolean(
		hasConfiguredWebSearch &&
		(toolsLoading || webSearchSelectionPendingNormalization || (activeWebSearch && !activeWebSearchDef))
	);

	const webSearchArgsBlocked =
		webSearchDefinitionPending ||
		Boolean(webSearchEnabled && activeWebSearchArgsStatus?.hasSchema && !activeWebSearchArgsStatus.isSatisfied);

	useEffect(() => {
		onWebSearchArgsBlockedChange?.(webSearchArgsBlocked);
	}, [onWebSearchArgsBlockedChange, webSearchArgsBlocked]);

	const attachedToolKeys = useMemo(() => {
		return new Set(toolEntries.map(n => toolIdentityKey(n.bundleID, n.bundleSlug, n.toolSlug, n.toolVersion)));
	}, [toolEntries]);

	const availableTools = useMemo<ToolListItem[]>(() => {
		if (!toolMenuOpen || toolsLoading) return [];
		const providerSDKType = currentProviderSDKType.toString();

		return toolData.filter(it => {
			// Web search is a specially handled tool
			if (it.toolDefinition.llmToolType === ToolStoreChoiceType.WebSearch) return false;

			// If we know the provider's SDK type, restrict SDK tools to matching ones.
			if (it.toolDefinition.type === ToolImplType.SDK && it.toolDefinition.sdkImpl) {
				const sdkType = it.toolDefinition.sdkImpl.sdkType;
				if (!sdkType) return false;
				return sdkType === providerSDKType;
			}

			// Non-SDK tools (Go/HTTP/etc.) are always shown.
			return true;
		});
	}, [currentProviderSDKType, toolData, toolMenuOpen, toolsLoading]);

	const closeTemplateMenu = () => {
		templateMenuState.hide();
	};

	const closeAttachmentMenu = () => {
		attachmentMenuState.hide();
	};

	const handleTemplatePick = async (item: PromptTemplateListItem) => {
		try {
			const tmpl = await promptStoreAPI.getPromptTemplate(item.bundleID, item.templateSlug, item.templateVersion);
			await onInsertTemplate({
				bundleID: item.bundleID,
				templateSlug: item.templateSlug,
				templateVersion: item.templateVersion,
				template: tmpl,
			});
		} catch {
			await onInsertTemplate({
				bundleID: item.bundleID,
				templateSlug: item.templateSlug,
				templateVersion: item.templateVersion,
			});
		} finally {
			closeTemplateMenu();
		}
	};

	const handleAttachToolPick = (item: ToolListItem) => {
		onAttachTool(item, getAutoExecForTool(item));
	};

	const handleDetachToolPick = (key: string) => {
		onDetachToolByKey(key);
	};

	const handleAttachmentPickFiles = async () => {
		await onAttachFiles();
		closeAttachmentMenu();
	};

	const handleAttachmentPickDirectory = async () => {
		await onAttachDirectory();
		closeAttachmentMenu();
	};

	const handleAttachmentPickURL = () => {
		closeAttachmentMenu();
		setIsUrlModalOpen(true);
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
		// Treat selection as "make this tool active", but preserve any other
		// configured web-search tools (if present) by moving it to the front.
		setWebSearchTemplates((prev: WebSearchChoiceTemplate[]) => {
			const tmpl = webSearchTemplateFromToolListItem(tool);
			const key = webSearchIdentityKey(tmpl);
			const rest = prev.filter(
				(p: { bundleID: string; toolSlug: string; toolVersion: string }) => webSearchIdentityKey(p) !== key
			);
			return [tmpl, ...rest];
		});
	};

	return (
		<div
			className="bg-base-200 w-full overflow-hidden"
			data-attachments-bottom-bar
			aria-label="Templates, tools, attachments, skills, and system prompt"
		>
			<div className="flex items-center gap-1 p-1 text-xs shadow-none">
				<div className="flex items-center gap-1">
					<PickerButton
						label="Attachments"
						icon={<FiPaperclip size={16} />}
						buttonRef={attachmentButtonRef}
						menuState={attachmentMenuState}
						shortcut={shortcutLabels.attachments}
						disabled={isInputLocked}
					/>
					<Menu
						store={attachmentMenuState}
						gutter={8}
						overflowPadding={8}
						portal
						className={menuClasses}
						data-menu-kind="attachments"
						autoFocusOnShow
					>
						<MenuItem
							onClick={() => {
								void handleAttachmentPickFiles();
							}}
							className={menuItemClasses}
						>
							<FiUpload size={14} />
							<span>Multiple Files...</span>
						</MenuItem>
						<MenuItem
							onClick={() => {
								void handleAttachmentPickDirectory();
							}}
							className={menuItemClasses}
						>
							<FiFolder size={14} />
							<span>Folder...</span>
						</MenuItem>
						<MenuItem onClick={handleAttachmentPickURL} className={menuItemClasses}>
							<FiLink size={14} />
							<span>Link or URL...</span>
						</MenuItem>
					</Menu>

					<SystemPromptSection systemPrompt={systemPrompt} />

					<PickerButton
						label="Prompts"
						icon={<FiFilePlus size={16} />}
						buttonRef={templateButtonRef}
						menuState={templateMenuState}
						shortcut={shortcutLabels.templates}
						disabled={isInputLocked}
					/>
					<Menu
						store={templateMenuState}
						gutter={8}
						overflowPadding={8}
						portal
						className={menuClasses}
						data-menu-kind="templates"
						autoFocusOnShow
					>
						{!templateMenuOpen ? null : templatesLoading ? (
							<div className={`${menuItemClasses} text-base-content/60 cursor-default`}>Loading templates…</div>
						) : templateData.length === 0 ? (
							<div className={`${menuItemClasses} text-base-content/60 cursor-default`}>No templates available</div>
						) : (
							templateData.map(item => (
								// For tooltip and display name we use a humanized slug.
								// Note: we use title on the item so long names are fully visible when truncated.
								<MenuItem
									key={`${item.bundleID}-${item.templateSlug}-${item.templateVersion}`}
									onClick={() => {
										void handleTemplatePick(item);
									}}
									className={menuItemClasses}
									title={`${item.templateSlug.replace(/[-_]/g, ' ')} • v${item.templateVersion}`}
								>
									<FiFilePlus size={14} className="text-warning" />
									<span className="truncate">{item.templateSlug.replace(/[-_]/g, ' ')}</span>
									<span className="text-base-content/50 ml-auto text-[10px] uppercase" aria-hidden="true">
										{item.templateVersion}
									</span>
								</MenuItem>
							))
						)}
					</Menu>

					<PickerButton
						label="Tools"
						icon={<FiTool size={16} />}
						buttonRef={toolButtonRef}
						menuState={toolMenuState}
						shortcut={shortcutLabels.tools}
						disabled={isInputLocked}
					/>
					<Menu
						store={toolMenuState}
						gutter={8}
						overflowPadding={8}
						portal
						className={menuClasses}
						data-menu-kind="tools"
						autoFocusOnShow
					>
						{!toolMenuOpen ? null : toolsLoading ? (
							<div className={`${menuItemClasses} text-base-content/60 cursor-default`}>Loading tools…</div>
						) : availableTools.length === 0 ? (
							<div className={`${menuItemClasses} text-base-content/60 cursor-default`}>No tools available</div>
						) : (
							availableTools.map(item => {
								const display = item.toolDefinition.displayName || item.toolSlug || 'Tool';
								const slug = `${item.bundleSlug ?? item.bundleID}/${item.toolSlug}@${item.toolVersion}`;
								const key = toolIdentityKey(item.bundleID, item.bundleSlug, item.toolSlug, item.toolVersion);
								const isAttached = attachedToolKeys.has(key);
								const supportsAutoExecute =
									item.toolDefinition.llmToolType === ToolStoreChoiceType.Function ||
									item.toolDefinition.llmToolType === ToolStoreChoiceType.Custom;

								return (
									<ToolMenuRow
										key={`${item.bundleID}-${item.toolSlug}-${item.toolVersion}`}
										menuItemClassName={`rounded-xl px-0 py-0 text-sm outline-none transition-colors hover:bg-base-200 data-[active-item]:bg-base-300 overflow-hidden ${isAttached ? 'bg-base-200' : ''}`}
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
														handleDetachToolPick(key);
													}
										}
										primaryAction={
											isAttached
												? {
														kind: 'detach',
														onClick: () => {
															handleDetachToolPick(key);
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
							})
						)}
					</Menu>

					<SkillsBottomBarChip
						allSkills={allSkills}
						loading={skillsLoading}
						enabledSkillRefs={enabledSkillRefs}
						activeSkillRefs={activeSkillRefs}
						setEnabledSkillRefs={setEnabledSkillRefs}
						onEnableAll={onEnableAllSkills}
						onDisableAll={onDisableAllSkills}
						isInputLocked={isInputLocked}
					/>

					<WebSearchBottomBarChip
						eligibleTools={eligibleWebSearchTools}
						enabled={webSearchEnabled}
						selectedCount={compatibleWebSearchTemplates.length}
						selected={
							activeWebSearch
								? {
										bundleID: activeWebSearch.bundleID,
										toolSlug: activeWebSearch.toolSlug,
										toolVersion: activeWebSearch.toolVersion,
									}
								: undefined
						}
						canEdit={!!activeWebSearchDef?.toolDefinition.userArgSchema}
						argsStatus={activeWebSearchArgsStatus}
						onEnabledChange={handleWebSearchEnabled}
						onSelectTool={handleWebSearchToolSelected}
						isInputLocked={isInputLocked}
						onEditOptions={() => {
							// Open the unified tool-args modal targeting "web search".
							// (ToolArgsModalHost should apply this to the active web-search tool.)
							if (!activeWebSearch) return;
							dispatchOpenToolArgs({ kind: 'webSearch' }, toolArgsEventTarget);
						}}
					/>
				</div>

				<div className="ml-auto flex items-center gap-1">
					<CommandTipsMenu shortcutConfig={shortcutConfig} />
				</div>
			</div>
			<UrlAttachmentModal
				isOpen={isUrlModalOpen}
				onClose={() => {
					setIsUrlModalOpen(false);
				}}
				onAttachURL={onAttachURL}
			/>
		</div>
	);
});
