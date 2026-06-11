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

import { FiFilePlus, FiFolder, FiLink, FiPaperclip, FiUpload } from 'react-icons/fi';

import { Menu, MenuButton, MenuItem, type MenuStore, useStoreState } from '@ariakit/react';

import type { ProviderSDKType } from '@/spec/inference';
import type { PromptTemplate, PromptTemplateListItem } from '@/spec/prompt';
import type { SkillListItem, SkillRef } from '@/spec/skill';
import { type ToolListItem } from '@/spec/tool';

import { formatShortcut, type ShortcutConfig } from '@/lib/keyboard_shortcuts';

import { promptStoreAPI } from '@/apis/baseapi';

import {
	actionTriggerChipButtonClasses,
	ActionTriggerChipContent,
	actionTriggerMenuItemClasses,
	actionTriggerMenuWideClasses,
} from '@/components/action_trigger_chip';
import { HoverTip } from '@/components/ariakit_hover_tip';

import { UrlAttachmentModal } from '@/chats/composer/attachments/attachment_url_modal';
import { CommandTipsMenu } from '@/chats/composer/inputtips/command_tips_menu';
import { MCPBottomBarChip } from '@/chats/composer/mcp/mcp_bottom_bar_chip';
import type { UseComposerMCPResult } from '@/chats/composer/mcp/mcp_composer_types';
import type { AttachedToolEntry } from '@/chats/composer/platedoc/tool_document_ops';
import { SkillsBottomBarChip } from '@/chats/composer/skills/skills_bottom_bar_chip';
import { SystemPromptBottomBarChip } from '@/chats/composer/systemprompts/system_prompt_bottom_bar_chip';
import type { ComposerSystemPromptController } from '@/chats/composer/systemprompts/use_composer_system_prompt';
import { PromptTemplateDropdown } from '@/chats/composer/templates/prompt_template_dropdown';
import { ToolsBottomBarChip } from '@/chats/composer/tools/tools_bottom_bar_chip';
import { type WebSearchChoiceTemplate } from '@/chats/composer/tools/websearch_utils';
import { usePromptTemplates } from '@/prompts/lib/use_prompt_templates';
import type { ConversationToolStateEntry } from '@/tools/lib/conversation_tool_utils';

interface EditorBottomBarProps {
	onAttachFiles: () => Promise<void> | void;
	onAttachDirectory: () => Promise<void> | void;
	onAttachURL: (url: string) => Promise<void> | void;
	onOpenAttachmentUrlModal?: () => void;
	onUrlAttachmentModalClose?: () => void;
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
	mcpState: UseComposerMCPResult;
	mcpAppContextUpdateCount?: number;
	onClearMCPAppContextUpdates?: () => void;
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

export const EditorBottomBar = memo(function EditorBottomBar({
	onAttachFiles,
	onAttachDirectory,
	onAttachURL,
	onOpenAttachmentUrlModal,
	onUrlAttachmentModalClose,
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
	allSkills,
	skillsLoading = false,
	enabledSkillRefs,
	activeSkillRefs,
	setEnabledSkillRefs,
	onEnableAllSkills,
	onDisableAllSkills,
	isInputLocked = false,
	systemPrompt,
	mcpState,
	mcpAppContextUpdateCount = 0,
	onClearMCPAppContextUpdates,
}: EditorBottomBarProps) {
	const [isUrlModalOpen, setIsUrlModalOpen] = useState(false);
	const templateMenuOpen = useStoreState(templateMenuState, 'open');

	const shortcutLabels = useMemo(
		() => ({
			templates: formatShortcut(shortcutConfig.insertTemplate),
			tools: formatShortcut(shortcutConfig.insertTool),
			attachments: formatShortcut(shortcutConfig.insertAttachment),
		}),
		[shortcutConfig]
	);
	const { data: templateData, loading: templatesLoading } = usePromptTemplates();

	useEffect(() => {
		if (isInputLocked) {
			templateMenuState.hide();
			toolMenuState.hide();
			attachmentMenuState.hide();
			// eslint-disable-next-line react-hooks/set-state-in-effect
			setIsUrlModalOpen(false);
		}
	}, [attachmentMenuState, isInputLocked, templateMenuState, toolMenuState]);

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

	const handleAttachmentPickFiles = async () => {
		await onAttachFiles();
		closeAttachmentMenu();
	};

	const handleAttachmentPickDirectory = async () => {
		await onAttachDirectory();
		closeAttachmentMenu();
	};

	const handleAttachmentPickURL = () => {
		onOpenAttachmentUrlModal?.();
		closeAttachmentMenu();
		setIsUrlModalOpen(true);
	};

	return (
		<div
			className="bg-base-200 w-full overflow-hidden"
			data-attachments-bottom-bar
			aria-label="Templates, tools, attachments, skills, and system prompt"
		>
			<div className="flex items-center gap-1 overflow-x-auto p-1 text-xs shadow-none">
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
						className={actionTriggerMenuWideClasses}
						data-menu-kind="attachments"
						autoFocusOnShow
					>
						<MenuItem
							onClick={() => {
								void handleAttachmentPickFiles();
							}}
							className={actionTriggerMenuItemClasses}
						>
							<FiUpload size={14} />
							<span>Multiple Files...</span>
						</MenuItem>
						<MenuItem
							onClick={() => {
								void handleAttachmentPickDirectory();
							}}
							className={actionTriggerMenuItemClasses}
						>
							<FiFolder size={14} />
							<span>Folder...</span>
						</MenuItem>
						<MenuItem onClick={handleAttachmentPickURL} className={actionTriggerMenuItemClasses}>
							<FiLink size={14} />
							<span>Link or URL...</span>
						</MenuItem>
					</Menu>

					<PickerButton
						label="Prompts"
						icon={<FiFilePlus size={16} />}
						buttonRef={templateButtonRef}
						menuState={templateMenuState}
						shortcut={shortcutLabels.templates}
						disabled={isInputLocked}
					/>
					<PromptTemplateDropdown
						store={templateMenuState}
						open={templateMenuOpen}
						loading={templatesLoading}
						items={templateData}
						onPick={item => {
							void handleTemplatePick(item);
						}}
					/>

					<SystemPromptBottomBarChip systemPrompt={systemPrompt} isInputLocked={isInputLocked} />

					<ToolsBottomBarChip
						store={toolMenuState}
						buttonRef={toolButtonRef}
						shortcut={shortcutLabels.tools}
						currentProviderSDKType={currentProviderSDKType}
						attachedToolEntries={attachedToolEntries}
						conversationToolsState={conversationToolsState}
						setConversationToolsState={setConversationToolsState}
						onAttachTool={onAttachTool}
						onDetachToolByKey={onDetachToolByKey}
						onSetAttachedToolAutoExecute={onSetAttachedToolAutoExecute}
						onRemoveAttachedTool={onRemoveAttachedTool}
						onRemoveAllAttachedTools={onRemoveAllAttachedTools}
						onEditAttachedToolOptions={onEditAttachedToolOptions}
						onOpenAttachedToolDetails={onOpenAttachedToolDetails}
						onOpenConversationToolDetails={onOpenConversationToolDetails}
						webSearchTemplates={webSearchTemplates}
						setWebSearchTemplates={setWebSearchTemplates}
						onWebSearchArgsBlockedChange={onWebSearchArgsBlockedChange}
						toolArgsEventTarget={toolArgsEventTarget}
						isInputLocked={isInputLocked}
					/>

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
					<MCPBottomBarChip
						state={mcpState}
						isInputLocked={isInputLocked}
						appContextUpdateCount={mcpAppContextUpdateCount}
						onClearAppContextUpdates={onClearMCPAppContextUpdates}
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
					onUrlAttachmentModalClose?.();
				}}
				onAttachURL={onAttachURL}
			/>
		</div>
	);
});
