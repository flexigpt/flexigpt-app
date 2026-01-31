import { type Dispatch, type ReactNode, type RefObject, type SetStateAction, useMemo, useState } from 'react';

import { FiFilePlus, FiFolder, FiLink, FiPaperclip, FiTool, FiUpload } from 'react-icons/fi';

import { Menu, MenuButton, MenuItem, type MenuStore } from '@ariakit/react';
import { type PlateEditor, useEditorRef } from 'platejs/react';

import type { ProviderSDKType } from '@/spec/inference';
import type { PromptTemplateListItem } from '@/spec/prompt';
import { ToolImplType, type ToolListItem, ToolStoreChoiceType } from '@/spec/tool';

import { formatShortcut, type ShortcutConfig } from '@/lib/keyboard_shortcuts';

import { usePromptTemplates } from '@/hooks/use_template';
import { useTools } from '@/hooks/use_tool';

import { promptStoreAPI } from '@/apis/baseapi';

import { CommandTipsMenu } from '@/chats/attachments/attachment_command_tips_menu';
import { UrlAttachmentModal } from '@/chats/attachments/attachment_url_modal';
import { dispatchOpenToolArgs } from '@/chats/events/open_attached_toolargs';
import { insertTemplateSelectionNode } from '@/chats/templates/template_editor_utils';
import {
	computeToolUserArgsStatus,
	getToolNodesWithPath,
	insertToolSelectionNode,
	toolIdentityKey,
} from '@/chats/tools/tool_editor_utils';
import { WebSearchBottomBarChip } from '@/chats/tools/web_search_bottom_bar_chip';
import {
	getEligibleWebSearchTools,
	type WebSearchChoiceTemplate,
	webSearchIdentityKey,
	webSearchTemplateFromToolListItem,
} from '@/chats/tools/websearch_utils';

interface AttachmentBottomBarProps {
	onAttachFiles: () => Promise<void> | void;
	onAttachDirectory: () => Promise<void> | void;
	onAttachURL: (url: string) => Promise<void> | void;

	templateMenuState: MenuStore;
	toolMenuState: MenuStore;
	attachmentMenuState: MenuStore;

	templateButtonRef: RefObject<HTMLButtonElement | null>;
	toolButtonRef: RefObject<HTMLButtonElement | null>;
	attachmentButtonRef: RefObject<HTMLButtonElement | null>;

	shortcutConfig: ShortcutConfig;
	currentProviderSDKType: ProviderSDKType;

	onToolsChanged?: () => void;
	// Web-search state comes from EditorArea (separate UX/state)
	webSearchTemplates: WebSearchChoiceTemplate[];
	setWebSearchTemplates: Dispatch<SetStateAction<WebSearchChoiceTemplate[]>>;
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
	const tooltip = shortcut ? `${label} (${shortcut})` : label;
	return (
		<div className="tooltip tooltip-right" data-tip={tooltip}>
			<MenuButton
				ref={buttonRef}
				store={menuState}
				disabled={disabled}
				className="btn btn-ghost btn-circle btn-sm text-neutral-custom hover:text-base-content"
				aria-label={tooltip}
			>
				{icon}
			</MenuButton>
		</div>
	);
}

const menuClasses =
	'rounded-box bg-base-100 text-base-content z-50 max-h-72 min-w-[240px] overflow-y-auto border border-base-300 p-1 shadow-xl';

const menuItemClasses =
	'flex items-center gap-2 rounded-xl px-2 py-1 text-sm outline-none transition-colors ' +
	'hover:bg-base-200 data-[active-item]:bg-base-300';

/**
  Bottom bar for template/tool/attachment buttons and tips menus.
  The chips scroller now lives in a separate bar inside the editor.
*/
export function AttachmentBottomBar({
	onAttachFiles,
	onAttachDirectory,
	onAttachURL,

	templateMenuState,
	toolMenuState,
	attachmentMenuState,
	templateButtonRef,
	toolButtonRef,
	attachmentButtonRef,
	shortcutConfig,
	currentProviderSDKType,
	onToolsChanged,
	webSearchTemplates,
	setWebSearchTemplates,
}: AttachmentBottomBarProps) {
	const editor = useEditorRef() as PlateEditor;
	const [isUrlModalOpen, setIsUrlModalOpen] = useState(false);

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

	/**
	 * Per-tool UI preference for auto-execute in the picker list.
	 * Keyed by tool identity; defaults to toolDefinition.autoExecReco.
	 *
	 * This is UI-only until the tool is actually inserted (at which point it becomes
	 * a ToolSelectionElementNode.autoExecute and eventually ToolStoreChoice.autoExecute).
	 */
	const [toolAutoExecOverrides, setToolAutoExecOverrides] = useState<Record<string, boolean>>({});
	const getAutoExecForTool = useMemo(() => {
		return (item: ToolListItem): boolean => {
			const key = toolIdentityKey(item.bundleID, item.bundleSlug, item.toolSlug, item.toolVersion);
			const override = toolAutoExecOverrides[key];
			if (typeof override === 'boolean') return override;
			return item.toolDefinition.autoExecReco ?? false;
		};
	}, [toolAutoExecOverrides]);

	const toolEntries = getToolNodesWithPath(editor);
	const webSearchEnabled = webSearchTemplates.length > 0;

	const eligibleWebSearchTools = useMemo(() => {
		if (toolsLoading) return [];
		return getEligibleWebSearchTools(toolData, currentProviderSDKType);
	}, [toolData, toolsLoading, currentProviderSDKType]);

	// "Active" web-search tool for the bottom-bar UX (first one wins).
	const activeWebSearch = webSearchTemplates.length > 0 ? webSearchTemplates[0] : undefined;

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

	const attachedToolKeys = new Set(
		toolEntries.map(([node]) => toolIdentityKey(node.bundleID, node.bundleSlug, node.toolSlug, node.toolVersion))
	);

	const availableTools: ToolListItem[] = toolsLoading
		? []
		: toolData.filter(it => {
				// Web search is a specially handled tool
				if (it.toolDefinition.llmToolType === ToolStoreChoiceType.WebSearch) return false;

				// Hide tools that are already attached in this draft.
				if (attachedToolKeys.has(toolIdentityKey(it.bundleID, it.bundleSlug, it.toolSlug, it.toolVersion))) {
					return false;
				}

				// If we know the provider's SDK type, restrict SDK tools to matching ones.
				if (it.toolDefinition.type === ToolImplType.SDK && it.toolDefinition.sdkImpl) {
					const sdkType = it.toolDefinition.sdkImpl.sdkType;
					if (!sdkType) return false;
					return sdkType === currentProviderSDKType.toString();
				}

				// Non-SDK tools (Go/HTTP/etc.) are always shown.
				return true;
			});

	const closeTemplateMenu = () => {
		templateMenuState.hide();
	};
	const closeToolMenu = () => {
		toolMenuState.hide();
	};
	const closeAttachmentMenu = () => {
		attachmentMenuState.hide();
	};

	const handleTemplatePick = async (item: PromptTemplateListItem) => {
		try {
			const tmpl = await promptStoreAPI.getPromptTemplate(item.bundleID, item.templateSlug, item.templateVersion);
			insertTemplateSelectionNode(editor, item.bundleID, item.templateSlug, item.templateVersion, tmpl);
		} catch {
			insertTemplateSelectionNode(editor, item.bundleID, item.templateSlug, item.templateVersion);
		} finally {
			closeTemplateMenu();
			editor.tf.focus();
		}
	};

	const handleToolPick = async (item: ToolListItem, autoExecute: boolean) => {
		insertToolSelectionNode(
			editor,
			{
				bundleID: item.bundleID,
				bundleSlug: item.bundleSlug,
				toolSlug: item.toolSlug,
				toolVersion: item.toolVersion,
			},
			item.toolDefinition,
			{
				autoExecute,
			}
		);

		onToolsChanged?.();

		closeToolMenu();
		editor.tf.focus();
	};

	const handleAttachmentPickFiles = async () => {
		await onAttachFiles();
		closeAttachmentMenu();
		editor.tf.focus();
	};

	const handleAttachmentPickDirectory = async () => {
		await onAttachDirectory();
		closeAttachmentMenu();
		editor.tf.focus();
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
		// default enable: add the first eligible tool if none selected
		if (webSearchTemplates.length > 0 || eligibleWebSearchTools.length === 0) return;

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
			aria-label="Templates, tools, and attachments"
		>
			<div className="flex items-center gap-2 px-1 py-1 text-xs shadow-none">
				{/* Left: template / tool / attachment pickers */}
				<div className="flex items-center gap-1">
					<PickerButton
						label="Attach files or links"
						icon={<FiPaperclip size={16} />}
						buttonRef={attachmentButtonRef}
						menuState={attachmentMenuState}
						shortcut={shortcutLabels.attachments}
					/>
					<Menu
						store={attachmentMenuState}
						gutter={8}
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
					<PickerButton
						label="Insert template"
						icon={<FiFilePlus size={16} />}
						buttonRef={templateButtonRef}
						menuState={templateMenuState}
						shortcut={shortcutLabels.templates}
					/>
					<Menu store={templateMenuState} gutter={8} className={menuClasses} data-menu-kind="templates" autoFocusOnShow>
						{templatesLoading ? (
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
						label="Add tool"
						icon={<FiTool size={16} />}
						buttonRef={toolButtonRef}
						menuState={toolMenuState}
						shortcut={shortcutLabels.tools}
					/>
					<Menu store={toolMenuState} gutter={8} className={menuClasses} data-menu-kind="tools" autoFocusOnShow>
						{toolsLoading ? (
							<div className={`${menuItemClasses} text-base-content/60 cursor-default`}>Loading tools…</div>
						) : availableTools.length === 0 ? (
							<div className={`${menuItemClasses} text-base-content/60 cursor-default`}>No additional tools</div>
						) : (
							availableTools.map(item => (
								// Show full identity in the tooltip; visible text stays compact.
								// e.g. "my-tool (bundleSlug/toolSlug@version)".
								<MenuItem
									key={`${item.bundleID}-${item.toolSlug}-${item.toolVersion}`}
									onClick={() => {
										void handleToolPick(item, getAutoExecForTool(item));
									}}
									hideOnClick={false}
									className={menuItemClasses}
									title={`${item.toolSlug.replace(/[-_]/g, ' ')} • ${
										item.bundleSlug ?? item.bundleID
									}/${item.toolSlug}@${item.toolVersion}`}
								>
									<FiTool size={14} />
									<span className="truncate">{item.toolSlug.replace(/[-_]/g, ' ')}</span>
									{/* Auto-execute toggle: function/custom tools only (websearch is excluded from this menu). */}
									{(item.toolDefinition.llmToolType === ToolStoreChoiceType.Function ||
										item.toolDefinition.llmToolType === ToolStoreChoiceType.Custom) && (
										<div
											className="ml-auto flex items-center gap-2"
											onClick={e => {
												// Prevent toggling from selecting/inserting the tool.
												e.preventDefault();
												e.stopPropagation();
											}}
										>
											<span className="text-base-content/60 text-[10px] uppercase">Auto</span>
											<input
												type="checkbox"
												className="toggle toggle-xs"
												tabIndex={-1}
												checked={getAutoExecForTool(item)}
												onChange={e => {
													const key = toolIdentityKey(item.bundleID, item.bundleSlug, item.toolSlug, item.toolVersion);
													const next = e.currentTarget.checked;
													setToolAutoExecOverrides(prev => ({ ...prev, [key]: next }));
												}}
												aria-label={`Auto-execute ${item.toolSlug}`}
												title={getAutoExecForTool(item) ? 'Auto-execute enabled' : 'Auto-execute disabled'}
											/>
										</div>
									)}
									<span className="text-base-content/50 ml-auto text-[10px] uppercase" aria-hidden="true">
										{item.toolVersion}
									</span>
								</MenuItem>
							))
						)}
					</Menu>

					<WebSearchBottomBarChip
						eligibleTools={eligibleWebSearchTools}
						enabled={webSearchEnabled}
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
						onEditOptions={() => {
							// Open the unified tool-args modal targeting "web search".
							// (ToolArgsModalHost should apply this to the active web-search tool.)
							if (!activeWebSearch) return;
							dispatchOpenToolArgs({ kind: 'webSearch' });
						}}
					/>
				</div>

				{/* Right: keyboard shortcuts & tips menus */}
				<div className="ml-auto flex items-center gap-1">
					<CommandTipsMenu shortcutConfig={shortcutConfig} />
				</div>
			</div>
			{/* URL attachment dialog */}
			<UrlAttachmentModal
				isOpen={isUrlModalOpen}
				onClose={() => {
					setIsUrlModalOpen(false);
					editor.tf.focus();
				}}
				onAttachURL={onAttachURL}
			/>
		</div>
	);
}
