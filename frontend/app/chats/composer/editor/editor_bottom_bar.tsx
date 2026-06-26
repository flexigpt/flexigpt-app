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
import { SkillsBottomBarChip } from '@/chats/composer/skills/skills_bottom_bar_chip';
import { SystemPromptBottomBarChip } from '@/chats/composer/systemprompts/system_prompt_bottom_bar_chip';
import type { ComposerSystemPromptController } from '@/chats/composer/systemprompts/use_composer_system_prompt';
import type { PromptTemplateInsertArgs } from '@/chats/composer/templates/prompt_template_bottom_bar_chip';
import { PromptTemplateBottomBarChip } from '@/chats/composer/templates/prompt_template_bottom_bar_chip';
import { ToolsBottomBarChip } from '@/chats/composer/tools/tools_bottom_bar_chip';
import type { WebSearchChoiceTemplate } from '@/chats/composer/tools/websearch_utils';
import type { ConversationToolStateEntry } from '@/tools/lib/conversation_tool_utils';

interface EditorBottomBarProps {
	onAttachFiles: () => Promise<void> | void;
	onAttachDirectory: () => Promise<void> | void;
	onAttachURL: (url: string) => Promise<void> | void;
	onOpenAttachmentUrlModal?: () => void;
	onUrlAttachmentModalClose?: () => void;
	onInsertTemplate: (args: PromptTemplateInsertArgs) => Promise<void> | void;

	templateMenuState: MenuStore;
	toolMenuState: MenuStore;
	attachmentMenuState: MenuStore;
	systemPromptMenuState: MenuStore;
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
	onEnableAllSkills: () => void;
	onDisableAllSkills: () => void;
	isInputLocked?: boolean;
	systemPrompt: ComposerSystemPromptController;
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
	onInsertTemplate,
	templateMenuState,
	toolMenuState,
	attachmentMenuState,
	systemPromptMenuState,
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
	onEnableAllSkills,
	onDisableAllSkills,
	isInputLocked = false,
	systemPrompt,
	mcpState,
	mcpAppContextUpdateCount = 0,
	onClearMCPAppContextUpdates,
}: EditorBottomBarProps) {
	const shortcutLabels = useMemo(
		() => ({
			templates: formatShortcut(shortcutConfig.insertTemplate),
			tools: formatShortcut(shortcutConfig.insertTool),
			attachments: formatShortcut(shortcutConfig.insertAttachment),
			systemPrompt: formatShortcut(shortcutConfig.insertSystemPrompt),
			skills: formatShortcut(shortcutConfig.attachSkills),
			mcp: formatShortcut(shortcutConfig.attachMCP),
		}),
		[shortcutConfig]
	);

	return (
		<div
			className="bg-base-200 w-full overflow-hidden"
			data-attachments-bottom-bar
			aria-label="Templates, tools, attachments, system prompt, skills, and MCP"
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

					<PromptTemplateBottomBarChip
						store={templateMenuState}
						buttonRef={templateButtonRef}
						shortcut={shortcutLabels.templates}
						onInsertTemplate={onInsertTemplate}
						isInputLocked={isInputLocked}
					/>

					<SystemPromptBottomBarChip
						store={systemPromptMenuState}
						shortcut={shortcutLabels.systemPrompt}
						systemPrompt={systemPrompt}
						isInputLocked={isInputLocked}
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
						onEnableAll={onEnableAllSkills}
						onDisableAll={onDisableAllSkills}
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
				</div>

				<div className="ml-auto flex items-center gap-1">
					<CommandTipsMenu shortcutConfig={shortcutConfig} />
				</div>
			</div>
		</div>
	);
});
