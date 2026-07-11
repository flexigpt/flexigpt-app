import type { Dispatch, RefObject, SetStateAction } from 'react';
import { memo, useMemo } from 'react';

import type { MenuStore } from '@ariakit/react';

import type { ProviderSDKType } from '@/spec/inference';
import type { SkillListItem, SkillRef } from '@/spec/skill';
import type { ToolListItem } from '@/spec/tool';

import type { ShortcutConfig } from '@/lib/keyboard_shortcuts';
import { formatShortcut } from '@/lib/keyboard_shortcuts';

import { AttachmentBottomBarChip } from '@/chats/composer/attachments/attachment_bottom_bar_chip';
import { CommandTipsMenu } from '@/chats/composer/inputtips/command_tips_menu';
import { MCPBottomBarChip } from '@/chats/composer/mcp/mcp_bottom_bar_chip';
import type { UseComposerMCPResult } from '@/chats/composer/mcp/mcp_composer_types';
import type { AttachedToolEntry } from '@/chats/composer/platedoc/tool_document_ops';
import { SkillTemplateBottomBarChip } from '@/chats/composer/skills/skill_template_bottom_bar_chip';
import { SkillsBottomBarChip } from '@/chats/composer/skills/skills_bottom_bar_chip';
import type { ComposerSystemPromptController } from '@/chats/composer/skills/use_composer_system_prompt';
import { ToolsBottomBarChip } from '@/chats/composer/tools/tools_bottom_bar_chip';
import type { WebSearchChoiceTemplate } from '@/chats/composer/tools/websearch_utils';
import type { ConversationToolStateEntry } from '@/tools/lib/conversation_tool_utils';

interface EditorBottomBarProps {
	onAttachFiles: () => Promise<void> | void;
	onAttachDirectory: () => Promise<void> | void;
	onAttachURL: (url: string) => Promise<void> | void;
	onOpenAttachmentUrlModal?: () => void;
	onUrlAttachmentModalClose?: () => void;
	onInsertTemplateText: (text: string) => Promise<void> | void;
	onAttachTemplateResourcePaths?: (paths: string[]) => Promise<void> | void;

	templateMenuState: MenuStore;
	toolMenuState: MenuStore;
	attachmentMenuState: MenuStore;
	skillsMenuState: MenuStore;
	mcpMenuState: MenuStore;

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
	setActiveSkillRefs: Dispatch<SetStateAction<SkillRef[]>>;
	onEnableAllSkills: () => void;
	onDisableAllSkills: () => void;
	onRefreshSkills: () => Promise<void>;
	systemPrompt: ComposerSystemPromptController;
	isInputLocked?: boolean;
	mcpState: UseComposerMCPResult;
	mcpAppContextUpdateCount?: number;
	onClearMCPAppContextUpdates?: () => void;
}

export const EditorBottomBar = memo(function EditorBottomBar({
	onAttachFiles,
	onAttachDirectory,
	onAttachURL,
	onOpenAttachmentUrlModal,
	onUrlAttachmentModalClose,
	onInsertTemplateText,
	onAttachTemplateResourcePaths,
	templateMenuState,
	toolMenuState,
	attachmentMenuState,
	skillsMenuState,
	mcpMenuState,
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
	setActiveSkillRefs,
	onEnableAllSkills,
	onDisableAllSkills,
	onRefreshSkills,
	systemPrompt,
	isInputLocked = false,
	mcpState,
	mcpAppContextUpdateCount = 0,
	onClearMCPAppContextUpdates,
}: EditorBottomBarProps) {
	const shortcutLabels = useMemo(
		() => ({
			templates: formatShortcut(shortcutConfig.insertTemplate),
			tools: formatShortcut(shortcutConfig.insertTool),
			attachments: formatShortcut(shortcutConfig.insertAttachment),
			skills: formatShortcut(shortcutConfig.attachSkills),
			mcp: formatShortcut(shortcutConfig.attachMCP),
		}),
		[shortcutConfig]
	);

	return (
		<div
			className="bg-base-200 w-full shrink-0 overflow-hidden"
			data-attachments-bottom-bar
			aria-label="Templates, tools, attachments, skills, and MCP"
		>
			<div className="flex items-center gap-1 overflow-x-auto p-1 text-xs shadow-none">
				<div className="flex items-center gap-1">
					<AttachmentBottomBarChip
						store={attachmentMenuState}
						buttonRef={attachmentButtonRef}
						shortcut={shortcutLabels.attachments}
						onAttachFiles={onAttachFiles}
						onAttachDirectory={onAttachDirectory}
						onAttachURL={onAttachURL}
						onOpenAttachmentUrlModal={onOpenAttachmentUrlModal}
						onUrlAttachmentModalClose={onUrlAttachmentModalClose}
						isInputLocked={isInputLocked}
					/>

					<MCPBottomBarChip
						store={mcpMenuState}
						shortcut={shortcutLabels.mcp}
						state={mcpState}
						isInputLocked={isInputLocked}
						appContextUpdateCount={mcpAppContextUpdateCount}
						onClearAppContextUpdates={onClearMCPAppContextUpdates}
					/>

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
						store={skillsMenuState}
						shortcut={shortcutLabels.skills}
						allSkills={allSkills}
						loading={skillsLoading}
						enabledSkillRefs={enabledSkillRefs}
						activeSkillRefs={activeSkillRefs}
						setEnabledSkillRefs={setEnabledSkillRefs}
						setActiveSkillRefs={setActiveSkillRefs}
						onEnableAll={onEnableAllSkills}
						onDisableAll={onDisableAllSkills}
						onRefreshSkills={onRefreshSkills}
						systemPrompt={systemPrompt}
						isInputLocked={isInputLocked}
					/>

					<SkillTemplateBottomBarChip
						store={templateMenuState}
						buttonRef={templateButtonRef}
						shortcut={shortcutLabels.templates}
						onInsertTemplateText={onInsertTemplateText}
						onAttachResourcePaths={onAttachTemplateResourcePaths}
						isInputLocked={isInputLocked}
					/>
				</div>

				<div className="ml-auto flex items-center gap-1">
					<CommandTipsMenu shortcutConfig={shortcutConfig} />
				</div>
			</div>
		</div>
	);
});
